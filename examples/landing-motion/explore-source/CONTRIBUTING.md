# Contributing

New coding agents: read [repository instructions](AGENTS.md) and
[agent orientation](docs/agent-onboarding.md) before selecting setup or checks.

Use fabricated fixtures and the documented local audit route. Do not introduce live credentials or operational ciphertext into this public repository.

1. Read the architecture, threat model, and relevant ADR.
2. Keep policy decisions pure and I/O adapters narrow.
3. Add negative tests for new authority, parsing, lifecycle, and failure behavior.
4. Run formatting, package tests, and the applicable audit profile.
5. Update capability/qualification documentation honestly.

Source files follow Go conventions: lowercase package names, descriptive snake-case filenames, colocated `*_test.go`, and package-local `testdata/`. Documentation/assets use lowercase kebab case. Required metadata files retain conventional uppercase names. Do not add timestamped `final`, `new`, `latest`, author-name variants, arbitrary provider commands, or catch-all `utils` packages.

Code is maintained by humans. Use descriptive names, small cohesive functions, straightforward control flow and explicit error handling. Avoid cryptic abbreviations, compressed statements and repetitive generated-looking helpers. Conventional short names such as `ctx` and loop indexes are appropriate when their meaning is immediately clear. Complexity limits are ceilings, not style targets.

Document exported Go identifiers with GoDoc comments that begin with the identifier's name. Describe package responsibilities and externally observable behavior. At security boundaries, explain the invariant, trust assumption and reason for a non-obvious check: comments should preserve the rationale for authorization order, lifecycle restrictions, isolation and failure handling. Do not merely paraphrase each statement. Errors and comments must not contain credentials or sensitive provider payloads.

Run the repository's pinned formatter and analyzer checks, including `gofmt`; require deterministic formatting with no remaining diff. Review naming, comments and readability manually as well as by machine. Keep CLI help, policy examples, diagrams and operator commands consistent with the actual implementation. A platform example is not qualified until its stated acceptance test has run on that platform.

## Gitflow

`main` contains qualified stable releases; `develop` is the integration branch. Branch `feature/<descriptive-name>` from `develop` and merge it back through a reviewed pull request with the applicable checks. Do not push feature implementation directly to `main`.

Create `release/<version>` from `develop` to stabilize a release. After review and qualification, merge it into `main`, create its immutable version tag, and merge the release fixes back into `develop`. Create urgent `hotfix/<descriptive-name>` branches from `main`; review and verify the fix, release it through `main`, then merge it back into `develop` and any affected open release branch. Preserve the review and merge history; do not rewrite a published release tag.

During initial bootstrap, an empty repository-initialization commit may establish `main` and `develop`. Initial implementation belongs on its feature branch until the same qualification requirements are met. Branch presence is not a passing quality gate.

Keep the native core in Go; the official TypeScript and Python SDKs are explicit client boundaries with their own strict compatibility and qualification contract. Avoid unapproved production languages, embedded cryptography, SDK-wide implicit inheritance, unrestricted evaluators, and undeclared credential chains. Generated files must have one authoritative source and a reproducible check.

A passing test is not sufficient if the assertion merely restates the implementation. Security tests should prove denial before upstream retrieval, data minimization, complete activation, and meaningful failure behavior.

Maintainer account setup, public-index publishing, and operator post-installation procedures belong in `RELEASE.md`, not the product README. Do not add release credentials to examples or logs.

## Documentation and client integration

Keep architecture, user/operator, CLI/CI/MCP, SDK, enterprise, telemetry,
managed-service and distribution guides aligned to the complete v1 contract.
Record actual implementation and qualification in evidence/status records;
source review, a fixture, a schema field or a registry example is not a pass.
Validate local links/anchors, examples, schema-backed fields, actual command/tool
names and contained skill/plugin resources. Preserve the separate identities of
native runtime, language SDK, agent skill/plugin and remote service.

## Community and release readiness

Use the issue forms for synthetic bug reproductions and feature proposals, Discussions for usage questions, and the private route in `SECURITY.md` for vulnerabilities. Follow [SUPPORT.md](SUPPORT.md) and [community conduct](CODE_OF_CONDUCT.md).

The GitHub workflow runs Linux synthetic tests/race checks, vet, cross-builds and the actual source assurance gate. A successful compile is not native qualification; an incomplete coverage or analysis gate must remain failed. Use `umask 077` scoped to Unix development/test commands so fabricated private files meet the same permissions checks as runtime files. Do not relax file checks to accommodate a permissive host umask.

Release candidates and stable releases follow the version, artifact, verification and installation contract in `RELEASE.md`. Feature branches and draft PRs remain development artifacts until those requirements pass.
