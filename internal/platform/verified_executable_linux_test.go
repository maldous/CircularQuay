//go:build linux

package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestVerifiedExecutableRejectsInvalidIdentity(t *testing.T) {
	path := verifiedScriptFixture(t, "#!/bin/sh\nexit 0\n")
	digest := verifiedFixtureDigest(t, path)
	for _, test := range []struct{ name, path, digest string }{
		{"missing digest", path, ""},
		{"malformed digest", path, strings.Repeat("z", 64)},
		{"mismatched digest", path, strings.Repeat("0", 64)},
		{"relative path", "executable", digest},
		{"missing file", path + "-absent", digest},
		{"directory", filepath.Dir(path), digest},
	} {
		t.Run(test.name, func(t *testing.T) {
			executable, err := OpenVerifiedExecutable(test.path, test.digest)
			if !fault.Is(err, fault.Trust) || executable != nil {
				t.Fatalf("invalid identity admitted: %v", err)
			}
		})
	}
}

func TestVerifiedExecutableRejectsWritableFileAndAncestor(t *testing.T) {
	for _, ancestor := range []bool{false, true} {
		path := verifiedScriptFixture(t, "#!/bin/sh\nexit 0\n")
		digest := verifiedFixtureDigest(t, path)
		target := path
		if ancestor {
			target = filepath.Dir(path)
		}
		if err := os.Chmod(target, 0o770); err != nil {
			t.Fatal(err)
		}
		executable, err := OpenVerifiedExecutable(path, digest)
		if !fault.Is(err, fault.Trust) || executable != nil {
			t.Fatalf("writable boundary admitted: %v", err)
		}
	}
}

func TestVerifiedExecutableRejectsOversizedImage(t *testing.T) {
	path := verifiedScriptFixture(t, "#!/bin/sh\nexit 0\n")
	if err := os.Truncate(path, maxVerifiedExecutableBytes+1); err != nil {
		t.Fatal(err)
	}
	if executable, err := OpenVerifiedExecutable(path, strings.Repeat("0", 64)); !fault.Is(err, fault.Trust) || executable != nil {
		t.Fatalf("oversized image admitted: %v", err)
	}
}

func TestVerifiedExecutableRetainsNativeObjectAcrossNameChanges(t *testing.T) {
	for _, change := range []string{"replace", "unlink", "symlink"} {
		t.Run(change, func(t *testing.T) {
			path, directory := protectedPlatformHelper(t)
			executable := openVerifiedFixture(t, path)
			replaceVerifiedFixtureName(t, path, change)
			var output bytes.Buffer
			options := verifiedNativeOptions(directory, &output)
			if err := executable.Run(context.Background(), []string{"--platform-helper"}, options); err != nil {
				t.Fatalf("retained native launch failed: %v", err)
			}
			if output.String() != "allowed-status" {
				t.Fatal("replacement ran instead of verified object")
			}
		})
	}
}

