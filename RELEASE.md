# Maintainer release and operator handoff

This is the maintainer/operator entry point. It covers public distribution, account setup, independent installation/auditing, and every post-installation enrollment step. The product README covers functionality and user-facing usage; publisher credentials and registry procedures belong here.

All commands use a chosen immutable version. `v1.0.0` below is an example until a release with that tag is published. Do not interpret a manifest, example URL, generated report, or successful build as evidence of publication or operational qualification.

## 1. Identity, versions and evidence

| Item | Canonical identity |
| --- | --- |
| Repository/module | `github.com/maldous/CircularQuay` |
| Runtime command | `github.com/maldous/CircularQuay/cmd/cq` |
| Audit command | `github.com/maldous/CircularQuay/cmd/cq-audit` |
| Go SDK package | `github.com/maldous/CircularQuay/sdk/go` (root module) |
| TypeScript SDK | npm `@maldous/cq-sdk` |
| Python SDK | PyPI `cq-sdk`, import `cq` |
| Native executables | `cq`, `cq-audit` (`.exe` on Windows) |
| Public OCI package | `ghcr.io/maldous/cq` |
| Official MCP Registry name | `io.github.maldous/cq` |

Record application/tag, policy schema, runtime MCP protocol, Registry descriptor schema, SDK, native artifacts, OCI digest, MCP bundles, skill/plugin package and audit-profile versions separately. A Registry schema date is not an MCP protocol version.

CircularQuay is the long product name; CircularQuay is the short name. Owned technical names use lowercase `cq`, including the native executables `cq` and `cq-audit` and release-asset prefix. Actual SDK package coordinates, configuration/state namespaces and the `cq` operations skill retain their documented identities pending their own coordinated identity changes. These names do not provide an alternate CLI executable. The initial source implementation uses Go 1.27.4. The researched SonarQube Community reference is 26.10.0.133078; qualify the actual pinned server/scanner/analyzer/rules on release. The runtime MCP design targets protocol 2026-07-28 and a qualified official SDK, independently from Registry metadata schema.

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

