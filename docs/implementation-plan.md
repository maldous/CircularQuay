# Implementation and qualification plan

This is the durable engineering contract, not a completion report. Implementation and qualification use explicitly selected, isolated development environments; verify the actual executor and checkout for each session. Operators install on their own infrastructure and supply host-specific feedback; existing operational hosts and active recovery work are not changed by this implementation task.

V1 is one complete product contract. Work packages describe implementation
dependencies and parallel ownership; they do not define reduced releases or
defer a promised capability. Source checkpoints make changes reviewable. Fixes
move the implementation forward against the same acceptance requirements.
Launch requires the whole declared contract and its revision-bound evidence.

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
existing identity with authorized age-key mappings. Scoped UI/API/CLI/MCP
administration manages allowed principals, clients, windows, modification, and
revocation. Reuse available paid Netlify/Cloudflare identity and management
capabilities first; host only the missing key-specific UI/API within those
platforms. There is no separate email/onboarding suite. Key distribution is
explicitly distinct from protected local custody and remote unlocking. Bundle,
recipient, recovery, and authorization-domain isolation apply to every unlock
path. Private keys, released data keys, and plaintext cannot be recalled;
SOPS/age do not enforce expiry. Commercial packaging and financial scenarios
are assumptions, not chosen prices, traction, or deployment authorization.

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

## Evidence and remaining work

Actual outcomes are machine-readable audit reports and [qualification receipts](qualification.md), not checkmarks added to this plan. Current live-account, native-host, client and publication status is recorded in `RELEASE.md`. Claims must stay narrower than the observed evidence.
