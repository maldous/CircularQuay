# Development session handoff

Status: CircularQuay v1 development is active and **not production qualified**.
No release, package-index publication, production deployment, system/service
installation, credential migration, operational infrastructure change, or
live-provider qualification has occurred.
Do not claim a passed gate without its retained evidence.

## Resumption and retained scope

The user resumed CircularQuay delivery on 4 November 2026. The earlier pause
for BAU priority is revoked for this isolated project. Preserve the saved
source/security patch and existing task ownership. No BAU service,
operational credential, system/service installation, production deployment,
paid purchase, main merge, or release tag is authorized by the resumption.

The expanded v1 scope is published in the README and managed-service contract:
one-command session bootstrap, verified repository/tree configuration,
authenticated enterprise discovery, Azure Pipelines/GitHub Actions/GitLab CI
workload identity, complete overrides/CLI CRUD, optional MCP management and
delegation, and managed identity-to-age-key access with authorized UI/API.
The README and product/design/operator documents describe the final North Star;
actual implementation and qualification belong in status/evidence records.

## Current test follow-up and retained failure

Published test-only checkpoint `dec332ad1e60cde1f1400e9eb87c08a49aaccd1e`
has tree `d5f7546f14f42c39e4b65b11214f391993b33b06`; all four changed
files and all 273 working files were read back and preserved. Hosted source
run `41741675497` and security run `41741675310` caught the same compilation
error in both new Azure fixtures: the SDK transport interface requires `Do`,
not only `RoundTrip`. Linux tests and Staticcheck failed; govulncheck/gosec
were not reached in the security job. CodeQL passed with zero annotations.
The manual source review missed this interface mismatch and is not a passing
execution receipt. Do not represent this revision as coverage-only failure.

Retain `.local/qualification/ci-dec332a-native-cross.json`, including its
at-collection pending jobs and the checks not reached after compilation failed.

The next correction supplies an `http.Client` around each memory transport,
with a one-second fixture timeout. The pinned SDK interface and standard
client method were checked directly. Local heavy qualification remains held;
formatter parsing and source review do not establish compilation or tests.
A fresh exact-candidate hosted run must confirm correction. The production
adapter, policy/schema, workflow and security gates are unchanged.

## Completed 65be10 hosted receipt

Published checkpoint `65be10a48257b1207df411b857b2a67c68280ee1` has tree
`c756b0af4ed3e5bbc7009e7299d704ea17d0d18c`. Public readback verified
all 11 changed files and all 271 working files; draft PR #1 still targets
`develop`. Hosted source run `41739246845` passed native Linux/macOS/Windows,
normal/race tests, vet, bounded fuzzing, both complexity checks and all 26
cross-builds. The new aggregate check also passed. The source auditor accepts
the tool/helper manifest and passes unit tests, vet, formatting and source
identity. Its only required source-profile failure is coverage at 6,054/7,129
statements (84.92075%). Security run `41739247007` passed Staticcheck with all
checks, govulncheck and CodeQL with zero annotations. Gosec `-nosec` still
fails eight findings (G204 ×6, G103 ×2), with zero suppressions. Retain
`.local/qualification/ci-65be10a-native-cross.json`. No exact65 Sonar or
managed diff scan is claimed.

This candidate splits the two preceding cyclomatic violations without raising
the ceiling. Hosted Staticcheck now selects all checks; gosec ignores nosec
annotations; pinned gocognit uses exact recursion and rejects diagnostics.
The stable `Cross-build matrix` job requires the whole build matrix to succeed.
Workflow configuration alone does not enforce GitHub protections. Saving the
prepared additional required checks and all-severity CodeQL rules still waits
for GitHub's user-completed `Confirm access` authentication.

Local heavy tests/scans/builds remain held for BAU disk reserve. Three exactly
identified, completed CircularQuay Go build caches were cleared with pinned
`go clean -cache`, removing 3,018,936,320 allocated bytes (2.81 GiB). Every
tracked source byte, module/toolchain cache and nonreproducible qualification
artifact was preserved. At 2026-11-03 23:32:59 UTC, filesystem free space was
35,657,736,192 bytes; the reported 70 GiB reserve was still short by
39,504,191,488 bytes. Other host cleanup is not attributed to CircularQuay.
Retain `.local/qualification/resource-hold-latest-20261104.json` and its
exact ownership, cleanup and source-integrity receipts. A later 23:41:09 UTC
readback measured 618,975,252,480 available bytes (576.47 GiB), above the
reserve; the much larger external host recovery is not attributed to this
project. The bounded additional inventory identifies shared global caches,
active pinned project dependencies and completed task cache portions without
changing them. Retain `resource-reserve-readback-20261104.json` and
`resource-go-cache-additional-20261104.json`. Local heavy work remains held
for shared-resource coordination; hosted CI consumes separate build resources.
No new local heavy qualification is claimed.

## Previous revision evidence

Previous published checkpoint `6b0888307ebb15ea3ca13a4da42d4056102fe1e4`
has tree `35781cc4c4eca892e156e7cbc19abca8a0f82507`. Public readback verified
all nine changed files and all 268 working files. These receipts qualify only
that revision, not later source edits.

Source workflow `41733653552` passed Linux/macOS native tests and all 26
cross-builds. Windows job `126257137168` still fails
`TestToolInputOverlapAndPostReadNameFence` at protected-file activation;
the prior environment-path failure is absent. Source assurance accepts its
manifest and all Git/Go/gofmt probes, then fails `unit_tests` (exit 1) and
coverage at 5,810/7,055 statements (82.35294%). Do not call this coverage-only.
A separate restricted-PATH reproduction isolates the signed-repository
fixture's missing `ssh-keygen` dependency. CodeQL passes with zero annotations;
actual Linux gosec still reports eight unsuppressed findings. Retain
`.local/qualification/ci-6b08883-native-cross.json` and
`source-profile-helper-closure-20261104/before.json`.

