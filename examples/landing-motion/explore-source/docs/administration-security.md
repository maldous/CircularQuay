# Administration security

CircularQuay administration changes who can receive secrets and what a workload
can do. It therefore has a separate security boundary from configuration
download, login, ordinary workload execution and raw disclosure. These v1
requirements apply to the UI, API, CLI, official SDKs and optional management
MCP. Hosting on Netlify, Cloudflare or another platform does not replace them.

This is the security contract for [discovery and management](management-and-discovery.md).
The [security profiles](security-profiles.md) define product-wide enforcement;
[qualification](qualification.md) records implemented controls and exact evidence.

All configuration is client-side: retrieval through the user's network/VPN,
source authentication, composition, validation and local save do not send raw
or resolved policy, source credentials or protected references to a hosted
CircularQuay service, proxy, analytics or logs. The hosted service manages
separate authenticated key-service and age-private-key associations, remote
key wrap/unwrap, permitted age-key retrieval and necessary enrollment, metering
and billing. Its administration never implies remote configuration custody.
Repository publication and optional external identity/policy integrations
retain their separate explicit authority and minimal metadata boundaries.
Synthetic user/account and operator/admin frontend development/previews and a
loopback local client dashboard are authorized. They use mock adapters; they
do not authorize backend integration, operational key access or production
deployment. Browser CORS/file limits require a permitted client-side/native
route, never a hosted configuration proxy.

## Independent trust decisions

| Control | Required decision |
| --- | --- |
| Transport | Authenticate the intended service and enrolled TLS trust; protect every sensitive hop. |
| Identity | Validate the selected issuer's access-token profile for this tenant, client and resource. |
| Sender | Verify the enrolled session/client possession binding and replay controls. |
| Authority | Authorize this actor, operation, object and proposed field changes under current policy. |
| Change integrity | Verify the exact signed intent, required independent approvals and candidate revision. |
| Publication | Verify publisher authority, configuration provenance, target and monotonic accepted state. |
| Durability | Commit state, required audit and replay/approval consumption without a fail-open gap. |

A valid signature establishes possession of an authorized signing key for its
declared purpose. It does not make a malicious policy safe, prove an operator's
intent or make an expired/revoked permission current. A repository download
credential, CI identity or signed build cannot grant policy-publishing authority.

## ADMIN-01: Bootstrap and custody

Initial enrollment authenticates tenant ownership and independently obtained
service/issuer identities and trust roots. It records an approved trust
ceremony; DNS, repository variables, email matching and first contact cannot
create trust. Enrollment credentials are single-purpose, short-lived,
replay-protected and consumed atomically. An enrollment is revocable and cannot
bootstrap another tenant or a more privileged client.

Trust-root, issuer, signer, recovery and administrative-role changes require
separate authority and independent approval. A configuration cannot authorize
its own signer, lower its verification requirements or replace its trust files.
Recovery uses separately protected roots and an explicit quorum ceremony,
invalidates affected sessions/delegations, and records re-enrollment. It cannot
be invoked merely by editing local JSON or presenting a stale backup.

Signing, TLS, session, audit and encryption keys have distinct purposes and
rotation policies. Private administrative signing keys use qualified external
KMS/HSM or approved native protected custody; they are never placed in browser
bundles, model context, generic policy documents or repository artifacts.
Hardware protection, non-exportability and cryptographic-module validation are
separately evidenced properties, not implications of a configuration label.

Authorized hosted enrollment provisions both the profile's KMS association and
its age-private-key association through the selected qualified connection.
Neither account discovery nor OS home/profile credentials supply authority.
Creation uses an idempotent enrolled operation identity; partial or uncertain
provider work is reconciled before activation. Association provisioning is not
an age-key download, offline permission, billable fetch or cross-provider
distributed transaction. Strict enrollment satisfies its independent approval
and durable audit requirements before any protected effect.

