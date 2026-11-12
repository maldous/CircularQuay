# Working on CQ

Read [agent orientation](docs/agent-onboarding.md),
[the session handoff](docs/session-handoff.md), [contribution rules](CONTRIBUTING.md)
and [security policy](SECURITY.md) before editing. Verify the execution host,
repository root, remote, branch, HEAD and dirty tree; preserve existing work.

Use [the product guide](docs/product-guide.md) and
[implementation plan](docs/implementation-plan.md) for the complete v1 North
Star. Use source/help for executable capabilities and
[qualification](docs/qualification.md) for exact tested revisions. A design,
SDK name, skill or schema is not evidence of a deployed or qualified feature.

Coordinate shared writes and publication with the active implementation owner
identified by the current assignment/handoff. Agree disjoint paths; do not
reset another worker's changes or create a second status ledger. Keep transient
results in the existing qualification/handoff evidence, not the product README.

Use synthetic fixtures only. Do not inspect operational secret values, shell
profiles or unrelated credential stores. Do not install into system/service
paths, modify credentials/infrastructure, restart services, publish packages,
merge to `main` or tag a release under a documentation/development assignment.
Existing BAU work has priority. On the shared Foo host, local heavy builds,
resolvers, full tests, containers and audit runs remain held unless the active
owner explicitly releases that resource hold; read-only and bounded static
documentation checks can proceed.

For runtime operation, use the repository-owned
[`cq` skill](integrations/skills/cq/SKILL.md) and its contained references.
Inspect the selected executable's help and MCP catalog. Never invent a tool,
turn local config editing into remote administration, or infer a grant from
an instruction file. Workload execution and raw disclosure need their actual
separate authorization.

Report exact changed paths, purpose, checks and their scope, unresolved gates,
and the owner handoff/next action. Do not claim runtime verification from a
source or documentation check.

## Connected Windmill workspace

For the connected Foo Workspace MCP account, `list_workspaces` currently exposes
only `foo` (Foo). Use `workspace_id="foo"` on supported calls through that
connector and verify its current account binding before dependent work. A project
name, script path or historical workspace label does not select another workspace
or grant access; do not probe another workspace as a fallback.

Check separately exposed fixed wrappers against their own advertised binding and
supported arguments. If a wrapper still advertises `offmenu` as its workspace,
or its actual target cannot be inspected, do not assume it targets `foo`; stop
the dependent operation and report the mismatch. Do not add an unsupported workspace
argument. Preserve historical receipts and project/artifact names, including
OffMenu paths. This account-specific connector routing does not hardcode a
workspace into generic CircularQuay product schemas, examples or fixtures.

## Shared AI operating policy and bootstrap branch

Circular Quay is one of the five dedicated project roles. At startup, read
[the shared UIF worker policy](docs/agents/uif-worker-policy.md) and
[its repository manifest](docs/agents/uif-worker-policy.json) alongside the
project instructions above. Pass the policy version and hash to child agents.

During initial bootstrap, use the guidance on `feature/initial-implementation`
(PR #1), rather than the empty `develop` initialization tree. Verify the actual
branch and revision before work; preserve existing work and coordinate with the
implementation owner. This branch guidance does not authorize merging PR #1,
starting services or treating the product as qualified. When PR #1 is completed
and merged under its established controls, use current `develop` as the normal
baseline described by the shared policy. Publication alone does not prove that
existing agent checkouts or running sessions have adopted these instructions.
