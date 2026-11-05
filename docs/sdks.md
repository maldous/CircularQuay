# Official SDKs

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

Use CircularQuay directly from a Go, TypeScript or Python application. The SDKs
let a service check its declared workload, run an approved action, or manage a
local configuration without parsing terminal output or writing shell commands.

| Language | Package | Guide |
| --- | --- | --- |
| Go | `github.com/maldous/CircularQuay/sdk/go` | [Go examples and API](../sdk/go/README.md) |
| TypeScript | `@maldous/cq-sdk` | [TypeScript examples and API](../sdk/typescript/README.md) |
| Python | `cq-sdk`, imported as `cq` | [Python examples and API](../sdk/python/README.md) |

Select an exact qualified version from its release. Installation, registry
ownership, trusted publishing and provenance verification are covered in
[RELEASE.md](../RELEASE.md#official-sdk-distribution).

## Choose a connection

| Connection | Execution boundary | Your application supplies |
| --- | --- | --- |
| Go local | The linked CircularQuay core runs in-process. | Explicit policy/state locations and caller cancellation; verify the containing module and linked-core version. |
| TypeScript or Python local | A separate, explicitly selected native `cq` process. | Absolute executable location, expected SHA-256, policy/state locations and an explicit child environment. |
| MCP, in any SDK | An application-owned official MCP session to the selected CircularQuay server. | Transport, authentication when required, resource-specific authorization and session lifecycle. |

Local adapters do not require an MCP server. Language-package installation does
not install or authenticate a native executable; select and verify it separately
when using the TypeScript or Python local adapter. MCP access does not inherit
permission to edit the server's local configuration. Choose the boundary that
matches the operation, then use the same configuration and disclosure rules below.

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

Version inspection validates the four string fields `version`, `commit`, `go`
and `maturity`. The version is a complete Semantic Version of at most 256 UTF-8
bytes, with an optional `v` prefix. Malformed metadata is `protocol`; a valid
incompatible major is `unsupported`. Prerelease and build identifiers retain
their Semantic Version rules. Go checks its linked core; Python and TypeScript
inspect their explicitly selected native executable. Version metadata does not
authenticate a publisher or replace artifact verification.

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
Installed package bytes, supported runtime versions, linked-core and native
executable versions, and provenance are identified in each release's
qualification matrix. See
[verification](verification.md) for measured results and
[the enterprise contract](enterprise-integration.md) for operational controls.

## Governed lifecycle administration

The v1 [canonical policy](configuration-governance.md) and
[lifecycle contract](credential-lifecycle.md) use common typed resources,
authorization, validation, approvals, revision guards and required audit through
CLI/API/SDK/MCP. Local config CRUD does not implement remote SCIM, scanner or
issuer administration. Select only public methods/tools actually supported by
the chosen package and server; unsupported operations fail explicitly.

Provisioning, exposure-policy changes, exceptions and provider-native
rotation/revocation retain separate authority and secure credential references.
Return safe operation identifiers and allowlisted outcomes; never replacement
values, raw scanner responses or a guessed GitGuardian secret-hash binding.
Reconcile ambiguous issuer effects before retrying. SDK observers export only
safe telemetry, which does not guarantee immutable or durable audit.

Both self-contained and master/include configurations use the
[same resolved policy](configuration-repositories.md#single-file-or-master-and-includes-configuration),
with pinned source trust, namespaces, locks and complete atomic validation.
An SDK adapter does not gain authority to read/upload personal local files
through a remote service. Inspect actual methods and schema support before
using composition; local config CRUD is not a multi-source compiler by itself.
Return only viewer-authorized explain/provenance metadata and apply explicit
offline/refresh policy without weakening corporate grants.

SDKs share the [first-run initialization contract](management-and-discovery.md#first-run-and-minimal-managed-registration):
resolve policy and CLI/config hosted opt-outs before any hosted-service network,
authenticate the resolved identity and idempotently initialize BOTH hosted KMS
and age-key associations when enabled, as required by the latest product decision.
This is intended behavior, not current adapter support. Do not download an
offline age key or activate hybrid by association creation alone. Noninteractive clients use enrolled
workload identity or return an actionable interaction-required outcome. This
contract supplies no unimplemented SDK method or opt-out option by implication.
Migration requires the [same complete migration contract](managed-key-service.md#end-to-end-local-to-managed-migration)
across native/SDK/API/CLI/MCP: metadata-only planning, explicit source/destination
grants, verified backend format/export capability, staged output, durable per-asset
idempotency/recovery and atomic validated activation. Do not invent a migration
method or treat a mode option as key import. No private keys/config/variables enter
model-visible SDK diagnostics or logs; real backend evidence remains separate
from mock conformance, under the existing project assurance gates. A separate
frontend checkout may use synthetic/mock SDK-facing previews or loopback local
configuration simulations only; it cannot add a real SDK service/backend, host
proxy, provider/key/account access or production deployment.

Registration returns safe metadata, not a personal-file upload, payment consent
or enterprise remote-control grant; future key release remains separately
mapping/window-authorized and cannot revoke copied keys.

Association creation, genuine KMS operations and age fetches remain separately
typed events with explicit quota/tariff policy; retries reconcile one logical
operation even if provider attempts incur additional cost. Age-access metering
uses the [shared logical-event contract](managed-key-service.md#profile-utilization-and-fair-metering).
Preserve request-bound idempotency through safe reconciliation; never create
extra charges by retrying one logical access or collapse independent host
fetches into a profile-count unit. Cancellation/network loss after release
commit does not prove zero use; failed/denied precommit calls are not retrieval
uses. SDK observers expose no key bytes in usage diagnostics.

Client SDKs resolve/validate/save all user configuration locally through the
caller's files/network/VPN/source credentials. No config content or source
credential crosses to hosted backend/proxy/logs/analytics; remote hosted methods
operate only on authenticated profile/key associations and necessary service
metadata. Browser integration must honor CORS/file permissions and offer the
same options via permitted native/local integration, never a hosted fetch proxy.
Key values are protected and not displayed/logged.

The [account service contract](managed-key-service.md#end-customer-accounts-subscriptions-and-support)
adds customer-authorized account/usage/invoice/subscription/support methods only
when the selected implementation exposes them. Bound provider IDs to verified
tenant/account, keep safe paginated history, and reconcile duplicate/out-of-order
provider events and uncertain mutations through one durable intent. Financial
execution needs explicit user-approved limits plus applicable confirmation;
SDK installation grants none. Do not expose operator credentials/card fields,
retry a charge blindly or confuse a provider's compute invoice with profile uses.

## Agent integration and distribution

Applications own SDK lifecycle and transport. Agent clients additionally use the
[portable skill or client plugin](agent-integrations.md); neither layer adds an
implicit grant or exports provider bootstrap material. Runtime, SDK and plugin
identities remain independently versioned and verified. Publisher procedures
stay in [RELEASE.md](../RELEASE.md#official-sdk-distribution), while actual
execution and publication outcomes stay in [qualification](qualification.md).