func replaceVerifiedFixtureName(t *testing.T, path, change string) {
	t.Helper()
	if change == "unlink" {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		return
	}
	replacement := verifiedScriptFixture(t, "#!/bin/sh\nprintf replacement\n")
	if change == "symlink" {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(replacement, path); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
}

func TestVerifiedExecutableRetainsOriginallySymlinkedObject(t *testing.T) {
	path, directory := protectedPlatformHelper(t)
	link := filepath.Join(filepath.Dir(path), "selected")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	executable := openVerifiedFixture(t, link)
	replaceVerifiedFixtureName(t, link, "symlink")
	var output bytes.Buffer
	if err := executable.Run(context.Background(), []string{"--platform-helper"}, verifiedNativeOptions(directory, &output)); err != nil {
		t.Fatal(err)
	}
	if output.String() != "allowed-status" {
		t.Fatal("changed symlink redirected the launch")
	}
}

func TestVerifiedExecutableScriptPreservesInputsAndExitStatus(t *testing.T) {
	t.Setenv("CQ_UNEXPECTED_INHERITANCE", "must-not-reach-child")
	path := verifiedScriptFixture(t, "#!/bin/sh\nIFS= read -r input\nprintf '%s|%s|%s|%s|%s' \"$1\" \"$PWD\" \"$CQ_VALUE\" \"$input\" \"$CQ_UNEXPECTED_INHERITANCE\"\nprintf diagnostic >&2\nexit 17\n")
	executable := openVerifiedFixture(t, path)
	replaceVerifiedFixtureName(t, path, "replace")
	directory := secureTempDir(t)
	var output, diagnostic bytes.Buffer
	options := VerifiedExecutionOptions{
		Directory: directory, Environment: []string{"CQ_VALUE=synthetic"},
		Stdin: strings.NewReader("synthetic input\n"), Stdout: &output, Stderr: &diagnostic,
		WaitDelay: pipeDrainWaitDelay,
	}
	err := executable.Run(context.Background(), []string{"spaces and Unicode-✓-$()"}, options)
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 17 {
		t.Fatalf("script exit status changed: %v", err)
	}
	expected := "spaces and Unicode-✓-$()|" + directory + "|synthetic|synthetic input|"
	if output.String() != expected || diagnostic.String() != "diagnostic" {
		t.Fatal("script arguments, cwd, environment or streams changed")
	}
}

func TestVerifiedExecutableNilEnvironmentDoesNotInherit(t *testing.T) {
	t.Setenv("CQ_UNEXPECTED_INHERITANCE", "must-not-reach-child")
	path := verifiedScriptFixture(t, "#!/bin/sh\nprintf '%s' \"$CQ_UNEXPECTED_INHERITANCE\"\n")
	executable := openVerifiedFixture(t, path)
	var output bytes.Buffer
	if err := executable.Run(context.Background(), nil, VerifiedExecutionOptions{Stdout: &output}); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatal("nil environment inherited the parent environment")
	}
}

func TestVerifiedExecutableCancellationBeforeStart(t *testing.T) {
	path, directory := protectedPlatformHelper(t)
	executable := openVerifiedFixture(t, path)
	marker := filepath.Join(directory, "started")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := executable.Run(ctx, []string{"--platform-helper", lifecycleWaitMode, marker}, verifiedNativeOptions(directory, io.Discard))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled launch result = %v", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled process started")
	}
}

func TestVerifiedExecutableCloseAfterStartAndCancellation(t *testing.T) {
	path, directory := protectedPlatformHelper(t)
	executable := openVerifiedFixture(t, path)
	file := executable.state.file
	marker := filepath.Join(directory, "started")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- executable.Run(ctx, []string{"--platform-helper", lifecycleWaitMode, marker}, verifiedNativeOptions(directory, io.Discard))
	}()
	waitForHelperMarker(t, marker)
	if err := executable.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatal("Close retained the parent descriptor")
	}
	cancel()
	select {
	case err := <-result:
		var exit *exec.ExitError
		if !errors.As(err, &exit) && !errors.Is(err, context.Canceled) {
			t.Fatalf("CommandContext cancellation lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled process did not finish")
	}
}

