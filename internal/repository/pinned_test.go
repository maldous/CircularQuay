package repository

import (
	"context"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestFetchPinnedRejectsInvalidPinBeforeOpeningStore(t *testing.T) {
	t.Helper()
	_, err := FetchPinned(context.Background(), model.Repository{}, "not-a-full-commit")
	if !fault.Is(err, fault.Trust) || err.Error() != "trust_failure: repository_commit_pin" {
		t.Fatalf("invalid pin reached repository setup: %v", err)
	}
}

func TestPinnedFetchCandidateRejectsMismatchBeforeAcceptance(t *testing.T) {
	operations := fetchOperations{
		candidate: func(context.Context, *store) (string, error) {
			return "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", nil
		},
	}
	pinned := pinnedFetchOperations(operations, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	commit, err := pinned.candidate(context.Background(), &store{})
	if commit != "" || !fault.Is(err, fault.Trust) || err.Error() != "trust_failure: repository_commit_pin" {
		t.Fatalf("mismatched candidate reached acceptance: commit %q, error %v", commit, err)
	}
}

func TestPinnedFetchCandidateAllowsExactCommit(t *testing.T) {
	expected := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	operations := fetchOperations{
		candidate: func(context.Context, *store) (string, error) {
			return expected, nil
		},
	}
	commit, err := pinnedFetchOperations(operations, expected).candidate(context.Background(), &store{})
	if err != nil || commit != expected {
		t.Fatalf("exact candidate was not preserved: commit %q, error %v", commit, err)
	}
}
