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
