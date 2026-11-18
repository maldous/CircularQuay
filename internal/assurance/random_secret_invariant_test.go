package assurance

import (
	"encoding/hex"
	"testing"
)

func TestRandomSecretInvariantKeepsFullCanonicalEntropyEncoding(t *testing.T) {
	value := randomSecret()
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 || len(value) != 64 || hex.EncodeToString(decoded) != value {
		t.Fatal("random secret lost its complete canonical 32-byte encoding")
	}
}
