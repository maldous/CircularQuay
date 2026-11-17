package repository

import (
	"strings"
	"testing"
)

func TestGitBoundaryContractRejectsEmptyInvocation(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
	}{
		{name: "nil"},
		{name: "empty", arguments: []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if validGitInvocation(test.arguments) {
				t.Fatal("an empty invocation must not be admitted as a Git command")
			}
		})
	}
}

func TestGitBoundaryContractValidatesStandardArguments(t *testing.T) {
	// These are data-only invocations of the admission predicate. In particular,
	// an allowlisted command must not bypass control-character checks merely
	// because its command name is recognized. No transport or runner is called.
	for _, test := range []struct {
		name      string
		arguments []string
		valid     bool
	}{
		{
			name:      "ordinary_fetch",
			arguments: []string{"fetch", "https://example.invalid/config.git", "refs/heads/reviewed"},
			valid:     true,
		},
		{
			name:      "ordinary_ref_inspection",
			arguments: []string{"rev-parse", "refs/cq/candidate"},
			valid:     true,
		},
		{
			name:      "nul_in_object_argument",
			arguments: []string{"cat-file", "blob", "synthetic\x00value"},
		},
		{
			name:      "carriage_return_in_transport_argument",
			arguments: []string{"fetch", "https://example.invalid/config.git\r", "refs/heads/reviewed"},
		},
		{
			name:      "newline_in_ref_argument",
			arguments: []string{"ls-tree", "refs/cq/candidate\n--unexpected"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validGitInvocation(test.arguments); got != test.valid {
				t.Fatalf("invocation admission = %t, want %t", got, test.valid)
			}
		})
	}
}

func TestGitBoundaryContractRequiresHexForMatchingObjectIDs(t *testing.T) {
	// A matching response and expected value are not sufficient if both carry
	// the same malformed identifier. Keep equality true in the negative cases
	// so that the hexadecimal-shape guard, rather than a mismatch, must refuse.
	for _, test := range []struct {
		name  string
		id    string
		valid bool
	}{
		{name: "lowercase_sha1", id: strings.Repeat("a", 40), valid: true},
		{name: "decimal_sha1", id: strings.Repeat("1", 40), valid: true},
		{name: "lowercase_sha256", id: strings.Repeat("f", 64), valid: true},
		{name: "nonhex_letter", id: strings.Repeat("a", 39) + "g"},
		{name: "uppercase_letter", id: strings.Repeat("a", 39) + "A"},
		{name: "path_separator", id: strings.Repeat("a", 39) + "/"},
		{name: "embedded_nul", id: strings.Repeat("a", 39) + "\x00"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := matchingObjectID(test.id+"\n", test.id); got != test.valid {
				t.Fatalf("matching identifier admission = %t, want %t", got, test.valid)
			}
		})
	}
}
