# Development session handoff

Status: CircularQuay v1 development is active and **not production qualified**.
No release, package-index publication, production deployment, system/service
installation, credential migration, operational infrastructure change, or
live-provider qualification has occurred.
Do not claim a passed gate without its retained evidence.

## Resumption and retained scope

The user resumed CircularQuay delivery on 4 November 2026. The earlier pause
for BAU priority is revoked for this isolated project. Preserve the saved
source/security patch and existing task ownership. No BAU service,
operational credential, system/service installation, production deployment,
paid purchase, main merge, or release tag is authorized by the resumption.

The expanded v1 scope is published in the README and managed-service contract:
one-command session bootstrap, verified repository/tree configuration,
authenticated enterprise discovery, Azure Pipelines/GitHub Actions/GitLab CI
workload identity, complete overrides/CLI CRUD, optional MCP management and
delegation, and managed identity-to-age-key access with authorized UI/API.
The README and product/design/operator documents describe the final North Star;
actual implementation and qualification belong in status/evidence records.

The preserved combined patch still exactly matches the Go/schema/module
manifests in `.local/qualification/checkpoint5-20261104/`. Its whole-repository
normal/race tests, vet, Staticcheck, formatting, module tidy/verification,
cyclomatic/workflow checks and affected Windows compile-only checks passed.
No source changed during the documentation-only continuations. Native CI,
exact-candidate Sonar and a fresh independent managed audit must assess the
published candidate. Keep the unchanged quality gates and prior failed evidence.

## Latest bounded documentation result

The documentation-only commit `1d89ffbf65a03e4e6ef8f146944416b97356e38c`
is published and byte-verified on `feature/initial-implementation`. It adds the
standalone [managed key service](managed-key-service.md), a concise README link,
and architecture/operator/MCP/recovery references. The service uses an existing
trusted identity provider and explicit key mappings; scoped UI/API administration
manages allowed principals, grants, windows, changes, and revocation. Existing
paid Netlify/Cloudflare capabilities take preference over a custom UI; any missing
key-specific UI is hosted there. There is no email/onboarding suite. Distribution
cannot revoke already copied private keys; local custody and remote unlocking
remain explicitly distinct models. Financial scenarios are assumptions, not
chosen prices, benchmarks, traction, or deployment approval.