Retain the [whole-product control register and release decision](docs/security-profiles.md#whole-product-release-admission)
for the exact candidate, including the default/strict capability matrix.
Required profile controls have implemented and independently qualified
enforcement; a configuration field or capability advertisement cannot stand in
for it. Missing, stale, skipped, cancelled, unreadable or failed required checks
block promotion. Do not combine incompatible revisions' passing results.

Release acceptance uses the complete agreed v1 contract. Source checkpoints
and bounded checks support that end state; an incomplete candidate cannot be
declared complete by reducing the promised scope. Fix forward against observed
failures while preserving required data protection, reconciliation and recovery.
Record actual deployment profiles and evidence. Do not advertise strict
mode, premium WAF/bot services, contractual availability or geographic guarantees
without their actual implementation, plan entitlement and qualification. Spend
reductions use equivalent qualified deployment choices; they cannot waive
agreed capabilities, authentication, tenant
isolation, key protection, sensible abuse limits or the required engineering
gates for shipped code. Use the existing managed-service forecast for cost and
upgrade assumptions; no deployment, plan purchase or account connection follows
from that model.

### Version and branch policy

Use Semantic Versioning and Gitflow. `feature/<purpose>` and `hotfix/<purpose>` describe work; `develop` integrates reviewed changes; `release/<major.minor.patch>` stabilizes a selected release; `main` contains qualified stable releases. Preserve merge commits and back-merge release/hotfix fixes into `develop`. Do not use direct feature pushes to `main` or rewrite published history.

Every release has an immutable tag and a GitHub Release. Use `v1.0.0-rc.1`, `v1.0.0-rc.2`, then `v1.0.0` as the naming pattern. Candidate tags belong to the reviewed `release/1.0.0` branch, are marked **Pre-release**, and never become **Latest**. Stable tags belong to the reviewed stable merge on `main`; mark the selected stable release **Latest** only after verification. Never move an RC tag to a corrected commit or rename an RC into a stable release. Build the stable version from its own exact tag and record the comparison with the candidate.

An RC is a supported feedback stage for its explicitly listed test matrix, not a waiver of security or integrity gates. Complete the applicable source/security/artifact gates before any public tag, including an RC. Record wider native/provider acceptance still being collected. Stable promotion additionally requires the complete declared qualification matrix and independent review. Only the selected candidate's complete evidence can authorize promotion.

Use GitHub milestones for the candidate and final acceptance work. Link issues and reviewed PRs to the milestone; retain unresolved limitations in release notes. Close the milestone only when its stated acceptance is met, and use a new corrective version for post-release defects.

### Required GitHub Release contents

Draft the release first, attach and inspect every asset, then publish once. Immutability prevents repairing the published assets in place. GitHub's generated “Source code” archives are source downloads, not compiled installers. Follow the [GitHub release procedure](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository).

Each release must provide:

- A title such as `Circular Quay v1.0.0-rc.1`, maturity label, exact source commit, release date, changelog and links to reviewed changes.
- A platform/architecture table with direct links to each qualified asset, its size, verification instructions and required external tools. Never label a cross-build as a native acceptance pass.
- `cq_<version>_<os>_<arch>.tar.gz` for Linux/macOS and `.zip` for Windows. Each archive includes `cq`, `cq-audit`, the `cq` skill, license, release/install instructions, security/support guidance and qualification matrix.
- `cq_<version>_checksums.txt`, authenticated provenance and a complete SBOM including shipped external executables. Publish offline verification inputs when supported. A checksum beside a download is not independent publisher authentication.
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
gh release download v1.0.0 --repo maldous/CircularQuay --pattern cq_1.0.0_linux_amd64.tar.gz --pattern cq_1.0.0_checksums.txt
gh release verify-asset v1.0.0 ./cq_1.0.0_linux_amd64.tar.gz --repo maldous/CircularQuay
sha256sum --ignore-missing --check cq_1.0.0_checksums.txt
mkdir cq-1.0.0
tar -xzf cq_1.0.0_linux_amd64.tar.gz -C cq-1.0.0
```

Check archive entries before extraction; reject absolute/traversal paths and unexpected executables. The maintainer must verify these exact CLI commands against the qualified GitHub CLI version; source archives generated on demand are not the same as verified release assets. The [official verification documentation](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity) describes the trust route. For browser/offline downloads, use the release's authenticated verification bundle and platform hash tools; do not execute an unauthenticated installer.

For an operator-owned Unix per-user installation after verification, copy the binaries into a chosen user executable directory with mode `0755`, keep the documentation with the versioned archive, and explicitly add the chosen directory to PATH if necessary. `~/.local/bin` is a conventional choice. Use `shasum -a 256` on macOS when `sha256sum` is absent; stable macOS distribution additionally needs its stated signing/notarization qualification. Do not bypass Gatekeeper to manufacture a clean install result.

On Windows, verify the selected archive using the release's documented trust route and `Get-FileHash -Algorithm SHA256`, then use `Expand-Archive` into a versioned per-user directory. Run `cq.exe version` and `cq.exe doctor` there, or explicitly add that directory to the user PATH. The native core does not require Go, Bash, Node or Python. Verify the stated Authenticode/signing support; native ACL/DPAPI installation is a separate acceptance test.

Run `cq version`, `cq doctor` and a synthetic `plan` before enrollment. Feature dependencies such as SOPS and Git must be separately authenticated and installed only for the enabled feature. Copy/installing the binary does not configure a system service, import credentials or enroll a provider. Follow post-installation enrollment below for each of those explicit actions.

For upgrades, review release notes and schema compatibility, retain an independently restorable authorized backup, verify the new artifact and replace the selected executable atomically while following the workload's declared lifecycle procedure. Do not reactivate revoked or expired values during rollback. To remove a per-user installation, stop only services/schedules explicitly enrolled for it, remove its PATH entry and selected binaries, and explicitly revoke temporary grants before deciding which protected state/backups must be retained. Never indiscriminately delete shared credentials or unrelated service paths.

### Packaging toolchain and staging

The initial packaging target is GoReleaser OSS **2.19.0** with Go **1.27.4**, driven by `.goreleaser.yaml`. It produces native archives and a draft release; it does not establish signatures, SBOMs, native installer qualification or registry acceptance. Schema validation is narrower than an artifact build. Keep the build matrix labeled as a build matrix until native tests qualify each target.

Run `goreleaser check`, then `goreleaser release --snapshot --clean --parallelism 1` in an isolated checkout to inspect local development archives without creating a public tag or release. Do not use `--skip` to bypass a required qualification gate. For a qualified selected public tag, stage a draft, generate/verify required SBOMs and provenance, attach all evidence and install directions, test anonymous asset downloads in a clean environment, and only then publish the immutable release. RCs and stable releases use the same integrity requirements.

Native archives, Homebrew, Winget, Scoop, Linux packages and app stores are distinct distribution channels, each requiring its own maintained manifest, ownership/trust route, install/upgrade/uninstall tests and publication receipt. OCI/MCP/Go procedures below remain separately gated. Do not create paid signing accounts or broad long-lived credentials merely to add a badge or unqualified distribution route.

### Optional managed-key service enrollment

Use the standalone [managed-service contract](docs/managed-key-service.md) for
identity/key mappings, administration, custody limits, and the separate financial
model. Prefer already available paid Netlify/Cloudflare identity and management
interfaces; verify plan entitlements, seats, credits, and retention before
choosing a deployment. Synthetic user/account and operator/admin frontend
development/previews and a loopback local client dashboard are authorized.
That authorization excludes real backend integration and production deployment;
preview data stays synthetic and client configuration stays local. The minimal
hosted API and CLI/text authentication flow keep
their separately approved scope. Deployment and preview identities are separate;
previews contain synthetic keys only. Platform-account roles, site protection,
application identity, mapping/grant administration, and key reads are separate
permissions. Preserve server-side checks and redacted audit for every route.

Enroll exact trusted issuers/audiences, clients, stable subjects/domains, key
versions, grant windows, refresh/custody policy, and independently held recovery
recipients. Back up hosted keys, mappings, association-policy generations and revocations together;
restoring old grants must not undo a deny. Test key release, denied-domain access,
revocation, IdP freshness, audit failure, retention, and isolated restore before
operator acceptance. A mapped private key is intentionally disclosed to its
client: revoking the mapping stops future fetches but does not invalidate copied
keys or old ciphertext. Capacity, hosting costs, paid plans, release gates, and
native/hardware acceptance require their own actual receipts; no purchase or
production deployment follows from documentation.

Qualify both automatically provisioned associations: real remote key-service
wrap/unwrap and authenticated age-private-key retrieval. Record the selected
provider connection, key/version/algorithm, allowed operations, association
generation, protected authentication references and required provider context.
Prove independent disable/revocation, idempotent provisioning, partial/uncertain
outcome reconciliation, rotation and safe provider error classification.
Enrollment does not download an offline key or enable fallback. Online-only
has no local bypass; hybrid uses its separately approved finite snapshot path
only after a positively classified availability failure. Denial, trust failure,
malformed response, expiry, cancellation and unknown errors must block it.
Preserve existing copied-key/plaintext limits and never claim provider custody
from a key-service interface or SOPS metadata parser alone.

The [authorized launch profile](docs/security-profiles.md#authorized-launch-custody)
uses software custody within the verified existing paid Cloudflare or Netlify
allocation. It requires no paid external KMS or plan upgrade. Qualify the actual
host's runtime/secret/database/identity entitlement and CircularQuay-specific capacity
before selection; a paid plan label alone does not establish included credits,
database allowance, spare capacity or zero incremental cost. Do not change
shared-account limits or pause unrelated applications to enforce CircularQuay's budget.
Keep the runtime wrapping key outside the database, use independently random
profile data keys and vetted authenticated encryption, and publish the shared
runtime/KEK compromise boundary. A strict non-exportability/independent-custody
requirement must reject this software profile. Stronger native KMS/HSM remains
an explicit qualified migration destination, not a mandatory launch charge.

Configuration remains entirely client-side, including retrieval through the
operator's VPN/network, credentials, composition, validation and local save.
Never send configuration bodies, source credentials or protected references
to the hosted key service, a server proxy, analytics or logs. Browser limits
require a permitted client/native route. Test both CLI and configuration
opt-outs before any hosted registration/network access; a disabled hosted
service supports fully client-only use. Hosted metering records successful
logical profile/key access events, deduplicates retries, and cannot charge
local reuse or turn a denied/precommit failure into a successful fetch.

### Local-to-managed custody migration

Migration is an authorized, recoverable operation rather than a mode flag.
Inventory and validation run client-side under the existing key owner's
identity. Record the selected local age identities/public recipients, protected
OS or self-hosted KMS references, asset and key versions, association mappings,
offline windows and recovery dependencies. Do not scan unrelated credentials,
upload inventory/configuration or infer permission to export a key. Dry-run
output uses safe metadata and describes exactly which protected operations
require separate authority.

Classify each asset against actual source/destination provider capabilities:

| Existing custody | Permitted migration mechanism | Required proof |
| --- | --- | --- |
| Exportable local age identity | Explicitly authorized protected import where supported, or a new hosted identity followed by re-encryption for its public recipient. | Ownership, import protection, recipient match, encrypted recovery and independently verified new-identity decrypt. No implicit private-key upload during enrollment. |
| Existing envelope-encrypted data key | Native rewrap/re-encrypt-key operation where supported, or approved decrypt-and-wrap under the destination key. | Exact source/destination versions, authenticated context and successful destination unwrap; data-key retention and previous ciphertext copies remain risks. |
| Non-exportable OS/HSM/KMS key | Keep an authorized source connection, or decrypt on the qualified source host/provider and encrypt with a new destination data key. | Real export/import constraints, required source access, complete asset verification and supported algorithm/context conversion. A key reference is not portable key material. |
| Leaked/retired key or compromised data key | New independent key material and full asset re-encryption, plus issuer-native containment where applicable. | Rewrap alone does not repair prior plaintext/data-key exposure; retained old copies cannot be recalled. |

Bind the approved migration intent to asset/profile identities, source and
destination versions, target custody/availability policy and recovery plan.
Use idempotent operations, conditional revisions and a durable progress/audit
record. Handle interrupted writes, partial provisioning and uncertain provider
outcomes by reconciliation; never mark an asset migrated from a successful
association lookup alone. Preserve originals and provenance until verified
destination decryption, application activation and a tested recovery point.
Retirement and old-copy handling are separate approved actions.

For online-only activation, new accepted ciphertext cannot retain an offline
age recipient or another unintended OR path. A historical local key/copy
cannot be erased remotely. Explicit hybrid activation prepares compatible
encrypted fallback under its separately approved key/version/window, and tests
availability-only admission without widening permissions. Independent windows
or custody domains may require separate data keys and artifacts. Verify every
supported algorithm, provider import/export profile, native host, SDK/CLI path
and restart/recovery schedule before advertising migration support. A mock
or successful remote wrap alone is not end-to-end migration qualification.

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

### Product go-live and commercialization: service provisioning

This same release/operator document owns the product go-live and
commercialization provisioning procedure, alongside package-publisher account
setup and production enrollment. The browser/client baseline includes
project-specific observability, privacy and applicable external-service setup.
Each environment uses its own approved accounts, project registrations, grants
and consent records. Independently review any reused integration for licensing,
compatibility and the client/account/operator data-protection boundary.

Before activating a service, record its CircularQuay environment and surface,
purpose and data flow, accountable operator, actual existing entitlement and
allowance, approved administrative route, protected configuration references,
verification owner and cleanup/withdrawal procedure. Use separate development,
preview and production projects/settings where applicable. Reuse the verified
existing Cloudflare/Netlify allocation only within its actual entitlement;
this checklist authorizes no new account, paid signup, subscription, credential
acquisition, deployment or provider activation.

Use the existing [managed-service contract](docs/managed-key-service.md#end-customer-accounts-subscriptions-and-support)
for customer identity, usage/history, invoices/subscriptions and support
boundaries, and its [hosting economics](docs/managed-key-service.md#hosting-economics-and-capacity-assumptions)
for the separately recorded cost/allowance assumptions. These supporting
contracts do not move the main provisioning plan out of this document. Product
prices, paid support, payment/CRM provider selection and actual allowances
remain explicit decisions; operator hosting invoices are not customer billing.
Register only the selected CircularQuay-specific applications/projects through their
verified owners, with separately authorized setup and environment access.

Use explicit per-service states: **pending applicability**, **not applicable
with rationale**, **blocked with prerequisite**, **configured**, **deployed**
and **independently verified**. A source hook or valid disabled example is not
configured; configured is not deployed; deployed is not verified. Record actual
state/evidence in the candidate's existing qualification and operator records,
not credential values or account identifiers in this public document.

The managed service uses `circularquay.aldous.info` and `cq.aldous.info` only
after an authorized CircularQuay hosting target is identified and independently verified.
Register both names on that same target before applying the provider's exact
DNS requirements. Keep account, zone and site identifiers in protected operator
records. Do not create placeholder or dangling DNS records, infer a site from
an unrelated application, or change the existing apex, wildcard, mail or other
service records. Domain preparation does not activate a managed service or
expose a local developer preview. Record actual target, TLS, origin, issuer,
redirect and DNS readbacks in the existing qualification record before use.

| Service/category | CircularQuay-specific provisioning and applicability | Initial disposition until evidence exists |
| --- | --- | --- |
| Sentry/technical diagnostics | Verify the CircularQuay project/environment, protected destination reference, operator access and actual allowance. Choose browser/server event purposes separately, fixed release identity, sampling/retention, source-map access and redaction before sending. A filtered code-only server envelope does not establish a full browser SDK, replay or source-map setup. | Pending purpose, configuration and entitlement verification; no automatic replay, raw error/stack collection or private-context capture. |
| iubenda/privacy consent | Verify the CircularQuay site/policy registration, licensed policy languages and region/purpose settings. Optional collection defaults off; consent notices and controls must be accessible/localized and support withdrawal. | Pending CircularQuay configuration and licensed policy coverage. An Arabic UI translation does not establish an Arabic legal policy or vendor support. |
| AdSense/advertising | Decide suitability per surface and verify CircularQuay site approval, slots, policy and the selected consent/TCF requirements before any load or placement. Secret/configuration/account/operator pages do not inherit advertising applicability from a public landing. | Pending explicit applicability/approval; mark unsuitable surfaces not applicable with rationale. No ads, charges or site approval are inferred. |
| Zaraz/optional analytics | Verify a CircularQuay-specific purpose and allowlisted coarse events, consent enforcement and actual platform allowance. No event properties containing user/context data or help/search queries. | Pending purpose and configuration; collection remains off. Do not add Google Analytics by inference. |
| Support/transactional email or feedback | Select the actual approved CircularQuay support route and, only where applicable, a separately qualified sender, verified-recipient policy, confirmation and uncertain-delivery reconciliation. Verify the selected native support or external feedback route; do not enable a second route from historical settings. | Pending product route/sender/entitlement; synthetic tickets are not delivered messages. No duplicate helpdesk, vendor signup or automatic diagnostic attachments. |
| Cloudflare/Netlify application services | Verify the actual CircularQuay project, issuer/app registration, environment bindings, database/storage custody, roles and included capacity. Hosting-account administration, application login and workload/key permission are distinct. | Pending actual project/issuer/entitlement evidence. Native publication targets and synthetic UI roles do not establish a connected client service. |
| Status/discovery and engineering services | Select applicable status/monitoring links and property verification. Provision the required engineering services with their own owners, protected CI identities and allowances. | Pending actual service evidence; a verification tag/status link is not monitoring acceptance. Historical scans do not qualify the selected candidate or waive required Sonar/CodeQL and other engineering gates. |
| Browser/PWA/Android considerations | Identify actual shipped platform/package, permissions, offline/cache behavior, accessibility and vendor policy obligations. | Pending platform applicability; no native app or AdMob integration is implied by browser support. |

Do not copy another project's service endpoints or broaden CSP to unrelated vendor
origins. Transfer only reviewed source patterns with compatible ownership,
licensing, data protection and current contracts. The client configuration and
keys remain local under the existing [client ownership boundary](docs/configuration.md#client-side-configuration-ownership).
Keep private help/search input, configuration/source credentials, environment
values, protected references, tokens and key material out of vendor events,
URLs, error payloads, source maps and support attachments. Diagnostics require
an allowlisted metadata schema and separate privacy/security review; a higher
verbosity or an installed SDK cannot relax it.

#### Authorized setup routes and operator readback

Use the selected release's supported configuration contract; do not invent a
CLI flag, client setting or remote administration command from a vendor recipe.
Record the exact reviewed SDK/tool version when one is used. Authenticate
through the approved provider interface, with least-privilege operator access
and MFA/recovery custody. Separate public routing settings from administrative
credentials; public settings still need origin, shape and purpose validation.
Provision only the services selected for that environment and surface:

1. **Diagnostics:** select the CircularQuay organization/project in Sentry and
   obtain its routing configuration through the authorized owner. Set the
   environment, immutable release identity, allowance, retention, access and
   alert recipient. Review the pinned SDK's complete automatically collected
   fields and integrations; disabling default PII alone is insufficient.
   Enforce the application's allowlisted event projection before export, drop
   unknown fields and keep replay, raw exceptions, user context, breadcrumbs
   and tracing disabled unless separately qualified. Upload only the selected
   public source maps with an isolated build credential; keep private maps and
   upload credentials out of served assets. Verify a synthetic allowed event
   and rejected canaries at the destination. Use the official
   [data-filtering](https://github.com/getsentry/sentry-docs/blob/master/platform-includes/configuration/before-send/javascript.mdx)
   and [integration inventory](https://github.com/getsentry/sentry-docs/blob/master/platform-includes/configuration/integrations/javascript.mdx)
   for the actual SDK, not assumed defaults.
2. **Privacy and consent:** register the selected CircularQuay website/app in
   the approved iubenda account. Declare actual purposes/vendors and verify
   licensed policy languages, regional coverage and retention with the privacy
   owner. Configure notices and prior blocking in that site's Privacy Controls
   and Cookie Solution, then project only approved public configuration into
   the client. Verify decline, purpose-specific acceptance and withdrawal with
   actual network/storage inspection. Follow the official
   [setup guide](https://www.iubenda.com/en/help/1177-cookie-solution-getting-started/).
   A generated banner or vendor account does not establish legal compliance.
3. **Advertising, when approved:** the authorized publisher adds the chosen
   public site in AdSense, verifies ownership through a selected supported
   method and requests review. Record actual site approval and applicable
   privacy/messaging/CMP requirements before loading advertisements. Keep
   rejected or pending sites and inappropriate surfaces disabled; use vendor
   test procedures rather than clicking live ads. The official
   [site connection and approval procedure](https://support.google.com/adsense/answer/7584263?hl=en)
   distinguishes verification from approval. It does not grant permission to
   monetize protected configuration or administration pages.
4. **Optional analytics:** in the selected Cloudflare zone, configure the
   approved Zaraz tools and consent purposes, bind each tool to its purpose and
   provide reviewed English/Arabic notices where supported. If another CMP is
   selected, qualify its consent synchronization and withdrawal behavior rather
   than presenting conflicting consent choices. Permit only the coarse event
   contract; exclude page/query/configuration/user properties. Verify no tool
   loads before required consent and that withdrawal blocks later events.
   See [Zaraz consent setup](https://developers.cloudflare.com/zaraz/consent-management/enable-consent-management/).
5. **Hosting and application identity:** provision the chosen CircularQuay
   project with isolated preview/production bindings, database/storage,
   approved domains/redirects/issuers and least-privilege deployment identities.
   Verify actual capacity, restore, tenant isolation and runtime secret access.
   For Workers, use [secret bindings](https://developers.cloudflare.com/workers/configuration/secrets/)
   and the selected [D1 setup](https://developers.cloudflare.com/d1/get-started/)
   where applicable. Ordinary `wrangler secret put` deploys immediately;
   stage with the documented versioned-secret route when separate promotion
   is required. Do not run a secret-setting command as a harmless preflight.
   For Netlify application login, use the project's
   [Identity setup](https://docs.netlify.com/manage/security/secure-access-to-sites/identity/get-started/)
   and [registration/provider settings](https://docs.netlify.com/manage/security/secure-access-to-sites/identity/registration-login/).
   Select the approved registration policy and callback origins explicitly.
   Dashboard SSO, site access gating and application authentication do not
   confer workload/key authorization. Verify every protected action separately.
6. **Support and delivery:** select the approved native support route first.
   If transactional email is required, the authorized sender owner verifies
   the chosen domain, sender and delivery allowance; for Brevo use its official
   [domain authentication procedure](https://help.brevo.com/hc/en-us/articles/12163873383186-Authenticate-your-domain-with-Brevo-Brevo-code-DKIM-DMARC).
   Stage DNS changes through the domain owner and verify them independently.
   Store delivery credentials in a protected server reference. Exercise only
   authorized synthetic-recipient tests, including rejection and uncertain
   delivery. A feedback widget needs its own selected provider, consent and
   privacy review; it is not enabled merely because support exists.
7. **Status, discovery and engineering:** the service owner selects the actual
   monitoring/status provider, alert destination, public health projection and
   website property. Verify ownership through that provider's official route
   without placing administrative tokens in public metadata. Exercise a safe
   alert and recovery before advertising monitoring. Configure required
   repository/CI analysis services for the exact repository and candidate;
   retain the independent [qualification gates](docs/qualification.md).
   Search-engine verification is separate from health monitoring, and neither
   is an engineering or security pass.

The operator receipt records each selected service, environment/surface,
accountable owner, approved setup route, protected reference, observed state,
verification artifact and next action. Keep administrative configuration and
provider data outside the public repository. Recheck current provider terms,
entitlements and actual settings at the release cut; documentation establishes
no account ownership or completed provisioning.

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

Create an isolated `cq-audit` project with fabricated allowed/denied resources. Use separate scoped test identities. Verify retrieval, denial, rotation and revocation. Audit claims require actual native events and retention evidence. Akeyless API-key authentication is documented for temporary/POC use, not a production recommendation.

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
mkdir -p ../cq-build
go build -trimpath -o ../cq-build/cq ./cmd/cq
go build -trimpath -o ../cq-build/cq-audit ./cmd/cq-audit
../cq-build/cq --config examples/demo.json plan
```

PowerShell uses `New-Item -ItemType Directory -Force ../cq-build`, explicit `.exe` output names and `../cq-build/cq.exe`. Keep build products outside the source checkout so an audit can bind every source byte to its Git tree. Native installation does not require Go. A command named `go` must actually be the expected compiler; record `go version` rather than assuming the executable name proves a toolchain.

### Run synthetic assurance

From the repository root:

```sh
../cq-build/cq-audit run --profile source --tool-manifest /absolute/private/tools.json --output ../cq-evidence/source
../cq-build/cq-audit run --profile enterprise --tool-manifest /absolute/private/tools.json --output ../cq-evidence/enterprise
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

`cq-audit sonar-init --directory ABSOLUTE_PRIVATE_DIRECTORY` creates new
protected bootstrap state and refuses existing state. Start only the selected
isolated Compose project using `audit/compose.yaml` and that directory's
`compose.env`; choose an unused loopback port and satisfy the pinned server's
kernel/memory prerequisites on the audit machine. After the server is ready,
`cq-audit sonar-bootstrap --directory ABSOLUTE_PRIVATE_DIRECTORY --url http://127.0.0.1:PORT --project circularquay`
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

### Client-service and post-deployment verification

Perform these checks only for an explicitly authorized candidate/environment
and service activation. Preserve pending/not-applicable/blocked services in the
existing evidence rather than reporting the whole baseline enabled.

1. Bind the exact client/service artifact, configuration schema and release
   identifier to the approved CircularQuay-specific environment and protected settings.
   Recheck current allowance, purpose, role/tenant boundaries and approved
   provider administrative route without exposing credentials in readback.
2. Verify disabled/no-consent behavior by observing requests, scripts, frames,
   cookies and storage. Optional services must not collect before their
   qualified purpose/authority/consent. Keep consent state separate from private
   application state; define retention, deletion and access owners.
3. Verify only the selected script/connect/frame/image origins in CSP. Inspect
   actual delivered policy and network behavior; source configuration alone
   does not prove deployment. Test failed/missing/malformed settings and vendor
   outage without losing the current configuration/workbench draft.
4. With synthetic canaries, verify diagnostic redaction and release/environment
   mapping, bounded sampling/export budgets and selected source-map visibility.
   No config, key, user identifier, raw response, authorization material or
   help/search query may escape through telemetry or URLs. Keep native required
   audit qualification independent of best-effort client diagnostics.
5. For an approved optional service, verify opt-in and withdrawal, later consent
   changes and reload/cache behavior. Missing or invalid required consent must
   deny collection. Test third-party loading and advertising only on the
   specifically approved surfaces; retained vendor in-memory state may require
   reload after disabling, which the UI must explain.
6. Verify initial English and one Arabic locale across notices, settings,
   errors, accessible labels, help/search and applicable communications. Require
   complete reviewed translations for the declared scope, stable machine IDs,
   locale-aware plurals/number/date/time/currency behavior, RTL/bidi isolation,
   text expansion and keyboard/screen-reader/device checks. Explicit English
   fallback is not a completed Arabic translation; pseudolocalization is not
   language-quality evidence. Record licensed vendor-policy language gaps
   separately from app translation coverage.
   Exercise language and light/dark/system selectors before authentication and
   through landing, sign-in and client journeys, including keyboard/screen-reader,
   zoom/reflow, forced-color and reduced-motion behavior. Verify explicit device
   choices, actual profile preference scope where supported, browser defaults,
   reset/logout/account switching and blocked storage without losing form or
   configuration work. Country/region hints are approximate display defaults,
   never identity evidence; do not provision a new GeoIP service or send IPs
   merely to select a language or theme. Retain actual browser/assistive-technology
   evidence and missing support; a synthetic profile is not an authentication or
   profile-persistence pass.
7. Check client-local full-text help/search and progressively loaded public
   articles for no query/config upload, missing/stale/mixed-version behavior,
   cold/warm bytes, result latency and repeated-query responsiveness. Search
   corpus and locale completeness must reflect actual reviewed shipped text,
   not topic/control counts alone.
8. For applicable support/email, verify preview/confirmation, account/tenant and
   recipient checks, metadata-only idempotency, denied access and uncertain
   outcomes without automatic replay. Keep support delivery separate from an
   optional email-copy failure. Real outbound tests need their own authority.
9. Rehearse disabling the selected service, withdrawing consent, revoking only
   its CircularQuay-specific grant/reference and restoring the last qualified client
   version/configuration. Preserve unrelated tenants and their grants. Record
   observed results, limitations and remaining owners in the candidate's
   existing qualification evidence before marking the service verified.

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

Publish the public image to `ghcr.io/maldous/cq`, linked to the source repository using `org.opencontainers.image.source`. Public packages are free under current GitHub terms; package visibility still needs verification. Prove an anonymous digest-pinned pull. Do not assume public repository visibility made the image public.

Keyless signing and release attestations should bind the expected repository/workflow. Signing, MCPB certificate trust and checksum verification are separate assurances. Document every required permission and remove temporary credentials after use.

### Go module indexing

There is no separate Go publisher account. Go publishes a public module through immutable semantic-version Git tags. Preserve exact casing in `github.com/maldous/CircularQuay`.

After the audited commit/tag and release assets are ready, trigger discovery:

```sh
GOPROXY=https://proxy.golang.org go list -m -json github.com/maldous/CircularQuay@v1.0.0
go install github.com/maldous/CircularQuay/cmd/cq@v1.0.0
go install github.com/maldous/CircularQuay/cmd/cq-audit@v1.0.0
```

PowerShell equivalent for discovery:

```powershell
$env:GOPROXY = 'https://proxy.golang.org'
go list -m -json github.com/maldous/CircularQuay@v1.0.0
```

Verify the installed commands from a clean module/cache location and inspect:

- `https://pkg.go.dev/github.com/maldous/CircularQuay@v1.0.0`
- `https://pkg.go.dev/github.com/maldous/CircularQuay/cmd/cq@v1.0.0`

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
on GitHub, the npm `@maldous` scope, and the PyPI `cq-sdk` project. Name
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
`cq-sdk`, GitHub owner `maldous`, repository `CircularQuay`, the reviewed
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
npm install --save-exact @maldous/cq-sdk@1.0.0
python -m pip install 'cq-sdk==1.0.0'
```

Verify `pkg.go.dev/github.com/maldous/CircularQuay/sdk/go@v1.0.0`,
`npmjs.com/package/@maldous/cq-sdk/v/1.0.0`, and
`pypi.org/project/cq-sdk/1.0.0/` against the release receipt. Production
consumers use their approved package mirror/registry, dependency lock and
hash/provenance policy. Language packages do not silently fetch or install a
native executable. TypeScript/Python local adapters require a separately
authenticated native `cq` artifact and its exact digest.

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
2. Include exact ownership metadata `io.modelcontextprotocol.server.name="io.github.maldous/cq"`.
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

Validate, pack, inspect the explicit allowlist and test a fresh compatible client for each platform/architecture. Bundle authenticated feature dependencies or explicitly require them through `doctor`; `cq` alone does not imply SOPS is installed.

Host immutable `.mcpb` assets on GitHub Releases. Registry references require the exact `fileSha256`. Independently verify that digest: Registry documentation says clients, not the Registry itself, enforce it.

Do not promise automatic architecture selection by every client. Keep the first listing OCI-only if native selection is unqualified. MCPB X.509/PKCS#7 signing is not Sigstore provenance; a self-signed signature is not automatically trusted. If a target requires unavailable paid trust, record that channel incomplete rather than bypassing client security.

## Lifecycle integration qualification

Before claiming [canonical governance](docs/configuration-governance.md) or
[lifecycle integration](docs/credential-lifecycle.md), record the exact schema,
effective-policy compiler, directory/SCIM provider, scanner instance/entitlement,
repository/App installation and issuer adapter profiles. Verify independent
least-privilege credentials/permissions, hierarchy locks/group conflicts,
provision/deprovision freshness, metadata-only ingestion, exposure exceptions,
issuer-native remediation and cross-interface authorization/validation/audit.
Do not publish a connector recipe that silently transmits a credential or uses
GitGuardian's internal identifier as a local hash lookup.

Synthetic tests and read-only research establish no account connection, purchase,
scanner enrollment, live rotation/revocation or verified provider entitlement.
Live-profile qualification requires its own authorization and redacted receipts;
incident status, issuer containment, replacement activation and external audit
retention remain independently evidenced. OTel export qualification proves only
its safe signals and delivery behavior, not immutable storage. Keep unsupported
features visible in revision-scoped qualification without weakening the full-v1
product contract.

## 7. Skills and client-specific plugins

The versioned Agent Skills package lives at `integrations/skills/cq/`. Validate `SKILL.md` metadata, references and actual prerequisites, then test installation/reload/removal in each claimed client. The standard does not establish one universal public skills index. Tagged repository/archive distribution is a valid route; host-specific marketplaces need their own qualification.

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
- Disable/revoke optional client services under their CircularQuay-specific consent,
  retention and credential boundaries. Remove only the approved project scripts,
  slots/settings and grants, verify network/storage behavior after withdrawal,
  and retain required audit evidence. Withdraw only the selected CircularQuay
  grant; disabling a UI control does not establish provider-side data deletion.

If a credential enters a public asset or module, revoke/rotate at the issuer first. Tag deletion, retraction, descriptor withdrawal, or Git history rewriting does not make an exposed credential safe. Follow [incident response](docs/operators/incident-response.md).

## 9. Release-document acceptance

A developer unfamiliar with this project must be able to use only the tagged source and this document to install locally, run synthetic assurance on their infrastructure, understand unqualified capabilities, enroll one isolated free provider test, exercise optional MCP authorization, identify/publish eligible channels, and upgrade/remove test state safely.

Every remaining step needs a state, owner action, prerequisite, official route, expected result, verification, cleanup and qualification consequence. Missing resources remain **not run**, not successful. Credentials, MFA recovery codes, private grants and raw secret responses never appear in this ledger or public review material.


Commercial go-live additionally needs a source-bound readback for each required
applicable service: selected CircularQuay project/application, verified owner and current
entitlement, protected setting references, observed consent/redaction/CSP and
post-deployment checks, withdrawal/recovery evidence and remaining limits.
Required pending, blocked, failed or unverified capabilities block their
commercial activation; record them without claiming the full product ready.
A not-applicable disposition explains service/surface suitability and does not
waive the agreed v1 feature set. Keep package-channel publication, hosted-service
readiness, client integration and commercial/customer acceptance separately
observed, with accountable owners in the existing qualification records.
