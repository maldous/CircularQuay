# CircularQuay

**Portable secrets delivery, with explicit trust boundaries.**

A shell profile once made a developer's environment feel simple: open a
terminal, inherit the right variables, and get on with the job. Services,
containers, Windows sessions, CI runners, and hosted applications introduced
different startup and identity boundaries. Multiple AI providers and agent
workflows added more keys, scopes, and places to configure them.

CircularQuay grew out of the frustration of joining those pieces repeatedly. It
connects existing providers, established encryption, and native runtime
delivery through one workload-scoped policy. You choose where online
authorization is required and where an encrypted offline copy is appropriate.

CircularQuay’s app, command, and operations skill are named `cquay` (pronounced “c-key”).

CircularQuay resolves exact values from existing secret providers under a workload-scoped policy, then delivers one complete generation through native operating-system boundaries. It gives teams a portable way to move from provider authorization to a declared workload without inheriting an ambient shell environment or silently applying a partial refresh.

For example, a developer can use a synthetic configuration to plan the `demo` workload and run its declared command with only the values that policy authorizes; the same model can publish an approved encrypted snapshot or prepare a native service delivery target.


![Status: development](https://img.shields.io/badge/status-development-9a6700)
![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-0f766e)
![Implementation: Go](https://img.shields.io/badge/implementation-Go-087f8c)
[![Source workflow status](https://github.com/maldous/CircularQuay/actions/workflows/source.yml/badge.svg?branch=feature%2Finitial-implementation)](https://github.com/maldous/CircularQuay/actions/runs/41633578905)

[![Linux synthetic receipt](https://img.shields.io/badge/Linux%20synthetic%20receipt-24d%20passed-2f855a)](docs/qualification.md#bounded-receipts)
[![Staticcheck scoped receipt](https://img.shields.io/badge/Staticcheck%20scoped%20receipt-passed-2f855a)](docs/qualification.md#bounded-receipts)
[![govulncheck scoped receipt](https://img.shields.io/badge/govulncheck%20scoped%20receipt-0%20reachable-2f855a)](docs/qualification.md#bounded-receipts)
[![Cyclomatic scope](https://img.shields.io/badge/gocyclo%20scoped%20receipt-%E2%89%A410-2f855a)](docs/qualification.md#bounded-receipts)
[![Script and workflow analyzers](https://img.shields.io/badge/script%20and%20workflow%20receipts-passed%20scope-2f855a)](docs/qualification.md#bounded-receipts)
[![TLC finite model](https://img.shields.io/badge/TLC%20finite%20model-bounded%20pass-2f855a)](docs/qualification.md#bounded-receipts)

## Quick start

Until the first qualified release, build `cquay` from this development source
with Go 1.27.3. The following creates only a local synthetic fixture; it does
not contact a provider, read credentials, or start a service. Future verified
native downloads will appear on the [GitHub Releases page](https://github.com/maldous/CircularQuay/releases). Consumer installation, verification, release-candidate, and publisher procedures belong in [RELEASE.md](RELEASE.md).

### Linux and macOS

```sh
umask 077
cd -P "$(mktemp -d)"
git clone --branch feature/initial-implementation https://github.com/maldous/CircularQuay.git
cd CircularQuay
go version # requires Go 1.27.3
go build -o ./cquay ./cmd/cquay

./cquay init --output cquay.json
./cquay --config cquay.json doctor
./cquay --config cquay.json plan
./cquay --config cquay.json run --workload demo -- /usr/bin/printf '%s\n' 'Synthetic workload completed.'
```

### Windows PowerShell

```powershell
git clone --branch feature/initial-implementation https://github.com/maldous/CircularQuay.git
Set-Location CircularQuay
go version # requires Go 1.27.3
go build -o .\cquay.exe .\cmd\cquay

.\cquay.exe init --output cquay.json
.\cquay.exe --config cquay.json doctor
.\cquay.exe --config cquay.json plan
.\cquay.exe --config cquay.json run --workload demo -- C:\Windows\System32\cmd.exe /d /c 'echo Synthetic workload completed.'
```

The final command runs an explicitly named, trusted operating-system program
with a synthetic value. Do not use environment-dumping commands when a policy
contains credentials. A declared workload command has access to values that
policy delivers to it.

For policy editing, conditional updates, doctor behavior, and all command
flags, see the [v1 command reference](#v1-command-reference) below.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/architecture-overview-dark.svg">
  <img alt="Explicit provider mappings pass through CircularQuay policy into native workloads, approved snapshots, and individually authorized MCP disclosure." src="docs/assets/architecture-overview-light.svg">
</picture>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/environment-evolution-dark.svg">
  <img alt="Overlapping developer requirements evolved from shell profiles through dotenv, CI and containers, managed identities, and multi-platform AI workloads." src="docs/assets/environment-evolution-light.svg">
</picture>

## What it delivers

- **Explicit interoperability:** select exact provider resources and string fields; keep existing stores authoritative.
- **Workload isolation:** reject duplicate names and undeclared resource access; keep provider bootstrap credentials out of application environments.
- **Complete generations:** resolve required inputs before activation instead of applying a partial refresh.
- **Portable encryption:** use upstream SOPS and age rather than a new encryption format.
- **Native delivery:** use process environments, protected files, OS credential interfaces, and declared deployment targets.
- **Optional MCP:** expose authorized status and named actions, with a separate permission for each raw secret disclosure.
- **Inspectable assurance:** source, tests, a policy model, release identities, and a reproducible audit route.

This is an early implementation. A compiled adapter is not proof of live-provider compatibility, a passing unit test is not a reboot test, and a clean scanner is not a security certification. The [qualification matrix](docs/qualification.md) separates implementation from operational evidence.

## Choose an availability policy

| Mode | Intended use | Boundary |
| --- | --- | --- |
| Offline | Provisioned static values from local, approved artifacts | No central per-read audit or immediate remote revocation. |
| Online | Fresh provider retrieval or unwrap at each launch | Startup depends on the provider; auditing depends on the actual backend configuration. |
| Hybrid | Online retrieval with an explicitly approved encrypted fallback | Only qualifying availability failures permit fallback. Denial, TLS failure, malformed data, and expiry do not. |

A deployed snapshot is a delivery workflow for an offline or hybrid workload,
not a fourth availability mode. One-way publication into a platform's native
store gives the destination's permissions, rotation, and audit behavior to the
copy.

To use online, hybrid, snapshot, or publication operations, start with the
[complete Linux reference](examples/linux-deployment.json) and the
[configuration reference](docs/configuration.md). Copy it to an operator-owned
path and replace only its fabricated resource selectors and protected file
paths; keep actual credentials in the selected provider or protected identity
file. After that policy and its prerequisites are valid, use its declared
workload ID with the normal lifecycle commands:

```sh
CONFIG=/absolute/operator-owned/cquay.json
cquay --config "$CONFIG" plan
cquay --config "$CONFIG" run --workload reference-api -- /absolute/path/to/your/workload
cquay --config "$CONFIG" snapshot --workload reference-api
cquay --config "$CONFIG" sync --once --workload reference-api
```

The reference itself names nonexistent synthetic paths and cannot be run until
an operator supplies those external prerequisites. `snapshot` needs a completed
offline or hybrid snapshot declaration; `sync` needs at least one declared
target and only publishes values explicitly permitted by each binding.

Already delivered values cannot be recalled from process memory, old ciphertext, or an authorized client. Removing a recipient is not a substitute for rotating a compromised credential at its issuer.

## Try a credential-free workload

The `examples/` directory contains fabricated values only. From the repository root, with a built `cquay` executable:

```sh
cquay --config examples/demo.json plan
cquay --config examples/demo.json run --workload demo -- /usr/bin/env
```

The second command deliberately demonstrates the child's environment using **synthetic data**. Do not use environment-dumping commands with real credentials. Normal workload commands should consume their assigned values without printing them.

On Windows, use an absolute native executable rather than assuming Bash exists. The CLI does not implicitly evaluate a shell command string. A program named by a workload still has the authority to read values delivered to it.

Native archives and verified offline bundles are the intended consumer installation route. Only enabled features need their dependencies. The normal CLI does not require an installed Go, Python, Node, Java, or Docker runtime. Source builds require the pinned Go toolchain; the independent audit environment has additional prerequisites.

## Provider and platform boundaries

SOPS can use cloud KMS or OpenBao Transit to protect a document key. That is different from retrieving existing application values from a secret manager.

| Integration | Scope |
| --- | --- |
| JSON / literal dotenv / selected process variables | Explicit local configuration; no interpolation or shell-profile execution. |
| SOPS / age / approved Transit endpoint | Upstream encryption and scoped key release. |
| OpenBao / Vault KV | Exact mount, path, field, and supported version. |
| AWS SM/SSM, Azure Key Vault secrets, GCP Secret Manager | Native provider identities and explicit resource references. |
| Doppler, Infisical, Bitwarden, Akeyless, optional 1Password | Connector-specific identity, lifecycle, audit, dependency, and platform limits. |
| systemd / Windows / Termux / containers | Native startup and credential boundaries; actual devices require qualification. |
| Windmill / Workers / Netlify / cloud applications | Native one-way delivery where implemented; no inherited source-audit guarantee. |

Use `cquay providers` and the qualification matrix for the current executable's capabilities. Unsupported combinations must fail explicitly. Provider and platform support does not imply a free subscription, official Android support, or an upstream security endorsement.

Termux startup occurs after Android permits access following the first unlock. Multiple Termux processes share an application UID. Windows DPAPI machine protection relies on ACLs for service separation. Root or Administrator compromise defeats endpoint protections.

## MCP and AI agents

CircularQuay's MCP interface is optional. Local stdio uses the OS-user/launcher boundary. Separately configured HTTP requires a qualified OAuth issuer, the correct resource audience, and current authorization on each operation.

- List/status results are filtered by current grants.
- Named actions do not accept arbitrary commands or provider URLs.
- `read_secret` requires a separate grant for an exact configured key.
- Bootstrap keys, refresh credentials, and server identities must not be exposed as readable aliases.
- Raw action stdout/stderr is not an authorized substitute for a secret-read permission.

**An authorized raw read deliberately releases plaintext to the MCP client.** It can enter model context, transcripts, or the client's retention systems. CircularQuay cannot control that value after delivery. Grants prevent future toolkit access; they cannot erase previous disclosures.

Self-reported agent names are not identity. Strong separation between agents sharing one OS user requires an additional OS or authenticated-service boundary. A coding agent running caller-controlled code with injected credentials has disclosure-capable authority.

## Design and evidence

- [Configuration reference](docs/configuration.md) and [complete Linux deployment example](examples/linux-deployment.json)
- [Platform and architecture matrix](docs/platform-support.md)
- [Credential and native-store boundaries](docs/credential-boundaries.md)
- [Hosted identity and application connections](docs/hosted-identity.md)
- [Architecture and trust boundaries](docs/architecture.md)
- [Security model](docs/threat-model.md)
- [Assurance contract](docs/assurance.md)
- [Qualification matrix](docs/qualification.md)
- [Provider interoperability](docs/providers.md)
- [Security reporting](SECURITY.md)

The target is clean first-party analysis, meaningful coverage, and reproducible evidence. The receipt badges above link to exact bounded results; they do not represent a release gate, a complete audit, or current-head qualification. Published metrics must identify the commit, tools, rules, platform scope, and remaining upstream advisories.

## Related work

[SOPS](https://getsops.io/) supplies encryption, [age](https://age-encryption.org/) supplies portable recipients, and [OpenBao](https://openbao.org/) supplies established authorization and key-management capabilities. [Teller](https://github.com/tellerops/teller), [vals](https://github.com/helmfile/vals), and [Chamber](https://github.com/segmentio/chamber) already make provider retrieval and environment injection easier. [sops-nix](https://github.com/Mic92/sops-nix) and [Flux](https://fluxcd.io/flux/guides/mozilla-sops/) demonstrate native lifecycle integration.

CircularQuay focuses on their operational intersection: explicit inheritance and export policy, lifecycle-aware offline copies, native service delivery, optional authenticated disclosure, and evidence-backed recovery. It does not claim to invent provider interoperability or to replace those projects.

## License

Apache-2.0 for CircularQuay's own code and documentation. Dependencies retain their own licenses and notices.

## v1 command reference

This is the intended v1 interface for review. The documentation preview notice
above applies to commands whose implementation is being qualified on this
working branch.

All commands accept `--config PATH` (default `cquay.json`) and `--state-dir
PATH` before the command. `help`, `--help`, `-h`, `version`, and `providers`
need no policy. `version` identifies this working build as `1.0.0-dev`; it is
not a release assertion.

| Command | Usage and behavior |
| --- | --- |
| Create | `cquay init [--output PATH]` or `cquay config init [--output PATH]` creates a protected synthetic JSON policy and fixture, refusing to overwrite either. |
| Inspect configuration | `cquay config validate`; `cquay config get --pointer RFC6901`; `cquay config list [--pointer RFC6901]`. `get` and `list` return the whole-document SHA-256. The returned policy literal is visible to an authorized configuration reader. |
| Edit configuration | `cquay config set --pointer RFC6901 (--value-json JSON\|--value-file PATH\|-) [--expect-sha256 HEX]`; `cquay config delete --pointer RFC6901 [--expect-sha256 HEX]`; `cquay config replace --file PATH\|- [--expect-sha256 HEX]`. Values are JSON; `-` reads standard input; inputs are capped at 1 MiB. |
| Doctor | `cquay doctor [--fix\|--identity-only]`. A missing or unsafe selected policy is non-zero. `--identity-only` reports the local kernel-derived subject without checking policy. `--fix` canonically formats a valid protected JSON policy only; it does not alter grants, provider settings, services, or deployment state. |
| Plan and status | `cquay plan`; `cquay status --workload ID [--json]`. |
| Workload lifecycle | `cquay run --workload ID [--target ID] [-- ABSOLUTE-PROGRAM ARGS…]`; `cquay snapshot --workload ID [--target ID]`; `cquay apply --workload ID --target ID`; `cquay sync --workload ID [--target ID] [--once]`. `apply` requires an explicit target; only `run` accepts arguments after `--`. |
| MCP | `cquay mcp serve [--transport stdio\|http] [--tls-cert PATH --tls-key PATH]`. HTTP uses the policy listener address (or loopback `127.0.0.1:8080`); a non-loopback listener requires both TLS paths. |

Configuration operations are cooperative: each writer takes the private local
configuration lock, validates a candidate, then re-reads the current digest
before atomically replacing the file. `--expect-sha256` adds an optimistic
review check; it is not an external distributed CAS. Configuration editing and
`doctor --fix` accept protected JSON only. YAML may be loaded for normal policy
execution but is intentionally not mutated. A repeated `set` or `replace` with
the same value produces the same canonical document; a repeated `delete` fails
after the node is absent.

The separate assurance binary is for evidence generation, not workload
delivery: `cquay-audit run --profile source|enterprise --output DIR` with
optional `--source`, `--go`, `--sonar-url`, `--sonar-token-file`, and
`--sonar-project-key`. Its enterprise inputs and evidence handling are
documented in [RELEASE.md](RELEASE.md).

### Configuration CRUD

These examples edit public configuration declarations. They never supply a
provider token or private key. Start with the synthetic policy from the quick
start; `set` creates a missing object member or replaces an existing node.

```sh
# Create a synthetic policy and validate it.
cquay config init --output cquay.json
cquay --config cquay.json config validate

# List workloads and read one declaration. Both responses include sha256.
cquay --config cquay.json config list --pointer /workloads
cquay --config cquay.json config get --pointer /workloads/demo

# Create or update a complete non-secret environment object.
cquay --config cquay.json config set --pointer /workloads/demo/environment --value-json '{"APP_MODE":"development"}'

# Read a replacement node from stdin or a protected JSON file.
printf '%s\n' '{"APP_MODE":"test"}' | cquay --config cquay.json config set --pointer /workloads/demo/environment --value-file -
cquay --config cquay.json config set --pointer /workloads/demo/environment --value-file environment.json

# Delete that optional object; invalid or referenced-object deletions fail.
cquay --config cquay.json config delete --pointer /workloads/demo/environment

# Replace the complete policy with a validated, operator-prepared document.
cquay --config cquay.json config replace --file replacement-policy.json
```

Add `--expect-sha256` with the digest returned by the preceding `get` or `list`
to reject an edit based on an out-of-date document. For example, PowerShell can
read the JSON response without another utility:

```powershell
$review = .\cquay.exe --config cquay.json config get --pointer /workloads/demo | ConvertFrom-Json
.\cquay.exe --config cquay.json config set --pointer /workloads/demo/environment --value-json '{"APP_MODE":"development"}' --expect-sha256 $review.sha256
```

An empty pointer selects the complete document; `~0` escapes `~` and `~1`
escapes `/` in a member name. Array elements use numeric indices. Replace a
complete array to add entries. Parents must exist, and the resulting complete
policy must remain valid. Errors do not replace the previous policy.

### Diagnostics, planning and execution

```sh
cquay help
cquay --help
cquay version
cquay providers
cquay --config cquay.json doctor
cquay --config cquay.json doctor --fix
cquay doctor --identity-only
cquay --config cquay.json plan
cquay --config cquay.json --state-dir /absolute/private/state status --workload demo --json
cquay --config cquay.json run --workload demo -- /usr/bin/printf '%s\n' 'Synthetic workload completed.'
```

`doctor --fix` only formats valid protected JSON; it does not choose identities
or change permissions and grants. `doctor --identity-only` reports identity,
not workload readiness. A configured command can also be run without an
override: `cquay --config deployed-policy.json run --workload api`.

### Snapshots and native delivery

Use these commands with a fully provisioned policy such as the Linux reference,
with eligible bindings and declared snapshots/targets. They write configured
destinations; native activation occurs only when explicitly declared.

```sh
cquay --config deployed-policy.json snapshot --workload reference-api
cquay --config deployed-policy.json apply --workload reference-api --target systemd-credential
cquay --config deployed-policy.json sync --workload reference-api --once
cquay --config deployed-policy.json sync --workload reference-api --target systemd-credential --once
```

### Local and hosted MCP

```sh
cquay --config local-mcp.json mcp serve --transport stdio
cquay --config hosted-mcp.json mcp serve --transport http
cquay --config hosted-mcp.json mcp serve --transport http --tls-cert /absolute/tls/server.crt --tls-key /absolute/private/server.key
```

Hosted HTTP requires the exact configured issuer, resource, client and grants.
A listener is not an IdP or a managed multi-tenant service. The
[hosted identity boundary](docs/hosted-identity.md) describes that distinction.

### Independent assurance

```sh
cquay-audit run --profile source --source . --output audit-results/source --go /absolute/path/to/go
cquay-audit run --profile enterprise --source . --output audit-results/enterprise --go /absolute/path/to/go --sonar-url http://127.0.0.1:9000 --sonar-token-file /absolute/private/scanner.token --sonar-project-key circularquay
cquay-audit sonar-init --directory .local/sonar
cquay-audit sonar-bootstrap --directory .local/sonar --url http://127.0.0.1:9000 --project circularquay
```

The Sonar commands are for a new isolated audit instance. Follow the prerequisite,
bootstrap and evidence-custody instructions in [RELEASE.md](RELEASE.md); do not
point them at an operational instance. A failing or unexecuted gate remains a
failing or unexecuted gate.
