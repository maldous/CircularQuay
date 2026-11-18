package assurance

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

type sonarLifecycleStep struct {
	method, route string
	status        int
	body          string
}

type sonarLifecycleSequence struct {
	t     *testing.T
	steps []sonarLifecycleStep
	next  int
}

func newSonarLifecycleSequence(t *testing.T, steps []sonarLifecycleStep) *sonarLifecycleSequence {
	t.Helper()
	sequence := &sonarLifecycleSequence{t: t, steps: steps}
	t.Cleanup(func() {
		if sequence.next != len(steps) {
			t.Error("Sonar lifecycle stopped at an unexpected boundary")
		}
	})
	return sequence
}

func (sequence *sonarLifecycleSequence) factory(endpoint, username, password string) *sonarClient {
	client := newSonarClient(endpoint, username, password)
	client.client.Transport = sonarBoundaryTransport(func(request *http.Request) (*http.Response, error) {
		sequence.assertCredentials(request, username, password)
		return sequence.reply(request)
	})
	return client
}

func (sequence *sonarLifecycleSequence) assertCredentials(request *http.Request, username, password string) {
	sequence.t.Helper()
	gotUser, gotPassword, ok := request.BasicAuth()
	if !ok || gotUser != username || gotPassword != password || request.URL.User != nil {
		sequence.t.Fatal("lifecycle request lost its selected credential binding")
	}
	if _, bounded := request.Context().Deadline(); !bounded {
		sequence.t.Fatal("lifecycle request lost its bounded context")
	}
}

func (sequence *sonarLifecycleSequence) reply(request *http.Request) (*http.Response, error) {
	sequence.t.Helper()
	if sequence.next >= len(sequence.steps) {
		sequence.t.Fatal("unexpected request after selected Sonar refusal")
	}
	step := sequence.steps[sequence.next]
	sequence.next++
	if request.Method != step.method || request.URL.Path != step.route {
		sequence.t.Fatal("Sonar lifecycle effect order changed")
	}
	assertSonarLifecycleForm(sequence.t, request)
	return &http.Response{StatusCode: step.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(step.body)), Request: request}, nil
}

func assertSonarLifecycleForm(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Method != http.MethodPost {
		return
	}
	data, err := io.ReadAll(io.LimitReader(request.Body, 4097))
	if err != nil || len(data) > 4096 || request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatal("bootstrap request body escaped its bounded form contract")
	}
	form, err := url.ParseQuery(string(data))
	if err != nil {
		t.Fatal("bootstrap request was not a form")
	}
	assertSonarLifecycleScope(t, request.URL.Path, form)
}

func assertSonarLifecycleScope(t *testing.T, route string, form url.Values) {
	t.Helper()
	switch route {
	case "/api/permissions/add_user":
		if form.Get("projectKey") != "fixture" || !strings.HasPrefix(form.Get("login"), "cq-audit-") || form.Get("permission") != "user" && form.Get("permission") != "scan" {
			t.Fatal("audit principal permission escaped the selected project")
		}
	case "/api/user_tokens/generate":
		if form.Get("type") != "USER_TOKEN" || !strings.HasPrefix(form.Get("name"), "cq-audit-") || form.Get("expirationDate") == "" {
			t.Fatal("audit token lost its fixed type, principal or expiry")
		}
	}
}

func sonarLifecycleContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func sonarLifecycleGateSteps() []sonarLifecycleStep {
	steps := []sonarLifecycleStep{{http.MethodPost, "/api/qualitygates/create", 204, ""}, {http.MethodGet, "/api/qualitygates/show", 200, `{"conditions":[]}`}}
	for range requiredSonarConditions() {
		steps = append(steps, sonarLifecycleStep{http.MethodPost, "/api/qualitygates/create_condition", 204, ""})
	}
	return append(steps, sonarLifecycleStep{http.MethodPost, "/api/qualitygates/select", 204, ""}, sonarLifecycleStep{http.MethodPost, "/api/settings/set", 204, ""})
}

func sonarLifecycleBootstrapPrefix() []sonarLifecycleStep {
	return []sonarLifecycleStep{{http.MethodPost, "/api/users/change_password", 204, ""}, {http.MethodGet, "/api/authentication/validate", 200, `{"valid":true}`}}
}

