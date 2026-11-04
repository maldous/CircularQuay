package assurance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGoFailureMetadataRetainsOnlyBoundPublicLocations(t *testing.T) {
	scope, source := failureMetadataScope()
	capture := parseGoFailureCapture([]string{"test"}, []byte("# github.com/maldous/CircularQuay/internal/example\ninternal/example/value.go:17:3: synthetic failure\nFAIL\tgithub.com/maldous/CircularQuay/internal/example [build failed]\n"), false)
	metadata := summarizeGoFailure(capture, scope, source)
	if metadata.State != "observed" || len(metadata.Packages) != 1 || metadata.Packages[0] != "github.com/maldous/CircularQuay/internal/example" || len(metadata.Locations) != 1 || metadata.Locations[0] != (GoFailureLocation{Path: "internal/example/value.go", Line: 17}) {
		t.Fatalf("unexpected bounded metadata: %#v", metadata)
	}
}

func TestGoFailureMetadataEnforcesInclusiveLineLimit(t *testing.T) {
	scope, source := failureMetadataScope()
	accepted := summarizeGoFailure(parseGoFailureCapture([]string{"vet"}, []byte("internal/example/value.go:1000000:1: synthetic failure\n"), false), scope, source)
	if accepted.State != "observed" || len(accepted.Locations) != 1 || accepted.Locations[0].Line != maxGoFailureLine {
		t.Fatalf("maximum line was not retained: %#v", accepted)
	}
	rejected := summarizeGoFailure(parseGoFailureCapture([]string{"vet"}, []byte("internal/example/value.go:1000001:1: synthetic failure\n"), false), scope, source)
	if rejected.State != "unknown" || len(rejected.Locations) != 0 {
		t.Fatalf("over-limit line was not denied: %#v", rejected)
	}
}

func TestGoFailureMetadataDeniesUntrackedOrUnscopedCandidates(t *testing.T) {
	scope, source := failureMetadataScope()
	for _, capture := range []goFailureCapture{
		{packages: []string{"github.com/maldous/CircularQuay/internal/other"}},
		{locations: []GoFailureLocation{{Path: "internal/other/value.go", Line: 1}}},
		{locations: []GoFailureLocation{{Path: "../internal/example/value.go", Line: 1}}},
	} {
		if metadata := summarizeGoFailure(capture, scope, source); metadata.State != "unknown" || len(metadata.Packages) != 0 || len(metadata.Locations) != 0 {
			t.Fatalf("untrusted candidate disclosed metadata: %#v", metadata)
		}
	}
}

func TestGoFailureMetadataRedactsRawTextAndFailsClosedOnBounds(t *testing.T) {
	scope, source := failureMetadataScope()
	const secret = "synthetic-secret-must-not-appear"
	metadata := summarizeGoFailure(parseGoFailureCapture([]string{"vet"}, []byte(secret), false), scope, source)
	encoded, err := json.Marshal(metadata)
	if err != nil || metadata.State != "unknown" || strings.Contains(string(encoded), secret) {
		t.Fatalf("raw diagnostic leaked or unknown failure accepted: %s %#v", encoded, metadata)
	}
	locations := make([]GoFailureLocation, maxGoFailureLocations+1)
	for index := range locations {
		locations[index] = GoFailureLocation{Path: "internal/example/value.go", Line: index + 1}
	}
	if metadata := summarizeGoFailure(goFailureCapture{locations: locations}, scope, source); metadata.State != "overflow" || len(metadata.Packages) != 0 || len(metadata.Locations) != 0 {
		t.Fatalf("location cap did not fail closed: %#v", metadata)
	}
	if metadata := summarizeGoFailure(goFailureCapture{overflow: true}, scope, source); metadata.State != "overflow" {
		t.Fatalf("output overflow did not remain explicit: %#v", metadata)
	}
	var output strings.Builder
	for index := 0; index <= maxGoFailureLocations; index++ {
		output.WriteString("internal/example/value.go:")
		output.WriteString("1:1: synthetic failure\n")
	}
	capture := parseGoFailureCapture([]string{"test"}, []byte(output.String()), false)
	if !capture.overflow || len(capture.locations) != maxGoFailureLocations {
		t.Fatalf("parser did not stop at the location cap: %#v", capture)
	}
	output.Reset()
	for index := 0; index <= maxGoFailurePackages; index++ {
		output.WriteString("# github.com/maldous/CircularQuay/internal/example\n")
	}
	capture = parseGoFailureCapture([]string{"vet"}, []byte(output.String()), false)
	if !capture.overflow || len(capture.packages) != maxGoFailurePackages {
		t.Fatalf("parser did not stop at the package cap: %#v", capture)
	}
}

func TestExecuteGoSourceCheckAttachesOnlyBoundedFailureMetadata(t *testing.T) {
	scope, source := failureMetadataScope()
	command := func(context.Context, string, []string, string, []string) commandResult {
		return commandResult{exitCode: 1, goFailure: goFailureCapture{packages: []string{"github.com/maldous/CircularQuay/internal/example"}, locations: []GoFailureLocation{{Path: "internal/example/value_test.go", Line: 9}}}}
	}
	check := executeGoSourceCheck(context.Background(), command, "unit_tests", "go", []string{"test"}, ".", nil, scope, source)
	if check.Status != "failed" || check.Code != "command_failed" || check.Failure == nil || check.Failure.State != "observed" || len(check.Failure.Locations) != 1 || check.Failure.Locations[0].Path != "internal/example/value_test.go" {
		t.Fatalf("failure metadata changed command outcome or lost a tracked test location: %#v", check)
	}
}

func failureMetadataScope() (goSourceScope, *SourceIdentity) {
	return goSourceScope{packages: []string{".", "./internal/example"}}, &SourceIdentity{Files: []SourceFile{
		{Path: "main.go"},
		{Path: "internal/example/value.go"},
		{Path: "internal/example/value_test.go"},
	}}
}
