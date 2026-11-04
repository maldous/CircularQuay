# CircularQuay integrations

CircularQuay v1 is an integration layer. It reads exactly declared values from
existing systems, applies a workload-scoped policy, and delivers a complete
generation through an approved local or hosted boundary. It does not become a
vault, identity provider, universal credential broker, or general automation
platform merely by connecting to one.

This inventory records the integration contracts that make the v1 product
portable and operable. An integration is adopted only when it has a concrete
benefit, an explicit authority boundary, complete versioned configuration,
denial and lifecycle tests, and an operational owner. A product name, a login
button, or an accepted JSON field is not an integration. The schema, validator,
runtime and tests change together; an unsupported field is rejected rather
than accepted and ignored.

The [qualification matrix](qualification.md) is the source for implementation
and evidence status. This document intentionally states the v1 end-state
contract, not a claim that every adapter is already available or qualified.

## Core provider read-through

| Integration | V1 contract | Security and configuration boundary | Verification boundary |
| --- | --- | --- | --- |
| AWS Secrets Manager and SSM Parameter Store | Read an exact declared ARN or parameter identity and return a declared value only to the approved workload generation. | The account, region, canonical resource identity, selector/version and provider credential type are policy fields. The returned resource identity is verified before value selection; no ambient CLI login or inferred account is accepted. | Synthetic SDK responses prove positive neighbours and substituted/missing identities fail before release; live-provider receipt remains separate. |
| Azure Key Vault secrets | Read one exact vault, secret and declared version/lifecycle identity. | Tenant, vault origin, secret path and version are explicit. The returned HTTPS identity must match the canonical configured resource; keys and secrets stay distinct. | Synthetic transport cases cover malformed and substituted IDs, DNS/port normalisation and pre-credential rejection. |
| Google Secret Manager | Read one exact project, secret and version identity. | Project ID/number aliases and unpinned versions are not inferred. Returned project/secret base, concrete version and integrity evidence are bound before field selection. | Synthetic protobuf/SDK cases prove wrong or absent identity cannot reach a value release. |
| SOPS/age, OpenBao Transit and approved encrypted snapshots | Supply offline or hybrid encrypted material using a separately authorized recipient, KMS/Transit or snapshot profile. | Encryption, local custody, provider read authority, snapshot export and remote key release are distinct grants. An age recipient or data key retained by a client cannot be retroactively revoked. | Artifact, recipient, fallback, expiry and recovery tests are required per profile. See [architecture](architecture.md) and [managed-key custody](managed-key-service.md#distribution-local-custody-and-remote-unlocking). |

Provider adapters expose only their stated read-through capability. They do not
turn provider authentication into user authorization, a secret source into a
key-management service, or a destination snapshot into a provider write-back
path. The detailed source contract is in [providers.md](providers.md).

Use the [cloud-provider validation matrix](integrations/cloud-provider-validation.md)
for AWS/LocalStack, Azure/Azurite and Key Vault, Google Secret Manager, provider
SDKs, workload federation, and separate local, developer-environment and
production acceptance. [Keycloak and OpenBao fixtures](integrations/identity-and-provider-fixtures.md)
define the synthetic issuer and KV test contracts. Local emulators, hosted
deployment adapters and the real secret provider each require their own evidence.

## Identity, authorization and workload trust

CircularQuay is a protected resource. It uses an existing authorization server
when that server can issue an access token for a registered CircularQuay API;
it does not require Keycloak, Cloudflare Access, Netlify Identity, or any paid
broker. An OpenID Connect login alone is insufficient.

| Capability | V1 contract | Required configuration and enforcement |
| --- | --- | --- |
| OAuth 2.0/OIDC access tokens | A named issuer and typed access-token profile authenticate a caller for the registered CircularQuay resource. RFC 9068 access-token JWT profiles are supported where their issuer-specific contract is configured. | Pin trusted issuer discovery/JWKS, algorithm/key selection, audience/resource, token type, issuer, time claims, subject and client/authorized-party bindings. An ID token, a token for another API, or caller-provided key URL is denied. [RFC 9068](https://www.rfc-editor.org/rfc/rfc9068) defines the interoperable JWT access-token profile. |
| Scope, role and claim mapping | The policy maps verified claims to exact operations, workloads, source connections and managed-key mappings. | Claim names, value matching, issuer and client association are typed policy, never an implicit `dev`/`prod` convention. Unknown or ambiguous claims grant nothing. |
| DPoP and mutual TLS sender constraints | A named authority can require a sender-constrained access credential for remote API/MCP use. | DPoP validates proof signature, `htu`, `htm`, access-token `cnf.jkt`, bounded age, replay state and nonce behavior; mTLS validates the configured client certificate and token binding where the selected profile supports it. Both are optional profiles, not a global assumption. See [MCP operations](operators/mcp.md#optional-dpop-authority-profile). |
| Token exchange and introspection | Where an authorization server offers them, exchange and introspection are optional named authority capabilities for a configured resource and grant. | The client authentication method, requested audience/resource, actor/subject-token rules, issuer endpoint and cache/revocation bound are explicit. Introspection never silently changes a local JWT profile into an unbounded network dependency. [RFC 8693](https://www.rfc-editor.org/rfc/rfc8693) and [RFC 7662](https://www.rfc-editor.org/rfc/rfc7662) define these different protocols. |
| SCIM and directory lifecycle | SCIM can provision or remove *managed-service users and group metadata* from an enterprise directory. | SCIM provisioning is not an access token, runtime secret-provider identity, or per-request authorization decision. Its tenant, connector credential, group-to-grant mappings, reconciliation cadence and deprovisioning behavior are separately configured and audited. [RFC 7644](https://www.rfc-editor.org/rfc/rfc7644) defines the protocol. |
| SPIFFE/SPIRE workload identity | SPIFFE/SPIRE can complement a human/authorization-server identity with workload mTLS identity. | Trust domain, attestation, selector, SPIFFE ID, workload association and certificate rotation are explicit. A valid workload identity proves enrollment; it does not select a tenant, mapping, secret, or operation. [SPIFFE](https://spiffe.io/docs/latest/spiffe-about/overview/) supplies workload identity, not an application permission model. |

The effective permission is the intersection of verified identity, named client,
resource audience, current grant, operation, workload association and active
window. Revocation blocks future decisions only at the configured freshness
bound; it cannot recall a token, private key, plaintext or data key already
released. [Hosted identity](hosted-identity.md) specifies the resource-server
boundary and [managed-key service](managed-key-service.md) specifies the
identity-to-key mapping boundary.

## CI, repository and configuration bootstrap

CI is a workload-identity integration, not a blanket permission for every
pipeline or repository. Each trust relationship binds issuer, repository or
project identity, immutable workload identity, audience, allowed ref and
environment/protection context where available.

| Integration | V1 contract | Guardrails and tests |
| --- | --- | --- |
| GitHub Actions, GitLab CI/CD and Azure Pipelines | A selected CI platform presents a short-lived job assertion to a configured authority; the resulting grant authorizes only the declared workload and operation. | Bind stable repository/project IDs rather than display names, approved workflow/pipeline identity, audience, ref/environment rules and bounded expiry. Do not grant from an environment variable, branch name alone, or a generic CI marker. Synthetic signed assertions cover wrong issuer/audience/repository/ref/subject and expired credentials. |
| Git repository bootstrap | `cquay bootstrap` obtains a public or separately authenticated repository/tree configuration only when its exact repository, immutable revision/tree, signature policy and declared path are approved. | Downloaded configuration cannot introduce arbitrary executables, credential locations, endpoints or grants. Signed repository verification authenticates the configured artifact; it does not authenticate mutable provider values. See [Git and CI setup](configuration-repositories.md). |
| GitHub, GitLab and Azure configuration discovery | A CI runner or approved bootstrap locator may discover a declared configuration reference from its platform context. | Discovery is an input-selection convenience. It remains subject to the same explicit repository/tree/path pinning, signature/identity validation and workload grant as a supplied URL. Missing or ambiguous discovery fails rather than guessing. |
| Release and supply chain | Release candidates and stable releases are immutable, digest-identified artifacts with provenance, SBOM, signature and reproducible qualification evidence. | Use Gitflow release branches and v1 release candidates; a changed input creates a new candidate. Registry, signing identity, publication role and installation verification are separate operator responsibilities. See [RELEASE.md](../RELEASE.md) and [assurance](assurance.md). |

## Local platform, key custody and human presence

Protected local files, native projections and host key stores solve different
problems. CircularQuay makes the operator choose the boundary rather than
claiming a uniform security property across platforms.

| Integration | V1 contract | Boundary |
| --- | --- | --- |
| Linux, Windows, macOS, Android/Termux and containers | Deliver a complete generation through a qualified platform-specific native boundary. | File ownership/ACLs, system services, DPAPI, launch agents, containers and Android background behaviour are independently qualified. A machine-local convenience projection is not portable recovery custody. See [platform support](platform-support.md). |
| OS keychains and hardware-backed keys | Optional adapters can protect an enrolled local key or perform a bounded approved operation using the OS keychain, TPM, Secure Enclave, PIV/YubiKey or equivalent hardware. | Hardware/non-exportability, user presence, PIN/biometric prompt, recovery and unattended execution differ by device and adapter. No adapter represents a downloaded age key as magically non-exportable. See [credential boundaries](credential-boundaries.md). |
| Passkeys and mobile authenticators | An external IdP or managed-service UI can require passkey/MFA user presence for a human authentication event. | A passkey authenticates an account to its relying party; it is not a provider credential, workload identity, raw-secret read grant or unattended host approval. Mobile push/biometric policy stays with the IdP. |
| Managed local custody | A separately privileged local service can retain a key and perform a bounded bundle/workload operation. | IPC caller identity, bundle digest, workload, authorization domain, time window, policy revision, protected storage, service update path and audit all participate in the boundary. It is not a generic decrypt oracle. |

## Optional MCP and managed-service surface

MCP is optional. It exposes narrowly registered actions and filtered metadata;
it does not enroll a host or disclose values simply because an AI client is
connected. The managed key service is optional too: it adds authorised
identity-to-age-key mappings and administration, not email/password identity
management.

| Surface | V1 contract | Enforcement and testing |
| --- | --- | --- |
| Local stdio MCP | The launcher/OS identity is the local principal. Discovery, action, status and an explicit raw read use separate grants. | Shared OS users are a shared trust boundary. Synthetic disclosure, guessed-ID, cancellation and ownership cases must prove denied requests never release a value. |
| Remote MCP/API | An OAuth-protected resource accepts only its named authority/profile and resource URL. | Bind TLS, issuer, audience, client and operation; check Host/Origin/path policy and preserve replay/audit boundaries. Never forward inbound tokens to secret providers. [MCP operation guidance](operators/mcp.md) details the contract. |
| Managed UI/API/CLI | Scoped administrators create, update, inspect, expire and revoke mappings and grants, while users see only their effective access. | Every mutation uses server-side authorization, optimistic versioning, tenant/domain isolation and redacted durable audit. Browser controls, Cloudflare/Netlify account roles, and client-provided mapping IDs are not enforcement. See [managed key service](managed-key-service.md). |

## Cloudflare and Netlify adapter boundary

Cloudflare and Netlify are optional adapters for the managed UI/API. They
may supply hosting, ingress authentication, previews and local adapter
emulation. They never replace CircularQuay's Go policy core or become the
authoritative source of a secret-provider value, a key mapping, or an access
decision merely by hosting a page.

| Platform capability | Intended use | Configuration, test and portability boundary |
| --- | --- | --- |
| Cloudflare Workers, Wrangler and the Workers Vitest integration | Host an optional thin API/UI adapter; use `@cloudflare/vitest-plugin` with Miniflare for deterministic adapter unit/integration tests against simulated bindings. | Cloudflare recommends the Vitest integration for Workers and Pages Functions; it runs tests in the Workers runtime and isolates storage per test file. Default CI uses local simulated bindings; remote bindings are a distinct preview/staging profile with synthetic data and explicit write authority. [Workers Vitest](https://developers.cloudflare.com/workers/testing/vitest-integration/) and [local-development security limits](https://developers.cloudflare.com/workers/local-development/) apply: Miniflare is not a hardened sandbox. |
| Cloudflare Access | Protect an optional deployed UI/API with an existing enterprise IdP and issue an application assertion to the adapter. | Validate issuer, signature, audience and expiry inside the receiving service; a header alone is insufficient. Access controls ingress and session lifecycle, but does not decide a CircularQuay mapping/grant. Signing keys rotate and validation must follow the configured trust contract. [Cloudflare Access JWT validation](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/validating-json/) is the primary integration reference. |
| Cloudflare Go SDK and Terraform provider | Optional narrow operator adapter and reviewed desired-state delivery for Cloudflare resources. | Use least-privilege, operator-owned credentials; never put credentials in CircularQuay policy or make IaC state a runtime authorization proof. The [Go SDK](https://developers.cloudflare.com/api/go/) and [Terraform provider](https://developers.cloudflare.com/terraform/) are platform APIs, not an entitlement to alter an account. |
| Netlify Dev, `@netlify/vite-plugin` and Deploy Previews | Develop the web UI/API locally and run disposable PR preview browser/a11y/e2e checks with synthetic tenants and keys. | `netlify dev`/the Vite plugin are adapter-development environments, not a substitute for the Go core. Preview deployments use separate audience, storage, identity and audit namespaces; no production mappings, custody or secret values enter them. [Netlify local function execution](https://docs.netlify.com/api-and-cli-guides/cli-guides/manage-functions/) and [Deploy Previews](https://docs.netlify.com/deploy/deploy-types/deploy-previews/) define those facilities. |
| Netlify Database | Optional managed-service transactional state for mappings, grants and audit metadata where an operator selects Netlify. | The local CLI/Vite flow uses a real Postgres-compatible local database; migrations, tenant isolation, backup/restore and rollback remain CircularQuay acceptance gates. It is a provider service, not a free/portable persistence guarantee. [Local Database development](https://docs.netlify.com/build/data-and-storage/netlify-database/local-development/) is the primary reference. |
| Netlify Identity, external JWT and team SSO/SCIM | Use either Netlify Identity or an enterprise external-JWT/SSO profile to authenticate a hosted UI as selected by the operator. | Netlify Identity is an optional application-user service, not the required enterprise IdP; Netlify documents that its external-JWT RBAC alternative cannot run with Identity on the same project. Netlify team SSO/SCIM manages platform users, not CircularQuay grants. [Identity](https://docs.netlify.com/manage/security/secure-access-to-sites/identity/overview/), [external JWT RBAC](https://docs.netlify.com/manage/security/secure-access-to-sites/role-based-access-control/) and [team SSO](https://docs.netlify.com/manage/security/secure-netlify-access/configure-team-saml-sso/) are the current constraints. |
| Netlify Functions and Blobs | A Netlify-hosted UI may use Functions for its adapter API and Blobs only for non-secret, non-authoritative UI artifacts where its consistency model fits. | Go Functions remain supported but Go Functions cannot access Netlify Blobs. Blobs are last-write-wins with configurable consistency and have a sandboxed local store; they are not key/policy/audit authority. [Function configuration](https://docs.netlify.com/build/functions/configuration/) and [Blobs limitations](https://docs.netlify.com/build/data-and-storage/netlify-blobs/) are controlling references. |

Provider charges, account entitlements, access seats, preview availability and
remote-binding effects are deployment prerequisites. They are never bypassed
by treating a local emulator, dashboard role or browser session as CircularQuay
authorization. Platform adapters are tested at three levels: local synthetic
emulation, isolated preview with synthetic identities/data, and separately
approved production qualification.

## Application SDKs and agent clients

[Official Go, TypeScript and Python SDKs](sdks.md) retain explicit local or MCP
boundaries, strict response/error contracts, cancellation and separately verified
package/native versions. [Portable skills and client plugins](agent-integrations.md)
package the same scoped operating contract, not new permissions or an implicit
hosted service. UI/API/CLI/MCP management intersects the same tenant, actor,
operation and active-window ceiling; raw disclosure has its own grant.

Use [operational configuration](operational-configuration.md) for runtime
collector and release-awareness declarations. An SDK observer, client manifest,
CI environment or discovery hint cannot select a new credential source or widen
an enrolled trust relationship. Registry/package identities and client tests
are independently recorded in the release evidence.

## Observability, audit and enterprise operations

| Integration | V1 contract | Boundary |
| --- | --- | --- |
| Structured logs, metrics and traces | Export redacted lifecycle, decision and health signals to an operator-selected OpenTelemetry-compatible collector or SIEM adapter. | No raw provider values, bearer tokens, private keys, request bodies or decrypted configuration appears in telemetry. Correlation identifiers are designed to be non-secret; sampling and retention are operator-controlled. |
| SIEM and audit export | Emit durable, redacted administrative and authorization-decision records for approved audit consumers. | Audit records identify actor, authority, operation, object reference, policy/version, decision and time without recording key/plaintext/token material. Delivery failure behavior is explicit; key release requires the durable audit contract selected by the service profile. |
| Alerts, incidents and recovery | Integrate with existing on-call/incident systems using scoped webhooks or collector credentials. | Alert delivery cannot grant access or relax a failed retrieval. Recovery validates artifact digests, trust material, backup authority and fresh identities; it does not copy machine-bound key-store state as a portable restore. See [incident response](operators/incident-response.md). |

## Integration acceptance contract

Before a new integration enters a v1 configuration schema, CircularQuay requires
all of the following:

1. A user-facing benefit that cannot be safely supplied by an existing generic
   adapter or native platform boundary.
2. An explicit authority model: who authenticates, who authorizes, where state
   lives, who can revoke, and what a copied artifact can bypass.
3. A typed versioned configuration with strict validation, canonical identity
   handling, no unknown or ignored fields, and no default broad trust.
4. A least-privilege credential and network model that avoids operational
   secret inspection, shell-profile dependence and ambient developer login.
5. Synthetic positive, denial, malformed-input, expiry, revocation, retry and
   concurrency tests; local emulator tests where the platform provides one;
   and separate live-provider/operator qualification where it matters.
6. Redacted audit/telemetry, failure/recovery behavior, supported-platform
   documentation, release artifact coverage and a named operational owner.

That standard prevents a catalogue of superficial connectors while keeping the
v1 architecture open to the integrations enterprises actually need.
