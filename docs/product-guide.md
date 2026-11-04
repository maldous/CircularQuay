# How CircularQuay works

- **Explicit interoperability:** select exact provider resources and string fields; keep existing stores authoritative.
- **Optional enterprise identity:** connect trusted IdP identities and registered applications to configurable contexts, workloads, operations, and exact secret keys.
- **Time-bound access:** respect provider and offline deadlines; with identity integration enabled, also verify signed access grants and client-key possession.
- **Workload isolation:** reject duplicate names and undeclared resource access; keep provider bootstrap credentials out of application environments.
- **Complete generations:** resolve required inputs before activation instead of applying a partial refresh.
- **Portable encryption:** use upstream SOPS and age rather than a new encryption format.
- **Native delivery:** use process environments, protected files, OS credential interfaces, and declared deployment targets.
- **Optional MCP:** expose authorized status and named actions, with a separate permission for each raw secret disclosure.
- **Inspectable assurance:** source, tests, a policy model, release identities, and a reproducible audit route.

## Bootstrap, integrate and operate

`cquay bootstrap` selects local configuration first, then enrolled discovery;
explicit options override environment settings, local configuration and
verified discovery in that order. Discovery provides a trusted configuration
reference, never authority by itself. `cquay flow run` applies the same model
to GitHub Actions, GitLab CI and Azure Pipelines. See the
[configuration and CI guide](configuration-repositories.md).

The [official SDKs](sdks.md) give Go, TypeScript and Python applications the
same policy boundary, safe error categories, cancellation and explicit
secret-disclosure controls. [Operational configuration](operational-configuration.md)
selects remote metrics, traces, diagnostics and release awareness, with bounded
export and no secret-bearing telemetry. Required audit has its own durability
policy. Interactive release notices continue without consuming workload input;
automation remains non-interactive and installation remains explicit.

The [enterprise contract](enterprise-integration.md) defines identity lifecycle,
access review, audit retention/export, version compatibility, recovery and
extension governance for the complete v1 product. CLI and optional MCP
administration share scoped policy checks and audited, time-bound delegation.
The [portable skill and client-plugin guide](agent-integrations.md) explains
agent setup without adding implicit authority. The
[managed key service](managed-key-service.md) has its own linked UI/API
and custody contract. No client can recall plaintext or a copied age private
key; encryption alone does not enforce an access deadline.

The [qualification matrix](qualification.md) records tested platforms,
providers, release artifacts, and operational evidence.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/environment-evolution-dark.svg">
  <img alt="Overlapping developer requirements evolved from shell profiles through dotenv, CI and containers, managed identities, and multi-platform AI workloads." src="assets/environment-evolution-light.svg">
</picture>

Read [identity and expiry](identity-and-expiry.md), [runtime boundaries](runtime-boundaries.md), and [the architecture](architecture.md) for the trust and delivery model.

[Documentation](README.md) · [CircularQuay](../README.md)
