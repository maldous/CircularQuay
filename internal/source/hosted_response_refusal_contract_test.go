package source

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

// All requests terminate in the injected RoundTripper. No listener, DNS,
// external endpoint, credential chain or provider subprocess is involved.
type hostedRefusalRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip hostedRefusalRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

type hostedRefusalResponse struct {
	name, body, version, wantData, wantVersion, operation string
	kind                                                fault.Kind
	defaultEndpoint, unselected                          bool
}

func TestHostedResponseRefusalContractVaultNamespaceAndSelectedValue(t *testing.T) {
	tests := []hostedRefusalResponse{
		{name: "namespaced selected value", version: "2", wantVersion: "2", wantData: "SYNTHETIC_SELECTED",
			body: `{"data":{"data":{"PRIMARY":"SYNTHETIC_SELECTED","OTHER":"SYNTHETIC_UNSELECTED"},"metadata":{"version":2}}}`},
		{name: "selected type withheld", version: "2", kind: fault.Trust, operation: "value_type",
			body: `{"data":{"data":{"PRIMARY":{"OTHER":"SYNTHETIC_UNSELECTED"}},"metadata":{"version":2}}}`},
		{name: "partially decoded response withheld", version: "2", kind: fault.Trust, operation: "response_encoding",
			body: `{"data":{"data":{"PRIMARY":"SYNTHETIC_SELECTED"},"metadata":{"version":2}}} {"extra":true}`},
	}
	runHostedRefusalResponses(t, "vault", tests)
}

func TestHostedResponseRefusalContractDopplerRawDoesNotUseComputedFallback(t *testing.T) {
	tests := []hostedRefusalResponse{
		{name: "default endpoint selected field", defaultEndpoint: true, wantData: "SYNTHETIC_SELECTED",
			body: `{"value":{"raw":"{\"PRIMARY\":\"SYNTHETIC_SELECTED\",\"OTHER\":\"SYNTHETIC_UNSELECTED\"}","computed":"SYNTHETIC_COMPUTED"}}`},
		{name: "empty raw remains valid", unselected: true,
			body: `{"value":{"raw":"","computed":"SYNTHETIC_COMPUTED"}}`},
		{name: "computed only is missing", kind: fault.Missing, operation: "source_value",
			body: `{"value":{"computed":"SYNTHETIC_COMPUTED"}}`},
		{name: "nested value is not coerced", kind: fault.Trust, operation: "value_type",
			body: `{"value":{"raw":"{\"PRIMARY\":{\"OTHER\":\"SYNTHETIC_UNSELECTED\"}}"}}`},
		{name: "partially decoded response withheld", kind: fault.Trust, operation: "response_encoding",
			body: `{"value":{"raw":"SYNTHETIC_PARTIAL"}} trailing`},
	}
	runHostedRefusalResponses(t, "doppler", tests)
}

func TestHostedResponseRefusalContractInfisicalSelectionAndVersionRemainBound(t *testing.T) {
	tests := []hostedRefusalResponse{
		{name: "default endpoint selected field", defaultEndpoint: true, version: "2", wantVersion: "2", wantData: "SYNTHETIC_SELECTED",
			body: `{"secret":{"secretValue":"{\"PRIMARY\":\"SYNTHETIC_SELECTED\",\"OTHER\":\"SYNTHETIC_UNSELECTED\"}","version":2}}`},
		{name: "selected duplicate is refused", version: "2", kind: fault.Trust, operation: "json_duplicate",
			body: `{"secret":{"secretValue":"{\"PRIMARY\":\"SYNTHETIC_SELECTED\",\"PRIMARY\":\"SYNTHETIC_OTHER\"}","version":2}}`},
		{name: "hidden takes precedence over malformed raw and version", version: "2", kind: fault.Denied, operation: "source_value_hidden",
			body: `{"secret":{"secretValue":"SYNTHETIC_NOT_JSON","version":3,"secretValueHidden":true}}`},
		{name: "selected value with wrong returned version is withheld", version: "2", kind: fault.Trust, operation: "source_version_mismatch",
			body: `{"secret":{"secretValue":"{\"PRIMARY\":\"SYNTHETIC_SELECTED\"}","version":3}}`},
	}
	runHostedRefusalResponses(t, "infisical", tests)
}

