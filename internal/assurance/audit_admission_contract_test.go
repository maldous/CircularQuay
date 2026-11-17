package assurance

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestAuditAdmissionContractOutputFailurePreservesParent(t *testing.T) {
	root := privateAssuranceDirectory(t, "source")
	output := privateAssuranceDirectory(t, "output")
	blocked := filepath.Join(output, "cache")
	if err := platform.WriteProtected(blocked, []byte("owned sentinel")); err != nil {
		t.Fatal(err)
	}
	audit, err := prepareAudit(Options{Source: root, Output: output, Profile: "source"})
	if err == nil || !reflect.DeepEqual(audit, auditRun{}) {
		t.Fatal("blocked output child produced a partial audit")
	}
	data, err := platform.ReadProtected(blocked, true)
	if err != nil || string(data) != "owned sentinel" {
		t.Fatal("failed output preparation replaced the existing child")
	}
}

func TestAuditAdmissionContractFormattingUsesSelectedSibling(t *testing.T) {
	root := privateAssuranceDirectory(t, "source")
	goBinary := filepath.Join(root, "bin", "go")
	files := []string{filepath.Join(root, "a.go")}
	calls := 0
	command := func(_ context.Context, binary string, args []string, directory string, env []string) commandResult {
		calls++
		if binary != filepath.Join(root, "bin", "gofmt") || directory != root || !reflect.DeepEqual(args, append([]string{"-l"}, files...)) || !reflect.DeepEqual(env, []string{"OWNED=fixture"}) {
			t.Fatal("formatter escaped its selected sibling or exact source scope")
		}
		return commandResult{}
	}
	check, err := checkFormatting(context.Background(), command, root, goBinary, []string{"OWNED=fixture"}, files)
	if err != nil || check.Status != "passed" || calls != 1 {
		t.Fatal("finite successful formatter control was not retained")
	}
}

func TestAuditAdmissionContractMetadataAndRulesRefuseBeforeEffect(t *testing.T) {
	if got := metadataCheck("identity", false); got != (Check{Name: "identity", Required: true, Status: "failed", Code: "metadata_unavailable"}) {
		t.Fatalf("missing metadata check = %+v", got)
	}
	auditAdmissionUnboundRules(t)
}

func auditAdmissionUnboundRules(t *testing.T) {
	t.Helper()
	if args, env, ok := gitleaksArguments(VerifiedTool{}, nil); ok || args != nil || env != nil {
		t.Fatal("unbound secret-scan rules produced a command")
	}
	tool := VerifiedTool{DeclaredRules: &ToolRules{Kind: "file", Input: &ToolInput{Path: "unused"}}}
	if args, env, ok := gitleaksArguments(tool, nil); ok || args != nil || env != nil {
		t.Fatal("absent private HOME produced a secret-scan command")
	}
	if args, env, ok := vulnDBArguments([]string{"./owned"}, VerifiedTool{}, nil); ok || args != nil || env != nil {
		t.Fatal("unbound vulnerability snapshot produced a command")
	}
}

func TestAuditAdmissionContractInvalidGoIsNotAParsedWaiver(t *testing.T) {
	root := privateAssuranceDirectory(t, "source")
	name := filepath.Join(root, "invalid.go")
	if err := platform.WriteProtected(name, []byte("not a Go file")); err != nil {
		t.Fatal(err)
	}
	if analyzerWaiverFile(name, "invalid.go", "gosec") {
		t.Fatal("unparseable source was interpreted as an analyzer waiver")
	}
}

func TestAuditAdmissionContractExternalDependencyFailureCannotCallTool(t *testing.T) {
	tools := newQualifiedTools(VerifiedToolManifest{}, func(context.Context, string, []string, string, []string) commandResult {
		t.Fatal("missing dependencies reached a tool callback")
		return commandResult{}
	})
	check := externalCheck(context.Background(), tools, privateAssuranceDirectory(t, "source"), nil, true, externalToolCheck{name: "gosec", id: "gosec"})
	if check != (Check{Name: "gosec", Required: true, Status: "failed", Code: "dependency_digest_mismatch"}) {
		t.Fatalf("missing dependencies = %+v", check)
	}
}

