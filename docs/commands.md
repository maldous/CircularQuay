# Command reference

The command reference covers configuration, workload delivery, CI automation
and optional integrations. Revision-scoped testing and release qualification
are recorded in the [qualification evidence](qualification.md).

All commands accept `--config PATH` (default `cquay.json`) and `--state-dir
PATH` before the command. `help`, `--help`, `-h`, `version`, and `providers`
need no policy. `bootstrap` and `flow run` can resolve configuration before a
local policy exists. `version` reports the executable version.

| Command | Usage and behavior |
| --- | --- |
| Bootstrap | `cquay bootstrap [--config-source URL] [--config-ref REF] [--config-path PATH] [--config-sha256 SHA256] [--output PATH] [--non-interactive]` selects local or enrolled configuration, verifies remote references, and configures the authorized session. Equivalent source settings are `CQUAY_CONFIG_SOURCE`, `CQUAY_CONFIG_REF`, `CQUAY_CONFIG_PATH`, and `CQUAY_CONFIG_SHA256`. Unattended first enrollment pins the full commit and a configuration digest obtained through a trusted maintainer channel. |
| CI flow | `cquay flow run --workload ID [--non-interactive]` detects the supported runner, resolves configuration and workload identity, then executes the approved workload. Explicit configuration-source options override discovery. |
| Create | `cquay init [--output PATH]` or `cquay config init [--output PATH]` creates a protected synthetic JSON policy and fixture, refusing to overwrite either. |
| Inspect configuration | `cquay config validate`; `cquay config get --pointer RFC6901`; `cquay config list [--pointer RFC6901]`. `get` and `list` return the whole-document SHA-256. The returned policy literal is visible to an authorized configuration reader. |
| Edit configuration | `cquay config set --pointer RFC6901 (--value-json JSON\|--value-file PATH\|-) [--expect-sha256 HEX]`; `cquay config delete --pointer RFC6901 [--expect-sha256 HEX]`; `cquay config replace --file PATH\|- [--expect-sha256 HEX]`. Values are JSON; `-` reads standard input; inputs are capped at 1 MiB. |
| Doctor | `cquay doctor [--fix\|--identity-only]`. A missing or unsafe selected policy is non-zero. `--identity-only` reports the local kernel-derived subject without checking policy. `--fix` canonically formats a valid protected JSON policy only; it does not alter grants, provider settings, services, or deployment state. |
| Plan and status | `cquay plan`; `cquay status --workload ID [--json]`. |
| Workload lifecycle | `cquay run --workload ID [--target ID] [-- ABSOLUTE-PROGRAM ARGS…]`; `cquay snapshot --workload ID [--target ID]`; `cquay apply --workload ID --target ID`; `cquay sync --workload ID [--target ID] [--once]`. `apply` requires an explicit target; only `run` accepts arguments after `--`. |
| MCP | `cquay mcp serve [--transport stdio\|http] [--tls-cert PATH --tls-key PATH]`. HTTP uses the policy listener address (or loopback `127.0.0.1:8080`); a non-loopback listener requires both TLS paths. |

Configuration operations are cooperative: each writer takes the private local
configuration lock, validates a candidate, then re-reads the current digest
before atomically replacing the file. `--expect-sha256` adds an optimistic
review check; it is not an external distributed CAS. Configuration editing and
`doctor --fix` accept protected JSON only. YAML may be loaded for normal policy
execution but is intentionally not mutated. A repeated `set` or `replace` with
the same value produces the same canonical document; a repeated `delete` fails
after the node is absent.

The separate assurance binary is for evidence generation, not workload
delivery: `cquay-audit run --profile source|enterprise --tool-manifest PATH --output DIR` with
optional `--source`, `--go`, `--sonar-url`, `--sonar-token-file`, and
`--sonar-project-key`. Its enterprise inputs and evidence handling are
documented in [RELEASE.md](../RELEASE.md).

### Configuration CRUD

These examples edit public configuration declarations. They never supply a
provider token or private key. Start with the synthetic policy from the quick
start; `set` creates a missing object member or replaces an existing node.

