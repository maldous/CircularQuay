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

Owned SDK code is in the same release scope. TypeScript uses strict type
checking, ESLint/typescript-eslint, SonarJS and complexity ceilings of 10,
V8/c8 coverage and locked dependency audit. Python uses strict typing, Ruff,
Bandit, measured coverage and dependency audit. No inline suppression or
coverage exclusion may manufacture a pass. SDK package installation and
official MCP/native CLI conformance are separate checks, so an earlier
coverage failure cannot erase their evidence. Runtime and artifact versions,
dependency locks, source scope and exact results accompany each receipt.

The race lane is distinct from static release builds and is unavailable on some targets. Cross-compilation does not establish native correctness. Linux coverage cannot hide untested Windows/Android paths.

TLA+/TLC explores a bounded policy/state-machine model. Commit its assumptions, bounds and results, then replay traces against implementation tests. This does not formally verify the OS, TLS, providers, SDKs, or specification itself.

## Dependencies

The release policy is strict first-party cleanliness plus transparent upstream advisory reporting. Include every shipped dependency/executable in the SBOM. Publish tested mitigations and residual risks; block unmitigated exploitable conditions.

The current SOPS Vault-metadata advisory lists affected versions with an allowlist mitigation. Explicitly disable Vault endpoints for age-only operation and pin approved Transit endpoints. Do not suppress the advisory merely because the policy mitigates it.

## Evidence bundle

Every `cquay-audit run` needs a protected v1 [tool manifest](../internal/assurance/tool_manifest.schema.json) outside both the
source checkout and audit output. The portable schema describes structure;
the strict decoder also requires the integer spelling `1` and bounds paths to
4,096 UTF-8 bytes. Native loading enforces the host platform, protected local
files, canonical paths, location separation and actual digests. Profile tool
requirements are checked by the runner. The manifest declares fixed absolute paths, SHA-256
digests, version-output digests, and where applicable protected rules inputs.
The runner loads it before invoking Git, Go, gofmt, or an analyzer, rehashes
the selected executable, and checks its fixed version probe before each run.
`git`, `go`, and `gofmt` are mandatory. `--go` is optional, but a supplied
value must exactly match the manifest's Go path. The source workflow writes a
private manifest for these three tools and its declared `ssh-keygen` native
helper beside the temporary audit binary; other analyzers
are explicitly reported as not selected rather than resolved from PATH.

`gitleaks` needs a manifest-bound protected configuration file and
`govulncheck` needs a manifest-bound protected vulnerability snapshot. The
runner constructs the `file://` `GOVULNDB` URL from that snapshot path. An
enterprise run requires all nine tools and a complete bound Sonar JRE/plugin
runtime. A missing runtime binding records a required enterprise failure or
an optional source `not_run` result, and prevents launch of an unbound scanner.
The direct Sonar route is a separate diagnostic;
it does not satisfy the enterprise toolchain gate.

The runner supplies gitleaks an owner-only empty ignore file outside source
and enables `--ignore-gitleaks-allow`; a source `.gitleaksignore` therefore
cannot turn findings into a pass. Staticcheck runs its fixed `all` checks and
gosec runs with `-nosec`; source-owned Staticcheck/gosec configuration and
inline suppression markers, plus Actionlint ignore configuration, fail the
corresponding check rather than reducing its rules.

These checks identify opened binary bytes and version output, but they do not
authenticate a publisher, bind transitive Go/runtime inputs, close the
hash-to-exec race, or sandbox child processes.

Before and after an analyzer that consumes a manifest-bound rules file or
snapshot, the runner revalidates its protected metadata and complete digest.
That catches ordinary configuration drift and prevents accepting the result
after observed drift. It does not make use atomic against an actor able to
replace and restore files. Child PATH resolution is restricted to private
verified copies of the canonical `git` and `go` filenames and the optional,
explicitly declared `ssh-keygen` helper. Analyzer directories do not contribute
helpers. The optional v1 `helpers` array accepts only `ssh-keygen`, with its
absolute path and SHA-256; unknown helpers, empty arrays, extra fields and
source/output overlap are rejected. Canonical path collisions with tools are
also rejected; this is not a hard-link uniqueness guarantee. The helper is
copied privately and both original and copy are revalidated around every
command. Its separate report records preparation and current integrity, not
a version probe or proof that the helper ran. Omit it only when the audited
source does not require it; a missing dependency remains a real test failure. The Go distribution, Git helper/DLL dependencies,
the operating system, and child-process sandboxing remain separate closure
evidence.

The audit runner must record commit/artifact hashes, tool and rule versions, exact scope, tests, coverage numerator/denominator, model/mutation results, sanitized raw analysis, review decisions, upstream advisories, and native/provider/client qualification.

Source evidence uses a complete Git checkout rooted at the selected source directory. Before target tools execute, the v1 report binds the commit/tree and an ordered manifest of tracked regular files, Git modes/blob IDs and matching raw SHA-256 bytes. Ignored or untracked inputs fail the inventory check. Only the exact selected protected output subtree is excluded. A second binding detects post-check drift. Git replacement objects and ambient global/system configuration are disabled; checked-out line endings must match the tree. Windows records Git modes without claiming Unix executable-bit enforcement. File counts, individual reads and total source bytes are bounded. Manifest-owned files and package imports define formatting, tests, coverage, vet and Go-aware analyzer arguments, keeping generated output outside those scopes. A manifest directory without a native Go build fails the check instead of silently reducing coverage.

These manifests detect ordinary injection and drift. They do not form an atomic filesystem snapshot, authenticate the installed Git/analyzers, or isolate source tests from the auditor's host authority. Those execution and tool-qualification boundaries require separate evidence.

Failed Go test and vet checks can include an optional v1 `failure` object.
Its `state` is `observed`, `unknown` or `overflow`; an observed object contains
only manifest-owned package names and tracked public Go `path`/`line` pairs.
The parser retains at most 16 packages and 32 locations, with line numbers
from 1 through 1,000,000. Unknown identities, malformed attribution and
overflow do not expose partial raw diagnostics. Command arguments, diagnostic
messages, endpoints and values remain absent. This metadata does not change
the check's failure, exit code or qualification requirements.

No operational credentials or raw MCP secret responses belong in public evidence. Synthetic assurance works without external secret-provider accounts. Live-provider qualification is opt-in and separate.

## Current status

This document defines acceptance criteria. Actual results belong in generated audit evidence and [qualification.md](qualification.md). An omitted or skipped check is not a pass.
