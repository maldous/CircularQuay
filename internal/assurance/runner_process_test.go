package assurance

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if code, helper := runAssuranceHelper(os.Args); helper {
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func runAssuranceHelper(arguments []string) (int, bool) {
	if len(arguments) != 3 || arguments[1] != "--assurance-helper" {
		return 0, false
	}
	switch arguments[2] {
	case "nonzero":
		return 7, true
	case "overflow":
		_, _ = io.WriteString(os.Stdout, strings.Repeat("x", 2<<20))
		return 0, true
	case "hold-pipe":
		return holdPipe(), true
	case "leave-pipe-open":
		child := exec.Command(arguments[0], "--assurance-helper", "hold-pipe")
		child.Env = helperRuntimeEnvironment()
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if child.Start() != nil {
			return 8, true
		}
		return 0, true
	case "wait":
		return waitForCancellation(), true
	default:
		return 0, false
	}
}

func waitForCancellation() int {
	marker := os.Getenv("CQ_ASSURANCE_STARTED")
	if marker == "" || os.WriteFile(marker, []byte("ready"), 0o600) != nil {
		return 9
	}
	time.Sleep(10 * time.Second)
	return 0
}

func holdPipe() int {
	deadline := time.NewTimer(10 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			return 0
		case <-ticker.C:
			if _, err := io.WriteString(os.Stdout, "."); err != nil {
				return 0
			}
		}
	}
}

func helperRuntimeEnvironment() []string {
	var environment []string
	for _, key := range []string{"SystemRoot", "SYSTEMROOT"} {
		if value := os.Getenv(key); value != "" {
			environment = append(environment, key+"="+value)
		}
	}
	return environment
}

func TestRunCommandClassifiesProcessFailuresWithoutOutput(t *testing.T) {
	binary, directory, environment := assuranceHelper(t)
	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{"nonzero", []string{"--assurance-helper", "nonzero"}, 7},
		{"overflow", []string{"--assurance-helper", "overflow"}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runCommand(context.Background(), binary, test.args, directory, environment)
			if result.exitCode != test.code || result.missing || len(result.output) != 0 {
				t.Fatalf("result = exit %d, missing %t, output bytes %d; want exit %d without retained output", result.exitCode, result.missing, len(result.output), test.code)
			}
		})
	}
}

func TestRunCommandCancellationAndPipeDelayAreBounded(t *testing.T) {
	binary, directory, environment := assuranceHelper(t)
	t.Run("cancellation", func(t *testing.T) {
		marker := filepath.Join(directory, "started")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		result := make(chan commandResult, 1)
		environment := append(append([]string{}, environment...), "CQ_ASSURANCE_STARTED="+marker)
		go func() {
			result <- runCommand(ctx, binary, []string{"--assurance-helper", "wait"}, directory, environment)
		}()
		waitForHelperMarker(t, marker)
		started := time.Now()
		cancel()
		select {
		case got := <-result:
			if got.exitCode == 0 || got.missing || time.Since(started) > 5*time.Second {
				t.Fatalf("canceled command result = %#v after %s", got, time.Since(started))
			}
		case <-time.After(5 * time.Second):
			t.Fatal("started command did not honor cancellation")
		}
	})
	t.Run("inherited pipe", func(t *testing.T) {
		started := time.Now()
		result := runCommand(context.Background(), binary, []string{"--assurance-helper", "leave-pipe-open"}, directory, environment)
		if result.exitCode != 1 || result.missing || len(result.output) != 0 || time.Since(started) > 5*time.Second {
			t.Fatalf("delayed pipe result = %#v after %s", result, time.Since(started))
		}
	})
}

func waitForHelperMarker(t *testing.T, marker string) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("helper did not start")
		case <-ticker.C:
		}
	}
}

func assuranceHelper(t *testing.T) (string, string, []string) {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	directory := privateAssuranceDirectory(t, "process")
	return binary, directory, auditEnvironment(filepath.Join(directory, "home"))
}
