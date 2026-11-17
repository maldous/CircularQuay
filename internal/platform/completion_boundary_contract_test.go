package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func completionFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	if !fault.Is(err, kind) || err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatalf("unexpected bounded fault: %v", err)
	}
}

func completionFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(secureTempDir(t), name)
	if err := WriteProtected(path, []byte("owned")); err != nil {
		t.Fatal(err)
	}
	return path
}

func completionOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func completionClosed(t *testing.T, f *os.File) {
	t.Helper()
	if n, err := f.Write([]byte("must not write")); n != 0 || !errors.Is(err, os.ErrClosed) {
		t.Fatal("descriptor remains writable")
	}
}

func TestCompletionDirectoryProtectionRefusal(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "new")
	calls := 0
	err := ensurePrivateDirectoryWithProtection(path, func(path string, created bool) error {
		if !created {
			t.Fatal("new directory not detected")
		}
		return protectNewDirectoryWithOperation(path, created, func(got string) error {
			calls++
			if info, err := os.Stat(got); err != nil || !info.IsDir() {
				t.Fatal("protection preceded creation")
			}
			return os.ErrPermission
		})
	})
	completionFault(t, err, fault.Apply, "protect private directory")
	if calls != 1 {
		t.Fatal("protection call count")
	}
}

func TestCompletionReceiptLifecycleRefusals(t *testing.T) {
	for _, phase := range []string{"identity", "write", "sync"} {
		t.Run(phase, func(t *testing.T) {
			completionReceiptPhase(t, phase)
		})
	}
}

func completionReceiptPhase(t *testing.T, phase string) {
	path := completionFile(t, "receipt")
	ops := defaultPlatformBoundaryOperations()
	var file *os.File
	ops.receipt = func(name string) (*os.File, error) {
		var err error
		file, err = openReceiptLog(name)
		if err == nil && phase == "identity" {
			_ = file.Close()
		}
		return file, err
	}
	ops.write = func(f *os.File, data []byte) (int, error) {
		if phase == "write" {
			return 0, os.ErrPermission
		}
		return f.Write(data)
	}
	ops.sync = func(f *os.File) error { return os.ErrPermission }
	err := appendProtectedWithOperations(path, []byte(`{"owned":true}`), ops)
	if phase == "identity" {
		completionFault(t, err, fault.Trust, "receipt log changed")
	} else {
		completionFault(t, err, fault.Apply, phase+" receipt")
	}
	completionClosed(t, file)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "owned"
	if phase == "sync" { want += "{\"owned\":true}\n" }
	if string(data) != want {
		t.Fatal("unexpected committed receipt bytes")
	}
}

func TestCompletionReceiptOpenAndLockReopenRefusals(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "new")
	ops := defaultPlatformBoundaryOperations()
	var opened *os.File
	ops.open = func(path string, flags int, mode os.FileMode) (*os.File, error) {
		var err error
		opened, err = openInParent(path, flags, mode)
		return opened, err
	}
	ops.protect = func(string) error { return os.ErrPermission }
	f, err := openReceiptLogWithOperations(path, ops)
	if f != nil {
		t.Fatal("failed receipt escaped")
	}
	completionFault(t, err, fault.Apply, "protect receipt log")
	completionClosed(t, opened)
	path = completionFile(t, "existing")
	calls := 0
	open := func(name string, flags int, mode os.FileMode) (*os.File, error) {
		calls++
		if flags&os.O_EXCL != 0 {
			return openInParent(name, flags, mode)
		}
		return nil, os.ErrPermission
	}
	ops = defaultPlatformBoundaryOperations()
	ops.open = open
	f, err = openReceiptLogWithOperations(path, ops)
	if f != nil || calls != 2 {
		t.Fatal("receipt reopen ordering")
	}
	completionFault(t, err, fault.Apply, "open receipt log")
	calls = 0
	f, created, err := openOrCreateLockWithOperation(path, open)
	if f != nil || created || calls != 2 {
		t.Fatal("lock reopen ordering")
	}
	completionFault(t, err, fault.Apply, "open lock file")
}

func TestCompletionProtectedOpenRefusals(t *testing.T) {
	path := completionFile(t, "protected")
	f, err := openProtectedWithOperation(path, true, func(string, int, os.FileMode) (*os.File, error) { return nil, os.ErrPermission })
	if f != nil {
		t.Fatal("failed open returned descriptor")
	}
	completionFault(t, err, fault.Denied, "open protected file")
	other := completionOpen(t, completionFile(t, "other"))
	f, err = openProtectedWithOperation(path, true, func(string, int, os.FileMode) (*os.File, error) { return other, nil })
	if f != nil {
		t.Fatal("wrong identity escaped")
	}
	completionFault(t, err, fault.Trust, "protected file changed")
	completionClosed(t, other)
}

