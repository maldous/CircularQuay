//go:build !windows

package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func trustedAliasProgram(t *testing.T) string {
	t.Helper()
	path := "/usr/bin/true"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("native synthetic exit program unavailable")
	}
	if err := ValidateExecutable(path); err != nil {
		t.Fatal("native synthetic exit program not trusted")
	}
	return path
}

func selectionSymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func makeAliasMutable(t *testing.T, directory string) {
	t.Helper()
	if err := os.Chmod(directory, 0o777); err != nil {
		t.Fatal(err)
	}
}

func assertAliasRejected(t *testing.T, path string) {
	t.Helper()
	if err := ValidateExecutable(path); !fault.Is(err, fault.Trust) {
		t.Fatalf("mutable executable alias accepted: %v", err)
	}
}

func TestExecutableAliasRejectsMutableLexicalParent(t *testing.T) {
	program := trustedAliasProgram(t)
	root := t.TempDir()
	alias := filepath.Join(root, "helper")
	selectionSymlink(t, program, alias)
	if err := ValidateExecutable(alias); err != nil {
		t.Fatalf("protected executable alias rejected: %v", err)
	}
	makeAliasMutable(t, root)
	assertAliasRejected(t, alias)
}

func TestExecutableAliasChecksIntermediateSymlinkTargets(t *testing.T) {
	program := trustedAliasProgram(t)
	root := t.TempDir()
	mutable := filepath.Join(root, "mutable")
	if err := os.Mkdir(mutable, 0o700); err != nil {
		t.Fatal(err)
	}
	hop := filepath.Join(mutable, "hop")
	selectionSymlink(t, program, hop)
	outer := filepath.Join(root, "outer")
	selectionSymlink(t, hop, outer)
	if err := ValidateExecutable(outer); err != nil {
		t.Fatalf("protected intermediate alias rejected: %v", err)
	}
	makeAliasMutable(t, mutable)
	assertAliasRejected(t, outer)
}

func TestExecutableAliasDoesNotCleanAwayTraversedMutableComponents(t *testing.T) {
	program := trustedAliasProgram(t)
	root := t.TempDir()
	step := filepath.Join(root, "mutable", "step")
	if err := os.MkdirAll(step, 0o700); err != nil {
		t.Fatal(err)
	}
	selectionSymlink(t, program, filepath.Join(root, "target"))
	alias := step + "/../../target"
	if err := ValidateExecutable(alias); err != nil {
		t.Fatalf("protected dot-dot traversal rejected: %v", err)
	}
	makeAliasMutable(t, filepath.Dir(step))
	assertAliasRejected(t, alias)
}

func TestPinnedScriptAliasPreservesConfiguredInvocationPath(t *testing.T) {
	root := t.TempDir()
	program := filepath.Join(root, "program")
	contents := []byte("#!/bin/sh\nprintf '%s' \"$0\"\n")
	if err := os.WriteFile(program, contents, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "configured-alias")
	selectionSymlink(t, program, alias)
	digest := sha256.Sum256(contents)
	work := model.Workload{Command: []string{alias}, CommandSHA256: hex.EncodeToString(digest[:]), Directory: root}
	var output bytes.Buffer
	code, err := RunWithIO(context.Background(), work, model.Generation{}, nil, nil, &output, io.Discard)
	if err != nil || code != 0 || output.String() != alias {
		t.Fatalf("configured script invocation changed: code=%d error=%v output=%q", code, err, output.String())
	}
}