func TestSonarLifecycleContractBootstrapStageFailuresStopBeforeLaterEffects(t *testing.T) {
	project := append(sonarLifecycleBootstrapPrefix(), sonarLifecycleStep{http.MethodPost, "/api/projects/create", 204, ""})
	profiles := append(append([]sonarLifecycleStep{}, project...), sonarLifecycleGateSteps()...)
	cases := []struct {
		name, expected string
		steps          []sonarLifecycleStep
	}{
		{"administrator", "sonar administrator authentication failed", []sonarLifecycleStep{{http.MethodPost, "/api/users/change_password", 503, "private marker"}, {http.MethodGet, "/api/authentication/validate", 200, `{"valid":false}`}}},
		{"project", "sonar project initialization failed", append(sonarLifecycleBootstrapPrefix(), sonarLifecycleStep{http.MethodPost, "/api/projects/create", 503, "private marker"}, sonarLifecycleStep{http.MethodGet, "/api/components/show", 503, "private marker"})},
		{"gate", "sonar gate must be created in a fresh audit instance", append(project, sonarLifecycleStep{http.MethodPost, "/api/qualitygates/create", 503, "private marker"})},
		{"profiles", "sonar request rejected", append(profiles, sonarLifecycleStep{http.MethodGet, "/api/qualityprofiles/search", 503, "private marker"})},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := privateAssuranceDirectory(t, "sonar-lifecycle-bootstrap")
			password := filepath.Join(directory, sonarAdminPasswordFile)
			original := []byte("synthetic-administrator\n")
			if err := platform.WriteProtected(password, original); err != nil {
				t.Fatal(err)
			}
			sequence := newSonarLifecycleSequence(t, testCase.steps)
			err := bootstrapSonarWithFactory(sonarLifecycleContext(t), directory, "http://127.0.0.1:1", "fixture", BootstrapSonarOptions{TokenValidityDays: 7}, sequence.factory)
			assertSonarBoundaryError(t, err, testCase.expected)
			assertSonarLifecycleRetainedState(t, directory, password, original)
		})
	}
}

func assertSonarLifecycleRetainedState(t *testing.T, directory, path string, original []byte) {
	t.Helper()
	got, err := platform.ReadProtected(path, true)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("bootstrap refusal changed its existing administrator input")
	}
	if _, err := os.Lstat(filepath.Join(directory, "scanner.token")); !os.IsNotExist(err) {
		t.Fatal("failed bootstrap published a scanner credential")
	}
}

func sonarLifecycleTokenSteps() []sonarLifecycleStep {
	return []sonarLifecycleStep{
		{http.MethodPost, "/api/users/create", 204, ""},
		{http.MethodPost, "/api/permissions/add_user", 204, ""},
		{http.MethodPost, "/api/permissions/add_user", 204, ""},
		{http.MethodPost, "/api/user_tokens/generate", 200, `{"token":"synthetic-issued-token"}`},
	}
}

func TestSonarLifecycleContractTokenStagesRefuseWithoutPublication(t *testing.T) {
	for stage := range sonarLifecycleTokenSteps() {
		t.Run(sonarLifecycleTokenSteps()[stage].route, func(t *testing.T) {
			steps := append([]sonarLifecycleStep{}, sonarLifecycleTokenSteps()[:stage+1]...)
			steps[stage].status, steps[stage].body = 503, "private marker"
			directory := privateAssuranceDirectory(t, "sonar-lifecycle-token")
			sequence := newSonarLifecycleSequence(t, steps)
			client := sequence.factory("http://127.0.0.1:1", "synthetic-administrator", "synthetic-password")
			err := createSonarAuditTokenWithFactory(sonarLifecycleContext(t), client, directory, "fixture", 7, sequence.factory)
			assertSonarBoundaryError(t, err, "sonar request rejected")
			if _, err := os.Lstat(filepath.Join(directory, "scanner.token")); !os.IsNotExist(err) {
				t.Fatal("incomplete token provisioning published a token")
			}
		})
	}
}

