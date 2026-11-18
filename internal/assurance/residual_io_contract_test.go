package assurance

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

type residualScopedFile struct {
	*os.File
	afterStat func() error
	closed    int
}

func (file *residualScopedFile) Stat() (os.FileInfo, error) {
	info, err := file.File.Stat()
	if err != nil {
		return nil, err
	}
	if file.afterStat != nil {
		if err := file.afterStat(); err != nil {
			return nil, err
		}
	}
	return info, nil
}

func (file *residualScopedFile) Close() error {
	file.closed++
	return file.File.Close()
}

func TestResidualIOContractScopedSourceReadDetectsRealCloseAndGrowth(t *testing.T) {
	for _, mutation := range []string{"none", "closed", "grown"} {
		t.Run(mutation, func(t *testing.T) { residualScopedRead(t, mutation) })
	}
}

func residualScopedRead(t *testing.T, mutation string) {
	t.Helper()
	root := privateAssuranceDirectory(t, "residual-source-read")
	original := []byte("package fixture\n")
	path := sourceInputOwnedFile(t, root, "owned.go", original)
	var held *residualScopedFile
	open := func(name string) (scopedSourceFile, error) {
		if name != path {
			t.Fatal("scoped reader selected an unrelated path")
		}
		file, err := platform.OpenProtected(name, false)
		if err != nil {
			return nil, err
		}
		held = &residualScopedFile{File: file}
		held.afterStat = residualReadMutation(file, path, mutation)
		return held, nil
	}
	data, err := boundedSourceReadWithOpen(path, open)
	assertResidualScopedRead(t, mutation, held, data, original, err)
}

func assertResidualScopedRead(t *testing.T, mutation string, held *residualScopedFile, data, original []byte, err error) {
	t.Helper()
	if held == nil || held.closed != 1 {
		t.Fatal("scoped read lost its exact descriptor closure")
	}
	if mutation == "none" {
		if err != nil || !bytes.Equal(data, original) {
			t.Fatal("stable scoped read control changed its bytes")
		}
		return
	}
	if err == nil || err.Error() != "source file read failed" || data != nil {
		t.Fatal("closed or grown source produced partial bytes")
	}
}

func residualReadMutation(file *os.File, path, mutation string) func() error {
	switch mutation {
	case "closed":
		return file.Close
	case "grown":
		return func() error { return os.Truncate(path, maxSourceFileBytes+1) }
	default:
		return nil
	}
}

func TestResidualIOContractMissingTaggedFileAndPrivateRuleRoot(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-missing-inputs")
	if data, err := boundedSourceRead(filepath.Join(root, "missing.go")); err == nil || data != nil {
		t.Fatal("missing protected source produced bytes")
	}
	err := validateTaggedIntegrationFile(root, SourceFile{Path: "missing_test.go"})
	if err == nil || err.Error() != "cannot read tagged integration file" {
		t.Fatal("missing tagged source did not retain its exact refusal")
	}
	if sum, err := bindToolInputTree(filepath.Join(root, "absent"), "", ""); err == nil || sum != "" {
		t.Fatal("missing private rules root produced an input digest")
	}
}

func TestResidualIOContractManifestCanonicalFailurePreservesInputBeforeAuthority(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-manifest")
	path := sourceInputOwnedFile(t, root, "manifest.json", []byte("{}"))
	calls := 0
	canonical := func(name string) (string, error) {
		calls++
		if name != path {
			t.Fatal("manifest canonicalization selected an unrelated input")
		}
		return "", os.ErrNotExist
	}
	name, data, err := readToolManifestWithCanonical(path, canonical)
	if err == nil || err.Error() != "invalid tool manifest location" || name != "" || data != nil || calls != 1 {
		t.Fatal("failed manifest canonicalization produced partial authority")
	}
	toolExecutionBoundaryBytes(t, path, []byte("{}"))
}

func TestResidualIOContractExecutableValidationCannotPublishIdentity(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-executable")
	original := []byte("owned inert tool")
	path := writeTool(t, filepath.Join(root, canonicalToolName("git")), string(original))
	seen := map[string]bool{}
	calls := 0
	validate := func(name string) error {
		calls++
		if name != path {
			t.Fatal("executable validation selected an unrelated input")
		}
		return errors.New("synthetic native permission refusal")
	}
	name, sum, err := verifyExecutableIdentityWithValidation(path, digest(original), "", "", seen, validate)
	if err == nil || err.Error() != "invalid tool protection" || name != "" || sum != "" || calls != 1 || len(seen) != 0 {
		t.Fatal("failed executable validation published identity or consumed admission")
	}
	toolExecutionBoundaryBytes(t, path, original)
}

