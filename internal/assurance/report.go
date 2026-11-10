package assurance

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

// Check records bounded metadata. Raw command output never becomes report content.
type Check struct {
	Name       string             `json:"name"`
	Status     string             `json:"status"`
	Required   bool               `json:"required"`
	Code       string             `json:"code,omitempty"`
	ExitCode   int                `json:"exit_code,omitempty"`
	DurationMS int64              `json:"duration_ms"`
	Failure    *GoFailureMetadata `json:"failure,omitempty"`
}

// GoFailureMetadata contains only source-bound metadata from a failed Go test
// or vet command. It never stores command arguments, endpoints, values, or
// diagnostic text.
type GoFailureMetadata struct {
	State     string              `json:"state"`
	Packages  []string            `json:"packages,omitempty"`
	Locations []GoFailureLocation `json:"locations,omitempty"`
}

// GoFailureLocation identifies a tracked public Go source line.
type GoFailureLocation struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// Report ties honest results to a source revision and the selected platform.
type Report struct {
	Version             int                 `json:"version"`
	Profile             string              `json:"profile"`
	Commit              string              `json:"commit,omitempty"`
	Platform            string              `json:"platform"`
	GoVersion           string              `json:"go_version,omitempty"`
	StartedAt           time.Time           `json:"started_at"`
	CompletedAt         time.Time           `json:"completed_at"`
	Passed              bool                `json:"passed"`
	ProductionQualified bool                `json:"production_qualified"`
	Coverage            Coverage            `json:"coverage"`
	Source              *SourceIdentity     `json:"source,omitempty"`
	Sonar               *SonarSummary       `json:"sonar,omitempty"`
	ToolManifest        *ToolManifestReport `json:"tool_manifest,omitempty"`
	Checks              []Check             `json:"checks"`
	Limitations         []string            `json:"limitations"`
}

// ToolManifestReport identifies the manifest and the fixed tool identities
// used by this audit. A passed probe only confirms the declared version output
// for that execution; it is not publisher authentication or sandbox evidence.
type ToolManifestReport struct {
	SHA256  string           `json:"sha256"`
	Tools   []ToolIdentity   `json:"tools"`
	Helpers []HelperIdentity `json:"helpers,omitempty"`
}

// HelperIdentity records verified bytes and whether a private helper copy was
// prepared. Prepared is not evidence of invocation or a successful version probe.
type HelperIdentity struct {
	VerifiedHelper
	Prepared         bool `json:"prepared"`
	IntegrityCurrent bool `json:"integrity_current"`
}

type ToolIdentity struct {
	ID      string     `json:"id"`
	Path    string     `json:"path"`
	SHA256  string     `json:"sha256"`
	Version string     `json:"declared_version"`
	Rules   *ToolRules `json:"declared_rules,omitempty"`
	Probe   ToolProbe  `json:"probe"`
}

func (r *Report) finish() {
	r.Passed = true
	for _, check := range r.Checks {
		if check.Required && check.Status != "passed" {
			r.Passed = false
		}
	}
	// Source analysis cannot attest native reboot tests, account grants or independent review.
	r.ProductionQualified = false
	r.CompletedAt = time.Now().UTC()
}

// WriteReport atomically writes private report metadata without raw tool diagnostics.
func WriteReport(directory string, report Report) error {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("invalid audit output directory")
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return errors.New("audit report encoding failed")
	}
	path, err := filepath.Abs(filepath.Join(directory, "report.json"))
	if err != nil {
		return errors.New("invalid audit report path")
	}
	if err = platform.WriteProtected(path, append(encoded, '\n')); err != nil {
		return errors.New("private audit report write failed")
	}
	return nil
}
