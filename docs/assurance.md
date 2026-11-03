# Assurance contract

The intended claim is: **this revision passes the published assurance profile for the recorded toolchain, platform, and provider matrix.** It is not a security certification or proof of absence of vulnerabilities.

## First-party gates

- Zero first-party compiler/linter/analysis warnings and errors.
- Zero first-party Sonar issues at every severity across overall and new code.
- No accepted/false-positive status, `NOSONAR`, broad exclusion, or reduced rule set used to manufacture a clean result.
- Evidence-backed hotspot review when emitted by the pinned analyzer.
- Zero complexity-rule violations; maximum per-function cyclomatic/cognitive complexity of 10.
- Target 100% coverable first-party executable statement/line coverage with the actual denominator, platform scope, and missing paths reported.
- Every security decision outcome exercised by meaningful positive/negative tests.
- Security-relevant non-equivalent mutants detected; equivalent classifications explained.

Literal zero cyclomatic complexity is not a useful requirement: executable functions contribute to the metric. Go statement coverage is not branch coverage, MC/DC, or proof of correct assertions.

## SonarQube Community

Use an exact reference server/scanner/analyzer/ruleset and a separate latest-stable qualification lane. The researched starting reference is Community Build 26.10.0.133078. Current Go analysis supports the selected language version; verify exact support when upgrading.

The custom gate covers overall and new code, waits for asynchronous completion, and disables small-change exemptions. Missing coverage reports, incomplete source scope, or analysis failure block the result. A ratings are insufficient without zero issue counts.

A fresh independent server may emit review-required hotspots even after another instance reviewed them. Publish the review ledger, raw scan, profile, gate and exclusions. Do not auto-mark them safe.

Community analysis is one component. Its language/advanced-analysis coverage is not identical to paid editions. Shell/PowerShell, advanced dataflow, native process behavior and provider identity require complementary evidence.

## Complementary checks

Formatting, vet, Staticcheck, focused gosec, govulncheck, CodeQL, fuzzing, race tests, property/fault tests, shell/workflow analyzers, repository/asset checks, secret scanning, SBOM/license analysis, native integration and actual reboot/restore tests each have a distinct role.

The race lane is distinct from static release builds and is unavailable on some targets. Cross-compilation does not establish native correctness. Linux coverage cannot hide untested Windows/Android paths.

TLA+/TLC explores a bounded policy/state-machine model. Commit its assumptions, bounds and results, then replay traces against implementation tests. This does not formally verify the OS, TLS, providers, SDKs, or specification itself.

## Dependencies

The release policy is strict first-party cleanliness plus transparent upstream advisory reporting. Include every shipped dependency/executable in the SBOM. Publish tested mitigations and residual risks; block unmitigated exploitable conditions.

The current SOPS Vault-metadata advisory lists affected versions with an allowlist mitigation. Explicitly disable Vault endpoints for age-only operation and pin approved Transit endpoints. Do not suppress the advisory merely because the policy mitigates it.

## Evidence bundle

The audit runner must record commit/artifact hashes, tool and rule versions, exact scope, tests, coverage numerator/denominator, model/mutation results, sanitized raw analysis, review decisions, upstream advisories, and native/provider/client qualification.

Source evidence uses a complete Git checkout rooted at the selected source directory. Before target tools execute, the v1 report binds the commit/tree and an ordered manifest of tracked regular files, Git modes/blob IDs and matching raw SHA-256 bytes. Ignored or untracked inputs fail the inventory check. Only the exact selected protected output subtree is excluded. A second binding detects post-check drift. Git replacement objects and ambient global/system configuration are disabled; checked-out line endings must match the tree. Windows records Git modes without claiming Unix executable-bit enforcement. File counts, individual reads and total source bytes are bounded. Manifest-owned files and package imports define formatting, tests, coverage, vet and Go-aware analyzer arguments, keeping generated output outside those scopes. A manifest directory without a native Go build fails the check instead of silently reducing coverage.

These manifests detect ordinary injection and drift. They do not form an atomic filesystem snapshot, authenticate the installed Git/analyzers, or isolate source tests from the auditor's host authority. Those execution and tool-qualification boundaries require separate evidence.

No operational credentials or raw MCP secret responses belong in public evidence. Synthetic assurance works without external secret-provider accounts. Live-provider qualification is opt-in and separate.

## Current status

This document defines acceptance criteria. Actual results belong in generated audit evidence and [qualification.md](qualification.md). An omitted or skipped check is not a pass.
