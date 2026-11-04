# Cloud-provider validation matrix

CircularQuay reads declared values from cloud secret providers; it does not
turn a developer login, an emulator, or a hosting platform into an authority.
Each provider profile is a separate trust boundary. A configuration must name
the provider, account or tenant, resource, workload identity method, and
version selector. A successful local emulation run is useful adapter evidence,
but it never qualifies a production identity, IAM/RBAC policy, federation,
revocation, audit trail, or provider availability.

This matrix is the implementation and qualification contract for cloud source
adapters. Current implemented status and release evidence remain in
[qualification.md](../qualification.md); this document does not represent an
unimplemented harness or deployment adapter as available.

## Boundaries that must stay separate

| Boundary | Purpose | It cannot establish |
| --- | --- | --- |
| CircularQuay provider adapter | Resolves a declared resource and verifies returned identity/version evidence before releasing a selected value. | Cloud-account authorization, managed-service user authorization, or a provider's durable audit record. |
| Local emulator or synthetic transport | Exercises deterministic SDK request, parsing, error and response-identity paths with fixtures. | IAM/RBAC, workload federation, instance metadata, real KMS custody, expiry, revocation, CloudTrail/Activity Log/Cloud Audit Logs, or service availability. |
| Remote developer integration environment | Uses a dedicated, disposable account/subscription/project and synthetic values to test the real provider. | Production access or an approval to promote a managed service. |
| Managed-service runtime/deployment adapter | Hosts the optional managed UI/API and its own tenant, storage and deployment integration. | A secret-provider identity or permission. It is tested independently from the provider under test. |
| Production acceptance | Exercises the exact deployed artifact against an approved provider environment and records redacted evidence. | Any broader account or platform entitlement. |

All local HTTPS test servers use an instance-specific CA trust bundle. A
harness must not change a machine-wide trust store, disable certificate checks,
or accept a caller-supplied endpoint or credential from an ambient environment.
Every fixture uses synthetic values and is destroyed with its isolated test
environment.

## Current adapter anchors

| Provider | Source type | Current implementation boundary | Relevant code |
| --- | --- | --- | --- |
| AWS Secrets Manager | `aws-sm` | Exact Secrets Manager ARN; account and region are checked through STS. Only EC2 IMDSv2-role and ECS task-role credentials are selected. Profiles, endpoint overrides and ambient key chains are rejected. | [`internal/source/cloud.go`](../../internal/source/cloud.go) (`awsConfiguration`, `awsCredentialsProvider`, `readAWSSecret`) |
| AWS Systems Manager Parameter Store | `aws-ssm` | Exact SSM parameter ARN; same account/region/identity boundary as `aws-sm`; response ARN and version are checked. | [`internal/source/cloud.go`](../../internal/source/cloud.go) (`readAWSParameter`) |
| Azure Key Vault secrets | `azure` | Explicit HTTPS vault origin, exact tenant and secret/version response identity. Only explicit workload identity or managed identity profiles are selected; authority and metadata endpoint overrides are rejected. | [`internal/source/cloud.go`](../../internal/source/cloud.go) (`azureCredential`, `readAzure`, `validateAzureSecretIdentity`) |
| Google Secret Manager | `gcp` | Exact project/secret/version resource. Only declared Compute metadata service-account identity is selected; custom endpoint, token file, profile and metadata-host override are rejected. Payload CRC32C and returned identity are checked. | [`internal/source/cloud.go`](../../internal/source/cloud.go) (`readGCP`, `validateGCPIdentity`, `gcpSecretValue`) |

The pinned SDKs are AWS SDK for Go v2, Azure Identity and Key Vault Secrets
for Go, and `cloud.google.com/go/secretmanager`. Existing synthetic tests use
the generated SDK clients and verify substituted resources, versions, provider
errors and payload integrity; see `internal/source/cloud_*_test.go`.

## Validation matrix

