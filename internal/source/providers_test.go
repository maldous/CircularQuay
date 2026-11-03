package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func providerTestMain() bool {
	if len(os.Args) < 2 || (os.Args[1] != "secret" && os.Args[1] != "read") {
		return false
	}
	if os.Getenv("UNRELATED_SECRET") != "" {
		os.Exit(81)
	}
	if os.Args[1] == "read" {
		if os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") == "" {
			os.Exit(82)
		}
		fmt.Print("synthetic-op-value")
		os.Exit(0)
	}
	return providerVendorTestCommand()
}

func providerVendorTestCommand() bool {
	if os.Getenv("BWS_ACCESS_TOKEN") == "" {
		os.Exit(83)
	}
	if len(os.Args) > 3 && os.Args[3] == "overflow" {
		_, _ = io.Copy(os.Stdout, bytes.NewReader(bytes.Repeat([]byte("x"), 4<<20+10)))
		os.Exit(0)
	}
	if len(os.Args) > 3 && os.Args[3] == "failure" {
		os.Exit(84)
	}
	fmt.Print(`{"projectId":"synthetic-project","value":"synthetic-bws-value","revisionDate":"synthetic-revision"}`)
	os.Exit(0)
	return true
}

func tokenFixture(t *testing.T) string {
	t.Helper()
	directory := privateSourceDirectory(t, "credential")
	p := filepath.Join(directory, "token")
	if err := platform.WriteProtected(p, []byte("synthetic-token")); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRegistryAndLiteralSources(t *testing.T) {
	testRegistryScope(t)
	testVersionCapabilities(t)
	testJSONLiteralSource(t)
	testEnvironmentLiteralSource(t)
	if len(Capabilities()) < 10 {
		t.Fatal("capability inventory missing")
	}
}

func testVersionCapabilities(t *testing.T) {
	t.Helper()
	types := []struct {
		name      string
		versioned bool
	}{
		{"json", false}, {"env", false}, {"sops", false}, {"doppler", false}, {"akeyless", false}, {"bitwarden", false}, {"onepassword", false},
		{"openbao", true}, {"vault", true}, {"infisical", true}, {"aws-sm", true}, {"aws-ssm", true}, {"azure", true}, {"gcp", true},
	}
	capabilities := map[string]bool{}
	for _, capability := range Capabilities() {
		capabilities[capability.Type] = capability.Versioned
	}
	for _, test := range types {
		t.Run(test.name, func(t *testing.T) {
			testVersionCapability(t, test.name, test.versioned, capabilities)
		})
	}
}

func testVersionCapability(t *testing.T, name string, versioned bool, capabilities map[string]bool) {
	t.Helper()
	if model.SourceSupportsVersion(name) != versioned || capabilities[name] != versioned {
		t.Fatal("capability inventory disagrees with version contract")
	}
	calls := 0
	registry := NewRegistry()
	registry.Register(name, ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		calls++
		return model.Value{Data: "synthetic"}, nil
	}))
	source, binding := versionCapabilityRequest(name)
	_, err := registry.Read(context.Background(), source, binding)
	if versioned {
		requireVersionedRead(t, err, calls)
		return
	}
	requireUnversionedRejection(t, err, calls)
}

func versionCapabilityRequest(name string) (model.Source, model.Binding) {
	source := model.Source{Type: name, Resources: []string{"declared"}}
	binding := model.Binding{Resource: "declared", Version: "2"}
	if name == "json" || name == "sops" {
		source.Path = "/declared"
		binding.Resource = "/declared"
	}
	return source, binding
}

func requireVersionedRead(t *testing.T, err error, calls int) {
	t.Helper()
	if err != nil || calls != 1 {
		t.Fatalf("supported selector was not delivered: %v", err)
	}
}

func requireUnversionedRejection(t *testing.T, err error, calls int) {
	t.Helper()
	if !fault.Is(err, fault.Invalid) || calls != 0 {
		t.Fatalf("unsupported selector reached adapter: %v", err)
	}
}

