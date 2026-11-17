package platform

import (
	"io"
	"os"
)

// Operations are private, per-call dependencies. Public entry points always
// supply the concrete production defaults; policy cannot replace them.
type platformBoundaryOperations struct {
	open    func(string, int, os.FileMode) (*os.File, error)
	protect func(string) error
	receipt func(string) (*os.File, error)
	lock    func(string) (*os.File, bool, error)
	write   func(*os.File, []byte) (int, error)
	sync    func(*os.File) error
}

func defaultPlatformBoundaryOperations() platformBoundaryOperations {
	return platformBoundaryOperations{openInParent, protectFile, openReceiptLog, openOrCreateLock, (*os.File).Write, (*os.File).Sync}
}

type executableBoundaryOperations struct {
	open    func(string) (*os.File, string, os.FileInfo, error)
	digest func(io.Reader, int64) ([]byte, bool, error)
}

func defaultExecutableBoundaryOperations() executableBoundaryOperations {
	return executableBoundaryOperations{openVerifiedExecutable, boundedExecutableDigest}
}

type creationBoundaryOperations struct {
	open    func(*os.Root, string, int, os.FileMode) (*os.File, error)
	populate func(*os.File, string, []byte) error
	sync    func(string) error
}

func defaultCreationBoundaryOperations() creationBoundaryOperations {
	return creationBoundaryOperations{(*os.Root).OpenFile, populateCreatedFile, syncDirectory}
}
