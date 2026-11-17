package assurance

import (
	"reflect"
	"strings"
	"testing"
)

func TestGoFailureBoundaryContractOverflowAndScannerRefuseRawData(t *testing.T) {
	got := parseGoFailureCapture([]string{"test"}, []byte("must not be retained"), true)
	if !reflect.DeepEqual(got, goFailureCapture{overflow: true}) {
		t.Fatal("overflow retained unbounded failure data")
	}
	got = scanGoFailureOutput(goFailureCapture{}, []byte(strings.Repeat("x", 65<<10)))
	if !got.unknown || len(got.packages) != 0 || len(got.locations) != 0 {
		t.Fatal("oversized diagnostic line became trusted failure metadata")
	}
}

func TestGoFailureBoundaryContractAbsentIdentityCannotAuthorizeLocations(t *testing.T) {
	if allowed, ok := allowedGoFailurePaths(nil); ok || allowed != nil {
		t.Fatal("nil identity supplied an allowed-path set")
	}
	if got, ok := boundFailureLocations([]GoFailureLocation{{Path: "owned.go", Line: 1}}, nil); ok || got != nil {
		t.Fatal("nil identity authorized a failure location")
	}
	for _, line := range []int{0, maxGoFailureLine + 1} {
		if allowedFailureLocation(GoFailureLocation{Path: "owned.go", Line: line}, map[string]bool{"owned.go": true}) {
			t.Fatal("out-of-range failure location was accepted")
		}
	}
}

func TestGoFailureBoundaryContractLocationsAreUniqueAndDeterministic(t *testing.T) {
	candidates := []GoFailureLocation{{Path: "b.go", Line: 3}, {Path: "a.go", Line: 4}, {Path: "a.go", Line: 2}, {Path: "a.go", Line: 2}}
	before := append([]GoFailureLocation(nil), candidates...)
	got, ok := boundAllowedFailureLocations(candidates, map[string]bool{"a.go": true, "b.go": true})
	want := []GoFailureLocation{{Path: "a.go", Line: 2}, {Path: "a.go", Line: 4}, {Path: "b.go", Line: 3}}
	if !ok || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(candidates, before) {
		t.Fatal("failure locations were not sorted, deduplicated, and input-preserving")
	}
}
