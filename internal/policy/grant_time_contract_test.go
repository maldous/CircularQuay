package policy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/model"
	"gopkg.in/yaml.v3"
)

func TestAuthorizeAtHonorsGrantWindowBoundaries(t *testing.T) {
	start := grantTime(t, "2030-01-02T03:04:05Z")
	end := grantTime(t, "2030-01-02T04:04:05Z")
	policy := grantWindowPolicy(start, end)
	for _, test := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"before start", start.Add(-time.Nanosecond), false},
		{"at start", *start, true},
		{"before end", end.Add(-time.Nanosecond), true},
		{"at end", *end, false},
		{"after end", end.Add(time.Nanosecond), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := AuthorizeAt(policy, "operator", "", "demo", "status", "", test.now); got != test.want {
				t.Fatalf("AuthorizeAt(%s) = %t, want %t", test.now, got, test.want)
			}
		})
	}
	if AuthorizeAt(policy, "operator", "", "demo", "status", "", time.Time{}) {
		t.Fatal("zero decision time authorized a grant")
	}
}

func TestGrantWindowsRemainAdditiveAndOmittedWindowsRemainCompatible(t *testing.T) {
	now := grantTime(t, "2030-01-02T03:04:05Z")
	policy := grantWindowPolicy(nil, now)
	policy.Grants = append(policy.Grants, model.Grant{Subject: "operator", Workload: "demo", Operations: []string{"status"}})
	if err := Validate(policy); err != nil {
		t.Fatal(err)
	}
	if !AuthorizeAt(policy, "operator", "", "demo", "status", "", *now) {
		t.Fatal("unbounded matching grant did not remain additive")
	}
}

func TestGrantWindowValidationRejectsZeroAndInvertedBounds(t *testing.T) {
	start := grantTime(t, "2030-01-02T03:04:05Z")
	end := grantTime(t, "2030-01-02T04:04:05Z")
	zero := time.Time{}
	for _, grant := range []model.Grant{
		{Subject: "operator", Workload: "demo", Operations: []string{"status"}, NotBefore: &zero},
		{Subject: "operator", Workload: "demo", Operations: []string{"status"}, ExpiresAt: &zero},
		{Subject: "operator", Workload: "demo", Operations: []string{"status"}, NotBefore: end, ExpiresAt: start},
		{Subject: "operator", Workload: "demo", Operations: []string{"status"}, NotBefore: start, ExpiresAt: start},
	} {
		policy := validPolicy()
		policy.Grants = []model.Grant{grant}
		if Validate(policy) == nil {
			t.Fatal("invalid grant time window accepted")
		}
	}
}

func TestGrantWindowJSONAndYAMLRoundTripsAndSchemaRejectsMalformedTime(t *testing.T) {
	start := grantTime(t, "2030-01-02T03:04:05Z")
	end := grantTime(t, "2030-01-02T04:04:05Z")
	policy := grantWindowPolicy(start, end)
	for _, test := range []struct {
		name   string
		encode func(model.Policy) ([]byte, error)
	}{
		{"JSON", func(policy model.Policy) ([]byte, error) { return json.Marshal(policy) }},
		{"YAML", func(policy model.Policy) ([]byte, error) { return yaml.Marshal(policy) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := test.encode(policy)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := Parse(data, "/private/grant-window-policy")
			if err != nil || !sameGrantWindow(parsed.Grants[0], start, end) {
				t.Fatalf("grant window round trip: policy=%+v err=%v", parsed.Grants, err)
			}
		})
	}
	assertInvalidGrantTimesRejected(t, policy)
}

func grantWindowPolicy(start, end *time.Time) model.Policy {
	policy := validPolicy()
	policy.Grants = []model.Grant{{Subject: "operator", Workload: "demo", Operations: []string{"status"}, NotBefore: start, ExpiresAt: end}}
	return policy
}

func grantTime(t *testing.T, value string) *time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return &parsed
}

func sameGrantWindow(grant model.Grant, start, end *time.Time) bool {
	return grant.NotBefore != nil && grant.ExpiresAt != nil && grant.NotBefore.Equal(*start) && grant.ExpiresAt.Equal(*end)
}

func assertInvalidGrantTimesRejected(t *testing.T, policy model.Policy) {
	t.Helper()
	data := marshalGrantWindowPolicy(t, policy)
	for _, test := range []grantTimeCase{
		{"malformed", func(grant map[string]any) { grant["not_before"] = "not-a-timestamp" }, false},
		{"zero", func(grant map[string]any) { grant["not_before"] = "0001-01-01T00:00:00Z" }, true},
		{"inverted", func(grant map[string]any) { grant["expires_at"] = "2030-01-02T03:04:05Z" }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertInvalidGrantTimeCase(t, data, test)
		})
	}
}

type grantTimeCase struct {
	name        string
	set         func(map[string]any)
	schemaValid bool
}

func marshalGrantWindowPolicy(t *testing.T, policy model.Policy) []byte {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertInvalidGrantTimeCase(t *testing.T, data []byte, test grantTimeCase) {
	t.Helper()
	document := grantTimeDocument(t, data)
	test.set(document["grants"].([]any)[0].(map[string]any))
	assertGrantTimeSchema(t, document, test.schemaValid)
	assertGrantTimeParseRejects(t, document)
}

func grantTimeDocument(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func assertGrantTimeSchema(t *testing.T, document map[string]any, wantValid bool) {
	t.Helper()
	if gotValid := resolvedPolicySchema(t).Validate(document) == nil; gotValid != wantValid {
		t.Fatalf("schema validity = %t, want %t", gotValid, wantValid)
	}
}

func assertGrantTimeParseRejects(t *testing.T, document map[string]any) {
	t.Helper()
	candidate, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(candidate, "/private/grant-window-policy"); err == nil {
		t.Fatal("Parse accepted invalid grant timestamp")
	}
}
