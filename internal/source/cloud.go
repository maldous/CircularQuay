package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/credentials/endpointcreds"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/transport"
)

const awsVersionStagePrefix = "stage:"
const gcpVersionPathSeparator = "/versions/"

func registerCloud(r *Registry) {
	r.Register("aws-sm", ReadFunc(readAWSSecret))
	r.Register("aws-ssm", ReadFunc(readAWSParameter))
	r.Register("azure", ReadFunc(readAzure))
	r.Register("gcp", ReadFunc(readGCP))
}

// awsConfiguration selects workload credentials, never a developer profile or ambient key.
func awsConfiguration(ctx context.Context, s model.Source) (aws.Config, error) {
	if s.Account == "" || s.Region == "" || s.Profile != "" || s.Endpoint != "" {
		return aws.Config{}, fault.New(fault.Invalid, "aws_identity")
	}
	provider, err := awsCredentialsProvider(s)
	if err != nil {
		return aws.Config{}, err
	}
	cfg := aws.Config{Region: s.Region, Credentials: aws.NewCredentialsCache(provider), HTTPClient: cloudHTTPClient(nil)}
	result, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return aws.Config{}, cloudError(err)
	}
	if aws.ToString(result.Account) != s.Account {
		return aws.Config{}, fault.New(fault.Denied, "aws_account")
	}
	return cfg, nil
}

func awsCredentialsProvider(s model.Source) (aws.CredentialsProvider, error) {
	switch s.Identity {
	case "ec2":
		return ec2rolecreds.New(func(o *ec2rolecreds.Options) {
			o.Client = imds.New(imds.Options{Endpoint: "http://169.254.169.254", HTTPClient: metadataHTTPClient(), EnableFallback: aws.FalseTernary})
		}), nil
	case "ecs":
		return ecsCredentialsProvider()
	default:
		return nil, fault.New(fault.Unsupported, "aws_identity_method")
	}
}

func ecsCredentialsProvider() (aws.CredentialsProvider, error) {
	path := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#\r\n\x00") {
		return nil, fault.New(fault.Invalid, "ecs_identity")
	}
	return endpointcreds.New("http://169.254.170.2"+path, func(o *endpointcreds.Options) { o.HTTPClient = metadataHTTPClient() }), nil
}

func validateAWSResource(s model.Source, b model.Binding, service string) error {
	parts := strings.SplitN(b.Resource, ":", 6)
	if len(parts) != 6 || parts[0] != "arn" || parts[2] != service || parts[3] != s.Region || parts[4] != s.Account || parts[5] == "" {
		return fault.New(fault.Denied, "aws_resource")
	}
	return nil
}

// Link-local metadata traffic must not cross a proxy or redirect trust boundary.
func metadataHTTPClient() *http.Client {
	client := cloudHTTPClient(metadataTransport())
	client.Timeout = 5 * time.Second
	return client
}

func metadataTransport() *http.Transport {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.Proxy = nil
	return base
}

func cloudHTTPClient(base http.RoundTripper) *http.Client { return transport.NewHTTPClient(base) }

func readAWSSecret(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
	if err := validateAWSResource(s, b, "secretsmanager"); err != nil {
		return model.Value{}, err
	}
	cfg, err := awsConfiguration(ctx, s)
	if err != nil {
		return model.Value{}, err
	}
	return awsSecretValue(ctx, secretsmanager.NewFromConfig(cfg), s, b)
}

func awsSecretValue(ctx context.Context, client *secretsmanager.Client, s model.Source, b model.Binding) (model.Value, error) {
	in := &secretsmanager.GetSecretValueInput{SecretId: aws.String(b.Resource)}
	if strings.HasPrefix(b.Version, awsVersionStagePrefix) {
		in.VersionStage = aws.String(strings.TrimPrefix(b.Version, awsVersionStagePrefix))
	} else if b.Version != "" {
		in.VersionId = aws.String(b.Version)
	}
	result, err := client.GetSecretValue(ctx, in)
	if err != nil {
		return model.Value{}, cloudError(err)
	}
	if result.SecretString == nil {
		return model.Value{}, fault.New(fault.Unsupported, "binary_secret")
	}
	if err := validateAWSReturnedARN(b, result.ARN); err != nil {
		return model.Value{}, err
	}
	if err := validateAWSSecretVersion(b, result); err != nil {
		return model.Value{}, err
	}
	data, err := selectValue(*result.SecretString, b.Field)
	if err != nil {
		return model.Value{}, err
	}
	return model.Value{Data: data, Version: aws.ToString(result.VersionId), AuditVerified: s.AuditVerified}, nil
}

