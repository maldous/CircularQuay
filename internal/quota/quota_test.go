package quota

import (
	"errors"
	"testing"
)

type fixtureProvider struct {
	status Status
	err    error
}

func (f fixtureProvider) Status(string) (Status, error) { return f.status, f.err }

func TestCheckRequiresEnforcedHardQuotaWithReserve(t *testing.T) {
	limits := Limits{MaximumBytes: 16, ReserveBytes: 4, MaximumInodes: 3, ReserveInodes: 2}
	cases := []struct {
		name   string
		status Status
		err    error
	}{
		{"accepted", Status{HardBytes: 20, UsedBytes: 16, HardInodes: 5, UsedInodes: 3, UserEnforced: true}, nil},
		{"accounting only", Status{HardBytes: 100, HardInodes: 10}, ErrUnenforced},
		{"byte quota too large", Status{HardBytes: 21, HardInodes: 5, UserEnforced: true}, ErrInsufficient},
		{"byte quota exhausted", Status{HardBytes: 20, UsedBytes: 17, HardInodes: 5, UsedInodes: 3, UserEnforced: true}, ErrInsufficient},
		{"inode quota exhausted", Status{HardBytes: 20, UsedBytes: 16, HardInodes: 5, UsedInodes: 4, UserEnforced: true}, ErrInsufficient},
		{"unlimited hard quota", Status{HardBytes: 0, HardInodes: 0, UserEnforced: true}, ErrInsufficient},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := Check(fixtureProvider{status: test.status}, "synthetic", limits)
			if !errors.Is(err, test.err) {
				t.Fatalf("Check() error = %v, want %v", err, test.err)
			}
		})
	}
}

func TestCheckPreservesProviderFailureAndOverflowFails(t *testing.T) {
	providerError := errors.New("synthetic")
	if err := Check(fixtureProvider{err: providerError}, "synthetic", Limits{}); !errors.Is(err, providerError) {
		t.Fatalf("provider error = %v", err)
	}
	status := Status{HardBytes: ^uint64(0), HardInodes: ^uint64(0), UserEnforced: true}
	limits := Limits{MaximumBytes: ^uint64(0), ReserveBytes: 1, MaximumInodes: 1}
	if err := Check(fixtureProvider{status: status}, "synthetic", limits); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("overflow accepted: %v", err)
	}
}
