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


The [060639 receipt](qualification.md#exact-060639-development-receipt) records native Linux/macOS/Windows, all 26 cross-builds, both complexity ceilings, Staticcheck, govulncheck and CodeQL passes. Coverage and gosec failed. The latest historical Sonar receipt also fails coverage; it is not a current-source Sonar result. Independent audit, live providers, reboot/recovery and release-artifact qualification have separate acceptance evidence.

See [integration testing](integration-testing.md) for disposable provider validation and [security reporting](../SECURITY.md) for private vulnerability handling.

[Documentation](README.md) · [CircularQuay](../README.md)
