# Security policy

CircularQuay handles sensitive configuration. The project is under development and has no production-security certification or support SLA.

## Report privately

When available, use the repository's **Security → Report a vulnerability** private reporting route. Do not put real credentials, secret values, bearer tokens, recovery keys, private endpoint details, or unredacted scanner reports in public issues. If private reporting is not available, open a public issue asking for a private reporting channel without describing exploit details or including protected data.

Include a version/commit, affected capability, a minimal reproduction using fabricated values, the trust boundary crossed, and the expected versus observed behavior. Preserve evidence privately. Do not attempt to validate a report against other users' systems.

## Review scope

Review authorization before retrieval, per-user MCP disclosure, endpoint/identity binding, generation atomicity, lifecycle/expiry, permitted fallback, native privilege boundaries, file handling, logging, and release/update trust. No third-party backend, OS, compiler, or dependency is assumed infallible.

An authorized `read_secret` response is an intentional plaintext disclosure. It is not a promise that the client or model will keep the value confidential. A report that bypasses the required grant, exposes bootstrap material, crosses users, or leaks into another output channel is in scope.

## Dependency advisories

Publish upstream advisories and tested mitigations separately from first-party findings. Do not suppress an affected executable from the SBOM. The SOPS Vault-metadata advisory, [GHSA-jgf3-f6rg-8x3h](https://github.com/getsops/sops/security/advisories/GHSA-jgf3-f6rg-8x3h), requires explicit restricted `SOPS_HC_VAULT_ALLOWLIST` handling. Removing a finding from a dashboard is not remediation.

## Leaked credentials

Revoke or rotate at the issuer according to the affected service's incident policy. Deleting a file, retracting a Go module, removing registry metadata, or rewriting Git history cannot recall previous copies. Follow [the incident runbook](docs/operators/incident-response.md).

## Qualification

The [qualification matrix](docs/qualification.md) records what has actually been tested. Do not infer support for an unlisted version, provider entitlement, platform, or MCP client. Production qualification requires the published assurance profile and independent review.
