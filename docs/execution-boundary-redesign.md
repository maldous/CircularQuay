# Execution-boundary redesign

## Status and decision

This document is a design for preserving every current execution and quota capability while removing Gosec exceptions. **It changes no implementation, is not qualified, and is not deployed.** No feature may be disabled to meet the analyzer result, and no `#nosec`, global exclusion, dynamic `Cmd.Path` assignment, or check-only wrapper is an acceptable substitute.

The target is a capability boundary: callers retain their current command, provider, workload, activation, and quota semantics, but the operating system receives a verified executable or directory handle rather than a pathname that can be substituted after validation.

## Replacement scope

The eventual implementation replaces the direct launch points in these files, retaining their caller-specific validation, environment, limits, cancellation, and fault mapping:

- `internal/platform/process.go` — enrolled workload command execution.
- `internal/platform/activation_linux.go` — fixed systemd activation.
- `internal/source/sops.go` — configured SOPS operations.
- `internal/source/vendor.go` — configured Bitwarden and 1Password CLI operations.
- `internal/repository/git.go` — native Git operations including hardened SHA-1 and SSH-signature policy.
- `internal/assurance/runner.go` — manifest-bound assurance tools.
- `internal/quota/quota_linux.go` — current-UID user hard-quota inspection.

The shared execution capability may preserve existing Git and SOPS behavior. Replacing those tools with in-process libraries is neither required nor assumed by this design.

## Verified executable capability

`internal/platform` will own an opaque `VerifiedExecutable` capability. It is created only after the existing absolute-path, ownership, mode, ancestor, size, and expected SHA-256 checks have been applied to one open executable object. Its constructor retains the checked handle through process creation; callers cannot manufacture it from a path.

The capability does not validate command arguments. Each present caller continues to validate its own fixed subcommands, source identifiers, working directory, environment, input/output limits, credentials, context deadline, and error mapping. This prevents a generic string-to-process helper from becoming a new authority boundary.

### Linux

The Linux launcher will pass the verified file as `Cmd.ExtraFiles[0]` and execute the literal path `/proc/self/fd/3`. `os/exec` assigns `ExtraFiles[0]` to child descriptor 3. The child therefore executes the checked open file description while retaining normal `Cmd.Dir`, environment, standard streams, `CommandContext`, and `WaitDelay` behavior.

This is a real pathname TOCTOU closure: execution is authorized by the retained file handle rather than by reopening the checked pathname. Linux documents descriptor execution specifically as a checksum-before-execution pattern. The existing permissions and ancestry checks remain mandatory because descriptor execution cannot protect bytes that a malicious writer is still allowed to modify.

The implementation must not silently fall back to the original pathname. It must also explicitly establish its script contract. Linux documents special close-on-exec behavior for interpreter scripts, so native executables and supported scripts require separate acceptance coverage before either is promised.

For systemd, production execution will use the literal fixed command family. The `path` field in the internal test seam must not remain a production-controlled input; tests can inject a narrow runner instead.

### Windows

The Windows backend will use maintained `golang.org/x/sys/windows` APIs to open the canonical executable with `GENERIC_READ | GENERIC_EXECUTE` and only `FILE_SHARE_READ`. It hashes and validates through that held handle, then calls `CreateProcess` with the exact canonical application path while retaining the handle until process creation completes.

The held image handle denies writer and delete sharing, preventing a cooperating process from changing, replacing, or deleting the reviewed image during the validation-to-start window. This is a path-based Windows process launch with a file-handle lock invariant, not a claim that Windows executes directly by handle.

The backend must preserve current working-directory, explicit-environment, DLL-search, standard-stream, cancellation, exit-status, and cleanup semantics. Its process lifecycle needs an explicit owner, including timeout cancellation and any required job-object containment, before it can replace `os/exec` on Windows.

### Darwin and other Unix hosts

No documented, portable Darwin descriptor-execution primitive has been established for the supported binary and script forms. A Darwin backend is accepted only after a host probe and native acceptance tests demonstrate its exact launch contract. If the probe fails, that host must report a clear unsupported execution-boundary condition; it must not reopen a previously checked dynamic pathname.

A native launcher may be introduced only with a separately reviewed distribution, signing, upgrade, and rollback design. Whether that component can remain CGO-free in the main Go build is unproved and must be decided through the host-probe work, rather than assumed.

## Linux user-quota boundary

