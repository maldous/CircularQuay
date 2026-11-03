package repository

import (
	"context"
	"github.com/maldous/CircularQuay/internal/fault"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
)

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

func TestAcceptedPointerStrictAndProtected(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
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
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git required for signed repository fixture")
	}
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("OpenSSH required for signed repository fixture")
	}
	root := t.TempDir()
	if err = os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(root, "remote")
	if err = os.Mkdir(remote, 0700); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + root, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_TERMINAL_PROMPT=0"}
	invoke := func(program string, args ...string) string {
		t.Helper()
		cmd := exec.Command(program, args...)
		cmd.Env = env
		cmd.Dir = remote
		data, err := cmd.Output()
		if err != nil {
			t.Fatalf("synthetic fixture command failed: %s: %v", filepath.Base(program), err)
		}
		return strings.TrimSpace(string(data))
	}
	key := filepath.Join(root, "signing")
	invoke(keygen, "-q", "-t", "ed25519", "-N", "", "-f", key)
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	signers := filepath.Join(root, "allowed-signers")
	if err = os.WriteFile(signers, append([]byte("fixture "), pub...), 0600); err != nil {
		t.Fatal(err)
	}
	invoke(git, "init", "--quiet", "--object-format=sha1", "--initial-branch=data")
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(remote, "values.json"), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		invoke(git, "add", "--", "values.json")
	}
	commit := func(sign bool) string {
		t.Helper()
		args := []string{"-c", "user.name=Synthetic Fixture", "-c", "user.email=fixture@example.invalid", "-c", "gpg.format=ssh", "-c", "user.signingkey=" + key, "commit", "--quiet", "-m", "synthetic data"}
		if sign {
			args = append(args, "-S")
		} else {
			args = append(args, "--no-gpg-sign")
		}
		invoke(git, args...)
		return invoke(git, "rev-parse", "HEAD")
	}
	write(`{"value":"synthetic-first"}`)
	first := commit(true)
	token := filepath.Join(root, "token")
	if err = os.WriteFile(token, []byte("synthetic-fetch-token"), 0600); err != nil {
		t.Fatal(err)
	}
	repo := model.Repository{URL: "https://example.invalid/synthetic.git", Directory: filepath.Join(root, "cache"), Ref: "refs/heads/data", AllowedSignersFile: signers, InitialCommit: first, TokenFile: token}
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	runGit = func(ctx context.Context, environment []string, limit int64, args ...string) ([]byte, error) {
		// Only this test transport permits a local fixture. Production allows HTTPS
		// exclusively and never accepts an environment-controlled remote override.
		for i, arg := range args {
			if arg == repo.URL {
				args[i] = remote
			}
		}
		args = append([]string{"-c", "protocol.file.allow=always"}, args...)
		return executeGit(ctx, environment, limit, args...)
	}
	acceptedDir, err := Fetch(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(token); err != nil {
		t.Fatal(err)
	}
	if current, err := Current(repo); err != nil || current != acceptedDir {
		t.Fatalf("offline verification failed without fetch token: %v", err)
	}
	if err = os.WriteFile(token, []byte("synthetic-fetch-token"), 0600); err != nil {
		t.Fatal(err)
	}
	write(`{"value":"synthetic-second"}`)
	second := commit(true)
	secondDir, err := Fetch(context.Background(), repo)
	if err != nil || filepath.Base(secondDir) != second {
		t.Fatalf("signed descendant rejected: %v", err)
	}
	write(`{"value":"synthetic-unsigned"}`)
	commit(false)
	if _, err = Fetch(context.Background(), repo); !fault.Is(err, fault.Trust) {
		t.Fatalf("unsigned descendant accepted: %v", err)
	}
	if current, err := Current(repo); err != nil || current != secondDir {
		t.Fatal("failed fetch replaced accepted pointer")
	}
	invoke(git, "reset", "--hard", first)
	if _, err = Fetch(context.Background(), repo); !fault.Is(err, fault.Trust) {
		t.Fatal("signed rollback accepted")
	}
	if err = os.WriteFile(filepath.Join(secondDir, "values.json"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Current(repo); !fault.Is(err, fault.Trust) {
		t.Fatal("tampered materialization accepted")
	}
}
