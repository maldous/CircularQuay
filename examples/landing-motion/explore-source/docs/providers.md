# Provider interoperability

The provider registry resolves exact source declarations. Existing managers stay authoritative. CQ does not need to copy all values into OpenBao. See [credential and identity boundaries](credential-boundaries.md) for native-storage, explicit workload-identity, hosted-connection and excluded-profile boundaries.

## Separate capabilities

Authentication, KMS/Transit key release, value retrieval and publication are distinct. SOPS's KMS support protects an envelope key; it is not an existing-value secret-store importer. OpenBao plugins that authenticate cloud identities or generate credentials do not inherently federate every cloud store.

| Source | Explicit selectors |
| --- | --- |
| JSON | Policy-relative path and exact top-level string field. |
| Literal dotenv/process environment | Exact declared names; no shell expansion, recursion or profile execution. |
| SOPS | Approved executable/input, identity source and restricted Vault endpoint metadata. |
| OpenBao/Vault KV v2 | Instance, namespace when applicable, mount/path/field/version and native identity. |
| AWS SM/SSM | Account, region, exact ARN/name and version/stage/label. |
| Azure | Tenant/vault/secret/version; keys and secrets use different APIs. |
| GCP | Project/secret/version and payload integrity. |
| Doppler | Exact project/config; whole-config upstream read scope disclosed. |
| Infisical | Project/environment/path/key; imports/reference expansion disabled unless approved. |
| Bitwarden | Project/secret identity and supported vendor decryption. |
| Akeyless | Exact item/identity within account entitlement. |
| 1Password | Service-account CLI read of one exact `op://vault/item/field` reference. 1Password Environments are unsupported. |

`cq providers` reports what this executable implements. A source whose dependency, authentication or lifecycle combination is unsupported must fail explicitly. No table entry alone qualifies a live provider.

## Explicit adapter profiles

The provider table describes selector shape, not every upstream authentication
method. The native cloud-reader profiles select AWS EC2/ECS credentials, Azure
managed/workload identity, and a GCE metadata service account; they do not
fall back to a developer profile, interactive login, service-account key file,
or arbitrary metadata endpoint. The hosted-manager adapters consume an
operator-provisioned protected token file; they do not create or renew a
provider session.

For vendor CLIs, Bitwarden is a fixed `bws secret get` invocation using a
machine-account access token, and 1Password is a fixed service-account `op
read` invocation. CQ does not use desktop approval, a user vault
session, `op run`, `op inject`, or 1Password Environments. Infisical reads an
exact value with imports and reference expansion disabled; its Universal Auth
and OIDC exchanges are separate identity profiles, not implicit behavior of this adapter. These boundaries are
intentional and are detailed in [credential and identity boundaries](credential-boundaries.md).

## Composition

Each binding has one authority. Duplicate output names fail; an upstream object may require a broader read than the selected field. Split upstream records when their access boundaries differ.

A fetched value remains literal even if it looks like `ref+...`, a shell expression, URL, or template. No arbitrary command-based adapter is exposed. Unrestricted `vals` evaluation is not a privileged policy boundary; its `exec`, `file` and expansion utilities serve a different trust model.

Use explicit production identity mechanisms. A missing workload identity must not fall through to an administrator's CLI login or a different account. Audit and error outputs contain approved metadata only, never SDK response bodies.

## Export and lifecycle

Read authority is not publication or offline-export authority. One-way targets have separate identities and ownership. Conditional writes are used only where actually supported; there is no cross-provider transaction.

Static values may be snapshot candidates with explicit policy. Leased/dynamic/expiring credentials preserve native validity and cannot become indefinite encrypted copies. Stored bytes do not extend a lease. Non-exportable keys stay remote operation references.

Provider-native read auditing must be configured and qualified. Do not describe all remote retrieval as audited, or imply that publication carries the origin's revocation/audit guarantees into another platform.
