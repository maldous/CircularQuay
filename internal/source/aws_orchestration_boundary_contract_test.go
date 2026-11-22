package source

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

const awsOrchestrationAccount = "123456789012"
const awsOrchestrationRegion = "ap-southeast-2"

func TestAWSOrchestrationBoundaryContractIdentityShapePrecedesDependencies(t *testing.T) {
	for _, field := range []string{"account", "region", "profile", "endpoint"} {
		t.Run(field, func(t *testing.T) {
			assertAWSOrchestrationIdentityShape(t, field)
		})
	}
}

func assertAWSOrchestrationIdentityShape(t *testing.T, field string) {
	t.Helper()
	source := awsOrchestrationSource()
	switch field {
	case "account":
		source.Account = ""
	case "region":
		source.Region = ""
	case "profile":
		source.Profile = "unapproved-developer-profile"
	case "endpoint":
		source.Endpoint = "https://unapproved.example.test"
	}
	config, err := awsConfigurationWithTransport(context.Background(), source, func(model.Source) (aws.CredentialsProvider, error) {
		t.Fatal("invalid identity selected credentials")
		return nil, fault.New(fault.Trust, "unexpected credentials")
	}, awsOrchestrationTripwire(t))
	assertAWSOrchestrationConfigRefusal(t, config, err, fault.Invalid, "aws_identity")
}

func TestAWSOrchestrationBoundaryContractClosedProviderFailurePrecedesHTTP(t *testing.T) {
	source := awsOrchestrationSource()
	source.Identity = "unsupported-synthetic-method"
	config, err := awsConfigurationWithTransport(context.Background(), source, awsCredentialsProvider, awsOrchestrationTripwire(t))
	assertAWSOrchestrationConfigRefusal(t, config, err, fault.Unsupported, "aws_identity_method")
}

func TestAWSOrchestrationBoundaryContractProductionWrappersKeepClosedIdentitySelection(t *testing.T) {
	source := awsOrchestrationSource()
	source.Identity = "unsupported-synthetic-method"
	config, err := awsConfiguration(context.Background(), source)
	assertAWSOrchestrationConfigRefusal(t, config, err, fault.Unsupported, "aws_identity_method")
	value, err := readAWSSecret(context.Background(), source, model.Binding{Resource: awsOrchestrationARN("secretsmanager")})
	assertAWSOrchestrationValueRefusal(t, value, err, fault.Unsupported, "aws_identity_method")
	value, err = readAWSParameter(context.Background(), source, model.Binding{Resource: awsOrchestrationARN("ssm")})
	assertAWSOrchestrationValueRefusal(t, value, err, fault.Unsupported, "aws_identity_method")
}

func TestAWSOrchestrationBoundaryContractCanceledIdentityNeverRequestsValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fixture := &awsOrchestrationFixture{cancel: cancel}
	config, err := fixture.configure(t, ctx, awsOrchestrationSource())
	assertAWSOrchestrationConfigRefusal(t, config, err, fault.Trust, "provider_failure")
	if ctx.Err() != context.Canceled || !reflect.DeepEqual(fixture.events, []string{"identity"}) {
		t.Fatal("identity cancellation selected a later value request")
	}
}

func TestAWSOrchestrationBoundaryContractSTSFailureWithholdsConfiguration(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		kind      fault.Kind
		operation string
	}{
		{"denied", fault.Denied, "provider_request"},
		{"malformed", fault.Trust, "provider_failure"},
		{"wrong-account", fault.Denied, "aws_account"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := &awsOrchestrationFixture{scenario: scenario.name}
			source := awsOrchestrationSource()
			config, err := fixture.configure(t, context.Background(), source)
			assertAWSOrchestrationConfigRefusal(t, config, err, scenario.kind, scenario.operation)
			fixture.assertBodiesClosed(t)
			if !reflect.DeepEqual(fixture.events, []string{"identity"}) {
				t.Fatal("failed identity request selected a value adapter")
			}
			if !reflect.DeepEqual(source, awsOrchestrationSource()) {
				t.Fatal("identity failure mutated the declared source")
			}
		})
	}
}