func TestProviderVersionResponseEvidence(t *testing.T) {
	payload := []byte(`{"KEY":"value"}`)
	checksum := int64(crc32.Checksum(payload, crc32.MakeTable(crc32.Castagnoli)))
	tests := []struct {
		name string
		err  error
		want fault.Kind
	}{
		{"vault_match", vaultVersionError(2, "2"), ""},
		{"vault_mismatch", vaultVersionError(3, "2"), fault.Trust},
		{"infisical_match", infisicalVersionError(4, "4"), ""},
		{"infisical_mismatch", infisicalVersionError(5, "4"), fault.Trust},
		{"aws_secret_id_match", validateAWSSecretVersion(model.Binding{Version: "id"}, &secretsmanager.GetSecretValueOutput{VersionId: aws.String("id")}), ""},
		{"aws_secret_id_mismatch", validateAWSSecretVersion(model.Binding{Version: "id"}, &secretsmanager.GetSecretValueOutput{VersionId: aws.String("other")}), fault.Trust},
		{"aws_secret_stage_match", validateAWSSecretVersion(model.Binding{Version: "stage:CURRENT"}, &secretsmanager.GetSecretValueOutput{VersionStages: []string{"CURRENT"}}), ""},
		{"aws_secret_stage_mismatch", validateAWSSecretVersion(model.Binding{Version: "stage:CURRENT"}, &secretsmanager.GetSecretValueOutput{VersionStages: []string{"PREVIOUS"}}), fault.Trust},
		{"aws_parameter_match", validateAWSParameterVersion(model.Binding{Version: "2"}, &ssmtypes.Parameter{Version: 2}), ""},
		{"aws_parameter_label_match", validateAWSParameterVersion(model.Binding{Version: "approved"}, &ssmtypes.Parameter{Version: 2, Selector: aws.String(":approved")}), ""},
		{"aws_parameter_mismatch", validateAWSParameterVersion(model.Binding{Version: "2"}, &ssmtypes.Parameter{Version: 3}), fault.Trust},
		{"azure_match", validateReturnedVersion(model.Binding{Version: "version"}, "version"), ""},
		{"azure_mismatch", validateReturnedVersion(model.Binding{Version: "version"}, "other"), fault.Trust},
		{"gcp_match", gcpVersionError(payload, checksum, "projects/project/secrets/key/versions/2", "2"), ""},
		{"gcp_mismatch", gcpVersionError(payload, checksum, "projects/project/secrets/key/versions/3", "2"), fault.Trust},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want == "" && test.err != nil {
				t.Fatalf("version evidence rejected: %v", test.err)
			}
			if test.want != "" && !fault.Is(test.err, test.want) {
				t.Fatalf("version mismatch was not rejected: %v", test.err)
			}
		})
	}
}

func vaultVersionError(actual int, requested string) error {
	_, err := vaultValue(vaultResult{Data: struct {
		Data     map[string]any `json:"data"`
		Metadata struct {
			Version      int    `json:"version"`
			DeletionTime string `json:"deletion_time"`
			Destroyed    bool   `json:"destroyed"`
		} `json:"metadata"`
	}{Data: map[string]any{"KEY": "value"}, Metadata: struct {
		Version      int    `json:"version"`
		DeletionTime string `json:"deletion_time"`
		Destroyed    bool   `json:"destroyed"`
	}{Version: actual}}}, model.Source{}, model.Binding{Name: "KEY", Version: requested})
	return err
}

func infisicalVersionError(actual int, requested string) error {
	value := "value"
	_, err := infisicalValue(&value, actual, false, model.Source{}, model.Binding{Version: requested})
	return err
}

func gcpVersionError(payload []byte, checksum int64, name, requested string) error {
	_, err := gcpSecretValue(&secretmanagerpb.AccessSecretVersionResponse{Name: name, Payload: &secretmanagerpb.SecretPayload{Data: payload, DataCrc32C: &checksum}}, model.Source{Project: "project"}, model.Binding{Field: "KEY", Resource: "projects/project/secrets/key", Version: requested})
	return err
}