| Provider and service | Local or synthetic route | Free/tool boundary | Remote developer and live acceptance | Identity and minimum privilege |
| --- | --- | --- | --- | --- |
| AWS Secrets Manager (`aws-sm`) | SDK transport fixtures are mandatory. A future opt-in LocalStack harness may exercise `GetSecretValue` with synthetic secrets and explicit static fixture credentials. | [LocalStack's current plan table](https://docs.localstack.cloud/aws/licensing/) lists Secrets Manager, SSM, STS, KMS and IAM for the non-commercial Hobby plan, while Base, Ultimate and Enterprise are commercial subscriptions. The table is service availability, not API parity; the [Secrets Manager page](https://docs.localstack.cloud/aws/services/secretsmanager/) publishes operation coverage separately. Recheck both pages when pinning a harness image or plan. | Use a distinct AWS account and roles with synthetic secrets. Record exact artifact digest, region, declared ARN, caller-account confirmation, expected allow and deny cases, redacted CloudTrail/audit evidence and cleanup. A LocalStack pass cannot qualify STS federation, EC2/ECS metadata behaviour, IAM evaluation, KMS custody or CloudTrail. | Grant `secretsmanager:GetSecretValue` and only required metadata actions on exact declared ARNs. Bind the role trust policy to the real workload identity and exact account/conditions. Do not grant `*`, use developer profiles, or accept long-lived access keys for the runtime. |
| AWS Systems Manager Parameter Store (`aws-ssm`) | Same transport fixture contract. A future LocalStack harness may cover `GetParameter` and exact version/ARN handling using synthetic SecureString values. | LocalStack lists SSM Parameter Store on Hobby and its plan table warns that service listing does not establish feature parity. KMS use in a SecureString path is an additional service interaction, not proof of AWS KMS properties. | Use an isolated account/role and synthetic parameters. Verify both a permitted exact parameter and denied sibling/path, version handling, STS caller account and redacted audit receipt. | Grant `ssm:GetParameter` only for exact parameter ARNs and the minimal KMS decrypt permission when the selected parameter requires it. Keep role trust conditions separate from resource permissions. |
| AWS STS, KMS, IAM and metadata | These are support dependencies of the AWS read path, not CircularQuay secret sources. Synthetic SDK fixtures cover request/error wiring only. | LocalStack currently lists STS, KMS and IAM on Hobby, but its documentation expressly separates service availability from API coverage. No LocalStack result establishes EC2 IMDSv2, ECS task metadata, IAM policy semantics, federation, KMS custody or audit parity. | Live acceptance must use the selected EC2 role, ECS task role, OIDC federation or other real workload form and demonstrate expiry/revocation with provider audit evidence. | Preserve the current explicit credential-provider model. Metadata traffic must stay link-local, proxy-free and redirect-denying. Never let a container, proxy or environment variable redirect metadata traffic. |
| Azure Key Vault secrets (`azure`) | Generated-client transport fixtures and a WireMock-compatible HTTPS fake are suitable for request shape, response identity, version and error contracts. | [Azurite](https://learn.microsoft.com/azure/storage/common/storage-use-azurite) is free and emulates **only** Azure Blob, Queue and Table storage. It is not an Azure Key Vault emulator and cannot qualify this adapter. No Key Vault emulator is assumed. | Use an isolated Azure subscription/resource group/vault with synthetic secrets and a dedicated tenant/client or managed identity. A free Azure account may provide introductory credit, but is not a permanent free Key Vault qualification entitlement; budget and account approval are operator concerns. Validate RBAC/access policy, workload federation or managed identity, expiry, disable/delete/recover behaviour and redacted Activity Log/audit evidence on the real vault. | Give the workload only Key Vault data-plane secret read permission on the named vault and declared secret scope where the selected authorization model permits it. The explicit tenant, client ID and vault origin remain pinned; no `DefaultAzureCredential`, Azure CLI cache, authority-host override or metadata override may become a runtime fallback. |
| Google Secret Manager (`gcp`) | Synthetic protobuf/client tests and a fake gRPC service cover resource, version, CRC32C and error paths. | Google Cloud SDK emulators are service-specific (for example Bigtable, Datastore and Firestore), and the [Firebase Local Emulator Suite](https://firebase.google.com/docs/emulator-suite) covers listed Firebase services. Neither is a Secret Manager emulator. The [Secret Manager documentation](https://cloud.google.com/secret-manager/docs) does not publish an official local emulator route; treat that absence as no emulator support, never as generic GCP emulator coverage. | Use a separate GCP project with synthetic secrets and a dedicated workload principal. Record artifact digest, exact project/secret/version, expected allow/deny and redacted Cloud Audit Logs. Real-provider acceptance is required for IAM, metadata server, federation, short-token expiry and revocation. | Prefer [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation) for external workloads over service-account keys. Bind a workload pool provider's attribute mapping and condition to the exact external workload, then grant the narrow Secret Manager accessor role on the exact secret/resource. Do not rely on Application Default Credentials or user login as a runtime authority. |

## LocalStack: required implementation before use

CircularQuay currently rejects `Source.Endpoint` for both AWS source types,
which prevents an accidental endpoint override from changing an EC2/ECS
workload's authority. LocalStack is therefore **not** a current runtime path.
The existing AWS SDK v2 test configuration's `BaseEndpoint` is test-only
fixture code, not a policy feature.

An implementation may add a separately typed, explicitly enabled **test
profile** only after all of the following are delivered and qualified:

1. A new schema field/profile that is unavailable to normal production AWS
   workloads and cannot be activated through `AWS_*` endpoint or credential
   environment variables.
2. Strict URL parsing: loopback or an explicitly provisioned isolated test
   network only; no URL userinfo, query, fragment, redirects or proxy routing.
   Plain HTTP is limited to an ephemeral loopback harness; remote harnesses use
   HTTPS and the instance-specific CA bundle.
3. Service-specific AWS SDK v2 endpoint wiring. AWS documents `BaseEndpoint`
   and service `EndpointResolverV2` as explicit client configuration in its
   [endpoint configuration guide](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html).
   The profile must set an explicit resolver/client endpoint for Secrets
   Manager, SSM and the required STS call; it must not install a global
   resolver or change normal AWS endpoint selection.
4. Explicit synthetic fixture credentials supplied from a protected test file;
   no shared config, profile, EC2/ECS metadata, web-identity chain or shell
   credentials. The test profile must fail closed if any required endpoint,
   credential file, CA bundle or account/region assertion is missing.
5. Tests proving that a production profile still rejects an endpoint, a
   non-loopback endpoint, environment-driven endpoint/credentials and any
   response whose ARN/account/version differs from policy. Tests must also
   prove that the local path never declares IAM, metadata or federation parity.

The account check needs special handling in this test profile: LocalStack STS
may return a synthetic account identity, so the profile must require the
fixture's declared account and test that it matches exactly. It must never
relax the existing live STS account confirmation.

## Azure and Google development routes

Azure Key Vault needs either a controlled fake for adapter contracts or a real,
isolated vault for integration qualification. For a repeatable remote
developer environment, use reviewed infrastructure as code to create a unique
resource group, vault, identity/RBAC assignment, synthetic secret and cleanup
scope. [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/overview)
templates can carry Bicep or Terraform assets; its template guidance describes
an `infra` directory and explicit resource mappings. Azure CLI, `azd`, ARM and
Bicep are provisioning tools, never implicit runtime credentials.

Google Secret Manager likewise requires either a fake for adapter contracts or
an isolated project for integration qualification. Google documents Workload
Identity Federation for GitHub, GitLab, AWS, Azure, OIDC and SAML sources, with
attribute mappings and conditions to restrict accepted external identities.
The harness must use a separate workload pool/provider per environment, bind
stable repository/workload attributes, use short-lived tokens and record only
redacted token-exchange and Cloud Audit evidence.

The current `gcp` adapter intentionally supports only Compute metadata
identity. Supporting external Workload Identity Federation requires a new,
typed identity profile that accepts a protected, validated external-account
credential configuration and assertion source, pins the expected audience and
provider/resource name, rejects ambient Application Default Credentials, and
uses the official Google authentication/token-source path. It must retain the
existing exact project/secret/version and CRC32C checks. A profile is not
accepted into the configuration schema until this enforcement and its live
federation qualification exist.

The current Azure adapter intentionally supports explicit workload and managed
identities, not `DefaultAzureCredential` or the Azure Developer CLI cache.
`az`, `azd`, ARM and Bicep may provision a disposable test environment under
operator control; they must not become a hidden credential source for the
CircularQuay process.

## Harness plan and evidence

| Priority | Deliverable | Acceptance evidence |
| --- | --- | --- |
| 1 | Keep SDK-level synthetic contracts for every adapter: positive read, malformed response, wrong resource, wrong version, missing value, denied, unavailable and no-secret diagnostic. | Deterministic test receipt; fixtures contain only synthetic values. |
| 2 | Add an AWS-only LocalStack test profile after the fail-closed design above is implemented. Pin image and plan/version in the harness. | LocalStack service/API coverage reference, test receipt, endpoint/credential/CA isolation and cleanup receipt. |
| 3 | Add real isolated AWS, Azure and GCP integration environments using reviewed IaC and provider-native short-lived workload identity. | Per-provider allow/deny/expiry/revocation/audit receipt, resource IDs redacted as appropriate, teardown receipt. |
| 4 | Test managed-service deployment adapters separately: local Netlify Dev/Wrangler or equivalent adapter tests, then scoped preview tests with synthetic tenants/data. | Separate adapter test receipt. It does not count as a provider acceptance result. |
| 5 | Qualify the exact release candidate artifact in the approved production-like environment before promotion. A rebuild, rebrand, dependency/SDK change, policy/migration change or changed provider configuration requires new qualification. | Immutable artifact digest, provenance/SBOM, reviewer approval, migration/rollback/health/audit evidence and recorded provider acceptance results. |

Every live route requires bounded cleanup and backup/restore expectations for
the component it owns, while preserving independent audit evidence. It must
test concurrency, authorization isolation, time expiry and revocation rather
than infer those properties from a green emulator run.

## Operator review checklist

Before accepting a provider integration, verify all of these against current
vendor documentation and the exact proposed profile:

- The provider feature, plan/licence and API operation are available for the
  intended non-production and production use; service-table presence alone is
  insufficient.
- The implementation chooses only a named identity method and rejects ambient
  developer credentials, endpoint overrides and provider-specific metadata
  overrides unless an explicitly qualified profile says otherwise.
- Network origin, custom CA, proxy and redirect handling are policy-controlled
  and fail closed.
- Permissions name the smallest resource set and the workload trust policy is
  constrained by stable workload identity, audience and environment/tenant
  conditions where the provider supports them.
- The exact source identity, returned resource/version, checksum where offered,
  expiry and audit behaviour have live-provider receipts. No secret value,
  token, private key, credential file or unredacted provider response appears
  in those receipts.
