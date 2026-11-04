# CircularQuay documentation

Start with the guide that matches your task. The [project landing page](../README.md)
introduces CircularQuay; these guides explain how to configure, operate and
verify it.

## Get started

- [Install, build and run a synthetic workload](getting-started.md)
- [Choose online, offline or hybrid access](getting-started.md#choose-an-availability-policy)
- [Understand how CircularQuay works](product-guide.md)
- [Compare capabilities with existing tools](comparison.md)

## Configure and automate

- [Configuration reference and v1 JSON schema](configuration.md)
- [Complete Linux deployment example](../examples/linux-deployment.json)
- [CLI commands, CRUD and examples](commands.md)
- [Configuration repositories, GitHub Actions, GitLab CI and Azure Pipelines](configuration-repositories.md)
- [Provider interoperability](providers.md) and [provider operator guide](operators/providers.md)
- [Platform and architecture matrix](platform-support.md)
- [Observability, diagnostics and release awareness](operational-configuration.md)

## Connect identities and tools

- [Integration guide](integrations.md)
- [Official Go, TypeScript and Python SDKs](sdks.md)
- [Enterprise identity, audit, observability, lifecycle and extension governance](enterprise-integration.md)
- [Default and strict security profiles](security-profiles.md)
- [Trusted discovery and complete administration lifecycle](management-and-discovery.md)
- [Administration signatures, approvals, trust and recovery](administration-security.md)
- [AWS, Azure, Google Cloud and emulator validation](integrations/cloud-provider-validation.md)
- [Keycloak and OpenBao fixture contracts](integrations/identity-and-provider-fixtures.md)
- [Compatibility gateway and independent local composition](integrations/compatibility-gateway.md)
- [Identity, key access and expiry explained](identity-and-expiry.md)
- [Hosted identity and application connections](hosted-identity.md)
- [Optional managed key service](managed-key-service.md)
- [Provider, native platform and AI boundaries](runtime-boundaries.md)
- [Agent skills and client-plugin contract](agent-integrations.md)
- [MCP operator guide](operators/mcp.md)
- [Credential stores, hardware and native protection](credential-boundaries.md)

## Develop, verify and release

- [Fast AI-agent orientation](agent-onboarding.md) and [repository instructions](../AGENTS.md)

- [Integration testing locally and in CI](integration-testing.md)
- [Testing and audit results](verification.md)
- [Dependency review, SBOMs and provenance](integrations/supply-chain.md)
- [Assurance contract](assurance.md) and [qualification evidence](qualification.md)
- [Architecture](architecture.md), [security model](threat-model.md) and [language ADR](adr/0001-language-and-assurance.md)
- [Implementation and qualification plan](implementation-plan.md)
- [Maintainer release and operator guide](../RELEASE.md)
- [Recovery](operators/recovery.md) and [incident response](operators/incident-response.md)
- [Contributing](../CONTRIBUTING.md), [support](../SUPPORT.md) and [private security reporting](../SECURITY.md)

Release and qualification records identify actual tested artifacts and remaining
gates. Design documents describe the v1 product contract; they are not evidence
that an integration, account, hosted service or release has been qualified.