func TestSonarLifecycleContractTokenValueAndExclusivePublication(t *testing.T) {
	for _, value := range []string{`{"token":""}`, `{"token":"private\nmarker"}`} {
		t.Run(value, func(t *testing.T) {
			directory := privateAssuranceDirectory(t, "sonar-lifecycle-invalid-token")
			steps := sonarLifecycleTokenSteps()
			steps[3].body = value
			sequence := newSonarLifecycleSequence(t, steps)
			client := sequence.factory("http://127.0.0.1:1", "synthetic-administrator", "synthetic-password")
			err := createSonarAuditTokenWithFactory(sonarLifecycleContext(t), client, directory, "fixture", 7, sequence.factory)
			assertSonarBoundaryError(t, err, "invalid Sonar scanner credential")
			if _, err := os.Lstat(filepath.Join(directory, "scanner.token")); !os.IsNotExist(err) {
				t.Fatal("invalid issued token was published")
			}
		})
	}
}

func TestSonarLifecycleContractTokenPublicationPreservesExistingFile(t *testing.T) {
	directory := privateAssuranceDirectory(t, "sonar-lifecycle-existing-token")
	path := filepath.Join(directory, "scanner.token")
	original := []byte("existing synthetic credential\n")
	if err := platform.WriteProtected(path, original); err != nil {
		t.Fatal(err)
	}
	sequence := newSonarLifecycleSequence(t, sonarLifecycleTokenSteps())
	client := sequence.factory("http://127.0.0.1:1", "synthetic-administrator", "synthetic-password")
	if err := createSonarAuditTokenWithFactory(sonarLifecycleContext(t), client, directory, "fixture", 7, sequence.factory); err == nil {
		t.Fatal("issued token overwrote the existing credential")
	}
	got, err := platform.ReadProtected(path, true)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("token publication refusal changed the existing credential")
	}
}

func TestSonarLifecycleContractTokenControlWritesOnlyOwnedCredential(t *testing.T) {
	directory := privateAssuranceDirectory(t, "sonar-lifecycle-token-control")
	sequence := newSonarLifecycleSequence(t, sonarLifecycleTokenSteps())
	client := sequence.factory("http://127.0.0.1:1", "synthetic-administrator", "synthetic-password")
	if err := createSonarAuditTokenWithFactory(sonarLifecycleContext(t), client, directory, "fixture", 7, sequence.factory); err != nil {
		t.Fatal("bounded token control was refused")
	}
	got, err := readToken(filepath.Join(directory, "scanner.token"))
	if err != nil || got != "synthetic-issued-token" {
		t.Fatal("token control did not retain the exact issued credential")
	}
}

func sonarLifecycleInspectSteps() []sonarLifecycleStep {
	revision := strings.Repeat("a", 40)
	return []sonarLifecycleStep{
		{http.MethodGet, "/api/ce/task", 200, `{"task":{"componentKey":"fixture","status":"SUCCESS","analysisId":"fixture-analysis","warningCount":0}}`},
		{http.MethodGet, "/api/project_analyses/search", 200, `{"analyses":[{"key":"fixture-analysis","revision":"` + revision + `"}]}`},
		{http.MethodGet, "/api/qualitygates/project_status", 200, `{"projectStatus":{"status":"OK"}}`},
		{http.MethodGet, "/api/issues/search", 200, `{"total":0,"paging":{"total":0}}`},
		{http.MethodGet, "/api/hotspots/search", 200, `{"paging":{"total":0}}`},
		{http.MethodGet, "/api/measures/component", 200, `{"component":{"measures":[{"metric":"coverage","value":"100"},{"metric":"duplicated_lines_density","value":"0"}]}}`},
		{http.MethodGet, "/api/project_analyses/search", 200, `{"analyses":[{"key":"fixture-analysis","revision":"` + revision + `"}]}`},
	}
}

