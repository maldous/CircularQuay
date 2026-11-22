package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

func TestDPoPMetadataDescribesEveryAuthorityRequirement(t *testing.T) {
	for _, name := range []string{"bearer", "dpop", "mixed"} {
		t.Run(name, func(t *testing.T) {
			config := dpopMetadataPolicy(t, name)
			service := makeService(t, config, &fakeBackend{}, Options{Audit: &memoryAudit{}, DPoPStateDirectory: filepath.Dir(privateAuditPath(t))})
			handler, err := service.httpHandler(func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
				t.Error("public metadata attempted token verification")
				return nil, auth.ErrInvalidToken
			})
			if err != nil {
				t.Fatal(err)
			}
			assertDPoPMetadata(t, handler, name)
		})
	}
}

func dpopMetadataPolicy(t *testing.T, profile string) model.Policy {
	t.Helper()
	if profile == "bearer" {
		return namedAuthorityServicePolicy(t)
	}
	config := senderConstrainedPolicy(t)
	if profile == "mixed" {
		ordinary := config.MCP.Authorities["workforce"]
		ordinary.SenderConstraint = nil
		ordinary.Audience = "other-registered-api"
		config.MCP.Authorities["ordinary"] = ordinary
	}
	return config
}

func assertDPoPMetadata(t *testing.T, handler http.Handler, profile string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "https://cq.example.test/.well-known/oauth-protected-resource/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var metadata oauthex.ProtectedResourceMetadata
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &metadata) != nil {
		t.Fatalf("protected-resource metadata unavailable: status=%d", response.Code)
	}
	algorithms := []string{"ES256", "RS256"}
	if profile == "bearer" {
		algorithms = nil
	}
	if !slices.Equal(metadata.DPOPSigningAlgValuesSupported, algorithms) || metadata.DPOPBoundAccessTokensRequired != (profile == "dpop") {
		t.Fatalf("metadata disagrees with configured profile %q", profile)
	}
}
