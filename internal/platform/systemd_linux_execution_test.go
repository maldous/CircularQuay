//go:build linux && !android

package platform

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestApplySystemdCommandContractAndPublication(t *testing.T) {
	directory := privateSystemdTestDirectory(t)
	path := filepath.Join(directory, "APP_SECRET")
	plain := []byte("synthetic-input")
	ciphertext := []byte("synthetic-encrypted-output")
	if err := WriteProtected(path, []byte("old-projection")); err != nil {
		t.Fatal(err)
	}
	err := applySystemdWithRunner(context.Background(), model.Target{Path: path}, plain, systemdSuccessRunner(t, plain, ciphertext))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadProtected(path, true)
	if err != nil || !reflect.DeepEqual(got, ciphertext) {
		t.Fatalf("published output = %q, %v", got, err)
	}
}

func systemdSuccessRunner(t *testing.T, plain, ciphertext []byte) systemCommandRunner {
	t.Helper()
	return func(ctx context.Context, command systemCommand) error {
		assertSystemdCommand(t, ctx, command, plain)
		_, err := command.stdout.Write(ciphertext)
		return err
	}
}

func assertSystemdCommand(t *testing.T, ctx context.Context, command systemCommand, plain []byte) {
	t.Helper()
	if command.path != "/usr/bin/systemd-creds" || !reflect.DeepEqual(command.args, []string{"encrypt", "--name=APP_SECRET", "-", "-"}) || len(command.env) != 0 {
		t.Fatalf("command = %#v", command)
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("missing deadline")
	}
	input, err := io.ReadAll(command.stdin)
	if err != nil || !reflect.DeepEqual(input, plain) {
		t.Fatalf("stdin = %q, %v", input, err)
	}
}

func TestApplySystemdRejectsInvalidNameBeforeRunner(t *testing.T) {
	called := false
	err := applySystemdWithRunner(context.Background(), model.Target{Path: "relative"}, []byte("synthetic"), func(context.Context, systemCommand) error {
		called = true
		return nil
	})
	if !fault.Is(err, fault.Invalid) || called {
		t.Fatalf("invalid path = %v, called %t", err, called)
	}
}

func TestApplySystemdFailurePreservesExistingProjection(t *testing.T) {
	directory := privateSystemdTestDirectory(t)
	path := filepath.Join(directory, "APP_SECRET")
	if err := WriteProtected(path, []byte("old")); err != nil {
		t.Fatal(err)
	}
	err := applySystemdWithRunner(context.Background(), model.Target{Path: path}, []byte("new"), func(context.Context, systemCommand) error { return errors.New("synthetic failure") })
	if !fault.Is(err, fault.Apply) {
		t.Fatalf("runner failure = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "old" {
		t.Fatalf("projection after failure = %q, %v", got, err)
	}
}

func TestApplySystemdEmptyOutputPreservesExistingProjection(t *testing.T) {
	directory := privateSystemdTestDirectory(t)
	path := filepath.Join(directory, "APP_SECRET")
	if err := WriteProtected(path, []byte("old")); err != nil {
		t.Fatal(err)
	}
	err := applySystemdWithRunner(context.Background(), model.Target{Path: path}, []byte("new"), func(context.Context, systemCommand) error { return nil })
	if !fault.Is(err, fault.Apply) {
		t.Fatalf("empty output = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "old" {
		t.Fatalf("projection after empty output = %q, %v", got, err)
	}
}

func TestApplySystemdBoundsOutputBeforePublication(t *testing.T) {
	directory := privateSystemdTestDirectory(t)
	path := filepath.Join(directory, "APP_SECRET")
	err := applySystemdWithRunner(context.Background(), model.Target{Path: path}, []byte("synthetic"), func(_ context.Context, command systemCommand) error {
		_, err := command.stdout.Write(make([]byte, MaxDocumentSize+1))
		return err
	})
	if !fault.Is(err, fault.Apply) {
		t.Fatalf("oversize output = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("projection exists after oversize output: %v", err)
	}
}

func TestBoundedSystemdOutputCannotBypassLimitWithIOCopy(t *testing.T) {
	output := &boundedSystemdOutput{}
	n, err := io.Copy(output, io.LimitReader(zeroReader{}, MaxDocumentSize+1))
	if n != MaxDocumentSize || !errors.Is(err, errSystemdOutputTooLarge) || output.buffer.Len() != MaxDocumentSize {
		t.Fatalf("copy result = %d, %v, %d bytes", n, err, output.buffer.Len())
	}
}

func TestSystemdCommandBoundsOrphanedOutputPipe(t *testing.T) {
	start := time.Now()
	err := runSystemdCommand(context.Background(), systemCommand{
		path: os.Args[0], args: []string{"-test.run=^TestSystemdCommandHelper$"},
		env: []string{"CQUAY_SYSTEMD_HELPER=parent"}, stdout: io.Discard, stderr: io.Discard,
	})
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("orphaned output result = %v", err)
	}
	if elapsed := time.Since(start); elapsed < systemdWaitDelay || elapsed > 5*systemdWaitDelay {
		t.Fatalf("wait delay elapsed = %v", elapsed)
	}
}

func TestSystemdCommandHelper(t *testing.T) {
	switch os.Getenv("CQUAY_SYSTEMD_HELPER") {
	case "child":
		writeUntilPipeClosed(10 * systemdWaitDelay)
	case "parent":
		child := exec.Command(os.Args[0], "-test.run=^TestSystemdCommandHelper$")
		child.Env = []string{"CQUAY_SYSTEMD_HELPER=child"}
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
	}
}

func writeUntilPipeClosed(limit time.Duration) {
	deadline := time.Now().Add(limit)
	data := make([]byte, 4096)
	for time.Now().Before(deadline) {
		if _, err := os.Stdout.Write(data); err != nil {
			return
		}
	}
}

type zeroReader struct{}

func (zeroReader) Read(data []byte) (int, error) {
	for index := range data {
		data[index] = 0
	}
	return len(data), nil
}

func privateSystemdTestDirectory(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}
