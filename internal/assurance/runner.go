package assurance

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
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
	revision := command(ctx, "git", []string{"rev-parse", "HEAD"}, root, environment)
	if revisionPattern.Match(bytes.TrimSpace(revision.output)) {
		report.Commit = strings.TrimSpace(string(revision.output))
	}
	version := command(ctx, options.GoBinary, []string{"version"}, root, environment)
	if found := goVersionPattern.FindSubmatch(version.output); len(found) > 1 {
		report.GoVersion = string(found[1])
	}
	report.Checks = append(report.Checks, metadataCheck("source_revision", report.Commit != ""), metadataCheck("go_toolchain", report.GoVersion != ""))
	report.Checks = append(report.Checks, repositoryCheck(root))
	goFiles, err := sourceFiles(root)
	if err != nil {
		return report, err
	}
	formatArguments := append([]string{"-l"}, goFiles...)
	gofmt := "gofmt"
	if filepath.IsAbs(options.GoBinary) {
		gofmt = filepath.Join(filepath.Dir(options.GoBinary), "gofmt")
	}
	format := execute(ctx, command, "format", true, gofmt, formatArguments, root, environment)
	if format.Status == "passed" {
		result := command(ctx, gofmt, formatArguments, root, environment)
		if len(bytes.TrimSpace(result.output)) > 0 {
			format.Status = "failed"
			format.Code = "format_changes_required"
		}
	}
	report.Checks = append(report.Checks, format)
	coveragePath := filepath.Join(output, "coverage.out")
	report.Checks = append(report.Checks, execute(ctx, command, "unit_tests", true, options.GoBinary, []string{"test", "-count=1", "-covermode=atomic", "-coverpkg=./...", "-coverprofile=" + coveragePath, "./..."}, root, environment))
	report.Checks = append(report.Checks, coverageCheck(coveragePath, &report.Coverage))
	report.Checks = append(report.Checks, execute(ctx, command, "go_vet", true, options.GoBinary, []string{"vet", "./..."}, root, environment))
	addExternalChecks(ctx, command, root, environment, options.Profile == "enterprise", &report)
	addSonar(ctx, command, root, environment, options, coveragePath, &report)
	report.Checks = append(report.Checks, Check{Name: "native_reboot_and_restore", Status: "not_run", Code: "requires_qualified_native_environment"}, Check{Name: "independent_review", Status: "not_run", Code: "requires_independent_reviewer"})
	report.finish()
	if err = WriteReport(output, report); err != nil {
		return report, err
	}
	return report, nil
}

func prepareOutput(root, raw string) (string, error) {
	if raw == "" {
		return "", errors.New("audit output directory is required")
	}
	directory, err := filepath.Abs(raw)
	if err != nil || directory == root {
		return "", errors.New("invalid audit output directory")
	}
	if relative, relErr := filepath.Rel(root, directory); relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
		first := strings.Split(relative, string(filepath.Separator))[0]
		if first != "audit-results" && first != ".local" {
			return "", errors.New("audit output must not overlap source")
		}
	}
	if info, statErr := os.Lstat(directory); statErr == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return "", errors.New("unsafe audit output directory")
	}
	if err = os.MkdirAll(directory, 0700); err != nil {
		return "", errors.New("cannot create audit output directory")
	}
	if err = os.Chmod(directory, 0700); err != nil {
		return "", errors.New("cannot protect audit output directory")
	}
	if err = os.MkdirAll(filepath.Join(directory, "home"), 0700); err != nil {
		return "", errors.New("cannot create audit home")
	}
	return directory, nil
}

func auditEnvironment(home string) []string {
	environment := []string{"HOME=" + home, "USERPROFILE=" + home, "GOTOOLCHAIN=local", "CGO_ENABLED=0"}
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
	file, err := os.Open(filename)
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

func execute(ctx context.Context, command commandFunc, name string, required bool, binary string, args []string, directory string, environment []string) Check {
	start := time.Now()
	result := command(ctx, binary, args, directory, environment)
	check := Check{Name: name, Required: required, Status: "passed", ExitCode: result.exitCode, DurationMS: time.Since(start).Milliseconds()}
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
		{"secret_scan", "gitleaks", []string{"dir", "--redact=100", "--no-banner", "."}},
		{"workflow_lint", "actionlint", nil},
	}
	for _, check := range checks {
		report.Checks = append(report.Checks, execute(ctx, command, check.name, required, check.binary, check.arguments, root, environment))
	}
}

func addSonar(ctx context.Context, command commandFunc, root string, environment []string, options Options, coverage string, report *Report) {
	required := options.Profile == "enterprise"
	if options.SonarURL == "" {
		report.Checks = append(report.Checks, Check{Name: "sonarqube", Status: "not_run", Required: required, Code: "sonar_not_configured"})
		return
	}
	if !validSonarURL(options.SonarURL) || !projectPattern.MatchString(options.SonarProjectKey) {
		report.Checks = append(report.Checks, Check{Name: "sonarqube", Status: "failed", Required: required, Code: "invalid_sonar_configuration"})
		return
	}
	token, err := readToken(options.SonarTokenFile)
	if err != nil {
		report.Checks = append(report.Checks, Check{Name: "sonarqube", Status: "failed", Required: required, Code: "sonar_token_unavailable"})
		return
	}
	arguments := []string{"-Dsonar.host.url=" + options.SonarURL, "-Dsonar.projectKey=" + options.SonarProjectKey, "-Dsonar.go.coverage.reportPaths=" + coverage, "-Dsonar.qualitygate.wait=true", "-Dsonar.qualitygate.ignoreSmallChanges=false"}
	check := execute(ctx, command, "sonarqube", required, "sonar-scanner", arguments, root, append(environment, "SONAR_TOKEN="+token))
	report.Checks = append(report.Checks, check)
	if check.Status != "not_run" {
		summary, inspectErr := inspectSonar(ctx, options.SonarURL, token, options.SonarProjectKey)
		verification := Check{Name: "sonarqube_clean_counts", Required: required, Status: "failed", Code: "sonar_verification_failed"}
		if inspectErr == nil {
			report.Sonar = &summary
			if summary.Gate == "OK" && summary.Issues == 0 && summary.Hotspots == 0 && summary.Coverage == 100 && summary.Duplication == 0 {
				verification.Status = "passed"
				verification.Code = ""
			} else {
				verification.Code = "sonar_findings_or_incomplete_coverage"
			}
		}
		report.Checks = append(report.Checks, verification)
	}
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
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 4096 {
		return "", errors.New("invalid token file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return "", errors.New("unprotected token file")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", errors.New("token read failed")
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return "", errors.New("token changed during open")
	}
	data, err := io.ReadAll(io.LimitReader(file, 4097))
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
