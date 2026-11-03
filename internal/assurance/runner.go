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

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
var goVersionPattern = regexp.MustCompile(`^go version (go[0-9]+\.[0-9]+(?:\.[0-9]+)?) [a-z0-9]+/[a-z0-9]+\s*$`)
var projectPattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,200}$`)

// Run executes a local source audit and creates a sanitized evidence bundle.
func Run(ctx context.Context, options Options) (Report, error) { return run(ctx, options, runCommand) }

func run(ctx context.Context, options Options, command commandFunc) (Report, error) {
	if options.Profile != "source" && options.Profile != "enterprise" {
		return Report{}, errors.New("unknown audit profile")
	}
	root, err := filepath.Abs(options.Source)
	if err != nil {
		return Report{}, errors.New("invalid source directory")
	}
	output, err := prepareOutput(root, options.Output)
	if err != nil {
		return Report{}, err
	}
	if options.GoBinary == "" {
		options.GoBinary = "go"
	}
	report := Report{Version: 1, Profile: options.Profile, Platform: runtime.GOOS + "/" + runtime.GOARCH, StartedAt: time.Now().UTC(), Limitations: []string{
		"Automated source analysis is not security certification.",
		"Coverage is measured for the current native build, not unexecuted platforms.",
		"Independent review, real reboot and live provider qualification are separate evidence.",
	}}
	environment := auditEnvironment(filepath.Join(output, "home"))
	recordSourceIdentity(ctx, command, root, options.GoBinary, environment, &report)
	report.Checks = append(report.Checks, repositoryCheck(root))
	format, err := checkFormatting(ctx, command, root, options.GoBinary, environment)
	if err != nil {
		return report, err
	}
	report.Checks = append(report.Checks, format)
	coveragePath := filepath.Join(output, "coverage.out")
	report.Checks = append(report.Checks, command.execute(ctx, "unit_tests", true, options.GoBinary, []string{"test", "-count=1", "-covermode=atomic", "-coverpkg=" + modulePrefix + "...", "-coverprofile=" + coveragePath, "./..."}, root, environment))
	report.Checks = append(report.Checks, coverageCheck(coveragePath, &report.Coverage))
	report.Checks = append(report.Checks, command.execute(ctx, "go_vet", true, options.GoBinary, []string{"vet", "./..."}, root, environment))
	addExternalChecks(ctx, command, root, environment, options.Profile == "enterprise", &report)
	addSonar(ctx, command, root, environment, options, coveragePath, &report)
	report.Checks = append(report.Checks, Check{Name: "native_reboot_and_restore", Status: "not_run", Code: "requires_qualified_native_environment"}, Check{Name: "independent_review", Status: "not_run", Code: "requires_independent_reviewer"})
	clean := command(ctx, "git", []string{"status", "--porcelain", "--untracked-files=normal"}, root, environment)
	report.Checks = append(report.Checks, metadataCheck("source_tree_clean_after_checks", successfulEmptyOutput(clean)))
	report.finish()
	if err = WriteReport(output, report); err != nil {
		return report, err
	}
	return report, nil
}

func recordSourceIdentity(ctx context.Context, command commandFunc, root, goBinary string, environment []string, report *Report) {
	revision := command(ctx, "git", []string{"rev-parse", "HEAD"}, root, environment)
	if revisionPattern.Match(bytes.TrimSpace(revision.output)) {
		report.Commit = strings.TrimSpace(string(revision.output))
	}
	version := command(ctx, goBinary, []string{"version"}, root, environment)
	if found := goVersionPattern.FindSubmatch(version.output); len(found) > 1 {
		report.GoVersion = string(found[1])
	}
	report.Checks = append(report.Checks, metadataCheck("source_revision", report.Commit != ""), metadataCheck("go_toolchain", report.GoVersion != ""))
	clean := command(ctx, "git", []string{"status", "--porcelain", "--untracked-files=normal"}, root, environment)
	report.Checks = append(report.Checks, metadataCheck("source_tree_clean", successfulEmptyOutput(clean)))
}

func successfulEmptyOutput(result commandResult) bool {
	return !result.missing && result.exitCode == 0 && len(bytes.TrimSpace(result.output)) == 0
}

func checkFormatting(ctx context.Context, command commandFunc, root, goBinary string, environment []string) (Check, error) {
	goFiles, err := sourceFiles(root)
	if err != nil {
		return Check{}, err
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
	for _, path := range []string{directory, filepath.Join(directory, "home")} {
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
	first := strings.Split(relative, string(filepath.Separator))[0]
	return first != "audit-results" && first != ".local"
}

func auditEnvironment(home string) []string {
	environment := []string{"HOME=" + home, "USERPROFILE=" + home, "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOMAXPROCS=2", "GOFLAGS=-p=1", "SONAR_SCANNER_JAVA_OPTS=-Xmx1g -XX:ActiveProcessorCount=2"}
	for _, key := range []string{"PATH", "SystemRoot", "SYSTEMROOT", "TEMP", "TMP", "TMPDIR", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy", "SSL_CERT_FILE", "SSL_CERT_DIR", "GOPROXY", "GOSUMDB", "GOMODCACHE", "GOCACHE"} {
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

func addExternalChecks(ctx context.Context, command commandFunc, root string, environment []string, required bool, report *Report) {
	checks := []struct {
		name, binary string
		arguments    []string
	}{
		{"staticcheck", "staticcheck", []string{"./..."}},
		{"gosec", "gosec", []string{"-quiet", "./..."}},
		{"govulncheck", "govulncheck", []string{"./..."}},
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
	arguments := []string{"-Dsonar.host.url=" + options.SonarURL, "-Dsonar.projectKey=" + options.SonarProjectKey, "-Dsonar.scm.revision=" + report.Commit, "-Dsonar.go.coverage.reportPaths=" + coverage, "-Dsonar.scanner.metadataFilePath=" + metadata, "-Dsonar.qualitygate.wait=true", "-Dsonar.qualitygate.ignoreSmallChanges=false"}
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
	bytes.Buffer
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len()+n > 1<<20 {
		b.overflow = true
		return n, nil
	}
	_, err := b.Buffer.Write(p)
	return n, err
}

func runCommand(ctx context.Context, binary string, arguments []string, directory string, environment []string) commandResult {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, arguments...)
	cmd.Dir = directory
	cmd.Env = environment
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
