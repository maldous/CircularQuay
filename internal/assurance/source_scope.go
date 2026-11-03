package assurance

import (
	"errors"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// goSourceScope is the manifest-bound native Go input set for every Go-aware
// audit tool. It intentionally names directories rather than using ./... so an
// audit-result tree inside Source cannot add a generated package to a check.
// A directory whose files are all excluded by the native build remains named:
// the Go tool must reject that incomplete native scope rather than silently
// reducing the coverage denominator.
type goSourceScope struct {
	packages []string
	coverage []string
	files    []string
}

func sourceGoScope(root string, source *SourceIdentity) (goSourceScope, error) {
	if source == nil {
		return goSourceScope{}, errors.New("source scope missing")
	}
	packages := map[string]bool{}
	coverage := map[string]bool{}
	files := make([]string, 0, len(source.Files))
	for _, file := range source.Files {
		if !strings.HasSuffix(file.Path, ".go") {
			continue
		}
		directory := path.Dir(file.Path)
		argument := "."
		importPath := strings.TrimSuffix(modulePrefix, "/")
		if directory != "." {
			argument = "./" + directory
			importPath += "/" + directory
		}
		packages[argument] = true
		coverage[importPath] = true
		files = append(files, filepath.Join(root, filepath.FromSlash(file.Path)))
	}
	if len(packages) == 0 || len(files) == 0 {
		return goSourceScope{}, errors.New("source scope has no Go files")
	}
	return goSourceScope{packages: sortedScope(packages), coverage: sortedScope(coverage), files: sortedFiles(files)}, nil
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
func sourcePackageArguments(source *SourceIdentity) []string {
	scope, err := sourceGoScope("", source)
	if err != nil {
		return nil
	}
	return scope.packages
}
