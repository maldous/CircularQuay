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
	association := transitTestAssociation()
	requestContext := transitTestContext()
	plaintext := []byte("synthetic-data-key")
	ciphertext := "vault:v1:" + base64.StdEncoding.EncodeToString([]byte("synthetic-ciphertext"))
	var contexts []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Vault-Token") != "synthetic-token" || request.Header.Get("X-Vault-Namespace") != "synthetic-namespace" {
			t.Fatal("missing protected Transit request headers")
		}
		switch request.URL.Path {
		case "/v1/transit/keys/workload-key":
			writeTransitJSON(t, response, transitKeyDocument(true, false, false, false))
		case "/v1/transit/encrypt/workload-key":
			var body transitEncryptRequest
			decodeTransitJSON(t, request, &body)
			if body.Plaintext != base64.StdEncoding.EncodeToString(plaintext) || body.KeyVersion != 1 || body.Context == "" {
				t.Fatal("unexpected Transit encryption body")
			}
			contexts = append(contexts, body.Context)
			writeTransitJSON(t, response, map[string]any{"data": map[string]any{"ciphertext": ciphertext, "key_version": 1}})
		case "/v1/transit/decrypt/workload-key":
			var body transitDecryptRequest
			decodeTransitJSON(t, request, &body)
			if body.Ciphertext != ciphertext || body.Context == "" {
				t.Fatal("unexpected Transit decryption body")
			}
			contexts = append(contexts, body.Context)
			writeTransitJSON(t, response, map[string]any{"data": map[string]any{"plaintext": base64.StdEncoding.EncodeToString(plaintext)}})
		default:
			t.Fatalf("unexpected Transit path %q", request.URL.Path)
		}
	}))
	defer server.Close()

	transit := transitTestAdapter(t, server, transitTestConfiguration(server.URL))
	wrapped, err := transit.Wrap(context.Background(), WrapRequest{Association: association, Context: requestContext, Plaintext: plaintext})
	if err != nil || string(wrapped.Ciphertext) != ciphertext || wrapped.Receipt != transitReceipt(association, requestContext) {
		t.Fatalf("wrap result=%+v error=%v", wrapped.Receipt, err)
	}
	unwrapped, err := transit.Unwrap(context.Background(), UnwrapRequest{Association: association, Context: requestContext, Ciphertext: wrapped.Ciphertext})
	if err != nil || string(unwrapped.Plaintext) != string(plaintext) || unwrapped.Receipt != transitReceipt(association, requestContext) {
		t.Fatalf("unwrap receipt=%+v error=%v", unwrapped.Receipt, err)
	}
	if len(contexts) != 2 || contexts[0] != contexts[1] {
		t.Fatalf("Transit context was not stable across wrap and unwrap")
	}
	decoded, err := base64.StdEncoding.DecodeString(contexts[0])
	if err != nil || string(decoded) != string(transitDerivedContext(requestContext)) {
		t.Fatal("Transit request did not bind the service-built stable context")
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
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				calls++
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			configuration := transitTestConfiguration(server.URL)
			resolver := transitConfigurationFunc(func(_ context.Context, _ KMSAssociation) (TransitConfiguration, error) {
				if test.configuration {
					cancel()
				}
				return configuration, nil
			})
			credentials := transitTokenFunc(func(_ context.Context, _ string) (string, error) {
				if !test.configuration {
					cancel()
				}
				return "synthetic-token", nil
			})
			adapter, err := NewTransit(transport.Client{HTTP: server.Client()}, resolver, credentials)
			if err != nil {
				t.Fatal(err)
			}
			_, err = adapter.Wrap(ctx, WrapRequest{Association: transitTestAssociation(), Context: transitTestContext(), Plaintext: []byte("synthetic")})
			if !fault.Is(err, fault.Denied) || calls != 0 {
				t.Fatalf("cancelled %s error=%v requests=%d", test.name, err, calls)
			}
		})
	}
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
