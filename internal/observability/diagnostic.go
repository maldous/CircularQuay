package observability

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

const diagnosticFile = "observability.jsonl"

// diagnostic is an independent local health signal. It never writes stdout or
// stderr, so CLI JSON remains machine-readable. A write failure is retained in
// process health but cannot turn a successful workload into a retryable error.
type diagnostic struct {
	level string
	path  string

	mu       sync.Mutex
	writeErr uint64
}

type diagnosticRecord struct {
	Time        time.Time `json:"time"`
	Event       string    `json:"event"`
	Phase       Phase     `json:"phase,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	Outcome     Outcome   `json:"outcome,omitempty"`
	Correlation string    `json:"correlation,omitempty"`
}

func newDiagnostic(level, directory string) (*diagnostic, error) {
	if level == "off" {
		return &diagnostic{level: level}, nil
	}
	if directory == "" || !filepath.IsAbs(directory) {
		return nil, errDiagnosticDirectory
	}
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		return nil, errDiagnosticDirectory
	}
	return &diagnostic{level: level, path: filepath.Join(directory, diagnosticFile)}, nil
}

var errDiagnosticDirectory = &diagnosticError{}

type diagnosticError struct{}

func (*diagnosticError) Error() string { return "protected diagnostic directory required" }

func (d *diagnostic) record(ctx context.Context, record diagnosticRecord) {
	if d == nil || d.level == "off" || ctx.Err() != nil {
		return
	}
	if d.level == "error" && record.Outcome != OutcomeFailed && record.Outcome != OutcomeUnavailable && record.Outcome != OutcomeDenied && record.Outcome != OutcomeExpired && record.Outcome != OutcomeCancelled {
		return
	}
	data, err := json.Marshal(record)
	if err == nil {
		err = platform.AppendProtected(d.path, append(data, '\n'))
	}
	if err != nil {
		d.mu.Lock()
		d.writeErr++
		d.mu.Unlock()
	}
}

func (d *diagnostic) health() uint64 {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.writeErr
}
