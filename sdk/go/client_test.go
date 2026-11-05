package cq

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeCaller struct {
	tools      *mcp.ListToolsResult
	listErr    error
	result     *mcp.CallToolResult
	callErr    error
	callParams *mcp.CallToolParams
	callCtx    context.Context
	onCall     func()
}

func (f *fakeCaller) ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return f.tools, f.listErr
}

func (f *fakeCaller) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	f.callCtx, f.callParams = ctx, params
	if f.onCall != nil {
		f.onCall()
	}
	return f.result, f.callErr
}

type panicObserver struct{}

func (panicObserver) Observe(Event) { panic("observer failure") }

func TestClientConformsToCurrentToolCatalog(t *testing.T) {
	fake := &fakeCaller{tools: &mcp.ListToolsResult{Tools: []*mcp.Tool{
		{Name: "list_workloads", Description: "List workloads"},
		{Name: "get_status", Description: "Get status"},
		{Name: "run_workload", Description: "Run workload"},
		{Name: "read_secret", Description: "Read secret"},
	}}}
	client, err := newClient(fake)
	if err != nil {
		t.Fatal(err)
	}
	tools, err := client.DiscoverTools(context.Background())
	if err != nil || len(tools) != 4 || tools[3].Name != "read_secret" {
		t.Fatalf("tools=%+v err=%v", tools, err)
	}
}

func TestClientRejectsUnknownAndMalformedStructuredResponses(t *testing.T) {
	for _, response := range []any{
		map[string]any{"workload": "alpha", "mode": "offline", "state": "ready", "unexpected": true},
		map[string]any{"workload": "other", "mode": "offline", "state": "ready"},
		map[string]any{"workload": "alpha", "mode": "offline", "state": "invented"},
	} {
		t.Run("protocol", func(t *testing.T) {
			client, err := newClient(&fakeCaller{result: &mcp.CallToolResult{StructuredContent: response}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.GetStatus(context.Background(), "alpha")
			if category(err) != CategoryProtocol {
				t.Fatalf("category=%q error=%v", category(err), err)
			}
		})
	}
}

func TestListWorkloadsRejectsUnsortedOrDuplicateResponse(t *testing.T) {
	for _, workloads := range [][]string{{"beta", "alpha"}, {"alpha", "alpha"}} {
		client, err := newClient(&fakeCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{"workloads": workloads}}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.ListWorkloads(context.Background())
		if category(err) != CategoryProtocol {
			t.Fatalf("workloads=%v category=%q error=%v", workloads, category(err), err)
		}
	}
}

func TestClientPropagatesContextAndDoesNotExposeProtocolError(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "sentinel")
	fake := &fakeCaller{callErr: errors.New("SYNTHETIC_PRIVATE_PROTOCOL_MESSAGE")}
	client, err := newClient(fake)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetStatus(ctx, "alpha")
	if category(err) != CategoryProtocol || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_PROTOCOL_MESSAGE") {
		t.Fatalf("safe error=%v", err)
	}
	if fake.callCtx.Value(contextKey{}) != "sentinel" || fake.callParams.Name != "get_status" {
		t.Fatalf("context or tool not propagated: %#v %#v", fake.callCtx, fake.callParams)
	}
}

func TestToolErrorIsDeniedWithoutRetainingToolText(t *testing.T) {
	fake := &fakeCaller{result: &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: "SYNTHETIC_PRIVATE_TOOL_ERROR"}},
	}}
	client, err := newClient(fake)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetStatus(context.Background(), "alpha")
	if category(err) != CategoryDenied || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_TOOL_ERROR") {
		t.Fatalf("safe error=%v", err)
	}
}

func TestDisclosureRequiresExplicitClientAndRedactsSecret(t *testing.T) {
	marker := "SYNTHETIC_SECRET_MUST_NOT_ESCAPE"
	fake := &fakeCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{
		"alias": "TOKEN", "value": marker, "version": "v1",
	}}}
	client, err := newClient(fake, WithObserver(panicObserver{}))
	if err != nil {
		t.Fatal(err)
	}
	secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
	if err != nil || secret.Value() != marker || fake.callParams.Name != "read_secret" {
		t.Fatalf("secret=%v error=%v params=%+v", secret, err, fake.callParams)
	}
	encoded, err := json.Marshal(secret)
	if err != nil || strings.Contains(string(encoded), marker) || strings.Contains(secret.String(), marker) || strings.Contains(secret.GoString(), marker) {
		t.Fatalf("secret redaction failed: json=%s error=%v", encoded, err)
	}
}

func TestSecretMissingOrLegacyZeroExpiryNormalizesToZero(t *testing.T) {
	for _, response := range []map[string]any{
		{"alias": "TOKEN", "value": "synthetic"},
		{"alias": "TOKEN", "value": "synthetic", "expires_at": "0001-01-01T00:00:00Z"},
	} {
		client, err := newClient(&fakeCaller{result: &mcp.CallToolResult{StructuredContent: response}})
		if err != nil {
			t.Fatal(err)
		}
		secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
		if err != nil || !secret.ExpiresAt.IsZero() {
			t.Fatalf("secret=%v err=%v", secret, err)
		}
	}
}

