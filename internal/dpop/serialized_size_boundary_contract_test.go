package dpop

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestStoreSerializedSizeContractInternalMutationCannotReplaceBoundedState(t *testing.T) {
	store, now := testStore(t)
	writeCache(t, store, cacheWithEntries(1, digest("existing"), now))
	before := readCacheBytes(t, store)
	calls := 0
	// Public Challenge/Consume use bounded fixed digest keys. This deliberately
	// oversized private mutation tests the defensive final serialization fence,
	// not a claim that public callers can submit arbitrary replay-cache keys.
	err := store.update(context.Background(), func(cache *persistedCache, _ time.Time) error {
		calls++
		cache.Entries[strings.Repeat("x", platform.MaxDocumentSize)] = cacheEntry{Kind: "nonce", Binding: digest("binding"), Expires: now.Add(time.Minute).Unix()}
		return nil
	})
	if !errors.Is(err, ErrInvalid) || calls != 1 || !bytes.Equal(before, readCacheBytes(t, store)) {
		t.Fatal("oversized internal serialization replaced the bounded replay state")
	}
}
