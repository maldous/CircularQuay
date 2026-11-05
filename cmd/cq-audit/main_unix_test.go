//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

const auditSignalHelperEnvironment = "CQ_TEST_AUDIT_SIGNAL_HELPER"

func TestAuditSignalContextCancelsForSIGTERM(t *testing.T) {
	if os.Getenv(auditSignalHelperEnvironment) == "1" {
		runAuditSignalHelper()
		return
	}
	command, lines, stderr := startAuditSignalHelper(t)
	if !lines.Scan() || lines.Text() != "ready" {
		t.Fatal("audit signal helper did not initialize")
	}
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if !lines.Scan() || lines.Text() != "cancelled" {
		t.Fatal("SIGTERM did not cancel the audit context")
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("audit signal helper did not exit gracefully: %v; stderr=%s", err, stderr.String())
	}
}

func startAuditSignalHelper(t *testing.T) (*exec.Cmd, *bufio.Scanner, *strings.Builder) {
	t.Helper()
	deadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(deadline, os.Args[0], "-test.run=^TestAuditSignalContextCancelsForSIGTERM$")
	command.Env = []string{auditSignalHelperEnvironment + "=1"}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	command.Stderr = &stderr
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	return command, bufio.NewScanner(stdout), &stderr
}

func runAuditSignalHelper() {
	ctx, cancel := auditSignalContext(context.Background())
	defer cancel()
	fmt.Println("ready")
	<-ctx.Done()
	fmt.Println("cancelled")
}