func TestSecretProviderVersionIsOpaqueAndPortable(t *testing.T) {
	for _, version := range []string{
		"projects/synthetic/secrets/fixture/versions/7",
		"2026-11-04T12:00:00Z", // Bitwarden RFC3339 revision value.
		"mtime:2026-11-04T12:00:00Z",
	} {
		t.Run(version, func(t *testing.T) {
			client, err := newClient(&fakeCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{
				"alias": "TOKEN", "value": "synthetic", "version": version,
			}}})
			if err != nil {
				t.Fatal(err)
			}
			secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
			if err != nil || secret.Version != version {
				t.Fatalf("version=%q secret=%+v error=%v", version, secret, err)
			}
		})
	}
}

func TestSecretMetadataAndValueUTF8ByteBoundaries(t *testing.T) {
	providerVersion := strings.Repeat("é", maxProviderVersionBytes/len("é"))
	secretValue := strings.Repeat("é", maxSecretValueBytes/len("é"))
	assertSecretMetadataByteBoundaries(t, providerVersion, secretValue)
	assertProviderVersionControlHandling(t)
	secret := readSecretBoundaryFixture(t, providerVersion, secretValue)
	if len(secret.Value()) != maxSecretValueBytes || secret.Version != providerVersion {
		t.Fatalf("bounded secret was not preserved: value bytes=%d version bytes=%d", len(secret.Value()), len(secret.Version))
	}
}

func assertSecretMetadataByteBoundaries(t *testing.T, providerVersion, secretValue string) {
	t.Helper()
	if !validProviderVersion(providerVersion) || validProviderVersion(providerVersion+"é") {
		t.Fatal("provider-version byte boundary not enforced")
	}
	if !validSecretValue(secretValue) || validSecretValue(secretValue+"é") || validSecretValue(string([]byte{0xff})) {
		t.Fatal("secret-value UTF-8 byte boundary not enforced")
	}
}

func assertProviderVersionControlHandling(t *testing.T) {
	t.Helper()
	if !validProviderVersion("C1\u0085-permitted") {
		t.Fatal("provider version rejected a non-C0 Unicode control")
	}
	for _, version := range []string{"line\nbreak", "del\x7f", string([]byte{0xff})} {
		if validProviderVersion(version) {
			t.Fatalf("unsafe provider version accepted: %q", version)
		}
	}
}

func readSecretBoundaryFixture(t *testing.T, providerVersion, secretValue string) Secret {
	t.Helper()
	client, err := newClient(&fakeCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{
		"alias": "TOKEN", "value": secretValue, "version": providerVersion,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

func TestSecretExpiryCategoriesAreSafeAndStable(t *testing.T) {
	for name, response := range map[string]map[string]any{
		"expired":         {"alias": "TOKEN", "value": "synthetic", "expires_at": "2000-01-01T00:00:00Z"},
		"malformed":       {"alias": "TOKEN", "value": "synthetic", "expires_at": "not-a-timestamp"},
		"space_separated": {"alias": "TOKEN", "value": "synthetic", "expires_at": "2099-01-02 03:04:05Z"},
	} {
		t.Run(name, func(t *testing.T) {
			client, err := newClient(&fakeCaller{result: &mcp.CallToolResult{StructuredContent: response}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
			want := CategoryExpired
			if name != "expired" {
				want = CategoryProtocol
			}
			if category(err) != want {
				t.Fatalf("category=%q error=%v", category(err), err)
			}
		})
	}
}

func TestSecretExpiryAcceptsOnlyRFC3339LexicalTimestamps(t *testing.T) {
	for _, expiry := range []string{
		"2099-01-02T03:04:05Z",
		"2099-01-02T03:04:05.123456+10:30",
	} {
		t.Run(expiry, func(t *testing.T) {
			client, err := newClient(&fakeCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{
				"alias": "TOKEN", "value": "synthetic", "expires_at": expiry,
			}}})
			if err != nil {
				t.Fatal(err)
			}
			secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
			if err != nil || secret.ExpiresAt.IsZero() {
				t.Fatalf("expiry=%q secret=%+v error=%v", expiry, secret, err)
			}
		})
	}
}

func TestCanceledContextDoesNotCallTool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fake := &fakeCaller{}
	client, err := newClient(fake)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.RunWorkload(ctx, "alpha")
	if category(err) != CategoryCancelled || fake.callParams != nil {
		t.Fatalf("category=%q called=%v", category(err), fake.callParams != nil)
	}
}

func TestLateToolResponseAfterCancellationIsNotDelivered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fake := &fakeCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{
		"workload": "alpha", "mode": "offline", "state": "ready",
	}}, onCall: cancel}
	client, err := newClient(fake)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetStatus(ctx, "alpha")
	if category(err) != CategoryCancelled {
		t.Fatalf("category=%q error=%v", category(err), err)
	}
}

func category(err error) Category {
	var safe *Error
	if errors.As(err, &safe) {
		return safe.Category
	}
	return ""
}
