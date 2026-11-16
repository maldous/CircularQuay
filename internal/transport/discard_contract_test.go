package transport

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestCallerTransportDiscardPreservesExactLimitAndBoundsOverflow(t *testing.T) {
	for _, scenario := range []struct {
		name string
		size int
	}{
		{"exact limit", maxPayload},
		{"one byte over", maxPayload + 1},
		{"larger response", maxPayload + 1024},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			reader := strings.NewReader(strings.Repeat("x", scenario.size))
			body := &trackedReadCloser{Reader: reader}
			requests := 0
			// A caller-owned raw transport exercises the Client's independent
			// discard bound, without the hardened transport's earlier refusal.
			client := Client{HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: http.StatusOK, ContentLength: -1, Body: body}, nil
			})}}
			err := client.DoJSON(context.Background(), http.MethodGet, testProviderURL(t), nil, nil, nil)
			assertDiscardBoundaryOutcome(t, err, scenario.size)
			if requests != 1 || !body.closed {
				t.Fatalf("discard lifecycle changed: requests %d, closed %t", requests, body.closed)
			}
			consumed := scenario.size - reader.Len()
			if consumed != min(scenario.size, maxPayload+1) {
				t.Fatalf("discard consumed %d bytes instead of its exact bounded allowance", consumed)
			}
		})
	}
}

func assertDiscardBoundaryOutcome(t *testing.T, err error, size int) {
	t.Helper()
	if size == maxPayload {
		if err != nil {
			t.Fatalf("exact-limit discarded response was refused: %v", err)
		}
		return
	}
	if !fault.Is(err, fault.Trust) {
		t.Fatalf("oversized discarded response escaped trust refusal: %v", err)
	}
}
