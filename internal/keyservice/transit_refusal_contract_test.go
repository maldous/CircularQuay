package keyservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/transport"
)

func TestTransitRefusalContractCiphertextVersionBeforePrivateAccess(t *testing.T) {
	for _, provider := range []string{transitProviderOpenBao, transitProviderVault} {
		for _, version := range []string{"2", "4294967295"} {
			t.Run(provider+"/"+version, func(t *testing.T) {
				counts := &transitAdmissionCounts{}
				adapter := newTransitAdmissionAdapter(t, counts, nil)
				request := transitAdmissionRequest()
				request.Association.Provider = provider
				request.Ciphertext = []byte("vault:v" + version + ":c3ludGhldGlj")
				result, err := adapter.Unwrap(context.Background(), request)
				requireTransitAdmissionFault(t, result, err, fault.Trust, "transit_ciphertext_version")
				requireNoTransitAdmissionAccess(t, counts)
				if string(request.Ciphertext) != "vault:v"+version+":c3ludGhldGlj" {
					t.Fatal("refused unwrap mutated its caller's ciphertext")
				}
			})
		}
	}
}

func TestTransitRefusalContractAdmissionPrecedence(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		name      string
		ctx       context.Context
		modify    func(*UnwrapRequest)
		kind      fault.Kind
		operation string
	}{
		{"empty-before-nil-context", nil, func(r *UnwrapRequest) { r.Ciphertext = nil }, fault.Invalid, "transit_unwrap_request"},
		{"oversize-before-cancel", cancelled, func(r *UnwrapRequest) { r.Ciphertext = make([]byte, maximumTransitMaterial+1) }, fault.Invalid, "transit_unwrap_request"},
		{"syntax-before-nil-context", nil, func(r *UnwrapRequest) { r.Ciphertext = []byte("vault:v02:c3ludGhldGlj") }, fault.Invalid, "transit_ciphertext"},
		{"syntax-before-cancel", cancelled, func(r *UnwrapRequest) { r.Ciphertext = []byte("not-ciphertext") }, fault.Invalid, "transit_ciphertext"},
		{"nil-context-before-version", nil, nil, fault.Denied, "transit_cancelled"},
		{"cancel-before-version", cancelled, nil, fault.Denied, "transit_cancelled"},
		{"provider-before-version", context.Background(), func(r *UnwrapRequest) { r.Association.Provider = "unsupported-synthetic" }, fault.Unsupported, "transit_provider"},
		{"association-before-version", context.Background(), func(r *UnwrapRequest) { r.Association.Algorithm = "unsupported-synthetic" }, fault.Unsupported, "transit_association"},
		{"invalid-enrolled-version", context.Background(), func(r *UnwrapRequest) { r.Association.KeyVersion = "0" }, fault.Unsupported, "transit_association"},
		{"purpose-before-version", context.Background(), func(r *UnwrapRequest) { r.Context.Purpose = "other-purpose" }, fault.Invalid, "transit_context"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			counts := &transitAdmissionCounts{}
			adapter := newTransitAdmissionAdapter(t, counts, nil)
			request := transitAdmissionRequest()
			request.Ciphertext = []byte("vault:v2:c3ludGhldGlj")
			if tc.modify != nil {
				tc.modify(&request)
			}
			result, err := adapter.Unwrap(tc.ctx, request)
			requireTransitAdmissionFault(t, result, err, tc.kind, tc.operation)
			requireNoTransitAdmissionAccess(t, counts)
		})
	}
}

func TestTransitRefusalContractMatchingVersionStillUnwraps(t *testing.T) {
	counts := &transitAdmissionCounts{}
	request := transitAdmissionRequest()
	adapter := newTransitAdmissionAdapter(t, counts, func(r *http.Request) (*http.Response, error) {
		return transitAdmissionUnwrapReply(t, request, r)
	})
	result, err := adapter.Unwrap(context.Background(), request)
	assertTransitAdmissionUnwrapResult(t, result, request, err)
	assertTransitAdmissionSuccessCounts(t, counts, "matching version did not use exactly the selected synthetic collaborators")
}