func TestAWSOrchestrationBoundaryContractMatchingAccountReturnsExactConfiguration(t *testing.T) {
	fixture := &awsOrchestrationFixture{}
	config, err := fixture.configure(t, context.Background(), awsOrchestrationSource())
	if err != nil || config.Region != awsOrchestrationRegion || config.Credentials == nil || config.HTTPClient == nil {
		t.Fatal("matching account did not return bounded configuration", err)
	}
	want := aws.Config{Region: awsOrchestrationRegion, Credentials: config.Credentials, HTTPClient: config.HTTPClient}
	if !reflect.DeepEqual(config, want) {
		t.Fatal("configuration introduced additional endpoint or identity settings")
	}
	fixture.assertBodiesClosed(t)
	if fixture.factories != 1 || fixture.retrievals != 1 || !reflect.DeepEqual(fixture.events, []string{"identity"}) {
		t.Fatal("identity construction did not use only the explicit collaborator")
	}
}

func TestAWSOrchestrationBoundaryContractARNAdmissionPrecedesConfigurator(t *testing.T) {
	for _, service := range []string{"secretsmanager", "ssm"} {
		for _, denied := range []string{"account", "region", "service"} {
			t.Run(service+"-"+denied, func(t *testing.T) {
				assertAWSOrchestrationARNRefusal(t, service, denied)
			})
		}
	}
}

func assertAWSOrchestrationARNRefusal(t *testing.T, service, denied string) {
	t.Helper()
	resource := awsOrchestrationARN(service)
	switch denied {
	case "account":
		resource = strings.Replace(resource, awsOrchestrationAccount, "999999999999", 1)
	case "region":
		resource = strings.Replace(resource, awsOrchestrationRegion, "us-east-1", 1)
	case "service":
		resource = strings.Replace(resource, ":"+service+":", ":other:", 1)
	}
	configure := func(context.Context, model.Source) (aws.Config, error) {
		t.Fatal("unadmitted ARN selected identity")
		return aws.Config{}, fault.New(fault.Trust, "unexpected identity")
	}
	value, err := awsOrchestrationRead(service)(context.Background(), awsOrchestrationSource(), model.Binding{Resource: resource}, configure)
	assertAWSOrchestrationValueRefusal(t, value, err, fault.Denied, "aws_resource")
}

func TestAWSOrchestrationBoundaryContractConfiguratorFailureNeverSelectsValue(t *testing.T) {
	for _, service := range []string{"secretsmanager", "ssm"} {
		t.Run(service, func(t *testing.T) {
			refusal := fault.New(fault.Denied, "provider_request")
			calls := 0
			configure := func(_ context.Context, source model.Source) (aws.Config, error) {
				calls++
				if !reflect.DeepEqual(source, awsOrchestrationSource()) {
					t.Fatal("configurator received a different source")
				}
				return aws.Config{}, refusal
			}
			value, err := awsOrchestrationRead(service)(context.Background(), awsOrchestrationSource(), model.Binding{Resource: awsOrchestrationARN(service)}, configure)
			assertAWSOrchestrationValueRefusal(t, value, err, fault.Denied, "provider_request")
			if calls != 1 || err != refusal {
				t.Fatal("identity refusal was replaced or repeated")
			}
		})
	}
}

func TestAWSOrchestrationBoundaryContractIdentityPrecedesExactGeneratedValueRequest(t *testing.T) {
	for _, service := range []string{"secretsmanager", "ssm"} {
		t.Run(service, func(t *testing.T) {
			assertAWSOrchestrationValue(t, service)
		})
	}
}

func assertAWSOrchestrationValue(t *testing.T, service string) {
	t.Helper()
	fixture := &awsOrchestrationFixture{}
	configure := func(ctx context.Context, source model.Source) (aws.Config, error) {
		return fixture.configure(t, ctx, source)
	}
	binding := model.Binding{Resource: awsOrchestrationARN(service), Field: "KEY", Version: "7"}
	source := awsOrchestrationSource()
	source.AuditVerified = true
	value, err := awsOrchestrationRead(service)(context.Background(), source, binding, configure)
	want := model.Value{Data: "SYNTHETIC_VALUE", Version: "7", AuditVerified: true}
	if err != nil || value != want {
		t.Fatal("generated value request lost its scoped result", err)
	}
	fixture.assertBodiesClosed(t)
	if !reflect.DeepEqual(fixture.events, []string{"identity", service}) || fixture.factories != 1 || fixture.retrievals != 1 {
		t.Fatal("value was requested before matching identity or used another credential source")
	}
	if binding.Resource != awsOrchestrationARN(service) || binding.Field != "KEY" || binding.Version != "7" {
		t.Fatal("value retrieval mutated the declared binding")
	}
}

