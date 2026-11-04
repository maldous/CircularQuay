# How CircularQuay fits with existing tools

CircularQuay brings provider access, session setup, CI automation, native
delivery, and agent permissions into one workload policy. The rows below show
the capabilities together, including where existing tools already overlap.

**✓** = available in the described scope; **❌** = a documented gap, explained
in its footnote; **—** = unknown, not established as equivalent, or outside the
compared product's scope. A dash is not a claim that a feature is absent.
Vendor features may require a particular edition, integration, or subscription.

| Capability | CircularQuay | OpenBao[^compare-bao] | Vault[^compare-vault] | Infisical[^compare-infisical] | Doppler[^compare-doppler] | 1Password[^compare-onepassword] | SOPS + age[^compare-sops] |
| --- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| Read multiple existing providers without migrating their values | ✓ | — | — | — | — | — | — |
| Bind selected fields or secret references to workload variables | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| Resolve every required input before activating a complete generation | ✓ | — | — | — | — | — | — |
| One-command configuration, authentication, and scoped shell bootstrap | ✓ | — | — | — | — | — | — |
| Bootstrap from a public/private Git repository, tree, or file URL | ✓ | — | — | — | — | — | — |
| Verify configuration revisions, signatures, and update provenance | ✓ | — | — | — | — | — | — |
| Discover enrolled configuration through enterprise/device/cloud context | ✓ | — | — | — | — | — | — |
| Explicit overrides with deterministic precedence and safe diagnostics | ✓ | — | — | — | — | — | — |
| Schema-backed configuration CRUD and a validation/repair doctor | ✓ | — | — | — | — | — | — |
| Scoped machine identities and noninteractive authentication | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| Federate CI-issued identity tokens without a stored provider token | ✓ | ✓ | ✓ | ✓ | ✓ | — | — |
| One flow command detects GitHub Actions, GitLab CI, or Azure Pipelines | ✓ | — | — | — | — | — | — |
| Discover CI configuration from job context or an enrolled service | ✓ | — | — | — | — | — | — |
| Policy-controlled delivery to the job process and optional later steps | ✓ | — | — | — | — | — | — |
| Deliver credentials through a child-process environment | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Deliver through declared protected OS credential interfaces | ✓ | — | — | — | — | — | — |
| One policy spans native services, containers, CI, and cloud targets | ✓ | — | — | — | — | — | — |
| Encrypted offline configuration or fallback artifact | ✓ | — | — | — | ✓ | — | ✓ |
| Offline approval binds identity, application, workload, and expiry | ✓ | — | — | — | ❌[^compare-offline-expiry] | — | — |
| Hybrid fallback limited to approved availability failures | ✓ | — | — | — | — | — | — |
| Optional managed age-key access with identity-based grants and administration | ✓ | — | — | — | — | — | — |
| Optional enterprise IdP alongside standalone local-policy operation | ✓ | — | — | — | — | — | — |
| Configurable contexts, scopes, roles, clients, and access windows | ✓ | — | — | — | — | — | — |
| MCP integration for AI tools or secret administration | ✓ | — | ✓ | ✓ | ✓ | — | — |
| Hosted remote MCP endpoint | ✓ | — | ❌[^compare-vault-remote] | — | — | — | — |
| Run tool processes with credentials supplied outside prompt text | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Separate MCP action permissions and exact-key raw-read grants | ✓ | — | — | — | — | — | — |
| MCP configuration CRUD, policy, grants, revocation, and audit | ✓ | — | — | — | — | — | — |
| Audited agent-to-agent runtime delegation with scope and a deadline | ✓ | — | — | — | — | — | — |
| Independent audit command and reproducible qualification evidence | ✓ | — | — | — | — | — | — |

The combination is CircularQuay's value: one configuration connects an approved
identity and workload to existing providers, online or authorized offline use,
and the right runtime delivery path. Stopping future delivery does not recall
an already issued credential; its original provider controls revocation and
expiry. Process injection describes how credentials reach a tool; it does not
prevent that tool from disclosing them. Authorized raw-secret reads deliberately
return plaintext. Release, implementation, and test receipts are recorded
separately in the [qualification evidence](qualification.md).

