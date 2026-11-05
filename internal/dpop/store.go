package dpop

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/state"
)

const (
	cacheName         = "dpop-replay"
	cacheFile         = "dpop-replay.json"
	maxCacheSize      = 4096
	maxBindingEntries = 64
)

// PersistentStore is a protected single-replica replay and nonce store. It
// survives restart but must never be shared by independently running cq
// processes; clustered deployments need an atomic shared store before DPoP is
// enabled.
type PersistentStore struct {
	directory string
	now       func() time.Time
	random    io.Reader
}

type persistedCache struct {
	Version uint64                `json:"version"`
	Entries map[string]cacheEntry `json:"entries"`
}

type cacheEntry struct {
	Kind    string `json:"kind"`
	Binding string `json:"binding,omitempty"`
	Expires int64  `json:"expires"`
}

// NewPersistentStore establishes the private directory early so an insecure
// state location fails before an HTTP listener accepts a sender-constrained request.
func NewPersistentStore(stateDirectory string) (*PersistentStore, error) {
	if !filepath.IsAbs(stateDirectory) {
		return nil, ErrInvalid
	}
	directory := filepath.Join(stateDirectory, "dpop")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		return nil, err
	}
	return &PersistentStore{directory: directory, now: time.Now, random: rand.Reader}, nil
}

// Challenge stores a nonce bound to one authority, key thumbprint and external
// resource. The nonce itself is never written to disk.
func (s *PersistentStore) Challenge(ctx context.Context, binding string, expires time.Time) (string, error) {
	if !validStoreInput(ctx, binding, expires) {
		return "", ErrInvalid
	}
	nonce, err := s.newNonce()
	if err != nil {
		return "", err
	}
	err = s.update(ctx, func(cache *persistedCache, now time.Time) error {
		prune(cache, now)
		if !future(expires, now) || len(cache.Entries) >= maxCacheSize || bindingEntries(cache, binding) >= maxBindingEntries {
			return ErrInvalid
		}
		cache.Entries[nonceKey(nonce)] = cacheEntry{Kind: "nonce", Binding: binding, Expires: expires.Unix()}
		return nil
	})
	if err != nil {
		return "", err
	}
	if !future(expires, s.clock()) {
		return "", ErrInvalid
	}
	return nonce, nil
}

// Consume atomically verifies a nonce, consumes it, records the proof ID and
// creates the nonce to be used by the next request.
func (s *PersistentStore) Consume(ctx context.Context, binding, nonce, replay string, expires time.Time) (string, error) {
	if !validStoreInput(ctx, binding, expires) || !validNonce(nonce) || !validDigest(replay) {
		return "", ErrInvalid
	}
	next, err := s.newNonce()
	if err != nil {
		return "", err
	}
	err = s.update(ctx, func(cache *persistedCache, now time.Time) error {
		return consumeCache(cache, binding, nonce, replay, next, expires, now)
	})
	if err != nil {
		return "", err
	}
	if !future(expires, s.clock()) {
		return "", ErrInvalid
	}
	return next, nil
}

func consumeCache(cache *persistedCache, binding, nonce, replay, next string, expires, now time.Time) error {
	prune(cache, now)
	if !future(expires, now) || bindingEntries(cache, binding)+1 > maxBindingEntries {
		return ErrInvalid
	}
	if !nonceAvailable(cache, binding, nonce, replay, now) {
		return ErrInvalid
	}
	delete(cache.Entries, nonceKey(nonce))
	if len(cache.Entries)+2 > maxCacheSize {
		return ErrInvalid
	}
	cache.Entries[replay] = cacheEntry{Kind: "replay", Binding: binding, Expires: expires.Unix()}
	cache.Entries[nonceKey(next)] = cacheEntry{Kind: "nonce", Binding: binding, Expires: expires.Unix()}
	return nil
}

func nonceAvailable(cache *persistedCache, binding, nonce, replay string, now time.Time) bool {
	if _, seen := cache.Entries[replay]; seen {
		return false
	}
	entry, found := cache.Entries[nonceKey(nonce)]
	return found && entry.Kind == "nonce" && entry.Binding == binding && entry.Expires > now.Unix()
}

func validStoreInput(ctx context.Context, binding string, expires time.Time) bool {
	return ctx != nil && ctx.Err() == nil && validDigest(binding) && !expires.IsZero()
}

func (s *PersistentStore) update(ctx context.Context, change func(*persistedCache, time.Time) error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	unlock, err := (state.Store{Directory: s.directory}).Lock(cacheName)
	if err != nil {
		return err
	}
	defer unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	cache, err := s.read()
	if err != nil {
		return err
	}
	if err := change(&cache, s.clock()); err != nil {
		return err
	}
	cache.Version = 1
	data, err := json.Marshal(cache)
	if err != nil || len(data) > platform.MaxDocumentSize {
		return ErrInvalid
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return platform.WriteProtected(filepath.Join(s.directory, cacheFile), data)
}

func (s *PersistentStore) read() (persistedCache, error) {
	path := filepath.Join(s.directory, cacheFile)
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return persistedCache{Version: 1, Entries: make(map[string]cacheEntry)}, nil
	} else if err != nil {
		return persistedCache{}, err
	}
	data, err := platform.ReadProtected(path, true)
	if err != nil {
		return persistedCache{}, err
	}
	var cache persistedCache
	if jsondocument.Decode(data, &cache) != nil || cache.Version != 1 || cache.Entries == nil || len(cache.Entries) > maxCacheSize || !validCache(cache) {
		return persistedCache{}, ErrInvalid
	}
	return cache, nil
}

func validCache(cache persistedCache) bool {
	for key, entry := range cache.Entries {
		if !validEntryKey(key, entry) || entry.Expires <= 0 || !validDigest(entry.Binding) {
			return false
		}
	}
	return true
}

func future(expires, now time.Time) bool { return !expires.IsZero() && now.Before(expires) }

func bindingEntries(cache *persistedCache, binding string) int {
	count := 0
	for _, entry := range cache.Entries {
		if entry.Binding == binding {
			count++
		}
	}
	return count
}

func validEntryKey(key string, entry cacheEntry) bool {
	if entry.Kind == "nonce" {
		return len(key) == 45 && strings.HasPrefix(key, "n:") && validDigest(key[2:])
	}
	return entry.Kind == "replay" && validDigest(key)
}

func validDigest(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && base64.RawURLEncoding.EncodeToString(decoded) == value
}

func prune(cache *persistedCache, now time.Time) {
	for key, entry := range cache.Entries {
		if entry.Expires <= now.Unix() {
			delete(cache.Entries, key)
		}
	}
}

func (s *PersistentStore) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *PersistentStore) newNonce() (string, error) {
	var value [32]byte
	if s.random == nil {
		return "", ErrInvalid
	}
	if _, err := io.ReadFull(s.random, value[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

func validNonce(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == 32 && base64.RawURLEncoding.EncodeToString(decoded) == value
}

func nonceKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "n:" + base64.RawURLEncoding.EncodeToString(sum[:])
}
