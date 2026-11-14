# MCP operation and disclosure

MCP is optional. It does not enroll a host, authenticate a secret-provider identity or grant access merely because a client connects.

## Local stdio

Use an absolute verified executable and protected policy:

```sh
cq --config /absolute/path/policy.json mcp serve --transport stdio
```

Use the client-specific configuration method, with no bootstrap secret values in the client command or manifest. The effective subject comes from the kernel identity: `local:uid:<numericUID>` on Linux/Android and `local:sid:<SID>` on Windows. Copy the `local_subject` reported by `cq doctor`; a display name, `USER` environment variable or caller-supplied agent name does not establish identity. Agents sharing one OS identity are not strongly separated; use separate OS users/sandboxes or authenticated service connections when needed. Read-only shared use is allowed only within that trust boundary. An MCP action that can run untrusted code still requires an exact raw-read grant for every disclosure; action permission alone never releases a value.

The [synthetic policy](../../examples/mcp-local.json) shows an exact key read grant. Its UID is a placeholder; replace it with the observed identity before use. It grants no action execution and contains no operational credential.

## Hosted or loopback HTTP

HTTP remains disabled until issuer, audience, trusted key/discovery configuration, allowed hosts/origins and audit storage are configured. Bind loopback by default. Use TLS for remote access. A local TLS-terminating proxy may connect over loopback only when its boundary is trusted; `Forwarded` and `X-Forwarded-*` do not establish transport security, caller identity, or the protected-resource URL. Configure the public resource URL explicitly with its exact non-root path; requests with a query or another path are denied. For DPoP, the actual `Host` must equal that configured resource host even if `mcp.hosts` permits another host for the listener, and an absolute-form request URL must have the configured scheme and host. The proxy must preserve that external canonical `Host`; CircularQuay still obtains the resource from policy, not request headers. This profile does not emit CORS response headers; `origins` filters incoming Origin values and does not enable browser access across origins.

CircularQuay is a protected resource, not an authorization server. Configure a qualified OAuth issuer/resource audience, supported clients/scopes, short-lived credentials and revocation behavior. A Google ID token, OpenBao login or plain Cloudflare Access assertion is not automatically the complete MCP OAuth contract. Do not forward inbound MCP tokens to upstream providers.

An identity-provider combination must pass the exact selected protocol/client profile, including any upstream experimental feature it requires. Price, availability of a login screen or a vendor's MCP label is not interoperability or production qualification.

### Optional DPoP authority profile

Use DPoP only by adding `sender_constraint` to the selected named authority in
the policy. It requires `type: "dpop"`, `nonce: "required"`,
`replay_store: "local_persistent"`, a proof age from 1 through 300 seconds,
and optional clock skew from zero through 30 seconds no greater than that age.
The omitted skew defaults to zero. Keep a bearer authority bearer-only; adding
this field changes that authority to require `Authorization: DPoP` and one
proof header, and it requires its issuer to mint RFC 9449 `cnf.jkt` bound to
the client's DPoP public key.

The client begins with a proof for the configured resource's exact URL, HTTP
method, access-token hash, fresh bounded identifier/time, and matching ES256
P-256 or RS256 2048--8192-bit public JWK. The resource responds to the initial
nonce-less proof with `401`, `WWW-Authenticate: DPoP error="use_dpop_nonce"`,
and `DPoP-Nonce`; the next proof uses that nonce. Each accepted request
atomically consumes the nonce and replay identifier and returns the nonce for
the next proof. Treat the access token, proof, and every nonce as sensitive:
exclude them from proxy, server, tracing, audit, and client logs.

The replay/nonce directory is protected persistent state under `--state-dir`.
Do not delete or reset it as a recovery step. This persistent-local-state profile is for one
replica using local filesystem state; do not run multiple hosts against copied,
shared, or independently reset state. Confirm denial of wrong scheme, absent or
wrong `cnf.jkt`, invalid proof, missing/used nonce, replayed `jti`, stale or
future `iat`, changed path/query/method, wrong `ath`, changed policy, and
restarted service before enabling the authority. DPoP limits bearer-token theft
only while the client private key remains protected; it cannot retract secrets
already disclosed or protect a compromised client key or OS administrator.

