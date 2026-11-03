//go:build windows

package platform

import (
	"strings"
	"unsafe"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"golang.org/x/sys/windows"
)

func isServiceSID(sid string) bool { return strings.HasPrefix(sid, "S-1-5-80-") }

func applyDPAPI(target model.Target, data []byte) error {
	if len(data) == 0 {
		return fault.New(fault.Invalid, "empty DPAPI projection")
	}
	var sid string
	if target.Resource != "" {
		parsed, err := windows.StringToSid(target.Resource)
		if err != nil || !isServiceSID(parsed.String()) {
			return fault.New(fault.Invalid, "explicit service SID required")
		}
		sid = parsed.String()
	}
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var output windows.DataBlob
	if windows.CryptProtectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_LOCAL_MACHINE|windows.CRYPTPROTECT_UI_FORBIDDEN, &output) != nil {
		return fault.New(fault.Apply, "protect DPAPI projection")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	protected := unsafe.Slice(output.Data, int(output.Size))
	if err := WriteProtected(target.Path, protected); err != nil {
		return err
	}
	if err := grantFile(target.Path, sid); err != nil {
		return fault.New(fault.Apply, "grant projection service SID")
	}
	return nil
}

// ReadDPAPI decrypts only an ACL-protected local projection. It does not export
// the machine's DPAPI master key or imply portability to another machine.
func ReadDPAPI(path string) ([]byte, error) {
	data, err := ReadProtected(path, true)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fault.New(fault.Invalid, "empty DPAPI blob")
	}
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var output windows.DataBlob
	if windows.CryptUnprotectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output) != nil {
		return nil, fault.New(fault.Denied, "read DPAPI projection")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return append([]byte(nil), unsafe.Slice(output.Data, int(output.Size))...), nil
}
