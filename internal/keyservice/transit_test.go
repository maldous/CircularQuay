package keyservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/transport"
)

func TestTransitWrapAndUnwrapVerifyNativeConfigurationAndStableContext(t *testing.T) {
	fixture := newTransitWrapUnwrapFixture(t)
	server := httptest.NewTLSServer(fixture)
	defer server.Close()

	association := transitTestAssociation()
	requestContext := transitTestContext()
	transit := transitTestAdapter(t, server, transitTestConfiguration(server.URL))
	wrapped, err := transit.Wrap(context.Background(), WrapRequest{Association: association, Context: requestContext, Plaintext: fixture.plaintext})
	assertTransitWrapResult(t, wrapped, association, requestContext, fixture.ciphertext, err)
	unwrapped, err := transit.Unwrap(context.Background(), UnwrapRequest{Association: association, Context: requestContext, Ciphertext: wrapped.Ciphertext})
	assertTransitUnwrapResult(t, unwrapped, association, requestContext, fixture.plaintext, err)
	fixture.assertStableContext(requestContext)
}

type transitWrapUnwrapFixture struct {
	t          *testing.T
	plaintext  []byte
	ciphertext string
	contexts   []string
}

func newTransitWrapUnwrapFixture(t *testing.T) *transitWrapUnwrapFixture {
	return &transitWrapUnwrapFixture{
		t:          t,
		plaintext:  []byte("synthetic-data-key"),
		ciphertext: "vault:v1:" + base64.StdEncoding.EncodeToString([]byte("synthetic-ciphertext")),
	}
}

func (fixture *transitWrapUnwrapFixture) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if !fixture.validHeaders(request) {
		fixture.t.Fatal("missing protected Transit request headers")
	}
	switch request.URL.Path {
	case "/v1/transit/keys/workload-key":
		writeTransitJSON(fixture.t, response, transitKeyDocument(true, false, false, false))
	case "/v1/transit/encrypt/workload-key":
		fixture.encrypt(response, request)
	case "/v1/transit/decrypt/workload-key":
		fixture.decrypt(response, request)
	default:
		fixture.t.Fatalf("unexpected Transit path %q", request.URL.Path)
	}
}

func (fixture *transitWrapUnwrapFixture) validHeaders(request *http.Request) bool {
	return request.Header.Get("X-Vault-Token") == "synthetic-token" &&
		request.Header.Get("X-Vault-Namespace") == "synthetic-namespace"
}

func (fixture *transitWrapUnwrapFixture) encrypt(response http.ResponseWriter, request *http.Request) {
	var body transitEncryptRequest
	decodeTransitJSON(fixture.t, request, &body)
	if body.Plaintext != base64.StdEncoding.EncodeToString(fixture.plaintext) || body.KeyVersion != 1 || body.Context == "" {
		fixture.t.Fatal("unexpected Transit encryption body")
	}
	fixture.contexts = append(fixture.contexts, body.Context)
	writeTransitJSON(fixture.t, response, map[string]any{"data": map[string]any{"ciphertext": fixture.ciphertext, "key_version": 1}})
}

func (fixture *transitWrapUnwrapFixture) decrypt(response http.ResponseWriter, request *http.Request) {
	var body transitDecryptRequest
	decodeTransitJSON(fixture.t, request, &body)
	if body.Ciphertext != fixture.ciphertext || body.Context == "" {
		fixture.t.Fatal("unexpected Transit decryption body")
	}
	fixture.contexts = append(fixture.contexts, body.Context)
	writeTransitJSON(fixture.t, response, map[string]any{"data": map[string]any{"plaintext": base64.StdEncoding.EncodeToString(fixture.plaintext)}})
}

func assertTransitWrapResult(t *testing.T, result WrapResult, association KMSAssociation, request AssociationContext, ciphertext string, err error) {
	t.Helper()
	if err != nil || string(result.Ciphertext) != ciphertext || result.Receipt != transitReceipt(association, request) {
		t.Fatalf("wrap result=%+v error=%v", result.Receipt, err)
	}
}

func assertTransitUnwrapResult(t *testing.T, result UnwrapResult, association KMSAssociation, request AssociationContext, plaintext []byte, err error) {
	t.Helper()
	if err != nil || string(result.Plaintext) != string(plaintext) || result.Receipt != transitReceipt(association, request) {
		t.Fatalf("unwrap receipt=%+v error=%v", result.Receipt, err)
	}
}

