package source

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// These response-boundary contracts use only existing in-memory SDK transports
// with explicit synthetic identities, or pure GCP response objects. They never
// enter production credential/configuration factories or contact a provider.
func TestCloudValueRefusalContractAWSBinaryPrecedesIdentity(t *testing.T) {
	binding := model.Binding{Resource: syntheticAWSSecretARN, Version: "expected"}
	body := `{"ARN":"untrusted-resource","VersionId":"other","SecretBinary":"aW5lcnQ="}`
	value, err := cloudValueAWSSecret(t, body, binding)
	requireCloudValueRefusal(t, value, err, fault.Unsupported, "binary_secret")

	// A present empty string is not a missing/binary secret.
	body = `{"ARN":"` + syntheticAWSSecretARN + `","VersionId":"expected","SecretString":""}`
	value, err = cloudValueAWSSecret(t, body, binding)
	requireCloudValueSuccess(t, value, err, model.Value{Version: "expected", AuditVerified: true})
}

func TestCloudValueRefusalContractAWSFieldAfterResourceAndVersion(t *testing.T) {
	cases := []struct {
		name, resource, version, operation string
	}{
		{"resource first", "untrusted-resource", "other", "source_resource_mismatch"},
		{"version before field", syntheticAWSSecretARN, "other", "source_version_mismatch"},
		{"field after bound identity", syntheticAWSSecretARN, "expected", "value_type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"ARN":"` + tc.resource + `","VersionId":"` + tc.version + `","SecretString":"{\"requested\":7}"}`
			binding := model.Binding{Resource: syntheticAWSSecretARN, Version: "expected", Field: "requested"}
			value, err := cloudValueAWSSecret(t, body, binding)
			requireCloudValueRefusal(t, value, err, fault.Trust, tc.operation)
		})
	}
}

func TestCloudValueRefusalContractSSMMissingPrecedesIdentity(t *testing.T) {
	for _, body := range []string{
		`{}`,
		`{"Parameter":{"ARN":"untrusted-resource","Version":99}}`,
	} {
		value, err := cloudValueAWSParameter(t, body, model.Binding{Resource: syntheticAWSSSMARN, Version: "3"})
		requireCloudValueRefusal(t, value, err, fault.Missing, "source_value")
	}
	body := `{"Parameter":{"ARN":"` + syntheticAWSSSMARN + `","Value":"","Version":3}}`
	value, err := cloudValueAWSParameter(t, body, model.Binding{Resource: syntheticAWSSSMARN, Version: "3"})
	requireCloudValueSuccess(t, value, err, model.Value{Version: "3", AuditVerified: true})
}

func TestCloudValueRefusalContractSSMDecodeFailureIsSanitized(t *testing.T) {
	// The generated client, not the adapter's field selector, refuses this body.
	// The in-memory transport is still the only possible HTTP destination.
	value, err := cloudValueAWSParameter(t, `{"Parameter":SYNTHETIC_PRIVATE_RESPONSE`, model.Binding{Resource: syntheticAWSSSMARN})
	requireCloudValueRefusal(t, value, err, fault.Trust, "provider_failure")
}

func TestCloudValueRefusalContractSSMFieldWithholdsAllMetadata(t *testing.T) {
	body := `{"Parameter":{"ARN":"` + syntheticAWSSSMARN + `","Value":"{\"requested\":7}","Version":3}}`
	binding := model.Binding{Resource: syntheticAWSSSMARN, Version: "3", Field: "requested"}
	value, err := cloudValueAWSParameter(t, body, binding)
	requireCloudValueRefusal(t, value, err, fault.Trust, "value_type")
}

func TestCloudValueRefusalContractAzureMissingPrecedesIdentity(t *testing.T) {
	binding := model.Binding{Resource: "fixture", Version: "expected", Field: "requested"}
	body := `{"id":"https://other.synthetic/secrets/untrusted/other"}`
	value, err := cloudValueAzureSecret(t, body, "http://untrusted.synthetic", binding)
	requireCloudValueRefusal(t, value, err, fault.Missing, "source_value")
}

func TestCloudValueRefusalContractAzureFieldAfterOriginAndVersion(t *testing.T) {
	cases := []struct {
		name, endpoint, version, operation string
	}{
		{"configured origin first", "http://untrusted.synthetic", "other", "provider_endpoint"},
		{"returned version before field", syntheticAzureEndpoint, "other", "source_version_mismatch"},
		{"field after bound identity", syntheticAzureEndpoint, "expected", "value_type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"value":"{\"requested\":7}","id":"https://vault.synthetic/secrets/fixture/` + tc.version + `"}`
			binding := model.Binding{Resource: "fixture", Version: "expected", Field: "requested"}
			value, err := cloudValueAzureSecret(t, body, tc.endpoint, binding)
			requireCloudValueRefusal(t, value, err, fault.Trust, tc.operation)
		})
	}
}

