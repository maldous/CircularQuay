# Agent skills, client plugins and MCP

The user explicitly authorized initial hosted SOFTWARE crypto + age using only
existing minimum paid Cloudflare/Netlify: vetted crypto, protected KEK outside
DB, independently random profile DEKs and scoped auth/tenant isolation/required
audit. Runtime/shared-KEK compromise may expose all reachable DEKs; this is not
independent non-exportable/HSM custody. Stronger required profiles are unavailable
here, with versioned managed/HSM migration retained. No paid external KMS/IdP/
monitor/helpdesk/upgrade or account connection is a launch prerequisite or
authorized now. Actual existing tiers/issuer metadata is unverified; conditional
CF-first versus Netlify-included-Identity must be qualified, not invented. The
current forecast is hosting-only, with product tiers distinct from provider plans,
allocated versus already-paid incremental spend and allowance/DB-active inflections;
no labor, CRM/payment fees or customer price/margin is embedded. Synthetic/mock
frontend previews and loopback local-configuration simulations are authorized
separately; production or real-backend UI remains unapproved, and real backend
tests remain separate from static design evidence.

CircularQuay's agent integration combines portable operating instructions with
an explicitly enrolled runtime connection. The skill is named `cquay`, pronounced
“c-key”; installing instructions does not install a service, authenticate a
caller or grant access to values. Identity-provider and MCP integration are
optional. The CLI and official SDKs also operate independently.

## Components and identities

The native runtime, internal key-service package and synthetic frontend are
separate integration surfaces. Source `internal/keyservice` exposes Go core
decisions and software/Transit primitives through explicit dependencies; its
presence adds no CLI verb, public SDK method, MCP tool or hosted URL. Inspect
the real dispatcher/catalog before invoking anything. Browser account/operator
roles, payments, crypto and fleet actions are synthetic presentation, not grants.

For a browser task, use the assigned frontend checkout's `web/README.md` and
`web/coverage.md`, its pinned schema/example hashes and exact interaction
receipts. Do not create a second preview process or copy local configuration
exports into a review/upload bundle. Typed control reachability is distinct
from individual edit round trips and native/provider validation. The phone
route is user-confirmed development access, not a deployed backend or remote
configuration endpoint. Changes to schema/fixtures require coordinated
frontend reconciliation and renewed evidence, not an inferred compatibility pass.

