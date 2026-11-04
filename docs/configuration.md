# Policy configuration

CircularQuay policy version 1 declares authority and delivery intent. It never
contains secret values, provider passwords, OAuth refresh tokens, age private
keys, bearer tokens, browser sessions, or a shell-evaluated command string. Keep those
in separately protected operator-managed locations and reference their paths
only where a supported adapter requires one.

The canonical JSON Schema is
[`internal/policy/schema/policy-v1.schema.json`](../internal/policy/schema/policy-v1.schema.json).
The Go loader and semantic validator are authoritative when the schema and
runtime behavior differ. The loader rejects unknown JSON fields, duplicate JSON
keys, trailing JSON values, YAML aliases/custom tags, and YAML unknown fields.
It accepts JSON and YAML. It does **not** accept a top-level `$schema` field,
so do not add one to a policy; editors can associate the schema out of band.

[`examples/linux-deployment.json`](../examples/linux-deployment.json) is a
complete, fabricated Linux reference. It declares one SOPS/age-backed hybrid
workload, an encrypted systemd credential projection, and an optional hosted
MCP authorization surface with a named identity authority and signed repository.
It also declares remote telemetry with an illustrative collector and a local
minimum runtime while disabling release-feed connections. Collector access,
retention and optional credential/CA references use the
[operational configuration contract](operational-configuration.md).
The paths, recipient, executable, DNS names, and
subject are illustrative; provision replacements under the intended operator
and service identities before use. Pin the actual initial signed commit and
enroll its signing trust locally. The reference is
structurally and semantically valid policy, not a claim that those placeholder
paths, identity, systemd unit, provider, or MCP issuer are qualified.

## Canonical governance and lifecycle

The [configuration-governance contract](configuration-governance.md) defines
one tenant policy graph, typed inheritance, defaults, locked settings,
deterministic multi-group resolution and bounded user overrides. Supported
runtime JSON is its effective policy, not a second independently editable
control-plane authority. Credentials remain protected references.

[Lifecycle integration](credential-lifecycle.md) applies that graph to optional
workload OIDC/SPIFFE, posture/Conditional Access, external policy decisions,
SCIM, provider-native rotation/revocation, GitHub scanning and optional
GitGuardian decisions. CLI, API, UI, SDK and MCP share authorization, validation,
approval, revision and required-audit rules. Use only fields supported by the
selected schema/runtime; do not add design fields to an executable example,
invent environment settings or bypass an ancestor lock with local CRUD.

### Physical files and the resolved model

Choose a self-contained file or a master with explicit trusted external
includes; both resolve to one validated policy under the
[canonical governance contract](configuration-governance.md). Includes carry
approved source/version/digest pins, namespaces, source order and conflict
rules, while inherited locks and authority ceilings remain mandatory. A child
source cannot enlarge its own authority. Bound graph loading and reject cycles,
duplicate source/object identities and unresolved conflicts before atomic
activation of the complete generation.

