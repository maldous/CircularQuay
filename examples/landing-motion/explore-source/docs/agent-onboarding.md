# Agent orientation

## Fast start for an assigned task

1. Verify the assigned host, checkout, branch, HEAD and dirty tree using the
   commands below. Read the current assignment, repository instructions,
   handoff, contribution rules and security policy; agree paths with the active
   owner before writes. Do not restart a completed workstream from stale task context.
2. Read the [product guide](product-guide.md) for the complete v1 result, then
   only the contracts and source relevant to the assigned boundary. The
   [implementation plan](implementation-plan.md) is acceptance criteria, not
   proof every operation is callable.
3. Separate native source, internal key-service core, synthetic browser preview
   and deployed service. Select the actual executable/SDK/MCP catalog or pinned
   frontend schema before naming a command, field, capability or passed check.
4. Use the existing owner's checkout/process/evidence. For frontend work read
   that checkout's `web/README.md`, `web/coverage.md` and retained interaction
   receipts; `web/` need not exist in the native checkout. Check snapshot hashes
   against the authoritative schema/example before editing or claiming parity.
5. Perform already-authorized bounded work, verify the changed boundary and
   hand back exact paths/hashes, disposition and next action in the existing
   handoff/qualification workflow. Source/static checks, rendered controls,
   representative browser edits, real backend tests and production qualification
   are different evidence. Respect the selected executor's resource admission
   and exact restrictions in the current assignment.

### Standalone product boundary

CircularQuay builds and runs from its own repository with the documented
native, SDK and browser toolchains. No other project checkout, maintenance
service or shared orchestration installation is a product prerequisite.

For each selected operation, bind the exact repository/tree, toolchain and
dependency pins, authorized paths, private data/state roots, reviewer and
publication destination. External automation may invoke the documented product
interfaces under its own authorization; it supplies no additional product
capability, credential, runtime grant or source acceptance.

Keep credentials and customer configuration client-owned and isolated from
unrelated projects. Coordinate disjoint changes with the existing integration
owner and preserve current work and evidence. Qualify each requested provider
integration independently; an optional integration is not a default dependency.

The current source includes `internal/keyservice` with distinct enrollment,
wrap/unwrap and age retrieval, a fixed software JWE adapter using maintained Go
JOSE, and constrained native Transit operations. It requires caller-supplied
authorization, storage and provisioning. It is **not** a hosted endpoint or
integrated public CLI/SDK/MCP enrollment route; do not invent one. The browser
workbench/account/operator surfaces simulate those workflows separately.

Phone reachability is recorded for the existing frontend route. Reuse
that owner's process and authenticated forwarding; this does not authorize a
new public tunnel, firewall change or competing server. Device/origin browser
storage is separate, so phone reachability does not transfer desktop drafts.
An original icon contact sheet and its exact provenance are not a finished
vector/logo deliverable. Use the selected reference, not a guessed replacement.

## Launch and evidence boundaries

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

CircularQuay (CLI `cq`, the letters C,Q) connects existing secret providers
to approved workloads. Start at the [product guide](product-guide.md) for the
North Star; the [implementation plan](implementation-plan.md) defines the full
v1 acceptance contract. This guide routes development work to existing sources
of truth and describes executable boundaries; it is not another work tracker.

## Establish location, authority and current state

Run these harmless checks in the assigned checkout before editing:

```sh
hostname
pwd
git rev-parse --show-toplevel
git remote -v
git branch --show-current
git rev-parse HEAD
git status --short
```