Exact-head direct Sonar CE `707d4da6-6bd9-4042-b4a5-56a854f3e8f0` succeeded
with analysis `d936850a-ee82-4581-8a03-429f94e20fb6` and the requested SCM
revision. All open issues, including S1192, are zero. Gate ERROR is solely
coverage: 78.3% overall / 79.7% new versus the unchanged 100% gate. Fresh Go
union statements match 5,953/7,055 (84.37987%). Scanner warnings are zero.
Current-analysis and hotspot-list APIs still return 403; do not claim an
independent latest-analysis fence or hotspot-list clearance. Retain the
prior Nov10 token-expiry warning and `.local/sonar/summary-6b0888307ebb.txt`.
This diagnostic does not satisfy the auditor's runtime-binding gate.

Fresh managed scan `e53999d5-b8a8-4190-95a4-2b40c48bd43d` sealed all nine
files of `2d1f100` to `6b088830`, with actual populated and closed inventory,
zero candidates/findings and no exclusions. Focused assurance validation
passed; CLI validation exceeded its observation window and is not counted.
An earlier sealed scan with zero populated review rows is invalid qualification
evidence, regardless of its zero findings or semantic completeness claim.
Canonical evidence remains under the task's Codex Security scan store. This
is diff evidence, not a complete independent production audit.

Previous published checkpoint `bfaa66cc116093588dd1053a301bdf90a8ecef28`
has tree `0afc2f5f7fcaf00f1547b29f849540cf78cfc89f`; its native Linux/macOS/
Windows, all 26 cross-builds and CodeQL passed. Its synthetic merge
`3be5764a6b551982bfa4962fc7c35a8c7000df2f` has the same tree. Those results
do not qualify 2d1f or later edits. Retain `ci-bfaa66-native-cross.json`.

The exact-head scanner uploaded its report with the requested SCM revision.
CE `f8564a8c-d33e-470a-a829-a9b2558305dd` succeeded and returned analysis
`8a844c31-e83c-4be8-ba85-2bb12be93b69`. Gate ERROR reports 79.9% overall /
82.4% new coverage and four open code smells (three argument-count findings
and one shadowed builtin). Fresh statements are 5,554/6,441 (86.22885%).
The current-analysis and hotspot-list APIs returned HTTP 403; do not claim
an independent latest-server-analysis fence or hotspot-list clearance from
these receipts. The scanner's hotspot metric is zero. The separate old audit
route also failed tool checks and protected token reading because the shared
cache ancestor is group-writable. No permission was relaxed; direct scanning
used only the existing task analysis credential in process environment.
Retain `.local/sonar/summary-bfaa66cc1160.txt`, its artifact hashes, and the
prior 2026-11-10 token-expiry warning. The four source issues have readable fixes guardedly integrated from a
separate bounded checkout; only a fresh exact-head scan can confirm closure.

Managed diff scan `56553a87-7692-4deb-9432-d0f716652dc3` sealed all 37 files
of `97b757a` → `bfaa66c` with zero findings/candidates and complete coverage.
Pinned Go 1.27.3 focused DPoP/identity/MCP/policy validation passed. Artifacts
are under `.codex/state/plugins/codex-security/scans/repo/` in the task host's
Codex home; this is a complete diff review, not a full production audit.

Frozen local checkpoint21 adds optional named-authority DPoP, strict issuer
`cnf.jkt` binding, required persistent nonce/replay state and a request-local
SDK verifier handoff. Whole-module normal/race tests, vet, Staticcheck with all
checks, formatting, module checks, both complexity ceilings, actionlint,
13 Windows and four Darwin test compilations passed with identical source
manifests. Coverage is 5,554/6,441 (86.22885%), below 100%. Retain
`.local/qualification/checkpoint21-20261104/` and the separate failed gosec
receipts; exact committed native/Sonar/security qualification is separate.

The profile advertises its supported algorithms and all-authority requirement
through protected-resource metadata, checks the actual request host/path and
any absolute-form scheme, and never derives its trust origin from forwarded
headers. A reproduced alias/scheme/port gap failed before the fix and passes
afterward. Signed issuer-to-SDK tests prove raw-read actor/deadline/audit
behavior, replay denial, and that a rejected fresh Bearer proof remains usable
under the required DPoP scheme. The cache is single replica/local filesystem,
with required one-use nonces, expiry and fixed global/per-binding capacity.

The final sender-constraint model has independent acceptance/restart witnesses,
mixed token/proof expiries and live-window replay semantics. TLC checked
127,067 distinct states at depth 20; all four private guard mutations produced
actual counterexamples. Earlier weaker models and malformed mutation attempts
are historical evidence, not qualifying results. Retain raw final logs under
`.local/qualification/sender-constraint-final/`. The model is a bounded
abstraction with trusted host/clock/store/crypto assumptions, not exact program
refinement or proof against host administrators.

The first combined capture failed one new policy test's cognitive ceiling.
That test was split into readable helpers and the fresh combined capture
passed; retain `checkpoint21-20261104-before-test-refactor` as historical
failure evidence. Schema review also caught optional-skew decoder tags and
missing cross-field age/skew constraints; regression tests failed before the
fixes and cover both sides of every small-age boundary afterward.

Typed audit-tool inputs have been guardedly integrated into the primary
checkout from a frozen isolated slice. Protected tools and typed rules inputs,
fixed probes, private canonical Git/Go helpers, pre/post rules checks and
source-waiver denials are implemented. An actual tiny synthetic source audit
passed with protected tools under a poisoned ambient PATH. The primary merge
preserved other files and three-way merged RELEASE.md; focused README audit
commands now name the required manifest without changing the author's opening.