[^compare-bao]: OpenBao documents [JWT/OIDC authentication](https://openbao.org/docs/auth/jwt/), [field-selecting templates and process environments](https://openbao.org/docs/agent-and-proxy/agent/template/), and [agent caching](https://openbao.org/docs/agent-and-proxy/agent/caching/). Its cache covers managed tokens and leased secrets; this is not evidence of an equivalent portable, identity-bound offline approval.

[^compare-vault]: Vault documents [JWT/OIDC federation](https://developer.hashicorp.com/vault/docs/auth/jwt), [agent field selection and runtime delivery](https://developer.hashicorp.com/vault/docs/agent-and-proxy/agent/template), and an [MCP server](https://developer.hashicorp.com/vault/ai/mcp-server). Its separate [Enterprise Agentic IAM](https://developer.hashicorp.com/vault/ai) also governs agent identities and delegated workflows; the dash in Vault's delegation cell does not deny that capability.

[^compare-infisical]: Infisical documents [OIDC machine identities](https://infisical.com/docs/documentation/platform/identities/oidc-auth/general) and [credential injection into MCP servers](https://infisical.com/blog/managing-secrets-mcp-servers). The MCP mark includes this integration route; it does not imply that the example custom server provides every administration feature in this table.

[^compare-doppler]: Doppler documents [scoped service tokens](https://docs.doppler.com/docs/service-tokens), [CI OIDC identities](https://docs.doppler.com/docs/service-account-identities), [encrypted fallback and runtime delivery](https://docs.doppler.com/docs/automatic-fallbacks), and an [experimental MCP server](https://docs.doppler.com/docs/mcp). OIDC identities require its Team or Enterprise plan. Its MCP interface includes raw-value reads and configuration writes.

[^compare-onepassword]: This column compares 1Password Secrets Automation and CLI: [scoped service accounts](https://www.1password.dev/service-accounts) and [runtime secret references](https://www.1password.dev/cli/secrets-scripts). [SaaS Manager MCP](https://support.1password.com/saas-manager-mcp-server/) is a separate product/API and is not counted as a Secrets Automation MCP endpoint.

[^compare-sops]: [SOPS](https://getsops.io/) supports encrypted files with offline age identities or online key services. Its CLI documents [selected-value extraction](https://getsops.io/docs/usage/common-operations/#extract-a-sub-part-of-a-document-tree) and [process injection](https://getsops.io/docs/usage/advanced/#passing-secrets-to-other-processes) with `exec-env`; these are encryption/runtime utilities, not evidence of a hosted policy or delegation service. The original key or provider still controls decryption authority.

[^compare-offline-expiry]: Doppler's [fallback FAQ](https://docs.doppler.com/docs/automatic-fallbacks#faq) states that a fallback file can be used indefinitely and has no built-in expiry; an expiry check or cleanup needs additional work. This cross concerns that built-in offline deadline, not Doppler's online token lifetimes or an external enforcement layer.

[^compare-vault-remote]: Vault's [MCP transport documentation](https://developer.hashicorp.com/vault/ai/mcp-server/reference/protocols) explicitly limits the documented server to local use. This cross describes that server's deployment scope, not Vault's remote APIs or Enterprise Agentic IAM.

## Related work

[SOPS](https://getsops.io/) supplies encryption, [age](https://age-encryption.org/) supplies portable recipients, and [OpenBao](https://openbao.org/) supplies established authorization and key-management capabilities. [Teller](https://github.com/tellerops/teller), [vals](https://github.com/helmfile/vals), and [Chamber](https://github.com/segmentio/chamber) already make provider retrieval and environment injection easier. [sops-nix](https://github.com/Mic92/sops-nix) and [Flux](https://fluxcd.io/flux/guides/mozilla-sops/) demonstrate native lifecycle integration.

CircularQuay focuses on their operational intersection: explicit inheritance and export policy, lifecycle-aware offline copies, native service delivery, optional authenticated disclosure, and evidence-backed recovery. It does not claim to invent provider interoperability or to replace those projects.

[Documentation](README.md) · [CircularQuay](../README.md)
