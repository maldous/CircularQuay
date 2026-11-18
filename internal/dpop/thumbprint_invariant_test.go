package dpop

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
)

func TestThumbprintInvariantAdmittedAlgorithmsMatchPinnedLibrary(t *testing.T) {
	for _, algorithm := range []string{"ES256", "RS256"} {
		t.Run(algorithm, func(t *testing.T) { assertThumbprintInvariant(t, algorithm) })
	}
}

func assertThumbprintInvariant(t *testing.T, algorithm string) {
	t.Helper()
	var public crypto.PublicKey
	if algorithm == "ES256" {
		public = &proofECKey(t).PublicKey
	} else {
		public = &proofRSAKey(t, 2048).PublicKey
	}
	declared := jose.JSONWebKey{Key: public}
	raw, err := json.Marshal(declared)
	if err != nil {
		t.Fatal(err)
	}
	want, err := declared.Thumbprint(crypto.SHA256)
	if err != nil || len(want) != 32 {
		t.Fatal("admitted key shape violated the pinned SHA-256 thumbprint contract")
	}
	key, got, err := publicJWK(raw, algorithm)
	if err != nil || key == nil || got != base64.RawURLEncoding.EncodeToString(want) {
		t.Fatal("public key admission changed its library-derived thumbprint")
	}
}
