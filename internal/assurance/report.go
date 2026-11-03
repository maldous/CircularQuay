package assurance

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
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
	file, err := os.CreateTemp(directory, ".report-*")
	if err != nil {
		return errors.New("audit report staging failed")
	}
	name := file.Name()
	defer func() { _ = os.Remove(name) }()
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(append(encoded, '\n'))
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		return errors.New("audit report write failed")
	}
	if err = os.Rename(name, filepath.Join(directory, "report.json")); err != nil {
		return errors.New("audit report activation failed")
	}
	return nil
}