The README repository/tree bootstrap and CI/discovery scenario was already
published at `7b68c6116786779392e01024c102093c33254a0d`. The bounded product
presentation is a final v1 North Star, while actual managed-service status is
separate in [qualification](qualification.md#managed-service-design-status).
The updated user-supplied UIF diagram presents the same architecture without
promoting its old operational snapshot into a current BAU claim. Its artifact
and visual/structural receipts are retained in the private continuation record.
No implementation, service, credential, paid subscription, or BAU deployment
was changed. All existing uncommitted source was preserved during that documentation-only
turn. The later user resumption above reactivates isolated development and
qualification only.

## Start here

- Project/module: `CircularQuay` / `github.com/maldous/CircularQuay`. The app,
  binary, release-asset prefix, and skill are `cquay` ("c-key");
  `cquay-audit` is separate.
- Development branch: `feature/initial-implementation`. This continuation
  includes `41ea71365464417875c7162cc98c87262cda47f8`; use `git rev-parse HEAD`
  and `git status` to establish the actual checkpoint and subsequent changes.
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

Grant time windows and the six reviewed remediation patches are implemented
with focused synthetic tests. They are not qualified capabilities until fresh
combined, full, native, and Sonar checks pass.

README ordering was directly revised by the user and remains preserved.
The Android rendering investigation found the raw Markdown, fences, HTML, and
linked SVGs clean; browser rendering worked. Treat that as an app/cache issue
unless new evidence reproduces a source-side rendering defect. Do not reorder
the README without a user-directed reason.

## Recent local evidence

The retained frozen-source receipts are under
`.local/qualification/checkpoint5-20261104/`. Before/after Go, module and schema
hash manifests match. These are local evidence for this checkpoint's source,
not release evidence.
Using a fresh owner-only temporary directory under root-owned sticky `/tmp`,
the frozen checkpoint's normal tests, race tests, vet, gofmt, Staticcheck
2026.2.2, `gocyclo -over 10`, actionlint, and module verification passed.
The same checks correctly reject a `.local` temporary directory with an unsafe
`0775` ancestor; do not relax that protection to make tests pass.

After source changes, run the applicable checks from protected temporary
storage and retain exact command, tool version, revision/tree state, and result.
The source-profile 100% owned-code coverage
gate remains unchanged and is not met by the last published CI receipt.

## Qualification and release blockers

The Compute Engine task for published revision
`41ea71365464417875c7162cc98c87262cda47f8` (tree
`d6bfeedf511434a9d09afef931c6109a295e19c7`) succeeded with analysis ID
`06a03452-7d22-4590-9c47-8dfdb6f0d455`, but its quality gate is ERROR: one
`go:S3776` cognitive-complexity smell at `internal/cli/init_test.go` line 40
(cognitive complexity 16; allowed 10), 75.5% overall coverage, 78.7% new-line
coverage, zero reported bugs/vulnerabilities/hotspots/duplication, and one
analysis-token expiry warning. The current patch requires an exact-candidate
Sonar recheck.
Sonar, full unchanged coverage, live providers, native reboot/recovery,
installers, registry publication, and independent audit remain incomplete.
The independent managed audit at the published checkpoint confirmed six
findings: three high and three medium. The high findings cover Windows path
ownership/ancestor trust and environment bootstrap aliases/field narrowing; the
medium findings cover HTTP authorization expiry, cloud-SDK transport controls,
and an empty Infisical path scope. Remediations for all six findings, plus
grant time windows, are implemented with focused synthetic tests; fresh
combined/full/native/Sonar qualification is pending. This static-only audit did
not qualify native behavior, live providers, installations, or services. Keep
detailed evidence in the owner-held security channel rather than reproducing
sensitive attack paths in public documentation.

GitHub source run 41660198848 at published revision
`41ea71365464417875c7162cc98c87262cda47f8` failed Linux job 126002042064 at
`go mod tidy` before tests because a new test's SSH import left a metadata-only
module diff. The preserved patch contains the resulting `go.mod`/`go.sum`
repair; its local tidy-diff check passed and fresh CI is required. Its coverage job 126002042348 failed the
unchanged 100% gate at 3,967/4,860 statements (81.6255%). macOS job
126002042623 completed `go test ./...`, and all 26 cross-builds succeeded.
Windows job 126002042419 failed only platform
`TestFailedCreationCleanupPreservesAReplacement` and a signed-repository Git
commit exit 128; its other packages, including the new Windows CreateDACL tests,
passed. The cleanup-fixture correction is a newer working-tree change that
needs a native rerun; the Git failure is not fixed or diagnosed. Security run
41660198518 completed CodeQL build/analyze/upload and passed
Staticcheck/govulncheck, but gosec job 126001310144 failed with 11 unsuppressed
findings. All three G304 findings are resolved in this revision.

The snapshot at revision `960` contains 25 native archives plus one source
archive (26 total) with exact checksum/provenance evidence; its Linux smoke is
only a smoke test, not production qualification. Cross-build/package work is
build evidence only.
Native Windows/macOS/Linux runtime, restart/recovery, installer, signed
artifact, anonymous download, SBOM/provenance, and registry qualification still
require real evidence. The
signed repository declaration is not a claim that `sync` can fetch/deploy it
until its integration, quota, recovery, and adverse-path qualification are
complete.

GitHub protections currently have four required GitHub Actions checks bound;
code scanning is configured as `Any` / `Any`, tag rules are active, and the
advanced CodeQL source workflow has completed after the user-approved switch
from default setup. Verify settings by read-back after any administrative change. Earlier
administrative writes returned HTTP 403; record permission gaps rather than
bypassing them.

The current continuation adds exclusive protected file creation for `init` and
Sonar test credentials: it validates the parent, anchors creation through
`os.Root`, refuses existing names, applies native protection before writing,
and syncs the file and directory. Audit coverage is opened through the protected
file boundary. Concurrent-creator, no-clobber, cleanup-identity, usable-init,
and Windows DACL tests cover the new behavior; native rechecks remain required.
Initialization writes a synthetic fixture and policy separately; a failure
between those creations can leave the synthetic fixture and is not reported
as successful initialization.

The native fixture corrections use JSON encoding for Windows paths, canonical
temporary roots for macOS, and in-memory synthetic signing-key generation while
retaining real Git signing/verification. Case-collision decisions have a pure
cross-platform oracle plus filesystem integration where distinct names are
representable. The Unix executable-alias test now creates the alias rather
than assuming a `/bin/true` layout. Native macOS CI prefers the protected OS
Git/OpenSSH paths; a rejected helper reports only sanitized file metadata.

The current continuation also hardens native policy and literal-source reads,
adds schema/Parse parity regressions and cloud/CLI/doctor negative tests, repairs
relative/default-path config replacement, and corrects native test fixtures.
Windows directory ACL inheritance preserves the private allowed principals
for ordinary child creation; these tests passed in the native Windows package
run at revision `960`. The new exclusive-create DACL tests require a fresh run. The bounded Go engineering review found the config-repair defect, which
failed both regression cases before the fix and passed afterward. This is a
scoped review, not the independent managed security recheck or release audit.

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
