# Maintainer release and operator handoff

This is the maintainer/operator entry point. It covers public distribution, account setup, independent installation/auditing, and every post-installation enrollment step. The product README covers functionality and user-facing usage; publisher credentials and registry procedures belong here.

All commands use a chosen immutable version. `v0.1.0` below is an example until a release with that tag is published. Do not interpret a manifest, example URL, generated report, or successful build as evidence of publication or operational qualification.

## 1. Identity, versions and current state

| Item | Canonical identity |
| --- | --- |
| Repository/module | `github.com/maldous/CircularQuay` |
| Runtime command | `github.com/maldous/CircularQuay/cmd/cquay` |
| Audit command | `github.com/maldous/CircularQuay/cmd/cquay-audit` |
| Native executables | `cquay`, `cquay-audit` (`.exe` on Windows) |
| Public OCI package | `ghcr.io/maldous/circularquay` |
| Official MCP Registry name | `io.github.maldous/circularquay` |

Record application/tag, policy schema, runtime MCP protocol, Registry descriptor schema, SDK, native artifacts, OCI digest, MCP bundles, skill/plugin package and audit-profile versions separately. A Registry schema date is not an MCP protocol version.

The initial source implementation uses Go 1.27.3. The researched SonarQube Community reference is 26.10.0.133078; qualify the actual pinned server/scanner/analyzer/rules on release. The runtime MCP design targets protocol 2026-07-28 and a qualified official SDK, independently from Registry metadata schema.

### Initial work ledger

| Activity | Current recorded state | Remaining action |
| --- | --- | --- |
| GitHub repository | Public empty repository created by its owner; source publication in progress | Push only reviewed synthetic/code assets, then independently verify visibility and commit. |
| Native release / Go tag / pkg.go.dev | Not published by this implementation task | Complete gates and publish a new immutable version. |
| GHCR / MCP Registry / MCPB | Not published | Verify artifacts and metadata before publishing. |
| External test-provider accounts | Not attempted in this implementation task | Use the confirmed free routes below when running connected tests. |
| Local source/audit execution | Consult generated evidence | A check not run is not a pass. |
| Native reboot/device/provider qualification | Pending operator-owned infrastructure | Record exact versions, tests and sanitized receipts. |
| Public plugin directory | Policy/eligibility-gated | Do not promise acceptance of the raw-secret product. |

Update this ledger from evidence, not intention. Record **not attempted**, **account required**, **validated locally**, **published**, **independently verified**, **awaiting external approval**, **unavailable**, or **experimental**. Do not invent a CAPTCHA, entitlement failure or policy approval that was never observed.

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
go mod download
go mod verify
go build -trimpath -o ./dist/cquay ./cmd/cquay
go build -trimpath -o ./dist/cquay-audit ./cmd/cquay-audit
./dist/cquay --config examples/demo.json plan
```

PowerShell uses explicit `.exe` output names and `./dist/cquay.exe`. Native installation does not require Go. A command named `go` must actually be the expected compiler; record `go version` rather than assuming the executable name proves a toolchain.

### Run synthetic assurance

From the repository root:

```sh
./dist/cquay-audit run --profile source --output ./audit-results/source
./dist/cquay-audit run --profile enterprise --output ./audit-results/enterprise
```

Supported audit inputs include `--source DIR`, `--go ABSOLUTE_GO`, and explicit Sonar connection flags `--sonar-url`, `--sonar-token-file`, and `--sonar-project-key`. The token file must be protected and outside public source. Never place its value in arguments or logs.

The source profile is not the enterprise profile. Enterprise requirements that are missing or not run must produce a non-pass result, not a green qualification. Raw tool stdout/stderr is not automatically public evidence.

The local Sonar route uses the versioned environment under `audit/`; follow its actual pinned bootstrap instructions and loopback-only bindings. Provision temporary audit credentials into private local state. Import the exact profile/gate, run tests/coverage first, wait for asynchronous analysis/gate completion, verify source inventory and coverage scope, and export sanitized receipts. Do not assume an A rating equals zero issues.

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
GOPROXY=https://proxy.golang.org go list -m -json github.com/maldous/CircularQuay@v0.1.0
go install github.com/maldous/CircularQuay/cmd/cquay@v0.1.0
go install github.com/maldous/CircularQuay/cmd/cquay-audit@v0.1.0
```

PowerShell equivalent for discovery:

```powershell
$env:GOPROXY = 'https://proxy.golang.org'
go list -m -json github.com/maldous/CircularQuay@v0.1.0
```

Verify the installed commands from a clean module/cache location and inspect:

- `https://pkg.go.dev/github.com/maldous/CircularQuay@v0.1.0`
- `https://pkg.go.dev/github.com/maldous/CircularQuay/cmd/cquay@v0.1.0`

Use Go tooling rather than manually lowercasing or constructing escaped proxy paths. Version v2+ needs `/v2` in module/import/install paths. Private dependencies require `GOPRIVATE` before resolution; the operational ciphertext repository must not become a public Go module.

Never move/reuse a published tag. For a defect, publish a higher corrected version and a `retract` directive where appropriate. Retraction is not deletion, and cached source may remain accessible. Negative lookups before a tag exists can be cached; do not retag to fix propagation. Official reference: <https://go.dev/doc/modules/publishing>.

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

The versioned Agent Skills package lives at `integrations/skills/circularquay-operations/`. Validate `SKILL.md` metadata, references and actual prerequisites, then test installation/reload/removal in each claimed client. The standard does not establish one universal public skills index. Tagged repository/archive distribution is a valid route; host-specific marketplaces need their own qualification.

Skill instructions cannot grant provider or MCP authority. Experimental `allowed-tools` metadata does not replace server authorization. No package should instruct an agent to solicit passwords, bypass scopes, dump environments or perform rotations by default.

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
