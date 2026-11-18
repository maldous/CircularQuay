package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/maldous/CircularQuay/internal/model"
)

// TestCanonicalSchemaRejectsStructuralDrift keeps the published JSON Schema
// honest about document shape. Runtime authorization, filesystem protection,
// endpoint trust, and provider semantics are deliberately checked by Load and
// Validate; JSON Schema alone cannot establish those security properties.
func TestCanonicalSchemaRejectsStructuralDrift(t *testing.T) {
	resolved := resolvedPolicySchema(t)
	valid := policyJSONValue(t, validPolicy())
	if err := resolved.Validate(valid); err != nil {
		t.Fatalf("canonical policy shape rejected: %v", err)
	}
	for _, mutation := range []func(map[string]any){
		func(value map[string]any) { value["unexpected"] = true },
		func(value map[string]any) { value["version"] = 2 },
		func(value map[string]any) {
			value["sources"].(map[string]any)["fixture"].(map[string]any)["unexpected"] = true
		},
		func(value map[string]any) {
			value["workloads"].(map[string]any)["demo"].(map[string]any)["bindings"].([]any)[0].(map[string]any)["unexpected"] = true
		},
	} {
		candidate := policyJSONValue(t, validPolicy())
		mutation(candidate)
		if err := resolved.Validate(candidate); err == nil {
			t.Fatal("schema accepted structural drift")
		}
	}
}

// TestCanonicalSchemaAndValidatorRejectRepresentableContractViolations keeps
// structural policy rules aligned with the pure Go validator. Rules involving
// filesystem ownership, network trust reachability, and map cross-references
// remain Go responsibilities and are covered separately below.
func TestCanonicalSchemaAndValidatorRejectRepresentableContractViolations(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"source identifier", func(value map[string]any) {
			value["sources"].(map[string]any)["bad/name"] = value["sources"].(map[string]any)["fixture"]
		}},
		{"empty local path", func(value map[string]any) { sourceDocument(value)["path"] = "" }},
		{"provider identity", func(value map[string]any) {
			value["sources"].(map[string]any)["fixture"] = awsSourceDocument("managed")
		}},
		{"provider required token", func(value map[string]any) { value["sources"].(map[string]any)["fixture"] = sopsTokenDocument() }},
		{"selector", func(value map[string]any) { bindingDocument(value)["field"] = "*" }},
		{"partial snapshot", func(value map[string]any) {
			workloadDocument(value)["snapshot"] = map[string]any{"path": "/private/snapshot"}
		}},
		{"target contract", addInvalidFileTarget},
		{"grant disclosure", func(value map[string]any) {
			value["grants"] = []any{map[string]any{"subject": "operator", "workload": "demo", "operations": []any{"read"}}}
		}},
		{"grant operation", func(value map[string]any) {
			value["grants"] = []any{map[string]any{"subject": "operator", "workload": "demo", "operations": []any{"cancel"}}}
		}},
		{"mcp authority tuple", func(value map[string]any) { value["mcp"] = map[string]any{"issuer": "https://issuer.example.test"} }},
		{"repository completeness", func(value map[string]any) {
			value["repository"] = map[string]any{"url": "https://repository.example.test/policy"}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := policyJSONValue(t, validPolicy())
			test.edit(candidate)
			assertSchemaAndValidatorReject(t, candidate)
		})
	}
}

