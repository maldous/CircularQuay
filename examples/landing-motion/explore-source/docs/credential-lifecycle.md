# Identity lifecycle, exposure and provider remediation

CircularQuay's complete v1 contract joins provisioning, exposure decisions and
authorized provider-native remediation through the
[one canonical configuration](configuration-governance.md). These are typed
resources in that policy graph, not separate scanner, directory or agent policy
files. [Administration security](administration-security.md) supplies the common
authorization, validation, approval and required-audit boundary for CLI, API,
UI, official SDK and optional MCP. [Qualification](qualification.md) records
actual implementation, supported profiles and revision-bound evidence.

The policy can live in one self-contained file or a master plus explicitly
trusted department/local includes. Both resolve to the same authority and
lifecycle model; source splitting grants no new rights. The
[corporate GitHub and personal local case](configuration-repositories.md#corporate-github-and-personal-local-configuration)
keeps domain grants, decryption keys, caches and viewer-scoped provenance
separate. No personal material uploads to a corporate service or telemetry path.
Explicit version-refresh/trust and optional finite offline-cache policy preserve
locks, known exposure/revocation and complete validated activation.

All configuration for these capabilities stays on the client, including
corporate/personal files and URLs, source credentials, resolution/validation,
explain and local save. No hosted config backend/proxy/analytics/log upload is
permitted. Hosted profile/key association and necessary service metadata are
separate from client policy; optional capabilities do not widen that boundary.

## Provision and deprovision where supported

An explicitly enrolled SCIM connector manages user/group metadata in the
managed service for its exact tenant and directory authority. Declare direction,
supported resources/attributes, authentication reference, stable identifiers,
reconciliation schedule and maximum accepted evidence age. Qualify each
provider's profile; do not assume every IdP, plan or deployment offers SCIM.
The standards define [user/group schemas](https://www.rfc-editor.org/rfc/rfc7643.html)
and [provisioning operations](https://www.rfc-editor.org/rfc/rfc7644.html).

Provisioning does not authenticate a runtime request or grant secret access.
An approved association maps the directory resource to the exact trusted
issuer/subject and tenant. Display names, email reuse, a caller-supplied group
or a provisioning token cannot select a privileged identity. Import only
necessary metadata; no passwords, application secrets or private keys enter
SCIM resources, ordinary policy, model output or diagnostic exports.

Create/update, membership changes, disablement and removal use authenticated,
bounded, idempotent processing with revision/conflict handling and reconciliation.
A supported `active: false` update and a supported delete both trigger the
declared offboarding policy. Preserve deletion/disablement history and ordering
so delayed events, retries or an old backup cannot silently reactivate access.
A new enrollment or reactivation requires current authority; it is not a replay
of an earlier allow.

The canonical policy resolves multiple group associations deterministically:
apply its typed combination rules and ancestor locks, reject unresolved scalar
conflicts, and recompute the affected effective policy. Never choose the last
arriving group, alphabetical group or most permissive result. A group change
cannot enable self-approval, widen a signed grant or override an issuer denial.

Offboarding and compromise fence affected sessions, delegation, approvals,
queued effects and future releases against the current authorization generation.
Recheck after provider waits and before effects. Directory reconciliation,
issuer-token expiry/introspection and local/offline enforcement have separate
freshness bounds. Record their actual propagation limits; directory disablement
cannot recall plaintext, copied age keys or credentials already delivered.

## Workload identity, posture and external decisions

Optional [OIDC workload federation](https://docs.github.com/en/actions/reference/security/oidc)
verifies the enrolled issuer, signature/keys, exact audience, expiry and
subject/claim profile before exchange or resource use. Bind the approved
repository/workflow/ref/environment or platform identity, using the actual
issuer claim format and stable IDs; do not infer it from a runner variable or
copy an unrelated example subject. Untrusted forks and unintended workflows
cannot inherit a privileged identity. Exchange credentials are short-lived,
audience-specific and protected; no default credential-chain or human-login
fallback is introduced. Qualify each CI/cloud/issuer profile independently.

[SPIFFE/SPIRE](https://spiffe.io/docs/latest/spiffe/concepts/) profiles enroll
explicit trust domains/bundles, attestation and selectors, exact SPIFFE IDs,
credential type and rotation behavior. A URI supplied by a caller is not proof
of an SVID. Verify the supported X.509 possession or JWT-SVID audience profile;
one credential type does not establish the other. Bundle federation is an
approved trust change, not blanket cross-domain access. Identity evidence
intersects the canonical tenant/workload/operation policy and never selects an
arbitrary provider, resource or administrative role by itself.

Device posture and Conditional Access use fresh, authenticated evidence from
the configured authority bound to subject/client/device or workload and the
intended resource/session. Reject an arbitrary `compliant` flag, user-agent,
IP address or stale cached allow as proof. Policy names supported signals,
maximum age, challenge/step-up behavior and revocation propagation; a required
unsupported/unknown profile denies. Noninteractive SDK/MCP/CI routes return a
safe challenge/denial rather than simulating compliance or prompting forever.
[User-device compliance](https://learn.microsoft.com/en-us/entra/identity/conditional-access/policy-all-users-device-compliance)
and [workload Conditional Access](https://learn.microsoft.com/en-us/entra/identity/conditional-access/workload-identity)
have different applicability and entitlement constraints. A headless workload
is not automatically a compliant user device, and token validation alone does
not prove a fresh posture decision or immediate revocation.

An optional external policy engine such as
[OPA](https://github.com/open-policy-agent/opa/blob/main/docs/docs/integration.md)
supplies an additional bounded decision. Canonical configuration fixes its
approved endpoint/trust/authentication reference, named decision schema,
policy/bundle identity and revision, allowed metadata inputs, finite budgets,
freshness and unavailable/unknown behavior. Admit only qualified policies with
approved builtins/network behavior; no arbitrary policy upload or remote
credential discovery follows from enabling an evaluator. Send no secrets,
secret-derived hashes, raw tokens or provider/scanner bodies, and constrain its
own decision logs and traces as well as CircularQuay telemetry.

Treat an explicit well-typed allow as only one necessary condition. HTTP 200
alone is not an allow: OPA can return an undefined result without a `result`
field. Missing, malformed, stale, unknown, denied or unavailable required
decisions deny before effects. Bind cached decisions to tenant, normalized
input, actor/session, exact action/resource, policy/bundle revision,
authorization generation and deadline; invalidate on relevant changes. Recheck
current authority after a remote wait and before release/mutation. An external
allow cannot override a lock, issuer denial, exposure block, required approval,
local policy, expiry or audit failure. A decision ID is correlation, not a grant.

## Separate evidence and permissions

| Fact or action | Meaning | Does not establish |
| --- | --- | --- |
| Exposure report | A configured detector observed an occurrence within its visible perimeter. | Complete coverage, exact current credential binding or issuer invalidity. |
| Alert/incident workflow status | The scanner's open, resolved, ignored or dismissal state and its provenance. | Rotation, revocation or permission to resume delivery. |
| Credential validity evidence | A named provider/vendor check, observation time and any manual override. | A permanent guarantee or proof that a new version replaced the old one. |
| Issuer revocation | Verified issuer-side invalidation of the exact credential/version. | Removal of copies or reconfiguration of every consumer. |
| Verified rotation | A distinct authorized replacement, its validation/activation and evidence for the old version's disposition. | Erasure of the old exposure or permission to close a vendor incident. |
| Exception | Approved temporary use of exact still-authorized operations under a declared exposure policy. | Secret access, incident dismissal, trust bypass or overriding issuer revocation. |

Directory provisioning, GitHub alert read, GitGuardian incident read, incident
mutation and provider rotation/revocation each require their own scoped grant.
Use separately protected credential references and selected instance/account,
tenant, repository/installation, provider connection and resource identities.
Reading scanner metadata cannot enroll repositories, enable scanning, connect
accounts, create tokens, purchase a plan or revoke a provider credential.

## GitHub scanning and optional GitGuardian

GitHub secret-scanning integration consumes eligible repository alerts under an
explicit repository/App installation boundary. GitGuardian is an optional
additional source with its own instance/workspace scope and entitlement.
Neither supplies universal public-secret detection. The
[canonical gate](configuration-governance.md#optional-gitguardian-incident-gate)
defines inherited block/warn/authorized-exception decisions and unknown/stale
evidence handling. Apply the same decision rules to each enrolled source;
conflicting evidence cannot clear a known block by choosing the more permissive
scanner. A required strict gate denies affected release when evidence is missing
or too old. A known exposure remains blocked unless a separately approved,
finite, exact-version exception permits the requested operation; warning mode
presents that exception and is not an independent allow path. No warning or
interactive acknowledgement creates permission.

GitHub documents distinct
[alert-read and alert-write permissions](https://docs.github.com/en/rest/secret-scanning/secret-scanning).
Its alert response can contain a `secret` field; read-only access is not a
promise of a secret-free response. Qualified list/get requests must explicitly
set `hide_secret=true`; unsupported hiding or a response with an unhidden secret
is a rejected profile/response, not data to redact and continue using. Isolate
parsing at the connector boundary, bound the response, reject sensitive payloads
before persistence/instrumentation, and emit only the approved metadata DTO. Disable raw request/response capture,
exception-body logging, tracing enrichment and HTTP debug dumps. If the selected
profile cannot meet that boundary, reject it. Do not present full API responses
to an agent or log collector.

GitGuardian's
[internal secret identifier](https://docs.gitguardian.com/internal-monitoring/remediate/investigate-incidents)
is not a local hash-matching protocol. Associate an incident with tenant,
provider/account, opaque credential reference and immutable version/generation
using separately authorized metadata and recorded provenance. Never compute or
send a secret-derived hash to invent that association. A guessed detector/name,
operator assertion or incident number alone is not cryptographic value equality.
Unknown/ambiguous bindings require review and cannot authorize automatic
remediation of a credential.

Generic GitGuardian
[custom webhooks](https://docs.gitguardian.com/platform/configure-alerting/notifiers-integrations/custom-webhook)
can carry matched secrets. They are not enabled in a metadata-only profile.
Admit an event route only after qualifying its payload minimization,
authentication/signature, tenant/version binding, replay/order handling and
failure behavior. Do not substitute an unverified webhook or broad credential
for a denied API route. A no-match result is scoped negative evidence only;
incident closure, validity overrides and source deletion never erase a known
exposure block or prove issuer remediation.

## Authorized provider-native rotation and revocation

Each provider adapter declares its supported operations, resource types,
permissions, verification route and failure semantics. A secret-store read,
new stored version or changed alias is not necessarily rotation at the
credential's issuer. Use the issuer-native API only where its selected profile
supports it and the owner explicitly authorizes the exact credential action.
An unsupported provider returns an explicit unsupported result and the bounded
manual-owner next step; it does not fall back to arbitrary scripts or guessed
revoke endpoints. Read-through remains independently usable.

A remediation intent binds actor, tenant, provider connection, immutable old
version, action, policy revision/generation, approvals, deadline and scoped
idempotency identity. Recheck before the upstream effect and before activating
replacement delivery. Required durable audit records intent and safe outcome;
provider credentials and returned replacement values stay inside protected
runtime/custody paths. An agent gets a safe operation identifier and approved
status, never replacement key bytes or raw provider responses.

For routine rotation, prepare the replacement under the declared issuer
workflow, verify it without disclosure, resolve a complete replacement
generation, activate approved consumers, then verify the old version's required
revocation/expiry. Record partial consumer activation and propagation windows.
For compromise, the authorized incident policy may require immediate issuer
revocation and stopped delivery before replacement, accepting the resulting
outage. A routine overlap window cannot override that emergency policy.

Provider mutation, storage and consumer activation are not one distributed
transaction. Persist stages and reconcile uncertain outcomes before retrying.
A timeout is not proof that a revoke/rotate failed. Record the exact issuer
operation/version outcome where available; never create repeated replacement
credentials after an ambiguous response. Cancellation stops new work within its
supported boundary and does not undo an admitted issuer effect. Recovery cannot
restore a known compromised version or clear its exposure history. Scanner
resolution is a separately authorized workflow after the relevant evidence is
reviewed, not a side effect of a successful local configuration edit.

## Compromise workflow and operational export

1. Authenticate the exposure/offboarding evidence and its exact tenant,
   credential version or subject association; quarantine uncertain bindings.
2. Persist the incident and fence affected future release under the canonical
   policy. A warning accompanies an explicitly approved finite exception
   when the policy permits use of a known exposed version; preserve known blocks during outages, retries, offline use and restore.
3. Execute the separately approved issuer revocation/rotation and session/grant
   response. Reconcile ambiguous effects and record residual propagation limits.
4. Validate and activate a complete authorized replacement; verify old-version
   containment and affected consumers. Keep unresolved effects visible.
5. Review incident disposition and any separately approved scanner update.
   Retain the old exposure, evidence and exceptions without claiming recall.

Export allowlisted operational outcomes through
[OpenTelemetry configuration](operational-configuration.md). Signals may show
safe operation category, outcome, duration, freshness/backlog and exporter health;
they exclude secrets, secret-derived hashes, raw scanner/provider bodies,
credential references and uncontrolled identity/repository labels. Use bounded
attributes/cardinality and opaque correlation where permitted. SDK observers,
CLI, MCP, API and webhook diagnostics share the same minimization boundary.
[OpenTelemetry's sensitive-data guidance](https://opentelemetry.io/docs/security/handling-sensitive-data/)
requires deliberate collection controls; an exporter does not make payloads safe.

OTLP export is best-effort operational delivery, not an immutable audit store,
exactly-once ledger or incident-completion guarantee. Collector receipt does not
prove durable required audit; collector loss does not waive it. Audit integrity,
retention, access control, checkpoints and recovery are separately selected and
qualified under the [enterprise audit contract](enterprise-integration.md#audit-contract).

## Forward migrations and complete delivery

Deliver these capabilities under the same complete-v1 contract, with the
necessary reviewed schema, data and API migrations. Version canonical and
effective policy, directory associations, exposure/credential history,
operation state and decision-cache bindings explicitly. Validate the exact
migration inputs and invariants, fence concurrent writers, and retain protected
progress/evidence so an interrupted migration can resume or reconcile safely.
An incompatible client, incomplete migration or unknown resource version denies
rather than dropping an unsupported safeguard or enabling an ad hoc settings
file. No design-only JSON key is silently accepted by an older runtime.

Fix forward through newly authenticated revisions and corrected migration
steps. Preserve locks, maximum accepted generations, compromised/revoked
version history, directory tombstones, approval/replay state and required audit.
Do not regain availability by reactivating an older permissive policy, resetting
evidence or presenting partial interface support as the finished product.
Existing backup/restore custody remains necessary; it is not a rollback-first
delivery strategy or authority to reuse a compromised credential. Local,
preview and release deployments use the same versioned contracts and retain
their independent trust, synthetic-data and qualification boundaries.

## Acceptance evidence

Qualify SCIM create/update/group conflict/disable/delete/replay/reconciliation,
cross-tenant and stale-directory cases. Exercise lock/default resolution through
every interface, self-approval denial, authorization change during provider waits,
incident-version mismatch, secret-bearing payload rejection, pagination/rate-limit
and freshness failures, warning/exception expiry, and issuer-revoked denial.

Provider tests cover unsupported operations, wrong issuer/account/version,
partial rotation/activation, revoke-before-replace compromise response, duplicate
and uncertain effects, cancellation, recovery and old-version fencing. Synthetic
OTLP collectors must prove that fixtures cannot leak through logs, metrics,
traces, SDK hooks or error paths. Mock success does not establish live-provider
entitlement, SCIM interoperability, account permissions or installed-client
administration parity. Record each exact revision, profile and result separately.

Workload/posture/evaluator fixtures also cover wrong issuer/audience/subject,
trust-domain/SVID mismatch, stale posture, headless unsupported challenges,
external allow against a local deny, undefined/malformed/time-limited decisions,
policy revision drift and authorization changes during evaluation. Migration
fixtures prove interrupted/concurrent progress cannot resurrect removed access,
discard known exposure or bypass a locked requirement. These are necessary
acceptance requirements, not fabricated passing gates or live enrollments.