Integration review caught CI gofmt help being used as a multiline version
label and an incorrect expected help exit code. The corrected exact workflow
block generates an accepted protected manifest. Rooted bounded helper copies
stay non-executable until complete digest verification. Source-profile Sonar
is explicitly `not_run`, and enterprise Sonar is required failure, until full
scanner/runtime/remote-rules binding exists; the direct diagnostic route does
not satisfy that gate. Full runtime provenance, atomic use and child-process
isolation remain unresolved. Combined checkpoint22 checks and publication
are separate from the earlier isolated receipts; consult actual HEAD/status.

The primary checkout also integrates `flow run` dispatch for GitHub Actions,
GitLab CI and Azure Pipelines environment contexts. Explicit local config
suppresses ambient remote selection; explicit source fields override their
individual environment defaults. Remote retrieval requires matching immutable
commit/digest pins, uses a protected per-run temporary policy and removes it
on return. Tests cover selection, real declared child delivery, error cleanup,
pre/post-fetch cancellation and inherited CI markers. A sanitized stderr
receipt identifies configuration provenance and labels the runner context as
`hint_not_identity`. Child stdout/stderr remains workload-controlled, not
redacted. The README unattended example now supplies both pins. No CI token
verification, mutable/enrolled discovery, later-step export or interactive
login is added by this wrapper. These remain separate v1 requirements.

The first combined checkpoint22 capture failed the new flow integration
test's cognitive ceiling (20 versus 10). Root split the test into cohesive
fixture helpers and retained the failed capture. An expanded offline zizmor
run with the auditor persona also required a rationale for CodeQL's existing
`security-events: write` grant; the permission is now documented in place,
with no scope or threshold change. Fresh combined source/static receipts,
not isolated worker claims, determine publication readiness.

Frozen combined checkpoint22 passed all 31 recorded source/test/static/
format/module/workflow/compile checks with identical before/after source
manifests. It includes whole-module normal/race tests, vet, Staticcheck with
all checks, cyclomatic and exact-recursion cognitive ceilings of 10,
13 Windows and six Darwin package test compilations. Coverage is
5,953/7,055 statements (84.37987%), below the unchanged 100% gate. Final scoped
gosec still fails with eight Linux and nine Windows findings, zero Go errors
and zero suppressions. Offline config-free/no-ignore zizmor auditor-persona
collection now reports zero findings. The exact workflow-generated manifest
and public auditor passed a real tiny synthetic smoke under a poisoned PATH;
unselected tools were explicitly not run. These receipts do not qualify
later native CI, exact-head Sonar, a complete audit, providers or production.
Retain `.local/qualification/checkpoint22-20261104/`,
`checkpoint22-final-static-20261104`, `checkpoint22-workflow-security`, and
`workflow-manifest-auditor-green-20261104`. Publication and exact-revision
readback remain separate guarded steps.

Frozen local checkpoint23 passes all 31 recorded checks with identical source
manifests after the fixture, provisioning and duplicated-label fixes. Native
statement coverage is unchanged at 5,953/7,055 (84.37987%); scoped gosec still
fails eight Linux/nine Windows findings with zero Go errors/suppressions.
No threshold or platform guard was relaxed. Windows/Darwin compilation is
not native proof; the actual next-candidate CI must confirm the fixes.

The old shared writable tool layout was reproduced as `tool_manifest_invalid`
with no coverage. A private copied full Go distribution/Git layout rebuilt
the actual auditor with the copied compiler and passed a real poisoned-PATH
tiny source audit (1/1 fixture statements). These are fixture receipts, not
CircularQuay's coverage gate. Root also checked the pinned setup-go source:
for modern Go it adds PATH and does not export GOROOT. The workflow derives
the canonical distribution from that PATH and passes copied GOROOT only to
children. Selection tests pass with both absent and stale ambient GOROOT.
Offline strict/no-ignore zizmor auditor collection reports zero findings.
Retain `checkpoint23-20261104`, `checkpoint23-static-20261104`,
`checkpoint23-workflow-security`, `audit-runner-provisioning-20261104`, and
`setup-go-path-contract-20261104` under `.local/qualification`.

The host filesystem exhausted space during an isolated worker's extra test.
Root cleared only this checkout's disposable `.local/cache/gobuild` using
pinned `go clean -cache`, freeing about 16 GiB; every working source byte,
module cache and retained receipt was preserved. No unrelated data, service
or system configuration was touched. The subsequent full capture passed.
The intermediate capture with a workflow-only source-fence change is retained
as `checkpoint23-20261104-before-setup-go-path-contract`; it is not the final
combined qualification.

## Exact 20eac0 development receipt

Checkpoint `20eac0a2a99faf5d9f51c126d66ece3db107bf01` has tree
`1871522215dd75f2692b2833333095181d5d1b91`; public readback verified all
15 changed files and all 271 working files. Source workflow `41736690731`
passed normal/race tests, vet and bounded fuzzing; native Windows/macOS and
all 26 cross-builds passed. Linux's job nevertheless failed its cyclomatic
step: `validManifest` and `qualifiedTools.environment` each scored 11 versus
10. They are split into cohesive shape/copy helpers in the following slice;
the ceiling is unchanged and exact later execution must confirm the correction.

The actual source auditor now prepares and verifies its declared private SSH
helper and passes `unit_tests`. Its only source-profile failure is coverage:
6,050/7,125 statements (84.91228%) versus 100%. This does not replace a fresh
Sonar profile or qualify later changes. Security run `41736690751` passed
CodeQL with zero annotations, Staticcheck and govulncheck; gosec still failed
eight unsuppressed findings. Retain
`.local/qualification/ci-20eac0a-native-cross.json`. No exact20 Sonar or managed
security diff scan was run under the local capacity hold.

The next workflow adds a stable `Cross-build matrix` aggregate that passes
only when the entire matrix succeeds. It is a named result suitable for
GitHub-required checks; source configuration alone is not server enforcement.
The branch-rule browser is usable again. Stronger Go-security/CodeQL and all-
severity code-scanning rules are prepared, but GitHub's fresh `Confirm access`
authentication currently blocks saving; the API still reads the old four
required checks. Do not call the prepared change enforced until saved and
read back independently. The visible in-app GitHub tab is retained for user
authentication, with no credentials or codes requested in chat.

