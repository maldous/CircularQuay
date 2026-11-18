package platform

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestCreateProtectedExclusivePreservesExistingBytes(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "initial.json")
	expected := []byte(`{"value":"synthetic"}`)
	if err := CreateProtectedExclusive(path, expected); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFileMetadata(path, true); err != nil {
		t.Fatal("created file did not receive native secret protection")
	}
	if err := CreateProtectedExclusive(path, []byte("replacement")); err == nil {
		t.Fatal("exclusive creation replaced an existing file")
	}
	actual, err := ReadProtected(path, true)
	if err != nil || !bytes.Equal(actual, expected) {
		t.Fatal("failed creation changed the original bytes")
	}
}

func TestCreateProtectedExclusiveRejectsInvalidDestinations(t *testing.T) {
	root := secureTempDir(t)
	for _, path := range []string{"relative.json", root, filepath.Join(root, "missing", "value.json")} {
		if err := CreateProtectedExclusive(path, []byte("synthetic")); err == nil {
			t.Fatalf("invalid destination accepted: %s", path)
		}
	}
	oversized := filepath.Join(root, "oversized")
	if err := CreateProtectedExclusive(oversized, make([]byte, MaxDocumentSize+1)); !fault.Is(err, fault.Invalid) {
		t.Fatal("oversized document accepted")
	}
	if _, err := os.Lstat(oversized); !os.IsNotExist(err) {
		t.Fatal("invalid input created a destination")
	}
}

func TestConcurrentExclusiveCreationHasOneCompleteWinner(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "concurrent")
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, value := range []string{"synthetic-first", "synthetic-second"} {
		go func() {
			<-start
			results <- CreateProtectedExclusive(path, []byte(value))
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		if <-results == nil {
			successes++
		}
	}
	actual, err := ReadProtected(path, true)
	if successes != 1 || err != nil || (string(actual) != "synthetic-first" && string(actual) != "synthetic-second") {
		t.Fatal("exclusive creators did not retain exactly one complete protected result")
	}
}

func TestFailedCreationCleanupPreservesAReplacement(t *testing.T) {
	directory := secureTempDir(t)
	path := filepath.Join(directory, "created")
	if err := CreateProtectedExclusive(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	created := createdFileIdentity(t, path)
	if err := os.Rename(path, filepath.Join(directory, "moved")); err != nil {
		t.Fatal(err)
	}
	if err := CreateProtectedExclusive(path, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	removeCreatedFile(root, "created", created)
	actual, err := ReadProtected(path, true)
	if err != nil {
		t.Fatalf("replacement cannot be read after cleanup: %v", err)
	}
	if string(actual) != "replacement" {
		t.Fatal("failed creator removed another writer's replacement")
	}
	removeCreatedFile(root, "moved", created)
	if _, err := os.Lstat(filepath.Join(directory, "moved")); !os.IsNotExist(err) {
		t.Fatal("failed creator did not remove its own file")
	}
}

func createdFileIdentity(t *testing.T, path string) os.FileInfo {
	t.Helper()
	file, err := OpenProtected(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	// Match production's descriptor-based identity capture. On Windows,
	// os.Stat may defer its file-ID lookup until SameFile, after the name has
	// been replaced. File.Stat binds the identity while this handle is open.
	created, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	return created
}
