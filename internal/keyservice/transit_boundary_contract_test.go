package keyservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/transport"
)

func TestTransitBoundaryContractProtectedCredentialIsExplicitAndCancelable(t *testing.T) {
	path := transitBoundaryCredentialFile(t)
	directory := filepath.Dir(path)
	const content = "synthetic-transit-token\n"
	reader := ProtectedCredentialReader{}
	token, err := reader.Token(context.Background(), path)
	if err != nil || token != "synthetic-transit-token" {
		t.Fatal("explicit owned credential did not preserve its bounded token")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	token, err = reader.Token(cancelled, filepath.Join(directory, "absent"))
	requireTransitBoundaryError(t, err, fault.Denied, "transit_cancelled")
	if token != "" {
		t.Fatal("cancelled credential read returned material")
	}
	token, err = reader.Token(context.Background(), filepath.Join(directory, "absent"))
	requireTransitBoundaryError(t, err, fault.Denied, "credential_file")
	if token != "" {
		t.Fatal("missing credential read returned material")
	}
	data, err := platform.ReadProtected(path, true)
	if err != nil || string(data) != content {
		t.Fatal("credential reads changed their owned input")
	}
}

func transitBoundaryCredentialFile(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "owned-credential")
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "token")
	const content = "synthetic-transit-token\n"
	if err := platform.WriteProtected(path, []byte(content)); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTransitBoundaryContractConstructorRequiresBothCollaborators(t *testing.T) {
	configuration := transitConfigurationFunc(func(context.Context, KMSAssociation) (TransitConfiguration, error) {
		t.Fatal("constructor resolved configuration")
		return TransitConfiguration{}, nil
	})
	credential := transitTokenFunc(func(context.Context, string) (string, error) {
		t.Fatal("constructor resolved a credential")
		return "", nil
	})
	for _, collaborators := range []struct {
		name          string
		configuration TransitConfigurationResolver
		credential    CredentialReader
	}{
		{"configuration", nil, credential},
		{"credential", configuration, nil},
	} {
		t.Run(collaborators.name, func(t *testing.T) {
			adapter, err := NewTransit(transport.Client{}, collaborators.configuration, collaborators.credential)
			requireTransitBoundaryError(t, err, fault.Invalid, "transit_dependencies")
			if adapter != nil {
				t.Fatal("missing collaborator produced an adapter")
			}
		})
	}
}

func TestTransitBoundaryContractWrapAdmissionPrecedesPrivateAccess(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		change    func(*WrapRequest)
		kind      fault.Kind
		operation string
	}{
		{"empty material", func(r *WrapRequest) { r.Plaintext = nil }, fault.Invalid, "transit_wrap_request"},
		{"oversize material", func(r *WrapRequest) { r.Plaintext = make([]byte, maximumTransitMaterial+1) }, fault.Invalid, "transit_wrap_request"},
		{"provider", func(r *WrapRequest) { r.Association.Provider = "synthetic-unsupported" }, fault.Unsupported, "transit_provider"},
		{"nondigit version", func(r *WrapRequest) { r.Association.KeyVersion = "1x" }, fault.Unsupported, "transit_association"},
		{"purpose", func(r *WrapRequest) { r.Context.Purpose = "another-purpose" }, fault.Invalid, "transit_context"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := newTransitBoundaryFixture(t)
			request := WrapRequest{Association: transitTestAssociation(), Context: transitTestContext(), Plaintext: []byte("synthetic-key")}
			scenario.change(&request)
			before := bytes.Clone(request.Plaintext)
			result, err := fixture.adapter().Wrap(context.Background(), request)
			requireTransitBoundaryError(t, err, scenario.kind, scenario.operation)
			if len(result.Ciphertext) != 0 || result.Receipt != (KMSReceipt{}) || !bytes.Equal(request.Plaintext, before) {
				t.Fatal("refused admission returned material or changed input")
			}
			fixture.requireCalls()
		})
	}
}