func testRegistryScope(t *testing.T) {
	ctx := context.Background()
	r := NewRegistry()
	r.Register("test", ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		return model.Value{Data: "safe"}, nil
	}))
	if _, err := r.Read(ctx, model.Source{Type: "test"}, model.Binding{Resource: "undeclared"}); !fault.Is(err, fault.Denied) {
		t.Fatal("undeclared read allowed")
	}
	if v, err := r.Read(ctx, model.Source{Type: "test", Resources: []string{"declared"}}, model.Binding{Resource: "declared"}); err != nil || v.Data != "safe" {
		t.Fatal("declared reader failed")
	}
	if _, err := r.Read(ctx, model.Source{Type: "unknown", Resources: []string{"declared"}}, model.Binding{Resource: "declared"}); !fault.Is(err, fault.Unsupported) {
		t.Fatal("unknown reader accepted")
	}
}

func testJSONLiteralSource(t *testing.T) {
	ctx := context.Background()
	r := NewRegistry()
	file := privateLiteralFile(t, "values.json")
	write := func(s string) {
		t.Helper()
		if err := platform.WriteProtected(file, []byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"KEY":"$(literal)","NUMBER":1}`)
	s := model.Source{Type: "json", Path: file}
	v, err := r.Read(ctx, s, model.Binding{Name: "KEY"})
	if err != nil || v.Data != "$(literal)" {
		t.Fatal("literal data interpreted")
	}
	if !strings.HasPrefix(v.Version, "file:") {
		t.Fatal("literal metadata hint was not preserved")
	}
	testJSONLiteralRejections(t, r, s, write)
}

func testJSONLiteralRejections(t *testing.T, r *Registry, s model.Source, write func(string)) {
	t.Helper()
	ctx := context.Background()
	for _, field := range []string{"MISSING", "NUMBER"} {
		if _, err := r.Read(ctx, s, model.Binding{Field: field}); err == nil {
			t.Fatal("invalid selected field accepted")
		}
	}
	for _, data := range []string{`not-json`, `{} {}`, `null`} {
		write(data)
		if _, err := r.Read(ctx, s, model.Binding{Name: "KEY"}); err == nil {
			t.Fatal("malformed source accepted")
		}
	}
	if _, err := readJSON(ctx, model.Source{Path: "relative"}, model.Binding{}); !fault.Is(err, fault.Invalid) {
		t.Fatal("relative file accepted")
	}
	testLiteralSourceProtection(t, r)
}

func testEnvironmentLiteralSource(t *testing.T) {
	ctx := context.Background()
	r := NewRegistry()
	file := privateLiteralFile(t, "values.env")
	write := func(s string) {
		t.Helper()
		if err := platform.WriteProtected(file, []byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TEST_LITERAL_VALUE", "literal-env")
	es := model.Source{Type: "env", Resources: []string{"TEST_LITERAL_VALUE"}}
	v, err := r.Read(ctx, es, model.Binding{Resource: "TEST_LITERAL_VALUE"})
	if err != nil || v.Data != "literal-env" {
		t.Fatal("explicit environment failed")
	}
	es.Path = file
	write("TEST_LITERAL_VALUE=literal-dotenv\n#comment\n")
	v, err = r.Read(ctx, es, model.Binding{Resource: "TEST_LITERAL_VALUE"})
	if err != nil || v.Data != "literal-dotenv" {
		t.Fatal("literal dotenv failed")
	}
	for _, data := range []string{"KEY=$(command)", "KEY=${EXPAND}", "KEY=`command`", "invalid", "KEY=one\nKEY=two"} {
		write(data)
		if _, err := r.Read(ctx, es, model.Binding{Resource: "TEST_LITERAL_VALUE"}); err == nil {
			t.Fatal("unsafe dotenv accepted")
		}
	}
}

func privateLiteralFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(privateSourceDirectory(t, "literal"), name)
}

func privateSourceDirectory(t *testing.T, name string) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(base, "cquay-source-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	directory := filepath.Join(root, name)
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func testLiteralSourceProtection(t *testing.T, registry *Registry) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	marker := "MUST-NOT-READ"
	t.Run("public file", func(t *testing.T) { testUntrustedLiteralSource(t, registry, publicLiteralFile(t, marker), marker) })
	t.Run("symlink", func(t *testing.T) { testUntrustedLiteralSource(t, registry, symlinkLiteralFile(t, marker), marker) })
	t.Run("unsafe parent", func(t *testing.T) {
		testUntrustedLiteralSource(t, registry, unsafeParentLiteralFile(t, marker), marker)
	})
}

func testUntrustedLiteralSource(t *testing.T, registry *Registry, path, marker string) {
	t.Helper()
	value, err := registry.Read(context.Background(), model.Source{Type: "json", Path: path}, model.Binding{Name: "KEY"})
	if !fault.Is(err, fault.Trust) || value.Data == marker {
		t.Fatalf("untrusted literal source result = %#v, %v", value, err)
	}
}

func publicLiteralFile(t *testing.T, marker string) string {
	t.Helper()
	path := privateLiteralFile(t, "public.json")
	writeLiteralMarker(t, path, marker)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func symlinkLiteralFile(t *testing.T, marker string) string {
	t.Helper()
	target := privateLiteralFile(t, "target.json")
	writeLiteralMarker(t, target, marker)
	link := filepath.Join(filepath.Dir(target), "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	return link
}

func unsafeParentLiteralFile(t *testing.T, marker string) string {
	t.Helper()
	directory := filepath.Dir(privateLiteralFile(t, "placeholder"))
	path := filepath.Join(directory, "parent.json")
	writeLiteralMarker(t, path, marker)
	if err := os.Chmod(directory, 0o777); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeLiteralMarker(t *testing.T, path, marker string) {
	t.Helper()
	if err := platform.WriteProtected(path, []byte(`{"KEY":"`+marker+`"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestVaultAndHostedContracts(t *testing.T) {
	requests := 0
	server := httptest.NewTLSServer(hostedContractHandler(t, &requests))
	defer server.Close()
	r := NewRegistryWithClient(server.Client())
	token := tokenFixture(t)
	assertHostedContracts(t, r, server.URL, token)
	before := requests
	_, err := r.Read(context.Background(), model.Source{Type: "openbao", Endpoint: server.URL, Mount: "kv", TokenFile: token, Resources: []string{"permitted"}}, model.Binding{Resource: "forbidden"})
	if !fault.Is(err, fault.Denied) || requests != before {
		t.Fatal("denial made an upstream request")
	}
}

func hostedContractHandler(t *testing.T, requests *int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests++
		switch r.URL.Path {
		case "/v1/kv/data/app":
			writeVaultContract(t, w, r)
		case "/v3/configs/config/secret":
			writeDopplerContract(t, w, r)
		case "/api/v4/secrets/KEY":
			writeInfisicalContract(t, w, r)
		case "/get-secret-value":
			writeAkeylessContract(t, w, r)
		default:
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "RAW-ERROR-MUST-NOT-LEAK")
		}
	})
}

