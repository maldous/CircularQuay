# Provider, platform and AI boundaries

SOPS can use cloud KMS or OpenBao Transit to protect a document key. That is different from retrieving existing application values from a secret manager.

| Integration | Scope |
| --- | --- |
| JSON / literal dotenv / selected process variables | Explicit local configuration; no interpolation or shell-profile execution. |
| SOPS / age / approved Transit endpoint | Upstream encryption and scoped key release. |
| OpenBao / Vault KV | Exact mount, path, field, and supported version. |
| AWS SM/SSM, Azure Key Vault secrets, GCP Secret Manager | Native provider identities and explicit resource references. |
| Doppler, Infisical, Bitwarden, Akeyless, optional 1Password | Connector-specific identity, lifecycle, audit, dependency, and platform limits. |
| systemd / Windows / Termux / containers | Native startup and credential boundaries; actual devices require qualification. |
| Windmill / Workers / Netlify / cloud applications | Explicit native one-way delivery profiles; no inherited source-audit guarantee. |

Use `cq providers` and the qualification matrix for the current executable's capabilities. Unsupported combinations must fail explicitly. Provider and platform support does not imply a free subscription, official Android support, or an upstream security endorsement.

Termux startup occurs after Android permits access following the first unlock. Multiple Termux processes share an application UID. Windows DPAPI machine protection relies on ACLs for service separation. Root or Administrator compromise defeats endpoint protections.

## SDK, discovery and operational boundaries

CLI, CI and official SDK entry points share policy and complete-generation
checks. Local Go uses its linked core; Python/TypeScript local uses a separately
verified native executable; MCP uses the application's enrolled session.
Package installation, discovery, version metadata and client names do not
establish workload or administrative authority.

Metrics, traces, SDK hooks and structured diagnostics carry only the approved
operational vocabulary. Required audit is separately durable; best-effort
telemetry cannot satisfy it. Advisory release checks cannot authorize code
installation or bypass a minimum-version policy. See [SDKs](sdks.md),
[agent clients](agent-integrations.md) and [operational configuration](operational-configuration.md).

## MCP and AI agents

CircularQuay's MCP interface is optional. Local stdio uses the OS-user/launcher boundary. Separately configured HTTP requires a qualified OAuth issuer, the correct resource audience, and current authorization on each operation.

- List/status results are filtered by current grants.
- Named actions do not accept arbitrary commands or provider URLs.
- `read_secret` requires a separate grant for an exact configured key.
- Bootstrap keys, refresh credentials, and server identities must not be exposed as readable aliases.
- Raw action stdout/stderr is not an authorized substitute for a secret-read permission.

**An authorized raw read deliberately releases plaintext to the MCP client.** It can enter model context, transcripts, or the client's retention systems. CircularQuay cannot control that value after delivery. Grants prevent future toolkit access; they cannot erase previous disclosures.

Self-reported agent names are not identity. Strong separation between agents sharing one OS user requires an additional OS or authenticated-service boundary. A coding agent running caller-controlled code with injected credentials has disclosure-capable authority.

[Documentation](README.md) · [CircularQuay](../README.md)
