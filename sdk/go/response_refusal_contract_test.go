package cq

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The caller below is entirely in-process. These contracts neither connect an
// official session nor exercise local CLI, provider, credential or workload I/O.
func TestSDKResponseRefusalConstructorsRequireExplicitCaller(t *testing.T) {
	for _, construct := range []func() (*Client, error){
		func() (*Client, error) { return New(nil) },
		func() (*Client, error) { return newClient(nil) },
	} {
		client, err := construct()
		assertSDKResponseRefusal(t, err, CategoryInvalid)
		if client != nil {
			t.Fatal("missing caller created a usable client")
		}
	}
}

func TestSDKResponseRefusalCatalogNeverReturnsPartialTools(t *testing.T) {
	valid := &mcp.Tool{Name: "get_status", Description: "Bounded status"}
	cursor := &mcp.ListToolsResult{}
	cursor.NextCursor = "SYNTHETIC_PRIVATE_CURSOR"
	for _, test := range []struct {
		name   string
		result *mcp.ListToolsResult
	}{
		{"nil response", nil},
		{"continuation cursor", cursor},
		{"too many tools", &mcp.ListToolsResult{Tools: []*mcp.Tool{valid, valid, valid, valid, valid}}},
		{"nil after valid tool", &mcp.ListToolsResult{Tools: []*mcp.Tool{valid, nil}}},
		{"unknown after valid tool", &mcp.ListToolsResult{Tools: []*mcp.Tool{valid, {Name: "SYNTHETIC_UNKNOWN_TOOL"}}}},
		{"duplicate after valid tool", &mcp.ListToolsResult{Tools: []*mcp.Tool{valid, valid}}},
		{"unbounded description", &mcp.ListToolsResult{Tools: []*mcp.Tool{{Name: "get_status", Description: strings.Repeat("x", maxTextLength+1)}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := &sdkResponseRefusalCaller{tools: test.result}
			tools, err := sdkResponseRefusalClient(t, caller).DiscoverTools(context.Background())
			assertSDKResponseRefusal(t, err, CategoryProtocol)
			if tools != nil || caller.lists != 1 || caller.calls != 0 {
				t.Fatal("malformed catalogue returned partial tools or changed dispatch count")
			}
		})
	}
}

func TestSDKResponseRefusalDiscoveryHonorsContextAndSanitizesErrors(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer stop()
	for _, test := range []struct {
		name string
		ctx  context.Context
		want Category
	}{
		{"nil", nil, CategoryInvalid},
		{"canceled", canceled, CategoryCancelled},
		{"expired", expired, CategoryUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := &sdkResponseRefusalCaller{}
			tools, err := sdkResponseRefusalClient(t, caller).DiscoverTools(test.ctx)
			assertSDKResponseRefusal(t, err, test.want)
			if tools != nil || caller.lists != 0 {
				t.Fatal("invalid context reached catalogue dispatch")
			}
		})
	}
	ctx, lateCancel := context.WithCancel(context.Background())
	defer lateCancel()
	caller := &sdkResponseRefusalCaller{tools: &mcp.ListToolsResult{}, onList: lateCancel}
	tools, err := sdkResponseRefusalClient(t, caller).DiscoverTools(ctx)
	assertSDKResponseRefusal(t, err, CategoryCancelled)
	if tools != nil || caller.lists != 1 {
		t.Fatal("late canceled catalogue response was delivered or retried")
	}
	caller = &sdkResponseRefusalCaller{listErr: errors.New("SYNTHETIC_PRIVATE_LIST_FAILURE")}
	tools, err = sdkResponseRefusalClient(t, caller).DiscoverTools(context.Background())
	assertSDKResponseRefusal(t, err, CategoryProtocol)
	if tools != nil || caller.lists != 1 {
		t.Fatal("catalogue failure was delivered or retried")
	}
}

func TestSDKResponseRefusalStructuredStatusWithholdsData(t *testing.T) {
	for _, test := range []struct {
		name   string
		result *mcp.CallToolResult
		want   Category
	}{
		{"nil response", nil, CategoryDenied},
		{"missing structured content", &mcp.CallToolResult{}, CategoryDenied},
		{"tool denial with structured prefix", &mcp.CallToolResult{IsError: true, StructuredContent: map[string]any{"workload": "alpha"}}, CategoryDenied},
		{"unencodable structured value", &mcp.CallToolResult{StructuredContent: make(chan int)}, CategoryProtocol},
		{"oversized structured value", &mcp.CallToolResult{StructuredContent: strings.Repeat("x", maxStructuredResponseBytes+1)}, CategoryProtocol},
		{"wrong field type", &mcp.CallToolResult{StructuredContent: map[string]any{"workload": "alpha", "mode": "offline", "state": 7}}, CategoryProtocol},
		{"unknown mode", &mcp.CallToolResult{StructuredContent: map[string]any{"workload": "alpha", "mode": "invented", "state": "ready"}}, CategoryProtocol},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := &sdkResponseRefusalCaller{result: test.result}
			status, err := sdkResponseRefusalClient(t, caller).GetStatus(context.Background(), "alpha")
			assertSDKResponseRefusal(t, err, test.want)
			if status != (Status{}) || caller.calls != 1 {
				t.Fatal("refused status returned partial data or retried")
			}
		})
	}
}

