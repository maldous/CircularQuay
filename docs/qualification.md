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
