# Threat model

## Assets and trusted components

Assets include provider bootstrap identities, age private keys, application values, configuration authority, audit evidence, native service identities, and release provenance.

The trusted computing base includes the chosen OS, compiler/runtime, upstream cryptographic tools, identity providers, SDK dependencies, protected policy, and authorized publishers. CircularQuay does not protect an authorized value from a compromised endpoint administrator or client that has already received it.

## Invariants

1. A denied request makes no upstream value or credential-issuance call.
2. One logical output has one explicitly declared source authority.
3. Provider values remain literal and cannot introduce another reference or command.
4. Output collisions, including Windows-relevant case collisions, fail.
5. Every required value is resolved and validated before a complete generation activates.
6. Online launch never selects a persistent value cache.
7. Hybrid fallback occurs only for declared availability errors and eligible static values.
8. Denial, invalid TLS, malformed data, trust failure, and expiry cannot authorize fallback.
9. Dynamic/leased credentials cannot become indefinite snapshots.
10. Native publication and MCP disclosure each require their own grant.
11. Bootstrap dependencies are acyclic.
12. A secret value appears in toolkit output only at an intentional authorized delivery/disclosure boundary.

## Threats and controls

| Threat | Control and residual boundary |
| --- | --- |
| Compromised configuration source | Protected connection/activation policy; exact resource bindings; authenticated artifact revision; values never execute. Authorized publishers remain trusted. |
| Confused deputy | Source and destination accounts independently bound; narrow readers and writers; no estate-wide implicit broker. |
| Token exfiltration/SSRF | HTTPS origin/audience pinning, no credential-bearing cross-origin redirects, explicit SOPS Vault allowlist. |
| Identity fallback | Production mechanisms selected explicitly; no accidental broad interactive login. |
| Cross-user MCP access | Per-request verified identity and current grants; filtered discovery; ownership checks; no shared result cache. |
| Prompt-injected tool request | Fixed capability surface; no arbitrary provider URL, shell, grant edit, or bulk export. Model judgment is not authorization. |
| Use-only action extracts credentials | Trusted pinned action plus constrained inputs/results; caller-controlled code requires disclosure-capable authority. |
| Availability outage | Eligible last-known-good static snapshot for offline/hybrid; explicit online failure; no silent partial activation. |
| Audit outage | Mandatory disclosure receipt prevents release when it cannot be recorded; native backend guarantees are stated separately. |
| Local compromise | Dedicated identities and permissions limit blast radius. Root/Administrator defeats endpoint protection. |
| Release compromise | Pinned verified artifacts, reproducible inputs, SBOM/provenance, immutable versions and independent install tests. |

## Bootstrap, SDK, plugin and telemetry threats

| Threat | Control and residual boundary |
| --- | --- |
| Spoofed discovery or CI context | Explicit enrollment, immutable source/digest trust and authenticated workload claims; DNS hints, environment markers and public URLs are never grants. |
| SDK/native substitution or response confusion | Separately verified package and binary/module identity, strict bounded version/DTO parsing and shared error semantics. Version metadata alone is not publisher authentication. |
| Cancel/retry causes duplicate effects | Caller cancellation propagates with bounded cleanup; do not automatically retry effectful calls after an ambiguous result. Status reconciliation cannot undo a completed release. |
| Plugin or tool instruction expands authority | Contained versioned resources, fixed catalog and per-operation checks. Skill text, tool output and caller-selected actor/tenant labels are untrusted as grants. |
| Telemetry exfiltration or exporter outage | Allowlisted bounded signals, protected collector/TLS/credential references and no value-bearing errors/hooks. Best-effort export cannot satisfy or bypass required durable audit. |
| Malicious update feed or downgrade | Advisory notice/check separated from local minimum version and authenticated artifact installation. No automatic code download, execution, prompt or trust-store change. |
| Confused administrative delegation | Exact grantor/recipient/client/tenant/workload/operation/window ceiling and durable metadata audit; grant management does not imply key retrieval. |

The [enterprise contract](enterprise-integration.md), [SDK contract](sdks.md),
[agent-client contract](agent-integrations.md) and
[operational configuration](operational-configuration.md) supply the matching
configuration, failure and acceptance boundaries.

## Governed lifecycle threats

The [canonical governance](configuration-governance.md) and
[lifecycle contracts](credential-lifecycle.md) address directory-event replay or
identity reassignment, permissive multi-group/override resolution, stale posture,
confused workload identity, and an external allow overriding local denial.
Required unknown/stale decisions deny; locks, exact identity/version binding,
current-generation rechecks and forward migrations preserve known denials.

Scanner APIs/webhooks can carry plaintext; bounded allowlisted ingestion and
separate protected credentials prevent raw responses becoming diagnostics.
Incident closure or a guessed secret-hash association cannot trigger automatic
issuer mutation or clear exposure. Partial/uncertain rotation/revocation needs
durable staged reconciliation and separate consumer/issuer evidence. OTel
export does not prove immutable audit or recall released material. These
controls require their own implementation/qualification; naming an integration
is not evidence that its boundary is enforced.