type awsOrchestrationReader func(context.Context, model.Source, model.Binding, func(context.Context, model.Source) (aws.Config, error)) (model.Value, error)

func awsOrchestrationRead(service string) awsOrchestrationReader {
	if service == "ssm" {
		return readAWSParameterWithConfiguration
	}
	return readAWSSecretWithConfiguration
}

func awsOrchestrationSource() model.Source {
	return model.Source{Account: awsOrchestrationAccount, Region: awsOrchestrationRegion, Identity: "ec2"}
}

func awsOrchestrationARN(service string) string {
	resource := "secret:synthetic"
	if service == "ssm" {
		resource = "parameter/synthetic"
	}
	return "arn:aws:" + service + ":" + awsOrchestrationRegion + ":" + awsOrchestrationAccount + ":" + resource
}

func awsOrchestrationTripwire(t *testing.T) http.RoundTripper {
	t.Helper()
	return cloudRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("pre-effect refusal selected HTTP")
		return nil, fault.New(fault.Trust, "unexpected HTTP")
	})
}

type awsOrchestrationFixture struct {
	scenario   string
	factories  int
	retrievals int
	events     []string
	bodies     []*awsOrchestrationResponseBody
	cancel     context.CancelFunc
}

func (fixture *awsOrchestrationFixture) configure(t *testing.T, ctx context.Context, source model.Source) (aws.Config, error) {
	t.Helper()
	credentials := func(actual model.Source) (aws.CredentialsProvider, error) {
		fixture.factories++
		if !reflect.DeepEqual(actual, source) {
			t.Fatal("credential selector received a different source")
		}
		return aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			fixture.retrievals++
			return aws.Credentials{AccessKeyID: "SYNTHETIC_ACCESS_KEY", SecretAccessKey: "SYNTHETIC_SECRET_KEY", SessionToken: "SYNTHETIC_SESSION_TOKEN", Source: "synthetic-owned-fixture"}, nil
		}), nil
	}
	return awsConfigurationWithTransport(ctx, source, credentials, cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		return fixture.roundTrip(t, request)
	}))
}

func (fixture *awsOrchestrationFixture) roundTrip(t *testing.T, request *http.Request) (*http.Response, error) {
	t.Helper()
	assertAWSOrchestrationRequest(t, request, len(fixture.events))
	switch request.URL.Host {
	case "sts.ap-southeast-2.amazonaws.com":
		fixture.events = append(fixture.events, "identity")
		if fixture.cancel != nil {
			fixture.cancel()
			return nil, request.Context().Err()
		}
		return fixture.identityResponse(t, request), nil
	case "secretsmanager.ap-southeast-2.amazonaws.com":
		fixture.events = append(fixture.events, "secretsmanager")
		return fixture.valueResponse(t, request, "secretsmanager"), nil
	case "ssm.ap-southeast-2.amazonaws.com":
		fixture.events = append(fixture.events, "ssm")
		return fixture.valueResponse(t, request, "ssm"), nil
	default:
		t.Fatal("generated client selected an unlisted origin")
		return nil, fault.New(fault.Trust, "unexpected origin")
	}
}

func assertAWSOrchestrationRequest(t *testing.T, request *http.Request, previous int) {
	t.Helper()
	if previous >= 2 {
		t.Fatal("generated client exceeded the identity-plus-value request bound")
	}
	if request.Method != http.MethodPost || request.URL.Scheme != "https" || request.URL.Path != "/" || request.URL.RawQuery != "" {
		t.Fatal("generated client escaped the fixed synthetic POST origin/path")
	}
}

