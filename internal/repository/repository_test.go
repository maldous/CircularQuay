package repository

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
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
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
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
	for _, change := range []func(string, map[string]string){
		func(directory string, _ map[string]string) {
			_ = os.WriteFile(filepath.Join(directory, "values.json"), []byte("changed"), 0600)
		},
		func(directory string, _ map[string]string) {
			_ = os.Chmod(filepath.Join(directory, "values.json"), 0700)
		},
		func(_ string, tree map[string]string) { tree["missing.json"] = blobID([]byte("missing"), 40) },
	} {
		directory, tree := materializedReleaseFixture(t, map[string]string{"values.json": "synthetic"})
		change(directory, tree)
		if err := verifyRelease(directory, tree); !fault.Is(err, fault.Trust) {
			t.Fatal("changed materialization accepted")
		}
	}
}

func TestVerifyReleaseRejectsInsecureDirectoryAndWalkError(t *testing.T) {
	directory, tree := materializedReleaseFixture(t, map[string]string{"values.json": "synthetic"})
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := verifyRelease(directory, tree); !fault.Is(err, fault.Trust) {
		t.Fatal("public release directory accepted")
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
	token := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(token, []byte("synthetic-token"), 0600); err != nil {
		t.Fatal(err)
	}
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

func materializedReleaseFixture(t *testing.T, files map[string]string) (string, map[string]string) {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	tree := make(map[string]string, len(files))
	for name, content := range files {
		path := filepath.Join(directory, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		tree[name] = blobID([]byte(content), 40)
	}
	return directory, tree
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
	fixture := signedRepositoryFixture{root: root, remote: filepath.Join(root, "remote"), git: git, key: filepath.Join(root, "signing"), signers: filepath.Join(root, "allowed-signers"), token: filepath.Join(root, "token")}
	fixture.environment = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + root, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_TERMINAL_PROMPT=0"}
	if err = os.Mkdir(fixture.remote, 0700); err != nil {
		t.Fatal(err)
	}
	fixture.invoke(t, keygen, "-q", "-t", "ed25519", "-N", "", "-f", fixture.key)
	publicKey, err := os.ReadFile(fixture.key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(fixture.signers, append([]byte("fixture "), publicKey...), 0600); err != nil {
		t.Fatal(err)
	}
	fixture.invoke(t, git, "init", "--quiet", "--object-format=sha1", "--initial-branch=data")
	fixture.writeToken(t)
	return fixture
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
	if err := os.WriteFile(f.token, []byte("synthetic-fetch-token"), 0600); err != nil {
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
	directory, err := Fetch(context.Background(), repo)
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
	if _, err := Fetch(context.Background(), repo); !fault.Is(err, fault.Trust) {
		t.Fatalf("%s accepted: %v", name, err)
	}
}