// TestRawSchemaAndParseAgreeOnBoundarySyntax covers syntax that is meaningful
// before Parse binds policy-relative filesystem references. Cross-references,
// ownership, and remote trust are intentionally tested as semantic rules.
func TestRawSchemaAndParseAgreeOnBoundarySyntax(t *testing.T) {
	valid := relativePolicyDocument(t)
	assertSchemaAndParseAccept(t, valid)
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"selector dot", func(value map[string]any) { bindingDocument(value)["field"] = "." }},
		{"selector parent", func(value map[string]any) { bindingDocument(value)["field"] = "../value" }},
		{"text carriage return", func(value map[string]any) { workloadDocument(value)["directory"] = "bad\rpath" }},
		{"text null", func(value map[string]any) { sourceDocument(value)["path"] = "bad\x00path" }},
		{"bare snapshot executable", func(value map[string]any) { workloadDocument(value)["snapshot"].(map[string]any)["sops"] = "sops" }},
		{"grant keys without read", func(value map[string]any) {
			value["grants"] = []any{map[string]any{"subject": "operator", "workload": "demo", "operations": []any{"run"}, "keys": []any{"DEMO_MESSAGE"}}}
		}},
		{"malformed https port", func(value map[string]any) {
			value["mcp"].(map[string]any)["issuer"] = "https://issuer.example.test:bad"
		}},
		{"repository parent reference", func(value map[string]any) { value["repository"].(map[string]any)["ref"] = "refs/heads/../main" }},
		{"repository lock reference", func(value map[string]any) { value["repository"].(map[string]any)["ref"] = "refs/heads/main.lock" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := relativePolicyDocument(t)
			test.edit(candidate)
			assertSchemaAndParseReject(t, candidate)
		})
	}
}

func relativePolicyDocument(t *testing.T) map[string]any {
	t.Helper()
	policy := validPolicy()
	changeWorkload(&policy, func(workload *model.Workload) {
		workload.Mode = "hybrid"
		workload.Directory = "work"
		workload.Bindings[0].AllowPublish = true
		workload.Snapshot = model.Snapshot{
			Path: "snapshot.json", Recipients: []string{"age1" + strings.Repeat("q", 58)},
			IdentityFile: "identity.txt", SOPS: "./bin/sops",
		}
		workload.Targets = map[string]model.Target{"sink": {Type: "file", Path: "output.json", AcceptSnapshot: true}}
	})
	policy.Repository = model.Repository{
		URL: "https://repository.example.test/policy", Directory: "repository", Ref: "refs/heads/main",
		AllowedSignersFile: "signers", InitialCommit: strings.Repeat("a", 40), TokenFile: "token",
	}
	policy.MCP = model.MCP{
		Issuer: "https://issuer.example.test", JWKSURL: "https://issuer.example.test/keys",
		Audience: "http://127.0.0.1:8787/mcp", Origins: []string{"http://localhost:8787"},
	}
	return policyJSONValue(t, policy)
}

func assertSchemaAndParseAccept(t *testing.T, candidate map[string]any) {
	t.Helper()
	if err := resolvedPolicySchema(t).Validate(candidate); err != nil {
		t.Fatalf("schema rejected policy accepted by Parse: %v", err)
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, "/private/policy.json"); err != nil {
		t.Fatalf("Parse rejected schema-valid policy: %v", err)
	}
}

func assertSchemaAndParseReject(t *testing.T, candidate map[string]any) {
	t.Helper()
	if err := resolvedPolicySchema(t).Validate(candidate); err == nil {
		t.Fatal("schema accepted invalid policy syntax")
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, "/private/policy.json"); err == nil {
		t.Fatal("Parse accepted invalid policy syntax")
	}
}

func sourceDocument(value map[string]any) map[string]any {
	return value["sources"].(map[string]any)["fixture"].(map[string]any)
}

func workloadDocument(value map[string]any) map[string]any {
	return value["workloads"].(map[string]any)["demo"].(map[string]any)
}

func bindingDocument(value map[string]any) map[string]any {
	return workloadDocument(value)["bindings"].([]any)[0].(map[string]any)
}

func awsSourceDocument(identity string) map[string]any {
	return map[string]any{
		"type": "aws-sm", "account": "123456789012", "region": "ap-southeast-2",
		"resources": []any{"fixture"}, "identity": identity,
	}
}

func sopsTokenDocument() map[string]any {
	return map[string]any{
		"type": "sops", "path": "/fixtures/values.sops.json", "sops": "/usr/bin/sops",
		"identity": "token_file", "token_file": "/private/token",
	}
}

