# Next-agent prompt

Continue the approved CircularQuay PR #1 correction and integration work in
`/home/user/src/2026-11-03/circularquay-dev`, branch
`feature/initial-implementation`.

Read `AGENTS.md` and the **Current handover — 6 November 2026** section of
`docs/session-handoff.md` first. It supersedes historical status below it.
Inspect actual HEAD, dirty tree and remote PR state before acting. Completed
source and preparation changes are committed locally; the last published PR
head is `071e06b`. No new corrections have been pushed or runtime-qualified.

The user already authorized commit/push/merge to `develop` and the CI correction
pass. They explicitly require zero Gosec exceptions **and preservation of all
existing capabilities**, and approved a narrowly pinned Darwin Intel source-build
route. Do not ask for those approvals again, weaken gates, disable capabilities,
merge to main, publish a release, or deploy the backend/authentication app.

Foo's resource hold remains: no local Go/build/test/analyzer, SDK/UI runtime,
install/resolver, Java or Docker. Use hosted CI; preserve the existing 5182
preview process and older worktrees. Do not create a new Linear issue or tracker.
Use the exact existing private checkpoint57 evidence store named in the handover.

First review and ordinarily push the local completed commits, then inspect exact
candidate-bound hosted results, including the new public frontend and Darwin
preparation jobs. Resolve the Python review thread only after verification.
Continue the execution/quota redesign from
`docs/execution-boundary-redesign.md`; it is design-only. The experimental probe
workflow is archived privately and its fixtures do not exist. Keep it out of
active CI until complete. The Darwin preparation job produces proposals only;
existing vulnerable runtime locks remain unchanged and must not be called fixed.

PR #1 remains protected and blocked. Preserve required checks, resolved
conversations and independent last-push approval. Never claim merge, deployment,
coverage or qualification without exact evidence. Complete the authorized work
and merge only when those requirements actually pass.