func writeVaultContract(t *testing.T, w http.ResponseWriter, request *http.Request) {
	if request.Header.Get("X-Vault-Token") != "synthetic-token" || request.URL.Query().Get("version") != "2" {
		t.Error("vault request scope mismatch")
	}
	fmt.Fprint(w, `{"data":{"data":{"KEY":"vault-value"},"metadata":{"version":2}},"lease_duration":3600}`)
}

func writeDopplerContract(t *testing.T, w http.ResponseWriter, request *http.Request) {
	if request.URL.Query().Get("name") != "KEY" || request.Header.Get("Authorization") != "Bearer synthetic-token" {
		t.Error("doppler scope mismatch")
	}
	fmt.Fprint(w, `{"value":{"raw":"doppler-value","computed":"not-selected"}}`)
}

func writeInfisicalContract(t *testing.T, w http.ResponseWriter, request *http.Request) {
	if request.URL.Query().Get("secretPath") != "/application" || request.URL.Query().Get("includeImports") != "false" || request.URL.Query().Get("expandSecretReferences") != "false" {
		t.Error("implicit inheritance enabled")
	}
	fmt.Fprint(w, `{"secret":{"secretValue":"infisical-value","version":4}}`)
}

func writeAkeylessContract(t *testing.T, w http.ResponseWriter, request *http.Request) {
	var body map[string]any
	if json.NewDecoder(request.Body).Decode(&body) != nil || body["token"] != "synthetic-token" {
		t.Error("akeyless auth mismatch")
	}
	fmt.Fprint(w, `{"KEY":"akeyless-value"}`)
}

type hostedContractCase struct{ typ, resource, version, path, want string }

