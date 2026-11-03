# MCP operation and disclosure

MCP is optional. It does not enroll a host, authenticate a secret-provider identity or grant access merely because a client connects.

## Local stdio

Use an absolute verified executable and protected policy:

```sh
cquay --config /absolute/path/policy.json mcp serve --transport stdio
```

Use the client-specific configuration method, with no bootstrap secret values in the client command or manifest. The effective subject comes from the kernel identity: `local:uid:<numericUID>` on Linux/Android and `local:sid:<SID>` on Windows. Copy the `local_subject` reported by `cquay doctor`; a display name, `USER` environment variable or caller-supplied agent name does not establish identity. Agents sharing one OS identity are not strongly separated; use separate OS users/sandboxes or authenticated service connections when needed. Read-only shared use is allowed only within that trust boundary. An MCP action that can run untrusted code still requires an exact raw-read grant for every disclosure; action permission alone never releases a value.

The [synthetic policy](../../examples/mcp-local.json) shows an exact key read grant. Its UID is a placeholder; replace it with the observed identity before use. It grants no action execution and contains no operational credential.

## Hosted or loopback HTTP

HTTP remains disabled until issuer, audience, trusted key/discovery configuration, allowed hosts/origins and audit storage are configured. Bind loopback by default. Use TLS for remote access. A reverse proxy's headers must not manufacture identity.

CircularQuay is a protected resource, not an authorization server. Configure a qualified OAuth issuer/resource audience, supported clients/scopes, short-lived credentials and revocation behavior. A Google ID token, OpenBao login or plain Cloudflare Access assertion is not automatically the complete MCP OAuth contract. Do not forward inbound MCP tokens to upstream providers.

A zero-additional-cost identity-provider combination must still pass protocol qualification. Current Keycloak guidance marks parts of latest MCP support experimental; do not describe it as production-qualified without evidence and risk review.

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

## Acceptance and removal

Test wrong issuer/audience/client/expiry, Host/Origin/DNS rebinding, cross-user discovery/read/status, guessed IDs, policy changes, secret-free non-read results, cache isolation and resource limits. Synthetic read markers must appear only in authorized responses, never server logs or audit bundles.

Disable the interface, revoke client/grant/provider identities separately, stop native actions according to their policy, and preserve required audit evidence. Removing a registry entry or client configuration does not revoke a credential.
