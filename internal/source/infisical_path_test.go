package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestInfisicalPathRejectsBeforeCredentialAndRequest(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	registry := NewRegistryWithClient(server.Client())
	for _, secretPath := range []string{"", "relative", "//duplicate", "/./dot", "/parent/../escape", "/trailing/", "/query?value", string([]byte{'/', 0xff})} {
		t.Run(secretPath, func(t *testing.T) {
			_, err := registry.Read(context.Background(), model.Source{
				Type: "infisical", Endpoint: server.URL, Project: "project", Environment: "test",
				Path: secretPath, TokenFile: "/not/read/for/invalid/infisical/path", Resources: []string{"KEY"},
			}, model.Binding{Name: "KEY", Resource: "KEY"})
			if !fault.Is(err, fault.Invalid) {
				t.Fatalf("invalid path %q error = %v", secretPath, err)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatal("invalid Infisical path reached provider")
	}
}

func TestInfisicalPathCanonicalScopes(t *testing.T) {
	for _, secretPath := range []string{"/", "/application", "/application/production"} {
		if !validInfisicalPath(secretPath) {
			t.Fatalf("canonical Infisical path rejected: %q", secretPath)
		}
	}
}
