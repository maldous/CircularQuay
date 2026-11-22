# Credential and identity boundaries

This document defines credential, identity and custody boundaries for the v1
product. Artifact and platform qualification are recorded separately. CircularQuay maps an already-authorized
identity to exact reads and deliveries. It does not become the identity
provider, credential broker, vault, mobile application, or hosted connection
manager.

## Optional managed-key boundary

The [managed key service](managed-key-service.md) is a separate optional key
custodian/distribution authority. Core workload delivery does not become an
identity provider or general vault. A distributed private key is copyable;
separate-identity local custody and per-bundle remote unlocking have distinct
trust assumptions. Hardware/private-key protection does not recall plaintext
or released data keys. Every recipient and recovery path participates in the
bundle's real authorization boundary.

## Hosted database protection and compromise boundaries

This is a **design and qualification requirement**, not a deployed database or
cryptographic implementation. The inspected repository and retained records do
not establish an active hosted database, envelope implementation or configured
provider controls. Select **Cloudflare or Netlify** against the actual required
profile; this comparison does not authorize deploying both. Reuse the selected
provider's native services before adding tools. Account/CRM, billing and admin
records need tenant authorization and privacy controls as well as age private
keys needing confidential custody. Client configuration and source credentials
remain client-side and must never enter either hosted store.

### Data and key custody

