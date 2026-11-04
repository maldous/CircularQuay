# Discovery and management

CircularQuay provides one v1 management contract across self-hosted and managed
deployments. Operators, applications and approved agents can find their service,
manage declared access and inspect the resulting decisions. Client enrollment
requires its own grant and interactive or attested trust ceremony; a general
agent grant or tool listing does not authorize enrollment.
The CLI, UI, API, official SDKs and optional MCP interface enforce the same
authority and revision rules. Standalone operation remains independent of a
hosted service, identity provider or MCP connection.

This document defines the product contract. Exact implementation, package and
interoperability evidence belongs in [qualification](qualification.md).
The mandatory trust controls are defined in [administration security](administration-security.md)
and the product-wide [security profiles](security-profiles.md).

## Three discovery boundaries

| Boundary | Information discovered | Trust requirement |
| --- | --- | --- |
| Configuration | Repository/artifact, immutable revision, document digest and workload selection. | Operator-enrolled signer or independently obtained immutable pins; verification precedes parsing, activation and execution. |
| Service | Tenant service/resource, API and schema versions, capability profiles and supported interfaces. | Explicit enrollment binds the service origin, tenant, trust roots and authorized bootstrap route. Network location alone establishes none of these. |
| Authorization | The resource's accepted authorization servers and each issuer's authentication endpoints and supported profiles. | Validate the resource and issuer identities and the selected protocol profile; obtain a token for that exact resource. |

Explicit client choices take precedence over enrolled hints. Without an override,
an existing selected local policy precedes remote discovery. CI variables,
repository files, DNS and cloud metadata are candidate locators, not grants.
Discovery never installs a service, starts a system daemon, changes an issuer or
widens authority implicitly. Interactive first enrollment presents the exact
trust change; unattended enrollment requires preapproved trust material.

