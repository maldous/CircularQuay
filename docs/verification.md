# Testing and verification

Current workflow results are available from [GitHub Actions](https://github.com/maldous/CircularQuay/actions). A passing tool or platform is one scoped result; release acceptance requires every applicable gate in [the assurance contract](assurance.md).

The badges below are retained **historical scoped receipts**, not current-head or release badges. Each links to its recorded commit, platform, tool and limitations in [qualification](qualification.md#bounded-receipts).

[![Linux synthetic receipt](https://img.shields.io/badge/Linux%20synthetic%20receipt-c2%20passed-2f855a)](qualification.md#bounded-receipts)
[![CodeQL scoped receipt](https://img.shields.io/badge/CodeQL%20advanced%20receipt-c2%20passed-2f855a)](qualification.md#bounded-receipts)
[![Staticcheck scoped receipt](https://img.shields.io/badge/Staticcheck%20scoped%20receipt-passed-2f855a)](qualification.md#bounded-receipts)
[![govulncheck scoped receipt](https://img.shields.io/badge/govulncheck%20scoped%20receipt-0%20reachable-2f855a)](qualification.md#bounded-receipts)
[![Cyclomatic scope](https://img.shields.io/badge/gocyclo%20scoped%20receipt-%E2%89%A410-2f855a)](qualification.md#bounded-receipts)
[![Script and workflow analyzers](https://img.shields.io/badge/script%20and%20workflow%20receipts-passed%20scope-2f855a)](qualification.md#bounded-receipts)
[![TLC finite model](https://img.shields.io/badge/TLC%20finite%20model-bounded%20pass-2f855a)](qualification.md#bounded-receipts)


## Read the evidence for the right revision

The retained [20703ec8 hosted receipt](qualification.md#exact-20703ec8-hosted-receipt)
records an earlier completed set of source, security, SDK, Sonar, WireMock and dependency-review
runs for that exact tree. It includes native Go SDK and WireMock successes,
installed-client results and the unresolved source, security, SDK, dependency
and coverage failures. A successful test or analyzer is not a passing language
matrix or release gate. Later corrections require their own hosted execution;
consult the [qualification matrix](qualification.md) for subsequent receipts.

| Evidence | What it establishes | What it does not establish |
| --- | --- | --- |
| Source review and syntax checks | The documented scope was inspected or parsed. | Successful builds, runtime behaviour or an unchanged later tree. |
| Native SDK, installed-client and protocol tests | The recorded package, platform, dependency set and allowed/denied behaviour passed the named checks. | Every SDK language/platform, live-provider access, registry publication or complete coverage. |
| Browser preview checks | The exact preview's syntax and focused Node tests on the named runtime and OS, once executed. | Exhaustive browser interactions, accessibility, protected native storage, real server authorization or complete owned-code coverage. |
| Sonar and language coverage | Results for the recorded source inventory and imported coverage reports. | Coverage for an unimported language, absent new-code metrics or a later revision. Go, Python and TypeScript reports must be bound to the scanned candidate. |
| CodeQL, dependency and secret scanning | The selected analysis, audit or scan phase completed with its recorded outcomes. | A zero-alert count from upload alone, closed secret incidents or a cleared finding without readback. |
| Artifact and registry verification | Exact published package bytes and installation/provenance checks, once recorded. | A release merely because source, manifests or workflow definitions exist. |

The [historical bounded receipts](qualification.md#bounded-receipts) remain
available for their original scopes. Compare source revision, tree, tool/runtime
versions, dependency locks and report inventories before combining evidence.
Partial successes and unexecuted checks remain visible; failed gates are not
waived by a later source correction or a green badge for a different workflow.

Independent audit, live providers, reboot/recovery and release-artifact qualification have separate acceptance evidence. No release or production qualification is established.

The [browser workflow](../.github/workflows/browser.yml) uses dependency-free
checks across pinned Node 22/24/26 on Linux, macOS and Windows. The preview
remains synthetic; historical producer browser evidence binds its own frozen
source, not later integration fixes. Required Sonar and whole-product coverage
gates still apply to shipped frontend code. The README workflow badge reports
actual branch status and does not assert a security or release pass.

See [integration testing](integration-testing.md) for disposable provider validation and [security reporting](../SECURITY.md) for private vulnerability handling.

[Documentation](README.md) · [CircularQuay](../README.md)
