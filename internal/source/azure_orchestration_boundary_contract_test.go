package source

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestAzureOrchestrationBoundaryContractAdmissionPrecedesDependencies(t *testing.T) {
	for _, test := range []struct {
		name     string
		endpoint string
		resource string
		kind     fault.Kind
	}{
		{"endpoint", "https://vault.synthetic?", "fixture", fault.Trust},
		{"name", syntheticAzureEndpoint, "other/fixture", fault.Invalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			credential := func(model.Source) (azcore.TokenCredential, error) {
				t.Fatal("refused source reached credential selection")
				return nil, nil
			}
			client := func(string, azcore.TokenCredential, *azsecrets.ClientOptions) (*azsecrets.Client, error) {
				t.Fatal("refused source reached SDK construction")
				return nil, nil
			}
			value, err := readAzureWithClients(context.Background(), model.Source{Endpoint: test.endpoint}, model.Binding{Resource: test.resource}, credential, client, azureOrchestrationTripwire(t))
			assertRejectedProviderValue(t, value, err, test.kind, "PRIVATE_DATA")
		})
	}
}

func TestAzureOrchestrationBoundaryContractCredentialFailureStopsConstruction(t *testing.T) {
	wanted := fault.New(fault.Denied, "azure_identity")
	credentialCalls := 0
	credential := func(source model.Source) (azcore.TokenCredential, error) {
		credentialCalls++
		if source.Endpoint != syntheticAzureEndpoint || source.Account != "synthetic-tenant" {
			t.Fatal("identity selector lost the admitted source")
		}
		return nil, wanted
	}
	client := func(string, azcore.TokenCredential, *azsecrets.ClientOptions) (*azsecrets.Client, error) {
		t.Fatal("identity refusal reached SDK construction")
		return nil, nil
	}
	value, err := readAzureWithClients(context.Background(), azureOrchestrationSource(), azureOrchestrationBinding(), credential, client, azureOrchestrationTripwire(t))
	assertRejectedProviderValue(t, value, err, fault.Denied, "PRIVATE_DATA")
	if err != wanted || credentialCalls != 1 {
		t.Fatalf("identity failure was replaced or retried: %v, calls=%d", err, credentialCalls)
	}
}

func TestAzureOrchestrationBoundaryContractConstructorFailureIsSanitized(t *testing.T) {
	calls := 0
	client := func(endpoint string, credential azcore.TokenCredential, options *azsecrets.ClientOptions) (*azsecrets.Client, error) {
		calls++
		assertAzureOrchestrationOptions(t, endpoint, credential, options)
		return nil, errors.New("PRIVATE_DATA")
	}
	value, err := readAzureWithClients(context.Background(), azureOrchestrationSource(), azureOrchestrationBinding(), azureOrchestrationCredential, client, azureOrchestrationTripwire(t))
	assertRejectedProviderValue(t, value, err, fault.Invalid, "PRIVATE_DATA")
	if err.Error() != "invalid_policy: azure_client" || calls != 1 {
		t.Fatalf("SDK failure shape or call count changed: %v, calls=%d", err, calls)
	}
}

func TestAzureOrchestrationBoundaryContractGeneratedSDKUsesContainedTransport(t *testing.T) {
	calls := 0
	closed := false
	base := cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		calls++
		assertAzureOrchestrationRequest(t, request)
		body := &azureOrchestrationBody{Reader: strings.NewReader(`{"value":"SYNTHETIC_AZURE_VALUE","id":"https://vault.synthetic/secrets/fixture/expected"}`), closed: &closed}
		return &http.Response{StatusCode: http.StatusOK, Request: request, Header: http.Header{"Content-Type": {"application/json"}}, Body: body}, nil
	})
	value, err := readAzureWithClients(context.Background(), azureOrchestrationSource(), azureOrchestrationBinding(), azureOrchestrationCredential, azsecrets.NewClient, base)
	if err != nil || value.Data != "SYNTHETIC_AZURE_VALUE" || value.Version != "expected" || !value.AuditVerified {
		t.Fatalf("admitted generated-SDK value was not preserved: %#v, %v", value, err)
	}
	if calls != 1 || !closed {
		t.Fatalf("SDK transport escaped one-request/closed-body lifetime: calls=%d closed=%t", calls, closed)
	}
}

func TestAzureOrchestrationBoundaryContractProviderRefusalDoesNotReleaseValue(t *testing.T) {
	calls := 0
	base := cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		calls++
		assertAzureOrchestrationRequest(t, request)
		return &http.Response{StatusCode: http.StatusForbidden, Request: request, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"Forbidden","message":"PRIVATE_DATA"}}`))}, nil
	})
	value, err := readAzureWithClients(context.Background(), azureOrchestrationSource(), azureOrchestrationBinding(), azureOrchestrationCredential, azsecrets.NewClient, base)
	assertRejectedProviderValue(t, value, err, fault.Denied, "PRIVATE_DATA")
	if calls != 1 {
		t.Fatalf("denied request was retried: %d", calls)
	}
}

func azureOrchestrationSource() model.Source {
	return model.Source{Endpoint: syntheticAzureEndpoint, Account: "synthetic-tenant", AuditVerified: true}
}

func azureOrchestrationBinding() model.Binding {
	return model.Binding{Resource: "fixture", Version: "expected"}
}

func azureOrchestrationCredential(model.Source) (azcore.TokenCredential, error) {
	return syntheticSDKAccessToken{}, nil
}

func azureOrchestrationTripwire(t *testing.T) http.RoundTripper {
	t.Helper()
	return cloudRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("pre-request refusal reached a transport")
		return nil, nil
	})
}

func assertAzureOrchestrationOptions(t *testing.T, endpoint string, credential azcore.TokenCredential, options *azsecrets.ClientOptions) {
	t.Helper()
	if endpoint != syntheticAzureEndpoint || credential == nil || options == nil {
		t.Fatal("SDK constructor lost admitted source or credentials")
	}
	client, ok := options.Transport.(*http.Client)
	if !ok || client.Transport == nil || client.CheckRedirect == nil {
		t.Fatal("SDK constructor omitted hardened explicit transport")
	}
	if client.CheckRedirect(nil, nil) == nil {
		t.Fatal("SDK constructor permits redirects")
	}
}

func assertAzureOrchestrationRequest(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Method != http.MethodGet || request.URL.Scheme != "https" || request.URL.Host != "vault.synthetic" {
		t.Fatalf("request escaped fixed synthetic origin: %s", request.URL)
	}
	if request.URL.Path != "/secrets/fixture/expected" || request.URL.Query().Get("api-version") == "" {
		t.Fatal("generated SDK did not retain resource/version parameters")
	}
}

type azureOrchestrationBody struct {
	*strings.Reader
	closed *bool
}

func (body *azureOrchestrationBody) Close() error {
	*body.closed = true
	return nil
}
