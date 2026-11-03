package repository

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
	"github.com/maldous/CircularQuay/internal/quota"
	"golang.org/x/crypto/ssh"
)

func privateRepositoryDirectory(t *testing.T, name string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, name)
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestLocalCommandsDoNotLoadFetchToken(t *testing.T) {
	s := &store{repo: model.Repository{TokenFile: filepath.Join(t.TempDir(), "absent-token"), URL: "https://example.invalid/repo"}}
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	called := false
	runGit = func(context.Context, []string, int64, ...string) ([]byte, error) {
		called = true
		return []byte("synthetic"), nil
	}
	if _, err := s.command(context.Background(), 4096, "cat-file", "commit", strings.Repeat("a", 40)); err != nil || !called {
		t.Fatal("local signature verification requires transport credentials")
	}
	if _, err := s.command(context.Background(), 4096, "arbitrary-command"); !fault.Is(err, fault.Invalid) {
		t.Fatal("unregistered Git command accepted")
	}
}

func TestSignatureVerificationRejectsWritablePATHShadow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission fixture")
	}
	directory := t.TempDir()
	fake := filepath.Join(directory, "ssh-keygen")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fake, 0777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	s := &store{repo: model.Repository{URL: "https://example.invalid/repo"}}
	called := false
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	runGit = func(context.Context, []string, int64, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}
	if _, err := s.verificationCommand(context.Background(), 4096, "verify-commit", strings.Repeat("a", 40)); !fault.Is(err, fault.Trust) {
		t.Fatalf("writable verifier accepted: %v", err)
	}
	if called {
		t.Fatal("untrusted verifier reached Git")
	}
}

func TestTreeRejectsPortableCollisions(t *testing.T) {
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	object := strings.Repeat("a", 40)
	cases := [][]string{{"Foo", "foo/x"}, {"foo/x", "Foo"}, {"Foo/x", "foo/y"}, {"x", "x/y"}, {"x/y", "x"}, {"x", "x"}, {"../escape"}, {"a/../b"}, {"a/./b"}, {"."}, {".."}}
	for _, paths := range cases {
		t.Run(strings.Join(paths, "_"), func(t *testing.T) {
			var tree strings.Builder
			for _, p := range paths {
				tree.WriteString("100644 blob " + object + "\t" + p + "\x00")
			}
			runGit = func(context.Context, []string, int64, ...string) ([]byte, error) { return []byte(tree.String()), nil }
			if _, err := (&store{}).readTree(context.Background(), object); err == nil {
				t.Fatal("unsafe tree accepted")
			}
		})
	}
}

func TestPortablePathsRejectDotSegments(t *testing.T) {
	for _, p := range []string{".", "..", "../escape", "a/../../escape", "a/../b", "a/./b"} {
		if policy.ValidRepositoryPath(p) {
			t.Errorf("unsafe path accepted: %q", p)
		}
	}
}

func TestRepositoryFileGuards(t *testing.T) {
	testCommitIdentifiers(t)
	testBlobIdentifiers(t)
	testTokensAndReadLimit(t)
}

func testCommitIdentifiers(t *testing.T) {
	t.Helper()
	for _, commit := range []string{strings.Repeat("a", 40), strings.Repeat("f", 40)} {
		if !validCommit(commit) {
			t.Fatalf("valid commit rejected: %q", commit)
		}
	}
	for _, commit := range []string{"", strings.Repeat("a", 39), strings.Repeat("A", 40), strings.Repeat("g", 40), strings.Repeat("a", 64)} {
		if validCommit(commit) {
			t.Fatalf("invalid commit accepted: %q", commit)
		}
	}
}

func testBlobIdentifiers(t *testing.T) {
	t.Helper()
	data := []byte("synthetic")
	if len(blobID(data, 40)) != 40 || len(blobID(data, 64)) != 64 {
		t.Fatal("unsupported Git object format digest")
	}
}

func testTokensAndReadLimit(t *testing.T) {
	t.Helper()
	for _, value := range [][]byte{nil, []byte("token\nvalue"), []byte("token\x00value")} {
		if _, err := safeToken(value); err == nil {
			t.Fatal("unsafe fetch token accepted")
		}
	}
	if token, err := safeToken([]byte(" token ")); err != nil || token != "token" {
		t.Fatal("safe token rejected")
	}
	path := filepath.Join(privateRepositoryDirectory(t, "input"), "input")
	writeProtectedRepositoryFixture(t, path, []byte("synthetic"))
	if _, err := readProtected(path, true, 1); !fault.Is(err, fault.Trust) {
		t.Fatal("read limit did not reject oversized input")
	}
}

func TestVerifyReleaseRejectsChangedMaterialization(t *testing.T) {
	directory, tree := materializedReleaseFixture(t, map[string]string{"values.json": "synthetic"})
	if err := verifyRelease(directory, tree); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "extra.json"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyRelease(directory, tree); !fault.Is(err, fault.Trust) {
		t.Fatal("extra materialized file accepted")
	}
}

