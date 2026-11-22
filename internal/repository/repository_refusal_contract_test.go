package repository

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// These contracts use private fabricated files and injected early failures.
// They never select native fetch operations or reach Git/SSH hardening.
func TestRepositoryRefusalContractIdentityFailurePrecedesLockAndQuota(t *testing.T) {
	s := repositoryRefusalFixture(t)
	var events []string
	operations := repositoryRefusalOperations(t, s.repo.Directory, &events)
	operations.validateIdentity = func() error {
		events = append(events, "identity")
		return errors.New("SYNTHETIC_PRIVATE_IDENTITY_FAILURE")
	}
	result, err := s.fetch(context.Background(), operations)
	assertRepositoryRefusalFault(t, err, fault.Unsupported, "disk_quota_identity")
	if result != "" || !reflect.DeepEqual(events, []string{"identity"}) {
		t.Fatal("identity refusal produced a release or reached later admission")
	}
	assertRepositoryRefusalAbsent(t, filepath.Join(s.repo.Directory, "fetch.lock"))
	assertRepositoryRefusalAbsent(t, filepath.Join(s.repo.Directory, ".fetch-tmp"))
}

func TestRepositoryRefusalContractOccupiedLockPreservesExistingOwner(t *testing.T) {
	s := repositoryRefusalFixture(t)
	lock := filepath.Join(s.repo.Directory, "fetch.lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		t.Fatal("could not create the synthetic occupied lock")
	}
	sentinel := filepath.Join(lock, "keep")
	writeProtectedRepositoryFixture(t, sentinel, []byte("synthetic existing owner"))
	var events []string
	result, err := s.fetch(context.Background(), repositoryRefusalOperations(t, s.repo.Directory, &events))
	assertRepositoryRefusalFault(t, err, fault.Unavailable, "locked")
	if result != "" || !reflect.DeepEqual(events, []string{"identity"}) {
		t.Fatal("occupied lock admitted later work or produced a release")
	}
	assertRepositoryRefusalBytes(t, sentinel, []byte("synthetic existing owner"))
	assertRepositoryRefusalAbsent(t, filepath.Join(s.repo.Directory, ".fetch-tmp"))
}

func TestRepositoryRefusalContractPreparationFailureReleasesOnlyItsLock(t *testing.T) {
	s := repositoryRefusalFixture(t)
	s.temporary = filepath.Join(s.repo.Directory, "prior-temporary")
	before := s.temporary
	var events []string
	operations := repositoryRefusalOperations(t, s.repo.Directory, &events)
	expected := failure(fault.Trust, "temporary_directory")
	operations.prepareTemporary = func(received *store) (func(), error) {
		if received != s {
			t.Fatal("preparation received a different repository store")
		}
		events = append(events, "prepare")
		return nil, expected
	}
	result, err := s.fetch(context.Background(), operations)
	if err != expected || result != "" || s.temporary != before {
		t.Fatal("failed preparation changed its error, output or prior temporary state")
	}
	if !reflect.DeepEqual(events, []string{"identity", "quota", "prepare"}) {
		t.Fatal("preparation refusal changed admission order")
	}
	assertRepositoryRefusalAbsent(t, filepath.Join(s.repo.Directory, "fetch.lock"))
	assertRepositoryRefusalAbsent(t, filepath.Join(s.repo.Directory, ".fetch-tmp"))
}

func TestRepositoryRefusalContractQuotaErrorMappingIsSanitized(t *testing.T) {
	if err := mapQuotaError(nil); err != nil {
		t.Fatal("absence of a quota error became an unsupported failure")
	}
	err := mapQuotaError(errors.New("SYNTHETIC_PRIVATE_QUOTA_DIAGNOSTIC"))
	assertRepositoryRefusalFault(t, err, fault.Unsupported, "disk_quota")
}

func TestRepositoryRefusalContractTemporaryCollisionPreservesFileAndState(t *testing.T) {
	s := repositoryRefusalFixture(t)
	s.temporary = filepath.Join(s.repo.Directory, "prior-temporary")
	before := s.temporary
	path := filepath.Join(s.repo.Directory, ".fetch-tmp")
	data := []byte("synthetic file, not a temporary directory")
	writeProtectedRepositoryFixture(t, path, data)
	cleanup, err := prepareFetchTemporary(s)
	assertRepositoryRefusalFault(t, err, fault.Trust, "temporary_directory")
	if cleanup != nil || s.temporary != before {
		t.Fatal("failed temporary preparation installed cleanup or changed prior state")
	}
	assertRepositoryRefusalBytes(t, path, data)
}

func TestRepositoryRefusalContractDeclarationDeniesBeforeDirectoryCreation(t *testing.T) {
	s := repositoryRefusalFixture(t)
	repo := s.repo
	repo.URL = ""
	repo.Directory = filepath.Join(s.repo.Directory, "must-not-create")
	opened, err := openStore(repo, true)
	assertRepositoryRefusalFault(t, err, fault.Invalid, "declaration")
	if opened != nil {
		t.Fatal("invalid declaration returned a partially usable store")
	}
	assertRepositoryRefusalAbsent(t, repo.Directory)
}

