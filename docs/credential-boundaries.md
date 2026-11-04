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