[Configuration composition](configuration-repositories.md#single-file-or-master-and-includes-configuration)
adds include-cycle/alias, namespace escape, path/SSRF, pin drift, precedence and
partial-activation threats. Master-authorized source/namespace/authority limits,
bounded loading, typed conflicts, enterprise locks and full graph validation
precede one atomic activation. Source order or departmental ownership cannot
self-enroll trust. Corporate/personal composition keeps grants/keys/cache
partitioned and prevents personal file/value/provenance uploads through remote
interfaces, explain or telemetry. Stale/unavailable inputs follow explicit
refresh/offline policy and cannot infer production access.

## Disclosure limitations

MCP raw values can enter client/model context, transcripts, or telemetry. No server response annotation can guarantee downstream retention. Invoking an arbitrary application also does not guarantee that application's logs remain secret-free.

Revocation denies future authorized retrieval; it cannot erase copied ciphertext, a released data key, or application memory. Cryptographic rotation, SOPS data-key rotation, and actual issuer credential rotation are separate operations.

Offline access is outside authoritative central per-read auditing. Local receipts are operational evidence, not proof against a hostile machine owner.

## Managed keys and user administration

The [managed-service boundary](managed-key-service.md) separates identity,
mapping administration, key retrieval, and local/remote custody. Cross-domain
mapping confusion, stale authorization, alternate recipient bypass, restore of
revoked grants, and accidental logging of distributed keys are in scope.
Server-side authorization applies to every UI/API/CLI/MCP mutation and release;
Netlify/Cloudflare login or site access alone is not a per-key grant. A copied
private key, document data key, or plaintext defeats later fetch revocation.
Offline expiry assumes trusted local enforcement and does not withstand a
hostile endpoint administrator by cryptographic promise alone.

## Product-wide security profiles and key custody

The [default/strict design](security-profiles.md) applies to every shipped
surface with the same engineering gates. Default quiet operation cannot turn
an absent approval, expired grant or failed verification into permission.
Strict requirements are authenticated, monotonic requirements, not a local
Boolean. The [qualification record](qualification.md) identifies actual
enforcement and outstanding evidence; these invariants do not establish a
qualified strict runtime.

| Threat | Control and residual boundary |
| --- | --- |
| Lower profile through another client or recovery | Bind effective requirements and accepted generations into authenticated context; enforce at every admission/effect/release point and preserve them through restore/failover. Unsupported required controls deny. |
| Automatic enrollment creates unauthorized cloud resources | Use the explicitly selected qualified connection, scoped enrollment authority, idempotency and independent approval/audit where required. Never infer a cloud account or grant from OS profile, environment or first contact. |
| Half-enrollment or uncertain provider/storage outcome | Provision distinct KMS and age associations, conditionally activate only a complete matching profile and reconcile uncertain effects. Association creation is neither key release nor a billable fetch. |
| Profile/artifact/key-context substitution | Use authoritative tenant/profile/association generations, exact provider key/version/algorithm and approved opaque artifact/purpose context. Maintained native KMS enforces its authenticated binding; metadata labels alone do not. |
| Age association bypasses online-only KMS | Do not include a usable offline recipient or release its private key through implicit fallback. Hybrid activation has separate explicit authority, compatible key/artifact versions and a finite offline window. Ordinary multiple recipients are OR. |
| Denial disguised as outage | Only positively classified adapter availability can reach hybrid fallback; auth/trust/integrity/context/version/expiry/audit/cancellation/unknown failures deny. A blocked network is not proof the provider failed. |
| Enrollment/unwrap races revocation | Reauthorize current association and grant state after waits before material release; conditionally commit effects and retain durable audit/idempotency/reconciliation state. No claim of atomic external-provider creation. |
| Migration leaks or weakens custody | Keep inventory client-side, authorize actual key transfer/native crypto separately, verify supported import/export and destination activation, and retain recovery/provenance. Rewrap does not repair copied data keys; historical local copies remain usable under their old keys. |
| Cost or convenience relaxes qualification | Apply the same source/security/coverage/Sonar/artifact gates to both profiles; record plan/entitlement and native/provider assumptions separately. Optional premium services are not mandatory baseline controls. |

KMS custody prevents export of the master key only within the selected
provider's verified assumptions. Returned data keys, age private keys and
plaintext remain copyable. Local administrators, compromised providers,
untrustworthy clocks and authorized consumers remain explicit trust boundaries;
neither signatures nor a strict profile can promise remote recall of copies.

The [authorized launch software profile](security-profiles.md#authorized-launch-custody)
does not claim such non-exportability. Its protected runtime KEK is separate
from the database and wraps independently random profile data keys. A database
dump alone lacks that KEK; an authorized-runtime, deployment-authority or shared
KEK compromise can reach all accessible profile records. Per-profile data keys
do not remove that shared compromise domain. This is the explicitly accepted
launch tradeoff, not unchanged independent KMS/HSM custody. Scoped auth,
tenant isolation, safe audit, bounded abuse controls, rotation/recovery and
release gates remain mandatory. A stronger locked custody requirement rejects
software; migration to a qualified stronger profile remains explicit.

## Review scope

Review policy, endpoint/account selection, input parsing, native process/file boundaries, lifecycle and fallback, per-user MCP, audit outputs, installation, and release trust. Test denial before retrieval and failure paths using fabricated data. The [assurance contract](assurance.md) and [implementation plan](implementation-plan.md) define evidence and outstanding qualification.