The current quota code reads enforced hard byte and inode limits for the process's current effective UID using `quotactl_fd(2)` against an opened directory descriptor. That is a **user quota** capability. Project-quota APIs are not substitutes: they use different accounting and can require elevated privileges.

To preserve the capability without direct Go `unsafe` syscall pointers, introduce a narrowly scoped native quota helper with this contract:

1. The Go process opens the target directory with the existing trusted-directory policy and sends that one descriptor over a Unix `SOCK_SEQPACKET` connection.
2. The helper obtains peer credentials from the socket. It rejects mismatched real/effective IDs, privileged identities, and any request whose claimed identity is not the peer's UID.
3. The request contains a fixed operation code and bounded protocol version only. It accepts no pathname, arbitrary UID, command number, pointer, environment, or executable request.
4. The helper calls the Linux quota ABI only for `Q_XGETQSTATV` and `Q_XGETQUOTA` for that peer UID. It returns a fixed-size response containing the enforcement bit and four unsigned counters: hard bytes, used bytes, hard inodes, and used inodes.
5. Both sides reject malformed lengths, duplicated descriptors, unexpected ancillary data, invalid versions, response overflow, invalid flags, and disconnected peers. Errors reveal no filesystem path or secret.
6. The helper never elevates privileges. It is authenticated as a fixed local component, runs with the caller's identity or a deliberately reviewed service identity, and refuses any operation that could query another user or change quota state.

The helper is a product boundary, not an analyzer workaround. Its source, build provenance, signature, installation ownership/mode, socket ownership/mode, protocol compatibility, and upgrade process require review before adoption. If that component is not accepted, preserving current quota behavior and eliminating direct Go G103 cannot both be claimed.

## Acceptance evidence

Each backend requires tests that prove the security property as well as compatibility:

- Linux: pathname replacement, symlink replacement, write/delete/rename races, verified-content mismatch, `/proc` absence, native binary launch, supported-script launch, distinct working directory, cancellation, exit code, output limits, and descriptor cleanup.
- Windows: write/delete/replace conflicts while the handle is held, canonical-path and command-line quoting, working directory, DLL-search restrictions, cancellation, exit status, process/job cleanup, and handle cleanup.
- Darwin: host probe plus native binary and supported-script tests, working directory, cancellation, and a proof that no dynamic pathname is reopened.
- Quota helper: peer-UID binding, real/effective-UID mismatch, root/capability rejection, arbitrary-UID rejection, malformed packet and ancillary-data handling, closed/extra descriptor handling, counter bounds, un-enforced quota, unsupported filesystem, and no quota mutation.
- All callers: retain current provider/tool argument restrictions, environments, credential boundaries, timeouts, output bounds, and public error categories.

Gosec is rerun only after those tests and platform-specific qualification are authorized. A zero-finding result alone is not acceptance evidence.

## Rollout and backout

1. Land the opaque capability API and Linux implementation behind no behavior-changing fallback. Qualify Linux with the acceptance suite before routing any caller.
2. Migrate one caller category at a time while preserving its existing public contract and rollback tests.
3. Deliver Windows only after native Windows qualification; deliver Darwin only after its host probe and acceptance suite establish a supported primitive.
4. Deliver the quota helper as a separately versioned component with an explicit installation and compatibility contract.
5. Keep the previous implementation available only during a measured migration window. If a backend fails qualification or production health checks, roll back that caller to the last qualified implementation for its OS; do not bypass the capability boundary with unchecked dynamic execution.
6. Remove the old direct syscall/direct dynamic-launch path only after all supported hosts and caller categories have qualified the replacement.

## Primary references

- [Go `os/exec` `Cmd.ExtraFiles` documentation](https://pkg.go.dev/os/exec): extra files become child descriptors `3+i`; it is unsupported on Windows.
- [Go `os/exec` source](https://go.dev/src/os/exec/exec.go): process creation receives the configured directory, environment, and file descriptors.
- [Linux `fexecve(3)`](https://man7.org/linux/man-pages/man3/fexecve.3.html): execution by descriptor, checksum-before-exec rationale, writable-content limitation, and script caveat.
- [Linux `quotactl_fd(2)`](https://man7.org/linux/man-pages/man2/quotactl_fd.2.html): descriptor-scoped quota control interface.
- [x/sys Windows API surface](https://pkg.go.dev/golang.org/x/sys/windows): maintained Windows `CreateFile` and process primitives.
