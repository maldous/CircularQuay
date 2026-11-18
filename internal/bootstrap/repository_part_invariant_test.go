package bootstrap

import (
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestRepositoryPartInvariantEndpointsExcludeDotComponents(t *testing.T) {
	for _, component := range []string{".", ".."} {
		if validRepositoryPart(component) {
			t.Fatal("dot component passed the primary repository-part predicate")
		}
		for _, parts := range [][]string{{component, "repo", "tree", testRef}, {"owner", component, "tree", testRef}} {
			owner, repository, err := repositoryParts(parts)
			if owner != "" || repository != "" || !fault.Is(err, fault.Invalid) {
				t.Fatal("dot component escaped the primary repository admission")
			}
		}
	}
	owner, repository, err := repositoryParts([]string{"owner-1", "repo.name", "tree", testRef})
	if err != nil || owner != "owner-1" || repository != "repo.name" {
		t.Fatal("ordinary repository name changed during invariant simplification")
	}
}
