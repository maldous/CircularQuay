package dpop

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestStoreBoundaryContractInitializationPreservesOccupiedDirectory(t *testing.T) {
	root := storeBoundaryExistingDirectory(t)
	name := filepath.Join(root, "dpop")
	if err := os.WriteFile(name, []byte("owned sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	if store, err := NewPersistentStore(root); err == nil || store != nil {
		t.Fatal("occupied store directory produced a store")
	}
	data, err := os.ReadFile(name)
	if err != nil || string(data) != "owned sentinel" {
		t.Fatal("store initialization replaced an owned file")
	}
}

func TestStoreBoundaryContractInvalidChallengeAndEntropyHaveNoStateEffect(t *testing.T) {
	store, now := testStore(t)
	store.random = failingReader{}
	if nonce, err := store.Challenge(nil, digest("binding"), now.Add(time.Minute)); nonce != "" || !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid challenge consulted entropy or returned a nonce")
	}
	if nonce, err := store.Challenge(context.Background(), digest("binding"), now.Add(time.Minute)); nonce != "" || err == nil || errors.Is(err, ErrInvalid) {
		t.Fatal("valid challenge did not retain its entropy failure")
	}
	if _, err := os.Lstat(cachePath(store)); !os.IsNotExist(err) {
		t.Fatal("pre-mutation refusal created a cache")
	}
	store.random = nil
	if nonce, err := store.newNonce(); nonce != "" || !errors.Is(err, ErrInvalid) {
		t.Fatal("absent entropy source produced a nonce")
	}
}

func TestStoreBoundaryContractCancellationAfterLockReleasesBeforeRead(t *testing.T) {
	store, _ := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	acquired, released := 0, 0
	acquire := func(name string) (func() error, error) {
		acquired++
		if name != cacheName {
			t.Fatal("unexpected replay lock name")
		}
		cancel()
		return func() error { released++; return nil }, nil
	}
	err := store.updateWithLock(ctx, func(*persistedCache, time.Time) error {
		t.Fatal("post-lock cancellation reached the mutation")
		return nil
	}, acquire)
	if !errors.Is(err, context.Canceled) || acquired != 1 || released != 1 {
		t.Fatal("post-lock cancellation did not release its exact owned lock")
	}
	if _, err := os.Lstat(cachePath(store)); !os.IsNotExist(err) {
		t.Fatal("post-lock cancellation created a cache")
	}
}

