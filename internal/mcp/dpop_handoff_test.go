package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

func TestVerifiedTokenHandoffRequiresExactRequestToken(t *testing.T) {
	info := &auth.TokenInfo{UserID: "synthetic-user"}
	for name, cached := range map[string]any{
		"absent":           nil,
		"wrong type":       "synthetic-token",
		"other token":      verifiedToken{raw: "other-token", info: info},
		"missing identity": verifiedToken{raw: "synthetic-token"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if cached != nil {
				ctx = context.WithValue(ctx, verifiedTokenKey{}, cached)
			}
			if result, err := cachedTokenVerifier()(ctx, "synthetic-token", nil); result != nil || !errors.Is(err, auth.ErrInvalidToken) {
				t.Fatalf("unverified SDK handoff accepted: result=%v err=%v", result != nil, err)
			}
		})
	}
	ctx := context.WithValue(context.Background(), verifiedTokenKey{}, verifiedToken{raw: "synthetic-token", info: info})
	if result, err := cachedTokenVerifier()(ctx, "synthetic-token", nil); result != info || err != nil {
		t.Fatalf("verified SDK handoff failed: result=%v err=%v", result == info, err)
	}
}
