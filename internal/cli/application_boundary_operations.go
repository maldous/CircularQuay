package cli

import (
 "context"
 "os"
 "path/filepath"

 "github.com/maldous/CircularQuay/internal/state"
)

// These package-private, per-application path observations are never supplied
// by Runner input or serialized policy. Nil selects the unchanged OS defaults.
type applicationPathOperations struct {
 absolute func(string) (string, error)
 configurationDirectory func() (string, error)
}

func (a application) absolutePath(path string) (string, error) {
 if a.paths != nil && a.paths.absolute != nil { return a.paths.absolute(path) }
 return filepath.Abs(path)
}

func (a application) userConfigDirectory() (string, error) {
 if a.paths != nil && a.paths.configurationDirectory != nil { return a.paths.configurationDirectory() }
 return os.UserConfigDir()
}

type initializationOperations struct {
 mkdir func(string, os.FileMode) error
 write func(string, []byte) error
}

type configurationCommitOperations struct {
 unchanged func(string, string, bool) error
 context func(context.Context) error
 write func(string, []byte) error
}

type configurationFormatOperations struct {
 read func(string, bool) ([]byte, error)
 context func(context.Context) error
 write func(string, []byte) error
}

func (r targetReconciler) writeRecord(record state.Record) error {
 if r.write != nil { return r.write(record) }
 return r.store.Write(record)
}
