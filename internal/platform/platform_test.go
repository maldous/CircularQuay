package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "--platform-helper" {
		if os.Getenv("PARENT_SECRET") != "" || os.Getenv("SOPS_AGE_KEY") != "" {
			os.Exit(90)
		}
		if len(os.Args) > 2 && os.Args[2] == "exit" {
			os.Exit(7)
		}
		if os.Getenv("APP_SECRET") != "synthetic-✓-$()\nline" {
			os.Exit(91)
		}
		_, _ = io.WriteString(os.Stdout, "allowed-status")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func fixtureGeneration() model.Generation {
	return model.Generation{Workload: "demo", Values: map[string]model.Value{"APP_SECRET": {Data: "synthetic-✓-$()\nline"}}}
}

func TestRunNoInheritanceAndPreservesArguments(t *testing.T) {
	t.Setenv("PARENT_SECRET", "must-not-reach-child")
	t.Setenv("SOPS_AGE_KEY", "must-not-reach-child")
	exe, directory := protectedPlatformHelper(t)
	w := model.Workload{Command: []string{exe, "--platform-helper", "$(not-executed)", "spaces and Unicode-✓"}, Directory: directory}
	var out bytes.Buffer
	code, err := RunWithIO(context.Background(), w, fixtureGeneration(), nil, nil, &out, io.Discard)
	if err != nil || code != 0 || out.String() != "allowed-status" {
		t.Fatalf("isolated execution = code %d, error %v, output %q", code, err, out.String())
	}
	code, err = RunWithIO(context.Background(), w, fixtureGeneration(), []string{exe, "--platform-helper", "exit"}, nil, io.Discard, io.Discard)
	if err != nil || code != 7 {
		t.Fatal("child exit status lost")
	}
}

func TestRunRejectsExpiredGenerationBeforeLaunch(t *testing.T) {
	exe, directory := protectedPlatformHelper(t)
	gen := fixtureGeneration()
	value := gen.Values["APP_SECRET"]
	value.ExpiresAt = time.Now().Add(-time.Second)
	gen.Values["APP_SECRET"] = value
	var out bytes.Buffer
	_, err := RunWithIO(context.Background(), model.Workload{Command: []string{exe, "--platform-helper"}, Directory: directory}, gen, nil, nil, &out, io.Discard)
	if !fault.Is(err, fault.Expired) || out.Len() != 0 {
		t.Fatal("expired generation reached child launch")
	}
}

// protectedPlatformHelper copies this test binary beneath test-owned protected
// directories. Windows validates every DLL search directory before exposing a
// generation, so the Go test runner's working and build directories are not a
// valid positive fixture for a secret-bearing child process.
func protectedPlatformHelper(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(secureTempDir(t), "helper")
	bin := filepath.Join(root, "bin")
	work := filepath.Join(root, "work")
	for _, directory := range []string{bin, work} {
		if err := EnsurePrivateDirectory(directory); err != nil {
			t.Fatal(err)
		}
	}
	target := copyPlatformTestBinary(t, bin)
	if err := ValidateExecutable(target); err != nil {
		t.Fatalf("protected helper rejected: %v", err)
	}
	return target, work
}

func copyPlatformTestBinary(t *testing.T, bin string) string {
	t.Helper()
	sourcePath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	target := filepath.Join(bin, "platform-helper"+filepath.Ext(sourcePath))
	destination, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(destination, source); err != nil {
		_ = destination.Close()
		t.Fatal(err)
	}
	if err = destination.Close(); err != nil {
		t.Fatal(err)
	}
	if err = protectFile(target); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(target, 0o700); err != nil {
		t.Fatal(err)
	}
	return target
}

func TestInvalidLaunchAndEnvironment(t *testing.T) {
	for _, args := range [][]string{nil, {"relative"}, {filepath.Join(secureTempDir(t), "script.cmd")}, {"/bin/echo", "bad\x00arg"}} {
		if _, err := RunWithIO(context.Background(), model.Workload{Command: args}, fixtureGeneration(), nil, nil, io.Discard, io.Discard); !fault.Is(err, fault.Invalid) {
			t.Fatal("unsafe executable accepted")
		}
	}
	gen := fixtureGeneration()
	if _, err := Environment(map[string]string{"app_secret": "collision"}, gen); !fault.Is(err, fault.Invalid) {
		t.Fatal("case collision accepted")
	}
	gen.Values = map[string]model.Value{"SOPS_AGE_KEY": {Data: "forbidden"}}
	if _, err := Environment(nil, gen); !fault.Is(err, fault.Trust) {
		t.Fatal("bootstrap injection accepted")
	}
	gen.Values = map[string]model.Value{"KEY": {Data: "NUL\x00value"}}
	if _, err := Environment(nil, gen); !fault.Is(err, fault.Invalid) {
		t.Fatal("NUL value accepted")
	}
	if _, err := Environment(map[string]string{"BAD=NAME": "value"}, fixtureGeneration()); !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid name accepted")
	}
	if _, err := Environment(map[string]string{"PATH": "/approved/bin"}, fixtureGeneration()); err != nil {
		t.Fatal("trusted nonsecret PATH rejected")
	}
}

