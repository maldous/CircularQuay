//go:build !windows

package dpop

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestStoreFailsClosedForUnprotectedCache(t *testing.T) {
	s, now := testStore(t)
	if _, err := s.Challenge(context.Background(), digest("binding"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cachePath(s), 0o644); err != nil {
		t.Fatal(err)
	}
	assertChallengeFails(t, s, now)
}