func validateAWSReturnedARN(b model.Binding, actual *string) error {
	if actual == nil || *actual != b.Resource {
		return fault.New(fault.Trust, "source_resource_mismatch")
	}
	return nil
}

func validateAWSSecretVersion(b model.Binding, result *secretsmanager.GetSecretValueOutput) error {
	if b.Version == "" {
		return nil
	}
	if stage, ok := strings.CutPrefix(b.Version, awsVersionStagePrefix); ok {
		if stage == "" {
			return fault.New(fault.Invalid, "source_version")
		}
		for _, returned := range result.VersionStages {
			if returned == stage {
				return nil
			}
		}
		return fault.New(fault.Trust, "source_version_mismatch")
	}
	return validateReturnedVersion(b, aws.ToString(result.VersionId))
}

func readAWSParameter(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
	if err := validateAWSResource(s, b, "ssm"); err != nil {
		return model.Value{}, err
	}
	cfg, err := awsConfiguration(ctx, s)
	if err != nil {
		return model.Value{}, err
	}
	return awsParameterValue(ctx, ssm.NewFromConfig(cfg), s, b)
}

func awsParameterValue(ctx context.Context, client *ssm.Client, s model.Source, b model.Binding) (model.Value, error) {
	name := b.Resource
	if b.Version != "" {
		name += ":" + b.Version
	}
	result, err := client.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name), WithDecryption: aws.Bool(true)})
	if err != nil {
		return model.Value{}, cloudError(err)
	}
	if result.Parameter == nil || result.Parameter.Value == nil {
		return model.Value{}, fault.New(fault.Missing, "source_value")
	}
	if err := validateAWSReturnedARN(b, result.Parameter.ARN); err != nil {
		return model.Value{}, err
	}
	if err := validateAWSParameterVersion(b, result.Parameter); err != nil {
		return model.Value{}, err
	}
	data, err := selectValue(*result.Parameter.Value, b.Field)
	if err != nil {
		return model.Value{}, err
	}
	return model.Value{Data: data, Version: strconv.FormatInt(result.Parameter.Version, 10), AuditVerified: s.AuditVerified}, nil
}

func validateAWSParameterVersion(b model.Binding, parameter *ssmtypes.Parameter) error {
	if b.Version == "" {
		return nil
	}
	actual := strconv.FormatInt(parameter.Version, 10)
	if actual == b.Version {
		return nil
	}
	if parameter.Selector != nil && *parameter.Selector == ":"+b.Version {
		return nil
	}
	return fault.New(fault.Trust, "source_version_mismatch")
}

func azureCredential(s model.Source) (azcore.TokenCredential, error) {
	if s.Account == "" {
		return nil, fault.New(fault.Invalid, "azure_tenant")
	}
	switch s.Identity {
	case "workload":
		return azureWorkloadCredential(s)
	case "managed":
		return azureManagedCredential(s)
	default:
		return nil, fault.New(fault.Unsupported, "azure_identity_method")
	}
}

func azureWorkloadCredential(s model.Source) (azcore.TokenCredential, error) {
	if s.Profile == "" || s.TokenFile == "" {
		return nil, fault.New(fault.Invalid, "azure_workload_identity")
	}
	if err := platform.ValidateProtectedPath(s.TokenFile); err != nil {
		return nil, err
	}
	if os.Getenv("AZURE_AUTHORITY_HOST") != "" {
		return nil, fault.New(fault.Trust, "azure_authority_override")
	}
	credential, err := azidentity.NewWorkloadIdentityCredential(azureWorkloadCredentialOptions(s, nil))
	if err != nil {
		return nil, fault.New(fault.Denied, "azure_identity")
	}
	return tenantCredential{credential: credential, tenant: s.Account}, nil
}

// azureWorkloadCredentialOptions keeps the workload assertion exchange on the
// same bounded, redirect-denying transport used for provider requests.
func azureWorkloadCredentialOptions(s model.Source, base http.RoundTripper) *azidentity.WorkloadIdentityCredentialOptions {
	return &azidentity.WorkloadIdentityCredentialOptions{
		TenantID:      s.Account,
		ClientID:      s.Profile,
		TokenFilePath: s.TokenFile,
		ClientOptions: azcore.ClientOptions{Transport: cloudHTTPClient(base)},
	}
}

func azureManagedCredential(s model.Source) (azcore.TokenCredential, error) {
	if s.Profile == "" {
		return nil, fault.New(fault.Invalid, "azure_managed_identity")
	}
	if azureMetadataOverride() {
		return nil, fault.New(fault.Trust, "azure_metadata_override")
	}
	options := &azidentity.ManagedIdentityCredentialOptions{ClientOptions: azcore.ClientOptions{Transport: metadataHTTPClient()}, ID: azidentity.ClientID(s.Profile)}
	credential, err := azidentity.NewManagedIdentityCredential(options)
	if err != nil {
		return nil, fault.New(fault.Denied, "azure_identity")
	}
	return tenantCredential{credential: credential, tenant: s.Account}, nil
}

