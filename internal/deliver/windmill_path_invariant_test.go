package deliver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

type windmillPathTransport func(*http.Request) (*http.Response, error)

func (roundTrip windmillPathTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestWindmillPathInvariantCopiedDynamicStringsRemainBoundAfterLookup(t *testing.T) {
	for _, resource := range []string{"key", "folder/key", "percent%20key", "space key", "unicode-ü"} {
		for _, exists := range []bool{false, true} {
			t.Run(resource, func(t *testing.T) { windmillCopiedPath(t, resource, exists) })
		}
	}
}

func windmillCopiedPath(t *testing.T, resource string, exists bool) {
	t.Helper()
	target := model.Target{Endpoint: "https://windmill.invalid", Project: "owned-workspace", Resource: resource}
	calls := 0
	client := transport.Client{HTTP: &http.Client{Transport: windmillPathTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		assertWindmillPathCredentials(t, request)
		if calls == 1 {
			if request.Method != http.MethodGet || request.URL.Path != "/api/w/owned-workspace/variables/exists/"+resource {
				t.Fatal("lookup changed its exact admitted path")
			}
			// An injected transport may change the caller's outer Target, but it
			// cannot change the copied strings held by the current operation.
			target.Project, target.Resource, target.Endpoint = "bad#project", "bad#resource", "http://untrusted.invalid"
			return windmillPathResponse(request, map[bool]string{false: "false", true: "true"}[exists]), nil
		}
		assertWindmillPathPublication(t, request, resource, exists, calls)
		return windmillPathResponse(request, "{}"), nil
	})}}
	if err := windmill(context.Background(), client, target, "synthetic-token", "owned-payload"); err != nil || calls != 2 {
		t.Fatal("admitted immutable path failed its exact two-request control")
	}
}

func assertWindmillPathCredentials(t *testing.T, request *http.Request) {
	t.Helper()
	if request.URL.Scheme != "https" || request.URL.Host != "windmill.invalid" || request.URL.RawQuery != "" || request.URL.User != nil || request.Header.Get("Authorization") != "Bearer synthetic-token" {
		t.Fatal("publication changed its admitted origin or credential binding")
	}
}

func assertWindmillPathPublication(t *testing.T, request *http.Request, resource string, exists bool, calls int) {
	t.Helper()
	want := "/api/w/owned-workspace/variables/create"
	if exists {
		want = "/api/w/owned-workspace/variables/update/" + resource
	}
	if calls != 2 || request.Method != http.MethodPost || request.URL.Path != want {
		t.Fatal("publication changed route, method or request count after lookup")
	}
	data, err := io.ReadAll(io.LimitReader(request.Body, 4097))
	if err != nil || len(data) > 4096 {
		t.Fatal("publication exceeded its bounded owned request")
	}
	assertWindmillPathBody(t, data, resource, exists)
}

func assertWindmillPathBody(t *testing.T, data []byte, resource string, exists bool) {
	t.Helper()
	var body map[string]any
	if json.Unmarshal(data, &body) != nil || body["value"] != "owned-payload" || body["is_secret"] != true {
		t.Fatal("publication lost exact value or secret declaration")
	}
	if exists {
		if len(body) != 2 {
			t.Fatal("update widened its declared body")
		}
	} else if len(body) != 3 || body["path"] != resource {
		t.Fatal("create lost its exact resource binding")
	}
}

func windmillPathResponse(request *http.Request, data string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(data)), Header: http.Header{"Content-Type": {"application/json"}}, Request: request}
}
