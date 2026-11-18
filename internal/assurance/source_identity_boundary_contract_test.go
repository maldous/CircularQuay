package assurance

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSourceIdentityBoundaryContractTreeRecords(t *testing.T) {
	blob := strings.Repeat("a", 40)
	cases := []struct {
		name   string
		record string
		valid  bool
	}{
		{name: "regular", record: "100644 blob " + blob + "\tsource.go", valid: true},
		{name: "executable", record: "100755 blob " + blob + "\ttools/check", valid: true},
		{name: "missing separator", record: "100644 blob " + blob},
		{name: "extra header field", record: "100644 blob " + blob + " extra\tsource.go"},
		{name: "submodule", record: "160000 commit " + blob + "\tmodule"},
		{name: "symlink", record: "120000 blob " + blob + "\tlink"},
		{name: "short identity", record: "100644 blob " + blob[:39] + "\tsource.go"},
		{name: "nonhex identity", record: "100644 blob " + strings.Repeat("g", 40) + "\tsource.go"},
		{name: "control in path", record: "100644 blob " + blob + "\tsource\ngo"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			assertSourceIdentityBoundaryTreeRecord(t, item.record, item.valid, blob)
		})
	}
}

func assertSourceIdentityBoundaryTreeRecord(t *testing.T, record string, valid bool, blob string) {
	t.Helper()
	entry, err := parseTreeEntry(record)
	if !valid {
		if err == nil || entry != (treeEntry{}) {
			t.Fatalf("malformed tree entry returned %+v, %v", entry, err)
		}
		return
	}
	fields := strings.SplitN(record, "\t", 2)
	if err != nil || entry.path != fields[1] || entry.mode != fields[0][:6] || entry.blob != blob {
		t.Fatalf("valid tree entry returned %+v, %v", entry, err)
	}
}

func TestSourceIdentityBoundaryContractGitCommandRefusal(t *testing.T) {
	for _, result := range []commandResult{{missing: true}, {exitCode: 1}} {
		result.output = []byte("fabricated command detail must not become identity")
		command := sourceIdentityBoundaryCommand(t, "selected-root", result)
		output, err := gitOutput(context.Background(), command, "selected-root", nil, "ls-tree", "-r", "-z", "HEAD")
		if output != nil || err == nil || err.Error() != "source git command failed" {
			t.Fatalf("failed Git command returned output %q, %v", output, err)
		}
	}
}

func sourceIdentityBoundaryCommand(t *testing.T, root string, result commandResult) commandFunc {
	t.Helper()
	calls := 0
	t.Cleanup(func() {
		if calls != 1 {
			t.Errorf("pure fake command called %d times, want 1", calls)
		}
	})
	return func(_ context.Context, binary string, args []string, directory string, environment []string) commandResult {
		calls++
		want := []string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "ls-tree", "-r", "-z", "HEAD"}
		if binary != "git" || !reflect.DeepEqual(args, want) || directory != root || len(environment) != 0 {
			t.Fatalf("unexpected fake command: %q %q in %q", binary, args, directory)
		}
		return result
	}
}

func TestSourceIdentityBoundaryContractTreeFileBudget(t *testing.T) {
	for _, count := range []int{0, 10_000, 10_001} {
		t.Run(fmt.Sprintf("files=%d", count), func(t *testing.T) {
			assertSourceIdentityBoundaryTreeFileBudget(t, count)
		})
	}
}

func assertSourceIdentityBoundaryTreeFileBudget(t *testing.T, count int) {
	t.Helper()
	var output strings.Builder
	for index := 0; index < count; index++ {
		_, _ = fmt.Fprintf(&output, "100644 blob %s\tfile-%05d.go\x00", strings.Repeat("a", 40), index)
	}
	command := sourceIdentityBoundaryCommand(t, "selected-root", commandResult{output: []byte(output.String())})
	entries, err := gitTreeEntries(context.Background(), command, "selected-root", nil)
	if count != 10_000 {
		if err == nil || entries != nil {
			t.Fatalf("invalid file count %d returned %d entries, %v", count, len(entries), err)
		}
		return
	}
	if err != nil || len(entries) != count || entries[0].path != "file-00000.go" || entries[count-1].path != "file-09999.go" {
		t.Fatalf("boundary tree returned %d entries, %v", len(entries), err)
	}
}