func TestTransitBoundaryContractInvalidResolvedTokenNeverReachesHTTP(t *testing.T) {
	for _, token := range []string{"", strings.Repeat("s", maximumTransitToken+1), "synthetic\nprivate", string([]byte{0xff})} {
		for _, operation := range []string{"wrap", "unwrap"} {
			t.Run(operation+"/token", func(t *testing.T) {
				fixture := newTransitBoundaryFixture(t)
				fixture.token = token
				material, receipt, err := transitBoundaryInvoke(t, context.Background(), fixture.adapter(), operation)
				requireTransitBoundaryEmpty(t, material, receipt, err, fault.Trust, "transit_credential")
				fixture.requireCalls("configuration", "credential")
			})
		}
	}
}

func TestTransitBoundaryContractProviderStatusesWithholdAllResults(t *testing.T) {
	for _, phase := range []string{"keys", "encrypt", "decrypt"} {
		for _, status := range []struct {
			code int
			kind fault.Kind
		}{{http.StatusForbidden, fault.Denied}, {http.StatusNotFound, fault.Missing}, {http.StatusInternalServerError, fault.Trust}} {
			t.Run(phase+"/status", func(t *testing.T) {
				assertTransitBoundaryStatus(t, phase, status.code, status.kind)
			})
		}
	}
}

func assertTransitBoundaryStatus(t *testing.T, phase string, status int, kind fault.Kind) {
	t.Helper()
	fixture := newTransitBoundaryFixture(t)
	fixture.respond = func(request *http.Request) (*http.Response, error) {
		if transitBoundaryPhase(t, request) == phase {
			response := transitAdmissionResponse("SYNTHETIC_PRIVATE_PROVIDER_BODY")
			response.StatusCode = status
			return response, nil
		}
		return fixture.success(request), nil
	}
	material, receipt, err := transitBoundaryInvoke(t, context.Background(), fixture.adapter(), transitBoundaryOperation(phase))
	requireTransitBoundaryEmpty(t, material, receipt, err, kind, "transit_provider")
	fixture.requireCalls(transitBoundaryPhaseCalls(phase)...)
}

func TestTransitBoundaryContractCancellationAfterResponsePreventsRelease(t *testing.T) {
	for _, phase := range []string{"keys", "encrypt", "decrypt"} {
		for _, fails := range []bool{false, true} {
			t.Run(phase+"/cancellation", func(t *testing.T) {
				assertTransitBoundaryCancellation(t, phase, fails)
			})
		}
	}
}

func assertTransitBoundaryCancellation(t *testing.T, phase string, fails bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fixture := newTransitBoundaryFixture(t)
	fixture.respond = func(request *http.Request) (*http.Response, error) {
		response := fixture.success(request)
		if transitBoundaryPhase(t, request) == phase {
			cancel()
			if fails {
				_ = response.Body.Close()
				return nil, errors.New("SYNTHETIC_PRIVATE_TRANSPORT_ERROR")
			}
		}
		return response, nil
	}
	material, receipt, err := transitBoundaryInvoke(t, ctx, fixture.adapter(), transitBoundaryOperation(phase))
	requireTransitBoundaryEmpty(t, material, receipt, err, fault.Denied, "transit_cancelled")
	fixture.requireCalls(transitBoundaryPhaseCalls(phase)...)
}

func TestTransitBoundaryContractMalformedProviderMaterialHasNoPartialReceipt(t *testing.T) {
	for _, scenario := range []struct {
		name, phase, body, operation string
	}{
		{"key version", "encrypt", `{"data":{"ciphertext":"vault:v1:c3ludGhldGlj","key_version":2}}`, "transit_encrypt_response"},
		{"ciphertext version", "encrypt", `{"data":{"ciphertext":"vault:v2:c3ludGhldGlj","key_version":1}}`, "transit_encrypt_response"},
		{"ciphertext control", "encrypt", `{"data":{"ciphertext":"vault:v1:c3ludGhldGlj\u0000","key_version":1}}`, "transit_encrypt_response"},
		{"plaintext encoding", "decrypt", `{"data":{"plaintext":"not base64!"}}`, "transit_decrypt_response"},
		{"plaintext empty", "decrypt", `{"data":{"plaintext":""}}`, "transit_decrypt_response"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := newTransitBoundaryFixture(t)
			fixture.respond = func(request *http.Request) (*http.Response, error) {
				if transitBoundaryPhase(t, request) == scenario.phase {
					return transitAdmissionResponse(scenario.body), nil
				}
				return fixture.success(request), nil
			}
			material, receipt, err := transitBoundaryInvoke(t, context.Background(), fixture.adapter(), transitBoundaryOperation(scenario.phase))
			requireTransitBoundaryEmpty(t, material, receipt, err, fault.Trust, scenario.operation)
			fixture.requireCalls(transitBoundaryPhaseCalls(scenario.phase)...)
		})
	}
}

