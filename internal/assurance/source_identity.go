package assurance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	maxSourceFiles     = 10_000
	maxSourceBytes     = 32 << 20
	maxSourceFileBytes = 1 << 20
	sourceBindTimeout  = 2 * time.Minute
)

var errSourceChangedDuringRead = errors.New("source file changed during read")

// SourceIdentity binds audit evidence to the Git tree and the checked-out
// bytes observed before or after target tools run. It detects ordinary source
// drift; it is not an atomic filesystem snapshot or an OS sandbox.
type SourceIdentity struct {
	Commit         string       `json:"commit"`
	Tree           string       `json:"tree"`
	ManifestSHA256 string       `json:"manifest_sha256"`
	Files          []SourceFile `json:"files"`
}

// SourceFile records one tracked regular file and its Git object identity.
// SHA256 is calculated from the checked-out bytes after they match the known
// Git blob bytes.
type SourceFile struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Blob   string `json:"blob"`
	SHA256 string `json:"sha256"`
}

func bindSource(ctx context.Context, command commandFunc, root, output string, environment []string) (SourceIdentity, error) {
	ctx, cancel := context.WithTimeout(ctx, sourceBindTimeout)
	defer cancel()
	if !physicalSourceRoot(root) {
		return SourceIdentity{}, errors.New("source root must be a physical directory")
	}
	if err := sourceRepositoryRoot(ctx, command, root, environment); err != nil {
		return SourceIdentity{}, err
	}
	commit, err := gitValue(ctx, command, root, environment, "rev-parse", "HEAD")
	if err != nil || !revisionPattern.MatchString(commit) {
		return SourceIdentity{}, errors.New("source revision unavailable")
	}
	tree, err := gitValue(ctx, command, root, environment, "rev-parse", "HEAD^{tree}")
	if err != nil || !revisionPattern.MatchString(tree) {
		return SourceIdentity{}, errors.New("source tree unavailable")
	}
	entries, err := gitTreeEntries(ctx, command, root, environment)
	if err != nil {
		return SourceIdentity{}, err
	}
	files, err := bindTrackedFiles(ctx, command, root, environment, entries)
	if err != nil {
		return SourceIdentity{}, err
	}
	if err = rejectUnexpectedFiles(root, output, files); err != nil {
		return SourceIdentity{}, err
	}
	return SourceIdentity{Commit: commit, Tree: tree, Files: files, ManifestSHA256: manifestSHA256(files)}, nil
}

