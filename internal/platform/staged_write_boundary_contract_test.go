package platform

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestStagedWriteBoundaryContractFailureOrderAndCloseOnce(t *testing.T) {
	for _, test := range []struct {
		phase     string
		operation string
		events    []string
	}{
		{"chmod", "protect staged file", []string{"chmod", "close"}},
		{"write", "write staged file", []string{"chmod", "write", "close"}},
		{"sync", "sync staged file", []string{"chmod", "write", "sync", "close"}},
		{"close", "close staged file", []string{"chmod", "write", "sync", "close"}},
	} {
		t.Run(test.phase, func(t *testing.T) {
			fixture := &stagedWriteBoundaryHandle{failure: test.phase}
			data := []byte("owned synthetic staged bytes")
			err := writeAndSyncHandle(fixture, data)
			assertStagedWriteBoundaryFault(t, err, test.operation)
			assertStagedWriteBoundaryLifecycle(t, fixture, test.events, data)
		})
	}
}

func TestStagedWriteBoundaryContractPositiveControl(t *testing.T) {
	fixture := &stagedWriteBoundaryHandle{}
	data := []byte("owned synthetic staged bytes")
	if err := writeAndSyncHandle(fixture, data); err != nil {
		t.Fatal("complete finite staged lifecycle failed")
	}
	assertStagedWriteBoundaryLifecycle(t, fixture, []string{"chmod", "write", "sync", "close"}, data)
}

func TestStagedWriteBoundaryContractProductionFileDefaults(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "owned-stage-control")
	data := []byte("actual owned protected file")
	if err := WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
	got, err := ReadProtected(path, true)
	if err != nil || !reflect.DeepEqual(got, data) {
		t.Fatal("ordinary protected file writer changed the real output")
	}
}

type stagedWriteBoundaryHandle struct {
	failure string
	events  []string
	mode    os.FileMode
	data    []byte
	closes  int
}

func (handle *stagedWriteBoundaryHandle) Chmod(mode os.FileMode) error {
	handle.events = append(handle.events, "chmod")
	handle.mode = mode
	return handle.failureAt("chmod")
}

func (handle *stagedWriteBoundaryHandle) Write(data []byte) (int, error) {
	handle.events = append(handle.events, "write")
	handle.data = append([]byte(nil), data...)
	if err := handle.failureAt("write"); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (handle *stagedWriteBoundaryHandle) Sync() error {
	handle.events = append(handle.events, "sync")
	return handle.failureAt("sync")
}

func (handle *stagedWriteBoundaryHandle) Close() error {
	handle.events = append(handle.events, "close")
	handle.closes++
	return handle.failureAt("close")
}

func (handle *stagedWriteBoundaryHandle) failureAt(phase string) error {
	if handle.failure == phase {
		return errors.New("synthetic staged lifecycle failure")
	}
	return nil
}

func assertStagedWriteBoundaryFault(t *testing.T, err error, operation string) {
	t.Helper()
	if !fault.Is(err, fault.Apply) || err.Error() != string(fault.Apply)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatal("staged lifecycle exposed a raw error or lost its exact safe operation")
	}
}

func assertStagedWriteBoundaryLifecycle(t *testing.T, handle *stagedWriteBoundaryHandle, events []string, data []byte) {
	t.Helper()
	if !reflect.DeepEqual(handle.events, events) || handle.closes != 1 || handle.mode != 0o600 {
		t.Fatal("staged lifecycle ran out of order, repeated close or changed the protection mode")
	}
	if handle.failure != "chmod" && !reflect.DeepEqual(handle.data, data) {
		t.Fatal("staged write changed the supplied bytes")
	}
	if string(data) != "owned synthetic staged bytes" {
		t.Fatal("staged lifecycle mutated caller-owned input")
	}
}
