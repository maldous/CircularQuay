"""Contract tests for the Python SDK using synthetic, in-memory MCP shapes."""

from __future__ import annotations

import asyncio
import json
import unittest
from dataclasses import asdict
from types import SimpleNamespace

from cq import (
    CircularQuayClient,
    CircularQuayError,
    ErrorCategory,
    MCPClientSessionAdapter,
    ToolName,
)
from cq import client as client_module

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

    def test_requires_explicit_non_null_session_and_adapter(self) -> None:
        with self.assertRaisesRegex(CircularQuayError, "mcp_session"):
            MCPClientSessionAdapter(None)  # type: ignore[arg-type]
        with self.assertRaisesRegex(CircularQuayError, "mcp_adapter"):
            CircularQuayClient(None)  # type: ignore[arg-type]

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

    async def test_rejects_invalid_complete_workload_and_run_contracts(self) -> None:
        invalid_workloads = (
            {"workloads": ["beta", "alpha"]},
            {"workloads": ["alpha", "alpha"]},
            {"workloads": ["."]},
            {"workloads": "alpha"},
        )
        for payload in invalid_workloads:
            with self.subTest(workloads=payload):
                self.session.responses["list_workloads"] = response(payload)
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.list_workloads()
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

        for payload in (
            {"workload": "alpha", "mode": "invalid", "state": "ready"},
            {"workload": "alpha", "mode": "online", "state": "invalid"},
        ):
            with self.subTest(status=payload):
                self.session.responses["get_status"] = response(payload)
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.get_status("alpha")
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

        for payload in (
            {"id": "job-1", "status": "invalid", "exit_code": 0},
            {"id": "job-1", "status": "completed", "exit_code": True},
            {"id": "job-1", "status": "completed", "exit_code": 256},
        ):
            with self.subTest(run=payload):
                self.session.responses["run_workload"] = response(payload)
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.run_workload("alpha")
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_rejects_denied_and_unstructured_mcp_responses(self) -> None:
        self.session.responses["list_workloads"] = response({}, is_error=True)
        with self.assertRaises(CircularQuayError) as denied:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.DENIED, denied.exception.category)

        self.session.responses["list_workloads"] = SimpleNamespace(is_error=False)
        with self.assertRaises(CircularQuayError) as missing:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, missing.exception.category)

        self.session.responses["list_workloads"] = response({"workloads": [], "extra": None})
        with self.assertRaises(CircularQuayError) as extra:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, extra.exception.category)

    async def test_rejects_non_mapping_and_non_encodable_structured_content(self) -> None:
        self.session.responses["list_workloads"] = response(["alpha"])
        with self.assertRaises(CircularQuayError) as non_mapping:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, non_mapping.exception.category)

        self.session.responses["list_workloads"] = SimpleNamespace(
            structuredContent={"workloads": {"alpha"}}, content=[], isError=False
        )
        with self.assertRaises(CircularQuayError) as non_encodable:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, non_encodable.exception.category)

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

    async def test_rejects_unknown_duplicate_and_oversized_catalogs(self) -> None:
        for tools in (
            [SimpleNamespace(name="unknown")],
            [SimpleNamespace(name="list_workloads"), SimpleNamespace(name="list_workloads")],
            [SimpleNamespace(name="list_workloads")] * 5,
        ):
            with self.subTest(tools=tools):
                self.session.tools = tools
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
        for expiry in ("not-a-timestamp", "2026-01-02 03:04:05Z", "2026-02-31T03:04:05Z"):
            with self.subTest(expiry=expiry):
                self.session.responses["read_secret"] = response(
                    {"alias": "API_KEY", "value": "synthetic", "expires_at": expiry}
                )
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.disclosures().read_secret("alpha", "API_KEY")
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_rejects_secret_alias_mismatch_and_invalid_handles_before_transport(self) -> None:
        self.session.responses["read_secret"] = response(
            {"alias": "OTHER", "value": "synthetic"}
        )
        with self.assertRaises(CircularQuayError) as mismatch:
            await self.client.disclosures().read_secret("alpha", "API_KEY")
        self.assertEqual(ErrorCategory.PROTOCOL, mismatch.exception.category)
        calls_before = len(self.session.calls)
        for workload, key in (("", "API_KEY"), ("alpha", "bad/key")):
            with self.subTest(workload=workload, key=key):
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.disclosures().read_secret(workload, key)
                self.assertEqual(ErrorCategory.INVALID, captured.exception.category)
        self.assertEqual(calls_before, len(self.session.calls))

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

    async def test_normalizes_timeout_and_os_errors_without_retaining_causes(self) -> None:
        for failure in (TimeoutError("synthetic"), OSError("synthetic")):
            with self.subTest(failure=type(failure).__name__):
                self.session.responses["list_workloads"] = failure
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.list_workloads()
                self.assertEqual(ErrorCategory.UNAVAILABLE, captured.exception.category)
                self.assertIsNone(captured.exception.__cause__)

    async def test_accepts_success_without_an_observer(self) -> None:
        client = CircularQuayClient(MCPClientSessionAdapter(self.session))
        self.session.responses["list_workloads"] = response({"workloads": ["alpha"]})
        self.assertEqual(("alpha",), await client.list_workloads())

    def test_rejects_non_utf8_secret_text_without_raising_encoder_error(self) -> None:
        self.assertFalse(client_module._bounded_utf8_text("\ud800", 1))

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

    def test_structured_dto_budget_fails_closed_for_non_encodable_and_oversized_values(
        self,
    ) -> None:
        self.assertFalse(client_module._within_structured_budget({"value": {1, 2}}))
        self.assertFalse(
            client_module._within_structured_budget(
                {"value": "x" * (client_module._MAX_STRUCTURED_BYTES + 1)}
            )
        )

    async def test_mapping_responses_accept_protocol_field_spellings(self) -> None:
        async def catalog() -> object:
            return {"tools": [{"name": "list_workloads"}], "next_cursor": ""}

        self.session.list_tools = catalog  # type: ignore[method-assign]
        self.assertEqual((ToolName.LIST_WORKLOADS,), (await self.client.discover_tools()).available)
        self.session.responses["list_workloads"] = {
            "structured_content": {"workloads": ["alpha"]}, "is_error": False
        }
        self.assertEqual(("alpha",), await self.client.list_workloads())
        self.session.responses["list_workloads"] = {"structuredContent": {"workloads": []}}
        self.assertEqual((), await self.client.list_workloads())
        self.session.responses["list_workloads"] = {}
        with self.assertRaises(CircularQuayError) as captured:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_malformed_catalog_shapes_fail_without_partial_authorization(self) -> None:
        for raw in ({}, {"tools": "list_workloads"}, {"tools": [{}]}, {"tools": [None]}):
            with self.subTest(raw=raw):
                async def catalog(result: object = raw) -> object:
                    return result

                self.session.list_tools = catalog  # type: ignore[method-assign]
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.discover_tools()
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_workload_catalog_accepts_exact_limit_and_rejects_larger_catalog(self) -> None:
        workloads = [f"workload-{index:04}" for index in range(client_module._MAX_WORKLOADS)]
        self.session.responses["list_workloads"] = response({"workloads": workloads})
        self.assertEqual(tuple(workloads), await self.client.list_workloads())
        self.session.responses["list_workloads"] = response(
            {"workloads": [*workloads, "workload-over-limit"]}
        )
        with self.assertRaises(CircularQuayError) as captured:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_protocol_types_are_required_for_status_run_and_secret(self) -> None:
        cases = (
            ("get_status", {"workload": "alpha", "mode": None, "state": "ready"}),
            ("get_status", {"workload": "alpha", "mode": "online", "state": 1}),
            ("run_workload", {"id": "run-1", "status": None, "exit_code": 0}),
            ("run_workload", {"id": "run-1", "status": "completed", "exit_code": "0"}),
            ("run_workload", {"id": "run-1", "status": "completed", "exit_code": -1}),
            ("read_secret", {"alias": "API_KEY", "value": None}),
            ("read_secret", {"alias": "API_KEY", "value": "synthetic", "version": 1}),
            ("read_secret", {"alias": "API_KEY", "value": "synthetic", "version": "a\nb"}),
        )
        for operation, payload in cases:
            with self.subTest(operation=operation, fields=tuple(payload)):
                self.session.responses[operation] = response(payload)
                with self.assertRaises(CircularQuayError) as captured:
                    if operation == "get_status":
                        await self.client.get_status("alpha")
                    elif operation == "run_workload":
                        await self.client.run_workload("alpha")
                    else:
                        await self.client.disclosures().read_secret("alpha", "API_KEY")
                self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

    async def test_invalid_handles_fail_before_any_transport_call(self) -> None:
        for workload in (None, ".", "..", "a" * 129, "café", "bad/handle"):
            with self.subTest(workload=workload):
                with self.assertRaises(CircularQuayError) as captured:
                    await self.client.get_status(workload)  # type: ignore[arg-type]
                self.assertEqual(ErrorCategory.INVALID, captured.exception.category)
        self.assertEqual([], self.session.calls)

    async def test_unrecognized_structured_fields_and_non_string_keys_are_rejected(self) -> None:
        for payload in ({str(index): None for index in range(9)}, {1: "synthetic"}):
            self.session.responses["list_workloads"] = SimpleNamespace(structuredContent=payload)
            with self.assertRaises(CircularQuayError) as captured:
                await self.client.list_workloads()
            self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)
