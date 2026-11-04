# Administration security

CircularQuay administration changes who can receive secrets and what a workload
can do. It therefore has a separate security boundary from configuration
download, login, ordinary workload execution and raw disclosure. These v1
requirements apply to the UI, API, CLI, official SDKs and optional management
MCP. Hosting on Netlify, Cloudflare or another platform does not replace them.

This is the security contract for [discovery and management](management-and-discovery.md).
The [security profiles](security-profiles.md) define product-wide enforcement;
[qualification](qualification.md) records implemented controls and exact evidence.

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
