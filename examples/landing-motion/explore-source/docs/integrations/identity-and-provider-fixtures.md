# Identity and provider interoperability fixtures

This contract defines an opt-in, disposable local qualification profile for
CircularQuay's existing HTTP identity and OpenBao KV v2 readers. It is test
data, not a deployment template. The fixture never changes a production
authority, provider, system trust store, credential, or service.

All identities, client secrets, tokens, values, certificates, and paths in this
profile are fabricated. A harness creates them in a private temporary directory,
binds every listener to loopback, removes containers and temporary material at
completion, and must not log a raw token, bootstrap token, or returned value.

## Scope and entry conditions

The profile is intentionally optional. Core CircularQuay does not require
Keycloak, OpenBao, Docker, a Java runtime, or a container runtime. A developer
runs it only through its documented, digest-selected integration target with a suitable
local container runtime is available. It must be a separate, opt-in CI job as
well; normal unit, race, fuzzing, and cross-platform build jobs stay independent
of it.

Use an official, digest-pinned image only after its image digest is independently
recorded in the harness or lock file. This repository deliberately does not put
an unverified mutable Keycloak or OpenBao image tag in this fixture. The runner
must reject an unpinned image reference.

The profile exercises the selected resource-server and provider
adapters. It does not demonstrate a hosted control plane, user login, dynamic
client registration, SCIM, external authorization-server discovery, token
issuance by CircularQuay, provider administration, renewal, or production
recovery.

## Local TLS and endpoint binding

Both configured authority and OpenBao source endpoints must be HTTPS. The policy
schema deliberately rejects HTTP for those declarations, including loopback.
For every run, generate a short-lived private test CA and leaf certificates with
the actual loopback DNS/IP subject alternative names used by the containers.
Pass that CA only to the test HTTP clients or process-local trust configuration.
Do not set `InsecureSkipVerify`, install the CA into an operating-system trust
store, or reuse a certificate/private key from any environment.

Keycloak's issuer and JWKS endpoint are discovered only to construct the
ephemeral test policy after the local server starts. The policy still pins their
exact strings and requires the same origin; CircularQuay does not dynamically
trust discovery during normal authorization. The public MCP resource URL is a
separate loopback URL. It is not inferred from the token audience.

