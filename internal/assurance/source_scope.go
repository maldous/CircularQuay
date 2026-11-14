package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"go/build/constraint"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maldous/CircularQuay/internal/platform"
)

const (
	taggedIntegrationDirectory = "internal/integration"
	maxBuildConstraintLines    = 64
	taggedIntegrationCheckName = "tagged_integration"
	taggedIntegrationCheckCode = "requires_qualified_tagged_integration_lane"
)

// goSourceScope is the manifest-bound native Go input set for every Go-aware
// audit tool. It intentionally names directories rather than using ./... so an
// audit-result tree inside Source cannot add a generated package to a check.
// A dedicated integration-only package is retained in taggedIntegrationPackages
// and must be qualified in its own fixture-bearing lane. It is not passed to an
// untagged Go command because that command cannot build the package at all.
type goSourceScope struct {
	packages                  []string
	coverage                  []string
	files                     []string
	taggedIntegrationPackages []string
}

type scopedGoFile struct {
	filename string
	argument string
	coverage string
	tagged   bool
}

func sourceGoScope(root string, source *SourceIdentity) (goSourceScope, error) {
	if source == nil {
		return goSourceScope{}, errors.New("source scope missing")
	}
	packages := map[string]bool{}
	coverage := map[string]bool{}
	taggedIntegration := map[string]bool{}
	files := make([]string, 0, len(source.Files))
	for _, file := range source.Files {
		scoped, included, err := scopeGoFile(root, file)
		if err != nil {
			return goSourceScope{}, errors.New("tagged integration source is invalid")
		}
		if !included {
			continue
		}
		files = append(files, scoped.filename)
		if scoped.tagged {
			taggedIntegration[scoped.argument] = true
			continue
		}
		packages[scoped.argument] = true
		coverage[scoped.coverage] = true
	}
	if len(packages) == 0 || len(files) == 0 {
		return goSourceScope{}, errors.New("source scope has no Go files")
	}
	return goSourceScope{
		packages:                  sortedScope(packages),
		coverage:                  sortedScope(coverage),
		files:                     sortedFiles(files),
		taggedIntegrationPackages: sortedScope(taggedIntegration),
	}, nil
}

func scopeGoFile(root string, file SourceFile) (scopedGoFile, bool, error) {
	if !strings.HasSuffix(file.Path, ".go") {
		return scopedGoFile{}, false, nil
	}
	directory := path.Dir(file.Path)
	argument := "."
	coverage := strings.TrimSuffix(modulePrefix, "/")
	if directory != "." {
		argument = "./" + directory
		coverage += "/" + directory
	}
	scoped := scopedGoFile{filename: filepath.Join(root, filepath.FromSlash(file.Path)), argument: argument, coverage: coverage}
	if !isTaggedIntegrationFile(file.Path) {
		return scoped, true, nil
	}
	if err := validateTaggedIntegrationFile(root, file); err != nil {
		return scopedGoFile{}, false, err
	}
	scoped.tagged = true
	return scoped, true, nil
}

func isTaggedIntegrationFile(name string) bool {
	return strings.HasPrefix(name, taggedIntegrationDirectory+"/")
}

// validateTaggedIntegrationFile keeps the build-tag exception finite. The
// source bytes must still match the manifest that binds the audit report, and
// only test files carrying the dedicated integration tag may use this lane.
func validateTaggedIntegrationFile(root string, file SourceFile) error {
	if root == "" || !strings.HasSuffix(file.Path, "_test.go") {
		return errors.New("invalid tagged integration file")
	}
	data, err := boundedSourceRead(filepath.Join(root, filepath.FromSlash(file.Path)))
	if err != nil {
		return errors.New("cannot read tagged integration file")
	}
	digest := sha256.Sum256(data)
	if file.SHA256 == "" || file.SHA256 != hex.EncodeToString(digest[:]) {
		return errors.New("tagged integration identity mismatch")
	}
	if !hasIntegrationBuildConstraint(string(data)) {
		return errors.New("tagged integration build constraint missing")
	}
	return nil
}

func boundedSourceRead(filename string) ([]byte, error) {
	opened, err := platform.OpenProtected(filename, false)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	info, err := opened.Stat()
	if err != nil || info.Size() < 0 || info.Size() > maxSourceFileBytes {
		return nil, errors.New("source file size invalid")
	}
	data, err := io.ReadAll(io.LimitReader(opened, maxSourceFileBytes+1))
	if err != nil || len(data) > maxSourceFileBytes {
		return nil, errors.New("source file read failed")
	}
	return data, nil
}

func hasIntegrationBuildConstraint(source string) bool {
	lines := strings.Split(source, "\n")
	for index, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if index >= maxBuildConstraintLines || !validIntegrationHeaderLine(line) {
			return false
		}
		if strings.HasPrefix(line, "//go:build") {
			return validIntegrationConstraint(lines, index, line)
		}
	}
	return false
}

func validIntegrationHeaderLine(line string) bool {
	return !strings.HasPrefix(line, "// +build") && (line == "" || strings.HasPrefix(line, "//"))
}

func validIntegrationConstraint(lines []string, index int, line string) bool {
	if index+1 >= len(lines) || strings.TrimSuffix(lines[index+1], "\r") != "" {
		return false
	}
	expression, err := constraint.Parse(line)
	return err == nil && expression.String() == "integration"
}

// taggedIntegrationQualificationCheck makes the required fixture-bearing lane
// visible in a source report. The source profile never turns an unavailable
// tagged run into an untagged coverage pass.
func taggedIntegrationQualificationCheck(scope goSourceScope) (Check, bool) {
	if len(scope.taggedIntegrationPackages) == 0 {
		return Check{}, false
	}
	return Check{
		Name:     taggedIntegrationCheckName,
		Required: true,
		Status:   "not_run",
		Code:     taggedIntegrationCheckCode,
	}, true
}

func sortedScope(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedFiles(files []string) []string {
	sort.Strings(files)
	return files
}

// sourcePackageArguments remains the scanner-facing form for callers that do
// not need coverage imports or manifest-owned filenames.
func sourcePackageArguments(scope goSourceScope) []string {
	return append([]string(nil), scope.packages...)
}
