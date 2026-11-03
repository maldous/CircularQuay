package assurance

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var requiredFiles = []string{"go.mod", "go.sum", "README.md", "RELEASE.md", "LICENSE", "SECURITY.md", "CONTRIBUTING.md", "CHANGELOG.md"}

func repositoryCheck(root string) Check {
	check := Check{Name: "repository_policy", Status: "passed", Required: true}
	for _, name := range requiredFiles {
		info, err := os.Lstat(filepath.Join(root, name))
		if err != nil || !info.Mode().IsRegular() {
			check.Status = "failed"
			check.Code = "required_metadata_missing"
			return check
		}
	}
	assets := assetValidator{root: root, seen: map[string]bool{}}
	err := filepath.WalkDir(root, assets.visit)
	if err != nil {
		check.Status = "failed"
		check.Code = "repository_naming_violation"
	}
	return check
}

type assetValidator struct {
	root string
	seen map[string]bool
}

func (v assetValidator) visit(name string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return errors.New("repository enumeration failed")
	}
	relative, err := filepath.Rel(v.root, name)
	if err != nil {
		return errors.New("invalid repository path")
	}
	if relative == "." {
		return nil
	}
	if entry.IsDir() && skipDirectory(entry.Name()) {
		return filepath.SkipDir
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return errors.New("repository symlink forbidden")
	}
	if !validAssetName(entry.Name()) {
		return errors.New("invalid asset name")
	}
	folded := strings.ToLower(filepath.ToSlash(relative))
	if v.seen[folded] {
		return errors.New("case collision")
	}
	v.seen[folded] = true
	return nil
}

func sourceFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.New("source enumeration failed")
		}
		if entry.IsDir() && skipDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("source symlink forbidden")
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, name)
		}
		return nil
	})
	if err != nil || len(files) == 0 {
		return nil, errors.New("no readable Go source")
	}
	return files, nil
}

func skipDirectory(name string) bool {
	switch name {
	case ".git", ".local", ".sonar", ".scannerwork", "vendor", "node_modules", "dist", "audit-results":
		return true
	}
	return false
}

func validAssetName(name string) bool {
	if name == "" || strings.TrimSpace(name) != name || strings.HasSuffix(name, ".") {
		return false
	}
	for _, letter := range name {
		if letter < 33 || letter > 126 || strings.ContainsRune(`<>:"\|?*`, letter) {
			return false
		}
	}
	stem := strings.ToUpper(strings.Split(name, ".")[0])
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}