func TestProtectedAtomicDelivery(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "private", "values.json")
	if err := Apply(context.Background(), model.Target{Type: "file", Path: path}, fixtureGeneration()); err != nil {
		t.Fatal(err)
	}
	data, err := ReadProtected(path, true)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]string
	if json.Unmarshal(data, &values) != nil || values["APP_SECRET"] != fixtureGeneration().Values["APP_SECRET"].Data {
		t.Fatal("JSON delivery changed literal value")
	}
	if err := WriteProtected(path, []byte(`{"KEY":"next"}`)); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".cquay-stage-*"))
	if err != nil || len(files) != 0 {
		t.Fatal("staging file retained")
	}
}

func TestStageProtectedFileRemovesFailedStage(t *testing.T) {
	dir := secureTempDir(t)
	_, err := stageProtectedFileWithWriter(dir, []byte("synthetic"), func(_ *os.File, _ []byte) error {
		return errors.New("synthetic write failure")
	})
	if err == nil {
		t.Fatal("failed stage write accepted")
	}
	files, err := filepath.Glob(filepath.Join(dir, ".cquay-stage-*"))
	if err != nil || len(files) != 0 {
		t.Fatal("failed stage file retained")
	}
}

func TestProtectedFileRejections(t *testing.T) {
	testRelativeAndDirectoryRejections(t)
	dir := secureTempDir(t)
	testMissingAndOversizeRejections(t, dir)
	if runtime.GOOS != "windows" {
		testUnixFileRejections(t, dir)
	}
}

