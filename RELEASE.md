# Maintainer release and operator handoff

This is the maintainer/operator entry point. It covers public distribution, account setup, independent installation/auditing, and every post-installation enrollment step. The product README covers functionality and user-facing usage; publisher credentials and registry procedures belong here.

All commands use a chosen immutable version. `v1.0.0` below is an example until a release with that tag is published. Do not interpret a manifest, example URL, generated report, or successful build as evidence of publication or operational qualification.

## 1. Identity, versions and evidence

| Item | Canonical identity |
| --- | --- |
| Repository/module | `github.com/maldous/CircularQuay` |
| Runtime command | `github.com/maldous/CircularQuay/cmd/cquay` |
| Audit command | `github.com/maldous/CircularQuay/cmd/cquay-audit` |
| Go SDK package | `github.com/maldous/CircularQuay/sdk/go` (root module) |
| TypeScript SDK | npm `@maldous/cquay-sdk` |
| Python SDK | PyPI `cquay-sdk`, import `cquay` |
| Native executables | `cquay`, `cquay-audit` (`.exe` on Windows) |
| Public OCI package | `ghcr.io/maldous/circularquay` |
| Official MCP Registry name | `io.github.maldous/circularquay` |

Record application/tag, policy schema, runtime MCP protocol, Registry descriptor schema, SDK, native artifacts, OCI digest, MCP bundles, skill/plugin package and audit-profile versions separately. A Registry schema date is not an MCP protocol version.

CircularQuay is the project name. The app, executable, release asset prefix and operations skill are `cquay`, pronounced “see-key”; the separate audit executable is `cquay-audit`. The initial source implementation uses Go 1.27.3. The researched SonarQube Community reference is 26.10.0.133078; qualify the actual pinned server/scanner/analyzer/rules on release. The runtime MCP design targets protocol 2026-07-28 and a qualified official SDK, independently from Registry metadata schema.

### Release and qualification records

Record source revision/tree, platform, tool/runtime versions, tested scope,
artifact digests, registry/client identity, outcomes and independent readback in
[qualification](docs/qualification.md) and the release evidence bundle. The
[session handoff](docs/session-handoff.md) retains in-progress engineering
receipts. This operator guide defines the complete release process, not a live
publication ledger. A command example or manifest is not a published package.

Each channel has its own recorded state: not attempted, account required,
validated locally, published, independently verified or awaiting external
approval. Preserve failures and permission/entitlement limits without inventing
an approval, namespace, CAPTCHA or unavailable feature.

### Security-profile and administration handoff

Record the selected [security profile](docs/security-profiles.md), immutable
cryptographic-profile definitions, enrolled trust roots/issuers, approved
signers and thresholds, native custody/provider capabilities, audit destinations,
revocation/freshness bounds and recovery owners. Verify the effective profile
and deny unsupported required capabilities before enrolling real workloads.
The [administration controls](docs/administration-security.md) and
[management lifecycle](docs/management-and-discovery.md) are part of the same
release acceptance, including signed changes, independent approvals, monotonic
state and downgrade-negative evidence across all interfaces. Operator changes
use typed versioned contracts and protected references, not undocumented flags
or model-visible bootstrap credentials. Retain the exact installed-byte,
deployment, backup/restore and independent review receipts; no profile name
establishes certification or replaces missing qualification.

### Version and branch policy

Use Semantic Versioning and Gitflow. `feature/<purpose>` and `hotfix/<purpose>` describe work; `develop` integrates reviewed changes; `release/<major.minor.patch>` stabilizes a selected release; `main` contains qualified stable releases. Preserve merge commits and back-merge release/hotfix fixes into `develop`. Do not use direct feature pushes to `main` or rewrite published history.

Every release has an immutable tag and a GitHub Release. Use `v1.0.0-rc.1`, `v1.0.0-rc.2`, then `v1.0.0` as the naming pattern. Candidate tags belong to the reviewed `release/1.0.0` branch, are marked **Pre-release**, and never become **Latest**. Stable tags belong to the reviewed stable merge on `main`; mark the selected stable release **Latest** only after verification. Never move an RC tag to a corrected commit or rename an RC into a stable release. Build the stable version from its own exact tag and record the comparison with the candidate.

An RC is a supported feedback stage for its explicitly listed test matrix, not a waiver of security or integrity gates. Complete the applicable source/security/artifact gates before any public tag, including an RC. Record wider native/provider acceptance still being collected. Stable promotion additionally requires the complete declared qualification matrix and independent review. Only the selected candidate's complete evidence can authorize promotion.

Use GitHub milestones for the candidate and final acceptance work. Link issues and reviewed PRs to the milestone; retain unresolved limitations in release notes. Close the milestone only when its stated acceptance is met, and use a new corrective version for post-release defects.

### Required GitHub Release contents

Draft the release first, attach and inspect every asset, then publish once. Immutability prevents repairing the published assets in place. GitHub's generated “Source code” archives are source downloads, not compiled installers. Follow the [GitHub release procedure](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository).

Each release must provide:

