# Connected-provider setup

Live provider tests are optional and use dedicated fabricated projects. Basic local installation and synthetic auditing require none of these accounts. Human administration, machine retrieval and destination publication are separate identities.

## Free account routes

| Provider | Confirmed setup route | Required free-plan check |
| --- | --- | --- |
| Doppler | https://dashboard.doppler.com/register?plan=developer | Choose Developer, not Team trial. Verify current user/token quota; activity history is not automatically read auditing. |
| Infisical | https://app.infisical.com/signup | Choose Free, not Pro/Advanced trial. Count humans and machines toward the identity allowance. |
| Bitwarden Secrets Manager | https://bitwarden.com/help/sign-up-for-secrets-manager/ | Create/use a Free organization, enable Secrets Manager through its subscription settings, and verify the $0 selection. Avoid the marketed enterprise trial. |
| Akeyless | https://console.akeyless.io/registration | Follow https://docs.akeyless.io/docs/account-quickstart; activate email and confirm Secrets Management Free, not the Password Manager limits. |

The published screens and terms are authoritative. If a free entitlement cannot be confirmed, stop the connected test and report it unqualified. Do not add billing information, accept automatic paid conversion, or share privileged identities merely to fit a quota.

For every test:

1. Register/verify the account through the official route and configure supported MFA/recovery.
2. Create an isolated `cquay-audit` project/environment with fabricated values.
3. Establish an allowed resource and a separate denied resource.
4. Create a least-privilege machine identity for the allowed resource only.
5. Store its bootstrap credential outside the repository through protected storage; policy contains only its reference.
6. Bind the expected account/project/environment and exact resource in policy.
7. Prove the allowed read and a denied read, with no value in logs.
8. Rotate/revoke the test identity and verify the next access behavior.
9. Inspect native auditing only where the plan actually provides it; record retention and event semantics.
10. Remove test projects/identities and retain a sanitized receipt.

Akeyless documents API-key authentication as temporary/POC-oriented, not the production recommendation. A short synthetic test does not qualify that method for permanent production service use.

## Native clouds

AWS, Azure and GCP connectors are bring-your-existing-account options. No new billable resource or account is required by the free core. Use native workload identity or narrowly scoped federation; select explicit account/tenant/project and resource. Do not fall back to a broad developer CLI login.

AWS retrieval and KMS decryption are distinct permissions. ECS task execution-role injection differs from application task-role access. Azure secret access is distinct from key operations; diagnostic audit settings require explicit setup. GCP secret reads require appropriate Data Access logging. Qualify successful and denied events, not just login.

## Existing secrets managers

OpenBao/Vault KV reads select exact mount/path/field/version. Do not enable dynamic engines or import another manager's data just to perform a read. SOPS wrapping-key backends are a different capability.

Infisical imports/reference expansion, recursive SSM paths, broad SaaS configurations and full 1Password Environments can retrieve more than the selected delivered field. Disable expansion by default and document the actual server-side read grant.

Machine access must not depend on a user confirming a desktop unlock after each reboot. Existing provider sessions may expire; native service authentication and renewal need their own acceptance tests.

## Publication

Destination writes require a separate approved target mapping, identity and owner. Use only mapped names. Retain provider version metadata where possible, conditional writes where supported, and one managed writer per destination resource. Missing source data does not silently delete destinations. No bidirectional synchronization or cache write-back.

Windmill variables/resources, Worker bindings and Netlify runtime values become native deployed snapshots. A source audit event identifies retrieval by the publisher; it does not establish subsequent per-user read auditing at the destination.