func (fixture *awsOrchestrationFixture) identityResponse(t *testing.T, request *http.Request) *http.Response {
	t.Helper()
	form, err := url.ParseQuery(string(awsOrchestrationRequestBody(t, request)))
	if err != nil || !reflect.DeepEqual(form, url.Values{"Action": {"GetCallerIdentity"}, "Version": {"2011-06-15"}}) {
		t.Fatal("generated STS client changed its identity operation")
	}
	if fixture.scenario == "denied" {
		return fixture.response(request, http.StatusForbidden, "text/xml", `<ErrorResponse><Error><Type>Sender</Type><Code>AccessDenied</Code><Message>SYNTHETIC_PRIVATE_DETAIL</Message></Error><RequestId>synthetic</RequestId></ErrorResponse>`)
	}
	if fixture.scenario == "malformed" {
		return fixture.response(request, http.StatusOK, "text/xml", `<GetCallerIdentityResponse><broken`)
	}
	account := awsOrchestrationAccount
	if fixture.scenario == "wrong-account" {
		account = "999999999999"
	}
	body := `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><GetCallerIdentityResult><Arn>arn:aws:sts::` + account + `:assumed-role/synthetic/session</Arn><UserId>SYNTHETIC</UserId><Account>` + account + `</Account></GetCallerIdentityResult><ResponseMetadata><RequestId>synthetic</RequestId></ResponseMetadata></GetCallerIdentityResponse>`
	return fixture.response(request, http.StatusOK, "text/xml", body)
}

func (fixture *awsOrchestrationFixture) valueResponse(t *testing.T, request *http.Request, service string) *http.Response {
	t.Helper()
	var input map[string]any
	if err := json.Unmarshal(awsOrchestrationRequestBody(t, request), &input); err != nil {
		t.Fatal("generated value request was not JSON")
	}
	if service == "ssm" {
		if request.Header.Get("X-Amz-Target") != "AmazonSSM.GetParameter" || !reflect.DeepEqual(input, map[string]any{"Name": awsOrchestrationARN(service) + ":7", "WithDecryption": true}) {
			t.Fatal("SSM lost exact resource/version/decryption scope")
		}
		return fixture.response(request, http.StatusOK, "application/x-amz-json-1.1", `{"Parameter":{"ARN":"`+awsOrchestrationARN(service)+`","Value":"{\"KEY\":\"SYNTHETIC_VALUE\"}","Version":7}}`)
	}
	if request.Header.Get("X-Amz-Target") != "secretsmanager.GetSecretValue" || !reflect.DeepEqual(input, map[string]any{"SecretId": awsOrchestrationARN(service), "VersionId": "7"}) {
		t.Fatal("Secrets Manager lost exact resource/version scope")
	}
	return fixture.response(request, http.StatusOK, "application/x-amz-json-1.1", `{"ARN":"`+awsOrchestrationARN(service)+`","SecretString":"{\"KEY\":\"SYNTHETIC_VALUE\"}","VersionId":"7"}`)
}

func awsOrchestrationRequestBody(t *testing.T, request *http.Request) []byte {
	t.Helper()
	data, err := io.ReadAll(io.LimitReader(request.Body, 16385))
	if err != nil || len(data) > 16384 {
		t.Fatal("synthetic request exceeded its finite body contract")
	}
	return data
}

func (fixture *awsOrchestrationFixture) response(request *http.Request, code int, contentType, text string) *http.Response {
	body := &awsOrchestrationResponseBody{Reader: strings.NewReader(text)}
	fixture.bodies = append(fixture.bodies, body)
	return &http.Response{Request: request, StatusCode: code, Header: http.Header{"Content-Type": {contentType}}, Body: body}
}

type awsOrchestrationResponseBody struct {
	*strings.Reader
	closed bool
}

func (body *awsOrchestrationResponseBody) Close() error {
	body.closed = true
	return nil
}

func (fixture *awsOrchestrationFixture) assertBodiesClosed(t *testing.T) {
	t.Helper()
	for _, body := range fixture.bodies {
		if !body.closed {
			t.Fatal("generated SDK left an owned synthetic response body open")
		}
	}
}

func assertAWSOrchestrationConfigRefusal(t *testing.T, config aws.Config, err error, kind fault.Kind, operation string) {
	t.Helper()
	assertJSONRefusalError(t, err, kind, operation)
	if !reflect.DeepEqual(config, aws.Config{}) {
		t.Fatal("identity refusal returned a partial credential/client configuration")
	}
}

func assertAWSOrchestrationValueRefusal(t *testing.T, value model.Value, err error, kind fault.Kind, operation string) {
	t.Helper()
	assertJSONRefusalError(t, err, kind, operation)
	if value != (model.Value{}) {
		t.Fatal("identity refusal returned a partial value or metadata")
	}
}
