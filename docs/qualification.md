# Qualification matrix

This matrix distinguishes implementation from independent or live qualification. The source is in development; no production qualification, certification, availability SLA, or universal provider/client compatibility is claimed.

| Capability | Implementation | Operational evidence |
| --- | --- | --- |
| Strict policy and complete generations | In development; consult package tests | Independent review pending. |
| Local literal configuration | Synthetic example provided | Installed-artifact run pending. |
| SOPS/age and Transit | In development | Selected-version round trip, denial and native-key custody pending. |
| Cloud/SaaS readers | Connector-specific; inspect `cquay providers` | Live-provider qualification pending; mocks do not qualify accounts. |
| Native publication | Connector-specific | Target entitlement/identity and deployment tests pending. |
| Linux/systemd | Implementation/templates in development | Cold reboot/offline recovery pending on an operator-owned host. |
| Windows | Implementation/templates in development | Actual SID/ACL/DPAPI, no-login reboot and restore pending. |
| Termux | Recipe/implementation in development | Real-device post-unlock, DNS/TLS and background tests pending. |
| Containers | Tooling/templates in development | Image-history, runtime and recreation verification pending. |
| Local MCP | In development | Client/version and disclosure tests pending. |
| Hosted MCP | Experimental | Issuer/resource/client conformance and cross-user tests pending. |
| CI flow dispatch | Explicit local or immutable pinned GitHub configuration, exact runner-context hints and declared workload execution | Synthetic selection/cancellation/cleanup and child-delivery tests; context markers do not authenticate CI job claims. Enrolled discovery, CI token verification and later-step exports remain incomplete. |
| Audit-tool manifests | Protected external v1 tool/rule declarations, fixed probes, private Git/Go helpers and source-waiver denials | Real tiny Linux fixture passes selected checks; full runtime provenance, atomic use, OS isolation and enterprise scanner binding remain incomplete. |
| Optional DPoP | Named-authority sender constraints with required persistent nonces and replay denial | Synthetic issuer-to-MCP read/denial tests; native, live issuer/client and cluster qualification are separate and remain incomplete. |
| Sonar and full assurance | Harness in development | No independent completed gate claimed here. |
| Public distribution | Source publication in progress | No release, module-index, OCI or MCP Registry publication claimed. |

## Source workflow scope

The committed workflow defines Linux formatting, module-integrity, unit, race, vet, bounded Go-fuzz and both complexity ceilings. The cognitive check uses the pinned
type-aware analyzer and rejects warnings or fallback diagnostics. It also defines synthetic native test jobs for the GitHub-hosted `macos-15-intel` and `windows-2025` runners, plus Linux cross-compilation for the release build matrix. Workflow definition alone is not a result; the revision-scoped receipts below, rather than this definition, record actual runner, tool, and platform outcomes.

The native jobs exercise only synthetic Go behavior on the runner architecture. They do not prove release archives, Windows ARM64 behavior, DACL/DPAPI service isolation, macOS signing/notarization, service-manager enrollment, reboot recovery, provider connectivity, or consumer installation.

The source audit profile invokes Staticcheck, gosec, govulncheck, gitleaks and actionlint when they are selected in the protected external tool manifest. They remain optional in the source profile, so an unselected analyzer is reported as `not_run` rather than a pass. No analyzer is resolved from ambient PATH. Source Sonar also remains `not_run` until scanner/runtime/remote-rules binding exists; the direct diagnostic route is separate. The enterprise profile and the 100% owned-statement/Sonar requirements remain unchanged; no absent external check, unexecuted fuzz target, cross-build, or configured workflow satisfies them.

Local complementary tooling has exercised TLC, ShellCheck/shfmt, PSScriptAnalyzer, Hadolint/Compose, offline zizmor, Syft SBOM/license inventory, and a bounded mutation sample. The retained development receipts are described in the session handoff; these checks do not yet have a complete committed CI toolchain or release-artifact receipt. Native reboot/restore/provider trials remain separate requirements. No sampled result, static script analysis or SBOM inventory establishes production qualification.

Record each qualification receipt with a source commit, artifact digest, platform/provider/client version, setup, test outcome and sanitized evidence link. Do not mark a row tested based on a plan, cross-compilation, generated manifest, or absence of a complaint.

Existing operational machines are outside this implementation's deployment scope. Operators install from documented artifacts and provide their own installation feedback.

## Exact 98c9caa integration receipt

