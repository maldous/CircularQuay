package deliver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func deliveryToken(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(p, []byte("synthetic-token"), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func generation() model.Generation {
	return model.Generation{Workload: "synthetic", Values: map[string]model.Value{"KEY": {Data: "synthetic-value"}}}
}

func TestPublicationRejectsUnauthorizedAndLeasedValues(t *testing.T) {
	ctx := context.Background()
	target := model.Target{Type: "cloudflare"}
	if !fault.Is(Apply(ctx, target, generation()), fault.Denied) {
		t.Fatal("publication allowed without grant")
	}
	target.AcceptSnapshot = true
	if !fault.Is(Apply(ctx, target, model.Generation{}), fault.Invalid) {
		t.Fatal("empty generation allowed")
	}
	for _, v := range []model.Value{{Data: "value", Leased: true}, {Data: "value", Renewable: true}, {Data: "value", ExpiresAt: time.Now().Add(time.Hour)}} {
		g := generation()
		g.Values["KEY"] = v
		if !fault.Is(Apply(ctx, target, g), fault.Denied) {
			t.Fatal("expiring secret published")
		}
	}
	if !fault.Is(Apply(ctx, target, generation()), fault.Invalid) {
		t.Fatal("missing credential configuration allowed")
	}
	if !fault.Is(ValidateLifecycle(model.Generation{Values: map[string]model.Value{"KEY": {ExpiresAt: time.Now().Add(-time.Second)}}}, time.Now()), fault.Expired) {
		t.Fatal("expired generation accepted")
	}
	if ValidateLifecycle(generation(), time.Now()) != nil {
		t.Fatal("static generation rejected")
	}
}

func TestCloudflarePublicationSingleBundle(t *testing.T) {
	requests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "PUT" || r.URL.Path != "/client/v4/accounts/account/workers/scripts/script/secrets" || r.Header.Get("Authorization") != "Bearer synthetic-token" {
			t.Error("publication escaped target")
		}
		var body map[string]string
		if json.NewDecoder(r.Body).Decode(&body) != nil || body["name"] != "CQUAY_CONFIG" || body["type"] != "secret_text" {
			t.Error("incorrect binding")
		}
		var values map[string]string
		if json.Unmarshal([]byte(body["text"]), &values) != nil || values["KEY"] != "synthetic-value" {
			t.Error("incomplete bundle")
		}
		fmt.Fprint(w, `{"success":true}`)
	}))
	defer server.Close()
	target := model.Target{Type: "cloudflare", Endpoint: server.URL, Account: "account", Resource: "script", TokenFile: deliveryToken(t), AcceptSnapshot: true}
	if err := ApplyWithClient(context.Background(), server.Client(), target, generation()); err != nil || requests != 1 {
		t.Fatal("single bundle publication failed", err)
	}
	target.Account = ""
	if !fault.Is(ApplyWithClient(context.Background(), server.Client(), target, generation()), fault.Invalid) {
		t.Fatal("unscoped publication accepted")
	}
}

func TestNetlifyPublicationScope(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && r.URL.Path == "/api/v1/accounts/account/env/BUNDLE" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != "POST" || r.URL.Path != "/api/v1/accounts/account/env" || r.URL.Query().Get("site_id") != "site" {
			t.Error("Netlify scope mismatch")
		}
		var body []struct {
			Key    string   `json:"key"`
			Secret bool     `json:"is_secret"`
			Scopes []string `json:"scopes"`
			Values []struct {
				Context string `json:"context"`
				Value   string `json:"value"`
			} `json:"values"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 1 || !body[0].Secret || body[0].Values[0].Context != "production" {
			t.Error("context or secret protection missing")
		}
		w.WriteHeader(201)
	}))
	defer server.Close()
	target := model.Target{Type: "netlify", Endpoint: server.URL, Account: "account", Project: "site", Environment: "production", Path: "BUNDLE", TokenFile: deliveryToken(t), AcceptSnapshot: true}
	if err := ApplyWithClient(context.Background(), server.Client(), target, generation()); err != nil {
		t.Fatal("Netlify contract failed", err)
	}
}

func TestWindmillCreateAndUpdate(t *testing.T) {
	for _, exists := range []bool{false, true} {
		calls := 0
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method == "GET" {
				fmt.Fprint(w, exists)
				return
			}
			var body map[string]any
			if json.NewDecoder(r.Body).Decode(&body) != nil || body["is_secret"] != true {
				t.Error("unprotected Windmill variable")
			}
			if exists && !strings.Contains(r.URL.Path, "/update/") {
				t.Error("existing variable not updated")
			}
			if !exists && (body["path"] != "f/synthetic/config" || !strings.HasSuffix(r.URL.Path, "/create")) {
				t.Error("creation scope mismatch")
			}
			w.WriteHeader(201)
		}))
		target := model.Target{Type: "windmill", Endpoint: server.URL, Project: "synthetic", Resource: "f/synthetic/config", TokenFile: deliveryToken(t), AcceptSnapshot: true}
		err := ApplyWithClient(context.Background(), server.Client(), target, generation())
		server.Close()
		if err != nil || calls != 2 {
			t.Fatal("Windmill contract failed", err)
		}
	}
}

func TestPublicationFailureDoesNotReturnValues(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		kind   fault.Kind
	}{{403, "secret-in-upstream-error", fault.Denied}, {200, `{"success":false,"errors":["secret-in-upstream-error"]}`, fault.Apply}, {302, "secret-in-upstream-error", fault.Trust}} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
		target := model.Target{Type: "cloudflare", Endpoint: server.URL, Account: "account", Resource: "script", TokenFile: deliveryToken(t), AcceptSnapshot: true}
		err := ApplyWithClient(context.Background(), server.Client(), target, generation())
		server.Close()
		if !fault.Is(err, tc.kind) || strings.Contains(err.Error(), "secret-in-") {
			t.Fatal("unsafe publication error")
		}
	}
}
