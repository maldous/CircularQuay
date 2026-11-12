---
name: cq
description: Configure, inspect and run CircularQuay workloads through its CLI, official SDKs or optional MCP connection. Use for bootstrap, policy CRUD, CI flow, telemetry and scoped lifecycle administration; raw disclosure requires its own authorization.
license: Apache-2.0
metadata:
  compatibility: Requires the selected CircularQuay executable or official SDK/MCP adapter, protected policy and state, and dependencies for enabled features.
---

# cq

The `cq` skill connects existing providers to approved applications through
the CircularQuay CLI, official SDKs or an optional MCP connection.
Use the installed release's help and capability contract. Product guides define
the complete v1 end state; release/qualification records establish the exact
available and tested artifact. A skill, plugin, SDK or agent name grants nothing.

## Select the boundary

Route the task before acting: native CLI/official SDK/MCP for actual runtime
operations; the assigned frontend checkout for synthetic browser development;
product contracts for capabilities not implemented by the selected interface.
Reuse the existing owner/process and exact revision evidence. Browser role
selectors and simulated success never grant provider, payment, fleet or key
authority. All client configuration/variables/source credentials stay local.

The current source's internal key-service core and software/Transit adapters do
not create a hosted endpoint, public enrollment command or extra MCP method.
Inspect actual help/catalog, then read only the contained workflow/security
references relevant to the task. For browser work, see the preview boundaries in
[operating workflows](references/operations.md#synthetic-browser-development).

- Use the CLI for operator-selected local policy, bootstrap and workload actions.
- Use an official Go, TypeScript or Python SDK when an application owns the
  integration. Keep its credentials, cancellation and process/session lifecycle
  explicit. Python/TypeScript local adapters require a separately authenticated,
  digest-selected native executable; package installation is not binary trust.
- Use optional MCP only through the configured stdio OS identity or authenticated
  HTTP resource. Discover its actual catalog; do not invent administrative tools
  or assume every client/server release exposes the same operations.

For task-specific commands and prerequisites, read
[operating workflows](references/operations.md). Read
[security boundaries](references/boundaries.md) before delivery, delegation,
raw disclosure, custody changes or incident recovery.

The current repository MCP catalog contains `list_workloads`, `get_status`,
`run_workload` and `read_secret`, subject to authorization filtering. Remote
configuration CRUD, managed-key grant administration and agent delegation are
v1 requirements; they are not tools in that catalog. Use local CLI/SDK policy
editing only under its separate operator authority. Current remote bootstrap
requires an explicit GitHub source, full commit and configuration digest; do
not attempt automatic discovery or sign-in based solely on the product design.

## Inspect before acting

### Shared UIF project routing

CircularQuay's standalone build, run and delivery do not require UIF or another
subproject. UIF is the required general harness only when the optional shared
operational lane is selected; CircularQuay is the secret/config/workload
product, while OffMenu owns its separate household/product/food/stock/meal-data
outcomes and rights/privacy. Do not default a CircularQuay task to OffMenu or share its
credentials, state/data roots or toolchain. Multi-project support remains
required/planned for that lane until actual accepted-source/job/isolation evidence covers the
affected profiles. Preserve existing coordinators, tasks, leases, reviewers,
publication routes and controlled unit handover. CircularQuay migration and OffMenu BAU/
data work proceed in parallel, with no global gate between independent work.
Recover verified canonical record/project/view bindings from the owners; no
guessed IDs, new dispatcher or duplicate issue tree. See
[shared project workflow](references/operations.md#shared-uif-project-profiles).

Use `version`, `providers`, protected `config validate`, `doctor`, `plan` and
scoped `status` as appropriate. These inspect declarations or metadata; do not
retrieve a secret to validate a policy. Configuration reads may expose literal
policy content: minimize output and keep it in the authorized review context.
Unknown settings, unsupported profiles and unavailable capabilities fail closed.

Bootstrap uses local policy first unless an explicit override selects another
source. Authenticate and verify enrolled discovery; a DNS hint, CI variable or
public Git URL is not trust. Use immutable revision/digest pins for unattended
first enrollment. Never read shell profiles or unrelated credential caches.

## Apply the authorized task

Policy CRUD uses validated JSON, RFC 6901 pointers and whole-document SHA-256
preconditions. Re-read after a conflict rather than overwriting another writer.
Changes to grants, tenant mappings, trust, export, snapshots or native activation
need the task's corresponding authority; workload permission does not imply it.

Invoke only exact declared workloads/actions and constrained inputs. Do not
substitute an arbitrary shell, provider URL, executable or mutable script.
Deliver through the runtime and return safe status or allowlisted results.
Caller-controlled code with injected secrets is disclosure-capable.

CLI, UI/API and MCP administration use the same scoped authorization ceiling.
Delegation names grantor, recipient, workload/operation, timing and audit reason;
it cannot widen the grantor's authority. Use only the selected interface's
supported operations. Provider credentials, age private keys and refresh tokens
are bootstrap material, never ordinary model-visible management fields.

## Diagnose without disclosure

Configure telemetry through protected `observability` declarations and explicit
collector trust/credential references. Values, value hashes, tokens, raw errors,
arguments and provider bodies do not belong in logs, metrics, traces or hooks.
Required audit has a separate durability contract. Use synthetic fixtures for
failure, cancellation, overload and redaction checks.

Release awareness is notice-and-continue. `update check --json` is the explicit
inspection route; SDK/MCP/CI sessions never prompt or install updates. A failed
advisory check cannot waive an explicit local minimum version. Verify a new
artifact and obtain installation authority separately.

## Governed lifecycle administration

Read [lifecycle task boundaries](references/lifecycle.md) for canonical
hierarchy/defaults/locks, SCIM, workload OIDC/SPIFFE, posture/Conditional Access,
external policy engines, scanner gates and provider-native remediation. These
v1 requirements do not create tools or fields in the selected release. Use only
its verified capabilities under common validation/authorization/approval/audit.
GitHub/GitGuardian/issuer permissions and protected references remain separate;
no account connection, purchase, scan transmission or live mutation is implied.
Incident closure does not prove revocation or rotation. Warning/exception policy
cannot override issuer denial; a known exposure permits use only through a
separately approved finite exact-version exception, presented with a warning.
GitHub metadata requests require `hide_secret=true` and rejection of unhidden
secret responses. Return allowlisted metadata, never raw alerts or
replacement keys; never infer GitGuardian identifiers from a local secret hash.
OTel export is not immutable audit. Fix forward through approved migrations
without resetting locks, exposure, revocation or replay state.

## Self-contained or composed policy

One canonical model can come from a self-contained file or explicitly trusted
master/includes. Inspect actual loader/schema support before use; never invent
include flags, concatenate files or upload personal local sources to a remote
service. Apply master-approved pins, namespaces, typed order/conflict/group
rules and enterprise locks; a child source cannot enlarge its authority.
Validate the complete bounded graph before one atomic activation. Corporate
and personal grants/keys stay separate, with no personal content or unauthorized
provenance in corporate explain/telemetry. Offline cache and refresh/trust rules
must be explicit; unavailable input grants no extra access.

## Hosted key service initialization

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

For migration from existing local age/OS/self-hosted KMS and encrypted assets,
follow the complete staged workflow in references/operations.md. Mode/enrollment
does not migrate keys. Preserve originals, verify provider import/export and
non-exportable limits, retain separate grants/recovery and qualify actual
backend behavior under existing project assurance. No operational key inventory,
upload/migration, paid setup or lower security standard is implied.

The product's first-run contract checks config, then trusted discovery/manual
fallback, authenticates and initializes the minimal hosted key service unless
CLI or configuration disables it. Resolve either opt-out before any hosted
network call; validate precedence/enterprise locks and deny conflicts rather
than registering anyway. Inspect actual installed capabilities: this contract
creates no flag, schema or tool. Configure KMS per profile and inspect actual
adapter/schema support before use.

Authorized enrollment must idempotently provision BOTH genuine hosted KMS and
age-key associations; this supersedes the earlier age-only hosted baseline.
Current adapter support is not established. Association creation does not fetch
an offline age key or activate hybrid, and grants only approved profile scope. CI uses
idempotent workload-identity enrollment. It grants no arbitrary key or corporate
membership. Do not upload personal config/secrets/keys, import a key, purchase a
plan or enroll enterprise orchestration by implication. Free allowance/upgrade
opportunity has undecided event allowances/prices and no automatic charge. Windows govern
future age-private-key release only; copied keys remain usable. Self-hosted,
offline and disabled-hosted policies retain their separate approved boundaries.

The original age-retrieval unit is one hosted age-key access event, not profile/key count.
Keep association creation and genuine KMS operations separately typed; their
quota/tariff is explicit, not automatically the age-fetch billable event.
Independent twenty-host fetches are twenty uses; local reuse is zero. A logical
access and its automatic retries reconcile one committed event; denied/precommit
failure adds no retrieval use. Never infer consumption from stored key slots.

Production custom admin UI/screens remain unapproved. A separate frontend
checkout may build synthetic/mock functional user, account, operator and
administrator previews and loopback local-configuration simulations. It must not
connect to a real backend/host proxy, operational keys, providers or accounts,
deploy to production, purchase services or add a suite. Reuse existing platform
services and preserve the confirmed CLI/text flow.

All config is client-side: files/URLs via user network/VPN/source credentials,
full resolve/validate/explain and local save. Never send config content/source
credentials to hosted backend/proxy/analytics/logs. Respect browser CORS/file
permissions via permitted native CLI/SDK/local integration, never server proxy.
Hosted state is only authenticated profile→KMS/age and necessary enrollment/
window/meter/billing; protect key values with no display/logging. CLI/config
disable before hosted network supports fully client-only operation. The clarified
KMS capability is distinct from hosted age retrieval. The implemented authorized
enrollment flow must provision both associations; this assignment does not
authorize live paid provisioning or account connection. Never invent a tool. Apply the custody/mode rules in references/operations.md.
Preview authorization does not make custom UI a delivered or production-approved
service.

The interface design includes a public landing page and an authenticated client SPA.
A separate frontend checkout may implement it only as a synthetic/mock functional
preview or loopback local-configuration simulation; production operation remains
unapproved. Client OTel uses
user-selected query/collector endpoints with actual network/VPN/CORS/auth or
permitted local/native integration. No general browser OTLP listener, hosted
telemetry/config ingestion or immutable-audit guarantee follows from it.
Production-grade minimal hosted service needs qualified WAF/bot/DDoS, headless
rate limits, consistency/custody/recovery and measured geo objectives; no plan
label or forecast authorizes provisioning or proves those properties.

End-customer account/usage/history/invoice/subscription/support capabilities
are separate from Cloudflare/Netlify operator administration. Use only actually
supported customer-scoped methods with common authorization, tenant isolation,
idempotency/reconciliation and audit. Provider IDs are not ownership proof.
Reuse hosted payment entry; no card fields, admin keys or portal session URLs
in model logs. Unattended financial actions require explicit user-approved
operation/account/plan/amount/currency/time limits plus applicable confirmation.
Capability is not consent to charge/upgrade. No live provider setup, purchase
or configuration upload follows from these instructions. Synthetic/mock previews
remain bounded to no backend/provider/key/account access and do not authorize
production UI work.

Role permissions and tenant/org/customer/group/environment/fleet/key-metadata/
billing scope bind in one assignment tuple; do not mix one role's actions with
another assignment's broader scope. Individual support sees only assigned
enterprise/customer subsets. The single protected global application superuser
is a distinct design role, not support, shared agent login or raw-key access.
Deny/conflict/time/delegation/revocation and financial confirmation remain.
Customer enterprise admin and selected-fleet work require explicit authorizing
profile+provider+target grants and actual CORS/VPN/agent capability. No central
config broker/storage or invented fleet command is permitted; report conflicts
and unsupported options. Synthetic/mock console previews and loopback
local-configuration simulations are authorized separately; production or
real-backend console implementation remains unapproved.

## Raw disclosure and recovery

`read_secret` deliberately returns plaintext to the client/model context and
may enter its retention systems. It requires explicit disclosure authority for
one exact alias and a current server grant. No bulk export, bootstrap-key read
or alternate-account/cache workaround follows from a denial.

Revocation stops future authorized delivery; it cannot recall plaintext, copied
age private keys or released data keys. Offline expiry depends on trusted local
enforcement and cannot defeat a hostile endpoint administrator. Rotate exposed
application credentials at their issuer under the actual incident scope.
Preserve audit, replay and fallback state. Installation, production changes,
credential rotation, registry publication and history rewriting remain separate
actions with their own owner-authorized scope.
