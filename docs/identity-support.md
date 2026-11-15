# Identity support and evidence

This inventory describes source at commit `c7627a0e320861c2ed3f63d42e711b9fae2e0639`,
tree `29a9642f42d4fde20cc111a066801e7c94ba2ad5`.
It separates implemented resource-server checks, authored synthetic tests,
retained execution receipts and external integration obligations. Inspecting
source or a test definition is not a new test result. No live identity-provider,
client, account, managed-hosting or release qualification is established here.

Read [hosted identity](hosted-identity.md) for the full intended contract and
[qualification](qualification.md) for revision-bound results and remaining gates.

## Implemented verifier and transport boundaries

The implemented named-authority token profiles are exactly `rfc9068` and
`microsoft-entra-v2`, declared in
[identity.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/identity.go#L11).
Provider names are integration choices, not additional token profiles.

| Boundary | Implemented source | Authored evidence at the selected revision |
| --- | --- | --- |
| Named trust | A locally configured authority pins issuer, same-origin JWKS URL, audience and profile. Verification accepts RS256/ES256 and validates signed claims, bounded token/claim inputs and configured client/subject/scope/role filters. Token claims cannot introduce an authority or key URL; ambiguous acceptance denies. [Verifier](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify.go#L30), [routing](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/identity.go#L120). | [verify_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify_test.go): `TestRFC9068VerifierBindsSignedIdentityAndFilters`, `TestRFC9068VerifierRejectsProfileAndPermissionViolations`, `TestVerifierRejectsAmbiguousJWTJSON`; [identity_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/identity_test.go): `TestNamedTokenRoutingDeniesMissingAndAmbiguousAuthority`. |
| RFC 9068 | Requires the `at+jwt` header, nonempty `client_id` and `jti`, consistent optional `azp`, valid time claims and exact configured issuer/audience. Principal identity is `issuer|sub`. [Profile handling](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify.go#L359), [header](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify.go#L429). | [verify_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify_test.go): the RFC 9068 tests above reject altered token type, malformed permissions and expiry/not-before violations; `TestVerifierRejectsMalformedTokensAndDisallowedIdentity` rejects disallowed identity and malformed tokens. These use a synthetic issuer. |
| Microsoft Entra v2 | Requires `JWT` token type, version `2.0`, a configured tenant GUID in the issuer path, GUID API audience, matching `tid`, GUID `oid`/`azp`, and scopes or application roles. Principal identity is `issuer|tid/oid`. [Entra profile](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify.go#L382). | [verify_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify_test.go): `TestMicrosoftEntraV2VerifierBindsTenantObjectAndPermissions`, `TestMicrosoftEntraV2RejectsIDAndUnqualifiedTokenProfiles`, `TestMicrosoftEntraV2AcceptsApplicationRoleWithoutDelegatedScope`. These are synthetic claim-profile tests, not an Entra tenant trial. |
| Authority-specific private CA | Optional paired protected PEM path/digest selects replacement roots for that authority. A caller-supplied client cannot override this trust path. [Verifier construction](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify.go#L30). | [identity_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/identity_test.go): `TestProductionNamedVerifierUsesAuthorityReplacementRoots`; [verify_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify_test.go): `TestVerifierRejectsAmbiguousOrMalformedTrustStoreConfiguration`. Test presence does not supply current native CA-store qualification. |
| MCP resource discovery | Resource metadata advertises configured issuers/scopes and DPoP metadata. The resource URL and named token audience are separate fields. [HTTP handler](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/http.go#L35), [configured metadata](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/identity.go#L14). | [identity_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/identity_test.go): `TestNamedTokenAudienceIsIndependentOfResourceURL`, `TestNamedMetadataPublishesOnlyConfiguredScopesAndIssuers`; [http_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/http_test.go): `TestHTTPGuardsAndMetadata`. Metadata does not implement an authorization server or register a client. |
| HTTP request boundary | TLS or a loopback hop, configured Host/Origin checks, bounded body/concurrency/time and a verified-token deadline precede MCP dispatch. Forwarded headers do not establish trusted transport. The caller owns TLS/server deadlines. [HTTP controls](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/http.go#L86). | [http_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/http_test.go): `TestHTTPGuardsAndMetadata`, `TestHTTPRejectsMissingOrExpiredTokenDeadline`, `TestNoScopeAndInvalidJWTReachNoBackend`, `TestBoundedHTTPConcurrency`. |
| Operation authorization | Verified identity intersects current authority fingerprint, authority-wide and operation-specific scopes/roles, exact principal/client/workload/key grants and time windows. Local principals cannot consume named-authority grants. [Principal authorization](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/service.go#L74). | [authorityservice_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/authorityservice_test.go): `TestNamedAuthorityPrincipalRequiresExactCurrentContract`, `TestNamedAuthorityFingerprintRechecksLoadedPolicy`; [expiry_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/expiry_test.go): expiry fences before provider work, after resolution/audit and before an action. |

The compact single-issuer MCP configuration uses the RFC 9068 verifier and
binds its resource to its configured audience
([http.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/http.go#L199)).
It does not add a third token profile. The named-authority route requires
policy validation before constructing its verifiers.

## Optional sender constraints

| Boundary | Implemented source | Authored evidence at the selected revision |
| --- | --- | --- |
| Token confirmation and scheme | A named authority explicitly selects DPoP. Its token must carry the supported `cnf.jkt` confirmation. Bearer authorities reject confirmation claims; a sender-constrained authority cannot downgrade to Bearer. [Confirmation](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/verify.go#L276), [HTTP scheme selection](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/dpop_auth.go#L30). | [dpop_contract_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/identity/dpop_contract_test.go): `TestDPoPConfirmationClaimContract`; [dpop_auth_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/dpop_auth_test.go): `TestRequireAccessTokenSeparatesBearerAndDPoPAuthorities`, `TestRequireAccessTokenValidDPoPProofAndSchemeFence`. |
| Proof binding | The restricted RFC 9449 verifier checks ES256/P-256 or RS256/RSA key constraints, signature, public-key thumbprint, bounded identifier/time, exact method/canonical resource and access-token hash. [Proof verifier](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/dpop/dpop.go#L82). | [proof_verifier_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/dpop/proof_verifier_test.go): `TestVerifierAcceptsES256AndRS256Proofs`, `TestVerifierRejectsBindingMismatchesBeforeStore`, `TestVerifierRejectsMultipleProofHeadersBeforeStore`, `TestVerifierBoundsReplayLifetimeToTokenExpiry`, malformed-header and unsafe-key cases. |
| Nonce and replay state | Required one-use nonces and replay decisions bind authority-policy fingerprint, key and resource. The included protected persistent store is explicitly single-replica and must not be shared by independently running processes. A cluster needs a suitable atomic shared store before DPoP is enabled. [Store contract](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/dpop/dpop.go#L63), [store limits](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/dpop/store.go#L28). | [store_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/dpop/store_test.go): expiry, malformed/unreadable state, write failure, capacity and cancellation cases. [dpop_integration_test.go](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/internal/mcp/dpop_integration_test.go#L28): `TestCompleteDPoPHTTPMCPReadRejectsReplayAndBearerDowngrade` uses a synthetic issuer, loopback MCP server, fake backend and persistent replay state. |

DPoP here is resource-server proof validation. It does not issue access tokens,
enroll or protect client private keys, implement mTLS/private-key JWT client
authentication, or qualify a live issuer/client/cluster. A stolen client key or
already disclosed plaintext remains outside its protection. Local policy
revocation and issuer-side session revocation retain the separate bounds
documented in [hosted identity](hosted-identity.md#resource-and-identity-contract).

## External integrations and design obligations

| Surface | Support claim at this revision | Required evidence or input before a stronger claim |
| --- | --- | --- |
| Auth0, Keycloak and Okta custom API access | Candidate external authorization servers when configured to emit a supported typed access-token profile; their names do not imply interoperability qualification. | Selected service/version/plan, dedicated API audience, actual token profile and resource/client/permission/denial tests. Existing operator prerequisites remain in [RELEASE.md](../RELEASE.md). |
| Microsoft Entra | Implemented v2 claim-profile validation, with synthetic tests listed above. | Actual tenant/app registration, entitlement and issuer/audience/client/role/scope lifecycle evidence. |
| OIDC login, SAML, MFA/passkeys, social providers, PKCE and client registration | External authorization-server/client responsibilities in the intended architecture. They are not additional implemented CircularQuay token-verifier profiles or proof of a connected login provider. | Selected integration's actual supported flow and configuration, trust/redirect/session/revocation tests and appropriate account authority. |
| Netlify Identity / Cloudflare Access | Applicable identity/login/edge interoperability remains a service- and plan-specific requirement. An edge assertion, login token or product capability name is not automatically a supported CircularQuay API access token. | Exact selected source/adapter/profile and verified entitlement/setup/interop evidence. No live integration or provider equivalence is established by this inventory. |
| Linking, provisioning, SCIM, delegated exchange and centralized administration | Intended association, least-privilege, revocation and audit obligations. Current verifier/MCP checks do not establish a complete managed control plane. | Implemented protected enrollment/state/administration and exact end-to-end qualification. See [management and discovery](management-and-discovery.md) and [qualification](qualification.md). |
| Managed hosting and tenant isolation | A verifier/listener is a component. It does not establish account/project entitlement, durable shared quota/ledger state, per-tenant isolation, recovery or production readiness. | Actual operator-owned destination, approved setup and independently bound operational evidence. No account creation, key activation, paid action or deployment follows from this document. |

## Retained qualification and its limits

The existing [qualification matrix](https://github.com/maldous/CircularQuay/blob/c7627a0e320861c2ed3f63d42e711b9fae2e0639/docs/qualification.md) records:

- A November 4 `internal/identity` package receipt: normal/race/vet and both
  complexity checks passed; coverage 183/183 statements. Its table does not
  supply a standalone exact commit for that row. Preserve it as historical
  package evidence; do not assign it to this selected revision.
- Frozen local checkpoint21 based on
  `97b757af8a91c7dce5227f8b6230fa3d98545d1a`: synthetic signed
  issuer-to-MCP reads, persistent nonce handling, replay and Bearer-downgrade
  denial passed within the recorded broader batch. Whole-module coverage was
  5,554/6,441 statements, below the unchanged 100% requirement.
- Managed DPoP diff review `97b757a` to `bfaa66c`, scan
  `56553a87-7692-4deb-9432-d0f716652dc3`: all 37 changed files reviewed,
  zero reportable findings/candidates, focused pinned Go 1.27.3 synthetic
  validation passed. That is a bounded diff result.
- [Native/cross-build run 41724962521](https://github.com/maldous/CircularQuay/actions/runs/41724962521)
  at `bfaa66cc116093588dd1053a301bdf90a8ecef28`,
  tree `0afc2f5f7fcaf00f1547b29f849540cf78cfc89f`: Linux/macOS/Windows
  native tests and 26 cross-builds passed. The same retained row records failed
  required coverage and eight unsuppressed gosec findings.

These are received historical records, not newly inspected raw execution
artifacts or rerun results. None is silently extended to
`c7627a0e320861c2ed3f63d42e711b9fae2e0639`. The current source/test map above establishes
where behavior is implemented and tested in source; current-candidate,
live-provider/client, cluster, release and whole-product acceptance remain
separate requirements. Keep the existing strict thresholds and action-specific
identity, account, runtime and publication boundaries.