func TestCompletionStageAndActivationSyncRefusals(t *testing.T) {
	dir := secureTempDir(t)
	name, err := stageProtectedFileWithProtection(dir, []byte("owned"), writeAndSync, func(string) error { return os.ErrPermission })
	if name != "" {
		t.Fatal("failed stage returned name")
	}
	completionFault(t, err, fault.Apply, "protect staging ACL")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("failed stage leaked file")
	}
	name, err = stageProtectedFile(dir, []byte("new"))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "target")
	err = activateStagedFileWithSync(name, target, dir, func(got string) error {
		if got != dir {
			t.Fatal("wrong sync directory")
		}
		return os.ErrPermission
	})
	completionFault(t, err, fault.Apply, "sync destination directory")
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "new" {
		t.Fatal("rename effect incorrectly hidden after sync failure")
	}
}

func TestCompletionLockProtectionAndIdentity(t *testing.T) {
	for _, phase := range []string{"protect", "identity"} {
		t.Run(phase, func(t *testing.T) {
			completionLockPhase(t, phase)
		})
	}
}

func completionLockPhase(t *testing.T, phase string) {
	path := filepath.Join(secureTempDir(t), "lock")
	ops := defaultPlatformBoundaryOperations()
	var opened *os.File
	ops.lock = func(name string) (*os.File, bool, error) {
		file, created, err := openOrCreateLock(name)
		opened = file
		if err == nil && phase == "identity" {
			_ = file.Close()
		}
		return file, created, err
	}
	if phase == "protect" { ops.protect = func(string) error { return os.ErrPermission } }
	file, err := openLockFileWithOperations(path, ops)
	if file != nil {
		t.Fatal("failed lock escaped")
	}
	if phase == "protect" {
		completionFault(t, err, fault.Apply, "protect lock file")
	} else {
		completionFault(t, err, fault.Trust, "lock file changed")
	}
	completionClosed(t, opened)
}

func TestCompletionExecutableReadAndStableIdentity(t *testing.T) {
	for _, phase := range []string{"read", "changed"} {
		t.Run(phase, func(t *testing.T) {
			completionExecutablePhase(t, phase)
		})
	}
}

