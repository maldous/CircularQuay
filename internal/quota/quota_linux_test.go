//go:build linux

package quota

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
)

func TestLinuxQuotaABISizesAndBlockConversion(t *testing.T) {
	if got := unsafe.Sizeof(quotaStatV{}); got != 160 {
		t.Fatalf("fs_quota_statv size = %d, want 160", got)
	}
	if got := unsafe.Sizeof(diskQuota{}); got != 112 {
		t.Fatalf("fs_disk_quota size = %d, want 112", got)
	}
	if blocksToBytes(2) != 1024 || blocksToBytes(^uint64(0)) != ^uint64(0) {
		t.Fatal("quota block conversion is not saturating 512-byte conversion")
	}
}

func TestStatusForDirectoryRequiresFlagsAndReadsCurrentUIDQuota(t *testing.T) {
	control := func(_ int, command, id int, address unsafe.Pointer) error {
		switch command {
		case qXGetQStatV:
			stat := (*quotaStatV)(address)
			stat.Flags = fsQuotaUserAccount | fsQuotaUserEnforced
		case qXGetQuota:
			if id != int((*diskQuota)(address).ID) {
				t.Fatal("quota query did not use disk quota identity")
			}
			disk := (*diskQuota)(address)
			disk.BlockHardLimit = 4
			disk.BlockCount = 2
			disk.InodeHardLimit = 7
			disk.InodeCount = 3
		default:
			t.Fatalf("unexpected quota command: %x", command)
		}
		return nil
	}
	status, err := statusForDirectoryWithControl(9, control)
	if err != nil {
		t.Fatal(err)
	}
	if status.HardBytes != 2048 || status.UsedBytes != 1024 || status.HardInodes != 7 || status.UsedInodes != 3 || !status.UserEnforced {
		t.Fatalf("unexpected quota status: %#v", status)
	}
}

func TestStatusForDirectoryFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		control quotaControlFunc
		want    error
	}{
		{"status error", func(int, int, int, unsafe.Pointer) error { return errors.New("synthetic") }, ErrUnsupported},
		{"accounting only", func(_ int, command, _ int, address unsafe.Pointer) error {
			if command == qXGetQStatV {
				(*quotaStatV)(address).Flags = fsQuotaUserAccount
			}
			return nil
		}, ErrUnenforced},
		{"quota error", func(_ int, command, _ int, address unsafe.Pointer) error {
			if command == qXGetQStatV {
				(*quotaStatV)(address).Flags = fsQuotaUserAccount | fsQuotaUserEnforced
				return nil
			}
			return errors.New("synthetic")
		}, ErrUnsupported},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := statusForDirectoryWithControl(9, test.control)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestQuotaControlRejectsInvalidDescriptor(t *testing.T) {
	if err := quotaControl(-1, qXGetQStatV, 0, unsafe.Pointer(&quotaStatV{})); err == nil {
		t.Fatal("invalid descriptor accepted")
	}
}

func TestNativeProviderOnlyReturnsSafeQuotaFailures(t *testing.T) {
	_, err := Native().Status(t.TempDir())
	if err != nil && !errors.Is(err, ErrUnsupported) && !errors.Is(err, ErrUnenforced) {
		t.Fatalf("native probe returned an unexpected error: %v", err)
	}
	if _, err = Native().Status("/path/that/does/not/exist"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unopened directory error = %v", err)
	}
}

func TestValidateIdentityRejectsRootAndQuotaBypassCapability(t *testing.T) {
	clean := []byte("CapInh:\t0000000000000000\nCapPrm:\t0000000000000000\nCapEff:\t0000000000000000\nCapAmb:\t0000000000000000\n")
	if err := validateIdentity(1000, 1000, clean); err != nil {
		t.Fatalf("unprivileged identity rejected: %v", err)
	}
	if err := validateIdentity(0, 1000, clean); !errors.Is(err, ErrPrivileged) {
		t.Fatalf("real root accepted: %v", err)
	}
	if err := validateIdentity(1000, 0, clean); !errors.Is(err, ErrPrivileged) {
		t.Fatalf("effective root accepted: %v", err)
	}
	for _, field := range []string{"CapInh", "CapPrm", "CapEff", "CapAmb"} {
		status := strings.Replace(string(clean), field+":\t0000000000000000", field+":\t0000000001000000", 1)
		if err := validateIdentity(1000, 1000, []byte(status)); !errors.Is(err, ErrPrivileged) {
			t.Fatalf("%s CAP_SYS_RESOURCE accepted: %v", field, err)
		}
	}
	admin := strings.Replace(string(clean), "CapEff:\t0000000000000000", "CapEff:\t0000000000200000", 1)
	if err := validateIdentity(1000, 1000, []byte(admin)); !errors.Is(err, ErrPrivileged) {
		t.Fatalf("CAP_SYS_ADMIN accepted: %v", err)
	}
	if err := validateIdentity(1000, 1000, []byte("CapEff:\t0\n")); !errors.Is(err, ErrPrivileged) {
		t.Fatalf("incomplete capability status accepted: %v", err)
	}
}

func TestCapabilitySetParsingAndFilePrivilegeGuards(t *testing.T) {
	if _, ok := capabilitySets([]byte("CapInh:\tnot-hex\nCapPrm:\t0\nCapEff:\t0\nCapAmb:\t0\n")); ok {
		t.Fatal("malformed capability set accepted")
	}
	if capabilitySetName("CapBnd") || !capabilitySetName("CapEff") {
		t.Fatal("capability field selection changed")
	}
}

func TestNativeIdentityAndExecutablePrivilegeGuards(t *testing.T) {
	identity := testNativeIdentity(t)
	path := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(path, []byte("synthetic"), 0700); err != nil {
		t.Fatal(err)
	}
	err := ValidateUnprivilegedExecution(path)
	if identity == nil && err != nil {
		t.Fatalf("ordinary helper rejected: %v", err)
	}
	if identity != nil && !errors.Is(err, ErrPrivileged) {
		t.Fatalf("privileged identity was not rejected: %v", err)
	}
	testFilePrivilegeGuards(t)
	if err := ValidateUnprivilegedExecution(path + "-missing"); !errors.Is(err, ErrPrivileged) {
		t.Fatalf("missing helper accepted: %v", err)
	}
}

func testNativeIdentity(t *testing.T) error {
	t.Helper()
	identity := ValidateFetchIdentity()
	if identity != nil && !errors.Is(identity, ErrPrivileged) {
		t.Fatalf("identity result = %v", identity)
	}
	return identity
}

func testFilePrivilegeGuards(t *testing.T) {
	t.Helper()
	for _, test := range []struct {
		name       string
		mode       os.FileMode
		capability []byte
		err        error
	}{
		{"setuid", os.ModeSetuid | 0700, nil, nil},
		{"setgid", os.ModeSetgid | 0700, nil, nil},
		{"capability", 0700, []byte{1}, nil},
		{"xattr failure", 0700, nil, errors.New("synthetic")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateFilePrivilege(test.mode, test.capability, test.err); !errors.Is(err, ErrPrivileged) {
				t.Fatalf("privileged helper accepted: %v", err)
			}
		})
	}
}
