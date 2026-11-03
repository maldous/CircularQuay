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
| Sonar and full assurance | Harness in development | No independent completed gate claimed here. |
| Public distribution | Source publication in progress | No release, module-index, OCI or MCP Registry publication claimed. |

## Source workflow scope

The committed workflow defines Linux formatting, module-integrity, unit, race, vet, bounded Go-fuzz and cyclomatic-complexity checks. It also defines synthetic native test jobs for the GitHub-hosted `macos-15-intel` and `windows-2025` runners, plus Linux cross-compilation for the release build matrix. Workflow definition is not a result: no successful run, runner image, tool output or platform receipt is claimed here until a specific run is retained as evidence.

The native jobs exercise only synthetic Go behavior on the runner architecture. They do not prove release archives, Windows ARM64 behavior, DACL/DPAPI service isolation, macOS signing/notarization, service-manager enrollment, reboot recovery, provider connectivity, or consumer installation.

The source audit profile invokes Staticcheck, gosec, govulncheck, gitleaks and actionlint when they are supplied by the audit environment. They remain optional in that profile today, so a missing external binary is reported as `not_run` rather than a source-profile pass for that tool. The enterprise profile and the 100% owned-statement/Sonar requirements remain unchanged; no absent external check, unexecuted fuzz target, cross-build, or configured workflow satisfies them.

Local complementary tooling has exercised TLC, ShellCheck/shfmt, PSScriptAnalyzer, Hadolint/Compose, offline zizmor, Syft SBOM/license inventory, and a bounded mutation sample. The retained development receipts are described in the session handoff; these checks do not yet have a complete committed CI toolchain or release-artifact receipt. Native reboot/restore/provider trials remain separate requirements. No sampled result, static script analysis or SBOM inventory establishes production qualification.

Record each qualification receipt with a source commit, artifact digest, platform/provider/client version, setup, test outcome and sanitized evidence link. Do not mark a row tested based on a plan, cross-compilation, generated manifest, or absence of a complaint.

Existing operational machines are outside this implementation's deployment scope. Operators install from documented artifacts and provide their own installation feedback.

## Bounded receipts

The README receipt badges link here. A receipt applies only to the listed
revision and scope; it is not a release gate, audit conclusion, security
certification, native qualification, or assertion about a later working tree.
The static local receipts below were collected in an isolated development
checkout. Their detailed logs are retained outside tracked source and are not
release evidence.

| Receipt | Revision and time | Scope and observed result | Limit |
| --- | --- | --- | --- |
| [Linux synthetic GitHub Actions job](https://github.com/maldous/CircularQuay/actions/runs/41633578905/job/125909071713) | `24d8401fd57043eb1958804dc57a1c28504b1832`, 2026-11-03 | GitHub-hosted Ubuntu 24.04 with Go 1.27.3: formatting, module integrity, unit tests, race tests, vet, three 15-second fuzz targets, and tracked-Go-source `gocyclo -over 10` passed. | This one successful job does not override the failed Windows/macOS/source-assurance jobs in the same [failed run](https://github.com/maldous/CircularQuay/actions/runs/41633578905), nor qualify native behavior. |
| Staticcheck | `308670c62198a1a5e168d0c4583686a65098db6b`, 2026-11-03 11:53 UTC | Staticcheck 2026.2.2 (module 0.8.2), built and run with Go 1.27.3 on `cmd/...` and `internal/...`: exit 0. | Local bounded receipt from a development checkout; the revision differs from the GitHub Actions receipt and later changes require a rerun. |
| govulncheck | `308670c62198a1a5e168d0c4583686a65098db6b`, 2026-11-03 11:53 UTC | govulncheck 1.9.0 with Go 1.27.3: zero reachable vulnerabilities. | The report retained three required-module advisories without reachable calls: GO-2026-6866 and GO-2026-6860 in `golang.org/x/net@v0.61.0`, and GO-2026-5932 in `golang.org/x/crypto@v0.58.0`; it is not an all-dependency clearance. |
| Cyclomatic complexity | `24d8401fd57043eb1958804dc57a1c28504b1832`, 2026-11-03 | The successful Linux job ran pinned `github.com/fzipp/gocyclo@v0.6.0` against all tracked Go sources; no function exceeded 10. | Cyclomatic complexity is distinct from Sonar cognitive complexity and does not make the Sonar gate pass. |
| Shell and packaging analyzers | Development checkout, 2026-11-03 | ShellCheck 0.9.0 and shfmt 3.12.0 passed tracked shell templates; Hadolint 2.12.0 passed the Dockerfile; Docker Compose 5.1.4 parsed the example Compose files without interpolation; actionlint 1.7.12 passed workflows. | Static/template-only scope. systemd verification could not complete because the example executable is intentionally absent; no image or service runtime is qualified. |
| PowerShell and workflow security analyzers | Development checkout, 2026-11-03 | PSScriptAnalyzer 1.25.0, invoked with PowerShell 7.6.9 using `-NoProfile -NonInteractive`, reported no finding for `packaging/windows/enroll-example.ps1`; zizmor 1.32.0 offline, config-free strict collection reported no finding for `.github/workflows`. | No Windows execution, registry/service test, remote-action provenance, or online zizmor audit occurred. |
| TLA+/TLC finite model | Development checkout, 2026-11-03 | TLA+ tools 1.8.0 completed the checked `formal/Delivery.tla` / `Delivery.cfg` finite model. | The model has bounded assumptions and does not verify operating systems, TLS, provider SDKs, the specification itself, or complete implementation conformance. |

## Open gates and non-badge evidence

These results are intentionally not represented by a green README badge.

| Evidence | Current recorded state | Consequence |
| --- | --- | --- |
| Source workflow run 41633578905 | Overall failure: Linux synthetic job passed, but Windows synthetic, macOS synthetic, and source-assurance jobs failed. | No repository-wide CI-pass claim and no required-check enforcement should be inferred. |
| SonarQube | The 2026-11-03 scan of `24d8401fd57043eb1958804dc57a1c28504b1832` has two remaining issues, 72.1% coverage, and a quality-gate error. | Sonar and the 100% owned-coverage gate are failing; remediation and a fresh scan are required. |
| Managed security scan | The sealed owner-held report records five validated findings: one high and four medium. | This is active remediation work, not an audit pass. The report is not reproduced here because it is retained in the owner’s secured evidence channel. |
| SBOM and mutation sample | An SBOM/license inventory was generated; the bounded mutation sample scored 81.99%. | Neither result certifies security or satisfies the required complete mutation evidence. |
| Uncommitted JSON tests | A local JSON coverage result reached 100% only with uncommitted new tests. | It is not published evidence and cannot support a current-source or release claim. |
