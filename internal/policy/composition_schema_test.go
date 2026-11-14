package policy

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func compositionDocument() map[string]any {
	return map[string]any{
		"version": 1,
		"composition": map[string]any{
			"conflicts": "reject",
			"includes": []any{map[string]any{
				"id": "personal", "namespace": "personal", "sha256": strings.Repeat("a", 64),
				"file": "personal.json", "base_directory": "private", "visibility": "local",
			}},
		},
	}
}

func compositionInclude(document map[string]any) map[string]any {
	return document["composition"].(map[string]any)["includes"].([]any)[0].(map[string]any)
}

func TestCompositionSchemaKeepsLocationsPinnedAndClosed(t *testing.T) {
	schema := resolvedPolicySchema(t)
	if err := schema.Validate(compositionDocument()); err != nil {
		t.Fatalf("explicit local composition rejected: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"unknown composition field", func(document map[string]any) { document["composition"].(map[string]any)["trust_me"] = true }},
		{"unpinned include", func(document map[string]any) { delete(compositionInclude(document), "sha256") }},
		{"unknown include field", func(document map[string]any) { compositionInclude(document)["headers"] = map[string]any{} }},
		{"unbounded local refresh", func(document map[string]any) { compositionInclude(document)["refresh"] = "online" }},
		{"shared local document", func(document map[string]any) { compositionInclude(document)["visibility"] = "shared" }},
		{"last writer wins", func(document map[string]any) { document["composition"].(map[string]any)["conflicts"] = "last" }},
		{"empty includes", func(document map[string]any) { document["composition"].(map[string]any)["includes"] = []any{} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := compositionDocument()
			test.edit(document)
			if err := schema.Validate(document); err == nil {
				t.Fatal("schema accepted an unsafe composition shape")
			}
		})
	}
}

func repositoryCompositionDocument() map[string]any {
	document := compositionDocument()
	include := compositionInclude(document)
	delete(include, "file")
	include["repository"] = map[string]any{
		"url": "https://github.com/example/policies", "directory": "repository",
		"ref": "refs/heads/configuration", "allowed_signers_file": "signers",
		"initial_commit": strings.Repeat("a", 40),
	}
	include["path"] = "department/policy.json"
	include["commit"] = strings.Repeat("b", 40)
	include["visibility"] = "shared"
	include["refresh"] = "online"
	return document
}

func TestCompositionSchemaRequiresExplicitRepositoryCacheContract(t *testing.T) {
	schema := resolvedPolicySchema(t)
	if err := schema.Validate(repositoryCompositionDocument()); err != nil {
		t.Fatalf("signed repository descriptor rejected: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"missing commit", func(include map[string]any) { delete(include, "commit") }},
		{"missing refresh", func(include map[string]any) { delete(include, "refresh") }},
		{"offline without deadline", func(include map[string]any) { include["refresh"] = "offline" }},
		{"online with fallback", func(include map[string]any) { include["cache_expires_at"] = "2030-01-01T00:00:00Z" }},
		{"repository traversal", func(include map[string]any) { include["path"] = "../policy.json" }},
		{"empty repository", func(include map[string]any) { include["repository"] = map[string]any{} }},
		{"ambiguous location", func(include map[string]any) { include["file"] = "personal.json" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := repositoryCompositionDocument()
			test.edit(compositionInclude(document))
			if err := schema.Validate(document); err == nil {
				t.Fatal("schema accepted an ambiguous repository/cache boundary")
			}
		})
	}
}

func TestUnresolvedCompositionCannotBecomeRuntimePolicy(t *testing.T) {
	data, err := json.Marshal(compositionDocument())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeDocument(data, "/private/master.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(decoded); err == nil {
		t.Fatal("intermediate composition passed runtime validation")
	}
	if _, err = Parse(data, "/private/master.json"); !fault.Is(err, fault.Unsupported) {
		t.Fatal("single-document parser silently accepted unresolved composition")
	}
}
