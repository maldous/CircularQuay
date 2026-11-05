# Security policy

CQ handles sensitive configuration. No product capability, analyzer result or deployment implies production-security certification or a support SLA; actual qualification is recorded separately.

## Report privately

When available, use the repository's **Security → Report a vulnerability** private reporting route. Do not put real credentials, secret values, bearer tokens, recovery keys, private endpoint details, or unredacted scanner reports in public issues. If private reporting is not available, open a public issue asking for a private reporting channel without describing exploit details or including protected data.

Include a version/commit, affected capability, a minimal reproduction using fabricated values, the trust boundary crossed, and the expected versus observed behavior. Preserve evidence privately. Do not attempt to validate a report against other users' systems.

## Review scope

Review authorization before retrieval, per-user MCP disclosure, endpoint/identity binding, generation atomicity, lifecycle/expiry, permitted fallback, native privilege boundaries, file handling, logging, and release/update trust. No third-party backend, OS, compiler, or dependency is assumed infallible.

An authorized `read_secret` response is an intentional plaintext disclosure. It is not a promise that the client or model will keep the value confidential. A report that bypasses the required grant, exposes bootstrap material, crosses users, or leaks into another output channel is in scope.

## SDK, agent and operational surfaces

Include official Go/TypeScript/Python adapters, native executable digest and
version checks, strict response parsing, cancellation, child environment/output,
plugin/skill configuration, bootstrap/discovery, and policy CRUD in the review.
Inspect telemetry/hook redaction, bounded exporter failure and update-feed trust
without exposing credentials or accepting release metadata as code authority.

Management grants, tenant mappings and delegated access have the same
server-side ceiling across CLI/UI/API/MCP. Required audit precedes protected
release. A raw read or arbitrary authorized program can retain plaintext;
private-key distribution, local custody and remote unlocking have distinct
revocation limits. Reports should identify the actual boundary, not assume that
an SDK, biometric prompt or model-response annotation guarantees recall.

Include the complete [administration trust contract](docs/administration-security.md)
and [security-profile enforcement](docs/security-profiles.md) in design and
implementation review: signer/issuer substitution, ambiguous signed data,
replay/rollback, approval and revocation races, profile downgrades, alternate
ingress and recovery are in scope. Baseline protections apply in every profile.

Review configuration composition as a client-side trust boundary: pinned
sources, dependency/namespace conflicts, ancestor locks, local reference bases,
private/corporate custody, complete candidate validation and safe provenance.
Raw/resolved configuration, source credentials and protected references must
not reach a hosted backend, proxy, analytics or logs. The hosted service's
narrow KMS wrap/unwrap and age-private-key operations do not authorize configuration custody.
Include first-run opt-outs before hosted network access, registration
authentication, retry/metering consistency and required-audit failures. Default
ease of use cannot waive any baseline; strict requirements must be verifiably
enforced, with unsupported deployments denied rather than silently downgraded.

Review dual-association enrollment, partial/uncertain provisioning, independent
KMS/age grants and revocation, authenticated context, key/version substitution,
typed provider failures and authorized hybrid windows. A copied data key,
plaintext or age private key cannot be remotely recalled. Check both profiles
against the same whole-product engineering and independent qualification gates;
quiet automation must not become implicit approval or a weaker verifier.

The authorized launch custody profile is a hosted **software** key service
within an existing paid Cloudflare or Netlify allocation, using vetted crypto,
a protected runtime wrapping key separate from the database and independently
random per-profile data keys. This explicitly changes the earlier independent
native-KMS custody assumption; it does not claim non-exportable/HSM protection.
A compromised authorized runtime/shared wrapping key can expose all reachable
profile keys. Per-profile data keys alone do not prevent that exposure.
Review exact context/version binding, runtime/preview/DB separation, audit,
backup/rotation/recovery and stronger-custody migration. Independent managed
KMS/HSM remains optional; policies requiring it must deny software custody.
Cost optimization cannot waive authentication, tenant isolation, protected
secret handling or any engineering/release gate. Actual host entitlements and
deployment remain independently qualified; no paid upgrade is assumed.

## Dependency advisories

Publish upstream advisories and tested mitigations separately from first-party findings. Do not suppress an affected executable from the SBOM. The SOPS Vault-metadata advisory, [GHSA-jgf3-f6rg-8x3h](https://github.com/getsops/sops/security/advisories/GHSA-jgf3-f6rg-8x3h), requires explicit restricted `SOPS_HC_VAULT_ALLOWLIST` handling. Removing a finding from a dashboard is not remediation.

## Leaked credentials

Revoke or rotate at the issuer according to the affected service's incident policy. Deleting a file, retracting a Go module, removing registry metadata, or rewriting Git history cannot recall previous copies. Follow [the incident runbook](docs/operators/incident-response.md).

## Qualification

The [qualification matrix](docs/qualification.md) records what has actually been tested. Do not infer support for an unlisted version, provider entitlement, platform, or MCP client. Production qualification requires the published assurance profile and independent review.
