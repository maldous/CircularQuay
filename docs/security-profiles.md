# Security profiles and qualification

CircularQuay applies one security engineering and qualification contract to the
whole v1 product: native runtime, providers, encrypted offline state, identity,
administration, SDKs, optional MCP, agent packages, telemetry and distribution.
Security profiles select enforceable operational requirements. They do not
change whether a signature, permission or isolation boundary is checked.

The **default** profile keeps everyday use straightforward after explicit setup.
The **strict** profile enforces an enterprise's additional approval, custody,
freshness, audit and deployment requirements. Both use the same authorization
engine and fail closed at protected boundaries. No profile is a claim of
certification or immunity from compromise. Actual implementation and test
evidence is recorded in [qualification](qualification.md).

Profiles apply through [canonical hierarchy and locked controls](configuration-governance.md)
and [lifecycle workflows](credential-lifecycle.md). Optional SCIM or exposure
integrations become mandatory for a descendant when locked by an authorized
ancestor; a local preference cannot disable them. Required freshness failures
deny affected release in strict mode. Warning/exception policy never permits
issuer-revoked credentials, self-approval or bypass of required audit. Provider
mutation requires its own supported capability and grant. OTel export remains
operational delivery, not an immutable-store or required-audit guarantee.

## Safeguards that never become optional

Every profile validates its schemas and supported versions, protects local
state, verifies the selected artifact/signature and TLS trust, authorizes exact
resources and actions before retrieval/effects, honors expiry and separates
tenants and identities. Unknown fields, unsupported capabilities, malformed
signed objects and trust failures cannot be treated as a permissive default.
Provider credentials are separate from delivered application values. Diagnostic
output, telemetry, audit and model-visible metadata exclude secrets and tokens.
Input, network, process, queue and storage work has explicit limits.

An identity provider, hosted service and MCP connection remain optional product
components. An enabled identity-controlled feature must satisfy its declared
identity profile. A customer cannot obtain identity enforcement simply by
setting a mode name, and absence of an IdP is never converted into authenticated
identity. The same rule applies to hardware custody, remote audit and revocation.

## Operational profile contract

| Boundary | Default | Strict |
| --- | --- | --- |
| Initial use | Explicit local setup and a synthetic example; optional integrations remain disabled until enrolled. | Explicit approved enrollment and a signed policy/trust baseline before protected use. |
| Configuration | Validated protected local policy, or authenticated pinned/signed remote policy under its selected trust contract. | Signed, revisioned policy; independently approved trust and privilege changes; durable anti-rollback and freshness enforcement. |
| Identity | Declared local operator authority or an explicitly enabled qualified identity profile. | Named qualified human/workload profiles, bounded session and revocation freshness, phishing-resistant human administration and sender constraints. |
| Secret selection | Exact declared sources, workloads and grants; no ambient credential fallback. | The same checks plus approved provider/account/resource/version selectors and machine-verifiable entitlement to the selected custody/release profile. |
| Provider behavior | Selected adapter and configured native authentication lifecycle with finite retry/fallback rules. | Only qualified adapter/profile combinations, required provider audit where available and explicit failure behavior for unavailable evidence. |
| Offline access | Only an explicitly declared encrypted snapshot and its permitted fallback, scope and deadline. | Online authorization by default; offline use requires separate signed approval, finite freshness/expiry and qualified host/clock/custody assumptions. |
| Disclosure and export | Workload use, raw reads and offline export have separate explicit permissions. | Additional approvals/step-up and tightly bounded export/disclosure; disabled unless the strict policy explicitly permits that boundary. |
| Native delivery | Protected owned files/processes and complete generations; declared application lifecycle. | Qualified native target, protected identity/custody and verified privilege/lifecycle controls; unsupported targets deny protected operation. |
| Administrative changes | Explicit scoped authority, signed change integrity and the mandatory management controls. | Independent approval/quorum and separation of duties for high-impact changes; protected signing custody and transaction-bound step-up. |
| Audit | Required audit is durable before any operation whose policy requires it, including protected management/disclosure; failure denies that operation. | Durable audit required for every protected operation, with separately protected integrity checkpoints, approved retention/export and delivery-failure rules. |
| Observability | Safe local diagnostics; remote export is opt-in, bounded and cannot weaken required audit. | Approved destinations, residency, identity, retention and monitoring controls, with verified redaction and operational health. |
| Release/update trust | Authenticated selected artifacts and explicit update checks; advisory notices never install or consume application input. | Enforced approved publishers/build provenance, minimum supported versions and qualified native/SDK/package identities; air-gap trust and freshness are explicit. |
| Recovery | Protected recovery that preserves expiry, grants and accepted revisions. | Independently approved recovery, tested restore/failover, writer fencing and restoration of revocation/replay/audit state before availability. |

Strict controls are conjunctive requirements, not optional warnings. A target
without a required hardware module, trustworthy clock, native protection,
provider entitlement, durable audit or revocation guarantee is incompatible
with that strict deployment. Validation identifies the missing capability and
denies the affected operation. It does not emulate assurance by accepting a
configuration flag or silently substituting a weaker mechanism.

