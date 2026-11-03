package source

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/aws/smithy-go"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
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
	if os.Getenv("BWS_ACCESS_TOKEN") == "" {
		os.Exit(83)
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
	p := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(p, []byte("synthetic-token"), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRegistryAndLiteralSources(t *testing.T) {
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
	file := filepath.Join(t.TempDir(), "values.json")
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(file, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"KEY":"$(literal)","NUMBER":1}`)
	s := model.Source{Type: "json", Path: file}
	v, err := r.Read(ctx, s, model.Binding{Name: "KEY"})
	if err != nil || v.Data != "$(literal)" {
		t.Fatal("literal data interpreted")
	}
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
	t.Setenv("CQUAY_TEST_LITERAL", "literal-env")
	es := model.Source{Type: "env", Resources: []string{"CQUAY_TEST_LITERAL"}}
	v, err = r.Read(ctx, es, model.Binding{Resource: "CQUAY_TEST_LITERAL"})
	if err != nil || v.Data != "literal-env" {
		t.Fatal("explicit environment failed")
	}
	es.Path = file
	write("CQUAY_TEST_LITERAL=literal-dotenv\n#comment\n")
	v, err = r.Read(ctx, es, model.Binding{Resource: "CQUAY_TEST_LITERAL"})
	if err != nil || v.Data != "literal-dotenv" {
		t.Fatal("literal dotenv failed")
	}
	for _, data := range []string{"KEY=$(command)", "KEY=${EXPAND}", "KEY=`command`", "invalid", "KEY=one\nKEY=two"} {
		write(data)
		if _, err := r.Read(ctx, es, model.Binding{Resource: "CQUAY_TEST_LITERAL"}); err == nil {
			t.Fatal("unsafe dotenv accepted")
		}
	}
	if len(Capabilities()) < 10 {
		t.Fatal("capability inventory missing")
	}
}

func TestVaultAndHostedContracts(t *testing.T) {
	requests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/v1/kv/data/app":
			if r.Header.Get("X-Vault-Token") != "synthetic-token" || r.URL.Query().Get("version") != "2" {
				t.Error("vault request scope mismatch")
			}
			fmt.Fprint(w, `{"data":{"data":{"KEY":"vault-value"},"metadata":{"version":2}},"lease_duration":3600}`)
		case "/v3/configs/config/secret":
			if r.URL.Query().Get("name") != "KEY" || r.Header.Get("Authorization") != "Bearer synthetic-token" {
				t.Error("doppler scope mismatch")
			}
			fmt.Fprint(w, `{"value":{"raw":"doppler-value","computed":"not-selected"}}`)
		case "/api/v4/secrets/KEY":
			if r.URL.Query().Get("includeImports") != "false" || r.URL.Query().Get("expandSecretReferences") != "false" {
				t.Error("implicit inheritance enabled")
			}
			fmt.Fprint(w, `{"secret":{"secretValue":"infisical-value","version":4}}`)
		case "/get-secret-value":
			var body map[string]any
			if json.NewDecoder(r.Body).Decode(&body) != nil || body["token"] != "synthetic-token" {
				t.Error("akeyless auth mismatch")
			}
			fmt.Fprint(w, `{"KEY":"akeyless-value"}`)
		default:
			w.WriteHeader(403)
			fmt.Fprint(w, "RAW-ERROR-MUST-NOT-LEAK")
		}
	}))
	defer server.Close()
	r := NewRegistryWithClient(server.Client())
	token := tokenFixture(t)
	for _, tc := range []struct{ typ, resource, version, want string }{{"openbao", "app", "2", "vault-value"}, {"vault", "app", "2", "vault-value"}, {"doppler", "KEY", "", "doppler-value"}, {"infisical", "KEY", "4", "infisical-value"}, {"akeyless", "KEY", "2", "akeyless-value"}} {
		s := model.Source{Type: tc.typ, Endpoint: server.URL, Mount: "kv", TokenFile: token, Project: "project", Environment: "test", Resources: []string{tc.resource}}
		v, err := r.Read(context.Background(), s, model.Binding{Name: "KEY", Resource: tc.resource, Version: tc.version})
		if err != nil || v.Data != tc.want {
			t.Fatalf("%s contract failed: %v", tc.typ, err)
		}
		if v.Leased {
			t.Fatal("static KV duration incorrectly treated as lease")
		}
	}
	before := requests
	_, err := r.Read(context.Background(), model.Source{Type: "openbao", Endpoint: server.URL, Mount: "kv", TokenFile: token, Resources: []string{"permitted"}}, model.Binding{Resource: "forbidden"})
	if !fault.Is(err, fault.Denied) || requests != before {
		t.Fatal("denial made an upstream request")
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
		_, err := r.Read(context.Background(), model.Source{Type: typ, Endpoint: server.URL, Mount: "kv", Project: "project", Environment: "test", TokenFile: tokenFixture(t), Resources: []string{"app"}}, model.Binding{Name: "KEY", Resource: "app"})
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
	if _, _, err := vendorCommand(model.Source{Type: "onepassword", Project: "permitted"}, model.Binding{Resource: "op://other/item/field"}); !fault.Is(err, fault.Denied) {
		t.Fatal("cross-vault read allowed")
	}
	if _, _, err := vendorCommand(model.Source{Type: "onepassword", Environment: "unsupported"}, model.Binding{}); !fault.Is(err, fault.Unsupported) {
		t.Fatal("unimplemented environment claimed")
	}
}

func TestCloudScopeAndErrors(t *testing.T) {
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
	if _, err := azureCredential(model.Source{}); !fault.Is(err, fault.Invalid) {
		t.Fatal("missing tenant accepted")
	}
	if _, err := azureCredential(model.Source{Account: "tenant", Identity: "developer"}); !fault.Is(err, fault.Unsupported) {
		t.Fatal("developer Azure fallback accepted")
	}
	if _, err := azureCredential(model.Source{Account: "tenant", Identity: "workload"}); !fault.Is(err, fault.Invalid) {
		t.Fatal("incomplete workload identity accepted")
	}
	if name, err := gcpResource(model.Source{Project: "project"}, model.Binding{Resource: "projects/project/secrets/key"}); err != nil || name != "projects/project/secrets/key/versions/latest" {
		t.Fatal("exact GCP resource failed")
	}
	if _, err := gcpResource(model.Source{Project: "project"}, model.Binding{Resource: "projects/other/secrets/key"}); !fault.Is(err, fault.Denied) {
		t.Fatal("cross-project read allowed")
	}
	if _, err := gcpResource(model.Source{Project: "project"}, model.Binding{Resource: "projects/project/secrets/key", Version: "../escape"}); !fault.Is(err, fault.Invalid) {
		t.Fatal("version traversal allowed")
	}
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
	data := []byte("fixture")
	crc := int64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
	if !validCRC(data, &crc) || validCRC(data, nil) {
		t.Fatal("integrity check failed")
	}
}