func TestHostedResponseRefusalContractAkeylessRequestAndSelectionStayExact(t *testing.T) {
	tests := []hostedRefusalResponse{
		{name: "default endpoint selected field", defaultEndpoint: true, wantData: "SYNTHETIC_SELECTED",
			body: `{"fixture":"{\"PRIMARY\":\"SYNTHETIC_SELECTED\",\"OTHER\":\"SYNTHETIC_UNSELECTED\"}","sibling":"SYNTHETIC_SIBLING"}`},
		{name: "sibling does not replace requested resource", kind: fault.Missing, operation: "source_field",
			body: `{"sibling":"SYNTHETIC_SIBLING"}`},
		{name: "requested resource must be string", kind: fault.Trust, operation: "value_type",
			body: `{"fixture":{"PRIMARY":"SYNTHETIC_SELECTED"}}`},
		{name: "malformed selected document withheld", kind: fault.Trust, operation: "json_source",
			body: `{"fixture":"SYNTHETIC_NOT_JSON"}`},
		{name: "partially decoded response withheld", kind: fault.Trust, operation: "response_encoding",
			body: `{"fixture":"SYNTHETIC_PARTIAL"} {"extra":true}`},
	}
	runHostedRefusalResponses(t, "akeyless", tests)
}

func TestHostedResponseRefusalContractLocalRefusalsNeverReachTransport(t *testing.T) {
	tests := []struct {
		name, provider, operation string
		kind                      fault.Kind
		alter                     func(*model.Source, *model.Binding)
	}{
		{"Vault endpoint before missing credential", "vault", "provider_endpoint", fault.Trust, func(source *model.Source, _ *model.Binding) {
			source.Endpoint = "https://provider.synthetic?forbidden=selector"
			source.TokenFile = privateLiteralFile(t, "absent.token")
		}},
		{"Vault credential missing", "vault", "credential_file", fault.Denied, func(source *model.Source, _ *model.Binding) {
			source.TokenFile = privateLiteralFile(t, "absent.token")
		}},
		{"Vault namespace is not header injection", "vault", "credential_format", fault.Invalid, func(source *model.Source, _ *model.Binding) {
			source.Namespace = "synthetic\r\nUNSAFE"
		}},
		{"hosted credential missing", "akeyless", "credential_file", fault.Denied, func(source *model.Source, _ *model.Binding) {
			source.TokenFile = privateLiteralFile(t, "absent.token")
		}},
		{"Doppler project missing", "doppler", "doppler_scope", fault.Invalid, func(source *model.Source, _ *model.Binding) {
			source.Project = ""
		}},
		{"Doppler selector unsupported", "doppler", "source_version_unsupported", fault.Invalid, func(_ *model.Source, binding *model.Binding) {
			binding.Version = "2"
		}},
		{"Infisical project missing", "infisical", "infisical_scope", fault.Invalid, func(source *model.Source, _ *model.Binding) {
			source.Project = ""
		}},
		{"Infisical endpoint invalid", "infisical", "provider_endpoint", fault.Trust, func(source *model.Source, _ *model.Binding) {
			source.Endpoint = "https://provider.synthetic?forbidden=selector"
		}},
		{"Akeyless selector unsupported", "akeyless", "source_version_unsupported", fault.Invalid, func(_ *model.Source, binding *model.Binding) {
			binding.Version = "2"
		}},
		{"Akeyless endpoint invalid", "akeyless", "provider_endpoint", fault.Trust, func(source *model.Source, _ *model.Binding) {
			source.Endpoint = "https://provider.synthetic?forbidden=selector"
		}},
		{"unknown provider after selected credential", "unknown-synthetic", "source_type", fault.Unsupported, func(*model.Source, *model.Binding) {}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, binding := hostedRefusalSelection(t, test.provider)
			test.alter(&source, &binding)
			calls := 0
			client := transport.Client{HTTP: &http.Client{Transport: hostedRefusalRoundTripper(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, errors.New("SYNTHETIC_UNEXPECTED_REQUEST")
			})}}
			value, err := readHostedRefusalProvider(client, test.provider, source, binding)
			assertHostedRefusal(t, value, err, test.kind, test.operation)
			if calls != 0 {
				t.Fatal("local refusal reached the injected transport")
			}
		})
	}
}

func runHostedRefusalResponses(t *testing.T, provider string, tests []hostedRefusalResponse) {
	t.Helper()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, binding := hostedRefusalSelection(t, provider)
			binding.Version = test.version
			if test.unselected {
				binding.Field = ""
			}
			if test.defaultEndpoint {
				source.Endpoint = ""
			}
			client, calls := hostedRefusalResponseClient(t, provider, source, binding, test.body)
			value, err := readHostedRefusalProvider(client, provider, source, binding)
			if test.kind != "" {
				assertHostedRefusal(t, value, err, test.kind, test.operation)
			} else if err != nil || value != (model.Value{Data: test.wantData, Version: test.wantVersion, AuditVerified: true}) {
				t.Fatal("selected response did not retain only its declared value and metadata")
			}
			if *calls != 1 {
				t.Fatalf("in-process request count = %d, want 1", *calls)
			}
		})
	}
}