Use provider encryption at rest and TLS as the storage baseline. Add a vetted
application AEAD envelope profile for age private-key bytes and sensitive
account fields where the threat assessment requires protection against database
exports. Generate independent random DEKs per profile/key version or sensitive
record, with the selected library's cryptographically secure generator; a
profile-wide DEK means compromise exposes that profile's entire DEK domain.
Public profile IDs, predictable seeds and custom permutations are **not encryption
secrets**. Use maintained cryptographic APIs, not an authored cipher or protocol.
[Envelope encryption](https://docs.cloud.google.com/kms/docs/envelope-encryption)
is the established layering pattern, not a decision to buy Google KMS.

Keep only ciphertext, authenticated format/algorithm version, nonce, wrapped DEK,
KEK version and necessary safe metadata in the database. Authenticate tenant,
profile, record, key version and purpose with canonical, unambiguous AAD; derive
the expected context from verified authorization and authoritative record identity,
not solely attacker-editable row labels. Reject unknown algorithms/versions,
wrong context and failed tags before returning plaintext. Ensure nonce uniqueness
per encryption key for data and any AEAD wrapping, including retries, concurrency
and restore. Follow the selected API's nonce and size limits. AAD is public and
does not itself prevent replay of a complete older valid row: restored versions,
revocations and meter state need authoritative reconciliation/fences. These
requirements follow [RFC 5116](https://www.rfc-editor.org/rfc/rfc5116.html).

Keep KEKs outside the database, dumps and application repository. First evaluate
the selected host's restricted production secret mechanism and established runtime
crypto support. A native runtime secret is a lean custody option, **not an HSM
or non-exportable KMS**: authorized application code can read/use it. Scope decrypt
authority to the custody service, keep plaintext lifetimes short, cap bulk reads,
alert on abnormal decrypt/export activity and separate support/account runtime
credentials from key custody where that produces an enforceable boundary. A
second table under the same all-powerful credential is not such a boundary.
This host-internal custody choice is separate from an explicitly selected
[user-profile KMS](managed-key-service.md#per-profile-age-and-kms-custody), which
is now a required hosted capability with dual association provisioning in the
product flow. This design does not establish a configured provider.
The launch custody profile uses a software crypto service with protected host
secrets outside the database, vetted crypto and random per-profile DEKs within
an existing paid Cloudflare or Netlify allocation. Secret storage alone is
not KMS. Runtime/shared KEK compromise can expose all reachable DEKs: this launch
tradeoff does not meet independent non-exportable/HSM custody unchanged. No
external paid KMS is required at launch; stronger profiles remain unavailable
until separately funded/qualified. Keep scoped auth/isolation/audit/recovery and
the versioned managed/HSM migration path; no new provisioning is authorized.

Version KEKs and DEKs; distinguish wrapping-key rotation/rewrap from replacement
of exposed age private keys. Retain protected recovery keys only for the declared
backup window; test restoration before retiring old KEKs. Encrypt exported
backups with independently controlled access, retain no plaintext exports and
include key recovery, deletion and incident response in retention policy. Minimize
account PII; keep payment-card entry with the payment provider. Do not log keys,
tokens, configuration contents, source URLs/credentials, plaintext PII or support
attachments containing them. Infrastructure administrators remain a separate
trust boundary from application support roles.

### Native provider control mapping

Official documentation was checked on 4 November 2026. Netlify describes
[Netlify Database as managed PostgreSQL](https://docs.netlify.com/build/data-and-storage/netlify-database/);
that establishes the engine family, not configured RLS or constrained runtime
roles. Every row requires exact target/plan readback and the listed application
tests before a protection claim.
Cost references use the existing [forecast](managed-key-service.md#hosting-economics-and-capacity-assumptions),
not a second budget. Unknown entitlement or cost is an unresolved selection gate.

| Requirement | Cloudflare candidate and explicit configuration | Netlify candidate and explicit configuration | Verification, entitlement and cost |
| --- | --- | --- | --- |
| Database storage protection | [D1](https://developers.cloudflare.com/d1/reference/data-security/) encrypts stored objects automatically with Cloudflare-managed AES-256 keys and uses TLS. Bind only the required database to the designated Worker; constrain operator tokens separately. | Verify the exact Netlify Database service's at-rest/TLS and operator-access guarantees; the reviewed [database permissions](https://docs.netlify.com/build/data-and-storage/netlify-database/access-control/) page documents dashboard access, not encryption/key ownership. Do not infer customer-managed KEKs or PostgreSQL roles from it. | Record service, region, account, database and current guarantees; inspect bindings/roles without secret values. D1 operations/storage and Netlify compute/egress are modeled; Netlify storage tariff and custody costs remain unresolved. Provider EAR does not protect an authorized SQL export. |
| Envelope/key custody | [Workers secrets](https://developers.cloudflare.com/workers/configuration/secrets/) provide encrypted bindings; [Web Crypto](https://developers.cloudflare.com/workers/runtime-apis/web-crypto/) supports AES-GCM and key wrapping. Separate production secrets and service bindings from previews and account/support code. | Use [environment scopes and deploy contexts](https://docs.netlify.com/build/environment-variables/overview/) to restrict custody secrets to production Functions, with no browser/build/preview exposure; confirm actual secret-visibility restrictions and Secrets Controller entitlement before relying on them. Select a maintained runtime crypto library. | Test ciphertext/context swaps, actual rotation/restore and preview denial. Neither mechanism alone is an independently authorized unwrap service. Launch uses authorized software custody with explicit runtime risk; measure actual crypto CPU/DB/host meters. The current forecast is hosting-only; paid native KMS is a future optional cost, not a launch prerequisite. |
| Tenant and field authorization | D1 is SQLite-based: no PostgreSQL RLS claim. Use verified server tenant/profile context, parameterized tenant predicates for every query/write/join/list/aggregate/job, composite tenant keys and constrained bindings. | If the selected PostgreSQL service permits the required roles/policies, use the RLS contract below. A dashboard Team Owner/Developer permission is operator authority, not customer isolation. Verify actual database role provisioning and pool behavior; unsupported controls are a gap. | Exercise customer, scoped support, agent and global-admin paths across two tenants, including background work and exports. D1 can use separate databases for justified isolation, with migration/binding/limit cost; PostgreSQL connection/pool/role costs must be measured. |
| Authentication | Worker verifies the configured application's issuer, audience, subject and scopes; operator Cloudflare Access/team login alone is not customer authentication. | Use actually entitled [Netlify Identity](https://docs.netlify.com/manage/security/secure-access-to-sites/identity/plans-and-pricing/) or the chosen supported issuer, then enforce account RBAC server-side. | Test invalid/revoked/wrong-audience identities and every UI/API/CLI/SDK/MCP route. Included Identity pricing does not prove machine-flow suitability; external IdP/enterprise entitlements remain explicit costs. |
| Abuse and authoritative quotas | Use native [rate limits](https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/) for local approximate abuse control; durable transactional state enforces global usage/idempotency. Select available zone rules without challenging unattended clients. | Qualify entitled [rate limiting](https://docs.netlify.com/manage/security/secure-access-to-sites/rate-limiting/) and durable database quota transactions; [WAF](https://docs.netlify.com/manage/security/secure-access-to-sites/web-application-firewall/) requires Enterprise + High-Performance Edge. | Test distributed bursts, retries, denial and limit-store outage. WAF is not tenant authorization or exact billing. Lean forecast includes D1 admission writes; premium protection is optional/quoted, not silently included in Netlify Pro. |
| Logs and evidence | Use native Workers logs/metrics with redaction, explicit retention and alerts. | Use native function/observability logs, confirm retention/export access on the selected plan. | Synthetic canary-secret leakage test must cover app, edge, traces and support/export surfaces. Forecast's log allowance/retention is bounded; longer durable audit/export is a separately quoted requirement, not an immutable-log claim. |
| Backup and recovery | [D1 Time Travel](https://developers.cloudflare.com/d1/reference/time-travel/) on the production storage backend is automatic: Paid 30 days, Free 7 days, no additional history/restore charge. Restrict destructive restore authority. | [Netlify database backups](https://docs.netlify.com/build/data-and-storage/netlify-database/backup-and-recovery/) are daily and on production publish. Scheduled retention: Free 3 days, Personal 7, Pro/Enterprise 30; on-publish retention differs. Restrict restore to the appropriate operator. | Restore synthetic data with the correct historical KEK, then reconcile revocations/usage and verify fail-closed fences. Daily backup is not 15-minute RPO. Longer exports, encrypted key recovery and restore labor are additional costs; no restore has run here. |

### PostgreSQL row security contract

For a qualified PostgreSQL target, use a dedicated runtime role that is neither
table owner nor superuser and has no `BYPASSRLS`; separate migration/maintenance
credentials. Enable RLS on all tenant tables; without an applicable policy,
enabled RLS denies normal row reads and writes. Table owners normally bypass
RLS; `FORCE ROW LEVEL SECURITY` subjects owners to it. Superusers and roles with
`BYPASSRLS` still bypass it, including when `FORCE` is set. Use `USING` to
restrict rows selected, updated or deleted, and `WITH CHECK` to constrain new
inserted/updated rows, including tenant changes. Remove runtime DDL, role
administration, `TRUNCATE` and broad
maintenance capabilities. Review views, security-definer functions, joins,
partitions, exports and background jobs for bypass paths. PostgreSQL documents
these limitations in its [PostgreSQL 17 row-security reference](https://www.postgresql.org/docs/17/ddl-rowsecurity.html).

Set transaction-local tenant context only after server identity/grant verification,
on the same connection and transaction as the protected queries. Never accept a
client-supplied tenant setting as proof; never use session-level context that can
bleed through pooled connections. Missing/malformed context must deny. Prove
alternating-tenant, rollback, timeout and concurrent-pool isolation with the actual
driver/pool. RLS can contain query mistakes; a compromised application credential
that can set arbitrary trusted context can still impersonate tenants. Scope
credentials/stores or use independently enforced identity where that threat must
be contained. No PostgreSQL version, RLS privilege or pooled-context behavior has
been verified for a deployed CircularQuay target.

### Compromise model and residual risk

| Compromise | Expected protection and remaining exposure |
| --- | --- |
| Database dump/export without KEK or decrypt authority | AEAD envelope keeps covered private-key/sensitive fields confidential; unencrypted account metadata, relationships, volume and timestamps can still leak. EAR alone does not protect a logical export. |
| One independent DEK exposed | Its record/profile domain is exposed; independent DEKs restrict cryptographic spread. Released age keys and copied ciphertext remain usable and cannot be recalled. |
| Authorized application runtime compromised | Attacker can abuse accessible data and unwrap authority, impersonate context and leak plaintext. Short access, separate custody authority, rate/bulk limits and alerts reduce opportunity; per-record DEKs do not guarantee containment. |
| Shared KEK or sufficiently privileged operator compromised | Every reachable wrapped DEK under that KEK can be exposed, including retained backups. Random per-record DEKs do not fix shared-KEK compromise. Partition custody only where independently enforceable and plan incident key replacement. |
| Old valid database/backup restored | AEAD can still validate old content. Reconcile key versions, revoked grants and usage fences against authoritative recovery evidence before serving; if freshness is uncertain, remain unavailable. |

The [integration cases](integration-testing.md#hosted-database-security-validation)
are required evidence for implementation. They are currently **unexecuted**;
static documentation checks and a standalone crypto example would not establish
application isolation, safe custody or deployment protection.

## Protected storage and native delivery profiles

| Surface | Profile contract | Boundary |
| --- | --- | --- |
| Policy loading | The loader rejects a writable or symlinked policy leaf, bounds its input, and checks that the opened file is the inspected inode. | Leaf validation alone does not establish complete ownership and trusted-ancestor custody; operator-controlled paths and protected bootstrap/state checks remain separate requirements. |
| Bootstrap files, state, and audit receipts | Protected-file helpers require absolute regular paths, reject symlinks/reparse points, bound reads, and use staged replacement for writes. Unix secret files use owner-only permissions; Windows checks protected DACLs. | This is protected filesystem storage, not a host keychain or credential-manager integration. Operators still provision each bootstrap value outside policy. |
| Linux native projection | A `systemd` target invokes the fixed `/usr/bin/systemd-creds encrypt` command with a constrained credential name, then writes the encrypted output through the protected-file helper. | It is a systemd credential projection, not SOPS ciphertext, a desktop keyring, or evidence of a qualified reboot/recovery path. |
| Windows native projection | A `windows` target uses DPAPI LocalMachine with UI forbidden, then applies a protected DACL. A declared service SID may receive read access. | DPAPI LocalMachine is machine-bound convenience; ACLs provide the service separation. It is not Windows Credential Manager, Windows Hello, a user keychain, or portable recovery material. |
| Process delivery | `run` starts an absolute executable with a clean constructed environment. Provider bootstrap names are denied from the delivered environment. | A launched program can still read and disclose values delivered to it. No process environment is a durable secret store. |
| Local MCP identity | Stdio identifies the local caller from kernel identity. HTTP mode verifies configured issuer, audience, signing algorithm, access-token type, subject, client, scopes, and current grants. | It is a resource-server check, not user login, SSO, dynamic client registration, SCIM, or an authorization server. |

macOS/iCloud Keychain, Windows Credential Manager/Hello, Linux Secret Service,
Android Keystore, iOS Keychain, FIDO/WebAuthn, TPM, Secure Enclave and biometric
presence are distinct custody/identity profiles, not aliases for protected files
or DPAPI. Select an explicitly documented adapter with its own enrollment,
authorization, unattended-use and recovery contract; do not infer that any of
these facilities is enabled by a file path or platform name. Actual adapter and
device support belongs in the [qualification matrix](qualification.md).

User-presence protection and unattended workloads have different purposes.
Apple Keychain can require Face ID or Touch ID before a read, and Secure Enclave
private keys are generated on-device and are not importable as arbitrary
existing secret bytes. That suits an interactive user or local signing key, but
would deliberately prevent a headless service from starting after reboot. See
Apple's [Keychain services](https://developer.apple.com/documentation/security/keychain-services),
[user-presence example](https://developer.apple.com/documentation/localauthentication/accessing-keychain-items-with-face-id-or-touch-id),
and [Secure Enclave constraints](https://developer.apple.com/documentation/Security/protecting-keys-with-the-secure-enclave).

## Provider and workload identity profiles

Each source declaration binds account/project/environment/namespace, resource,
identity method, and any version selector before a reader receives a request.
The executable does not silently select a human CLI session or a broad default
profile when the declared machine identity is unavailable.

| Integration | Explicit identity profile | Excluded from this profile |
| --- | --- | --- |
| AWS Secrets Manager and SSM | EC2 instance metadata role or ECS task credentials, followed by STS account confirmation. | AWS web identity, IRSA, arbitrary profiles, ambient access keys, and external credential-process chains. |
| Azure Key Vault secrets | Managed identity or workload identity with declared tenant/client/token-file constraints. | Interactive Azure CLI fallback and arbitrary authority/metadata overrides. |
| Google Secret Manager | GCE metadata service account with an exact declared account. | Application-default credentials, service-account key files, Workload Identity Federation, and arbitrary metadata hosts. |
| OpenBao/Vault KV v2 | Declared endpoint, mount/path/field/version and configured native bootstrap material. Lease metadata is retained. | Dynamic-engine issuance, lease renewal/revocation orchestration, or a general renewal daemon. |
| Doppler | Exact project/config read through the provider API using a protected bootstrap token file. | Human CLI login, service-account identity-token exchange, config inheritance as implicit authority, and use of the vendor CLI fallback cache. |
| Infisical | Exact project/environment/path/key read, with imports and secret-reference expansion disabled. | Universal Auth or OIDC login/exchange, session renewal, and an imported or expanded secret graph. |
| Bitwarden Secrets Manager | Fixed `bws secret get <id> --output json` invocation with `BWS_ACCESS_TOKEN` from a protected file and project identity checked in the response. | Password-manager desktop unlock, user vault/session reuse, and machine-account lifecycle management. |
| 1Password | Fixed `op read op://vault/item/field --no-newline` invocation with `OP_SERVICE_ACCOUNT_TOKEN` from a protected file. The declared project is the vault name. | 1Password Environments, `op run`, `op inject`, desktop approval/unlock, Connect, and interactive account sessions. |
| Akeyless | Exact named item through the documented API path using a protected bootstrap token. | Identity-token exchange, dynamic-secret lifecycle management, and provider administration. |

The 1Password profile selects the narrow service-account CLI form in
[1Password's scripting guidance](https://developer.1password.com/docs/cli/secrets-scripts/).
It does **not** support 1Password Environments even though that product can be
used by `op run` outside CircularQuay.

Bitwarden machine accounts are provider-managed non-human identities scoped to
projects/secrets and have provider event logs; their access token remains
bootstrap material that the operator must protect. See [Bitwarden machine
accounts](https://bitwarden.com/help/machine-accounts/). Doppler distinguishes
config-scoped service tokens from service-account and short-lived identity
tokens; the protected-token profile uses only its supplied bootstrap token and does not
perform the identity-token flow. See [Doppler service tokens](https://docs.doppler.com/docs/service-tokens)
and its [API authentication types](https://docs.doppler.com/reference/api).
Infisical Machine Identity Universal Auth and OIDC require separately declared
identity-exchange profiles; a protected API token does not imply either flow.

## Architectural boundary for hosted and managed connections

For a hosted application, each environment should have a separately declared
provider identity and exact resource mapping. A production association is not
inherited from development because names happen to match. The provider remains
authoritative for account membership, consent, token issuance, revocation, MFA,
hardware authenticators, audit retention, and provider-side rotation.
CircularQuay receives only the configured identity path and never stores a
user's provider password, browser session, recovery code, or OAuth refresh
token in policy.

This also applies to CI. A release or deployment job should obtain a
short-lived provider credential directly through the provider's constrained
federation mechanism. For GitHub Actions, an OIDC token requires only a
job-scoped `id-token: write` permission, but the receiving provider must bind
issuer, audience, subject, repository, branch/tag or environment as
appropriate. GitHub documents the narrow permission and need for claim
constraints in its [OIDC guidance](https://docs.github.com/en/actions/how-tos/secure-your-work/security-harden-deployments/oidc-in-cloud-providers)
and [OIDC claim reference](https://docs.github.com/en/actions/reference/security/oidc).
Hosted administration manages authorized associations through the
[managed-service contract](managed-key-service.md); local reader profiles do not
create those associations or inherit its provider credentials.

Kubernetes operator, admission-controller and target profiles are separate
integrations, not consequences of the local reader. Use a provider-supported external-secret integration or the
upstream Secrets Store CSI Driver for an authorized Pod, then qualify it as a
separate deployment path. Kubernetes describes the CSI driver as a DaemonSet
that retrieves external-store secrets and mounts them into specifically
authorized Pods, and warns that environment delivery can leak more readily
than file permissions. See [Kubernetes secret good practices](https://kubernetes.io/docs/concepts/security/secrets-good-practices/).

## Lifecycle and audit boundary

CircularQuay validates expiry and preserves lease/renewable metadata. It does
not snapshot, publish, or disclose a leased/renewable value through MCP, and
it rejects expired values. It does not renew a lease, create dynamic
credentials, rotate a provider secret, revoke a provider identity, or run a
universal background renewal service. Those activities remain the chosen
provider's lifecycle responsibility.

The local MCP audit profile writes a protected JSONL receipt before approved
release. Central durable audit, retention, SIEM export and administration use
the [enterprise audit contract](enterprise-integration.md#audit-contract), with
explicit storage, export and outage semantics. Neither a local receipt nor
best-effort telemetry replaces provider audit or proves that a native copy
retains upstream read semantics. Qualify each selected boundary separately.

These limits follow the project's core design: one authority per value, no new
cryptography or vault, no global credential broker, no general process
supervisor, and no universal lease-renewal daemon. They keep the executable's
authority smaller than the providers and platform identity systems it connects.