- A title such as `cquay v1.0.0-rc.1`, maturity label, exact source commit, release date, changelog and links to reviewed changes.
- A platform/architecture table with direct links to each qualified asset, its size, verification instructions and required external tools. Never label a cross-build as a native acceptance pass.
- `cquay_<version>_<os>_<arch>.tar.gz` for Linux/macOS and `.zip` for Windows. Each archive includes `cquay`, `cquay-audit`, the `cquay` skill, license, release/install instructions, security/support guidance and qualification matrix.
- `cquay_<version>_checksums.txt`, authenticated provenance and a complete SBOM including shipped external executables. Publish offline verification inputs when supported. A checksum beside a download is not independent publisher authentication.
- Source archive, exact toolchain/build inputs and audit evidence for the release commit. Record artifact digests and reproducibility scope.
- Installation, upgrade, removal, migration and recovery directions; known limitations; compatibility changes; security-advisory links; and the community/private-support routes.
- Separate publication receipts for Go module indexing, GHCR, MCP Registry/MCPB and any qualified package-manager listing. Do not advertise a channel merely because its manifest exists.
- Official SDK versions, supported language/native/MCP compatibility, direct registry links, package digests, SBOMs, provenance and independent installation receipts.

The release page is `https://github.com/maldous/CircularQuay/releases/tag/<tag>` and asset URLs follow `https://github.com/maldous/CircularQuay/releases/download/<tag>/<asset>`. Replace every placeholder with an existing verified asset. The [Releases index](https://github.com/maldous/CircularQuay/releases) is the entry point until releases exist; do not present example `v1.0.0` links as current downloads.

### Download and install a qualified native archive

These are operator instructions for a published qualified version; no such release is claimed yet. Choose a specific tag, OS and architecture from its release page. Browse the explicit RC release page when evaluating a candidate; do not use `/releases/latest` for RC selection.

Where a compatible GitHub CLI is installed, an example Linux/amd64 route is:

```sh
gh release verify v1.0.0 --repo maldous/CircularQuay
gh release download v1.0.0 --repo maldous/CircularQuay --pattern cquay_1.0.0_linux_amd64.tar.gz --pattern cquay_1.0.0_checksums.txt
gh release verify-asset v1.0.0 ./cquay_1.0.0_linux_amd64.tar.gz --repo maldous/CircularQuay
sha256sum --ignore-missing --check cquay_1.0.0_checksums.txt
mkdir cquay-1.0.0
tar -xzf cquay_1.0.0_linux_amd64.tar.gz -C cquay-1.0.0
```

Check archive entries before extraction; reject absolute/traversal paths and unexpected executables. The maintainer must verify these exact CLI commands against the qualified GitHub CLI version; source archives generated on demand are not the same as verified release assets. The [official verification documentation](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity) describes the trust route. For browser/offline downloads, use the release's authenticated verification bundle and platform hash tools; do not execute an unauthenticated installer.

For an operator-owned Unix per-user installation after verification, copy the binaries into a chosen user executable directory with mode `0755`, keep the documentation with the versioned archive, and explicitly add the chosen directory to PATH if necessary. `~/.local/bin` is a conventional choice. Use `shasum -a 256` on macOS when `sha256sum` is absent; stable macOS distribution additionally needs its stated signing/notarization qualification. Do not bypass Gatekeeper to manufacture a clean install result.

On Windows, verify the selected archive using the release's documented trust route and `Get-FileHash -Algorithm SHA256`, then use `Expand-Archive` into a versioned per-user directory. Run `cquay.exe version` and `cquay.exe doctor` there, or explicitly add that directory to the user PATH. The native core does not require Go, Bash, Node or Python. Verify the stated Authenticode/signing support; native ACL/DPAPI installation is a separate acceptance test.

Run `cquay version`, `cquay doctor` and a synthetic `plan` before enrollment. Feature dependencies such as SOPS and Git must be separately authenticated and installed only for the enabled feature. Copy/installing the binary does not configure a system service, import credentials or enroll a provider. Follow post-installation enrollment below for each of those explicit actions.

For upgrades, review release notes and schema compatibility, retain an independently restorable authorized backup, verify the new artifact and replace the selected executable atomically while following the workload's declared lifecycle procedure. Do not reactivate revoked or expired values during rollback. To remove a per-user installation, stop only services/schedules explicitly enrolled for it, remove its PATH entry and selected binaries, and explicitly revoke temporary grants before deciding which protected state/backups must be retained. Never indiscriminately delete shared credentials or unrelated service paths.

### Packaging toolchain and staging

The initial packaging target is GoReleaser OSS **2.19.0** with Go **1.27.3**, driven by `.goreleaser.yaml`. It produces native archives and a draft release; it does not establish signatures, SBOMs, native installer qualification or registry acceptance. Schema validation is narrower than an artifact build. Keep the build matrix labeled as a build matrix until native tests qualify each target.

Run `goreleaser check`, then `goreleaser release --snapshot --clean --parallelism 1` in an isolated checkout to inspect local development archives without creating a public tag or release. Do not use `--skip` to bypass a required qualification gate. For a qualified selected public tag, stage a draft, generate/verify required SBOMs and provenance, attach all evidence and install directions, test anonymous asset downloads in a clean environment, and only then publish the immutable release. RCs and stable releases use the same integrity requirements.

Native archives, Homebrew, Winget, Scoop, Linux packages and app stores are distinct distribution channels, each requiring its own maintained manifest, ownership/trust route, install/upgrade/uninstall tests and publication receipt. OCI/MCP/Go procedures below remain separately gated. Do not create paid signing accounts or broad long-lived credentials merely to add a badge or unqualified distribution route.

### Optional managed-key service enrollment

Use the standalone [managed-service contract](docs/managed-key-service.md) for
identity/key mappings, administration, custody limits, and the separate financial
model. Prefer already available paid Netlify/Cloudflare identity and management
interfaces; verify plan entitlements, seats, credits, and retention before
choosing a deployment. Build only missing key-specific screens and host their
UI/API within those environments. Deployment and preview identities are separate;
previews contain synthetic keys only. Platform-account roles, site protection,
application identity, mapping/grant administration, and key reads are separate
permissions. Preserve server-side checks and redacted audit for every route.

Enroll exact trusted issuers/audiences, clients, stable subjects/domains, key
versions, grant windows, refresh/custody policy, and independently held recovery
recipients. Back up keys, mappings, policy revisions, and revocations together;
restoring old grants must not undo a deny. Test key release, denied-domain access,
revocation, IdP freshness, audit failure, retention, and isolated restore before
operator acceptance. A mapped private key is intentionally disclosed to its
client: revoking the mapping stops future fetches but does not invalidate copied
keys or old ciphertext. Capacity, hosting costs, paid plans, release gates, and
native/hardware acceptance require their own actual receipts; no purchase or
production deployment follows from documentation.

### Managed-service validation and promotion

Use the [managed-service development contract](docs/managed-key-service.md#local-development-integration-and-controlled-delivery)
for local, CI, preview and production instances. Record the service artifact
digest, configuration schema, migration sequence and fixture versions together.
The developer instance uses loopback listeners, private disposable state,
synthetic tenants and keys, and real verified authentication. Trust its test CA
only in the selected clients; never alter the host trust store or disable TLS
verification. Local commands and CI execute the same integration suite in
[integration testing](docs/integration-testing.md).

Qualify the selected release candidate with authentication and tenant-isolation
tests, scope/window/expiry and revocation tests, concurrent-update conflicts,
migration and rollback rehearsals, independently restorable encrypted backups,
health checks and durable redacted audit. Keep emulator, live-provider and
hosting-preview receipts separate. Preview identities and storage cannot access
production keys or mappings. Verify each platform's actual entitlement before
using its managed services; local emulation does not establish that entitlement.

Promote only the exact qualified artifact digest through the approved release
environment. Changed build inputs, version bytes, migrations or packaging create
a new candidate and repeat the applicable gates. Retain deployment, migration,
rollback and post-promotion health receipts. A staged workflow or design document
is not a running service or a production acceptance result; consult the
[managed-service qualification record](docs/qualification.md#managed-service-design-status).

## 2. Prerequisites and accounts

### Separate four environments

| Environment | Requirements |
| --- | --- |
| Consumer runtime | Qualified native artifact, protected writable state/configuration and dependencies for enabled features. No language runtime required for the core CLI. |
| Source build | Pinned Go toolchain, module inputs and native toolchain dependencies for the selected build. |
| Independent audit | Pinned analyzer/model/test tools, sufficient isolated resources, and optional local SonarQube environment. |
| Publisher/connected tests | Namespace ownership or dedicated synthetic provider account; never ordinary runtime administrator credentials. |

Basic offline installation and synthetic local auditing require no external secret-provider account. OpenBao and SonarQube can run on the operator's own infrastructure without a SaaS subscription. Do not modify an unrelated operational host to satisfy an audit kernel setting or resource requirement; use an isolated machine with the documented prerequisites.

### GitHub and publishing

An independent maintainer can create a free account at <https://github.com/signup>, verify email, configure MFA/passkey and recovery custody, and create a public repository. A fork published under another owner must change module, OCI, MCP and signing identities consistently; it must not impersonate `maldous`.

No separate Go-index account is needed. GitHub-based MCP publishing does not require a paid Registry account or purchased domain. Public native downloads and public GHCR pulls do not require a consumer publishing account.

Prefer repository-scoped ephemeral Actions credentials with only the job permissions needed: release content writes, package writes, keyless-signing OIDC and attestation writes. Avoid persistent PATs. Human browser authentication, repository download access, service credentials and provider decryption permissions remain separate.

### Confirmed free testing routes

| Provider | Account setup | Entitlement and identity constraints |
| --- | --- | --- |
| Doppler | [Developer signup](https://dashboard.doppler.com/register?plan=developer) | Select Developer, not a Team trial. Verify current quota; free activity history does not establish per-read audit entitlement. |
| Infisical | [Signup](https://app.infisical.com/signup) | Select Free, not Pro/Advanced trial. Human and machine identities share its allowance. |
| Bitwarden SM | [Official procedure](https://bitwarden.com/help/sign-up-for-secrets-manager/) | Create/use a Free organization; enable Secrets Manager through its subscription settings and verify the $0 selection. |
| Akeyless | [Registration](https://console.akeyless.io/registration) and [account quickstart](https://docs.akeyless.io/docs/account-quickstart) | Verify Secrets Management Free, not Password Manager limits. Email activation/profile setup may require a human. |

For every account, record purpose, owner action, official route, confirmed free tier/quota, least-privilege scope, verification, expiry and cleanup. Keep account identifiers and bootstrap values in protected local records, not this public file.

Create an isolated `cquay-audit` project with fabricated allowed/denied resources. Use separate scoped test identities. Verify retrieval, denial, rotation and revocation. Audit claims require actual native events and retention evidence. Akeyless API-key authentication is documented for temporary/POC use, not a production recommendation.

Stop if the free entitlement cannot be confirmed or billing/automatic conversion is required. Mark the connected test unqualified. Do not share a privileged identity to evade quotas. AWS/Azure/GCP and paid-provider live tests require a separately authorized existing account and budget; they are not core installation dependencies.

Detailed setup and cleanup: [provider runbook](docs/operators/providers.md).

## 3. Independent local installation and audit

### Obtain and authenticate the release

Choose the exact OS/architecture and qualified feature set. Authenticate the expected repository/workflow identity, then verify the artifact checksum. A checksum downloaded beside an untrusted binary is not independent publisher authentication.

Use a signed installer, manual archive/browser route, or verified offline bundle. Document the operator's bootstrap trust source. Never execute a raw `curl | sh` stream or disable TLS/certificate verification. A bundle's verifier is not automatically trustworthy merely because it was in the same bundle.

On platforms with GitHub CLI available, the qualified release should support the documented `gh release verify` / `gh release verify-asset` route. Where attestations are provided, verify the exact expected repository identity. Provide detached evidence/offline verification inputs where supported. Do not assume `gh`, Bash, PowerShell, Docker, a package manager or a compiler already exists on a consumer host.

### Build alternative

From a clean selected checkout with the pinned Go toolchain:

```sh
umask 077
go mod download
go mod verify
mkdir -p ../cquay-build
go build -trimpath -o ../cquay-build/cquay ./cmd/cquay
go build -trimpath -o ../cquay-build/cquay-audit ./cmd/cquay-audit
../cquay-build/cquay --config examples/demo.json plan
```

PowerShell uses `New-Item -ItemType Directory -Force ../cquay-build`, explicit `.exe` output names and `../cquay-build/cquay.exe`. Keep build products outside the source checkout so an audit can bind every source byte to its Git tree. Native installation does not require Go. A command named `go` must actually be the expected compiler; record `go version` rather than assuming the executable name proves a toolchain.

### Run synthetic assurance

From the repository root:

```sh
../cquay-build/cquay-audit run --profile source --tool-manifest /absolute/private/tools.json --output ../cquay-evidence/source
../cquay-build/cquay-audit run --profile enterprise --tool-manifest /absolute/private/tools.json --output ../cquay-evidence/enterprise
```

Supported audit inputs include required `--tool-manifest ABSOLUTE_PATH`, optional `--source DIR` and `--go ABSOLUTE_GO`, and explicit Sonar connection flags `--sonar-url`, `--sonar-token-file`, and `--sonar-project-key`. The protected manifest binds every selected executable and rules input; its `go` path must match `--go` when supplied. The token file must be protected and outside public source. Never place its value in arguments or logs.

Create the manifest in a private operator directory from already-selected,
immutable tool releases: record each absolute executable path, SHA-256, fixed
probe output digest, and any protected rules/snapshot digest. Protect the file
with owner-only access and keep it outside source and evidence paths. The
manifest identifies local bytes and probe output only; it does not authenticate
a publisher, prove a complete runtime or dependency closure, or make execution
atomic or sandboxed.

Signed-repository tests also need an explicitly declared native helper. Add
`"helpers": [{"id": "ssh-keygen", "path": "/absolute/private/ssh-keygen",
"sha256": "<64 lowercase hexadecimal characters>"}]` to the v1 manifest,
using the actual protected helper digest and the native `.exe` filename on
Windows. This category accepts only `ssh-keygen` and has no invented version
probe. The auditor prepares a private canonical copy; it never adds an
ambient system directory to the child PATH to satisfy a missing dependency.

Select the root of a clean, complete Git checkout. The v1 report binds its commit, tree, tracked modes/blob IDs and raw SHA-256 file manifest before and after target checks. Unexpected files, including ignored Go inputs and unrelated prior audit output, block execution. Only the exact selected protected output subtree is excluded; an output inside the checkout must be under `audit-results` or `.local/audit-results`. External evidence directories avoid mixing results from successive profiles. Git replacement objects and ambient global/system Git configuration are disabled. Source manifests detect ordinary drift; they do not provide an atomic snapshot, trusted analyzer provenance or an operating-system sandbox. Run selected source only within an appropriately isolated audit environment.

The source profile is not the enterprise profile. Enterprise requirements that are missing or not run must produce a non-pass result, not a green qualification. Raw tool stdout/stderr is not automatically public evidence.

The local Sonar route uses the versioned environment under `audit/`; follow its actual pinned bootstrap instructions and loopback-only bindings. Provision temporary audit credentials into private local state. Import the exact profile/gate, run tests/coverage first, wait for asynchronous analysis/gate completion, verify source inventory and coverage scope, and export sanitized receipts. Do not assume an A rating equals zero issues.

`cquay-audit sonar-init --directory ABSOLUTE_PRIVATE_DIRECTORY` creates new
protected bootstrap state and refuses existing state. Start only the selected
isolated Compose project using `audit/compose.yaml` and that directory's
`compose.env`; choose an unused loopback port and satisfy the pinned server's
kernel/memory prerequisites on the audit machine. After the server is ready,
`cquay-audit sonar-bootstrap --directory ABSOLUTE_PRIVATE_DIRECTORY --url http://127.0.0.1:PORT --project circularquay`
creates the project, gate, profile and dedicated Browse/Execute Analysis
principal. Its token stays in a protected file; no credential value is printed.
Bootstrap accepts `--token-validity-days 7..30`, default `7`, for the newly
created finite-lifetime token. It does not rotate an existing operator token.

The [hosted ephemeral Sonar workflow](.github/workflows/sonar.yml) creates a
fresh isolated runner-local instance, selects `30` days, verifies the scanner
archive against [its pinned lock](tools/sonar-scanner.json), and destroys the
instance after evidence collection. Prerequisite or integrity failures stop
execution; coverage failures still permit scanner/readback evidence, and any
actual warning, failed readback or failed quality gate fails the job. This
diagnostic producer does not replace the independent auditor's transitive
runtime binding. Retain redacted evidence separately from bootstrap state;
cleanup removes only the explicitly owned Compose project and its volumes.

### Branch-aware Sonar qualification

Select a Sonar deployment whose actual entitlement and analyzer versions cover
the release's languages and branch model. Retain overall and new-code coverage,
duplication, issue, hotspot and warning readbacks, with the exact source revision
and analysis ID. A successful upload or completed Compute Engine task is one
phase of qualification.

For a public open-source repository, an accepted
[SonarQube Cloud OSS organization](https://docs.sonarsource.com/sonarqube-cloud/administering-sonarcloud/managing-subscription/subscription-plans#oss-plan)
provides branch/PR analysis and custom quality profiles/gates. Verify the actual
OSS entitlement before configuring the project. The ordinary Free plan has a
different branch scope. Follow the
[GitHub onboarding procedure](https://docs.sonarsource.com/sonarqube-cloud/getting-started/github)
to bind only the selected repository. Keep human project/gate administration
separate from the analysis principal, grant that principal only the required
project analysis access, and retain a finite, revocable CI token in the selected
GitHub Actions secret. Credential values never belong in source, command
arguments, reports or support messages. Fork PRs receive no analysis credential.

Configure and read back a dedicated
[quality gate and profile](https://docs.sonarsource.com/sonarqube-cloud/standards/managing-quality-gates/introduction-to-quality-gates)
with the unchanged CircularQuay requirements: 100% owned coverage, zero
duplication, zero owned issues/hotspots/warnings/dispositions, and both complexity
ceilings of 10. Retain overall-code assertions alongside PR new-code checks.
Qualify every declared language version against the actual analyzer; scanner
runtime compatibility does not establish analyzed-language support.

The initial empty `develop` branch is the real PR base. Use a branch-capable
analysis route with that actual reference; retain both branch/SCM associations.
An ephemeral first Community Build analysis has no new-code period, and a
specific-analysis UUID denotes a server snapshot rather than a Git commit.
Absent required new-code metrics remain a failed gate. Use the revision-scoped
[qualification record](docs/qualification.md) for actual support and results;
this operator procedure does not establish account enrollment or a passing scan.

See [assurance contract](docs/assurance.md), the audit environment's own instructions, and generated report schema. Publish commit/tool versions, real results, hotspot review, upstream advisories and untested platforms. Do not include raw MCP responses or operational values in the report.

## 4. Post-installation enrollment

An installed binary is not an enrolled environment. Complete these operator actions explicitly:

1. Record platform/version, binary digest, feature dependencies and CA trust.
2. Establish operator-owned protected config/state directories and trust anchors.
3. Select/create a dedicated workload identity, separate from human administration.
4. Register exact source resources and optional destination/export permissions.
5. Provision referenced bootstrap material through protected storage; no inline secret configuration.
6. Validate bootstrap acyclicity, policy and allowed/denied synthetic access.
7. Seed approved static offline/hybrid material before testing disconnected startup.
8. Configure only the chosen native service/scheduler, with reviewed paths and permissions.
9. Configure native audit destinations, retention, backup, update ownership and recovery custody.
10. Enable optional MCP only after issuer/audience/client/disclosure policy is qualified.
11. Run reboot, outage, denial, rotation and independent restore tests.
12. Record remaining features as unqualified and remove temporary grants/resources when finished.
13. Select telemetry signals and protected collector trust/credential references;
    validate safe export, bounded outage behavior and the independent required
    audit path. Configure collector-side retention, access and alerting.
14. Choose update channel/feed or disable checks, and record any local minimum
    runtime requirement. Verify offline behavior and recovery commands before
    enabling that requirement. See [operational configuration](docs/operational-configuration.md).

Platform details: [installation](docs/operators/installation.md), [MCP](docs/operators/mcp.md), [recovery](docs/operators/recovery.md).

Linux credentials, Windows SID/ACL/DPAPI, Termux first-unlock/background behavior, container recreation, Windmill resource publication and cloud deployment each have different acceptance criteria. Do not claim one service inherits a human shell environment or that a workflow/configuration reload necessarily refreshes a cached credential.

No existing operational host is installed by this cloud build task. Operators follow these directions on their own infrastructure and provide installation feedback.

## 5. Build and publish native and Go channels

### Release preflight

- Select an exact reviewed commit and new semantic version.
- Confirm clean tracked source, locked dependencies and recognized license/notices.
- Run the complete required assurance suite and inspect scope/exclusions.
- Inspect archives, images, metadata, skill/plugin assets and generated docs for accidental private material.
- Rebuild unsigned artifacts in clean equivalent environments where reproducibility is claimed.
- Record version compatibility and non-atomic channel publication status.

Do not publish a tag merely to test whether the code works. A public Go tag may be indexed before native binary publication is complete.

### Native artifacts and OCI

Use the pinned GoReleaser configuration. Stage native archives, installers, offline audit bundles, checksums, SBOMs and provenance before publishing an immutable release. Verify anonymous downloads and inspect exact archive contents.

Publish the public image to `ghcr.io/maldous/circularquay`, linked to the source repository using `org.opencontainers.image.source`. Public packages are free under current GitHub terms; package visibility still needs verification. Prove an anonymous digest-pinned pull. Do not assume public repository visibility made the image public.

Keyless signing and release attestations should bind the expected repository/workflow. Signing, MCPB certificate trust and checksum verification are separate assurances. Document every required permission and remove temporary credentials after use.

### Go module indexing

There is no separate Go publisher account. Go publishes a public module through immutable semantic-version Git tags. Preserve exact casing in `github.com/maldous/CircularQuay`.

After the audited commit/tag and release assets are ready, trigger discovery:

```sh
GOPROXY=https://proxy.golang.org go list -m -json github.com/maldous/CircularQuay@v1.0.0
go install github.com/maldous/CircularQuay/cmd/cquay@v1.0.0
go install github.com/maldous/CircularQuay/cmd/cquay-audit@v1.0.0
```

PowerShell equivalent for discovery:

```powershell
$env:GOPROXY = 'https://proxy.golang.org'
go list -m -json github.com/maldous/CircularQuay@v1.0.0
```

Verify the installed commands from a clean module/cache location and inspect:

- `https://pkg.go.dev/github.com/maldous/CircularQuay@v1.0.0`
- `https://pkg.go.dev/github.com/maldous/CircularQuay/cmd/cquay@v1.0.0`

Use Go tooling rather than manually lowercasing or constructing escaped proxy paths. Private dependencies require `GOPRIVATE` before resolution; the operational ciphertext repository must not become a public Go module.

Never move/reuse a published tag. For a defect, publish a higher corrected version and a `retract` directive where appropriate. Retraction is not deletion, and cached source may remain accessible. Negative lookups before a tag exists can be cached; do not retag to fix propagation. Official reference: <https://go.dev/doc/modules/publishing>.

### Official SDK distribution

The three SDKs follow one v1 compatibility and conformance contract. Their
registry records are independent distribution channels; a successful native
release does not establish npm/PyPI ownership or publication. Use the
[SDK guide](docs/sdks.md) for consumer APIs and this section for publisher setup.

| Release | Git/root Go module | npm package version / channel | Python distribution version |
| --- | --- | --- | --- |
| Candidate | `v1.0.0-rc.1` | `1.0.0-rc.1` / `next` | `1.0.0rc1` |
| Stable | `v1.0.0` | `1.0.0` / `latest` | `1.0.0` |

These are naming examples, not published versions. Development metadata does
not constitute an RC. Set each package's version in reviewed source before
qualification. The Go SDK belongs to the root module: use the root `v1...`
tag, never `sdk/go/v1...` or a new nested module. Stable package metadata and
bytes differ from RC metadata and bytes. Build and qualify the stable tag as
its own immutable version; never relabel RC files or replace a published version.

#### Registry configuration after repository setup

The authorized registry owner completes these actions through each registry's
normal authenticated account interface. Record account custody, MFA/recovery,
package maintainers, protected release environment and expected repository
identity without recording credentials. Confirm real ownership of `maldous`
on GitHub, the npm `@maldous` scope, and the PyPI `cquay-sdk` project. Name
availability alone is not ownership. Resolve any conflict before changing
consumer instructions; never publish a misleading alternate namespace.

Configure separate GitHub release environments for npm, PyPI and TestPyPI.
Restrict allowed release refs, approvals and publisher identity. Register the
**actual reviewed publisher workflow filename** and exact environment name
with each registry after that workflow is installed. The SDK validation
workflow has read-only permissions and is not a publisher. Grant `id-token:
write` only to the isolated publishing job; source builds and PR tests have
no publication authority. Pin publishing actions/toolchains to reviewed immutable
versions and require the checked-out source to match the triggering release tag.

For npm, configure the package's trusted publisher for GitHub user `maldous`,
repository `CircularQuay`, the exact workflow filename and npm environment.
Use a supported hosted runner and qualified Node/npm versions (the documented
minimum is Node 22.14.0/npm 11.5.1). Prefer stage-only publishing with human MFA
approval; enable direct publication or dist-tag management only when the
approved workflow needs them. New publisher bindings expire if their first
publication is not completed within the registry's documented window. After
verification, disallow persistent publishing tokens. If initial namespace
creation needs a human registry action, complete it through that interface.
Do not create a broad token to bypass missing ownership. See
[npm trusted publishing](https://docs.npmjs.com/trusted-publishers/).

For PyPI, configure an existing or pending trusted publisher for project
`cquay-sdk`, GitHub owner `maldous`, repository `CircularQuay`, the reviewed
workflow filename and the protected PyPI environment. Configure TestPyPI
separately with its own environment/publisher. Use the official pinned PyPA
publishing action with OIDC and attestations enabled. It uploads previously
qualified wheel/sdist files; the publishing job does not rebuild them.
TestPyPI uses a distinct index and OIDC audience; a successful rehearsal does
not publish or promote a package to PyPI. See
[PyPI trusted publishing](https://docs.pypi.org/trusted-publishers/using-a-publisher/)
and [attestations](https://docs.pypi.org/attestations/).

No Go-index account or upload credential is needed. Publish the qualified
root-module tag through the existing Git release process and verify discovery
using the normal module proxy/checksum route. See
[Go module publishing](https://go.dev/doc/modules/publishing).

#### Build, inspect and publish the selected bytes

Use isolated, reproducible build environments with committed dependency locks,
exact toolchain versions and no provider credentials. Run native CLI/MCP
conformance, cross-language error/expiry/cancellation/redaction tests, all
owned-code quality/coverage/security gates and package-installation tests.
Preserve logs, source/tree identity, lock digests, artifact hashes, license
inventory, SBOM and provenance in the release receipt. Inspect archive entries
against the package allowlist; reject fixtures, caches, credentials, private
state, unexpected executables and undeclared lifecycle scripts.

For TypeScript, run locked installation, type checking, tests and build in
`sdk/typescript`, then inspect `npm pack --json` and its resulting tarball.
Install that exact tarball into a clean consumer and rerun conformance before
publication. Publish/stage the verified tarball, not an unreviewed directory.
Use `next` for a candidate and `latest` for a qualified stable version; channel
pointers never change the immutable version's bytes. Verify provenance binds
the expected repository, workflow, source commit and tarball digest.

For Python, build wheel and sdist from `sdk/python` using the release's locked
build inputs. Install the wheel in a clean virtual environment; separately
build a wheel from the sdist and verify its contents and behavior. Publish the
exact retained files with the trusted publisher. Verify both registry SHA-256
values, project URLs, supported Python metadata, license and attestations.
Dependency installation for TestPyPI must use explicit approved sources; do
not add an unrestricted extra index that permits dependency confusion.

After public registry readback, verify anonymous consumer routes for the chosen
version. The following stable examples are valid only once that exact version
has actually been published and independently verified:

```sh
go get github.com/maldous/CircularQuay/sdk/go@v1.0.0
npm install --save-exact @maldous/cquay-sdk@1.0.0
python -m pip install 'cquay-sdk==1.0.0'
```

Verify `pkg.go.dev/github.com/maldous/CircularQuay/sdk/go@v1.0.0`,
`npmjs.com/package/@maldous/cquay-sdk/v/1.0.0`, and
`pypi.org/project/cquay-sdk/1.0.0/` against the release receipt. Production
consumers use their approved package mirror/registry, dependency lock and
hash/provenance policy. Language packages do not silently fetch or install a
native executable. TypeScript/Python local adapters require a separately
authenticated native `cquay` artifact and its exact digest.

Treat optional SDK transports and their native dependency availability as
separate platform qualification. The recorded Intel macOS Python locks select
`cryptography 48.0.1`; the hosted audit reports PYSEC-2026-3552/3553/3554.
The [upstream changelog](https://github.com/pyca/cryptography/blob/main/CHANGELOG.rst)
records removal of Intel macOS wheels in 49.0.0, and the inspected fixed
[PyPI releases](https://pypi.org/project/cryptography/) provide macOS ARM wheels.
This is an unresolved wheel/security prerequisite. Retain Intel qualification,
binary-only installs, reviewed hashes and the audit gate. Do not permit source
archives, suppress an advisory, silently remove the platform or substitute an
unverified wheel to manufacture a release pass. Recheck actual upstream
artifacts and all affected locks before claiming that boundary resolved.

Record each registry as published and independently verified only after its
actual readback and clean-consumer test. A permission, namespace, entitlement
or registry failure leaves that channel unqualified. Remove temporary
publisher grants and rehearsal resources without deleting retained evidence.
Withdraw a faulty version through the ecosystem's deprecation/yank/retraction
route and issue a higher corrective version; cached copies cannot be recalled.

## 6. Publish MCP distribution

The official MCP Registry hosts metadata, not executable packages. It currently supports OCI/MCPB and other ecosystems but not a native Go package type. CircularQuay therefore uses its real native/OCI artifacts rather than a JavaScript/Python wrapper created solely for listing.

### OCI-first Registry route

1. Publish and independently verify the immutable public image.
2. Include exact ownership metadata `io.modelcontextprotocol.server.name="io.github.maldous/circularquay"`.
3. Generate `packaging/mcp/server.json` using the pinned official schema, independently from the runtime MCP protocol.
4. Set exact namespace/version/repository, OCI digest, stdio transport and arguments equivalent to `mcp serve --transport stdio`.
5. Include no operational endpoints, credentials, broad host mounts or undeclared grants.
6. Pin and authenticate the official `mcp-publisher` binary before use.

Documented commands:

```sh
mcp-publisher validate packaging/mcp/server.json
mcp-publisher login github
mcp-publisher publish packaging/mcp/server.json
```

Release automation uses `mcp-publisher login github-oidc` with narrow `id-token: write`. Registry selection belongs to login. Do not put a PAT into command arguments.

Verify the exact namespace/version through the official `/v0.1/servers` API, then test an external client's initialization, launch arguments, allowed access and denial. Namespace ownership is not runtime OAuth, endpoint ownership, a security review, or marketplace approval. Registry schemas/availability may change; retain the exact descriptor and publication receipt.

Only add a public `remotes` URL after hostname control, TLS, OAuth metadata/resource audience, user isolation and endpoint operation are qualified. Do not publish a private/local endpoint or placeholder. A private installation needs no Registry account.

Official procedures: <https://github.com/modelcontextprotocol/registry/blob/main/docs/modelcontextprotocol-io/quickstart.mdx> and <https://github.com/modelcontextprotocol/registry/blob/main/docs/reference/cli/commands.md>.

### Native MCPB

Use `server.type: binary` and the packaged native executable. The packing CLI's Node requirement belongs to the maintainer environment; the binary server does not require end-user Node/Python.

Validate, pack, inspect the explicit allowlist and test a fresh compatible client for each platform/architecture. Bundle authenticated feature dependencies or explicitly require them through `doctor`; `cquay` alone does not imply SOPS is installed.

Host immutable `.mcpb` assets on GitHub Releases. Registry references require the exact `fileSha256`. Independently verify that digest: Registry documentation says clients, not the Registry itself, enforce it.

Do not promise automatic architecture selection by every client. Keep the first listing OCI-only if native selection is unqualified. MCPB X.509/PKCS#7 signing is not Sigstore provenance; a self-signed signature is not automatically trusted. If a target requires unavailable paid trust, record that channel incomplete rather than bypassing client security.

## 7. Skills and client-specific plugins

The versioned Agent Skills package lives at `integrations/skills/cquay/`. Validate `SKILL.md` metadata, references and actual prerequisites, then test installation/reload/removal in each claimed client. The standard does not establish one universal public skills index. Tagged repository/archive distribution is a valid route; host-specific marketplaces need their own qualification.

Skill instructions cannot grant provider or MCP authority. Experimental `allowed-tools` metadata does not replace server authorization. No package should instruct an agent to solicit passwords, bypass scopes, dump environments or perform rotations by default.

The [agent integration guide](docs/agent-integrations.md) separates portable
skill source, UI metadata, client manifests, MCP transport and registry bundles.
Package every referenced skill resource and preserve the same identity/version
across client overlays. Verify native archive inclusion using the selected
GoReleaser file inventory; do not equate an archive glob with an installed plugin.
A client manifest declares only verified executable/resource paths and the
selected transport; it contains no operational credentials or invented App ID.

For any client plugin channel record its packaging schema, account/namespace requirements, supported interfaces, privacy disclosures, reviewer/sample-data route, positive/negative tests, update/removal flow and external eligibility.

Current OpenAI public-plugin guidelines restrict processing authentication secrets and unofficial pass-through connectors. Public acceptance of CircularQuay's full `read_secret` surface is not promised. A reduced wrapper is not automatically accepted either. Desktop-only submission has external eligibility requirements. Preserve direct/local MCP functionality, mark the public channel policy-gated, and do not hide capability to bypass review.

References: <https://agentskills.io/specification>, <https://developers.openai.com/plugins/plugin-guidelines>, <https://developers.openai.com/plugins/deploy/desktop-only>.

## 8. Withdrawal, rollback, incident and cleanup

Publication across native releases, Go proxies, OCI and Registry metadata is not atomic. Record each channel independently. Never replace published bytes under an existing version.

- Publish a corrective higher version and retain safe compatibility notes.
- Retract a bad Go version through a later module version.
- Deprecate/hide an MCP record through the documented lifecycle commands; this does not recall installed/cached artifacts.
- Registry logout removes local credentials, not necessarily server-side tokens; perform explicit revocation/expiry cleanup.
- Remove test grants/accounts and private temporary files while retaining required evidence.
- Do not roll back to revoked, expired or compromised configuration.

If a credential enters a public asset or module, revoke/rotate at the issuer first. Tag deletion, retraction, descriptor withdrawal, or Git history rewriting does not make an exposed credential safe. Follow [incident response](docs/operators/incident-response.md).

## 9. Release-document acceptance

A developer unfamiliar with this project must be able to use only the tagged source and this document to install locally, run synthetic assurance on their infrastructure, understand unqualified capabilities, enroll one isolated free provider test, exercise optional MCP authorization, identify/publish eligible channels, and upgrade/remove test state safely.

Every remaining step needs a state, owner action, prerequisite, official route, expected result, verification, cleanup and qualification consequence. Missing resources remain **not run**, not successful. Credentials, MFA recovery codes, private grants and raw secret responses never appear in this ledger or public review material.
