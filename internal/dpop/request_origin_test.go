package dpop

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerifierRejectsDifferentRequestOriginBeforeStore(t *testing.T) {
	now := proofNow()
	key := proofECKey(t)
	proof, jkt := validProof(t, key, now, "nonce")
	for name, target := range map[string]string{
		"host alias": "https://alias.example.test/mcp",
		"scheme":     "http://mcp.example.test/mcp",
		"port":       "https://mcp.example.test:8443/mcp",
	} {
		t.Run(name, func(t *testing.T) {
			request := proofRequest(t, target, proof)
			request.Header.Set("Forwarded", "proto=https;host=mcp.example.test")
			request.Header.Set("X-Forwarded-Host", "mcp.example.test")
			request.Header.Set("X-Forwarded-Proto", "https")
			store := &proofStore{}
			_, err := proofVerifier(now, store).Verify(context.Background(), request, "synthetic-token", "authority", jkt, now.Add(time.Minute))
			if err == nil || store.challenges != 0 || store.consumes != 0 {
				t.Fatalf("different request origin reached store: %v", err)
			}
		})
	}
}

func TestVerifierAcceptsOriginFormWithPreservedExternalHost(t *testing.T) {
	now := proofNow()
	key := proofECKey(t)
	proof, jkt := validProof(t, key, now, "nonce")
	request := httptest.NewRequest("POST", "/mcp", nil)
	request.Host = "mcp.example.test"
	request.Header.Set("DPoP", proof)
	store := &proofStore{}
	if _, err := proofVerifier(now, store).Verify(context.Background(), request, "synthetic-token", "authority", jkt, now.Add(time.Minute)); err != nil || store.consumes != 1 {
		t.Fatalf("preserved external origin was denied: %v", err)
	}
}
