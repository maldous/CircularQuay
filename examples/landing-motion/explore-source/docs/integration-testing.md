# Integration testing

The launch custody profile uses software KMS and age-key custody on Cloudflare
or Netlify. A protected runtime wrapping key stays outside the database, and
each profile has an independently random data key. Scoped authorization,
tenant isolation and required audit apply. Runtime or shared-key compromise
can expose reachable profile keys; software custody does not provide independent
HSM or non-exportability guarantees.

Each deployment must verify provider plans, issuer metadata, entitlements and
required custody. Additional services or upgrades require separate operator
approval. Hosting forecasts exclude labor, CRM/payment fees and customer pricing.
Synthetic frontend previews do not establish a connected backend or production
service. Implementation and qualification evidence must match the selected
release and deployment.

The complete v1 contract is tested at distinct boundaries: deterministic core
and adapter cases, disposable process/emulator fixtures, installed SDK/client
conformance, and separately authorized native/provider/recovery acceptance.
Each receipt binds its exact source, runtime, dependency locks, artifact and
scope. A passing fixture does not qualify a live account or a release.

The [SDK guide](sdks.md) defines cross-language connection/error/disclosure
contracts; [operational configuration](operational-configuration.md) requires
synthetic collector/feed, outage, cancellation and redaction cases. The
[agent-client contract](agent-integrations.md) adds client installation, catalog,
permission, reload and removal checks. [Enterprise acceptance](enterprise-integration.md#acceptance-gates)
keeps administration, durable audit and tenant isolation in that same release
gate. Use the [qualification records](qualification.md) for actual execution.

## Synthetic WireMock process profile

This opt-in suite starts a real WireMock 3.13.2 process and exercises the
production OpenBao KV v2 adapter over loopback HTTPS. It uses fabricated data
only. A normal `go test ./...` has no Java or WireMock dependency; a requested
tagged run fails if the selected JAR is absent or does not match the tracked
digest.

Prerequisites are Java 21 with `keytool`, Go 1.27.3, and a local copy of the
official [WireMock standalone JAR](https://repo.maven.apache.org/maven2/org/wiremock/wiremock-standalone/3.13.2/wiremock-standalone-3.13.2.jar).
Download it and compare both the official checksum and the tracked manifest:

```sh
curl --fail --location --output /absolute/path/wiremock-standalone-3.13.2.jar \
  https://repo.maven.apache.org/maven2/org/wiremock/wiremock-standalone/3.13.2/wiremock-standalone-3.13.2.jar
curl --fail --location \
  https://repo.maven.apache.org/maven2/org/wiremock/wiremock-standalone/3.13.2/wiremock-standalone-3.13.2.jar.sha256
shasum -a 256 /absolute/path/wiremock-standalone-3.13.2.jar
```

On Linux or macOS, run:

```sh
umask 077
CQ_WIREMOCK_JAR=/absolute/path/wiremock-standalone-3.13.2.jar \
go test -p 1 -tags=integration ./internal/integration
```

In PowerShell, run:

```powershell
$manifest = Get-Content -Raw tools/integration-tools.json | ConvertFrom-Json
$jar = 'C:\absolute\wiremock-standalone-3.13.2.jar'
Invoke-WebRequest $manifest.wiremock.jar -OutFile $jar
if ((Invoke-WebRequest $manifest.wiremock.source).Content.Split()[0] -ne $manifest.wiremock.sha256) { throw "Published digest differs from the tracked manifest" }
if ((Get-FileHash -Algorithm SHA256 $jar).Hash.ToLower() -ne $manifest.wiremock.sha256) { throw "Downloaded JAR digest mismatch" }
$env:CQ_WIREMOCK_JAR = $jar
go test -p 1 -tags=integration ./internal/integration
```

The suite creates a 0700 temporary home/state directory, a per-run PKCS#12
server certificate, and dynamic loopback-only HTTP/HTTPS ports. Its client
trusts only that certificate; it neither disables TLS verification nor changes
global trust. Startup, requests, response decoding, process reaping, and
request journals are bounded. It checks successful versioned KV v2 retrieval,
401/403 denial, malformed JSON, wrong version evidence, and policy scope
rejection before provider contact. Synthetic markers are never emitted by test
errors or CI artifacts.

GitHub Actions downloads the same pinned JAR, verifies its digest, and runs the
same command on Linux, macOS, and Windows. This is synthetic protocol evidence,
not a claim about a live OpenBao deployment or production credentials.

## User-profile KMS and hybrid validation

The [profile custody contract](managed-key-service.md#per-profile-age-and-kms-custody)
is required work; the current adapter rejects cloud-KMS recipients/key groups.
Existing age/Transit/resolver test source is evidence of established contracts,
not a cloud-KMS test pass. No new runtime tests were run under the resource hold.
When an actual adapter exists, reuse the established harness and synthetic
fixtures; provider sandbox access requires separate authorization.

| Case | Required evidence |
| --- | --- |
| Profile validation/parity | Exact provider/key/version/region/account/auth identity and scoped operations; unknown/conflicting/unsupported fields rejected consistently in CLI/API/SDK/MCP. Local age, hosted age and KMS remain distinct. Synthetic/mock frontend previews may validate only fabricated data and loopback local configuration; they do not qualify UI parity, a backend, provider access or deployment. |
| Automatic dual enrollment | Authorized enrollment provisions BOTH genuine native-backed KMS and age associations; retries/concurrency/partial failure/uncertain responses reconcile the same resources. Report ready only after both exist; enforce provider budgets/quotas and prevent orphan duplicate keys. No offline private key download or hybrid activation during association creation. |
| Launch software crypto operation | Actual maintained Web Crypto/library wrap/unwrap with protected KEK outside DB, independently random profile DEKs, unique nonce and exact server-resolved tenant/profile/purpose/version/window context; wrong context/tenant/ciphertext/tag denied. Secret storage or returning an age key cannot satisfy this crypto-service test. No config/variables/source credentials or key bytes in logs. Runtime/shared-KEK compromise remains an admitted launch risk, not a passing independent-custody test. |
| Future managed/HSM operation | Only for a separately funded selected profile, actual native SDK/provider/module, identity/key/version/context/custody and recovery evidence; mocks or the software launch do not qualify it. |
| Online-only | KMS outage, denial, missing identity, wrong key/tenant, TLS/trust, malformed output and revoked access all fail without age/snapshot/ambient-credential substitution. |
| Authorized hybrid | Qualifying availability failure restores only a complete approved encrypted snapshot within window and matching policy/scope, with permitted bindings and audit compatibility. No partial mixed generation or silent mode change. |
| Prohibited hybrid downgrade | Denial, TLS/trust failure, malformed response, expired/stale/future/mismatched snapshot, denial fence or audit requirement rejects fallback; retry/restart cannot erase a persisted denial fence. |
| Recipient and recovery semantics | Detect undeclared age/other alternative unlock recipients. Ordinary recipients are OR; only claim AND/quorum after actual group/threshold and recovery-path tests. No KMS-required online claim with an offline bypass. |
| Hybrid transition and compatible artifacts | Explicit activation verifies or generates compatible KMS/age recipient wrapping or approved snapshot, exact artifact/data-key/key-version/window binding and independent key domains where required. No assumed cross-format KMS-to-age decrypt. Online-only artifact has no offline recipient; switching back cannot recall previously retained material. |
| First run and cost | Hosted opt-outs/locks resolve before network; authorized implemented enrollment provisions both associations, but is not live setup permission here. Separately meter association creation, KMS operations and age fetches; retries reconcile logical usage while provider calls can cost more. Measure actual wraps/unwraps/rotation/audit/network/latency before pricing. |
| Residual risk | Demonstrate that authorized clients obtain data keys/plaintext and can retain them; documentation must not promise remote expiry or mandatory future KMS contact. No real key bytes in fixtures/logs. |

## Local-to-managed migration and KMS assurance evidence

The [migration contract](managed-key-service.md#end-to-end-local-to-managed-migration)
inherits the existing [assurance contract](assurance.md) and applicable existing
control IDs; it does not introduce a lower security standard. These cases are
**unexecuted requirements**, not a real backend test pass. Use synthetic fixture
keys/assets and actual pinned adapters; obtaining a provider sandbox, inspecting
operational key stores, importing/exporting real keys and running live migration
require separate scope/authority. Missing authorized backend is unqualified.

| Case | Required positive/negative evidence |
| --- | --- |
| Inventory/reconcile | Explicit selected metadata sources, public recipients, source/destination provenance and ciphertext/key versions; duplicates/aliases, unknown owner and missing key deny before migration. No unrelated key-store scan, private-key hash or config/value upload. |
| Provider/custody capabilities | Verified export/import algorithms and formats; non-exportable hardware/provider key cannot be extracted. Exercise supported source unwrap/rewrap and unsupported/missing-permission paths; a copied opaque DPAPI/provider ciphertext is not a successful migration. |
| Rewrap versus re-encrypt | Exact cleartext equivalence after authorized destination decrypt, recipient/version/domain changes and new data key where required. Removing an old recipient without rotating a retained data key is not revocation; ordinary recipient alternatives cannot masquerade as quorum. |
| Auth and replay | Cross-tenant/profile/purpose, wrong/old/revoked key versions, expired approval, duplicate/altered operation ID and revoked permission during a wait deny before unauthorized upstream access/effect. All actual interfaces share the ceiling. |
| Interrupted migration | Crash/cancel/timeouts at every state, concurrent revision, partial provider creation and uncertain response resume/reconcile the same resource/asset. No duplicate paid keys, partial-generation activation, plaintext intermediate/log leak or forged ready state. |
| Online/hybrid activation | Online-only has no age offline recipient; explicit hybrid transition verifies compatible artifact/key versions/window and complete eligible snapshot. Availability-only permitted fallback succeeds; denial/TLS/trust/malformed/expiry never downgrade. Restart/restore preserves denial fences. |
| Audit, recovery and retirement | Required audit unavailable/full/corrupt prevents protected effect. Restore original/staged/destination state with protected recovery and accepted generations; deletion/retirement only separately authorized after dependencies verified. Retained old copies remain an explicit residual risk. |
| Native integration/assurance | Record exact provider/account/region/key algorithm/runtime/artifact and existing control/test IDs. Real wrapping/unwrapping, authenticated context, scoped grants, custody/non-exportability and recovery observed separately from mocks. Apply existing issue/complexity/coverage/mutation/independent-review gates without exclusions manufactured for this feature. |
| Cost reconciliation | Count source/destination unwrap/wrap, key/rotation lifetime, duplicate provider attempts, temporary double storage, local conversion, required evidence, free owners and billing separately. Validate full dual-service cost inputs against authorized actual billing readback before a profit claim. |

The existing administration control IDs provide this planning map; each still
needs its exact enforcement/test/source/artifact/provider receipt, not a checkmark
from this document:

| Existing control | KMS/local-adapter/migration application |
| --- | --- |
| ADMIN-01 / ADMIN-02 | Independent trust/custody ceremony, actual non-exportability and immutable crypto profile; strict format/key/purpose validation without library-default downgrade. |
| ADMIN-03 / ADMIN-04 | Key/version rotation, accepted generations and freshness; exact authenticated identity/sender, replay and old-context rejection through interruption/recovery. |
| ADMIN-05 / ADMIN-06 | Source read/export, target import/wrap, activation, offline export and retirement are distinct scoped/approved effects; all actual interfaces and local file/OS boundaries enforce them. |
| ADMIN-07 | Required durable secret-free audit before effects; uncertain creation/release, interrupted migration, backups and denial fences reconcile safely. |
| ADMIN-08 / ADMIN-09 / ADMIN-10 | Customer financial authority, restricted operator operations and scoped roles remain distinct; migration/resource creation does not grant billing consent, global key disclosure or all-tenant access. |

## Hosted database security validation

These are **unexecuted integration requirements** for the
[hosted database design](credential-boundaries.md#hosted-database-protection-and-compromise-boundaries).
No deployed database, application envelope adapter or configured KEK target has
been verified. Use the existing synthetic fixture/contract harness when a real
adapter exists; do not present a crypto toy test as application protection or
start a second test framework. No live data, database migration, restore or
provider account setup is authorized by this plan.

| Case | Required evidence from the actual implementation/selected provider |
| --- | --- |
| Tenant/account boundaries | Two synthetic tenants and scoped customer/support/admin/agent identities; deny cross-tenant key reads, account history, CRM, billing metadata, list/count/join/export and async jobs. Recheck revoked/changed scope after waits and before effects. |
| D1 versus PostgreSQL | D1 query/predicate/composite-key coverage and binding scope; for PostgreSQL read back nonowner/non-superuser/NOBYPASSRLS runtime, policies and grants. Test missing context, tenant-changing writes, view/function bypass and alternating/concurrent pooled transactions with rollback/timeouts. |
| Ciphertext authenticity | Substitute tenant/profile/record/version/purpose, wrapped DEK, ciphertext, nonce and tag; fail closed before plaintext or billing success. Reject unknown formats, malformed/oversize inputs and mismatched authoritative AAD. |
| Randomness and nonce lifecycle | Review maintained crypto API use and independent random DEK generation; prove the chosen nonce strategy across retries, concurrent writes, KEK rewrap and restore. Public profile IDs/seeds/custom permutations cannot supply key entropy. |
| Custody and compromise | Database-only export cannot decrypt covered fields; account/support identity cannot access KEK or bulk unwrap. Explicitly demonstrate what the authorized custody runtime can decrypt, and record the remaining shared-KEK/runtime-compromise blast radius. |
| Rotation, backups and rollback | Restore synthetic encrypted records and historical KEKs through the actual backup path, reconcile authoritative revoked grants/key versions/meter fences and fail closed on uncertainty. Verify retirement/deletion dependencies and measure RTO/RPO; no 15-minute RPO inferred from daily backups. |
| Secret leakage and bulk abuse | Seed synthetic canary values; inspect logs, traces, edge diagnostics, errors, support exports and backups. Exercise pagination/bulk limits, alerts and denial paths without exposing key bytes or client configuration. |
| Provider qualification | Record exact account/project/region/runtime/database/plan, secret scope, production/preview separation, identity, WAF/rate limits, backup/log retention and billed usage. Separate documented capability, configured readback, contract pass and deployed evidence. |

## Payment provider validation layers

The payment provider is unselected. Reuse the existing native synthetic HTTPS
fixtures and pinned WireMock process harness for an implemented adapter; the
current OpenBao suite does not already test a payment service. Do not introduce
a new framework or provider account merely because a billing design exists.
If Stripe is selected, its [official Go or Node SDK](https://docs.stripe.com/sdks)
is a candidate according to the actual service runtime; pin a supported SDK/API
version and verify its provider-specific HTTP/test transport. This document
adds no dependency, runtime method, credential or account setup command.

| Layer | Required evidence | What it cannot establish |
| --- | --- | --- |
| Deterministic domain tests | Fake clock/provider and isolated state prove successful-fetch events, retries/independent hosts, denials, quota and durable idempotency. Include tenant/account/RBAC scopes, one-superuser invariant/recovery and financial-approval expiry. | Vendor API compatibility or a real payment/entitlement. |
| WireMock / native contract fixtures | Pinned synthetic provider request/response shapes exercise actual adapter/SDK boundary, timeouts/429/5xx, wrong credentials, bounded pagination, duplicate/out-of-order events and lost mutation replies. | A working provider account or production acceptance. |
| Cryptographic webhook fixtures | Verify signatures over exact bounded raw bytes using the selected official verifier and synthetic keys; tamper/stale/wrong-secret/account/replay cases deny, deduplicate legitimate retry and re-query authoritative provider state. No raw event/card/token logging. | An Internet-delivered authenticated event or production webhook configuration. |
| Separately authorized provider sandbox | Once a provider/test account and protected test-only credentials are explicitly authorized, run synthetic subscription/invoice/portal/decline/cancellation/reconciliation profiles under documented quotas. | Real funds, live customers, production IAM, custody, availability or financial approval. |
| Exact deployment qualification | Installed artifact, actual auth/storage/protection, residency/recovery and headless/geo/data-isolation tests bound to the authorized target. | A full production pass from mock/sandbox success alone. |

Fixtures must cover one committed successful-fetch event despite automatic
retries; independent-host accesses counted separately; no customer use for
precommit failures/denials; postcommit uncertainty reconciled once. Exercise
quota boundary/concurrency, plan/subscription states, no paid redirect flag
shortcut, no duplicate invoice/charge/upgrade on timeout, and failed/expired
financial authority or missing required confirmation. A webhook is not permission
to create a key grant. Stale events cannot restore a newer revoked entitlement.

For a Stripe candidate, [signature handling](https://docs.stripe.com/events/manage-webhook-endpoints)
uses exact body bytes, the enrolled endpoint secret and appropriate time bounds;
JSON reserialization before verification is not equivalent. Use stable event
IDs and a durable processing/reconciliation record for duplicates and order,
not an assumption that delivery is once or chronological. Vendor idempotency
version/retention limits do not replace the service's durable intent.

[Sandbox testing](https://docs.stripe.com/testing) uses test values and does not
move funds; it is not authorized merely by this design request. Do not use live
cards, live customer creation, real charges, payment/account configuration or
unrelated keys. Do not load-test a vendor sandbox as if it were the local mock.
Generate only synthetic markers, exclude raw events/payment data/config from
artifacts, and keep provider connections off by default. Source/contract/mock/
sandbox/deployment evidence remains separately labeled in the existing
qualification/handoff records; no billing test has run from this documentation
change. BAU resource holds still apply.
