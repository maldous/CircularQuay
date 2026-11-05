"""Behavioral boundary contracts for the Python SDK's safe client interfaces."""

from __future__ import annotations

import hashlib
import json
import math
import os
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

from cq import (
    CircularQuayClient,
    CircularQuayError,
    ErrorCategory,
    LocalCLIClient,
    MCPClientSessionAdapter,
    RunStdio,
)


def _response(value: object, *, error: bool = False) -> object:
    return SimpleNamespace(structuredContent=value, isError=error)


class _Session:
    def __init__(self) -> None:
        self.tools: object = []
        self.responses: dict[str, object] = {}
        self.calls: list[str] = []

    async def list_tools(self) -> object:
        return SimpleNamespace(tools=self.tools)

    async def call_tool(self, name: str, arguments: dict[str, str]) -> object:
        del arguments
        self.calls.append(name)
        result = self.responses[name]
        if isinstance(result, BaseException):
            raise result
        return result


class MCPBoundaryTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.session = _Session()
        self.client = CircularQuayClient(MCPClientSessionAdapter(self.session))

    async def test_server_error_and_transport_failure_have_safe_categories(self) -> None:
        self.session.responses["list_workloads"] = _response({}, error=True)
        with self.assertRaises(CircularQuayError) as denied:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.DENIED, denied.exception.category)

        self.session.responses["list_workloads"] = TimeoutError("synthetic")
        with self.assertRaises(CircularQuayError) as unavailable:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.UNAVAILABLE, unavailable.exception.category)
        self.assertIsNone(unavailable.exception.__cause__)

    async def test_mcp_protocol_rejects_ambiguous_catalog_and_workload_payloads(self) -> None:
        self.session.tools = [
            SimpleNamespace(name="list_workloads"),
            SimpleNamespace(name="list_workloads"),
        ]
        with self.assertRaises(CircularQuayError) as duplicate_catalog:
            await self.client.discover_tools()
        self.assertEqual(ErrorCategory.PROTOCOL, duplicate_catalog.exception.category)

        for workloads in (["beta", "alpha"], ["alpha", "alpha"]):
            with self.subTest(workloads=workloads):
                self.session.responses["list_workloads"] = _response({"workloads": workloads})
                with self.assertRaises(CircularQuayError) as invalid:
                    await self.client.list_workloads()
                self.assertEqual(ErrorCategory.PROTOCOL, invalid.exception.category)

    async def test_status_and_run_response_boundaries_are_protocol_errors(self) -> None:
        self.session.responses["get_status"] = _response(
            {"workload": "alpha", "mode": "unknown", "state": "ready"}
        )
        with self.assertRaises(CircularQuayError) as invalid_status:
            await self.client.get_status("alpha")
        self.assertEqual(ErrorCategory.PROTOCOL, invalid_status.exception.category)

        self.session.responses["run_workload"] = _response(
            {"id": "job", "status": "completed", "exit_code": True}
        )
        with self.assertRaises(CircularQuayError) as invalid_run:
            await self.client.run_workload("alpha")
        self.assertEqual(ErrorCategory.PROTOCOL, invalid_run.exception.category)

    async def test_client_rejects_invalid_call_handles_before_transport(self) -> None:
        with self.assertRaises(CircularQuayError) as invalid:
            await self.client.get_status("../not-a-handle")
        self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)
        self.assertEqual([], self.session.calls)

    async def test_structured_dto_budget_and_timestamp_grammar_fail_closed(self) -> None:
        self.session.responses["list_workloads"] = _response(
            {"workloads": [], "padding": "x" * 1_048_577}
        )
        with self.assertRaises(CircularQuayError) as oversized:
            await self.client.list_workloads()
        self.assertEqual(ErrorCategory.PROTOCOL, oversized.exception.category)

        for value in ("2026-01-02 03:04:05Z", "2026-01-02T03:04:05", "bad"):
            with self.subTest(value=value):
                self.session.responses["read_secret"] = _response(
                    {"alias": "synthetic", "value": "synthetic", "expires_at": value}
                )
                with self.assertRaises(CircularQuayError) as malformed:
                    await self.client.disclosures().read_secret("alpha", "synthetic")
                self.assertEqual(ErrorCategory.PROTOCOL, malformed.exception.category)


class LocalCommandBoundaryTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.directory = tempfile.TemporaryDirectory()
        root = Path(self.directory.name)
        executable = root / "cq"
        executable.write_bytes(b"synthetic-cq")
        os.chmod(executable, 0o700)
        self.client = LocalCLIClient(
            executable,
            hashlib.sha256(executable.read_bytes()).hexdigest(),
            root / "policy.json",
            root / "state",
            environment={},
        )
        self.calls: list[tuple[str, tuple[str, ...], bytes | None]] = []

    def tearDown(self) -> None:
        self.directory.cleanup()

    async def _captured(
        self, operation: str, arguments: tuple[str, ...], *, stdin: bytes | None = None
    ) -> tuple[bytes, int, ErrorCategory | None]:
        self.calls.append((operation, arguments, stdin))
        value: object = {}
        if operation == "version":
            value = {"version": "1.0.0", "commit": "synthetic", "go": "1.27", "maturity": "stable"}
        return json.dumps(value).encode("utf-8"), 0, None

    async def test_json_commands_keep_values_on_stdin_and_use_exact_cli_arguments(self) -> None:
        with mock.patch.object(self.client, "_captured", new=self._captured):
            root = Path(self.directory.name)
            self.assertEqual("1.0.0", (await self.client.version())["version"])
            await self.client.assert_runtime_compatibility()
            await self.client.providers()
            await self.client.update_check()
            await self.client.init(output=root / "new.json")
            await self.client.config_init(output=root / "config.json")
            await self.client.doctor(fix=True)
            await self.client.config_validate()
            await self.client.config_get("/workloads/alpha")
            await self.client.config_list("/workloads")
            await self.client.config_set("/workloads/alpha", {"enabled": True})
            await self.client.config_delete("/workloads/alpha", expect_sha256="a" * 64)
            await self.client.config_replace({"workloads": []})
            await self.client.plan()
            await self.client.status("alpha")
            await self.client.snapshot("alpha", target="user")
            await self.client.apply("alpha", target="user")
            await self.client.sync("alpha")
            await self.client.bootstrap(output=root / "bootstrap.json")

        set_call = next(call for call in self.calls if call[0] == "config_set")
        self.assertEqual(
            ("config", "set", "--pointer", "/workloads/alpha", "--value-file", "-"),
            set_call[1],
        )
        self.assertEqual(b'{"enabled":true}', set_call[2])
        self.assertEqual(("update", "check", "--json"), self.calls[3][1])
        self.assertIn(("sync", ("sync", "--workload", "alpha", "--once"), None), self.calls)

    async def test_inherited_commands_keep_streams_out_of_sdk_memory(self) -> None:
        calls: list[tuple[str, tuple[str, ...]]] = []

        async def inherited(operation: str, arguments: tuple[str, ...]) -> int:
            calls.append((operation, arguments))
            return 7

        with mock.patch.object(self.client, "_inherit", new=inherited):
            flow = await self.client.flow("alpha")
            run = await self.client.run(
                "alpha", override=("/bin/synthetic", "--mode"), stdio=RunStdio.INHERIT
            )

        self.assertEqual((7, 7), (flow, run))
        self.assertEqual(
            ("flow", ("flow", "run", "--workload", "alpha", "--non-interactive")), calls[0]
        )
        self.assertEqual(
            ("run", ("run", "--workload", "alpha", "--", "/bin/synthetic", "--mode")),
            calls[1],
        )

    async def test_local_boundary_rejections_do_not_spawn_a_process(self) -> None:
        calls: list[str] = []

        async def unexpected_captured(*_: object, **__: object) -> object:
            calls.append("captured")
            raise AssertionError("invalid input reached captured subprocess boundary")

        async def unexpected_inherited(*_: object, **__: object) -> object:
            calls.append("inherited")
            raise AssertionError("invalid input reached inherited subprocess boundary")

        for operation in (
            lambda: self.client.doctor(fix=True, identity_only=True),
            lambda: self.client.config_get("/~2"),
            lambda: self.client.config_set("/a", math.nan),
            lambda: self.client.config_replace({"n": 9_007_199_254_740_992}),
            lambda: self.client.apply("alpha", target=None),
            lambda: self.client.run("alpha", override=("relative",), stdio=RunStdio.INHERIT),
        ):
            with (
                self.subTest(operation=operation),
                mock.patch.object(self.client, "_captured", new=unexpected_captured),
                mock.patch.object(self.client, "_inherit", new=unexpected_inherited),
            ):
                with self.assertRaises(CircularQuayError) as invalid:
                    await operation()
                self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)
        self.assertEqual([], calls)

    async def test_native_json_failures_are_sanitized_and_typed(self) -> None:
        async def invalid_json(
            _: str, __: tuple[str, ...], *, stdin: bytes | None = None
        ) -> tuple[bytes, int, ErrorCategory | None]:
            del stdin
            return b"not-json", 0, None

        with mock.patch.object(self.client, "_captured", new=invalid_json):
            with self.assertRaises(CircularQuayError) as malformed:
                await self.client.providers()
        self.assertEqual(ErrorCategory.PROTOCOL, malformed.exception.category)

        async def denied(
            _: str, __: tuple[str, ...], *, stdin: bytes | None = None
        ) -> tuple[bytes, int, ErrorCategory | None]:
            del stdin
            return b"", 3, ErrorCategory.DENIED

        with mock.patch.object(self.client, "_captured", new=denied):
            with self.assertRaises(CircularQuayError) as rejected:
                await self.client.plan()
        self.assertEqual(ErrorCategory.DENIED, rejected.exception.category)

    async def test_remote_pins_require_https_and_all_immutable_fields_before_spawn(self) -> None:
        ref = "a" * 40
        digest = "b" * 64
        calls: list[str] = []

        async def unexpected_captured(*_: object, **__: object) -> object:
            calls.append("captured")
            raise AssertionError("invalid pins reached captured subprocess boundary")

        async def unexpected_inherited(*_: object, **__: object) -> object:
            calls.append("inherited")
            raise AssertionError("invalid pins reached inherited subprocess boundary")

        with (
            mock.patch.object(self.client, "_captured", new=unexpected_captured),
            mock.patch.object(self.client, "_inherit", new=unexpected_inherited),
        ):
            for source in (
                "http://example.test/policy",
                "https://token@example.test/policy",
                "https://example.test/policy?x=1",
            ):
                with self.subTest(source=source):
                    with self.assertRaises(CircularQuayError) as invalid:
                        await self.client.bootstrap(
                            config_source=source,
                            config_ref=ref,
                            config_path="policy.json",
                            config_sha256=digest,
                        )
                    self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)
            with self.assertRaises(CircularQuayError) as incomplete:
                await self.client.flow("alpha", config_source="https://example.test/policy")
            self.assertEqual(ErrorCategory.INVALID, incomplete.exception.category)
        self.assertEqual([], calls)