func TestResidualIOContractPostCopyRecheckRejectsOnlyOwnedCopyDrift(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-helper-source")
	output := privateAssuranceDirectory(t, "residual-helper-output")
	original := []byte("owned inert native helper")
	path := writeTool(t, filepath.Join(root, canonicalToolName(sshKeygenHelperID)), string(original))
	helper := VerifiedHelper{ID: sshKeygenHelperID, Path: path, SHA256: digest(original)}
	tools := newQualifiedTools(VerifiedToolManifest{Helpers: []VerifiedHelper{helper}}, nil)
	calls := 0
	copy := func(binary helperBinary, target string) error {
		calls++
		if !reflect.DeepEqual(binary, helper.binary()) || target != filepath.Join(output, canonicalToolName(helper.ID)) {
			t.Fatal("helper copy escaped its explicit binding")
		}
		if err := copyHelperBinary(binary, target); err != nil {
			t.Fatal(err)
		}
		return os.WriteFile(target, []byte("changed owned copy"), 0700)
	}
	if err := tools.copyNativeHelpersWithCopy(output, copy); !errors.Is(err, os.ErrPermission) || calls != 1 || tools.helpersMatch() {
		t.Fatal("post-copy identity drift became an admitted native helper")
	}
	toolExecutionBoundaryBytes(t, path, original)
}

func TestResidualIOContractPostEnableRefusalPreservesOwnedBytes(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-helper-enable")
	original := []byte("owned inert helper")
	path := sourceInputOwnedFile(t, root, canonicalToolName("git"), original)
	calls := 0
	validate := func(name string) bool {
		calls++
		info, err := os.Stat(name)
		if err != nil || name != path || !info.Mode().IsRegular() {
			t.Fatal("post-enable validator did not receive the owned regular file")
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
			t.Fatal("post-enable validator ran before owner-only execute mode")
		}
		return false
	}
	if err := enableHelperWithValidation(path, validate); !errors.Is(err, os.ErrPermission) || calls != 1 {
		t.Fatal("post-enable validation failure was ignored")
	}
	toolExecutionBoundaryBytes(t, path, original)
}

func TestResidualIOContractReportPathFailurePreservesOriginalReport(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-report")
	initial := Report{Version: 1, Commit: "owned-original"}
	if err := WriteReport(root, initial); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "report.json")
	original, err := platform.ReadProtected(path, true)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	resolve := func(name string) (string, error) {
		calls++
		if name != path {
			t.Fatal("report path resolution selected an unrelated output")
		}
		return "", errors.New("private path marker")
	}
	err = writeReportWithPath(root, Report{Passed: true}, resolve)
	if err == nil || err.Error() != "invalid audit report path" || calls != 1 {
		t.Fatal("report path failure lost its fixed pre-write refusal")
	}
	retained, err := platform.ReadProtected(path, true)
	if err != nil || !bytes.Equal(retained, original) {
		t.Fatal("path failure changed the existing report")
	}
}

func TestResidualIOContractAssetRelativePathAndNameRefusals(t *testing.T) {
	root := privateAssuranceDirectory(t, "residual-asset")
	path := sourceInputOwnedFile(t, root, "bad name.go", []byte("package fixture\n"))
	entry := sourceInputEntry(t, path)
	validator := assetValidator{root: "relative", seen: map[string]bool{}}
	if err := validator.visit(path, entry, nil); err == nil || err.Error() != "invalid repository path" || len(validator.seen) != 0 {
		t.Fatal("relative root callback produced a repository path")
	}
	validator.root = root
	if err := validator.visit(path, entry, nil); err == nil || err.Error() != "invalid asset name" || len(validator.seen) != 0 {
		t.Fatal("invalid owned asset name was recorded as admitted")
	}
}

// Compile-time interface assertions keep the per-call adapters limited to their
// real reader and finite copy contracts rather than an unrestricted dispatcher.
var _ scopedSourceFile = (*os.File)(nil)
var _ scopedSourceFile = (*residualScopedFile)(nil)