func TestSDKResponseRefusalWorkloadListIsBoundedAndExact(t *testing.T) {
	for _, workloads := range [][]string{
		make([]string, maxWorkloads+1),
		{"alpha", "SYNTHETIC/INVALID"},
	} {
		caller := &sdkResponseRefusalCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{"workloads": workloads}}}
		result, err := sdkResponseRefusalClient(t, caller).ListWorkloads(context.Background())
		assertSDKResponseRefusal(t, err, CategoryProtocol)
		if result != nil || caller.calls != 1 {
			t.Fatal("invalid workload list returned a prefix or retried")
		}
	}
	want := []string{"alpha", "beta"}
	caller := &sdkResponseRefusalCaller{result: &mcp.CallToolResult{StructuredContent: map[string]any{"workloads": want}}}
	result, err := sdkResponseRefusalClient(t, caller).ListWorkloads(context.Background())
	if err != nil || !reflect.DeepEqual(result, want) || caller.calls != 1 {
		t.Fatal("bounded sorted workload list was not preserved")
	}
}

func TestSDKResponseRefusalRunDTOIsBoundedWithoutExecutingWorkload(t *testing.T) {
	for _, test := range []struct {
		name string
		wire RunResult
	}{
		{"invalid identifier", RunResult{ID: "SYNTHETIC/INVALID", Status: "completed"}},
		{"unknown status", RunResult{ID: "run-1", Status: "invented"}},
		{"negative exit", RunResult{ID: "run-1", Status: "failed", ExitCode: -1}},
		{"oversized exit", RunResult{ID: "run-1", Status: "failed", ExitCode: 256}},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := &sdkResponseRefusalCaller{result: &mcp.CallToolResult{StructuredContent: test.wire}}
			result, err := sdkResponseRefusalClient(t, caller).RunWorkload(context.Background(), "alpha")
			assertSDKResponseRefusal(t, err, CategoryProtocol)
			if result != (RunResult{}) || caller.calls != 1 {
				t.Fatal("refused run DTO returned data or retried")
			}
		})
	}
	assertSDKResponseRefusalExactRunRequest(t)
	assertSDKResponseRefusalInvalidWorkloadDoesNotDispatch(t)
}

func assertSDKResponseRefusalExactRunRequest(t *testing.T) {
	t.Helper()
	want := RunResult{ID: "run-1", Status: "completed", ExitCode: 0}
	caller := &sdkResponseRefusalCaller{result: &mcp.CallToolResult{StructuredContent: want}}
	result, err := sdkResponseRefusalClient(t, caller).RunWorkload(context.Background(), "alpha")
	if err != nil || result != want || caller.calls != 1 || caller.params.Name != "run_workload" || !reflect.DeepEqual(caller.params.Arguments, map[string]any{"workload": "alpha"}) {
		t.Fatal("in-process run request lost its exact tool or arguments")
	}
}

func assertSDKResponseRefusalInvalidWorkloadDoesNotDispatch(t *testing.T) {
	t.Helper()
	caller := &sdkResponseRefusalCaller{}
	result, err := sdkResponseRefusalClient(t, caller).RunWorkload(context.Background(), "SYNTHETIC/INVALID")
	assertSDKResponseRefusal(t, err, CategoryInvalid)
	if result != (RunResult{}) || caller.calls != 0 {
		t.Fatal("invalid workload dispatched a call")
	}
}

func TestSDKResponseRefusalSecretWireFailuresDoNotDisclose(t *testing.T) {
	for _, test := range []struct {
		name string
		wire map[string]any
	}{
		{"alias mismatch", map[string]any{"alias": "OTHER", "value": "SYNTHETIC_PRIVATE_VALUE"}},
		{"oversized value", map[string]any{"alias": "TOKEN", "value": strings.Repeat("x", maxSecretValueBytes+1)}},
		{"oversized version", map[string]any{"alias": "TOKEN", "value": "SYNTHETIC_PRIVATE_VALUE", "version": strings.Repeat("x", maxProviderVersionBytes+1)}},
		{"numeric expiry", map[string]any{"alias": "TOKEN", "value": "SYNTHETIC_PRIVATE_VALUE", "expires_at": 17}},
		{"invalid calendar date", map[string]any{"alias": "TOKEN", "value": "SYNTHETIC_PRIVATE_VALUE", "expires_at": "2099-02-30T00:00:00Z"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := &sdkResponseRefusalCaller{result: &mcp.CallToolResult{StructuredContent: test.wire}}
			secret, err := sdkResponseRefusalClient(t, caller).Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
			assertSDKResponseRefusal(t, err, CategoryProtocol)
			if secret != (Secret{}) || caller.calls != 1 {
				t.Fatal("invalid secret wire value returned data or retried")
			}
		})
	}
}

type sdkResponseRefusalCaller struct {
	tools   *mcp.ListToolsResult
	listErr error
	result  *mcp.CallToolResult
	lists   int
	calls   int
	params  *mcp.CallToolParams
	onList  func()
}

func (c *sdkResponseRefusalCaller) ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	c.lists++
	if c.onList != nil {
		c.onList()
	}
	return c.tools, c.listErr
}

func (c *sdkResponseRefusalCaller) CallTool(_ context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	c.calls++
	c.params = params
	return c.result, nil
}

func sdkResponseRefusalClient(t *testing.T, caller *sdkResponseRefusalCaller) *Client {
	t.Helper()
	client, err := newClient(caller, nil)
	if err != nil {
		t.Fatal("could not create the synthetic in-process client")
	}
	return client
}

func assertSDKResponseRefusal(t *testing.T, err error, want Category) {
	t.Helper()
	var safe *Error
	if !errors.As(err, &safe) || safe.Category != want || err.Error() != "cq: "+string(want) || errors.Unwrap(err) != nil {
		t.Fatalf("wrong or unsafe SDK error, want category %s", want)
	}
}
