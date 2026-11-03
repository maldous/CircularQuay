package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestDoctorDependenciesClassifiesOnlyDeclaredAbsolutePaths(t *testing.T) {
	dependencies := doctorDependencies([]string{"relative-tool", "/definitely/missing/cquay-tool", "relative-tool"})
	if len(dependencies) != 2 || dependencies[0].State != "invalid_name" || dependencies[1].State != "missing" {
		t.Fatal("doctor dependency classification changed")
	}
}

func TestDoctorMetadataDoesNotReadMissingFile(t *testing.T) {
	metadata := doctorMetadata([]string{"/definitely/missing/cquay-bootstrap"})
	if len(metadata) != 1 || metadata[0].Kind != "protected_file" || metadata[0].State != "missing" {
		t.Fatal("doctor metadata classification changed")
	}
}

func TestDoctorReadinessBlackBox(t *testing.T) {
	directory := privateCLIDirectory(t)
	missing := filepath.Join(directory, "missing.json")
	assertIdentityOnlyDoctor(t, missing)
	assertMissingConfigDoctor(t, missing)
	assertBootstrapReadinessDoctor(t)
}

func TestDoctorRejectsInvalidConfigAndReportsUntrustedMetadata(t *testing.T) {
	directory := privateCLIDirectory(t)
	invalid := filepath.Join(directory, "invalid.json")
	if err := platform.WriteProtected(invalid, []byte(`{"version":1}`)); err != nil {
		t.Fatal(err)
	}
	result := runDoctor(t, invalid)
	if result.code != 2 || !strings.Contains(result.diagnostics, `"error":"invalid_policy"`) {
		t.Fatalf("invalid policy result: %+v", result)
	}

	if runtime.GOOS == "windows" {
		return // chmod cannot safely simulate a foreign writable Windows ACL.
	}
	config, bootstrap, _ := doctorPolicyFixture(t, false)
	if err := os.Chmod(bootstrap, 0644); err != nil {
		t.Fatal(err)
	}
	result = runDoctor(t, config)
	if result.code != 1 || result.diagnostics != "" || !doctorHasDependency(result.report, bootstrap, "untrusted") {
		t.Fatalf("untrusted bootstrap readiness: %+v", result)
	}
}

func TestDoctorFixFormatsProtectedJSONOnly(t *testing.T) {
	config, bootstrap, secret := doctorPolicyFixture(t, true)
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	assertDoctorFix(t, runDoctor(t, config, "--fix"), "formatted")
	after, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) || !bytes.HasSuffix(after, []byte("\n")) {
		t.Fatal("doctor fix did not normalize config JSON")
	}
	if strings.Contains(string(after), secret) {
		t.Fatal("doctor fix copied bootstrap content into config")
	}

	result := runDoctor(t, config, "--fix")
	assertDoctorFix(t, result, "already_formatted")
	if !doctorHasDependency(result.report, bootstrap, "ready") {
		t.Fatalf("formatted policy did not retain protected bootstrap dependency: %+v", result.report)
	}
}

func doctorPolicyFixture(t *testing.T, compact bool) (config, bootstrap, secret string) {
	t.Helper()
	directory := privateCLIDirectory(t)
	bootstrap = filepath.Join(directory, "bootstrap.json")
	secret = "synthetic-doctor-bootstrap-content"
	if err := platform.WriteProtected(bootstrap, []byte(`{"VALUE":"`+secret+`"}`)); err != nil {
		t.Fatal(err)
	}
	policy := model.Policy{
		Version: 1,
		Sources: map[string]model.Source{
			"fixture": {Type: "json", Path: bootstrap},
		},
		Workloads: map[string]model.Workload{
			"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "fixture", Field: "VALUE", AllowOffline: true}}},
		},
	}
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if !compact {
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		data, err = json.MarshalIndent(document, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
	}
	config = filepath.Join(directory, "policy.json")
	if err := platform.WriteProtected(config, data); err != nil {
		t.Fatal(err)
	}
	return config, bootstrap, secret
}

type doctorRun struct {
	code        int
	report      doctorReport
	diagnostics string
	output      string
}

func runDoctor(t *testing.T, config string, args ...string) doctorRun {
	t.Helper()
	var output, diagnostics bytes.Buffer
	code := (Runner{Out: &output, Err: &diagnostics}).Run(context.Background(), append([]string{"--config", config, "doctor"}, args...))
	var report doctorReport
	if output.Len() > 0 {
		if err := json.Unmarshal(output.Bytes(), &report); err != nil {
			t.Fatalf("doctor output is not JSON: %v: %s", err, output.Bytes())
		}
	}
	return doctorRun{code: code, report: report, diagnostics: diagnostics.String(), output: output.String()}
}

func doctorHasDependency(report doctorReport, path, state string) bool {
	for _, dependency := range report.Dependencies {
		if dependency.Path == path && dependency.State == state {
			return true
		}
	}
	return false
}

func assertIdentityOnlyDoctor(t *testing.T, missing string) {
	t.Helper()
	result := runDoctor(t, missing, "--identity-only")
	if result.code != 0 || result.diagnostics != "" || result.report.Config != "not_checked" {
		t.Fatalf("identity-only readiness: %+v", result)
	}
}

func assertMissingConfigDoctor(t *testing.T, missing string) {
	t.Helper()
	result := runDoctor(t, missing)
	if result.code != 1 || result.diagnostics != "" || result.report.Config != "missing_default" {
		t.Fatalf("missing config readiness: %+v", result)
	}
	if _, err := os.Lstat(missing); !os.IsNotExist(err) {
		t.Fatalf("doctor created missing config: %v", err)
	}
}

func assertBootstrapReadinessDoctor(t *testing.T) {
	t.Helper()
	config, bootstrap, secret := doctorPolicyFixture(t, false)
	if err := os.Remove(bootstrap); err != nil {
		t.Fatal(err)
	}
	assertDoctorBootstrapState(t, runDoctor(t, config), bootstrap, secret, "missing", 1)
	if err := platform.WriteProtected(bootstrap, []byte(`{"VALUE":"`+secret+`"}`)); err != nil {
		t.Fatal(err)
	}
	assertDoctorBootstrapState(t, runDoctor(t, config), bootstrap, secret, "ready", 0)
}

func assertDoctorBootstrapState(t *testing.T, result doctorRun, bootstrap, secret, state string, code int) {
	t.Helper()
	if result.code != code || result.diagnostics != "" || !doctorHasDependency(result.report, bootstrap, state) {
		t.Fatalf("bootstrap readiness: expected code=%d state=%s, result=%+v", code, state, result)
	}
	if strings.Contains(result.output, secret) {
		t.Fatal("doctor output exposed bootstrap content")
	}
}

func assertDoctorFix(t *testing.T, result doctorRun, state string) {
	t.Helper()
	if result.code != 0 || result.diagnostics != "" || result.report.Fix != state {
		t.Fatalf("doctor fix %s: %+v", state, result)
	}
}