func TestVerifiedExecutableZeroClosedAndCopiedCapabilities(t *testing.T) {
	path := verifiedScriptFixture(t, "#!/bin/sh\nexit 0\n")
	executable := openVerifiedFixture(t, path)
	copyOfCapability := *executable
	if err := executable.Close(); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []*VerifiedExecutable{nil, {}, executable, &copyOfCapability} {
		if err := candidate.Run(context.Background(), nil, VerifiedExecutionOptions{}); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("unavailable capability launched: %v", err)
		}
		if err := candidate.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVerifiedExecutableConcurrentCloseAndStart(t *testing.T) {
	path := verifiedScriptFixture(t, "#!/bin/sh\nexit 0\n")
	executable := openVerifiedFixture(t, path)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	result := make(chan error, 8)
	for range 8 {
		go func() {
			<-start
			result <- executable.Run(ctx, nil, VerifiedExecutionOptions{WaitDelay: pipeDrainWaitDelay})
		}()
	}
	close(start)
	if err := executable.Close(); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		select {
		case err := <-result:
			if err != nil && !errors.Is(err, os.ErrClosed) {
				t.Fatalf("close/start descriptor ownership failed: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("concurrent launches did not complete")
		}
	}
}

func TestVerifiedExecutablePreservesCallerOutputLimit(t *testing.T) {
	path := verifiedScriptFixture(t, "#!/bin/sh\nprintf 1234567890123456\n")
	executable := openVerifiedFixture(t, path)
	limitError := errors.New("synthetic output limit")
	output := &verifiedLimitedWriter{remaining: 4, limitError: limitError}
	err := executable.Run(context.Background(), nil, VerifiedExecutionOptions{Stdout: output, WaitDelay: pipeDrainWaitDelay})
	if !errors.Is(err, limitError) || output.written != 4 {
		t.Fatalf("caller output bound lost: %v", err)
	}
}

type verifiedLimitedWriter struct {
	remaining  int
	written    int
	limitError error
}

func (writer *verifiedLimitedWriter) Write(data []byte) (int, error) {
	count := min(len(data), writer.remaining)
	writer.remaining -= count
	writer.written += count
	if count != len(data) {
		return count, writer.limitError
	}
	return count, nil
}

func TestVerifiedExecutableBoundsOrphanedOutputPipe(t *testing.T) {
	path, directory := protectedPlatformHelper(t)
	executable := openVerifiedFixture(t, path)
	marker := filepath.Join(directory, "started")
	completed := filepath.Join(directory, "completed")
	t.Cleanup(func() { waitForHelperMarkerWithin(t, completed, 15*time.Second) })
	started := time.Now()
	err := executable.Run(context.Background(), []string{"--platform-helper", "orphan-pipe", marker, completed}, verifiedNativeOptions(directory, &bytes.Buffer{}))
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("orphaned pipe error = %v", err)
	}
	if time.Since(started) >= 5*time.Second {
		t.Fatal("WaitDelay did not bound pipe draining")
	}
}

func TestVerifiedExecutableStartFailureDoesNotReopenName(t *testing.T) {
	path := verifiedScriptFixture(t, "not an executable format\n")
	executable := openVerifiedFixture(t, path)
	replaceVerifiedFixtureName(t, path, "replace")
	var output bytes.Buffer
	if err := executable.Run(context.Background(), nil, VerifiedExecutionOptions{Stdout: &output}); err == nil {
		t.Fatal("invalid held image fell back to a replacement pathname or shell")
	}
	if output.Len() != 0 {
		t.Fatal("replacement ran after descriptor launch failure")
	}
	if _, err := executable.state.file.Stat(); err != nil {
		t.Fatal("failed start consumed the capability")
	}
}

func TestVerifiedExecutableRejectsOpenWriterAtLaunch(t *testing.T) {
	path, directory := protectedPlatformHelper(t)
	executable := openVerifiedFixture(t, path)
	writer, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if err := executable.Run(context.Background(), []string{"--platform-helper"}, verifiedNativeOptions(directory, io.Discard)); err == nil {
		t.Fatal("native image executed while open for writing")
	}
}

func TestVerifiedExecutableFailureClosesCheckedDescriptor(t *testing.T) {
	path := verifiedScriptFixture(t, "#!/bin/sh\nexit 0\n")
	before := verifiedFixtureOpenDescriptors(t, path)
	for range 3 {
		if executable, err := OpenVerifiedExecutable(path, strings.Repeat("0", 64)); err == nil || executable != nil {
			t.Fatal("mismatched image accepted")
		}
	}
	if after := verifiedFixtureOpenDescriptors(t, path); after != before {
		t.Fatal("failed constructor leaked executable descriptors")
	}
}

func verifiedFixtureOpenDescriptors(t *testing.T, path string) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err == nil && target == path {
			count++
		}
	}
	return count
}

func verifiedScriptFixture(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(secureTempDir(t), "executable")
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func verifiedFixtureDigest(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:])
}

func openVerifiedFixture(t *testing.T, path string) *VerifiedExecutable {
	t.Helper()
	executable, err := OpenVerifiedExecutable(path, verifiedFixtureDigest(t, path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executable.Close(); err != nil {
			t.Error(err)
		}
	})
	return executable
}

func verifiedNativeOptions(directory string, output io.Writer) VerifiedExecutionOptions {
	return VerifiedExecutionOptions{
		Directory: directory, Environment: []string{"APP_SECRET=synthetic-✓-$()\nline"},
		Stdout: output, Stderr: io.Discard, WaitDelay: pipeDrainWaitDelay,
	}
}