OpenBao's TCP listener defaults to TLS; this fixture keeps TLS enabled and binds
only to a loopback address. Its API address and the `sources.openbao.endpoint`
value must be the same loopback HTTPS origin. See OpenBao's [TCP listener
documentation](https://openbao.org/docs/configuration/listener/tcp/) and
[configuration reference](https://openbao.org/docs/configuration/).

## Keycloak RFC 9068 fixture

`internal/integration/testdata/keycloak/cq-it-realm.json` is a realm import
for a disposable realm named `cq-it`. It has no users, no interactive login,
and no administrator account. The harness supplies its own short-lived local
administrator bootstrap outside the export. The only client secret in the export
is the literal `fixture-only-not-a-production-secret`; it is public synthetic
test material and must never be accepted by a non-fixture server.

The export uses a service-account client, `cq-fixture-client`, so a local
client-credentials request has a stable client identity. It requires a Keycloak
version that supports the per-client RFC 9068 access-token header option. The
client attribute `access.token.header.type.rfc9068: true` produces `typ: at+jwt`;
Keycloak documents that this option was introduced in 26.2 and is disabled by
default. See [Keycloak 26.2](https://www.keycloak.org/2025/04/keycloak-2620-released).

The harness must issue a token and assert its signed, non-secret shape before it
attempts a CircularQuay request:

| Claim or header | Required fixture result | Why CircularQuay requires it |
| --- | --- | --- |
| Header | `alg: RS256`, `typ: at+jwt`, nonempty `kid` | The RFC 9068 profile accepts only RS256/ES256 and rejects a generic JWT header. |
| `iss` | Exact temporary Keycloak realm issuer | Named authorities pin the issuer. |
| `aud` | Contains `cq-fixture-audience` | The export's audience mapper adds this exact custom audience. |
| `sub` | Nonempty Keycloak service-account subject | Principal identity is issuer-qualified with this subject. |
| `client_id` | `cq-fixture-client` | The service-account client-scope mapper supplies this claim for the client-credentials flow. |
| `jti`, `iat`, `exp` | Present and internally consistent | The verifier requires a token identifier, issue time and unexpired lifetime. |
| `scope` | Contains `cq.read` | The linked client scope explicitly enables inclusion in the access-token scope claim. |
| `roles` | Array containing `read-access` | A dedicated client-role mapper exposes a flat top-level role array; Keycloak's default nested `resource_access` claim is not consumed by this profile. |

The test policy declares one named `rfc9068` authority with the exact issuer,
same-origin JWKS endpoint, `cq-fixture-audience`, client allowlist
`cq-fixture-client`, and only the `read` permission. That permission requires
both `cq.read` and `read-access`, demonstrating the existing conjunctive
scope-and-role rule. The local grant then names the same authority, exact
issuer-qualified subject, client, workload, operation, key, and a short
time window. Names are examples, not reserved development/production labels.

Keycloak client scopes and audience mappers are the relevant configuration
surface. Their names become access-token scope and audience values only when
configured for the access token; inspect the signed token in the harness rather
than treating an import declaration as proof. See Keycloak's [client-scope
reference](https://www.keycloak.org/docs/26.3.5/server_admin/) and [protocol
mapper reference](https://www.keycloak.org/admin-api/protocol-mappers).

The required acceptance cases are:

1. A token with the complete expected shape reaches the protected MCP request
   and allows only the declared `read` action and key.
2. A wrong issuer, JWKS key, signature algorithm, `typ`, audience, client,
   subject, missing scope, missing role, malformed claim, or expired token is
   denied before a protected action.
3. A signed otherwise-valid token cannot bypass a missing or expired local
   grant, a different workload, operation, or key.
4. After the realm signs a new token with a rotated signing key, the verifier
   observes a changed non-secret `kid`, refetches the pinned same-origin JWKS,
   and accepts only a valid new token. The evidence records key identifiers and
   outcomes, never token strings.

The realm export does not turn on DPoP because the ordinary fixture proves strict
bearer behavior. A separate DPoP variation uses another disposable client with
Keycloak's `dpop.bound.access.tokens` client attribute enabled. It obtains a
token with a client-generated DPoP proof and requires `cnf.jkt`; the
CircularQuay authority then selects `sender_constraint` and the request uses
the `DPoP` authorization scheme. It must prove the nonce challenge and valid
retry, and deny a missing proof, wrong key, wrong method/URI, bad access-token
hash, expired proof, and replay. Keycloak documents its DPoP client switch and
the `cnf.jkt` binding at [Securing applications with
DPoP](https://www.keycloak.org/securing-apps/dpop). A bearer authority must not
accept `cnf` or a DPoP header.

## OpenBao KV v2 fixture

The OpenBao fixture enables a KV v2 mount named `cq-fixture` and writes one
synthetic record at `workload/demo`. It creates a fixture-only policy with
`read` capability on exactly:

```
cq-fixture/data/workload/demo
```

It issues one short-lived synthetic token for that policy and writes it to a
private temporary token file. The equivalent CircularQuay declaration is:

```json
{
  "type": "openbao",
  "endpoint": "https://localhost:PORT",
  "mount": "cq-fixture",
  "resources": ["workload/demo"],
  "identity": "token_file",
  "token_file": "/private-temporary-path/openbao-token"
}
```

The binding names the exact resource, selects one string field, and pins an
explicit numeric KV version where the scenario requires it. The reader profile
issues `GET /v1/<mount>/data/<resource>`, sends `X-Vault-Token`, sends
`X-Vault-Namespace` only when the policy declares a namespace, and adds
`version=<n>` only for an explicit binding version. It rejects destroyed or
deleted versions and rejects a returned version different from the requested
one. OpenBao documents the KV v2 `data/` API prefix and versioned reads in its
[KV v2 reference](https://openbao.org/docs/secrets/kv/kv-v2/) and [HTTP read
example](https://openbao.org/docs/next/commands/read/).

The fixture must prove all of the following without recording a returned value:

| Case | Expected result |
| --- | --- |
| Exact mount/resource/field/version and fixture token | One approved synthetic field is resolved, with returned version evidence. |
| Different resource or undeclared field | CircularQuay denies before an OpenBao request, or OpenBao's exact ACL denies it. |
| Different mount/path or a broad `/*` ACL | The fixture setup is rejected; no wildcard policy is acceptable. |
| Missing/invalid token or unrelated policy | OpenBao denies and CircularQuay does not expose an upstream body. |
| Destroyed/deleted or substituted version | CircularQuay fails closed. |
| Expired/renewable/dynamic provider response | It remains non-exportable/non-snapshotable according to current lifecycle rules. |

OpenBao KV is a source adapter test. It does not test or authorize dynamic
secret engines, token renewal, token revocation orchestration, provider-side
administration, or a general credential broker.

## Expiry and revocation evidence

The Keycloak test uses a short synthetic access-token lifetime and records
successful authorization before expiry and denial after expiry. Disabling the
service account, logging it out, or revoking a refresh token does **not** make
a locally verified self-contained access JWT invalid immediately. CircularQuay's local JWT profile denies further decisions when its named
authority/grant/policy changes, but cannot recall a disclosed value or launched
child. Immediate issuer revocation requires a separately selected and qualified
introspection or authenticated event channel, its client authentication,
outage/failure policy and propagation bound. A logout test alone supplies none
of that evidence.

## Evidence and acceptance boundaries

Each run records only fixture identity metadata: image digests, local endpoint
origins, TLS leaf/CA fingerprints, issuer, audience, JWT header algorithm/type
and `kid`, sorted claim names, synthetic principal/client identifiers or their
hashes, expiry-relative timings, selected source/version metadata, request
counts, and authorization outcomes. It omits raw JWTs, client secrets, OpenBao
tokens, private keys, certificates, and secret values.

A complete fixture receipt binds harness/image digests, isolated TLS trust,
client and server versions, exact source/tree, allowed/denied operations and
cleanup. It does not qualify native reboot/recovery, live-provider lifecycle,
multi-replica DPoP or managed UI/API. Record the actual harness, provider and
platform outcomes in the [qualification matrix](../qualification.md). A design,
fixture declaration or source review is never an executed interoperability pass.