func physicalSourceRoot(root string) bool {
	info, err := os.Lstat(root)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func sourceRepositoryRoot(ctx context.Context, command commandFunc, root string, environment []string) error {
	bare, err := gitValue(ctx, command, root, environment, "rev-parse", "--is-bare-repository")
	if err != nil || bare != "false" {
		return errors.New("source must be a nonbare repository")
	}
	top, err := gitValue(ctx, command, root, environment, "rev-parse", "--show-toplevel")
	if err != nil || !sameDirectory(root, top) {
		return errors.New("source must be repository top level")
	}
	prefix, err := gitValue(ctx, command, root, environment, "rev-parse", "--show-prefix")
	if err != nil || prefix != "" {
		return errors.New("source must not select a repository subdirectory")
	}
	return nil
}

func sameDirectory(first, second string) bool {
	left, leftErr := os.Stat(first)
	right, rightErr := os.Stat(second)
	return leftErr == nil && rightErr == nil && left.IsDir() && right.IsDir() && os.SameFile(left, right)
}

type treeEntry struct {
	mode string
	blob string
	path string
}

func gitTreeEntries(ctx context.Context, command commandFunc, root string, environment []string) ([]treeEntry, error) {
	output, err := gitOutput(ctx, command, root, environment, "ls-tree", "-r", "-z", "HEAD")
	if err != nil || len(output) == 0 {
		return nil, errors.New("source tree entries unavailable")
	}
	var entries []treeEntry
	for _, record := range strings.Split(string(output), "\x00") {
		if record == "" {
			continue
		}
		entry, err := parseTreeEntry(record)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		if len(entries) > maxSourceFiles {
			return nil, errors.New("source tree exceeds file budget")
		}
	}
	return entries, nil
}

func parseTreeEntry(record string) (treeEntry, error) {
	parts := strings.SplitN(record, "\t", 2)
	fields := strings.Fields(parts[0])
	if len(parts) != 2 || len(fields) != 3 || fields[1] != "blob" || !safeRepositoryPath(parts[1]) {
		return treeEntry{}, errors.New("invalid source tree entry")
	}
	if fields[0] != "100644" && fields[0] != "100755" {
		return treeEntry{}, errors.New("source tree contains nonregular file")
	}
	if !revisionPattern.MatchString(fields[2]) {
		return treeEntry{}, errors.New("invalid source blob identity")
	}
	return treeEntry{mode: fields[0], blob: fields[2], path: parts[1]}, nil
}

func safeRepositoryPath(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || hasUnsafePathRune(path) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func hasUnsafePathRune(path string) bool {
	for _, rune := range path {
		if rune < 0x20 || rune == 0x7f || rune == '\\' || rune == ':' {
			return true
		}
	}
	return false
}

func bindTrackedFiles(ctx context.Context, command commandFunc, root string, environment []string, entries []treeEntry) ([]SourceFile, error) {
	files := make([]SourceFile, 0, len(entries))
	budget := sourceBudget{}
	for _, entry := range entries {
		file, err := bindTrackedFile(ctx, command, root, environment, entry, &budget)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

type sourceBudget struct{ bytes int64 }

func (b *sourceBudget) add(size int64) error {
	if size < 0 || size > maxSourceFileBytes || size > maxSourceBytes-b.bytes {
		return errors.New("source tree exceeds byte budget")
	}
	b.bytes += size
	return nil
}

func bindTrackedFile(ctx context.Context, command commandFunc, root string, environment []string, entry treeEntry, budget *sourceBudget) (SourceFile, error) {
	path := filepath.Join(root, filepath.FromSlash(entry.path))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || !matchesTrackedMode(info.Mode(), entry.mode) || budget.add(info.Size()) != nil {
		return SourceFile{}, errors.New("tracked source file differs from tree")
	}
	known, err := gitOutput(ctx, command, root, environment, "cat-file", "blob", entry.blob)
	if err != nil {
		return SourceFile{}, errors.New("source blob unavailable")
	}
	actual, err := fileSHA256(root, entry.path, info)
	if err != nil || sha256.Sum256(known) != actual {
		return SourceFile{}, errors.New("tracked source bytes differ from tree")
	}
	return SourceFile{Path: entry.path, Mode: entry.mode, Blob: entry.blob, SHA256: hex.EncodeToString(actual[:])}, nil
}

func matchesTrackedMode(mode fs.FileMode, gitMode string) bool {
	return runtime.GOOS == "windows" || executableMode(mode) == (gitMode == "100755")
}

func executableMode(mode fs.FileMode) bool { return mode&0111 != 0 }

func rejectUnexpectedFiles(root, output string, files []SourceFile) error {
	tracked := make(map[string]bool, len(files))
	for _, file := range files {
		tracked[file.Path] = true
	}
	inventory := sourceFileInventory{root: root, output: output, tracked: tracked}
	return filepath.WalkDir(root, inventory.visit)
}

type sourceFileInventory struct {
	root    string
	output  string
	tracked map[string]bool
}

func (v sourceFileInventory) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return errors.New("source inventory unavailable")
	}
	if path == v.root {
		return nil
	}
	if path == filepath.Join(v.root, ".git") {
		return skipGitMetadata(v.root, path, entry)
	}
	if path == v.output {
		return skipSourceExclusion(entry)
	}
	return v.validate(path, entry)
}

func skipSourceExclusion(entry fs.DirEntry) error {
	if entry.IsDir() {
		return filepath.SkipDir
	}
	return errors.New("invalid source exclusion")
}

func skipGitMetadata(root, path string, entry fs.DirEntry) error {
	if entry.Type()&os.ModeSymlink != 0 {
		return errors.New("source git metadata is a symlink")
	}
	if entry.IsDir() {
		return filepath.SkipDir
	}
	if !entry.Type().IsRegular() || !validGitWorktreeFile(root, path) {
		return errors.New("invalid source git metadata")
	}
	return nil
}

func validGitWorktreeFile(root, path string) bool {
	data, err := readRootFile(root, filepath.Base(path), 4096)
	if err != nil || len(data) > 4096 {
		return false
	}
	value := strings.TrimSuffix(string(data), "\n")
	if !strings.HasPrefix(value, "gitdir: ") || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	directory := strings.TrimPrefix(value, "gitdir: ")
	if !filepath.IsAbs(directory) {
		directory = filepath.Join(root, directory)
	}
	metadata, err := os.OpenRoot(filepath.Clean(directory))
	if err != nil {
		return false
	}
	return metadata.Close() == nil
}

func (v sourceFileInventory) validate(path string, entry fs.DirEntry) error {
	relative, err := filepath.Rel(v.root, path)
	if err != nil || !safeRepositoryPath(filepath.ToSlash(relative)) {
		return errors.New("invalid source path")
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return errors.New("source symlink forbidden")
	}
	if entry.IsDir() || v.tracked[filepath.ToSlash(relative)] {
		return nil
	}
	return errors.New("unexpected source file")
}

func manifestSHA256(files []SourceFile) string {
	hash := sha256.New()
	for _, file := range files {
		_, _ = io.WriteString(hash, file.Path+"\x00"+file.Mode+"\x00"+file.Blob+"\x00"+file.SHA256+"\n")
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func readRootFile(directory, name string, maximum int64) ([]byte, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, maximum+1))
}

func fileSHA256(root, name string, expected os.FileInfo) ([sha256.Size]byte, error) {
	if expected.Size() < 0 || expected.Size() > maxSourceFileBytes {
		return [sha256.Size]byte{}, errors.New("source file exceeds byte budget")
	}
	directory, file, err := openBoundSourceFile(root, name, expected)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer directory.Close()
	defer file.Close()
	sum, err := boundedFileSHA256(file, expected.Size())
	if err != nil || !fileStillMatches(file, expected) {
		return [sha256.Size]byte{}, errSourceChangedDuringRead
	}
	return sum, nil
}

func openBoundSourceFile(root, name string, expected os.FileInfo) (*os.Root, *os.File, error) {
	directory, err := os.OpenRoot(root)
	if err != nil {
		return nil, nil, err
	}
	file, err := directory.Open(filepath.FromSlash(name))
	if err != nil {
		_ = directory.Close()
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil || !matchesNativeFileState(expected, info) {
		_ = file.Close()
		_ = directory.Close()
		return nil, nil, errSourceChangedDuringRead
	}
	return directory, file, nil
}

func boundedFileSHA256(file *os.File, expectedSize int64) ([sha256.Size]byte, error) {
	hash := sha256.New()
	read, err := io.Copy(hash, io.LimitReader(file, maxSourceFileBytes+1))
	if err != nil || read != expectedSize {
		return [sha256.Size]byte{}, errSourceChangedDuringRead
	}
	var sum [sha256.Size]byte
	copy(sum[:], hash.Sum(nil))
	return sum, nil
}

func fileStillMatches(file *os.File, expected os.FileInfo) bool {
	after, err := file.Stat()
	return err == nil && matchesNativeFileState(expected, after)
}

func matchesNativeFileState(expected, actual os.FileInfo) bool {
	if !os.SameFile(expected, actual) || expected.Size() != actual.Size() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return executableMode(expected.Mode()) == executableMode(actual.Mode())
}

func gitValue(ctx context.Context, command commandFunc, root string, environment []string, args ...string) (string, error) {
	output, err := gitOutput(ctx, command, root, environment, args...)
	return strings.TrimSpace(string(output)), err
}

func gitOutput(ctx context.Context, command commandFunc, root string, environment []string, args ...string) ([]byte, error) {
	fixed := append([]string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false"}, args...)
	result := command(ctx, "git", fixed, root, environment)
	if result.missing || result.exitCode != 0 {
		return nil, errors.New("source git command failed")
	}
	return result.output, nil
}
