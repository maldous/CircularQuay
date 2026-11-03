# Development session handoff

Status: development checkpoint, NOT production qualified. No release, package index publication, infrastructure deployment, Foo installation, real credential migration, or live provider qualification has occurred.

## Repository and workflow

- Owner/repository: maldous/CircularQuay; Go module github.com/maldous/CircularQuay.
- Main CLI cquay; standalone audit CLI cquay-audit; Apache-2.0.
- Gitflow: main is stable/released, develop integration, feature/* work, release/* stabilization, hotfix/* from main with back-merges. Current work is feature/initial-implementation. Do not merge this checkpoint to main or tag a release.
- User requires professional consistent paths/names, gofmt, human-readable functions and GoDoc/security rationale. Human developers will maintain this project.
- GitHub repository security settings have NOT yet been applied/verified. Owner explicitly authorized hardening. Use available free protections; document admin permission/entitlement gaps rather than claiming success.

## Authorization and isolation

The user authorized implementation and free interoperability test accounts; no paid resources/trials/recurring costs. The user initially required cloud-only work and no installation on Foo. A local worker on Foo is now an offered handoff option, not authorization to install into services, alter infrastructure, read operational secret values, migrate credentials, restart services, or interfere with active OffMenu BAU recovery. If using Foo, use an isolated developer checkout and synthetic fixtures only. Do not inspect inherited shell profiles or environment values. No need to ask again to finish already authorized reversible development.

## Implemented source (verification incomplete)

- Strict policy/schema and exact authorization; offline/online/hybrid workload model; SOPS subprocess adapter with age or pinned OpenBao Transit metadata.
- Lifecycle-aware complete generations and explicit encrypted offline snapshots; metadata-only state and native locks.
- Provider adapters: local JSON/literal dotenv, OpenBao/Vault KV2, Doppler, Infisical, Akeyless, official AWS/Azure/GCP SDKs, fixed Bitwarden/1Password CLI operations. No 1Password Environments support.
- Publication to Cloudflare Workers/Netlify/Windmill as one JSON bundle; native file/systemd/Windows delivery and declared activation helpers.
- Optional official-SDK MCP stdio and HTTP, kernel UID/SID local identity, granular grants, single-key disclosure, registered actions, metadata audit receipts, cancellation/revocation checks.
- Standalone audit runner, Sonar bootstrap/parser/profile/config, formal Delivery model, README architecture images, operator and release docs, platform templates/examples.
- Signed Git data repository package exists but is experimental, untested and not integrated in CLI resolution/sync.

## Last observed verification (do not infer final-tree qualification)

Earlier full compilation/test run compiled all packages; two MCP fixture failures were subsequently fixed. Policy tests passed before additional repository/activation branches (old 100% coverage no longer applies). Provider/source/delivery tests passed before the final interrupted helper-refactor operation; latest transport rerun reported PASS. Latest MCP command passed race tests at 91.2% statement coverage. Audit package/CLI tests and audit CLI build passed. Earlier Windows crosscompile predates new activation/ACL additions. Final root tests added after the earlier global run are unverified. No actual Sonar server scan, TLC model check, independent audit, native Windows/systemd reboot test or live cloud-provider test occurred.

Cloud-only upstream test tools were verified: Go 1.27.3 /workspace/.tools/go/bin/go (system /usr/bin/go is a board game); SOPS 3.13.3 /workspace/.tools/bin/sops verified exact Sigstore release identity + signed checksum; age 1.3.2 /workspace/.tools/bin/age-keygen verified official Sigsum proof. Tool binaries/identities are not source assets and are not committed.

## Required immediate integration work

1. Inspect the checkpoint, run gofmt on Go source, go mod tidy, then GOMAXPROCS=2 go test -p 1 ./... . Preserve actual results; no fake passing qualification. Cloud runtime repeatedly disconnected; all agents paused before this commit.
2. internal/state/store.go still has Unix-only directory/file checks and direct OS writes. Reuse platform.EnsurePrivateDirectory/ReadProtected/WriteProtected for Windows ACLs, reject symlink lockfiles and trailing/malformed state JSON. Verify newly added state tests on each platform.
3. internal/cli/commands.go currently holds the workload lock across the entire child run: release it after resolving/recording generation and before a long-running child. doctor must use mcp.LocalPrincipal(), not os/user or USER. Refactor into readable resolution/application/activation helpers.
4. sync is incomplete: add --once compatibility, explicitly fetch declared signed repo, apply mapped targets and native activation safely. Track resolved/applied/activated/pending-activation states separately; never report healthy without a real probe. Do not skip changes based solely on source mtime/size or absent revisions. Do not mark activation success until it actually succeeds; retry failed activation.
5. Resolver needs strict decrypted snapshot JSON decoding, exact payload structure and scope checks, cancellation checks, safe version metadata bounds, and preserved denial fence. Only Unavailable may use an explicitly permitted hybrid static snapshot; denial/TLS/trust/expiry must never trigger fallback. AuditRequired forbids offline fallback. SOPS errors currently conservatively classify Denied; automatic Transit-outage fallback is not qualified.
6. Signed repository package must be hardened before use: load TokenFile only for authenticated fetch (Current must work without it); reuse platform native protected I/O/ACL/durable accepted pointer; initialize SHA256 repos for 64hex pins or reject; reject dot/dotdot path segments and all case-fold ancestor/file collisions; add fake-Git fault matrix and real ephemeral SSH-signed fixtures. Two-minute timeout does not bound pack disk use: require operational quota. It compiled with no tests; do not advertise complete secure sync.
7. Native platform additions need final compilation/testing. Add ctx cancellation check before activation, especially Windows SCM stop. Inspect interrupted patch that added real SOPS test/platform tests. Verify platform.ValidateExecutable and native DACL checks. Run opt-in real SOPS age roundtrip with CQUAY_TEST_SOPS and CQUAY_TEST_AGE_KEYGEN only in isolated dev workspace.
8. Inspect whether provider cloud.go helper extraction (awsSecretValue/awsParameterValue/azureSecretValue/gcpSecretValue) landed. Run source/deliver/transport tests. Add official SDK success-path fixtures and adverse paths. Azure only explicit VM IMDS or workload identity; AWS only explicit EC2/ECS with STS binding; GCP exact SA email. No live credential qualification.
9. MCP strict profile requires signed typ=at+jwt, issuer/audience/exp/sub/client_id/iat/jti, RS256 or ES256; azp may only agree. cquay:mcp plus operation scopes. Generic ID tokens/opaque tokens unsupported; this is not an authorization server/login/DCR service. Trust configuration changes require restart; grants reload live. Local logs are durable metadata, not centralized tamper-proof evidence. 91.2% coverage does not meet required release threshold.
10. Complete assurance packaging: CI workflows, installers, GoReleaser, container/MCP/MCPB assets, distribution instructions in RELEASE.md (not README). Apply complexity <=10 profile through Sonar API, run actual Sonar scanner and fix findings. Do not weaken tests/coverage/suppressions to fabricate clean output. 100% coverable owned-code target and zero owned issues/warnings remain unmet. Keep upstream dependency advisories transparent. No Sonar server started.

## Required documentation corrections

Update provider/operator docs with exact native identity restrictions; JSON/env mtime+size are observation hints only; Infisical US default/no imports/reference expansion; 1Password Environments unsupported. Native publishers write one JSON bundle (CQUAY_CONFIG default), not independent injected keys. Netlify functions-only granular scopes may need Pro; never broaden exposure automatically. Include exact target fields and dedicated target/context ownership. Align CLI flags/examples and windows target name. Finish MCP strict-profile docs; RELEASE exact sonar-init/compose/status/sonar-bootstrap/audit path, release/hotfix Gitflow, account/index/registry/operator postinstall instructions and uncompleted-activity ledger. Validate Markdown local links, JSON/XML/SVG, demos and diagrams.

## GitHub security work

Inspect actual API/app permissions first. Add/review CODEOWNERS, issue/PR templates, Dependabot for Go and Actions, pinned-action CI with read-only default token and no pull_request_target execution of untrusted code. Configure available free branch/ruleset protection for main/develop/release/tags, no force/deletion, PR/check requirements, review dismissal and resolution; record solo-maintainer review tradeoffs. Enable available dependency graph, vulnerability alerts/updates, code/secret scanning and push protection, private vulnerability reporting, minimal Actions permissions and approved actions policy. Use release-only scoped OIDC for attestations/signing and verified provenance/SBOM/checksums. Do not create long-lived publish tokens unless necessary/authorized; do not alter account-wide MFA settings. Settings currently unverified; ensure admin denials are reported.

## Next-session prompt

Continue maldous/CircularQuay from feature/initial-implementation. Read this handoff, CONTRIBUTING.md, SECURITY.md, docs/implementation-plan.md and RELEASE.md before editing. This is an incomplete development checkpoint, not a qualified release. Finish authorized implementation and tests using Gitflow and human-maintainable Go. Keep development isolated; do not install into Foo services or touch OffMenu recovery, existing credentials, shell profiles or infrastructure. Use synthetic fixtures; do not disclose secrets. First inspect git status and run formatting plus bounded tests, fix minimum build/integration issues above, then complete root integration and rigorous qualification. Configure GitHub security where permissions allow; explicitly document unavailable controls. No release/main merge or production assurance claim until real required gates pass. Distribution/account instructions belong in RELEASE.md, feature documentation in README. Provide actual evidence, remaining limitations and precise next steps.
