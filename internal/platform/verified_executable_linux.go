//go:build linux

package platform

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
)

// VerifiedExecutable retains the executable object checked by
// OpenVerifiedExecutable. Its zero value cannot launch a process. Copies share
// one lifetime; Close invalidates all copies for subsequent launches.
//
// This Linux-only boundary is not yet used by production callers. It closes
// pathname replacement races, not modification by a trusted owner or root.
// Qualification of native executables and interpreter scripts is separate.
type VerifiedExecutable struct {
	state *verifiedExecutableState
}

type verifiedExecutableState struct {
	mu   sync.Mutex
	file *os.File
	arg0 string
}

// VerifiedExecutionOptions carries caller-owned process policy. Callers must
// validate arguments, directory, environment, streams, limits and authorization
// before Run. A nil Environment means an empty environment, never inheritance.
// Blocking readers/writers require their own cancellation contract; WaitDelay
// bounds pipe draining but neither contains descendants nor interrupts arbitrary
// caller I/O. No executable path or descriptor is exposed through these options.
type VerifiedExecutionOptions struct {
	Directory   string
	Environment []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	WaitDelay   time.Duration
}

// OpenVerifiedExecutable checks an absolute path, its ownership, permissions,
// ancestors, bounded size and expected SHA-256, retaining the checked open file
// until Close. The digest is mandatory and authenticates bytes, not arguments,
// an interpreter or executable dependencies. The caller must close the result.
func OpenVerifiedExecutable(path, expectedSHA256 string) (*VerifiedExecutable, error) {
	expected, err := expectedExecutableDigest(path, expectedSHA256)
	if err != nil {
		return nil, err
	}
	file, resolved, before, err := openVerifiedExecutable(path)
	if err != nil {
		return nil, err
	}
	if err := verifyRetainedExecutable(file, resolved, before, expected); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &VerifiedExecutable{state: &verifiedExecutableState{file: file, arg0: path}}, nil
}

func verifyRetainedExecutable(file *os.File, resolved string, before os.FileInfo, expected []byte) error {
	// Check the open object's ownership as well as the named object's policy.
	if !trustedFile(before, false) || !withinExecutableReadBound(before.Size()) {
		return fault.New(fault.Trust, executableSizeOperation)
	}
	digest, exceeded, err := boundedExecutableDigest(file, maxVerifiedExecutableBytes)
	if err != nil || exceeded {
		return fault.New(fault.Trust, executableSizeOperation)
	}
	if !stableVerifiedExecutable(file, resolved, before) || !bytesEqual(digest, expected) {
		return fault.New(fault.Trust, "registered executable changed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fault.New(fault.Trust, "registered executable unavailable")
	}
	return nil
}

// Close releases the retained descriptor. It is idempotent, safe alongside Run,
// and waits only for an in-progress process creation, not for child completion.
// A started child owns its inherited descriptor independently of this handle.
func (executable *VerifiedExecutable) Close() error {
	if executable == nil || executable.state == nil {
		return nil
	}
	state := executable.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.file == nil {
		return nil
	}
	err := state.file.Close()
	state.file = nil
	return err
}

// Run executes the retained object with arguments excluding argv[0], which is
// the originally checked path. It returns ordinary os/exec start/wait errors
// (including *exec.ExitError) for caller-specific fault mapping. Cancellation
// retains CommandContext semantics. Run always waits for a started child.
//
// Linux maps ExtraFiles[0] to descriptor 3. The literal proc path is the only
// launch target: missing procfs or an unsupported executable fails without a
// pathname or shell fallback. Descriptor 3 intentionally survives exec so a
// shebang interpreter can reopen it. Such scripts require a trusted interpreter
// and native acceptance evidence; the script digest does not pin that interpreter.
func (executable *VerifiedExecutable) Run(ctx context.Context, arguments []string, options VerifiedExecutionOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", arguments...)
	cmd.Dir = options.Directory
	cmd.Env = append([]string{}, options.Environment...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = options.Stdin, options.Stdout, options.Stderr
	cmd.WaitDelay = options.WaitDelay
	if err := executable.start(cmd); err != nil {
		return err
	}
	return cmd.Wait()
}

func (executable *VerifiedExecutable) start(cmd *exec.Cmd) error {
	if executable == nil || executable.state == nil {
		return os.ErrClosed
	}
	state := executable.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.file == nil {
		return os.ErrClosed
	}
	cmd.Args[0] = state.arg0
	cmd.ExtraFiles = []*os.File{state.file}
	// Hold the parent handle until Start has either transferred descriptor 3
	// to the child or failed. Close must not race descriptor reuse in Start.
	return cmd.Start()
}
