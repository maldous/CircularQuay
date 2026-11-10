package source

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/transport"
)

type cloudRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip cloudRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestCloudHTTPClientRejectsRedirectAndOversizedSDKResponse(t *testing.T) {
	t.Run("redirect", func(t *testing.T) {
		client := cloudHTTPClient(cloudRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"https://redirect.invalid"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}))
		if _, err := client.Get("https://provider.invalid"); err == nil {
			t.Fatal("provider redirect accepted")
		}
	})
	t.Run("chunked oversized", func(t *testing.T) {
		client := cloudHTTPClient(cloudRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, ContentLength: -1, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 4<<20+1)))}, nil
		}))
		response, err := client.Get("https://provider.invalid")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if _, err = io.ReadAll(response.Body); !errors.Is(err, transport.ErrResponseTooLarge) {
			t.Fatalf("oversized SDK response result = %v", err)
		}
	})
}
