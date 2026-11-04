# Agent skills, client plugins and MCP

CircularQuay's agent integration combines portable operating instructions with
an explicitly enrolled runtime connection. The skill is named `cquay`, pronounced
“see-key”; installing instructions does not install a service, authenticate a
caller or grant access to values. Identity-provider and MCP integration are
optional. The CLI and official SDKs also operate independently.

## Components and identities

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
original grant. Model-visible administration returns metadata, never key bytes.

An approved tool can consume scoped values inside its process and return
allowlisted results. `read_secret` is a separate intentional plaintext release;
a client/model may retain it. A plugin cannot promise that hidden tool calls,
response annotations or secret wrappers erase those copies.

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
