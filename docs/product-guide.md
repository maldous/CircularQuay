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

The [qualification matrix](qualification.md) records tested platforms,
providers, release artifacts, and operational evidence.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/environment-evolution-dark.svg">
  <img alt="Overlapping developer requirements evolved from shell profiles through dotenv, CI and containers, managed identities, and multi-platform AI workloads." src="assets/environment-evolution-light.svg">
</picture>

Read [identity and expiry](identity-and-expiry.md), [runtime boundaries](runtime-boundaries.md), and [the architecture](architecture.md) for the trust and delivery model.

[Documentation](README.md) · [CircularQuay](../README.md)
