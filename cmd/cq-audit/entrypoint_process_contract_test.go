package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The only process is this exact test image. Its selected child calls the real
// main with fixed harmless arguments, retaining real exit and signal behavior.
func TestAuditContractOwnImageExitAndCoverage(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	coverageDirectory := entrypointCoverageDirectory(t)
	before := entrypointCounterFiles(t, coverageDirectory)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestAuditContractSelectedChild$")
	command.WaitDelay = time.Second
	command.Dir = t.TempDir()
	command.Env = []string{"CQ_TEST_Audit_CHILD=1", "HOME=" + command.Dir, "TMPDIR=" + command.Dir, "TMP=" + command.Dir, "TEMP=" + command.Dir, "SystemRoot=" + os.Getenv("SystemRoot"), "GOCOVERDIR=" + coverageDirectory}
	output := &entrypointBoundedOutput{}
	command.Stdout, command.Stderr = output, output
	err = command.Run()
	var status *exec.ExitError
	if !errors.As(err, &status) || status.ExitCode() != 2 {
		t.Fatal("owned invalid-argument entrypoint lost exit status two")
	}
	if !strings.Contains(output.buffer.String(), "usage: cq-audit run") || output.overflow || ctx.Err() != nil {
		t.Fatal("owned entrypoint lost bounded expected output or completion")
	}
	if coverageDirectory != "" && !entrypointOwnCounter(t, coverageDirectory, command.Process.Pid, before) {
		t.Fatal("instrumented entrypoint child did not retain its original counter pod")
	}
}

func TestAuditContractSelectedChild(t *testing.T) {
	if os.Getenv("CQ_TEST_Audit_CHILD") != "1" {
		return
	}
	os.Args = []string{"cq-audit", "invalid"}
	main()
	t.Fatal("real main returned instead of using its exit contract")
}

func entrypointCoverageDirectory(t *testing.T) string {
	t.Helper()
	selected := flag.Lookup("test.gocoverdir")
	if selected == nil || selected.Value.String() == "" {
		return ""
	}
	directory := selected.Value.String()
	if !filepath.IsAbs(directory) {
		t.Fatal("Go test coverage directory is not absolute")
	}
	return directory
}

func entrypointCounterFiles(t *testing.T, directory string) map[string]bool {
	t.Helper()
	result := map[string]bool{}
	if directory == "" {
		return result
	}
	matches, err := filepath.Glob(filepath.Join(directory, "covcounters.*"))
	if err != nil {
		t.Fatal("Go test counter inventory failed")
	}
	for _, name := range matches {
		result[name] = true
	}
	return result
}

func entrypointOwnCounter(t *testing.T, directory string, pid int, before map[string]bool) bool {
	t.Helper()
	// Go's pinned emitter names each pod covcounters.<metadata>.<pid>.<time>.
	// Other test processes may write this directory; only our actual child PID
	// and a filename absent before launch establish this child's retention.
	matches, err := filepath.Glob(filepath.Join(directory, "covcounters.*."+strconv.Itoa(pid)+".*"))
	if err != nil {
		t.Fatal("own child counter inventory failed")
	}
	for _, name := range matches {
		if !before[name] {
			return true
		}
	}
	return false
}

type entrypointBoundedOutput struct {
	buffer   bytes.Buffer
	overflow bool
}

func (output *entrypointBoundedOutput) Write(data []byte) (int, error) {
	if output.overflow || len(data) > (64<<10)-output.buffer.Len() {
		output.overflow = true
		return 0, io.ErrShortBuffer
	}
	return output.buffer.Write(data)
}
