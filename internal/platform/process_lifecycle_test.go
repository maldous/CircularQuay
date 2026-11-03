package platform

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

const lifecycleWaitMode = "mark-and-wait"

type runResult struct {
	code int
	err  error
}

func TestRunCancelledBeforeStartDoesNotLaunch(t *testing.T) {
	executable, directory := protectedPlatformHelper(t)
	marker := filepath.Join(secureTempDir(t), "started")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RunWithIO(ctx, lifecycleWorkload(executable, directory, lifecycleWaitMode, marker), fixtureGeneration(), nil, nil, io.Discard, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled launch error = %v", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled launch marker = %v", err)
	}
}

func TestRunCancellationAfterStartReturnsContextError(t *testing.T) {
	executable, directory := protectedPlatformHelper(t)
	marker := filepath.Join(secureTempDir(t), "started")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := startLifecycleRun(ctx, lifecycleWorkload(executable, directory, lifecycleWaitMode, marker), io.Discard)
	waitForHelperMarker(t, marker)
	cancel()
	assertContextResult(t, receiveLifecycleResult(t, result), context.Canceled)
}

func TestRunDeadlineAfterStartReturnsContextError(t *testing.T) {
	executable, directory := protectedPlatformHelper(t)
	marker := filepath.Join(secureTempDir(t), "started")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	result := startLifecycleRun(ctx, lifecycleWorkload(executable, directory, lifecycleWaitMode, marker), io.Discard)
	waitForHelperMarker(t, marker)
	assertContextResult(t, receiveLifecycleResultWithin(t, result, 10*time.Second), context.DeadlineExceeded)
}

func TestRunBoundsOrphanedOutputPipe(t *testing.T) {
	executable, directory := protectedPlatformHelper(t)
	marker := filepath.Join(secureTempDir(t), "started")
	completed := filepath.Join(secureTempDir(t), "completed")
	t.Cleanup(func() { waitForHelperMarkerWithin(t, completed, 15*time.Second) })
	started := time.Now()
	workload := model.Workload{Command: []string{executable, "--platform-helper", "orphan-pipe", marker, completed}, Directory: directory}
	code, err := RunWithIO(context.Background(), workload, fixtureGeneration(), nil, nil, &bytes.Buffer{}, io.Discard)
	if !fault.Is(err, fault.Apply) || code != 0 {
		t.Fatalf("orphaned pipe result = code %d, error %v", code, err)
	}
	if elapsed := time.Since(started); elapsed >= 5*time.Second {
		t.Fatalf("orphaned output pipe waited %s", elapsed)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("orphan helper did not start: %v", err)
	}
}

func lifecycleWorkload(executable, directory, mode, marker string) model.Workload {
	return model.Workload{Command: []string{executable, "--platform-helper", mode, marker}, Directory: directory}
}

func startLifecycleRun(ctx context.Context, workload model.Workload, stdout io.Writer) <-chan runResult {
	result := make(chan runResult, 1)
	go func() {
		code, err := RunWithIO(ctx, workload, fixtureGeneration(), nil, nil, stdout, io.Discard)
		result <- runResult{code: code, err: err}
	}()
	return result
}

func waitForHelperMarker(t *testing.T, marker string) {
	t.Helper()
	waitForHelperMarkerWithin(t, marker, 5*time.Second)
}

func waitForHelperMarkerWithin(t *testing.T, marker string, limit time.Duration) {
	t.Helper()
	deadline := time.NewTimer(limit)
	defer deadline.Stop()
	for {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("workload helper did not start")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func receiveLifecycleResult(t *testing.T, result <-chan runResult) runResult {
	t.Helper()
	return receiveLifecycleResultWithin(t, result, 5*time.Second)
}

func receiveLifecycleResultWithin(t *testing.T, result <-chan runResult, limit time.Duration) runResult {
	t.Helper()
	select {
	case received := <-result:
		return received
	case <-time.After(limit):
		t.Fatal("workload run did not return")
		return runResult{}
	}
}

func assertContextResult(t *testing.T, result runResult, expected error) {
	t.Helper()
	if result.code != 0 || !errors.Is(result.err, expected) {
		t.Fatalf("context result = code %d, error %v", result.code, result.err)
	}
}
