package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/platform"
)

// Event intentionally cannot contain a secret value, bearer token or provider body.
type Event struct {
	Time      time.Time `json:"time"`
	Subject   string    `json:"subject"`
	Client    string    `json:"client,omitempty"`
	Workload  string    `json:"workload"`
	Operation string    `json:"operation"`
	Key       string    `json:"key,omitempty"`
	Outcome   string    `json:"outcome"`
}

// Auditor must durably record required receipts before disclosure succeeds.
type Auditor interface {
	Record(context.Context, Event) error
}

// FileAuditor writes protected JSONL metadata without plaintext response bodies.
type FileAuditor struct {
	path string
}

// NewFileAuditor requires an existing protected parent; it creates no directories.
func NewFileAuditor(path string) (*FileAuditor, error) {
	if !filepath.IsAbs(path) {
		return nil, fault.New(fault.Invalid, "audit path")
	}
	parent, err := os.Lstat(filepath.Dir(path))
	if err != nil || !parent.IsDir() || parent.Mode()&os.ModeSymlink != 0 {
		return nil, fault.New(fault.Trust, "audit parent")
	}
	// The parent exists: the shared native helper validates without changing its
	// permissions. Windows checks its DACL; Unix checks owner-only access.
	if err := platform.EnsurePrivateDirectory(filepath.Dir(path)); err != nil {
		return nil, fault.New(fault.Trust, "audit parent")
	}
	if _, err := os.Lstat(path); err == nil {
		if err := platform.ValidateProtectedPath(path); err != nil {
			return nil, fault.New(fault.Trust, "audit file")
		}
	} else if !os.IsNotExist(err) {
		return nil, fault.New(fault.Trust, "audit file")
	}
	return &FileAuditor{path: path}, nil
}

func (a *FileAuditor) Record(ctx context.Context, event Event) (err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseAudit, ProviderKind: "mcp"})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if ctx.Err() != nil {
		return fault.New(fault.Unavailable, "audit cancelled")
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fault.New(fault.Invalid, "audit receipt")
	}
	if err := platform.AppendProtected(a.path, append(data, '\n')); err != nil {
		return fault.New(fault.Apply, "audit write")
	}
	return nil
}
