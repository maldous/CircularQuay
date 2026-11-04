# CircularQuay Go SDK

`github.com/maldous/CircularQuay/sdk/go` gives Go applications access to the linked CircularQuay core and its optional Model Context Protocol (MCP) interface.

It wraps a session you create with the official [MCP Go SDK](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp) v1.8.0 dependency pinned by this repository.

The SDK never finds credentials, reads shell profiles, starts `cquay`, creates a network transport, retries calls, or caches secrets. Your application owns the session, transport, authentication, TLS policy, and lifecycle.

## MCP client

Construct and authenticate an official MCP session using your explicit transport configuration, then adapt the connected session:

```go
session, err := mcp.NewClient(&mcp.Implementation{Name: "example", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{
    Endpoint: "https://cquay.example.invalid/mcp", HTTPClient: configuredHTTPClient, MaxRetries: 0,
}, nil)
if err != nil { /* handle connection error */ }
defer session.Close()

client, err := cquay.New(session)
status, err := client.GetStatus(ctx, "payments")
```

The four typed operations mirror the current MCP catalog:

```go
tools, err := client.DiscoverTools(ctx)
workloads, err := client.ListWorkloads(ctx)
status, err := client.GetStatus(ctx, "payments")
run, err := client.RunWorkload(ctx, "payments")
```

Plaintext is intentionally separate and visible in review. It still requires a server-side identity, policy grant, and audited disclosure:

```go
disclosure := client.Disclosure(cquay.AllowPlaintext{})
secret, err := disclosure.ReadSecret(ctx, "payments", "DATABASE_PASSWORD")
if err != nil { /* inspect only the safe cquay.Error category */ }
password := secret.Value() // plaintext; avoid logging or serializing it
```

`Secret` redacts `String`, Go-syntax formatting, and JSON. `Secret.Version` preserves the provider's opaque, bounded UTF-8 version identifier, so it can represent a Google Secret Manager resource path, a Bitwarden RFC3339 revision, or another provider-specific value. Go cannot promise that strings are zeroized after use. A valid elapsed `expires_at` is reported as the safe `expired` category; a malformed timestamp is `protocol`.

## Local client

`NewLocal` calls the linked CircularQuay core in-process rather than spawning a subprocess or accepting a generic command string. It requires absolute configuration and state paths, so it cannot silently use a user's ambient default state:

```go
local, err := cquay.NewLocal(cquay.LocalOptions{
    ConfigPath: "/srv/cquay/cquay.json", StateDirectory: "/srv/cquay/state",
})
version, err := local.AssertRuntimeCompatibility(ctx)
plan, err := local.Plan(ctx)
status, err := local.Status(ctx, "payments")
updated, err := local.ConfigSet(ctx, "/workloads/payments", json.RawMessage(policyJSON), priorSHA256)
```

The local client provides typed entry points for configuration CRUD and validation, doctor, plan, status, snapshot, sync, apply, bootstrap, CI flow, explicit update awareness, and linked-core v1 compatibility (`Version` and `AssertRuntimeCompatibility`). Those compatibility methods validate metadata from the embedded core; they do not inspect or pin an external executable. Operations that can execute a child process (`RunWorkload` and `Flow`) discard child stdout and stderr and return only an exit code. JSON-returning methods expose only bounded public CLI JSON DTOs; they do not expose internal Go types. `ConfigSet` and `ConfigReplace` accept portable I-JSON: finite IEEE-754 numbers, integers in the safe IEEE-754 range, and bounded document depth, collection size, and node count. Use the CLI's raw file interface for a wider valid JSON declaration.

## Observability

`cquay.WithObserver` and `cquay.WithLocalObserver` receive only operation name, safe outcome category, and duration. They never receive tool arguments, workload IDs, aliases, endpoints, tokens, or secret values, and an observer panic cannot change an operation's result. `cquay.Error` uses the v1 categories `invalid`, `denied`, `trust`, `unavailable`, `expired`, `unsupported`, `protocol`, `cancelled`, `integrity`, and `process`; it never retains remote diagnostic text.

## Versions and installation

Select the exact qualified root-module release and import this package from
that version. [The release guide](../../RELEASE.md#official-sdk-distribution)
defines Go module indexing, integrity verification, compatibility and immutable
RC/stable versions. [Qualification evidence](../../docs/qualification.md)
identifies the actual tested source, runtime, transport and distribution scope.
