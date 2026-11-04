package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

const (
	pipeDrainWaitDelay         = 250 * time.Millisecond
	maxVerifiedExecutableBytes = 128 << 20
)

// Run executes a workload with a clean environment. It never passes bootstrap
// credentials to the workload. Override is for an explicitly local CLI caller;
// MCP actions must use the enrolled workload command and never pass an override.
func Run(ctx context.Context, work model.Workload, gen model.Generation, override []string) (int, error) {
	return RunWithIO(ctx, work, gen, override, os.Stdin, os.Stdout, os.Stderr)
}

// RunWithIO permits an MCP caller to discard application output explicitly.
// Caller-supplied blocking readers/writers need their own cancellation contract;
// the pipe-drain bound does not contain descendants or interrupt arbitrary I/O.
func RunWithIO(ctx context.Context, work model.Workload, gen model.Generation, override []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	args, env, directory, err := launchInputs(work, gen, override)
	if err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	cmd.Dir = directory
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	// A child can exit while a descendant retains its standard streams. Bound
	// the pipe drain so that such a descriptor does not indefinitely retain the
	// caller, but do not claim this closes or contains that descendant.
	cmd.WaitDelay = pipeDrainWaitDelay
	runErr := cmd.Run()
	if err := ctx.Err(); err != nil {
		// CommandContext can surface cancellation as an ExitError after killing
		// the immediate child. Preserve the caller's cancellation contract rather
		// than misrepresenting it as an ordinary workload exit status.
		return 0, err
	}
	if runErr != nil {
		if exit, ok := runErr.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, fault.New(fault.Apply, "launch workload")
	}
	return 0, nil
}

func launchInputs(work model.Workload, gen model.Generation, override []string) ([]string, []string, string, error) {
	args, err := ValidateCommandIdentity(work, override)
	if err != nil {
		return nil, nil, "", err
	}
	env, err := Environment(work.Environment, gen)
	if err != nil {
		return nil, nil, "", err
	}
	if !generationCurrent(gen) {
		return nil, nil, "", fault.New(fault.Expired, "generation expired before launch")
	}
	directory, err := launchDirectory(work, gen, args[0], env)
	if err != nil {
		return nil, nil, "", err
	}
	return args, env, directory, nil
}

// ValidateCommandIdentity selects and verifies the command admitted for a
// workload. A pinned command admits only its exact configured argument vector,
// including a local override. Callers use this before resolution effects, and
// launchInputs repeats it immediately before environment construction to narrow
// the time-of-check/time-of-use window. It does not authorize the command or
// prevent policy loading performed before this boundary.
func ValidateCommandIdentity(work model.Workload, override []string) ([]string, error) {
	if work.CommandSHA256 != "" && len(override) != 0 && !sameArgumentVector(work.Command, override) {
		return nil, fault.New(fault.Trust, "registered command override denied")
	}
	args := work.Command
	if len(override) != 0 {
		args = override
	}
	if !validExecutable(args) {
		return nil, fault.New(fault.Invalid, "absolute executable required")
	}
	if err := ValidateExecutable(args[0]); err != nil {
		return nil, err
	}
	if work.CommandSHA256 != "" {
		if err := VerifyExecutable(args[0], work.CommandSHA256); err != nil {
			return nil, err
		}
	}
	return append([]string(nil), args...), nil
}

