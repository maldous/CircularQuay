package keyservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/transport"
)

func TestTransitConfigurationFailuresStopBeforeCredentialAndProviderAccess(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		kind fault.Kind
	}{
		{"invalid configuration", wrappedTransitConfigurationError(fault.Invalid), fault.Invalid},
		{"denied configuration", wrappedTransitConfigurationError(fault.Denied), fault.Denied},
		{"missing configuration", wrappedTransitConfigurationError(fault.Missing), fault.Missing},
		{"unqualified availability", wrappedTransitConfigurationError(fault.Unavailable), fault.Trust},
		{"untyped private error", errors.New(transitConfigurationPrivateMarker), fault.Trust},
	} {
		for _, operation := range []string{"wrap", "unwrap"} {
			t.Run(test.name+"/"+operation, func(t *testing.T) {
				checkTransitConfigurationFailure(t, operation, test.err, test.kind)
			})
		}
	}
}

const transitConfigurationPrivateMarker = "synthetic-private-endpoint-and-credential-reference"

func wrappedTransitConfigurationError(kind fault.Kind) error {
	return fmt.Errorf("%s: %w", transitConfigurationPrivateMarker, fault.New(kind, transitConfigurationPrivateMarker))
}

func checkTransitConfigurationFailure(t *testing.T, operation string, resolverError error, wantKind fault.Kind) {
	t.Helper()
	credentialReads, providerRequests := 0, 0
	adapter, err := NewTransit(
		transport.Client{HTTP: &http.Client{Transport: transitFailureTransport{calls: &providerRequests}}},
		transitConfigurationFunc(func(context.Context, KMSAssociation) (TransitConfiguration, error) {
			return TransitConfiguration{}, resolverError
		}),
		transitTokenFunc(func(context.Context, string) (string, error) {
			credentialReads++
			return "synthetic-token", nil
		}),
	)
	if err != nil {
		t.Fatal("explicit Transit collaborators were rejected")
	}
	material, receipt, err := transitFailureOperation(adapter, operation)
	if len(material) != 0 || receipt != (KMSReceipt{}) {
		t.Fatal("configuration failure returned key material or a success receipt")
	}
	if credentialReads != 0 || providerRequests != 0 {
		t.Fatal("configuration failure reached credentials or the provider")
	}
	checkSanitizedTransitConfigurationError(t, err, wantKind)
}

func transitFailureOperation(adapter *Transit, operation string) ([]byte, KMSReceipt, error) {
	if operation == "wrap" {
		result, err := adapter.Wrap(context.Background(), WrapRequest{
			Association: transitTestAssociation(), Context: transitTestContext(), Plaintext: []byte("synthetic-key"),
		})
		return result.Ciphertext, result.Receipt, err
	}
	result, err := adapter.Unwrap(context.Background(), UnwrapRequest{
		Association: transitTestAssociation(), Context: transitTestContext(), Ciphertext: []byte("vault:v1:c3ludGhldGlj"),
	})
	return result.Plaintext, result.Receipt, err
}

func checkSanitizedTransitConfigurationError(t *testing.T, err error, wantKind fault.Kind) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != wantKind || safe.Operation != "transit_configuration" {
		t.Fatal("configuration failure lost its safe public category or operation")
	}
	if strings.Contains(err.Error(), transitConfigurationPrivateMarker) || errors.Unwrap(err) != nil {
		t.Fatal("configuration failure retained private error details")
	}
}

// This transport can record an attempted effect, but never opens a connection.
type transitFailureTransport struct{ calls *int }

func (transport transitFailureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	*transport.calls++
	return nil, errors.New("unexpected synthetic provider request")
}
