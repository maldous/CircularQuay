// Package quota proves that a directory is protected by a kernel-enforced
// filesystem quota before an operation writes data supplied by an untrusted
// remote peer.
package quota

import (
	"errors"
	"math"
)

var (
	// ErrUnsupported means the platform cannot prove quota enforcement.
	ErrUnsupported = errors.New("kernel quota enforcement is not provable")
	// ErrUnenforced means accounting is available without a hard enforced cap.
	ErrUnenforced = errors.New("filesystem quota is not enforced")
	// ErrInsufficient means the configured hard quota cannot contain the work.
	ErrInsufficient = errors.New("filesystem quota is insufficient")
	// ErrPrivileged means the process could bypass a filesystem hard quota.
	ErrPrivileged = errors.New("process can bypass filesystem quota")
)

// Status is the current hard quota and usage for the process identity.
// Bytes are filesystem-accounted bytes, not application estimates.
type Status struct {
	HardBytes    uint64
	UsedBytes    uint64
	HardInodes   uint64
	UsedInodes   uint64
	UserEnforced bool
}

// Limits describes the maximum work a caller will allow plus its temporary
// reserve. Both resources need a hard kernel-enforced quota.
type Limits struct {
	MaximumBytes  uint64
	MaximumInodes uint64
	ReserveBytes  uint64
	ReserveInodes uint64
}

// Provider obtains an enforcement status for the filesystem containing path.
// It must not change quota configuration.
type Provider interface {
	Status(path string) (Status, error)
}

// Check verifies that the current identity has enough kernel-enforced quota
// for limits. It deliberately rejects accounting-only quota configurations.
func Check(provider Provider, path string, limits Limits) error {
	status, err := provider.Status(path)
	if err != nil {
		return err
	}
	if !status.UserEnforced {
		return ErrUnenforced
	}
	if insufficient(status.HardBytes, status.UsedBytes, limits.MaximumBytes, limits.ReserveBytes) {
		return ErrInsufficient
	}
	if insufficient(status.HardInodes, status.UsedInodes, limits.MaximumInodes, limits.ReserveInodes) {
		return ErrInsufficient
	}
	return nil
}

func insufficient(hard, used, maximum, reserve uint64) bool {
	if hard == 0 || used > hard {
		return true
	}
	ceiling, overflow := add(maximum, reserve)
	if overflow {
		return true
	}
	if hard > ceiling {
		return true
	}
	available := hard - used
	return available < reserve
}

func add(left, right uint64) (uint64, bool) {
	if math.MaxUint64-left < right {
		return 0, true
	}
	return left + right, false
}
