# Development session handoff

Status: CircularQuay v1 development is active and **not production qualified**.
No release, package-index publication, production deployment, system/service
installation, credential migration, operational infrastructure change, or
live-provider qualification has occurred.
Do not claim a passed gate without its retained evidence.

## Start here

- Project/module: `CircularQuay` / `github.com/maldous/CircularQuay`. The app,
  binary, release-asset prefix, and skill are `cquay` ("c-key");
  `cquay-audit` is separate.
- Development branch: `feature/initial-implementation`. This implementation
  checkpoint builds on README-only commit `0023fef`; use `git rev-parse HEAD`
  and `git status` to establish the actual revision and subsequent changes.
- Read [CONTRIBUTING.md](../CONTRIBUTING.md), [SECURITY.md](../SECURITY.md),
  the durable [implementation plan](implementation-plan.md), and
  [RELEASE.md](../RELEASE.md) before editing. The plan is the scope contract;
  this file records current evidence and continuation risk.
- Use Gitflow: `main` is released/stable, `develop` integrates, `feature/*`
  develops, `release/*` stabilizes, and `hotfix/*` starts from `main`. Do not
  merge to `main`, tag, publish, or deploy until the required gates genuinely
  pass.

## Isolation and authorization

Work only in an isolated developer checkout with synthetic fixtures. Do not
install CircularQuay into service or system paths, restart services, alter
infrastructure or existing credentials, inspect operational secret values or
shell-profile credentials, or interfere with OffMenu BAU recovery. Do not
create broad persistent credentials to bypass an unavailable GitHub permission.

The user authorized reversible development and free test tooling, not paid
resources, production deployments, or a universal credential daemon. Keep
publishing, account, registry, and operator procedures in `RELEASE.md`; keep
end-user/product material in `README.md`.

## Current implementation checkpoint

This checkpoint includes JSON policy CRUD and safe doctor
formatting; protected metadata/state and path-ownership hardening; strict
source, snapshot, target, grant, and repository contracts; repository quota
work; provider contract tests; and extended release cross-build configuration.
The expected policy/CLI contract is version 1 and the development version is
`1.0.0-dev`.

The CLI supports only policy operations `list`, `status`, `run`, and `read`.
`run` authorizes the enrolled workload; read keys restrict only `read`.
`cancel` is request-context cancellation, not a policy permission. Configuration
editing is JSON-only, protected, bounded, strict-decoded, validated, and
atomically replaced under a local lock plus digest recheck. YAML remains a
loadable policy format but is intentionally not mutated by the editor.

The v1 identity direction does not add a Keycloak dependency. Where a custom
API is supported, trust an existing authorization server/IdP directly (for
example Entra ID or Auth0) through named trust/configuration scope, role,
client, subject, resource, context, and grant declarations. Do not hard-code
development/production identity assumptions. DPoP, mTLS, private-key JWT, and
offline-artifact binding are not implemented or qualified.

README ordering was directly revised by the user and preserved in `0023fef`.
The Android rendering investigation found the raw Markdown, fences, HTML, and
linked SVGs clean; browser rendering worked. Treat that as an app/cache issue
unless new evidence reproduces a source-side rendering defect. Do not reorder
the README without a user-directed reason.

## Recent local evidence

The retained fresh-worktree receipts are under
`.local/qualification/fresh-current-tree-20261104/`. They are local evidence
for the source captured by this checkpoint, not release evidence for `0023fef`.
Using a fresh owner-only temporary directory under root-owned sticky `/tmp`,
the latest normal tests,
race tests, vet, gofmt, Staticcheck 2026.2.2, and `gocyclo -over 10` passed.
The same checks correctly reject a `.local` temporary directory with an unsafe
`0775` ancestor; do not relax that protection to make tests pass.

After source changes, run the applicable checks from protected temporary
storage and retain exact command, tool version, revision/tree state, and result.
The source-profile 100% owned-code coverage
gate remains unchanged and is not met by the last published CI receipt.

## Qualification and release blockers

The latest recorded SonarQube analysis of `24d8401` failed: two issues,
72.1% overall coverage, and 76.7% new-line coverage. The recorded source
coverage is 78.3714% (3,051/3,893 statements). It reported zero bugs and
vulnerabilities, but that does not pass the quality gate. This checkpoint
contains fixes for five managed findings, pending a fresh recheck. Sonar, full
unchanged coverage, live providers,
native reboot/recovery, installers, registry publication, and independent audit
remain incomplete.

Cross-build/package work is build evidence only. Native Windows/macOS/Linux
runtime, restart/recovery, installer, signed artifact, anonymous download,
SBOM/provenance, and registry qualification still require real evidence. The
signed repository declaration is not a claim that `sync` can fetch/deploy it
until its integration, quota, recovery, and adverse-path qualification are
complete.

GitHub protections currently have four required GitHub Actions checks bound;
code scanning is configured as `Any` / `Any`, tag rules are active, and the
advanced CodeQL source workflow accompanies this checkpoint while default
setup is still active; complete and verify that transition before claiming an
advanced scan. Verify settings by read-back after any administrative change. Earlier
administrative writes returned HTTP 403; record permission gaps rather than
bypassing them.

## Priority continuation

1. Inspect `git status`; preserve concurrent work and do not overwrite others.
   Format and run bounded tests with private temporary storage before changing
   behavior.
2. Integrate and review the current policy/CLI/provider/repository/platform
   changes. Keep strict validation and protected-file semantics; add focused
   adverse-path tests where contract behavior changed.
3. Re-run SonarQube against the exact candidate revision, resolve every owned
   issue and coverage gap without exclusions or weakened rules, and retain the
   quality-gate receipt.
4. Complete unresolved sync/application/activation/recovery behavior and its
   strict snapshot/repository validation; do not infer health from an attempted
   activation or a source timestamp.
5. Perform qualified native, provider, installer, release, registry, and
   independent-security evidence. Only then consider a `release/1.0.0` branch,
   `v1.0.0-rc.1`, and ultimately `v1.0.0` under the gates in `RELEASE.md`.

## Do not regress these boundaries

- No inline secret values, provider bootstrap tokens, private keys, or browser
  sessions in policy, fixtures, logs, screenshots, or documentation.
- No silent offline fallback after denial, trust/TLS failure, malformed input,
  expiry, or policy mismatch.
- No arbitrary command/provider URI through MCP; raw reads require an exact
  explicit grant and are deliberate disclosure.
- No quality-gate suppression, fabricated green badge, unconditional release
  claim, or broad credential created to evade a missing permission.
- No production assertion from compilation, cross-build, static analysis, or a
  synthetic fixture alone.