func TestOpenProtectedReturnsVerifiedDescriptor(t *testing.T) {
	dir := secureTempDir(t)
	path := filepath.Join(dir, "source")
	if err := os.WriteFile(path, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := OpenProtected(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil || string(data) != "synthetic" {
		t.Fatalf("verified descriptor read = %q, %v", data, err)
	}
}

func testRelativeAndDirectoryRejections(t *testing.T) {
	t.Helper()
	if _, err := ReadProtected("relative", true); !fault.Is(err, fault.Trust) {
		t.Fatal("relative source accepted")
	}
	if err := WriteProtected("relative", nil); !fault.Is(err, fault.Invalid) {
		t.Fatal("relative destination accepted")
	}
	dir := secureTempDir(t)
	if _, err := ReadProtected(dir, true); !fault.Is(err, fault.Trust) {
		t.Fatal("directory source accepted")
	}
	if err := WriteProtected(dir, nil); !fault.Is(err, fault.Trust) {
		t.Fatal("directory replaced")
	}
	if err := ValidateProtectedPath(filepath.Join(dir, "missing")); !fault.Is(err, fault.Trust) {
		t.Fatal("missing protected path accepted")
	}
}

func testMissingAndOversizeRejections(t *testing.T, dir string) {
	t.Helper()
	if _, err := ReadProtected(filepath.Join(dir, "missing"), true); !fault.Is(err, fault.Missing) {
		t.Fatal("missing source misclassified")
	}
	if err := WriteProtected(filepath.Join(dir, "large"), make([]byte, MaxDocumentSize+1)); !fault.Is(err, fault.Invalid) {
		t.Fatal("large destination accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "large"), make([]byte, MaxDocumentSize+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProtected(filepath.Join(dir, "large"), false); !fault.Is(err, fault.Invalid) {
		t.Fatal("large source accepted")
	}
	if err := AppendProtected(filepath.Join(dir, "nested", "audit.jsonl"), []byte(`{"action":"synthetic"}`)); err != nil {
		t.Fatal("missing private receipt parent was not established")
	}
}

func testUnixFileRejections(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "public")
	if err := os.WriteFile(path, []byte("synthetic"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProtected(path, true); !fault.Is(err, fault.Trust) {
		t.Fatal("public bootstrap file accepted")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProtected(link, false); !fault.Is(err, fault.Trust) {
		t.Fatal("symlink read accepted")
	}
	if err := WriteProtected(link, nil); !fault.Is(err, fault.Trust) {
		t.Fatal("symlink replaced")
	}
}

func TestNativeUnsupportedAndInvalidValues(t *testing.T) {
	if err := Apply(context.Background(), model.Target{Type: "other"}, fixtureGeneration()); !fault.Is(err, fault.Unsupported) {
		t.Fatal("unknown target accepted")
	}
	if runtime.GOOS != "windows" {
		if _, err := ReadDPAPI("unused"); !fault.Is(err, fault.Unsupported) {
			t.Fatal("DPAPI advertised on unsupported platform")
		}
		if err := Apply(context.Background(), model.Target{Type: "windows"}, fixtureGeneration()); !fault.Is(err, fault.Unsupported) {
			t.Fatal("DPAPI advertised on unsupported platform")
		}
	}
	if _, err := EncodeValues(model.Generation{Values: map[string]model.Value{"KEY": {Data: "bad\x00"}}}); !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid delivery encoded")
	}
	if _, err := EncodeValues(model.Generation{Values: map[string]model.Value{"KEY": {Data: strings.Repeat("x", MaxDocumentSize+1)}}}); !fault.Is(err, fault.Invalid) {
		t.Fatal("oversize delivery encoded")
	}
}

func TestProtectedAuditRecords(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "audit.jsonl")
	appendConcurrentProtectedAuditRecords(t, path)
	assertProtectedAuditRecords(t, path)
	assertInvalidAuditRecordsRejected(t, path)
	if err := ValidateProtectedPath(path); err != nil {
		t.Fatal("protected audit metadata rejected")
	}
}

func appendConcurrentProtectedAuditRecords(t *testing.T, path string) {
	t.Helper()
	var group sync.WaitGroup
	for range 20 {
		group.Go(func() {
			if err := AppendProtected(path, []byte("{\n\"action\":\"synthetic\"\n}")); err != nil {
				t.Error("protected append failed")
			}
		})
	}
	group.Wait()
}

func assertProtectedAuditRecords(t *testing.T, path string) {
	t.Helper()
	data, err := ReadProtected(path, true)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte{'\n'})
	if len(lines) != 20 {
		t.Fatal("receipts were concatenated, split or lost")
	}
	for _, line := range lines {
		if !json.Valid(line) {
			t.Fatal("invalid JSONL receipt")
		}
	}
}

func assertInvalidAuditRecordsRejected(t *testing.T, path string) {
	t.Helper()
	for _, invalid := range [][]byte{nil, []byte("null"), []byte("[]"), []byte("{}{}"), make([]byte, 65537)} {
		if err := AppendProtected(path, invalid); !fault.Is(err, fault.Invalid) {
			t.Fatal("invalid receipt accepted")
		}
	}
}

func TestPrivateDirectoryAndExecutableTrust(t *testing.T) {
	dir := filepath.Join(secureTempDir(t), "created")
	if err := EnsurePrivateDirectory(dir); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrivateDirectory("relative"); !fault.Is(err, fault.Invalid) {
		t.Fatal("relative state directory accepted")
	}
	testExecutableTrust(t)
	testExistingDirectoryPermissions(t, dir)
}

func testExecutableTrust(t *testing.T) {
	t.Helper()
	exe := testExecutableDigest(t)
	testUnixExecutableTrust(t, exe)
}

func testExecutableDigest(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if err := VerifyExecutable(exe, hex.EncodeToString(digest[:])); err != nil {
		t.Fatal("reviewed executable rejected")
	}
	if err := VerifyExecutable(exe, strings.Repeat("0", 64)); !fault.Is(err, fault.Trust) {
		t.Fatal("changed executable accepted")
	}
	if err := VerifyExecutable(exe, "bad"); !fault.Is(err, fault.Trust) {
		t.Fatal("invalid digest accepted")
	}
	if err := ValidateExecutable("relative"); !fault.Is(err, fault.Trust) {
		t.Fatal("relative helper accepted")
	}
	return exe
}

func testUnixExecutableTrust(t *testing.T, exe string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	assertTrustedExecutableAlias(t, exe)
	assertMutableExecutableAliasRejected(t)
}

func assertTrustedExecutableAlias(t *testing.T, exe string) {
	t.Helper()
	alias := filepath.Join(secureTempDir(t), "trusted-helper-link")
	if err := os.Symlink(exe, alias); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExecutable(alias); err != nil {
		t.Fatal("trusted executable alias rejected")
	}
}

func assertMutableExecutableAliasRejected(t *testing.T) {
	t.Helper()
	unsafe := filepath.Join(secureTempDir(t), "unsafe-helper")
	if err := os.WriteFile(unsafe, []byte("synthetic"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unsafe, 0o666); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(secureTempDir(t), "unsafe-helper-link")
	if err := os.Symlink(unsafe, link); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExecutable(link); !fault.Is(err, fault.Trust) {
		t.Fatal("symbolic executable to mutable target accepted")
	}
}

func secureTempDir(t *testing.T) string {
	t.Helper()
	base := os.TempDir()
	if runtime.GOOS != "windows" {
		// Test data must exercise the same immutable-ancestor rule as production.
		// Do not inherit an arbitrary TMPDIR when the standard root-owned sticky
		// directory is available.
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "cquay-platform-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func testExistingDirectoryPermissions(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := EnsurePrivateDirectory(dir); !fault.Is(err, fault.Trust) {
			t.Fatal("public existing state directory accepted")
		}
		link := filepath.Join(secureTempDir(t), "private-directory-link")
		if err := os.Symlink(secureTempDir(t), link); err != nil {
			t.Fatal(err)
		}
		if err := EnsurePrivateDirectory(link); !fault.Is(err, fault.Trust) {
			t.Fatal("symbolic private directory accepted")
		}
	}
}

func TestActivationDeclarationRejectsInjection(t *testing.T) {
	if err := Activate(context.Background(), model.Target{}); err != nil {
		t.Fatal("no activation should have no effect")
	}
	for _, target := range []model.Target{
		{Service: "demo.service"}, {Activation: "restart"},
		{Service: "-evil.service", Activation: "restart"},
		{Service: "demo.service;touch /tmp/invalid", Activation: "restart"},
		{Service: "demo.service", Activation: "arbitrary"},
	} {
		if err := Activate(context.Background(), target); !fault.Is(err, fault.Invalid) {
			t.Fatal("unsafe native activation accepted")
		}
	}
}