func (fixture *transitWrapUnwrapFixture) assertStableContext(request AssociationContext) {
	if len(fixture.contexts) != 2 || fixture.contexts[0] != fixture.contexts[1] {
		fixture.t.Fatalf("Transit context was not stable across wrap and unwrap")
	}
	decoded, err := base64.StdEncoding.DecodeString(fixture.contexts[0])
	if err != nil || string(decoded) != string(transitDerivedContext(request)) {
		fixture.t.Fatal("Transit request did not bind the service-built stable context")
	}
}

func TestTransitUsesExplicitEnrolledKeyVersionWithinNativeWindow(t *testing.T) {
	association := transitTestAssociation()
	ciphertext := "vault:v1:" + base64.StdEncoding.EncodeToString([]byte("synthetic-ciphertext"))
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/transit/keys/workload-key":
			writeTransitJSON(t, response, transitKeyDocumentVersions(2, map[string]int{"1": 1, "2": 2}, true, false, false, false))
		case "/v1/transit/encrypt/workload-key":
			var body transitEncryptRequest
			decodeTransitJSON(t, request, &body)
			if body.KeyVersion != 1 {
				t.Fatalf("requested version=%d", body.KeyVersion)
			}
			writeTransitJSON(t, response, map[string]any{"data": map[string]any{"ciphertext": ciphertext, "key_version": 1}})
		default:
			t.Fatalf("unexpected Transit path %q", request.URL.Path)
		}
	}))
	defer server.Close()
	transit := transitTestAdapter(t, server, transitTestConfiguration(server.URL))
	result, err := transit.Wrap(context.Background(), WrapRequest{Association: association, Context: transitTestContext(), Plaintext: []byte("synthetic")})
	if err != nil || string(result.Ciphertext) != ciphertext {
		t.Fatalf("pinned key version result=%q error=%v", result.Ciphertext, err)
	}
}

func TestTransitRejectsUnboundOrExportableNativeConfiguration(t *testing.T) {
	cases := []struct {
		name       string
		derived    bool
		convergent bool
		exportable bool
		backup     bool
	}{
		{name: "not_derived", derived: false},
		{name: "convergent", derived: true, convergent: true},
		{name: "exportable", derived: true, exportable: true},
		{name: "plaintext_backup", derived: true, backup: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			posts := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/v1/transit/keys/workload-key" {
					writeTransitJSON(t, response, transitKeyDocument(test.derived, test.convergent, test.exportable, test.backup))
					return
				}
				posts++
				response.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()
			transit := transitTestAdapter(t, server, transitTestConfiguration(server.URL))
			_, err := transit.Wrap(context.Background(), WrapRequest{Association: transitTestAssociation(), Context: transitTestContext(), Plaintext: []byte("synthetic")})
			if !fault.Is(err, fault.Trust) || posts != 0 {
				t.Fatalf("native configuration error=%v posts=%d", err, posts)
			}
		})
	}
}

func TestTransitRejectsWrongNativeVersionAndCancelledRequestWithoutProviderEffect(t *testing.T) {
	serverCalls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		serverCalls++
		writeTransitJSON(t, response, transitKeyDocument(true, false, false, false))
	}))
	defer server.Close()
	configuration := transitTestConfiguration(server.URL)
	transit := transitTestAdapter(t, server, configuration)
	association := transitTestAssociation()
	association.KeyVersion = "2"
	_, err := transit.Wrap(context.Background(), WrapRequest{Association: association, Context: transitTestContext(), Plaintext: []byte("synthetic")})
	if !fault.Is(err, fault.Trust) || serverCalls != 1 {
		t.Fatalf("wrong version error=%v calls=%d", err, serverCalls)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = transit.Wrap(cancelled, WrapRequest{Association: transitTestAssociation(), Context: transitTestContext(), Plaintext: []byte("synthetic")})
	if !fault.Is(err, fault.Denied) || serverCalls != 1 {
		t.Fatalf("cancelled error=%v calls=%d", err, serverCalls)
	}
}

