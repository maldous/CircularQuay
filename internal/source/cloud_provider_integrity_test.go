package source

import (
	"context"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// TestCloudProviderClientsRejectMismatchedEvidence exercises real generated SDK
// clients over an in-memory transport.  The invariant is that an otherwise
// well-formed provider response cannot substitute a secret or parameter version.
func TestCloudProviderClientsRejectMismatchedEvidence(t *testing.T) {
	t.Run("secrets manager version", func(t *testing.T) {
		client := secretsmanager.NewFromConfig(syntheticAWSConfig(t, "secretsmanager.GetSecretValue", `{"ARN":"`+syntheticAWSSecretARN+`","VersionId":"other","SecretString":"SYNTHETIC_VALUE"}`))
		value, err := awsSecretValue(context.Background(), client, model.Source{}, model.Binding{Resource: syntheticAWSSecretARN, Version: "expected"})
		if !fault.Is(err, fault.Trust) || value.Data != "" || strings.Contains(err.Error(), "SYNTHETIC_VALUE") {
			t.Fatalf("mismatched secret version result = %#v, %v", value, err)
		}
	})
	t.Run("parameter version", func(t *testing.T) {
		client := ssm.NewFromConfig(syntheticAWSConfig(t, "AmazonSSM.GetParameter", `{"Parameter":{"ARN":"`+syntheticAWSSSMARN+`","Value":"SYNTHETIC_VALUE","Version":3}}`))
		value, err := awsParameterValue(context.Background(), client, model.Source{}, model.Binding{Resource: syntheticAWSSSMARN, Version: "2"})
		if !fault.Is(err, fault.Trust) || value.Data != "" || strings.Contains(err.Error(), "SYNTHETIC_VALUE") {
			t.Fatalf("mismatched parameter version result = %#v, %v", value, err)
		}
	})
}

func syntheticAWSConfig(t *testing.T, target, response string) aws.Config {
	t.Helper()
	return aws.Config{
		Region:       "ap-southeast-2",
		Credentials:  aws.AnonymousCredentials{},
		BaseEndpoint: aws.String("https://provider.synthetic"),
		HTTPClient: &http.Client{Transport: cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
			if request.URL.Host != "provider.synthetic" || request.Header.Get("X-Amz-Target") != target {
				t.Fatalf("unexpected synthetic AWS request %s %q", request.URL, request.Header.Get("X-Amz-Target"))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/x-amz-json-1.1"}},
				Body:       io.NopCloser(strings.NewReader(response)),
			}, nil
		})},
	}
}

// TestGCPPayloadIntegrityRejectsTampering makes the integrity boundary visible:
// the caller receives neither a value nor a usable version when CRC evidence is
// absent or changed in transit.
func TestGCPPayloadIntegrityRejectsTampering(t *testing.T) {
	payload := []byte(`{"KEY":"SYNTHETIC_VALUE"}`)
	checksum := int64(crc32.Checksum(payload, crc32.MakeTable(crc32.Castagnoli)))
	response := &secretmanagerpb.AccessSecretVersionResponse{
		Name:    "projects/synthetic/secrets/fixture/versions/7",
		Payload: &secretmanagerpb.SecretPayload{Data: payload, DataCrc32C: &checksum},
	}
	source := model.Source{Project: "synthetic"}
	binding := model.Binding{Resource: "projects/synthetic/secrets/fixture", Field: "KEY", Version: "7"}

	value, err := gcpSecretValue(response, source, binding)
	if err != nil || value.Data != "SYNTHETIC_VALUE" || value.Version != response.Name {
		t.Fatalf("verified payload result = %#v, %v", value, err)
	}

	response.Payload.Data = []byte(`{"KEY":"TAMPERED"}`)
	value, err = gcpSecretValue(response, source, binding)
	if !fault.Is(err, fault.Trust) || value.Data != "" || value.Version != "" {
		t.Fatalf("tampered payload result = %#v, %v", value, err)
	}
}

func TestCloudErrorsDoNotPromoteUnclassifiedFailures(t *testing.T) {
	if err := cloudCode("ParameterNotFound"); !fault.Is(err, fault.Missing) {
		t.Fatalf("known missing cloud error = %v", err)
	}
	if err := cloudCode("synthetic-unclassified-failure"); !fault.Is(err, fault.Trust) {
		t.Fatalf("unclassified cloud error = %v", err)
	}
}
