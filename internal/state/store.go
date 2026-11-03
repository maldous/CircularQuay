// Package state stores redacted activation metadata and serializes local reconciliation.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// Record has no value-bearing fields.
type Record struct {
	Workload    string                  `json:"workload"`
	Generation  string                  `json:"generation,omitempty"`
	State       string                  `json:"state"`
	Mode        string                  `json:"mode"`
	Offline     bool                    `json:"offline"`
	LastChecked time.Time               `json:"last_checked"`
	LastSuccess time.Time               `json:"last_success,omitempty"`
	Error       fault.Kind              `json:"error,omitempty"`
	Targets     map[string]TargetRecord `json:"targets,omitempty"`
	Health      string                  `json:"health,omitempty"`
	Receipts    []model.Receipt         `json:"receipts,omitempty"`
}

// TargetRecord records one target's projection and requested service transition.
// Current identifies entries from the reconciliation named by Record.Generation;
// retained false entries belong to an earlier, unselected reconciliation. An
// activated receipt is not evidence that the consumer passed a health probe.
type TargetRecord struct {
	State       string     `json:"state"`
	Generation  string     `json:"generation,omitempty"`
	Current     bool       `json:"current,omitempty"`
	AppliedAt   time.Time  `json:"applied_at,omitempty"`
	ActivatedAt time.Time  `json:"activated_at,omitempty"`
	Error       fault.Kind `json:"error,omitempty"`
}

// Store owns a private metadata directory.
type Store struct{ Directory string }

func (s Store) path(name string) (string, error) {
	if !namePattern.MatchString(name) || s.Directory == "" {
		return "", fault.New(fault.Invalid, "state path")
	}
	directory, err := filepath.Abs(s.Directory)
	if err != nil {
		return "", fault.New(fault.Invalid, "state directory")
	}
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		return "", err
	}
	return filepath.Join(directory, name+".json"), nil
}

// Read returns metadata; absence is explicit and contains no invented success.
func (s Store) Read(name string) (Record, error) {
	p, err := s.path(name)
	if err != nil {
		return Record{}, err
	}
	if _, err := os.Lstat(p); os.IsNotExist(err) {
		return Record{Workload: name, State: "not_provisioned"}, nil
	}
	data, err := platform.ReadProtected(p, true)
	if err != nil {
		return Record{}, err
	}
	if len(data) > 1<<20 {
		return Record{}, fault.New(fault.Trust, "state size")
	}
	var r Record
	if jsondocument.Decode(data, &r) != nil || r.Workload != name {
		return Record{}, fault.New(fault.Trust, "state record")
	}
	return r, nil
}

// Write atomically replaces metadata with restrictive permissions from creation.
func (s Store) Write(r Record) error {
	p, err := s.path(r.Workload)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fault.New(fault.Apply, "state encode")
	}
	if len(b) > 1<<20 {
		return fault.New(fault.Invalid, "state size")
	}
	return platform.WriteProtected(p, b)
}

// Lock obtains a nonblocking native process lock. A crash releases the OS lock.
func (s Store) Lock(name string) (func() error, error) {
	p, err := s.path(name)
	if err != nil {
		return nil, err
	}
	f, err := platform.OpenLockFile(p + ".lock")
	if err != nil {
		return nil, err
	}
	if err = lock(f); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return nil, fault.New(fault.Apply, "state lock cleanup")
		}
		return nil, fault.New(fault.Unavailable, "reconciliation busy")
	}
	return func() error {
		e := unlock(f)
		c := f.Close()
		if e != nil || c != nil {
			return fault.New(fault.Apply, "state unlock")
		}
		return nil
	}, nil
}
