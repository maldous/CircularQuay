# Security profiles and qualification

This page specifies the final v1 security design. It is not an implementation
or qualification report; the [qualification matrix](qualification.md#managed-service-design-status)
records the selected revision's controls and gaps. The design uses present-tense
product behavior as the North Star, without asserting that an unavailable
profile or deployment has passed verification.

CQ applies one security engineering and qualification contract to the
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

The North Star includes both profiles in the complete agreed v1 contract.
Optional integration selection is a deployment choice; it does not reduce the
product's acceptance scope. A development checkpoint with incomplete controls
is not a qualified launch. The published artifact and release notes identify
supported profiles and evidenced limits. Fundamental
authentication, tenant isolation, key protection and bounded abuse controls
remain mandatory; all shipped code retains the same engineering gates.

Profiles apply through [canonical hierarchy and locked controls](configuration-governance.md)
and [lifecycle workflows](credential-lifecycle.md). Optional SCIM or exposure
integrations become mandatory for a descendant when locked by an authorized
ancestor; a local preference cannot disable them. Required freshness failures
deny affected release in strict mode. Warning/exception policy never permits
issuer-revoked credentials, self-approval or bypass of required audit. Provider
mutation requires its own supported capability and grant. OTel export remains
operational delivery, not an immutable-store or required-audit guarantee.

Both profiles preserve client-only configuration ownership: reads over the
user's network/VPN/source credentials, full resolve/validate and local save.
No hosted config store/proxy/log/analytics upload is allowed. Authorized hosted
enrollment provisions two separate profile associations: remote key-service
wrap/unwrap and authenticated age-private-key retrieval. Necessary service
metadata and bounded cryptographic key operations do not authorize configuration
custody. CLI/config disable precedes hosted network. Strict mode cannot require
a config upload or silently turn on a disabled hosted service. Authorized
synthetic frontend development/previews do not authorize real backend
integration or production deployment; the local client dashboard stays loopback.

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
| Initial use | Protected local setup and a synthetic example; first-run hosted key-service association honors both opt-outs before network access and authenticates before registration. Other integrations require enrollment. | Explicit approved enrollment and a signed policy/trust baseline before protected use. |
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

## Authorized launch custody

The initial hosted service uses a **software key service** on one existing paid
Cloudflare or Netlify deployment, within its verified minimum-cost entitlement.
It performs real remote wrapping/unlocking using maintained cryptographic
libraries and the platform's supported primitives. A protected wrapping key is
kept outside the database; independent random per-profile data keys protect the
stored records with a vetted authenticated-encryption/envelope profile. Public
IDs never derive encryption keys. Native secret bindings and environment
variables supply protected runtime inputs; they are not a hardware KMS.

This is an explicitly authorized launch custody tradeoff. The runtime can access
or use its software master key. Compromise of that runtime, deployment authority
or shared wrapping key can expose every reachable profile data key and record.
Random per-profile data keys do not create independent compromise domains under
a shared wrapping key. Separate database encryption, role/tenant checks and
audit still protect their own boundaries; they do not eliminate that risk.
The launch does not claim non-exportable/HSM-backed or independently enforced
per-profile master-key custody.

Authentication, exact scope/tenant authorization, complete key/profile binding,
protected secret delivery, bounded abuse controls, required durable audit,
rotation/recovery and the engineering gates remain mandatory. Preview/local
identities and storage cannot access production wrapping keys. The chosen host
must provide the required durable state and verified secret/runtime controls;
a low price or an existing subscription cannot substitute for qualification.

Independent managed KMS/HSM and qualified Transit connections remain explicit
stronger custody choices and migration destinations. They are not mandatory
paid launch dependencies. A strict policy requiring non-exportability or
independent native custody rejects the launch software profile; a cost setting
cannot waive that locked requirement. Software-custody support is never
advertised as satisfying the former independent-custody assumption unchanged.
No availability-mode default is inferred from the custody choice.

Actual host entitlement and qualified implementation determine the selected
deployment. This design does not select an unverified provider account, deploy
both hosts, authorize a purchase or establish a running service. See the
[qualification record](qualification.md#managed-service-design-status) and
[cost model](managed-key-service.md) for current implementation and assumptions.

## Custody, availability and release decisions

The software envelope profile uses maintained JOSE implementations with fixed
`A256KW` key wrapping and `A256GCM` content encryption. The 256-bit runtime KEK
is explicitly versioned and resolved outside the data store. The maintained
library generates fresh content-encryption keys and nonces; the application
does not author a cipher, KDF or nonce scheme. One flattened JWE JSON recipient
has protected algorithm, encryption, key ID, key-version and enrolled-association
marker headers plus authenticated opaque profile/purpose/artifact context.
The marker is metadata, not hardware or provider attestation. Unknown algorithms,
unprotected overrides, duplicate fields, malformed encodings, context/version
substitution and over-limit inputs deny. This profile wraps 32-byte data keys;
record encryption, random profile-key provisioning and storage are separate
qualified boundaries. Cross-language interoperability and platform/runtime
qualification require actual vectors and tests, not the library names alone.

An **application KMS profile** belongs to the enrolled CQ user or
workload profile. It is not an operating-system `USERPROFILE`, home directory,
cloud CLI profile or ambient credential chain. Its qualified connection binds
provider, account/tenant/project, endpoint and trust, key identity/version,
algorithm, permitted operations, authentication identity and association
generation. Environment and scope names are operator-defined; development and
production are examples, never built-in permission classes.

The optional hosted service provisions both KMS and age associations during
authorized enrollment. It uses an explicitly selected, authorized provider
connection and maintained cryptographic primitives under its declared software,
native managed-KMS or other qualified custody profile. Association
creation is idempotent and independently audited. Partial or uncertain creation
requires reconciliation; it cannot activate a half-enrolled profile. Enrollment
does not release an age private key, approve offline use or charge for a key
fetch that did not occur. No ambient cloud account discovery creates a KMS grant.

KMS wrap/unwrap and age private-key retrieval have distinct action grants,
lifetimes, key versions and revocation state. The hosted service authorizes its
own exact operation; a client-supplied profile ID, valid login or billing role
does not grant key access. Native provider authorization remains a second
boundary. Provider key management, application association management and
plaintext release are separate permissions. Software custody cannot manufacture
an independent provider key-policy boundary. Local age custody remains an
independent selection for explicitly permitted local use.

Wrapped-key records bind the enrolled tenant/profile, association generation,
purpose, algorithm, key version and bounded artifact identity using the selected
provider/library's authenticated-context mechanism. Context contains only approved
nonsecret opaque identifiers: provider audit systems may record it. Raw/resolved
configuration, its graph/provenance digest, source credentials and protected
references remain client-side. A provider that cannot enforce the required
binding is incompatible with that profile; an unverified context field is not
cryptographic protection. Plaintext data keys, private keys, tokens and raw
provider errors never enter ordinary state, logs, metrics or traces.

| Selected availability mode | Successful path | Failure behavior |
| --- | --- | --- |
| Online | Current authorized remote operation using the required KMS/key association. | No local age or snapshot bypass. Unavailability denies the operation. |
| Hybrid | Current authorized remote operation; separately approved local fallback is prepared under its own finite window. | Only a positively classified availability failure can reach fallback. Complete encrypted snapshot, matching scope/key/version, offline permissions, freshness, expiry, required audit and revocation fences must all permit it. |
| Offline | Explicit approved local inputs and custody within their permitted lifetime. | No discovery, online permission inference or automatic widening of the offline window. |

Authentication/authorization denial, disabled or destroyed keys, association or
context mismatch, stale identity/revocation/approval evidence, TLS/trust failure,
malformed or oversized responses, integrity failure, expiry, clock anomalies,
required-audit failure, cancellation and unknown errors deny release. They are
not availability failures. Opaque SOPS exit codes cannot establish a provider
outage. A qualified adapter emits safe typed failures; neither diagnostic-text
matching nor a generic retry policy may turn a denial into fallback.

Strict offline/hybrid use requires its separate signed permission and accepted
host/clock/custody assumptions. It cannot provide the immediate revocation of
an online operation during disconnection. An attacker can block connectivity;
availability fallback therefore represents a bounded policy choice, not proof
that the provider itself failed. No profile recalls a copied plaintext value,
data key or age private key. KMS revocation prevents future authorized provider
operations, not decryption using key material already copied by a consumer.

Flat SOPS recipients are alternatives (OR). A qualified key-group profile
declares the exact groups, backend/key identities and threshold; recipients
within a group remain alternatives. Extra recipients, a substituted group
graph or a lower threshold deny acceptance. A threshold of one is not a quorum.
Unsupported group metadata is rejected rather than silently flattened.

Changing custody follows the [authorized migration procedure](../RELEASE.md#local-to-managed-custody-migration).
Online-only artifacts do not retain an offline recipient simply because the
profile also has an age association. Independent key IDs/windows use separate
data keys or artifacts where required. A mode change cannot make an already
copied local key non-exportable or remove its ability to decrypt old copies.
Rewrap and full re-encryption have different compromise-recovery properties;
provider import/export and hardware restrictions determine what is possible.

## Effective security context

The effective profile is a verified security context, not a user-supplied
Boolean. Its signed envelope binds the profile identifier and revision,
tenant and trusted issuer, enrollment and authority generation, applicable
control identifiers and bounds, compatible client/service capabilities,
not-before/deadline, and the digest of the complete resolved policy graph.
The graph, its digest and configuration provenance remain client-side; they
are not uploaded to the hosted key service. That service independently
authorizes its narrow profile/key operation using its own enrolled authority.
Approvals and exceptions bind that same context, operation and resource
versions; they cannot be replayed against a different revision or tenant.
Trust enrollment is separate from the policy it authenticates. A policy cannot
introduce its own approving key or declare itself enterprise-approved.

Acceptance checks the signature's explicit algorithm/key/purpose, exact
audience and issuer/tenant association, validity and revocation freshness,
supported contract and required capabilities. Durable accepted generations
prevent restoration of an older policy, trust set or profile after restart,
cache use, backup restore or failover. A failure to read that state is a
denial, not a new enrollment. Changing a trust root, lowering a locked
requirement or resetting accepted generations needs separately authorized
recovery with the same audit and separation of duties.

The policy compiler derives requirements once from authenticated inputs.
Each operation retains its resolved configuration snapshot and the associated
security context. The enforcement point verifies current authority before
provider access and rechecks it after waits, before plaintext disclosure,
publication or activation. Service replicas use durable fencing and approval
consumption; a losing or uncertain writer cannot repeat a non-idempotent effect.
Cancellation and expiry prevent new effects. Completed effects with an
uncertain response remain subject to reconciliation and cannot be reported as
if they had never happened.

| Control | Enforcement point | Required adversarial evidence |
| --- | --- | --- |
| Authenticated context | Enrollment, policy compilation, request admission and artifact verification. | Unknown issuer/key/algorithm, wrong tenant/audience, altered graph, stale trust and unsupported contract deny before protected reads. |
| Monotonic requirements | Every interface, descendant, cached/offline context and recovery path. | Local edits, flags, environment, SDK options, alternate endpoints and restored state cannot lower an ancestor's requirements or accepted generation. |
| Exact authority | Resource/action admission, provider boundary and pre-effect recheck. | A read grant cannot publish, rotate, administer or disclose; group changes, expiry and revocation during a provider wait deny the subsequent effect. |
| Custody and isolation | Credential loading, snapshot handling, target delivery and native process creation. | Cross-tenant/profile keys, private/corporate include collisions, redirected paths, hostile children and unsupported native boundaries fail within the declared host assumptions. |
| Approval and step-up | High-impact management intent and one-time commit. | Self-approval, repeated approvers, changed intent, expired proofs, replay and concurrent consumption cannot satisfy quorum. |
| Durable audit | Before an operation whose effective requirements mandate audit. | Unavailable/full storage, invalid checkpoints and failed durable admission block the effect; optional telemetry cannot substitute. |
| Bounded availability | Queues, parsers, provider requests, retries, caches and offline releases. | Oversized input, cyclic includes, stale decisions, clock anomalies, outage and retry storms never widen scope or lifetime. |
| Safe outcomes | CLI/API/SDK/MCP errors, events, metrics and diagnostic export. | Synthetic canary secrets and protected references remain absent; unknown outcomes preserve operation identity for authorized reconciliation. |

Every supported operation has a control mapping and positive/negative tests;
an unmapped protected operation is not eligible for release. The mapping covers
first-run authentication and opt-outs, hosted key retrieval, local workloads,
offline export, configuration CRUD/composition, provider lifecycle,
administration, updates, recovery and each official client boundary. A missing
implementation cannot be represented by accepting the profile name and
skipping the missing checks. Exact implementation status belongs in
[qualification](qualification.md#managed-service-design-status).

### Configuration and capability integrity

The runtime policy references its enrolled security context; the reference
does not confer authority. The separately verified context supplies the
required profile and control bounds. Enrollment, service deployment and
runtime policy have separate schemas and trust inputs. A downloaded policy,
local JSON field or client request cannot sign its own requirements, enroll
its own issuer or assert that a capability is qualified. Unsupported context
versions, missing references and unreadable protected state deny the affected
operation. Invalid strict selection never falls back to default.

Per-workload preferences cannot lower the effective context. In particular,
an `audit_required: false` preference cannot disable strict durable audit;
an advisory update setting cannot disable an enforced minimum version; and
an offline preference cannot create signed offline permission. Required
controls apply equally to enrollment, key release, configuration changes,
administration and recovery, including operations that have no workload field.
Capability evidence binds the actual implementation, module/provider profile,
platform, key version and artifact. A hardware, revocation or audit capability
name without that evidence does not satisfy the requirement.

Profile inspection returns safe effective-control identifiers, accepted
revisions and precise compatibility failures. Unattended clients can use
those results to distinguish required authentication, approval, configuration
and unavailable capability without receiving a secret or bypass instruction.
There is no automatic repair that grants authority, replaces trust, resets
anti-rollback state or weakens a required control.

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
step-up. The first-run hosted key-service association follows its declared
configuration, authentication and CLI/config opt-out checks before network
access. Optional telemetry, unrelated provider enrollment and arbitrary service
discovery do not start merely because an executable is launched. Self-hosted
and offline configurations keep their declared independent boundaries.

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

### Whole-product release admission

One versioned control register covers every shipped executable, SDK, service,
installer, workflow, skill/plugin, schema and enabled integration. Each entry
binds its stable control ID to the threat, applicable runtime profiles,
enforcement location, exact positive/negative tests, platform/provider scope,
artifact identity and independent reviewer. Every protected operation maps to
that register. A missing mapping or unsupported required control blocks that
operation and its advertised release capability.

| Qualification boundary | Mandatory adversarial cases |
| --- | --- |
| Enrollment and identity | Forged issuer/client/tenant, replayed enrollment, ambient credentials, duplicate or partial provisioning, suspension and revocation during waits. |
| Configuration and administration | Signature/algorithm substitution, altered candidate, ancestor-lock downgrade, self-approval, approval replay, conflicting writers and restored old generations. |
| KMS and age | Cross-profile/environment/artifact/context replay, independent key/association revocation, rotation and version pinning, malformed provider response, outage versus denial, and no release after cancellation. |
| Runtime and offline delivery | Partial generations, redirected paths, unsupported host protection, clock rollback, expired or incomplete snapshots, fallback fences, interrupted activation and recovery. |
| Interfaces and diagnostics | Identical authorization through native CLI, Go/TypeScript/Python SDKs and each enabled API/MCP route; secret-canary checks for output, audit, metrics, traces and packaged examples. |
| Service durability and billing | Audit failure, concurrent one-time effects, unknown outcomes, duplicate metering, idempotent retries, tenant crossover, backup/restore and failover without resurrection. |
| Distribution and operations | Tampered archives/signatures/provenance, publisher substitution, installed-package conformance, native upgrade/removal/reboot, provider entitlement and recovery exercises. |

Release admission joins the applicable results for the **same immutable
candidate**. It requires complete source and owned-code coverage, successful
Sonar analysis and gate/readbacks, all other required analyzers, native and
provider evidence, exact artifact consumer verification and independent design
and implementation review. Pending, missing, skipped, cancelled, stale,
unreadable or failed evidence cannot satisfy a required check. A CI workflow
file, badge, successful upload, mock or cross-build is not the corresponding
qualification. Administrative permission gaps are recorded, not bypassed.

Default and strict use the same reviewed implementation and engineering gates.
ASVS 5.0.0 applicability for hosted identity, key release and administration is
assessed against its highest verification level; any inapplicable requirement
has an explicit boundary rationale. Runtime default does not select a lower
verification level. Native and offline properties receive their own threat and
platform evidence because a web standard cannot cover those boundaries alone.
SSDF requirements cover development and vulnerability response for the whole
product. Standards mappings, scanner results and control registers do not by
themselves establish certification.

ISO/IEC 27001 control mapping supports an organisation's security programme;
certification belongs to a separately assessed organisational scope. FIPS or
hardware assurance requires the actual validated module, version, operating
environment and evidence. A strict profile, encrypted file, signed artifact or
passing scanner cannot establish either certification.

The assurance claim always states its adversary and custody assumptions.
Signatures establish authenticity and integrity of accepted inputs; they do
not force a modified client on an attacker-controlled host to execute a check.
A server independently enforces future server operations. Offline enforcement
depends on the qualified host, protected state, clock and key custody named
by its profile. A compromised administrator/kernel or copied plaintext/key
cannot be neutralized by a local profile flag. Such deployments cannot claim
properties that their actual trust boundary cannot enforce.

See [administration security](administration-security.md),
[enterprise integration](enterprise-integration.md), [threat model](threat-model.md),
[assurance gates](assurance.md) and [release procedures](../RELEASE.md).
