# Documentation inventory and dispositions

This map inventories reviewed documentation source
`e30577022e53d3debcfb747ba28f6e477bfb7137`, tree
`351df54e3febe72d473712ae5391b2143ec37661`. That documentation branch is not a
release or a claim that its instructions are installed everywhere. Select the
actual product revision and qualification evidence for the consuming task.

The counts below describe that named parent, before this map was added. They
are reproducible source observations, not automatically refreshed live status.
Product maintainers own the versioned documents; current assignments, private
custody, review and publication controls retain their separate authority.

## Complete tracked-document inventory

Git tracks 1,860 paths at the inspected source, including 148 Markdown files and
29 `.txt` files. No Markdown, MDX, RST or text guide is outside the groups below;
the `.txt` inputs are classified separately rather than counted as prose guides.
The only tracked `AGENTS.md` is the repository-root file; no nested `AI.md`,
`AGENTS.md` or repository-local `.agents/skills/` package is present.

| Source group | Markdown files | Audience and disposition |
| --- | ---: | --- |
| Root entrypoints | 8 | `AGENTS.md`, `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `RELEASE.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`: product, contributor, security and release contracts. |
| [`docs/`](README.md), including its subdirectories | 48 | Canonical product/design, configuration, operator, assurance, architecture and evidence references. Use the maintained index and category map below. |
| [Synthetic examples](../examples/README.md) | 1 | Fabricated policy inputs and prerequisites, not provisioned accounts, credentials or permissions. |
| [`examples/landing-motion/`](../examples/landing-motion/README.md), excluding `explore-source/` | 11 | Presentation briefs/reviews, demo-service boundaries and asset rights/provenance. These are not native-runtime or live-backend acceptance. |
| [`examples/landing-motion/explore-source/`](../examples/landing-motion/explore-source/README.md) | 53 | Frozen, explicitly selected product-document snapshot for the public Explore reader. Do not substitute these copies for the selected checkout's canonical source. |
| [Formal model guides](../formal/README.md) | 2 | `README.md` and `SenderConstraint.md`: finite model assumptions, reproduction and implementation limits. |
| [Portable product skill](../integrations/skills/cq/SKILL.md) | 4 | Skill plus contained operations/boundaries/lifecycle references; readable instructions are not an installed executable or authenticated MCP session. |
| [SDK and conformance guides](sdks.md) | 5 | Go, Python, TypeScript and synthetic conformance READMEs, plus Python's Darwin source-build contract. Each language and native artifact has its own qualification boundary. |
| [Synthetic browser guides](../web/README.md) | 5 | Browser README, product/design/coverage guides and the landing brief under `web/.impeccable/`. Simulated flows are distinct from native or production services. |
| `.impeccable/critique/` | 10 | Retained dated presentation and workspace-hook observations. They are evidence, not new execution instructions or fresh rendered acceptance. |
| [Pull-request template](../.github/pull_request_template.md) | 1 | Contributor checklist; a checked box does not replace the required result or independent review. |

The 29 text inputs comprise 26 Python dependency/requirements files under
`sdk/python/` and three landing inputs: `ads.txt`, `robots.txt` and
`requirements-explore.txt`. Those files control their actual dependency,
indexing/advertising or generation boundary; they are not additional user guides.
The root [LICENSE](../LICENSE), package licenses, JSON schemas/manifests,
workflow definitions and source help remain authoritative non-Markdown inputs.

To reproduce the inventory without installing a documentation service:

```sh
git ls-tree -r --name-only e30577022e53d3debcfb747ba28f6e477bfb7137
```

Use blob identities and a Git diff to refresh affected rows after a source
change. A file count or path index does not establish semantic API coverage.
Do not upload private source to an external index merely to enumerate files.

## Canonical category and gap map

| Category / readers | Maintained source | Coverage and remaining boundary |
| --- | --- | --- |
| Agent and contributor orientation | [AGENTS](../AGENTS.md), [onboarding](agent-onboarding.md), [continuation checklist](next-agent-prompt.md), [contribution rules](../CONTRIBUTING.md) | Route current tasks, source/toolchain checks, resource admission and ownership. Reconcile dated handoffs with the actual assignment; no guide can transfer a claim or install an optional tool. |
| Product overview and acceptance | [README](../README.md), [product guide](product-guide.md), [implementation plan](implementation-plan.md), [comparison](comparison.md) | Describe the complete v1 contract. Intended behavior, a schema name and a capability label are not proof of implementation or a qualified release. |
| Architecture and rationale | [Architecture](architecture.md), [language ADR](adr/0001-language-and-assurance.md), [runtime boundaries](runtime-boundaries.md), [execution redesign](execution-boundary-redesign.md) | Preserve design assumptions and the native/SDK/browser/service distinction. A proposal is not evidence that all callers migrated or a security finding closed. |
| Policy and configuration | [Configuration](configuration.md), [repository/bootstrap contracts](configuration-repositories.md), [composition reference](composition-reference.md), [governance](configuration-governance.md), [discovery/administration](management-and-discovery.md) | CLI/schema examples and full-v1 requirements coexist. Verify actual parser, dispatcher, precedence and atomic-activation behavior before naming an available operation. |
| CLI and native installation | [Commands](commands.md), [getting started](getting-started.md), [installation](operators/installation.md), [platform matrix](platform-support.md), [release guide](../RELEASE.md) | Use source help and exact toolchain/artifact pins. A successful source build does not establish every platform, installer, signature or native lifecycle result. |
| Provider and identity consumers | [Provider map](provider-readiness.md), [provider contract](providers.md), [operator guide](operators/providers.md), [identity support](identity-support.md), [hosted identity](hosted-identity.md), [identity/expiry](identity-and-expiry.md), [credential boundaries](credential-boundaries.md) | Constructor/registration and credential/version boundaries have source references. Actual selected account/resource/permission, executable and identity-provider interoperability remain separately qualified. |
| SDK and agent integration | [SDK overview](sdks.md), [Go](../sdk/go/README.md), [Python](../sdk/python/README.md), [TypeScript](../sdk/typescript/README.md), [conformance](../sdk/conformance/README.md), [agent integration](agent-integrations.md), [MCP operations](operators/mcp.md) | Linked Go, separately selected native processes and caller-owned MCP sessions are different interfaces. A portable skill is not a published client plugin, server deployment or authenticated release. No exhaustive symbol-by-symbol API review is claimed by this inventory. |
| Security and enterprise control | [Security policy](../SECURITY.md), [threat model](threat-model.md), [security profiles](security-profiles.md), [administration security](administration-security.md), [enterprise integration](enterprise-integration.md), [credential lifecycle](credential-lifecycle.md) | Preserve grants, tenant separation, client-local configuration, required audit and strict-profile enforcement. Design/default ease of use cannot waive unsupported or unverified controls. |
| Managed service and migration | [Managed-key-service contract](managed-key-service.md), [discovery](management-and-discovery.md), [recovery](operators/recovery.md) | Internal key-service source is not a hosted, authenticated durable enrollment/billing service. Selected custody, authorization/storage/audit, interrupted migration and provider entitlement need their actual evidence. |
| Observability and updates | [Operational configuration](operational-configuration.md), [enterprise guide](enterprise-integration.md), [incident response](operators/incident-response.md) | Separate optional telemetry/advisory checks from required audit, minimum versions and trusted installation. Source configuration is not collector, exporter, outage or live-service qualification. |
| Browser and public presentation | [Browser source guide](../web/README.md), [product](../web/PRODUCT.md), [design](../web/DESIGN.md), [coverage](../web/coverage.md), [landing guide](../examples/landing-motion/README.md), [asset provenance](assets/README.md) | Public landing/Help content, synthetic workbench/account/operator simulations and real backend delivery are distinct. Preserve the current frontend owner, approved brief and original private/sanitized-source boundaries. |
| Assurance and evidence | [Assurance](assurance.md), [verification](verification.md), [qualification](qualification.md), [integration testing](integration-testing.md), [supply chain](integrations/supply-chain.md), [finite models](../formal/README.md) | Record the exact source/artifact, command, toolchain, platform, failure/skip and reviewer. Existing receipts keep their original scope; source review, a model pass or unrelated workflow is not whole-product acceptance. |
| Support, conduct and releases | [Support](../SUPPORT.md), [conduct](../CODE_OF_CONDUCT.md), [release](../RELEASE.md), [changelog](../CHANGELOG.md) | Keep actual channels, rights and distribution facts aligned. An example version, registry name or proposed tariff is not a released artifact, selected entitlement or commercial agreement. |

The canonical `docs/` index also routes the detailed integration/fixture guides
and SVG asset notes. No new command, provider profile, remote administrator
interface or release procedure is introduced by this map.

## Copies, historical evidence and update routing

The [landing guide](../examples/landing-motion/README.md) identifies the Explore
snapshot as an explicit working-tree capture with its own base and per-file
hashes. Normal generation uses those frozen inputs; new Markdown is not
automatically exported. Its explicit local refresh option is a separate selected
operation, not permission to copy every current guide or private record.
Update canonical product source first, then qualify a separately selected public
Help capture through its existing manifest/generator and owner. Do not hand-edit
frozen copies to make a search result appear current.

Qualification documents, critique records and archived snapshots retain their
stated dates/revisions. Do not relabel old results as this revision's execution,
sum coverage from separate branches, or silently rewrite retained failures.
Private checkpoints, customer configuration, credentials, operational receipts
and transient worker allocation are not public documentation inputs.

For a documentation change, inspect the exact changed text, target links,
examples, named commands, source interfaces and disclosure boundary. Obtain
author-distinct review. Reuse accepted unaffected product evidence; run only the
checks required by genuinely changed behavior under the selected executor's
resource admission. Never run product suites just to decorate a prose change.

## Deliberate cross-project differences

The inspected [ANQR instructions](https://github.com/maldous/anqr-v3/blob/036a403ab1e7f82b72e3c6006937532e1657f2a3/AGENTS.md)
govern its TypeScript/Node product and distinguish source checks from application
journeys. CircularQuay instead has a Go native core, explicit language adapters
and selected CLI/MCP boundaries. Matching evidence/ownership vocabulary does not
make the interfaces or toolchains interchangeable.

The inspected [Cloud source map](https://github.com/maldous/aldous-info-platform/blob/6eec52b7bfe5f884d3a141a4754de59d6b2376ae/docs/SOURCE_MAP.md)
describes a static public export, with separate live-site publisher custody.
CircularQuay's public presentation likewise does not establish native/provider
or hosted-service qualification. These are scoped documentation comparisons,
not imported code, shared runtime dependencies or mutual release gates.

Remaining semantic API, whole-product security, rendered/device, provider,
artifact/install, migration/recovery and release obligations stay with their
existing contracts and owners. Completing this inventory does not complete
those outcomes or authorize publication to a default/release destination.
