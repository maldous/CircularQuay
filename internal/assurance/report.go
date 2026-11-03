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
	Name       string `json:"name"`
	Status     string `json:"status"`
	Required   bool   `json:"required"`
	Code       string `json:"code,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

// Report ties honest results to a source revision and the selected platform.
type Report struct {
	Version             int           `json:"version"`
	Profile             string        `json:"profile"`
	Commit              string        `json:"commit,omitempty"`
	Platform            string        `json:"platform"`
	GoVersion           string        `json:"go_version,omitempty"`
	StartedAt           time.Time     `json:"started_at"`
	CompletedAt         time.Time     `json:"completed_at"`
	Passed              bool          `json:"passed"`
	ProductionQualified bool          `json:"production_qualified"`
	Coverage            Coverage      `json:"coverage"`
	Sonar               *SonarSummary `json:"sonar,omitempty"`
	Checks              []Check       `json:"checks"`
	Limitations         []string      `json:"limitations"`
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
