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
