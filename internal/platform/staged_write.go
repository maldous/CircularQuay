package platform

import (
	"os"

	"github.com/maldous/CircularQuay/internal/fault"
)

// stagedWriteHandle is a per-call lifecycle seam. Production supplies only the
// opened *os.File; it is not configurable by policy or a package-global hook.
type stagedWriteHandle interface {
	Chmod(os.FileMode) error
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

func writeAndSyncHandle(f stagedWriteHandle, data []byte) error {
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fault.New(fault.Apply, "protect staged file")
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fault.New(fault.Apply, "write staged file")
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fault.New(fault.Apply, "sync staged file")
	}
	if err := f.Close(); err != nil {
		return fault.New(fault.Apply, "close staged file")
	}
	return nil
}
