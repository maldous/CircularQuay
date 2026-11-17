package deliver

import (
	"context"
	"net/http"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

type publicationBoundaryTransport struct{ calls int }

func (transport *publicationBoundaryTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls++
	return nil, fault.New(fault.Unavailable, "synthetic transport fence")
}

func TestPublicationBoundaryContractEndpointRefusesBeforeTokenTransmission(t *testing.T) {
	contained := &publicationBoundaryTransport{}
	client := transport.Client{HTTP: &http.Client{Transport: contained}}
	target := model.Target{Account: "fixture", Project: "fixture", Environment: "fixture", Endpoint: "http://forbidden.invalid"}
	err := netlify(context.Background(), client, target, "synthetic-private-token", "synthetic-private-bundle")
	if !fault.Is(err, fault.Trust) || contained.calls != 0 {
		t.Fatal("invalid endpoint transmitted credentials or changed refusal category")
	}
}
