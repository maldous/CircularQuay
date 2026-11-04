# Agent orientation

CircularQuay (`cquay`, pronounced “see-key”) connects existing secret providers
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
Follow [Gitflow](../CONTRIBUTING.md#gitflow). The current assignment and
[session handoff](session-handoff.md) identify the implementation owner,
publication coordination and pending work. Agree file ownership before shared
writes and preserve other contributors' changes. If chat inspection is
unavailable, the handoff and existing local receipts still provide bounded
context; do not interpret missing context as permission to overwrite work.

| Question | Canonical source |
| --- | --- |
| What should v1 deliver? | [Product](product-guide.md), [implementation plan](implementation-plan.md), [managed service](managed-key-service.md). |
| What exists and is callable? | Selected executable help, CLI dispatcher, SDK public APIs and authenticated MCP catalog; see the boundaries below. |
| What passed for which revision? | [Qualification](qualification.md), [verification](verification.md), retained `.local/qualification/` receipts when present. |
| What is underway and who publishes? | Current assignment and [session handoff](session-handoff.md); keep new scoped evidence in that existing workflow. |
| What gates must remain intact? | [Assurance](assurance.md), [security](../SECURITY.md), [contribution rules](../CONTRIBUTING.md), [release procedures](../RELEASE.md). |

Local receipts may be ignored development evidence. They do not replace the
committed qualification record or qualify a later HEAD/dirty tree. Do not copy
full chat histories, invent project/issue associations or create a parallel
controller, status database or qualification ledger.

## Navigate the implementation

| Source | Responsibility |
| --- | --- |
| [`cmd/`](../cmd/) and [`internal/cli/`](../internal/cli/) | Native `cquay` / independent `cquay-audit` entry points, command parsing and dispatch. |
| [`internal/policy/`](../internal/policy/), [`internal/model/`](../internal/model/), [`internal/jsondocument/`](../internal/jsondocument/) | Policy loading/schema, typed declarations and conditional JSON editing. |
| [`internal/bootstrap/`](../internal/bootstrap/), [`internal/repository/`](../internal/repository/) | Pinned GitHub configuration retrieval and repository trust. |
| [`internal/source/`](../internal/source/), [`internal/resolve/`](../internal/resolve/) | Provider adapters and complete workload resolution. |
| [`internal/deliver/`](../internal/deliver/), [`internal/platform/`](../internal/platform/), [`internal/state/`](../internal/state/) | Native projection, host protection, durable state and recovery boundaries. |
| [`internal/identity/`](../internal/identity/), [`internal/mcp/`](../internal/mcp/), [`internal/dpop/`](../internal/dpop/), [`internal/transport/`](../internal/transport/) | Identity verification, filtered MCP operations, sender constraints and client trust. |
| [`internal/observability/`](../internal/observability/), [`internal/update/`](../internal/update/) | Bounded telemetry and advisory release checks, separate from required audit and installation. |
| [`internal/assurance/`](../internal/assurance/), [`formal/`](../formal/), [workflows](../.github/workflows/) | Evidence generation, bounded models and hosted qualification. |
| [`sdk/`](../sdk/) and [`integrations/skills/cquay/`](../integrations/skills/cquay/) | Official application adapters and portable agent instructions. |

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
that an artifact passed qualification. Inspect `cquay help` / `cquay version`
only after selecting and authenticating the executable. The
[command reference](commands.md) defines flags and protected paths.

- Local inspection: `cquay --config PATH config validate`, `doctor`, `plan`
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
- Updates: `cquay --config PATH update check --json` is explicit advisory
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

The established [hosted key service](managed-key-service.md) is simple
authenticated identifier-to-age-private-key registration/storage/retrieval,
with minimal safe metering/free allowance and an explicit upgrade opportunity.
Enterprise orchestration, remote unlocking and optional integrations are not
assumed base-subscription requirements or costs. The
[first-run contract](management-and-discovery.md#first-run-and-minimal-managed-registration)
initializes it after config/discovery/manual selection and authentication unless
disabled via CLI or config. Both opt-outs must be resolved before any hosted
network call; validate enterprise locks and fail before network on a conflict.

Repository/source review found no `KVM` mode/enum; it is the user's supplied
label, not a verified `--mode` value or a term to silently rename. Current
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
The user authorized cost-derived proposals; actual account/payment/vendor setup
and an unsupported mode remain unauthorized by documentation alone.

### Client-only configuration and hosted state

All files/URLs/config options resolve, validate, explain and save locally on the
browser/native client, through the user's VPN/network/source credentials.
Never upload content/source credentials to the hosted backend/proxy/analytics/
logs, even for corporate sources. Browser CORS/file permissions are not solved
with a server proxy; use permitted native CLI/SDK/local client integration.
Hosted state is only profile-to-age-key association/retrieval and necessary
enrollment/window/usage/billing. No key values are logged/displayed. CLI/config
disable resolves before hosted network for fully client-only use. The user also
said PVM; neither PVM nor KVM has a verified source enum/flag. Keep the naming
uncertainty explicit rather than inventing support. This remains design and
adds no authorization for deferred custom UI.

### UI authorization boundary

Expanded custom administration UI implementation remains deferred while the
baseline stabilizes. Earlier authored screen proposals were not human-approved.
The user has now explicitly confirmed the narrow [landing/client-SPA design](managed-key-service.md#explicitly-confirmed-ui-design-implementation-deferred),
which is design confirmation only, not implementation approval. Do not
start a UI workstream or describe those proposals as human-agreed/built. Reuse
Cloudflare/Netlify services where appropriate and retain the confirmed CLI/text
first-run authentication flow. This does not cancel other explicitly approved
CircularQuay capabilities.

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
Configuration remains local and custom UI implementation remains deferred.

The [restricted operator console design](managed-key-service.md#restricted-remotely-hosted-operator-administration)
is not the customer billing portal or local config SPA. Scoped support may see
public key metadata, activity/utilization and justified coarse geography, never
routine plaintext private keys or client config. Read-only defaults, strong
operator auth, privacy/retention, independent maintenance authority and audit
apply. Reuse provider consoles; custom UI remains deferred. ISO27001 alignment
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
builds use the workflow-pinned Go toolchain (currently 1.27.3; verify the current
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
change, the documented native build is `go build -o ./cquay ./cmd/cquay`;
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
