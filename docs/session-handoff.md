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

The checkpoint6 source batch has identical before/after source manifests in
`.local/qualification/checkpoint6-20261104/`. Its whole-repository normal/race
tests, vet, Staticcheck, formatting, module tidy/verification, cyclomatic and
cognitive-complexity/workflow checks, and 12 affected Windows compile-only
checks passed. No source changed during those checks. Native CI,
exact-candidate Sonar, and a fresh independent managed audit must assess a
committed candidate. Keep the unchanged quality gates and prior failed evidence.

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
  includes `4886fc8660706909376f3543ad7cab186d9256b9`; use `git rev-parse HEAD`
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

The checkpoint6 source batch adds named RFC 9068 and Entra v2 API authorities;
configurable all-of scope and roles; exact grants and current-trust
fingerprints; a strict shared legacy verifier; immutable GitHub bootstrap only
(full SHA plus trusted SHA-256); signed-repository CLI/MCP runtime with
`FetchCurrent` offline/fallback guards and hardened Git hashing/fsck; strict
snapshot validation and full-snapshot MCP reads; durable fallback pre-read
fencing across restart; and the Azure workload-identity transport constructor
test. It also includes Windows fixture repairs. Branch discovery, `flow`, and
an authentication UI remain unimplemented. This batch remains unqualified until
a combined candidate passes fresh native, Sonar, and independent checks.

README ordering was directly revised by the user and remains preserved.
The Android rendering investigation found the raw Markdown, fences, HTML, and
linked SVGs clean; browser rendering worked. Treat that as an app/cache issue
unless new evidence reproduces a source-side rendering defect. Do not reorder
the README without a user-directed reason.

## Recent local evidence

The retained frozen-source receipts are under
`.local/qualification/checkpoint6-20261104/`. The before/after source manifests
are identical. These are local evidence for this captured source batch, not release
evidence.
Using a fresh owner-only temporary directory under root-owned sticky `/tmp`,
Go 1.27.3 normal and race tests, vet, gofmt, `go mod tidy -diff`, `go mod
verify`, Staticcheck 2026.2.2, `gocyclo -over 10`, exact-recursion `gocognit
-over 10`, and actionlint passed. Compile-only Windows checks passed for the 12
affected packages. The local statement coverage is 4,787/5,742 (83.3682%), so
the unchanged 100% gate still fails.
The copied 213-file nonignored source scan found zero gitleaks findings. Its
binary SHA-256 is retained with the receipt, but its version output is the
build-process default; this is bounded local evidence only. An initial incorrect
CLI invocation is retained as evidence history and is not a code failure.
The `internal/identity` package separately passed normal/race/vet and both
complexity checks with 183/183 statements covered; that package receipt does
not qualify the repository as a whole.
The same checks correctly reject a `.local` temporary directory with an unsafe
`0775` ancestor; do not relax that protection to make tests pass.

After source changes, run the applicable checks from protected temporary
storage and retain exact command, tool version, revision/tree state, and result.
The source-profile 100% owned-code coverage
gate remains unchanged and is not met by the last published CI receipt.

## Qualification and release blockers

The Compute Engine task for revision
`4886fc8660706909376f3543ad7cab186d9256b9` (tree
`132f80e52ead1f61b08239fec97ad440829d8172`) succeeded as task
`7dbc9ce0-8aa7-4df5-81e8-5c7cfdba6b71` with analysis ID
`9ea50dd4-ab95-4a3b-aff7-d1510405f708`, but its quality gate is ERROR: 75.7%
overall coverage, 78.8% new-line coverage, zero reported bugs/vulnerabilities/
hotspots/duplication, and one critical MCP-action-literal code smell. The
working-tree fix requires a fresh exact-candidate Sonar recheck.
Sonar, full unchanged coverage, live providers, native reboot/recovery,
installers, registry publication, and independent audit remain incomplete.
The managed static recheck at this checkpoint reported one medium Azure
workload-identity transport finding. A new cloud test accompanies the local
fix, but an independent recheck is pending. This static-only review does not
qualify native behavior, live providers, installations, or services. Keep
detailed evidence in the owner-held security channel rather than reproducing
sensitive attack paths in public documentation.

GitHub source run 41677515031 at revision
`4886fc8660706909376f3543ad7cab186d9256b9` passed native Linux and macOS and
all 26 cross-builds. Its Windows job 126061996996 failed two fixtures; local
fixture repairs require a fresh native Windows run and are not qualified by this
receipt. The source workflow's Sonar-style coverage was 4,072/4,969 statements
(81.9481%); the retained local Sonar profile measured 4,071/4,969 (81.928%).
Both fail the unchanged 100% gate. Security run 41677515226 passed CodeQL but
failed gosec with 11 unsuppressed findings. Earlier `41ea` evidence remains in
the qualification matrix as historical evidence. Retained native/cross and
Sonar artifact checksum manifests bind these receipts to the recorded candidate.

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
