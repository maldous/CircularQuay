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
MCP authorization surface. The paths, recipient, executable, DNS names, and
subject are illustrative; provision replacements under the intended operator
and service identities before use. It has no repository block because signed
repository fetching is not yet wired into the CLI sync path. The reference is
structurally and semantically valid policy, not a claim that those placeholder
paths, identity, systemd unit, provider, or MCP issuer are qualified.

## Loading and filesystem boundary

`cquay --config PATH` loads JSON or YAML. Relative references are resolved
against the policy's directory, not the caller's working directory. The policy
file must be a regular file that is not writable by group or others; keep its
directory controlled by the operator. Bootstrap and identity files are checked
by the protected-file helpers when read: use absolute paths, regular files,
no symlinks/reparse points, owner-only Unix secret permissions, and protected
Windows DACLs. See [credential and identity boundaries](credential-boundaries.md)
for the current native-storage boundary.

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

## Root fields

| Field | Type and required state | Meaning and constraints |
| --- | --- | --- |
| `version` | integer, required; must be `1` | Policy contract version. |
| `sources` | object map, required and non-empty | Source ID to source declaration. IDs use lowercase letters/digits, `_` or `-`, start alphanumeric, and are at most 64 characters. |
| `workloads` | object map, required and non-empty | Workload ID to workload declaration; IDs use the same rule as source IDs. |
| `grants` | array, optional | Exact MCP subject/client/workload/operation grants. An omitted list grants nothing. |
| `mcp` | object, optional | MCP listener and trusted authorization-server declarations. It does not configure login or token issuance. |
| `repository` | object, optional | Signed release-source declaration. Current policy validation supports it, but the CLI does not yet fetch it during `sync`; do not rely on it for deployment. |

## `sources.<id>` fields

`type` is required. Valid types are `sops`, `json`, `env`, `openbao`, `vault`,
`aws-sm`, `aws-ssm`, `azure`, `gcp`, `doppler`, `infisical`, `bitwarden`,
`akeyless`, and `onepassword`. Current adapter and identity limits are in
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

## `workloads.<id>` and bindings

| Field | Type and required state | Meaning and constraints |
| --- | --- | --- |
| `mode` | `offline`, `online`, or `hybrid`; required | `online` resolves fresh and cannot have a snapshot. `offline` uses only eligible local/static inputs and cannot set `audit_required`. `hybrid` may use an approved static snapshot only for qualifying unavailable failures. Denial, TLS/trust failure, malformed data, and expiry do not authorize fallback. |
| `bindings` | non-empty array, required | Exact output declarations described below. All required bindings resolve before a generation is used. |
| `environment` | object string-to-string, optional | Literal non-secret base environment. Names must be valid and cannot collide case-insensitively with bindings or protected bootstrap names. |
| `command` | string array, optional | Absolute program and arguments for `run`; no shell evaluation. Empty is valid for projection-only policies. |
| `command_sha256` | 64 lowercase hex characters, optional | Exact executable identity check for the declared command. Compute it from the installed reviewed executable; do not use a placeholder digest. MCP `run` requires both a command and this digest. |
| `untrusted_code` | boolean, optional, default `false` | Records that the workload may execute untrusted code. It does not reduce authorization or create a sandbox. |
| `directory` | string path, optional | Working directory for the declared command. |
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
array. `client` and `keys` are optional. Subjects and clients are exact strings,
not patterns; an empty client is local-only and does not match hosted clients.
`workload` must name a declared workload. Policy version 1 accepts only
`list`, `status`, `run`, and `read`. `read` requires a non-empty `keys` array
of exact declared binding names. Keys constrain only `read`; `run` authorizes
the whole enrolled workload. Request cancellation follows the caller's request
context and is not a grant operation.

| `mcp` field | Type | Meaning |
| --- | --- | --- |
| `issuer`, `audience`, `jwks_url` | HTTPS URLs | All three are required together for hosted HTTP authorization. `audience` is the protected resource URL and must include a non-root path such as `https://mcp.example.test/mcp`. They declare a trusted issuer/resource/key set; CircularQuay does not run login or issue tokens. |
| `address` | `host:port` string | Optional listener address. Bind and exposure decisions remain operator-controlled. |
| `origins` | unique HTTPS URL array | Allowed browser origins, if HTTP mode is used. |
| `hosts` | unique string array | Allowed Host values, without wildcard/URL syntax. |
| `audit_path` | string path | Protected local metadata-only audit receipt path. |

An empty `mcp` object is valid. Configuring HTTP requires an independently
qualified issuer, audience, client registration, transport, and isolation
boundary; it is not established by this file alone.

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
commands, snapshots, and state may not come from that repository. Although the
package validates signed repository declarations, `sync` does not yet invoke
fetch/current resolution; treat the block as an unsupported deployment path
until that integration and its quota/recovery qualification are complete.
