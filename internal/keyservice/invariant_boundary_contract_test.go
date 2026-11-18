package keyservice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/transport"
)

// These are component contracts for general typed validator/mapper inputs.
// They do not claim a validated public request or concrete provider emits an
// impossible value, and never replace a cryptographic operation or result.
func TestKeyServiceInvariantBoundaryContractVersionPredicates(t *testing.T) {
	for _, value := range []string{"", "0", "00", "-1", "4294967296", "not-version"} {
		if _, ok := transitVersionNumber(value); ok {
			t.Fatal("noncanonical Transit version accepted")
		}
	}
	assertInvariantCanonicalVersions(t)
	for _, allowZero := range []bool{false, true} {
		if _, ok := transitComparableVersion("not-version", allowZero); ok {
			t.Fatal("malformed comparable Transit version accepted")
		}
	}
}

func assertInvariantCanonicalVersions(t *testing.T) {
	t.Helper()
	if number, ok := transitVersionNumber("4294967295"); !ok || number != 4294967295 {
		t.Fatal("maximum canonical Transit version refused")
	}
	if number, ok := transitComparableVersion("0", true); !ok || number != 0 {
		t.Fatal("explicit zero minimum refused")
	}
}

func TestKeyServiceInvariantBoundaryContractVersionRange(t *testing.T) {
	assertInvariantVersionRange(t, "1", "0", "2", true)
	assertInvariantVersionRange(t, "2", "2", "2", true)
	assertInvariantVersionRange(t, "bad", "0", "2", false)
	assertInvariantVersionRange(t, "1", "bad", "2", false)
	assertInvariantVersionRange(t, "1", "0", "bad", false)
	assertInvariantVersionRange(t, "1", "2", "3", false)
	assertInvariantVersionRange(t, "3", "0", "2", false)
}

func assertInvariantVersionRange(t *testing.T, value, minimum, maximum string, want bool) {
	t.Helper()
	if transitVersionInRange(value, minimum, maximum) != want {
		t.Fatal("Transit component range predicate changed")
	}
}

func TestKeyServiceInvariantBoundaryContractActionPredicate(t *testing.T) {
	config := transitKeyConfig{latestVersion: "1", minEncryptionVersion: "0"}
	if !transitConfigActionMatches("1", config, transitActionWrap) {
		t.Fatal("valid wrapping action refused")
	}
	if !transitConfigActionMatches("1", config, transitActionUnwrap) {
		t.Fatal("valid unwrapping action refused")
	}
	if transitConfigActionMatches("1", config, transitAction(0)) {
		t.Fatal("unknown typed action accepted")
	}
}

func TestKeyServiceInvariantBoundaryContractGeneralErrorMapper(t *testing.T) {
	for _, kind := range []fault.Kind{fault.Invalid, fault.Unsupported} {
		mapped := transitProviderFailure(context.Background(), fault.New(kind, "PRIVATE_COMPONENT_DETAIL"))
		assertInvariantFault(t, mapped, kind, "transit_provider")
	}
	mapped := transitProviderFailure(context.Background(), errors.New("PRIVATE_COMPONENT_DETAIL"))
	assertInvariantFault(t, mapped, fault.Trust, "transit_provider")
}

func TestKeyServiceInvariantBoundaryContractJWEInputFence(t *testing.T) {
	association, binding := softwareFixture()
	valid := invariantEnvelope(t, association, binding)
	if err := validateSoftwareJWE(valid, association, binding); err != nil {
		t.Fatal("fixed format-only JWE control refused", err)
	}
	for _, value := range [][]byte{nil, {0xff}, make([]byte, softwareCipherMax+1)} {
		assertInvariantFault(t, validateSoftwareJWE(value, association, binding), fault.Invalid, "software_jwe")
	}
}

