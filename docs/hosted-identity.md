# Hosted identity and application connections

CircularQuay's HTTP interface is an OAuth protected MCP resource. Authentication
comes from an external authorization server; CircularQuay decides which exact
workload, operation and key that authenticated caller may access. Provider
credentials remain separately bound to declared source and target identities.
This separation also applies when the authorization server is self-hosted.

Keycloak is an optional implementation of that external role, not a CircularQuay
dependency. An existing IdP can supply the role directly when it can register a
custom API and issue access tokens for that API's audience and permissions. No
additional hosted broker is required for that arrangement. A provider's OIDC
login support alone does not establish custom-API authorization support.

Auth0's RFC 9068 API profile, Keycloak's explicitly configured access-token
profile, Microsoft Entra's v2 custom-API scopes/roles, and a suitably entitled
Okta custom authorization server are integration choices. Select the matching
typed profile and register a dedicated CircularQuay API audience; a login
application's token is not a substitute. Account and publication prerequisites
belong in [RELEASE.md](../RELEASE.md). See the primary
[Auth0 token-profile reference](https://auth0.com/docs/get-started/apis/configure-access-token-profile),
[Entra API permission reference](https://learn.microsoft.com/en-us/entra/identity-platform/scenario-protected-web-api-expose-scopes),
and [Okta authorization-server reference](https://developer.okta.com/docs/concepts/auth-servers/).

## Resource and identity contract

| Capability | Authorization boundary |
| --- | --- |
| External issuer | Named authorities pin exact issuer, same-origin JWKS endpoint, API audience and typed token profile. Public resource URLs are separate from token audiences. Duplicate issuer/audience authorities are rejected. |
| Private CA | A named authority can pin a canonical absolute protected PEM bundle and its SHA-256 for JWKS retrieval. That bundle replaces system roots only for that authority; it is reread and rehashed when the policy loads and when its verifier is created. Provider and CLI custom CA support remain separate integrations. |
| Access tokens | Signed RS256/ES256 access tokens with strict JSON claims, exact issuer/audience, issue/expiry/not-before checks and a stable principal/client. RFC 9068 requires `at+jwt`; Entra v2 validates tenant, object and client GUIDs plus API scopes or roles. ID-token profiles and unsupported sender-proof claims are rejected, except for the explicit DPoP confirmation contract below. |
| Client discovery | Protected Resource Metadata and bearer challenges identify the public resource, its configured authorization servers and supported scopes. The external server owns registration, login and token endpoints. |
| Resource authorization | Operator-defined scope and role requirements intersect exact authority, issuer-qualified principal, client, workload, operation, key and local time-window grants. Missing permissions deny; no development/production labels are reserved. |
| Local revocation | Removing a grant or changing selected authority trust/permissions denies further authorization checks, including checks after resolution and before release/effect. Already launched children and completed disclosures retain their delivered values. |
| Issuer revocation | Locally verified JWTs remain bounded by signed expiry. Immediate issuer-session revocation additionally needs a qualified online revocation/introspection channel; disabling login alone cannot recall an issued bearer token. |
| Managed administration | The optional managed key service provides separately authorized mapping/grant administration and key retrieval. Platform-account roles do not implicitly grant access to application keys. |
| Audit | Durable release-authorization receipts contain identity/operation metadata without plaintext. A receipt precedes release and does not prove delivery to the client. Central retention and administrative audit have their own managed-service boundary. |

The [configuration reference](configuration.md#grants-and-mcp) defines the JSON
fields and exact conjunction rules. The [qualification matrix](qualification.md)
records implementation, native tests, live-provider interoperability and release
evidence separately. Shared hosting also requires the isolation contract below;
a listener and verifier alone do not establish a managed service. Native stores
and interactive authentication are described in
[credential boundaries](credential-boundaries.md).

### Optional DPoP-bound access tokens

A named authority can opt into the version-1 DPoP profile with its
`sender_constraint`; the issuer remains responsible for issuing the access
token. For that authority, it must issue an RFC 9449 `cnf.jkt` confirmation
claim for the client's DPoP public key. CircularQuay accepts the token only as
the `DPoP` authorization scheme with a proof whose embedded JWK thumbprint
matches that claim. The profile permits ES256 with a P-256 key or RS256 with a
2048--8192-bit RSA key, verifies the proof's bounded `jti` and `iat`, exact
method, configured canonical `htu`, access-token hash (`ath`), and JWK
thumbprint, and requires a one-use nonce plus persistent atomic replay
decision. Its nonce/replay binding includes the selected authority, key and
resource and the current authority-policy fingerprint.

An omitted `sender_constraint` selects strict bearer behavior for that named
authority: bearer tokens must not carry `cnf`, and DPoP authorization or proof
headers are rejected. This keeps bearer and sender-constrained authorities
separate rather than accepting either scheme for one authority.

DPoP reduces the value of a stolen access token when the corresponding client
private key remains protected. It cannot recall plaintext or secrets already
disclosed, protect a compromised client private key, or defend against an OS
administrator who can inspect the running client or protected local state.
Issuer-side JWT revocation still has the signed-token expiry bound described
above unless a separately qualified online revocation or introspection channel
is available. CircularQuay does not implement key custody, login, token
issuance, mTLS, or private-key JWT authentication for this profile.

## Managed key access and administration

The optional [managed key service](managed-key-service.md) binds trusted
issuer/subject, authorization domain, client, mapping, operations, and windows.
Its administration UI and API preserve the distinction between mapping/grant
management and private-key retrieval. Trusted IdP scopes, roles, and groups
constrain these operations; an unrecognized issuer or token profile grants
nothing. Service revocation denies new release decisions, while issuer changes
retain their documented token/event propagation bound. Key distribution cannot
invalidate an already copied key. Existing Netlify/Cloudflare identity and
administration capabilities take precedence over building equivalents.

## Identity association

An external identity is the exact verified issuer and subject pair. Issuer
strings retain the matching rules of the protocol; case folding or removing
path/trailing-slash differences must not merge issuers. Email addresses,
usernames, account labels and client-supplied agent names are not stable
identity keys. OpenID Connect specifies the stability of the issuer/subject
pair in [Core section 5.7](https://openid.net/specs/openid-connect-core-1_0.html#ClaimStability).

The hosted authorization boundary distinguishes these associations:

| Association | Required authority |
| --- | --- |
| External login to an organization principal | Explicit organization membership and an approved link to the verified issuer/subject. Linking an existing principal requires proof or administrator-governed recovery; matching email is insufficient. |
| Principal to an application | Registered OAuth client and the caller's current grant. Authentication of a person does not authorize every application that acts for them. |
| Principal/application to an environment | Explicit membership and operation grants for each operator-defined context. Membership in one context grants no authority in another. |
| Environment/workload to a provider connection | Exact declared provider account, tenant/project/namespace, resources and dedicated workload identity reference. |
| Delegated caller to an action | Separate subject and current actor, intended resource, allowed operation, expiry and current policy. A downstream identity is not inferred from a human login. |

The managed service keeps typed tenant, environment, principal and connection
records with explicit ownership and associations. Shared hosting requires isolation
of policy, state, cryptographic material, audit, caches, locks, background jobs
and workload execution as well as request authorization. A user-provided tenant
or environment parameter cannot establish that boundary.

## Authentication and connection ownership

The external IdP/authorization server owns human login, MFA/passkeys, client
registration, consent, token issuance and session lifecycle. SAML federation
can terminate there rather than introducing a second login implementation in
CircularQuay. Browser/native clients use an appropriately registered OAuth
flow, exact redirect binding and PKCE. The authorization server issues access
tokens for the exact CircularQuay resource. The
[MCP authorization specification](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization)
defines resource metadata, discovery and client-registration interoperability;
[RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html) describes OAuth security
requirements. These standards do not qualify an untested integration.

For the cited MCP version, an available pre-registration is used first. Without
a prior relationship, Client ID Metadata Documents are the preferred route when
the authorization server advertises support. Dynamic Client Registration is a
deprecated compatibility fallback, not a reason to open unrestricted
registration. Registration credentials remain bound to the issuing
authorization server.

Machines use separately enrolled non-human identities, with provider-native
workload federation where supported. A CI assertion, cloud identity token or
SPIFFE identity is not automatically an access token for CircularQuay. An
authorized issuer must establish the exact trust and resource binding.
Provisioning and group synchronization are administrative inputs, not automatic
permission grants. Offboarding includes local emergency disable and a defined
revocation bound; asynchronous directory updates cannot supply that guarantee
alone.

Connection records contain ownership, environment, exact upstream identity
references, allowed resources, policy revision and active/disabled/revoked
state. They do not contain passwords, refresh tokens, private keys or bearer
values. Sensitive bootstrap material belongs in its separately protected store.
Creation, modification, linking, approval and removal have distinct audited
administrative permissions. Metadata discovery also requires filtering because
provider accounts and resource names can disclose sensitive structure.

Inbound CircularQuay bearer tokens are never forwarded as provider credentials.
Where on-behalf-of access is supported by an upstream system, a qualified
external authorization server performs a resource-bound exchange with explicit
subject/actor and downscoped authority. [RFC 8693](https://www.rfc-editor.org/rfc/rfc8693.html)
defines token-exchange semantics; implementing it does not by itself propagate
revocation or create trust between environments. An exchange is enabled only
through an explicitly supported, qualified external authorization server.

## High-impact actions and managed operation

Production activation, permission changes and intentional raw reads can require
additional approval under hosted policy. An approval must bind the subject,
actor/client, tenant/environment, connection, exact operation and key aliases,
policy revision, expiry and single-use request identifier. A generic prior MFA
or fingerprint event is not authorization for a different action. Cancellation,
replay, policy change, denial and expiry fail closed.

A hosted capability includes operating requirements beyond authentication:
per-tenant quotas and request budgets, idempotent changes, bounded retries,
auditable configuration history, scoped administrative recovery, backup and
restore, documented availability and revocation bounds, and tested upgrade and
rollback behavior. Tenant data retention, export and deletion have explicit
ownership and privacy rules. Audit records correlate the authenticated subject,
application/actor, environment, connection, decision, policy revision and result
without values, bearer tokens or SDK response bodies.

The [implementation contract](implementation-plan.md) and
[qualification matrix](qualification.md) govern support claims. Each hosted feature requires its own acceptance evidence; MCP, TLS, a cloud
reader and an IdP token do not establish one another's guarantees.