func TestSonarLifecycleContractInspectEveryStageRetainsSafePartialState(t *testing.T) {
	for stage := range sonarLifecycleInspectSteps() {
		t.Run(sonarLifecycleInspectSteps()[stage].route, func(t *testing.T) {
			steps := append([]sonarLifecycleStep{}, sonarLifecycleInspectSteps()[:stage+1]...)
			steps[stage].status, steps[stage].body = 503, "private marker"
			sequence := newSonarLifecycleSequence(t, steps)
			client := sequence.factory("http://sonar.invalid", "synthetic-token", "")
			summary, err := inspectSonarWithClient(sonarLifecycleContext(t), client, "fixture", strings.Repeat("a", 40), "fixture-task")
			assertSonarBoundaryError(t, err, "sonar request rejected")
			if summary != sonarLifecyclePartialSummary(stage) {
				t.Fatal("failed inspection changed the exact safe partial summary")
			}
		})
	}
	sequence := newSonarLifecycleSequence(t, sonarLifecycleInspectSteps())
	client := sequence.factory("http://sonar.invalid", "synthetic-token", "")
	summary, err := inspectSonarWithClient(sonarLifecycleContext(t), client, "fixture", strings.Repeat("a", 40), "fixture-task")
	if err != nil || !summary.clean() || summary.TaskID != "fixture-task" {
		t.Fatal("complete in-process inspection control was refused")
	}
}

type sonarLifecycleCloseFailure struct{ *os.File }

func (file sonarLifecycleCloseFailure) Close() error {
	if err := file.File.Close(); err != nil {
		return err
	}
	return errors.New("synthetic close uncertainty")
}

func TestSonarLifecycleContractMetadataCloseFailureDoesNotPublishAPath(t *testing.T) {
	directory := privateAssuranceDirectory(t, "sonar-lifecycle-close")
	var owned string
	create := func(parent, pattern string) (sonarMetadataFile, error) {
		file, err := os.CreateTemp(parent, pattern)
		if err != nil {
			return nil, err
		}
		owned = file.Name()
		return sonarLifecycleCloseFailure{file}, nil
	}
	path, err := newSonarMetadataWithIO(directory, filepath.Abs, create)
	assertSonarBoundaryError(t, err, "sonar task metadata close failed")
	if path != "" || owned == "" {
		t.Fatal("uncertain close published a path or did not exercise creation")
	}
	data, err := platform.ReadProtected(owned, true)
	if err != nil || len(data) != 0 {
		t.Fatal("uncertain close lost its bounded owned file")
	}
}

func TestSonarLifecycleContractPathFailuresArePreEffectAndSanitized(t *testing.T) {
	calls := 0
	resolve := func(string) (string, error) { calls++; return "", errors.New("private path marker") }
	create := func(string, string) (sonarMetadataFile, error) {
		t.Fatal("path refusal reached creation")
		return nil, nil
	}
	path, err := newSonarMetadataWithIO("relative", resolve, create)
	assertSonarBoundaryError(t, err, "invalid sonar metadata directory")
	if path != "" {
		t.Fatal("path refusal produced a metadata path")
	}
	task, err := sonarMetadataTaskWithPath("relative", resolve)
	assertSonarBoundaryError(t, err, "invalid sonar metadata path")
	token, err := readTokenWithPath("relative", resolve)
	assertSonarBoundaryError(t, err, "invalid token path")
	assertSonarBoundaryError(t, initSonarWithPath("relative", resolve), "invalid sonar state directory")
	if task != "" || token != "" || calls != 4 {
		t.Fatal("path refusal returned authority or skipped a selected resolver")
	}
}

func TestSonarLifecycleContractVerificationRetainsReportAndNeverPromotes(t *testing.T) {
	for _, state := range []string{"missing", "failed", "dirty", "clean"} {
		t.Run(state, func(t *testing.T) {
			directory := privateAssuranceDirectory(t, "sonar-lifecycle-report")
			metadata := filepath.Join(directory, "task.properties")
			if state != "missing" {
				if err := platform.WriteProtected(metadata, []byte("ceTaskId=fixture-task\n")); err != nil {
					t.Fatal(err)
				}
			}
			assertSonarLifecycleReportState(t, state, metadata)
		})
	}
}

func assertSonarLifecycleReportState(t *testing.T, state, metadata string) {
	t.Helper()
	prefix := []Check{{Name: "owned-previous-check", Status: "passed"}}
	report := Report{Commit: strings.Repeat("a", 40), Checks: append([]Check{}, prefix...)}
	options := Options{SonarURL: "http://sonar.invalid", SonarProjectKey: "fixture"}
	calls := 0
	inspect := func(_ context.Context, endpoint, token, project, revision, task string) (SonarSummary, error) {
		calls++
		assertSonarLifecycleInspectionBinding(t, []string{endpoint, token, project, revision, task}, []string{options.SonarURL, "synthetic-token", "fixture", report.Commit, "fixture-task"})
		return sonarLifecycleSummary(state)
	}
	verifySonarRunWithInspector(sonarLifecycleContext(t), options, "synthetic-token", metadata, true, &report, inspect)
	if len(report.Checks) != 2 || !reflect.DeepEqual(report.Checks[:1], prefix) || report.Passed || report.ProductionQualified {
		t.Fatal("verification changed prior checks or promoted whole report qualification")
	}
	assertSonarLifecycleCheck(t, state, calls, report)
}

