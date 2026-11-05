# Qualification matrix

This matrix distinguishes implementation from independent or live qualification. The source is in development; no production qualification, certification, availability SLA, or universal provider/client compatibility is claimed.

| Capability | Implementation | Operational evidence |
| --- | --- | --- |
| Strict policy and complete generations | In development; consult package tests | Independent review pending. |
| Local literal configuration | Synthetic example provided | Installed-artifact run pending. |
| SOPS/age and Transit | In development | Selected-version round trip, denial and native-key custody pending. |
| Cloud/SaaS readers | Connector-specific; inspect `cq providers` | Live-provider qualification pending; mocks do not qualify accounts. |
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
| Remote discovery and administration | [Complete v1 contract](management-and-discovery.md) and [security controls](administration-security.md) defined; no managed control-plane implementation | Runtime local CRUD/bootstrap and four-operation MCP do not establish remote administration, enrollment or generic service discovery. |
| Product-wide assurance profiles | [Default/strict contract](security-profiles.md) defined; no profile field or unified enforcement in the current policy model/schema | No strict-mode, downgrade-resistance or enterprise-compliance qualification is claimed. Existing baseline controls require their own exact evidence. |
| Client-side composition | Bounded JSON master/includes resolver, native CLI integration and private-domain guards added as source changes | Source/static review only; new tests have not run under the shared-host execution hold. Full hierarchy/profile enforcement and service parity are not established. |
| Canonical hierarchy and lifecycle | [Governance](configuration-governance.md) and [lifecycle](credential-lifecycle.md) contracts defined; current schema remains the flat local runtime policy | Group/default/lock compilation, managed sessions/SCIM, provider-native mutation, scanner gates and cross-interface lifecycle administration remain unimplemented. |
| Hosted KMS and age associations | Both associations are required by the hosted enrollment design; local age, hosted age-key retrieval and remote KMS custody remain distinct | No hosted enrollment, provider provisioning, key-service endpoint, policy/SDK/MCP integration or live KMS qualification is established. Current SOPS source still rejects cloud KMS and key groups. |
| Automatic startup and integration readiness | Source audits and a single v1 contract are being reconciled; current init generates synthetic configuration/fixtures | Real first authentication, authorized profile/key setup, fast repeat reuse and configured-integration partial/uncertain/headless recovery are not implemented or qualified. |
| Default metrics and early opt-out | Current native telemetry defaults off and rejects disabled configuration with a retained endpoint | The required default-on bounded metrics and CLI/config opt-out before affected client/reference/store/exporter/network work remain implementation and qualification gaps. Browser demo metadata does not establish native behavior. |

## Source workflow scope

GitHub readback on 2026-11-05 confirms active Gitflow branch rules for `main`,
`develop` and `release/**/*`, with eight GitHub Actions checks required and no
bypass actors. The checks include `Hosted ephemeral SonarQube gate` and
`SDK qualification`, alongside source/native/security/CodeQL checks. The rule
requires an up-to-date branch, pull-request approval, last-push approval and
resolved review threads, blocks deletion/force pushes, and requires CodeQL
results at all alert thresholds. Active `v*` tag rules prevent updates,
deletion and non-fast-forward changes. This is enforcement evidence, not a
passing analysis, branch-merge, artifact-signing or release qualification.
The current failed checks remain blocking. Receipt
`checkpoint57/github-approved-hardening-pr-readback.json`, SHA-256
`b8752510745fe719e15a2f728443ff7e812cea7f67103bf1d5a620fa99c6c7a5`,
retains the exact before/after rules and CQ draft-PR metadata readback.
Historical administrative HTTP 403 results remain historical; these successful
ruleset reads and authenticated browser saves do not establish every other
administrative permission or release-protection setting.

