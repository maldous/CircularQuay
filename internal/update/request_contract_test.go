package update

import (
	"context"
	"net/url"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestDefaultReleaseEndpointIsCredentialFreeAndRepositoryScoped(t *testing.T) {
	endpoint, err := feedEndpoint("")
	if err != nil || endpoint.String() != "https://api.github.com/repos/maldous/CircularQuay/releases" {
		t.Fatalf("default release endpoint escaped the product repository: %v err=%v", endpoint, err)
	}
	request, err := feedRequest(context.Background(), endpoint, "")
	if err != nil || request.Method != "GET" || request.Header.Get("Authorization") != "" || request.Header.Get("Accept") != "application/vnd.github+json" {
		t.Fatalf("public release request gained credentials or changed its contract: %v err=%v", request, err)
	}
}

func TestMalformedReleaseRequestFailsBeforeTransport(t *testing.T) {
	endpoint := &url.URL{Scheme: "https", Host: "invalid\x00host"}
	request, err := feedRequest(context.Background(), endpoint, "")
	if request != nil || !fault.Is(err, fault.Invalid) {
		t.Fatalf("malformed endpoint created a request: request=%v err=%v", request, err)
	}
}

func TestReleaseVersionNormalizationPreservesCompleteIdentity(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
		valid bool
	}{
		{"1.2.3", "v1.2.3", true},
		{"1.2.3-rc.1", "v1.2.3-rc.1", true},
		{"1.2", "v1.2.0", false},
		{"01.2.3", "", false},
	} {
		t.Run(test.input, func(t *testing.T) {
			version, valid := canonicalVersion(test.input)
			if version != test.want || valid != test.valid {
				t.Fatalf("version normalization = (%q, %v), want (%q, %v)", version, valid, test.want, test.valid)
			}
		})
	}
}