func hostedRefusalSelection(t *testing.T, provider string) (model.Source, model.Binding) {
	t.Helper()
	return model.Source{Type: provider, Endpoint: "https://provider.synthetic", Mount: "kv", Namespace: "synthetic-namespace",
		Project: "synthetic-project", Environment: "synthetic-environment", Path: "/application", TokenFile: tokenFixture(t), AuditVerified: true},
		model.Binding{Name: "IGNORED_NAME", Resource: "fixture", Field: "PRIMARY"}
}

func readHostedRefusalProvider(client transport.Client, provider string, source model.Source, binding model.Binding) (model.Value, error) {
	if provider == "vault" {
		return readVault(context.Background(), client, source, binding)
	}
	return readHosted(context.Background(), client, provider, source, binding)
}

func hostedRefusalResponseClient(t *testing.T, provider string, source model.Source, binding model.Binding, body string) (transport.Client, *int) {
	t.Helper()
	calls := new(int)
	client := transport.Client{HTTP: &http.Client{Transport: hostedRefusalRoundTripper(func(request *http.Request) (*http.Response, error) {
		(*calls)++
		assertHostedRefusalRequest(t, request, provider, source.Endpoint == "", binding.Version)
		return &http.Response{Request: request, StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	return client, calls
}

func assertHostedRefusalRequest(t *testing.T, request *http.Request, provider string, defaultEndpoint bool, version string) {
	t.Helper()
	host := "provider.synthetic"
	if defaultEndpoint {
		host = map[string]string{"doppler": "api.doppler.com", "infisical": "us.infisical.com", "akeyless": "api.akeyless.io"}[provider]
	}
	if request.URL.Scheme != "https" || request.URL.Host != host || request.Header.Get("Accept") != "application/json" {
		t.Fatal("request escaped the selected synthetic origin or response contract")
	}
	wantPath := map[string]string{"vault": "/v1/kv/data/fixture", "doppler": "/v3/configs/config/secret", "infisical": "/api/v4/secrets/fixture", "akeyless": "/get-secret-value"}[provider]
	if request.URL.Path != wantPath {
		t.Fatal("request did not select the exact resource path")
	}
	query := url.Values{}
	switch provider {
	case "vault":
		if request.Header.Get("X-Vault-Token") != "synthetic-token" || request.Header.Get("X-Vault-Namespace") != "synthetic-namespace" {
			t.Fatal("Vault request lost its selected token or namespace")
		}
	case "doppler":
		query = url.Values{"project": {"synthetic-project"}, "config": {"synthetic-environment"}, "name": {"fixture"}}
	case "infisical":
		query = url.Values{"projectId": {"synthetic-project"}, "environment": {"synthetic-environment"}, "secretPath": {"/application"}, "expandSecretReferences": {"false"}, "includeImports": {"false"}}
	}
	if version != "" {
		query.Set("version", version)
	}
	if request.URL.Query().Encode() != query.Encode() {
		t.Fatal("request broadened or lost its declared selectors")
	}
	if provider == "akeyless" {
		assertHostedRefusalAkeylessBody(t, request)
		return
	}
	if request.Method != http.MethodGet || request.Body != nil {
		t.Fatal("read-only provider request changed method or gained a body")
	}
	if provider != "vault" && request.Header.Get("Authorization") != "Bearer synthetic-token" {
		t.Fatal("hosted request lost its selected bearer token")
	}
}

func assertHostedRefusalAkeylessBody(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" || request.Header.Get("Authorization") != "" {
		t.Fatal("Akeyless token escaped its intended JSON request body")
	}
	var payload struct {
		Names []string `json:"names"`
		Token string   `json:"token"`
	}
	decoder := json.NewDecoder(io.LimitReader(request.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&payload) != nil || len(payload.Names) != 1 || payload.Names[0] != "fixture" || payload.Token != "synthetic-token" {
		t.Fatal("Akeyless request did not minimize its selected token and resource")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		t.Fatal("Akeyless request contained trailing body content")
	}
}

func assertHostedRefusal(t *testing.T, value model.Value, err error, kind fault.Kind, operation string) {
	t.Helper()
	if !fault.Is(err, kind) || err.Error() != string(kind)+": "+operation {
		t.Fatalf("wrong or unsafe hosted failure for %s", operation)
	}
	if value != (model.Value{}) {
		t.Fatal("failed hosted response released partial data or metadata")
	}
}
