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
