package assurance

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeldFileLifecycleContractBoundSourceReadFailureReturnsWholeZero(t *testing.T) {
	root := privateAssuranceDirectory(t, "source")
	data := []byte("package fixture\n")
	name := sourceInputOwnedFile(t, root, "source.go", data)
	before, err := sourceIdentityBoundaryFileStat(name)
	if err != nil {
		t.Fatal(err)
	}
	var held *os.File
	digest := func(file *os.File, size, maximum int64) ([sha256.Size]byte, error) {
		held = file
		if size != int64(len(data)) || maximum != int64(len(data)) {
			t.Fatal("source reader lost its descriptor size bound")
		}
		return [sha256.Size]byte{}, errors.New("synthetic read failure")
	}
	sum, err := fileSHA256WithDigest(root, "source.go", before, int64(len(data)), nil, digest)
	if !errors.Is(err, errSourceChangedDuringRead) || sum != ([sha256.Size]byte{}) || held == nil {
		t.Fatal("source read failure produced partial identity")
	}
	if _, err := held.Stat(); err == nil {
		t.Fatal("source read failure retained its open descriptor")
	}
	toolExecutionBoundaryBytes(t, name, data)
}

func TestHeldFileLifecycleContractToolHashFailuresCloseAndPreserveInput(t *testing.T) {
	for _, drift := range []bool{false, true} {
		t.Run(map[bool]string{false: "read-error", true: "size-drift"}[drift], func(t *testing.T) { heldFileToolHash(t, drift) })
	}
}

func heldFileToolHash(t *testing.T, drift bool) {
	t.Helper()
	root := privateAssuranceDirectory(t, "tools")
	name := writeTool(t, filepath.Join(root, canonicalToolName("git")), "owned inert tool")
	var held *os.File
	digest := func(file *os.File) (string, int64, error) {
		held = file
		if !drift {
			return "", 0, errors.New("synthetic digest failure")
		}
		result, size, err := boundedToolHash(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("changed owned inert tool with different length"), 0700); err != nil {
			t.Fatal(err)
		}
		return result, size, nil
	}
	if sum, err := executableSHA256WithHash(name, digest); err == nil || sum != "" || held == nil {
		t.Fatal("failed or changed tool hash produced usable identity")
	}
	if _, err := held.Stat(); err == nil {
		t.Fatal("tool hash refusal leaked its descriptor")
	}
}

func TestHeldFileLifecycleContractFailedHelperCopyRemovesIncompleteTarget(t *testing.T) {
	root := privateAssuranceDirectory(t, "helper-copy")
	source := writeTool(t, filepath.Join(root, canonicalToolName("git")), "owned inert source")
	target := filepath.Join(root, "copy")
	copyBytes := func(destination, input *os.File) (int64, error) {
		if _, err := destination.Write([]byte("partial")); err != nil {
			t.Fatal(err)
		}
		if err := destination.Close(); err != nil {
			t.Fatal(err)
		}
		return 0, errors.New("synthetic copy failure")
	}
	if err := writeHelperWithCopy(source, target, copyBytes); !errors.Is(err, os.ErrInvalid) {
		t.Fatal("failed helper copy changed its refusal")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal("failed helper copy retained an incomplete target")
	}
	toolExecutionBoundaryBytes(t, source, []byte("owned inert source"))
}

func TestHeldFileLifecycleContractWrongDigestRemovesCopyBeforeEnable(t *testing.T) {
	root := privateAssuranceDirectory(t, "helper-copy")
	source := writeTool(t, filepath.Join(root, canonicalToolName("git")), "owned inert source")
	target := filepath.Join(root, "copy")
	err := copyHelperBinary(helperBinary{id: "git", path: source, sha256: strings.Repeat("0", 64)}, target)
	if !errors.Is(err, os.ErrInvalid) {
		t.Fatal("wrong-digest helper copy was enabled")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal("wrong-digest helper copy retained a target")
	}
}

func TestHeldFileLifecycleContractOwnedRuleSizeAndAggregateBounds(t *testing.T) {
	root := privateAssuranceDirectory(t, "rules")
	oversized := heldFileSparseRule(t, root, "oversized", maxToolInputBytes+1)
	if sum, err := boundedInputDigest(oversized, oversized.Name()); err == nil || sum != "" {
		t.Fatal("oversized held rule produced a digest")
	}
	if err := oversized.Close(); err != nil {
		t.Fatal(err)
	}
	paths := heldFileAggregateRules(t, root)
	before := append([]string(nil), paths...)
	if sum, err := digestInputFiles(root, paths); err == nil || sum != "" {
		t.Fatal("snapshot above the aggregate byte limit produced a digest")
	}
	if !bytes.Equal([]byte(strings.Join(paths, "\x00")), []byte(strings.Join(before, "\x00"))) {
		t.Fatal("already sorted aggregate input order changed")
	}
}

func heldFileSparseRule(t *testing.T, root, name string, size int64) *os.File {
	t.Helper()
	path := sourceInputOwnedFile(t, root, name, nil)
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	return file
}

func heldFileAggregateRules(t *testing.T, root string) []string {
	t.Helper()
	paths := make([]string, 0, maxToolInputTotal/maxToolInputBytes+1)
	for index := 0; index <= maxToolInputTotal/maxToolInputBytes; index++ {
		file := heldFileSparseRule(t, root, string(rune('a'+index))+".rules", maxToolInputBytes)
		paths = append(paths, file.Name())
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return paths
}

func TestHeldFileLifecycleContractToolHashByteLimit(t *testing.T) {
	root := privateAssuranceDirectory(t, "tool-hash")
	file := heldFileSparseRule(t, root, "inert-oversized", maxToolBinaryBytes+1)
	if sum, count, err := boundedToolHash(file); err == nil || sum != "" || count != 0 {
		t.Fatal("oversized inert tool produced a bounded executable hash")
	}
}
