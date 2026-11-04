# One configuration, governed access

CircularQuay uses one client-resolved canonical v1 policy graph for organisational defaults,
groups, environments, workloads, permitted user choices and optional
integrations. CLI, API, UI, SDK and optional MCP operate on the same typed,
versioned resources and authorization rules. A tool-specific settings file is
not a second source of policy authority.

This is the end-state configuration contract. [Qualification](qualification.md)
records actual schema/runtime implementation and evidence. The
[management lifecycle](management-and-discovery.md),
[administration controls](administration-security.md) and
[security profiles](security-profiles.md) govern every change described here.

## One effective policy

The canonical organisation/tenant policy declares defaults, group associations,
named environments, workload bindings, allowed overrides and locked settings.
Each approved revision compiles to one immutable effective runtime policy with
redacted field provenance and a stable policy/authorization generation. The
runtime policy, protected enrollment/custody store and managed deployment
schema have distinct responsibilities; they are not competing operator-policy
files. Credentials remain protected references outside ordinary JSON.

One effective policy can come from a self-contained file or a master with
explicit external includes. Physical file separation preserves department
ownership; it does not create independent policy engines or last-write-wins
authority. The master selects trusted locations, immutable revisions/digests,
namespaces, dependencies, reference bases and permitted overrides. Fragments
cannot introduce trust, grants or executable behavior themselves. Resolution
rejects duplicate identities, conflicts, cycles, excessive graphs and invalid
cross-references before exposing one complete candidate.

## Client-side configuration boundary

All configuration handling stays on the client. The native CLI, official SDK
or a permitted browser client reads files/URLs using the user's own network,
VPN and explicitly selected source credentials, resolves and validates the
complete graph, then saves locally. Neither raw nor resolved configuration,
personal includes, source credentials or protected reference details enter a
CircularQuay hosted backend, server proxy, analytics or logs. Local validation
cannot silently become remote configuration administration.

The hosted key service accepts only the authenticated profile-to-age-key
association and retrieval contract, with necessary enrollment, metering and
billing metadata. It does not store configuration or resolve providers on the
client's behalf. A configuration digest/provenance view is local diagnostic
state, not additional hosted request metadata. Optional external identity or
policy decisions use their separately approved minimal metadata contract;
they do not authorize sending configuration bodies or source credentials.

Browser CORS and file-permission limits are real compatibility boundaries.
A browser uses an explicitly permitted source or approved local native
CLI/SDK bridge; it never routes the configuration through a hosted proxy to
work around those limits. Custom administration screens remain deferred and
unapproved for implementation. This client-side design does not authorize
building or deploying them.

A corporate signed repository and a protected personal local file can
contribute to the same environment without sharing custody. Personal values,
references and raw documents stay local. They cannot override corporate locks
or grants, acquire remote disclosure permissions, be republished to corporate
targets or share corporate decryption credentials. Mixed-domain snapshots are
rejected. Provenance excludes personal origins and document digests from
exported views. Once an authorized native workload receives plaintext, its
behavior still depends on the declared host and workload trust boundary;
configuration validation cannot recall or control a copied value.

The JSON composition contract is closed and explicit. Every include is pinned
by SHA-256; repository inputs also require the existing authenticated,
signed-commit/ancestry contract and an exact commit. Offline repository use is
explicit with a finite cache deadline; an online trust failure never silently
selects a cache. All child data/credential paths bind to the master's declared
reference base, independently of the fragment's storage location. Local
configuration edits validate the complete candidate before atomic replacement;
they never write included documents. This is candidate atomicity, not a claim
of a distributed transaction across repository cache acceptance and runtime
activation. See the [field reference and synthetic example](composition-reference.md)
and [configuration repositories](configuration-repositories.md).

## Hierarchy and locks

| Layer | Responsibility | Limitation |
| --- | --- | --- |
| Organisation/tenant | Trust, integration accounts, baseline restrictions, profile floors, defaults and locked settings. | Cannot override issuer restrictions, release trust or the product's mandatory baseline. |
| Group association | Apply explicitly assigned bounded roles and configuration requirements. | Membership alone does not grant management or secret access. |
| Environment | Select an approved named context, provider/account/resource bindings and session requirements. | A label is not identity or authorization; it cannot inherit a foreign environment's credentials. |
| Workload | Declare exact resources, operations, delivery, lifecycle and tighter constraints. | Cannot widen an ancestor lock, authority ceiling or signed artifact restriction. |
| User/session override | Change explicitly allowlisted non-custody settings within declared bounds and lifetime. | Cannot alter trust, grant access, disable a required gate or silently become central desired state. |

