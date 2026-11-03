package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestCloudAWSRejectsInvalidAuthorityBeforeProviderUse(t *testing.T) {
	source := model.Source{Account: "123456789012", Region: "ap-southeast-2", Identity: "ec2"}
	invalid := model.Binding{Resource: "arn:aws:secretsmanager:ap-southeast-2:other:secret:fixture"}
	if _, err := readAWSSecret(context.Background(), source, invalid); !fault.Is(err, fault.Denied) {
		t.Fatalf("cross-account secret reached provider setup: %v", err)
	}
	for _, candidate := range []model.Source{
		{Account: "123", Region: "region", Identity: "profile"},
		{Account: "123", Region: "region", Identity: "ec2", Profile: "developer"},
		{Account: "123", Region: "region", Identity: "ecs", Endpoint: "https://override.example.test"},
	} {
		if _, err := awsConfiguration(context.Background(), candidate); err == nil {
			t.Fatal("AWS configuration accepted an ambient or override authority")
		}
	}
}

func TestCloudAWSWorkloadProviderSelectionAndMetadataTransport(t *testing.T) {
	if _, err := awsCredentialsProvider(model.Source{Identity: "ec2"}); err != nil {
		t.Fatalf("EC2 workload provider was not selected: %v", err)
	}
	t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "/v2/credentials/synthetic")
	if _, err := awsCredentialsProvider(model.Source{Identity: "ecs"}); err != nil {
		t.Fatalf("ECS workload provider was not selected: %v", err)
	}
	if _, err := awsCredentialsProvider(model.Source{Identity: "environment"}); !fault.Is(err, fault.Unsupported) {
		t.Fatalf("ambient AWS identity was accepted: %v", err)
	}
	for _, path := range []string{"", "relative", "https://metadata.example.test", "/path?query", "/path\nnext"} {
		t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", path)
		if _, err := ecsCredentialsProvider(); !fault.Is(err, fault.Invalid) {
			t.Fatalf("unsafe ECS metadata path %q was accepted: %v", path, err)
		}
	}
	client := metadataHTTPClient()
	transport := metadataTransport()
	if transport.Proxy != nil || client.Timeout != 5*time.Second {
		t.Fatal("metadata client can inherit a proxy or lacks its fixed deadline")
	}
	if err := client.CheckRedirect(nil, nil); err == nil || errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("metadata redirect was not rejected: %v", err)
	}
}

func TestCloudAzureRejectsOverrideAndInvalidCredentialSelection(t *testing.T) {
	workload := model.Source{Account: "synthetic-tenant", Profile: "synthetic-client", TokenFile: privateCloudTokenFile(t), Identity: "workload"}
	t.Setenv("AZURE_AUTHORITY_HOST", "https://override.example.test")
	if _, err := azureCredential(workload); !fault.Is(err, fault.Trust) {
		t.Fatalf("Azure authority override was accepted: %v", err)
	}
	t.Setenv("AZURE_AUTHORITY_HOST", "")
	t.Setenv("IMDS_ENDPOINT", "http://override.example.test")
	managed := model.Source{Account: "synthetic-tenant", Profile: "synthetic-client", Identity: "managed"}
	if _, err := azureCredential(managed); !fault.Is(err, fault.Trust) {
		t.Fatalf("Azure metadata override was accepted: %v", err)
	}
	if _, err := azureCredential(model.Source{Account: "tenant", Identity: "environment"}); !fault.Is(err, fault.Unsupported) {
		t.Fatalf("unimplemented Azure identity selector was accepted: %v", err)
	}
}

func TestCloudTenantCredentialRejectsUntrustedTenantTokens(t *testing.T) {
	for _, test := range []struct {
		name  string
		token string
		err   error
		want  fault.Kind
	}{
		{"malformed", "not-a-token", nil, fault.Trust},
		{"oversized", strings.Repeat("a", 65537), nil, fault.Trust},
		{"missing tenant", syntheticTenantToken(t, ""), nil, fault.Denied},
		{"wrong tenant", syntheticTenantToken(t, "other-tenant"), nil, fault.Denied},
		{"provider failure", "", errors.New("synthetic provider failure"), fault.Trust},
	} {
		t.Run(test.name, func(t *testing.T) {
			credential := tenantCredential{credential: &syntheticCloudTokenCredential{token: test.token, err: test.err}, tenant: "expected-tenant"}
			if _, err := credential.GetToken(context.Background(), azpolicy.TokenRequestOptions{}); !fault.Is(err, test.want) {
				t.Fatalf("tenant validation result = %v, want %s", err, test.want)
			}
		})
	}
}

func TestCloudGCPRejectsInvalidConfigurationBeforeMetadataUse(t *testing.T) {
	binding := model.Binding{Resource: "projects/synthetic-project/secrets/fixture"}
	for _, source := range []model.Source{
		{Project: "synthetic-project", Account: "service@example.test", Identity: "metadata", Endpoint: "https://override.example.test"},
		{Project: "synthetic-project", Account: "service@example.test", Identity: "metadata", TokenFile: "/private/token"},
		{Project: "synthetic-project", Account: "default", Identity: "metadata"},
	} {
		if _, err := readGCP(context.Background(), source, binding); !fault.Is(err, fault.Unsupported) {
			t.Fatalf("invalid GCP identity reached metadata provider: %v", err)
		}
	}
	t.Setenv("GCE_METADATA_HOST", "metadata.override.example.test")
	source := model.Source{Project: "synthetic-project", Account: "service@example.test", Identity: "metadata"}
	if _, err := readGCP(context.Background(), source, binding); !fault.Is(err, fault.Unsupported) {
		t.Fatalf("GCP metadata override was accepted: %v", err)
	}
	wrongProject := model.Binding{Resource: "projects/other/secrets/fixture"}
	if _, err := readGCP(context.Background(), source, wrongProject); !fault.Is(err, fault.Denied) {
		t.Fatalf("cross-project GCP resource reached metadata provider: %v", err)
	}
}

type syntheticCloudTokenCredential struct {
	token string
	err   error
}

func (c *syntheticCloudTokenCredential) GetToken(context.Context, azpolicy.TokenRequestOptions) (azcore.AccessToken, error) {
	if c.err != nil {
		return azcore.AccessToken{}, c.err
	}
	return azcore.AccessToken{Token: c.token}, nil
}

func syntheticTenantToken(t *testing.T, tenant string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"tid": tenant})
	if err != nil {
		t.Fatal(err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func privateCloudTokenFile(t *testing.T) string {
	t.Helper()
	directory := privateSourceDirectory(t, "cloud")
	path := filepath.Join(directory, "token")
	if err := platform.WriteProtected(path, []byte("synthetic-token")); err != nil {
		t.Fatal(err)
	}
	return path
}
