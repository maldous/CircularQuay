package cli

import (
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestParseWorkloadRequest(t *testing.T) {
	t.Parallel()

	request, err := parseWorkloadRequest([]string{"run", "--workload", "demo", "--", "program", "argument"})
	if err != nil {
		t.Fatal(err)
	}
	if request.command != "run" || request.id != "demo" || len(request.override) != 2 {
		t.Fatalf("run request = %+v, %v", request, err)
	}
	requireInvalidWorkloadRequest(t, []string{})
	requireInvalidWorkloadRequest(t, []string{"sync", "--workload", "demo", "unexpected"})
	request, err = parseWorkloadRequest([]string{"apply", "--workload", "demo"})
	if err != nil || request.target != "" {
		t.Fatalf("apply request = %+v, %v", request, err)
	}
}

func requireInvalidWorkloadRequest(t *testing.T, args []string) {
	t.Helper()
	if _, err := parseWorkloadRequest(args); !fault.Is(err, fault.Invalid) {
		t.Fatalf("arguments %q error = %v", args, err)
	}
}