func TestTransitRefusalContractResourceBeforeCredential(t *testing.T) {
	cases := []struct {
		name      string
		keyID     string
		mount     string
		endpoint  string
		minimum   string
		kind      fault.Kind
		operation string
	}{
		{"parent-key", "..", "transit", "https://transit.example.invalid", "0", fault.Invalid, "transit_resource"},
		{"query-key", "workload?query", "transit", "https://transit.example.invalid", "0", fault.Invalid, "transit_resource"},
		{"backslash-key", "workload\\key", "transit", "https://transit.example.invalid", "0", fault.Invalid, "transit_resource"},
		{"dot-mount", "workload-key", ".", "https://transit.example.invalid", "0", fault.Invalid, "transit_resource"},
		{"trailing-mount", "workload-key", "transit/", "https://transit.example.invalid", "0", fault.Invalid, "transit_resource"},
		{"endpoint-before-resource", "..", "transit", "http://transit.example.invalid", "0", fault.Trust, "transit_endpoint"},
		{"configuration-before-resource", "..", "transit", "https://transit.example.invalid", "01", fault.Invalid, "transit_configuration"},
	}
	for _, tc := range cases {
		for _, operation := range []string{"wrap", "unwrap"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				counts := &transitAdmissionCounts{}
				configuration := transitTestConfiguration(tc.endpoint)
				configuration.Mount = tc.mount
				configuration.MinEncryptionVersion = tc.minimum
				adapter := newTransitAdmissionAdapterConfigured(t, counts, configuration, nil)
				association := transitTestAssociation()
				association.KeyID = tc.keyID
				material, receipt, err := transitAdmissionResourceOperation(adapter, association, operation)
				requireTransitAdmissionError(t, err, tc.kind, tc.operation)
				if len(material) != 0 || receipt != (KMSReceipt{}) {
					t.Fatal("invalid resource returned material or a success receipt")
				}
				if counts.configuration != 1 || counts.credential != 0 || counts.http != 0 {
					t.Fatal("invalid resource or configuration reached credentials or provider transport")
				}
			})
		}
	}
}

func TestTransitRefusalContractValidResourceStillWraps(t *testing.T) {
	counts := &transitAdmissionCounts{}
	adapter := newTransitAdmissionAdapter(t, counts, func(r *http.Request) (*http.Response, error) {
		return transitAdmissionWrapReply(t, r)
	})
	association := transitTestAssociation()
	requestContext := transitTestContext()
	result, err := adapter.Wrap(context.Background(), WrapRequest{
		Association: association, Context: requestContext, Plaintext: []byte("synthetic-key"),
	})
	assertTransitAdmissionWrapResult(t, result, association, requestContext, err)
	assertTransitAdmissionSuccessCounts(t, counts, "valid resource did not use exactly the selected synthetic collaborators")
}

func transitAdmissionUnwrapReply(t *testing.T, request UnwrapRequest, r *http.Request) (*http.Response, error) {
	t.Helper()
	assertTransitAdmissionRequestBinding(t, r, "matching version reached an unbound synthetic request")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v1/transit/keys/workload-key":
		return transitAdmissionKeyReply(t), nil
	case r.Method == http.MethodPost && r.URL.Path == "/v1/transit/decrypt/workload-key":
		assertTransitAdmissionDecryptBody(t, r, request.Ciphertext)
		return transitAdmissionResponse(`{"data":{"plaintext":"c3ludGhldGljLWtleQ=="}}`), nil
	default:
		t.Fatal("matching version used an unexpected synthetic method or path")
		return nil, errors.New("unexpected synthetic Transit request")
	}
}

func transitAdmissionWrapReply(t *testing.T, r *http.Request) (*http.Response, error) {
	t.Helper()
	assertTransitAdmissionRequestBinding(t, r, "valid resource reached an unbound synthetic request")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v1/transit/keys/workload-key":
		return transitAdmissionKeyReply(t), nil
	case r.Method == http.MethodPost && r.URL.Path == "/v1/transit/encrypt/workload-key":
		assertTransitAdmissionEncryptBody(t, r)
		return transitAdmissionResponse(`{"data":{"ciphertext":"vault:v1:c3ludGhldGlj","key_version":1}}`), nil
	default:
		t.Fatal("valid resource used an unexpected synthetic method or path")
		return nil, errors.New("unexpected synthetic Transit request")
	}
}

func assertTransitAdmissionRequestBinding(t *testing.T, r *http.Request, message string) {
	t.Helper()
	if r.URL.Scheme != "https" || r.URL.Host != "transit.example.invalid" ||
		r.Header.Get("X-Vault-Token") != "synthetic-token" ||
		r.Header.Get("X-Vault-Namespace") != "synthetic-namespace" {
		t.Fatal(message)
	}
}

func transitAdmissionKeyReply(t *testing.T) *http.Response {
	t.Helper()
	data, err := json.Marshal(transitKeyDocument(true, false, false, false))
	if err != nil {
		t.Fatal(err)
	}
	return transitAdmissionResponse(string(data))
}

func assertTransitAdmissionDecryptBody(t *testing.T, r *http.Request, ciphertext []byte) {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 2 ||
		body["ciphertext"] != string(ciphertext) || body["context"] == "" {
		t.Fatal("matching version changed the bounded synthetic decrypt request")
	}
}

func assertTransitAdmissionEncryptBody(t *testing.T, r *http.Request) {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 3 ||
		string(body["plaintext"]) != `"c3ludGhldGljLWtleQ=="` ||
		string(body["key_version"]) != "1" || len(body["context"]) <= 2 {
		t.Fatal("valid resource changed the bounded synthetic encrypt request")
	}
}