func invariantEnvelope(t *testing.T, association KMSAssociation, binding AssociationContext) []byte {
	t.Helper()
	header := invariantHeader(t, association)
	encode := base64.RawURLEncoding.EncodeToString
	envelope := softwareEnvelope{
		Protected:    header,
		EncryptedKey: encode(make([]byte, 40)),
		IV:           encode(make([]byte, 12)),
		Ciphertext:   encode(make([]byte, softwareKeyBytes)),
		Tag:          encode(make([]byte, 16)),
		AAD:          encode(softwareAAD(binding)),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	// This is only a shape/binding validator control, not authenticated JWE or
	// evidence that these zero fixture bytes can be decrypted successfully.
	return data
}

func TestKeyServiceInvariantBoundaryContractHeaderDecodeFence(t *testing.T) {
	association, _ := softwareFixture()
	valid := invariantHeader(t, association)
	if !softwareHeaderMatches(valid, association) {
		t.Fatal("canonical component header refused")
	}
	for _, encoded := range []string{"!", valid + "="} {
		if softwareHeaderMatches(encoded, association) {
			t.Fatal("invalid raw base64 header accepted")
		}
	}
}

func TestKeyServiceInvariantBoundaryContractSerializedReleaseOwnsRefusal(t *testing.T) {
	association, binding := softwareFixture()
	valid := invariantEnvelope(t, association, binding)
	before := append([]byte(nil), valid...)
	result, err := softwareWrappedValue(valid, association, binding)
	if err != nil || !bytes.Equal(result.Ciphertext, before) || result.Receipt != softwareReceipt(association, binding) {
		t.Fatal("format-only release control changed bytes or receipt")
	}
	invalid := []byte("not-a-JWE")
	result, err = softwareWrappedValue(invalid, association, binding)
	assertInvariantFault(t, err, fault.Invalid, "software_jwe")
	if len(result.Ciphertext) != 0 || result.Receipt != (KMSReceipt{}) || !bytes.Equal(invalid, make([]byte, len(invalid))) {
		t.Fatal("serialized refusal retained owned bytes or returned partial state")
	}
	// These are direct owned-byte release contracts, not synthetic returns
	// attributed to NewEncrypter, EncryptWithAuthData or the JOSE parser.
}

func TestKeyServiceInvariantBoundaryContractPlaintextReleaseOwnsRefusal(t *testing.T) {
	association, binding := softwareFixture()
	valid := bytes.Repeat([]byte("x"), softwareKeyBytes)
	result, err := softwareUnwrappedValue(valid, association, binding)
	if err != nil || !bytes.Equal(result.Plaintext, valid) || result.Receipt != softwareReceipt(association, binding) {
		t.Fatal("owned plaintext release control failed")
	}
	invalid := bytes.Repeat([]byte("x"), softwareKeyBytes-1)
	result, err = softwareUnwrappedValue(invalid, association, binding)
	assertInvariantFault(t, err, fault.Trust, "software_plaintext")
	if len(result.Plaintext) != 0 || result.Receipt != (KMSReceipt{}) || !bytes.Equal(invalid, make([]byte, len(invalid))) {
		t.Fatal("plaintext refusal retained owned bytes or returned partial state")
	}
}

func TestKeyServiceInvariantBoundaryContractValidatedRoutesKeepExactPaths(t *testing.T) {
	configuration := TransitConfiguration{Endpoint: "https://synthetic.example.test/base/", Mount: "team/transit"}
	association := KMSAssociation{KeyID: "department/key", KeyVersion: "7"}
	assertInvariantRoutes(t, configuration, association, "https://synthetic.example.test/base/v1/team/transit/", "department/key")
	association.KeyID = "keys%2Fshadow"
	assertInvariantRoutes(t, configuration, association, "https://synthetic.example.test/base/v1/team/transit/", "keys%252Fshadow")
}

func assertInvariantRoutes(t *testing.T, configuration TransitConfiguration, association KMSAssociation, prefix, escapedKey string) {
	t.Helper()
	routes, err := newTransitRoutes(configuration, association)
	if err != nil || routes.version != 7 {
		t.Fatal("valid route metadata control failed")
	}
	before := routes.keys.String()
	if before != prefix+"keys/"+escapedKey {
		t.Fatal("validated key path changed")
	}
	for _, operation := range []string{"encrypt", "decrypt"} {
		endpoint := routes.operation(operation)
		if endpoint == routes.keys || endpoint.String() != prefix+operation+"/"+escapedKey {
			t.Fatal("constant-operation path lost exact bytes or isolation")
		}
		endpoint.Path = "/owned-clone-only"
	}
	if routes.keys.String() != before {
		t.Fatal("operation URL aliased the validated key path")
	}
}

func TestKeyServiceInvariantBoundaryContractRoutePlanRefusesBadMetadata(t *testing.T) {
	configuration := TransitConfiguration{Endpoint: "https://synthetic.example.test", Mount: "../outside"}
	association := KMSAssociation{KeyID: "key", KeyVersion: "1"}
	routes, err := newTransitRoutes(configuration, association)
	assertInvariantFault(t, err, fault.Invalid, "transit_resource")
	if routes != (transitRoutes{}) {
		t.Fatal("bad path returned a partial route plan")
	}
	configuration.Mount = "transit"
	association.KeyVersion = "0"
	routes, err = newTransitRoutes(configuration, association)
	assertInvariantFault(t, err, fault.Trust, "transit_key_configuration")
	if routes != (transitRoutes{}) {
		t.Fatal("bad version returned a partial route plan")
	}
	// No resolver, bootstrap identity or provider is selected by this pure
	// metadata plan. Public authorize still validates association/config first.
}

func TestKeyServiceInvariantBoundaryContractAuthorizationRefusalOrder(t *testing.T) {
	assertInvariantTransitOrder(t, "association", fault.Unsupported, "transit_association", nil)
	assertInvariantTransitOrder(t, "initial-cancelled", fault.Denied, "transit_cancelled", nil)
	assertInvariantTransitOrder(t, "path", fault.Invalid, "transit_resource", []string{"configuration"})
	assertInvariantTransitOrder(t, "config-denied", fault.Denied, "transit_configuration", []string{"configuration"})
	assertInvariantTransitOrder(t, "config-cancelled", fault.Denied, "transit_cancelled", []string{"configuration"})
	assertInvariantTransitOrder(t, "credential-denied", fault.Denied, "transit_credential", []string{"configuration", "credential"})
	assertInvariantTransitOrder(t, "credential-cancelled", fault.Denied, "transit_cancelled", []string{"configuration", "credential"})
}

func assertInvariantTransitOrder(t *testing.T, mode string, kind fault.Kind, operation string, want []string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	probe := invariantTransitOrderFixture(mode, cancel)
	association := transitTestAssociation()
	if mode == "association" {
		association.KeyVersion = "0"
	}
	if mode == "initial-cancelled" {
		cancel()
	}
	client := transport.Client{HTTP: &http.Client{Transport: transitAdmissionTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("refusal combination reached a provider request")
		return nil, errors.New("unselected HTTP")
	})}}
	adapter, err := NewTransit(client, probe, probe)
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Wrap(ctx, WrapRequest{Association: association, Context: transitTestContext(), Plaintext: []byte("synthetic-material")})
	assertInvariantFault(t, err, kind, operation)
	if len(result.Ciphertext) != 0 || result.Receipt != (KMSReceipt{}) || !reflect.DeepEqual(probe.calls, want) {
		t.Fatal("authorization/refusal order or zero-result semantics changed")
	}
}

