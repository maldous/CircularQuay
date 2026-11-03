package assurance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SonarSummary contains counts only; API response bodies are never published.
type SonarSummary struct {
	Gate        string  `json:"quality_gate"`
	Issues      int     `json:"issue_count"`
	Hotspots    int     `json:"review_required_hotspots"`
	Coverage    float64 `json:"coverage_percent"`
	Duplication float64 `json:"duplicated_lines_percent"`
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
		return errors.New("Sonar request unavailable")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("Sonar request rejected")
	}
	if destination == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		if err != nil {
			return errors.New("Sonar response failed")
		}
		return nil
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(destination); err != nil {
		return errors.New("invalid Sonar response")
	}
	return nil
}

func inspectSonar(ctx context.Context, endpoint, token, project string) (SonarSummary, error) {
	client := newSonarClient(endpoint, token, "")
	var summary SonarSummary
	var gate struct {
		ProjectStatus struct {
			Status string `json:"status"`
		} `json:"projectStatus"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/qualitygates/project_status", url.Values{"projectKey": {project}}, &gate); err != nil {
		return summary, err
	}
	summary.Gate = gate.ProjectStatus.Status
	var issues struct {
		Total  int `json:"total"`
		Paging struct {
			Total int `json:"total"`
		} `json:"paging"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/issues/search", url.Values{"componentKeys": {project}, "ps": {"1"}}, &issues); err != nil {
		return summary, err
	}
	summary.Issues = issues.Total
	if issues.Paging.Total > summary.Issues {
		summary.Issues = issues.Paging.Total
	}
	var hotspots struct {
		Paging struct {
			Total int `json:"total"`
		} `json:"paging"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/hotspots/search", url.Values{"projectKey": {project}, "status": {"TO_REVIEW"}, "ps": {"1"}}, &hotspots); err != nil {
		return summary, err
	}
	summary.Hotspots = hotspots.Paging.Total
	var measures struct {
		Component struct {
			Measures []struct {
				Metric string `json:"metric"`
				Value  string `json:"value"`
			} `json:"measures"`
		} `json:"component"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/measures/component", url.Values{"component": {project}, "metricKeys": {"coverage,duplicated_lines_density"}}, &measures); err != nil {
		return summary, err
	}
	seen := map[string]bool{}
	for _, measure := range measures.Component.Measures {
		number, err := strconv.ParseFloat(measure.Value, 64)
		if err != nil || number < 0 || number > 100 {
			return summary, errors.New("invalid Sonar metric")
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
		return summary, errors.New("Sonar coverage metrics missing")
	}
	return summary, nil
}

// InitSonar creates local bootstrap files without printing or accepting secret values.
func InitSonar(directory string) error {
	if directory == "" {
		return errors.New("Sonar state directory required")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return errors.New("Sonar state creation failed")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("invalid Sonar state directory")
	}
	if err = os.Chmod(directory, 0700); err != nil {
		return errors.New("Sonar state protection failed")
	}
	for _, name := range []string{"compose.env", "admin.password", "scanner.token"} {
		if _, err = os.Lstat(filepath.Join(directory, name)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("Sonar state already exists")
		}
	}
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
	return writeNewPrivate(filepath.Join(directory, "admin.password"), []byte("Cq9!"+password+"\n"))
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
	password, err := readToken(filepath.Join(directory, "admin.password"))
	if err != nil {
		return err
	}
	initial := newSonarClient(endpoint, "admin", "admin")
	// Failure can mean the dedicated server was initialized previously; validate the configured password below.
	_ = initial.call(ctx, http.MethodPost, "/api/users/change_password", url.Values{"login": {"admin"}, "previousPassword": {"admin"}, "password": {password}}, nil)
	client := newSonarClient(endpoint, "admin", password)
	var auth struct {
		Valid bool `json:"valid"`
	}
	if err = client.call(ctx, http.MethodGet, "/api/authentication/validate", nil, &auth); err != nil || !auth.Valid {
		return errors.New("Sonar administrator authentication failed")
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
	name, err := randomSecret()
	if err != nil {
		return err
	}
	var created struct {
		Token string `json:"token"`
	}
	if err = client.call(ctx, http.MethodPost, "/api/user_tokens/generate", url.Values{"name": {"cquay-audit-" + name[:12]}, "type": {"PROJECT_ANALYSIS_TOKEN"}, "projectKey": {project}}, &created); err != nil {
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
	if err := client.call(ctx, http.MethodPost, "/api/projects/create", url.Values{"project": {project}, "name": {"CircularQuay independent audit"}}, nil); err == nil {
		return nil
	}
	var component struct {
		Component struct {
			Key string `json:"key"`
		} `json:"component"`
	}
	if err := client.call(ctx, http.MethodGet, "/api/components/show", url.Values{"component": {project}}, &component); err != nil || component.Component.Key != project {
		return errors.New("Sonar project initialization failed")
	}
	return nil
}

func configureSonarGate(ctx context.Context, client *sonarClient, project string) error {
	const name = "CircularQuay Enterprise"
	if err := client.call(ctx, http.MethodPost, "/api/qualitygates/create", url.Values{"name": {name}}, nil); err != nil {
		return errors.New("Sonar gate must be created in a fresh audit instance")
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
		return errors.New("Sonar quality gate assignment failed")
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
		return errors.New("Sonar strict condition configuration failed")
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
			return errors.New("Sonar comprehensive profile assignment failed")
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
	return sonarQualityProfile{}, errors.New("Sonar comprehensive language profile unavailable")
}

func sonarComplexityProfile(ctx context.Context, client *sonarClient, base sonarQualityProfile) (sonarQualityProfile, error) {
	var profile sonarQualityProfile
	name := "CircularQuay Go comprehensive"
	if err := client.call(ctx, http.MethodPost, "/api/qualityprofiles/copy", url.Values{"fromKey": {base.Key}, "toName": {name}}, &profile); err != nil {
		return profile, err
	}
	if profile.Key == "" || profile.Name != name {
		return profile, errors.New("Sonar profile copy failed")
	}
	// Copying the complete built-in profile retains its recommended rules. Only
	// the cognitive-complexity ceiling is tightened. Cyclomatic complexity is
	// checked by the separate pinned Go analyzer; SonarGo has no S1541 rule.
	parameters := url.Values{"key": {profile.Key}, "rule": {"go:S3776"}, "params": {"threshold=10"}}
	if err := client.call(ctx, http.MethodPost, "/api/qualityprofiles/activate_rule", parameters, nil); err != nil {
		return profile, errors.New("Sonar complexity rule configuration failed")
	}
	return profile, nil
}
