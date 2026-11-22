package source

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

func TestCloudCoverageRejectsMalformedAWSSelectorsBeforeUse(t *testing.T) {
	source := model.Source{Account: "123456789012", Region: "ap-southeast-2"}
	for _, resource := range []string{
		"",
		"arn:aws:secretsmanager:ap-southeast-2:123456789012",
		"arn:aws:ssm:ap-southeast-2:123456789012:secret:fixture",
		"arn:aws:secretsmanager:other-region:123456789012:secret:fixture",
		"arn:aws:secretsmanager:ap-southeast-2:other:secret:fixture",
		"arn:aws:secretsmanager:ap-southeast-2:123456789012:",
	} {
		if err := validateAWSResource(source, model.Binding{Resource: resource}, "secretsmanager"); !fault.Is(err, fault.Denied) {
			t.Fatalf("unsafe AWS resource %q = %v, want denied", resource, err)
		}
	}

	result := &secretsmanager.GetSecretValueOutput{VersionId: awsString("opaque"), VersionStages: []string{"stable"}}
	if err := validateAWSSecretVersion(model.Binding{Version: "stage:"}, result); !fault.Is(err, fault.Invalid) {
		t.Fatalf("empty AWS stage selector = %v, want invalid", err)
	}
	if err := validateAWSSecretVersion(model.Binding{Version: "stage:current"}, result); !fault.Is(err, fault.Trust) {
		t.Fatalf("unreturned AWS stage selector = %v, want trust failure", err)
	}
}

func TestCloudCoverageGCPIdentityAndVersionSelectorsFailClosed(t *testing.T) {
	valid := model.Source{Project: "project", Account: "service@example.test", Identity: "metadata"}
	if err := validateGCPIdentity(valid); err != nil {
		t.Fatalf("valid metadata identity = %v", err)
	}
	for _, source := range []model.Source{
		{Project: "project", Account: "service@example.test", Identity: "workload"},
		{Project: "project", Account: "", Identity: "metadata"},
		{Project: "project", Account: "default", Identity: "metadata"},
		{Project: "project", Account: "service@example.test", Identity: "metadata", Endpoint: "https://override.invalid"},
		{Project: "project", Account: "service@example.test", Identity: "metadata", TokenFile: "/private/token"},
		{Project: "project", Account: "service@example.test", Identity: "metadata", Profile: "developer"},
	} {
		if err := validateGCPIdentity(source); !fault.Is(err, fault.Unsupported) {
			t.Fatalf("unsafe GCP identity %#v = %v, want unsupported", source, err)
		}
	}

	resource := "projects/project/secrets/fixture"
	if name, err := gcpResource(valid, model.Binding{Resource: resource}); err != nil || name != resource+"/versions/latest" {
		t.Fatalf("implicit GCP latest selector = %q, %v", name, err)
	}
	for _, version := range []string{"latest", "child/version", "child?query", "child#fragment", "child\x00suffix"} {
		if _, err := gcpResource(valid, model.Binding{Resource: resource, Version: version}); !fault.Is(err, fault.Invalid) {
			t.Fatalf("unsafe GCP version %q = %v, want invalid", version, err)
		}
	}
}

func TestCloudCoverageAzureOriginAndVersionCanonicalization(t *testing.T) {
	origin, err := azureOrigin("https://VaUlT.Synthetic:8443")
	if err != nil || origin != "https://vault.synthetic:8443" {
		t.Fatalf("canonical Azure origin = %q, %v", origin, err)
	}
	for _, endpoint := range []string{
		"http://vault.synthetic",
		"https://user@vault.synthetic",
		"https://vault.synthetic/secrets",
		"https://vault.synthetic/%2f",
		"https://vault.synthetic?query=value",
	} {
		if _, err := azureOrigin(endpoint); !fault.Is(err, fault.Trust) {
			t.Fatalf("ambiguous Azure endpoint %q = %v, want trust failure", endpoint, err)
		}
	}
	for _, path := range []string{
		"/secrets/other/version",
		"/secrets/fixture/",
		"/secrets/fixture/..",
		"/secrets/fixture/version/child",
		"/secrets/fixture/version?query",
	} {
		if _, err := azureSecretVersion(path, "fixture"); !fault.Is(err, fault.Trust) {
			t.Fatalf("unsafe Azure returned path %q = %v, want trust failure", path, err)
		}
	}
}

func TestCloudCoverageTransportDefaultsRemainBoundedAndRedirectDenying(t *testing.T) {
	base := cloudRoundTripper(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("synthetic transport should not be reached")
	})
	client := cloudHTTPClient(base)
	if client.Timeout != 30*time.Second || client.Transport == nil {
		t.Fatal("cloud client did not configure bounded transport defaults")
	}
	if _, err := client.Transport.RoundTrip(nil); err == nil || err.Error() != "synthetic transport should not be reached" {
		t.Fatalf("cloud transport did not preserve the supplied base behavior: %v", err)
	}
	if err := client.CheckRedirect(nil, nil); !errors.Is(err, transport.DenyRedirect(nil, nil)) {
		t.Fatalf("cloud redirect policy = %v, want denied", err)
	}
	metadata := metadataHTTPClient()
	if metadata.Timeout != 5*time.Second {
		t.Fatalf("metadata timeout = %s, want 5s", metadata.Timeout)
	}
	if err := metadata.CheckRedirect(nil, nil); !errors.Is(err, transport.DenyRedirect(nil, nil)) {
		t.Fatalf("metadata redirect policy = %v, want denied", err)
	}
}

func awsString(value string) *string { return &value }

func TestCloudCoverageTenantCredentialAcceptsOnlyMatchingTenant(t *testing.T) {
	credential := tenantCredential{
		credential: &syntheticCloudTokenCredential{token: syntheticTenantToken(t, "TENANT-A")},
		tenant:     "tenant-a",
	}
	if _, err := credential.GetToken(context.Background(), azpolicy.TokenRequestOptions{}); err != nil {
		t.Fatalf("case-insensitive matching tenant was rejected: %v", err)
	}
}
