// Package state stores redacted activation metadata and serializes local reconciliation.
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// Record has no value-bearing fields.
type Record struct {
	Workload    string          `json:"workload"`
	Generation  string          `json:"generation,omitempty"`
	State       string          `json:"state"`
	Mode        string          `json:"mode"`
	Offline     bool            `json:"offline"`
	LastChecked time.Time       `json:"last_checked"`
	LastSuccess time.Time       `json:"last_success,omitempty"`
	Error       fault.Kind      `json:"error,omitempty"`
	Receipts    []model.Receipt `json:"receipts,omitempty"`
}

// Store owns a private metadata directory.
type Store struct{ Directory string }

func (s Store) path(name string) (string, error) {
	if !namePattern.MatchString(name) || s.Directory == "" {
		return "", fault.New(fault.Invalid, "state path")
	}
	if err := os.MkdirAll(s.Directory, 0o700); err != nil {
		return "", fault.New(fault.Apply, "state directory")
	}
	info, err := os.Lstat(s.Directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return "", fault.New(fault.Trust, "state directory permissions")
	}
	return filepath.Join(s.Directory, name+".json"), nil
}

// Read returns metadata; absence is explicit and contains no invented success.
func (s Store) Read(name string) (Record, error) {
	p, err := s.path(name)
	if err != nil {
		return Record{}, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return Record{Workload: name, State: "not_provisioned"}, nil
	}
	if err != nil {
		return Record{}, fault.New(fault.Apply, "state read")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() > 1<<20 {
		return Record{}, fault.New(fault.Trust, "state size")
	}
	var r Record
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || r.Workload != name {
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
	f, err := os.CreateTemp(s.Directory, ".state-*")
	if err != nil {
		return fault.New(fault.Apply, "state stage")
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return fault.New(fault.Apply, "state write")
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return fault.New(fault.Apply, "state sync")
	}
	if err = f.Close(); err != nil {
		return fault.New(fault.Apply, "state close")
	}
	if err = os.Rename(tmp, p); err != nil {
		return fault.New(fault.Apply, "state promote")
	}
	return nil
}

// Lock obtains a nonblocking native process lock. A crash releases the OS lock.
func (s Store) Lock(name string) (func() error, error) {
	p, err := s.path(name)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(p+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fault.New(fault.Apply, "state lock")
	}
	if err = lock(f); err != nil {
		f.Close()
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
