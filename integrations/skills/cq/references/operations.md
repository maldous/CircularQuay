# Operating workflows

The launch custody profile uses software KMS and age-key custody on Cloudflare
or Netlify. A protected runtime wrapping key stays outside the database, and
each profile has an independently random data key. Scoped authorization,
tenant isolation and required audit apply. Runtime or shared-key compromise
can expose reachable profile keys; software custody does not provide independent
HSM or non-exportability guarantees.

Each deployment must verify provider plans, issuer metadata, entitlements and
required custody. Additional services or upgrades require separate operator
approval. Hosting forecasts exclude labor, CRM/payment fees and customer pricing.
Synthetic frontend previews do not establish a connected backend or production
service. Implementation and qualification evidence must match the selected
release and deployment.

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
| Select configuration | `cq bootstrap --config-source URL --config-ref FULL_COMMIT --config-path PATH --config-sha256 SHA256 --output PATH --non-interactive` | Pins come from a trusted maintainer route. Explicit settings override environment, local policy and enrolled discovery; they cannot widen access. |
| Inspect | `cq --config PATH config validate`, `doctor`, `plan`; `status --workload ID --json` | Protect paths and minimize output. Validation is not a provider access test. |
| Edit declarations | `config get`, `list`, `set`, `delete`, `replace` with documented pointer/file flags | Validate the complete candidate and use `--expect-sha256` for conditional writes. JSON values are non-secret declarations. |
| Execute | `cq --config PATH run --workload ID`; `flow run --workload ID --non-interactive` | Declared identities and executable trust apply; runner markers are context, not authentication. |
| Publish a projection | `snapshot --workload ID`, `apply --workload ID --target ID`, `sync --workload ID --once` | Export, target write and native activation are separate grants. An offline copy does not preserve central per-read audit. |
| Inspect updates | `cq --config PATH update check --json` | Advisory metadata only; no automatic install, restart, prompt or grant. |
| Serve MCP | `cq --config PATH mcp serve --transport stdio` or the enrolled HTTP profile | Use an absolute verified executable at launch; remote HTTP needs its explicit issuer/resource/TLS boundary. |

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

For SCIM, workload federation/SPIFFE, posture/Conditional Access, external policy
engines, exposure gates and issuer rotation/revocation, read
[lifecycle tasks](lifecycle.md). These design requirements are not additional
commands in the table. Discover actual capabilities, retain separate grants,
use canonical locks and common administration checks, and return a bounded
unsupported result for absent tooling. Never add unrecognized policy fields or
turn a local config write into a remote issuer effect.

Configuration may be self-contained or a master with trusted pinned includes;
one resolved model is not a one-physical-file restriction. Verify actual
resolver/schema support rather than adding unrecognized keys or inventing a
command. Explicit order/namespace/conflict rules preserve enterprise locks.
Corporate GitHub + protected personal local composition must not upload personal
content, keys or unauthorized provenance. Domain grants/cache stay separate,
refresh/offline policy is explicit, and every required input validates before
atomic activation. No silent omission, unsafe concatenation or permissive
last-writer-wins workaround is authorized.

The first-run hosted key service contract is config→trusted discovery→manual
fallback→resolved-identity authentication→minimal authorized initialization,
unless disabled via CLI or config. Resolve both opt-out surfaces and locks
before any hosted network. This file supplies no invented flags, registration
tool, endpoint or KMS enum. Use the actual selected release's capabilities;
unsupported tooling is not a reason to connect accounts manually. CI enrollment
is idempotent with workload identity. Latest decision supersedes age-only hosted
baseline: authorized enrollment provisions BOTH genuine hosted KMS and age-key
associations, with distinct operations/windows and safe resource reconciliation.
It downloads no offline key and activates no hybrid mode by itself.
No personal material upload, automatic payment or copied-key revocation is implied.

Customer utilization is successful logical hosted access, not profile/key slots:
one daily hosted fetch=one use, twenty independent hosts/fetches=twenty uses,
local reuse=zero. Denied/precommit failures add no uses; committed uncertain
access reconciles once and same-logical-access retry never adds a second charge.
No idempotency sharing across distinct clients/versions/requests is authorized.
Production custom administration UI/screens remain unapproved; a separate
frontend checkout may construct only synthetic/mock functional previews and
loopback local-configuration simulations. They do not imply delivery or authorize
a real backend/host proxy, operational keys, provider/account access, deployment,
purchase or a new suite.

