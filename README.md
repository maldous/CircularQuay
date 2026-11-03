# CircularQuay

**Portable secrets delivery, with explicit trust boundaries.**

> **v1 documentation preview:** this working branch presents the intended
> CircularQuay v1 experience. Implementation and production qualification are
> incomplete. Use synthetic data only; see the [qualification evidence](docs/qualification.md).

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

### Intended v1 command reference

The interface below is the v1 documentation target and is awaiting its
implementation-branch change. It is included for review, not as a claim that
the current public branch already provides it. The default policy name will be
`cquay.json`.

```text
cquay config init
cquay config validate
cquay config get --pointer /workloads/demo
cquay config list --pointer /workloads
cquay config set --pointer /workloads/demo --value-file -
cquay config delete --pointer /grants/0 --expect-sha256 <current-document-digest>
cquay config replace --file policy.json --expect-sha256 <current-document-digest>
cquay doctor [--fix]
```

Pointers use RFC 6901. `set` replaces one complete JSON node atomically; use
`--value-file -` to pass JSON on standard input instead of placing
credential-looking literals on a command line. `replace` accepts one complete
JSON policy. These operations validate the resulting policy before writing and
do not retrieve provider values. Configuration literals remain visible to an
authorized configuration reader; put credentials in the declared providers.
The optional digest makes
an edit conditional on the document a caller reviewed. `doctor --fix` is
limited to safe local formatting corrections; it does not repair provider,
identity, service, or deployment configuration.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/architecture-overview-dark.svg">
  <img alt="Explicit provider mappings pass through CircularQuay policy into native workloads, approved snapshots, and individually authorized MCP disclosure." src="docs/assets/architecture-overview-light.svg">
</picture>

A shell profile once made a developer's environment feel simple: open a terminal, inherit the right variables, and get on with the job. Services, containers, Windows sessions, CI runners, and hosted applications introduced different startup and identity boundaries. Multiple AI providers and agent workflows added more keys, scopes, and places to configure them.

CircularQuay grew out of the frustration of joining those pieces together repeatedly. It connects existing providers, established encryption, and native runtime delivery through one workload-scoped policy. You choose where online authorization is required and where an encrypted offline copy is appropriate.

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