HTTP authorization discovery follows [Protected Resource Metadata](https://www.rfc-editor.org/rfc/rfc9728.html)
and [Authorization Server Metadata](https://www.rfc-editor.org/rfc/rfc8414.html)
or the selected OpenID Connect discovery profile. MCP uses its
[authorization specification](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization).
Provider-specific connection profiles remain explicit; Keycloak is one
interoperability target, not a mandatory component.

Metadata retrieval has bounded size, time, depth and caching. Origin changes,
redirects, private-network access, DNS rebinding, issuer substitution, stale
metadata and signing-key rotation follow the enrolled trust policy. Private
developer services require explicit local/private-network enrollment. A
discovery response cannot supply executable commands or override these checks.

Enrollment is a typed, revocable resource with an immutable opaque identifier.
It binds tenant, canonical service origin, resource audience, discovery trust
anchor, permitted origins/redirects, issuer/profile, client registration,
protected credential reference and sender-constraint key. Issued/expiry,
last-used, rotation and revocation generations have explicit semantics.
Disabling, transferring, offboarding and deleting an enrollment includes client,
device and agent deregistration and the required retention decision. Recovery
and enrollment custody ceremonies do not pass credentials through model-visible
management output. There is no implicit trust on first use.

Signed discovery binds service, tenant, audience, schema/capability versions,
signer, validity window and anti-replay generation. Cache keys include the
enrollment and authorization context; maximum TTL, permitted staleness and
rotation overlap are finite. TLS, exact redirect handling and the signed-object
profile are enforced independently. A new issuer key or registration document
does not silently replace the enrolled trust root.

The signed enrollment/service-discovery object constrains accepted resource
and authorization-server metadata. RFC 9728/8414 and OIDC documents are
separately retrieved and validated under their standard TLS/issuer/origin
rules; they are not required to carry a CircularQuay signature. They cannot
introduce an unenrolled issuer, root, audience, tenant, registration or redirect.

## One typed administration model

The versioned API and JSON Schemas define resource identities, fields,
validation, state transitions, errors and permission requirements. CLI and SDK
methods, UI forms and MCP tools map to those same operations. Interface parity
means the same authorized lifecycle is automatable; it does not expose raw
custody material to every interface.

| Resource family | Complete lifecycle |
| --- | --- |
| Tenants and service enrollment | Create/enroll, inspect, suspend, transfer under approval, offboard and verify deletion/retention. |
| Users, groups, clients and workloads | Associate verified identities, inspect effective associations, update membership/mappings, disable and remove. Issuer and subject identity remain distinct from display names and email. |
| Provider connections and secret references | Declare account/project/namespace, authenticate the selected machine profile, validate metadata, update, disable and remove. Provider administration exists only where the selected adapter explicitly supports it. |
| Policy and configuration revisions | Read/list, create, validate, diff, simulate, approve, publish and activate an immutable revision; inspect effective provenance and reconcile observed state. Corrections move forward through a new revision. |
| Grants and associations | Create/list/read, constrain/update, expire and revoke permissions for exact principals, workloads, resources and operations. Environment labels and scopes are operator-defined. |
| Delegations and approvals | Request, approve/deny, inspect, expire, cancel and revoke; retain the original actor, recipient and bounded authority chain. |
| Sessions, leases and operations | Inspect status, renew only within policy, revoke/cancel and reconcile an uncertain result using its operation identifier. |
| Audit, telemetry and integration settings | Configure authorized destinations, retention, event subscriptions and limits; inspect delivery health and export permitted evidence. |

Authorized operators can explain an effective decision, simulate a proposed
policy and export access-review evidence without retrieving a secret. These
diagnostics apply their own visibility rules and do not reveal another tenant's
resources or associations.

Import/export and bulk operations declare their scope, validation and atomicity.
An interrupted or partially rejected batch cannot be reported as complete.
Sensitive bootstrap enrollment, key import, recovery and credential rotation use
dedicated custody operations and protected references. Generic configuration
CRUD, browser bundles, model-visible tool results and diagnostic exports do not
carry bootstrap keys or provider credential values.

Connection custody references are opaque, typed and bound to an owning provider
account, allowed resource selector and rotation generation. Use, read and
update are separate permissions. Metadata validation does not retrieve values
unless that is an explicitly authorized adapter operation. Connection checking,
provider resource discovery and provider administration are distinct actions.

A local `config` write edits the selected local document. A remote mutation
requires an explicitly selected service, tenant, identity and management
operation. Missing remote context fails; it never falls back to a local write
or silently promotes a local edit into a server change.

Remote desired state, observed service state, the effective runtime revision and
a local cached/offline snapshot are separate objects. Reconciliation records
their provenance, precedence, drift and conflicts; it cannot overwrite a newer
revision or widen local/central authority by choosing the more permissive copy.

## Identity and delegated authority

The management service verifies issuer, tenant, subject, client, audience,
operation and resource for every decision. Access is deny-by-default. Permission
to run a workload, read a secret, alter a grant, administer a tenant, publish a
configuration and enroll an integration are independent permissions.

Selected enterprise profiles support workforce and service identities,
workload federation, group/attribute mappings and joiner/mover/leaver events.
[SCIM](https://www.rfc-editor.org/rfc/rfc7644.html) is an optional provisioning
profile; provisioning an identity does not itself grant secret access.
Authentication source-of-truth, mapping ownership and deprovisioning behavior
are explicit. An assertion from an untrusted issuer or a changed email address
cannot attach itself to an existing principal.

Provisioning identifiers have declared ownership and immutability. Signed or
authenticated lifecycle events bind the intended tenant/audience and have
sequencing, deduplication and conflict rules. Group removal, suspension and
deprovisioning invalidate their dependent authority within the declared
propagation bound. A separately authorized local emergency disable takes
precedence. Provisioning and issuer-metadata changes cannot create a management
role or workload grant by themselves.

Interactive clients use the enrolled issuer's supported secure login profile;
automation uses a declared machine/workload profile. Client registration
supports explicitly selected preregistration and qualified metadata-based
profiles, with compatibility registration where required. No flow requires a
user's password to be copied into CircularQuay. The issuer remains responsible
for token issuance; CircularQuay does not need to become a general-purpose IdP.

Delegation attenuates the intersection of the grantor's current authority,
recipient constraints, target resource and requested operation. Time windows,
maximum lifetime, delegation depth, audience, sender constraints and approval
requirements are policy-defined. Actor and subject remain separately auditable.
An optional [OAuth token-exchange profile](https://www.rfc-editor.org/rfc/rfc8693.html)
uses a qualified authorization server and preserves those semantics; inbound
CircularQuay tokens are never reused as provider credentials.

High-impact operations support separation of duties, approval quorum,
transaction-bound step-up authentication and tightly scoped emergency access.
Approval binds the exact proposed revision/action, expires and cannot authorize
a changed payload. An approver or delegate cannot authorize its own privilege
increase merely by editing a role or supplying an agent label.
Delegation and approval IDs retain the original authority ceiling, current
revocation generation and a safe reason classification. Current authority is
rechecked at commit and immediately before provider contact or release.
Emergency access has a named owner, finite duration and mandatory post-use
review; indirect group/role changes cannot circumvent separation of duties.

## Revocation, expiry and consistency

Online release and management decisions use the selected revocation profile:
local policy/grant generation, authenticated issuer events and/or
[token introspection](https://www.rfc-editor.org/rfc/rfc7662.html).
The contract declares maximum propagation delay, cache freshness, outage
behavior and replica consistency. A logout or provisioning deletion is not
treated as proof that every previously issued token is already invalid.

Revocation invalidates affected delegations, approvals, sessions, renewal and
cached authorization decisions according to that profile. Recovery restores
deny/revocation, replay and audit state before admitting new release decisions.
Discovery and tool-catalog caches are partitioned by authorization context and
invalidated when their authority changes.

Approved offline operation retains its declared deadline and local enforcement
assumptions. Central revocation does not recall a disclosed value, distributed
private/data key or running application's copy, and cannot guarantee that a
hostile endpoint administrator obeys local expiry. Issuer-side rotation and
strict online custody are separate policy choices.

## SDK and MCP capability profiles

Services advertise versioned workload and management capabilities with exact
schemas, resource budgets and required permissions. The official Go, TypeScript
and Python SDKs expose typed operations for the selected profile, including
pagination, cancellation, safe errors, revision conflicts and operation status.
A caller selects its local, workload-MCP or remote-management boundary
explicitly. Unknown or incompatible profiles fail closed.

Management MCP has a distinct endpoint, resource audience and catalog from the
bounded workload/disclosure profile. Enabling administration does not enlarge
that profile's four-tool catalog or weaken its response validation. Both
profiles can be served by the same deployment while retaining separate grants,
token audiences, SDK entry points and resource budgets.

MCP catalog handling follows the selected
[tool discovery protocol](https://modelcontextprotocol.io/specification/2026-07-28/server/tools).
Management catalogs support bounded pages, total tools/bytes, cursor lifetime,
deterministic ordering and authorization-aware change notification. Clients do
not execute unknown tool names or infer permission from a listed tool,
description, read-only hint or approval annotation. The server reauthorizes
every invocation using its actual arguments, even after a cached listing.
Management results apply field-level metadata visibility, including resource
associations that could reveal protected structure. They never return
enrollment credentials, bootstrap material or custody keys to a model.

Management operations expose metadata and constrained actions. Raw disclosure
remains an explicitly separate interface and grant. Arbitrary shell execution,
opaque administrative scripts and universal provider-token passthrough are not
substitutes for typed management methods. Confused-deputy and token-audience
controls follow [OAuth security best practice](https://www.rfc-editor.org/rfc/rfc9700.html)
and [resource indicators](https://www.rfc-editor.org/rfc/rfc8707.html).

## Safe changes and durable operation

Every mutation validates the complete candidate and its cross-resource
constraints. Revision/ETag preconditions prevent lost updates. Idempotency keys
are bound to tenant, actor, operation and request digest; a different request
cannot reuse a previous success. Potentially effectful actions are not
automatically replayed after an uncertain response. Long-running operations
provide durable status and reconciliation; cancellation does not imply undo.

The management API defines immutable resource IDs, server-assigned revisions,
create/replace/patch/delete semantics and explicit preconditions, including
create-if-absent. Versions, capability negotiation, compatibility and
deprecation are published together. Pagination has stable ordering and a
declared snapshot-consistency model; opaque cursors bind actor, tenant, filters,
revision and finite expiry. Idempotency identifiers use cryptographic
randomness, finite retention and authorization-checked response replay.
Request/correlation IDs do not contain user payloads. Operation-status access
is authorized independently, and terminal outcomes survive retries/restarts.
No secret/token/value hash is used as an identifier or idempotency digest;
management bodies contain protected references, not custody values.

Authorization, required audit intent and state mutation share a durable
transaction or an explicitly specified outbox/recovery protocol. A grant or
protected release cannot become effective without the required audit record.
Audit includes actor/delegation, scope, decision, revision, reason and operation
correlation without secret values, tokens, value hashes or raw provider errors.
Tamper evidence, retention, authorized export, residency and delivery-failure
behavior are separately configured and tested.

Signed/authenticated webhooks have bounded retries, replay protection,
deduplication, ordering/generation rules and a dead-letter/reconciliation route.
Subscriptions verify callback ownership, enforce outbound destination controls,
rotate scoped signing keys and authorize each versioned event delivery. Event
retention and replay access are explicit. Audit/backup policies define tenant
residency, encryption/key ownership, legal holds, deletion order and authorized
export provenance; restoring a backup cannot restore revoked authority.
Rate limits, quotas and bounded filtering/pagination apply per tenant,
principal, resource and operation. Unknown resource existence is not disclosed
across tenant boundaries. Metrics and traces use bounded safe labels and do not
replace durable security audit.

Local developer and production managed-service instances use the same schemas,
authorization engine, migrations and API contracts. Deployment profiles declare
durable storage, writer fencing, replica consistency, backup/restore, RPO/RTO,
health/readiness, shutdown and compatibility boundaries. Netlify, Cloudflare
and other hosting adapters preserve these semantics rather than translating
eventually consistent caches into an immediate-revocation promise.

## Configuration and acceptance

Runtime policy, client enrollment and managed-service deployment have distinct
version-one schema responsibilities. Every enabled profile exposes its issuer,
trust, resource selectors, scope/association rules, time and freshness limits,
approval/delegation policy, credential references, audit and recovery settings.
Managed-only settings do not become silently ignored fields in local runtime
policy. No enabled feature depends on an undocumented environment variable or
hard-coded development/production role. Unknown fields and incompatible
combinations fail validation before effects.
The version-one management API, resource/error/event schemas and operation map
are authoritative for remote administration; they do not overload the runtime
policy document or the local enrollment store. All three language SDKs and
optional management MCP use those same typed contracts.

The API/schema, CLI, SDK, UI and MCP operation map has one traceable contract.
Acceptance covers positive operations and tenant crossover, scope escalation,
stale revisions, replay, partial failures, expiry, revocation races, issuer/key
rotation, lost responses, audit/storage failure, malicious discovery,
schema/capability mismatch, quotas and recovery. Synthetic fixtures run locally
and in CI; provider, IdP, host and installed-client claims additionally require
their own qualified interoperability evidence.

See [managed service](managed-key-service.md),
[enterprise integration](enterprise-integration.md),
[configuration repositories](configuration-repositories.md),
[agent integration](agent-integrations.md) and
[publisher/operator procedures](../RELEASE.md).
