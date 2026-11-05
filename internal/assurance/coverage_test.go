package assurance

import (
	"strings"
	"testing"
)

func TestCoverageSummary(t *testing.T) {
	profile := "mode: atomic\n" + modulePrefix + "internal/policy/policy.go:1.1,3.2 3 8\n" + modulePrefix + "internal/policy/policy.go:4.1,5.2 1 0\n" + modulePrefix + "cmd/cq/main.go:1.1,2.1 4 1\n"
	result, err := ParseCoverage(strings.NewReader(profile))
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 8 || result.Covered != 7 || result.Percent != 87.5 || result.Files != 2 {
		t.Fatalf("unexpected summary: %+v", result)
	}
}

func TestRejectInvalidCoverage(t *testing.T) {
	valid := modulePrefix + "internal/policy/policy.go:1.1,2.2 1 1\n"
	cases := map[string]string{
		"empty":               "",
		"header":              "mode: other\n" + valid,
		"malformed":           "mode: set\nnot a coverage block\n",
		"unowned":             "mode: set\nexample.com/private/main.go:1.1,2.2 1 1\n",
		"traversal":           "mode: set\n" + modulePrefix + "../private.go:1.1,2.2 1 1\n",
		"backslash":           "mode: set\n" + modulePrefix + "a\\b.go:1.1,2.2 1 1\n",
		"test_file":           "mode: set\n" + modulePrefix + "a_test.go:1.1,2.2 1 1\n",
		"duplicate":           "mode: set\n" + valid + modulePrefix + "internal/policy/policy.go:1.1,2.2 2 1\n",
		"reversed_line":       "mode: set\n" + modulePrefix + "a.go:3.1,2.2 1 1\n",
		"reversed_column":     "mode: set\n" + modulePrefix + "a.go:1.3,1.2 1 1\n",
		"zero_coordinate":     "mode: set\n" + modulePrefix + "a.go:0.1,2.2 1 1\n",
		"overflow_coordinate": "mode: set\n" + modulePrefix + "a.go:999999999999.1,2.2 1 1\n",
		"overflow_statements": "mode: set\n" + modulePrefix + "a.go:1.1,2.2 99999999999999999999 1\n",
		"large_statements":    "mode: set\n" + modulePrefix + "a.go:1.1,2.2 10000001 1\n",
		"overflow_execution":  "mode: set\n" + modulePrefix + "a.go:1.1,2.2 1 99999999999999999999\n",
		"no_statements":       "mode: set\n" + modulePrefix + "a.go:1.1,2.2 0 0\n",
	}
	for name, profile := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseCoverage(strings.NewReader(profile)); err == nil {
				t.Fatal("invalid profile accepted")
			}
		})
	}
}

func TestCoverageModes(t *testing.T) {
	for _, mode := range []string{"set", "count", "atomic"} {
		profile := "mode: " + mode + "\n" + modulePrefix + "a.go:1.1,1.1 1 18446744073709551615\n"
		if _, err := ParseCoverage(strings.NewReader(profile)); err != nil {
			t.Fatal(err)
		}
	}
}

func FuzzCoverage(f *testing.F) {
	f.Add("mode: set\n" + modulePrefix + "a.go:1.1,2.2 1 1\n")
	f.Add("mode: set\n../../credentials:1.1,2.2 1 1\n")
	f.Fuzz(func(t *testing.T, input string) {
		result, err := ParseCoverage(strings.NewReader(input))
		if err == nil && (result.Total <= 0 || result.Covered < 0 || result.Covered > result.Total || result.Percent < 0 || result.Percent > 100) {
			t.Fatal("invalid accepted summary")
		}
	})
}

func TestCoverageMergesRepeatedWholeModuleBlocks(t *testing.T) {
	profile := "mode: atomic\n" + modulePrefix + "a.go:1.1,2.1 3 0\n" + modulePrefix + "a.go:1.1,2.1 3 8\n" + modulePrefix + "a.go:1.1,2.1 3 2\n" + modulePrefix + "a.go:3.1,4.1 1 0\n"
	result, err := ParseCoverage(strings.NewReader(profile))
	if err != nil || result.Total != 4 || result.Covered != 3 || result.Percent != 75 {
		t.Fatalf("cross-package coverage did not preserve denominator: %+v, %v", result, err)
	}
}