## Current catalog and administration contract

The current server registers `list_workloads`, `get_status`, `run_workload` and
`read_secret`; authorization filters the catalog and each call. Configuration
CRUD, managed-key mappings/grants, revocation administration and agent
delegation are complete-v1 requirements, not additional tools in this catalog.
CLI/SDK local policy editing does not grant remote policy mutation. See the
[agent integration guide](../agent-integrations.md#connection-and-management)
and the selected artifact's [qualification record](../qualification.md).

## Grants

Grant exact workloads and operations to verified subjects/clients. Filter discovery and independently check guessed calls. Raw read permission includes explicit key aliases and is separate from invocation, snapshot/publication, and administrative authority. Internal identities cannot be exposed as read aliases.

Native provider audit often records the server's machine identity. Correlate it with the authenticated MCP actor receipt without claiming the upstream provider saw the human identity.

## Raw reads

`read_secret` releases one selected static value in the authorized structured response. This intentionally places plaintext in the client and potentially a model context, transcripts, telemetry or logs. The server cannot guarantee downstream retention or revoke prior disclosures. Obtain the operational owner's explicit disclosure approval before granting it.

No wildcard exports, arbitrary provider URLs, callback recipients, output paths or dynamic code are accepted. Offline/hybrid disclosure still has its usual central-audit limits. A required local receipt must be recorded before release; failure denies disclosure.

## Named actions

Register trusted executable/workflow identity and constrained non-code inputs. Return a safe status/run ID and allowlisted outputs, not raw stdout/stderr. A mutable repository/script or coding agent with injected secrets is disclosure-capable and requires that different trust grant.

Queued/native jobs must retain owner/client/policy identity and recheck permission at dispatch, status, result and cancellation. Do not retain the incoming bearer token as the job credential. Revocation fences future work; stopping in-flight work follows the registered native drain/cancellation behavior.

## Optional managed-key administration

[Managed-key UI, CLI, MCP, and API operations](../managed-key-service.md) use the
same verified issuer/subject and scoped management authorization. Grant changes
cannot exceed the actor's authority; mapping administration does not grant key
retrieval. Key material is bootstrap custody, not a normal raw-read alias or
model-visible administrative field. Audit changes and release decisions using
metadata only. Revocation fences future fetches and actions; it cannot erase a
key or plaintext already delivered. Reuse Netlify/Cloudflare identity and
access-management surfaces, with a thin hosted grant UI only where needed.

## SDK and agent clients

Official SDK callers own their MCP session, authentication, transport and
cancellation; the local SDK adapter is a distinct boundary. Inspect the actual
catalog and negotiated profile, enforce bounded structured responses, and keep
raw-value interfaces explicitly selected. Language-package, skill/plugin and
server versions have separate compatibility and artifact records.

Follow the [agent-client contract](../agent-integrations.md) for packaging,
installation, configuration reload and removal. Neither a plugin manifest nor
SDK version metadata authenticates a server or widens a grant. SDK observers,
logs and traces carry only approved metadata; client shutdown and collector
failure cannot authorize repeating an action that may have completed.

## Lifecycle management parity

V1 management exposes the [canonical policy](../configuration-governance.md)
and [lifecycle resources](../credential-lifecycle.md) under common authorization,
validation, approvals, revision guards and required audit. This does not extend
the current four-tool workload catalog by inference. Never fabricate SCIM,
scanner-exception or provider-revocation tools, use raw-read to correlate a
credential, or wrap a denied action in a mutable script. Supported asynchronous
mutations return safe operation state; cancellation/timeout does not prove an
issuer effect was undone. Keep scanner/provider payloads and replacement bytes
outside model context and telemetry.

## Acceptance and removal

Test wrong issuer/audience/client/expiry, Host/Origin/DNS rebinding, cross-user discovery/read/status, guessed IDs, policy changes, secret-free non-read results, cache isolation and resource limits. Synthetic read markers must appear only in authorized responses, never server logs or audit bundles.

Disable the interface, revoke client/grant/provider identities separately, stop native actions according to their policy, and preserve required audit evidence. Removing a registry entry or client configuration does not revoke a credential.
