# Integration testing

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
CQUAY_WIREMOCK_JAR=/absolute/path/wiremock-standalone-3.13.2.jar \
go test -p 1 -tags=integration ./internal/integration
```

In PowerShell, run:

```powershell
$manifest = Get-Content -Raw tools/integration-tools.json | ConvertFrom-Json
$jar = 'C:\absolute\wiremock-standalone-3.13.2.jar'
Invoke-WebRequest $manifest.wiremock.jar -OutFile $jar
if ((Invoke-WebRequest $manifest.wiremock.source).Content.Split()[0] -ne $manifest.wiremock.sha256) { throw "Published digest differs from the tracked manifest" }
if ((Get-FileHash -Algorithm SHA256 $jar).Hash.ToLower() -ne $manifest.wiremock.sha256) { throw "Downloaded JAR digest mismatch" }
$env:CQUAY_WIREMOCK_JAR = $jar
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