## Bounded continuation after the 6b088 receipt

The next source slice declares `ssh-keygen` as an optional fixed native helper
in the v1 audit manifest, distinct from probed analyzers. Its protected original
and private canonical copy are checked around commands; case-insensitive PATH
and GOROOT aliases are removed. No ambient helper directory or invented
version probe is accepted. The source workflow declares the copied dependency.
Schema/parser, drift and private-PATH tests accompany the implementation.

The Windows name-fence fixture now renames its open original aside and creates
a same-size/mode replacement, proving the original descriptor remains readable
and the named identity differs. Production sharing/ACL protections are unchanged.
A separate workload lifecycle fix preserves context cancellation/deadline errors
and bounds waiting for inherited output pipes with a 250ms WaitDelay. This is
not descendant containment or a bound on an arbitrary blocked caller reader or
writer.

The remote workflow mirrors the published analysis requirements with
Staticcheck `-checks=all`, gosec `-nosec`, and pinned gocognit v1.3.0 using
`-exact-recursion -over 10` on native owned directories. Nonempty diagnostic
output, including a type-aware fallback warning, fails that check. Workflow
configuration is not evidence of a passed execution or GitHub enforcement.

A BAU capacity notice arrived during this slice. Local heavy tests, scanners,
builds, downloads and cache copies were held; no task-owned Go/test/scanner
process remained at the resource readback. Free space was about 11.9GiB versus
the coordinator's 70GiB recovery reserve. The read-only inventory retained
exact ownership/size evidence under `.local/qualification/resource-hold-20261104.json`;
no cross-project cleanup, service or permission change occurred. Existing
source and nonreproducible evidence were preserved. No new complete local
qualification is claimed for this slice. GitHub-hosted CI can independently
qualify a published candidate without consuming Foo build storage; Sonar and
other Foo-heavy qualification remain subject to the capacity hold.

The following test-only candidate guardedly imports two synthetic provider
contract suites from `/home/user/src/2026-11-04/circularquay-provider-contracts`.
It exercises real AWS/Azure generated SDKs over memory transports and GCP
response evidence after the identity boundary. Positive adjacent cases prove
that stage/selector/version mismatches, missing payloads, denied/missing/error
responses and malformed JSON cannot disclose any returned value or version.
Failures require a complete zero `model.Value` and marker-free errors. Azure's
503 fixture disables SDK retries and preserves the current conservative
trust-failure classification. Every fixture is synthetic and makes no network
request. No provider adapter, policy/schema, permission or quality gate changes.

Pinned gofmt parsing and independent source-only review are separate from
execution. Local heavy validation remains held; exact-candidate hosted tests,
coverage, static checks and native runs are required before these assertions
can be reported as passing. Retain `checkpoint26-transfer.json` and
`checkpoint26-provider-review.md`; the transfer receipt records original worker
hashes, while root review records the final primary hashes and improvements.

## Latest bounded documentation result

The documentation-only commit `1d89ffbf65a03e4e6ef8f146944416b97356e38c`
is published and byte-verified on `feature/initial-implementation`. It adds the
standalone [managed key service](managed-key-service.md), a concise README link,
and architecture/operator/MCP/recovery references. The service uses an existing
trusted identity provider and explicit key mappings; scoped UI/API administration
manages allowed principals, grants, windows, changes, and revocation. Existing
paid Netlify/Cloudflare capabilities take preference over a custom UI; any missing
key-specific UI is hosted there. There is no email/onboarding suite. Distribution
cannot revoke already copied private keys; local custody and remote unlocking
remain explicitly distinct models. Financial scenarios are assumptions, not
chosen prices, benchmarks, traction, or deployment approval.

