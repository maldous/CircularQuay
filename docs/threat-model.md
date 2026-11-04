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

## Review scope

Review policy, endpoint/account selection, input parsing, native process/file boundaries, lifecycle and fallback, per-user MCP, audit outputs, installation, and release trust. Test denial before retrieval and failure paths using fabricated data. The [assurance contract](assurance.md) and [implementation plan](implementation-plan.md) define evidence and outstanding qualification.
