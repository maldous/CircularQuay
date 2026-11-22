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

func TestTransitCredentialFailuresReturnNoMaterialBeforeHTTP(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		kind fault.Kind
	}{
		{"invalid credential", privateTransitCredentialError(fault.Invalid), fault.Invalid},
		{"denied credential", privateTransitCredentialError(fault.Denied), fault.Denied},
		{"missing credential", privateTransitCredentialError(fault.Missing), fault.Trust},
		{"unqualified availability", privateTransitCredentialError(fault.Unavailable), fault.Trust},
		{"untyped private error", errors.New(transitCredentialPrivateMarker), fault.Trust},
	} {
		for _, operation := range []string{"wrap", "unwrap"} {
			t.Run(test.name+"/"+operation, func(t *testing.T) {
				checkTransitCredentialFailure(t, operation, test.err, test.kind)
			})
		}
	}
}

const transitCredentialPrivateMarker = "synthetic-private-token-resolver-details"

func privateTransitCredentialError(kind fault.Kind) error {
	return fmt.Errorf("%s: %w", transitCredentialPrivateMarker, fault.New(kind, transitCredentialPrivateMarker))
}

func checkTransitCredentialFailure(t *testing.T, operation string, resolverError error, wantKind fault.Kind) {
	t.Helper()
	credentialRead, providerRequests := false, 0
	adapter, err := NewTransit(
		transport.Client{HTTP: &http.Client{Transport: transitFailureTransport{calls: &providerRequests}}},
		transitConfigurationFunc(func(context.Context, KMSAssociation) (TransitConfiguration, error) {
			return transitTestConfiguration("https://transit.example.invalid"), nil
		}),
		transitTokenFunc(func(_ context.Context, reference string) (string, error) {
			if reference != "synthetic-credential-reference" {
				t.Fatal("credential resolution used an unbound reference")
			}
			credentialRead = true
			// A resolver may supply a partial value with its error. It must not
			// authorize a request or survive in the returned failure.
			return transitCredentialPrivateMarker, resolverError
		}),
	)
	if err != nil {
		t.Fatal("explicit Transit collaborators were rejected")
	}
	material, receipt, err := transitFailureOperation(adapter, operation)
	if !credentialRead || providerRequests != 0 {
		t.Fatal("credential failure was not enforced before provider access")
	}
	if len(material) != 0 || receipt != (KMSReceipt{}) {
		t.Fatal("credential failure returned key material or a success receipt")
	}
	checkSanitizedTransitCredentialError(t, err, wantKind)
}

func checkSanitizedTransitCredentialError(t *testing.T, err error, wantKind fault.Kind) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != wantKind || safe.Operation != "transit_credential" {
		t.Fatal("credential failure lost its safe public category or operation")
	}
	if strings.Contains(err.Error(), transitCredentialPrivateMarker) || errors.Unwrap(err) != nil {
		t.Fatal("credential failure retained private error details")
	}
}
