package assurance

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestSourceInputContractGitAdmissionFailuresReturnWholeZero(t *testing.T) {
	for _, stage := range []string{"bare", "prefix", "commit", "tree", "entries"} {
		t.Run(stage, func(t *testing.T) { sourceInputGitRefusal(t, stage) })
	}
}

func sourceInputGitRefusal(t *testing.T, stage string) {
	t.Helper()
	root := privateAssuranceDirectory(t, "source")
	calls := 0
	command := func(_ context.Context, binary string, args []string, directory string, _ []string) commandResult {
		calls++
		if binary != "git" || directory != root || calls > 6 {
			t.Fatal("source refusal escaped the bounded fake Git sequence")
		}
		return commandResult{output: []byte(sourceInputGitReply(t, root, stage, args))}
	}
	identity, err := bindSource(context.Background(), command, root, "", nil)
	if err == nil || !reflect.DeepEqual(identity, SourceIdentity{}) {
		t.Fatal("invalid Git identity produced partial source authority")
	}
}

func sourceInputGitReply(t *testing.T, root, stage string, args []string) string {
	t.Helper()
	selected := strings.Join(args[4:], " ")
	replies := map[string]string{
		"rev-parse --is-bare-repository": "false", "rev-parse --show-toplevel": root,
		"rev-parse --show-prefix": "", "rev-parse HEAD": strings.Repeat("a", 40),
		"rev-parse HEAD^{tree}": strings.Repeat("b", 40), "ls-tree -r -z HEAD": "malformed\x00",
	}
	failures := map[string]string{"bare": "rev-parse --is-bare-repository", "prefix": "rev-parse --show-prefix", "commit": "rev-parse HEAD", "tree": "rev-parse HEAD^{tree}"}
	value, exists := replies[selected]
	if !exists {
		t.Fatal("unexpected Git subcommand")
	}
	if selected == failures[stage] {
		return "invalid"
	}
	return value
}

func TestSourceInputContractInventoriesRejectCallbackErrorsAndInvalidExclusions(t *testing.T) {
	sentinel := errors.New("synthetic callback failure")
	if err := (sourceFileInventory{}).visit("unused", nil, sentinel); err == nil || err.Error() != "source inventory unavailable" {
		t.Fatal("source enumeration failure was not sanitized")
	}
	if err := (assetValidator{}).visit("unused", nil, sentinel); err == nil || err.Error() != "repository enumeration failed" {
		t.Fatal("repository enumeration failure was not sanitized")
	}
	if err := (&sourceInventory{}).visit("unused", nil, sentinel); err == nil || err.Error() != "source enumeration failed" {
		t.Fatal("Go enumeration failure was not sanitized")
	}
	root := privateAssuranceDirectory(t, "source")
	name := sourceInputOwnedFile(t, root, "audit-results", []byte("not a directory"))
	entry := sourceInputEntry(t, name)
	if err := skipSourceExclusion(entry); err == nil {
		t.Fatal("a regular file was silently excluded as an output directory")
	}
	if err := skipGitMetadata(root, name, entry); err == nil {
		t.Fatal("an invalid regular Git metadata marker was accepted")
	}
}

func sourceInputEntry(t *testing.T, path string) fs.DirEntry {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fs.FileInfoToDirEntry(info)
}

func sourceInputOwnedFile(t *testing.T, root, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSourceInputContractPathsAndUnreadableRootsReturnNoBytes(t *testing.T) {
	root := privateAssuranceDirectory(t, "source")
	name := sourceInputOwnedFile(t, root, "owned.go", []byte("package fixture\n"))
	entry := sourceInputEntry(t, name)
	inventory := sourceFileInventory{root: root, tracked: map[string]bool{"owned.go": true}}
	if err := inventory.validate(filepath.Join(root, "..", "outside.go"), entry); err == nil {
		t.Fatal("outside-root inventory path was admitted")
	}
	if data, err := readRootFile(filepath.Join(root, "absent"), "owned.go", 32); err == nil || data != nil {
		t.Fatal("absent root produced bytes")
	}
	if data, err := readRootFile(root, "absent.go", 32); err == nil || data != nil {
		t.Fatal("absent rooted file produced bytes")
	}
}

func TestSourceInputContractRuleInputsRejectRelativeOverlapAndEmpty(t *testing.T) {
	if sum, err := bindToolInput("file", "relative", "", ""); err == nil || sum != "" {
		t.Fatal("relative rule input produced a digest")
	}
	root := privateAssuranceDirectory(t, "rules")
	if sum, err := bindToolInputFile(filepath.Join(root, "absent"), "", ""); err == nil || sum != "" {
		t.Fatal("missing rule file produced a digest")
	}
	file := sourceInputOwnedFile(t, root, "rules.json", []byte("owned rules"))
	if sum, err := bindToolInputFile(file, root, ""); err == nil || sum != "" {
		t.Fatal("source-overlapping rule file produced a digest")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	sourceInputTreeRefusals(t, root)
}

func sourceInputTreeRefusals(t *testing.T, root string) {
	t.Helper()
	if sum, err := bindToolInputTree(root, root, ""); err == nil || sum != "" {
		t.Fatal("source-overlapping rule tree produced a digest")
	}
	if sum, err := bindToolInputTree(root, "", ""); err == nil || sum != "" {
		t.Fatal("empty rule snapshot produced a digest")
	}
	if sum, err := digestInputFiles(root, []string{root}); err == nil || sum != "" {
		t.Fatal("root itself was accepted as a snapshot member")
	}
	if sum, err := digestInputFiles(root, []string{filepath.Join(root, "absent")}); err == nil || sum != "" {
		t.Fatal("absent snapshot member produced a digest")
	}
}

func TestSourceInputContractHeldDescriptorReadAndSizeFences(t *testing.T) {
	root := privateAssuranceDirectory(t, "rules")
	name := sourceInputOwnedFile(t, root, "rules.json", []byte("owned rules"))
	file, err := platform.OpenProtected(name, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := file.Seek(0, 2); err != nil {
		t.Fatal(err)
	}
	if sum, size, err := boundedInputFileDigest(file, name); err == nil || sum != "" || size != 0 {
		t.Fatal("short held-descriptor read produced an input digest")
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if sum, size, err := boundedInputFileDigest(file, name); err == nil || sum != "" || size != 0 {
		t.Fatal("closed descriptor produced an input digest")
	}
}

func TestSourceInputContractTaggedIdentityAndHeaderRefusals(t *testing.T) {
	root := privateAssuranceDirectory(t, "source")
	name := "internal/integration/fixture_test.go"
	sourceInputOwnedFile(t, root, name, []byte("package integration\n"))
	file := SourceFile{Path: name, SHA256: strings.Repeat("a", 64)}
	if err := validateTaggedIntegrationFile(root, file); err == nil {
		t.Fatal("tagged source with wrong manifest digest was accepted")
	}
	file.SHA256 = digest([]byte("package integration\n"))
	if err := validateTaggedIntegrationFile(root, file); err == nil {
		t.Fatal("tagged source without dedicated build constraint was accepted")
	}
	if hasIntegrationBuildConstraint("") || hasIntegrationBuildConstraint("// comment only") {
		t.Fatal("no integration constraint became a tagged lane")
	}
	if scope, err := sourceGoScope(root, nil); err == nil || !reflect.DeepEqual(scope, goSourceScope{}) {
		t.Fatal("absent source identity produced a Go scope")
	}
}
