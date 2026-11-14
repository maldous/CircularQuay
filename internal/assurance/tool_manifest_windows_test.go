//go:build windows

package assurance

import (
	"os"

	"github.com/maldous/CircularQuay/internal/platform"
	"golang.org/x/sys/windows"
)

// openFixtureRuleInputForReplacement uses FILE_SHARE_DELETE only in this test
// fixture. Production OpenProtected keeps its stricter handle-sharing policy;
// this handle exists solely to prove that the post-read name fence rejects a
// successfully replaced path on Windows.
func openFixtureRuleInputForReplacement(path string) (*os.File, error) {
	if err := platform.ValidateFileMetadata(path, true); err != nil {
		return nil, err
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return nil, os.ErrInvalid
	}
	return file, nil
}
