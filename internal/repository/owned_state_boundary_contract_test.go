package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

// These cases stop at pure parsing or owned-file refusals. No callback selects
// native quota, Git/OpenSSH, a network request, or a process-global fake runner.
func TestOwnedRepositoryBoundaryContractPublicEntriesRefuseOccupiedStore(t *testing.T) {
	for _, operation := range []string{"fetch", "pinned", "current"} {
		t.Run(operation, func(t *testing.T) {
			s := repositoryRefusalFixture(t)
			repo := s.repo
			repo.Directory = filepath.Join(filepath.Dir(s.repo.Directory), "occupied")
			data := []byte("SYNTHETIC_EXISTING_STORE_FILE")
			writeProtectedRepositoryFixture(t, repo.Directory, data)
			value, err := ownedRepositoryEntry(t, operation, repo)
			kind, label := fault.Trust, "repository_directory"
			if operation == "current" {
				kind, label = fault.Apply, "create private directory"
			}
			requireOwnedRepositoryError(t, err, kind, label)
			if value != "" {
				t.Fatal("occupied store produced a release")
			}
			assertRepositoryRefusalBytes(t, repo.Directory, data)
		})
	}
}

func ownedRepositoryEntry(t *testing.T, operation string, repo model.Repository) (string, error) {
	t.Helper()
	switch operation {
	case "fetch":
		return Fetch(context.Background(), repo)
	case "pinned":
		return FetchPinned(context.Background(), repo, strings.Repeat("a", 40))
	case "current":
		return CurrentContext(context.Background(), repo)
	default:
		t.Fatal("unknown synthetic repository entry")
		return "", errors.New("unsupported synthetic entry")
	}
}

func TestOwnedRepositoryBoundaryContractTreeCountIsExactAndPortable(t *testing.T) {
	for _, size := range []int{maximumFiles, maximumFiles + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			data := ownedRepositoryTree(size)
			before := bytes.Clone(data)
			tree, err := parseTree(data)
			assertOwnedRepositoryTree(t, size, tree, err)
			if !bytes.Equal(data, before) {
				t.Fatal("tree parsing mutated serialized input")
			}
		})
	}
	for _, data := range [][]byte{nil, {0}, {0, 0}} {
		tree, err := parseTree(data)
		requireOwnedRepositoryError(t, err, fault.Trust, "repository_empty_tree")
		if tree != nil {
			t.Fatal("empty tree returned partial entries")
		}
	}
}

func ownedRepositoryTree(size int) []byte {
	var data strings.Builder
	for index := 0; index < size; index++ {
		fmt.Fprintf(&data, "100644 blob %s\titem-%04d.json%c", strings.Repeat("a", 40), index, 0)
	}
	return []byte(data.String())
}

func assertOwnedRepositoryTree(t *testing.T, size int, tree map[string]string, err error) {
	t.Helper()
	if size > maximumFiles {
		requireOwnedRepositoryError(t, err, fault.Trust, "repository_tree_size")
		if tree != nil {
			t.Fatal("oversized tree returned a usable prefix")
		}
		return
	}
	if err != nil || len(tree) != size {
		t.Fatal("exact tree-count boundary lost entries")
	}
	for index := 0; index < size; index++ {
		if tree[fmt.Sprintf("item-%04d.json", index)] != strings.Repeat("a", 40) {
			t.Fatal("tree parsing changed a declared portable path or object")
		}
	}
}

func TestOwnedRepositoryBoundaryContractReleaseDestinationNeverReplacesFiles(t *testing.T) {
	for _, occupyReleases := range []bool{false, true} {
		t.Run(fmt.Sprint(occupyReleases), func(t *testing.T) {
			assertOwnedRepositoryDestination(t, occupyReleases)
		})
	}
}

func assertOwnedRepositoryDestination(t *testing.T, occupyReleases bool) {
	t.Helper()
	s := repositoryRefusalFixture(t)
	releases := filepath.Join(s.repo.Directory, "releases")
	path := releases
	operation, kind := "repository_releases_directory", fault.Trust
	if !occupyReleases {
		if err := platform.EnsurePrivateDirectory(releases); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(releases, strings.Repeat("a", 40))
		operation, kind = "create private directory", fault.Apply
	}
	data := []byte("SYNTHETIC_EXISTING_DESTINATION")
	writeProtectedRepositoryFixture(t, path, data)
	parent, destination, accepted, err := s.releaseDestination(context.Background(), strings.Repeat("a", 40), map[string]string{"data.json": strings.Repeat("b", 40)})
	requireOwnedRepositoryError(t, err, kind, operation)
	if parent != "" || destination != "" || accepted {
		t.Fatal("occupied path returned a partial accepted destination")
	}
	assertRepositoryRefusalBytes(t, path, data)
}

func TestOwnedRepositoryBoundaryContractMaterializedWritesPreserveOccupiedEntries(t *testing.T) {
	for _, parentFile := range []bool{false, true} {
		t.Run(fmt.Sprint(parentFile), func(t *testing.T) {
			assertOwnedRepositoryMaterialization(t, parentFile)
		})
	}
}

func assertOwnedRepositoryMaterialization(t *testing.T, parentFile bool) {
	t.Helper()
	staging := privateRepositoryDirectory(t, "owned-staging")
	occupied := filepath.Join(staging, "occupied")
	path, label := "occupied", "repository_release_write"
	if parentFile {
		writeProtectedRepositoryFixture(t, occupied, []byte("SYNTHETIC_EXISTING_PARENT"))
		path, label = "occupied/child.json", "repository_release_parent"
	} else {
		if err := platform.EnsurePrivateDirectory(occupied); err != nil {
			t.Fatal(err)
		}
		writeProtectedRepositoryFixture(t, filepath.Join(occupied, "keep"), []byte("SYNTHETIC_EXISTING_DIRECTORY"))
	}
	err := writeMaterializedFile(staging, path, []byte("SYNTHETIC_NEW_BLOB"))
	requireOwnedRepositoryError(t, err, fault.Trust, label)
	assertOwnedRepositoryOccupied(t, occupied, parentFile)
}

