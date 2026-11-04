# cquay-sdk

`cquay-sdk` is the official Python v1 client contract for CircularQuay.  It
offers two deliberately separate adapters:

- `CircularQuayClient` speaks to an already connected, caller-owned official
  MCP `ClientSession` and exposes discovery, workload listing, status and
  enrolled actions.
- `LocalCLIClient` invokes the installed native `cquay` program with an
  operator-provided absolute executable, expected SHA-256, absolute policy and
  absolute state directory.  It exposes the complete current native command
  surface: init, configuration CRUD, doctor, plan, status, run, snapshot,
  apply, sync, bootstrap, flow, version and providers.

The package distribution name is `cquay-sdk` and it imports as `cquay`.

## Install

Install the core package for local CLI use.  The MCP transport is deliberately
optional because applications own their selected official MCP transport and its
authentication lifecycle.

```sh
python -m pip install 'cquay-sdk==1.0.0.dev0'
python -m pip install 'cquay-sdk[mcp]==1.0.0.dev0'
```

Use a pinned public registry, an allowlisted index URL, and the release's
published package hashes in production.  Do not place provider credentials,
access tokens, or private values in package configuration, command arguments or
logs.

## MCP client

Create and initialize the official session yourself.  The adapter neither
starts a process nor takes ownership of credentials, retries, caches, profiles
or session shutdown.  This stdio form is the current official Python MCP SDK
shape:

```python
import asyncio

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client
from cquay import CircularQuayClient, MCPClientSessionAdapter


async def main() -> None:
    params = StdioServerParameters(
        command="/absolute/operator-verified/cquay",
        args=[
            "--config", "/absolute/protected/cquay.json",
            "--state-dir", "/absolute/protected/cquay-state",
            "mcp", "serve", "--transport", "stdio",
        ],
        env={},  # explicit minimal environment; never inherit provider credentials
    )
    async with stdio_client(params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            cquay = CircularQuayClient(MCPClientSessionAdapter(session))
            tools = await cquay.discover_tools()
            if tools.includes("list_workloads"):
                print(await cquay.list_workloads())


asyncio.run(main())
```

For hosted MCP, construct the official `streamable_http_client` session with
the exact protected-resource URL and an application-owned authenticated HTTP
client, then wrap that initialized `ClientSession` in
`MCPClientSessionAdapter` in the same way.  The SDK never forwards an MCP token
to a secret provider or treats an IdP login as provider authorization.

`read_secret` is not present on `CircularQuayClient`.  Its deliberately explicit
interface makes the plaintext boundary visible in review:

```python
disclosure = await cquay.disclosures().read_secret("payments", "PAYMENT_API_KEY")
deliver_to_the_authorized_consumer(disclosure.reveal())
audit_metadata = disclosure.redacted_dict()  # never serializes plaintext
```

`repr(disclosure)`, `str(disclosure)` and `redacted_dict()` redact the value.
Python strings may be copied by runtimes and libraries, so this package does not
promise memory zeroization or recall a value after disclosure.

Provider versions are opaque metadata, not workload handles.  The SDK accepts a
provider version of up to 4,096 UTF-8 bytes and rejects C0 or DEL control
characters; secret values are bounded to 65,536 UTF-8 bytes.  A malformed expiry
is a protocol error, while a valid present expiry in the past is reported as an
`expired` error category.

## Local CLI client

The local client validates executable identity and digest before each process
start, uses `exec` without a shell, and receives configuration JSON on standard
input rather than argv.  It captures only bounded JSON protocol responses.
`run` is different: it requires an explicit `RunStdio.INHERIT` choice and
passes child standard streams through without capture because child output can
contain values.

```python
from cquay import LocalCLIClient, RunStdio

cquay = LocalCLIClient(
    executable="/absolute/operator-verified/cquay",
    expected_sha256="<64 lowercase hexadecimal characters from the release verification route>",
    config="/absolute/protected/cquay.json",
    state_dir="/absolute/protected/cquay-state",
    environment={},
)

review = await cquay.config_get("/workloads/payments")
await cquay.config_set(
    "/workloads/payments/environment",
    {"APP_MODE": "production"},
    expect_sha256=review["sha256"],
)
await cquay.run("payments", stdio=RunStdio.INHERIT)
```

The optional `environment` is explicit and starts empty.  The SDK never copies
the parent process environment and rejects common provider, vault and bootstrap
credential prefixes.  If a platform needs process variables such as a Windows
system directory, pass only those documented platform variables after reviewing
them.  Provider and workload identity remain the native policy's responsibility.

Remote bootstrap and flow sources must be absolute HTTPS URLs without userinfo,
query parameters, or fragments.  Pin the repository reference, configuration
path, and SHA-256 explicitly; do not put tokens or other credentials in a URL.

The configuration convenience methods accept portable I-JSON: finite numbers
and integers from `-9007199254740991` through `9007199254740991`.  Use the
operator-controlled native CLI for a wider raw JSON declaration where the
policy schema permits it; the SDK does not silently round integers.

Each adapter accepts an optional observer.  It receives only operation name,
outcome category and duration.  It never receives IDs, arguments, URLs, values,
tokens, response bodies or raw errors; observer failures cannot turn a successful
operation into a retry.

Python follows structured-concurrency conventions: caller cancellation is
re-raised as `asyncio.CancelledError` after any owned local child is reaped.  The
observer records that outcome as `cancelled`; it is not converted into an
ordinary `CircularQuayError`.

The SDK's strict output validation is a protocol boundary, not a replacement for
native release signing, protected policy/state ownership, IdP configuration,
provider qualification, or endpoint security.  Consult the repository's
operator and release guidance before enabling a real provider or hosted MCP.

The local adapter's methods map one-for-one to current CLI operations:
`version`, `providers`, `update_check`, `init`, `config_init`, `config_validate`, `config_get`,
`config_list`, `config_set`, `config_delete`, `config_replace`, `doctor`,
`plan`, `status`, `run`, `snapshot`, `apply`, `sync`, `bootstrap`, and `flow`.
`assert_runtime_compatibility()` checks that the pinned executable reports a v1
runtime before an operator relies on it. `update_check()` is an explicit,
structured update-awareness check; it never schedules checks, prompts,
downloads, installs, or activates software.