Inheritance is typed, not a recursive merge of arbitrary maps. Denials combine
by union, allowed authority by intersection, required safeguards by union,
time windows by intersection and applicable limits by the stricter bound.
Defaults fill only unconstrained settings. A lower layer may override a
permitted preference under its declared rule; it cannot relax a security floor.

Multiple group memberships have deterministic, documented combination rules.
Conflicting same-level scalar settings fail validation unless the canonical
policy supplies an explicitly approved resolution. Neither map insertion order,
group-name sorting, arrival order nor the last login chooses the winner.
Cycles, duplicate identities, ambiguous associations, unknown groups and
unsatisfiable combinations fail closed. The effective-policy inspection view
shows the contributing revisions and safe reason without values or membership
data the caller cannot inspect.

Locked ancestor settings apply to every descendant and interface, including
local JSON/CLI/environment selection, SDK calls, MCP, imported configuration,
offline state and recovery. A locked value or requirement can change only
through its separately authorized, signed and approved central revision. A
user override binds actor, exact fields, context, base revision, finite expiry
and approval; it is auditable and cannot outlive its parent authority.

## Clean environment sessions

The default session selects the explicitly declared development context, whose
name is operator-defined. An absent or ambiguous context is not a permissive
fallback. Staging, production and other privileged contexts require explicit
selection and fresh authorization under their declared policy. Those names are
examples, not reserved permission categories.

A clean session binds tenant, issuer/subject/client, environment, workload,
effective revision, credential versions, assurance profile and expiry. It
minimizes inherited environment and descriptors and uses separate state,
cache, replay, audit and approval namespaces. Switching context does not reuse
an incompatible identity, token, offline snapshot or prior approval. SDKs and
non-interactive automation receive a safe denial when fresh interaction is
required; they cannot prompt indefinitely or select a more privileged context.

Membership onboarding, movement and offboarding are typed, auditable operations.
IdP/SCIM associations have an authenticated source, stable subject/group IDs,
ordering and bounded propagation. Group removal or user/device/key compromise
invalidates affected sessions, delegations, approvals and future release rights
under the declared revocation profile. Every interface uses the same authority
generation and checks it after provider waits and before effects. Age keys have
no intrinsic expiry: downloaded keys, ciphertext and plaintext cannot be
remotely recalled. Issuer-side rotation/revocation and strict online custody
remain distinct mechanisms.

## Shared identity, credential and reporting lifecycle

The same canonical configuration declares these optional integration profiles;
they do not introduce independent user, policy or credential silos.

| Integration | Scoped capability | Security boundary |
| --- | --- | --- |
| SCIM | Provision, associate, update, suspend and deprovision users/groups where the selected IdP supports the qualified profile. | Authenticated provisioning changes identity associations; it never grants secret or management authority by itself. Membership ownership, sequencing and revocation propagation are explicit. |
| Provider-native lifecycle | Request and verify credential rotation/revocation using the selected provider's actual supported API and approved account/resource scope. | Separate administration grant and protected credential reference; reading a secret never authorizes rotating or revoking it. Unsupported operations deny, and unknown outcomes require reconciliation. |
| GitHub secret-scanning and GitGuardian signals | Ingest qualified, authorized exposure metadata and apply one tenant-and-credential-version block and approved-exception policy. | Separate source accounts/entitlements and verified nonsecret associations; a signal is not authorization or issuer revocation proof. |
| OpenTelemetry operational/event export | Export only approved, bounded, redacted metrics, traces and audit-event metadata to enrolled destinations. | Telemetry delivery is not an immutable or complete security audit. The required durable audit ledger remains independently enforced. |

Lifecycle operations bind immutable tenant, identity/credential reference,
environment, provider account/resource, current version, effective policy and
authority generation. CLI/API/SDK/UI and optional management MCP have the same
typed validation, permissions, approvals, audit and safe outcomes. Provider and
integration credentials never become model-visible administrative arguments.

Rotation creates and verifies a distinct credential version, updates approved
references, performs the declared consumer cutover and independently checks old
version revocation/retention. Repeating an uncertain provider request is not a
rotation strategy. Concurrent changes, provider outage and partial cutover use
durable operation status and signed forward correction; no fallback restores
a compromised, expired or revoked version. Every provider/account and operation
is separately qualified rather than inferred from another adapter's success.

GitHub's selected [secret-scanning API profile](https://docs.github.com/en/rest/secret-scanning/secret-scanning)
is separately qualified for its
repository visibility, plan, App/token permissions and payload privacy. Alert
responses can contain secrets, so raw alert bodies, locations and secret
fields cannot enter the shared incident DTO. Vendor status/reasons, exposure
and validity keep their provenance; neither GitHub nor GitGuardian credential
access implies permission to mutate the other service or the secret issuer.