func sonarLifecycleSummary(state string) (SonarSummary, error) {
	if state == "failed" {
		return SonarSummary{}, errors.New("private upstream marker")
	}
	summary := SonarSummary{Gate: "OK", Coverage: 100, TaskID: "fixture-task", AnalysisID: "fixture-analysis"}
	if state == "dirty" {
		summary.Warnings = 1
	}
	return summary, nil
}

func assertSonarLifecycleCheck(t *testing.T, state string, calls int, report Report) {
	t.Helper()
	wantCalls := 1
	if state == "missing" {
		wantCalls = 0
	}
	want := Check{Name: "sonarqube_clean_counts", Required: true, Status: "failed", Code: "sonar_verification_failed"}
	if state == "dirty" {
		want.Code = "sonar_findings_or_incomplete_coverage"
	}
	if state == "clean" {
		want.Status, want.Code = "passed", ""
	}
	if calls != wantCalls || report.Checks[1] != want {
		t.Fatal("verification lost exact mandatory outcome or call count")
	}
	assertSonarLifecycleSummary(t, state, report.Sonar)
}

func assertSonarLifecycleSummary(t *testing.T, state string, summary *SonarSummary) {
	t.Helper()
	if state == "missing" || state == "failed" {
		if summary != nil {
			t.Fatal("failed verification published an analysis summary")
		}
		return
	}
	want, _ := sonarLifecycleSummary(state)
	if summary == nil || *summary != want {
		t.Fatal("successful inspection lost exact safe summary metadata")
	}
}

func assertSonarLifecycleInspectionBinding(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatal("verification did not retain its source, task or credential binding")
	}
}

func TestSonarLifecycleContractBootstrapExclusiveWritePreservesOccupiedInput(t *testing.T) {
	directory := privateAssuranceDirectory(t, "sonar-lifecycle-existing-compose")
	path := filepath.Join(directory, "compose.env")
	original := []byte("synthetic existing bootstrap input\n")
	if err := platform.WriteProtected(path, original); err != nil {
		t.Fatal(err)
	}
	if err := writeSonarBootstrap(directory); err == nil {
		t.Fatal("bootstrap replaced an exclusively owned input")
	}
	retained, err := platform.ReadProtected(path, true)
	if err != nil || !bytes.Equal(retained, original) {
		t.Fatal("bootstrap write refusal changed the existing input")
	}
	if _, err := os.Lstat(filepath.Join(directory, sonarAdminPasswordFile)); !os.IsNotExist(err) {
		t.Fatal("first bootstrap write refusal reached administrator publication")
	}
}

func TestSonarLifecycleContractAuditPathRefusalBeforeOutputPreparation(t *testing.T) {
	options := Options{Profile: "source", Source: "relative", Output: "must-not-be-created"}
	calls := 0
	resolve := func(path string) (string, error) {
		calls++
		if path != options.Source {
			t.Fatal("audit path resolution used an unrelated input")
		}
		return "", errors.New("private path marker")
	}
	audit, err := prepareAuditWithPath(options, resolve)
	assertSonarBoundaryError(t, err, "invalid source directory")
	if calls != 1 || !reflect.DeepEqual(audit, auditRun{}) {
		t.Fatal("source path refusal created partial audit authority")
	}
}

func sonarLifecyclePartialSummary(failedStage int) SonarSummary {
	if failedStage == 0 {
		return SonarSummary{}
	}
	summary := SonarSummary{AnalysisID: "fixture-analysis", TaskID: "fixture-task"}
	if failedStage > 2 {
		summary.Gate = "OK"
	}
	if failedStage > 5 {
		summary.Coverage = 100
	}
	return summary
}
