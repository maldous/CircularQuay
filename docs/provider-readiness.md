# Provider readiness map

This map describes the compiled provider readers at source commit
`c7627a0e320861c2ed3f63d42e711b9fae2e0639`, tree
`29a9642f42d4fde20cc111a066801e7c94ba2ad5`. It is a source snapshot,
not an installed-artifact, account-connectivity or release qualification
receipt. Recheck the selected revision when wiring an embedding application.

[Provider interoperability](providers.md) defines selector and composition
boundaries. [Credential profiles](credential-boundaries.md#provider-and-workload-identity-profiles)
define the intentionally excluded authentication paths.
[Qualification](qualification.md) records revision-scoped execution evidence;
[the cloud validation contract](integrations/cloud-provider-validation.md)
defines the separate synthetic, live-provider and deployment requirements.

## Constructor and consumer boundaries

[`source.NewRegistry()`](../internal/source/providers.go) delegates to
`NewRegistryWithClient(nil)`. Construction registers the local, hosted,
vendor and cloud reader functions without making network calls. It does not
authenticate a provider, inspect the configured credential files, probe
executables, renew sessions or establish readiness.

`NewRegistryWithClient(client)` injects the supplied HTTP client into the
OpenBao/Vault and hosted-manager paths that capture its transport client.
The cloud readers create their own SDK clients and transports; the vendor
readers invoke fixed executables. This injection point is not a universal
cloud/vendor transport override.

The default registry does **not** register SOPS.
[`application.engine`](../internal/cli/cli.go) constructs the registry and
explicitly adds `Register("sops", source.SOPSReader{})` before supplying it to
[`resolve.Engine`](../internal/resolve/engine.go).
An embedding that constructs a registry directly must make its intended
SOPS registration explicit. Configuration cannot load executable plugins;
`Register` is a compiled-code integration boundary.

`cq providers` prints the static `source.Capabilities()` list, including
SOPS. Its `read`, `versioned` and `maturity` fields describe implementation
boundaries. The list neither inspects a particular registry instance nor
performs credential, executable, network or live-account checks. In particular,
a capability row is not evidence that a directly constructed registry has
SOPS installed or that a configured source can currently be read.

At request time `Registry.Read` checks cancellation, exact source/resource
scope and version capability before dispatch, rejects an absent reader, and
discards an adapter result if cancellation is observed after the read.
The resolver then checks value bounds, expiry and required provider-audit
metadata before recording a generation. These checks do not turn a read grant
into publication, offline-export or raw-disclosure authority.

## Reader prerequisites and version evidence

The following are implementation prerequisites and response checks, not a
claim that the named external dependencies or provider permissions have been
provisioned. `Binding.Field` selects an output value; it does not narrow the
upstream account or object read grant.

| Source type / implementation | Required selected input | Version and lifecycle behavior |
| --- | --- | --- |
| `json` / [`readJSON`](../internal/source/json.go) | Absolute protected file; JSON object with an exact top-level string field (`Field`, otherwise binding name). | Explicit version selectors rejected. Returned file modification-time/size is an observation only; equality cannot suppress refresh or publication. Duplicate keys, non-object documents and non-string selected values deny. |
| `env` / [`readEnvironment`](../internal/source/json.go) | Exact allowed resource name; process environment when `Path` is empty, otherwise a protected literal dotenv file. Protected bootstrap/environment names deny. | Explicit version selectors rejected. Dotenv file metadata is observational; process environment has no revision. Dotenv does not execute shell/profile code and rejects expansion syntax and duplicate names. |
| `sops` / [`SOPSReader.Read`](../internal/source/sops.go) | Explicit registration; protected encrypted input and absolute trusted SOPS executable. Age-only input uses its declared identity file; Transit-only input requires declared token, HTTPS Vault allowlist and engine/key allowlist. Metadata is checked before supplying the token. | Explicit version selectors rejected. Encrypted-document digest is an observation. Mixed age/Transit recipients, cloud KMS, PGP and key groups deny. Isolated subprocess environment, finite timeout and bounded output do not establish a live Transit or native-key custody receipt. |
| `openbao`, `vault` / [`readVault`](../internal/source/http.go) | Declared endpoint, mount, resource, optional namespace and protected token file; KV v2 data path and string field. | Optional positive numeric version is requested and checked against returned metadata. Deleted/destroyed versions deny. Lease ID, renewable flag and positive lease duration are retained as lifecycle metadata; no lease-renewal operation is implemented here. |
| `doppler` / [`readDoppler`](../internal/source/http.go) | Protected token file, exact project/config and named resource; declared endpoint or compiled default. | Explicit version selectors rejected. Adapter selects the returned raw value; no session exchange, renewal or CLI fallback cache. |
| `infisical` / [`readInfisical`](../internal/source/http.go) | Protected token file, exact project/environment/path/key; canonical explicit path. Imports and reference expansion are disabled. | Optional unsigned numeric version is sent and checked against returned version. Hidden or absent value denies. Universal Auth/OIDC exchanges and session renewal are separate, unsupported identity profiles for this reader. |
| `akeyless` / [`readAkeyless`](../internal/source/http.go) | Protected token file and exact item; declared endpoint or compiled default. Fixed `get-secret-value` request. | Explicit version selectors rejected. No token exchange, dynamic-secret lifecycle or provider administration is supplied. |
| `aws-sm` / [`readAWSSecret`](../internal/source/cloud.go) | Exact Secrets Manager ARN with declared account/region; explicit EC2 or ECS workload identity. STS caller account must match. Profiles and endpoint overrides deny. | Optional version ID or `stage:NAME`; returned ARN and version/stage evidence must match. Binary secret responses are unsupported. Selected string is literal unless an exact JSON field is requested. |
| `aws-ssm` / [`readAWSParameter`](../internal/source/cloud.go) | Exact SSM ARN with the same account/region and EC2/ECS identity constraints. | Optional numeric version or label; response must provide the exact ARN plus matching numeric version or returned selector. Requests decryption; this is not separate KMS service qualification. |
| `azure` / [`readAzure`](../internal/source/cloud.go) | Explicit HTTPS vault origin, tenant/account and client/profile. Workload identity requires protected token file; managed identity uses declared client ID. Authority and metadata overrides deny. | Optional secret version is checked against returned HTTPS vault/secret/version identity. Returned expiry is retained when present. Keys and secrets remain separate APIs; no developer CLI fallback. |
| `gcp` / [`readGCP`](../internal/source/cloud.go) | Exact project/secret and non-default declared Compute metadata service account. Custom endpoint, token file, profile and metadata-host override deny. | Omitted version requests latest but requires a concrete returned identity; explicit `latest` selector is rejected. Explicit versions must match; project/secret base and CRC32C are checked. The SDK client is closed after the read. External federation/ADC/key-file profiles are not implemented. |
| `bitwarden` / [`readVendor`](../internal/source/vendor.go) | Absolute trusted vendor executable, protected token file, project and secret ID. Fixed `bws secret get ID --output json`, with only the designated token in an isolated child environment plus platform essentials. | Explicit version selectors rejected. Response project must match and value must exist; revision date is observational. No machine-account creation, desktop unlock or session lifecycle management. |
| `onepassword` / [`readVendor`](../internal/source/vendor.go) | Absolute trusted executable, protected service-account token file and exact `op://vault/item/field`; project names that vault. Fixed `op read REF --no-newline`. | Explicit version selectors rejected. No Environments, Connect, desktop session, `op run` or `op inject`. Both vendor paths use temporary private state, a 30-second timeout, discarded stderr and bounded stdout. |

The version-selector authority is
[`model.SourceSupportsVersion`](../internal/model/model.go), enforced by
`validateVersionCapability` before registry dispatch. A returned file hint,
SOPS digest or vendor timestamp does not make a source version-selectable.
Returned version/resource checks are necessary in addition to sending a
selector upstream.

AWS credentials use fixed link-local EC2/ECS metadata routes; metadata HTTP
is proxy-free and redirect-denying. Azure uses explicit workload or managed
identity constructors. Google uses `google.ComputeTokenSource`, not an
ambient credential chain. These source choices do not prove live identity,
IAM/RBAC, expiry, revocation or provider audit behavior.

## Dependency identity

The source baseline's [`go.mod`](../go.mod) declares Go `1.27.4` and these
direct provider dependencies. These pins describe source dependencies; they
are not installed SDK, provider-server or vendor-CLI qualification results.

| Dependency | Declared version |
| --- | --- |
| AWS SDK core / credentials / EC2 IMDS | `v1.47.5` / `v1.20.13` / `v1.20.5` |
| AWS Secrets Manager / SSM / STS | `v1.50.8` / `v1.81.2` / `v1.51.6` |
| Azure azcore / azidentity / azsecrets | `v1.23.3` / `v1.15.0` / `v1.5.1` |
| Google Secret Manager | `v1.23.1` |
| Google API / gRPC / OAuth2 | `v0.304.0` / `v1.84.0` / `v0.37.0` |

SOPS, age-keygen, `bws` and `op` are external executable inputs, not these
Go module dependencies. Qualify the selected protected binary's identity and
version separately; this map supplies no universal supported version range.
The [SOPS advisory boundary](../SECURITY.md#dependency-advisories) remains
applicable.

## Existing tests versus executed receipts

These are tests present in the selected source. Reading them does not prove
they ran, passed, reached an external dependency or qualified a later revision.

| Existing test source | Boundary exercised by its source |
| --- | --- |
| [`providers_test.go`](../internal/source/providers_test.go) | `TestRegistryAndLiteralSources` covers scope, literal-file/environment handling and version-capability agreement; `TestProviderVersionResponseEvidence` checks response versions/stages/labels. `TestVaultAndHostedContracts` uses synthetic HTTPS; denial/malformed and fixed vendor-command cases use fabricated inputs. |
| [`registry_cancellation_contract_test.go`](../internal/source/registry_cancellation_contract_test.go) | Pre-dispatch cancellation, cancelled environment reads, cancellation during adapter execution, unchanged successful result and unchanged adapter error. |
| [`cloud_test.go`](../internal/source/cloud_test.go) | Invalid authority/credential selection, metadata transport and tenant-token rejection using synthetic input. |
| [`cloud_response_contract_test.go`](../internal/source/cloud_response_contract_test.go) | AWS stage/selector, Azure response version and Google response evidence cannot release a substituted value through generated-client or protobuf fixtures. |
| [`cloud_resource_identity_test.go`](../internal/source/cloud_resource_identity_test.go) | AWS ARN, Azure origin/path/URL shapes and unpinned Google returned-resource identity, including rejection before credential selection. |
| [`sops_test.go`](../internal/source/sops_test.go) | Isolated fake-command age/Transit, metadata-before-token, recipient downgrade, bounded failure and snapshot cases. `TestSOPSRealAgeRoundTrip` is a separate opt-in real-crypto test: it skips unless `CQ_TEST_SOPS` and `CQ_TEST_AGE_KEYGEN` are configured; its presence alone is not a real-SOPS pass. |

The [committed integration status](qualification.md#integration-implementation-and-qualification-status)
records a loopback HTTPS WireMock OpenBao KV v2 suite. The
[exact 208bfe0 hosted receipt](qualification.md#exact-208bfe0-hosted-receipt)
links [WireMock run 41790956340](https://github.com/maldous/CircularQuay/actions/runs/41790956340)
and records all three synthetic jobs passing for publication
`208bfe016a8e00c55d9e2394b2a930eec64cd606`, tree
`386b60dbdc59d95aee63c1ca7dfeee9e2a71368d`.
That is historical revision-scoped synthetic protocol evidence, not a new
inspection of that workflow, a PR47 provider pass or a live OpenBao receipt.
Other failed gates in that receipt remain failed.

The same committed status states that the real issuer/provider container suite
has not run and that LocalStack runtime profiles and external GCP federation
remain unimplemented or unqualified. A planned emulator, an SDK fixture and
a live provider are different inputs. This source-only map adds no execution
receipt.

## Remaining readiness and lifecycle evidence

For an actual selected integration, record the immutable executable/artifact,
provider and dependency versions, exact identity method, redacted resource
scope, required permissions, selected version and lifecycle policy, allow/deny
and cancellation outcomes, and independently verified audit/cleanup evidence
in the existing qualification workflow. Do not include secret values, tokens
or raw provider responses.

Protected token files are operator-provisioned inputs, not automatic enrollment
or renewal. `AuditVerified` is supplied source metadata; copying it into a
returned value does not configure or independently verify provider-native
audit. Resolver expiry checks do not establish token renewal, rotation,
revocation, recovery or a universal background supervisor.

Read-through, required audit, raw disclosure, publication and offline export
retain separate authority. Leased/renewable or expired values cannot become
indefinite snapshots; provider lifecycle and retained-copy limitations still
apply. No registry entry, capability label, test source or historical fixture
pass establishes a hosted managed service, production access or release
readiness.
