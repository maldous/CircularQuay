package assurance

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

type sonarBoundaryTransport func(*http.Request) (*http.Response, error)

func (transport sonarBoundaryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type sonarBoundaryReply struct {
	method string
	route  string
	status int
	body   string
	stream io.ReadCloser
}

// Every request terminates in this in-process transport. An unexpected request
// fails the test rather than falling back to a network, proxy, or local server.
func newSonarBoundaryClient(t *testing.T, replies []sonarBoundaryReply) *sonarClient {
	t.Helper()
	next := 0
	client := newSonarClient("http://sonar.invalid", "synthetic-user", "synthetic-password")
	client.client.Transport = sonarBoundaryTransport(func(request *http.Request) (*http.Response, error) {
		if next >= len(replies) {
			t.Error("unexpected Sonar request after the selected refusal")
			return nil, errors.New("synthetic unexpected request")
		}
		reply := replies[next]
		next++
		if request.Method != reply.method || request.URL.Path != reply.route {
			t.Error("Sonar failure sequence used an unexpected method or route")
		}
		body := reply.stream
		if body == nil {
			body = io.NopCloser(strings.NewReader(reply.body))
		}
		return &http.Response{StatusCode: reply.status, Body: body, Header: make(http.Header), Request: request}, nil
	})
	t.Cleanup(func() {
		if next != len(replies) {
			t.Error("Sonar fixture did not reach the selected failure boundary")
		}
	})
	return client
}

func assertSonarBoundaryError(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || err.Error() != expected {
		t.Fatal("missing or unexpected sanitized Sonar refusal")
	}
}

func TestSonarBoundaryContractRequestRefusalBeforeTransport(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		method   string
		context  context.Context
	}{
		{"invalid_endpoint", "http://[::1", http.MethodGet, context.Background()},
		{"invalid_method", "http://sonar.invalid", "invalid\nmethod", context.Background()},
		{"nil_context", "http://sonar.invalid", http.MethodGet, nil},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			client := newSonarBoundaryClient(t, nil)
			client.endpoint = testCase.endpoint
			err := client.call(testCase.context, testCase.method, "/api/fixture", url.Values{"fixture": {"public"}}, nil)
			assertSonarBoundaryError(t, err, "invalid Sonar request")
		})
	}
}

func TestSonarBoundaryContractSourceAndTaskIdentityBeforeTransport(t *testing.T) {
	client := newSonarBoundaryClient(t, nil)
	err := inspectSonarRevision(context.Background(), client, "fixture", "invalid revision", "fixture-analysis")
	assertSonarBoundaryError(t, err, "invalid sonar source revision")
	var summary SonarSummary
	err = inspectSonarTask(context.Background(), client, "fixture", "invalid/task", &summary)
	assertSonarBoundaryError(t, err, "invalid sonar task identity")
	if summary != (SonarSummary{}) {
		t.Fatal("invalid task identity changed the summary")
	}
}

type sonarBoundaryBody struct {
	reader io.Reader
	bytes  int
	closed int
}

func (body *sonarBoundaryBody) Read(data []byte) (int, error) {
	n, err := body.reader.Read(data)
	body.bytes += n
	return n, err
}

func (body *sonarBoundaryBody) Close() error {
	body.closed++
	return nil
}

type sonarBoundaryReadError struct{}

func (sonarBoundaryReadError) Read([]byte) (int, error) {
	return 0, errors.New("synthetic-private-reader-error")
}

func TestSonarBoundaryContractResponseDrainFailureAndLimit(t *testing.T) {
	t.Run("read_error", func(t *testing.T) {
		body := &sonarBoundaryBody{reader: sonarBoundaryReadError{}}
		client := newSonarBoundaryClient(t, []sonarBoundaryReply{{http.MethodGet, "/api/fixture", http.StatusOK, "", body}})
		err := client.call(context.Background(), http.MethodGet, "/api/fixture", nil, nil)
		assertSonarBoundaryError(t, err, "sonar response failed")
		if body.bytes != 0 || body.closed != 1 {
			t.Fatal("failed response drain did not retain its owned-body lifecycle")
		}
	})
	t.Run("one_mebibyte_limit", func(t *testing.T) {
		reader := strings.NewReader(strings.Repeat("x", (1<<20)+1))
		body := &sonarBoundaryBody{reader: reader}
		client := newSonarBoundaryClient(t, []sonarBoundaryReply{{http.MethodGet, "/api/fixture", http.StatusOK, "", body}})
		if err := client.call(context.Background(), http.MethodGet, "/api/fixture", nil, nil); err != nil {
			t.Fatal("bounded response drain rejected its valid control")
		}
		if body.bytes != 1<<20 || reader.Len() != 1 || body.closed != 1 {
			t.Fatal("response drain read beyond its bound or lost body closure")
		}
	})
}