All config stays on the client: user-owned network/VPN/source credentials,
complete single/multiple-source resolution/validation/explain and local save.
No contents/source credentials reach hosted backend/proxy/analytics/logs.
Browser CORS/file restrictions require permitted native/local CLI/SDK integration,
not a server fetch proxy. Hosted state is profile→KMS/age plus necessary
service metadata; keys are never logged/displayed. CLI/config disable resolves
before hosted network. KMS is configured per profile; no
guessed flag, production UI authority or automatic subscription follows from
this design. The separately authorized previews remain synthetic/mock and
loopback-only.

Customer account/usage/invoice/plan/quota/subscription/support operations require
exact tenant/account/field grants; hosting dashboard/admin APIs are not customer
billing or CRM. Reconcile signed provider events and lost/duplicate mutations
idempotently. Future unattended financial actions need explicit bounded user
authority and applicable confirmation; capability alone never authorizes charges.
Use actual qualified provider-hosted payment entry, not card custody/admin keys
or guessed SDK/MCP/CLI methods. No provider setup/purchase/config upload or production custom UI implementation
is authorized by design documentation. Synthetic/mock previews retain the
separate no-backend/no-provider/no-key boundary.

## Product execution boundary

Build and operate CircularQuay through its documented standalone toolchains
and CLI, SDK or MCP interfaces. Every client must select the exact executable,
source/configuration, authorized operations, private state and result destination.
External orchestration supplies no additional product authority or capability.

Preserve current work, disjoint ownership, review and publication controls.
Isolate credentials, data and state from unrelated projects. Diagnose failures
against the selected product revision and operation; an unavailable optional
integration does not block unrelated standalone work. Deployment/account
configuration and maintenance-session management are outside this skill.

## Synthetic browser development

The assigned frontend checkout may provide interactive local-client, customer
account and operator previews. Read its actual `web/README.md`, package scripts,
coverage matrix and pinned schema/example evidence. That tree may be separate
from the native source checkout; do not assume it is installed or integrated.
Reuse the existing owner/process/phone forwarding, not a competing server,
public tunnel or firewall change. Reported phone reachability does not establish
public deployment or backend authentication.

Use synthetic data only. Browser storage is per device/origin and is not a
protected credential vault. Saved local config, variables, source credentials
and exports must never reach account/operator transport, a hosted fetch proxy,
logs, analytics or review uploads. Mock apply/run launches no process; a demo
role selector, subscription confirmation, key release or fleet action has no
real authority. Native/file/signature/include/provider checks remain separate.

Record source/schema/example hashes, actual interactions and limitations.
Finding a rendered control for every schema path does not exercise every edit,
provider branch or lifecycle failure. Verify save/discard/navigation/re-import
semantics and privacy at the changed boundary. Read current owner-reported
failures before relying on an older screenshot or browser pass. Requalify after
schema/source changes. Exact original brand references/provenance do not imply
a finished vector or permit inventing a different selected asset.

## User-profile KMS and availability modes

KMS is configured per profile. Distinguish
local age, minimal hosted age-private-key retrieval and remote KMS master-key
custody. The inspected SOPS adapter rejects cloud-KMS recipients and key groups;
upstream SOPS support or cloud secret-reader SDKs do not prove adapter support.
Inspect actual schema/help/qualification; unsupported fields must fail rather
than silently use another custody path. Do not invent a flag/tool/provider grant.

A remote KMS profile needs exact provider/account/project/tenant/region/key and
version policy, declared auth identity, minimum operation grants, trusted
endpoints and recovery rules. Credentials/config remain client-side. Keep
hosted KMS and age operations distinct; honor hosted opt-out/locks before network.
Automatic dual association provisioning is required implemented product behavior
under authorized enrollment, not live paid/account setup permission here. No
offline-key release, hybrid activation or purchase follows merely from association.