| Component | Owning source or contract | Responsibility |
| --- | --- | --- |
| Portable skill | [`integrations/skills/cquay/SKILL.md`](../integrations/skills/cquay/SKILL.md) and its contained references | Select safe operations, preserve task scope and keep values out of model-visible diagnostics. |
| Skill UI metadata | [`agents/openai.yaml`](../integrations/skills/cquay/agents/openai.yaml) | Display the skill and supply a default invocation prompt; no runtime grant or server endpoint. |
| Client plugin | The selected release's client-specific manifest and verified connection configuration | Package the same skill and declare only the selected client's supported transport/dependencies. |
| MCP server | [`mcp serve`](operators/mcp.md) with protected policy | Authenticate the connection and enforce per-operation grants before provider access and release. |
| MCP Registry / MCPB | [Distribution contract](../RELEASE.md#6-publish-mcp-distribution) | Publish metadata or a verified client bundle, separately from the runtime protocol and skill format. |
| Official language SDK | [Go, TypeScript and Python](sdks.md) | Support application-owned local or MCP execution with strict responses, cancellation and explicit disclosure. |

The native archive configuration includes the skill at its versioned source
path; verify the selected built archive before installation. The current
repository contains this portable skill and UI metadata, but no client-plugin
manifest, verified App binding or MCP Registry descriptor. Packaging and
publication remain release-owner work. A client plugin
contains its own declared manifest, skill resources and transport configuration;
it is not inferred from a skills directory or an example provider policy.
Release records identify the actual package, schema, supported clients and
installation evidence. Do not invent a plugin/App ID, hosted MCP endpoint,
OAuth client or public-listing approval when preparing a package.

## General UIF project profiles

The required shared UIF harness must distinguish the CircularQuay product from
OffMenu and other project profiles. Multi-project capability remains planned/
unqualified until actual accepted-source/job/isolation evidence covers both
affected profiles. Do not inherit another project's credentials, state/data
roots, toolchain, MCP permissions, reviewer or publication target. Profile names
and skill instructions are not authorization or proof a selector is implemented.

CQ migration and OffMenu configuration/data/BAU work continue in parallel, with
only real per-unit dependencies. Use current authoritative coordination, the
single canonical migration/impact record and existing task/lease handover. The
source owner retains sole Linear writing for this assignment. Verified references
are UIF `P-UIF-16`, OffMenu `P-UIF-14` and
[CQ `P-UIF-17`](https://linear.app/maldous/project/circularquay-656f1967b41c).
The source owner has populated the canonical [CQ handoff migration checklist](session-handoff.md#uif-migration-and-delivery-closure)
with obligations, ownership and actual harness gaps. Dynamic view/filter
creation/readback remains pending. Do not
turn those gaps into a second issue tree, new planner/dispatcher or guessed
selector. Review/publication and source/job/isolation evidence apply separately
to each affected profile; independent work proceeds within its authority. A
project record does not admit a task/lease, cut over runtime or expand production.

Read the [canonical subproject map](product-guide.md#required-general-uif-harness-and-project-migration)
before selecting any route: CQ secret/config/workload and crypto/age semantics,
OffMenu household/product/food data and private profile semantics, and UIF's
general infrastructure/orchestration/governance are distinct. Each product keeps
its actual accepted repo/ref, toolchain, privacy/state roots, permissions,
reviewers and publication evidence. Multi-project integration must demonstrate
that separation; a product adapter or failed project job alone does not qualify
or condemn the shared harness.

## Connection and management

[Discovery and management](management-and-discovery.md) separates configuration,
service and authorization discovery and defines the remote administration
resource lifecycle. Its [security contract](administration-security.md) applies
equally to agent calls. A distinct management catalog does not expand the
bounded workload catalog; each has its own audience, schemas and grants.

For local stdio, select an absolute authenticated executable, protected policy
and state directory, and the intended OS identity. Do not put tokens, passwords,
age private keys or shell-profile evaluation in the client manifest. Agents
sharing an OS user share that trust boundary; use a separate OS or authenticated
service boundary for stronger separation.

For HTTP, enroll the exact resource URL and qualified authorization-server
profile, client, subject, scopes and optional sender constraint. An ID token,
site password, login cookie or self-reported agent label is not an operation
grant. Inbound credentials are never forwarded as provider credentials.

The v1 management contract covers configuration, mappings, grant windows,
revocation and audit through the same authority ceiling as CLI/UI/API
administration. Discover the selected server's catalog and match the operation
to its documented contract; do not fabricate tool names or substitute a mutable
script for a missing administrative capability. Delegation records grantor,
recipient, workload, operations, expiry and reason, and cannot exceed the
original grant. Model-visible administration returns metadata, never key bytes. A separate
frontend checkout may use only synthetic/mock administration data and loopback
local-configuration simulations; it cannot supply a real backend, host proxy,
provider/key/account access or production operation.

An approved tool can consume scoped values inside its process and return
allowlisted results. `read_secret` is a separate intentional plaintext release;
a client/model may retain it. A plugin cannot promise that hidden tool calls,
response annotations or secret wrappers erase those copies.

## Lifecycle administration

Agent clients follow the [canonical hierarchy and locks](configuration-governance.md)
and [lifecycle contract](credential-lifecycle.md). V1 CLI/API/SDK/MCP parity means
one authorization/validation/approval/audit engine; a plugin is not a shortcut
around a missing or denied operation. Optional SCIM, exposure exceptions and
provider-native rotation/revocation require their separately authorized profiles.
Inspect the selected release's actual catalog/schema; do not fabricate commands
or request secrets to correlate an incident. A read-only connector may receive
sensitive fields. GitHub list/get profiles require `hide_secret=true` and reject
unhidden secrets; only approved bounded metadata reaches the model. A warning
presents a separately authorized finite exception, never permission to bypass a
known exposure block.

Optional workload/posture/external-policy profiles use that same canonical
ceiling and finite freshness, not caller-supplied identity or an unchecked OPA
allow. Unsupported challenges/decisions fail closed, and migrations fix forward
without discarding locks or exposure/revocation state.

No skill text can connect an account, purchase access, dismiss a vendor incident
or revoke an issuer credential by implication. Keep directory, GitHub,
GitGuardian and provider credentials in their independent protected references.
Incident closure, issuer invalidity and replacement activation remain distinct
facts. Approved OTel export never implies immutable or durable required audit.

The [single-file or master/include contract](configuration-repositories.md#single-file-or-master-and-includes-configuration)
also applies to plugin/agent clients. One resolved model does not require one
physical file. Select only supported resolver operations and never invent an
include schema, remote local-file fetch or unsafe manual merge. Corporate locks
survive personal/department sources; personal values, keys and unauthorized
provenance never upload through a plugin, explain tool or telemetry hook.

## Hosted key service first run

For existing local AGE/KMS assets use the [complete migration contract](managed-key-service.md#end-to-end-local-to-managed-migration):
client-local metadata planning, source/destination permissions, actual import/
export or non-exportable limits, rewrap versus re-encrypt, protected staged
output, per-asset idempotency/recovery and separately approved retirement. A
mode flag or dual enrollment is not permission to inventory/upload local keys.
Apply existing project security/assurance gates; unavailable backend is
unqualified, and mock conformance is not real provider migration evidence.

The [first-run contract](management-and-discovery.md#first-run-and-minimal-managed-registration)
initializes the hosted KMS/age profile associations after configuration
resolution and authentication, unless CLI or config explicitly disables it.
Both opt-outs apply before any hosted network call, under validated enterprise
locks; a conflict is not permission to register anyway. An agent uses only the
selected release's actual enrollment/opt-out capabilities, not invented tools.
Idempotent CI uses the declared workload identity and no human prompt.

Authorized registration now must provision BOTH genuine hosted KMS and age-key
associations, superseding the earlier age-only baseline. It is neither a paid
subscription consent, offline-key download, hybrid activation, local-key import
nor enterprise config-orchestration enrollment. No personal config,
provider value, private key or sensitive provenance uploads by default. The
user clarified KVM/PVM meant KMS configured per profile; that requires actual
adapter/schema qualification, not a guessed mode flag. Free allowance and an
upgrade opportunity use undecided event allowances/prices and never auto-charge. Key
windows gate future private-key release; copied keys retain decryption ability.

Hosted utilization is an [access event](managed-key-service.md#profile-utilization-and-fair-metering),
not a profile/key slot. Independent host fetches count separately; local reuse
and failed/denied precommit calls add no customer uses. A retry of a committed
logical access reconciles its one event, not a new charge or extra grant.

Production custom admin UI/screens remain unapproved. A separate frontend
checkout may build synthetic/mock dashboard previews and loopback local
configuration simulations, but intended UI parity is not authority for an agent
to publish a dashboard or connect a real backend, key, provider or account.
Reuse existing platform services while preserving CLI/text onboarding.

All config operations and file/URL reads stay client-side with the user's
network/VPN/source credentials, full option validation and local save. Never
send contents/source credentials through a hosted backend, proxy, tool response,
analytics or logs. Browser CORS/file permissions are real; use permitted native
CLI/SDK/client integration, not a server-side workaround. The hosted API is
profile-to-KMS/age association/retrieval plus necessary enrollment/meter/billing,
not remote client-config CRUD. Protect key bytes from display/logging. Apply the
[per-profile KMS/hybrid contract](managed-key-service.md#per-profile-age-and-kms-custody)
consistently across actual interfaces; KMS support is not inferred from cloud
secret-reader SDKs.

The explicitly confirmed [landing/client-SPA design](managed-key-service.md#explicitly-confirmed-ui-design-and-bounded-preview-work)
is authorized only for synthetic/mock functional previews and loopback
local-configuration simulations. Production or real-backend implementation
remains unapproved. Optional user-configured client OTel queries/collection do
not authorize a hosted telemetry service, bypass CORS/VPN or make a browser an
OTLP listener. No agent may treat design text as production build approval.

[Account/subscription operations](managed-key-service.md#end-customer-accounts-subscriptions-and-support)
are customer-scoped, not access to hosting-provider administration. CLI/API/SDK/
MCP share authorization, safe metadata DTOs, idempotency, current provider-state
reconciliation and audit. Use provider-hosted payment entry; no card/provider
admin credentials or portal session secrets in model logs. Future unattended
financial mutations require explicit user boundaries and applicable confirmation,
including exact tenant/action/plan/amount/currency/time; unsupported methods and
missing authority fail rather than use an admin-token workaround. This is design,
not an account connection, payment or custom UI implementation approval.

## Packaging and qualification

The release owner validates the selected manifest schema, package identity,
version, entry point, relative resource paths, license, dependency declarations
and privacy description. Include all skill references and UI metadata; reject
escaping paths, symlinks, dependency caches, synthetic private state and real
credentials. Client-specific overlays preserve the same identity and scope.

Test installation, discovery, invocation, denial, cancellation, configuration
reload, update and removal in each claimed client. Verify a named action's
outputs and a raw-read denial independently. Package or listing validation does
not qualify the server, live provider, SDK or native host. A public directory
has its own eligibility and privacy review; passing a local client test does
not establish acceptance there. [Publisher/operator directions](../RELEASE.md#7-skills-and-client-specific-plugins)
remain separate from the product guides.

The [qualification matrix](qualification.md) records actual package availability
and client/platform evidence. Product and skill instructions describe the
complete v1 contract without substituting a design for a passing receipt.
