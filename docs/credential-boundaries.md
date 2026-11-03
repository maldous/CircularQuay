# Credential and identity boundaries

This document describes the current executable, not a platform qualification
or a promise of provider compatibility. CircularQuay maps an already-authorized
identity to exact reads and deliveries. It does not become the identity
provider, credential broker, vault, mobile application, or hosted connection
manager.

## Current local protection surface

| Surface | Current behavior | Boundary |
| --- | --- | --- |
| Policy loading | The loader rejects a writable or symlinked policy leaf, bounds its input, and checks that the opened file is the inspected inode. | It does not apply the protected-file helper's complete ownership and trusted-ancestor checks. |
| Bootstrap files, state, and audit receipts | Protected-file helpers require absolute regular paths, reject symlinks/reparse points, bound reads, and use staged replacement for writes. Unix secret files use owner-only permissions; Windows checks protected DACLs. | This is protected filesystem storage, not a host keychain or credential-manager integration. Operators still provision each bootstrap value outside policy. |
| Linux native projection | A `systemd` target invokes the fixed `/usr/bin/systemd-creds encrypt` command with a constrained credential name, then writes the encrypted output through the protected-file helper. | It is a systemd credential projection, not SOPS ciphertext, a desktop keyring, or evidence of a qualified reboot/recovery path. |
| Windows native projection | A `windows` target uses DPAPI LocalMachine with UI forbidden, then applies a protected DACL. A declared service SID may receive read access. | DPAPI LocalMachine is machine-bound convenience; ACLs provide the service separation. It is not Windows Credential Manager, Windows Hello, a user keychain, or portable recovery material. |
| Process delivery | `run` starts an absolute executable with a clean constructed environment. Provider bootstrap names are denied from the delivered environment. | A launched program can still read and disclose values delivered to it. No process environment is a durable secret store. |
| Local MCP identity | Stdio identifies the local caller from kernel identity. HTTP mode verifies configured issuer, audience, signing algorithm, access-token type, subject, client, scopes, and current grants. | It is a resource-server check, not user login, SSO, dynamic client registration, SCIM, or an authorization server. |

CircularQuay has no implementation for macOS Keychain, iCloud Keychain, Windows
Credential Manager, Linux Secret Service/keyrings, Android Keystore, iOS
Keychain, FIDO/WebAuthn, TPM, Secure Enclave, fingerprint readers, Face ID, or
Touch ID. Those are not aliases for the protected-file or DPAPI paths above.

User-presence protection and unattended workloads have different purposes.
Apple Keychain can require Face ID or Touch ID before a read, and Secure Enclave
private keys are generated on-device and are not importable as arbitrary
existing secret bytes. That suits an interactive user or local signing key, but
would deliberately prevent a headless service from starting after reboot. See
Apple's [Keychain services](https://developer.apple.com/documentation/security/keychain-services),
[user-presence example](https://developer.apple.com/documentation/localauthentication/accessing-keychain-items-with-face-id-or-touch-id),
and [Secure Enclave constraints](https://developer.apple.com/documentation/Security/protecting-keys-with-the-secure-enclave).

## Current provider and workload identity surface

Each source declaration binds account/project/environment/namespace, resource,
identity method, and any version selector before a reader receives a request.
The executable does not silently select a human CLI session or a broad default
profile when the declared machine identity is unavailable.

| Integration | Current executable identity path | Explicitly not implemented by this adapter |
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

The current 1Password behavior matches the narrow service-account CLI form in
[1Password's scripting guidance](https://developer.1password.com/docs/cli/secrets-scripts/).
It does **not** support 1Password Environments even though that product can be
used by `op run` outside CircularQuay.

Bitwarden machine accounts are provider-managed non-human identities scoped to
projects/secrets and have provider event logs; their access token remains
bootstrap material that the operator must protect. See [Bitwarden machine
accounts](https://bitwarden.com/help/machine-accounts/). Doppler distinguishes
config-scoped service tokens from service-account and short-lived identity
tokens; the current adapter uses only the supplied bootstrap token and does not
perform the identity-token flow. See [Doppler service tokens](https://docs.doppler.com/docs/service-tokens)
and its [API authentication types](https://docs.doppler.com/reference/api).
Infisical Machine Identity Universal Auth and OIDC are provider features, not
current CircularQuay adapter behavior.

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
CircularQuay has no hosted control plane that creates those associations or
holds their long-lived credentials.

For Kubernetes, this executable has no operator, admission controller, or
Kubernetes target. Use a provider-supported external-secret integration or the
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

The built-in audit path is a protected local JSONL receipt for approved MCP
operations. It is not a centralized immutable audit system, SIEM export,
provider-audit replacement, or proof that a destination keeps provider read
audit semantics. Required provider auditing must be configured and qualified
by the operator.

These limits follow the project's core design: one authority per value, no new
cryptography or vault, no global credential broker, no general process
supervisor, and no universal lease-renewal daemon. They keep the executable's
authority smaller than the providers and platform identity systems it connects.
