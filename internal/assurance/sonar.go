package assurance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

const sonarAdminPasswordFile = "admin.password"

// SonarSummary contains counts only; API response bodies are never published.
type SonarSummary struct {
	Gate        string  `json:"quality_gate"`
	Issues      int     `json:"issue_count"`
	Hotspots    int     `json:"review_required_hotspots"`
	Coverage    float64 `json:"coverage_percent"`
	Duplication float64 `json:"duplicated_lines_percent"`
	Warnings    int     `json:"analysis_warning_count"`
	AnalysisID  string  `json:"analysis_id"`
	TaskID      string  `json:"task_id"`
}

type sonarClient struct {
	endpoint           string
	client             *http.Client
	username, password string
}

func newSonarClient(endpoint, username, password string) *sonarClient {
	return &sonarClient{endpoint: strings.TrimRight(endpoint, "/"), username: username, password: password, client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (c *sonarClient) call(ctx context.Context, method, route string, parameters url.Values, destination any) error {
	address := c.endpoint + route
	var body io.Reader
	if method == http.MethodGet {
		address += "?" + parameters.Encode()
	} else {
		body = strings.NewReader(parameters.Encode())
	}
	request, err := http.NewRequestWithContext(ctx, method, address, body)
	if err != nil {
		return errors.New("invalid Sonar request")
	}
	if method != http.MethodGet {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	request.SetBasicAuth(c.username, c.password)
	response, err := c.client.Do(request)
	if err != nil {
		return errors.New("sonar request unavailable")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("sonar request rejected")
	}
	if destination == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		if err != nil {
			return errors.New("sonar response failed")
		}
		return nil
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(destination); err != nil {
		return errors.New("invalid Sonar response")
	}
	return nil
}

func inspectSonar(ctx context.Context, endpoint, token, project, revision, taskID string) (SonarSummary, error) {
	client := newSonarClient(endpoint, token, "")
	var summary SonarSummary
	if err := inspectSonarTask(ctx, client, project, taskID, &summary); err != nil {
		return summary, err
	}
	if err := inspectSonarRevision(ctx, client, project, revision, summary.AnalysisID); err != nil {
		return summary, err
	}
	var gate struct {
		ProjectStatus struct {
			Status string `json:"status"`
		} `json:"projectStatus"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/qualitygates/project_status", url.Values{"analysisId": {summary.AnalysisID}}, &gate); err != nil {
		return summary, err
	}
	summary.Gate = gate.ProjectStatus.Status
	if err := inspectSonarCounts(ctx, client, project, &summary); err != nil {
		return summary, err
	}
	return summary, inspectSonarMeasures(ctx, client, project, &summary)
}

func inspectSonarRevision(ctx context.Context, client *sonarClient, project, revision, analysisID string) error {
	var result struct {
		Analyses []struct {
			Key      string `json:"key"`
			Revision string `json:"revision"`
		} `json:"analyses"`
	}
	if !revisionPattern.MatchString(revision) {
		return errors.New("invalid sonar source revision")
	}
	if err := client.call(ctx, http.MethodGet, "/api/project_analyses/search", url.Values{"project": {project}, "ps": {"1"}}, &result); err != nil {
		return err
	}
	if len(result.Analyses) != 1 || result.Analyses[0].Key != analysisID || result.Analyses[0].Revision != revision {
		return errors.New("sonar analysis revision mismatch")
	}
	return nil
}

func inspectSonarTask(ctx context.Context, client *sonarClient, project, taskID string, summary *SonarSummary) error {
	var tasks struct {
		Task struct {
			ComponentKey string `json:"componentKey"`
			Status       string `json:"status"`
			AnalysisID   string `json:"analysisId"`
			WarningCount *int   `json:"warningCount"`
		} `json:"task"`
	}
	if !projectPattern.MatchString(taskID) {
		return errors.New("invalid sonar task identity")
	}
	if err := client.call(ctx, http.MethodGet, "/api/ce/task", url.Values{"id": {taskID}, "additionalFields": {"warnings"}}, &tasks); err != nil {
		return err
	}
	if tasks.Task.Status != "SUCCESS" || tasks.Task.ComponentKey != project || tasks.Task.AnalysisID == "" || tasks.Task.WarningCount == nil || *tasks.Task.WarningCount < 0 {
		return errors.New("sonar analysis incomplete")
	}
	summary.AnalysisID = tasks.Task.AnalysisID
	summary.Warnings = *tasks.Task.WarningCount
	summary.TaskID = taskID
	return nil
}

func newSonarMetadata(directory string) (string, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return "", errors.New("invalid sonar metadata directory")
	}
	file, err := os.CreateTemp(directory, ".sonar-task-*.properties")
	if err != nil {
		return "", errors.New("sonar task metadata creation failed")
	}
	if err = file.Close(); err != nil {
		return "", errors.New("sonar task metadata close failed")
	}
	return file.Name(), nil
}

func sonarMetadataTask(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("invalid sonar metadata path")
	}
	data, err := platform.ReadProtected(path, true)
	if err != nil || len(data) > 4096 {
		return "", errors.New("sonar task metadata unavailable")
	}
	var taskID string
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(strings.TrimSuffix(line, "\r"), "=")
		if !found || key != "ceTaskId" {
			continue
		}
		if taskID != "" || !projectPattern.MatchString(value) {
			return "", errors.New("invalid sonar task metadata")
		}
		taskID = value
	}
	if taskID == "" {
		return "", errors.New("sonar task identity missing")
	}
	return taskID, nil
}

func inspectSonarCounts(ctx context.Context, client *sonarClient, project string, summary *SonarSummary) error {
	var issues struct {
		Total  *int `json:"total"`
		Paging struct {
			Total *int `json:"total"`
		} `json:"paging"`
	}
	// Include every non-fixed state exposed by the pinned reference API.
	// Accepted and false-positive dispositions must block, never waive, this gate.
	if err := client.call(ctx, http.MethodGet, "/api/issues/search", url.Values{"componentKeys": {project}, "issueStatuses": {"OPEN,CONFIRMED,FALSE_POSITIVE,ACCEPTED,IN_SANDBOX"}, "ps": {"1"}}, &issues); err != nil {
		return err
	}
	if issues.Total == nil || issues.Paging.Total == nil || *issues.Total < 0 || *issues.Total != *issues.Paging.Total {
		return errors.New("sonar issue counts missing or inconsistent")
	}
	summary.Issues = *issues.Total
	var hotspots struct {
		Paging struct {
			Total *int `json:"total"`
		} `json:"paging"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/hotspots/search", url.Values{"project": {project}, "status": {"TO_REVIEW"}, "ps": {"1"}}, &hotspots); err != nil {
		return err
	}
	if hotspots.Paging.Total == nil || *hotspots.Paging.Total < 0 {
		return errors.New("sonar hotspot count missing")
	}
	summary.Hotspots = *hotspots.Paging.Total
	return nil
}

func inspectSonarMeasures(ctx context.Context, client *sonarClient, project string, summary *SonarSummary) error {
	var measures struct {
		Component struct {
			Measures []struct {
				Metric string `json:"metric"`
				Value  string `json:"value"`
			} `json:"measures"`
		} `json:"component"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/measures/component", url.Values{"component": {project}, "metricKeys": {"coverage,duplicated_lines_density"}}, &measures); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, measure := range measures.Component.Measures {
		number, err := parseSonarPercent(measure.Value)
		if err != nil {
			return err
		}
		switch measure.Metric {
		case "coverage":
			summary.Coverage = number
			seen[measure.Metric] = true
		case "duplicated_lines_density":
			summary.Duplication = number
			seen[measure.Metric] = true
		}
	}
	if !seen["coverage"] || !seen["duplicated_lines_density"] {
		return errors.New("sonar coverage metrics missing")
	}
	return nil
}

func parseSonarPercent(value string) (float64, error) {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number > 100 {
		return 0, errors.New("invalid sonar metric")
	}
	return number, nil
}

// InitSonar creates local bootstrap files without printing or accepting secret values.
func InitSonar(directory string) error {
	if directory == "" {
		return errors.New("sonar state directory required")
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return errors.New("invalid sonar state directory")
	}
	if err = platform.EnsurePrivateDirectory(directory); err != nil {
		return errors.New("sonar state protection failed")
	}
	if err = sonarStateAbsent(directory); err != nil {
		return err
	}
	return writeSonarBootstrap(directory)
}

func sonarStateAbsent(directory string) error {
	for _, name := range []string{"compose.env", sonarAdminPasswordFile, "scanner.token"} {
		if _, err := os.Lstat(filepath.Join(directory, name)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("sonar state already exists")
		}
	}
	return nil
}

func writeSonarBootstrap(directory string) error {
	password, err := randomSecret()
	if err != nil {
		return err
	}
	if err = writeNewPrivate(filepath.Join(directory, "compose.env"), []byte("SONAR_DB_PASSWORD="+password+"\n")); err != nil {
		return err
	}
	password, err = randomSecret()
	if err != nil {
		return err
	}
	return writeNewPrivate(filepath.Join(directory, sonarAdminPasswordFile), []byte("Cq9!"+password+"\n"))
}

func randomSecret() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", errors.New("secure random source unavailable")
	}
	return hex.EncodeToString(data), nil
}

func writeNewPrivate(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("private state creation failed")
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errors.New("private state write failed")
	}
	return nil
}

// BootstrapSonar initializes only a dedicated loopback test server, never an arbitrary remote instance.
func BootstrapSonar(ctx context.Context, directory, endpoint, project string) error {
	if !validSonarURL(endpoint) || !projectPattern.MatchString(project) {
		return errors.New("invalid Sonar bootstrap configuration")
	}
	u, _ := url.Parse(endpoint)
	if u.Scheme != "http" || !isLoopbackHost(u.Hostname()) {
		return errors.New("bootstrap requires a dedicated loopback server")
	}
	client, err := authenticateSonarAdministrator(ctx, directory, endpoint)
	if err != nil {
		return err
	}
	if err = createSonarProject(ctx, client, project); err != nil {
		return err
	}
	if err = configureSonarGate(ctx, client, project); err != nil {
		return err
	}
	if err = configureSonarProfiles(ctx, client, project); err != nil {
		return err
	}
	return createSonarAuditToken(ctx, client, directory, project)
}

func authenticateSonarAdministrator(ctx context.Context, directory, endpoint string) (*sonarClient, error) {
	password, err := readToken(filepath.Join(directory, sonarAdminPasswordFile))
	if err != nil {
		return nil, err
	}
	initial := newSonarClient(endpoint, "admin", "admin")
	// Failure can mean the dedicated server was initialized previously; validate the configured password below.
	_ = initial.call(ctx, http.MethodPost, "/api/users/change_password", url.Values{"login": {"admin"}, "previousPassword": {"admin"}, "password": {password}}, nil)
	client := newSonarClient(endpoint, "admin", password)
	var auth struct {
		Valid bool `json:"valid"`
	}
	if err = client.call(ctx, http.MethodGet, "/api/authentication/validate", nil, &auth); err != nil || !auth.Valid {
		return nil, errors.New("sonar administrator authentication failed")
	}
	return client, nil
}

func createSonarAuditToken(ctx context.Context, administrator *sonarClient, directory, project string) error {
	name, err := randomSecret()
	if err != nil {
		return err
	}
	login := "cquay-audit-" + name[:12]
	password, err := randomSecret()
	if err != nil {
		return err
	}
	password = "Cq9!" + password
	if err = administrator.call(ctx, http.MethodPost, "/api/users/create", url.Values{"login": {login}, "name": {"CircularQuay temporary audit"}, "local": {"true"}, "password": {password}}, nil); err != nil {
		return err
	}
	// A project-analysis token cannot read hotspots or completed task warnings.
	// Use a dedicated principal with only project-scoped Browse and Execute
	// Analysis permissions. It cannot administer projects or review findings.
	for _, permission := range []string{"user", "scan"} {
		if err = administrator.call(ctx, http.MethodPost, "/api/permissions/add_user", url.Values{"login": {login}, "projectKey": {project}, "permission": {permission}}, nil); err != nil {
			return err
		}
	}
	client := newSonarClient(administrator.endpoint, login, password)
	var created struct {
		Token string `json:"token"`
	}
	if err = client.call(ctx, http.MethodPost, "/api/user_tokens/generate", url.Values{"name": {login}, "type": {"USER_TOKEN"}, "expirationDate": {time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")}}, &created); err != nil {
		return err
	}
	if created.Token == "" || strings.ContainsAny(created.Token, "\r\n\x00") {
		return errors.New("invalid Sonar scanner credential")
	}
	return writeNewPrivate(filepath.Join(directory, "scanner.token"), []byte(created.Token+"\n"))
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func createSonarProject(ctx context.Context, client *sonarClient, project string) error {
	if err := client.call(ctx, http.MethodPost, "/api/projects/create", url.Values{"project": {project}, "name": {"CircularQuay independent audit"}, "visibility": {"private"}}, nil); err == nil {
		return nil
	}
	var component struct {
		Component struct {
			Key string `json:"key"`
		} `json:"component"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/components/show", url.Values{"component": {project}}, &component); err != nil || component.Component.Key != project {
		return errors.New("sonar project initialization failed")
	}
	return nil
}

func configureSonarGate(ctx context.Context, client *sonarClient, project string) error {
	const name = "CircularQuay Enterprise"
	if err := client.call(ctx, http.MethodPost, "/api/qualitygates/create", url.Values{"name": {name}}, nil); err != nil {
		return errors.New("sonar gate must be created in a fresh audit instance")
	}
	var existing struct {
		Conditions []sonarGateCondition `json:"conditions"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/qualitygates/show", url.Values{"name": {name}}, &existing); err != nil {
		return err
	}
	for _, condition := range requiredSonarConditions() {
		if err := setSonarCondition(ctx, client, name, condition, existing.Conditions); err != nil {
			return err
		}
	}
	if err := client.call(ctx, http.MethodPost, "/api/qualitygates/select", url.Values{"gateName": {name}, "projectKey": {project}}, nil); err != nil {
		return errors.New("sonar quality gate assignment failed")
	}
	return client.call(ctx, http.MethodPost, "/api/settings/set", url.Values{"key": {"sonar.qualitygate.ignoreSmallChanges"}, "value": {"false"}, "component": {project}}, nil)
}

type sonarGateCondition struct {
	ID        string `json:"id"`
	Metric    string `json:"metric"`
	Operator  string `json:"op"`
	Threshold string `json:"error"`
}

func requiredSonarConditions() []sonarGateCondition {
	return []sonarGateCondition{
		{Metric: "violations", Operator: "GT", Threshold: "0"},
		{Metric: "new_violations", Operator: "GT", Threshold: "0"},
		{Metric: "coverage", Operator: "LT", Threshold: "100"},
		{Metric: "new_coverage", Operator: "LT", Threshold: "100"},
		{Metric: "duplicated_lines_density", Operator: "GT", Threshold: "0"},
		{Metric: "new_duplicated_lines_density", Operator: "GT", Threshold: "0"},
		{Metric: "security_hotspots_reviewed", Operator: "LT", Threshold: "100"},
		{Metric: "new_security_hotspots_reviewed", Operator: "LT", Threshold: "100"},
	}
}

func setSonarCondition(ctx context.Context, client *sonarClient, name string, condition sonarGateCondition, existing []sonarGateCondition) error {
	parameters := url.Values{"gateName": {name}, "metric": {condition.Metric}, "op": {condition.Operator}, "error": {condition.Threshold}}
	route := "/api/qualitygates/create_condition"
	// Current servers seed a new gate with weaker new-code defaults. Tighten
	// those existing conditions; trying to create them again is rejected.
	for _, previous := range existing {
		if previous.Metric == condition.Metric {
			route = "/api/qualitygates/update_condition"
			parameters.Set("id", previous.ID)
			parameters.Del("gateName")
			break
		}
	}
	if err := client.call(ctx, http.MethodPost, route, parameters, nil); err != nil {
		return errors.New("sonar strict condition configuration failed")
	}
	return nil
}

type sonarQualityProfile struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Language    string `json:"language"`
	BuiltIn     bool   `json:"isBuiltIn"`
	ActiveRules int    `json:"activeRuleCount"`
}

func configureSonarProfiles(ctx context.Context, client *sonarClient, project string) error {
	var found struct {
		Profiles []sonarQualityProfile `json:"profiles"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/qualityprofiles/search", nil, &found); err != nil {
		return err
	}
	for _, language := range []string{"go", "json", "yaml", "docker", "xml", "web", "text", "secrets"} {
		profile, err := completeSonarProfile(found.Profiles, language)
		if err != nil {
			return err
		}
		if language == "go" {
			profile, err = sonarComplexityProfile(ctx, client, profile)
			if err != nil {
				return err
			}
		}
		parameters := url.Values{"project": {project}, "language": {language}, "qualityProfile": {profile.Name}}
		if err = client.call(ctx, http.MethodPost, "/api/qualityprofiles/add_project", parameters, nil); err != nil {
			return errors.New("sonar comprehensive profile assignment failed")
		}
	}
	return nil
}

func completeSonarProfile(profiles []sonarQualityProfile, language string) (sonarQualityProfile, error) {
	for _, name := range []string{"Sonar way comprehensive", "Sonar way"} {
		for _, profile := range profiles {
			if profile.Language == language && profile.Name == name && profile.BuiltIn && (language != "go" || profile.ActiveRules > 0) {
				return profile, nil
			}
		}
	}
	return sonarQualityProfile{}, errors.New("sonar comprehensive language profile unavailable")
}

func sonarComplexityProfile(ctx context.Context, client *sonarClient, base sonarQualityProfile) (sonarQualityProfile, error) {
	var profile sonarQualityProfile
	name := "CircularQuay Go comprehensive"
	if err := client.call(ctx, http.MethodPost, "/api/qualityprofiles/copy", url.Values{"fromKey": {base.Key}, "toName": {name}}, &profile); err != nil {
		return profile, err
	}
	if profile.Key == "" || profile.Name != name {
		return profile, errors.New("sonar profile copy failed")
	}
	// Copying the complete built-in profile retains its recommended rules. Only
	// the cognitive-complexity ceiling is tightened. Cyclomatic complexity is
	// checked by the separate pinned Go analyzer; SonarGo has no S1541 rule.
	parameters := url.Values{"key": {profile.Key}, "rule": {"go:S3776"}, "params": {"threshold=10"}}
	if err := client.call(ctx, http.MethodPost, "/api/qualityprofiles/activate_rule", parameters, nil); err != nil {
		return profile, errors.New("sonar complexity rule configuration failed")
	}
	return profile, nil
}