func addInvalidFileTarget(value map[string]any) {
	bindingDocument(value)["allow_publish"] = true
	workloadDocument(value)["targets"] = map[string]any{
		"sink": map[string]any{"type": "file", "path": "/private/output", "token_file": "/private/token", "accept_snapshot": true},
	}
}

func assertSchemaAndValidatorReject(t *testing.T, candidate map[string]any) {
	t.Helper()
	if err := resolvedPolicySchema(t).Validate(candidate); err == nil {
		t.Fatal("schema accepted a representable contract violation")
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	var decoded model.Policy
	if err = decodePolicy(data, &decoded); err != nil {
		t.Fatalf("strict Go decoder rejected a model-shaped document: %v", err)
	}
	if err = Validate(decoded); err == nil {
		t.Fatal("Go semantic validation accepted a schema violation")
	}
}

func TestLinuxDeploymentReferenceMatchesSchemaAndSemanticPolicy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "linux-deployment.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if err = resolvedPolicySchema(t).Validate(document); err != nil {
		t.Fatalf("reference violates canonical JSON Schema: %v", err)
	}
	loaded := loadReferencePolicy(t, data)
	if err = Validate(loaded); err != nil {
		t.Fatalf("reference violates semantic policy validation: %v", err)
	}
	assertLinuxReferenceIntent(t, loaded)
}

func loadReferencePolicy(t *testing.T, data []byte) model.Policy {
	t.Helper()
	directory := privatePolicyDirectory(t)
	path := filepath.Join(directory, "linux-deployment.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	policy, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func assertLinuxReferenceIntent(t *testing.T, policy model.Policy) {
	t.Helper()
	workload, ok := policy.Workloads["reference-api"]
	if !ok || workload.Mode != "hybrid" || len(workload.Bindings) != 1 || len(workload.Targets) != 1 {
		t.Fatal("reference deployment lost its declared lifecycle boundary")
	}
	source, ok := policy.Sources["service-sops"]
	if !ok || source.Type != "sops" || source.AgeIdentityFile == "" || workload.Bindings[0].AllowPublish == false {
		t.Fatal("reference deployment lost explicit source or publication authority")
	}
}

func TestSemanticPolicyValidationComplementsJSONSchema(t *testing.T) {
	resolved := resolvedPolicySchema(t)
	candidate := policyJSONValue(t, validPolicy())
	candidate["workloads"].(map[string]any)["demo"].(map[string]any)["bindings"].([]any)[0].(map[string]any)["source"] = "missing"
	if err := resolved.Validate(candidate); err != nil {
		t.Fatalf("schema unexpectedly rejected structurally valid policy: %v", err)
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	var decoded model.Policy
	if err = decodePolicy(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err = Validate(decoded); err == nil {
		t.Fatal("semantic cross-reference validation was replaced by JSON Schema")
	}
}

func TestParseRejectsCrossOriginMCPKeySetAfterSchemaValidation(t *testing.T) {
	candidate := policyJSONValue(t, validPolicy())
	candidate["mcp"] = map[string]any{
		"issuer": "https://issuer.example.test", "audience": "https://mcp.example.test/resource",
		"jwks_url": "https://keys.example.test/jwks",
	}
	if err := resolvedPolicySchema(t).Validate(candidate); err != nil {
		t.Fatalf("schema should leave origin association to policy validation: %v", err)
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, "/private/policy.json"); err == nil {
		t.Fatal("cross-origin key set accepted")
	}
}

func resolvedPolicySchema(t *testing.T) *jsonschema.Resolved {
	t.Helper()
	var schema jsonschema.Schema
	if err := json.Unmarshal(Schema, &schema); err != nil {
		t.Fatal(err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("canonical schema is invalid: %v", err)
	}
	return resolved
}

func policyJSONValue(t *testing.T, policy model.Policy) map[string]any {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	value := make(map[string]any)
	if err = json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
