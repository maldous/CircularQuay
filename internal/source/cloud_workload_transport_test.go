package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	azcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

const syntheticAuthorityHost = "https://authority.synthetic"

func TestAzureWorkloadIdentityTransportRejectsRedirectAndOversizeResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		response func() *http.Response
		want     error
	}{
		{
			name: "redirect",
			response: func() *http.Response {
				return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {"https://redirect.invalid/token"}}, Body: io.NopCloser(strings.NewReader(""))}
			},
			want: transport.DenyRedirect(nil, nil),
		},
		{
			name: "oversize response",
			response: func() *http.Response {
				return &http.Response{StatusCode: http.StatusOK, ContentLength: 4<<20 + 1, Body: io.NopCloser(strings.NewReader(""))}
			},
			want: transport.ErrResponseTooLarge,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			credential, requests := syntheticWorkloadCredential(t, test.response)
			if _, err := credential.GetToken(context.Background(), azpolicy.TokenRequestOptions{Scopes: []string{"https://vault.azure.net/.default"}}); err == nil || !strings.Contains(err.Error(), test.want.Error()) {
				t.Fatalf("GetToken() error = %v, want transport error containing %q", err, test.want)
			}
			if requests.metadata != 1 || requests.token != 1 || requests.redirect != 0 {
				t.Fatalf("requests = %+v, want one metadata request, one token request, and no redirect replay", requests)
			}
		})
	}
}

type workloadRequests struct{ metadata, token, redirect int }

func syntheticWorkloadCredential(t *testing.T, tokenResponse func() *http.Response) (*azidentity.WorkloadIdentityCredential, *workloadRequests) {
	t.Helper()
	requests := &workloadRequests{}
	source := model.Source{Account: "synthetic-tenant", Profile: "synthetic-client", TokenFile: privateCloudTokenFile(t)}
	options := azureWorkloadCredentialOptions(source, workloadTransport(t, requests, tokenResponse))
	if options.ClientOptions.Transport == nil {
		t.Fatal("production workload options omitted the bounded transport")
	}
	options.DisableInstanceDiscovery = true
	options.ClientOptions.Cloud = azcloud.Configuration{ActiveDirectoryAuthorityHost: syntheticAuthorityHost}
	options.ClientOptions.Retry = azpolicy.RetryOptions{MaxRetries: -1}
	credential, err := azidentity.NewWorkloadIdentityCredential(options)
	if err != nil {
		t.Fatal(err)
	}
	return credential, requests
}

func workloadTransport(t *testing.T, requests *workloadRequests, tokenResponse func() *http.Response) http.RoundTripper {
	t.Helper()
	return cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/synthetic-tenant/v2.0/.well-known/openid-configuration":
			requests.metadata++
			return workloadDiscoveryResponse(), nil
		case "/token":
			requests.token++
			return validatedTokenResponse(t, request, tokenResponse), nil
		default:
			if request.URL.Host == "redirect.invalid" {
				requests.redirect++
			}
			t.Fatalf("unexpected workload request %s", request.URL)
			return nil, nil
		}
	})
}

func validatedTokenResponse(t *testing.T, request *http.Request, response func() *http.Response) *http.Response {
	t.Helper()
	if request.Method != http.MethodPost || !strings.Contains(request.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		t.Fatal("workload token exchange did not post a form")
	}
	return response()
}

func workloadDiscoveryResponse() *http.Response {
	body := `{"token_endpoint":"` + syntheticAuthorityHost + `/token","issuer":"` + syntheticAuthorityHost + `/synthetic-tenant/v2.0","authorization_endpoint":"` + syntheticAuthorityHost + `/synthetic-tenant/oauth2/v2.0/authorize","device_authorization_endpoint":"` + syntheticAuthorityHost + `/synthetic-tenant/oauth2/v2.0/devicecode"}`
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