func TestStoreBoundaryContractCancellationAtBothMutationFences(t *testing.T) {
	store, now := testStore(t)
	writeCache(t, store, cacheWithEntries(1, digest("existing"), now))
	before := readCacheBytes(t, store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := store.update(ctx, func(cache *persistedCache, _ time.Time) error {
		delete(cache.Entries, nonceKey(string(rune(0))))
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || !bytes.Equal(before, readCacheBytes(t, store)) {
		t.Fatal("cancellation after mutation changed persisted replay state")
	}
	err = store.update(ctx, func(*persistedCache, time.Time) error {
		t.Fatal("pre-canceled update reached its callback")
		return nil
	})
	if !errors.Is(err, context.Canceled) || !bytes.Equal(before, readCacheBytes(t, store)) {
		t.Fatal("pre-canceled update changed persisted replay state")
	}
}

func TestStoreBoundaryContractConsumeExpiryRetainsCommittedReplay(t *testing.T) {
	store, now := testStore(t)
	binding, nonce, replay := digest("binding"), testNonce(), digest("replay")
	expires := now.Add(time.Minute)
	writeCache(t, store, persistedCache{Version: 1, Entries: map[string]cacheEntry{nonceKey(nonce): {Kind: "nonce", Binding: binding, Expires: expires.Unix()}}})
	store.random = bytes.NewReader(bytes.Repeat([]byte{1}, 32))
	calls := 0
	store.now = func() time.Time {
		calls++
		if calls == 1 {
			return now
		}
		return expires
	}
	if next, err := store.Consume(context.Background(), binding, nonce, replay, expires); next != "" || !errors.Is(err, ErrInvalid) {
		t.Fatal("post-commit expiry exposed the next nonce")
	}
	cache, err := store.read()
	if err != nil || calls != 2 || len(cache.Entries) != 2 {
		t.Fatal("post-commit expiry lost committed replay state")
	}
	if _, found := cache.Entries[nonceKey(nonce)]; found || cache.Entries[replay].Kind != "replay" {
		t.Fatal("post-commit expiry restored the consumed nonce or lost replay denial")
	}
}

func TestStoreBoundaryContractReadFailureAndPruneAreExplicit(t *testing.T) {
	store, now := testStore(t)
	cache := persistedCache{Version: 1, Entries: map[string]cacheEntry{"expired": {Expires: now.Unix()}, "live": {Expires: now.Add(time.Second).Unix()}}}
	prune(&cache, now)
	if len(cache.Entries) != 1 || cache.Entries["live"].Expires != now.Add(time.Second).Unix() {
		t.Fatal("prune did not preserve the live sibling")
	}
	file := filepath.Join(storeBoundaryExistingDirectory(t), "not-a-directory")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	store.directory = file
	if got, err := store.read(); err == nil || got.Version != 0 || got.Entries != nil {
		t.Fatal("unreadable cache location produced a partial cache")
	}
	before := time.Now()
	current := (&PersistentStore{}).clock()
	if current.Before(before) || current.After(time.Now()) {
		t.Fatal("default store clock escaped its observation bracket")
	}
}

func storeBoundaryExistingDirectory(t *testing.T) string {
	t.Helper()
	directory := privateStateDirectory(t)
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestStoreBoundaryContractProofHelpersDoNotDiscloseOrAdmitInvalidKeys(t *testing.T) {
	challenge := &NonceError{Value: "synthetic-private-nonce"}
	if challenge.Error() != ErrNonce.Error() || !errors.Is(challenge, ErrNonce) || strings.Contains(challenge.Error(), challenge.Value) {
		t.Fatal("nonce error disclosed its nonce or lost error identity")
	}
	storeBoundaryKeyRefusals(t)
}

func storeBoundaryKeyRefusals(t *testing.T) {
	t.Helper()
	if header, key, thumbprint, err := proofKey("%"); err == nil || header.Type != "" || key != nil || thumbprint != "" {
		t.Fatal("bad header encoding produced partial key authority")
	}
	if algorithmKeyMatches("ES256", "not a key") || validES256Key(nil) {
		t.Fatal("unsupported or nil public key was admitted")
	}
	key := &ecdsa.PublicKey{Curve: elliptic.P256(), X: big.NewInt(0), Y: big.NewInt(0)}
	if validES256Key(key) {
		t.Fatal("off-curve public point was admitted")
	}
}

func TestStoreBoundaryContractOriginAndProofIdentifiersStayBound(t *testing.T) {
	resource, err := url.Parse("https://resource.invalid/mcp")
	if err != nil {
		t.Fatal(err)
	}
	request := &http.Request{Host: resource.Host, URL: &url.URL{Scheme: "https", Host: "substitute.invalid", Path: "/mcp"}}
	if requestOriginMatchesResource(request, resource) {
		t.Fatal("matching Host concealed a different absolute URL origin")
	}
	for _, suffix := range []string{"\"", "\\", "\x01"} {
		if validJTI(strings.Repeat("a", 16) + suffix) {
			t.Fatal("proof identifier accepted forbidden characters")
		}
	}
}

type storeBoundaryRefusal struct{ challenges, consumes int }

func (store *storeBoundaryRefusal) Challenge(context.Context, string, time.Time) (string, error) {
	store.challenges++
	return "", errors.New("synthetic store refusal")
}

func (store *storeBoundaryRefusal) Consume(context.Context, string, string, string, time.Time) (string, error) {
	store.consumes++
	return "", errors.New("unexpected synthetic consume")
}

func TestStoreBoundaryContractChallengeFailureCannotDiscloseOrConsume(t *testing.T) {
	store := &storeBoundaryRefusal{}
	now := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	verifier := Verifier{Store: store, Config: Config{Resource: "https://resource.invalid/mcp", MaxAge: time.Minute}}
	next, err := verifier.consume(context.Background(), proof{ID: "synthetic-proof-id", IssuedAt: now.Unix()}, "synthetic-authority", "synthetic-thumbprint", now.Add(time.Minute))
	if next != "" || !errors.Is(err, ErrInvalid) || store.challenges != 1 || store.consumes != 0 {
		t.Fatal("challenge refusal disclosed or consumed a nonce")
	}
}