The GitHub adapter selects an explicitly authorized App installation limited
to exact immutable repository IDs and read-only secret-scanning-alert
permissions. Its qualified list/get requests require `hide_secret=true`; the
documented default is false. Responses with unhidden secrets are rejected, and
alert-location, contents, Actions-secret, bypass, dismissal and administration
routes are outside this profile. Only a bounded approved metadata allowlist is
admitted; free-form comments, source paths/URLs and user/repository dumps are
excluded. Unavailable or unauthorized visibility is unknown evidence, not an
absence of findings. GitHub Enterprise/SaaS API and entitlement differences are
qualified independently; polling is the default until a nonsecret webhook
profile is proven.

A GitHub finding association binds tenant, provider connection/account, opaque
credential reference, immutable credential version/rotation generation, GitHub
instance/installation, immutable repository ID and alert number. Approved
nonsecret metadata establishes the association; neither the secret nor a
secret-derived hash is a match key. GitHub open/resolved state, a `revoked`
resolution reason, dismissal, bypass and removed occurrences remain workflow
signals. They cannot clear the version's block. Only independently verified
issuer replacement/rotation and an approved distinct version can admit the new
credential, or a separately authorized finite exception can permit its exact
still-valid version/actions. A revoked or expired credential is always denied.

## Optional GitGuardian incident gate

The canonical policy can enroll a GitGuardian incident gate for selected
tenants, provider connections, credential versions, environments and workloads.
It is disabled until explicitly configured and authorized. Enablement,
account/source scope, freshness requirements, mandatory exposure blocks and
separately approved exceptions use the same management contracts and inherited
locks. Warnings present an authorized exception, never an independent allow
mode. Optional
does not mean a user can turn off an organisation-required control.