func azureMetadataOverride() bool {
	for _, name := range []string{"IDENTITY_ENDPOINT", "IDENTITY_HEADER", "MSI_ENDPOINT", "MSI_SECRET", "IMDS_ENDPOINT", "AZURE_POD_IDENTITY_AUTHORITY_HOST"} {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}

// The native SDK authenticates this token. Checking its tenant binds an expected
// account selector; it does not authenticate arbitrary JWTs or replace the
// resource server's signature verification.
type tenantCredential struct {
	credential azcore.TokenCredential
	tenant     string
}

func (c tenantCredential) GetToken(ctx context.Context, opts azpolicy.TokenRequestOptions) (azcore.AccessToken, error) {
	token, err := c.credential.GetToken(ctx, opts)
	if err != nil {
		return azcore.AccessToken{}, cloudError(err)
	}
	parts := strings.Split(token.Token, ".")
	if len(parts) != 3 || len(token.Token) > 65536 {
		return azcore.AccessToken{}, fault.New(fault.Trust, "azure_token_account")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return azcore.AccessToken{}, fault.New(fault.Trust, "azure_token_account")
	}
	var claims struct {
		Tenant string `json:"tid"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Tenant == "" || !strings.EqualFold(claims.Tenant, c.tenant) {
		return azcore.AccessToken{}, fault.New(fault.Denied, "azure_tenant")
	}
	return token, nil
}

func readAzure(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
	if _, err := azureOrigin(s.Endpoint); err != nil {
		return model.Value{}, err
	}
	if strings.Contains(b.Resource, "/") {
		return model.Value{}, fault.New(fault.Invalid, "azure_secret_name")
	}
	cred, err := azureCredential(s)
	if err != nil {
		return model.Value{}, err
	}
	client, err := azsecrets.NewClient(s.Endpoint, cred, &azsecrets.ClientOptions{ClientOptions: azcore.ClientOptions{Transport: cloudHTTPClient(nil)}})
	if err != nil {
		return model.Value{}, fault.New(fault.Invalid, "azure_client")
	}
	return azureSecretValue(ctx, client, s, b)
}

func azureSecretValue(ctx context.Context, client *azsecrets.Client, s model.Source, b model.Binding) (model.Value, error) {
	result, err := client.GetSecret(ctx, b.Resource, b.Version, nil)
	if err != nil {
		return model.Value{}, cloudError(err)
	}
	if result.Value == nil {
		return model.Value{}, fault.New(fault.Missing, "source_value")
	}
	version, err := validateAzureSecretIdentity(s, b, result.ID)
	if err != nil {
		return model.Value{}, err
	}
	if err = validateReturnedVersion(b, version); err != nil {
		return model.Value{}, err
	}
	data, err := selectValue(*result.Value, b.Field)
	if err != nil {
		return model.Value{}, err
	}
	v := model.Value{Data: data, Version: version, AuditVerified: s.AuditVerified}
	if result.Attributes != nil && result.Attributes.Expires != nil {
		v.ExpiresAt = *result.Attributes.Expires
	}
	return v, nil
}

func validateAzureSecretIdentity(s model.Source, b model.Binding, id *azsecrets.ID) (string, error) {
	if id == nil {
		return "", fault.New(fault.Trust, "source_resource_mismatch")
	}
	configured, err := azureOrigin(s.Endpoint)
	if err != nil {
		return "", err
	}
	returned, err := azureSecretURL(string(*id))
	if err != nil || configured != azureURLOrigin(returned) {
		return "", fault.New(fault.Trust, "source_resource_mismatch")
	}
	return azureSecretVersion(returned.Path, b.Resource)
}

func azureOrigin(raw string) (string, error) {
	u, err := azureSecretURL(raw)
	if err != nil || (u.Path != "" && u.Path != "/") {
		return "", fault.New(fault.Trust, "provider_endpoint")
	}
	return azureURLOrigin(u), nil
}

func azureSecretURL(raw string) (*url.URL, error) {
	u, err := transport.Endpoint(raw)
	if err != nil || u.ForceQuery || strings.Contains(raw, "#") || u.RawPath != "" || u.EscapedPath() != u.Path || u.Hostname() == "" || strings.Contains(u.Hostname(), "%") {
		return nil, fault.New(fault.Trust, "source_resource_mismatch")
	}
	return u, nil
}

func azureURLOrigin(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" {
		port = "443"
	}
	return "https://" + net.JoinHostPort(host, port)
}

func azureSecretVersion(path, resource string) (string, error) {
	prefix := "/secrets/" + resource + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", fault.New(fault.Trust, "source_resource_mismatch")
	}
	version := strings.TrimPrefix(path, prefix)
	if version == "" || version == "." || version == ".." || strings.ContainsAny(version, "/\\\x00\r\n?#") {
		return "", fault.New(fault.Trust, "source_resource_mismatch")
	}
	return version, nil
}

func gcpResource(s model.Source, b model.Binding) (string, error) {
	parts := strings.Split(b.Resource, "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[1] != s.Project || parts[2] != "secrets" || parts[3] == "" {
		return "", fault.New(fault.Denied, "gcp_resource")
	}
	version := b.Version
	if version == "latest" {
		return "", fault.New(fault.Invalid, "gcp_version")
	}
	if version == "" {
		version = "latest"
	}
	if strings.ContainsAny(version, "/\x00?#") {
		return "", fault.New(fault.Invalid, "gcp_version")
	}
	return b.Resource + gcpVersionPathSeparator + version, nil
}

func readGCP(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
	name, err := gcpResource(s, b)
	if err != nil {
		return model.Value{}, err
	}
	if err := validateGCPIdentity(s); err != nil {
		return model.Value{}, err
	}
	client, err := secretmanager.NewClient(ctx, option.WithTokenSource(google.ComputeTokenSource(s.Account)), option.WithEndpoint("secretmanager.googleapis.com:443"))
	if err != nil {
		return model.Value{}, cloudError(err)
	}
	defer client.Close()
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return model.Value{}, cloudError(err)
	}
	return gcpSecretValue(result, s, b)
}

func validateGCPIdentity(s model.Source) error {
	if s.Identity != "metadata" || s.Account == "" || s.Account == "default" || s.Endpoint != "" || s.TokenFile != "" || s.Profile != "" || os.Getenv("GCE_METADATA_HOST") != "" {
		return fault.New(fault.Unsupported, "gcp_identity_method")
	}
	return nil
}

func gcpSecretValue(result *secretmanagerpb.AccessSecretVersionResponse, s model.Source, b model.Binding) (model.Value, error) {
	if result.Payload == nil {
		return model.Value{}, fault.New(fault.Missing, "source_value")
	}
	if !validCRC(result.Payload.Data, result.Payload.DataCrc32C) {
		return model.Value{}, fault.New(fault.Trust, "gcp_payload_integrity")
	}
	version, err := validateGCPReturnedResource(s, b, result.Name)
	if err != nil {
		return model.Value{}, err
	}
	data, err := selectValue(string(result.Payload.Data), b.Field)
	if err != nil {
		return model.Value{}, err
	}
	return model.Value{Data: data, Version: b.Resource + gcpVersionPathSeparator + version, AuditVerified: s.AuditVerified}, nil
}

func validateGCPReturnedResource(s model.Source, b model.Binding, actual string) (string, error) {
	if _, err := gcpResource(s, b); err != nil {
		return "", err
	}
	prefix := b.Resource + gcpVersionPathSeparator
	if !strings.HasPrefix(actual, prefix) {
		return "", fault.New(fault.Trust, "source_resource_mismatch")
	}
	version := strings.TrimPrefix(actual, prefix)
	if version == "" || version == "latest" || strings.ContainsAny(version, "/\x00?#") {
		return "", fault.New(fault.Trust, "source_resource_mismatch")
	}
	if b.Version != "" && version != b.Version {
		return "", fault.New(fault.Trust, "source_version_mismatch")
	}
	return version, nil
}

func cloudError(err error) error {
	var api smithy.APIError
	if errors.As(err, &api) {
		return cloudCode(api.ErrorCode())
	}
	var azure *azcore.ResponseError
	if errors.As(err, &azure) {
		return transport.Status(azure.StatusCode)
	}
	switch status.Code(err) {
	case codes.PermissionDenied, codes.Unauthenticated:
		return fault.New(fault.Denied, "provider_request")
	case codes.NotFound:
		return fault.New(fault.Missing, "provider_request")
	case codes.Unavailable, codes.DeadlineExceeded:
		return fault.New(fault.Unavailable, "provider_connection")
	}
	return fault.New(fault.Trust, "provider_failure")
}

func cloudCode(code string) error {
	switch code {
	case "AccessDenied", "AccessDeniedException", "UnrecognizedClientException", "ExpiredToken", "ExpiredTokenException":
		return fault.New(fault.Denied, "provider_request")
	case "ResourceNotFoundException", "ParameterNotFound":
		return fault.New(fault.Missing, "provider_request")
	default:
		return fault.New(fault.Trust, "provider_failure")
	}
}
