package deliver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

type deliveryRoundTrip func(*http.Request) (*http.Response, error)

func (fn deliveryRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestDefaultPublicationEndpointsPreserveSingleTargetScope(t *testing.T) {
	for _, test := range []struct {
		kind string
		host string
		path string
		body string
	}{
		{"cloudflare", "api.cloudflare.com", "/client/v4/accounts/account/workers/scripts/script/secrets", `{"success":true}`},
		{"netlify", "api.netlify.com", "/api/v1/accounts/account/env/BUNDLE", ``},
	} {
		t.Run(test.kind, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: deliveryRoundTrip(func(request *http.Request) (*http.Response, error) {
				calls++
				if request.Method != http.MethodPut || request.URL.Scheme != "https" || request.URL.Host != test.host || request.URL.Path != test.path {
					t.Errorf("publication escaped the selected default target: %s %s", request.Method, request.URL)
				}
				if request.Header.Get("Authorization") != "Bearer synthetic-token" {
					t.Error("publication omitted the explicit synthetic credential")
				}
				if test.kind == "netlify" && request.URL.Query().Get("site_id") != "site" {
					t.Error("default Netlify endpoint lost the selected site")
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})}
			target := model.Target{Type: test.kind, Account: "account", Resource: "script", Project: "site", Environment: "production", Path: "BUNDLE", TokenFile: deliveryToken(t), AcceptSnapshot: true}
			if err := ApplyWithClient(context.Background(), client, target, generation()); err != nil || calls != 1 {
				t.Fatalf("default endpoint did not perform exactly one scoped publication: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestInvalidPublicationPathsAreDeniedBeforeTransport(t *testing.T) {
	for _, kind := range []string{"cloudflare", "netlify", "windmill"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: deliveryRoundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, errors.New("transport must not be reached")
			})}
			target := model.Target{Type: kind, Endpoint: "https://provider.example.test", Account: "account", Resource: "../escape", Project: "synthetic", Environment: "production", Path: "../escape", TokenFile: deliveryToken(t), AcceptSnapshot: true}
			if err := ApplyWithClient(context.Background(), client, target, generation()); !fault.Is(err, fault.Invalid) || calls != 0 {
				t.Fatalf("invalid provider path reached transport: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestWindmillReadFailureDoesNotAttemptPublication(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: deliveryRoundTrip(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != http.MethodGet || request.URL.Path != "/api/w/synthetic/variables/exists/f/synthetic/config" {
			t.Errorf("read failure led to an unexpected effect: %s %s", request.Method, request.URL)
		}
		return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("synthetic-provider-error"))}, nil
	})}
	target := model.Target{Type: "windmill", Endpoint: "https://provider.example.test", Project: "synthetic", Resource: "f/synthetic/config", TokenFile: deliveryToken(t), AcceptSnapshot: true}
	err := ApplyWithClient(context.Background(), client, target, generation())
	if !fault.Is(err, fault.Denied) || calls != 1 {
		t.Fatalf("failed existence read was followed by publication: calls=%d err=%v", calls, err)
	}
	if strings.Contains(err.Error(), "synthetic-provider-error") {
		t.Fatal("provider failure escaped its sanitized category")
	}
}