The README repository/tree bootstrap and CI/discovery scenario was already
published at `7b68c6116786779392e01024c102093c33254a0d`. The bounded product
presentation is a final v1 North Star, while actual managed-service status is
separate in [qualification](qualification.md#managed-service-design-status).
The updated user-supplied UIF diagram presents the same architecture without
promoting its old operational snapshot into a current BAU claim. Its artifact
and visual/structural receipts are retained in the private continuation record.
No implementation, service, credential, paid subscription, or BAU deployment
was changed. All existing uncommitted source was preserved during that documentation-only
turn. The later user resumption above reactivates isolated development and
qualification only.

## Start here

- Project/module: `CircularQuay` / `github.com/maldous/CircularQuay`. The app,
  binary, release-asset prefix, and skill are `cquay` ("c-key");
  `cquay-audit` is separate.
- Development branch: `feature/initial-implementation`. This continuation
  includes `4886fc8660706909376f3543ad7cab186d9256b9`; use `git rev-parse HEAD`
  and `git status` to establish the actual checkpoint and subsequent changes.
- Read [CONTRIBUTING.md](../CONTRIBUTING.md), [SECURITY.md](../SECURITY.md),
  the durable [implementation plan](implementation-plan.md), and
  [RELEASE.md](../RELEASE.md) before editing. The plan is the scope contract;
  this file records current evidence and continuation risk.
- Use Gitflow: `main` is released/stable, `develop` integrates, `feature/*`
  develops, `release/*` stabilizes, and `hotfix/*` starts from `main`. Do not
  merge to `main`, tag, publish, or deploy until the required gates genuinely
  pass.

## Isolation and authorization

Work only in an isolated developer checkout with synthetic fixtures. Do not
install CircularQuay into service or system paths, restart services, alter
infrastructure or existing credentials, inspect operational secret values or
shell-profile credentials, or interfere with OffMenu BAU recovery. Do not
create broad persistent credentials to bypass an unavailable GitHub permission.

The user authorized reversible development and free test tooling, not paid
resources, production deployments, or a universal credential daemon. Keep
publishing, account, registry, and operator procedures in `RELEASE.md`; keep
end-user/product material in `README.md`.

## Current implementation checkpoint

This checkpoint includes JSON policy CRUD and safe doctor
formatting; protected metadata/state and path-ownership hardening; strict
source, snapshot, target, grant, and repository contracts; repository quota
work; provider contract tests; and extended release cross-build configuration.
The expected policy/CLI contract is version 1 and the development version is
`1.0.0-dev`.

The CLI supports only policy operations `list`, `status`, `run`, and `read`.
`run` authorizes the enrolled workload; read keys restrict only `read`.
`cancel` is request-context cancellation, not a policy permission. Configuration
editing is JSON-only, protected, bounded, strict-decoded, validated, and
atomically replaced under a local lock plus digest recheck. YAML remains a
loadable policy format but is intentionally not mutated by the editor.

The v1 identity direction does not add a Keycloak dependency. Where a custom
API is supported, trust an existing authorization server/IdP directly (for
example Entra ID or Auth0) through named trust/configuration scope, role,
client, subject, resource, context, and grant declarations. Do not hard-code
development/production identity assumptions. The integrated optional DPoP
profile requires issuer-bound keys and persistent single-replica nonce/replay
state; native and issuer/client interoperability remain unqualified. mTLS,
private-key JWT and offline-artifact binding are not implemented or qualified.

Grant time windows and the six reviewed remediation patches are implemented
with focused synthetic tests. They are not qualified capabilities until fresh
combined, full, native, and Sonar checks pass.

The checkpoint6 source batch added named RFC 9068 and Entra v2 API authorities;
configurable all-of scope and roles; exact grants and current-trust
fingerprints; a strict shared legacy verifier; immutable GitHub bootstrap only
(full SHA plus trusted SHA-256); signed-repository CLI/MCP runtime with
`FetchCurrent` offline/fallback guards and hardened Git hashing/fsck; strict
snapshot validation and full-snapshot MCP reads; durable fallback pre-read
fencing across restart; and the Azure workload-identity transport constructor
test. It also includes Windows fixture repairs. Mutable branch/enrolled discovery and an authentication UI remain
unimplemented. Immutable `flow run` dispatch is integrated with the explicit
limitations above. This batch remains unqualified until
a combined candidate passes fresh native, Sonar, and independent checks.

README ordering was directly revised by the user and remains preserved.
The Android rendering investigation found the raw Markdown, fences, HTML, and
linked SVGs clean; browser rendering worked. Treat that as an app/cache issue
unless new evidence reproduces a source-side rendering defect. Do not reorder
the README without a user-directed reason.

## Recent local evidence

Checkpoint19 receipts under `.local/qualification/checkpoint19-20261104/`
record the 24 listed source/test/static/format/module/workflow/compile
checks passing with identical source manifests. Coverage is 5,224/6,082
(85.89280%), below the unchanged gate. Real synthetic children prove oversized
stdout denial, cancellation after startup and finite inherited-pipe waits;
reader-only `io.Copy` regressions exercise the former promoted `ReadFrom`
bypass. These mechanisms do not contain descendants or sandbox source code.
Repository tests prove invalid accepted/config/object/materialized state denies
before release, and cancellation returns no release while cleaning temporary
verification state. Linux seams verify fixed argv, empty environment and
failure/empty/oversized-output preservation of the prior projection without
starting a service. Native execution and exact-candidate Sonar remain separate.
The separate scoped gosec receipt under
`.local/qualification/checkpoint19-static-20261104/` fails: Linux has eight
findings (G204 ×6, G103 ×2), Windows has nine (G204 ×5, G103 ×4), with zero Go
errors and zero suppressions. The Linux count decreased because two systemd
invocations now share one execution boundary; it is not a cleared risk or
weakened rule set. Both scans preserve identical source manifests.

Checkpoint18 receipts under `.local/qualification/checkpoint18-20261104/`
record whole-repository normal/race tests, vet, Staticcheck, formatting,
tidy-diff/module verification, both complexity ceilings of 10, actionlint and
12 Windows test-compilation checks passing with identical source manifests.
Merged coverage is 5,133/6,048 statements (84.87103%), below the unchanged 100%
gate. The real signed-repository regression verifies protected offline Git
temporary state without fetch credentials and its cleanup after return.
Native Windows execution and exact-candidate Sonar remain separate evidence.

Checkpoint17 receipts under `.local/qualification/checkpoint17-20261104/`
record Go 1.27.3 whole-module normal/race tests, vet, Staticcheck, formatting,
tidy-diff/module verification, both complexity ceilings of 10, actionlint and
12 Windows test-compilation checks passing with identical before/after source
manifests. Coverage is 5,114/6,022 statements (84.92195%), below the unchanged
100% gate. The intermediate checkpoint15 test-helper cognitive-complexity
failure is retained; the refactored helper passes checkpoint17. These local
results do not establish native Windows, Sonar or independent qualification.

Checkpoint10 receipts are retained under
`.local/qualification/checkpoint10-20261104/`. Whole-repository normal/race tests,
vet, Staticcheck, formatting, tidy-diff/module verification, cyclomatic and
exact-recursion cognitive ceilings of 10, actionlint, and 12 Windows test
compilations passed with identical before/after source manifests. Merged
whole-module statement coverage is 4,899/5,765 (84.9783%), below 100%. An earlier
checkpoint8 diagnostic-helper complexity failure is retained; the refactored
helper passed checkpoint9 without changing the ceiling. Checkpoint10 includes
explicit DPAPI input/output bounds: Windows gosec no longer reports the two
length-conversion findings. Its nine remaining G103/G204 findings are retained;
they are distinct from the Linux lane's nine findings. Windows/macOS cross-static
checks do not establish native execution.

Historical checkpoint6 and checkpoint7 receipts remain retained. The earlier
checkpoint7 source receipt is
`.local/qualification/checkpoint7-20261104/`, with identical before/after
manifests for Go, JSON, workflows, and canonical model fences. These are local
evidence, not release evidence.
Using a fresh owner-only temporary directory under root-owned sticky `/tmp`,
Go 1.27.3 normal and race tests, vet, gofmt, `go mod tidy -diff`, `go mod
verify`, Staticcheck 2026.2.2, `gocyclo -over 10`, exact-recursion `gocognit
-over 10`, and actionlint passed. Compile-only Windows checks passed for the 12
affected packages. The checkpoint7 whole-module merged `-coverpkg` profile is
4,881/5,744 (84.97562674%), below the unchanged 100% gate. It succeeds older
checkpoint6 local coverage (4,787/5,742, 83.3682%) and is distinct from the
41a8 production-assurance CI profile (4,828/5,742, 84.0822%); none passes the
gate.
Pinned gitleaks v8.30.1 scanned a captured 216-file inventory before this receipt update (SHA-256
`0caf87f382e74e55bee6d7ef3f4f7ac9c839bb4d17183764ac3ff0e29752e74b`)
with zero findings. This remains bounded local working-tree evidence, not a
history, artifact, credential, or production certification. The old copied
213-file scan and its incorrect initial invocation remain historical evidence.
TLC also completed the durable-fence model with trusted-store and lock
assumptions: 14,818,232 generated states, 76,360 distinct states, depth 20,
an empty queue, and no error. It does not qualify native Windows permission
failure behavior, full implementation conformance, or host recovery.
The `internal/identity` package separately passed normal/race/vet and both
complexity checks with 183/183 statements covered; that package receipt does
not qualify the repository as a whole.
The same checks correctly reject a `.local` temporary directory with an unsafe
`0775` ancestor; do not relax that protection to make tests pass.

After source changes, run the applicable checks from protected temporary
storage and retain exact command, tool version, revision/tree state, and result.
The source-profile 100% owned-code coverage gate remains unchanged and is not
met by checkpoint19 or the last published CI receipt.

## Qualification and release blockers

Published candidate `54fb84a2d3ccd65d0cc7c3d7a780c7c5daa40507` has tree
`325e75f2016bfd055a1289e4203c583f062c29bc`. Public readback verified all 21
changed files and all 236 working files. Source run 41713357341 tested a
synthetic merge with the same public tree: Linux/Windows and all 26
cross-builds passed. macOS failed the audit bootstrap fixture because its
private temporary path included a system symlink; other listed packages
passed. Do not misclassify that assertion as a dependency/setup failure.
The captured checkpoint20 changes only that test fixture to use its physical
path, preserving production checks. Audit-package normal/race/vet/Staticcheck,
format/complexity and three Darwin/Windows test compilations passed with
unchanged source manifests. All production inputs match checkpoint19; native
macOS revalidation remains required. Retain
`.local/qualification/checkpoint20-20261104/`.

Security run 41713357268 passed CodeQL. Actual Linux gosec reported eight
findings; the private receipt preserves and corrects an earlier unmeasured
carried-forward count. Sonar CE `e6af4d51-c9be-4860-844d-89715fd6b4c5` and
current analysis `c8623fc7-2362-4019-9ff8-88546a7de9a6` have exact revision
fences and zero issues/smells/bugs/vulnerabilities/hotspots/duplication, but gate
ERROR at 79.4% overall / 81.8% new coverage. Keep the Nov10 token warning.
Go coverage remains 5,224/6,082 (85.89279842%), below 100%.

Managed delta `94b6f845-d4f0-4df1-b630-31c673f6294b` is sealed with complete
coverage of the 19 authoritative source/test diff items and zero findings.
Focused package validation passed. This does not qualify the whole repository,
host sandbox/tool provenance, live providers, native services or recovery.
Retain `.local/qualification/ci-54fb84a-native-cross.json` and
`.local/sonar/summary-54fb84a2d3cc.txt`. No production action occurred.

Published candidate `42a3806d68a39583584d22e51469278cfa698cac` has exact tree
`d592a8ba1f2c9acb1106982d6eb82a9d94af4921`. Source run 41706881432 tested
synthetic merge `834345b94f89aa3d1287cf747a85bbd9131f61eb`; public readback
confirms its tree equals the candidate tree. Native Linux/macOS/Windows and
all 26 cross-builds passed. Security run 41706881558 passed CodeQL but failed
gosec with nine findings. Coverage remains 5,133/6,048 (84.87103175%), below
100%. The source-profile job has no Sonar endpoint; it is separate from the
completed exact-head Sonar route.

Sonar CE `1b97ca13-5ab7-48ec-98b8-f2e0cc96e96b` completed with current
analysis `534d0cde-eb0b-45b7-9af4-72cd824f3ff0` and exact revision fences.
Both prior smells are resolved, with zero issues, bugs, vulnerabilities,
hotspots and duplication. The gate is ERROR at 78.3% overall / 81.5% new
coverage. Keep the 2026-11-10 token-expiry warning.

Exact-head Windows cross-static Staticcheck passed all 19 owned packages;
gosec retains nine unsuppressed G103/G204 findings. Before/after HEAD, tree,
index and file-content fences matched. Managed diff
`0100370b-bbe7-4a3c-91ad-5b78c011cd04` is sealed with zero reportable findings
and partial completeness. Its deferred native runtime and broader-audit scope
are not retroactively satisfied by the separate native test result.
Retained receipts are `.local/qualification/ci-42a3806-native-cross.json`,
`.local/qualification/static-42a3806-20261104/`, and
`.local/sonar/summary-42a3806d68a3.txt`. These are not production qualification.

Published candidate `4a485258c3a72a7227502f359b4a049996b4ba61` has exact tree
`f8c53890e348cf8b7fdb56a749b0f8aaca0d00d5`. Native Linux/macOS, all 26
cross-builds, and CodeQL passed. gosec failed with nine findings and source
assurance was 5,114/6,022 (84.92195284%), below the unchanged 100% gate.

Native Windows failed in more than one way: a googleapi module-proxy timeout
blocked CLI/resolve/source setup, while tests that ran failed trusted-directory
preflight and offline `Current` Git temporary-file creation. Do not attribute
all failures to the proxy or treat all tests as inconclusive.

Sonar CE task `79b7e0cd-e546-4d68-848e-be7ab4e90113` and analysis
`9c02dbab-ce75-4d3b-828e-de1bf483d2c9` completed with zero bugs,
vulnerabilities, hotspots, and duplication, but its gate is ERROR at 78.4%
overall and 81.6% new coverage, with `go:S1192` (five `rev-parse` literals)
and `go:S978` (rune issue). Its token-expiry warning for 2026-11-10 remains.
This is not a pass.

Sealed managed audit delta `cad20f55-3074-47e5-8dce-f7648dc9141b` from e553
to 4a has zero new reportable findings but partial completeness; host sandbox,
tool provenance, and native Windows are deferred. It is not an audit pass.

The follow-up published in `42a` replaces the repeated Git command literal and renames
the shadowing identifier. It adds protected working/executable directories to
positive platform/CLI fixtures and a protected per-call temporary directory for
offline `Current` verification, with prior-temp restoration and cleanup. Whole
checkpoint18 checks passed locally; later exact-candidate receipts are recorded
above. No native configuration, credentials or
infrastructure changed.

Published candidate `8ed18f35be811f722068a3cf62d3093038fa8d93` has exact tree
`b42f899e839d85febc7ff06105b60cf07c41e5bc`. Native Linux/macOS and all 26
cross-builds passed, as did CodeQL. Native Windows failed with two empty SSH
probes and a path-oracle failure; gosec retains nine findings. CI whole-module
coverage was 84.9957%, below the unchanged 100% gate.

Its Sonar CE task `dbd6d2fa-3814-4b06-a01e-e3bc7632a184` completed successfully
with analysis `81a9ef98-5a95-4e3d-a706-39362381c61e` and zero open issues, but
its quality gate is ERROR at 78.6% overall and 82.1% new coverage. The retained
receipt warns that its token expires 2026-11-10. This is not a pass.

The captured 4a follow-up integrated the v1 source manifest before and after target
checks, rejects unexpected/ignored files, disables Git replacement/global/system
configuration, and requires the actual repository root. It bounds file reads and
preserves worktree metadata support. A source-root alias is now rejected before
Go version probing; its prior later formatting failure already prevented source
tests, so the regression does not demonstrate arbitrary test execution. Manifest
checks do not provide an atomic snapshot, host sandbox, or analyzer provenance.

That 4a batch supplies the Windows OS-known ProgramData folder to native Git/SSH
without inheriting a caller override; compares canonical Windows directories by
filesystem identity; and adds a real SDK tool-catalog no-read-grant regression.
CI audit binaries/evidence and Sonar scratch are outside the reviewed source
inventory. Tests, coverage, vet, Staticcheck, govulncheck, gosec and formatting receive
manifest-owned packages/imports/files rather than traversing generated output.
Manifest directories without a native Go build fail closed instead of silently
reducing coverage; current packages provide native fallbacks. Native Windows execution, exact-candidate Sonar, and independent
qualification of this batch remain required.

Published candidate `e55327d513b0753ce08ae82adffbc4cb3cd5d2d4` has exact tree
`1523d6ac067ef94270bd015fa60ad296f086f021`. Native Linux/macOS, all 26
cross-builds, and CodeQL passed; native Windows failed, and the explicit
verifier hypothesis was refuted. gosec still has nine findings. Its Sonar CE
task `bb229fa5-0bb7-4db0-bb8e-f88b9f6d99c1` succeeded with analysis
`50ea05ae-0367-4ca7-aa16-e5111349b6c4` and zero open issues, but the quality
gate is ERROR at 79.1% overall and 83.4% new coverage; the receipt also warns
that its token expires 2026-11-10. This is not a pass.

The sealed `e553` managed scan retained eight validated findings: one high
(auditor-host execution), five medium (assurance binding/PATH, SIGTERM, Windows
DLL, and fresh-scope controls), and two low (UID and DPAPI ACL). Its completeness
is partial and it is not an audit pass. Three assurance issues remain open;
keep sensitive paths in the owner-held report.

The captured `8ed` remediation batch added live-generation-versus-snapshot,
legacy-state, mixed-UID, DPAPI staging ACL, Windows protected-loader-directory,
SIGTERM, and MCP base-context/active-HTTP-cancellation fixes. Its focused tests
and cross-builds are revision-scoped evidence only; later active work requires
its own frozen-candidate checks. Do not claim every audit finding is fixed.

The Compute Engine task
`e52a9f8f-b66c-48f2-8a8b-425a533cab12` for revision
`41a8da186c71d94b420303328fbe06fcee708602` succeeded with analysis ID
`dbe51b3f-8bae-4075-9ce0-0a72482e42c3`, but its gate is ERROR: 78.3% overall
coverage (6,134/7,833 lines), 82.5% new coverage, and two code smells
(`go:S107` and `go:S1192`); reported bugs, vulnerabilities, hotspots, and
duplication are zero. Sonar line coverage is distinct from the Go statement
profiles and does not change the 100% gate. At the 41a8 receipt, the two working-tree smell fixes required a fresh exact-candidate Sonar recheck; later e553 and 8ed scans record zero issues while coverage remains failing.
Sonar, full unchanged coverage, live providers, native reboot/recovery,
installers, registry publication, and independent audit remain incomplete.
The sealed static managed audit at this checkpoint reported zero validated
findings, but its completeness is partial and it is not an audit pass. It
rejected a documentation contradiction as non-exploitable, while the stale
repository-sync wording is corrected here. It does not qualify native behavior,
live providers, installations, or services. Keep detailed evidence in the
owner-held security channel rather than reproducing sensitive attack paths in
public documentation.

GitHub source run 41686510514 tested the open PR revision
`41a8da186c71d94b420303328fbe06fcee708602` through GitHub's synthetic merge
`6fd454e6bc77888f4e40d7eec37e6bbe4c7a9748`. Native Linux/macOS and all 26
cross-builds passed. Windows job 126093565982 failed only repository
`TestSignedRepositoryAcceptanceAndRollback` with `trust_failure:
repository_signature`; its platform package passed. Source assurance failed at
84.08220132358063% against the unchanged 100% gate. Security run 41686510624
passed CodeQL but failed gosec with nine unsuppressed findings. The frozen
static receipt found zero reachable govulncheck vulnerabilities and one required
module advisory, `GO-2026-5932` in `golang.org/x/crypto@v0.58.0`. Earlier
41ea/4886 evidence remains in the qualification matrix as historical evidence.
A later dependency follow-up updates that module to its fixed pseudo-version
and reports zero local govulncheck findings; the historical 41a8 advisory is
retained as revision-scoped evidence. Later e553 and 8ed Sonar scans record
zero issues but fail coverage. The 8ed native Windows SSH probes and path oracle
are failures requiring diagnosis, not an unverified signing-fixture hypothesis.

The current batch also fixes a listener-cleanup deadlock. Its retained
reproduction failed before the fix and the focused normal/race/vet checks pass
after it. This is bounded local evidence, not a full or native qualification.

The snapshot at revision `960` contains 25 native archives plus one source
archive (26 total) with exact checksum/provenance evidence; its Linux smoke is
only a smoke test, not production qualification. Cross-build/package work is
build evidence only.
Native Windows/macOS/Linux runtime, restart/recovery, installer, signed
artifact, anonymous download, SBOM/provenance, and registry qualification still
require real evidence. The
signed repository runtime is wired into `sync`, including `FetchCurrent` and
its offline/fallback guards. That integration is not deployment qualification:
quota, recovery, native, and adverse-path evidence remain required.

Actual GitHub ruleset readback confirms active Gitflow branch protections
with four required Actions checks, one approval, stale-review dismissal,
last-push approval and thread resolution, no bypass actors, plus active
immutable `v*` tag rules. The fetched ruleset does not include a code-scanning
rule or required Go security/CodeQL status. The earlier `Any` / `Any` UI note
is not sufficient evidence of enforcement. Legacy branch-protection reads
for main/develop both return 403 from the integration; do not infer those
settings. The read-only browser fallback failed before navigation with an undefined
`timeoutMs` binding, so remaining enforcement/metadata cannot be inferred
from UI state. No administrative change was made. Issues, Projects, Wiki and Discussions are enabled;
repository description/topics are empty. Advanced CodeQL has run following
the user-approved switch. Retain
`.local/qualification/github-hardening-readback-2d1f-20261104.json`. Earlier
administrative writes returned 403; preserve permission gaps without creating
broad credentials or claiming that committed settings are enforced.

The current continuation adds exclusive protected file creation for `init` and
Sonar test credentials: it validates the parent, anchors creation through
`os.Root`, refuses existing names, applies native protection before writing,
and syncs the file and directory. Audit coverage is opened through the protected
file boundary. Concurrent-creator, no-clobber, cleanup-identity, usable-init,
and Windows DACL tests cover the new behavior; native rechecks remain required.
Initialization writes a synthetic fixture and policy separately; a failure
between those creations can leave the synthetic fixture and is not reported
as successful initialization.

The native fixture corrections use JSON encoding for Windows paths, canonical
temporary roots for macOS, and in-memory synthetic signing-key generation while
retaining real Git signing/verification. Case-collision decisions have a pure
cross-platform oracle plus filesystem integration where distinct names are
representable. The Unix executable-alias test now creates the alias rather
than assuming a `/bin/true` layout. Native macOS CI prefers the protected OS
Git/OpenSSH paths; a rejected helper reports only sanitized file metadata.

The current continuation also hardens native policy and literal-source reads,
adds schema/Parse parity regressions and cloud/CLI/doctor negative tests, repairs
relative/default-path config replacement, and corrects native test fixtures.
Windows directory ACL inheritance preserves the private allowed principals
for ordinary child creation; these tests passed in the native Windows package
run at revision `960`. The new exclusive-create DACL tests require a fresh run. The bounded Go engineering review found the config-repair defect, which
failed both regression cases before the fix and passed afterward. This is a
scoped review, not the independent managed security recheck or release audit.

## Priority continuation

1. Inspect `git status`; preserve concurrent work and do not overwrite others.
   Format and run bounded tests with private temporary storage before changing
   behavior.
2. Integrate and review the current policy/CLI/provider/repository/platform
   changes. Keep strict validation and protected-file semantics; add focused
   adverse-path tests where contract behavior changed.
3. Re-run SonarQube against the exact candidate revision, resolve every owned
   issue and coverage gap without exclusions or weakened rules, and retain the
   quality-gate receipt.
4. Complete unresolved sync/application/activation/recovery behavior and its
   strict snapshot/repository validation; do not infer health from an attempted
   activation or a source timestamp.
5. Perform qualified native, provider, installer, release, registry, and
   independent-security evidence. Only then consider a `release/1.0.0` branch,
   `v1.0.0-rc.1`, and ultimately `v1.0.0` under the gates in `RELEASE.md`.

## Do not regress these boundaries

- No inline secret values, provider bootstrap tokens, private keys, or browser
  sessions in policy, fixtures, logs, screenshots, or documentation.
- No silent offline fallback after denial, trust/TLS failure, malformed input,
  expiry, or policy mismatch.
- No arbitrary command/provider URI through MCP; raw reads require an exact
  explicit grant and are deliberate disclosure.
- No quality-gate suppression, fabricated green badge, unconditional release
  claim, or broad credential created to evade a missing permission.
- No production assertion from compilation, cross-build, static analysis, or a
  synthetic fixture alone.
