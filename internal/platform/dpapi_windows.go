//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"golang.org/x/sys/windows"
)

func isServiceSID(sid string) bool { return strings.HasPrefix(sid, "S-1-5-80-") }

func applyDPAPI(target model.Target, data []byte) error {
	return applyDPAPIWithGrant(target, data, grantFile)
}

// applyDPAPIWithGrant prepares the final service DACL on an owner-only staged
// blob. The destination is replaced only after that ACL preparation succeeds,
// so a failed grant cannot withdraw service access from the last projection.
func applyDPAPIWithGrant(target model.Target, data []byte, grant func(string, string) error) error {
	sid, err := dpapiServiceSID(target.Resource)
	if err != nil {
		return err
	}
	protected, err := protectDPAPI(data)
	if err != nil {
		return err
	}
	return publishDPAPIProjection(target.Path, protected, sid, grant)
}

func dpapiServiceSID(resource string) (string, error) {
	if resource == "" {
		return "", nil
	}
	parsed, err := windows.StringToSid(resource)
	if err != nil || !isServiceSID(parsed.String()) {
		return "", fault.New(fault.Invalid, "explicit service SID required")
	}
	return parsed.String(), nil
}

func protectDPAPI(data []byte) ([]byte, error) {
	return protectDPAPIWith(data, func(input, output *windows.DataBlob) error {
		return windows.CryptProtectData(input, nil, nil, 0, nil, windows.CRYPTPROTECT_LOCAL_MACHINE|windows.CRYPTPROTECT_UI_FORBIDDEN, output)
	})
}

func protectDPAPIWith(data []byte, protect func(*windows.DataBlob, *windows.DataBlob) error) ([]byte, error) {
	input, err := boundedDPAPIInput(data, "DPAPI projection")
	if err != nil {
		return nil, err
	}
	var output windows.DataBlob
	if protect(&input, &output) != nil {
		return nil, fault.New(fault.Apply, "protect DPAPI projection")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return boundedDPAPIOutput(output, fault.Apply, "protected DPAPI projection")
}

func publishDPAPIProjection(path string, protected []byte, sid string, grant func(string, string) error) error {
	if !filepath.IsAbs(path) || len(protected) > MaxDocumentSize {
		return fault.New(fault.Invalid, "protected destination")
	}
	dir := filepath.Dir(path)
	if err := protectedDestination(dir, path); err != nil {
		return err
	}
	name, err := stageProtectedFile(dir, protected)
	if err != nil {
		return err
	}
	defer os.Remove(name)
	if err := grant(name, sid); err != nil {
		return fault.New(fault.Apply, "grant projection service SID")
	}
	return activateStagedFile(name, path, dir)
}

// ReadDPAPI decrypts only an ACL-protected local projection. It does not export
// the machine's DPAPI master key or imply portability to another machine.
func ReadDPAPI(path string) ([]byte, error) {
	data, err := ReadProtected(path, true)
	if err != nil {
		return nil, err
	}
	input, err := boundedDPAPIInput(data, "DPAPI blob")
	if err != nil {
		return nil, err
	}
	var output windows.DataBlob
	if windows.CryptUnprotectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output) != nil {
		return nil, fault.New(fault.Denied, "read DPAPI projection")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return boundedDPAPIOutput(output, fault.Denied, "decrypted DPAPI projection")
}

func boundedDPAPIInput(data []byte, name string) (windows.DataBlob, error) {
	size := uint64(len(data))
	const maxDataBlobLength = uint64(1<<32 - 1)
	if size == 0 || size > MaxDocumentSize || size > maxDataBlobLength {
		return windows.DataBlob{}, fault.New(fault.Invalid, "bounded "+name+" required")
	}
	return windows.DataBlob{Size: uint32(size), Data: &data[0]}, nil
}

func boundedDPAPIOutput(blob windows.DataBlob, kind fault.Kind, name string) ([]byte, error) {
	if blob.Data == nil || blob.Size == 0 || blob.Size > uint32(MaxDocumentSize) {
		return nil, fault.New(kind, "bounded "+name+" required")
	}
	return append([]byte(nil), unsafe.Slice(blob.Data, int(blob.Size))...), nil
}