func TestRepositoryRefusalContractStoreDirectoryCannotReplaceExistingFile(t *testing.T) {
	s := repositoryRefusalFixture(t)
	repo := s.repo
	repo.Directory = filepath.Join(s.repo.Directory, "not-a-directory")
	data := []byte("synthetic existing file")
	writeProtectedRepositoryFixture(t, repo.Directory, data)
	opened, err := openStore(repo, true)
	assertRepositoryRefusalFault(t, err, fault.Trust, "directory")
	if opened != nil {
		t.Fatal("invalid directory returned a partially usable store")
	}
	assertRepositoryRefusalBytes(t, repo.Directory, data)
}

func TestRepositoryRefusalContractStoreRequiresBoundedNonblankSigners(t *testing.T) {
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"missing", nil},
		{"blank", []byte(" \t\r\n")},
		{"oversized", bytes.Repeat([]byte("x"), (64<<10)+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := repositoryRefusalFixture(t)
			repo := s.repo
			repo.AllowedSignersFile = filepath.Join(filepath.Dir(repo.Directory), "rejected-signers")
			if test.data != nil {
				writeProtectedRepositoryFixture(t, repo.AllowedSignersFile, test.data)
			}
			opened, err := openStore(repo, false)
			assertRepositoryRefusalFault(t, err, fault.Trust, "signers")
			if opened != nil {
				t.Fatal("invalid signer input returned a partially usable store")
			}
		})
	}
}

func TestRepositoryRefusalContractAcceptedStateRequiresOneCanonicalBoundedObject(t *testing.T) {
	for _, test := range []struct {
		name      string
		transform func(string) string
		operation string
	}{
		{"noncanonical field", func(valid string) string { return strings.Replace(valid, `"url"`, `"URL"`, 1) }, "accepted_state"},
		{"missing fields", func(string) string { return `{"url":"https://example.invalid/config.git"}` }, "accepted_state"},
		{"unknown field", func(valid string) string { return strings.TrimSuffix(valid, "}") + `,"unexpected":true}` }, "accepted_state"},
		{"trailing document", func(valid string) string { return valid + `{}` }, "accepted_state"},
		{"null object", func(string) string { return `null` }, "accepted_state"},
		{"oversized object", func(string) string { return strings.Repeat(" ", 4097) }, "file_size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := repositoryRefusalFixture(t)
			valid := string(acceptedStateJSON(s.repo.URL, s.repo.Ref, strings.Repeat("a", 40)))
			data := []byte(test.transform(valid))
			path := filepath.Join(s.repo.Directory, "accepted.json")
			writeProtectedRepositoryFixture(t, path, data)
			state, err := s.readAccepted()
			assertRepositoryRefusalFault(t, err, fault.Trust, test.operation)
			if state != (accepted{}) {
				t.Fatal("rejected accepted-state document returned partial identity")
			}
			assertRepositoryRefusalBytes(t, path, data)
		})
	}
}

func repositoryRefusalFixture(t *testing.T) *store {
	t.Helper()
	root := privateRepositoryDirectory(t, "repository-refusal")
	directory := filepath.Join(root, "cache")
	signers := filepath.Join(root, "synthetic-signers")
	writeProtectedRepositoryFixture(t, signers, []byte("synthetic nonempty signer fixture, not a key\n"))
	repo := model.Repository{
		URL:                "https://example.invalid/config.git",
		Directory:          directory,
		Ref:                "refs/heads/data",
		AllowedSignersFile: signers,
		InitialCommit:      strings.Repeat("a", 40),
	}
	s, err := openStore(repo, true)
	if err != nil {
		t.Fatal("could not open the bounded synthetic repository store")
	}
	return s
}

func repositoryRefusalOperations(t *testing.T, directory string, events *[]string) fetchOperations {
	t.Helper()
	return fetchOperations{
		validateIdentity: func() error {
			*events = append(*events, "identity")
			return nil
		},
		assertQuota: func(received string) error {
			if received != directory {
				t.Fatal("quota callback received a different repository directory")
			}
			*events = append(*events, "quota")
			return nil
		},
		prepareTemporary: func(*store) (func(), error) {
			t.Fatal("unexpected preparation must stop before Git hardening")
			return nil, failure(fault.Trust, "temporary_directory")
		},
		candidate: func(context.Context, *store) (string, error) {
			t.Fatal("early refusal reached an unselected candidate stage")
			return "", failure(fault.Trust, "candidate")
		},
	}
}

func assertRepositoryRefusalFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) {
		t.Fatal("repository refusal did not return a typed safe failure")
	}
	want := "repository_" + operation
	if safe.Kind != kind || safe.Operation != want || err.Error() != string(kind)+": "+want || errors.Unwrap(err) != nil {
		t.Fatal("repository refusal changed category/operation or retained upstream diagnostics")
	}
}

func assertRepositoryRefusalAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("early refusal left or created an unselected path")
	}
}

func assertRepositoryRefusalBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal("refusal removed or modified an existing synthetic file")
	}
}
