package bootstrap

import (
	"context"
	"net/http"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestRequestBoundaryContractMalformedAddressCannotReachTransport(t *testing.T) {
	fetcher := testFetcher(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid request reached transport")
		return nil, nil
	}))
	response, err := fetcher.get(context.Background(), "https://invalid%host/fixture")
	if response != nil || !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid request returned a response or changed error category")
	}
}