```sh
# Create a synthetic policy and validate it.
cquay config init --output cquay.json
cquay --config cquay.json config validate

# List workloads and read one declaration. Both responses include sha256.
cquay --config cquay.json config list --pointer /workloads
cquay --config cquay.json config get --pointer /workloads/demo

# Create or update a complete non-secret environment object.
cquay --config cquay.json config set --pointer /workloads/demo/environment --value-json '{"APP_MODE":"development"}'

# Read a replacement node from stdin or a protected JSON file.
printf '%s\n' '{"APP_MODE":"test"}' | cquay --config cquay.json config set --pointer /workloads/demo/environment --value-file -
cquay --config cquay.json config set --pointer /workloads/demo/environment --value-file environment.json

# Delete that optional object; invalid or referenced-object deletions fail.
cquay --config cquay.json config delete --pointer /workloads/demo/environment

# Replace the complete policy with a validated, operator-prepared document.
cquay --config cquay.json config replace --file replacement-policy.json
```

Add `--expect-sha256` with the digest returned by the preceding `get` or `list`
to reject an edit based on an out-of-date document. For example, PowerShell can
read the JSON response without another utility:

```powershell
$review = .\cquay.exe --config cquay.json config get --pointer /workloads/demo | ConvertFrom-Json
.\cquay.exe --config cquay.json config set --pointer /workloads/demo/environment --value-json '{"APP_MODE":"development"}' --expect-sha256 $review.sha256
```

An empty pointer selects the complete document; `~0` escapes `~` and `~1`
escapes `/` in a member name. Array elements use numeric indices. Replace a
complete array to add entries. Parents must exist, and the resulting complete
policy must remain valid. Errors do not replace the previous policy.

### Diagnostics, planning and execution

```sh
cquay help
cquay --help
cquay version
cquay providers
cquay --config cquay.json doctor
cquay --config cquay.json doctor --fix
cquay doctor --identity-only
cquay --config cquay.json plan
cquay --config cquay.json --state-dir /absolute/private/state status --workload demo --json
cquay --config cquay.json run --workload demo -- /usr/bin/printf '%s\n' 'Synthetic workload completed.'
```

`doctor --fix` only formats valid protected JSON; it does not choose identities
or change permissions and grants. `doctor --identity-only` reports identity,
not workload readiness. A configured command can also be run without an
override: `cquay --config deployed-policy.json run --workload api`.

### Snapshots and native delivery

Use these commands with a fully provisioned policy such as the Linux reference,
with eligible bindings and declared snapshots/targets. They write configured
destinations; native activation occurs only when explicitly declared.

```sh
cquay --config deployed-policy.json snapshot --workload reference-api
cquay --config deployed-policy.json apply --workload reference-api --target systemd-credential
cquay --config deployed-policy.json sync --workload reference-api --once
cquay --config deployed-policy.json sync --workload reference-api --target systemd-credential --once
```

### Local and hosted MCP

```sh
cquay --config local-mcp.json mcp serve --transport stdio
cquay --config hosted-mcp.json mcp serve --transport http
cquay --config hosted-mcp.json mcp serve --transport http --tls-cert /absolute/tls/server.crt --tls-key /absolute/private/server.key
```

Hosted HTTP requires the exact configured issuer, resource, client and grants.
A listener is not an IdP or a managed multi-tenant service. The
[hosted identity boundary](hosted-identity.md) describes that distinction.

### Independent assurance

```sh
cquay-audit run --profile source --source . --tool-manifest /absolute/private/tools.json --output audit-results/source --go /absolute/path/to/go
cquay-audit run --profile enterprise --source . --tool-manifest /absolute/private/tools.json --output audit-results/enterprise --go /absolute/path/to/go --sonar-url http://127.0.0.1:9000 --sonar-token-file /absolute/private/scanner.token --sonar-project-key circularquay
cquay-audit sonar-init --directory .local/sonar
cquay-audit sonar-bootstrap --directory .local/sonar --url http://127.0.0.1:9000 --project circularquay
```

The Sonar commands are for a new isolated audit instance. Follow the prerequisite,
bootstrap and evidence-custody instructions in [RELEASE.md](../RELEASE.md); do not
point them at an operational instance. A failing or unexecuted gate remains a
failing or unexecuted gate.

[Documentation](README.md) · [CircularQuay](../README.md)