func TestSonarBoundaryContractProtectedMetadataSizeAndAvailability(t *testing.T) {
	directory := privateAssuranceDirectory(t, "sonar-boundary-metadata")
	prefix := "ceTaskId=owned-fixture-task\n"
	atLimit := []byte(prefix + strings.Repeat("#", 4096-len(prefix)))
	validPath := filepath.Join(directory, "at-limit.properties")
	oversizedPath := filepath.Join(directory, "over-limit.properties")
	if err := platform.WriteProtected(validPath, atLimit); err != nil {
		t.Fatal("could not create owned metadata control")
	}
	if err := platform.WriteProtected(oversizedPath, append(bytes.Clone(atLimit), 'x')); err != nil {
		t.Fatal("could not create owned oversized metadata")
	}
	got, err := sonarMetadataTask(validPath)
	if err != nil || got != "owned-fixture-task" {
		t.Fatal("exact metadata byte limit rejected its valid control")
	}
	for _, path := range []string{filepath.Join(directory, "missing.properties"), directory, oversizedPath} {
		got, err := sonarMetadataTask(path)
		assertSonarBoundaryError(t, err, "sonar task metadata unavailable")
		if got != "" {
			t.Fatal("unavailable metadata supplied a task identity")
		}
	}
}

func TestSonarBoundaryContractOwnedStateRefusesInvalidShape(t *testing.T) {
	directory := privateAssuranceDirectory(t, "sonar-boundary-state")
	occupied := filepath.Join(directory, "occupied")
	original := []byte("synthetic non-directory fixture")
	if err := platform.WriteProtected(occupied, original); err != nil {
		t.Fatal("could not create owned non-directory fixture")
	}
	assertSonarBoundaryError(t, InitSonar(occupied), "sonar state protection failed")
	path, err := newSonarMetadata(filepath.Join(directory, "absent-parent"))
	assertSonarBoundaryError(t, err, "sonar task metadata creation failed")
	if path != "" {
		t.Fatal("failed metadata creation returned a path")
	}
	client, err := authenticateSonarAdministrator(context.Background(), directory, "://invalid")
	assertSonarBoundaryError(t, err, "invalid token file")
	if client != nil {
		t.Fatal("missing private administrator input created a client")
	}
	retained, err := platform.ReadProtected(occupied, true)
	if err != nil || !bytes.Equal(retained, original) {
		t.Fatal("state refusal changed the existing fixture")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "occupied" {
		t.Fatal("state refusal created bootstrap or metadata files")
	}
}

func sonarBoundaryGatePrefix() []sonarBoundaryReply {
	return []sonarBoundaryReply{
		{http.MethodPost, "/api/qualitygates/create", http.StatusNoContent, "", nil},
		{http.MethodGet, "/api/qualitygates/show", http.StatusOK, `{"conditions":[]}`, nil},
	}
}

func sonarBoundaryGateSelectionFailure() []sonarBoundaryReply {
	replies := sonarBoundaryGatePrefix()
	for count := 0; count < 8; count++ {
		replies = append(replies, sonarBoundaryReply{http.MethodPost, "/api/qualitygates/create_condition", http.StatusNoContent, "", nil})
	}
	return append(replies, sonarBoundaryReply{http.MethodPost, "/api/qualitygates/select", http.StatusServiceUnavailable, "synthetic-private-provider-body", nil})
}

func TestSonarBoundaryContractGateFailuresStopBeforeLaterEffects(t *testing.T) {
	cases := []struct {
		name    string
		replies []sonarBoundaryReply
		want    string
	}{
		{"create", []sonarBoundaryReply{{http.MethodPost, "/api/qualitygates/create", http.StatusServiceUnavailable, "synthetic-private-provider-body", nil}}, "sonar gate must be created in a fresh audit instance"},
		{"show", []sonarBoundaryReply{{http.MethodPost, "/api/qualitygates/create", http.StatusNoContent, "", nil}, {http.MethodGet, "/api/qualitygates/show", http.StatusServiceUnavailable, "synthetic-private-provider-body", nil}}, "sonar request rejected"},
		{"condition", append(sonarBoundaryGatePrefix(), sonarBoundaryReply{http.MethodPost, "/api/qualitygates/create_condition", http.StatusServiceUnavailable, "synthetic-private-provider-body", nil}), "sonar strict condition configuration failed"},
		{"assignment", sonarBoundaryGateSelectionFailure(), "sonar quality gate assignment failed"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			client := newSonarBoundaryClient(t, testCase.replies)
			err := configureSonarGate(context.Background(), client, "fixture")
			assertSonarBoundaryError(t, err, testCase.want)
		})
	}
}

