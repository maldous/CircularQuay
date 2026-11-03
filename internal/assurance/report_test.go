package assurance

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestFinishDoesNotClaimQualification(t *testing.T) {
	for _, test := range []struct {
		name   string
		checks []Check
		passed bool
	}{
		{"passed", []Check{{Required: true, Status: "passed"}}, true},
		{"missing_required", []Check{{Required: true, Status: "not_run"}}, false},
		{"failed", []Check{{Required: true, Status: "failed"}}, false},
		{"optional", []Check{{Required: false, Status: "not_run"}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := Report{Checks: test.checks}
			report.finish()
			if report.Passed != test.passed || report.ProductionQualified || report.CompletedAt.IsZero() {
				t.Fatal("invalid assurance claim")
			}
		})
	}
}

func TestWritePrivateAtomicReport(t *testing.T) {
	directory := privateAssuranceDirectory(t, "report")
	input := Report{Version: 1, Profile: "source", Checks: []Check{{Name: "fixture", Status: "passed"}}}
	assertPrivateReportWritten(t, directory, input)
	assertInvalidReportRejected(t, directory, input)
	assertReportActivationFailure(t, directory, input)
}

func assertPrivateReportWritten(t *testing.T, directory string, input Report) {
	t.Helper()
	if err := WriteReport(directory, input); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got Report
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || got.Profile != "source" {
		t.Fatal("report lost metadata")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatal("staging file left behind")
	}
}

func assertInvalidReportRejected(t *testing.T, directory string, input Report) {
	t.Helper()
	if err := WriteReport(directory, Report{Coverage: Coverage{Percent: math.NaN()}}); err == nil {
		t.Fatal("non-json report accepted")
	}
	if err := WriteReport(filepath.Join(directory, "missing"), input); err == nil {
		t.Fatal("invalid directory accepted")
	}
}

func assertReportActivationFailure(t *testing.T, directory string, input Report) {
	t.Helper()
	if err := os.Remove(filepath.Join(directory, "report.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "report.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := WriteReport(directory, input); err == nil {
		t.Fatal("activation failure ignored")
	}
}
