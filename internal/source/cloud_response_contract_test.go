package source

import (
	"context"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// TestAWSSecretStageEvidenceCannotReleaseAnotherValue verifies that a
// syntactically successful provider response is not enough to disclose data:
// the returned stage must match the policy selector.
func TestAWSSecretStageEvidenceCannotReleaseAnotherValue(t *testing.T) {
	client, calls := awsSecretClient(t, `{"VersionId":"opaque","VersionStages":["stable"],"SecretString":"SYNTHETIC_AWS_VALUE"}`)
	value, err := awsSecretValue(context.Background(), client, model.Source{}, model.Binding{Resource: "fixture", Version: "stage:stable"})
	if err != nil || value.Data != "SYNTHETIC_AWS_VALUE" || value.Version != "opaque" || *calls != 1 {
		t.Fatalf("verified stage result = %#v, %v, calls=%d", value, err, *calls)
	}
	client, calls = awsSecretClient(t, `{"VersionId":"opaque","VersionStages":["other"],"SecretString":"SYNTHETIC_AWS_VALUE"}`)
	value, err = awsSecretValue(context.Background(), client, model.Source{}, model.Binding{Resource: "fixture", Version: "stage:stable"})
	assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_AWS_VALUE")
	if *calls != 1 {
		t.Fatalf("stage mismatch requests = %d, want 1", *calls)
	}
}

// TestAWSParameterSelectorEvidenceCannotReleaseAnotherValue verifies that an
// SSM response selector may satisfy a version policy only when it matches it.
func TestAWSParameterSelectorEvidenceCannotReleaseAnotherValue(t *testing.T) {
	client, calls := awsParameterClient(t, `{"Parameter":{"Value":"SYNTHETIC_PARAMETER_VALUE","Version":3,"Selector":":stable"}}`)
	value, err := awsParameterValue(context.Background(), client, model.Source{}, model.Binding{Resource: "fixture", Version: "stable"})
	if err != nil || value.Data != "SYNTHETIC_PARAMETER_VALUE" || value.Version != "3" || *calls != 1 {
		t.Fatalf("verified selector result = %#v, %v, calls=%d", value, err, *calls)
	}
	client, calls = awsParameterClient(t, `{"Parameter":{"Value":"SYNTHETIC_PARAMETER_VALUE","Version":3,"Selector":":other"}}`)
	value, err = awsParameterValue(context.Background(), client, model.Source{}, model.Binding{Resource: "fixture", Version: "stable"})
	assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_PARAMETER_VALUE")
	if *calls != 1 {
		t.Fatalf("selector mismatch requests = %d, want 1", *calls)
	}
}

func awsSecretClient(t *testing.T, body string) (*secretsmanager.Client, *int) {
	t.Helper()
	config, calls := responseAWSConfig(t, "secretsmanager.GetSecretValue", body)
	return secretsmanager.NewFromConfig(config), calls
}

func awsParameterClient(t *testing.T, body string) (*ssm.Client, *int) {
	t.Helper()
	config, calls := responseAWSConfig(t, "AmazonSSM.GetParameter", body)
	return ssm.NewFromConfig(config), calls
}

func responseAWSConfig(t *testing.T, target, body string) (aws.Config, *int) {
	t.Helper()
	calls := new(int)
	client := &http.Client{Transport: cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		(*calls)++
		if request.URL.Host != "provider.synthetic" || request.Header.Get("X-Amz-Target") != target {
			t.Fatalf("unexpected provider request %s %q", request.URL, request.Header.Get("X-Amz-Target"))
		}
		return &http.Response{Request: request, StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/x-amz-json-1.1"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	return aws.Config{Region: "ap-southeast-2", Credentials: aws.AnonymousCredentials{}, BaseEndpoint: aws.String("https://provider.synthetic"), HTTPClient: client}, calls
}

// TestAzureResponseVersionEvidenceCannotReleaseAnotherValue uses the actual
// generated Key Vault client so the test also proves the adapter's response
// decoding and request path are part of the boundary.
func TestAzureResponseVersionEvidenceCannotReleaseAnotherValue(t *testing.T) {
	client, calls := azureSecretClient(t, `{"value":"SYNTHETIC_AZURE_VALUE","id":"https://vault.synthetic/secrets/fixture/expected"}`)
	value, err := azureSecretValue(context.Background(), client, model.Source{}, model.Binding{Resource: "fixture", Version: "expected"})
	if err != nil || value.Data != "SYNTHETIC_AZURE_VALUE" || value.Version != "expected" || *calls != 1 {
		t.Fatalf("verified Azure result = %#v, %v, calls=%d", value, err, *calls)
	}
	client, calls = azureSecretClient(t, `{"value":"SYNTHETIC_AZURE_VALUE","id":"https://vault.synthetic/secrets/fixture/other"}`)
	value, err = azureSecretValue(context.Background(), client, model.Source{}, model.Binding{Resource: "fixture", Version: "expected"})
	assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_AZURE_VALUE")
	if *calls != 1 {
		t.Fatalf("Azure version mismatch requests = %d, want 1", *calls)
	}
}

func azureSecretClient(t *testing.T, body string) (*azsecrets.Client, *int) {
	t.Helper()
	calls := new(int)
	client, err := azsecrets.NewClient("https://vault.synthetic", syntheticSDKAccessToken{}, &azsecrets.ClientOptions{ClientOptions: azcore.ClientOptions{Transport: &http.Client{Timeout: time.Second, Transport: cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		(*calls)++
		if request.Method != http.MethodGet || request.URL.Path != "/secrets/fixture/expected" {
			t.Fatalf("unexpected Azure request %s", request.URL)
		}
		return &http.Response{Request: request, StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}})
	if err != nil {
		t.Fatal(err)
	}
	return client, calls
}

type syntheticSDKAccessToken struct{}

func (syntheticSDKAccessToken) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "synthetic", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

// TestGCPResponseEvidenceDoesNotReleaseValue checks the generated response
// object directly because the production constructor binds to link-local
// metadata; these cases are after that authority boundary.
func TestGCPResponseEvidenceDoesNotReleaseValue(t *testing.T) {
	data := []byte(`{"KEY":"SYNTHETIC_GCP_VALUE"}`)
	checksum := validCloudCRC(data)
	source := model.Source{Project: "synthetic"}
	binding := model.Binding{Resource: "projects/synthetic/secrets/fixture", Field: "KEY", Version: "7"}
	response := &secretmanagerpb.AccessSecretVersionResponse{Name: "projects/synthetic/secrets/fixture/versions/7", Payload: &secretmanagerpb.SecretPayload{Data: data, DataCrc32C: &checksum}}
	value, err := gcpSecretValue(response, source, binding)
	if err != nil || value.Data != "SYNTHETIC_GCP_VALUE" || value.Version != response.Name {
		t.Fatalf("verified GCP result = %#v, %v", value, err)
	}
	response.Name = "projects/synthetic/secrets/fixture/versions/other"
	value, err = gcpSecretValue(response, source, binding)
	assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_GCP_VALUE")
	value, err = gcpSecretValue(&secretmanagerpb.AccessSecretVersionResponse{}, source, binding)
	assertRejectedProviderValue(t, value, err, fault.Missing, "")
}

func validCloudCRC(data []byte) int64 {
	return int64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
}

func assertRejectedProviderValue(t *testing.T, value model.Value, err error, kind fault.Kind, marker string) {
	t.Helper()
	leaked := marker != "" && err != nil && strings.Contains(err.Error(), marker)
	if !fault.Is(err, kind) || value != (model.Value{}) || leaked {
		t.Fatalf("provider response leaked or was accepted: value=%#v err=%v", value, err)
	}
}