func TestTransitBoundaryContractKeyMetadataRefusesBeforeMaterialRequest(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		minimum string
		change  func(map[string]any)
	}{
		{"different key", "1", func(data map[string]any) { data["name"] = "other-key" }},
		{"retired version", "2", func(data map[string]any) {
			data["latest_version"] = 2
			data["min_decryption_version"] = 2
			data["keys"] = map[string]int{"1": 1, "2": 2}
		}},
		{"missing security metadata", "1", func(data map[string]any) { delete(data, "derived") }},
		{"invalid latest", "1", func(data map[string]any) { data["latest_version"] = 0 }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := newTransitBoundaryFixture(t)
			fixture.configuration.MinDecryptionVersion = scenario.minimum
			document := transitKeyDocument(true, false, false, false)
			scenario.change(document["data"].(map[string]any))
			fixture.respond = func(*http.Request) (*http.Response, error) {
				return transitBoundaryJSON(t, document), nil
			}
			material, receipt, err := transitBoundaryInvoke(t, context.Background(), fixture.adapter(), "wrap")
			requireTransitBoundaryEmpty(t, material, receipt, err, fault.Trust, "transit_key_configuration")
			fixture.requireCalls("configuration", "credential", "keys")
		})
	}
}

func TestTransitBoundaryContractMissingOrWrongDataEnvelopeRefuses(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":"SYNTHETIC_PRIVATE_DATA"}`} {
		t.Run("data envelope", func(t *testing.T) {
			fixture := newTransitBoundaryFixture(t)
			fixture.respond = func(*http.Request) (*http.Response, error) {
				return transitAdmissionResponse(body), nil
			}
			material, receipt, err := transitBoundaryInvoke(t, context.Background(), fixture.adapter(), "unwrap")
			requireTransitBoundaryEmpty(t, material, receipt, err, fault.Trust, "transit_provider")
			fixture.requireCalls("configuration", "credential", "keys")
		})
	}
}

func TestTransitBoundaryContractValidFakeTransportPreservesBoundReceipt(t *testing.T) {
	for _, operation := range []string{"wrap", "unwrap"} {
		t.Run(operation, func(t *testing.T) {
			fixture := newTransitBoundaryFixture(t)
			material, receipt, err := transitBoundaryInvoke(t, context.Background(), fixture.adapter(), operation)
			phase, want := "encrypt", "vault:v1:c3ludGhldGlj"
			if operation == "unwrap" {
				phase, want = "decrypt", "synthetic-key"
			}
			if err != nil || string(material) != want || receipt != transitAdmissionExpectedReceipt(transitTestAssociation(), transitTestContext()) {
				t.Fatal("valid synthetic transport lost its material or exact receipt")
			}
			fixture.requireCalls("configuration", "credential", "keys", phase)
		})
	}
}

type transitBoundaryFixture struct {
	t             *testing.T
	configuration TransitConfiguration
	token         string
	calls         []string
	respond       transitAdmissionTransport
}

func newTransitBoundaryFixture(t *testing.T) *transitBoundaryFixture {
	t.Helper()
	return &transitBoundaryFixture{t: t, configuration: transitTestConfiguration("https://transit.example.invalid"), token: "synthetic-token"}
}

func (f *transitBoundaryFixture) adapter() *Transit {
	f.t.Helper()
	adapter, err := NewTransit(
		transport.Client{HTTP: &http.Client{Transport: transitAdmissionTransport(f.roundTrip)}},
		transitConfigurationFunc(func(context.Context, KMSAssociation) (TransitConfiguration, error) {
			f.calls = append(f.calls, "configuration")
			return f.configuration, nil
		}),
		transitTokenFunc(func(_ context.Context, reference string) (string, error) {
			f.calls = append(f.calls, "credential")
			if reference != "synthetic-credential-reference" {
				f.t.Fatal("credential reference changed")
			}
			return f.token, nil
		}),
	)
	if err != nil {
		f.t.Fatal(err)
	}
	return adapter
}