GitGuardian documents an authenticated, scoped
[API](https://docs.gitguardian.com/api-docs/authentication); the adapter uses
the selected instance's qualified authentication profile and protected
credential reference. [Service accounts](https://docs.gitguardian.com/api-docs/service-accounts)
have plan/administrative requirements. Public-monitoring visibility, detector
coverage and API access depend on the actual account entitlement. No universal
free public-secret lookup or complete detection coverage is assumed.

GitGuardian API access, GitHub repository/App permissions and provider
credentials are three separate grants and protected references. Reading
incident metadata never authorizes repository enrollment, account connection,
source modification, secret scanning, incident resolution, token management or
provider revocation. No such connection or transmission follows merely from
loading the configuration. A GitHub integration has its own explicit
repository/installation boundary, owner and least-privilege permissions.

The base incident adapter uses only qualified read-only incident metadata
routes with exactly `incidents:read`; it does not call validity endpoints or
request `secrets:read`. A separate validity capability requires its own
endpoint/field/scope qualification. A route requiring cleartext access,
secret-derived fields or an unproven metadata-only response is incompatible
with this gate and remains disabled; broader scope is never silently added.
The adapter bounds
pagination, response size, time and retry/rate-limit handling, requests the
vendor's privacy mode where supported, and retains only an allowlisted DTO.
It does not call cleartext occurrence/secret endpoints, scan/create-incident
routes or send application credentials, policy bodies or secret-derived
hashes. Raw API responses do not enter logs, telemetry, storage or model output.
Privacy obfuscation is an additional vendor control, not a substitute for
endpoint and field restrictions.

GitGuardian's [custom webhook documentation](https://docs.gitguardian.com/platform/configure-alerting/notifiers-integrations/custom-webhook)
warns that event payloads can contain matched secrets. Generic custom webhooks
are therefore outside this metadata-only profile. An event profile is admitted
only after its nonsecret payload contract, vendor signature, freshness, tenant
binding and replay/deduplication behavior are independently qualified. Installing
a webhook without that evidence is not an acceptable shortcut.

### Binding and release decisions

An approved association binds immutable tenant, provider connection/account,
opaque credential reference, immutable provider version or rotation generation,
GitGuardian instance/workspace/source and incident identifier. Its provenance
states the independently authorized nonsecret metadata and approval used to
establish the association. Operator attestation is not falsely described as a
cryptographic value match. GitGuardian's internal `secret_hash` is not a local
hash lookup protocol and is not used as a CircularQuay value-derived identifier.
Unknown or ambiguous associations require review and cannot target another
credential or produce a clean bill of health.

| Observed evidence | Admission behavior |
| --- | --- |
| Authenticated, in-scope exposure associated with a credential version | Persist the exposure block and deny new use/release/activation of that version under the selected policy. |
| Resolved/ignored incident or removed source occurrence | Preserve the exposure block; workflow closure and deletion are not issuer rotation/revocation proof. |
| Verified issuer invalidity, expiry or revocation | Deny that credential regardless of warning policy or an exposure exception. |
| Independently verified replacement and approved distinct version | Evaluate the new version's own identity, entitlement, exposure and policy evidence. Never clear the old version's history or substitute an alias to the same compromised value. |
| Stale/unavailable evidence or missing account scope | Keep known blocks. Apply the explicitly declared unknown/freshness policy; a required strict gate denies affected new release. |
| No matching incident in the visible perimeter | Treat it as scoped negative evidence only, not proof that a credential was never exposed. |
| Explicit approved warning/exception | Permit only its exact still-authorized version/actions within its deadline; record the warning and required durable audit before effect. |

GitGuardian [incident statuses](https://docs.gitguardian.com/internal-monitoring/detect/incident-statuses)
and [remediation/validity checks](https://docs.gitguardian.com/internal-monitoring/remediate/remediate-incidents)
describe scanner workflow and selected provider capabilities. CircularQuay
records exposure, vendor workflow, vendor/custom validity, issuer revocation
and verified rotation separately. None is silently substituted for another.
Custom validity overrides have explicit provenance and cannot masquerade as
an independent issuer check.

An exposure exception binds incident, tenant, credential version, operations,
owner, safe reason, approval, finite expiry and review obligation. It cannot
close a vendor incident, grant secret access, override issuer revocation,
bypass audit or survive a version/context change. An organisation may lock
exceptions off entirely. A model-visible warning or CLI acknowledgement is not
an exception grant. Unattended operation follows the previously approved policy
and never waits for input to invent permission.

Evidence and block state are durable, tenant-partitioned and reconciled within
the declared freshness bound. Polls/events cannot roll back a newer block or
extend an old safe decision. A blocked version cannot be rereleased through a
cached generation, offline snapshot, MCP read, SDK, retry, alternate target or
restore. Offline behavior preserves the signed policy's finite evidence age
and known blocks; it cannot promise awareness of incidents learned only after
disconnection. Existing consumer copies remain outside recall.

## Schema and qualification integrity

The authoritative v1 schema defines every hierarchy, association, lock,
override, session and integration setting with cross-resource validation. The
effective-policy compiler, runtime and management engine enforce those same
definitions. Published examples contain only implemented schema fields; a
product requirement is not inserted as an ignored extension or ad hoc
environment variable. Cryptographic profiles and security floors cannot be
lowered through the generic configuration editor.

Acceptance includes deterministic multi-group results, conflicts/cycles,
lock/override attempts through every interface, environment/session crossover,
membership changes during provider waits, compromise/offboarding and recovery.
GitGuardian fixtures cover wrong instance/tenant/source/version, uncertain
mapping, sensitive payload rejection, stale/out-of-order evidence, rate limits,
audit/storage failure, workflow closure without rotation, replacement-only
admission, exception expiry and revoked-credential denial. Actual account,
API/plan, privacy, provider-remediation and installed-client evidence are
separate gates; mocks do not establish them.

## Workload, posture and external decision profiles

OIDC workload federation and SPIFFE/SPIRE identify an enrolled workload under
its qualified issuer/trust domain, audience, attestation and selector profile.
They do not automatically grant tenant, environment, provider or secret access.
Device-posture and Conditional Access requirements accept only authenticated,
fresh evidence from the configured authority; a client-supplied health flag,
device name or environment label is insufficient. Unsupported or unavailable
required posture denies rather than silently selecting another login path.

An optional external policy engine, such as OPA, can supply a separately
qualified bounded decision under the canonical policy. Its authenticated
endpoint, signed policy/bundle revision, safe input/output schema, deadlines,
freshness and failure rules are explicit. It receives only approved identity,
resource and context metadata. It cannot see secret values, execute arbitrary
embedded code or widen CircularQuay's tenant, grant, profile, approval or
revocation ceilings. External decisions combine with the local mandatory
checks; stale/unknown or conflicting required decisions deny. Decision
provenance and authority generation are rechecked before protected effects.

Compatibility and data changes move forward through authenticated, reviewed
migrations and corrected revisions. Migration/recovery preserves the strongest
applicable policy, key history, revocation, replay and audit state; it does not
restore access by reactivating an older permissive configuration. Local,
preview and production deployments use the same versioned contract and their
own independently qualified trust and data boundaries.

See [identity and credential workflows](credential-lifecycle.md),
[configuration](configuration.md), [enterprise integration](enterprise-integration.md),
[credential boundaries](credential-boundaries.md) and [release procedures](../RELEASE.md).
