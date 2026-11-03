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

Published checkpoint `42a3806d68a39583584d22e51469278cfa698cac` has tree
`d592a8ba1f2c9acb1106982d6eb82a9d94af4921`. Public readback verified all seven
changed files and preserved all 231 working files. Its source checks and
synthetic native Linux/macOS/Windows suite passed; all 26 cross-builds and
CodeQL passed. Coverage and gosec still fail. Exact Sonar reports zero issues
but fails coverage; the sealed managed diff has partial completeness.

The frozen local checkpoint19 follow-up is based on that published revision.
Its whole-repository normal/race tests, vet, Staticcheck, formatting, module
checks, both complexity ceilings of 10, actionlint and 12 Windows compilations
passed with identical source manifests. Coverage is 5,224/6,082 statements
(85.89280%), below 100%. Retain `.local/qualification/checkpoint19-20261104/`.
It propagates audit signals and repository request cancellation, prevents
promoted `bytes.Buffer.ReadFrom` from bypassing subprocess output caps, bounds
inherited-pipe waits, and tests Linux systemd invocation/publication contracts
without running host tools. Exact committed-candidate qualification is separate.

Optional DPoP and typed audit-tool inputs remain separate unmerged work. Do not
mistake their local files, staged diffs, schema declarations or focused tests
for delivered or qualified runtime features. Preserve task ownership and exact
retained evidence. The DPoP review requires protocol, replay/nonce expiry and
failure conformance; typed tool inputs still need actual probes, fixed
invocations and rule/server binding before the auditor can rely on them.

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
development/production identity assumptions. DPoP, mTLS, private-key JWT, and
offline-artifact binding are not implemented or qualified.

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
test. It also includes Windows fixture repairs. Branch discovery, `flow`, and
an authentication UI remain unimplemented. This batch remains unqualified until
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

GitHub protections currently have four required GitHub Actions checks bound;
code scanning is configured as `Any` / `Any`, tag rules are active, and the
advanced CodeQL source workflow has completed after the user-approved switch
from default setup. Verify settings by read-back after any administrative change. Earlier
administrative writes returned HTTP 403; record permission gaps rather than
bypassing them.

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
