# Platform support

CircularQuay's first public release line is **v1**. The first release
candidate is `v1.0.0-rc.1`; the first stable target is `v1.0.0`. An archive is
never a statement that a platform has completed native, provider, reboot, or
security qualification.

The source workflow has three distinct lanes:

| Lane | Meaning |
| --- | --- |
| Native synthetic test | Tests execute on a GitHub-hosted operating system. This is the only lane that can exercise native behavior in CI. |
| Cross-build | Both `cquay` and `cquay-audit` compile with `CGO_ENABLED=0`. It proves neither installation nor runtime behavior. |
| Protected-feature qualification | A retained, platform-specific receipt verifies filesystem protections, service behavior, provider paths, recovery and any enabled integrations. It is required before a production support claim. |

The release configuration may stage the following archive targets after the
specific release commit has passed its retained build lane. They are portable
build channels until their native and protected-feature qualification records
are published.

| Channel | Architectures | Native CI configured | Qualification boundary |
| --- | --- | --- | --- |
| Linux | 386, ARMv7, arm64, amd64, loong64, ppc64le, riscv64, s390x | Linux amd64 | Portable build channel; no universal Linux production claim. |
| macOS | amd64, arm64 | macOS amd64 | Portable build channel; arm64 needs native evidence. |
| Windows | 386, amd64, arm64 | Windows amd64 | Portable build channel; 386 and arm64 need native evidence. |
| FreeBSD | 386, amd64, arm64 | None | Portable build channel only. |
| NetBSD | 386, amd64, arm64 | None | Portable build channel only. |
| OpenBSD | 386, amd64, arm64 | None | Portable build channel only. |
| DragonFly BSD | amd64 | None | Portable build channel only. |
| illumos | amd64 | None | Portable build channel only. |
| Solaris | amd64 | None | Portable build channel only. |

Linux ARM artifacts are built with `GOARM=7`; an unspecified generic ARM build
is not substituted for that target. The portable Unix list deliberately omits
architectures that have no maintained release channel in this project, even
when a particular Go toolchain can compile them.

Android arm64 is an explicit **compile-only** target. It has no supported
installer, service manager integration, protected-storage qualification, or
mobile application package, so it is not a GitHub release archive. Android
386, amd64 and arm require external CGO linking and are excluded from the
`CGO_ENABLED=0` build lane. iOS is not a CLI distribution target.

Before publishing `v1.0.0-rc.1` or `v1.0.0`, retain the exact CI run, artifact
digests, SBOM/provenance, archive inspection and target-specific qualification
receipts. Follow [RELEASE.md](../RELEASE.md),
[qualification.md](qualification.md) and the operator documentation; do not
install an unqualified archive on a service host.
