"""Contract tests for the Python SDK using synthetic, in-memory MCP shapes."""

from __future__ import annotations

import asyncio
import json
import unittest
from dataclasses import asdict
from types import SimpleNamespace

from cquay import (
    CircularQuayClient,
    CircularQuayError,
    ErrorCategory,
    MCPClientSessionAdapter,
    ToolName,
)

_ASYNC_TEST_DEADLINE_SECONDS = 1


class FakeSession:
    def __init__(self) -> None:
        self.tools = [SimpleNamespace(name="list_workloads"), SimpleNamespace(name="get_status")]
        self.responses: dict[str, object] = {}
        self.calls: list[tuple[str, dict[str, str]]] = []

    async def list_tools(self) -> object:
        return SimpleNamespace(tools=self.tools)

    async def call_tool(self, name: str, arguments: dict[str, str]) -> object:
        self.calls.append((name, arguments))
        result = self.responses[name]
        if isinstance(result, BaseException):
            raise result
        return result


def response(value: object, *, is_error: bool = False) -> object:
    return SimpleNamespace(
        structuredContent=value,
        content=[SimpleNamespace(text=json.dumps(value))],
        isError=is_error,
    )


class MCPClientTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.session = FakeSession()
        self.events: list[object] = []
        self.client = CircularQuayClient(
            MCPClientSessionAdapter(self.session),
            observer=self.events.append,
        )

    async def test_current_tool_contract_and_safe_outputs(self) -> None:
        catalog = await self.client.discover_tools()
        self.assertTrue(catalog.includes(ToolName.LIST_WORKLOADS))
        self.assertFalse(catalog.includes("read_secret"))

        self.session.responses["list_workloads"] = response({"workloads": ["alpha", "beta"]})
        self.assertEqual(("alpha", "beta"), await self.client.list_workloads())

        self.session.responses["get_status"] = response(
            {"workload": "alpha", "mode": "online", "state": "ready"}
        )
        status = await self.client.get_status("alpha")
        self.assertEqual("ready", status.state)

        self.session.responses["run_workload"] = response(
            {"id": "job-1", "status": "completed", "exit_code": 0}
        )
        run = await self.client.run_workload("alpha")
        self.assertEqual("job-1", run.identifier)
        self.assertEqual(("run_workload", {"workload": "alpha"}), self.session.calls[-1])

    async def test_disclosure_requires_explicit_interface_and_redacts(self) -> None:
        self.assertFalse(hasattr(self.client, "read_secret"))
        marker = "SYNTHETIC_VALUE_NEVER_LOGGED"
        self.session.responses["read_secret"] = response(
            {"alias": "API_KEY", "value": marker, "version": "v1"}
        )
        disclosure = await self.client.disclosures().read_secret("alpha", "API_KEY")
        self.assertEqual(marker, disclosure.reveal())
        self.assertNotIn(marker, repr(disclosure))
        self.assertNotIn(marker, json.dumps(disclosure.redacted_dict()))

    async def test_rejects_ambiguous_or_unbounded_server_output(self) -> None:
        self.session.responses["list_workloads"] = response(
            {"workloads": ["alpha"], "unexpected": "channel"}
        )
        with self.assertRaisesRegex(CircularQuayError, "protocol"):
            await self.client.list_workloads()

        self.session.responses["get_status"] = response(
            {"workload": "other", "mode": "online", "state": "ready"}
        )
        with self.assertRaises(CircularQuayError):
            await self.client.get_status("alpha")

    async def test_rejects_text_fallback_even_when_text_looks_like_valid_json(self) -> None:
        # v1 consumes typed MCP structuredContent only.  Parsing this text would
        # silently collapse duplicate JSON keys in Python and diverge from peers.
        self.session.responses["list_workloads"] = SimpleNamespace(
            content=[SimpleNamespace(text='{"workloads":["alpha"],"workloads":["beta"]}')],
            isError=False,
        )
        with self.assertRaises(CircularQuayError) as captured:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_rejects_paginated_tool_catalog(self) -> None:
        self.session.tools = [SimpleNamespace(name="list_workloads")]

        async def paginated_tools() -> object:
            return SimpleNamespace(tools=self.session.tools, nextCursor="synthetic-next")

        self.session.list_tools = paginated_tools  # type: ignore[method-assign]
        with self.assertRaises(CircularQuayError) as captured:
            await self.client.discover_tools()
        self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_normalizes_legacy_zero_expiry_to_absent(self) -> None:
        self.session.responses["read_secret"] = response(
            {"alias": "API_KEY", "value": "synthetic", "expires_at": "0001-01-01T00:00:00Z"}
        )
        disclosed = await self.client.disclosures().read_secret("alpha", "API_KEY")
        self.assertIsNone(disclosed.expires_at)

    async def test_accepts_opaque_provider_versions_from_supported_provider_shapes(self) -> None:
        versions = (
            "projects/synthetic/secrets/fixture/versions/7",
            "2026-01-02T03:04:05Z",
            "2026-01-02T03:04:05+10:00",
        )
        for version in versions:
            with self.subTest(version=version):
                self.session.responses["read_secret"] = response(
                    {"alias": "API_KEY", "value": "synthetic", "version": version}
                )
                disclosure = await self.client.disclosures().read_secret("alpha", "API_KEY")
                self.assertEqual(version, disclosure.version)

    async def test_enforces_provider_version_and_secret_utf8_byte_limits(self) -> None:
        version_at_limit = "é" * 2_048
        value_at_limit = "é" * 32_768
        self.session.responses["read_secret"] = response(
            {"alias": "API_KEY", "value": value_at_limit, "version": version_at_limit}
        )
        disclosure = await self.client.disclosures().read_secret("alpha", "API_KEY")
        self.assertEqual(version_at_limit, disclosure.version)
        self.assertEqual(value_at_limit, disclosure.reveal())

        for invalid in (
            {"alias": "API_KEY", "value": "synthetic", "version": "é" * 2_049},
            {"alias": "API_KEY", "value": "synthetic", "version": "version\x7f"},
            {"alias": "API_KEY", "value": "é" * 32_769},
        ):
            with self.subTest(invalid=invalid.keys()):
                self.session.responses["read_secret"] = response(invalid)
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.disclosures().read_secret("alpha", "API_KEY")
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_rejects_an_expired_secret_disclosure(self) -> None:
        self.session.responses["read_secret"] = response(
            {"alias": "API_KEY", "value": "synthetic", "expires_at": "2000-01-01T00:00:00Z"}
        )
        with self.assertRaises(CircularQuayError) as captured:
            await self.client.disclosures().read_secret("alpha", "API_KEY")
        self.assertEqual(ErrorCategory.EXPIRED, captured.exception.category)

    async def test_rejects_malformed_secret_expiry_as_protocol(self) -> None:
        for expiry in ("not-a-timestamp", "2026-01-02 03:04:05Z"):
            with self.subTest(expiry=expiry):
                self.session.responses["read_secret"] = response(
                    {"alias": "API_KEY", "value": "synthetic", "expires_at": expiry}
                )
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.disclosures().read_secret("alpha", "API_KEY")
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_sanitizes_remote_failures_and_observer_metadata(self) -> None:
        marker = "SYNTHETIC_SERVER_TEXT_MUST_NOT_ESCAPE"
        self.session.responses["list_workloads"] = RuntimeError(marker)
        with self.assertRaises(CircularQuayError) as captured:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)
        self.assertNotIn(marker, str(captured.exception))
        self.assertIsNone(captured.exception.__cause__)
        event = asdict(self.events[-1])
        self.assertEqual({"operation", "outcome", "duration_ms"}, set(event))
        self.assertNotIn(marker, json.dumps(event))

    async def test_advisory_observer_failure_does_not_replace_success(self) -> None:
        def broken_observer(_: object) -> None:
            raise RuntimeError("synthetic observer failure")

        client = CircularQuayClient(
            MCPClientSessionAdapter(self.session),
            observer=broken_observer,
        )
        self.session.responses["list_workloads"] = response({"workloads": ["alpha"]})
        self.assertEqual(("alpha",), await client.list_workloads())

    async def test_cancellation_propagates(self) -> None:
        started = asyncio.Event()
        cancelled = asyncio.Event()

        async def blocked(_: str, arguments: dict[str, str]) -> object:
            del arguments
            started.set()
            try:
                await asyncio.Event().wait()
            except asyncio.CancelledError:
                cancelled.set()
                raise

        self.session.call_tool = blocked  # type: ignore[method-assign]
        task = asyncio.create_task(self.client.list_workloads())
        try:
            await asyncio.wait_for(
                started.wait(), timeout=_ASYNC_TEST_DEADLINE_SECONDS
            )
            task.cancel()
            with self.assertRaises(asyncio.CancelledError):
                await task
            self.assertTrue(cancelled.is_set())
            self.assertEqual("cancelled", self.events[-1].outcome)
        finally:
            if not task.done():
                task.cancel()
                await asyncio.gather(task, return_exceptions=True)