func TestCloudValueRefusalContractAzureRetainsDeclaredExpiryOnSuccess(t *testing.T) {
	body := `{"value":"{\"requested\":\"\"}","id":"https://vault.synthetic/secrets/fixture/expected","attributes":{"exp":1700000000}}`
	binding := model.Binding{Resource: "fixture", Version: "expected", Field: "requested"}
	value, err := cloudValueAzureSecret(t, body, syntheticAzureEndpoint, binding)
	// Returning an expiry is metadata propagation, not permission to use an
	// expired secret. This response adapter does not perform that policy check.
	want := model.Value{Version: "expected", AuditVerified: true, ExpiresAt: time.Unix(1700000000, 0)}
	requireCloudValueSuccess(t, value, err, want)
}

func TestCloudValueRefusalContractGCPBindingPrecedesReturnedIdentity(t *testing.T) {
	cases := []struct {
		name, project, version string
		kind                   fault.Kind
		operation              string
	}{
		{"project before returned identity", "other-project", "7", fault.Denied, "gcp_resource"},
		{"explicit latest before identity", "synthetic", "latest", fault.Invalid, "gcp_version"},
		{"invalid version before identity", "synthetic", "7/child", fault.Invalid, "gcp_version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := cloudValueGCPResponse(`{"requested":7}`, "untrusted-response-name")
			binding := model.Binding{Resource: "projects/synthetic/secrets/fixture", Version: tc.version, Field: "requested"}
			value, err := gcpSecretValue(response, model.Source{Project: tc.project, AuditVerified: true}, binding)
			requireCloudValueRefusal(t, value, err, tc.kind, tc.operation)
		})
	}
}

func TestCloudValueRefusalContractGCPFieldAfterIntegrityAndIdentity(t *testing.T) {
	name := "projects/synthetic/secrets/fixture/versions/7"
	binding := model.Binding{Resource: "projects/synthetic/secrets/fixture", Version: "7", Field: "requested"}
	source := model.Source{Project: "synthetic", AuditVerified: true}
	value, err := gcpSecretValue(cloudValueGCPResponse(`{"requested":7}`, name), source, binding)
	requireCloudValueRefusal(t, value, err, fault.Trust, "value_type")
	value, err = gcpSecretValue(cloudValueGCPResponse(`{"requested":""}`, name), source, binding)
	requireCloudValueSuccess(t, value, err, model.Value{Version: name, AuditVerified: true})
}

func cloudValueAWSSecret(t *testing.T, body string, binding model.Binding) (model.Value, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, calls := awsSecretClient(t, body)
	value, err := awsSecretValue(ctx, client, model.Source{AuditVerified: true}, binding)
	if *calls != 1 {
		t.Fatal("synthetic Secrets Manager response was not consumed exactly once")
	}
	return value, err
}

func cloudValueAWSParameter(t *testing.T, body string, binding model.Binding) (model.Value, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, calls := awsParameterClient(t, body)
	value, err := awsParameterValue(ctx, client, model.Source{AuditVerified: true}, binding)
	if *calls != 1 {
		t.Fatal("synthetic SSM response was not consumed exactly once")
	}
	return value, err
}

func cloudValueAzureSecret(t *testing.T, body, endpoint string, binding model.Binding) (model.Value, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, calls := azureSecretClient(t, body)
	value, err := azureSecretValue(ctx, client, model.Source{Endpoint: endpoint, AuditVerified: true}, binding)
	if *calls != 1 {
		t.Fatal("synthetic Azure response was not consumed exactly once")
	}
	return value, err
}

func cloudValueGCPResponse(body, name string) *secretmanagerpb.AccessSecretVersionResponse {
	data := []byte(body)
	checksum := validCloudCRC(data)
	return &secretmanagerpb.AccessSecretVersionResponse{
		Name:    name,
		Payload: &secretmanagerpb.SecretPayload{Data: data, DataCrc32C: &checksum},
	}
}

func requireCloudValueRefusal(t *testing.T, value model.Value, err error, kind fault.Kind, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) {
		t.Fatal("response refusal did not return a typed fault")
	}
	if typed.Kind != kind || typed.Operation != operation || err.Error() != string(kind)+": "+operation {
		t.Fatal("response refusal changed its category/operation or disclosed provider evidence")
	}
	if value != (model.Value{}) {
		t.Fatal("response refusal retained data, version, expiry, lease or audit metadata")
	}
}

func requireCloudValueSuccess(t *testing.T, value model.Value, err error, want model.Value) {
	t.Helper()
	if !value.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatal("verified response lost or changed its declared expiry")
	}
	// Compare every remaining field exactly without requiring identical timezone
	// location pointers for equal instants decoded from the synthetic wire body.
	value.ExpiresAt = want.ExpiresAt
	if err != nil || value != want {
		t.Fatal("verified response changed its value, version, lease or audit contract")
	}
}