Custody and availability modes differ. Existing `online` means required online
custody cannot bypass to age/offline. Explicit `hybrid` preserves authorized
encrypted-snapshot fallback on qualifying availability failure within the
approved window, with scope/freshness/denial-fence checks and audit compatibility.
Denial, TLS/trust failure, malformed response and expiry are not availability
failures. Existing `offline` uses explicitly authorized local custody. No silent
mode change or undeclared recipient. Use these same rules through actual
CLI/API/SDK/MCP; synthetic/mock UI previews and loopback local-configuration
simulations are authorized separately, while production or real-backend UI
remains unapproved.

Ordinary SOPS recipients are alternative unlock paths, not AND. Key-group quorum
requires actual supported/tested threshold configuration; it is currently
rejected by this adapter. Remote KMS keeps its master key, but a client can retain
unwrapped data keys/plaintext; an age fallback key permits offline decryption.
Never promise every future decrypt contacts KMS or remotely expires retained keys.

The latest hosted decision requires BOTH associations on authorized enrollment,
not merely optional third-party KMS. The authorized launch uses a SOFTWARE crypto adapter with vetted library/native
Web Crypto, protected KEK outside DB, random profile DEKs and server-resolved scoped
key/context; returning an age private key or reading a runtime secret is not KMS.
No provider or remote/local active-mode default is selected. Native KMS resources,
age recipients, artifact/key versions, independent key IDs/windows and safe
provisioning state need durable tenant-bound idempotency and partial-failure
reconciliation. Do not claim ready with one association missing.

Association creation releases no offline key and adds no age recipient to
online-only artifacts. An explicit authorized hybrid transition needs compatible
ciphertext/recipient or approved snapshot with exact versions/window/scope before
activation; never assume a KMS ciphertext blob is decryptable by age. Reverting
online-only cannot revoke copied old keys/plaintext. KMS may process protected
data keys, but never client config/variables/source credentials. Separate logical
association, crypto-operation and age-fetch events; only explicit tariff policy
can equate or bill them, and retries may still incur provider cost. Current
adapter rejects cloud KMS; this is required behavior, not an implemented tool.

## Local-to-managed migration

Migration is not a mode flag or enrollment side effect. Inspect only explicitly
selected client-local metadata for age/OS/self-hosted KMS definitions, public
recipients, ciphertext/key versions, ownership/profile/grants/windows, recovery
and dependencies. Reconcile unknowns/duplicates before effects; do not scan
unrelated key stores or hash private keys as identifiers. Automatic dual
associations do not authorize local-key read/export/import/upload.

An approved bounded plan distinguishes reuse of a supported reference, exportable
key import under actual provider format/algorithm/permission limits, supported
data-key rewrap and local payload re-encryption. Non-exportable hardware/provider
keys cannot be copied; use an authorized source operation or report unsupported.
Matching raw key bytes does not imply cross-provider ciphertext compatibility.
Preserve originals/provenance. Stage protected local outputs, authenticate and
verify destination plaintext/recipients/version/recovery before atomic activation.
No plaintext temp/log/model output. Durable per-asset migration IDs reconcile
interruption, uncertain creation, concurrent edits and retry without duplicate
paid keys or partial estate success. Recheck permission after waits/before effects.

Online-only cannot gain an offline recipient. Hybrid activation remains explicit,
with compatible artifact/key version, approved window and existing availability,
freshness, audit and denial-fence checks. Rollback/retirement is separately
authorized and preserves accepted generations; copied old keys/ciphertext/data
keys/plaintext remain usable. No actual key inventory or migration is authorized
by this skill design; use only real qualified capabilities, never invented tools.

Existing SECURITY/threat-model/security-profile/administration/assurance gates
apply equally to hosted KMS, local adapters and migration. Do not lower custody,
scopes, durable audit, replay/fallback or issue/coverage/complexity/review gates for
cost. The launch profile uses software custody;
stronger independent non-exportable/HSM claims are unavailable. Real backend/
recovery evidence is distinct from mocks; missing
authorized backend is unqualified, not pass/certification. Cost model must cover
only CF/Netlify hosting: both service workloads, lifecycle/free-owner traffic,
CPU/elapsed/DB/egress and allocated versus already-paid incremental spend with
inflection points. No labor/external KMS/CRM/payment/customer margin embedded.
