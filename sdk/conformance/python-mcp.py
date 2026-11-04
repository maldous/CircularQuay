"""Official-MCP conformance fixture for the Python CircularQuay SDK.

The harness supplies only synthetic policy paths through the three required
environment variables.  The fixture requests an explicit empty environment
override from the official stdio transport; it does not read or forward parent
provider credentials or profile settings, and it neither prints nor persists
the synthetic disclosure.
"""

from __future__ import annotations

import asyncio
import os
from pathlib import Path
from typing import NoReturn

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

from cquay import CircularQuayClient, CircularQuayError, ErrorCategory, MCPClientSessionAdapter

_TIMEOUT_SECONDS = 30
_SYNTHETIC_VALUE = "synthetic-only-not-a-credential"
_EXPECTED_TOOLS = ("get_status", "list_workloads", "read_secret")


def _required_absolute_path(name: str, *, kind: str) -> str:
    value = os.environ.get(name)
    if not value or not Path(value).is_absolute():
        _fail(f"missing absolute synthetic {kind} path")
    path = Path(value)
    if kind == "executable" and not path.is_file():
        _fail("synthetic executable is unavailable")
    if kind == "configuration" and not path.is_file():
        _fail("synthetic configuration is unavailable")
    if kind == "state directory" and not path.is_dir():
        _fail("synthetic state directory is unavailable")
    return value


async def _main() -> None:
    command = _required_absolute_path("CQUAY_EXECUTABLE", kind="executable")
    config = _required_absolute_path("CQUAY_CONFIG", kind="configuration")
    state_directory = _required_absolute_path("CQUAY_STATE_DIRECTORY", kind="state directory")
    parameters = StdioServerParameters(
        command=command,
        args=["--config", config, "--state-dir", state_directory, "mcp", "serve", "--transport", "stdio"],
        env={},
    )
    async with asyncio.timeout(_TIMEOUT_SECONDS):
        async with stdio_client(parameters) as (read, write):
            async with ClientSession(read, write) as session:
                await session.initialize()
                client = CircularQuayClient(MCPClientSessionAdapter(session))
                catalog = await client.discover_tools()
                _require(tuple(sorted(tool.value for tool in catalog.available)) == _EXPECTED_TOOLS, "unexpected catalog")
                _require(await client.list_workloads() == ("demo",), "unexpected workloads")
                status = await client.get_status("demo")
                _require(status.workload == "demo", "unexpected status")
                disclosure = await client.disclosures().read_secret("demo", "DEMO_API_KEY")
                _require(_SYNTHETIC_VALUE not in repr(disclosure), "secret disclosure repr leaked")
                _require(disclosure.reveal() == _SYNTHETIC_VALUE, "unexpected synthetic disclosure")
                await _assert_denied_status(client)


async def _assert_denied_status(client: CircularQuayClient) -> None:
    try:
        await client.get_status("not-granted")
    except CircularQuayError as error:
        _require(error.category is ErrorCategory.DENIED, "status was not denied")
        return
    _fail("ungranted status unexpectedly succeeded")


def _require(condition: bool, message: str) -> None:
    if not condition:
        _fail(message)


def _fail(message: str) -> NoReturn:
    raise RuntimeError(f"python MCP conformance failure: {message}")


if __name__ == "__main__":
    asyncio.run(_main())