func sameArgumentVector(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func generationCurrent(gen model.Generation) bool {
	now := time.Now()
	for _, value := range gen.Values {
		if !value.ExpiresAt.IsZero() && !value.ExpiresAt.After(now) {
			return false
		}
	}
	return true
}

// VerifyExecutable binds a registered action to reviewed bytes. Pinning the
// executable does not by itself make editable scripts or arbitrary code safe.
func VerifyExecutable(path, expected string) error {
	decoded, err := expectedExecutableDigest(path, expected)
	if err != nil {
		return err
	}
	f, resolved, before, err := openVerifiedExecutable(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if !withinExecutableReadBound(before.Size()) {
		return fault.New(fault.Trust, "registered executable size")
	}
	digest, exceeded, err := boundedExecutableDigest(f, maxVerifiedExecutableBytes)
	if err != nil {
		return fault.New(fault.Trust, "registered executable size")
	}
	if exceeded {
		return fault.New(fault.Trust, "registered executable size")
	}
	if !stableVerifiedExecutable(f, resolved, before) {
		return fault.New(fault.Trust, "registered executable changed")
	}
	if !bytesEqual(digest, decoded) {
		return fault.New(fault.Trust, "registered executable changed")
	}
	return nil
}

func expectedExecutableDigest(path, expected string) ([]byte, error) {
	decoded, err := hex.DecodeString(expected)
	if err != nil {
		return nil, fault.New(fault.Trust, "invalid executable identity")
	}
	if len(decoded) != sha256.Size {
		return nil, fault.New(fault.Trust, "invalid executable identity")
	}
	if !filepath.IsAbs(path) {
		return nil, fault.New(fault.Trust, "invalid executable identity")
	}
	return decoded, nil
}

func withinExecutableReadBound(size int64) bool {
	return size >= 0 && size <= maxVerifiedExecutableBytes
}

func boundedExecutableDigest(reader io.Reader, maximum int64) ([]byte, bool, error) {
	hash := sha256.New()
	read, err := io.Copy(hash, io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, false, err
	}
	return hash.Sum(nil), read > maximum, nil
}

func stableVerifiedExecutable(file *os.File, resolved string, before os.FileInfo) bool {
	afterFile, err := file.Stat()
	afterName, nameErr := os.Stat(resolved)
	if err != nil || nameErr != nil || !checkNativeSecurity(resolved, false) {
		return false
	}
	return unchangedVerifiedFile(before, afterFile) && unchangedVerifiedFile(before, afterName)
}

func unchangedVerifiedFile(before, after os.FileInfo) bool {
	return os.SameFile(before, after) &&
		before.Size() == after.Size() &&
		secureRegularFile(after)
}

func secureRegularFile(info os.FileInfo) bool {
	return info.Mode().IsRegular() && secureMode(info.Mode(), false)
}

func openVerifiedExecutable(path string) (*os.File, string, os.FileInfo, error) {
	if err := ValidateExecutable(path); err != nil {
		return nil, "", nil, err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", nil, fault.New(fault.Trust, "registered executable unavailable")
	}
	f, err := openInParent(resolved, os.O_RDONLY, 0)
	if err != nil {
		return nil, "", nil, fault.New(fault.Trust, "registered executable unavailable")
	}
	info, err := f.Stat()
	named, namedErr := os.Stat(resolved)
	if err != nil || namedErr != nil || !info.Mode().IsRegular() || !secureMode(info.Mode(), false) || !os.SameFile(info, named) {
		_ = f.Close()
		return nil, "", nil, fault.New(fault.Trust, "registered executable permissions")
	}
	return f, resolved, info, nil
}

// ValidateExecutable rejects mutable helper binaries before supplying bootstrap
// credentials. Authenticating upstream release bytes remains an operator step;
// mode and ACL checks do not replace signature verification.
func ValidateExecutable(path string) error {
	if !filepath.IsAbs(path) || strings.ContainsRune(path, '\x00') {
		return fault.New(fault.Trust, "absolute trusted executable required")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || !secureMode(info.Mode(), false) || !checkNativeSecurity(path, false) {
		return fault.New(fault.Trust, "trusted executable permissions")
	}
	return nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var different byte
	for i := range a {
		different |= a[i] ^ b[i]
	}
	return different == 0
}

// Environment constructs a deterministic, non-inheriting process environment.
func Environment(base map[string]string, gen model.Generation) ([]string, error) {
	values := make(map[string]string, len(base)+len(gen.Values)+2)
	seen := make(map[string]bool)
	for key, value := range nativeEnvironment() {
		values[key] = value
		seen[strings.ToUpper(key)] = true
	}
	for key, value := range base {
		if err := addEnvironment(values, seen, key, value, false); err != nil {
			return nil, err
		}
	}
	for key, value := range gen.Values {
		if err := addEnvironment(values, seen, key, value.Data, true); err != nil {
			return nil, err
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result, nil
}

func addEnvironment(values map[string]string, seen map[string]bool, key, value string, secret bool) error {
	if key == "" || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, '\x00') || seen[strings.ToUpper(key)] {
		return fault.New(fault.Invalid, "environment collision or invalid value")
	}
	if secret && forbiddenEnvironment(key) {
		return fault.New(fault.Trust, "bootstrap environment denied")
	}
	seen[strings.ToUpper(key)] = true
	values[key] = value
	return nil
}

func forbiddenEnvironment(key string) bool {
	u := strings.ToUpper(key)
	for _, prefix := range []string{"SOPS_", "VAULT_", "BAO_", "LD_", "DYLD_"} {
		if strings.HasPrefix(u, prefix) {
			return true
		}
	}
	switch u {
	case "PATH", "HOME", "USERPROFILE", "COMSPEC", "NODE_OPTIONS", "PYTHONPATH", "BASH_ENV", "ENV", "GIT_CONFIG", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM":
		return true
	}
	return false
}

func validExecutable(args []string) bool {
	if len(args) == 0 || !filepath.IsAbs(args[0]) || strings.ContainsRune(args[0], '\x00') {
		return false
	}
	switch strings.ToLower(filepath.Ext(args[0])) {
	case ".cmd", ".bat", ".ps1", ".sh":
		return false
	}
	for _, arg := range args[1:] {
		if strings.ContainsRune(arg, '\x00') {
			return false
		}
	}
	return true
}
