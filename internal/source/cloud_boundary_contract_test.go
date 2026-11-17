package source

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestCloudBoundaryContractSSMResourcePrecedesIdentity(t *testing.T) {
	for _, resource := range []string{
		"arn:aws:ssm:ap-southeast-2:other-account:parameter/fixture",
		syntheticAWSSecretARN,
	} {
		// If the resource guard is removed, this explicit unsupported identity
		// still refuses before credentials, STS or any provider construction.
		source := model.Source{
			Type:          "aws-ssm",
			Account:       "123456789012",
			Region:        "ap-southeast-2",
			Identity:      "unsupported-synthetic",
			Resources:     []string{resource},
			AuditVerified: true,
		}
		binding := model.Binding{Resource: resource, Version: "3", Field: "selected"}
		cloudBoundaryRefuseBoth(t, source, binding, ReadFunc(readAWSParameter), fault.Denied, "aws_resource")
	}
}

func TestCloudBoundaryContractAzureNamePrecedesIdentity(t *testing.T) {
	// The empty tenant is a second, distinct refusal before any credential
	// construction if the name guard is removed. Registry scope is explicit.
	source := model.Source{
		Type:          "azure",
		Endpoint:      syntheticAzureEndpoint,
		Resources:     []string{"fixture/child"},
		AuditVerified: true,
	}
	binding := model.Binding{Resource: "fixture/child", Version: "expected", Field: "selected"}
	cloudBoundaryRefuseBoth(t, source, binding, ReadFunc(readAzure), fault.Invalid, "azure_secret_name")
}

func cloudBoundaryRefuseBoth(t *testing.T, source model.Source, binding model.Binding, direct ReadFunc, kind fault.Kind, operation string) {
	t.Helper()
	value, err := direct.Read(context.Background(), source, binding)
	cloudBoundaryValueRefusal(t, value, err, kind, operation)
	value, err = NewRegistry().Read(context.Background(), source, binding)
	cloudBoundaryValueRefusal(t, value, err, kind, operation)
}

func cloudBoundaryValueRefusal(t *testing.T, value model.Value, err error, kind fault.Kind, operation string) {
	t.Helper()
	typed, ok := err.(*fault.Error)
	if !ok {
		t.Fatal("cloud refusal did not return a direct typed fault")
	}
	if typed.Kind != kind || typed.Operation != operation || err.Error() != string(kind)+": "+operation {
		t.Fatal("cloud refusal changed its category, operation or sanitized diagnostic")
	}
	if errors.Unwrap(err) != nil || value != (model.Value{}) {
		t.Fatal("cloud refusal retained a cause, value or provenance metadata")
	}
}

// This is only a synthetic tenant-wrapper decoding/forwarding contract. The
// callback neither authenticates Azure tokens nor retrieves/enrolls credentials.
func TestCloudBoundaryContractTenantPayloadRefusalAndForwarding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts := azpolicy.TokenRequestOptions{Scopes: []string{"https://vault.azure.net/.default"}}
	token := azcore.AccessToken{Token: "header.%.signature", ExpiresOn: time.Unix(2000000000, 0)}
	calls := 0
	credential := tenantCredential{
		tenant:     "expected-tenant",
		credential: cloudBoundaryTokenFunc(func(gotContext context.Context, gotOptions azpolicy.TokenRequestOptions) (azcore.AccessToken, error) {
			calls++
			if gotContext != ctx || !reflect.DeepEqual(gotOptions, opts) {
				t.Fatal("tenant wrapper changed the supplied context or request options")
			}
			return token, nil
		}),
	}
	got, err := credential.GetToken(ctx, opts)
	cloudBoundaryTokenRefusal(t, got, err)
	if calls != 1 {
		t.Fatal("malformed payload was not obtained exactly once from the finite fake")
	}

	token.Token = syntheticTenantToken(t, "expected-tenant")
	got, err = credential.GetToken(ctx, opts)
	if err != nil || got != token || calls != 2 {
		t.Fatal("matching synthetic tenant did not preserve the complete token exactly once")
	}
}

func cloudBoundaryTokenRefusal(t *testing.T, token azcore.AccessToken, err error) {
	t.Helper()
	typed, ok := err.(*fault.Error)
	if !ok {
		t.Fatal("malformed payload did not return a direct typed fault")
	}
	if typed.Kind != fault.Trust || typed.Operation != "azure_token_account" || err.Error() != "trust_failure: azure_token_account" {
		t.Fatal("malformed payload changed its safe refusal or exposed token data")
	}
	if token != (azcore.AccessToken{}) || errors.Unwrap(err) != nil {
		t.Fatal("malformed payload retained a token, expiry or upstream cause")
	}
}

type cloudBoundaryTokenFunc func(context.Context, azpolicy.TokenRequestOptions) (azcore.AccessToken, error)

func (f cloudBoundaryTokenFunc) GetToken(ctx context.Context, opts azpolicy.TokenRequestOptions) (azcore.AccessToken, error) {
	return f(ctx, opts)
}