The project remote is `maldous/CircularQuay`. Use the actual assigned checkout;
do not assume a host, path, branch, clean tree or uncommitted patch ownership.
Follow [Gitflow](../CONTRIBUTING.md#gitflow). The current assignment identifies the implementation owner,
publication coordination and pending work. Agree file ownership before shared
writes and preserve other contributors' changes. If prior-task context is
unavailable, the handoff and existing local receipts still provide bounded
context; do not interpret missing context as permission to overwrite work.

| Question | Canonical source |
| --- | --- |
| What should v1 deliver? | [Product](product-guide.md), [implementation plan](implementation-plan.md), [managed service](managed-key-service.md). |
| What exists and is callable? | Selected executable help, CLI dispatcher, SDK public APIs and authenticated MCP catalog; see the boundaries below. |
| What passed for which revision? | [Qualification](qualification.md), [verification](verification.md), retained `.local/qualification/` receipts when present. |
| What is underway and who publishes? | Current assignment and existing owner; retain transient session records outside product source. |
| What gates must remain intact? | [Assurance](assurance.md), [security](../SECURITY.md), [contribution rules](../CONTRIBUTING.md), [release procedures](../RELEASE.md). |

Local receipts may be ignored development evidence. They do not replace the
committed qualification record or qualify a later HEAD/dirty tree. Do not copy
full task histories, invent project/issue associations or create a parallel
controller, status database or qualification ledger.

## Navigate the implementation

| Source | Responsibility |
| --- | --- |
| [`cmd/`](../cmd/) and [`internal/cli/`](../internal/cli/) | Native `cq` / independent `cq-audit` entry points, command parsing and dispatch. |
| [`internal/policy/`](../internal/policy/), [`internal/model/`](../internal/model/), [`internal/jsondocument/`](../internal/jsondocument/) | Policy loading/schema, typed declarations and conditional JSON editing. |
| [`internal/bootstrap/`](../internal/bootstrap/), [`internal/repository/`](../internal/repository/) | Pinned GitHub configuration retrieval and repository trust. |
| [`internal/source/`](../internal/source/), [`internal/resolve/`](../internal/resolve/) | Provider adapters and complete workload resolution. |
| [`internal/keyservice/`](../internal/keyservice/) | Profile-bound core and software/Transit primitives; separate authorization/storage/provisioning dependencies, not a hosted service or public command surface. |
| [`internal/deliver/`](../internal/deliver/), [`internal/platform/`](../internal/platform/), [`internal/state/`](../internal/state/) | Native projection, host protection, durable state and recovery boundaries. |
| [`internal/identity/`](../internal/identity/), [`internal/mcp/`](../internal/mcp/), [`internal/dpop/`](../internal/dpop/), [`internal/transport/`](../internal/transport/) | Identity verification, filtered MCP operations, sender constraints and client trust. |
| [`internal/observability/`](../internal/observability/), [`internal/update/`](../internal/update/) | Bounded telemetry and advisory release checks, separate from required audit and installation. |
| [`internal/assurance/`](../internal/assurance/), [`formal/`](../formal/), [workflows](../.github/workflows/) | Evidence generation, bounded models and hosted qualification. |
| [`sdk/`](../sdk/) and [`integrations/skills/cq/`](../integrations/skills/cq/) | Official application adapters and portable agent instructions. |

Read [architecture](architecture.md), [threat model](threat-model.md),
[runtime boundaries](runtime-boundaries.md) and [the language ADR](adr/0001-language-and-assurance.md)
before changing a trust boundary. Provider authentication, workload permission,
offline export, native publication and plaintext disclosure are separate grants.
Resolve all required inputs before activation. Denial, trust failure and expiry
cannot become hybrid fallback. Required audit precedes protected disclosure;
telemetry cannot waive it. A client can retain plaintext or a distributed private
key, so revocation cannot promise recall.

## Usable commands and prerequisites

These are source-backed routes, not a claim that an executable is installed or
that an artifact passed qualification. Inspect `cq help` / `cq version`
only after selecting and authenticating the executable. The
[command reference](commands.md) defines flags and protected paths.

- Local inspection: `cq --config PATH config validate`, `doctor`, `plan`
  and `status --workload ID --json`. Use synthetic policy; configuration reads
  can expose literal declarations and should not become general diagnostic logs.
- Remote bootstrap: `bootstrap --config-source URL --config-ref FULL_COMMIT
  --config-path PATH --config-sha256 SHA256 --output PATH --non-interactive`.
  The selected GitHub URL and immutable commit must agree; both commit and
  file digest pins are mandatory even interactively. No automatic browser
  authentication, DNS/cloud discovery or service startup follows from this
  materialization command. See [bootstrap prerequisites](configuration-repositories.md#bootstrap-command-prerequisites).
- CI: `flow run --workload ID --non-interactive` requires one recognized runner
  context and local or pinned remote policy. Runner environment markers are
  hints; they do not authenticate a job or automatically exchange its token.
- Configuration writes: `config set`, `delete`, `replace` use validated JSON
  and optional whole-document `--expect-sha256` preconditions. They edit local
  policy under separate operator authority. Read again after a conflict.
- MCP: the source registers `list_workloads`, `get_status`, `run_workload`
  and `read_secret`, filtered by authorization. Discover the actual selected
  server. Managed-key mapping/grant CRUD, remote config administration and
  agent delegation are design requirements, not extra tools in that catalog.
- Updates: `cq --config PATH update check --json` is explicit advisory
  inspection. Startup notices continue without reading input; SDK/MCP/CI never
  prompt or install updates. A separate minimum-version policy can deny runtime
  effects independently of the feed.

The [SDK guide](sdks.md) distinguishes Go linked-core execution from
Python/TypeScript native-process adapters and caller-owned MCP sessions.
Package installation does not authenticate a native executable. The repository
skill and its `agents/openai.yaml` are instructions/UI metadata; no verified
client-plugin manifest, App binding, public listing or managed-service endpoint
is established by them. See [agent integration](agent-integrations.md) for the
package contract and [release procedures](../RELEASE.md) for publishing.

## Lifecycle design and executable capability

Read [canonical configuration governance](configuration-governance.md) and
[credential lifecycle](credential-lifecycle.md) for hierarchy/defaults/locks,
SCIM, provider-native remediation, GitHub/GitGuardian gates and secret-free
OTel export. These are full-v1 design requirements. The current CLI dispatcher,
workload MCP catalog and runtime schema do not supply new SCIM, scanner-gate,
rotation/revocation or exposure-exception administration merely because the
requirements are documented. Optional OIDC/SPIFFE, device posture/Conditional
Access and external policy engines likewise require their selected profile and
implementation evidence; enabling a mode name is insufficient. Do not invent
verbs, tools, JSON fields or direct provider calls to fill that gap; inspect the selected revision and its receipts.

Preserve separate account/operation grants, exact credential-version bindings
and durable exposure state. No account connection, purchase, scan transmission,
incident update or issuer mutation is authorized by this guide. Directory
removal, scanner closure, issuer revocation and verified rotation differ;
OTLP export is not immutable audit. Record capability gaps in the existing
qualification/handoff workflow while retaining the coherent product contract.

### Configuration composition and privacy

The intended contract supports a self-contained policy or a master with trusted
external includes, resolving to one model. Inspect the selected schema/loader
before adding include syntax or suggesting a compose/explain command; the
one-document policy reader does not substitute for a composition resolver or
establish include support by itself. Implementation owners retain resolver/schema/tests
and their revision-bound evidence. See
[composition and corporate/personal acceptance](configuration-repositories.md#single-file-or-master-and-includes-configuration).

Never concatenate arbitrary JSON/YAML, choose a last-writer-wins merge or upload
a personal local file to make a remote interface appear capable. Enforce source
pins/trust, namespaces, enterprise locks, typed conflicts/multi-membership,
finite loading and complete atomic activation. Keep personal values, keys,
credential references and unauthorized provenance outside corporate services,
logs, explain and telemetry. Optional offline cache and version refresh require
explicit domain policy; unavailable corporate input grants no production access.

### First-run evidence and hosted-service boundary

The latest explicit [hosted key service](managed-key-service.md) decision
supersedes age-only baseline: genuine hosted KMS plus authenticated age-private-key
service, with BOTH associations automatically provisioned by authorized enrollment,
with minimal safe metering/free allowance and an explicit upgrade opportunity.
Enterprise orchestration, exact-bundle remote authorization and optional integrations are not
assumed base-subscription requirements or costs. The
[first-run contract](management-and-discovery.md#first-run-and-minimal-managed-registration)
initializes it after config/discovery/manual selection and authentication unless
disabled via CLI or config. Both opt-outs must be resolved before any hosted
network call; validate enterprise locks and fail before network on a conflict.

KMS configuration is defined per profile.
Read the [custody and hybrid contract](managed-key-service.md#per-profile-age-and-kms-custody);
hosted KMS/dual provisioning is required but not established by the current adapter. Current
bootstrap/usage source does not prove this full initialization behavior, hosted
endpoint or opt-out tooling. Implementers must supply actual schema/command and
runtime test evidence. Do not connect accounts, upload personal material or
infer production permission from this product contract. Future key-release
windows cannot expire or remotely revoke a retained age private key.

[The customer meter](managed-key-service.md#profile-utilization-and-fair-metering)
counts hosted age-key access events: independent host fetches count separately,
local reuse counts zero, and a logical access/retry sequence is charged at most
once after durable release commit. Denied/precommit failures are not retrieval
uses. Do not equate profile count/key slots to consumption or present conditional
forecast margin floors or conditional plan proposals as launched prices/profit.
Cost models are planning inputs. They do not authorize account, payment, vendor
or deployment changes, and they do not establish an unsupported mode.

### Client-only configuration and hosted state

All files/URLs/config options resolve, validate, explain and save locally on the
browser/native client, through the user's VPN/network/source credentials.
Never upload content/source credentials to the hosted backend/proxy/analytics/
logs, even for corporate sources. Browser CORS/file permissions are not solved
with a server proxy; use permitted native CLI/SDK/local client integration.
Hosted state is only profile-to-KMS/age association/retrieval and necessary
enrollment/window/usage/billing. No key values are logged/displayed. CLI/config
disable resolves before hosted network for fully client-only use. KMS profile
configuration stays client-side, distinct from hosted age-key registration.
Automatic dual association provisioning is required product behavior once
authorized and implemented, not permission for live paid setup in this assignment.
It does not download offline keys or activate hybrid. No invented flag follows.
This remains design and adds no authorization beyond separately bounded
synthetic/mock previews and loopback local-configuration simulations.

### Migration and full-service economics

Local-to-managed migration and hosted KMS inherit the project's existing
SECURITY/threat-model/security-profile/administration/assurance definitions.
Read the [complete migration contract](managed-key-service.md#end-to-end-local-to-managed-migration)
before changes; mode/enrollment is not migration or key-upload permission.
Actual backend/hardware/import/format and interrupted recovery evidence must
exist separately from mocks. The revised full KMS+age forecast replaces earlier
age-only/native-KMS/labor-loaded prices for this question. It now shows hosting
only, product-tier workloads, total allocated versus already-paid incremental
cost, active-DB and allowance inflections. No labor, paid external KMS, CRM/
payment or customer margins are embedded. Unknown controls cannot be waived.

### UI authorization boundary

Production custom administration UI remains unapproved while the baseline
stabilizes. The confirmed [landing/client-SPA design](managed-key-service.md#explicitly-confirmed-ui-design-and-bounded-preview-work)
authorizes a separate frontend checkout for synthetic/mock functional user,
account, operator and administrator previews, plus loopback dashboard
local-configuration edits and simulations. It does not authorize a real backend
or host proxy, operational keys, provider/account access, production deployment,
purchase or a new suite. Do not describe preview work as a delivered service.
Reuse Cloudflare/Netlify services where appropriate and retain the CLI/text
first-run authentication flow.

### Customer service versus provider administration

The [account service](managed-key-service.md#end-customer-accounts-subscriptions-and-support)
is an explicit subsequent design capability: own account, profile-successful-
fetch usage/history, invoices, plans/quotas/subscription status and support.
Cloudflare/Netlify operator dashboards/bills are not customer billing/CRM.
Never expose operator API keys, accept another customer's provider ID or collect
payment-card fields. Reuse actually entitled customer portals/support services.
Machine financial actions require explicit user-approved boundaries and any
applicable confirmation; no automatic charge/upgrade follows from enablement.
Current docs establish no implemented billing command or connected provider.
Configuration remains local. Synthetic/mock account previews may be developed
separately; production account UI remains unapproved.

The [restricted operator console design](managed-key-service.md#restricted-remotely-hosted-operator-administration)
is not the customer billing portal or local config SPA. Scoped support may see
public key metadata, activity/utilization and justified coarse geography, never
routine plaintext private keys or client config. Read-only defaults, strong
operator auth, privacy/retention, independent maintenance authority and audit
apply. Reuse provider consoles; synthetic/mock operator previews remain
separate from unapproved production UI. ISO27001 alignment
is not certification. No live customer-data access follows from this design.

[Server account RBAC](managed-key-service.md#account-rbac-and-scoped-authority)
requires role+scope assignment tuples, no blanket support/customer access and
one protected global application superuser by design. AI agents have distinct
bounded identities; no role grants routine key bytes or bypasses financial
confirmation. [Enterprise selected-fleet work](managed-key-service.md#customer-enterprise-administration-and-selected-fleet)
needs actual reachable provider/agent capability and explicit profile scope.
No fleet-agent/admin command was verified in native source; no backend config
broker, VPN proxy or live fleet action is authorized by this design.

## Set up and verify a bounded change

Read [getting started](getting-started.md) for synthetic setup. Native source
builds use the workflow-pinned Go toolchain (currently 1.27.4; verify the current
[source workflow](../.github/workflows/source.yml) before selection). SDK
runtimes, development dependencies, lock/provenance and scripts are defined in
[`sdk/typescript/package.json`](../sdk/typescript/package.json),
[`sdk/python/pyproject.toml`](../sdk/python/pyproject.toml), their guides and the
[SDK workflow](../.github/workflows/sdk.yml). Do not regenerate locks or install
dependencies merely to orient yourself.

[AGENTS.md](../AGENTS.md) preserves the shared Foo resource hold and BAU priority.
While it applies, use read-only/source/static checks and owner-coordinated hosted
evidence. Builds, dependency resolution, full test/audit runs and containers are
not orientation steps. When the owner releases the hold for an isolated synthetic
change, the documented native build is `go build -o ./cq ./cmd/cq`;
a focused package test uses `go test ./internal/NAME` for the affected package,
with the selected toolchain, protected temporary files and Unix `umask 077`.
These are not substitutes for the full required gates.

For a documentation change, check local paths/anchors, balanced Markdown fences
and disclosure tags, JSON/YAML metadata and schema-backed examples, and run
`git diff --check`. Inspect rendered output when available; distinguish syntax
rendering from browser/client visual validation. For code, use meaningful
positive/denial/failure tests and the applicable pinned analyzers described in
[assurance](assurance.md) and [integration testing](integration-testing.md).

Hand back exact paths, revision/tree or scoped file hashes, check results and
limitations, remaining gates and the next owner action. Coordinate commit/push
with the implementation owner. Keep transient CI counts in qualification/handoff,
keep the PR draft until gates pass, and do not infer deployment authority from
local development success.