## Enforcement and downgrade resistance

Profile selection is part of validated, versioned policy. The effective
requirements combine centrally enforced service policy, enrolled client trust,
issuer constraints, local policy and the signed artifact's restrictions. The
result permits only the intersection of authority and satisfies the union of
applicable safeguards. Conflicts deny; a more permissive layer cannot override
a stronger requirement.

The profile and its generation are bound into signed publication, approvals,
session/lease context and effective configuration provenance. Runtime, service,
CLI, SDK, UI, MCP and every descendant worker enforce that context. A child
process, retry, cached catalog, offline snapshot, alternate endpoint or recovery
path cannot remove it. The server independently enforces its policy even when
a caller edits or replaces its client.

Lowering a centrally imposed profile or changing its trust requirements is an
explicit high-impact administrative change with independent approval, audit
and re-enrollment where needed. A CLI argument, environment variable, local
JSON edit, downloaded configuration, SDK option or UI preference cannot perform
that downgrade. There is no production `insecure`, signature-skip, audit-skip
or identity-bypass escape hatch. Isolated synthetic fixtures have separate
trust domains and cannot be promoted by changing their environment label.

Preflight validation evaluates declared requirements before provider contact
or effects and reports safe, machine-readable reasons. An operator can inspect
the effective profile, source revisions, capability compatibility and denied
requirements without exposing credentials. Validation cannot automatically
approve a trust change, relax ACLs or grant access. Corrective administration
uses the same scoped change/approval process as an ordinary configuration edit.

## Smooth unattended operation

Default ease of use means predictable setup and quiet operation, not hidden
authority. Once approved configuration and credentials are enrolled, routine
work runs without repeated confirmation unless its actual policy requires
step-up. Optional telemetry, provider enrollment, identity login and service
discovery do not start merely because an executable is launched.

Non-interactive calls never wait for a browser, biometric input or confirmation.
If interaction or fresh approval is required, they return a stable safe failure
and an authorized remediation route. They do not use a different credential,
continue with stale authorization or downgrade to offline access outside its
declared policy. Release-awareness notices are advisory and do not read stdin;
minimum-version enforcement is a separate pre-effect policy decision.

## Enforced engineering and release qualification

| Area | Required evidence |
| --- | --- |
| Design | Versioned threat model, data/authority flows, misuse cases, cryptographic assumptions and review of every privileged boundary. |
| Contracts | One authoritative schema/API/event source, cross-field integrity checks, compatibility rules and a traceable operation/control map. Unsupported fields fail validation. |
| Implementation | Readable reviewed Go and explicit SDK/adapter boundaries, documented security rationale, least privilege, safe errors and fail-closed lifecycle behavior. |
| Static and supply-chain analysis | Formatting, vet, complexity, Staticcheck, gosec, CodeQL, dependency/secret/license scanning, SBOM/provenance and the strict SonarQube gate. Required findings are resolved, not hidden through mode selection or exclusions. |
| Dynamic verification | Meaningful positive/negative tests, race/fuzz/conformance tests, failure injection, malformed signatures/schema, tenant crossover, expiry/revocation and durability schedules. |
| Interoperability | Qualified native OS/architecture, provider/IdP/SDK/client and installed-package evidence. A mock/emulator or cross-build is not a live provider/native acceptance result. |
| Deployment and recovery | Isolated local development, authenticated preview/prod boundaries, key lifecycle, backup/restore, migration, outage/failover and incident exercises. |
| Independent assurance | Independent review/audit with retained exact source/artifact/tool/scope evidence, unresolved findings and remediation verification. |
| Release | All applicable source/security/coverage/Sonar/artifact gates, immutable RC/stable identity, clean installation and operator handoff for the exact published bytes. |

Every profile uses these engineering gates. Strict-mode code is not the only
code that receives security tests, and default mode does not waive findings,
coverage or release requirements. Controls map to schema enforcement, code,
positive and adversarial tests, platform/integration qualification and operator
evidence. Incomplete evidence remains a failed or unqualified gate.

Qualification explicitly tests attempted profile downgrades through local
configuration, flags/environment, SDKs, MCP, imported policy, caches, failover,
backup restore and child processes. It also tests missing required audit or
hardware, unavailable issuers/collectors, expired approvals, compromised/rotated
signers, cross-tenant requests and malicious discovery. Default-mode tests prove
that quiet/non-interactive behavior never becomes fail-open behavior.

ISO/IEC 27001 control mapping supports an organisation's security programme;
certification belongs to a separately assessed organisational scope. FIPS or
hardware assurance requires the actual validated module, version, operating
environment and evidence. A strict profile, encrypted file, signed artifact or
passing scanner cannot establish either certification.

See [administration security](administration-security.md),
[enterprise integration](enterprise-integration.md), [threat model](threat-model.md),
[assurance gates](assurance.md) and [release procedures](../RELEASE.md).
