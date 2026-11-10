package source

import (
	"context"
	"hash/crc32"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

const (
	syntheticAWSSecretARN  = "arn:aws:secretsmanager:ap-southeast-2:123456789012:secret:fixture-abcdef"
	syntheticAWSSSMARN     = "arn:aws:ssm:ap-southeast-2:123456789012:parameter/fixture"
	syntheticAzureEndpoint = "https://vault.synthetic"
)

func TestCloudReturnedAWSResourceIdentity(t *testing.T) {
	secretBinding := model.Binding{Resource: syntheticAWSSecretARN}
	parameterBinding := model.Binding{Resource: syntheticAWSSSMARN}

	secretClient, _ := awsSecretClient(t, `{"ARN":"`+syntheticAWSSecretARN+`","SecretString":"SYNTHETIC_AWS_VALUE"}`)
	value, err := awsSecretValue(context.Background(), secretClient, model.Source{}, secretBinding)
	if err != nil || value.Data != "SYNTHETIC_AWS_VALUE" {
		t.Fatalf("verified Secrets Manager identity = %#v, %v", value, err)
	}
	for _, body := range []string{
		`{"SecretString":"SYNTHETIC_AWS_VALUE"}`,
		`{"ARN":"arn:aws:secretsmanager:ap-southeast-2:123456789012:secret:other-abcdef","SecretString":"SYNTHETIC_AWS_VALUE"}`,
	} {
		secretClient, _ = awsSecretClient(t, body)
		value, err = awsSecretValue(context.Background(), secretClient, model.Source{}, secretBinding)
		assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_AWS_VALUE")
	}

	parameterClient, _ := awsParameterClient(t, `{"Parameter":{"ARN":"`+syntheticAWSSSMARN+`","Value":"SYNTHETIC_SSM_VALUE"}}`)
	value, err = awsParameterValue(context.Background(), parameterClient, model.Source{}, parameterBinding)
	if err != nil || value.Data != "SYNTHETIC_SSM_VALUE" {
		t.Fatalf("verified SSM identity = %#v, %v", value, err)
	}
	for _, body := range []string{
		`{"Parameter":{"Value":"SYNTHETIC_SSM_VALUE"}}`,
		`{"Parameter":{"ARN":"arn:aws:ssm:ap-southeast-2:123456789012:parameter/other","Value":"SYNTHETIC_SSM_VALUE"}}`,
	} {
		parameterClient, _ = awsParameterClient(t, body)
		value, err = awsParameterValue(context.Background(), parameterClient, model.Source{}, parameterBinding)
		assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_SSM_VALUE")
	}
}

func TestCloudReturnedAzureResourceIdentity(t *testing.T) {
	binding := model.Binding{Resource: "fixture", Version: "expected"}
	source := model.Source{Endpoint: syntheticAzureEndpoint}
	client, _ := azureSecretClient(t, `{"value":"SYNTHETIC_AZURE_VALUE","id":"https://VAULT.SYNTHETIC:443/secrets/fixture/expected"}`)
	value, err := azureSecretValue(context.Background(), client, source, binding)
	if err != nil || value.Data != "SYNTHETIC_AZURE_VALUE" {
		t.Fatalf("verified Azure identity = %#v, %v", value, err)
	}
	for _, body := range []string{
		`{"value":"SYNTHETIC_AZURE_VALUE"}`,
		`{"value":"SYNTHETIC_AZURE_VALUE","id":"https://other.synthetic/secrets/fixture/expected"}`,
		`{"value":"SYNTHETIC_AZURE_VALUE","id":"https://vault.synthetic/secrets/other/expected"}`,
	} {
		client, _ = azureSecretClient(t, body)
		value, err = azureSecretValue(context.Background(), client, source, binding)
		assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_AZURE_VALUE")
	}
}

func TestCloudReturnedAzureURLShapes(t *testing.T) {
	binding := model.Binding{Resource: "fixture", Version: "expected"}
	source := model.Source{Endpoint: syntheticAzureEndpoint}
	for _, id := range []string{
		"https://vault.synthetic/secrets/fixture/.",
		"https://vault.synthetic/secrets/fixture/%00",
		"https://vault.synthetic/secrets/fixture/%23",
		"https://vault.synthetic/secrets/fixture/expected?",
		"https://vault.synthetic/secrets/fixture/expected#",
	} {
		client, _ := azureSecretClient(t, `{"value":"SYNTHETIC_AZURE_VALUE","id":"`+id+`"}`)
		value, err := azureSecretValue(context.Background(), client, source, binding)
		assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_AZURE_VALUE")
	}
	for _, endpoint := range []string{"https://vault.synthetic?", "https://vault.synthetic#", "https://[fe80::1%25en0]"} {
		if _, err := azureOrigin(endpoint); !fault.Is(err, fault.Trust) {
			t.Fatalf("ambiguous source endpoint accepted: %q: %v", endpoint, err)
		}
	}
}

func TestAzureRejectsAmbiguousEndpointBeforeCredentialSelection(t *testing.T) {
	// No identity is configured: reaching credential selection would return an
	// invalid-identity error. The URL trust failure must take precedence without
	// reading any assertion or reaching a provider transport.
	for _, endpoint := range []string{"https://vault.synthetic?", "https://vault.synthetic#", "https://[fe80::1%25en0]"} {
		value, err := readAzure(context.Background(), model.Source{Endpoint: endpoint}, model.Binding{Resource: "fixture"})
		assertRejectedProviderValue(t, value, err, fault.Trust, "")
	}
}

func TestCloudReturnedAzureIPv6Origins(t *testing.T) {
	binding := model.Binding{Resource: "fixture", Version: "expected"}
	client, _ := azureSecretClient(t, `{"value":"SYNTHETIC_AZURE_VALUE","id":"https://[::1]:443/secrets/fixture/expected"}`)
	value, err := azureSecretValue(context.Background(), client, model.Source{Endpoint: "https://[::1]"}, binding)
	if err != nil || value.Data != "SYNTHETIC_AZURE_VALUE" {
		t.Fatalf("verified IPv6 Azure origin = %#v, %v", value, err)
	}
	for _, test := range []struct{ endpoint, id string }{
		{"https://[::1]:8443", "https://[::1:8443]/secrets/fixture/expected"},
		{"https://[::1]:8443", "https://[::1]:443/secrets/fixture/expected"},
	} {
		client, _ = azureSecretClient(t, `{"value":"SYNTHETIC_AZURE_VALUE","id":"`+test.id+`"}`)
		value, err = azureSecretValue(context.Background(), client, model.Source{Endpoint: test.endpoint}, binding)
		assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_AZURE_VALUE")
	}
}

func TestCloudReturnedGCPResourceIdentityWhenUnpinned(t *testing.T) {
	data := []byte(`{"KEY":"SYNTHETIC_GCP_VALUE"}`)
	checksum := int64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
	source := model.Source{Project: "synthetic"}
	binding := model.Binding{Resource: "projects/synthetic/secrets/fixture", Field: "KEY"}
	value, err := gcpSecretValue(&secretmanagerpb.AccessSecretVersionResponse{Name: "projects/synthetic/secrets/fixture/versions/7", Payload: &secretmanagerpb.SecretPayload{Data: data, DataCrc32C: &checksum}}, source, binding)
	if err != nil || value.Data != "SYNTHETIC_GCP_VALUE" || value.Version != "projects/synthetic/secrets/fixture/versions/7" {
		t.Fatalf("verified unpinned GCP identity = %#v, %v", value, err)
	}
	for _, name := range []string{
		"",
		"projects/other/secrets/fixture/versions/7",
		"projects/synthetic/secrets/other/versions/7",
		"projects/synthetic/secrets/fixture-other/versions/7",
		"projects/synthetic/secrets/fixture/versions/",
		"projects/synthetic/secrets/fixture/versions/7/extra",
		"projects/synthetic/secrets/fixture/versions/latest",
	} {
		value, err = gcpSecretValue(&secretmanagerpb.AccessSecretVersionResponse{Name: name, Payload: &secretmanagerpb.SecretPayload{Data: data, DataCrc32C: &checksum}}, source, binding)
		assertRejectedProviderValue(t, value, err, fault.Trust, "SYNTHETIC_GCP_VALUE")
	}
}
