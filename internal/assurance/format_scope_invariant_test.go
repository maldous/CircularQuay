package assurance

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFormattingScopeInvariantAdmittedFilesCannotTriggerEmptyScope(t *testing.T) {
	root := privateAssuranceDirectory(t, "format-scope-invariant")
	path := sourceInputOwnedFile(t, root, "owned.go", []byte("package fixture\n"))
	source := &SourceIdentity{Files: []SourceFile{{Path: "owned.go"}}}
	scope, err := sourceGoScope(root, source)
	if err != nil || !reflect.DeepEqual(scope.files, []string{path}) || len(scope.packages) != 1 {
		t.Fatal("admitted Go scope lost its nonempty immutable file list")
	}
	calls := 0
	command := func(_ context.Context, binary string, args []string, directory string, _ []string) commandResult {
		calls++
		assertFormattingScopeArguments(t, binary, args, directory, root)
		return commandResult{}
	}
	check, err := checkFormatting(context.Background(), command, root, "", nil, scope.files)
	if err != nil || calls != 1 || check.Name != "format" || !check.Required || check.Status != "passed" {
		t.Fatal("nonempty scope did not produce its exact required formatting check")
	}
}

func assertFormattingScopeArguments(t *testing.T, binary string, args []string, directory, root string) {
	t.Helper()
	if binary != "gofmt" || !reflect.DeepEqual(args, []string{"-l", filepath.Join(root, "owned.go")}) || directory != root {
		t.Fatal("formatting escaped its exact admitted inputs")
	}
}