func completionExecutablePhase(t *testing.T, phase string) {
	path := completionFile(t, "inert")
	ops := defaultExecutableBoundaryOperations()
	ops.digest = func(reader io.Reader, maximum int64) ([]byte, bool, error) {
		if phase == "read" {
			return nil, false, os.ErrPermission
		}
		digest, exceeded, err := boundedExecutableDigest(reader, maximum)
		if writeErr := os.WriteFile(path, []byte("changed-length"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		return digest, exceeded, err
	}
	sum := sha256.Sum256([]byte("owned"))
	err := verifyExecutableWithOperations(path, hex.EncodeToString(sum[:]), ops)
	operation := executableSizeOperation
	if phase == "changed" {
		operation = "registered executable changed"
	}
	completionFault(t, err, fault.Trust, operation)
}

func TestCompletionExecutableAdmissionLateRefusals(t *testing.T) {
	path := completionFile(t, "inert")
	resolve := func(string) (string, error) { return "", os.ErrNotExist }
	f, name, info, err := openVerifiedExecutableWithOperations(path, resolve, openInParent)
	if f != nil || name != "" || info != nil {
		t.Fatal("failed resolve retained output")
	}
	completionFault(t, err, fault.Trust, "registered executable unavailable")
	f, name, info, err = openVerifiedExecutableWithOperations(path, filepath.EvalSymlinks, func(string, int, os.FileMode) (*os.File, error) { return nil, os.ErrPermission })
	if f != nil || name != "" || info != nil {
		t.Fatal("failed open retained output")
	}
	completionFault(t, err, fault.Trust, "registered executable unavailable")
	closed := completionOpen(t, path)
	_ = closed.Close()
	f, name, info, err = openVerifiedExecutableWithOperations(path, filepath.EvalSymlinks, func(string, int, os.FileMode) (*os.File, error) { return closed, nil })
	if f != nil || name != "" || info != nil {
		t.Fatal("failed stat retained output")
	}
	completionFault(t, err, fault.Trust, "registered executable permissions")
}

func TestCompletionCancellationAfterActualAdmission(t *testing.T) {
	path := completionFile(t, "inert")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	prepare := func(work model.Workload, gen model.Generation, override []string) ([]string, []string, string, error) {
		args, env, dir, err := launchInputs(work, gen, override)
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		return args, env, dir, err
	}
	code, err := runWithIOOperations(ctx, model.Workload{Command: []string{path}}, model.Generation{}, nil, nil, io.Discard, io.Discard, prepare, func(*exec.Cmd) error { t.Fatal("canceled workload reached execution"); return nil })
	if code != 0 || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation checkpoint changed")
	}
}

func TestCompletionCreateRefusalCleanup(t *testing.T) {
	for _, phase := range []string{"stat", "protect", "identity", "sync"} {
		t.Run(phase, func(t *testing.T) {
			completionCreatePhase(t, phase)
		})
	}
}

func completionCreatePhase(t *testing.T, phase string) {
	path := filepath.Join(secureTempDir(t), "created")
	ops := completionCreationOperations(phase)
	err := createProtectedExclusiveWithOperations(path, []byte("new"), ops)
	operation := map[string]string{"stat": "inspect created file", "protect": "protect created file", "identity": "created file changed", "sync": "sync created file directory"}[phase]
	kind := fault.Apply
	if phase == "identity" {
		kind = fault.Trust
	}
	completionFault(t, err, kind, operation)
	_, exists := os.Lstat(path)
	if phase == "stat" {
		if exists != nil {
			t.Fatal("pre-identity failure semantics changed")
		}
	} else if !os.IsNotExist(exists) {
		t.Fatal("failed creation retained owned name")
	}
}

func TestCompletionCreationRootLateRefusals(t *testing.T) {
	dir := secureTempDir(t)
	root, err := openCreationRootWithOperation(dir, func(string) (*os.Root, error) { return nil, os.ErrPermission })
	if root != nil {
		t.Fatal("failed root escaped")
	}
	completionFault(t, err, fault.Apply, "open protected creation directory")
	root, err = openCreationRootWithOperation(dir, func(path string) (*os.Root, error) {
		opened, err := os.OpenRoot(path)
		if err == nil {
			_ = opened.Close()
		}
		return opened, err
	})
	if root != nil {
		t.Fatal("closed root escaped")
	}
	completionFault(t, err, fault.Trust, "creation directory changed")
}

func completionSparseSize(t *testing.T, path string, size int64) {
	t.Helper()
	file := completionOpen(t, path)
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCompletionExecutableActualSizeBound(t *testing.T) {
	path := completionFile(t, "bounded-inert")
	completionSparseSize(t, path, maxVerifiedExecutableBytes+1)
	sum := sha256.Sum256([]byte("owned"))
	err := VerifyExecutable(path, hex.EncodeToString(sum[:]))
	completionFault(t, err, fault.Trust, executableSizeOperation)
	info, err := os.Stat(path)
	if err != nil || info.Size() != maxVerifiedExecutableBytes+1 {
		t.Fatal("bounded refusal changed the owned file")
	}
}

func TestCompletionExecutableActualGrowthBound(t *testing.T) {
	path := completionFile(t, "growing-inert")
	ops := defaultExecutableBoundaryOperations()
	var opened *os.File
	ops.open = func(name string) (*os.File, string, os.FileInfo, error) {
		file, resolved, before, err := openVerifiedExecutable(name)
		if err == nil {
			opened = file
			completionSparseSize(t, name, maxVerifiedExecutableBytes+1)
		}
		return file, resolved, before, err
	}
	sum := sha256.Sum256([]byte("owned"))
	err := verifyExecutableWithOperations(path, hex.EncodeToString(sum[:]), ops)
	completionFault(t, err, fault.Trust, executableSizeOperation)
	if opened == nil {
		t.Fatal("growth fixture never admitted")
	}
	if _, err := opened.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
		t.Fatal("bounded digest descriptor not closed")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != maxVerifiedExecutableBytes+1 {
		t.Fatal("growth refusal changed the owned file")
	}
}

func completionCreationOperations(phase string) creationBoundaryOperations {
	ops := defaultCreationBoundaryOperations()
	ops.open = func(root *os.Root, name string, flags int, mode os.FileMode) (*os.File, error) {
		file, err := root.OpenFile(name, flags, mode)
		if err == nil && phase == "stat" {
			_ = file.Close()
		}
		return file, err
	}
	ops.populate = func(file *os.File, name string, data []byte) error {
		if phase == "protect" { return populateCreatedFileWithProtection(file, name, data, func(string) error { return os.ErrPermission }) }
		if phase == "identity" {
			_ = file.Close()
		}
		return populateCreatedFile(file, name, data)
	}
	ops.sync = func(string) error { return os.ErrPermission }
	return ops
}
