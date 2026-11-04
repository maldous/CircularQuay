# Supply-chain integration contract

This document defines the v1 contract for dependency review and future release
evidence. It does not say that a release has been produced, signed, attested,
or qualified. Actual results belong in a release's evidence bundle and the
[qualification matrix](../qualification.md).

## Pull-request dependency review

The [Dependency review](../../.github/workflows/dependency-review.yml) workflow
runs on pull requests into `develop`, `main`, and release branches. It uses
GitHub's official `actions/dependency-review-action` at the complete release
commit `a1d282b36b6f3519aa1f3fc636f609c47dddb294` (v5.0.0), rather than its
mutable `v5` major-version branch. The upstream v5.0.0 release moved to the
Node 24 runtime and requires an Actions runner at least v2.327.1. Its release
commit is GitHub-verified; that upstream signature is useful provenance, not a
substitute for reviewing the action source and pinned bytes.

The job uses the ordinary `pull_request` event and the minimum `contents: read`
token. It does not use `pull_request_target`, secrets, a write token, a
self-hosted runner, or a checkout step. The action compares GitHub dependency
metadata; it therefore need not run pull-request code merely to determine the
dependency delta. GitHub documents dependency review as available for public
repositories. Its availability for a private repository depends on the
repository's GitHub Code Security or GitHub Advanced Security entitlement.
That entitlement has not been asserted here.

The pinned action's default is to fail when the pull request introduces a known
vulnerable dependency. This is an introduced-dependency gate, not a statement
that every component in the repository is vulnerability-free. The repository
has not declared a severity threshold, license allow-list, license deny-list,
or advisory exception process. Consequently this workflow supplies none: it
does not invent a license prohibition, silence an advisory, or treat an unknown
license as approved. A later, reviewed policy may add an explicit scoped
vulnerability threshold or license rule with its rationale in source control.

The action is deliberately separate from Dependabot. Dependabot proposes
version updates; dependency review examines the dependency delta in every pull
request, including a human-authored change. GitHub's dependency graph must stay
available for this integration to be useful.

## Distinct evidence, distinct questions

| Evidence | Question it answers | Does not establish |
| --- | --- | --- |
| `govulncheck` in the Go security workflow | Does a vulnerability affect code reachable from the current resolved module graph? | That a newly added but not-yet-reachable dependency is acceptable, or that non-Go shipped components are safe. |
| Dependency review | Does this pull request introduce a dependency with a known advisory? | Reachability, runtime behavior, source integrity, or an artifact inventory. |
| Release SBOM | Which declared components and external executables were shipped in this exact artifact? | That the artifact was built by the claimed workflow, is free of vulnerabilities, or will execute only those components. |
| Release provenance/attestation | Which repository, workflow, commit and build context produced an artifact? | That the source was secure, the release was independently tested, or a downloaded artifact is safe without verification. |

`govulncheck`, dependency review, an SBOM, and provenance are complementary.
None replaces the first-party static, dynamic, provider, native-recovery, or
independent-review requirements in the assurance contract.

## Candidate and stable release verification contract

Release candidates use immutable tags such as `v1.0.0-rc.1`; stable releases
use their own immutable `v1.0.0` tag. A stable release is built from its own
selected stable tag, never by renaming or retagging a candidate. Before either
is published, the release owner must record the exact tag, commit, workflow
run, archive digests, SBOM digest, provenance/attestation identity, qualified
platform matrix, and any remaining qualification limits.

For a future candidate or stable release, a verifier must obtain the selected
asset and its checksum manifest through an authenticated release route, verify
the named asset's SHA-256, inspect archive paths before extraction, and verify
the expected repository and release-artifact attestation identity. For example,
where a qualified release has published a GitHub attestation and a compatible
GitHub CLI is available:

```sh
tag=v1.0.0-rc.1 # Replace only with an existing selected release tag.
asset=cquay_1.0.0-rc.1_linux_amd64.tar.gz
checksums=cquay_1.0.0-rc.1_checksums.txt

gh release download "$tag" --repo maldous/CircularQuay \
  --pattern "$asset" --pattern "$checksums"
gh attestation verify "$asset" --repo maldous/CircularQuay
sha256sum --ignore-missing --check "$checksums"
tar -tzf "$asset"
```

The selected repository identity, exact asset filename and expected tag/commit
must be checked against the release record before extraction. A checksum fetched
beside an unauthenticated binary is only an integrity comparison, not publisher
authentication. `gh attestation verify` validates the attestation route; it is
not a substitute for checking the selected repository, release notes, platform
qualification, archive contents, or checksum manifest. Windows and macOS
operators use equivalent qualified platform hash and archive-inspection tools.

Stable promotion repeats the complete candidate verification for the distinct
stable artifact set, then records the candidate-to-stable commit comparison and
all native/provider/recovery/independent-review evidence required by
[RELEASE.md](../../RELEASE.md). An unchanged source tree does not excuse a
missing stable artifact attestation, SBOM, archive inspection, or native
qualification.

## Future attestation producer

No attestation producer is implemented by this integration. A future trusted
release workflow may generate a build-provenance attestation and an SBOM
attestation only after it has built the selected immutable tag's release assets.
GitHub documents `contents: read`, `id-token: write`, and `attestations: write`
as the minimum permissions for that producer. Those permissions belong only to
the protected release job; they must not be present in fork pull-request jobs
or ordinary test workflows. The proposed producer must pin every action to a
full commit, bind each attestation to an actual release asset and generated
SBOM, publish the verification instructions with the release, and receive a
separate review before it is treated as a required gate.

GitHub's artifact-attestation service uses Sigstore. For public repositories,
the resulting bundle is also recorded in the public transparency log; private
repository attestation behavior differs. An attestation is provenance evidence,
not a malware scan or a production qualification certificate. Offline
verification needs a documented verifier, trust-root/bootstrap route, the
artifact, and its published verification bundle; it cannot be assumed merely
because a release has a checksum file.

## References

- [GitHub dependency review](https://docs.github.com/en/code-security/concepts/supply-chain-security/dependency-review)
- [Dependency review action](https://github.com/actions/dependency-review-action)
- [Dependency review action v5.0.0 release commit](https://github.com/actions/dependency-review-action/commit/a1d282b36b6f3519aa1f3fc636f609c47dddb294)
- [GitHub artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations)
- [Generating and verifying GitHub artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)
- [Release and operator procedure](../../RELEASE.md)