[Corporate GitHub plus protected personal local configuration](configuration-repositories.md#corporate-github-and-personal-local-configuration)
keeps personal values, decryption keys and scoped provenance local; no corporate
upload follows from composition. Refresh/trust and optional offline-cache
rules are explicit per domain. The reference schema and executable commands
below remain version-specific; do not insert unimplemented include fields into
otherwise valid examples.

### Hosted key service initialization

The [first-run contract](management-and-discovery.md#first-run-and-minimal-managed-registration)
resolves CLI and configuration opt-outs before any hosted-service network call.
Both surfaces can disable the default hosted key service; CLI disable overrides
an ordinary configured default, configuration disable applies absent a CLI
selection, and enterprise locks are validated rather than silently bypassed.
Contradictory requirements deny before network. Use only the selected runtime's
implemented flag/schema spelling; this design adds no ignored policy fields.

Initialization follows resolved configuration/discovery/manual selection and
identity authentication, creates only the permitted minimal user/client/tenant
association and enables future key-release-window checks. It does not upload
personal files/keys, enroll enterprise orchestration or create a paid plan.
Self-hosted/offline profiles retain their separate trust; absence of hosted
registration cannot become a fallback to unauthorized access.

## Client-side configuration ownership

All user configuration remains client-side. A browser or native client reads
single/multiple local files or approved URLs through the user's own network,
VPN and source credentials, resolves and validates every supported option, and
saves the result locally. Neither configuration content nor source credentials
may be sent to the hosted backend, a server-side fetch/resolve proxy, analytics
or logs. Client-side explain/provenance follows the same boundary; metadata
must not become a disguised configuration upload.

Browser design supports explicit file permissions and CORS as enforced by the
source/browser. When those prevent access, use a separately authorized native
CLI/SDK or local client integration with equivalent validation and local-save
behavior. Do not bypass CORS with a hosted proxy, ask the server to fetch a
VPN URL, or silently support fewer configuration options. A future browser UI
is design only: expanded custom admin UI/screens remain deferred/unapproved.

The hosted key service stores authenticated profile-to-age-key associations
and necessary enrollment, window/usage and billing records only. Key values
travel through protected authorized storage/retrieval paths, never display or
logging. It is not a configuration authority/store or a resolver for the client.
CLI/config hosted disable is resolved before hosted network; disabled operation
can remain fully client-only under its actual local grants and key custody.

## Loading and filesystem boundary

`cquay --config PATH` loads JSON or YAML. Relative references are resolved
against the policy's directory, not the caller's working directory. The policy
file must be a regular file that is not writable by group or others; keep its
directory controlled by the operator. Bootstrap and identity files are checked
by the protected-file helpers when read: use absolute paths, regular files,
no symlinks/reparse points, owner-only Unix secret permissions, and protected
Windows DACLs. See [credential and identity boundaries](credential-boundaries.md)
for the native-storage and identity-profile boundaries.

Unless `--state-dir PATH` is supplied, cquay stores its private runtime
metadata below the operating system's user configuration directory, in
`cquay/state` (that is, `os.UserConfigDir()/cquay/state`). This location is
separate from the policy and from repository state declared in the policy. The
default fails closed when the legacy `circularquay/state` directory exists or
cannot be inspected; cquay neither deletes nor migrates it. Select and protect
an explicit `--state-dir` to retain the intended isolation boundary.

The policy itself records names, locations, resource selectors, and grants. A
selected `field` minimizes delivery, but does not reduce an upstream provider
permission that reads a larger object. Separate source records where upstream
access boundaries differ.

## Editing a policy locally

`cquay.json` is the default configuration name. `cquay config init [--output
PATH]` creates a protected JSON policy and its synthetic JSON fixture, refusing
to overwrite either. The older top-level `init` command remains equivalent.
Use `cquay --config PATH config validate` to strict-decode and semantically
validate a protected JSON policy without provider access.

`get --pointer RFC6901` returns the selected JSON value and the SHA-256 of the
whole document. `list [--pointer RFC6901]` returns sorted object keys or array
indexes and the same digest. Use that digest with `--expect-sha256` for a
compare-and-swap write:

```sh
cquay --config cquay.json config get --pointer /workloads/demo
cquay --config cquay.json config set --pointer /workloads/demo --value-file - --expect-sha256 "<sha256 from get>"
cquay --config cquay.json config delete --pointer /grants/0 --expect-sha256 "<sha256 from get>"
cquay --config cquay.json config replace --file complete-policy.json --expect-sha256 "<sha256 from get>"
```

`set` accepts exactly one JSON value through `--value-json` or `--value-file
PATH`; `--value-file -` reads standard input. `delete` requires an existing
non-root pointer. `replace --file PATH|-` accepts a complete JSON policy and,
unlike `set` or `delete`, can create a missing policy. Inputs are capped at 1
MiB. A candidate must strict-decode and pass `policy.Parse`; writes lock the
policy, recheck the digest, and atomically replace the protected file. `init`
is not repeatable when its output exists; `get`, `list`, and `validate` are
read-only; formatting and replacing the same complete document have the same
canonical document result, while a repeated delete fails after removal.

The editor does not retrieve provider values or expand paths. It deliberately
accepts JSON only: JSON and YAML can be loaded to run a policy, but YAML cannot
be edited through `config`. Do not pass credential values by `--value-json`;
they are command-line-visible. Put them in a declared provider and use
`--value-file -` for non-secret structured declarations when standard input is
the safer transport.

`cquay --config PATH doctor` checks the selected declaration and declared
executables without retrieving values, networking, or changing services. A
missing or unsafe configuration returns non-zero. `doctor --identity-only`
returns the local kernel-derived grant subject without checking configuration.
`doctor --fix` only rewrites an already-valid protected JSON policy in canonical
format; it is idempotent and does not repair policy meaning, providers,
identities, services, or deployment state.

## Runtime, client and management configuration

`observability` and `updates` are versioned policy objects. Their exact fields,
credential/CA references, finite budgets and examples are in
[operational configuration](operational-configuration.md). Optional identity,
MCP and managed-service administration retain separate grants; a telemetry
endpoint, update feed or local JSON edit does not select a remote tenant.

Bootstrap and CI source overrides select configuration provenance in the
[repository/discovery contract](configuration-repositories.md). SDK convenience
CRUD uses the [portable I-JSON profile](sdks.md#shared-contract); the CLI's raw
file interface supports wider valid policy declarations. Schema validation,
semantic validation, authentication and authorization are distinct checks.

The schema below contains the runtime declarations accepted by the selected
policy version. Broader integration and administration requirements are defined
in [integrations](integrations.md) and [managed keys](managed-key-service.md);
do not encode a design-only setting as an ignored JSON member.

## Root fields

| Field | Type and required state | Meaning and constraints |
| --- | --- | --- |
| `version` | integer, required; must be `1` | Policy contract version. |
| `sources` | object map, required and non-empty | Source ID to source declaration. IDs use lowercase letters/digits, `_` or `-`, start alphanumeric, and are at most 64 characters. |
| `workloads` | object map, required and non-empty | Workload ID to workload declaration; IDs use the same rule as source IDs. |
| `grants` | array, optional | Exact MCP subject/client/workload/operation grants. An omitted list grants nothing. |
| `mcp` | object, optional | MCP listener and trusted authorization-server declarations. It does not configure login or token issuance. |
| `repository` | object, optional | Signed release-source declaration. `sync` invokes its guarded `FetchCurrent` path; native, recovery, provider, and release qualification remain separate requirements. |
| `observability` | object, optional | Explicit OTLP/HTTP metrics, traces and logs with protected collector trust/credentials and finite budgets. See [operational configuration](operational-configuration.md). |
| `updates` | object, optional | Interactive release awareness or disabled checks, with an independently enforced optional minimum runtime. See [operational configuration](operational-configuration.md#release-awareness). |

## `sources.<id>` fields

`type` is required. Valid types are `sops`, `json`, `env`, `openbao`, `vault`,
`aws-sm`, `aws-ssm`, `azure`, `gcp`, `doppler`, `infisical`, `bitwarden`,
`akeyless`, and `onepassword`. Adapter profiles and identity boundaries are in
[provider interoperability](providers.md) and
[credential boundaries](credential-boundaries.md).

Only the field combinations below are accepted; non-empty fields outside an
adapter's listed contract are rejected before provider I/O. `audit_verified`
is an additional declaration accepted for every source. Paths become absolute
relative to the policy during loading.

| Source type | Required contract | Optional accepted fields |
| --- | --- | --- |
| `json` | `path` | `identity: ""` or `none` only. |
| `env` | non-empty `resources` | `path`; `identity: ""` or `none` only. |
| `sops` | `path`, absolute `sops`, and either an age identity with `identity: none`/empty, or Transit `token_file` with `identity: token_file`, `vault_allowlist`, and `vault_keys` | The selected decryption authority only. Age and Transit authority cannot be mixed. |
| `openbao`, `vault` | HTTPS `endpoint`, `mount`, `resources`, `identity: token_file`, `token_file` | `namespace`. |
| `doppler` | `project`, `environment`, `resources`, `identity: token_file`, `token_file` | HTTPS `endpoint`. |
| `infisical` | `project`, `environment`, `path`, `resources`, `identity: token_file`, `token_file` | HTTPS `endpoint`. |
| `akeyless` | `resources`, `identity: token_file`, `token_file` | HTTPS `endpoint`. |
| `bitwarden`, `onepassword` | absolute executable `path`, `project`, `resources`, `identity: token_file`, `token_file` | None. 1Password Environments are unsupported. |
| `aws-sm`, `aws-ssm` | `account`, `region`, `resources`, `identity: ec2` or `ecs` | None. |
| `azure` | HTTPS `endpoint`, `account`, `profile`, `resources`, plus either `identity: managed` without a token or `identity: workload` with `token_file` | None. |
| `gcp` | non-default `account`, `project`, `resources`, `identity: metadata` | None. |

| Field | Type | Required or valid when | Meaning and constraints |
| --- | --- | --- | --- |
| `type` | string | required | One of the source types above. |
| `endpoint` | string URL | required for `openbao`, `vault`, and `azure`; optional where the adapter supplies a default | HTTPS only, host required, no userinfo/query/fragment. |
| `account` | string | provider-specific | Exact cloud account or Azure tenant selector; it is not an interactive account choice. |
| `region` | string | AWS | Exact AWS region. |
| `project` | string | provider-specific | Exact project, including Bitwarden's expected project ID or 1Password vault name. |
| `environment` | string | provider-specific | Exact provider environment. For 1Password it is rejected because Environments are unsupported. |
| `namespace` | string | Vault/OpenBao where applicable | Declared namespace selector. |
| `mount` | string | Vault/OpenBao | Exact KV v2 mount selector. |
| `path` | string | required for local `json` and `sops`; provider-specific elsewhere | Local data path or provider path selector. It contains no secret bytes. |
| `resources` | unique string array | required for nonlocal sources | Exact permitted remote resource names. A binding must match one item exactly. |
| `identity` | enum | optional; adapter-specific | `""`, `none`, `token_file`, `workload`, `profile`, `environment`, `managed`, `ec2`, `ecs`, or `metadata`. Unsupported combinations fail; it does not enable broad fallback. |
| `token_file` | string path | token-backed source only | Reference to a protected bootstrap token; never put the token in policy. |
| `profile` | string | adapter-specific | Declared identity selector, such as an Azure client ID; this is not a developer-profile fallback. |
| `sops` | string path | SOPS source | Trusted SOPS executable path. |
| `age_identity_file` | string path | age-backed SOPS source | Protected age identity-file reference. It must not collide with a configured source data path. |
| `vault_allowlist` | unique string URL array | SOPS Transit metadata only | Explicit HTTPS Vault/OpenBao endpoint allowlist. It is invalid on other source types and cannot be combined with an age identity file. |
| `vault_keys` | unique string array | when `vault_allowlist` is present | Exact Transit key scopes; required with the allowlist. |
| `audit_verified` | boolean | optional, default `false` | Operator declaration that the source's audit capability has been verified. `audit_required` workloads require it; the declaration is not proof by itself. |

`json` and `sops` are local sources: a binding may omit `resource` or set it to
the source path. `env` reads an exact declared variable or dotenv name but does
not expand shell syntax. An explicit binding `version` is accepted only for
`openbao`, `vault`, `infisical`, `aws-sm`, `aws-ssm`, `azure`, and `gcp`.

Cloud resource selectors identify the exact secret, independently of its
version. AWS bindings use the complete canonical Secrets Manager or Parameter
Store ARN, and the returned ARN must match it. Azure responses must identify
the configured HTTPS vault origin and exact secret name. GCP responses must
identify the declared project and secret, including when a binding does not
pin a version. Missing or mismatched identity evidence fails before a value
is selected or delivered. A checksum alone does not identify the secret.

Use canonical provider identifiers in both the source scope and binding.
Partial ARNs and inferred project-ID/project-number or secret-name aliases are
not accepted as substitutes for the declared identity. If a provider returns a canonical
identifier different from an alias used in a request, configure that canonical
identifier explicitly. An omitted version allows the provider to select a
version of the declared secret; it does not permit a different secret.

## `workloads.<id>` and bindings

| Field | Type and required state | Meaning and constraints |
| --- | --- | --- |
| `mode` | `offline`, `online`, or `hybrid`; required | `online` resolves fresh and cannot have a snapshot. `offline` uses only eligible local/static inputs and cannot set `audit_required`. `hybrid` may use an approved static snapshot only for qualifying unavailable failures. Denial, TLS/trust failure, malformed data, and expiry do not authorize fallback. |
| `bindings` | non-empty array, required | Exact output declarations described below. All required bindings resolve before a generation is used. |
| `environment` | object string-to-string, optional | Literal non-secret base environment. Names must be valid and cannot collide case-insensitively with bindings or protected bootstrap names. |
| `command` | string array, optional | Absolute program and arguments for `run`; no shell evaluation. Empty is valid for projection-only policies. |
| `command_sha256` | 64 lowercase hex characters, optional | Exact executable identity check for the declared command. Compute it from the installed reviewed executable; do not use a placeholder digest. MCP `run` requires both a command and this digest. |
| `untrusted_code` | boolean, optional, default `false` | Records that the workload may execute untrusted code. It does not reduce authorization or create a sandbox. A Windows secret-bearing run requires this to be `false`. |
| `directory` | string path, optional | Working directory for the declared command. A secret-bearing run requires a protected working directory, executable directory, and every directory searched through `PATH`. |
| `snapshot` | object, optional | Explicit encrypted offline/hybrid snapshot declaration below. `online` workloads cannot set `snapshot.path`. |
| `targets` | object map, optional | Named one-way projection targets below. All bindings must set `allow_publish`, and each target must set `accept_snapshot: true`. |
| `audit_required` | boolean, optional, default `false` | Requires every source used by the workload to declare `audit_verified`. It also forbids offline mode and snapshot export. |

Each `bindings` entry has these fields:

| Field | Type and required state | Meaning and constraints |
| --- | --- | --- |
| `name` | string, required | Delivered environment/key name. It must be a unique valid environment name and cannot be a protected bootstrap name. |
| `source` | string, required | Source ID in `sources`. |
| `resource` | string, optional for local sources; required for remote sources | Exact permitted resource. No globs, traversal, backslashes, or wildcard syntax. |
| `field` | string, optional | Exact top-level string field selected from a structured provider value. Empty selects the complete string value. |
| `version` | string, optional | Exact supported provider version selector; unsupported source types fail validation. |
| `allow_offline` | boolean, optional, default `false` | Required for every offline binding and for any value intended for an encrypted snapshot. |
| `allow_publish` | boolean, optional, default `false` | Required for every binding when the workload defines any target. It grants no destination by itself. |

For process-environment and dotenv sources, a non-empty `field` selects one
top-level JSON string from the selected variable's value. Missing fields,
non-string fields, malformed JSON, and trailing documents fail instead of
returning the complete variable. Protected bootstrap and process-loader names
are rejected as input resources as well as output names; renaming an input such
as a provider bootstrap token does not grant permission to deliver it.

### `snapshot`

| Field | Type | Meaning |
| --- | --- | --- |
| `path` | string path | Protected local encrypted snapshot destination/source. |
| `recipients` | unique string array | Explicit age recipients for snapshot encryption. Required when creating a snapshot. |
| `identity_file` | string path | Protected age identity used to decrypt the snapshot. |
| `sops` | string path | Trusted SOPS executable for snapshot operations. |

Snapshots are absent when all four fields are omitted. Once configured, `path`,
`recipients`, `identity_file`, and `sops` are all required: the paths must be
absolute after loading, recipients must be non-empty and unique valid age
public recipients, and every binding must allow offline use. An `online` or
`audit_required` workload cannot configure a snapshot. Leased, renewable, or
expired values cannot become an offline snapshot or target publication.

### `targets.<id>`

Target IDs use the root ID rule. `type` is required and is one of `file`,
`systemd`, `windows`, `cloudflare`, `netlify`, or `windmill`.

| Field | Type | Meaning and constraints |
| --- | --- | --- |
| `type` | string | Projection adapter. Platform and provider qualification remains separate. |
| `path` | string path | Required absolute destination for `file`, `systemd`, and `windows`; optional destination binding name for Cloudflare/Netlify; rejected for Windmill. |
| `endpoint` | string URL | Required HTTPS Windmill endpoint; optional HTTPS Cloudflare/Netlify API override; no credentials in the URL. |
| `account`, `project`, `environment`, `resource` | strings | Exact destination selectors; required combinations are listed below. |
| `token_file` | string path | Protected destination bootstrap credential reference for cloud publication. |
| `accept_snapshot` | boolean | Must be `true` for every target, acknowledging it receives a deployed static copy. |
| `service` | string | Native systemd/Windows service name, required when activation is `restart` or `reload`. |
| `activation` | `none`, `restart`, or `reload` | `restart` requires `service`; `reload` is systemd-only. Omit activation and service together when no native action is required. |

`file` accepts only an absolute `path`. `systemd` accepts an absolute `path` and
optional `restart` or `reload` with `service`. `windows` accepts an absolute
`path`, an optional Service-SID `resource`, and optional `restart` with
`service`; reload is rejected. `cloudflare` requires `account`, `resource`,
and an absolute `token_file`. `netlify` requires `account`, `project`,
`environment`, and an absolute `token_file`. `windmill` requires HTTPS
`endpoint`, `project`, `resource`, and an absolute `token_file`. Unsupported
fields for a target are rejected rather than ignored.

For a `systemd` target, `path` is the encrypted credential file and its basename
is the credential name. The consuming unit must be separately configured with
the matching `LoadCredentialEncrypted=` entry. This is a deployment detail, not
a policy field, and it requires native qualification.

## Grants and MCP

Each `grants` entry requires `subject`, `workload`, and a non-empty `operations`
array. `authority`, `client`, and `keys` are optional. Subjects and clients are exact strings,
not patterns; an empty client is local-only and does not match hosted clients.
`workload` must name a declared workload. Policy version 1 accepts only
`list`, `status`, `run`, and `read`. `read` requires a non-empty `keys` array
of exact declared binding names. Keys constrain only `read`; `run` authorizes
the whole enrolled workload. Request cancellation follows the caller's request
context and is not a grant operation.

For named HTTP authorities, `authority` selects an exact entry in
`mcp.authorities`; `client` is required and `subject` is issuer-qualified.
An unnamed local or single-issuer grant cannot be consumed through a named
authority. Neither an IdP role nor a scope creates a local workload grant.

`not_before` and `expires_at` are optional RFC3339 local authorization bounds
on a grant. A grant is active from `not_before` inclusively until `expires_at`
exclusively; when both are present, `not_before` must precede `expires_at`.
They bound only cquay's policy decision. They do not change remote identity
claim validation, provider credential validity, or schedule work. Grants are
additive: an active unbounded grant can authorize an operation even when a
separate matching grant is not yet active or has expired.

For example, a bounded status grant can be expressed without assigning special
meaning to a project or environment name:

```json
{
  "subject": "https://identity.example.invalid|operator-123",
  "client": "reporting-assistant",
  "workload": "reference-api",
  "operations": ["status"],
  "not_before": "2030-01-02T09:00:00Z",
  "expires_at": "2030-01-02T10:00:00Z"
}
```

HTTP access also remains bounded by the verified access token's expiry. Its
deadline caps the request context and is checked again before disclosure or
workload launch after provider resolution. These deadlines stop new access
through CircularQuay; they do not recall plaintext or revoke an upstream
credential already delivered to another process.

`release_authorized` records durable authorization for a plaintext release
before attempting the response. It is not proof that a client received the value;
a final expiry, revocation, or cancellation check can still withhold the response.

| `mcp` field | Type | Meaning |
| --- | --- | --- |
| `resource` | HTTPS resource URL | Public MCP resource identifier with a non-root path, independent of the token audience. Used with `authorities`. Loopback HTTP is allowed for the local resource only. |
| `authorities` | named object, 1–8 entries | Explicit access-token issuers and permission mappings described below. Cannot be mixed with the single-issuer fields. |
| `issuer`, `audience`, `jwks_url` | URLs | Compact single-issuer alternative: all three are required together; `audience` is also the protected resource URL. Its scope names are `cquay:mcp` plus `cquay:list`, `cquay:status`, `cquay:run`, or `cquay:read`. Prefer named authorities for custom audiences or roles. |
| `address` | `host:port` string | Optional listener address. Bind and exposure decisions remain operator-controlled. |
| `origins` | unique HTTPS URL array | Request `Origin` allowlist for HTTP mode. It is an origin check, not a CORS implementation: this profile does not emit browser CORS permission headers. |
| `hosts` | unique string array | Allowed Host values, without wildcard/URL syntax. |
| `audit_path` | string path | Protected local metadata-only audit receipt path. |

An empty `mcp` object is valid. Configuring HTTP requires an independently
qualified issuer, audience, client registration, transport, and isolation
boundary; it is not established by this file alone.

Each `mcp.authorities.<name>` declares:

| Field | Meaning |
| --- | --- |
| `issuer`, `jwks_url` | Exact HTTPS issuer and same-origin public signing-key endpoint; no token-provided key locations or redirect discovery. |
| `trust_store_file`, `trust_store_sha256` | Optional pair for a private-CA issuer: canonical absolute protected PEM bundle and its 64-lowercase-hex SHA-256. When present, the bundle is the authority's complete replacement root store for JWKS retrieval; it does not append to system roots or affect another authority, provider, or CLI. Both fields are required together. |
| `audience` | Exact registered API identifier. It may differ from the public `resource` URL, including an Entra application GUID. |
| `profile` | `rfc9068` for signed `at+jwt` access tokens, or `microsoft-entra-v2` for the explicit Entra v2 API-token profile. Both verify RS256/ES256 signatures, issuer, audience and time claims. |
| `clients`, `subjects` | Optional exact allowlists. A principal must match one entry in each configured list. Empty lists leave that dimension to exact local grants. |
| `required_scopes`, `required_roles` | Optional requirements on every accepted token: all listed scopes and all listed roles must be present. |
| `permissions` | Maps `list`, `status`, `run`, or `read` to `{ "scopes": [...], "roles": [...] }`. Every declared scope and role is required; an omitted operation or empty permission denies it. |
| `sender_constraint` | Optional RFC 9449 DPoP profile for this named authority only, described below. It cannot be selected by the compact single-issuer form. |

No environment or permission label has built-in business meaning. For example,
`permissions.read.scopes = ["secrets:use"]` maps that registered API scope to a
raw-read operation; its exact subject, client, workload and key still need a
matching local grant. Scopes and roles are separate claim dimensions and are
never flattened into one string set. All claim lists are bounded and duplicate
entries are rejected.

RFC 9068 subjects use `issuer|sub`. Entra subjects use `issuer|tid/oid`, where
the canonical tenant and object GUIDs identify the principal; `azp` identifies
the client. Entra's issuer pins the tenant, `ver` must be `2.0`, and an API scope
or role must be present. Register a separate API audience from interactive login
clients. ID-token profiles, opaque tokens, unsupported sender-proof claims,
and tokens for other APIs are rejected.

### Optional DPoP sender constraint

DPoP is an opt-in, per-named-authority sender-constraint profile. The complete
declaration is:

```json
{
  "sender_constraint": {
    "type": "dpop",
    "proof_max_age_seconds": 60,
    "clock_skew_seconds": 5,
    "nonce": "required",
    "replay_store": "local_persistent"
  }
}
```

`type` must be `dpop`; `proof_max_age_seconds` is required and is from 1 to
300 seconds; `clock_skew_seconds` is optional, defaults to zero, and is from
zero to 30 seconds and no greater than the proof age. `nonce` must be
`required`, and `replay_store` must be `local_persistent`. The complete Linux
reference intentionally remains a bearer-policy example; adding this object
changes that named authority's authentication contract.

For an authority with this object, the issuer must mint an RFC 9449 access
token confirmation claim containing only `cnf.jkt`, and that thumbprint must
match the public key embedded in every DPoP proof. Clients send the token as
`Authorization: DPoP <access-token>` and exactly one `DPoP` proof header.
The proof is a bounded signed JWT (at most 16 KiB) with a 16--512-byte visible
ASCII `jti` (excluding quotes and backslashes), a fresh `iat`, the exact HTTP
method, the configured canonical resource URL in `htu`, and `ath` for that
access token. Its embedded public JWK must be P-256 with ES256 or RSA
2048--8192 bits with RS256; its JWK thumbprint must equal the token's `cnf.jkt`.
The request path must equal the configured resource path and contain no query.
The actual request `Host` must equal the configured resource host: an entry in
`mcp.hosts` does not create a DPoP resource alias. If a request uses an absolute
URL, its scheme and host must also equal the configured resource. CircularQuay
still takes the protected resource from policy, not from `Host`, `Forwarded`, or
related proxy headers. A trusted local TLS proxy must preserve the external
canonical `Host`. The replay/nonce binding also includes the named authority's
current policy fingerprint, resource and key thumbprint, so a changed authority
contract cannot reuse an earlier proof decision.

Protected-resource metadata advertises `ES256` and `RS256` when any named
authority uses DPoP. It advertises `dpop_bound_access_tokens_required: true`
only when every named authority requires proofs; mixed resources enforce the
selected authority's own contract.

The first validly formed proof without a nonce receives `401` with
`WWW-Authenticate: DPoP error="use_dpop_nonce"` and a `DPoP-Nonce` response
header. The next proof presents that nonce. A successful proof consumes that
nonce and proof identifier atomically and returns the next `DPoP-Nonce` for
the following proof. Operators and intermediaries must not log access tokens,
DPoP proofs, or nonce values.

The DPoP replay/nonce state is protected persistent state below the selected
`--state-dir`; it survives process restart. Run this profile on one replica
with one local filesystem state directory. Do not reset that directory to
recover a request, share it between hosts, or use this profile for a
multi-host deployment. Its local store does not supply cluster-wide replay
enforcement. The bounded cache permits 4,096 total live records and 64 per
authority/key/resource binding, counting nonces and replay records together;
capacity or persistence failure denies the request. Expired records are pruned.
The server clock and protected state are part of the trusted host boundary.

When `sender_constraint` is omitted, that authority is strict bearer: it
accepts `Authorization: Bearer` only and rejects DPoP authorization, DPoP
proof headers, and access tokens carrying a `cnf` confirmation claim. DPoP
does not add client key custody, login, token issuance, mTLS, or private-key
JWT support to CircularQuay.

The current protected policy is rechecked before provider access and effects.
Changing an authority's trust, audience, required claims or permission mapping
invalidates the verified principal's configuration fingerprint; removing a grant
also denies further authorization checks. This does not recall an earlier
plaintext disclosure. See [identity integration](hosted-identity.md) for issuer
revocation and [qualification](qualification.md) for actual tested scope.

## Pinned configuration bootstrap

`cquay bootstrap` creates a protected JSON configuration from an immutable
public GitHub tree, blob or raw URL. Supply `--config-ref` with the full 40-hex
commit and `--config-sha256` with a digest obtained through a trusted maintainer
channel. The URL's commit marker must equal the ref. `--config-path` is the
repository-relative file path. Equivalent environment settings are
`CQUAY_CONFIG_SOURCE`, `CQUAY_CONFIG_REF`, `CQUAY_CONFIG_PATH` and
`CQUAY_CONFIG_SHA256`; explicit flags take precedence.

Retrieval denies redirects, bounds the response to 1 MiB, verifies the digest,
strictly validates the policy for its destination, and creates it exclusively.
It refuses an existing output. The JSON receipt records source, commit, path,
digest and output path without policy bytes. With no source, bootstrap validates
the existing local configuration. Materialization does not retrieve provider
values, authenticate an operator, execute a workload or enroll a service.

A hash retrieved from the same untrusted location as the document does not
establish maintainer trust. Public configuration cannot select its own initial
trust anchors or silently replace a locally enrolled policy.

## `repository`

All fields are optional only when the entire object is omitted. If `url` is
present, the declaration requires `url`, `directory`, `ref`,
`allowed_signers_file`, and `initial_commit`; `token_file` is optional.

| Field | Type | Meaning and constraints |
| --- | --- | --- |
| `url` | HTTPS URL | Signed repository endpoint, no userinfo/query/fragment. |
| `directory` | absolute path | Private local repository state directory. |
| `ref` | string | `refs/heads/...` or `refs/tags/...`; traversal, hidden, lock, and malformed components are rejected. |
| `allowed_signers_file` | absolute path | Protected local SSH allowed-signers trust file, outside `directory`. |
| `initial_commit` | 40 lowercase hex | Immutable signed bootstrap commit. |
| `token_file` | absolute path, optional | Protected authenticated-fetch token; it is not needed to validate an accepted local release. |

Repository data paths may use `repo:` only for permitted local `json` or `sops`
source data paths. Bootstrap identities, executables, target destinations,
commands, snapshots, and state may not come from that repository. `sync` invokes
the signed repository fetch/current path with its declared trust, quota, and
offline/fallback guards. Treat it as an unqualified deployment path until native,
recovery, and adverse-path qualification is retained.