func TestSourceIdentityBoundaryContractWorktreeMetadata(t *testing.T) {
	cases := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "relative directory", value: "gitdir: metadata\n", valid: true},
		{name: "missing prefix", value: "metadata\n"},
		{name: "carriage return", value: "gitdir: metadata\r\n"},
		{name: "extra newline", value: "gitdir: metadata\n\n"},
		{name: "nul", value: "gitdir: metadata\x00\n"},
		{name: "missing directory", value: "gitdir: absent\n"},
		{name: "over budget", value: "gitdir: " + strings.Repeat("x", 4096)},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			root := privateAssuranceDirectory(t, "worktree-metadata")
			if err := os.Mkdir(filepath.Join(root, "metadata"), 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, ".git")
			if err := os.WriteFile(path, []byte(item.value), 0600); err != nil {
				t.Fatal(err)
			}
			if got := validGitWorktreeFile(root, path); got != item.valid {
				t.Fatalf("worktree metadata valid = %t, want %t", got, item.valid)
			}
		})
	}
}

func TestSourceIdentityBoundaryContractPhysicalRoot(t *testing.T) {
	root := privateAssuranceDirectory(t, "physical-root")
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("synthetic root fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{file, filepath.Join(root, "absent")} {
		command := func(context.Context, string, []string, string, []string) commandResult {
			t.Fatal("invalid physical root reached a command")
			return commandResult{}
		}
		identity, err := bindSource(context.Background(), command, path, "", nil)
		if err == nil || !reflect.DeepEqual(identity, SourceIdentity{}) {
			t.Fatalf("invalid physical root returned %+v, %v", identity, err)
		}
	}
	if !physicalSourceRoot(root) {
		t.Fatal("owned physical directory rejected")
	}
}

func TestSourceIdentityBoundaryContractReplacedDescriptor(t *testing.T) {
	root := privateAssuranceDirectory(t, "replaced-descriptor")
	name := "source.go"
	path := filepath.Join(root, name)
	data := []byte("package synthetic\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	expected, err := sourceIdentityBoundaryFileStat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the old inode allocated: this is a real pathname replacement, not
	// a fabricated FileInfo or an assumption about immediate inode reuse.
	if err = os.Rename(path, filepath.Join(root, "held-source")); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	actual, err := os.Lstat(path)
	if err != nil || os.SameFile(expected, actual) || expected.Size() != actual.Size() {
		t.Fatalf("replacement fixture did not preserve size with a distinct file: %v", err)
	}
	assertSourceIdentityBoundaryClosed(t, root, name, expected, errSourceChangedDuringRead)
	sum, err := fileSHA256(root, name, expected, int64(len(data)), nil)
	if !errors.Is(err, errSourceChangedDuringRead) || sum != ([sha256.Size]byte{}) {
		t.Fatalf("replaced file hash returned %x, %v", sum, err)
	}
}

func sourceIdentityBoundaryFileStat(path string) (os.FileInfo, error) {
	// Windows pathname Stat/Lstat may defer loading the file ID until SameFile.
	// Capture it from the original handle, then close before the rename so the
	// fixture neither follows the replacement name nor keeps a Windows file lock.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil {
		return nil, statErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return info, nil
}

func assertSourceIdentityBoundaryClosed(t *testing.T, root, name string, expected os.FileInfo, want error) {
	t.Helper()
	directory, file, err := openBoundSourceFile(root, name, expected)
	if directory != nil {
		_ = directory.Close()
	}
	if file != nil {
		_ = file.Close()
	}
	if directory != nil || file != nil || !errors.Is(err, want) {
		t.Fatalf("refused source returned directory=%v, file=%v, err=%v", directory, file, err)
	}
}

func TestSourceIdentityBoundaryContractMissingOpenAndPositiveControl(t *testing.T) {
	root := privateAssuranceDirectory(t, "bound-open")
	name := "source.go"
	data := []byte("package synthetic\n")
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	expected, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceIdentityBoundaryClosed(t, filepath.Join(root, "missing-root"), name, expected, os.ErrNotExist)
	assertSourceIdentityBoundaryClosed(t, root, "missing-file", expected, os.ErrNotExist)
	sum, err := fileSHA256(root, name, expected, int64(len(data)), nil)
	if err != nil || sum != sha256.Sum256(data) {
		t.Fatalf("unchanged bound source hash returned %x, %v", sum, err)
	}
}