func TestVerifyReleaseRejectsIncorrectContentAndModes(t *testing.T) {
	changes := []func(string, map[string]string){
		func(directory string, _ map[string]string) {
			_ = os.WriteFile(filepath.Join(directory, "values.json"), []byte("changed"), 0600)
		},
		func(_ string, tree map[string]string) { tree["missing.json"] = blobID([]byte("missing"), 40) },
	}
	if runtime.GOOS != "windows" {
		changes = append(changes, func(directory string, _ map[string]string) {
			_ = os.Chmod(filepath.Join(directory, "values.json"), 0700)
		})
	}
	for _, change := range changes {
		directory, tree := materializedReleaseFixture(t, map[string]string{"values.json": "synthetic"})
		change(directory, tree)
		if err := verifyRelease(directory, tree); !fault.Is(err, fault.Trust) {
			t.Fatal("changed materialization accepted")
		}
	}
}

func TestVerifyReleaseRejectsInsecureDirectoryAndWalkError(t *testing.T) {
	directory, tree := materializedReleaseFixture(t, map[string]string{"values.json": "synthetic"})
	if runtime.GOOS != "windows" {
		if err := os.Chmod(directory, 0755); err != nil {
			t.Fatal(err)
		}
		if err := verifyRelease(directory, tree); !fault.Is(err, fault.Trust) {
			t.Fatal("public release directory accepted")
		}
	}
	verifier := releaseVerifier{directory: directory, tree: tree}
	if err := verifier.visit(directory, nil, errors.New("synthetic walk failure")); !fault.Is(err, fault.Trust) {
		t.Fatal("walk failure accepted")
	}
}

func TestGitOutputAndEnvironmentBoundaries(t *testing.T) {
	output := &limitedOutput{remaining: 3}
	if n, err := output.Write([]byte("abc")); err != nil || n != 3 {
		t.Fatal("bounded output rejected exact limit")
	}
	if _, err := output.Write([]byte("d")); !fault.Is(err, fault.Trust) {
		t.Fatal("oversized Git output accepted")
	}
	token := filepath.Join(privateRepositoryDirectory(t, "credential"), "token")
	writeProtectedRepositoryFixture(t, token, []byte("synthetic-token"))
	s := &store{repo: model.Repository{URL: "https://example.invalid/repository", TokenFile: token}}
	local, err := s.environment(false, false)
	if err != nil || strings.Contains(strings.Join(local, "\n"), "synthetic-token") {
		t.Fatal("local Git command loaded fetch token")
	}
	fetch, err := s.environment(true, false)
	if err != nil || !strings.Contains(strings.Join(fetch, "\n"), "Authorization: Bearer synthetic-token") {
		t.Fatal("authenticated fetch did not provide isolated token header")
	}
}

func TestFetchRejectsBeforeGitWhenQuotaIsNotProven(t *testing.T) {
	directory := privateRepositoryDirectory(t, "repository")
	s := &store{repo: model.Repository{Directory: directory}}
	candidateCalled := false
	operations := fetchOperations{
		validateIdentity: func() error { return nil },
		assertQuota:      func(string) error { return failure(fault.Unsupported, "disk_quota") },
		prepareTemporary: func(*store) (func(), error) {
			t.Fatal("temporary directory prepared before quota proof")
			return nil, nil
		},
		candidate: func(context.Context, *store) (string, error) {
			candidateCalled = true
			return "", nil
		},
	}
	if _, err := s.fetch(context.Background(), operations); !fault.Is(err, fault.Unsupported) {
		t.Fatalf("unproven quota error = %v", err)
	}
	if candidateCalled {
		t.Fatal("Git candidate operation ran without quota proof")
	}
}

func TestFetchTemporaryIsRepositoryScoped(t *testing.T) {
	s := &store{repo: model.Repository{Directory: privateRepositoryDirectory(t, "repository")}}
	cleanup, err := prepareFetchTemporary(s)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	environment, err := s.environment(false, false)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(environment, "\n")
	for _, name := range []string{"TMPDIR=", "TEMP=", "TMP="} {
		if !strings.Contains(joined, name+s.temporary) {
			t.Fatalf("missing repository temporary setting %q", name)
		}
	}
}

func TestRepositoryQuotaLimitsRequireReleaseReserve(t *testing.T) {
	if err := quota.Check(fixtureQuota{quota.Status{HardBytes: maximumRepositoryBytes + maximumReleaseBytes, HardInodes: maximumRepositoryInodes + maximumFiles + repositoryTemporaryInodes, UserEnforced: true}}, "synthetic", repositoryQuotaLimits()); err != nil {
		t.Fatalf("boundary quota rejected: %v", err)
	}
}

