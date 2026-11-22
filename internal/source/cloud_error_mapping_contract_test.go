package source

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/maldous/CircularQuay/internal/fault"
)

// A missing provider value is not an availability outage. Neither a gRPC
// message nor a caller's wrapping detail may escape through its safe fault.
func TestCloudErrorMappingContractNotFoundRemainsMissingWithoutCause(t *testing.T) {
	missing := status.Error(codes.NotFound, "SYNTHETIC_PRIVATE_PROVIDER_DETAIL")
	for _, tc := range []struct {
		name     string
		upstream error
	}{
		{"direct", missing},
		{"wrapped", fmt.Errorf("SYNTHETIC_PRIVATE_WRAPPER_DETAIL: %w", missing)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertCloudErrorMissingMapping(t, tc.upstream)
		})
	}
}

func assertCloudErrorMissingMapping(t *testing.T, upstream error) {
	t.Helper()
	mapped := cloudError(upstream)
	typed, ok := mapped.(*fault.Error)
	if !ok {
		t.Fatal("missing provider response did not return a direct typed fault")
	}
	if typed.Kind != fault.Missing || typed.Operation != "provider_request" {
		t.Fatal("missing provider response changed its category or operation")
	}
	if mapped.Error() != "missing_value: provider_request" {
		t.Fatal("missing provider response disclosed details or changed its safe error")
	}
	if errors.Unwrap(mapped) != nil {
		t.Fatal("missing provider response retained an upstream error cause")
	}
}
