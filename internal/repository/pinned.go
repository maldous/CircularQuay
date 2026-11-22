package repository

import (
	"context"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// FetchPinned fetches exactly expectedCommit from repo before accepting a release.
//
// The pin is validated before opening local state or initiating transport. The
// candidate is compared after native fetch resolution and before the existing
// acceptance path verifies signatures, ancestry, object graph, materialization,
// or writes accepted state. This prevents a movable ref from being accepted as
// a different signed commit while retaining every native fetch protection.
func FetchPinned(ctx context.Context, repo model.Repository, expectedCommit string) (string, error) {
	if !validCommit(expectedCommit) {
		return "", failure(fault.Trust, "commit_pin")
	}
	s, err := openStore(repo, true)
	if err != nil {
		return "", err
	}
	return s.fetch(ctx, pinnedFetchOperations(nativeFetchOperations(), expectedCommit))
}

func pinnedFetchOperations(operations fetchOperations, expectedCommit string) fetchOperations {
	candidate := operations.candidate
	operations.candidate = func(ctx context.Context, s *store) (string, error) {
		commit, err := candidate(ctx, s)
		if err != nil {
			return "", err
		}
		if commit != expectedCommit {
			return "", failure(fault.Trust, "commit_pin")
		}
		return commit, nil
	}
	return operations
}