type fixtureQuota struct{ status quota.Status }

func (f fixtureQuota) Status(string) (quota.Status, error) { return f.status, nil }

func materializedReleaseFixture(t *testing.T, files map[string]string) (string, map[string]string) {
	t.Helper()
	directory := privateRepositoryDirectory(t, "release")
	tree := make(map[string]string, len(files))
	for name, content := range files {
		path := filepath.Join(directory, filepath.FromSlash(name))
		if err := platform.EnsurePrivateDirectory(filepath.Dir(path)); err != nil {
			t.Fatal(err)
		}
		writeProtectedRepositoryFixture(t, path, []byte(content))
		tree[name] = blobID([]byte(content), 40)
	}
	return directory, tree
}

func TestAcceptedPointerStrictAndProtected(t *testing.T) {
	dir := privateRepositoryDirectory(t, "accepted")
	s := &store{repo: model.Repository{Directory: dir, URL: "https://example.invalid/repo", Ref: "refs/heads/data"}}
	expected := accepted{URL: s.repo.URL, Ref: s.repo.Ref, Commit: strings.Repeat("a", 40)}
	if err := s.writeAccepted(expected); err != nil {
		t.Fatal(err)
	}
	got, err := s.readAccepted()
	if err != nil || got != expected {
		t.Fatal("acceptance did not round trip")
	}
	p := filepath.Join(dir, "accepted.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"commit":`, `"commit":"`+strings.Repeat("b", 40)+`","commit":`, 1))
	if err = os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.readAccepted(); err == nil {
		t.Fatal("ambiguous accepted pointer allowed")
	}
}

func TestSignedRepositoryAcceptanceAndRollback(t *testing.T) {
	fixture := newSignedRepositoryFixture(t)
	first := fixture.writeCommit(t, `{"value":"synthetic-first"}`, true)
	repo := fixture.repository(first)
	installSyntheticTransport(t, repo, fixture.remote)
	accepted := fetchRepository(t, repo)
	fixture.removeToken(t)
	requireCurrent(t, repo, accepted)
	fixture.writeToken(t)
	second := fixture.writeCommit(t, `{"value":"synthetic-second"}`, true)
	accepted = fetchRepository(t, repo)
	if filepath.Base(accepted) != second {
		t.Fatal("signed descendant did not become current")
	}
	fixture.writeCommit(t, `{"value":"synthetic-unsigned"}`, false)
	requireRejectedFetch(t, repo, "unsigned descendant")
	requireCurrent(t, repo, accepted)
	fixture.invoke(t, fixture.git, "reset", "--hard", first)
	requireRejectedFetch(t, repo, "signed rollback")
	if err := os.WriteFile(filepath.Join(accepted, "values.json"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Current(repo); !fault.Is(err, fault.Trust) {
		t.Fatal("tampered materialization accepted")
	}
}

type signedRepositoryFixture struct {
	root, remote, git, key, signers, token string
	environment                            []string
}

func newSignedRepositoryFixture(t *testing.T) signedRepositoryFixture {
	t.Helper()
	git := fixtureGit(t)
	root := privateRepositoryDirectory(t, "signed-repository")
	fixture := signedRepositoryFixture{root: root, remote: filepath.Join(root, "remote"), git: git, key: filepath.Join(root, "signing"), signers: filepath.Join(root, "allowed-signers"), token: filepath.Join(root, "token")}
	fixture.environment = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + root, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_TERMINAL_PROMPT=0"}
	if err := os.Mkdir(fixture.remote, 0700); err != nil {
		t.Fatal(err)
	}
	fixture.writeSigningKey(t)
	fixture.invoke(t, git, "init", "--quiet", "--object-format=sha1", "--initial-branch=data")
	fixture.writeToken(t)
	return fixture
}

func fixtureGit(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git required for signed repository fixture")
	}
	if err := platform.ValidateExecutable(git); err != nil {
		t.Fatalf("fixture Git is not trusted: %v; %s", err, executableTrustDiagnostic(git))
	}
	return git
}

// executableTrustDiagnostic is emitted only when the fixture refuses the
// resolved Git binary. It includes filesystem metadata needed to diagnose an
// untrusted ancestor without reading credentials, environment values, or file
// contents. Reflection keeps the test cross-compiled where UID metadata is
// represented differently or unavailable.
func executableTrustDiagnostic(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Sprintf("look_path=%q resolve_error=%q", path, err)
	}
	var result strings.Builder
	fmt.Fprintf(&result, "look_path=%q resolved_path=%q", path, resolved)
	for current := resolved; ; current = filepath.Dir(current) {
		fmt.Fprintf(&result, "; %s", pathMetadataDiagnostic(current))
		parent := filepath.Dir(current)
		if parent == current {
			return result.String()
		}
	}
}

func pathMetadataDiagnostic(path string) string {
	lstat, lstatErr := os.Lstat(path)
	stat, statErr := os.Stat(path)
	return fmt.Sprintf("path=%q lstat=%s stat=%s", path, fileMetadataDiagnostic(lstat, lstatErr), fileMetadataDiagnostic(stat, statErr))
}

func fileMetadataDiagnostic(info os.FileInfo, err error) string {
	if err != nil {
		return fmt.Sprintf("error=%q", err)
	}
	return fmt.Sprintf("mode=%s uid=%s", info.Mode(), metadataUID(info))
}

func metadataUID(info os.FileInfo) string {
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer && !value.IsNil() {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return "unavailable"
	}
	uid := value.FieldByName("Uid")
	if !uid.IsValid() {
		return "unavailable"
	}
	switch uid.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", uid.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", uid.Int())
	default:
		return "unavailable"
	}
}

