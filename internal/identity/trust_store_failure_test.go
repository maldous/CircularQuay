package identity

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestMissingPinnedTrustStoreDoesNotCreateVerifier(t *testing.T) {
	authority := Authority{
		Name: "synthetic", Issuer: "https://issuer.example.test",
		JWKSURL: "https://issuer.example.test/keys", Audience: "api://cq",
		Profile:          ProfileRFC9068,
		TrustStoreFile:   filepath.Join(t.TempDir(), "missing-authority-roots.pem"),
		TrustStoreSHA256: strings.Repeat("0", 64),
	}
	verifier, err := newVerifier(context.Background(), authority, nil, time.Now)
	if verifier != nil || !fault.Is(err, fault.Trust) {
		t.Fatalf("missing pinned trust roots created a verifier: verifier=%v err=%v", verifier != nil, err)
	}
	if strings.Contains(err.Error(), authority.TrustStoreFile) {
		t.Fatal("trust failure disclosed the local trust-store path")
	}
}