func TestSonarBoundaryContractProfileDiscoveryFailsClosed(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"unavailable", http.StatusServiceUnavailable, "synthetic-private-provider-body", "sonar request rejected"},
		{"missing_required_language", http.StatusOK, `{"profiles":[]}`, "sonar comprehensive language profile unavailable"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			client := newSonarBoundaryClient(t, []sonarBoundaryReply{{http.MethodGet, "/api/qualityprofiles/search", testCase.status, testCase.body, nil}})
			err := configureSonarProfiles(context.Background(), client, "fixture")
			assertSonarBoundaryError(t, err, testCase.want)
		})
	}
}

func sonarBoundaryCopiedProfile() sonarBoundaryReply {
	return sonarBoundaryReply{http.MethodPost, "/api/qualityprofiles/copy", http.StatusOK, `{"key":"copied-fixture","name":"CQ Go comprehensive"}`, nil}
}

func sonarBoundaryProfile(language string) []sonarQualityProfile {
	return []sonarQualityProfile{{Key: "fixture-" + language, Name: "Sonar way comprehensive", Language: language, BuiltIn: true, ActiveRules: 23}}
}

func TestSonarBoundaryContractProfileFailuresStopBeforeAssignment(t *testing.T) {
	cases := []struct {
		name     string
		language string
		replies  []sonarBoundaryReply
		want     string
	}{
		{"copy_unavailable", "go", []sonarBoundaryReply{{http.MethodPost, "/api/qualityprofiles/copy", http.StatusServiceUnavailable, "synthetic-private-provider-body", nil}}, "sonar request rejected"},
		{"copy_without_key", "go", []sonarBoundaryReply{{http.MethodPost, "/api/qualityprofiles/copy", http.StatusOK, `{"name":"CQ Go comprehensive"}`, nil}}, "sonar profile copy failed"},
		{"copy_wrong_name", "go", []sonarBoundaryReply{{http.MethodPost, "/api/qualityprofiles/copy", http.StatusOK, `{"key":"copied-fixture","name":"wrong-fixture-name"}`, nil}}, "sonar profile copy failed"},
		{"complexity_rule", "go", []sonarBoundaryReply{sonarBoundaryCopiedProfile(), {http.MethodPost, "/api/qualityprofiles/activate_rule", http.StatusServiceUnavailable, "synthetic-private-provider-body", nil}}, "sonar complexity rule configuration failed"},
		{"assignment", "json", []sonarBoundaryReply{{http.MethodPost, "/api/qualityprofiles/add_project", http.StatusServiceUnavailable, "synthetic-private-provider-body", nil}}, "sonar comprehensive profile assignment failed"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			client := newSonarBoundaryClient(t, testCase.replies)
			err := configureSonarProfile(context.Background(), client, "fixture", testCase.language, sonarBoundaryProfile(testCase.language))
			assertSonarBoundaryError(t, err, testCase.want)
		})
	}
}

func TestSonarBoundaryContractProfilePositiveControl(t *testing.T) {
	client := newSonarBoundaryClient(t, []sonarBoundaryReply{
		sonarBoundaryCopiedProfile(),
		{http.MethodPost, "/api/qualityprofiles/activate_rule", http.StatusNoContent, "", nil},
		{http.MethodPost, "/api/qualityprofiles/add_project", http.StatusNoContent, "", nil},
	})
	if err := configureSonarProfile(context.Background(), client, "fixture", "go", sonarBoundaryProfile("go")); err != nil {
		t.Fatal("complete synthetic profile control was rejected")
	}
}