func TestTransitResponseDTORejectsAmbiguousOrMalformedJSON(t *testing.T) {
	cases := []struct {
		name   string
		decode func([]byte) error
		body   []byte
	}{
		{name: "duplicate_derived", decode: decodeTransitKeyResponse, body: []byte(`{"data":{"derived":true,"derived":false}}`)},
		{name: "duplicate_exportable", decode: decodeTransitKeyResponse, body: []byte(`{"data":{"exportable":false,"exportable":true}}`)},
		{name: "duplicate_key_version", decode: decodeTransitEncryptResponse, body: []byte(`{"data":{"key_version":1,"key_version":2}}`)},
		{name: "duplicate_plaintext", decode: decodeTransitDecryptResponse, body: []byte(`{"data":{"plaintext":"YQ==","plaintext":"Yg=="}}`)},
		{name: "duplicate_data", decode: decodeTransitKeyResponse, body: []byte(`{"data":{},"data":{}}`)},
		{name: "invalid_utf8", decode: decodeTransitKeyResponse, body: append(append([]byte(`{"data":{"name":"`), 0xff), []byte(`"}}`)...)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(test.body); err == nil {
				t.Fatal("ambiguous or malformed native response accepted")
			}
		})
	}
}

func TestTransitKeyResponseRejectsNullOrMalformedEnrolledVersion(t *testing.T) {
	for _, body := range [][]byte{
		[]byte(`{"data":{"name":"workload-key","type":"aes256-gcm96","latest_version":1,"min_decryption_version":1,"min_encryption_version":0,"keys":{"1":null},"derived":true,"convergent_encryption":false,"exportable":false,"allow_plaintext_backup":false}}`),
		[]byte(`{"data":{"name":"workload-key","type":"aes256-gcm96","latest_version":1,"min_decryption_version":1,"min_encryption_version":0,"keys":{"not-a-version":1},"derived":true,"convergent_encryption":false,"exportable":false,"allow_plaintext_backup":false}}`),
	} {
		var response transitKeyResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("strict envelope decode error=%v", err)
		}
		if _, err := response.config(); err == nil {
			t.Fatal("invalid enrolled key version accepted")
		}
	}
}

func decodeTransitKeyResponse(data []byte) error {
	var response transitKeyResponse
	return json.Unmarshal(data, &response)
}

func decodeTransitEncryptResponse(data []byte) error {
	var response transitEncryptResponse
	return json.Unmarshal(data, &response)
}

func decodeTransitDecryptResponse(data []byte) error {
	var response transitDecryptResponse
	return json.Unmarshal(data, &response)
}

func TestTransitCancellationAfterPrivateCollaboratorPreventsProviderEffect(t *testing.T) {
	for _, test := range []struct {
		name          string
		configuration bool
	}{
		{name: "configuration", configuration: true},
		{name: "credential"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runTransitCancelledCollaboratorCase(t, test.name, test.configuration)
		})
	}
}

func runTransitCancelledCollaboratorCase(t *testing.T, name string, cancelAfterConfiguration bool) {
	t.Helper()
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls++
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	adapter := transitCancellationAdapter(t, server, transitTestConfiguration(server.URL), cancel, cancelAfterConfiguration)
	_, err := adapter.Wrap(ctx, WrapRequest{Association: transitTestAssociation(), Context: transitTestContext(), Plaintext: []byte("synthetic")})
	if !fault.Is(err, fault.Denied) || calls != 0 {
		t.Fatalf("cancelled %s error=%v requests=%d", name, err, calls)
	}
}

type cancellingTransitConfigurationResolver struct {
	configuration TransitConfiguration
	cancel        context.CancelFunc
	enabled       bool
}

func (resolver cancellingTransitConfigurationResolver) TransitConfiguration(context.Context, KMSAssociation) (TransitConfiguration, error) {
	if resolver.enabled {
		resolver.cancel()
	}
	return resolver.configuration, nil
}

type cancellingTransitCredentialReader struct {
	cancel  context.CancelFunc
	enabled bool
}

func (reader cancellingTransitCredentialReader) Token(context.Context, string) (string, error) {
	if reader.enabled {
		reader.cancel()
	}
	return "synthetic-token", nil
}

