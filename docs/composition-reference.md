# Composed configuration reference

CircularQuay accepts one self-contained policy or a JSON master that selects
pinned data-only fragments. The master remains the authority for grants,
trust, integration settings and privileged commands. Resolution happens on
the client and returns one fully validated policy; it does not upload
configuration or source credentials to the hosted key service.

The authoritative definitions are in the [v1 schema](../internal/policy/schema/policy-v1.schema.json).
Native, provider and security-profile qualification remains separately recorded
in the [qualification matrix](qualification.md#following-composition-and-security-design-changes).

## Local synthetic example

The [master](../examples/composition/master.json) selects the
[personal fragment](../examples/composition/personal.json) by its exact SHA-256.
The fragment declares a synthetic JSON source and an offline workload. It
contains no real credentials, executable command or publication target.

Copy the three files from `examples/composition` into a new private local
directory. On Unix, create that directory with mode `0700` and use mode `0600`
for the copied files; on Windows, restrict the directory and files to the
selected user using the documented [native protection requirements](runtime-boundaries.md).
Git does not preserve owner-only read/write permissions, so a checkout alone
does not establish those protections. Then use an authenticated selected
`cq` executable:

```sh
cq --config /absolute/private/example/master.json config validate
cq --config /absolute/private/example/master.json plan
```

PowerShell accepts the same commands with the selected native pathname:

```powershell
.\cq.exe --config C:\Private\Example\master.json config validate
.\cq.exe --config C:\Private\Example\master.json plan
```

The resulting workload is `personal__demo` and its source is `personal__data`.
Changing the fragment requires updating its master digest. An equality lock
asserts its final `offline` mode. `config get/list` inspect the master only;
they do not print the resolved private fragment. `config set/delete/replace`
edit that local master, validate the complete candidate and atomically replace
the master when valid. They never modify included files.

## Master and include fields

| Field | Contract |
| --- | --- |
| `version` | Integer `1`. Unknown fields, duplicate keys and nulls fail. |
| `composition.conflicts` | Exactly `reject`; there is no implicit deep merge or last-writer override. |
| `composition.includes` | Ordered list of 1–16 master-selected descriptors. Dependencies precede their users; graph depth is at most eight. |
| Include `id` | Unique lowercase identifier, at most 24 characters; `root` is reserved. |
| Include `namespace` | Unique lowercase identifier, at most 24 characters; `__` is reserved for qualification. Combined object names must fit the existing 64-character identifier contract. |
| Include `sha256` | Required 64-character lowercase digest of the exact fragment bytes. |
| Include `base_directory` | Required local reference base, relative to the master or absolute. Child data and credential references bind here, independently of where the fragment was stored. |
| Include `file` | Protected local fragment inside its declared base; selects `visibility: local`. It cannot also select a repository or refresh policy. |
| Include `repository` | Existing signed repository declaration: URL, separate local cache, ref, allowed signers and initial commit, with optional protected authentication reference. Selects `visibility: shared`. |
| Include `path` / `commit` | Repository-only safe relative fragment path and exact 40-character lowercase commit. Signature, ancestry, exact commit and file digest all apply. |
| Include `depends_on` | Unique earlier include IDs whose namespaced sources this fragment may reference. No implicit root or sibling access. |
| Include `refresh` | Repository-only, explicitly `online` or `offline`. Online trust/fetch failures never silently select an offline cache. |
| Include `cache_expires_at` | Required future RFC 3339 deadline only for admitting an offline repository configuration. Expired or unavailable verified cache denies; this is not a lifetime lease on an already running consumer. |
| `composition.overrides` | At most 128 explicit root-owned object-property edits beneath an existing source or workload. Each has an RFC 6901 `pointer` and non-null JSON `value`. Arrays are replaced as complete properties, not traversed by these edits. |
| `composition.locks` | At most 128 final equality assertions using the same pointer/value shape. Missing or unequal values deny. Overlapping or duplicate pointers within either list deny. |

Master, fragment and aggregate input limits are 1 MiB per document and 8 MiB
combined. Composed masters and fragments use JSON; existing self-contained
YAML policies retain their separate supported parsing route. The master must
list every source location; fragments cannot add transitive downloads.

Fragments declare `version`, `sources` and/or `workloads`. They cannot contain
grants, MCP, repository, composition, observability or update settings, even
as empty objects. Workload commands, directories, snapshots and targets are
root-owned. Provider data selectors and bounded credential-file references
may be declared; executable references require an explicit trusted master
override. Final validation rejects unsupported provider contracts and invalid
combinations after those edits.

## Privacy and authority

Local fragments contribute private sources. Their configured data/credential
references stay under their approved base, use separate domain custody and
cannot alias corporate configuration, trust material or repository caches.
Private bindings cannot publish or acquire named remote-identity grants;
their workloads cannot have publication targets. Mixed-domain snapshots deny.
Private fragment identifiers, origins and document digests are excluded from
the resolver's exported provenance view. Repository caches for configuration
includes are not provider-value stores; the separate root repository's
`repo:` ciphertext contract retains its own validation.

An equality lock in a protected local master is a local composition constraint.
It does not establish a centrally signed enterprise security profile or prove
that an authorized host administrator cannot replace the master. Strict
deployment additionally requires the [authenticated profile and accepted-generation controls](security-profiles.md#effective-security-context).
Provider authorization, local custody and an authorized consumer's ability to
retain plaintext remain separate boundaries.

See the [corporate GitHub and personal local scenario](configuration-repositories.md#corporate-github-and-personal-local-configuration)
and [canonical governance](configuration-governance.md).