func assertOwnedRepositoryOccupied(t *testing.T, path string, parentFile bool) {
	t.Helper()
	if parentFile {
		assertRepositoryRefusalBytes(t, path, []byte("SYNTHETIC_EXISTING_PARENT"))
		return
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		t.Fatal("materialization replaced an owned directory")
	}
	assertRepositoryRefusalBytes(t, filepath.Join(path, "keep"), []byte("SYNTHETIC_EXISTING_DIRECTORY"))
}

func TestOwnedRepositoryBoundaryContractFetchTokenRefusalsReturnNoEnvironment(t *testing.T) {
	for _, scenario := range []struct {
		name, operation string
		data            []byte
		kind            fault.Kind
	}{
		{"missing", "read protected file", nil, fault.Missing},
		{"oversized", "repository_file_size", bytes.Repeat([]byte{'s'}, (16<<10)+1), fault.Trust},
		{"multiline", "repository_token", []byte("SYNTHETIC_FIRST\nSYNTHETIC_SECOND"), fault.Trust},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s := repositoryRefusalFixture(t)
			s.repo.TokenFile = filepath.Join(filepath.Dir(s.repo.Directory), "token")
			if scenario.data != nil {
				writeProtectedRepositoryFixture(t, s.repo.TokenFile, scenario.data)
			}
			environment, err := s.environment(true, false)
			requireOwnedRepositoryError(t, err, scenario.kind, scenario.operation)
			if environment != nil {
				t.Fatal("refused token returned a partial child environment")
			}
			assertOwnedRepositoryTokenInput(t, s.repo.TokenFile, scenario.data)
		})
	}
}

func assertOwnedRepositoryTokenInput(t *testing.T, path string, data []byte) {
	t.Helper()
	if data == nil {
		assertRepositoryRefusalAbsent(t, path)
		return
	}
	assertRepositoryRefusalBytes(t, path, data)
}

func TestOwnedRepositoryBoundaryContractInitializationLossStopsBeforeGit(t *testing.T) {
	s := repositoryRefusalFixture(t)
	if err := os.Remove(s.repo.Directory); err != nil {
		t.Fatal("could not remove the empty owned store fixture")
	}
	err := s.initialize(context.Background())
	requireOwnedRepositoryError(t, err, fault.Trust, "repository_git_template")
	assertRepositoryRefusalAbsent(t, s.repo.Directory)
}

func TestOwnedRepositoryBoundaryContractGitConfigCannotReplaceOccupiedFile(t *testing.T) {
	s := repositoryRefusalFixture(t)
	data := []byte("SYNTHETIC_NOT_A_GIT_DIRECTORY")
	writeProtectedRepositoryFixture(t, s.git, data)
	err := s.checkGitConfig()
	requireOwnedRepositoryError(t, err, fault.Apply, "create private directory")
	assertRepositoryRefusalBytes(t, s.git, data)
}

func TestOwnedRepositoryBoundaryContractPinnedCandidateErrorIsNotAcceptance(t *testing.T) {
	var events []string
	s := repositoryRefusalFixture(t)
	operations := repositoryRefusalOperations(t, s.repo.Directory, &events)
	expected := failure(fault.Unavailable, "fetch")
	operations.candidate = func(context.Context, *store) (string, error) { return "", expected }
	value, err := pinnedFetchOperations(operations, strings.Repeat("a", 40)).candidate(context.Background(), s)
	if value != "" || err != expected || len(events) != 0 {
		t.Fatal("pinned candidate error was discarded or reached another stage")
	}
	assertRepositoryRefusalAbsent(t, filepath.Join(s.repo.Directory, "accepted.json"))
}

func TestOwnedRepositoryBoundaryContractRealEntriesDetectReplacedOrRemovedPaths(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprint(directory), func(t *testing.T) {
			assertOwnedRepositoryChangedEntry(t, directory)
		})
	}
}

func assertOwnedRepositoryChangedEntry(t *testing.T, directory bool) {
	t.Helper()
	root := privateRepositoryDirectory(t, "owned-entry")
	path := filepath.Join(root, "entry")
	if directory {
		if err := platform.EnsurePrivateDirectory(path); err != nil {
			t.Fatal(err)
		}
	} else {
		writeProtectedRepositoryFixture(t, path, []byte("SYNTHETIC_ENTRY"))
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatal("could not capture exactly one real owned entry")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal("could not remove the owned entry after its metadata capture")
	}
	kind, label := fault.Missing, "read protected file"
	if directory {
		writeProtectedRepositoryFixture(t, path, []byte("SYNTHETIC_DIRECTORY_REPLACEMENT"))
		kind, label = fault.Trust, "repository_release_directory"
	}
	verifier := releaseVerifier{store: repositoryRefusalFixture(t), ctx: context.Background(), directory: root, tree: map[string]string{"entry": strings.Repeat("a", 40)}}
	err = verifier.visit(path, entries[0], nil)
	requireOwnedRepositoryError(t, err, kind, label)
	if verifier.count != 0 || verifier.total != 0 {
		t.Fatal("refused entry was counted or hashed as an accepted blob")
	}
}

func requireOwnedRepositoryError(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != kind || safe.Operation != operation || errors.Unwrap(err) != nil {
		t.Fatal("owned-path refusal lost its exact safe category or operation")
	}
	if strings.Contains(err.Error(), "SYNTHETIC_PRIVATE") {
		t.Fatal("owned-path refusal exposed private diagnostic text")
	}
}