func assertHostedContracts(t *testing.T, registry *Registry, endpoint, token string) {
	for _, test := range []hostedContractCase{{"openbao", "app", "2", "", "vault-value"}, {"vault", "app", "2", "", "vault-value"}, {"doppler", "KEY", "", "", "doppler-value"}, {"infisical", "KEY", "4", "/application", "infisical-value"}, {"akeyless", "KEY", "", "", "akeyless-value"}} {
		test.assert(t, registry, endpoint, token)
	}
}

func (test hostedContractCase) assert(t *testing.T, registry *Registry, endpoint, token string) {
	t.Helper()
	source := model.Source{Type: test.typ, Endpoint: endpoint, Mount: "kv", Path: test.path, TokenFile: token, Project: "project", Environment: "test", Resources: []string{test.resource}}
	value, err := registry.Read(context.Background(), source, model.Binding{Name: "KEY", Resource: test.resource, Version: test.version})
	if err != nil || value.Data != test.want {
		t.Fatalf("%s contract failed: %v", test.typ, err)
	}
	if value.Leased {
		t.Fatal("static KV duration incorrectly treated as lease")
	}
}

func TestProviderDenialAndMalformedResponse(t *testing.T) {
	for _, body := range []string{`{"data":{"data":{"KEY":"x"},"metadata":{"destroyed":true}}}`, `{"data":{"data":{"KEY":"x"},"metadata":{"deletion_time":"today"}}}`, `{"data":{"data":{"KEY":1}}}`, `not-json`} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		r := NewRegistryWithClient(server.Client())
		_, err := r.Read(context.Background(), model.Source{Type: "openbao", Endpoint: server.URL, Mount: "kv", TokenFile: tokenFixture(t), Resources: []string{"app"}}, model.Binding{Name: "KEY", Resource: "app"})
		server.Close()
		if err == nil {
			t.Fatal("invalid provider response accepted")
		}
	}
	for _, typ := range []string{"openbao", "doppler", "infisical", "akeyless"} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(403)
			fmt.Fprint(w, "RAW-ERROR-MUST-NOT-LEAK")
		}))
		r := NewRegistryWithClient(server.Client())
		source := model.Source{Type: typ, Endpoint: server.URL, Mount: "kv", Project: "project", Environment: "test", TokenFile: tokenFixture(t), Resources: []string{"app"}}
		if typ == "infisical" {
			source.Path = "/application"
		}
		_, err := r.Read(context.Background(), source, model.Binding{Name: "KEY", Resource: "app"})
		server.Close()
		if !fault.Is(err, fault.Denied) || strings.Contains(err.Error(), "RAW-") {
			t.Fatal("denial not safely classified")
		}
	}
}

func TestNativeVendorContracts(t *testing.T) {
	t.Setenv("UNRELATED_SECRET", "must-not-inherit")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	token := tokenFixture(t)
	for _, tc := range []struct{ typ, resource, want string }{{"bitwarden", "test-id", "synthetic-bws-value"}, {"onepassword", "op://synthetic-project/item/field", "synthetic-op-value"}} {
		s := model.Source{Type: tc.typ, Path: exe, TokenFile: token, Project: "synthetic-project", Resources: []string{tc.resource}}
		v, err := r.Read(context.Background(), s, model.Binding{Resource: tc.resource})
		if err != nil || v.Data != tc.want {
			t.Fatalf("%s native contract failed: %v", tc.typ, err)
		}
	}
	overflow := model.Source{Type: "bitwarden", Path: exe, TokenFile: token, Project: "synthetic-project", Resources: []string{"overflow"}}
	if _, err := r.Read(context.Background(), overflow, model.Binding{Resource: "overflow"}); !fault.Is(err, fault.Trust) {
		t.Fatal("vendor subprocess output limit was not enforced")
	}
	if _, _, err := vendorCommand(model.Source{Type: "onepassword", Project: "permitted"}, model.Binding{Resource: "op://other/item/field"}); !fault.Is(err, fault.Denied) {
		t.Fatal("cross-vault read allowed")
	}
	if _, _, err := vendorCommand(model.Source{Type: "onepassword", Environment: "unsupported"}, model.Binding{}); !fault.Is(err, fault.Unsupported) {
		t.Fatal("unimplemented environment claimed")
	}
}