type invariantTransitOrderProbe struct {
	configuration      TransitConfiguration
	configurationError error
	credentialError    error
	cancel             context.CancelFunc
	cancelAt           string
	calls              []string
}

func invariantTransitOrderFixture(mode string, cancel context.CancelFunc) *invariantTransitOrderProbe {
	probe := &invariantTransitOrderProbe{configuration: transitTestConfiguration("https://synthetic.example.test"), cancel: cancel}
	switch mode {
	case "association", "config-denied":
		probe.configuration.Mount = "../outside"
		probe.configurationError = fault.New(fault.Denied, "synthetic configuration denial")
	case "path":
		probe.configuration.Mount = "../outside"
		probe.credentialError = fault.New(fault.Denied, "synthetic credential denial")
	case "config-cancelled":
		probe.configuration.Mount = "../outside"
		probe.cancelAt = "configuration"
	case "credential-denied":
		probe.credentialError = fault.New(fault.Denied, "synthetic credential denial")
	case "credential-cancelled":
		probe.cancelAt = "credential"
	}
	return probe
}

func (probe *invariantTransitOrderProbe) TransitConfiguration(context.Context, KMSAssociation) (TransitConfiguration, error) {
	probe.calls = append(probe.calls, "configuration")
	if probe.cancelAt == "configuration" {
		probe.cancel()
	}
	return probe.configuration, probe.configurationError
}

func (probe *invariantTransitOrderProbe) Token(context.Context, string) (string, error) {
	probe.calls = append(probe.calls, "credential")
	if probe.cancelAt == "credential" {
		probe.cancel()
	}
	return "synthetic-token", probe.credentialError
}

func invariantHeader(t *testing.T, association KMSAssociation) string {
	t.Helper()
	fields := map[string]string{
		"alg":            "A256KW",
		"enc":            "A256GCM",
		"kid":            association.KeyID,
		softwareHeaderKV: association.KeyVersion,
		softwareHeaderCA: association.ConfigurationAttestation,
	}
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func assertInvariantFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != kind || safe.Operation != operation {
		t.Fatal("component failure lost its exact sanitized classification")
	}
}

func TestKeyServiceInvariantBoundaryContractPublicWrapRejectsEscapedProtectedHeader(t *testing.T) {
	association, binding := softwareFixture()
	association.KeyID = strings.Repeat(`"\`, 512)
	association.KeyVersion = strings.Repeat(`"\`, 128)
	association.ConfigurationAttestation = strings.Repeat(`"\`, 512)
	plaintext := softwareTestDEK()
	before := bytes.Clone(plaintext)
	request := WrapRequest{Association: association, Context: binding, Plaintext: plaintext}
	if err := validateSoftwareWrapRequest(context.Background(), request); err != nil {
		t.Fatal("valid escaped metadata was refused before real encryption")
	}
	resolver := &softwareResolver{key: softwareTestKEK()}
	adapter := softwareAdapter(t, resolver)
	// All 2304 admitted metadata bytes require two JSON bytes each. The real
	// single-recipient JOSE protected header therefore exceeds 4096 bytes even
	// before its field names/punctuation. Do not fake a cryptographic result.
	result, err := adapter.Wrap(context.Background(), request)
	assertSoftwareLifetimeFault(t, err, fault.Invalid, "software_jwe")
	if result.Ciphertext != nil || result.Receipt != (KMSReceipt{}) {
		t.Fatal("oversized generated header released ciphertext or partial receipt")
	}
	if len(resolver.issued) != 1 || !resolver.allCleared() {
		t.Fatal("real wrap did not resolve exactly once and clear the owned KEK")
	}
	if !bytes.Equal(plaintext, before) {
		t.Fatal("rejected real wrap changed caller plaintext")
	}
}
