package cq

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// All callers here are in-process fixtures: no official session, transport,
// network, provider, credential or enrolled workload is opened or executed.
func TestClientBoundaryContractStatusRejectsInvalidHandleBeforeDispatch(t *testing.T) {
	calls := 0
	caller := &fakeCaller{onCall: func() { calls++ }}
	observer := &clientBoundaryObserver{}
	status, err := clientBoundaryFixture(t, caller, observer).GetStatus(context.Background(), "SYNTHETIC/INVALID")
	assertSDKResponseRefusal(t, err, CategoryInvalid)
	if status != (Status{}) || calls != 0 || caller.callParams != nil {
		t.Fatal("invalid status handle dispatched or returned a partial DTO")
	}
	assertClientBoundaryEvent(t, observer, "get_status", "invalid")
}

func TestClientBoundaryContractStatusPreservesAcceptedDTO(t *testing.T) {
	want := Status{Workload: "alpha", Mode: "offline", State: "ready"}
	calls := 0
	caller := &fakeCaller{
		result: &mcp.CallToolResult{StructuredContent: want},
		onCall: func() { calls++ },
	}
	observer := &clientBoundaryObserver{}
	status, err := clientBoundaryFixture(t, caller, observer).GetStatus(context.Background(), "alpha")
	if err != nil || status != want || calls != 1 {
		t.Fatal("accepted status was altered, refused or dispatched more than once")
	}
	assertClientBoundaryRequest(t, caller, "get_status", map[string]any{"workload": "alpha"})
	assertClientBoundaryEvent(t, observer, "get_status", "success")
}

func TestClientBoundaryContractListNetworkFailureWithholdsResultsAndDiagnostics(t *testing.T) {
	calls := 0
	caller := &fakeCaller{
		callErr: &net.OpError{Op: "read", Net: "synthetic", Err: errors.New("SYNTHETIC_PRIVATE_NETWORK_DIAGNOSTIC")},
		onCall:  func() { calls++ },
	}
	observer := &clientBoundaryObserver{}
	workloads, err := clientBoundaryFixture(t, caller, observer).ListWorkloads(context.Background())
	assertSDKResponseRefusal(t, err, CategoryUnavailable)
	if workloads != nil || calls != 1 {
		t.Fatal("failed workload listing returned data or retried")
	}
	assertClientBoundaryRequest(t, caller, "list_workloads", map[string]any{})
	assertClientBoundaryEvent(t, observer, "list_workloads", "unavailable")
}

func TestClientBoundaryContractDisclosureRequiresAnOwnedClient(t *testing.T) {
	for _, disclosure := range []*DisclosureClient{nil, {}} {
		secret, err := disclosure.ReadSecret(context.Background(), "alpha", "TOKEN")
		assertSDKResponseRefusal(t, err, CategoryInvalid)
		if secret != (Secret{}) {
			t.Fatal("missing disclosure client returned a partial secret")
		}
	}
}

func TestClientBoundaryContractDisclosureRejectsInvalidHandlesBeforeDispatch(t *testing.T) {
	for _, test := range []struct {
		name     string
		workload string
		key      string
	}{
		{"invalid workload", "SYNTHETIC/INVALID", "TOKEN"},
		{"invalid key", "alpha", "SYNTHETIC/INVALID"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			caller := &fakeCaller{onCall: func() { calls++ }}
			observer := &clientBoundaryObserver{}
			client := clientBoundaryFixture(t, caller, observer)
			secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), test.workload, test.key)
			assertSDKResponseRefusal(t, err, CategoryInvalid)
			if secret != (Secret{}) || calls != 0 || caller.callParams != nil {
				t.Fatal("invalid disclosure handle dispatched or returned a partial secret")
			}
			assertClientBoundaryEvent(t, observer, "read_secret", "invalid")
		})
	}
}

func TestClientBoundaryContractDeniedDisclosureWithholdsStructuredSecret(t *testing.T) {
	calls := 0
	caller := &fakeCaller{
		result: &mcp.CallToolResult{
			IsError: true,
			StructuredContent: map[string]any{
				"alias": "TOKEN", "value": "SYNTHETIC_PRIVATE_DENIED_VALUE", "version": "v1",
			},
		},
		onCall: func() { calls++ },
	}
	observer := &clientBoundaryObserver{}
	client := clientBoundaryFixture(t, caller, observer)
	secret, err := client.Disclosure(AllowPlaintext{}).ReadSecret(context.Background(), "alpha", "TOKEN")
	assertSDKResponseRefusal(t, err, CategoryDenied)
	if secret != (Secret{}) || secret.Value() != "" || calls != 1 {
		t.Fatal("denied disclosure returned secret data or retried")
	}
	assertClientBoundaryRequest(t, caller, "read_secret", map[string]any{"workload": "alpha", "key": "TOKEN"})
	assertClientBoundaryEvent(t, observer, "read_secret", "denied")
}

func TestClientBoundaryContractSecretJSONPreservesExpiryButRedactsValue(t *testing.T) {
	secret := Secret{
		Alias:     "TOKEN",
		Version:   "v1",
		ExpiresAt: time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC),
		value:     "SYNTHETIC_PRIVATE_SERIALIZED_VALUE",
	}
	before := secret
	encoded, err := json.Marshal(secret)
	if err != nil {
		t.Fatal("bounded secret metadata could not be serialized")
	}
	var got map[string]string
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal("redacted secret metadata was not a JSON string-field object")
	}
	want := map[string]string{
		"alias": "TOKEN", "version": "v1", "expires_at": "2030-01-02T03:04:05Z", "value": "[redacted]",
	}
	if !reflect.DeepEqual(got, want) || secret != before {
		t.Fatal("JSON omitted expiry, disclosed a value, added fields or mutated the secret")
	}
}

type clientBoundaryObserver struct{ events []Event }

func (o *clientBoundaryObserver) Observe(event Event) { o.events = append(o.events, event) }

func clientBoundaryFixture(t *testing.T, caller *fakeCaller, observer *clientBoundaryObserver) *Client {
	t.Helper()
	client, err := newClient(caller, WithObserver(observer))
	if err != nil {
		t.Fatal("could not create the synthetic client boundary fixture")
	}
	return client
}

func assertClientBoundaryRequest(t *testing.T, caller *fakeCaller, name string, arguments map[string]any) {
	t.Helper()
	if caller.callParams == nil {
		t.Fatal("expected in-process request was not dispatched")
	}
	if caller.callParams.Name != name || !reflect.DeepEqual(caller.callParams.Arguments, arguments) {
		t.Fatal("request changed the exact tool or argument set")
	}
}

func assertClientBoundaryEvent(t *testing.T, observer *clientBoundaryObserver, operation, outcome string) {
	t.Helper()
	if len(observer.events) != 1 {
		t.Fatal("operation did not report exactly one bounded metadata event")
	}
	event := observer.events[0]
	if event.Operation != operation || event.Outcome != outcome || event.Duration < 0 {
		t.Fatal("event changed operation, safe outcome or nonnegative duration")
	}
}
