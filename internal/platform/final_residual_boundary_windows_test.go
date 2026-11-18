//go:build windows

package platform

import (
 "os"
 "path/filepath"
 "reflect"
 "testing"

 "github.com/maldous/CircularQuay/internal/fault"
)

func TestFinalResidualWindowsLaunchInputsPropagatesDirectoryRefusal(t *testing.T) {
 for _, phase := range []string{"working", "PATH"} {
  t.Run(phase, func(t *testing.T) { finalResidualWindowsDirectory(t, phase) })
 }
}

func finalResidualWindowsDirectory(t *testing.T, phase string) {
 t.Helper()
 work, executable, pathDirectory := protectedLaunchFixture(t, filepath.Join(t.TempDir(), "launch"))
 work.Command = []string{executable}
 work.Environment = map[string]string{"PATH":pathDirectory}
 generation := secretGeneration()
 args, env, directory, err := launchInputs(work, generation, nil)
 finalResidualWindowsAdmission(t, args, env, directory, err, work.Command)
 operation := "absolute trusted directory required"
 if phase == "working" { work.Directory = "relative" } else {
  work.Environment["PATH"] = "relative"
  operation = "absolute PATH directory required"
 }
 args, env, directory, err = launchInputs(work, generation, nil)
 if args != nil || env != nil || directory != "" { t.Fatal("refused launch returned partial inputs") }
 expected := &fault.Error{Kind:fault.Trust, Operation:operation}
 if !reflect.DeepEqual(err, expected) { t.Fatal("directory refusal lost exact fault", err) }
 finalResidualWindowsBytes(t, executable)
}

func finalResidualWindowsAdmission(t *testing.T, args, env []string, directory string, err error, expected []string) {
 t.Helper()
 if err != nil || !reflect.DeepEqual(args, expected) || len(env) == 0 || directory == "" { t.Fatal("owned inert launch admission failed", err) }
}

func finalResidualWindowsBytes(t *testing.T, executable string) {
 t.Helper()
 data, err := os.ReadFile(executable)
 if err != nil || string(data) != "synthetic" { t.Fatal("inert executable changed") }
}