func assertTransitAdmissionUnwrapResult(t *testing.T, result UnwrapResult, request UnwrapRequest, err error) {
	t.Helper()
	wantReceipt := transitAdmissionExpectedReceipt(request.Association, request.Context)
	if err != nil || string(result.Plaintext) != "synthetic-key" || result.Receipt != wantReceipt {
		t.Fatal("matching enrolled version lost its plaintext or bound receipt")
	}
}

func assertTransitAdmissionWrapResult(t *testing.T, result WrapResult, association KMSAssociation, requestContext AssociationContext, err error) {
	t.Helper()
	wantReceipt := transitAdmissionExpectedReceipt(association, requestContext)
	if err != nil || string(result.Ciphertext) != "vault:v1:c3ludGhldGlj" || result.Receipt != wantReceipt {
		t.Fatal("valid resource lost its ciphertext or bound receipt")
	}
}

func transitAdmissionExpectedReceipt(association KMSAssociation, requestContext AssociationContext) KMSReceipt {
	return KMSReceipt{
		Provider:                 association.Provider,
		KeyID:                    association.KeyID,
		KeyVersion:               association.KeyVersion,
		Algorithm:                association.Algorithm,
		ConfigurationAttestation: association.ConfigurationAttestation,
		Context:                  requestContext,
	}
}

func assertTransitAdmissionSuccessCounts(t *testing.T, counts *transitAdmissionCounts, message string) {
	t.Helper()
	if counts.configuration != 1 || counts.credential != 1 || counts.http != 2 {
		t.Fatal(message)
	}
}

func transitAdmissionResourceOperation(adapter *Transit, association KMSAssociation, operation string) ([]byte, KMSReceipt, error) {
	if operation == "wrap" {
		result, err := adapter.Wrap(context.Background(), WrapRequest{
			Association: association, Context: transitTestContext(), Plaintext: []byte("synthetic-key"),
		})
		return result.Ciphertext, result.Receipt, err
	}
	request := transitAdmissionRequest()
	request.Association = association
	result, err := adapter.Unwrap(context.Background(), request)
	return result.Plaintext, result.Receipt, err
}

type transitAdmissionCounts struct{ configuration, credential, http int }

type transitAdmissionTransport func(*http.Request) (*http.Response, error)

func (f transitAdmissionTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTransitAdmissionAdapter(t *testing.T, counts *transitAdmissionCounts, respond transitAdmissionTransport) *Transit {
	t.Helper()
	return newTransitAdmissionAdapterConfigured(t, counts, transitTestConfiguration("https://transit.example.invalid"), respond)
}

func newTransitAdmissionAdapterConfigured(t *testing.T, counts *transitAdmissionCounts, configuration TransitConfiguration, respond transitAdmissionTransport) *Transit {
	t.Helper()
	adapter, err := NewTransit(
		transport.Client{HTTP: &http.Client{Transport: transitAdmissionTransport(func(r *http.Request) (*http.Response, error) {
			counts.http++
			if respond == nil {
				return nil, errors.New("unexpected synthetic Transit request")
			}
			return respond(r)
		})}},
		transitConfigurationFunc(func(context.Context, KMSAssociation) (TransitConfiguration, error) {
			counts.configuration++
			return configuration, nil
		}),
		transitTokenFunc(func(_ context.Context, reference string) (string, error) {
			counts.credential++
			if reference != "synthetic-credential-reference" {
				t.Fatal("credential reader received an unbound synthetic reference")
			}
			return "synthetic-token", nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func transitAdmissionRequest() UnwrapRequest {
	return UnwrapRequest{
		Association: transitTestAssociation(),
		Context:     transitTestContext(),
		Ciphertext:  []byte("vault:v1:c3ludGhldGlj"),
	}
}

func transitAdmissionResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func requireNoTransitAdmissionAccess(t *testing.T, counts *transitAdmissionCounts) {
	t.Helper()
	if *counts != (transitAdmissionCounts{}) {
		t.Fatal("denied admission reached private configuration, credentials or provider transport")
	}
}

func requireTransitAdmissionFault(t *testing.T, result UnwrapResult, err error, kind fault.Kind, operation string) {
	t.Helper()
	requireTransitAdmissionError(t, err, kind, operation)
	if len(result.Plaintext) != 0 || result.Receipt != (KMSReceipt{}) {
		t.Fatal("denied admission returned plaintext or a success receipt")
	}
}

func requireTransitAdmissionError(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var typed *fault.Error
	if !errors.As(err, &typed) || typed.Kind != kind || typed.Operation != operation ||
		err.Error() != string(kind)+": "+operation {
		t.Fatalf("unwrap refusal = %v, want %s: %s", err, kind, operation)
	}
}