func transitCancellationAdapter(t *testing.T, server *httptest.Server, configuration TransitConfiguration, cancel context.CancelFunc, cancelAfterConfiguration bool) *Transit {
	t.Helper()
	adapter, err := NewTransit(
		transport.Client{HTTP: server.Client()},
		cancellingTransitConfigurationResolver{configuration: configuration, cancel: cancel, enabled: cancelAfterConfiguration},
		cancellingTransitCredentialReader{cancel: cancel, enabled: !cancelAfterConfiguration},
	)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func TestTransitPreservesOnlyTypedTransportAvailability(t *testing.T) {
	for _, operation := range []string{"provider_connection", "provider_timeout"} {
		err := transitProviderFailure(context.Background(), fault.New(fault.Unavailable, operation))
		if !fault.Is(err, fault.Unavailable) {
			t.Fatalf("typed availability operation=%q error=%v", operation, err)
		}
	}
	for _, err := range []error{
		fault.New(fault.Unavailable, "provider_response"),
		fault.New(fault.Trust, "provider_tls"),
	} {
		if result := transitProviderFailure(context.Background(), err); !fault.Is(result, fault.Trust) {
			t.Fatalf("unqualified transport error=%v result=%v", err, result)
		}
	}
}

type transitConfigurationFunc func(context.Context, KMSAssociation) (TransitConfiguration, error)

func (function transitConfigurationFunc) TransitConfiguration(ctx context.Context, association KMSAssociation) (TransitConfiguration, error) {
	return function(ctx, association)
}

type transitTokenFunc func(context.Context, string) (string, error)

func (function transitTokenFunc) Token(ctx context.Context, reference string) (string, error) {
	return function(ctx, reference)
}

func transitTestAdapter(t *testing.T, server *httptest.Server, configuration TransitConfiguration) *Transit {
	t.Helper()
	adapter, err := NewTransit(
		transport.Client{HTTP: server.Client()},
		transitConfigurationFunc(func(_ context.Context, _ KMSAssociation) (TransitConfiguration, error) {
			return configuration, nil
		}),
		transitTokenFunc(func(_ context.Context, reference string) (string, error) {
			if reference != "synthetic-credential-reference" {
				t.Fatal("adapter selected an unexpected credential reference")
			}
			return "synthetic-token", nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func transitTestAssociation() KMSAssociation {
	return KMSAssociation{
		Provider:                 transitProviderOpenBao,
		KeyID:                    "workload-key",
		KeyVersion:               "1",
		Algorithm:                transitAlgorithm,
		Purpose:                  "workload-data-key",
		ConfigurationAttestation: "synthetic-attestation",
	}
}

func transitTestContext() AssociationContext {
	return AssociationContext{
		Tenant:          "synthetic-tenant",
		Profile:         "synthetic-profile",
		Application:     "synthetic-application",
		Principal:       "synthetic-principal",
		Generation:      "1",
		Purpose:         "workload-data-key",
		ArtifactID:      "synthetic-artifact",
		ArtifactVersion: "1",
	}
}

func transitTestConfiguration(endpoint string) TransitConfiguration {
	return TransitConfiguration{
		Endpoint:             endpoint,
		Mount:                "transit",
		Namespace:            "synthetic-namespace",
		CredentialReference:  "synthetic-credential-reference",
		MinDecryptionVersion: "1",
		MinEncryptionVersion: "0",
	}
}

func transitKeyDocument(derived, convergent, exportable, backup bool) map[string]any {
	return transitKeyDocumentVersions(1, map[string]int{"1": 1}, derived, convergent, exportable, backup)
}

func transitKeyDocumentVersions(latestVersion int, versions map[string]int, derived, convergent, exportable, backup bool) map[string]any {
	return map[string]any{"data": map[string]any{
		"name":                   "workload-key",
		"type":                   transitAlgorithm,
		"latest_version":         latestVersion,
		"min_decryption_version": 1,
		"min_encryption_version": 0,
		"keys":                   versions,
		"derived":                derived,
		"convergent_encryption":  convergent,
		"exportable":             exportable,
		"allow_plaintext_backup": backup,
	}}
}

func decodeTransitJSON(t *testing.T, request *http.Request, target any) {
	t.Helper()
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func writeTransitJSON(t *testing.T, response http.ResponseWriter, value any) {
	t.Helper()
	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(value); err != nil {
		t.Fatal(err)
	}
}
