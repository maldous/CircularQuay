# Working on CQ

Read [agent orientation](docs/agent-onboarding.md),
[contribution rules](CONTRIBUTING.md) and [security policy](SECURITY.md) before editing. Verify the execution host,
repository root, remote, branch, HEAD and dirty tree; preserve existing work.

Use [the product guide](docs/product-guide.md) and
[implementation plan](docs/implementation-plan.md) for the complete v1 North
Star. Use source/help for executable capabilities and
[qualification](docs/qualification.md) for exact tested revisions. A design,
SDK name, skill or schema is not evidence of a deployed or qualified feature.

Coordinate shared writes and publication with the active implementation owner
identified by the current assignment. Agree disjoint paths; do not
reset another worker's changes or create a second status ledger. Keep transient
execution records outside the product source; commit only product-specific
qualification evidence, not session allocation or maintenance status.

Use synthetic fixtures only. Do not inspect operational secret values, shell
profiles or unrelated credential stores. Do not install into system/service
paths, modify credentials/infrastructure, restart services, publish packages,
merge to `main` or tag a release under a documentation/development assignment.
Respect the selected executor's resource admission and the current assignment.
Shared-host heavy builds, resolvers, tests, containers and audits require the
applicable resource authorization; repository instructions do not release a
host hold or activate a service. Read-only and bounded static checks may proceed
within their actual grants.

For runtime operation, use the repository-owned
[`cq` skill](integrations/skills/cq/SKILL.md) and its contained references.
Inspect the selected executable's help and MCP catalog. Never invent a tool,
turn local config editing into remote administration, or infer a grant from
an instruction file. Workload execution and raw disclosure need their actual
separate authorization.

Report exact changed paths, purpose, checks and their scope, unresolved gates,
and the owner handoff/next action. Do not claim runtime verification from a
source or documentation check.

## Product checkout and bootstrap branch

This repository contains the CircularQuay product and its product-specific
contribution, security, build and qualification contracts. Build and run it
from this checkout using the documented toolchain and dependency pins. External
maintenance-account bindings, orchestration policies and session allocations
are supplied by the selected execution environment; they are not product
prerequisites or permission grants.

Pass the exact product task, source revision, owned paths, applicable security
requirements and current assignment restrictions to any delegated work.

During initial bootstrap, use the guidance on `feature/initial-implementation`
(PR #1), rather than the empty `develop` initialization tree. Verify the actual
branch and revision before work; preserve existing work and coordinate with the
implementation owner. This branch guidance does not authorize merging PR #1,
starting services or treating the product as qualified. When PR #1 is completed
and merged under its established controls, use current `develop` as the normal
product development baseline. Publication alone does not prove that
existing agent checkouts or running sessions have adopted these instructions.
