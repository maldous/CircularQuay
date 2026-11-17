package transport

import (
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestEndpointBoundaryContractEncodedHostControlsRemainRefused(t *testing.T) {
	for _, raw := range []string{"https://host.invalid%00/", "https://host.invalid%0d/", "https://host.invalid%0a/"} {
		if endpoint, err := Endpoint(raw); endpoint != nil || !fault.Is(err, fault.Trust) {
			t.Fatal("encoded host control produced a trusted endpoint")
		}
	}
	// These inputs are rejected by net/url before CQ's subsequent literal-host
	// control fence. This does not claim that redundant fence was executed.
}
