//go:build linux

package quota

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	qXGetQuota  = 0x5803
	qXGetQStatV = 0x5808

	fsQuotaUserAccount  = 1 << 0
	fsQuotaUserEnforced = 1 << 1
	fsUserQuota         = 1 << 0
	quotaVersion        = 1
	quotaBlockBytes     = 512
	capSysAdmin         = 21
	capSysResource      = 24
)

// Native obtains enforcement information using quotactl_fd(2), attached to
// the target directory descriptor. Q_XGETQSTATV supplies the enforcement bit;
// Q_XGETQUOTA supplies hard limits and current use for the current UID.
func Native() Provider { return nativeProvider{} }

// ValidateFetchIdentity rejects identities that can bypass hard quota limits.
// It reads only the calling process's non-secret capability metadata.
func ValidateFetchIdentity() error {
	return validateIdentity(os.Getuid(), os.Geteuid(), readCapabilityStatus())
}

// ValidateUnprivilegedExecution also rejects a helper carrying any file
// capability or set-ID bit. Such a helper could gain CAP_SYS_RESOURCE at exec.
func ValidateUnprivilegedExecution(path string) error {
	if err := ValidateFetchIdentity(); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return ErrPrivileged
	}
	capability, err := fileCapability(path)
	return validateFilePrivilege(info.Mode(), capability, err)
}

func validateFilePrivilege(mode os.FileMode, capability []byte, err error) error {
	if err != nil || mode&(os.ModeSetuid|os.ModeSetgid) != 0 || len(capability) != 0 {
		return ErrPrivileged
	}
	return nil
}

func readCapabilityStatus() []byte {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return nil
	}
	return data
}

func validateIdentity(realUID, effectiveUID int, status []byte) error {
	if realUID == 0 || effectiveUID == 0 {
		return ErrPrivileged
	}
	sets, ok := capabilitySets(status)
	if !ok || sets&dangerousCapabilityMask() != 0 {
		return ErrPrivileged
	}
	return nil
}

func dangerousCapabilityMask() uint64 {
	return uint64(1)<<capSysResource | uint64(1)<<capSysAdmin
}

func capabilitySets(status []byte) (uint64, bool) {
	var sets uint64
	found := 0
	for _, line := range strings.Split(string(status), "\n") {
		name, value, ok := strings.Cut(line, ":\t")
		if !ok || !capabilitySetName(name) {
			continue
		}
		parsed, err := strconv.ParseUint(strings.TrimSpace(value), 16, 64)
		if err != nil {
			return 0, false
		}
		sets |= parsed
		found++
	}
	return sets, found == 4
}

func capabilitySetName(name string) bool {
	return name == "CapInh" || name == "CapPrm" || name == "CapEff" || name == "CapAmb"
}

func fileCapability(path string) ([]byte, error) {
	data := make([]byte, 64)
	size, err := unix.Getxattr(path, "security.capability", data)
	if errors.Is(err, unix.ENODATA) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data[:size], nil
}

type nativeProvider struct{}

func (nativeProvider) Status(path string) (Status, error) {
	directory, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return Status{}, ErrUnsupported
	}
	defer unix.Close(directory)
	return statusForDirectory(directory)
}

func statusForDirectory(directory int) (Status, error) {
	return statusForDirectoryWithControl(directory, quotaControl)
}

type quotaControlFunc func(int, int, int, unsafe.Pointer) error

func statusForDirectoryWithControl(directory int, control quotaControlFunc) (Status, error) {
	var stat quotaStatV
	stat.Version = quotaVersion
	if err := control(directory, qXGetQStatV, 0, unsafe.Pointer(&stat)); err != nil {
		return Status{}, ErrUnsupported
	}
	if stat.Flags&fsQuotaUserAccount == 0 || stat.Flags&fsQuotaUserEnforced == 0 {
		return Status{}, ErrUnenforced
	}
	var disk diskQuota
	disk.Version = quotaVersion
	disk.Flags = fsUserQuota
	disk.ID = uint32(os.Getuid())
	if err := control(directory, qXGetQuota, int(disk.ID), unsafe.Pointer(&disk)); err != nil {
		return Status{}, ErrUnsupported
	}
	return Status{
		HardBytes:    blocksToBytes(disk.BlockHardLimit),
		UsedBytes:    blocksToBytes(disk.BlockCount),
		HardInodes:   disk.InodeHardLimit,
		UsedInodes:   disk.InodeCount,
		UserEnforced: true,
	}, nil
}

func quotaControl(directory, command, id int, address unsafe.Pointer) error {
	_, _, errno := unix.Syscall6(unix.SYS_QUOTACTL_FD, uintptr(directory), uintptr(command), uintptr(id), uintptr(address), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func blocksToBytes(blocks uint64) uint64 {
	if blocks > ^uint64(0)/quotaBlockBytes {
		return ^uint64(0)
	}
	return blocks * quotaBlockBytes
}

type quotaFileStatV struct {
	Inode   uint64
	Blocks  uint64
	Extents uint32
	Padding uint32
}

// quotaStatV matches struct fs_quota_statv from linux/dqblk_xfs.h.
type quotaStatV struct {
	Version      int8
	Padding1     uint8
	Flags        uint16
	InCore       uint32
	User         quotaFileStatV
	Group        quotaFileStatV
	Project      quotaFileStatV
	BlockTimer   int32
	InodeTimer   int32
	RTBlockTimer int32
	BlockWarn    uint16
	InodeWarn    uint16
	RTBlockWarn  uint16
	Padding3     uint16
	Padding4     uint32
	Padding2     [7]uint64
}

// diskQuota matches fs_disk_quota_t from linux/dqblk_xfs.h.
type diskQuota struct {
	Version         int8
	Flags           int8
	FieldMask       uint16
	ID              uint32
	BlockHardLimit  uint64
	BlockSoftLimit  uint64
	InodeHardLimit  uint64
	InodeSoftLimit  uint64
	BlockCount      uint64
	InodeCount      uint64
	InodeTimer      int32
	BlockTimer      int32
	InodeWarnings   uint16
	BlockWarnings   uint16
	InodeTimerHigh  int8
	BlockTimerHigh  int8
	RTTimerHigh     int8
	Padding2        int8
	RTBlockHard     uint64
	RTBlockSoft     uint64
	RTBlockCount    uint64
	RTBlockTimer    int32
	RTBlockWarnings uint16
	Padding3        int16
	Padding4        [8]byte
}