The whole-product default/strict design has an independent source review of
the policy schema and profile/administration contracts. It identifies the
missing authenticated context input and unified enforcement engine as release
blockers. It also requires strict audit to override a local false preference,
independent hosted authorization, signed offline exceptions, capability-bound
custody and durable anti-downgrade/recovery state. These are tracked by
[UIF-168](https://linear.app/maldous/issue/UIF-168), with exact-candidate
whole-product evidence in [UIF-174](https://linear.app/maldous/issue/UIF-174).
The review is design/source evidence, not a completed security audit or a
strict-profile implementation pass. Default mode retains the mandatory
baseline and the same engineering/release gates.

The committed workflow defines Linux formatting, module-integrity, unit, race, vet, bounded Go-fuzz and both complexity ceilings. The cognitive check uses the pinned
type-aware analyzer and rejects warnings or fallback diagnostics. It also defines synthetic native test jobs for the GitHub-hosted `macos-15-intel` and `windows-2025` runners, plus Linux cross-compilation for the release build matrix. Workflow definition alone is not a result; the revision-scoped receipts below, rather than this definition, record actual runner, tool, and platform outcomes.

The native jobs exercise only synthetic Go behavior on the runner architecture. They do not prove release archives, Windows ARM64 behavior, DACL/DPAPI service isolation, macOS signing/notarization, service-manager enrollment, reboot recovery, provider connectivity, or consumer installation.

The source audit profile invokes Staticcheck, gosec, govulncheck, gitleaks and actionlint when they are selected in the protected external tool manifest. They remain optional in the source profile, so an unselected analyzer is reported as `not_run` rather than a pass. No analyzer is resolved from ambient PATH. Source Sonar also remains `not_run` until scanner/runtime/remote-rules binding exists; the direct diagnostic route is separate. The enterprise profile and the 100% owned-statement/Sonar requirements remain unchanged; no absent external check, unexecuted fuzz target, cross-build, or configured workflow satisfies them.

Local complementary tooling has exercised TLC, ShellCheck/shfmt, PSScriptAnalyzer, Hadolint/Compose, offline zizmor, Syft SBOM/license inventory, and a bounded mutation sample. The retained development receipts are described in the session handoff; these checks do not yet have a complete committed CI toolchain or release-artifact receipt. Native reboot/restore/provider trials remain separate requirements. No sampled result, static script analysis or SBOM inventory establishes production qualification.

Record each qualification receipt with a source commit, artifact digest, platform/provider/client version, setup, test outcome and sanitized evidence link. Do not mark a row tested based on a plan, cross-compilation, generated manifest, or absence of a complaint.

Existing operational machines are outside this implementation's deployment scope. Operators install from documented artifacts and provide their own installation feedback.

## Exact b30f1308 hosted evidence

Published source `b30f1308bd384b04201df6d637fe016f8c8ba029`, tree
`650214efd4727ad2bec42f508868ddbec22dc9cc`, has exact public readback of
eleven selected paths and all 466 working files, clean at publication. Its
tested merge `e6b26b5f474385909929b1eead86acae04fbe941` has that same tree.
These results apply to that revision only.

[Source run 41940572706](https://github.com/maldous/CircularQuay/actions/runs/41940572706)
fails the new `TestVerifyRejectsThresholdAliasAndExpiry`: its expiry case
retains the earlier two-signer threshold and receives a threshold denial.
The captured Go profile is **8,666/10,412 statements (83.23088744%)** from a
failing test run, not a successful coverage qualification. Required tagged
integration remains unrun. Returned cross-build successes establish only
their individual compilation results.

[Security run 41940572849](https://github.com/maldous/CircularQuay/actions/runs/41940572849)
completes CodeQL analysis/upload, but its toolchain job fails Staticcheck
SA1019 on deprecated ECDSA coordinate access in the verifier and fixture.
Later gosec/govulncheck success is not established; this does not clear older
findings or establish an empty CodeQL alert inventory.

All nine [Browser jobs](https://github.com/maldous/CircularQuay/actions/runs/41940572887),
three [WireMock jobs](https://github.com/maldous/CircularQuay/actions/runs/41940572953)
and [Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41940572699)
pass their recorded scopes. [SDK qualification](https://github.com/maldous/CircularQuay/actions/runs/41940572896)
fails. Coverage, dependency and platform failures require their own exact
diagnosis; passing synthetic browser/provider jobs do not qualify SDKs.

[Sonar run 41940572785](https://github.com/maldous/CircularQuay/actions/runs/41940572785)
completes CE and exact current-analysis/revision checks with **zero issues,
hotspots, duplication and violations**. The gate remains **ERROR** at
**64.2%** coverage. Required new-code metrics are absent and the Python
3.15-to-3.14 analyzer warning remains; scanner exit 3 is failure. Its coverage
and the Go statement profile have different denominators.
`checkpoint57/b30f1308-hosted-evidence.json`, SHA-256
`f7eca6cbc9c4bdc3a1f730a9ea241d79d0902f7e9d4ce01bcf0e3e79403d942b`,
retains the bounded observation. No gate has been weakened and no workflow
was manually rerun or cancelled for this observation.

## Exact 77435c34 hosted evidence

Published source `77435c34a8bce5b7f23f9577c8eeaaf5c98cd5ee`, tree
`d884d7be255674a3c7877d50c8b6f57c9315a8c7`, has public readback of 25 selected
files and all 459 working files, clean at publication. The bounded hosted
receipt binds synthetic merge `e7af96be22d9df0230ccce7bd42e9028df319d1c` to
that same tree. This evidence qualifies neither later source bytes nor a release.

[Source run 41933069506](https://github.com/maldous/CircularQuay/actions/runs/41933069506)
passes source identity before/after checks, repository policy, formatting,
unit tests and vet. Linux synthetic complexity and Windows synthetic tests
pass. Owned coverage is **8,500/10,224 statements (83.13771518%)**, failing the
unchanged 100% gate; required tagged integration is `not_run`. Returned selected
cross-build successes are compile evidence only, not native installation or
reboot/custody acceptance. The connector's first job page does not establish
unreturned matrix entries.

All nine returned
[Browser jobs](https://github.com/maldous/CircularQuay/actions/runs/41933069656)
pass across Node 22.23.4, 24.22.2 and 26.13.0 on macOS, Windows and Ubuntu.
That is workflow evidence, not a browser coverage or full product acceptance.
All three returned
[WireMock jobs](https://github.com/maldous/CircularQuay/actions/runs/41933069630)
and [Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41933069548)
pass in their recorded scope. Individual current Security and SDK conclusions
were not fetched in this snapshot and remain unknown.

[Sonar run 41933069542](https://github.com/maldous/CircularQuay/actions/runs/41933069542)
completes CE and exact current-analysis/revision fencing with **zero issues,
zero hotspots and zero duplication groups**. Its quality gate remains
**ERROR**: coverage is **63.9%**, required new-code metrics are absent, and
the Python 3.15-to-3.14 analyzer warning remains. Scanner exit 3 is a failure.
The Go and Sonar percentages have different denominators; no unsupported
attribution explains their difference. No gate or target has been weakened.
`checkpoint56/77435c34-hosted-evidence.json`, SHA-256
`de73f1436c576b9fd8f11079d9bd3dda5f0be118f226e37aa5e6f37cfa426a32`,
retains the exact bounded observation. No workflow was rerun or cancelled.

## Signed-context and operator source boundary

The source adds a pure `internal/securitycontext` verifier with closed v1
envelope/claims schemas. It uses maintained JOSE verification, protected-only
headers, explicit algorithm/key/purpose, distinct actual public-key material,
threshold signatures, exact bindings and graph digest, canonical predecessor
generations and freshness checks. The opaque result is explicitly unadmitted:
caller-supplied trust/input does not establish protected enrollment or authority.
Its public summary omits graph, payload, provenance and fence digests.
Declared capabilities are requirements, not measured runtime satisfaction.

Two independent behavior reviews and a separate comment/schema-description
review bind the initial six-file snapshot; a separate seventh file adds
adversarial test source. The b30f1308 run above exposes the expiry fixture and
deprecated ECDSA access failures. The following independently reviewed source
correction gives expiry its own valid one-signer fixture, validates maintained
JOSE key structure before maintained `PublicKey.Bytes`, and adds nil-coordinate
and non-nil off-curve vectors. It retains the alias-threshold assertions and
uses no analyzer suppression. Pinned `gofmt` passes; these corrected bytes have
not been compiled, executed or analyzed locally. Receipt
`checkpoint57/securitycontext-ci-correction-integration.json` binds the two
corrected paths and final independent reviews. The
protected compiler, store, approval/bootstrap/recovery and unified enforcement
remain required; no CLI flag, profile field or strict-mode pass follows.

The same source candidate extends the existing [release/operator guide](../RELEASE.md#product-go-live-and-commercialization-service-provisioning)
with reviewed project-specific commercialization/service provisioning and
official setup routes. Its consent/privacy/CSP/redaction, English/Arabic,
post-deployment and withdrawal procedures are operator requirements. Source
review establishes no vendor account, entitlement, service activation, legal
compliance, deployment or commercial readiness.

Help source review binds 333 English entries and accepts exactly forty
punctuation corrections. It does not establish every field's conditional
semantics, actual UI/action coverage, translation quality or search acceptance.
The following integrated English-only snapshot contains 715 web files and
producer-scoped evidence of 52 current tests. It pins the exact distribution
manifest digest and byte length in the trusted release module before decoding
or fetching children; per-article hashes alone are insufficient. Local fulltext
search, progressive article loading and bounded caching use query-free
same-origin requests. F1 control context admits only static schema identifiers,
normalizes declared map/list paths and restores focus on Escape. Root has not
run Node or a browser; actual hosted, accessibility, relevance, device and
performance acceptance remain separate. This pin does not defend a compromised
trusted application bundle or host.

The English authoring JSON is compacted to 957,833 bytes with parsed content
equal to the corrected readable corpus. Its actual byte digest changes;
authored content version and semantics do not. The existing 1 MiB general file,
32 MiB aggregate and 10,000-file source limits remain unchanged, with only the
previously declared selected contact-sheet PNG exception. Receipt
`checkpoint57/frontend-help-source-integration.json` binds every integrated
file and the independent source reviews.

The separate Arabic corpus now has 333 drafts and zero reviewed entries.
Independent correction review accepts the prior field-language fixes only;
it is not language, UI, RTL or production acceptance. An initial closed v1
source-shard proposal remains private pending immutable bundle-root binding,
`contentStage` parity, consumer and fallback evidence. Presentation preferences
remain in the existing frontend lane. All current product contracts use one
direct initial v1 semantics; no legacy version converter or dual-version reader
is introduced. Historical incompatible proposals retain their original identity
and are not consumed or relabelled.

## Exact eac85a1 source and evidence boundary

Published source `eac85a1e642adc164c79fe85f1b8e4f0831d271f`, tree
`46cad58b3500f49ddd297819e2335babb3d4b538`, has exact public Git readback for
21 selected files and all 453 working files, clean at publication. Source-only
independent reviews cover early CLI/MCP command-identity preflight, finite
128 MiB executable hashing, configuration/Transit/assurance readability,
TypeScript response-boundary test source and Sonar browser-test classification.
The bounded source fingerprint admits only the exact declared contact-sheet
PNG up to 2 MiB, validates its PNG signature and compares the same bounded
working descriptor bytes with immutable Git blob bytes. The general 1 MiB file
limit, aggregate limits and full source identity remain enforced. A rejected
second pathname-hash proposal and its replacement are retained separately.

Static review covers 253 Go files using the pinned formatter, 67 Markdown files,
666 local links, 27 JSON documents, six JSON examples and three embedded Python
AST snippets. These checks are not compilation, executed tests, coverage or a
security audit. No local application/build/analyzer runtime was run for this
publication. Root browser source is unchanged from e103; the separate frontend
correction candidate has not been integrated or qualified by this checkpoint.

Automatic hosted runs are
[Source](https://github.com/maldous/CircularQuay/actions/runs/41924790410),
[Security](https://github.com/maldous/CircularQuay/actions/runs/41924790453),
[SDK](https://github.com/maldous/CircularQuay/actions/runs/41924790633),
[Sonar](https://github.com/maldous/CircularQuay/actions/runs/41924790471),
[Browser](https://github.com/maldous/CircularQuay/actions/runs/41924790469),
[WireMock](https://github.com/maldous/CircularQuay/actions/runs/41924790431) and
[Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41924790820).
The first bounded readback reports the same eight gosec findings and nine
failed browser jobs, with scoped WireMock and Dependency Review success. The
browser failure cause was not read and is not inferred from historical e103
failures. That original `checkpoint55/eac85a1-hosted-one-shot.json` is preserved.

The later `checkpoint55/eac85a1-hosted-followup.json` binds tested merge
`fc745e2cde08af922fad9cd76cea5ef8f537469f` to the exact candidate tree.
Source assurance passes identity before/after checks, repository policy,
formatting, unit tests and vet. It fails **8,496/10,218 statements
(83.147386964%)** against 100%; required tagged integration remains `not_run`.
The Linux complexity job reports two remaining violations: the matching-override
test at `internal/cli/command_preflight_test.go:76` is 12 and
`prepareWorkloadCommand` at `internal/cli/commands.go:186` is 11. The Android
cross-build stops on a protobuf v1.36.12 proxy ZIP HTTP/2 `INTERNAL_ERROR` before
first-party compilation; that failure is not evidence of a source compile bug
or a successful target build. The connector returns only the first job page.

Sonar completes CE and exact current-analysis/revision fencing, but the gate is
**ERROR**, with **64.1%** coverage, three issues, zero hotspots and zero
duplication groups/density. The issues are two duplicated error-operation strings
in `internal/platform/process.go` and the unhandled example-load promise in
`web/local/configuration.js`. Required new-coverage and new-duplication metrics
are absent; the Python 3.15-to-3.14 analyzer warning remains, and scanner exit 3
is a failed gate. The separately reported Go statement percentage and Sonar
percentage have different denominators; no unsupported attribution explains
the difference. No JavaScript execution-coverage pass follows from correct
browser-test classification.

The SDK first page returns 30 jobs, four successes and 26 failures. Its bounded
Python 3.11 artifact reports client 86%, local 79% and types 92%; these do not
satisfy the 100% gates. The first artifact page exposes no TypeScript coverage
artifact, so exact missing TypeScript lines/branches remain unknown in this
receipt. Dependency-audit, full-matrix, installed-artifact, native/provider and
independent qualification remain open. No workflow was manually rerun or
cancelled for either observation.

The product-wide default/strict enforcement remains a release blocker.
Independent private design review rejects self-referential policy/generation
hashes before public implementation; revised contracts require protected
enrollment, unforgeable local proof values, maintained-library signatures,
durable fencing and honest uncertain-effect recovery. A contract or profile
name does not establish implementation, downgrade resistance or compliance.

## Following reviewed frontend and native corrections

The next source candidate integrates the independently reviewed thirty-file
frontend snapshot through one full e103-based patch. Eighteen web paths change;
429 non-web working files retain their pre-integration hashes. Original frozen
packets, blocked reviews and the settled mobile screenshot remain preserved.
The actual example-load error handler is awaited and generic; invalid numeric
edits invalidate removal Undo without discarding the later draft, while earlier
invalid input still returns with Undo. Asset reads compare retained bigint
file identity with the opened descriptor before reading and revalidate paths
before/after bounded reads. Bare and populated query delimiters are rejected.
The source review does not claim atomic operating-system path containment.

The isolated frontend producer reports syntax success and **37/37 current
tests across six files**, plus actual synthetic browser checks of both Undo
schedules and canonical cleanup. Its separate 70-test discovery includes 33
historical snapshot tests; that is not seventy distinct current tests. Live
checks cover ten asset byte/MIME/security-header results, four prior boundary
denials and eight raw-query denials on the same owned loopback ports. Node
24.16.0 producer evidence is not the hosted runtime/platform matrix. Root
copied and hash-verified the immutable packet and reviewed its static source
and screenshot; no root Node/browser execution occurred.

The candidate also retains all command-preflight/target checks while separating
cohesive request validation from engine construction, preserves both matching
pinned and unpinned override test assertions, and names duplicated executable
fault-operation strings without changing values or error categories. The
Windows Python negative fixture explicitly denies an owned temporary directory
with an Everyone-write DACL instead of assuming every generic temporary
directory is unsafe. Native protected-root enforcement is unchanged; the
positive APPDATA init/doctor/plan fixture remains. These source reviews and
static formatting do not establish fresh complexity/Sonar/Windows passes.

`checkpoint55/frontend-correction-integration.json`, both corrective frontend
independent reviews, `native-readability-independent-review.json` and the
Windows fixture review retain exact hashes and limits. Required published
source/security/Sonar/coverage, full SDK/native/provider/artifact and independent
qualification remain open. Searchable help has an accepted sixteen-family
source inventory; it does not yet have a complete manual, per-control rendered
state mapping, versioned index implementation or search/accessibility pass.

## Exact e103a52 integration and security boundary

Published source `e103a52b8c0884d45f3ca853008f915cd52dcbe9`, tree
`56d13b056c71659f8223240c57247f7e502f6784`, integrates the synthetic
client/account/operator interfaces, reviewed browser boundaries, native pinned
command correction and SDK test-oracle repairs. Public Git readback verifies
41 selected files and all 450 working files, with a clean checkout at publication.
The original frontend checkout, commits and historical evidence remain intact.

[Browser run 41913450832](https://github.com/maldous/CircularQuay/actions/runs/41913450832)
discovers all five test files. The observed Linux, Windows and macOS jobs pass
syntax and 25 of 28 tests. Three guided regression fixtures hit schema
min-properties/identifier errors before their intended reference errors. The
runtime rejects those edits; the intended reference oracles still need corrected
fixtures and fresh evidence. These failures are not silently accepted or removed.

[Sonar run 41913450567](https://github.com/maldous/CircularQuay/actions/runs/41913450567)
completes CE/current-analysis fencing for merge `ff00a323`, whose tree matches
the candidate. Coverage is **62.7%**, with seven first-party issues, zero
hotspots, zero duplication groups/density and zero CE warnings. The scanner
still warns about Python 3.15 and exits 3; required new-coverage and
new-duplication metrics are absent. Only Go coverage is imported. The web
regression files were classified as production sources because the published
test patterns omitted them; no browser/JavaScript coverage is established.
`checkpoint53/sonar-e103-readback.json` retains the exact artifact and findings.
Correct test classification does not waive coverage for any production source.

The later bounded readback of
[Source run 41913450484](https://github.com/maldous/CircularQuay/actions/runs/41913450484)
finds the source-assurance job stopped at `source_identity_before_checks` with
`source_identity_invalid`; its 0/0 output is not a coverage measurement. The
original 1,499,917-byte PNG exceeds the source fingerprint's 1 MiB file budget.
The following correction requires a separate finite asset/capture allowance
with full immutable-blob and checked-out-byte binding. Independent review
rejects a second pathname hash as insufficient: it does not bind that read to
the recorded SHA-256. No source exclusion, digest bypass or global tool-output
increase establishes a valid correction.

[SDK run 41913450576](https://github.com/maldous/CircularQuay/actions/runs/41913450576)
has 30 jobs on the connector's observed first page, with four successes and 26
failures; that page is not the whole matrix. One retained Python coverage
artifact reports client 86%, local 79% and types 92%, below the unchanged
100% requirements. A Windows Python 3.11 job accepts a negative generic
temporary-root case, and the Python audit reports six vulnerabilities in
cryptography 48.0.1 locks. The fixed-version Intel build closure remains
unqualified. Source complexity and all eight gosec findings also remain open.
`checkpoint54/e103-provider-evidence-reconciliation.json` binds the exact
candidate/merge fence, job IDs, artifact and observed limitations. These are
historical e103 results, not results for the later source corrections.

The subsequent source correction introduces shared command-identity preflight
before CLI workload state, repository-backed secret materialization and value
resolution, and repeats validation before child environment/process creation.
MCP requires its enrolled pin and uses the same protected, bounded validator.
Hashing consumes at most the 128 MiB limit plus one overrun-detection byte,
then rechecks file/path identity, size, regular type and native protections.
Independent review rejected the initial stat-only bound; its immutable rejection
and the finite-read correction remain separate evidence. Configuration/policy
acquisition may precede this boundary, and pathname execution is not atomic.
Source tests and readability corrections require exact-candidate hosted evidence;
none establishes the unified strict-profile engine or closes a gosec finding.

The complete v1 security and coverage contract remains unchanged. Correcting
fixtures, source/test classification or complexity cannot substitute for full
native/provider/SDK/browser coverage, signed-context enforcement, durable
administration/audit and independent whole-product qualification.

## Exact 364fba9 hosted observation

Source/design commit `364fba9355905497634daebfd48797c5ff987e00`, tree
`9d29db23700316180452bcdf8372e692714ded6e`, has fresh hosted evidence.
[Security run 41904519089](https://github.com/maldous/CircularQuay/actions/runs/41904519089)
builds and analyzes successfully in its CodeQL job; this step result is not a
readback of zero security alerts. Staticcheck and govulncheck succeed. Gosec
fails with six G204 process-launch findings and two G103 unsafe-call findings;
their trust/taint and native-primitive boundaries require explicit review.
No exclusion, suppression or scanner-evasion rewrite closes those findings.

Bounded source triage identifies a local workload override path that skips a
configured executable digest. The following platform correction requires the
exact declared argument vector for pinned workloads and verifies its digest
even for an identical override. Unpinned explicitly local command selection
and MCP's enrolled-command requirement remain separate. New regression source
covers rejection before child-environment/process creation; it has not run.
This late platform check does not establish early CLI/provider preflight or
atomic native executable custody. All eight gosec findings and the broader
qualification requirements remain open until their actual evidence permits
closure. `checkpoint52/gosec-364-triage.json` retains the original assessment.

[Sonar run 41904519375](https://github.com/maldous/CircularQuay/actions/runs/41904519375)
completes CE and current-analysis fencing for synthetic merge `b322fa7`, with
the same tested tree as the candidate. Retained receipt
`checkpoint52/sonar-364-root-cause.json` binds the analysis and artifact.
Reported first-party issues, hotspots, CE warnings and duplication groups are
zero: the eight preceding Sonar findings are cleared for this analysis.
The gate remains failed at **67.6% Sonar coverage** against 100%. The collapsed
Go profile separately covers 8,392 of 10,108 statements (83.0233%); these are
different recorded denominators. Only Go coverage is imported by the selected
properties, and no unsupported attribution explains the difference.

Required new-coverage and new-duplication metrics remain unavailable. The
scanner still warns that Python 3.15 is analyzed as 3.14. Scanner exit 3 is a
failed gate, not a failed CE upload or proof that the warning is harmless.
The receipt identifies uncovered Go paths for substantive coverage work.
No analyzer target, warning requirement or owned-code denominator is reduced.

WireMock run 41904519066 and Dependency Review run 41904519043 succeed in
their own scopes. Source run 41904519046 and SDK run 41904519129 are still
in progress at this bounded capture. These results do not qualify later web
integration or source changes. No workflow was manually rerun or cancelled.

## Exact 74e17dd hosted observation

Development commit `74e17dd62ac83d6504d6589e9d7a46b26ee77252`, tree
`b47d902663d55c400498d0eebe6fbc68096613be`, has retained workflow readback in
`checkpoint50-security-design/74e17dd-current-workflow-observation-20261105.json`.

| Pull-request workflow | Recorded outcome |
| --- | --- |
| [Source checks 41833116936](https://github.com/maldous/CircularQuay/actions/runs/41833116936) | Failure |
| [Security analysis 41833116876](https://github.com/maldous/CircularQuay/actions/runs/41833116876) | Failure |
| [SDK qualification 41833116959](https://github.com/maldous/CircularQuay/actions/runs/41833116959) | Failure |
| [SonarQube 41833116707](https://github.com/maldous/CircularQuay/actions/runs/41833116707) | Failure |
| [Synthetic WireMock 41833116760](https://github.com/maldous/CircularQuay/actions/runs/41833116760) | Success in its synthetic scope |
| [Dependency Review 41833116621](https://github.com/maldous/CircularQuay/actions/runs/41833116621) | Success in its dependency scope |

Bounded logs and the redacted Sonar artifact identify three build errors:
two unused authorization results in the Transit adapter and a test converting
`UnwrapResult` instead of its plaintext field. Staticcheck and CodeQL stopped
at compilation; that is not a clean security scan. The following source
corrections retain authorization evaluation and the plaintext/clearing oracle.
Their independent source review and formatting do not establish compilation.

Sonar CE succeeded for synthetic merge
`cb0d5ec5bb41ae3df15212191bb98f74e55800ad`, whose tested tree matches the
candidate. Task `7566e5fa-2ee8-4176-ba89-c17559d03415` and analysis
`168ddde4-65e0-49f5-bcd6-5ba563f6ffaa` pass the revision fence. The quality
gate fails: coverage is 0.0, violations are eight, duplication is 0.0 and
reported hotspots are zero. The coverage file contains only its mode header
because the Go build failed; it is not a meaningful uncovered-line inventory.
Required new-coverage and new-duplication metrics are unavailable and remain
failures. Empty duplication groups do not imply a nonzero-duplication defect.
CE reports no warnings, while the scanner reports one Python 3.15 support
warning and analyzes it as 3.14. A completed upload does not satisfy the gate.

`checkpoint52/sonar-root-cause.json` binds the exact artifact digest, metrics,
warning and eight first-party findings. The following source refactors address
the listed complexity and duplicate-literal findings in assurance, CLI,
configuration and Transit code/tests, with independent source review. Actual
zero findings, complexity, warning-free analysis and full coverage still need
an exact-candidate hosted result; no threshold or runtime target is removed.

SDK qualification has separate failures. Node 24/Linux records 72.14% line and
statement coverage, 69.34% function coverage and 82.55% branch coverage against
the unchanged 100% requirements. The following lifecycle test covers native
child handling through the public client using stable built-in export
synchronization; it has not run locally. Windows installed/minimal Python
consumer fixtures encounter the native directory-trust denial. The fixture
correction selects an existing application-data candidate and retains the
generic temporary-root negative test; it does not relax native trust or ACLs.
Only hosted Windows execution can qualify that candidate.

The Python static lane also reports six advisories in pinned cryptography
48.0.1. Inspected fixed upstream releases lack macOS Intel binary wheels.
`checkpoint52/python-wheel-security-assessment.json` records primary metadata
and the closure gap. No vulnerable lock, invented hash, suppressed advisory,
silent platform removal or unqualified source build satisfies this gate.
The three native Go SDK jobs pass their own scope, not the failed overall SDK
or installed CLI qualification. Detailed source/security and SDK receipts are
`checkpoint52/source-security-root-cause.json` and
`checkpoint52/sdk-root-cause.json`.

No local heavy execution, manual workflow rerun or cancellation occurred for
these corrections. Native reboot, live-provider, tagged-lane provenance/coverage
joining, artifact installation and independent audit remain required.

## Exact c18be5a hosted observation

Published runtime/design commit
`c18be5a76bc82ed7479e8ddbb6b4d28828890fa4`, tree
`e85408243741ccbf0483ca28e75fa87cf04bd973`, has completed workflow metadata
readback at `2026-11-04T12:47:34Z`. The retained owner-held record is
`checkpoint50-security-design/c18-hosted-frontend-handoff-metadata.json`.

| Pull-request workflow | Recorded outcome |
| --- | --- |
| [Source checks 41825521377](https://github.com/maldous/CircularQuay/actions/runs/41825521377) | Failure |
| [Security analysis 41825521340](https://github.com/maldous/CircularQuay/actions/runs/41825521340) | Failure |
| [SonarQube 41825521327](https://github.com/maldous/CircularQuay/actions/runs/41825521327) | Failure |
| [SDK qualification 41825521301](https://github.com/maldous/CircularQuay/actions/runs/41825521301) | Failure |
| [Synthetic WireMock 41825521324](https://github.com/maldous/CircularQuay/actions/runs/41825521324) | Success in its synthetic scope |
| [Dependency Review 41825521332](https://github.com/maldous/CircularQuay/actions/runs/41825521332) | Success in its dependency scope |

Push-triggered source/security also fail; push WireMock succeeds. This bounded
observation reads metadata only and does not determine individual failure
causes, full test coverage or scanner findings. The preceding independent
source/design reviews, 249-file gofmt result and 650-link documentation check
do not override these failures. No workflow was manually rerun or cancelled.
The release remains unqualified. These runtime results do not execute the
separate local frontend candidate below.

A separate one-job Sonar log readback identifies failure at **Produce fresh
isolated Sonar evidence**. It records a coverage-test failure, below-100%
coverage and scanner-warning markers, plus strict quality-gate, duplication
and issue-readback failure markers. Redacted artifact upload succeeds. Receipt
`checkpoint50-security-design/c18-sonar-bounded-diagnostic.json` binds that
observation; no artifact contents, CE task, numeric metrics or issue details
were read. These markers cannot establish a successful scan, actual issue
counts or a passing gate.

## Isolated frontend candidate and evidence boundary

The source integration uses the final frozen sequence `0ecbc69` → `6057e39`
→ `45f51a21e1a311c355a7e53dae64573a17b23246`, whose final tree is
`d21ef707dea609d860db81f9a5da1d482e0ea25b`. Independent Git-object/worktree
readback binds 21 tracked files and seven retained synthetic artifacts to the
producer's `verification-actions.json`, SHA-256
`0456287864fe6610db59c2fb3d5e2c67eb6d93b06d068d615c9f2ea24318a07f`.
The schema/example bytes match the canonical native files; the two retained
Linux JSON downloads are semantically equal. No user configuration, credentials
or personal export was copied or uploaded. The earlier private Library ZIP
continues to represent its original historical checkpoint.

The producer reports 12 focused tests and individual changes to 131 editable
scalar controls, with six true constants, plus structural edits and selected
browser journeys. The scalar exercise uses a deliberately invalid all-fields
draft: it proves editing fidelity, not validity of every schema/provider branch
or native execution. The canonical Linux download/re-import round trip is a
separate valid synthetic case. Independent source/hash review does not rerun
these tests or the browser.

Integration review identified additional preview defects: symbolic static
assets could escape the serving root; guided renames could strand references;
unknown imported keys could inject markup into the sidebar; and inherited
JavaScript properties could masquerade as map entries. CSP limits the markup
case; no inline-script execution or global prototype pollution is established.
The integration corrects these boundaries and adds regression source, with
separate independent reviews. Malformed request targets, static asset sizes
and live-reload admission are bounded. These source corrections have **not**
run under the local execution hold. The original producer's 12-test/browser
receipt cannot qualify the changed integration source.

The new browser workflow runs dependency-free syntax and focused Node tests
on pinned Node 22/24/26 across Linux, macOS and Windows. Its definition and
source review are not nine passing jobs, full browser/accessibility coverage,
native validation or a release. No web source exclusion, coverage threshold
reduction or runtime target removal accompanies integration. The following
historical observations retain their original scope.

Local commit `0ecbc694f8454663fd9416d57a624f66314d75ee`, tree
`12c07b465845de2ad8f8cc17e24b0a4dce0d5820`, is on
`feature/browser-dashboards` in
`/home/user/src/2026-11-04/circularquay-browser-dev`. Its parent is
`10dde6d3b94fce98e3415e4ca8d8c0dcb649f0d0`; it adds 19 files exclusively
under `web/`. It is a separate local candidate, without a public commit or
verified remote preview URL. Its bytes are not integrated into this runtime
branch.

The candidate contains a synthetic masked environment workbench, guided
inspector, local `.env` import, typed configuration editor and separate
account/operator presentation surfaces. Provider, key, grant, identity,
billing, fleet and apply/run actions remain simulations. Browser persistence
is a demo store for synthetic values, not protected native secret storage.

The producer reports seven passing focused Node tests, browser journeys,
187 rendered schema-path entries, a valid Linux no-op round trip, seven exact
representative edits, re-import equality, unknown-value preservation, a
390-pixel viewport without page overflow and a clean browser console. The
coverage matrix distinguishes rendered controls from individually mutated
fields; it does not claim every field/provider combination was exercised.

Owner-held evidence in that checkout is `web/coverage.md` (SHA-256
`2e5dd8ed2315bd756ca9e895781de59d0a0c5714ba1b61c043f21d16b30ca612`)
and `web/output/playwright/verification.json` (SHA-256
`3095443acbaffad8def020d1a135c51c7d22abd9356546d9cbccb0aac699b714`).
Independent readback receipt
`checkpoint50-security-design/frontend-handoff-independent-readback.json`
(SHA-256 `e682ee5332d24186fed4c817a6c8ee9bcf77590cd957391c7cfd7e7311847454`)
verifies all 19 committed/worktree source hashes, the presence of all 16
listed nonempty artifacts, 187 inventory entries and the retained JSON
semantic equalities. It does not rerun Node, browser or native checks, or
independently establish that every control executed in a browser.

Bundled policy schema SHA-256
`1d34bfc0d3c93e6078390fe4318e957e49178aac580cf8aaf11a70b25d50dffd`
and Linux fixture SHA-256
`1d3b73c654f5371729e6005434c5cd15ca1c87d65cfecf1603ba6991c6429276`
match their source-owner snapshots. Recheck them before integration.
The producer reports one Node process on loopback ports 5180 (account/operator)
and 5181 (local client); development directions are in that checkout's
`web/README.md`. At this historical checkpoint, an external preview route and
the selected icon 13 asset remained unresolved. Native protected files,
signed policy/include resolution,
actual authentication/authorization, custody, providers, lifecycle, durable
audit, backend tenant isolation, distribution and production qualification
remain separate requirements. No UI control or reported browser success grants
real backend authority. Integration/publication requires coordinated review
against the current runtime branch and the unchanged release gates.

### Frontend continuation and phone access

Local continuation `6057e39dfa310f9c8ec7e8877ad1105256b45031`, tree
`8dc47a8e82ea704d28534ab8aac8cac557ed6d57`, contains a saved-proposal discard
persistence fix, updated evidence and the original selected icon contact sheet
with provenance. It does not contain an isolated finished vector/logo.
The producer's continuation receipt binds 21 tracked files, seven focused
tests and synthetic discard/navigation/reload, invalid-draft, selected-device
and interrupted-migration journeys. Source/evidence hash readback is distinct
from independently rerunning those journeys. Further frontend corrections
remain with that owner until their own frozen checkpoint and review.

The user subsequently confirmed phone access through existing authenticated
Termux SSH forwarding to loopback 5180/5181. The earlier receipt's false
phone-open field is historical and is preserved. This is human confirmation,
not an automated physical-device test, public tunnel or shared browser store.
The original contact-sheet SHA-256 is
`b50b0e0db86d4d39f4295a5f06c2ba5651c314ad088410e88aa9991ea05cbba6`;
icon 13 is the selected reference. The private synthetic evidence ZIP remains
private and unchanged; no saved configuration, credentials or user export is
included. These observations do not qualify a real managed backend or strict
profile. Current owner/issue routing is in the
[canonical closure checklist](session-handoff.md#uif-migration-and-delivery-closure).

## Exact 10dde6d hosted observation

Public commit `10dde6d3b94fce98e3415e4ca8d8c0dcb649f0d0`, tree
`9174f780b61636b6f6e9820264bebe0b446910c6`, has bounded public workflow
readback in `checkpoint46-10dde6d-hosted-workflow-readback.json` and a separate
safe diagnostic amendment in `checkpoint46-10dde6d-bounded-log-amendment.json`.
This observation is not a complete artifact or security qualification.

[Source 41815497020](https://github.com/maldous/CircularQuay/actions/runs/41815497020)
has 30 failed visible jobs. The bounded Linux and
[CodeQL 41815497012](https://github.com/maldous/CircularQuay/actions/runs/41815497012)
logs identify `internal/cli/cli.go:292:15: undefined: platform`; the following
source correction restores the required import. Linux also reports trust-store,
protected-path and composition fixture/validation mismatches. The static
security job fails before govulncheck/gosec, so no clean scan or finding
disposition follows. [Sonar 41815497048](https://github.com/maldous/CircularQuay/actions/runs/41815497048)
uploads but fails fresh evidence. Neither a successful upload nor the preceding
candidate's zero findings establishes a passed gate for this tree.

At capture [SDK 41815497053](https://github.com/maldous/CircularQuay/actions/runs/41815497053)
is queued with no test evidence. [WireMock 41815497163](https://github.com/maldous/CircularQuay/actions/runs/41815497163)
passes its three synthetic jobs and
[Dependency Review 41815497172](https://github.com/maldous/CircularQuay/actions/runs/41815497172)
passes. No workflow was manually retried or cancelled for these observations.
The prior d9ff832 source/SDK runs are cancelled, security/Sonar fail and
WireMock/Dependency Review pass at their own recorded scope. These states
remain distinct from the current unexecuted source amendments.

## Following custody and source-assurance changes

The following development source adds `internal/keyservice`: a profile-bound
enrollment/custody core with explicit trusted authorization, association storage,
provisioning and native-provider collaborators. It separates KMS wrap/unwrap
from deliberate hosted age-private-key retrieval, requires both associations
before activation, and treats partial/uncertain provisioning or storage effects
as reconciliation. Current-association authorization is rechecked after remote
waits before returning key material. Safe errors cannot retain collaborator
text. The native Transit adapter uses the established OpenBao/Vault API with
explicit protected connection credentials and provider-key checks; it is not
an AWS/Azure/Google KMS SDK or a new cryptographic construction. The software
adapter uses the already pinned Go JOSE library with an explicit
`A256KW`/`A256GCM` flattened JWE profile, injected private KEK resolver, protected
key/version headers and authenticated opaque artifact context. It performs
software cryptography, with no non-exportability or HSM claim. It does not
implement the hosting platform, production KEK storage or cross-language JWE
interoperability; its authored crypto tests have not run locally.

The shared HTTP classifier now limits availability to positively typed native
connection failures or a bounded request timeout, with caller cancellation and
TLS/trust failures taking precedence. Unknown errors, malformed/truncated body
reads and unexpected protocol responses deny. Native Unix/Windows errno sets
are explicit; unsupported native classifications remain conservative. Transit
uses only that finite safe connection/timeout classification and strictly
decodes bounded provider responses, including duplicate-field rejection.

This package is source work, not a running hosted service. It has no production
authorization/store/provisioning implementation, HTTP server, hosted-age
custodian, metering/billing engine or policy/CLI/SDK/MCP integration. Its
synthetic tests are not yet executed under the local resource hold. Native
provider identity/custody, actual round trips, failure classification and
end-to-end profile enforcement remain unqualified. No strict runtime profile,
automatic first-run enrollment or migration is enabled by adding this package.

The source corrections also restore the CLI's missing platform import and fix
omitted optional protected references being mistaken for an empty policy path.
Nonempty trust/credential/input ownership restrictions remain enforced. JSON
composition masters can omit root sources/workloads at the intermediate decode
boundary; ordinary JSON still requires them, and unresolved composition cannot
be executed through `Parse`. Complete resolved validation remains mandatory.
The private-snapshot test follows the actual first rejecting privacy check.

Source assurance now retains manifest-bound integration-tagged test files for
identity/formatting while separating their non-buildable directory from
untagged Go commands. The report adds required `tagged_integration` status
`not_run` with code `requires_qualified_tagged_integration_lane`, so this change
does **not** make the repository audit pass. Required fixture-lane evidence,
coverage/provenance joining and full source/security/Sonar qualification remain
outstanding. The real production executable denominator is not waived.

Independent static review and pinned formatting are narrower than compilation,
test/race/fuzz execution, native protection or a real provider trial. New hosted
results must bind the exact published source/tree and artifacts. The whole
product default/strict control register and highest-applicable ASVS/SSDF design
remain requirements, not a certification or completed qualification.

## Exact 2757155 hosted receipt

Publication `275715525b74c99200ee9afc83ff069f6129b4a2`, tree
`2e7bb1cf20fda50c07107a5dccb81154d57433ad`, was tested as synthetic
merge `8ebd570715148137fa61e6e38443f83b82c9e786` with the same tree.
These observations do not qualify the following composition or design changes.

[Sonar 41803634899](https://github.com/maldous/CircularQuay/actions/runs/41803634899)
completed CE task `5e9b0bdd-6169-455f-8651-7d556be0e44d` and analysis
`692633b5-b7da-4035-821a-d72465b266f9`, with zero CE warnings and all
retained strict readbacks HTTP 200. Issues, violations, hotspots, duplication
groups and overall duplicated-line density are zero. The gate remains ERROR:
66.5% coverage, missing new-code coverage/duplication metrics, and a Python
3.15→3.14 analyzer warning. Scanner exit 3 follows successful upload/CE and
does not imply a failed upload. Go statement coverage is 7,295/8,738
(83.485923552%); it has a different scope from Sonar line coverage.

[Source 41803634763](https://github.com/maldous/CircularQuay/actions/runs/41803634763)
passed its Linux synthetic complexity-ceiling step after the OTel extraction.
The separate source-assurance gate still fails the tagged integration package's
unit scope, coverage and vet. Bounded workflow readback also records the
[security toolchain failure](https://github.com/maldous/CircularQuay/actions/runs/41803634592),
[SDK failures](https://github.com/maldous/CircularQuay/actions/runs/41803634904),
three passing [WireMock jobs](https://github.com/maldous/CircularQuay/actions/runs/41803634597)
and passing [Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41803634887).
Those aggregate observations do not establish new finding dispositions, lock
qualification or native/provider acceptance. Retain
`checkpoint45-2757155-hosted-sonar.json` and its exact source/tree fence.

## Following composition and security-design changes

Source commit `5b6770594bd50a8b7459b9e597215c0f0ebb22fe`, tree
`0e5c4de6f12cbcc7bcb8593996c78381027b029e`, contains this composition
implementation. It has source/static review only; its subsequent hosted
qualification must bind these exact bytes and the selected documentation tree.

The new client-side resolver preserves single-document parsing and adds
explicit root-owned JSON includes, immutable pins, namespaces/dependencies,
bounded graph loading, conflict rejection, overrides and equality locks.
It validates a complete candidate after root edits, then exposes one policy
snapshot. CLI operational dispatch retains that snapshot. Raw `config get/list`
inspect only the protected root; validation and writes resolve the complete
candidate. No configuration or source credential is sent to a hosted key
service or server proxy.

Repository includes use the existing native signature/ancestry/object-graph
protections and the new `FetchPinned` pre-acceptance exact-commit check. Offline
cache use is explicit and time-bounded; online failures have no implicit cached
fallback. Admission rechecks cache deadlines and cancellation after graph
validation. That check is not a lease that automatically terminates an already
running consumer or recalls an age key/plaintext copy; complete pre-effect and
existing-generation lifetime qualification remains required. Repository cache acceptance is not a distributed transaction with
all other includes or runtime activation. The resolver's private-domain guards
cover declared path/reference aliases, publish/remote-grant restrictions and
mixed snapshots; they do not prove that two copied key files contain different
key material or protect plaintext from an authorized hostile native consumer.

New schema, resolver, repository-pin, CLI and privacy tests are source-reviewed
fixtures only until executed. Pinned formatting and bounded static checks are
narrower than compilation, unit/race/fuzz execution, native file protection or
Sonar/security qualification. No builds, resolvers, containers or runtime
tests ran locally under the shared Foo resource hold. The full canonical
organisation/group/lock and signed strict-profile engines remain unimplemented;
the composition master alone does not establish centrally imposed enterprise
authority or downgrade resistance.

## Exact fa767446 hosted receipt

Publication `fa767446175f76ae442ff81baff4b5d7ac2d73bf`, tree
`77843bdaec5eab412d1ed6a490b854af563c1b2b`, has verified public blobs for
all 62 changed files and preserves 392 working files. Its PR analysis uses
synthetic merge `cfdbad1966e5a0d3ecca56f946ec8bcc30aee3f2` with the same tree.
These are exact-candidate outcomes; the following documentation and OTel
extraction require their own hosted evidence.

[Source 41800110446](https://github.com/maldous/CircularQuay/actions/runs/41800110446)
passes Linux tests/race/vet and bounded fuzzing before one remaining cyclomatic
finding: `newOTelBackend` has complexity 12. The cognitive analyzer is not
reached. All previous assurance-parser complexity findings are absent.
The independent source auditor still fails unit tests attributed to the
tag-only integration package, vet with unknown attribution, and coverage at
7,295/8,735 statements (83.514596451%). Its source identity matches before and
after. A source-only extraction preserves signal ordering, error identity and
partial setup cleanup; no post-extraction analyzer/runtime pass is claimed.

[Security 41800110516](https://github.com/maldous/CircularQuay/actions/runs/41800110516)
passes Staticcheck, govulncheck and CodeQL build/analysis/upload. Gosec retains
six G204 and two G103 findings, unsuppressed. Revision-bound authoritative
CodeQL alert counts and current GitGuardian incident disposition remain absent.
Source scanner findings are separate from the planned runtime exposure gate.

[Sonar 41800110421](https://github.com/maldous/CircularQuay/actions/runs/41800110421)
completes CE task `35a7d8ac-dd45-4033-9f89-e94feddb0928` and analysis
`24d5e45d-f969-42a8-9294-cf3e15182736`; all retained readbacks are HTTP 200.
All four prior assurance findings are closed: issues, violations, hotspots and
duplication groups are zero; overall duplication is 0.0%. The strict gate still
fails at 66.6% coverage, absent required new-code metrics and one Python 3.15
fallback warning. CE warnings are zero. Only Go coverage is imported; Python
XML/TypeScript LCOV production/ingestion remain required.

[SDK 41800110386](https://github.com/maldous/CircularQuay/actions/runs/41800110386)
is explicitly partial at the retained observation: 30 of 31 observed jobs are
terminal and the macOS Node 26 lane remains active. Go SDK tests pass on all
three native platforms, and Go lock reproducibility passes. Completed Ubuntu/
Windows TypeScript lanes pass lint/type/audit, package membership and the
fresh installed ESM consumer. The selected Ubuntu Node 24 run has 18/18 tests
and both official TypeScript/Go MCP scripts passing; coverage still fails at
72.14% lines/statements, 69.34% functions and 82.55% branches.

Windows Python 3.12–3.15 installed/minimal consumers retain their failure.
The new native diagnostic succeeds and emits only finite synthetic records:
ordinary creation fails `parent_identity`, prepared creation fails
`directory_identity`, both `trust_failure`. These identify protected-directory
checks, not a proven cause or remedy; permissions were not relaxed and consumers
were not retried. Python coverage and Intel macOS wheel/advisory gates remain
unmet. Do not infer a full SDK or installed-client qualification pass.

[WireMock 41800110400](https://github.com/maldous/CircularQuay/actions/runs/41800110400)
and [Dependency Review 41800110517](https://github.com/maldous/CircularQuay/actions/runs/41800110517)
pass. Eight of nine automatic PR/push runs are terminal at capture; the SDK
workflow is active. No manual rerun/cancellation occurred. Retain the exact
source/security/Sonar/TypeScript/Windows diagnostic receipts and
`checkpoint43-fa767-workflow-observation.json`; none qualifies a later tree.

## Exact 208bfe0 hosted receipt

Publication `208bfe016a8e00c55d9e2394b2a930eec64cd606` has tree
`386b60dbdc59d95aee63c1ca7dfeee9e2a71368d`. Complete readback records the
six pull-request workflows and three push workflows for that head as terminal;
none was rerun or cancelled. This is revision-scoped evidence, not a release
or production qualification.

The [source workflow](https://github.com/maldous/CircularQuay/actions/runs/41790956273)
has 29 successful and two failed jobs. All four previously recorded SDK
cyclomatic findings are closed. The Linux synthetic job reaches three new
assurance parser/test `gocyclo` failures; `gocognit` is therefore not reached.
The source-audit format and source steps pass. Its unit failure is attributed
only to `internal/integration`. Source inspection identifies that its tag-only
tests are explicitly selected by an untagged audit invocation; the report does
not retain an individual diagnostic. An empty untagged file must not conceal
that scope gap. Vet fails without retained diagnostic attribution. The report records
7,273/8,707 owned statements (83.530492707%). The current four-file repair is
source-only evidence and needs a later hosted run.

The [security workflow](https://github.com/maldous/CircularQuay/actions/runs/41790956279)
completes Staticcheck, govulncheck and CodeQL upload. Upload success does not
prove a zero alert count because no authoritative alert readback is retained.
Gosec retains eight unsuppressed findings; the security workflow therefore
fails.

The [SDK workflow](https://github.com/maldous/CircularQuay/actions/runs/41790956319)
has four successful and 28 failed jobs. The three native Go SDK jobs and Go
lock reproducibility pass. All nine TypeScript lint/type/audit lanes pass;
the observed 18-test run reports 15 passes and three fixture failures. Windows now records
56.89% actual TypeScript coverage, below the required threshold. The package
SIGPIPE repair, installed-ESM consumer correction and three fixture repairs are
source-only and require a new hosted run. Selected TypeScript and Go MCP checks pass, but do not qualify the
complete SDK matrix.

Python tools pass on all static operating-system lanes. The 38-test report
contains 76 events and a Windows AST match. All five owned module rows are
captured, while total coverage remains below the required threshold. Linux and macOS complete all five runtime
versions through installed/minimal checks. Windows Python 3.11 passes; Windows
3.12--3.15 fail native trust/config-initialization and remain under
investigation. The sole Python 3.15 MCP check passes. The existing Intel macOS
wheel/advisory blocker remains unchanged.

[Sonar](https://github.com/maldous/CircularQuay/actions/runs/41790956278)
uploads and completes CE analysis, but its strict gate fails: it records four
issues, 66.5% coverage, zero duplication and zero hotspots, absent required
new-code metrics, and the Python 3.15 fallback warning. The test-scope patch
classifies the configured test paths correctly while keeping implementation and
tooling validators as source. It does not create or ingest Python coverage XML
or TypeScript LCOV, so that project-wide coverage result remains incomplete.

[WireMock](https://github.com/maldous/CircularQuay/actions/runs/41790956340)
passes all three synthetic jobs, and
[Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41790956284)
passes. These isolated results do not qualify the failed source, security, SDK
or Sonar gates. Historical GitGuardian incidents remain a current-readback
gap: retained source provenance is not an authoritative current alert list and
does not establish closure.

Retain `checkpoint41-all-run-state.json` and
`checkpoint41-sonar-208bfe0.json`, along with prior immutable receipts. The
associated source-only fixes, README link/disclosure review, and publication
readback are not substitutes for hosted execution, artifact verification or
production qualification.

## Exact 20703ec8 hosted receipt

Publication `20703ec8c1bd66eab4899ec6ce27d4f0385adc1b` has tree
`af4f26232a5dce50ecbc6e889defc7d75a7a1ea8`. Its tested PR merge is
`621596917d7c489a2a40a759756a0d02cfa17518`, with the same tree. Full job
readback records all six workflows terminal, with no pending, skipped or
cancelled job. This receipt is distinct from the earlier `a26e39e` evidence.

The [source workflow](https://github.com/maldous/CircularQuay/actions/runs/41786298796)
has 29 successful and two failed jobs. Cross-builds, Windows native tests and
macOS native tests pass. Linux stops at four remaining SDK `gocyclo` findings;
`gocognit` is not reached because the shell exits after that failure. The source
auditor also fails unit tests and vet, but its retained report has no individual
package or diagnostic attribution for either. It reports 7,168/8,592 owned
statements (83.426443203%), so this is neither a coverage-only failure nor a
pass for unattributed checks.

The [security workflow](https://github.com/maldous/CircularQuay/actions/runs/41786298795)
completes CodeQL analysis and upload, which does not prove a zero alert count.
Staticcheck and govulncheck pass; gosec retains eight unsuppressed findings
(six G204 process boundaries and two G103 native quota ABI pointers). The
former G115 heap-metric conversion is no longer reported after saturation, but
the workflow remains failed until the eight findings are resolved.

The [SDK workflow](https://github.com/maldous/CircularQuay/actions/runs/41786298659)
has four successful and 28 failed jobs. The four successes are Go SDK contract
tests on Linux, Windows and macOS plus Go dependency-lock reproducibility. Its
TypeScript lanes show a reviewed lock and `npm audit` with zero vulnerabilities,
but retain owned lint and type failures. The observed Linux Node 22 lane runs 12
tests but has 52.91% line and statement coverage against the unchanged 100% gate.
Windows reports a 0/0 denominator that exited successfully, a coverage-tool
failure rather than evidence of coverage. The reviewed corrective source and
coverage validator are source-only evidence and require a new hosted run.

Python Ruff, mypy and Bandit pass on all three static lanes, while the reviewed
lock audit remains blocked by `cryptography 48.0.1` advisories. Its static
coverage gate fails with actual Linux coverage rows: `__init__.py` and
`errors.py` are 100%, `types.py` is 92%, `client.py` is 79%, and
`local.py` is 62%. Windows records a failed, zero-row report and macOS Intel
also fails without a separate parsed artifact. Sampled installed consumer checks
pass on Ubuntu, Windows and macOS Intel, but do not qualify the matrix. The only
selected official MCP read (Ubuntu Python 3.15) fails before a test pass because
its required audit configuration is absent; source tracing attributes that to
the synthetic fixture. Other MCP lanes are skipped by design.

[Sonar](https://github.com/maldous/CircularQuay/actions/runs/41786298735)
uploads and completes CE analysis `4515e664-cfcf-4be6-8585-5cb923c4f9be` with
zero issues, hotspots and overall duplication. Its strict gate still fails at
63.9% coverage, absent new-code metrics and the Python 3.15 fallback warning.
The reviewed scope patch classifies actual test paths as tests and retains
implementation and validator paths as analyzed source. It has not implemented
Python XML/Cobertura or JavaScript/TypeScript LCOV generation or ingestion, so
the scan imports only Go coverage and the project-wide result cannot be read as
solely untested Go. Produce, bind and explicitly ingest revision-scoped reports
for every language before treating Sonar coverage as qualified.

[WireMock](https://github.com/maldous/CircularQuay/actions/runs/41786298636)
passes its Linux, macOS and Windows synthetic suites, and
[Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41786298740)
passes. These narrow successes do not carry forward source, security, SDK,
Sonar, alert-count, release or production qualification.

Retain `checkpoint40-candidate-run-state.json`,
`checkpoint40-candidate-integration-dependency.json`,
`checkpoint40-source-security-review.json`, `checkpoint40-sdk-go-gocyclo-repair.json`,
`checkpoint40-python-sdk-qualification.json`,
`checkpoint40-python-helper-review.json`,
`checkpoint40-python-boundary-review-final.json`,
`checkpoint40-typescript-review.json`,
`checkpoint40-typescript-boundary-review.json`,
`checkpoint40-sonar-scope-review.json` and `ci-20703ec8-sonar.json`. These
receipts include source-only reviews and prior hosted results; none is a fresh
release, deployment or production qualification.

## Exact a26e39e hosted receipt

Publication `a26e39e46930893259b8385c65953b0896e5bdab` has tree
`ee50f613145dc21d1d6ea1db24452cd206b25a0f`. Full job readback records
all six workflows completed, with no pending or cancelled job. The PR's
tested merge `9248c1b37a4d67b8fd90d6a8373138822a6d3240` has the same
tree; this equivalence does not qualify a later source revision.

The [source workflow](https://github.com/maldous/CircularQuay/actions/runs/41781361710)
has 27 successful and four failed jobs. All 26 cross-builds and their aggregate
pass. Linux normal/race tests, vet and bounded fuzzing pass before nine
cyclomatic-complexity findings. The original receipt reported only five
non-SDK findings; four additional SDK findings were present in the same job.
The five non-SDK findings were corrected before checkpoint 40, while the same
four SDK findings persisted because their relevant blobs were unchanged. Windows
native tests fail in Go SDK fixtures using POSIX-only paths; macOS native tests
fail in update fixtures under the symbolic native temporary root. The source auditor separately executes and
fails unit tests and vet, and fails statement coverage at 7,084/8,578
(82.583352763%). Its retained report does not attribute the unit/vet failures
to individual diagnostics; it is not a coverage-only failure.

The [security workflow](https://github.com/maldous/CircularQuay/actions/runs/41781361692)
completes its CodeQL job. Staticcheck and govulncheck complete before gosec
fails nine unsuppressed findings: six G204 process boundaries, two G103 native
quota ABI pointers, and one G115 heap-metric conversion. The following source
saturates that conversion and refactors the reported complexity paths. The
other eight findings remain unresolved. A job's successful analysis/upload
does not independently establish zero outstanding security alerts.

The [SDK workflow](https://github.com/maldous/CircularQuay/actions/runs/41781361730)
has three successful and 29 failed jobs, including the failed aggregate. Go
SDK tests pass on Linux/macOS; Go lock reproducibility passes. Windows Go SDK
fixtures fail before runtime behavior. All nine TypeScript lanes produce the
same patched lock and stop at the reviewed-lock comparison, before SDK tools
or tests. The following lock is copied from those independently reviewed bytes,
with updated input/artifact provenance; installation and audit need fresh
qualification.

Python's three static jobs pass mypy and Bandit but expose a helper parser
fixture failure, Ruff I001 and the lock audit failure. Retained coverage progress
shows every test completes, including cancellation. Five normal coverage-table
rows are rejected by the old generic diagnostic parser; the artifact records
`malformed_capture`, not missing-line attribution. Windows consumer fixtures
supply an escaped executable hash; macOS consumer fixtures use a symbolic
temporary path. The Linux Python MCP fixture also fails before protocol
interaction because its required private state directory has not been created.
The following source corrects the evidenced parser, import, process-lifecycle
duplication and fixture causes. Safe structural coverage
reporting preserves the actual nonzero result and the 100% threshold.

Six macOS Intel Python locks contain `cryptography 48.0.1`, with
PYSEC-2026-3552/3553/3554 reported by pip-audit. Fixed upstream releases no
longer provide Intel macOS wheels. Keep that platform, binary-only installation,
hash checking and vulnerability gates intact; this upstream-wheel limitation
remains a launch blocker. No source distribution or ignored advisory supplies
a passing result.

[Sonar](https://github.com/maldous/CircularQuay/actions/runs/41781361703)
completes upload, CE analysis `3e7514ab-9a0d-400f-8ab7-d155ba725389`
and revision-bound readbacks. CE warnings and hotspots are zero. The strict
gate fails at 63.7% coverage, 0.2% duplication, one critical Go test-complexity
issue, absent required new-code metrics and the Python 3.15 fallback warning.
The duplication collector correctly attributes one block to
`sdk/python/src/cquay/local.py` at lines 364 and 431. Scanner exit 3 follows
successful analysis and expresses gate failure, not a scanner crash.

This scan imports only the Go profile. Python and TypeScript are analyzed
without their language coverage reports, so the zero-coverage sensor includes
those files in the project-wide denominator. The 63.7% Sonar result is not
solely a Go test gap. Qualification also requires actual revision-bound Python
coverage XML and JavaScript/TypeScript LCOV production and explicit ingestion;
a Go profile cannot establish their coverage.

[WireMock](https://github.com/maldous/CircularQuay/actions/runs/41781361699)
passes on Linux, macOS and Windows; [Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41781361782)
passes. These results establish only their exact synthetic/dependency scope.
Retain `checkpoint39-run-state.json`, `checkpoint39-source-review.json`,
`ci-a26e39e-sonar.json` and the SDK/helper/lock review receipts. New source
and tests have source-only review; no local heavy execution or production
qualification is claimed.

## Exact 9d12512 SDK and telemetry receipt

Publication `9d12512b11b462908f69159a16144dd80a0d0764` has tree
`bcfb89e4aa6974945163861f0f8bd3ce21ed0760`. Its
[source](https://github.com/maldous/CircularQuay/actions/runs/41776969369)
and [security](https://github.com/maldous/CircularQuay/actions/runs/41776969295)
checks expose MCP named-result redeclarations and an update constant/test
helper collision. Tests and analyzers blocked by compilation are not passes.
The following source corrects those causes; it requires new compilation.
Final metadata records 31/31 source jobs and 2/2 security jobs failed;
Staticcheck/CodeQL stop at compilation and later security analysis is skipped.

The [SDK run](https://github.com/maldous/CircularQuay/actions/runs/41776969298)
passes Go lock reproducibility. All three Go SDK jobs fail before tests on
the MCP declarations. npm locks install and expose actual lint/type/test,
coverage and audit failures, including a recorded 52.89% line-coverage
sample and 14 moderate toolchain dependency advisories. Python static
tool installation and helper tests pass; structured reports attribute 64
Ruff, 28 mypy and three Bandit diagnostics to source. Started runtime lanes
install their reviewed closures. Coverage and later consumer/MCP results
remain pending at this receipt; they are not qualified by source review.

The later authenticated Ubuntu coverage artifact `13298656230` reports a
1,200,004 ms timeout at `test_cancellation_propagates`. The fixture's fake
session rejects the adapter's `arguments` keyword before setting its start
event. The following test repair supplies the correct contract, bounded
waiting and cleanup; it does not establish a product cancellation failure
or a post-repair test pass. Retain `checkpoint38-python-timeout.json`.

[Sonar](https://github.com/maldous/CircularQuay/actions/runs/41776969359)
uploads and completes CE analysis `027a6c1b-e18e-4d48-ac45-4ecb0107e208`
with six HTTP-200 revision-bound readbacks. It binds synthetic merge
`685444aa9a0ed7500b83954eec722a71c967101a`, whose tree matches this
publication. The previous 17 issues are absent; three helper-regex issues
remain. Coverage is 0.0% after compilation failure, duplication 0.2%, and
hotspots/CE warnings zero. Required new-code metrics are absent. A scanner
warning explicitly rejects Python 3.15 and falls back to 3.14. The warning
gate remains failed; current analyzer source does not qualify an unreleased
tool or establish support in the pinned version.

[WireMock](https://github.com/maldous/CircularQuay/actions/runs/41776969322)
passes its tagged synthetic suite on Linux, macOS and Windows, and
[Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41776969351)
passes. Retain `ci-9d12512-source-security.json`, `ci-9d12512-sonar.json`
and `ci-9d12512b-wiremock-dependency-review.json`. None of these results
qualifies production or a later source candidate.

## Exact 4c703c8 SDK and telemetry receipt

Checkpoint `4c703c84fd9c1647b9a479c5525fdd3bfc653962` has tree
`f098a84ef22731a2cf9556ab664199154f11fb20`. Its
[source run](https://github.com/maldous/CircularQuay/actions/runs/41767625772)
and [security run](https://github.com/maldous/CircularQuay/actions/runs/41767625840)
fail on the `collectorURLs` type/function name collision in observability.
Native tests and cross-builds cannot compile; Staticcheck and CodeQL do not
complete analysis. Later gosec/govulncheck steps are not reached. This is
separate from the resolved predecessor module/import-cycle failures.

The [SDK run](https://github.com/maldous/CircularQuay/actions/runs/41767625933)
passes Go dependency-lock reproducibility. Go SDK qualification fails at that
same compiler error. All nine npm lanes produce byte-identical lockfiles;
all 15 Python runtime lanes resolve wheel-only closures, including Python
3.15. They then fail the intentional missing-reviewed-lock gate, before
runtime or distribution qualification. The Python static jobs pass toolchain
installation and lock-helper tests, record analyzer/audit failures, and are
cancelled when coverage exceeds the 30-minute job limit. Their unavailable
logs do not establish individual rule diagnoses or an exact stalled test.
The aggregate SDK check fails.

[Sonar](https://github.com/maldous/CircularQuay/actions/runs/41767625773)
completes upload and Compute Engine analysis but fails strict quality gates:
0.0% coverage after the compilation failure, 0.2% duplication, 17 open
issues, and one Python compatibility-version warning. Hotspots are zero.
Required new-code metrics are absent and fail closed. The analysis binds
the PR synthetic merge `45fe4e1d7c099d2be2022e910bd0666c9442ebc3`,
whose tree matches this public candidate. Retain
the revision-bound `ci-4c703c8-source-security.json` and
`ci-4c703c8-sonar.json` receipts. The following source/CI corrections require
their own hosted checks; none of these results qualifies production.

[Dependency Review](https://github.com/maldous/CircularQuay/actions/runs/41767625888)
passes. The [WireMock run](https://github.com/maldous/CircularQuay/actions/runs/41767625763)
verifies the pinned JAR on Linux, macOS and Windows, then fails tagged-suite
compilation at the same observability error. It is a completed failure, not
a cancellation or a provider-suite pass. Retain
`ci-4c703c84-wiremock-dependency-review.json`.

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
| Frozen local checkpoint23 | Native-fixture/provisioning/Sonar-label follow-up based on `2d1f1003d7c720f8d1ee5884ec576786358548c2`, 2026-11-04 | All 31 local source/test/static/format/module/workflow/compile checks passed with identical source manifests. Coverage is unchanged at 5,953/7,055 statements (84.37987%). Strict offline zizmor auditor collection has zero findings. A private copied Go distribution rebuilt the real auditor and passed a poisoned-PATH tiny fixture; the old shared writable layout was explicitly denied. | Actual native Windows/macOS and hosted-runner confirmation is separate. CQ's 100% coverage and zero-gosec gates still fail; a 1/1 fixture is not a product gate pass. |
| Frozen local checkpoint22 | Combined auditor/CI-flow/Sonar-remediation capture based on `bfaa66cc116093588dd1053a301bdf90a8ecef28`, 2026-11-04 | All 31 recorded checks passed with identical source manifests: whole-module normal/race, vet, Staticcheck all, formatting, module integrity, both complexity ceilings, actionlint, 13 Windows and six Darwin package test compilations. Coverage is 5,953/7,055 statements (84.37987%). The exact workflow-manifest/public-auditor smoke passes selected checks under a poisoned PATH. Expanded offline zizmor auditor-persona collection reports zero findings. | Coverage still fails 100%; scoped gosec still has eight Linux / nine Windows unsuppressed findings. Compile checks and a tiny synthetic audit do not qualify native OS execution, complete tool/runtime closure, exact-head Sonar, CI identity assertions or production. |
| [Native and cross-build CI](https://github.com/maldous/CircularQuay/actions/runs/41724962521) | `bfaa66cc116093588dd1053a301bdf90a8ecef28`, tree `0afc2f5f7fcaf00f1547b29f849540cf78cfc89f` | Native Linux, macOS and Windows tests and all 26 cross-builds passed. Synthetic merge `3be5764a6b551982bfa4962fc7c35a8c7000df2f` has the same public tree. [Advanced CodeQL](https://github.com/maldous/CircularQuay/actions/runs/41724962531) passed. | Source assurance fails required coverage (5,554/6,441 statements); gosec fails eight unsuppressed findings. Native runners do not qualify providers, reboot/recovery, installers or releases. |
| Managed DPoP diff review | `97b757a` → `bfaa66c`, scan `56553a87-7692-4deb-9432-d0f716652dc3` | Sealed complete review of all 37 changed files; zero reportable findings/candidates. Focused pinned Go 1.27.3 synthetic validation passed. | Complete diff coverage is not a whole-repository, live-provider or production audit. |
| Protected audit-tool fixture | Frozen isolated auditor slice, 2026-11-04 | A real source-profile run on a tiny clean synthetic repository passed with protected Git/Go/gofmt/Staticcheck under a poisoned ambient PATH. Binary, manifest, source and report hashes are retained. Other analyzers were explicitly not run. | Fixture-only Linux evidence, not CQ's gate pass, publisher authentication, transitive runtime closure, atomic execution or an OS sandbox. |
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
authenticated key-mapping endpoint, managed association administration API, protected
local custody, bundle-bound remote unlocking, and new CLI/MCP management paths
are not implemented or qualified in this source checkpoint. Upstream plugin
and hosting documentation is research evidence, not a CQ integration
receipt. The hosting/capacity/revenue calculations are explicit assumptions,
not benchmarks, customer traction, approved prices, or operating accounts.
No service, paid plan, credential, or BAU infrastructure was created or changed.
Local-to-managed custody migration remains a required unimplemented and
unqualified boundary: client-side inventory, actual age/OS/KMS reuse/import/
rewrap/re-encryption, compatible artifact transition, idempotent interruption
recovery, destination activation and explicit original/retained-copy handling.
A mode flag or provider interface does not establish migration support. No
operational key was inspected, uploaded or migrated during this work.
The current forecast request is hosting-only: software key operations and age
fetches, requests/CPU/elapsed runtime, database/storage/audit/backup, egress and
paid/free usage cohorts. It separates incremental spend within an already paid
plan from total allocated hosting cost and shows public-plan breakpoints.
Labor, external KMS, CRM and payment fees do not belong in that subtotal.
Earlier age-only, partial-KMS and labor-loaded price/margin proposals do not
answer it. Product tiers are not provider hosting plans. Actual account
entitlement, spare capacity and incremental cost remain unverified; conditional
public-plan examples and customer tariffs must not be presented as account facts.
Synthetic user/account and operator/admin frontend development/previews and a
loopback local client dashboard are now explicitly authorized. This supersedes
the earlier frontend deferral only. Real backend integration, operational
key/provider access and production deployment remain outside that authorization.
The [isolated local frontend candidate](#isolated-frontend-candidate-and-evidence-boundary)
has retained synthetic browser evidence; it is not integrated or qualified by
this runtime candidate. No verified hosted worker/site/database/project identity
for this service is recorded; repository research alone cannot establish the
state of unrelated provider accounts.

The confirmed hosted-service contract requires real remote cryptographic
wrap/unwrap and authenticated age-private-key retrieval. The explicitly approved
launch profile uses software custody within an existing paid Cloudflare or
Netlify allocation, not a mandatory paid external/native KMS. A runtime wrapping
key stays separate from the database and independently random profile data
keys; runtime/shared-KEK compromise can expose all reachable wrapped records.
This changes the earlier independent-custody assumption and does not satisfy
strict non-exportability/HSM requirements. No software hosted backend or
platform deployment is qualified. Authorized profile enrollment
automatically provisions both associations; it does not download an offline
key, enable hybrid or create a successful-fetch billing event. The human
clarified that KVM/PVM meant KMS configured in the application user profile,
not an OS home/profile or ambient provider credential chain. CLI/config opt-outs
must precede hosted network access. Hosted initialization, provisioning,
opt-out precedence, retrieval and metering remain absent. Current SOPS rejects
cloud-KMS/key-group metadata and writes age-only snapshots. Online-only has no
offline bypass. Explicit hybrid fallback retains availability-only admission,
finite scope/window/freshness/expiry, audit and revocation requirements; denial,
trust, malformed responses, expiry, cancellation and unknown failures do not
permit it. All configuration remains client-side. The lean launch can defer optional
enterprise/premium infrastructure within an explicitly approved release scope;
it cannot waive fundamental security or the engineering gates for shipped code.

The [management/discovery](management-and-discovery.md),
[administration security](administration-security.md) and
[security profiles](security-profiles.md) documents specify the complete v1
acceptance contract. Source inspection finds no remote management API/resource
implementation, typed remote administration clients, enrollment/tenant/SCIM/
delegation/online-revocation control plane, signed administrative-intent verifier
or unified assurance-profile enforcement. The runtime policy model/schema has
no profile field or managed resource definitions. The Go, TypeScript and Python
MCP clients deliberately accept only the existing four-tool one-page workload
profile; that profile must not be weakened to accommodate a management catalog.
Local `config` CRUD/CAS and pinned/signed repository bootstrap are distinct
implemented boundaries, not remote administrative capability.

Required implementation includes authoritative v1 management API/resource/
error/event contracts, separate protected enrollment/deployment schemas, exact
signature/profile/key lifecycle, transactional approval/revocation/audit state,
typed interface parity and product-wide monotonic profile enforcement. Schema
fields cannot be advertised as enforcement before their runtime path exists.
The design has bounded independent source review only; protocol vectors,
downgrade/abuse testing, native/provider/host qualification and independent
security audit of those implementations remain required. No security mode or
ISO/FIPS certification is established by these documents.

The [canonical governance](configuration-governance.md) and
[credential lifecycle](credential-lifecycle.md) contracts additionally cover
hierarchical defaults/locks, deterministic groups, clean environment sessions,
SCIM, provider-native mutation, GitHub/GitGuardian exposure gates, lifecycle
event export, workload federation/SPIFFE, posture/Conditional Access, external
policy decisions and forward migrations. Current source has no corresponding
canonical hierarchy compiler, managed session/provisioning resources, immutable
credential-incident associations, issuer mutation workflow or cross-interface
lifecycle administration. The flat policy's provider/delivery/telemetry
environment fields are not an organisation/group/environment authority model.
Existing runtime OTLP operation telemetry does not establish lifecycle-event
integration or an immutable audit store. Existing named-issuer/DPoP checks and
individual provider authentication do not qualify every federation, posture,
SPIFFE or external-evaluator profile in the design.

Read-only official documentation research establishes relevant API/privacy
constraints, not account entitlement or a live connector. GitGuardian base
incident metadata uses the separately qualified `incidents:read` profile;
broader validity routes are not assumed safe. Generic GitGuardian webhooks may
carry matched secrets and are excluded. GitHub metadata requires qualified
read-only repository access and `hide_secret=true`; unhidden responses deny.
No credential value/hash lookup, account enrollment, API scan, provider
mutation or secret transmission occurred. Explicit nonsecret version binding,
durable admission and synthetic/live qualification remain implementation gates.

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
The exact 208bfe0 receipt above records all three hosted synthetic suites
passing; that does not qualify other providers or a later source revision.
Historically, an agent's two
compile-only/vet batches ran despite the local resource hold; they executed no
test bodies, had no retained exact-source/environment fence, and are not an
integration pass. The root stopped further local execution. Retain the private
`wiremock-compile-only-execution-20261104.json` receipt.

The ephemeral hosted Sonar workflow is a producer with strict coverage,
warning, issue, hotspot and current-analysis checks. It uses a fresh isolated
server and a finite project-scoped credential. Definition and source review do
not establish a successful analysis or the independent auditor's complete
runtime binding. The exact 208bfe0 receipt above records completed analysis with
a failed strict gate and a separate successful Dependency Review PR result.

Keycloak realm export and OpenBao fixture contracts are provided; a real
issuer/provider container suite has not run. LocalStack runtime profiles,
external GCP federation, SPIFFE/SPIRE adapters, compatibility gateway endpoints,
managed UI/API/local service, online introspection/revocation, hardware/keychain
adapters and hosting integrations remain unimplemented or unqualified as
described in their respective contracts. No new schema field advertises those
unimplemented capabilities. No production deployment or release follows from
these design documents or development candidates.
