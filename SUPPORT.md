# cquay support

CircularQuay is the project; `cquay` (pronounced “see-key”) is its app, executable and operations skill.

Use the selected version's qualified platform/provider/client matrix and
release notes to determine support. Certification, response-time commitments
and paid support terms require an explicit separate agreement; installation or
a passing analyzer does not create one. Actual release/channel availability is
recorded in [qualification](docs/qualification.md), not inferred from examples.

- Ask usage and design questions in [Discussions](https://github.com/maldous/CircularQuay/discussions).
- Report reproducible bugs and propose features through [Issues](https://github.com/maldous/CircularQuay/issues/new/choose). Use synthetic fixtures and include the exact version, OS/architecture and safe error category.
- Report vulnerabilities through the [private security channel](https://github.com/maldous/CircularQuay/security/advisories/new), following [SECURITY.md](SECURITY.md).
- Obtain release assets and release notes from [GitHub Releases](https://github.com/maldous/CircularQuay/releases). Follow [RELEASE.md](RELEASE.md) for authenticity verification, installation, upgrades and removal. Development snapshots and release candidates are explicitly prerelease builds.
- Contribute through a feature branch and pull request to `develop`, following [CONTRIBUTING.md](CONTRIBUTING.md).

Never include actual credential values, ciphertext from operational environments, private endpoint details, tokens, recovery material, environment dumps or raw MCP secret responses in public support requests. Maintainers should request a reduced synthetic reproduction rather than sensitive diagnostics.

For [lifecycle integrations](docs/credential-lifecycle.md), include only safe
operation IDs, exact supported profile/version and outcome categories. State
whether the problem concerns directory propagation, exposure evidence, a policy
exception, issuer mutation or consumer activation. Never include raw GitHub or
GitGuardian alert bodies, secret hashes or replacement credentials. Unsupported
SCIM/provider/plan capabilities need qualification; no support report authorizes
account enrollment, purchases, incident dismissal or live revocation.

The intended [customer account/support service](docs/managed-key-service.md#end-customer-accounts-subscriptions-and-support)
keeps authenticated customer usage/history, billing/invoice/subscription status
and support lifecycle separate from hosting-provider operator support. Reuse an
existing approved private channel or a qualified customer helpdesk; no Freshdesk,
Stripe or other account/plan is connected by this document. Public GitHub
Discussions/Issues remain for synthetic public material, never private billing
or key/config details. Customer ticket/history APIs must authorize exact tenant/
account and safe fields. No config/source-credential attachment, key dump,
payment-card field or automatic diagnostic upload belongs in support. Paid
support/SLA/SSO and unattended financial actions require explicit scope and
actual entitlements; custom support/account UI implementation remains deferred.

Before a release is marked stable, its release notes must identify supported versions, known limitations, migration and rollback constraints, dependencies and the evidence supporting each qualified capability. Registries and package managers have separate publication and verification records; a GitHub release alone does not establish availability elsewhere.
