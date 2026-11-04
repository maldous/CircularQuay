# Implementation and qualification plan

The user explicitly authorized initial hosted SOFTWARE crypto + age using only
existing minimum paid Cloudflare/Netlify: vetted crypto, protected KEK outside
DB, independently random profile DEKs and scoped auth/tenant isolation/required
audit. Runtime/shared-KEK compromise may expose all reachable DEKs; this is not
independent non-exportable/HSM custody. Stronger required profiles are unavailable
here, with versioned managed/HSM migration retained. No paid external KMS/IdP/
monitor/helpdesk/upgrade or account connection is a launch prerequisite or
authorized now. Actual existing tiers/issuer metadata is unverified; conditional
CF-first versus Netlify-included-Identity must be qualified, not invented. The
current forecast is hosting-only, with product tiers distinct from provider plans,
allocated versus already-paid incremental spend and allowance/DB-active inflections;
no labor, CRM/payment fees or customer price/margin is embedded. Production
custom UI remains outside deployment authorization; the separately authorized
synthetic frontend work proceeds in its isolated checkout. Real backend tests
remain separate from static design and preview evidence.

This is the durable engineering contract, not a completion report. Implementation and qualification use explicitly selected, isolated development environments; verify the actual executor and checkout for each session. Operators install on their own infrastructure and supply host-specific feedback; existing operational hosts and active recovery work are not changed by this implementation task.

V1 is one complete product contract. Work packages describe implementation
dependencies and parallel ownership; they do not define reduced releases or
defer a promised capability. Source checkpoints make changes reviewable. Fixes
move the implementation forward against the same acceptance requirements.
A full-v1 claim requires the whole declared contract and its revision-bound
evidence. The separately requested near-term lean secure launch can defer
optional enterprise functionality under an explicitly bounded release profile;
it does not abandon the North Star or waive engineering/security gates for
what is shipped. Record actual scope, limitations, cost and receipts in the
existing release/qualification records.

## Accepted product decisions

- Public project `maldous/CircularQuay`; Apache-2.0; native `cquay` and independent `cquay-audit` commands.
- One Go module with idiomatic package/file/asset names; native runtime distribution without requiring a language runtime.
- Explicit read-through interoperability; one authority per value and exact source/destination identity selection.
- SOPS/age offline, optional OpenBao Transit online, explicitly authorized hybrid snapshots, and separately labeled native deployed snapshots.
- Native Linux/systemd, Windows, Termux, container, Windmill, Stokowski and cloud recipes/adapters, qualified individually.
- Optional local/hosted authenticated MCP with filtered metadata, registered actions and separate per-user/client/key raw-read permissions.
- Provider-independent identity authorization: use an existing OAuth authorization server directly when it supports a registered CircularQuay API, or a self-hosted server such as Keycloak. No particular identity product, hosted broker or paid plan is a runtime prerequisite.
- Named issuer trust and typed access-token profiles, configurable scope/role mappings, exact principal/client/resource associations and bounded lifetimes. Context names and permissions are operator-defined; development/production labels are examples only.
- Standard asymmetric client authentication and sender constraints where qualified: private-key JWT, DPoP and mutual TLS. Offline identity authorization requires an issuer-authorized artifact binding and an explicit clock/revocation trust contract; an ordinary login token or OAuth refresh permission does not establish it.
- No extra recurring paid provider dependency for the personal reference; free testing accounts authorized, no automatic paid conversion.
- Strict first-party quality with transparent upstream advisory/mitigation reporting.
- Product-wide [default and strict security profiles](security-profiles.md),
  mandatory baseline safeguards in both, centrally enforced downgrade resistance
  and exact control-to-schema/code/test/deployment evidence.
- Official Go, TypeScript and Python SDKs share the supported CLI/MCP contract,
  explicit credentials, cancellation, safe errors and cross-language conformance.
- Runtime OpenTelemetry metrics, traces and structured diagnostics are v1
  capabilities, configured through protected policy with bounded export and no
  value-bearing telemetry. Required security audit remains independently durable.
- Policy-controlled release awareness supports interactive notices, explicit
  automation, stable/RC channels, enterprise feeds and air-gapped minimum versions.
- Public README for motivation/features/user usage; `RELEASE.md` for maintainer/operator account, index, audit, deployment and post-installation directions.

