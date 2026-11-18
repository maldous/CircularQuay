package dpop

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestStoreRejectsExpiredAndMalformedReplayBeforeWrite(t *testing.T) {
	s, now := testStore(t)
	binding := digest("binding")
	if _, err := s.Challenge(context.Background(), binding, now); err == nil {
		t.Fatal("expired challenge accepted")
	}
	nonce, err := s.Challenge(context.Background(), binding, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	before := readCacheBytes(t, s)
	if _, err := s.Consume(context.Background(), binding, nonce, "bad", now.Add(time.Minute)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed replay error = %v, want ErrInvalid", err)
	}
	if after := readCacheBytes(t, s); !bytes.Equal(before, after) {
		t.Fatal("malformed replay mutated the cache")
	}
}

func TestStoreUsesInjectedClockInsideMutation(t *testing.T) {
	s, now := testStore(t)
	expires := now.Add(time.Minute)
	s.now = func() time.Time { return expires }
	if _, err := s.Challenge(context.Background(), digest("binding"), expires); !errors.Is(err, ErrInvalid) {
		t.Fatalf("challenge at injected expiry = %v, want ErrInvalid", err)
	}
	s.now = func() time.Time { return now }
	if _, err := s.Challenge(context.Background(), digest("binding"), expires); err != nil {
		t.Fatalf("challenge before injected expiry: %v", err)
	}
}

func TestStoreFailureDoesNotReturnNonce(t *testing.T) {
	s, now := testStore(t)
	writeCacheBytes(t, s, []byte("null"))
	nonce, err := s.Challenge(context.Background(), digest("binding"), now.Add(time.Minute))
	if err == nil || nonce != "" {
		t.Fatalf("challenge failure returned nonce %q, err %v", nonce, err)
	}
}

func TestStoreDoesNotReturnNonceAfterExpiryDuringWrite(t *testing.T) {
	s, now := testStore(t)
	expires := now.Add(time.Minute)
	calls := 0
	s.now = func() time.Time {
		calls++
		if calls == 1 {
			return now
		}
		return expires
	}
	nonce, err := s.Challenge(context.Background(), digest("binding"), expires)
	if !errors.Is(err, ErrInvalid) || nonce != "" {
		t.Fatalf("expired after write = (%q, %v), want empty ErrInvalid", nonce, err)
	}
	if _, err := s.read(); err != nil {
		t.Fatalf("cache after expiry fence is unreadable: %v", err)
	}
}

func TestStoreFailsClosedForInvalidPersistedDocuments(t *testing.T) {
	for name, document := range map[string][]byte{
		"corrupt": []byte("not-json"),
		"null":    []byte("null"),
		"version": []byte(`{"version":2,"entries":{}}`),
		"binding": []byte(`{"version":1,"entries":{"n:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA":{"kind":"nonce","binding":"bad","expires":1893456000}}}`),
	} {
		t.Run(name, func(t *testing.T) {
			s, now := testStore(t)
			writeCacheBytes(t, s, document)
			assertChallengeFails(t, s, now)
		})
	}
}

func TestStoreFailsClosedForUnreadableCache(t *testing.T) {
	s, now := testStore(t)
	if err := os.Mkdir(cachePath(s), 0o700); err != nil {
		t.Fatal(err)
	}
	assertChallengeFails(t, s, now)
}

func TestStoreRejectsGlobalAndBindingCapacity(t *testing.T) {
	t.Run("global", func(t *testing.T) {
		s, now := testStore(t)
		writeCache(t, s, cacheWithEntries(maxCacheSize, digest("other"), now))
		assertChallengeFails(t, s, now)
	})
	t.Run("binding", func(t *testing.T) {
		s, now := testStore(t)
		binding := digest("binding")
		writeCache(t, s, cacheWithEntries(maxBindingEntries, binding, now))
		if _, err := s.Challenge(context.Background(), binding, now.Add(time.Minute)); !errors.Is(err, ErrInvalid) {
			t.Fatalf("binding capacity error = %v, want ErrInvalid", err)
		}
	})
}

func TestStoreRejectsConsumeAtGlobalAndBindingCapacity(t *testing.T) {
	t.Run("global", func(t *testing.T) {
		s, now := testStore(t)
		binding, nonce := digest("binding"), testNonce()
		cache := cacheWithEntries(maxCacheSize-1, digest("other"), now)
		cache.Entries[nonceKey(nonce)] = cacheEntry{Kind: "nonce", Binding: binding, Expires: now.Add(time.Minute).Unix()}
		writeCache(t, s, cache)
		assertConsumeFails(t, s, binding, nonce, now)
	})
	t.Run("binding", func(t *testing.T) {
		s, now := testStore(t)
		binding, nonce := digest("binding"), testNonce()
		cache := cacheWithEntries(maxBindingEntries-1, binding, now)
		cache.Entries[nonceKey(nonce)] = cacheEntry{Kind: "nonce", Binding: binding, Expires: now.Add(time.Minute).Unix()}
		writeCache(t, s, cache)
		assertConsumeFails(t, s, binding, nonce, now)
	})
}

func TestStoreFailsClosedForCancellationAndRandomness(t *testing.T) {
	s, now := testStore(t)
	binding := digest("binding")
	nonce, err := s.Challenge(context.Background(), binding, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	before := readCacheBytes(t, s)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if next, err := s.Consume(ctx, binding, nonce, digest("replay"), now.Add(time.Minute)); err == nil || next != "" {
		t.Fatalf("cancelled consume = (%q, %v)", next, err)
	}
	s.random = failingReader{}
	if next, err := s.Consume(context.Background(), binding, nonce, digest("replay"), now.Add(time.Minute)); err == nil || next != "" {
		t.Fatalf("randomness failure = (%q, %v)", next, err)
	}
	if after := readCacheBytes(t, s); !bytes.Equal(before, after) {
		t.Fatal("failed consume mutated the cache")
	}
}

func testStore(t *testing.T) (*PersistentStore, time.Time) {
	t.Helper()
	s, err := NewPersistentStore(privateStateDirectory(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	return s, now
}

func assertChallengeFails(t *testing.T, s *PersistentStore, now time.Time) {
	t.Helper()
	if nonce, err := s.Challenge(context.Background(), digest("binding"), now.Add(time.Minute)); err == nil || nonce != "" {
		t.Fatalf("unsafe cache challenge = (%q, %v)", nonce, err)
	}
}

func assertConsumeFails(t *testing.T, s *PersistentStore, binding, nonce string, now time.Time) {
	t.Helper()
	if next, err := s.Consume(context.Background(), binding, nonce, digest("replay"), now.Add(time.Minute)); err == nil || next != "" {
		t.Fatalf("capacity consume = (%q, %v)", next, err)
	}
}

func testNonce() string { return base64.RawURLEncoding.EncodeToString(make([]byte, 32)) }

func cacheWithEntries(count int, binding string, now time.Time) persistedCache {
	entries := make(map[string]cacheEntry, count)
	for index := range count {
		key := nonceKey(string(rune(index)))
		entries[key] = cacheEntry{Kind: "nonce", Binding: binding, Expires: now.Add(time.Minute).Unix()}
	}
	return persistedCache{Version: 1, Entries: entries}
}

func writeCache(t *testing.T, s *PersistentStore, cache persistedCache) {
	t.Helper()
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatal(err)
	}
	writeCacheBytes(t, s, data)
}

func writeCacheBytes(t *testing.T, s *PersistentStore, data []byte) {
	t.Helper()
	if err := platform.WriteProtected(cachePath(s), data); err != nil {
		t.Fatal(err)
	}
}

func readCacheBytes(t *testing.T, s *PersistentStore) []byte {
	t.Helper()
	data, err := platform.ReadProtected(cachePath(s), true)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func cachePath(s *PersistentStore) string { return filepath.Join(s.directory, cacheFile) }

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("synthetic random failure") }
