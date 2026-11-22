package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

const (
	awsErrorMarker   = "SYNTHETIC_AWS_ERROR_VALUE"
	azureErrorMarker = "SYNTHETIC_AZURE_ERROR_VALUE"
)

// TestAWSDeniedResponseDoesNotDiscloseProviderBody exercises the generated
// Secrets Manager client because a denial is safe only when its provider body
// cannot enter the returned value or diagnostic.
func TestAWSDeniedResponseDoesNotDiscloseProviderBody(t *testing.T) {
	client, calls := awsDeniedSecretClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	value, err := awsSecretValue(ctx, client, model.Source{}, model.Binding{Resource: "fixture"})
	assertRejectedProviderValue(t, value, err, fault.Denied, awsErrorMarker)
	if *calls != 1 {
		t.Fatalf("denied AWS requests = %d, want 1", *calls)
	}
}

func awsDeniedSecretClient(t *testing.T) (*secretsmanager.Client, *int) {
	t.Helper()
	calls := new(int)
	client := &http.Client{Transport: cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		(*calls)++
		if request.URL.Host != "provider.synthetic" || request.Header.Get("X-Amz-Target") != "secretsmanager.GetSecretValue" {
			t.Fatalf("unexpected AWS target %q", request.Header.Get("X-Amz-Target"))
		}
		return &http.Response{Request: request, StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/x-amz-json-1.1"}, "X-Amzn-Errortype": {"AccessDeniedException"}}, Body: io.NopCloser(strings.NewReader(`{"__type":"AccessDeniedException","message":"` + awsErrorMarker + `"}`))}, nil
	})}
	config := aws.Config{Region: "ap-southeast-2", Credentials: aws.AnonymousCredentials{}, BaseEndpoint: aws.String("https://provider.synthetic"), HTTPClient: client}
	return secretsmanager.NewFromConfig(config), calls
}

// TestAzureFailureResponsesDoNotDiscloseProviderBody binds each public fault
// category to a real Key Vault client response, including malformed JSON.
func TestAzureFailureResponsesDoNotDiscloseProviderBody(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		kind       fault.Kind
	}{
		{"denied", `{"error":{"message":"` + azureErrorMarker + `"}}`, http.StatusForbidden, fault.Denied},
		{"missing", `{"error":{"message":"` + azureErrorMarker + `"}}`, http.StatusNotFound, fault.Missing},
		// An HTTP 503 is untrusted provider evidence, not a transport outage that
		// could make a stale fallback appear freshly authorized.
		{"unavailable response", `{"error":{"message":"` + azureErrorMarker + `"}}`, http.StatusServiceUnavailable, fault.Trust},
		{"malformed", `{"value":"` + azureErrorMarker, http.StatusOK, fault.Trust},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, calls := azureErrorSecretClient(t, test.status, test.body)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			value, err := azureSecretValue(ctx, client, model.Source{}, model.Binding{Resource: "fixture", Version: "expected"})
			assertRejectedProviderValue(t, value, err, test.kind, azureErrorMarker)
			if *calls != 1 {
				t.Fatalf("%s Azure requests = %d, want 1", test.name, *calls)
			}
		})
	}
}

func azureErrorSecretClient(t *testing.T, status int, body string) (*azsecrets.Client, *int) {
	t.Helper()
	calls := new(int)
	options := &azsecrets.ClientOptions{ClientOptions: azcore.ClientOptions{
		Retry: policy.RetryOptions{MaxRetries: -1},
		Transport: &http.Client{Timeout: time.Second, Transport: cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
			(*calls)++
			if request.Method != http.MethodGet || request.URL.Path != "/secrets/fixture/expected" {
				t.Fatalf("unexpected Azure request %s", request.URL)
			}
			return &http.Response{Request: request, StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})},
	}}
	client, err := azsecrets.NewClient("https://vault.synthetic", syntheticSDKAccessToken{}, options)
	if err != nil {
		t.Fatal(err)
	}
	return client, calls
}
