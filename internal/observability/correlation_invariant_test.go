package observability

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCorrelationInvariantUsesFullEntropyAndPreservesExistingIdentity(t *testing.T) {
	value := correlation(context.Background())
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 16 || len(value) != 32 || hex.EncodeToString(decoded) != value {
		t.Fatal("correlation lost its complete canonical 16-byte encoding")
	}
	selected := strings.Repeat("a", 32)
	ctx := context.WithValue(context.Background(), correlationKey{}, selected)
	if got := correlation(ctx); got != selected {
		t.Fatal("existing bounded correlation identity was replaced")
	}
}
