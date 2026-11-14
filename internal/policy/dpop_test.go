package policy

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestNamedDPoPSenderConstraintValidation(t *testing.T) {
	valid := namedIdentityPolicy()
	valid.MCP.Authorities["workforce"] = withDPoP(valid.MCP.Authorities["workforce"])
	if err := Validate(valid); err != nil {
		t.Fatalf("valid DPoP policy rejected: %v", err)
	}
	for name, edit := range map[string]func(*model.SenderConstraint){
		"type":    func(c *model.SenderConstraint) { c.Type = "mtls" },
		"max age": func(c *model.SenderConstraint) { c.ProofMaxAgeSeconds = 0 },
		"skew":    func(c *model.SenderConstraint) { c.ClockSkewSeconds = 31 },
		"nonce":   func(c *model.SenderConstraint) { c.Nonce = "optional" },
		"replay":  func(c *model.SenderConstraint) { c.ReplayStore = "shared" },
	} {
		t.Run(name, func(t *testing.T) {
			p := namedIdentityPolicy()
			a := withDPoP(p.MCP.Authorities["workforce"])
			edit(a.SenderConstraint)
			p.MCP.Authorities["workforce"] = a
			if err := Validate(p); err == nil {
				t.Fatal("invalid DPoP policy accepted")
			}
		})
	}
}

func withDPoP(authority model.IdentityAuthority) model.IdentityAuthority {
	authority.SenderConstraint = &model.SenderConstraint{Type: "dpop", ProofMaxAgeSeconds: 60, ClockSkewSeconds: 5, Nonce: "required", ReplayStore: "local_persistent"}
	return authority
}

func TestDPoPClockSkewDefaultsToZero(t *testing.T) {
	for _, test := range []struct {
		name       string
		clockSkew  any
		schemaOkay bool
		runtimeOK  bool
	}{
		{name: "omitted", schemaOkay: true, runtimeOK: true},
		{name: "zero", clockSkew: 0, schemaOkay: true, runtimeOK: true},
		{name: "maximum", clockSkew: 30, schemaOkay: true, runtimeOK: true},
		{name: "negative", clockSkew: -1},
		{name: "over maximum", clockSkew: 31},
		{name: "unknown member", clockSkew: 0},
		{name: "null", clockSkew: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := dpopClockSkewDocument(t, test.name, test.clockSkew)
			policy := assertDPoPPolicyAcceptance(t, document, test.schemaOkay, test.runtimeOK)
			if test.name == "omitted" && policy.MCP.Authorities["workforce"].SenderConstraint.ClockSkewSeconds != 0 {
				t.Fatal("omitted clock skew did not default to zero")
			}
		})
	}
}

func dpopClockSkewDocument(t *testing.T, name string, value any) map[string]any {
	t.Helper()
	document := dpopPolicyDocument(t)
	constraint := senderConstraintDocument(document)
	if name == "omitted" {
		delete(constraint, "clock_skew_seconds")
	} else {
		constraint["clock_skew_seconds"] = value
	}
	if name == "unknown member" {
		constraint["unexpected"] = true
	}
	return document
}

func TestDPoPSchemaBoundsClockSkewByProofAge(t *testing.T) {
	for age := 1; age < 30; age++ {
		t.Run(strconv.Itoa(age), func(t *testing.T) {
			document := dpopPolicyDocument(t)
			constraint := senderConstraintDocument(document)
			constraint["proof_max_age_seconds"] = age
			constraint["clock_skew_seconds"] = age + 1
			assertSchemaAndParseReject(t, document)
			constraint["clock_skew_seconds"] = age
			assertSchemaAndParseAccept(t, document)
		})
	}
}

func assertDPoPPolicyAcceptance(t *testing.T, document map[string]any, schemaOkay, runtimeOkay bool) model.Policy {
	t.Helper()
	schemaErr := resolvedPolicySchema(t).Validate(document)
	if (schemaErr == nil) != schemaOkay {
		t.Fatalf("schema acceptance = %t, want %t: %v", schemaErr == nil, schemaOkay, schemaErr)
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	parsed, runtimeErr := Parse(data, "/private/policy.json")
	if (runtimeErr == nil) != runtimeOkay {
		t.Fatalf("runtime acceptance = %t, want %t: %v", runtimeErr == nil, runtimeOkay, runtimeErr)
	}
	return parsed
}

func dpopPolicyDocument(t *testing.T) map[string]any {
	t.Helper()
	policy := namedIdentityPolicy()
	authority := withDPoP(policy.MCP.Authorities["workforce"])
	policy.MCP.Authorities["workforce"] = authority
	return policyJSONValue(t, policy)
}

func senderConstraintDocument(document map[string]any) map[string]any {
	return document["mcp"].(map[string]any)["authorities"].(map[string]any)["workforce"].(map[string]any)["sender_constraint"].(map[string]any)
}