## Managed-key product contract

The optional [managed key service](managed-key-service.md) associates a verified
existing identity with BOTH authorized KMS and age-key associations. The latest
explicit decision requires automatic provisioning of both on authorized enrollment,
without offline-key download or hybrid activation. Scoped UI/API/CLI/MCP
administration manages allowed principals, clients, windows, modification, and
revocation. Reuse available paid Netlify/Cloudflare identity and management
capabilities first for the minimal hosted API and confirmed CLI/text flow.
Production custom key-specific UI/screens remain unapproved. A separate frontend
checkout may build synthetic/mock functional previews and loopback
local-configuration simulations, but no real backend, provider/key/account
access, deployment, purchase or new email/onboarding suite is authorized. Key distribution is
explicitly distinct from protected local custody and remote unlocking. Bundle,
recipient, recovery, and authorization-domain isolation apply to every unlock
path. Private keys, released data keys, and plaintext cannot be recalled;
SOPS/age do not enforce expiry. Commercial packaging and financial scenarios
are assumptions, not chosen prices, traction, or deployment authorization.

Production custom administration UI/screens remain unapproved, not a current
service implementation workstream. Prior UI proposals are not proof of delivery.
The explicit [landing/client-SPA design](managed-key-service.md#explicitly-confirmed-ui-design-and-bounded-preview-work)
authorizes a separate frontend checkout for synthetic/mock functional previews
and loopback local-configuration simulations only. It does not authorize real
backend/provider/key/account access, production deployment, purchase or a new
suite. Reuse existing platform services and preserve CLI/text onboarding and
product capabilities.

### Established baseline and first-run acceptance

The latest authorized launch baseline includes hosted SOFTWARE cryptographic
operations and authenticated age-private-key retrieval, with BOTH associations
automatically provisioned on authorized profile enrollment. This supersedes the
earlier age-only baseline, with explicit window checks, usage classes and an
opt-in subscription opportunity. It does not include enterprise config
orchestration or every optional integration by default. Preserve those as separately selected
CircularQuay capabilities and independently costed deployments. The user
clarified KVM/PVM meant **KMS**, configured per user profile. Implement the
[separate custody contract](managed-key-service.md#per-profile-age-and-kms-custody);
current source does not yet establish cloud-KMS support.

First run resolves existing configuration, discovery or manual fallback,
authenticates through the selected identity, then immediately initializes the
hosted key service unless explicitly disabled by CLI or configuration. Implement
both opt-out surfaces and deterministic precedence/lock-conflict handling
before any hosted-service network call. Test default initialization, CLI
disable, config disable, conflicting/locked settings, untrusted/invalid source,
self-hosted/offline paths and no personal-file/value/key uploads. Registration
must be authorized, idempotent and concurrency-safe for user/client/tenant and
noninteractive workload-identity CI enrollment; a retry creates no duplicate
association or billing event.

Managed release windows default on for future key retrieval; durations remain
policy-defined. Registration cannot grant an unauthorized key/tenant role or
production access. Ending a release window does not expire age or revoke copied
private keys/ciphertext. The hosted-access-event metric is defined; allowances,
thresholds, prices and payment/hosting vendors are undecided; test usage notices and explicit subscription opportunity
without automatic charging. Evaluate vendors against the existing baseline
forecast without provisioning accounts or enrolling trials/purchases.

[Profile utilization](managed-key-service.md#profile-utilization-and-fair-metering)
counts hosted access events, not profile count/key slots: one daily hosted fetch
is one use, twenty independent hosts/fetches are twenty, local reuse adds zero.
Test denied/precommit failures at zero customer uses, postcommit uncertain
release reconciled once, and same-logical-access retries/concurrent requests
without duplicate metering. Distinct hosts cannot share one idempotency identity
to bypass independent access accounting. Provider attempts/logs/storage still
incur hosting costs. The current forecast is HOSTING ONLY for Personal/Business/
Enterprise product workloads, with total allocated versus already-paid incremental
spend and allowance/DB-activity inflections. No labor, external KMS, CRM/payment
fees, customer prices or margins are embedded; these examples authorize no spend.

### Per-profile KMS and hybrid acceptance

Implement [custody selection](managed-key-service.md#per-profile-age-and-kms-custody)
in the user profile: local age, hosted age-private-key retrieval, or a qualified
remote KMS provider/key/auth identity with explicit scoped operation grants.
The current SOPS adapter rejects cloud-KMS/key-group metadata; reuse upstream
backends with constrained credentials rather than relabeling existing support.
Require common schema/validation and consistent CLI/API/SDK/MCP behavior;
intended client UI parity remains design only. Do not invent deployed support,
provider permissions, flags, tools or provisioning authority.

Keep custody distinct from availability mode. Online-only required KMS has no
offline bypass. Explicit hybrid retains approved complete encrypted snapshot
fallback on qualifying availability failure within its window, with existing
scope/freshness/audit/fence checks. Denial, TLS/trust failure, malformed response
or expired fallback cannot trigger downgrade. Test both allowed hybrid fallback
and each prohibited downgrade. Offline remains explicitly local. Ordinary SOPS
recipients are alternatives, not AND; quorum needs an actually supported and
tested group/threshold mechanism. Retained data keys/plaintext or age fallback
cannot be remotely revoked or made to require KMS on every future decrypt.

Implement dual-association enrollment with durable idempotency, partial-failure
reconciliation, resource budgets and readiness only after both associations
exist. Resolve hosted disable/locks before network. Association creation does
not release offline keys, activate hybrid or silently substitute age for
required online KMS. No live paid provisioning is authorized here. Add actual per-provider operations/key-version/
audit/recovery costs to the existing forecast before commercial commitment.

### Local-to-managed migration and existing assurance

Implement the [end-to-end migration contract](managed-key-service.md#end-to-end-local-to-managed-migration),
not just a mode flag. Client-local metadata inventory/reconciliation covers age,
OS/hardware/self-hosted KMS definitions, encrypted assets, recipient/key versions,
profile grants/windows and recovery dependencies. Distinguish authorized reuse,
export/import, compatible data-key rewrap and payload re-encryption. Non-exportable
hardware keys cannot be copied; actual provider formats/import permissions and
source unwrap authority must be verified. Preserve originals/provenance and
separate local-key import from automatic dual association creation.

Use a scoped approved plan, staged protected output, durable migration IDs and
per-asset reconciliation; interruption/retry cannot orphan duplicate paid keys
or activate a partial estate. Verify destination decrypt/content/recipient/key
version and recovery before atomic activation. Retirement and rollback need
current separate authority, with deny/replay generations preserved. Online-only
has no offline bypass; only explicit authorized hybrid can activate compatible
fallback under existing availability/window/freshness/audit rules. Old ciphertext,
private keys/data keys/plaintext remain usable by holders after migration.

Apply existing SECURITY/threat-model/administration/security-profile/assurance
controls, positive and adversarial tests and exact qualification evidence.
Real-backend/native custody/migration/recovery evidence is separate from mocks;
an unavailable authorized backend is unqualified. The explicitly authorized
launch SOFTWARE custody tradeoff permits runtime-accessible protected KEKs and
random profile DEKs, with shared-runtime/KEK blast radius disclosed; it does not
satisfy a stronger independent/HSM requirement. Auth, tenant/purpose scope, audit,
replay/fallback controls and first-party gates remain mandatory. The
revised [full dual-service model](managed-key-service.md#hosting-economics-and-capacity-assumptions)
is now hosting-only: actual native meter rates, explicit workload/free-owner/
DB-active assumptions, allocated versus incremental spend and inflection points.
Labor, paid external KMS, CRM/payment costs and customer margins are removed. No live key
inventory/upload/migration, paid provisioning or new account connection is
authorized by this implementation plan.

### Client-only configuration acceptance

All config content, source credentials, resolution/validation/explain and save
are client-side. Browser/native clients access sources over the user's own
network/VPN and credential boundary. Test that no content/credentials/provenance
leak through hosted backend/proxy/analytics/logs, including corporate sources.
Browser CORS/file restrictions require honest permitted native CLI/SDK/local
integration, never a server proxy or silent unsupported options. Save locally.
Hosted state includes authenticated profile→KMS/age associations, crypto operations
and necessary enrollment/window/metering/billing. Protect keys without display
or logging. CLI/config disable before network supports fully client-only use.
The user clarified PVM/KVM meant KMS. Per-profile KMS is a required capability,
not an existing enum/flag or permission for live key provisioning here.
Qualify compatible KMS/age ciphertext, recipient/key versions and independent
key IDs/windows for separately protected domains before hybrid activation.
This is design; production custom UI remains unapproved. Separately authorized
synthetic/mock previews and loopback local-configuration simulations do not
change the client-only or no-backend boundary.

### Near-term lean launch and long-term production acceptance

Minimize near-term Cloudflare/Netlify spend using actual tiers/limits and a
bounded secure service profile. Premium WAF/bot/SLA/multi-region/enterprise
identity may be deferred where justified by the measured requirements and
budget. Keep authentication, tenant isolation, provider plus application-level
key encryption/custody, durable event metering and sensible abuse controls.
The current hosting-only forecast distinguishes product workloads from provider
plans, already-paid incremental spend and measured allowance/DB-activity limits.
Paid external KMS/IdP/monitor/helpdesk/upgrades are not launch prerequisites.
Actual tier/project/qualified customer identity metadata is unverified, so choose
CF first only if those existing facts qualify it; otherwise evaluate the existing
Netlify included-Identity branch without purchasing or connecting accounts. UI stays deferred,
all config stays client-side, and no lower-priced plan inherits a higher-tier
entitlement or an unmeasured reliability claim.

### Minimal service production acceptance

The minimal service must satisfy [production protection and service objectives](managed-key-service.md#production-protection-and-service-objectives)
without becoming a config/backend orchestration service. Qualify exact WAF/bot/
DDoS and per-IP/client/profile/tenant limit entitlements, headless CLI/CI access,
authoritative revocation/idempotency/usage consistency, encrypted durable key
storage/custody/residency, independent backup/restore/rotation and secret-free
observability with no public key-response caching. Measure realistic
availability/latency and geography/failure cases; no platform-name or free-tier
claim substitutes for evidence. Price fixed and usage-dependent protection,
limiter, storage, authentication, retention and operational costs before a plan
is called profitable. The guide's SLOs/plan proposals are design/sensitivity
inputs, not achieved results, live tariffs or procurement permission.

### End-customer account service acceptance

[Account/subscription design](managed-key-service.md#end-customer-accounts-subscriptions-and-support)
includes authenticated profile-successful-fetch usage/history, invoices,
plans/quotas/status, support and customer lifecycle. Qualify actual customer
identity, provider-hosted payment/subscription portals and support services
separately from Cloudflare/Netlify operator dashboards. No provider-admin
credentials or payment-card custody enter the client/model. Configuration and source credentials remain entirely local; synthetic/mock
account previews may be developed separately, while production account UI
remains unapproved.

Relevant CLI/API/SDK/MCP operations require common scopes, field/tenant isolation,
idempotent meter/payment/subscription reconciliation, signed-event verification,
durable audit and safe operation state. Future unattended financial execution
requires an explicit user-approved action/account/amount/currency/plan/time
boundary and applicable confirmation; capability does not authorize charges.
Test wrong-customer IDs, stale/duplicate/out-of-order events, lost responses,
payment/portal/support denial, budget/approval expiry and all cross-tenant paths.
Price support seats/labor, account history/storage/traffic and provider Billing
fees in the existing forecast without buying or connecting any service.

### Restricted operator administration design

The [operator plane](managed-key-service.md#restricted-remotely-hosted-operator-administration)
is separate from customer billing and client-config SPA. It handles scoped
customer/key metadata (public recipient/fingerprint only), activity/security/
usage and coarse geographic context, safe database/support/monitoring and
maintenance. Strong operator auth, distinct privileges, bounded/audited reads
and mutations, key redaction, privacy/retention and exact provider entitlements
are required. No client config/source credentials or routine private-key display.
Reuse hosting/payment/support/monitoring consoles; custom implementation remains
deferred. ISO27001 alignment is an evidence goal, not certification. No live
customer data or maintenance action is authorized by design.

### Account RBAC and profile-scoped enterprise operations

[Account RBAC](managed-key-service.md#account-rbac-and-scoped-authority) separates
permissions from tuple-bound tenant/org/customer/group/environment/fleet/
key-metadata/billing scope. Exactly one protected global application superuser
is required by design, not ordinary support and not an implicit key-reader.
Use strong auth, scoped routine identities, audited use and controlled recovery
that never leaves two active roots. Individual support assignments cover only
specified enterprise/customer subsets; agents have distinct bounded identities.
Test deny/conflict/time/delegation/revocation and cross-assignment scope-mixing
across UI/API/CLI/SDK/MCP, including financial confirmation and key-custody denial.

[Customer enterprise/fleet design](managed-key-service.md#customer-enterprise-administration-and-selected-fleet)
is separate from provider support. Verify actual IdP/API scopes, browser/VPN/
CORS/file access and explicit selected-agent capability. All config/source
credentials stay client-side. If a central fleet broker or provider function
requires hosted config/credential storage, report the conflict/options rather
than relax the boundary. No fleet agent/control command is verified in this
source checkpoint, and custom console implementation remains deferred. Price
operator/SSO/audit/recovery and optional fleet/provider costs independently;
no live accounts, grants, fleet actions or infrastructure changes are authorized.

## Managed-service development and delivery contract

The managed service is developed as the same versioned artifact and migration
sequence used for controlled releases. Its local developer deployment is
loopback-isolated, carries its own tenant/domain, storage, certificate authority,
and synthetic fixtures, and exercises verified authentication and server-side
authorization. A local origin, test header, developer account, or environment
label is never an authorization shortcut. Keycloak, wire-compatible identity,
and selected OpenBao fixtures are optional protocol inputs with fabricated
identities and secrets; they do not define a required customer IdP or an
implicit production trust relationship.

Per-instance CA trust is isolated to the test clients and removed with the
fixture. No permanent trust-store modification or TLS bypass is acceptable.
Local and CI runs share pinned artifacts, policy fixtures, migrations, health
checks, cleanup rules, and redacted evidence. Parallel runs receive separate
tenant, storage, audit, and synthetic identity namespaces. Netlify Dev and
Wrangler adapter tests use that same authenticated contract, while preview
validation is independently scoped to a preview application, audience, storage,
and synthetic keys. It never has production mapping or custody access.

Before promotion, managed-service qualification covers authentication/audience
binding, tenant/domain isolation, scopes, optimistic mapping updates, expiry,
revocation, concurrent changes, backup/restore, migration/rollback, health,
and durable redacted audit. Provider integration, native runtime, preview, and
production receipts remain separate. [SPIFFE/SPIRE](managed-key-service.md#local-development-integration-and-controlled-delivery)
can complement an IdP with workload mTLS identity, but each trust domain,
attestation, selector, and policy association is explicit; workload identity
does not by itself grant tenant, mapping, or key access.

Release delivery follows the Gitflow contract: feature branches integrate into
`develop`, a release branch stabilizes v1 release candidates, and approved
stable artifacts reach `main`. Every release candidate and stable artifact is
immutable, versioned, digest-identified, and qualified as its own bytes.
Rebuilds, rebranding, changed build inputs, and changed migrations create a new
candidate and repeat the applicable gates. Production promotion uses the exact
tested artifact and approved environment only after migration/rollback, health,
authorization isolation, backup/restore, and audit evidence pass. Publisher,
account, registry, and operator procedures remain in [RELEASE.md](../RELEASE.md).
The [qualification matrix](qualification.md#managed-service-design-status)
records the actual managed-service implementation and qualification status.

## Non-negotiable boundaries

The complete v1 [discovery and management contract](management-and-discovery.md)
includes typed remote enrollment, tenant/principal/client/provider resources,
configuration revisions, scoped grants, delegation, approvals, revocation,
audit and recovery. Separate runtime-policy, client-enrollment and managed
deployment schemas prevent local CRUD from becoming implicit remote authority.
The API, official Go/TypeScript/Python SDKs, CLI/UI and optional management MCP
share one operation map. [Administration security](administration-security.md)
defines mandatory signed intent/publication, independent approval, trust/key
lifecycle, freshness, replay/rollback, tenant isolation and durable mutation.

Do not implement new cryptography, a vault, arbitrary expression evaluator, global credential broker, universal process supervisor or general lease-renewal daemon. Reuse upstream SOPS, native provider SDK/auth lifecycle and platform facilities.

Do not access existing operational values to populate this repository. No live secret, private recipient, token, native credential blob or operational ciphertext belongs in tracked source, assets, fixture reports or public release metadata.

No automatic history rewrite, estate-wide rotation or production service restart. A later approved migration must have explicit scope, one authoritative writer and tested rollback.

## Work packages and dependencies

| Package | Dependencies | Acceptance |
| --- | --- | --- |
| Foundation | Execution authorization | Canonical layout, model/schema, language/security ADR, synthetic examples and required metadata. |
| Policy/core | Foundation | Pure decisions, exact scopes, collision checks, complete generations and no value-bearing diagnostics. |
| Offline/SOPS | Policy/core | Restricted subprocess, authenticated inputs, explicit recipients/endpoint allowlist and safe encrypted snapshots. |
| OpenBao reference | Core and approved isolated capacity | Transit/KV, native service auth, audit/seal bootstrap, denial and independent restore. |
| Providers | Core | Narrow value adapters with identity, version, lifecycle, typed failure and export contracts. |
| Native delivery | Core | Protected process/file/native projections, correct restart/recreation behavior and scoped ownership. |
| MCP | Core/provider boundaries | Per-request verified identity, filtered discovery, action/read separation and safe output/audit behavior. |
| Official SDKs | Stable CLI/MCP contracts | Go, TypeScript and Python parity, typed safe errors, explicit disclosure, cancellation/reaping, native conformance, locked builds, registry install and provenance receipts. |
| Observability and operations | Core/runtime boundaries | Safe correlated metrics/traces/logs, finite queues/export/shutdown, collector failure isolation, operational health, update awareness and local version policy. |
| Managed hosted connections | Qualified MCP and explicit hosted scope | External IdP/resource-server separation; typed tenant/environment/principal/connection isolation; approved identity associations; client and operation grants; lifecycle/revocation bounds and administrative audit. Local/CI/preview/production use the controlled-delivery contract in [managed key service](managed-key-service.md); see [hosted identity](hosted-identity.md). |
| Administration and assurance profiles | Core trust and managed-service contracts | Authoritative v1 API/resource/error/event and enrollment/deployment schemas; signed configuration/change integrity; complete typed CLI/UI/SDK/MCP lifecycle; monotonic profile enforcement; approval/revocation/audit/HA and downgrade-negative evidence under [management](management-and-discovery.md), [administration security](administration-security.md) and [security profiles](security-profiles.md). |
| Assurance | Starts with foundation | Fresh independent Sonar route, scope/coverage validation, complementary checks, formal/conformance and negative tests. |
| Distribution | Tested artifacts | Immutable native/Go/OCI/MCP/MCPB metadata, signatures, provenance, free account directions and clean install. |
| Operator qualification | Relevant implementations | Real reboot/device/provider/client/restore receipts, not merely compilation. |

Independent packages can proceed in parallel after their contracts are fixed.
Identity interoperability is qualified by capability and configured token
profile, not by a provider's support for login alone. A resource server accepts
access tokens for its own audience, never an ID token as a bearer substitute.
Provider-specific claims such as delegated scopes, application roles and client
identity require explicit typed validation and mapping before they can grant
an operation. The v1 JSON schema, Go model, validator, runtime, negative tests
and operator reference change together; no accepted field may be ignored or
claim support for an unimplemented integration. External provider entitlements
and free-tier limits remain deployment prerequisites, not CircularQuay gates
that can be bypassed by broadening trust.

The initial public product and policy contract are v1. Release stabilization
uses `release/1.0.0` and `v1.0.0-rc.1` candidates before a qualified `v1.0.0`.
An optional channel is advertised only for its implemented, qualified scope;
its absence does not imply that an incomplete promised v1 capability is ready.

Capability reviews distinguish value storage, bootstrap custody, human login,
machine identity, non-exportable key operations, application authorization and
native delivery. A connector name does not establish all of these capabilities.
Host keychains and biometric/mobile approvals are separate integrations from
protected filesystem or systemd/DPAPI projections; their actual boundaries are
recorded in [credential boundaries](credential-boundaries.md). Interactive
user-presence requirements cannot silently replace unattended recovery.

## Interface and data contract

Policy version 1 declares sources, workloads, bindings, targets, snapshot settings, grants and optional MCP. The schema has one canonical source under its owning package. Unknown fields, duplicate keys and invalid combinations fail validation.

Each source is explicitly bound to a provider and allowed resource scope. Production authentication cannot quietly fall back to a broad interactive identity. Retrieved values are strings with source/version/lifecycle metadata; no recursive evaluation. Field selection is delivery minimization, not a claim of field-level upstream authorization.

Resolve every required binding before activation. Track declaration/accepted, resolved, projected and active state separately. Independent provider values have no global transaction. No source error silently deletes destinations or overwrites native rotation.

## Startup, update and failure behavior

Online requires fresh retrieval at every launch, including crash restart. Offline/hybrid retain approved static last-known-good material during qualifying availability failures without requiring a human login. Denial, TLS/trust failure, malformed response and expiry never authorize fallback. Existing running processes retain their delivered values.

Use native scheduling, with five-minute jittered local/Git/OpenBao defaults and provider-specific budgets/backoff. Warn after three failed scheduled checks and escalate after 24 hours without successful authenticated checking. Measure failed synchronization time, not the age of unchanged values.

Native encrypted systemd/DPAPI projections are machine-local convenience, not canonical portable backup. Termux is post-first-unlock and best effort under Android background restrictions. Dynamic credentials do not become perpetual snapshots. Snapshot/export permissions are distinct from reads.

## Security assurance

Target zero first-party compiler/linter/Sonar issues and analysis warnings/errors. Cover overall and new code; no ignored/suppressed status used to manufacture zero. Require review evidence for actual hotspots. Complexity-rule limits replace the meaningless requirement of zero executable complexity.

Target 100% coverable owned executable coverage, including native code in its appropriate lane. Publish denominator, missing instrumentation, per-platform results and legitimate aggregate. Add policy decisions, mutation/property/fuzz, fault and model-conformance evidence; execution coverage alone is not proof.

Pin build/scanner/analyzer/model tools. Use Go vet/Staticcheck/gosec/govulncheck, CodeQL where applicable, installer/workflow checks, SBOM/advisory/license analysis, source secret prevention, archive/image inspection and real integration tests. Preserve every upstream advisory and tested mitigation. Block unmitigated exploitable risk.

Model exact authorization, generation/fallback, expiry/rollback, bootstrap acyclicity and MCP disclosure/ownership using TLA+/TLC finite bounds. Publish assumptions and replay traces against code. No end-to-end formal-security or certification claim.

## Acceptance scenarios

1. Clean install with minimal PATH, no language runtime and no network for an offline bundle.
2. Exact allowed resource works; wrong account/tenant/project/resource fails before upstream value release.
3. Conflicting names, forbidden variables, malicious references, oversized/unsupported values and hostile endpoints fail safely.
4. Partial source/update/write/restart failures preserve a controlled complete prior generation.
5. Online crash restart generates a fresh source retrieval; authorized static hybrid fallback works only for availability failures.
6. Expired/leased/revoked values are not reactivated through cache or publication.
7. Fabricated markers stay out of toolkit logs/argv/artifacts, except an intentionally authorized MCP read response.
8. Linux/Windows no-login cold boot; Android's expected first-unlock limit and actual background behavior recorded.
9. Synthetic two-source worker resumes automatically after reconnect, rotation and reboot; isolated Windmill schedule variant.
10. MCP wrong issuer/audience/client, guessed IDs, cross-user access, bad Host/Origin and mutable action bypass fail.
11. Independent backup restoration works without old machine TPM/DPAPI state; OpenBao needs independent data/seal recovery.
12. Unfamiliar developer follows only the tagged source and `RELEASE.md` to install, audit, provision a free synthetic provider, enroll, distribute or identify the exact remaining step, and clean up.
13. The same allowed/denied/expired/cancelled operation has matching SDK semantics
    in Go, TypeScript and Python; no value enters argv, logs, error details or hooks.
14. Synthetic OTLP collectors receive only the declared safe vocabulary. Exporter
    outage, queue saturation, cancellation and shutdown stay bounded and cannot
    waive required audit or turn a completed effect into an automatic retry.
15. Interactive release notices never consume workload input; automation has no
    prompt. Disabled/offline checks make no connection, while an explicit local
    minimum version denies sensitive operations before credential or provider use.
16. SDK and native RC/stable artifacts have their own immutable versions and
    independently verified registry/download, installation and provenance receipts.
17. Enterprise access reviews, lifecycle/revocation, audit retention/export,
    tenant isolation and recovery satisfy the [enterprise contract](enterprise-integration.md),
    with clear customer responsibilities and no certification claim.
18. Signed administrative intent, exact candidate/target/revision and independent
    approval survive retries/concurrency; signature confusion, wrong signers,
    replay/rollback/freeze, tenant crossover and authority-change races deny.
19. Default operation is quiet and deterministic without bypassing any baseline
    control. Strict requirements propagate through every interface, descendant,
    cache and recovery path; unsupported assurance and attempted downgrade deny.

## Canonical configuration and lifecycle acceptance

The [configuration-governance contract](configuration-governance.md) and
[lifecycle contract](credential-lifecycle.md) are part of the complete v1
acceptance scope. Implement one typed policy graph with defaults, inherited
locks, deterministic group resolution, explicit override bounds and shared
CLI/API/UI/SDK/MCP authorization, validation, revision/approval and required audit.
No interface-specific policy file or local edit can bypass the effective ceiling.

Qualify optional SCIM provisioning/deprovisioning, reconciliation and stale-event
fencing; supported provider-native rotation/revocation with exact version binding,
partial/uncertain-effect recovery and old-version containment; GitHub scanning
and optional GitGuardian block/warn/authorized-exception decisions with separate
permissions, bounded metadata and freshness rules. Do not derive GitGuardian
identifiers from local secret hashes. Exposure, scanner closure, issuer
revocation and verified rotation have independent evidence. Unsupported provider
mutations remain explicit failures with a scoped owner next step.

Qualify optional workload OIDC/SPIFFE, device-posture/Conditional Access and
external policy-engine profiles against the same canonical authority ceiling.
Required stale/unknown/undefined decisions deny, including OPA HTTP success
without a typed allow. Complete versioned schema/data/API migrations preserve
locks, revocation/exposure history, replay and audit state; fix forward through
reviewed corrected revisions, never revive a permissive old policy or call
partial interface support the finished product.

Test secret-free lifecycle/compromise outcomes through OTLP and SDK observers;
export neither guarantees immutable audit nor waives durable required audit.
Account connections, purchases, secret transmission and live issuer mutation
need independent authorization and qualified capability evidence. Mock fixtures,
documentation and schema work do not establish any of those outcomes.

### Single-file and explicitly composed configuration acceptance

Both self-contained policy files and master policies with explicitly trusted
external department/local includes are required. They resolve to one common
validated contract across CLI/API/SDK/MCP and managed-service boundaries; no
single-physical-file restriction replaces that requirement. Authenticate each
source and version/digest pin, constrain namespace/ownership and transitive
fetch authority, and enforce deterministic typed precedence/conflicts and
multi-membership without weakening enterprise locks or approved override limits.

Test cycles, duplicate/aliased sources and object IDs, traversal/link escape,
bounded loading, pin/trust drift, stale/unavailable inputs, and complete graph
validation before atomic activation. Safe explain/provenance must enforce domain
and viewer permissions, not expose personal content or credential references.
The corporate GitHub + protected personal local acceptance case must prove
personal secrets/keys never upload to corporate services, keys/grants remain
separate, and personal data cannot override corporate locks or infer production
access. Explicit refresh/trust and optional finite offline-cache policy preserve
known denies and govern both new activation and existing-generation validity.
Required migrations fix forward without dropping single-file compatibility.
Implementation tests and revision-bound receipts remain distinct from this design.

## Evidence and remaining work

Actual outcomes are machine-readable audit reports and [qualification receipts](qualification.md), not checkmarks added to this plan. Current live-account, native-host, client and publication status is recorded in `RELEASE.md`. Claims must stay narrower than the observed evidence.

## Agent packaging and documentation acceptance

The portable cquay skill, skill UI metadata and client-plugin instructions share
one capability vocabulary with CLI, SDK and optional MCP. Package only contained,
versioned references and verified runtime connections; no manifest or skill can
create authority. [Agent integration](agent-integrations.md) defines installation,
reload/removal, compatibility, disclosure and client qualification.

Architecture, user/admin, provider/identity, CLI/CI/MCP, enterprise, telemetry,
managed-service and release guides describe this same complete v1 end state.
Schema-backed examples remain executable declarations; integration acceptance
requirements cannot be inserted as unsupported fields. Revision-scoped status,
open gates and publication receipts stay in qualification/handoff records.
