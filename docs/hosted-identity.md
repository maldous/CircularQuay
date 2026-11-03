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

For example, Auth0 supports a custom API with the RFC 9068 access-token profile
used by the current verifier; its hosted Free plan has provider-defined usage
and feature limits. Microsoft Entra supports custom-API scopes and application
roles, but its access-token claim profile differs from the current verifier and
requires a separately implemented and qualified profile. Okta's custom
authorization server supports custom APIs; its free Integrator plan is for
testing, while production requires the appropriate API Access Management
entitlement. These are integration paths, not live-provider qualification
receipts. See [Auth0 token profiles](https://auth0.com/docs/get-started/apis/configure-access-token-profile),
[Auth0 plan limits](https://auth0.com/pricing),
[Entra API permissions](https://learn.microsoft.com/en-us/entra/identity-platform/scenario-protected-web-api-expose-scopes),
and [Okta authorization servers](https://developer.okta.com/docs/concepts/auth-servers/).

## Executable support

| Capability | Current behavior |
| --- | --- |
| External issuer | One explicitly configured issuer, same-origin JWKS endpoint and canonical resource audience per policy. |
| Access tokens | Signed RS256/ES256 JWT access tokens with `typ: at+jwt`, issuer, audience, expiry, subject, client, issue time, token ID and scope validation. ID tokens are rejected. Opaque tokens and introspection are unsupported. |
| Client discovery | Protected Resource Metadata and bearer challenges identify the resource and its configured authorization server. The external server owns its own discovery, registration and token endpoints. |
| Resource authorization | Exact issuer-qualified subject, OAuth client, workload, operation and optional key grants. An empty local client grant is not a wildcard for HTTP clients. Secret reads recheck after retrieval; the CLI action backend rechecks grants and its workload/source definition after resolution before launching a registered command. |
| Local revocation | Removing a grant denies subsequent authorization checks, including the post-resolution checks above. It does not terminate an already launched child or recall a completed plaintext disclosure. |
| Issuer revocation | JWT validity is checked locally; there is no token-introspection or revocation-event integration. Disabling an issuer session alone does not establish immediate invalidation of an already issued access token. |
| Hosted management | No organization directory, tenant administration, account-linking service, connection-registration API, SCIM endpoint, consent UI, group-role mapping, delegation or approval service. |
| Audit | Protected local metadata receipts for configured MCP operations. No central immutable administrative audit or SIEM export is provided. |

This is an experimental interface, not a qualified public multi-tenant service.
An HTTP listener and token verifier do not establish SaaS isolation, an
administrative control plane or compatibility with every IdP and MCP client.
Native stores and interactive authentication are described in
[credential boundaries](credential-boundaries.md).

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

These are constraints for a managed hosted service, not implemented account
linking features. The current flat policy is not a substitute for typed tenant,
environment, principal and connection records. Shared hosting requires isolation
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
revocation or create trust between environments. The current executable has no
such exchange path.

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
[qualification matrix](qualification.md) govern support claims. Unsupported
hosted features are not implied by the presence of MCP, TLS, a cloud reader or
an IdP capable of issuing a token.
