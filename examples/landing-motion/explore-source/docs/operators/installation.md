# Operator installation and enrollment

Use a qualified release and platform from the qualification matrix. Existing production hosts are not modified by the build task; the operator chooses where to install and returns evidence from that infrastructure.

## Separate installation from enrollment

Installing a binary does not create a provider identity, grant repository access, seed an offline snapshot, configure a supervisor, or enable MCP. Basic synthetic local use needs no external provider account.

Before proceeding record the selected release/digest, OS/architecture, CA trust source, protected configuration/state locations, workload owner, enabled dependencies, source/destination scope and recovery custodian. Store actual identities outside this repository.

## Native or source route

For native releases, obtain the matching archive and release evidence, authenticate the expected publisher/repository, verify the checksum, extract into a versioned user-owned directory, and add only that directory to the chosen launcher's executable search path. A browser/file manager is an alternative to POSIX tools. Do not execute an unchecked installer stream.

For a source build, use the pinned Go toolchain and a clean tagged checkout:

```sh
go mod download
go mod verify
go build -trimpath -o ./dist/cq ./cmd/cq
go build -trimpath -o ./dist/cq-audit ./cmd/cq-audit
./dist/cq --config examples/demo.json plan
```

On Windows use `.exe` output names and explicit PowerShell paths. Go is a build prerequisite, not a normal binary-runtime prerequisite. Optional feature dependencies are checked separately; a binary alone does not include SOPS, Git, or a vendor CLI.

## Protected policy and identities

1. Create an operator-owned configuration directory and a separate runtime state directory.
2. Restrict modification to the intended operator/service administrator. The application must not rewrite its helper, policy or signing trust.
3. Copy a synthetic example; validate it before adding a live source.
4. Select a workload identity explicitly. Do not use an administrator's interactive session for unattended services.
5. Register new age public recipients or grant exact provider resources; never manually share one private identity across all hosts.
6. Provision referenced bootstrap material through protected OS storage. Do not place credential strings in policy, argv, source code or shell history.
7. Verify allowed and denied synthetic reads before enrollment.

Literal JSON/dotenv files are appropriate for non-secret configuration and fabricated tests. For actual offline secrets use SOPS and protected recipients.

## Native delivery

### Linux/systemd

Use a dedicated service user and the supplied qualified template. `LoadCredentialEncrypted=` consumes systemd's format, not SOPS ciphertext. Generate a native encrypted projection through the protected adapter; do not point it at `.sops.json`.

Keep offline/hybrid boot independent of a successful network fetch. Online launch must perform fresh resolution on every invocation, including `Restart=` paths. A one-shot dependency left active after success does not satisfy that requirement. Restart an affected consumer to refresh immutable native credentials; `daemon-reload` alone does not refresh them.

Inspect and adapt executable, configuration and state paths before enabling units. Enable only the chosen synthetic workload first. Verify native user permissions, reboot with networking unavailable, reconnect, rotate and restore before enrolling operational consumers.

### Windows

Use a dedicated native service identity or enabled service SID. Verify the SID in the process token. DPAPI LocalMachine is not per-service authorization: protect identity/projection/helper/configuration files with noninheriting ACLs for the required SYSTEM/administrator and consuming service SID only.

Use SCM for long-lived applications and Task Scheduler for reconciliation. Verify no-login cold boot under the actual identity. Do not assume a human Credential Manager session or S4U task has the required network/protected-file access. No machine-wide secret environment variables.

### Termux/Android

Install matching-source Termux applications and native dependencies, not a generic Linux ARM64 archive. Store identities in Termux's private directory, never shared `/sdcard` storage. Use runit/Termux:Boot and qualify the device's battery/background settings.

The device needs its first unlock after reboot. Android may terminate background processes, and multiple Termux processes share one application UID. Keep workloads needing guaranteed pre-unlock recovery elsewhere.

### Containers

Use BuildKit mounts for build-time credentials and scoped native secret/runtime files for execution. Never copy a real value into an image layer, `ARG`, `ENV`, manifest, or build log. Mount only assigned policy/state/credentials; do not mount the Docker socket or an estate-wide identity.

Compose file secret delivery is not Swarm's encrypted secret store. Changed environment/configuration needs targeted recreation; a plain restart and single-file atomic replacement are not assumed to refresh a running bind mount.

## SDK, agent and operational enrollment

Applications select [Go, TypeScript or Python](../sdks.md) and the local or MCP
adapter explicitly. Verify package provenance, dependency locks and the linked
core/native runtime version independently; Python/TypeScript local packages do
not silently install a trusted binary. Agent clients install the versioned
[skill or client plugin](../agent-integrations.md) and retain their configured
OS/service identity and per-operation grants.

Configure remote telemetry only through protected `observability` settings,
collector trust and credential references. Confirm synthetic redaction, exporter
outage and bounded shutdown; configure required audit separately. Select release
channel/check interval and an optional local minimum version in `updates`.
Startup notices continue without consuming workload input; upgrades follow the
verified artifact/migration procedure, never a feed-triggered installation.

## Post-installation acceptance

Run policy validation, a synthetic workload, denial tests, scheduler checks, offline/cold-boot recovery, issuer rotation and restore. Record the installed artifact digest and platform versions. Enable optional live providers and MCP only after their separate qualification. Review backup retention, update ownership and revocation before declaring the host enrolled.
