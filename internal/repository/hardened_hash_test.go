package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestHardenedSHA1VersionPolicy(t *testing.T) {
	for value, want := range map[string]bool{
		"git version 2.12.9\n":  false,
		"git version 2.13.0\n":  true,
		"git version 3.0.0\n":   true,
		"git version 2.13\n":    true,
		"git version unknown\n": false,
		"2.43.0\n":              false,
	} {
		if got := hardenedSHA1Version(value); got != want {
			t.Fatalf("hardenedSHA1Version(%q) = %t, want %t", value, got, want)
		}
	}
}

func TestSensitiveGitSubcommandsRejectUnexpectedArguments(t *testing.T) {
	for _, arguments := range [][]string{
		{"hash-object", "-w", "--stdin"},
		{"hash-object", "--no-filters", "-t", "blob", "--", "relative"},
		{"fsck", "--strict", "--no-dangling", "not-a-commit"},
		{"version", "--build-options"},
	} {
		if validGitInvocation(arguments) {
			t.Fatalf("sensitive Git invocation accepted: %q", arguments)
		}
	}
}

func TestHardenedSHA1RequirementUsesFixedVersionCommand(t *testing.T) {
	withMockGit(t, func(_ context.Context, _ []string, _ int64, arguments ...string) ([]byte, error) {
		if strings.Join(arguments, " ") != "--git-dir= version" {
			t.Fatalf("Git version command = %q", arguments)
		}
		return []byte("git version 2.13.0\n"), nil
	})
	if err := (&store{}).requireHardenedSHA1(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestHardenedSHA1RequirementRejectsOldOrMalformedVersion(t *testing.T) {
	for _, version := range []string{"git version 2.12.9\n", "git version unknown\n"} {
		t.Run(strings.TrimSpace(version), func(t *testing.T) {
			withMockGit(t, func(context.Context, []string, int64, ...string) ([]byte, error) {
				return []byte(version), nil
			})
			if err := (&store{}).requireHardenedSHA1(context.Background()); !fault.Is(err, fault.Unsupported) {
				t.Fatalf("version %q error = %v", version, err)
			}
		})
	}
}

func TestMaterializedBlobVerifierUsesFixedReadOnlyHashObject(t *testing.T) {
	path := protectedHashFixture(t)
	expected := strings.Repeat("a", 40)
	withMockGit(t, func(_ context.Context, _ []string, limit int64, arguments ...string) ([]byte, error) {
		want := []string{"--git-dir=/private/objects.git", "hash-object", "--no-filters", "-t", "blob", "--", path}
		if limit != 128 || strings.Join(arguments, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("hash-object invocation = %q, limit=%d", arguments, limit)
		}
		return []byte(expected + "\n"), nil
	})
	if err := (&store{git: "/private/objects.git"}).verifyMaterializedBlob(context.Background(), path, expected); err != nil {
		t.Fatal(err)
	}
}

func TestMaterializedBlobVerifierRejectsMalformedOrMismatchedOutput(t *testing.T) {
	path := protectedHashFixture(t)
	expected := strings.Repeat("a", 40)
	for _, output := range []string{"", expected, expected + "\r\n", strings.Repeat("b", 40) + "\n", strings.Repeat("A", 40) + "\n"} {
		t.Run("output_"+hex.EncodeToString([]byte(output)), func(t *testing.T) {
			withMockGit(t, func(context.Context, []string, int64, ...string) ([]byte, error) {
				return []byte(output), nil
			})
			if err := (&store{}).verifyMaterializedBlob(context.Background(), path, expected); !fault.Is(err, fault.Trust) {
				t.Fatalf("output %q error = %v", output, err)
			}
		})
	}
}

func TestObjectGraphVerifierUsesFixedStrictArguments(t *testing.T) {
	commit := strings.Repeat("a", 40)
	withMockGit(t, func(_ context.Context, _ []string, limit int64, arguments ...string) ([]byte, error) {
		want := []string{"--git-dir=/private/objects.git", "fsck", "--strict", "--no-dangling", commit}
		if limit != 16<<10 || strings.Join(arguments, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("fsck invocation = %q, limit=%d", arguments, limit)
		}
		return nil, nil
	})
	if err := (&store{git: "/private/objects.git"}).verifyObjectGraph(context.Background(), commit); err != nil {
		t.Fatal(err)
	}
}

func TestNativeGitHashObjectVerifiesProtectedMaterialization(t *testing.T) {
	path := protectedHashFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	s := &store{}
	environment, err := s.environment(false, false)
	if err != nil {
		t.Fatal(err)
	}
	output, err := executeGit(ctx, environment, 128, "hash-object", "--no-filters", "-t", "blob", "--", path)
	if err != nil {
		t.Skipf("Git is unavailable for repository feature verification: %v", err)
	}
	if err = s.requireHardenedSHA1(ctx); err != nil {
		t.Skipf("installed Git lacks required SHA-1 hardening: %v", err)
	}
	if err = s.verifyMaterializedBlob(ctx, path, strings.TrimSpace(string(output))); err != nil {
		t.Fatal(err)
	}
}

func withMockGit(t *testing.T, mock func(context.Context, []string, int64, ...string) ([]byte, error)) {
	t.Helper()
	previous := runGit
	runGit = mock
	t.Cleanup(func() { runGit = previous })
}

func protectedHashFixture(t *testing.T) string {
	t.Helper()
	directory := privateRepositoryDirectory(t, "hash")
	path := filepath.Join(directory, "value.json")
	writeProtectedRepositoryFixture(t, path, []byte(`{"value":"synthetic"}`))
	return path
}

// blobID preserves the existing repository fixture helper without using
// crypto/sha1. Production verification always uses the hardened Git path.
func blobID(data []byte, length int) string {
	if length == 64 {
		header := []byte("blob " + strconv.Itoa(len(data)) + "\x00")
		hash := sha256.Sum256(append(header, data...))
		return hex.EncodeToString(hash[:])
	}
	directory, err := os.MkdirTemp("", "cquay-blob-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "blob")
	if err = os.WriteFile(path, data, 0o600); err != nil {
		panic(err)
	}
	output, err := executeGit(context.Background(), []string{"PATH=" + os.Getenv("PATH")}, 128, "hash-object", "--no-filters", "-t", "blob", "--", path)
	if err != nil {
		panic(err)
	}
	value := strings.TrimSpace(string(output))
	if len(value) != length {
		panic("unexpected Git object identifier length")
	}
	return value
}
