//go:build !windows

package platform

import (
	"os"
	"testing"
)

func TestTrustedUnixModes(t *testing.T) {
	if !trustedUnixDirectoryMode(0o755, 0) {
		t.Fatal("root-owned immutable directory rejected")
	}
	if !trustedUnixDirectoryMode(0o777|os.ModeSticky, 0) {
		t.Fatal("root-owned sticky temporary directory rejected")
	}
	if trustedUnixDirectoryMode(0o775, os.Geteuid()) {
		t.Fatal("group-writable ancestor accepted")
	}
	if trustedUnixDirectoryMode(0o777|os.ModeSticky, os.Geteuid()) {
		t.Fatal("non-root sticky ancestor accepted")
	}
	if trustedUnixOwner(os.Geteuid() + 1) {
		t.Fatal("foreign owner accepted")
	}
}
