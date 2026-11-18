package transport

import (
	"fmt"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestEndpointControlInvariantParserRefusesLiteralAndEscapedControls(t *testing.T) {
	for value := 0; value <= 127; value++ {
		if value >= 32 && value != 127 {
			continue
		}
		for _, raw := range []string{
			"https://host" + string(rune(value)) + ".invalid/",
			fmt.Sprintf("https://host%%%02x.invalid/", value),
			fmt.Sprintf("https://[fe80::1%%25zone%%%02x]/", value),
		} {
			endpointControlRefusal(t, raw)
		}
	}
}

func endpointControlRefusal(t *testing.T, raw string) {
	t.Helper()
	if endpoint, err := Endpoint(raw); endpoint != nil || !fault.Is(err, fault.Trust) {
		t.Fatal("literal or escaped control passed the endpoint parser boundary")
	}
}

func TestEndpointControlInvariantOrdinaryHostsRemainAdmitted(t *testing.T) {
	for _, raw := range []string{"https://host.invalid/", "https://127.0.0.1:443/", "https://[::1]:443/"} {
		endpoint, err := Endpoint(raw)
		if err != nil || endpoint == nil || endpoint.String() != raw {
			t.Fatal("ordinary host changed during parser invariant simplification")
		}
	}
}
