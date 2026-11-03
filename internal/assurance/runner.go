package assurance

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

// Options declares execution inputs; no secret is accepted in command arguments.
type Options struct {
	Source          string
	Output          string
	Profile         string
	GoBinary        string
	SonarURL        string
	SonarTokenFile  string
	SonarProjectKey string
}

type commandResult struct {
	output   []byte
	exitCode int
	missing  bool
}
type commandFunc func(context.Context, string, []string, string, []string) commandResult

const auditCommandWaitDelay = 250 * time.Millisecond

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
var goVersionPattern = regexp.MustCompile(`^go version (go[0-9]+\.[0-9]+(?:\.[0-9]+)?) [a-z0-9]+/[a-z0-9]+\s*$`)
var projectPattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,200}$`)

// Run executes a local source audit and creates a sanitized evidence bundle.
func Run(ctx context.Context, options Options) (Report, error) { return run(ctx, options, runCommand) }

func run(ctx context.Context, options Options, command commandFunc) (Report, error) {
	audit, err := prepareAudit(options)
	if err != nil {
		return Report{}, err
	}
	return audit.run(ctx, command)
}

type auditRun struct {
	root        string
	output      string
	options     Options
	report      Report
	environment []string
}

func prepareAudit(options Options) (auditRun, error) {
	if options.Profile != "source" && options.Profile != "enterprise" {
		return auditRun{}, errors.New("unknown audit profile")
	}
	root, err := filepath.Abs(options.Source)
	if err != nil {
		return auditRun{}, errors.New("invalid source directory")
	}
	output, err := prepareOutput(root, options.Output)
	if err != nil {
		return auditRun{}, err
	}
	if options.GoBinary == "" {
		options.GoBinary = "go"
	}
	report := Report{Version: 1, Profile: options.Profile, Platform: runtime.GOOS + "/" + runtime.GOARCH, StartedAt: time.Now().UTC(), Limitations: []string{
		"Automated source analysis is not security certification.",
		"Coverage is measured for the current native build, not unexecuted platforms.",
		"Independent review, real reboot and live provider qualification are separate evidence.",
		"Source manifests detect ordinary drift but are not an atomic snapshot or OS sandbox.",
	}}
	return auditRun{root: root, output: output, options: options, report: report, environment: auditEnvironment(filepath.Join(output, "home"))}, nil
}

func (audit auditRun) run(ctx context.Context, command commandFunc) (Report, error) {
	identity, err := bindSource(ctx, runCommand, audit.root, audit.output, audit.environment)
	if err != nil {
		audit.report.Checks = append(audit.report.Checks, failedSourceIdentityCheck("source_identity_before_checks"))
		audit.report.finish()
		return writeFailedReport(audit.output, audit.report)
	}
	audit.report.Source = &identity
	recordSourceIdentity(ctx, command, audit.root, audit.options.GoBinary, audit.environment, &audit.report)
	audit.report.Checks = append(audit.report.Checks, repositoryCheck(audit.root))
	scope, err := sourceGoScope(audit.root, &identity)
	if err != nil {
		audit.report.Checks = append(audit.report.Checks, Check{Name: "source_go_scope", Required: true, Status: "failed", Code: "source_scope_missing"})
		audit.report.finish()
		return writeFailedReport(audit.output, audit.report)
	}
	format, err := checkFormatting(ctx, command, audit.root, audit.options.GoBinary, audit.environment, scope.files)
	if err != nil {
		return audit.report, err
	}
	audit.recordChecks(ctx, command, format, scope)
	audit.recordAfterIdentity(ctx, identity)
	audit.report.finish()
	if err = WriteReport(audit.output, audit.report); err != nil {
		return audit.report, err
	}
	return audit.report, nil
}

func (audit *auditRun) recordChecks(ctx context.Context, command commandFunc, format Check, scope goSourceScope) {
	audit.report.Checks = append(audit.report.Checks, format)
	coveragePath := filepath.Join(audit.output, "coverage.out")
	testArguments := append([]string{"test", "-count=1", "-covermode=atomic", "-coverpkg=" + strings.Join(scope.coverage, ","), "-coverprofile=" + coveragePath}, scope.packages...)
	audit.report.Checks = append(audit.report.Checks, command.execute(ctx, "unit_tests", true, audit.options.GoBinary, testArguments, audit.root, audit.environment))
	audit.report.Checks = append(audit.report.Checks, coverageCheck(coveragePath, &audit.report.Coverage))
	audit.report.Checks = append(audit.report.Checks, command.execute(ctx, "go_vet", true, audit.options.GoBinary, append([]string{"vet"}, scope.packages...), audit.root, audit.environment))
	addExternalChecks(ctx, command, audit.root, audit.environment, audit.options.Profile == "enterprise", scope, &audit.report)
	addSonar(ctx, command, audit.root, audit.environment, audit.options, coveragePath, &audit.report)
	audit.report.Checks = append(audit.report.Checks, Check{Name: "native_reboot_and_restore", Status: "not_run", Code: "requires_qualified_native_environment"}, Check{Name: "independent_review", Status: "not_run", Code: "requires_independent_reviewer"})
}

func (audit *auditRun) recordAfterIdentity(ctx context.Context, before SourceIdentity) {
	after, err := bindSource(ctx, runCommand, audit.root, audit.output, audit.environment)
	if err != nil || after.ManifestSHA256 != before.ManifestSHA256 || after.Tree != before.Tree || after.Commit != before.Commit {
		audit.report.Checks = append(audit.report.Checks, failedSourceIdentityCheck("source_identity_after_checks"))
		return
	}
	audit.report.Checks = append(audit.report.Checks, metadataCheck("source_identity_after_checks", true))
}

func writeFailedReport(output string, report Report) (Report, error) {
	if err := WriteReport(output, report); err != nil {
		return report, err
	}
	return report, nil
}

func failedSourceIdentityCheck(name string) Check {
	return Check{Name: name, Required: true, Status: "failed", Code: "source_identity_invalid"}
}

func recordSourceIdentity(ctx context.Context, command commandFunc, root, goBinary string, environment []string, report *Report) {
	if report.Source != nil {
		report.Commit = report.Source.Commit
	}
	version := command(ctx, goBinary, []string{"version"}, root, environment)
	if found := goVersionPattern.FindSubmatch(version.output); len(found) > 1 {
		report.GoVersion = string(found[1])
	}
	report.Checks = append(report.Checks, metadataCheck("source_revision", report.Commit != ""), metadataCheck("go_toolchain", report.GoVersion != ""), metadataCheck("source_identity_before_checks", report.Source != nil))
}

func checkFormatting(ctx context.Context, command commandFunc, root, goBinary string, environment, goFiles []string) (Check, error) {
	if len(goFiles) == 0 {
		return Check{}, errors.New("source scope has no Go files")
	}
	arguments := append([]string{"-l"}, goFiles...)
	gofmt := "gofmt"
	if filepath.IsAbs(goBinary) {
		gofmt = filepath.Join(filepath.Dir(goBinary), "gofmt")
	}
	start := time.Now()
	result := command(ctx, gofmt, arguments, root, environment)
	check := classifyCommand("format", true, result, time.Since(start))
	if check.Status == "passed" && len(bytes.TrimSpace(result.output)) > 0 {
		check.Status = "failed"
		check.Code = "format_changes_required"
	}
	return check, nil
}

func prepareOutput(root, raw string) (string, error) {
	if raw == "" {
		return "", errors.New("audit output directory is required")
	}
	directory, err := filepath.Abs(raw)
	if err != nil || directory == root {
		return "", errors.New("invalid audit output directory")
	}
	if outputOverlapsSource(root, directory) {
		return "", errors.New("audit output must not overlap source")
	}
	for _, path := range []string{directory, filepath.Join(directory, "home"), filepath.Join(directory, "cache"), filepath.Join(directory, "cache", "mod"), filepath.Join(directory, "cache", "build"), filepath.Join(directory, "tmp")} {
		if err = platform.EnsurePrivateDirectory(path); err != nil {
			return "", errors.New("cannot establish private audit directory")
		}
	}
	return directory, nil
}

func outputOverlapsSource(root, directory string) bool {
	relative, err := filepath.Rel(root, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return false
	}
	return !allowedOutputPath(relative)
}

func allowedOutputPath(relative string) bool {
	path := filepath.ToSlash(relative)
	return path == "audit-results" || strings.HasPrefix(path, "audit-results/") || path == ".local/audit-results" || strings.HasPrefix(path, ".local/audit-results/")
}

func auditEnvironment(home string) []string {
	output := filepath.Dir(home)
	environment := []string{"HOME=" + home, "USERPROFILE=" + home, "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "CGO_ENABLED=0", "GOMAXPROCS=2", "GOFLAGS=-p=1 -mod=readonly", "GOMODCACHE=" + filepath.Join(output, "cache", "mod"), "GOCACHE=" + filepath.Join(output, "cache", "build"), "TMPDIR=" + filepath.Join(output, "tmp"), "TEMP=" + filepath.Join(output, "tmp"), "TMP=" + filepath.Join(output, "tmp"), "GIT_NO_REPLACE_OBJECTS=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "SONAR_SCANNER_JAVA_OPTS=-Xmx1g -XX:ActiveProcessorCount=2"}
	for _, key := range []string{"PATH", "SystemRoot", "SYSTEMROOT", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy", "SSL_CERT_FILE", "SSL_CERT_DIR", "GOPROXY", "GOSUMDB"} {
		if value, ok := os.LookupEnv(key); ok {
			environment = append(environment, key+"="+value)
		}
	}
	return environment
}

func metadataCheck(name string, valid bool) Check {
	check := Check{Name: name, Required: true, Status: "passed"}
	if !valid {
		check.Status = "failed"
		check.Code = "metadata_unavailable"
	}
	return check
}

func coverageCheck(filename string, result *Coverage) Check {
	check := Check{Name: "statement_coverage", Required: true, Status: "failed", Code: "coverage_missing_or_invalid"}
	file, err := platform.OpenProtected(filename, false)
	if err != nil {
		return check
	}
	defer func() { _ = file.Close() }()
	parsed, err := ParseCoverage(file)
	if err != nil {
		return check
	}
	*result = parsed
	if parsed.Covered == parsed.Total {
		check.Status = "passed"
		check.Code = ""
	} else {
		check.Code = "coverage_below_100_percent"
	}
	return check
}

func (command commandFunc) execute(ctx context.Context, name string, required bool, binary string, args []string, directory string, environment []string) Check {
	start := time.Now()
	result := command(ctx, binary, args, directory, environment)
	return classifyCommand(name, required, result, time.Since(start))
}

func classifyCommand(name string, required bool, result commandResult, duration time.Duration) Check {
	check := Check{Name: name, Required: required, Status: "passed", ExitCode: result.exitCode, DurationMS: duration.Milliseconds()}
	if result.missing {
		check.Status = "not_run"
		check.Code = "tool_missing"
	} else if result.exitCode != 0 {
		check.Status = "failed"
		check.Code = "command_failed"
	}
	return check
}

func addExternalChecks(ctx context.Context, command commandFunc, root string, environment []string, required bool, scope goSourceScope, report *Report) {
	if len(scope.packages) == 0 {
		report.Checks = append(report.Checks, Check{Name: "gosec", Required: true, Status: "failed", Code: "source_scope_missing"})
		return
	}
	checks := []struct {
		name, binary string
		arguments    []string
	}{
		{"staticcheck", "staticcheck", scope.packages},
		{"gosec", "gosec", append([]string{"-quiet"}, sourcePackageArguments(report.Source)...)},
		{"govulncheck", "govulncheck", scope.packages},
		{"secret_scan", "gitleaks", []string{"git", "--redact=100", "--no-banner", "--ignore-gitleaks-allow", "."}},
		{"workflow_lint", "actionlint", nil},
	}
	for _, check := range checks {
		report.Checks = append(report.Checks, command.execute(ctx, check.name, required, check.binary, check.arguments, root, environment))
	}
}

func addSonar(ctx context.Context, command commandFunc, root string, environment []string, options Options, coverage string, report *Report) {
	required := options.Profile == "enterprise"
	if options.SonarURL == "" {
		report.Checks = append(report.Checks, Check{Name: "sonarqube", Status: "not_run", Required: required, Code: "sonar_not_configured"})
		return
	}
	if !validSonarURL(options.SonarURL) || !projectPattern.MatchString(options.SonarProjectKey) || !revisionPattern.MatchString(report.Commit) {
		report.Checks = append(report.Checks, Check{Name: "sonarqube", Status: "failed", Required: required, Code: "invalid_sonar_configuration"})
		return
	}
	token, err := readToken(options.SonarTokenFile)
	if err != nil {
		report.Checks = append(report.Checks, Check{Name: "sonarqube", Status: "failed", Required: required, Code: "sonar_token_unavailable"})
		return
	}
	metadata, err := newSonarMetadata(filepath.Dir(coverage))
	if err != nil {
		report.Checks = append(report.Checks, Check{Name: "sonarqube", Status: "failed", Required: required, Code: "sonar_metadata_unavailable"})
		return
	}
	arguments := []string{"-Dsonar.host.url=" + options.SonarURL, "-Dsonar.projectKey=" + options.SonarProjectKey, "-Dsonar.scm.revision=" + report.Commit, "-Dsonar.go.coverage.reportPaths=" + coverage, "-Dsonar.scanner.metadataFilePath=" + metadata, "-Dsonar.working.directory=" + filepath.Join(filepath.Dir(coverage), "scanner-work"), "-Dsonar.qualitygate.wait=true", "-Dsonar.qualitygate.ignoreSmallChanges=false"}
	check := command.execute(ctx, "sonarqube", required, "sonar-scanner", arguments, root, append(environment, "SONAR_TOKEN="+token))
	report.Checks = append(report.Checks, check)
	if check.Status != "not_run" {
		verifySonarRun(ctx, options, token, metadata, required, report)
	}
}

func verifySonarRun(ctx context.Context, options Options, token, metadata string, required bool, report *Report) {
	verification := Check{Name: "sonarqube_clean_counts", Required: required, Status: "failed", Code: "sonar_verification_failed"}
	defer func() { report.Checks = append(report.Checks, verification) }()
	taskID, err := sonarMetadataTask(metadata)
	if err != nil {
		return
	}
	summary, err := inspectSonar(ctx, options.SonarURL, token, options.SonarProjectKey, report.Commit, taskID)
	if err != nil {
		return
	}
	report.Sonar = &summary
	if summary.clean() {
		verification.Status = "passed"
		verification.Code = ""
	} else {
		verification.Code = "sonar_findings_or_incomplete_coverage"
	}
}

func (s SonarSummary) clean() bool {
	return s.Gate == "OK" && s.Issues == 0 && s.Hotspots == 0 && s.Coverage == 100 && s.Duplication == 0 && s.Warnings == 0
}

func validSonarURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" {
		return false
	}
	return u.Hostname() == "localhost" || net.ParseIP(u.Hostname()).IsLoopback()
}

func readToken(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("invalid token path")
	}
	info, err := os.Lstat(path)
	if err != nil || info.Size() > 4096 {
		return "", errors.New("invalid token file")
	}
	data, err := platform.ReadProtected(path, true)
	if err != nil || len(data) > 4096 {
		return "", errors.New("token read failed")
	}
	token := strings.TrimSpace(string(data))
	if token == "" || strings.ContainsAny(token, "\r\n\x00") {
		return "", errors.New("invalid token")
	}
	return token, nil
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.buffer.Len()+n > 1<<20 {
		b.overflow = true
		return n, nil
	}
	_, err := b.buffer.Write(p)
	return n, err
}

func (b *boundedBuffer) Bytes() []byte { return b.buffer.Bytes() }

func (b *boundedBuffer) String() string { return b.buffer.String() }

func runCommand(ctx context.Context, binary string, arguments []string, directory string, environment []string) commandResult {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, arguments...)
	cmd.Dir = directory
	cmd.Env = environment
	// CommandContext kills the immediate process on cancellation, while
	// WaitDelay bounds a finished process whose descendant retained an output
	// pipe. Neither mechanism establishes descendant containment.
	cmd.WaitDelay = auditCommandWaitDelay
	var buffer boundedBuffer
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer
	err := cmd.Run()
	if errors.Is(err, exec.ErrNotFound) {
		return commandResult{exitCode: 127, missing: true}
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return commandResult{exitCode: exit.ExitCode()}
		}
		return commandResult{exitCode: 1}
	}
	if buffer.overflow {
		return commandResult{exitCode: 1}
	}
	return commandResult{output: buffer.Bytes()}
}
