# Official SDKs

Use CircularQuay directly from a Go, TypeScript or Python application. The SDKs
let a service check its declared workload, run an approved action, or manage a
local configuration without parsing terminal output or writing shell commands.

| Language | Package | Guide |
| --- | --- | --- |
| Go | `github.com/maldous/CircularQuay/sdk/go` | [Go examples and API](../sdk/go/README.md) |
| TypeScript | `@maldous/cquay-sdk` | [TypeScript examples and API](../sdk/typescript/README.md) |
| Python | `cquay-sdk`, imported as `cquay` | [Python examples and API](../sdk/python/README.md) |

Select an exact qualified version from its release. Installation, registry
ownership, trusted publishing and provenance verification are covered in
[RELEASE.md](../RELEASE.md#official-sdk-distribution).

## Choose a connection

The **local adapter** uses an explicit policy and state directory. Go calls
the core in-process; TypeScript and Python call an explicitly selected native
executable whose SHA-256 is checked before each launch. Configuration JSON goes
through standard input. No adapter loads a shell profile, searches unrelated
credential stores or inherits provider sign-in credentials.

The **MCP adapter** uses a connected official MCP session owned by your
application. Your application selects transport, authentication, cancellation
and session shutdown. Discovery, workload listing, status and approved actions
use the same server-enforced policy as other MCP clients. Identity-provider
integration and MCP remain optional features of CircularQuay.

The SDK cannot authorize an operation that the server or local policy denies.
A caller's inbound CircularQuay access token never becomes a provider credential.
The managed service uses the same authorization boundaries; these adapters do
not invent administrative endpoints or grant remote policy-writing permission.

## Practical uses

- A Python worker validates configuration and starts its approved workload
  without collecting secrets in the parent program's environment.
- A TypeScript service shows authorized workload status and invokes an enrolled
  action with the request's cancellation signal.
- A Go controller applies compare-and-swap configuration changes and runs
  `doctor` before requesting activation.
- An AI connector discovers approved tools without automatically enabling raw
  secret reads or forwarding the model's credentials to a provider.

## Shared contract

Configuration CRUD uses RFC 6901 pointers and whole-document SHA-256
preconditions. Validate the complete candidate before atomic replacement;
conflicts require reading the current document and making a new decision.
`doctor` validates the selected declaration; its repair option canonicalizes a
valid JSON document rather than guessing identities or permissions.

All SDKs return the same safe error categories: `invalid`, `denied`, `trust`,
`unavailable`, `expired`, `unsupported`, `protocol`, `cancelled`, `integrity`
and `process`. Raw provider messages, native stderr, request bodies and
credentials are excluded. Cancellation does not imply that an already completed
remote action was undone; reconcile status before deciding whether to retry.
The SDKs never automatically repeat a potentially effectful action.

Raw secret reads require an explicitly selected disclosure interface and a
separate server grant. Secret wrappers redact ordinary string/JSON/debug
representations; deliberate `reveal` access returns plaintext. The caller
owns that disclosure and any subsequent copies. No SDK promises that a value
can be recalled or that a language runtime zeroizes every copy.

The common client profile accepts one catalog page with at most four known
tools, 4,096 workload handles and a 1 MiB structured response. Oversized or
malformed responses fail as `protocol`; no partial list is returned. These
are SDK resource budgets, not additional policy permissions. Configuration
convenience methods use portable I-JSON: finite numbers, integers within
the safe IEEE-754 range, depth at most 32, at most 1,024 members in one
collection and at most 16,384 nodes. The CLI's raw JSON file interface is
available for wider valid declarations.

Python propagates `asyncio.CancelledError` after bounded cleanup so structured
concurrency keeps its normal semantics; its observation category is
`cancelled`. Other errors use the shared SDK categories.

Local `run` and `flow` can execute applications whose output contains values.
Python and TypeScript require their documented stream handling; Go's local
adapter returns the exit result without capturing child output. Never route
application streams into a general diagnostic logger.

Observability hooks receive approved operation, outcome and duration metadata.
Attach your application's OpenTelemetry instrumentation to those hooks; tokens,
policy values, command arguments and response bodies are excluded. Applications
own their telemetry provider and must keep their hooks bounded. The SDK does
not replace a process-global telemetry provider or make telemetry a condition
of authorization.

Compatibility and conformance cover the same positive, denial, expiry,
cancellation, strict-response and redaction fixtures in all three languages.
Installed package bytes, supported runtime versions, native executable versions
and provenance are identified in each release's qualification matrix. See
[verification](verification.md) for measured results and
[the enterprise contract](enterprise-integration.md) for operational controls.
