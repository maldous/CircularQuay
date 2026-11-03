//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"golang.org/x/sys/windows"
)

const syntheticServiceSID = "S-1-5-80-1-2-3-4-5"

func TestDPAPIRejectsOversizeProjectionBeforeFFI(t *testing.T) {
	called := false
	_, err := protectDPAPIWith(make([]byte, MaxDocumentSize+1), func(*windows.DataBlob, *windows.DataBlob) error {
		called = true
		return nil
	})
	if !fault.Is(err, fault.Invalid) || called {
		t.Fatalf("oversize DPAPI projection result = %v, FFI called %t", err, called)
	}
	for _, blob := range []windows.DataBlob{{}, {Size: uint32(MaxDocumentSize + 1)}} {
		if _, err := boundedDPAPIOutput(blob, fault.Denied, "synthetic projection"); !fault.Is(err, fault.Denied) {
			t.Fatalf("invalid DPAPI output blob result = %v", err)
		}
	}
}

func TestDPAPIGrantFailurePreservesPublishedProjection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projection.bin")
	old := []byte("SYNTHETIC_OLD_DPAPI_PROJECTION")
	if err := WriteProtected(path, old); err != nil {
		t.Fatal(err)
	}
	if err := grantFile(path, syntheticServiceSID); err != nil {
		t.Fatal(err)
	}
	before := fileDACLText(t, path)

	err := applyDPAPIWithGrant(model.Target{Path: path, Resource: syntheticServiceSID}, []byte("SYNTHETIC_NEW_SECRET"), func(string, string) error {
		return errors.New("synthetic ACL failure")
	})
	if !fault.Is(err, fault.Apply) || strings.Contains(err.Error(), "SYNTHETIC_NEW_SECRET") {
		t.Fatalf("grant failure result = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(old) {
		t.Fatalf("published bytes after failed grant = %q, %v", after, err)
	}
	if got := fileDACLText(t, path); got != before {
		t.Fatalf("published ACL changed after failed grant: before %q after %q", before, got)
	}
	assertNoDPAPIStages(t, filepath.Dir(path))
}

func TestDPAPIStagesServiceGrantBeforePublication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projection.bin")
	old := []byte("SYNTHETIC_OLD_DPAPI_PROJECTION")
	if err := WriteProtected(path, old); err != nil {
		t.Fatal(err)
	}
	grantCalled := false
	err := applyDPAPIWithGrant(model.Target{Path: path, Resource: syntheticServiceSID}, []byte("SYNTHETIC_NEW_SECRET"), stagedServiceGrant(t, path, old, &grantCalled))
	if err != nil || !grantCalled {
		t.Fatalf("staged DPAPI publication result = %v, grant called %t", err, grantCalled)
	}
	assertDPAPIProjectionPublished(t, path, old)
	if !daclGrantsRead(t, path, syntheticServiceSID) {
		t.Fatal("published DPAPI projection lacks the configured service read grant")
	}
	assertNoDPAPIStages(t, filepath.Dir(path))
}

func stagedServiceGrant(t *testing.T, path string, old []byte, called *bool) func(string, string) error {
	t.Helper()
	return func(stage, sid string) error {
		*called = true
		if stage == path || sid != syntheticServiceSID {
			t.Fatalf("grant target = %q for %q", stage, sid)
		}
		published, err := os.ReadFile(path)
		if err != nil || string(published) != string(old) {
			t.Fatalf("destination changed before service grant: %q, %v", published, err)
		}
		return grantFile(stage, sid)
	}
}

func assertDPAPIProjectionPublished(t *testing.T, path string, old []byte) {
	t.Helper()
	published, err := os.ReadFile(path)
	if err != nil || string(published) == string(old) || strings.Contains(string(published), "SYNTHETIC_NEW_SECRET") {
		t.Fatalf("final DPAPI projection = %q, %v", published, err)
	}
}

func fileDACLText(t *testing.T, path string) string {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	return sd.String()
}

func daclGrantsRead(t *testing.T, path, sid string) bool {
	t.Helper()
	wanted, err := windows.StringToSid(sid)
	if err != nil {
		t.Fatal(err)
	}
	acl := fileDACL(t, path)
	for index := uint32(0); index < uint32(acl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, index, &ace); err != nil || ace == nil {
			t.Fatalf("file ACE %d = %v", index, err)
		}
		if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE && aceSID(ace).Equals(wanted) && ace.Mask&windows.FILE_READ_DATA != 0 && !writesFile(ace.Mask) {
			return true
		}
	}
	return false
}

func assertNoDPAPIStages(t *testing.T, directory string) {
	t.Helper()
	stages, err := filepath.Glob(filepath.Join(directory, ".cquay-stage-*"))
	if err != nil || len(stages) != 0 {
		t.Fatalf("DPAPI staging files = %v, %v", stages, err)
	}
}