Checkpoint `98c9caa7c5ed6b20f28cec720c73ca4939c4e63c` has tree
`01e7251bdb02b10b5f6304c0a4dfcf5f277442af`. The actual
[WireMock matrix](https://github.com/maldous/CircularQuay/actions/runs/41753111836)
passed on Linux, macOS and Windows, including Java setup, the pinned JAR
verification and the tagged synthetic OpenBao provider suite. The PR synthetic
merge has the same tree. This proves that suite, not a live provider deployment.

The [source run](https://github.com/maldous/CircularQuay/actions/runs/41753111878)
passed macOS and all 26 cross-builds. Windows failed a trust-store schema
fixture using a native absolute path; Linux found `validateIdentityAuthority`
at cyclomatic complexity 11 versus the unchanged ceiling of 10. Coverage was
6,246/7,303 statements (85.52650%) versus 100%.
[Security analysis](https://github.com/maldous/CircularQuay/actions/runs/41753111921)
retains eight unsuppressed gosec findings. These failures remain recorded even
when a later source candidate supplies corrections.

The first [hosted Sonar run](https://github.com/maldous/CircularQuay/actions/runs/41753111704)
stopped before bootstrap: the runner soft file-descriptor limit was 65,536,
below the required 131,072. Runner-only `vm.max_map_count` preparation reached
524,288. Cleanup also failed because it attempted Compose before its environment
file existed. No container, coverage, scanner, CE or analysis readback ran.
The following workflow correction raises only eligible runner soft limits,
records their hard limits, and distinguishes pre-bootstrap cleanup from a
partial Compose start. The actual follow-up result is required.

[Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41753111900)
identified three upstream advisories in `go-jose/v4` 4.1.4:
[JSON parsing](https://github.com/advisories/GHSA-vvm8-gf78-28g4),
[CBC-HMAC processing](https://github.com/advisories/GHSA-4w3g-jfj2-cw2q), and
[RSA JWK parsing](https://github.com/advisories/GHSA-mgcc-cww5-fp2q).
The minimal combined fixed version is 4.1.6. Its upstream 4,096-bit RSA JWK bound
is retained. The update's two checksums were read from the official Go checksum
database over HTTPS; no local resolver, signed-log proof, build or test execution
is claimed under the shared-resource hold. Exact-candidate hosted module and
identity tests remain required. Two upstream Scorecard warnings are retained
separately; they are not suppressed or represented as cleared.

## Exact 060639 development receipt

At `060639f36c1c7309769f34e3b88812e42d02cabf` (tree
`aef7be1f1b11f112ed04120ed58e61b17f864e46`),
[source workflow 41744753747](https://github.com/maldous/CircularQuay/actions/runs/41744753747)
passed native Linux/macOS/Windows, normal/race tests, vet, bounded fuzzing,
both complexity ceilings, all 26 cross-builds and the aggregate checks.
The synthetic merge has the same tree. The source auditor's unit tests and
vet passed; coverage alone failed at 6,125/7,173 statements (85.38966%),
below the unchanged 100% requirement.

[Security workflow 41744754352](https://github.com/maldous/CircularQuay/actions/runs/41744754352)
passed Staticcheck with all checks, govulncheck and CodeQL with zero
annotations. Gosec `-nosec` failed eight findings (G204 ×6, G103 ×2),
with zero suppressions. These tests exercise cloud-response identity
binding, including unpinned GCP responses and Azure origin ambiguity;
they do not establish live-provider compatibility. Retain
`.local/qualification/ci-060639f-native-cross.json`.
No exact060 Sonar, managed security audit or production qualification is claimed.

## Exact 018fea development receipt

At `018fea744d8dfb959072e1e95047c966b80ea011` (tree
`ebf49136ab6f5b248214e0353a7dae53a307a14d`),
[source workflow 41742555776](https://github.com/maldous/CircularQuay/actions/runs/41742555776)
passed native Linux/macOS/Windows, normal/race tests, vet, bounded fuzzing,
both complexity checks, all 26 cross-builds and the aggregate check. Actual
execution closes the preceding Azure-fixture compilation failure. The source
auditor's unit tests, vet, helper/probe integrity and source identity passed;
its only required failure is 6,076/7,129 statements (85.22934%) versus 100%.

[Security workflow 41742555781](https://github.com/maldous/CircularQuay/actions/runs/41742555781)
passed Staticcheck with all checks, govulncheck and CodeQL with zero
annotations. Gosec `-nosec` failed eight findings (G204 ×6, G103 ×2), with
zero suppressions. Synthetic merge `d6129d9648a41b37126016544d73f4d8c4bf2e89`
has the same tree. Retain `.local/qualification/ci-018fea7-native-cross.json`.
No exact018 Sonar, managed diff audit or production qualification is claimed.

## Exact dec332 development failure

At `dec332ad1e60cde1f1400e9eb87c08a49aaccd1e` (tree
`d5f7546f14f42c39e4b65b11214f391993b33b06`),
[source run 41741675497](https://github.com/maldous/CircularQuay/actions/runs/41741675497)
and [security run 41741675310](https://github.com/maldous/CircularQuay/actions/runs/41741675310)
exposed a compilation error in the two new Azure test fixtures. The SDK
transport requires `Do(*http.Request)`, while the fixture supplied only
`RoundTrip`. Linux tests and Staticcheck failed; later security tools did not
run in that failed job. CodeQL passed with zero annotations, but it does not
make the broken tests pass. This revision has no qualified new test or
coverage result.

Retain `.local/qualification/ci-dec332a-native-cross.json`, including its
at-collection pending jobs and the checks not reached after compilation failed.

The following correction wraps each memory transport in an `http.Client`,
which implements the SDK interface and bounds fixture requests to one second.
Pinned formatting/parser and manual interface review are separate from
compilation; only a fresh exact-candidate hosted run can confirm the fix.
No production adapter, policy, schema or gate changes in this correction.

## Exact 65be10 development receipt

At `65be10a48257b1207df411b857b2a67c68280ee1` (tree
`c756b0af4ed3e5bbc7009e7299d704ea17d0d18c`),
[source workflow 41739246845](https://github.com/maldous/CircularQuay/actions/runs/41739246845)
passed native Linux/macOS/Windows, whole-module normal/race tests, vet,
bounded fuzzing and all 26 cross-builds. Cyclomatic analysis and pinned
type-aware cognitive analysis with exact recursion passed; the cognitive
output was empty. The stable `Cross-build matrix` aggregate passed.
Synthetic merge `7b636f6c6a9d7d105a4ce110834065867a90ce12` has the same tree.

The source auditor passed helper integrity, tool probes, unit tests, vet,
formatting and source identity. Its only required source-profile failure is
coverage: 6,054/7,129 statements (84.92075%), below the unchanged 100% gate.
[Security workflow 41739247007](https://github.com/maldous/CircularQuay/actions/runs/41739247007)
passed Staticcheck with all checks, govulncheck and CodeQL with zero
annotations. Gosec `-nosec` failed eight findings (G204 ×6, G103 ×2), with
zero suppressions. Retain `.local/qualification/ci-65be10a-native-cross.json`.
No exact65 Sonar or managed security diff scan is claimed. Passing named
jobs does not qualify a release or establish required-check enforcement;
the prepared GitHub hardening still awaits fresh access authentication.

## Exact 20eac0 development receipt

At `20eac0a2a99faf5d9f51c126d66ece3db107bf01` (tree
`1871522215dd75f2692b2833333095181d5d1b91`),
[source workflow 41736690731](https://github.com/maldous/CircularQuay/actions/runs/41736690731)
passed normal/race tests, vet, bounded fuzzing, native Windows/macOS and all
26 cross-builds. Linux's job still failed because two functions exceeded the
cyclomatic ceiling (11 versus 10). Those functions are split in the following
candidate; only its actual run can establish closure.

The source auditor's declared SSH-helper preparation and isolated unit tests
passed. Coverage is its only remaining source-profile failure at this revision:
6,050/7,125 statements (84.91228%), below 100%.
[Security workflow 41736690751](https://github.com/maldous/CircularQuay/actions/runs/41736690751)
passed CodeQL with zero annotations, Staticcheck and govulncheck. Gosec still
failed eight unsuppressed findings. No exact20 Sonar or managed diff review
is claimed. Local heavy qualification remains held for BAU recovery capacity.

## Exact 6b088 development receipt

At `6b0888307ebb15ea3ca13a4da42d4056102fe1e4` (tree
`35781cc4c4eca892e156e7cbc19abca8a0f82507`),
[source workflow 41733653552](https://github.com/maldous/CircularQuay/actions/runs/41733653552)
passed Linux/macOS native tests and all 26 cross-builds. Windows still failed
one named-replacement fixture. The source audit accepted its manifest and
three tool probes, then failed unit tests and measured 5,810/7,055 statements
(82.35294%); a missing private-PATH SSH verifier was separately reproduced.
These are actual failures, not a coverage-only result.

[Security workflow 41733653564](https://github.com/maldous/CircularQuay/actions/runs/41733653564)
passed advanced CodeQL with zero annotations. Gosec still failed eight
unsuppressed Linux findings. The frozen local union profile measured
5,953/7,055 statements (84.37987%), below the unchanged 100% requirement.

Direct Sonar CE `707d4da6-6bd9-4042-b4a5-56a854f3e8f0` completed with analysis
`d936850a-ee82-4581-8a03-429f94e20fb6`, zero open issues and zero fresh scanner
warnings. Its gate remained ERROR solely at 78.3% overall / 79.7% new coverage.
The current-analysis and hotspot-list APIs returned 403, so no independent
latest-analysis fence or hotspot-list clearance is claimed. The prior Nov10
analysis-token expiry warning is retained. This direct diagnostic is separate
from the still-unqualified auditor scanner-runtime binding.

Fresh managed diff scan `e53999d5-b8a8-4190-95a4-2b40c48bd43d` covered and
closed all nine changed files from `2d1f100` to `6b088830`, with zero candidates
or findings. A previous scan with no populated review rows is invalid evidence
and was replaced. Neither result is a complete production audit. Later source
changes need their own exact-revision evidence.

## Bounded receipts

The README receipt badges link here. A receipt applies only to the listed
revision and scope; it is not a release gate, audit conclusion, security
certification, native qualification, or assertion about a later working tree.
The static local receipts below were collected in an isolated development
checkout. Their detailed logs are retained outside tracked source and are not
release evidence.

| Receipt | Revision and time | Scope and observed result | Limit |
| --- | --- | --- | --- |
| Frozen local checkpoint23 | Native-fixture/provisioning/Sonar-label follow-up based on `2d1f1003d7c720f8d1ee5884ec576786358548c2`, 2026-11-04 | All 31 local source/test/static/format/module/workflow/compile checks passed with identical source manifests. Coverage is unchanged at 5,953/7,055 statements (84.37987%). Strict offline zizmor auditor collection has zero findings. A private copied Go distribution rebuilt the real auditor and passed a poisoned-PATH tiny fixture; the old shared writable layout was explicitly denied. | Actual native Windows/macOS and hosted-runner confirmation is separate. CircularQuay's 100% coverage and zero-gosec gates still fail; a 1/1 fixture is not a product gate pass. |
| Frozen local checkpoint22 | Combined auditor/CI-flow/Sonar-remediation capture based on `bfaa66cc116093588dd1053a301bdf90a8ecef28`, 2026-11-04 | All 31 recorded checks passed with identical source manifests: whole-module normal/race, vet, Staticcheck all, formatting, module integrity, both complexity ceilings, actionlint, 13 Windows and six Darwin package test compilations. Coverage is 5,953/7,055 statements (84.37987%). The exact workflow-manifest/public-auditor smoke passes selected checks under a poisoned PATH. Expanded offline zizmor auditor-persona collection reports zero findings. | Coverage still fails 100%; scoped gosec still has eight Linux / nine Windows unsuppressed findings. Compile checks and a tiny synthetic audit do not qualify native OS execution, complete tool/runtime closure, exact-head Sonar, CI identity assertions or production. |
| [Native and cross-build CI](https://github.com/maldous/CircularQuay/actions/runs/41724962521) | `bfaa66cc116093588dd1053a301bdf90a8ecef28`, tree `0afc2f5f7fcaf00f1547b29f849540cf78cfc89f` | Native Linux, macOS and Windows tests and all 26 cross-builds passed. Synthetic merge `3be5764a6b551982bfa4962fc7c35a8c7000df2f` has the same public tree. [Advanced CodeQL](https://github.com/maldous/CircularQuay/actions/runs/41724962531) passed. | Source assurance fails required coverage (5,554/6,441 statements); gosec fails eight unsuppressed findings. Native runners do not qualify providers, reboot/recovery, installers or releases. |
| Managed DPoP diff review | `97b757a` → `bfaa66c`, scan `56553a87-7692-4deb-9432-d0f716652dc3` | Sealed complete review of all 37 changed files; zero reportable findings/candidates. Focused pinned Go 1.27.3 synthetic validation passed. | Complete diff coverage is not a whole-repository, live-provider or production audit. |
| Protected audit-tool fixture | Frozen isolated auditor slice, 2026-11-04 | A real source-profile run on a tiny clean synthetic repository passed with protected Git/Go/gofmt/Staticcheck under a poisoned ambient PATH. Binary, manifest, source and report hashes are retained. Other analyzers were explicitly not run. | Fixture-only Linux evidence, not CircularQuay's gate pass, publisher authentication, transitive runtime closure, atomic execution or an OS sandbox. |
| Frozen local checkpoint21 | Captured DPoP follow-up based on `97b757af8a91c7dce5227f8b6230fa3d98545d1a`, 2026-11-04 | Whole-repository normal/race tests, vet, Staticcheck with all checks, formatting, module tidy-diff/verification, cyclomatic and exact-recursion cognitive ceilings of 10, actionlint, 13 Windows and four Darwin test compilations passed with identical source manifests. Coverage is 5,554/6,441 statements (86.22885%). Signed synthetic issuer-to-MCP tests exercise persistent nonce handling, authorized reads, replay denial and a fresh Bearer downgrade denial followed by acceptance of the same unconsumed proof under DPoP. Schema/parser tests enforce omitted-skew defaults and every skew/proof-age boundary below 30 seconds. | The unchanged 100% gate fails. These are local synthetic and compilation receipts, not native, live issuer/client, cluster, release or exact-candidate Sonar qualification. |
| Sender-constraint finite model | `SenderConstraint.tla` SHA-256 `b962953ac879eb3f433ec9cce77f1843419217dee00d38b5f49f37f5c3b22822`; configuration `a9fd6f75aaf69045dfce9332d76a908b8ef60c63771a612cb8035f7fcd46f244` | TLC completed 1,081,397 generated / 127,067 distinct states at depth 20. Four private guard mutations produced counterexamples for replay, expiry, stale authority and restart persistence. Raw logs and the exact Java/jar identities are retained. | Two fingerprints, one key/JTI, three nonces, a fixed resource and time 0–2 only. The authorization-boundary abstraction assumes trusted clock, OS, store, native lock, issuer and cryptography; it is not whole-program refinement, cryptographic proof or resistance to a host administrator. |
| Checkpoint21 scoped gosec | Same frozen Go inputs, 2026-11-04 | Linux has eight findings (G204 ×6, G103 ×2); Windows has nine (G204 ×5, G103 ×4), with zero Go errors and zero suppressions. Source manifests match. | Both zero-finding gates remain failed; no rule or quality threshold is waived. |
| [Native and cross-build CI](https://github.com/maldous/CircularQuay/actions/runs/41715638268) | `97b757af8a91c7dce5227f8b6230fa3d98545d1a`, 2026-11-04 | Native Linux, macOS and Windows tests and all 26 cross-builds passed. Public readback verified synthetic merge `e9f0b1ed25b755ad86bbb18f8d9e01ed1fc81008` has the same tree as the head. The audit bootstrap fixture passes on native macOS after canonicalizing its temporary path. | Synthetic runner evidence only; no reboot, provider, installer or release-artifact qualification. Source assurance still fails coverage. |
| [CodeQL CI](https://github.com/maldous/CircularQuay/actions/runs/41715638372) | Same exact candidate and matching synthetic-merge tree | Advanced CodeQL completed successfully. Actual Linux gosec reported eight findings with zero suppressions. | CodeQL success does not make the security workflow pass; gosec remains failed. |
| Focused local checkpoint20 | Captured fixture-only follow-up based on `54fb84a2d3ccd65d0cc7c3d7a780c7c5daa40507`, 2026-11-04 | Audit-command normal/race tests, vet, Staticcheck, formatting and both complexity ceilings passed. Darwin amd64/arm64 and Windows amd64 test binaries compiled. Source manifests matched; the only source delta from checkpoint19 is canonicalization of the audit bootstrap test temporary path. | Production Go/module/workflow inputs are unchanged from checkpoint19. This package/compile receipt does not replace whole-repository coverage or native macOS execution. |
| Frozen local checkpoint19 | Captured source manifest based on `42a3806d68a39583584d22e51469278cfa698cac`, 2026-11-04 | Whole-repository normal/race tests, vet, Staticcheck, formatting, module tidy-diff/verification, cyclomatic and exact-recursion cognitive ceilings of 10, actionlint and 12 Windows test-compilation checks passed with identical source manifests. Coverage is 5,224/6,082 statements (85.89280%). Tests exercise real synthetic subprocess output caps and cancellation, signed-repository denial/context paths, and systemd invocation/publication contracts without calling host tools. | Coverage remains below 100%. Pipe wait bounds are not descendant containment; synthetic invocation tests are not systemd/reboot qualification. Exact-candidate native, Sonar and managed audit evidence remain separate. |
| Checkpoint19 scoped gosec | Same frozen source manifest, 2026-11-04 | Explicit 19 owned packages: Linux has 8 findings (G204 ×6, G103 ×2); Windows has 9 (G204 ×5, G103 ×4), zero Go errors and zero suppressions. Source manifests are identical before/after. | Both zero-finding gates fail. Consolidating two systemd calls into one execution boundary reduces the Linux count without clearing the underlying process warning; no rule is disabled. |
| Frozen local checkpoint18 | Captured source manifest based on `4a485258c3a72a7227502f359b4a049996b4ba61`, 2026-11-04 | Whole-repository normal/race tests, vet, Staticcheck, formatting, module tidy-diff/verification, cyclomatic/cognitive ceilings of 10, actionlint and 12 Windows test-compilation checks passed with identical source manifests. Coverage is 5,133/6,048 statements (84.87103%). | Coverage remains below the unchanged 100% gate. Local and compile-only evidence do not qualify native Windows, exact-candidate Sonar, providers, recovery or release artifacts. |
| Frozen local checkpoint17 | Captured source manifest based on `8ed18f35be811f722068a3cf62d3093038fa8d93`, 2026-11-04 | Go 1.27.3 whole-module normal/race tests, vet, Staticcheck, formatting, module tidy-diff/verification, cyclomatic and exact-recursion cognitive ceilings of 10, actionlint and 12 Windows test-compilation checks passed; source manifests are identical. Coverage is 5,114/6,022 statements (84.92195%). | The unchanged 100% gate fails. This bounded local receipt does not qualify native Windows, exact-candidate Sonar, independent audit, providers, recovery or release artifacts. |
| [Linux synthetic GitHub Actions job](https://github.com/maldous/CircularQuay/actions/runs/41649406181/job/125963290833) | `c2b929c7c2ad66ae8c05ca1740192d93dbe208a3`, 2026-11-04 | GitHub-hosted Ubuntu with Go 1.27.3: formatting, module integrity, unit tests, race tests, vet, three bounded fuzz targets, and tracked-Go-source `gocyclo -over 10` passed. All 26 Linux cross-build targets also passed in this source run. | This scoped Linux receipt does not override the Windows/macOS/coverage failures in the same source workflow and does not qualify native runtime behavior. |
| [CodeQL advanced GitHub Actions job](https://github.com/maldous/CircularQuay/actions/runs/41649406421/job/125969631483) | `c2b929c7c2ad66ae8c05ca1740192d93dbe208a3`, 2026-11-04 | Advanced CodeQL build, analysis, and SARIF upload completed successfully after the user-approved transition. | This is a workflow execution receipt, not an audit, alert count, security certification, or claim that every CodeQL query/future revision is clean. |
| Staticcheck | `308670c62198a1a5e168d0c4583686a65098db6b`, 2026-11-03 11:53 UTC | Staticcheck 2026.2.2 (module 0.8.2), built and run with Go 1.27.3 on `cmd/...` and `internal/...`: exit 0. | Local bounded receipt from a development checkout; the revision differs from the GitHub Actions receipt and later changes require a rerun. |
| govulncheck | `308670c62198a1a5e168d0c4583686a65098db6b`, 2026-11-03 11:53 UTC | govulncheck 1.9.0 with Go 1.27.3: zero reachable vulnerabilities. | The report retained three required-module advisories without reachable calls: GO-2026-6866 and GO-2026-6860 in `golang.org/x/net@v0.61.0`, and GO-2026-5932 in `golang.org/x/crypto@v0.58.0`; it is not an all-dependency clearance. |
| Cyclomatic complexity | `c2b929c7c2ad66ae8c05ca1740192d93dbe208a3`, 2026-11-04 | The successful Linux job ran pinned `github.com/fzipp/gocyclo@v0.6.0` against all tracked Go sources; no function exceeded 10. | Cyclomatic complexity is distinct from Sonar cognitive complexity and does not make the Sonar gate pass. |
| Shell and packaging analyzers | Development checkout, 2026-11-03 | ShellCheck 0.9.0 and shfmt 3.12.0 passed tracked shell templates; Hadolint 2.12.0 passed the Dockerfile; Docker Compose 5.1.4 parsed the example Compose files without interpolation; actionlint 1.7.12 passed workflows. | Static/template-only scope. systemd verification could not complete because the example executable is intentionally absent; no image or service runtime is qualified. |
| PowerShell and workflow security analyzers | Development checkout, 2026-11-03 | PSScriptAnalyzer 1.25.0, invoked with PowerShell 7.6.9 using `-NoProfile -NonInteractive`, reported no finding for `packaging/windows/enroll-example.ps1`; zizmor 1.32.0 offline, config-free strict collection reported no finding for `.github/workflows`. | No Windows execution, registry/service test, remote-action provenance, or online zizmor audit occurred. |
| TLA+/TLC finite model | Development checkout, 2026-11-03 | TLA+ tools 1.8.0 completed the checked `formal/Delivery.tla` / `Delivery.cfg` finite model. | The model has bounded assumptions and does not verify operating systems, TLS, provider SDKs, the specification itself, or complete implementation conformance. |
| Frozen local checkpoint6 | Captured source manifest based on `4886fc8660706909376f3543ad7cab186d9256b9`, 2026-11-04 | With Go 1.27.3 and a protected temporary directory, normal/race tests, vet, gofmt, module tidy-diff/verify, Staticcheck 2026.2.2, `gocyclo -over 10`, exact-recursion `gocognit -over 10`, and actionlint passed; all 12 affected Windows packages compiled. Before/after source manifests were identical. | Local bounded evidence only. Its native statement profile without `-coverpkg` was 4,787/5,742 (83.3682%), below the unchanged 100% gate; compilation does not qualify native Windows behavior, Sonar, providers, deployment, or release artifacts. |
| Local secret scan | Same checkpoint, 2026-11-04 | gitleaks scanned 213 copied nonignored files and returned zero findings. The receipt retains the scanner binary SHA-256; its version output was the build-process default. | This is a bounded copied-source scan, not a credential, history, release-artifact, or production security certification. An earlier incorrect CLI attempt is retained as evidence history, not a code failure. |
| `internal/identity` package | 2026-11-04 | Package normal/race/vet and both complexity checks passed; atomic coverage was 183/183 statements (100.0%). | Package-scoped receipt only; it does not qualify the repository, native identity integrations, or a release. |
| Frozen local checkpoint7 | Captured source batch based on `41a8da186c71d94b420303328fbe06fcee708602`, 2026-11-04 | Go 1.27.3 normal/race/vet, gofmt, tidy-diff, module verification, Staticcheck, `gocyclo -over 10`, exact-recursion `gocognit -over 10`, actionlint, and all 12 affected Windows compile-only checks passed. Before/after manifests covering Go, JSON, workflows, and canonical model fences were identical. The whole-module merged `-coverpkg` profile was 4,881/5,744 statements (84.97562674%). | The retained source manifest identifies this frozen local batch. Coverage remains below the unchanged 100% gate; Windows compilation is not native execution, and this receipt does not replace CI, Sonar, managed audit, provider, deployment, or release qualification. |
| Durable-fence TLA+/TLC model | Development checkout, 2026-11-04 | TLC completed the model with `Delivery.tla` SHA-256 `639ddb081f072604cc513aa015950ea41f29279f5f2a59e089754ae017e6b76c` and `Delivery.cfg` SHA-256 `78a61de041ef8a20515ab69439443cc3446e65af566724bbca90dd218e64c096`: 14,818,232 generated states, 76,360 distinct states, depth 20, empty queue, and no error. | The model assumes a trusted store and lock. It does not qualify native Windows permission-failure behavior, full implementation conformance, or recovery on an operator host. |
| Pinned gitleaks scan | Captured local batch, 2026-11-04 | gitleaks v8.30.1 scanned a captured 216-file inventory before this receipt update with SHA-256 `0caf87f382e74e55bee6d7ef3f4f7ac9c839bb4d17183764ac3ff0e29752e74b` and returned zero findings. | This is a bounded working-tree scan, not a history, artifact, credential, or production security certification. |

## Open gates and non-badge evidence

These results are intentionally not represented by a green README badge.

| Evidence | Current recorded state | Consequence |
| --- | --- | --- |
| Published candidate `2d1f1003d7c720f8d1ee5884ec576786358548c2` | Tree `214921fc8eaa2a4e5d995ae2cc224ebcca23d847`; source run `41729294511` rejects the hosted-runner manifest before source checks. Native Windows has two auditor fixture failures; macOS has two protected-directory drift-fixture failures. | These are actual failures, not the expected coverage-only gate. A 0/0 report does not measure coverage. Corrected provisioning/fixtures need fresh native evidence. |
| Direct Sonar diagnostic `2d1f100` | CE `8ef2b1a0-9fd1-4c4a-8901-79fd23036f14` succeeded with analysis `4109f6f5-d61d-4388-962e-44b2f8ab8bcc`; requested SCM revision is logged. Four prior issues are absent, but go:S1192 in flow.go remains open. Gate ERROR: 78.3% overall / 79.7% new coverage; native statements match 5,953/7,055. | Current-analysis/hotspot-list APIs return 403. Direct diagnostic scanning does not satisfy the unbound enterprise runtime gate. Coverage and zero-issue gates fail; a later fix needs a fresh scan. |
| Managed auditor/CI-flow diff review | `bfaa66c` → `2d1f100`, scan `8341c847-cb2c-4f6c-92f9-2d99de58e3e9` sealed all 29 files with complete coverage and zero findings/candidates. Focused assurance validation passed; CLI/MCP validation had protected-temp-directory limitations. | Complete diff review does not replace native tests, complete runtime/tool qualification, live providers or a production audit. |
| Exact-head SonarQube upload `bfaa66c` | CE `f8564a8c-d33e-470a-a829-a9b2558305dd` succeeded, returning analysis `8a844c31-e83c-4be8-ba85-2bb12be93b69`; scanner logs bind the requested SCM revision. Gate ERROR: 79.9% overall / 82.4% new coverage, plus four open code smells (three argument-count findings and one builtin shadow). The current-analysis and hotspot-list APIs returned 403; the metric reports zero hotspots. | No independent current-server-analysis fence or hotspot-list clearance is claimed. Both coverage thresholds and zero-issue requirements fail. The prior Nov10 token-expiry warning remains retained. |
| Exact SonarQube `97b757a` | CE `67acd20a-9998-41c7-a011-4ecce249c27a` completed; current analysis `1c0550bb-bc88-4d31-b978-af4b070a73cc` matches the exact revision/tree. Zero issues and hotspots; gate ERROR at 79.4% overall / 81.8% new coverage. Fresh native statements are 5,224/6,082 (85.89280%). | Both 100% coverage thresholds remain failed. The token-expiry warning for 2026-11-10 is retained; this result does not qualify a later revision. |
| Published candidate `54fb84a2d3ccd65d0cc7c3d7a780c7c5daa40507` | Exact tree `325e75f2016bfd055a1289e4203c583f062c29bc`. [Source run 41713357341](https://github.com/maldous/CircularQuay/actions/runs/41713357341) tested synthetic merge `04f62e0` with a matching public tree: native Linux/Windows and all 26 cross-builds passed. macOS failed only the audit cancelled-bootstrap fixture with `symbolic private directory`; other listed packages passed. [Security run 41713357268](https://github.com/maldous/CircularQuay/actions/runs/41713357268) passed CodeQL; actual Linux gosec reported 8 unsuppressed findings. Source coverage is 5,224/6,082 (85.89279842%). | The macOS fixture used a temporary path containing a system alias; checkpoint20 canonicalizes only that test path. Neither cross-compilation nor local Linux tests establish its native fix. Coverage and gosec remain failing. |
| SonarQube (`54fb`) | CE `e6af4d51-c9be-4860-844d-89715fd6b4c5` completed with current analysis `c8623fc7-2362-4019-9ff8-88546a7de9a6` and exact revision fences. Zero issues, smells, bugs, vulnerabilities, hotspots and duplication. Gate ERROR: 79.4% overall / 81.8% new coverage; the Nov10 token-expiry warning remains. | The unchanged 100% coverage and warning requirements remain unmet. |
| Managed audit delta (`42a` → `54fb`) | Sealed scan `94b6f845-d4f0-4df1-b630-31c673f6294b` reviewed all 19 authoritative source/test items with complete diff coverage and zero findings. Focused package validation passed. | Complete coverage applies to this diff, not a full repository or production audit. Host isolation/tool provenance, live providers, native systemd, recovery and release qualification remain separate. |
| Published candidate `42a3806d68a39583584d22e51469278cfa698cac` | Exact tree `d592a8ba1f2c9acb1106982d6eb82a9d94af4921`. [Source run 41706881432](https://github.com/maldous/CircularQuay/actions/runs/41706881432) tested synthetic merge `834345b94f89aa3d1287cf747a85bbd9131f61eb`, whose public tree matches the head: native Linux/macOS/Windows and all 26 cross-builds passed. CodeQL passed in [security run 41706881558](https://github.com/maldous/CircularQuay/actions/runs/41706881558); Linux gosec retained 9 findings. Whole-module coverage is 5,133/6,048 (84.87103175%). | The unchanged 100% gate fails. The source-profile job has no Sonar endpoint; the separate exact-head analysis below supplies Sonar evidence. These are synthetic native tests, not reboot, live-provider, installer or release qualification. |
| SonarQube (`42a`) | CE `1b97ca13-5ab7-48ec-98b8-f2e0cc96e96b` completed with current analysis `534d0cde-eb0b-45b7-9af4-72cd824f3ff0` and exact revision fences. Both prior smells are resolved: zero issues, code smells, bugs, vulnerabilities, hotspots or duplication. Gate ERROR remains at 78.3% overall and 81.5% new coverage. The 2026-11-10 token-expiry warning remains. | Zero issues does not pass the unchanged coverage or warning requirements. |
| Windows cross-static (`42a`) | Exact-head/index/content fences matched before and after explicit analysis of all 19 owned packages. Staticcheck passed; gosec retained 9 findings (G103 ×4, G204 ×5), with no suppressions or exclusions. | This is cross-static evidence, distinct from the passing native Windows suite above; it does not clear the zero-finding gate. |
| Managed audit delta (`4a` → `42a`) | Sealed scan `0100370b-bbe7-4a3c-91ad-5b78c011cd04` reported zero reportable findings with partial completeness. Native Windows runtime and prior broader gaps were deferred by that scan. | A separate native test pass does not retroactively expand audit scope. This is not an audit pass. |
| Published candidate `4a485258c3a72a7227502f359b4a049996b4ba61` | Exact tree `f8c53890e348cf8b7fdb56a749b0f8aaca0d00d5`: native Linux/macOS, all 26 cross-builds, and CodeQL passed. Security gosec failed with 9 findings. Source assurance was 5,114/6,022 (84.92195284%), below the unchanged 100% gate. Native Windows failed: a module-proxy timeout for googleapi blocked CLI/resolve/source setup, while tests that ran also failed trusted-directory preflight and offline `Current` Git temporary-file creation. | The proxy failure does not make all Windows assertions inconclusive; observed test failures need separate diagnosis. No passing subset qualifies Windows, security, providers, deployment, or a release. |
| SonarQube (`4a`) | CE task `79b7e0cd-e546-4d68-848e-be7ab4e90113` and analysis `9c02dbab-ce75-4d3b-828e-de1bf483d2c9` completed with zero bugs, vulnerabilities, hotspots, and duplication, but gate ERROR: 78.4% overall and 81.6% new coverage, plus `go:S1192` (five `rev-parse` literals) and `go:S978` (rune issue). The 2026-11-10 token-expiry warning remains. | Zero bugs/vulnerabilities/hotspots/duplication does not pass the gate; resolve both smells and coverage without suppressions, then rescan an exact candidate. |
| Managed audit delta (`e553` → `4a`) | Sealed diff `cad20f55-3074-47e5-8dce-f7648dc9141b` reported zero new reportable findings with partial completeness; host sandbox/tool provenance and native Windows remain deferred. | This is not an audit pass or full recheck. |
| Captured follow-up published in `42a` | The batch fixes the two Sonar smells and adds protected positive fixtures/work-bin coverage for platform/CLI plus isolated offline `Current` per-call verification temporary state. | Checkpoint18 and exact-candidate receipts are recorded above. No credentials, infrastructure or native configuration changed. |
| Published candidate `8ed18f35be811f722068a3cf62d3093038fa8d93` | Exact tree `b42f899e839d85febc7ff06105b60cf07c41e5bc`: native Linux/macOS and all 26 cross-builds passed; CodeQL passed. Native Windows failed with two empty SSH probes and a path-oracle failure. gosec still has 9 findings. CI whole-module coverage was 84.9957%, below the unchanged 100% gate. | Passing selected jobs do not qualify Windows, security, providers, deployment, or release behavior. Diagnose the Windows failures and retain a fresh native receipt. |
| SonarQube (`8ed`) | CE task `dbd6d2fa-3814-4b06-a01e-e3bc7632a184` and analysis `81a9ef98-5a95-4e3d-a706-39362381c61e` completed with zero open issues, but the gate is ERROR: 78.6% overall and 82.1% new coverage. The receipt retains the 2026-11-10 token-expiry warning. | Zero issues is not a passed quality gate. Coverage remains below the unchanged 100% requirement; resolve the warning through normal credential lifecycle handling. |
| Captured source-integrity/native-environment follow-up | Integrated v1 pre/post Git-tree/raw-byte manifests, ignored-input rejection, replacement-object/configuration fences, physical top-level source validation and bounded reads. Windows Git/SSH receives OS-derived ProgramData; native directory tests compare filesystem identity. A real SDK catalog test covers omission of raw reads without a grant. CI audit binaries/evidence are external to source, Sonar scratch is output-scoped, and Go tests, coverage, vet, formatting and Go-aware analyzer arguments come from the source manifest. | Local implementation is not qualification. Manifests are neither an atomic snapshot nor a host sandbox/tool-provenance guarantee. Fresh native Windows, exact-candidate Sonar and independent evidence remain required. |
| Published candidate `e55327d513b0753ce08ae82adffbc4cb3cd5d2d4` | Exact tree `1523d6ac067ef94270bd015fa60ad296f086f021`: native Linux/macOS, all 26 cross-builds, and CodeQL passed. Native Windows failed; the prior explicit-verifier explanation was refuted. gosec still has 9 findings. | Passing selected jobs do not qualify Windows, security, deployment, providers, or a release. The Windows failure needs diagnosis and a fresh native receipt. |
| SonarQube (`e553`) | CE task `bb229fa5-0bb7-4db0-bb8e-f88b9f6d99c1` completed successfully; analysis `50ea05ae-0367-4ca7-aa16-e5111349b6c4` has zero open issues but gate ERROR: 79.1% overall and 83.4% new coverage. The receipt records token expiry on 2026-11-10. | Zero issues is not a passed quality gate. Coverage remains below the unchanged 100% requirement; the token warning needs normal credential lifecycle handling, not a broad credential workaround. |
| Managed static audit (`e553`) | Sealed scan `89f4baf1-d6f3-4565-b53e-c4f788b0fe90` retained eight validated findings: one high (auditor-host execution), five medium (assurance binding/PATH, SIGTERM, Windows DLL, and fresh-scope controls), and two low (UID and DPAPI ACL). | Completeness is partial and this is not an audit pass. Keep detailed attack paths owner-held; three assurance issues remain open. |
| Captured remediation batch (checkpoint10) | Based on `e553`, the live-generation, legacy-state, quota identity, staged DPAPI ACL, protected Windows loader directories, SIGTERM, and MCP request-cancellation changes passed whole-repository normal/race tests, vet, Staticcheck, formatting, module checks, both complexity ceilings, workflow lint, and 12 Windows test-compilation checks. Source manifests matched; merged statement coverage was 4,899/5,765 (84.9783%). | Coverage fails the unchanged 100% gate. Native Windows, exact-candidate Sonar, and independent remediation qualification remain separate; three assurance findings remain open. |
| Windows/macOS cross-static analysis | Windows amd64 and macOS amd64/arm64 Staticcheck passed for the captured source. Windows gosec initially reported two DPAPI length-conversion findings; explicit input/output bounds cleared both, with nine G103/G204 findings remaining. | Cross-static analysis does not execute Windows or macOS behavior. Linux and Windows gosec reports cover different native files; neither lane passes the zero-finding gate. |
| [Source workflow run 41686510514](https://github.com/maldous/CircularQuay/actions/runs/41686510514) | The open PR at `41a8da186c71d94b420303328fbe06fcee708602` was tested as synthetic merge `6fd454e6bc77888f4e40d7eec37e6bbe4c7a9748`. Native Linux and macOS passed, all 26 cross-builds passed, and Windows job 126093565982 failed only `internal/repository` `TestSignedRepositoryAcceptanceAndRollback` with `trust_failure: repository_signature`; the Windows platform package passed. Its production assurance profile with `-coverpkg` was 4,828/5,742 statements (84.0822%), below the unchanged 100% threshold. | This PR is open and the jobs test GitHub's synthetic merge, not a release or deployment. Cross-build and selected native results do not establish Windows repository behavior, providers, recovery, or production qualification. |
| [Security workflow run 41686510624](https://github.com/maldous/CircularQuay/actions/runs/41686510624) | CodeQL passed; gosec failed with 9 unsuppressed findings. The frozen static receipt also records zero reachable govulncheck findings and one required-module advisory, `GO-2026-5932` in `golang.org/x/crypto@v0.58.0`. A later local dependency update reports zero govulncheck findings. | Do not claim a clean security workflow or dependency clearance until the gosec findings and module advisory are resolved or separately dispositioned and the exact candidate reruns. The historical advisory remains evidence for 41a8. |
| SonarQube | The Compute Engine task `e52a9f8f-b66c-48f2-8a8b-425a533cab12` for `41a8da186c71d94b420303328fbe06fcee708602` succeeded with analysis ID `dbe51b3f-8bae-4075-9ce0-0a72482e42c3`, but its gate is ERROR: 78.3% line coverage (6,134/7,833 lines), 82.5% new coverage, and two code smells (`go:S107` and `go:S1192`); reported bugs, vulnerabilities, hotspots, and duplication are zero. | Sonar line coverage and Go statement coverage have distinct scopes and neither changes the 100% gate. At that receipt the two smells required a recheck; later e553 and 8ed analyses record zero issues while coverage still fails. |
| Checkpoint7 follow-up (historical) | The captured dependency update had a focused zero-findings govulncheck result, and the two Sonar smell fixes and SSH-fixture hypothesis awaited exact-candidate checks. | Later e553 Sonar cleared the two smells; native Windows refuted the signing hypothesis. Those later receipts are recorded above without converting failed gates into a pass. |
| MCP listener lifecycle regression | Captured local batch, 2026-11-04 | The retained bounded reproduction failed before the cleanup fix; focused `internal/policy`, `internal/mcp`, and `internal/cli` normal/race/vet checks passed after it. | This is a focused local lifecycle receipt only; it does not qualify the full repository, native platforms, or release behavior. |
| Managed static audit | Sealed scan `0ffa3270-c1e3-4dc7-935f-ce57145c52b2` at `41a8` reported zero validated findings, including no reportable exploit from the stale repository-sync documentation contradiction. Its completeness is partial. | This is not an audit pass and does not qualify native behavior, live providers, installations, services, or the current working tree. |
| Frozen local checkpoint6 (historical bounded batch) | At capture, the frozen source batch passed its retained normal/race/vet/static/module/format/complexity/workflow checks and 12 Windows compile-only checks with unchanged source manifests. It contained named authorities, immutable GitHub bootstrap, repository runtime, resolver, snapshot/fallback, Azure transport, and Windows-fixture hardening. | Its native statement coverage was 4,787/5,742 (83.3682%), below the unchanged 100% gate. It is local evidence for that frozen batch only; later 41a8 CI/Sonar/audit receipts below record the subsequent candidate state. |
| [Source workflow run 41677515031](https://github.com/maldous/CircularQuay/actions/runs/41677515031) | At `4886fc8660706909376f3543ad7cab186d9256b9` (tree `132f80e52ead1f61b08239fec97ad440829d8172`), native Linux and macOS passed and all 26 cross-builds succeeded. Windows job 126061996996 failed two fixtures. The workflow's source-coverage result was 4,072/4,969 statements (81.9481%); the retained local Sonar profile measured 4,071/4,969 (81.928%). | Both coverage results fail the unchanged 100% gate. These are outcomes at 4886; later 41a8 receipts, including its distinct Windows repository failure, are recorded separately above. |
| [Security workflow run 41677515226](https://github.com/maldous/CircularQuay/actions/runs/41677515226) | CodeQL passed; gosec failed with 11 unsuppressed findings. | Do not claim a clean security workflow until all findings are resolved and the exact candidate reruns. |
| SonarQube (historical `4886`) | The Compute Engine task for `4886fc8660706909376f3543ad7cab186d9256b9` (tree `132f80e52ead1f61b08239fec97ad440829d8172`) succeeded as task `7dbc9ce0-8aa7-4df5-81e8-5c7cfdba6b71` with analysis ID `9ea50dd4-ab95-4a3b-aff7-d1510405f708`; its quality gate is ERROR: one critical MCP-action-literal code smell, 75.7% overall coverage, 78.8% new-line coverage, zero reported bugs/vulnerabilities/hotspots/duplication, and an analysis-token-expiry warning. | This is the outcome at 4886 only. Later 41a8 Sonar evidence is recorded above; neither receipt passes the unchanged gate. |
| Independent managed audit (historical `4886`) | The static-only recheck in a clean worktree at `4886fc8` completed with one medium Azure workload-identity transport finding. | This is not an audit pass and does not qualify native behavior, live providers, installations, or services. Later audit evidence is separately retained above. |
| [Source workflow run 41660198848](https://github.com/maldous/CircularQuay/actions/runs/41660198848) | At published `41ea71365464417875c7162cc98c87262cda47f8`, Linux job 126002042064 failed `go mod tidy` before tests because a new test's SSH import left a metadata-only module diff; coverage job 126002042348 failed the unchanged 100% gate at 3,967/4,860 statements (81.6255%). macOS job 126002042623 completed `go test ./...`, and all 26 cross-builds succeeded. Windows job 126002042419 failed platform `TestFailedCreationCleanupPreservesAReplacement` and a signed-repository Git commit exit 128; its other packages, including new Windows CreateDACL tests, passed. | This is the outcome at 41ea. Later 4886 and 41a8 receipts record subsequent candidates; no repository-wide CI-pass claim or native-platform qualification follows. |
| [Security workflow gosec](https://github.com/maldous/CircularQuay/actions/runs/41660198518/job/126001310144) | CodeQL job 126001309575 and Staticcheck/govulncheck succeeded; gosec failed with 11 unsuppressed findings. All three G304 findings are resolved in this revision. | Do not claim a clean security workflow until all findings are resolved and the exact candidate reruns. |
| SonarQube (historical `41ea`) | The Compute Engine task for published revision `41ea71365464417875c7162cc98c87262cda47f8` (tree `d6bfeedf511434a9d09afef931c6109a295e19c7`) succeeded with analysis ID `06a03452-7d22-4590-9c47-8dfdb6f0d455`, but its quality gate is ERROR: one `go:S3776` cognitive-complexity smell at `internal/cli/init_test.go` line 40 (cognitive complexity 16; allowed 10), 75.5% overall coverage, 78.7% new-line coverage, zero reported bugs/vulnerabilities/hotspots/duplication, and one token-expiry warning. | This is the outcome at 41ea only. Later 4886 and 41a8 receipts are separately retained; the unchanged gate remains unmet. |
| Independent managed audit (historical `41ea`) | The audit at `41ea` confirmed six findings: three high and three medium. The high findings cover Windows path ownership/ancestor trust and environment bootstrap aliases/field narrowing; the medium findings cover HTTP authorization expiry, cloud-SDK transport controls, and an empty Infisical path scope. | This is the historical audit result at 41ea, not an audit pass or a reproduction guide. Later static audits are retained separately; no native, provider, installation, or service qualification follows. Detailed evidence remains in the owner-held security channel. |
| SBOM and mutation sample | An SBOM/license inventory was generated; the bounded mutation sample scored 81.99%. | Neither result certifies security or satisfies the required complete mutation evidence. |

## Managed-service design status

The linked managed-key document and README describe the v1 North Star. The
authenticated key-mapping endpoint, managed administration UI/API, protected
local custody, bundle-bound remote unlocking, and new CLI/MCP management paths
are not implemented or qualified in this source checkpoint. Upstream plugin
and hosting documentation is research evidence, not a CircularQuay integration
receipt. The hosting/capacity/revenue calculations are explicit assumptions,
not benchmarks, customer traction, approved prices, or operating accounts.
No service, paid plan, credential, or BAU infrastructure was created or changed.

## Integration implementation and qualification status

The [integration inventory](integrations.md),
[cloud-provider validation matrix](integrations/cloud-provider-validation.md),
and [compatibility gateway](integrations/compatibility-gateway.md) describe v1
contracts. They are not receipts for every listed integration.

The per-authority private-CA candidate adds paired protected PEM/digest fields,
strict certificate parsing, authority-only replacement roots, load/verifier
integrity checks, fingerprint binding and writer/replay-state isolation.
Source review and formatter/JSON checks are separate from actual test execution;
the exact candidate still needs hosted normal/race/native/security results.

The tagged WireMock suite implements a real loopback HTTPS process harness for
the OpenBao KV v2 adapter, with six synthetic scenarios, exact and unmatched
request checks, private state, verified JAR copying and bounded process cleanup.
Its Linux/macOS/Windows workflow and local instructions use the same suite.
No Java/WireMock scenario has yet executed in this checkpoint. An agent's two
compile-only/vet batches ran despite the local resource hold; they executed no
test bodies, had no retained exact-source/environment fence, and are not an
integration pass. The root stopped further local execution. Retain the private
`wiremock-compile-only-execution-20261104.json` receipt.

The ephemeral hosted Sonar workflow is a new producer with strict coverage,
warning, issue, hotspot and current-analysis checks. It uses a fresh isolated
server and a finite project-scoped credential. Definition and source review do
not establish a successful analysis or the independent auditor's complete
runtime binding. Dependency review likewise needs its own real PR result.

Keycloak realm export and OpenBao fixture contracts are provided; a real
issuer/provider container suite has not run. LocalStack runtime profiles,
external GCP federation, SPIFFE/SPIRE adapters, compatibility gateway endpoints,
managed UI/API/local service, online introspection/revocation, hardware/keychain
adapters and hosting integrations remain unimplemented or unqualified as
described in their respective contracts. No new schema field advertises those
unimplemented capabilities. No production deployment or release follows from
these design documents or development candidates.