func TestLimitedBufferCopyCannotBypassLimit(t *testing.T) {
	var output limitedBuffer
	data := bytes.Repeat([]byte("x"), 4<<20+1)
	if _, err := io.Copy(&output, io.LimitReader(bytes.NewReader(data), int64(len(data)))); err != nil || !output.overflow || len(output.Bytes()) > 4<<20 {
		t.Fatal("vendor output cap bypassed by copy")
	}
}

func TestCloudScopeAndErrors(t *testing.T) {
	testAWSCloudScope(t)
	testAzureCloudIdentity(t)
	testGCPCloudScope(t)
	testCloudErrorClassification(t)
	testCloudCRC(t)
}

func testAWSCloudScope(t *testing.T) {
	s := model.Source{Account: "123456789012", Region: "test-region"}
	if validateAWSResource(s, model.Binding{Resource: "arn:aws:secretsmanager:test-region:123456789012:secret:key"}, "secretsmanager") != nil {
		t.Fatal("exact ARN failed")
	}
	if !fault.Is(validateAWSResource(s, model.Binding{Resource: "arn:aws:secretsmanager:test-region:other:secret:key"}, "secretsmanager"), fault.Denied) {
		t.Fatal("cross-account ARN allowed")
	}
	if _, err := awsConfiguration(context.Background(), model.Source{}); !fault.Is(err, fault.Invalid) {
		t.Fatal("ambiguous AWS identity accepted")
	}
	if _, err := awsConfiguration(context.Background(), model.Source{Account: "id", Region: "region", Identity: "profile"}); !fault.Is(err, fault.Unsupported) {
		t.Fatal("interactive AWS fallback accepted")
	}
	t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "https://malicious.invalid")
	if _, err := awsConfiguration(context.Background(), model.Source{Account: "id", Region: "region", Identity: "ecs"}); !fault.Is(err, fault.Invalid) {
		t.Fatal("credential endpoint override accepted")
	}
}

func testAzureCloudIdentity(t *testing.T) {
	if _, err := azureCredential(model.Source{}); !fault.Is(err, fault.Invalid) {
		t.Fatal("missing tenant accepted")
	}
	if _, err := azureCredential(model.Source{Account: "tenant", Identity: "developer"}); !fault.Is(err, fault.Unsupported) {
		t.Fatal("developer Azure fallback accepted")
	}
	if _, err := azureCredential(model.Source{Account: "tenant", Identity: "workload"}); !fault.Is(err, fault.Invalid) {
		t.Fatal("incomplete workload identity accepted")
	}
}

func testGCPCloudScope(t *testing.T) {
	if name, err := gcpResource(model.Source{Project: "project"}, model.Binding{Resource: "projects/project/secrets/key"}); err != nil || name != "projects/project/secrets/key/versions/latest" {
		t.Fatal("exact GCP resource failed")
	}
	if _, err := gcpResource(model.Source{Project: "project"}, model.Binding{Resource: "projects/other/secrets/key"}); !fault.Is(err, fault.Denied) {
		t.Fatal("cross-project read allowed")
	}
	if _, err := gcpResource(model.Source{Project: "project"}, model.Binding{Resource: "projects/project/secrets/key", Version: "../escape"}); !fault.Is(err, fault.Invalid) {
		t.Fatal("version traversal allowed")
	}
}

func testCloudErrorClassification(t *testing.T) {
	for _, err := range []error{&smithy.GenericAPIError{Code: "AccessDeniedException"}, &azcore.ResponseError{StatusCode: 403}, status.Error(codes.PermissionDenied, "must-not-leak")} {
		if !fault.Is(cloudError(err), fault.Denied) {
			t.Fatal("denial misclassified")
		}
	}
	if !fault.Is(cloudError(status.Error(codes.Unavailable, "must-not-leak")), fault.Unavailable) {
		t.Fatal("availability misclassified")
	}
	if !fault.Is(cloudCode("ParameterNotFound"), fault.Missing) {
		t.Fatal("missing parameter misclassified")
	}
}

func testCloudCRC(t *testing.T) {
	data := []byte("fixture")
	crc := int64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
	negative := int64(-1)
	if !validCRC(data, &crc) || validCRC(data, nil) || validCRC(data, &negative) {
		t.Fatal("integrity check failed")
	}
}