// writeSigningKey creates an OpenSSH-compatible key in process memory. Git
// still delegates signing and verification to its native SSH implementation;
// generating the fixture this way avoids platform-specific ssh-keygen CLI
// parsing, notably the empty-passphrase invocation on Windows.
func (f signedRepositoryFixture) writeSigningKey(t *testing.T) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	privateBlock, err := ssh.MarshalPrivateKey(private, "synthetic fixture")
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	writeProtectedRepositoryFixture(t, f.key, pem.EncodeToMemory(privateBlock))
	writeProtectedRepositoryFixture(t, f.signers, append([]byte("fixture "), ssh.MarshalAuthorizedKey(publicKey)...))
}

func (f signedRepositoryFixture) invoke(t *testing.T, program string, args ...string) string {
	t.Helper()
	cmd := exec.Command(program, args...)
	cmd.Env = f.environment
	cmd.Dir = f.remote
	data, err := cmd.Output()
	if err != nil {
		t.Fatalf("synthetic fixture command failed: %s: %v", filepath.Base(program), err)
	}
	return strings.TrimSpace(string(data))
}

func (f signedRepositoryFixture) writeCommit(t *testing.T, value string, sign bool) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.remote, "values.json"), []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
	f.invoke(t, f.git, "add", "--", "values.json")
	args := []string{"-c", "user.name=Synthetic Fixture", "-c", "user.email=fixture@example.invalid", "-c", "gpg.format=ssh", "-c", "user.signingkey=" + f.key, "commit", "--quiet", "-m", "synthetic data"}
	if sign {
		args = append(args, "-S")
	} else {
		args = append(args, "--no-gpg-sign")
	}
	f.invoke(t, f.git, args...)
	return f.invoke(t, f.git, "rev-parse", "HEAD")
}

func (f signedRepositoryFixture) writeToken(t *testing.T) {
	t.Helper()
	writeProtectedRepositoryFixture(t, f.token, []byte("synthetic-fetch-token"))
}

func writeProtectedRepositoryFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
}

func (f signedRepositoryFixture) removeToken(t *testing.T) {
	t.Helper()
	if err := os.Remove(f.token); err != nil {
		t.Fatal(err)
	}
}

func (f signedRepositoryFixture) repository(initial string) model.Repository {
	return model.Repository{URL: "https://example.invalid/synthetic.git", Directory: filepath.Join(f.root, "cache"), Ref: "refs/heads/data", AllowedSignersFile: f.signers, InitialCommit: initial, TokenFile: f.token}
}

func installSyntheticTransport(t *testing.T, repo model.Repository, remote string) {
	t.Helper()
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	runGit = func(ctx context.Context, environment []string, limit int64, args ...string) ([]byte, error) {
		for index, arg := range args {
			if arg == repo.URL {
				args[index] = remote
			}
		}
		args = append([]string{"-c", "protocol.file.allow=always"}, args...)
		return executeGit(ctx, environment, limit, args...)
	}
}

func fetchRepository(t *testing.T, repo model.Repository) string {
	t.Helper()
	s, err := openStore(repo, true)
	if err != nil {
		t.Fatal(err)
	}
	directory, err := s.fetch(context.Background(), syntheticFetchOperations())
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

func requireCurrent(t *testing.T, repo model.Repository, expected string) {
	t.Helper()
	if current, err := Current(repo); err != nil || current != expected {
		t.Fatal("accepted repository is not available offline")
	}
}

func requireRejectedFetch(t *testing.T, repo model.Repository, name string) {
	t.Helper()
	s, err := openStore(repo, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.fetch(context.Background(), syntheticFetchOperations()); !fault.Is(err, fault.Trust) {
		t.Fatalf("%s accepted: %v", name, err)
	}
}

func syntheticFetchOperations() fetchOperations {
	operations := nativeFetchOperations()
	operations.validateIdentity = func() error { return nil }
	operations.assertQuota = func(string) error { return nil }
	return operations
}
