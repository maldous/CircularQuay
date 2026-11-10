//go:build linux && !android

package platform

import (
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestSystemdCredentialName(t *testing.T) {
	name, err := systemdCredentialName("/run/credentials/demo.service/APP_SECRET")
	if err != nil || name != "APP_SECRET" {
		t.Fatal("safe systemd credential name rejected")
	}
	for _, path := range []string{"relative", "/run/credentials/demo/--flag", "/run/credentials/demo/name;command"} {
		if _, err := systemdCredentialName(path); !fault.Is(err, fault.Invalid) {
			t.Fatal("unsafe systemd credential name accepted")
		}
	}
}
