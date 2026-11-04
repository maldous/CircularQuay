# Operating workflows

Use the command vocabulary of the selected qualified executable. Current remote
bootstrap requires both pins even interactively; `flow run` uses local policy or
pinned GitHub retrieval and fails closed when discovery is unavailable.
Enrolled discovery, automatic sign-in and remote administration remain v1
requirements until the selected artifact establishes those capabilities. The product
reference is `docs/commands.md` in the versioned source/native archive; detailed
publisher and enrollment procedures are in `RELEASE.md`. Do not infer a command
or schema field solely from an intended integration requirement.

| Task | Route | Boundary |
| --- | --- | --- |
| Select configuration | `cquay bootstrap --config-source URL --config-ref FULL_COMMIT --config-path PATH --config-sha256 SHA256 --output PATH --non-interactive` | Pins come from a trusted maintainer route. Explicit settings override environment, local policy and enrolled discovery; they cannot widen access. |
| Inspect | `cquay --config PATH config validate`, `doctor`, `plan`; `status --workload ID --json` | Protect paths and minimize output. Validation is not a provider access test. |
| Edit declarations | `config get`, `list`, `set`, `delete`, `replace` with documented pointer/file flags | Validate the complete candidate and use `--expect-sha256` for conditional writes. JSON values are non-secret declarations. |
| Execute | `cquay --config PATH run --workload ID`; `flow run --workload ID --non-interactive` | Declared identities and executable trust apply; runner markers are context, not authentication. |
| Publish a projection | `snapshot --workload ID`, `apply --workload ID --target ID`, `sync --workload ID --once` | Export, target write and native activation are separate grants. An offline copy does not preserve central per-read audit. |
| Inspect updates | `cquay --config PATH update check --json` | Advisory metadata only; no automatic install, restart, prompt or grant. |
| Serve MCP | `cquay --config PATH mcp serve --transport stdio` or the enrolled HTTP profile | Use an absolute verified executable at launch; remote HTTP needs its explicit issuer/resource/TLS boundary. |

For CI, use the same `flow run` entry point in GitHub Actions, GitLab CI and
Azure Pipelines. Select short-lived job/provider identity, exact repository/ref
and audience mappings, verified configuration and explicit environment
precedence. Later-step export requires policy permission and the CI provider's
native mechanism; those steps then possess the values. Never print the job
identity token, export file contents or child environment.

Official SDKs keep local and MCP adapters distinct. Go local uses the linked
core. Python/TypeScript local selects an absolute native executable and its
expected SHA-256 with a clean explicit child environment. MCP callers own the
session, transport, authentication and cancellation. Safe error categories and
bounded public DTOs guide recovery; effectful actions are not automatically
retried after an ambiguous result. Reconcile status first.

The shared configuration convenience profile uses finite, portable I-JSON
values with bounded depth and collections. Raw CLI file operations serve wider
valid policy declarations. Never round a large identifier or loosen response
validation to force cross-language agreement. Redacting secret wrappers does
not prevent a caller from explicitly revealing and copying a value.

For telemetry and updates, edit protected `/observability` and `/updates`
objects with config CRUD, then validate. Do not add unsupported fields or
ambient exporter credentials. Run synthetic collector/feed cases only in an
authorized isolated environment; configuration validation alone proves no
remote delivery. No shutdown/export failure authorizes retry of an already
completed workload.

Managed-key UI/API/CLI/MCP administration uses the same verified subject,
client, tenant and operation ceiling. Manage metadata, scoped grant windows,
revocation and audit through the enrolled interface. Never synthesize a remote
admin endpoint from a local config command or expose key material to a model.
Directory provisioning, site-load protection and application authorization are
separate controls. Runtime observability is independently configurable without
a managed UI.

Use the release's client-install instructions to load/reload/remove the skill
or plugin. An MCP Registry record is discovery metadata, an MCPB is a client
bundle, and a language SDK is an application package; none authenticates the
others. Match identities, versions, contained paths and artifact digests, and
retain the selected client's positive and denial test evidence. Removal from a
client or registry does not revoke an issued provider credential.
