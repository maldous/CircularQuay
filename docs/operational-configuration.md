# Observability and release awareness

CircularQuay's v1 policy includes optional `observability` and `updates`
objects. An absent object makes no exporter or release-feed connection.
These controls are independent of provider authentication and required audit.
The [canonical schema](../internal/policy/schema/policy-v1.schema.json) and
[configuration guide](configuration.md) define the complete policy.

## Metrics, traces and structured diagnostics

Add an `observability` object to the policy, alongside `version`, `sources`
and `workloads`. This example selects all three OTLP/HTTP signals:

```json
{
  "enabled": true,
  "service_name": "cq",
  "environment": "engineering",
  "diagnostic_level": "info",
  "resource_attributes": {
    "service_namespace": "example-team",
    "service_instance_id": "example-instance"
  },
  "metrics": {"enabled": true, "interval_seconds": 30},
  "traces": {"enabled": true, "sample_ratio": 0.1},
  "logs": {"enabled": true},
  "endpoint": "https://collector.example.org",
  "timeout_seconds": 5,
  "export_interval_seconds": 5,
  "batch_size": 128,
  "queue_size": 1024
}
```

The collector endpoint is an HTTPS origin. CircularQuay selects `/v1/metrics`,
`/v1/traces` and `/v1/logs`; it does not accept URL credentials, arbitrary
headers, arbitrary resource maps or ambient `OTEL_*` configuration. Declare
each signal explicitly; a disabled signal contains only `"enabled": false`.
At least one signal must be enabled. To disable the whole object, replace it
with `{"enabled": false}`.

| Setting | Accepted bound |
| --- | --- |
| `timeout_seconds` | 1–30 seconds per export |
| `export_interval_seconds` | 1–60 seconds |
| `batch_size` | 1–512 records |
| `queue_size` | 1–8,192 records; at least the batch size |
| `metrics.interval_seconds` | 1–3,600 seconds when metrics are enabled |
| `traces.sample_ratio` | 0–1 when traces are enabled |
| `diagnostic_level` | `off`, `error`, `info` or `debug` |

Use an optional `token_file` for a protected collector-only bearer credential.
For private PKI, declare both `trust_store_file` and `trust_store_sha256`:
an absolute native path to a protected CA-only PEM bundle and its exact
lowercase SHA-256. These roots replace trust for this client only. They do
not change host trust. Use native Windows paths with JSON backslash escaping.
Never place token bytes or private keys in the policy.

Operations, outcomes, provider categories, duration and opaque correlation
identifiers support diagnosis and trends. Secret values, value hashes, secret
names, resource selectors, identities, request/response bodies, credentials,
command arguments and raw errors are excluded. Resource labels are deliberate
operator metadata: do not put personal or sensitive data in them. Higher
diagnostic verbosity never relaxes redaction.

Exporter work has finite queue, request and shutdown budgets. Telemetry is
best-effort; collector outages cannot authorize a request, waive required
audit, stall indefinitely or change a completed workload result into a
retriable failure. Required security audit follows its own durability contract.
Configure collector retention, access control, alerts and SIEM routing in the
collector platform. See [enterprise integration](enterprise-integration.md).

## Lifecycle events and audit separation

[Lifecycle and compromise workflows](credential-lifecycle.md) use the same
canonical observability policy for approved operation/outcome/duration,
evidence-freshness, queue/backlog and exporter-health signals. SCIM, GitHub,
GitGuardian and issuer adapters cannot attach raw bodies, secret-derived hashes,
credential references, replacement values or uncontrolled identity/source labels.
Bound event attributes and cardinality; preserve safe opaque correlation only
where the declared profile permits it. A scanner API's read-only permission
does not make its entire response suitable telemetry.

OTLP is best-effort operational export. Collector receipt does not establish
immutable retention, complete incident history, exactly-once delivery or durable
required audit. Storage integrity, independent checkpoints, retention and restore
are separate [audit controls](enterprise-integration.md#audit-contract). Export
failure cannot waive an exposure block, authorize remediation or turn an already
completed issuer mutation into an automatic retry.

## Release awareness

An interactive policy checks the stable channel at most once each day:

```json
{
  "check": "interactive",
  "channel": "stable",
  "interval_hours": 24,
  "timeout_seconds": 3
}
```

Place this object at `/updates`. `release-candidate` selects the RC channel.
With terminal input and diagnostics, startup displays a newer release and its
canonical upgrade-instructions link, then continues. It does not read a
response or consume application input. There is no prompt in SDK, MCP or
non-interactive sessions and no background download, installation or restart.

Use the same check explicitly from automation:

```sh
cq --config cq.json update check --json
```

The default feed is the public CircularQuay GitHub releases API. An enterprise
may select a credential-free HTTPS `feed_endpoint` with compatible release
metadata and optional protected `token_file` and pinned CA references.
Checks and responses are bounded; redirects and ambient proxies are denied.
The stable channel excludes prerelease versions even if a feed incorrectly
labels them as stable. The release-candidate channel includes eligible
prerelease and stable versions, using Semantic Version ordering.
Feed metadata is advisory, not publisher authentication for a binary. Follow
the release's signature/provenance and installation instructions before
changing an executable.

## Offline operation and minimum versions

Disable network checks while requiring a selected minimum runtime:

```json
{"check": "never", "minimum_version": "1.0.0"}
```

`minimum_version` uses a complete Semantic Version. It is enforced locally
before sensitive runtime effects, independently of feed availability. A
development or malformed build cannot satisfy a stable production minimum. An
explicit prerelease minimum follows Semantic Version prerelease ordering. Version,
configuration repair, `doctor` and explicit update inspection remain available
for recovery. A failed advisory network check continues the invocation; a
minimum-version denial requires an approved compatible binary.

With `check: never`, remote feed, channel, interval, credential and TLS settings
are rejected. No current-release claim is inferred from an unavailable feed
or an expired cache. An isolated environment can omit `minimum_version` when
its separately governed installation policy supplies the version boundary.

## SDK, CI and managed administration

The runtime's protected policy is authoritative for collector/export and local
minimum-version settings. SDK observers expose only operation, safe outcome and
duration metadata; applications own their OpenTelemetry provider. MCP clients
and CI jobs remain noninteractive. A disabled exporter or release check does
not disable required audit or permit a lower runtime than declared policy.

CLI config CRUD and authorized management interfaces validate the same
non-secret declarations, authority ceilings and revision preconditions. The
managed UI presents those settings without becoming a prerequisite for runtime
telemetry. Verify collector/feed credentials and trust separately from provider
identity. No browser or model receives key material as a configuration field.

## Validate and automate changes

Read the current object and its document hash:

```sh
cq --config cq.json config get --pointer /updates
cq --config cq.json config set --pointer /updates --value-file updates.json --expect-sha256 "<document sha256>"
cq --config cq.json config validate
cq --config cq.json doctor
```

Use the same commands from PowerShell with the selected `cq.exe` path.
Keep the policy and referenced files protected. `doctor` checks declarations
and local references; it does not contact a collector/feed or read a token's
contents. After validation, use synthetic traffic to verify export and
failure behavior with the exact selected collector. Configuration validation
alone does not establish successful delivery or production qualification.