func (f *transitBoundaryFixture) roundTrip(request *http.Request) (*http.Response, error) {
	f.t.Helper()
	assertTransitAdmissionRequestBinding(f.t, request, "request escaped the fixed synthetic origin or headers")
	f.calls = append(f.calls, transitBoundaryPhase(f.t, request))
	if f.respond != nil {
		return f.respond(request)
	}
	return f.success(request), nil
}

func (f *transitBoundaryFixture) success(request *http.Request) *http.Response {
	f.t.Helper()
	switch transitBoundaryPhase(f.t, request) {
	case "keys":
		return transitAdmissionKeyReply(f.t)
	case "encrypt":
		assertTransitAdmissionEncryptBody(f.t, request)
		return transitAdmissionResponse(`{"data":{"ciphertext":"vault:v1:c3ludGhldGlj","key_version":1}}`)
	default:
		assertTransitAdmissionDecryptBody(f.t, request, []byte("vault:v1:c3ludGhldGlj"))
		return transitAdmissionResponse(`{"data":{"plaintext":"c3ludGhldGljLWtleQ=="}}`)
	}
}

func (f *transitBoundaryFixture) requireCalls(want ...string) {
	f.t.Helper()
	if !slices.Equal(f.calls, want) {
		f.t.Fatal("Transit crossed an unexpected admission or release boundary")
	}
}

func transitBoundaryPhase(t *testing.T, request *http.Request) string {
	t.Helper()
	switch {
	case request.Method == http.MethodGet && request.URL.Path == "/v1/transit/keys/workload-key":
		return "keys"
	case request.Method == http.MethodPost && request.URL.Path == "/v1/transit/encrypt/workload-key":
		return "encrypt"
	case request.Method == http.MethodPost && request.URL.Path == "/v1/transit/decrypt/workload-key":
		return "decrypt"
	default:
		t.Fatal("unexpected synthetic Transit method or path")
		return ""
	}
}

func transitBoundaryOperation(phase string) string {
	if phase == "decrypt" {
		return "unwrap"
	}
	return "wrap"
}

func transitBoundaryPhaseCalls(phase string) []string {
	calls := []string{"configuration", "credential", "keys"}
	if phase != "keys" {
		calls = append(calls, phase)
	}
	return calls
}

func transitBoundaryInvoke(t *testing.T, ctx context.Context, adapter *Transit, operation string) ([]byte, KMSReceipt, error) {
	t.Helper()
	if operation == "wrap" {
		input := []byte("synthetic-key")
		result, err := adapter.Wrap(ctx, WrapRequest{Association: transitTestAssociation(), Context: transitTestContext(), Plaintext: input})
		if string(input) != "synthetic-key" {
			t.Fatal("Wrap changed caller material")
		}
		return result.Ciphertext, result.Receipt, err
	}
	input := []byte("vault:v1:c3ludGhldGlj")
	result, err := adapter.Unwrap(ctx, UnwrapRequest{Association: transitTestAssociation(), Context: transitTestContext(), Ciphertext: input})
	if string(input) != "vault:v1:c3ludGhldGlj" {
		t.Fatal("Unwrap changed caller material")
	}
	return result.Plaintext, result.Receipt, err
}

func transitBoundaryJSON(t *testing.T, value any) *http.Response {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal("invalid synthetic response fixture")
	}
	return transitAdmissionResponse(string(body))
}

func requireTransitBoundaryEmpty(t *testing.T, material []byte, receipt KMSReceipt, err error, kind fault.Kind, operation string) {
	t.Helper()
	requireTransitBoundaryError(t, err, kind, operation)
	if len(material) != 0 || receipt != (KMSReceipt{}) {
		t.Fatal("refused Transit request returned partial material or receipt")
	}
}

func requireTransitBoundaryError(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != kind || safe.Operation != operation {
		t.Fatal("Transit refusal lost its exact safe category or operation")
	}
	if errors.Unwrap(err) != nil || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE") {
		t.Fatal("Transit refusal retained private diagnostics")
	}
}