func TestAuditAdmissionContractSelectedAnalyzerWithoutRulesCannotLaunch(t *testing.T) {
	tools := toolExecutionBoundaryTools(t, []string{"git", "go", "gitleaks"}, func(context.Context, string, []string, string, []string) commandResult {
		t.Fatal("unbound analyzer rules reached a callback")
		return commandResult{}
	})
	for index := range tools.manifest.Tools {
		if tools.manifest.Tools[index].ID == "gitleaks" {
			tools.manifest.Tools[index].DeclaredRules = nil
		}
	}
	check := externalCheck(context.Background(), tools, privateAssuranceDirectory(t, "source"), nil, true, externalToolCheck{name: "secret_scan", id: "gitleaks"})
	if check != (Check{Name: "secret_scan", Required: true, Status: "failed", Code: "rules_not_bound"}) {
		t.Fatalf("unbound selected analyzer = %+v", check)
	}
}

func TestAuditAdmissionContractSonarFailureStagesDoNotLaunch(t *testing.T) {
	root := privateAssuranceDirectory(t, "source")
	for _, stage := range []string{"configuration", "token", "metadata"} {
		t.Run(stage, func(t *testing.T) { auditAdmissionSonarRefusal(t, root, stage) })
	}
}

func auditAdmissionSonarRefusal(t *testing.T, root, stage string) {
	t.Helper()
	options := Options{Profile: "enterprise", SonarURL: "https://sonar.invalid", SonarProjectKey: "fixture"}
	report := Report{Commit: strings.Repeat("a", 40)}
	want := "invalid_sonar_configuration"
	if stage == "configuration" {
		options.SonarURL = "not a URL"
	} else {
		options.SonarTokenFile = filepath.Join(root, "missing-token")
		want = "sonar_token_unavailable"
	}
	coverage := filepath.Join(root, "missing-parent", "coverage.out")
	if stage == "metadata" {
		options.SonarTokenFile = filepath.Join(root, "token")
		if err := platform.WriteProtected(options.SonarTokenFile, []byte("synthetic-token")); err != nil {
			t.Fatal(err)
		}
		want = "sonar_metadata_unavailable"
	}
	command := func(context.Context, string, []string, string, []string) commandResult {
		t.Fatal("preflight refusal reached scanner callback")
		return commandResult{}
	}
	addSonar(context.Background(), command, root, nil, options, coverage, &report)
	if !reflect.DeepEqual(report.Checks, []Check{{Name: "sonarqube", Status: "failed", Required: true, Code: want}}) || report.Sonar != nil {
		t.Fatalf("preflight stage %s returned unexpected sanitized state", stage)
	}
}

func TestAuditAdmissionContractMissingSonarTaskIsARequiredFailure(t *testing.T) {
	report := Report{Commit: strings.Repeat("a", 40)}
	verifySonarRun(context.Background(), Options{}, "synthetic", filepath.Join(t.TempDir(), "absent"), true, &report)
	if report.Sonar != nil || !reflect.DeepEqual(report.Checks, []Check{{Name: "sonarqube_clean_counts", Required: true, Status: "failed", Code: "sonar_verification_failed"}}) {
		t.Fatal("missing task metadata did not retain required verification failure")
	}
}

func TestAuditAdmissionContractUnusableHomeDoesNotOverwrite(t *testing.T) {
	root := privateAssuranceDirectory(t, "home")
	name := filepath.Join(root, "gitleaks-empty-ignore")
	if err := os.Mkdir(name, 0700); err != nil {
		t.Fatal(err)
	}
	if path, ok := emptyGitleaksIgnore([]string{"HOME=" + root}); ok || path != "" {
		t.Fatal("an occupied ignore path was accepted")
	}
	if info, err := os.Lstat(name); err != nil || !info.IsDir() {
		t.Fatal("ignore refusal replaced the occupied directory")
	}
}