KMS wrap/unwrap, age-key retrieval, rotation, disable and recovery have separate
exact grants and independently versioned association state. Where selected,
native provider key policy remains effective after application authorization.
The approved launch software profile has no independent native-KMS permission
boundary: its protected runtime wrapping key stays separate from the database,
but runtime/shared-key compromise can expose all reachable profile keys. It
cannot satisfy a policy requiring non-exportable or independent master custody.
The [launch custody decision](security-profiles.md#authorized-launch-custody)
does not waive any administration, isolation, audit or engineering requirement.
Pre-effect and
pre-release checks reject a revoked or changed association after a provider
wait. The [custody and availability matrix](security-profiles.md#custody-availability-and-release-decisions)
governs every interface; no administrator, support role or alternate route can
convert denial or a trust failure into authorized hybrid fallback.

## ADMIN-02: Strict signatures and parsing

Every accepted artifact and administrative change has one explicitly selected,
versioned signature profile implemented by maintained cryptographic libraries.
The profile fixes its envelope encoding, algorithm/key compatibility, minimum
key strength, permitted signers, signature threshold, signed fields and bounds.
There is no algorithm negotiation from an untrusted object or fallback to an
unsigned, differently encoded or less restrictive profile.

Every enabled signed object, sender constraint, protected storage/backup and
transport endpoint references an immutable cryptographic-profile ID/version.
Its published machine-readable definition fixes applicable signature, hash,
key-wrap, AEAD, KDF, TLS/certificate-validation, randomness, encoding and
canonicalization choices and their parameter/usage limits. Inapplicable
primitives are not implicitly enabled. Changing a profile is an approved trust
change, not a library default or remotely negotiated downgrade. This reference
does not itself assert FIPS or hardware-module validation.

For a [JWS](https://www.rfc-editor.org/rfc/rfc7515.html) profile, the verifier
checks the exact protected header and payload signing input before using
payload claims. It rejects `none`, symmetric/asymmetric algorithm confusion,
unknown critical headers, unsupported encodings, duplicate fields, malformed
UTF-8, trailing documents and over-limit input. Security-relevant header fields
must be protected. A bounded `kid` selects only an already authorized key in
the enrolled purpose/tenant keyset; it is not a file path, URL or database query.
Untrusted `jku`, `x5u`, embedded keys and certificate chains cannot introduce a
trust anchor. A DPoP proof's public JWK has its own explicit profile and must
match the access token's verified key binding.

JSON signed-object and digest profiles specify exact bytes or one named
canonicalization scheme, never a verifier's convenient reserialization.
Where [JCS](https://www.rfc-editor.org/rfc/rfc8785.html) is selected, strict
I-JSON inputs, interoperable numeric bounds and cross-language vectors apply;
no implicit Unicode normalization changes signed content. Outer envelope
parsing is bounded and non-authoritative. Payload interpretation, schema
validation and activation follow successful signature and context validation.

The signed change binds immutable tenant and configuration/resource IDs,
operation, target service/audience, actor/delegation, exact candidate digest,
base revision, new generation/epoch, request ID, schema/profile version,
issued/not-before/expiry times and required approval references. Digests refer
to metadata and protected references, never secret values. No unsigned outer
field may override an authenticated field or select a different target.

Git-backed publication verifies the enrolled repository, authorized SSH signer,
full commit, ref/path and exact materialized policy bytes. Git signature,
publisher authorization and administrative approval are independent checks.
No content filter, hook, submodule, alternate object, executable attachment or
remote dependency can change what was approved. A digest alone proves content
identity, not publisher trust. For a qualified artifact-distribution profile,
[TUF](https://theupdateframework.github.io/specification/latest/) supplies
standard root/target, threshold, expiry and rollback protections rather than
an invented replacement update protocol.

## ADMIN-03: Rotation, rollback and freshness

Key records bind purpose, tenant, algorithm, validity and revocation generation.
Root rotation requires the existing and new authorized thresholds under the
selected protocol; a newly downloaded key cannot approve
itself. Overlap is finite, and key-compromise handling distinguishes historical
verification from permission to make a new change. Unknown keys fail closed;
bounded key refresh cannot become an attacker-controlled fetch loop.

Accepted epochs, revisions and replay state are durable and monotonic. Reject
older generations, divergent successors, expired metadata, mixed revisions,
reused approvals and target substitution. Clients verify freshness as well as
signature validity. A freeze halts changes under a finite signed policy;
unfreezing is a new independently approved action. Last-known-good state is
usable only while its declared scope, authorization and expiry remain valid.
Clock rollback, excessive skew or lost monotonic state cannot extend a lease.
Recovery cannot silently reset the accepted revision or resurrect a revoked key.

## ADMIN-04: Authentication and sender constraints

Management accepts only enrolled access-token profiles for its exact resource.
It validates algorithm/key, issuer, tenant, subject, client, audience, token
type, validity, maximum lifetime and mapped permission claims. An ID token is
not a management bearer credential. Token families have mutually exclusive
validation rules following [JWT best practice](https://www.rfc-editor.org/rfc/rfc8725.html).
Unverified claims may locate only a bounded pre-enrolled verifier, never a new
issuer or JWKS origin. Issuer/JWKS caching, unknown-key refresh and rotation
have finite limits and independently validated trust.

Human administration uses phishing-resistant authentication/step-up through
a qualified issuer or WebAuthn profile. OAuth login uses its selected secure
flow, exact redirect binding, state/nonce and PKCE where required. Machine
administration uses short-lived workload/client credentials. Sender-constrained
profiles use qualified mTLS or [DPoP](https://www.rfc-editor.org/rfc/rfc9449.html),
with exact token/key/resource/method binding and atomic replay state across
replicas. A DPoP proof does not authenticate the request body: signed intent
and transaction-bound approval protect the actual change.

DPoP verification matches the verified access token's `cnf.jkt` to the proof
JWK thumbprint, bounds `iat`, rejects reused `jti`, and validates `htm`, `htu`,
access-token hash and the required server nonce. Replay state is partitioned
by authority, key and resource and is atomic across replicas. URI handling
follows RFC 9449, including its query/fragment exclusion and defined URI
normalization; a broad path alias cannot substitute for the intended resource.
Signed administrative intent separately binds every semantically relevant
request target/body field. Proofs and keys are not logged or reused as generic
client credentials. mTLS verifies the access-token certificate binding and
validated certificate identity at the TLS terminator; any proxy assertion
travels only over an authenticated non-bypassable channel, never a trusted
caller-supplied certificate header.

Session lifetime is bounded by identity, client, policy and grant deadlines.
Refresh cannot widen scopes, outlive a revoked association or silently switch
identity. Logout, issuer revocation, SCIM suspension and local disable have
explicit propagation/outage semantics; a valid self-contained JWT alone does
not promise immediate issuer-side revocation. Tokens and proofs never enter
URLs, ordinary logs, telemetry or upstream provider authentication.

## ADMIN-05: Authorization and independent approval

Every operation resolves its authoritative tenant/object, then checks current
actor, subject, client, resource, fields, action, time and delegation ceiling.
Opaque object IDs and filtered catalogs are not authorization. Read, list,
publish, grant, approve, enroll, recover, export and disclose are separate
permissions. Field-level controls protect custody references and sensitive
associations; generic patch or bulk import cannot bypass them.

Privilege increases, trust changes, secret-release policy, audit weakening,
strict-profile relaxation and recovery require independent approval and
transaction-bound step-up. Approval signs the exact action and revision,
target, approvers and expiry. Two signatures from one effective principal do
not satisfy two-person control. Direct or indirect role/group mutation cannot
enable self-approval; approvers cannot exceed their own current authority.

Immediately before committing a mutation or contacting a provider/releasing a
value, recheck current authorization, approval validity and revocation state.
Use a transaction, revision guard and writer fencing to close check/use races.
Break-glass access is separately enrolled, time-limited, scoped, independently
audited, alerted and subject to post-use review. It never bypasses cryptographic
validation, tenant isolation or required audit.

## ADMIN-06: Safe APIs, UI and agents

Strict schemas reject unknown fields, duplicate keys, ambiguous paths,
unbounded filters and unsupported operations before effects. Request, header,
decompression, nesting, catalog, cursor, concurrency and execution-time budgets
are finite. Database statements are parameterized. Configuration is data, not
shell, template or plugin code; runtime execution still needs its own grant.

Browser sessions use protected server-managed state, secure/HttpOnly cookies
and the selected SameSite policy, CSRF protection, exact allowed origins,
output encoding, restrictive CSP and anti-framing controls. No management
session or recovery material is stored in browser local storage. Origin and
redirect checks do not rely on caller-supplied Host/forwarded headers. Trusted
proxy identity requires an authenticated non-bypassable ingress that strips
untrusted provenance fields. CDN, preview and application caches cannot share
authenticated responses across tenants or principals.

Optional management MCP has its own audience/catalog and metadata-only results.
Tool descriptions, annotations, model instructions and prompt content cannot
grant authority or approve changes. Capabilities are matched to the selected
release's exact schemas; unknown tools fail closed. Enrollment, secret custody
and recovery credentials stay outside model-visible inputs and outputs.

Discovery, provider connections, telemetry destinations and webhooks enforce
scheme/origin, DNS, address and redirect rules on the actual connection and
every redirect. Credentials cannot cross origins. Callback ownership, bounded
delivery, event signatures, replay/deduplication and rotation are independently
verified. Explicit private-network profiles do not permit cloud metadata or
unrelated internal endpoints by default.

## ADMIN-07: Durability, audit and incident response

Tenant isolation covers database rows, key hierarchy, object storage, jobs,
replay/nonces, approvals, caches, exports, logs and backups. High-availability
writers share the required consistent state; a stale replica cannot authorize
a release or consume the same approval twice. ETag/preconditions and scoped
idempotency reject lost updates and mismatched retries. Uncertain outcomes
return an authorized durable operation identifier rather than repeating effects.

Required audit intent, authorization generation, consumed approvals/replay
state and mutation have one atomic or explicitly recoverable durable protocol.
Audit failure denies the protected effect. Finite denial logging and independent
alerting avoid letting unauthenticated traffic exhaust the release audit store.
Records have safe actor/action/revision/reason/correlation fields, integrity
checkpoints and access-controlled retention/export. Tokens, raw provider errors,
secret values and value hashes are prohibited. A hash chain without a separately
protected checkpoint does not prove that a privileged writer kept every event.

Storage encryption binds tenant/object/purpose using qualified managed
primitives. KEK rewrap, DEK replacement and algorithm migration are different
operations. Backups preserve revocation, session, replay, trust and audit state;
restore/failover fencing validates that state before serving. Retention, legal
hold, residency, key retirement and deletion include replicas, exports and
disaster recovery. Cryptographic erasure or memory zeroization is never inferred
from deleting one key reference or waiting for garbage collection.

Incident controls can suspend a tenant/client/signer and halt publication
without granting broader access. Recovery has tested RPO/RTO and explicit
operator ownership. No control promises to recall plaintext or keys already
copied by an authorized consumer or defeat an administrator controlling the
consumer's host.

## ADMIN-08: Customer accounts and financial authority

Customer identity, profile-attributed successful-access usage/history, invoices,
plans, quotas, subscription status and support have separate permissions from
key retrieval and hosting administration. A Cloudflare/Netlify operator API or
dashboard is never an end-customer account portal. Provider administration
credentials remain outside clients and model context. Customer-facing services
use only their actual qualified features and entitlements.

Payment entry uses an established provider-hosted flow without CircularQuay
card custody. Payment/subscription events authenticate their exact provider,
account, tenant and event identity; bounded replay/idempotency controls and
durable reconciliation handle duplicates, reordered events and uncertain
responses. Successful-fetch metering, invoice status and secret authorization
are distinct records. A webhook or browser return cannot self-assert payment,
erase usage, widen a grant or change another tenant's subscription.

CLI/API/SDK and any separately authorized management MCP expose the same typed
account operations and current tenant ceiling. Unattended financial changes
require explicit approved operations, spending/plan bounds, expiry and any
applicable confirmation. Enabling an integration, resolving configuration or
exceeding a quota never authorizes an automatic charge or upgrade. Usage reads,
support access, payment actions and plan changes are separately auditable;
card data, keys, configuration and source credentials stay out of those events.
This design authorizes no live payment setup, purchase or custom UI build.

## ADMIN-09: Restricted operator administration

The remotely hosted operator plane is separate from the customer's account
portal and the client's local configuration workspace. It manages scoped
customer/service metadata, support cases, safe database maintenance,
infrastructure health and secret-free telemetry. Reuse qualified hosting,
payment and monitoring consoles where they meet the required roles and audit
contract. Their operator APIs and credentials are never customer capabilities.
Custom console implementation retains the baseline-first deferral.

Operators authenticate with the required strong identity and scoped role;
support access is tenant-bound, time-limited and audited. Maintenance uses
typed reviewed operations, exact targets, concurrency checks, bounded effects
and reconciliation. Privileged database, key-custody, recovery and role changes
need their separate approval and retained evidence. There is no routine
plaintext age-key display, unrestricted database browser or configuration
upload route. Root-cause investigation uses approved metadata and redacted
diagnostics, not client configuration or source credentials.

Approved support metadata can include the tenant-scoped key identifier, public
age recipient, fingerprint of the public key, version/status and lifecycle
timestamps. It excludes the private key and secret-derived hashes. Activity
and security events retain authenticated source, UTC time, action, result and
request correlation. Geographic context is coarse and purpose-limited, with
explicit access, privacy and retention controls; it is not an implicit location
tracking feed. ISO/IEC 27001 mapping supports evidence and review within the
customer's assessed scope without asserting certification.

An emergency operation has an enrolled recovery purpose, finite authority,
independent approval where required and post-event review; it cannot silently
disable encryption, tenant isolation, signature validation or required audit.
Qualification uses synthetic isolated state and tests denial, audit/storage
failure, concurrent maintenance, restore and uncertain effects before the
operator capability is exposed. Design and test instructions authorize no
production deployment, paid service or live data mutation.

## ADMIN-10: Account roles and scoped delegation

Account administration separates an operation's role permissions from its
tenant, organisation, customer, group, environment, selected fleet, key-metadata
and billing scope. These are server-side account/authorization records, not
uploaded client configuration. An authenticated role alone is insufficient:
the operation and every target must fall within an active explicit assignment.

| Persona | Permitted purpose and default scope |
| --- | --- |
| Superuser administrator | Exactly one protected global administrator account, with explicit global account-control authority and separate audited use. This is not an ordinary support role. |
| Product owner | Approved product/catalog and commercial settings; no customer key or support access implied. |
| Service operator | Scoped service health and reviewed maintenance; no automatic customer business authority. |
| Support personnel | Individually assigned customer/enterprise subsets and allowed support actions, with finite access and no blanket tenant access. |
| Customer account owner | Own account membership, subscription and support permissions within explicitly assigned boundaries. |
| Enterprise administrator | Assigned organisation/group/environment and selected-fleet administration within the authorizing profile. |
| Billing administrator/reviewer | Separately assigned invoice/usage or approved subscription-change permissions; no key retrieval. |
| Auditor | Authorized redacted evidence inspection without mutation or private-key disclosure. |
| Workload or AI agent | Named nonhuman identity and explicit bounded delegation, expiry, actions and resource selection; no inherited human or global authority. |

The superuser account uses the strongest qualified human authentication and
protected recovery custody. Routine work uses scoped identities instead.
Global use, privilege changes and recovery are attributable and audited;
high-impact actions retain their required independent approval. A single global
account does not mean a shared password or exemption from controls. Its role
does not automatically grant raw age-private-key access, source credentials,
payment-card custody or permission to ingest client configuration.

Explicit denies prevail. Applicable assignments combine only under declared
rules; conflicting scopes or ownership deny. Delegation cannot exceed or
outlive the delegator, change tenants, self-approve or grant another global
account. Revocation and role/group changes invalidate affected sessions and
delegations within the qualified freshness bound, with a pre-effect recheck.
UI/API/CLI/SDK and optional management MCP use the same server-side evaluator;
the workload MCP catalog does not become an administration catalog.

Fleet/group/configuration actions are restricted to the authorizing profile
and explicitly selected targets. Client-side connections use the user's own
network/VPN and authenticated provider or enrolled agent capabilities; a
browser cannot reach an unavailable private fleet or bypass CORS. A capability
requiring central client-configuration storage conflicts with the custody
boundary and must be redesigned or explicitly resolved before implementation.
It is not silently implemented by storing configuration on the key service.

No live accounts, roles, fleet actions or privileges are created by this design.

## Qualification evidence

Each ADMIN control maps to implementation, schema enforcement, positive and
adversarial tests, deployment evidence and independent review. Mandatory cases
include signature corruption/confusion, duplicate/ambiguous JSON, signer and
issuer substitution, stale/expired/mixed state, key compromise/rotation,
cross-tenant access, approval bypass, changed bodies, replay across replicas,
concurrent revocation, lost responses, storage/audit outages and backup restore.
Fuzzing, race checks, static analysis, dependency/supply-chain scanning and
SonarQube complement protocol and deployed integration tests. None substitutes
for the others or constitutes a certification by itself.

See [the threat model](threat-model.md), [enterprise integration](enterprise-integration.md),
[assurance](assurance.md) and [release procedures](../RELEASE.md).
