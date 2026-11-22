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

const rootSignalHelperEnvironment = "CQ_TEST_ROOT_SIGNAL_HELPER"

func TestRootSignalNotifyContextCancelsForSIGTERM(t *testing.T) {
	if os.Getenv(rootSignalHelperEnvironment) == "1" {
		runRootSignalHelper()
		return
	}

	command, lines, stderr := startRootSignalHelper(t)
	expectRootSignalLine(t, lines, "ready")
	sendRootSIGTERM(t, command)
	expectRootSignalLine(t, lines, "cancelled")
	assertRootSignalHelperExited(t, command, lines, stderr)
}

func startRootSignalHelper(t *testing.T) (*exec.Cmd, *bufio.Scanner, *strings.Builder) {
	t.Helper()
	deadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(deadline, os.Args[0], "-test.run=^TestRootSignalNotifyContextCancelsForSIGTERM$")
	command.Env = append(os.Environ(), rootSignalHelperEnvironment+"=1")
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	command.Stderr = &stderr
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	return command, bufio.NewScanner(stdout), &stderr
}

func expectRootSignalLine(t *testing.T, lines *bufio.Scanner, want string) {
	t.Helper()
	if !lines.Scan() || lines.Text() != want {
		t.Fatalf("root signal helper line = %q, want %q", lines.Text(), want)
	}
}

func sendRootSIGTERM(t *testing.T, command *exec.Cmd) {
	t.Helper()
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
}

func assertRootSignalHelperExited(t *testing.T, command *exec.Cmd, lines *bufio.Scanner, stderr *strings.Builder) {
	t.Helper()
	if err := lines.Err(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("helper did not exit gracefully after SIGTERM: %v; stderr=%s", err, stderr.String())
	}
}

func runRootSignalHelper() {
	ctx, cancel := rootSignalNotifyContext(context.Background())
	defer cancel()
	fmt.Println("ready")
	<-ctx.Done()
	fmt.Println("cancelled")
}
