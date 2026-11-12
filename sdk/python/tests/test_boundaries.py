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
    local,
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
        # The production identity check intentionally requires a canonical
        # executable path; temporary roots can pass through platform aliases.
        self.root = Path(self.directory.name).resolve()
        root = self.root
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
            root = self.root
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
        replace_call = next(call for call in self.calls if call[0] == "config_replace")
        self.assertEqual(("config", "replace", "--file", "-"), replace_call[1])
        self.assertEqual(b'{"workloads":[]}', replace_call[2])
        self.assertEqual(("update", "check", "--json"), self.calls[3][1])
        self.assertIn(("sync", ("sync", "--workload", "alpha", "--once"), None), self.calls)

    async def test_inherited_commands_keep_streams_out_of_sdk_memory(self) -> None:
        calls: list[tuple[str, tuple[str, ...]]] = []
        program = str(self.root / "synthetic")

        async def inherited(operation: str, arguments: tuple[str, ...]) -> int:
            calls.append((operation, arguments))
            return 7

        with mock.patch.object(self.client, "_inherit", new=inherited):
            flow = await self.client.flow("alpha")
            run = await self.client.run(
                "alpha", override=(program, "--mode"), stdio=RunStdio.INHERIT
            )

        self.assertEqual((7, 7), (flow, run))
        self.assertEqual(
            ("flow", ("flow", "run", "--workload", "alpha", "--non-interactive")), calls[0]
        )
        self.assertEqual(
            ("run", ("run", "--workload", "alpha", "--", program, "--mode")),
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

    async def test_optional_command_arguments_preserve_native_contract(self) -> None:
        inherited_calls: list[tuple[str, tuple[str, ...]]] = []

        async def inherited(operation: str, arguments: tuple[str, ...]) -> int:
            inherited_calls.append((operation, arguments))
            return 0

        pins = {
            "config_source": "https://example.test/policy",
            "config_ref": "a" * 40,
            "config_path": "policy.json",
            "config_sha256": "b" * 64,
        }
        expected_pins = (
            "--config-source", "https://example.test/policy", "--config-ref", "a" * 40,
            "--config-path", "policy.json", "--config-sha256", "b" * 64,
        )
        with (
            mock.patch.object(self.client, "_captured", new=self._captured),
            mock.patch.object(self.client, "_inherit", new=inherited),
        ):
            await self.client.init()
            await self.client.config_init()
            await self.client.doctor(identity_only=True)
            await self.client.config_list()
            await self.client.config_get("/escaped~0name~1path")
            await self.client.bootstrap(**pins)
            await self.client.flow("alpha", **pins)
            await self.client.run("alpha", stdio=RunStdio.INHERIT)

        self.assertIn(("init", ("init",), None), self.calls)
        self.assertIn(("config_init", ("config", "init"), None), self.calls)
        self.assertIn(("doctor", ("doctor", "--identity-only"), None), self.calls)
        self.assertIn(("config_list", ("config", "list"), None), self.calls)
        self.assertIn(("bootstrap", ("bootstrap", "--non-interactive", *expected_pins), None),
                      self.calls)
        self.assertEqual(
            ("flow", ("flow", "run", "--workload", "alpha", "--non-interactive",
                      *expected_pins)), inherited_calls[0],
        )
        self.assertEqual(("run", ("run", "--workload", "alpha")), inherited_calls[1])

    async def test_invalid_pin_fields_and_handles_fail_before_transport(self) -> None:
        defaults: dict[str, object] = {
            "config_source": "https://example.test/policy",
            "config_ref": "a" * 40,
            "config_path": "policy.json",
            "config_sha256": "b" * 64,
        }
        invalid_pins = (
            ("config_source", ""), ("config_source", 1),
            ("config_source", "x" * 4097), ("config_source", "https://example.test/\x00"),
            ("config_source", "https://example.test:invalid/policy"),
            ("config_source", "https://[broken/policy"),
            ("config_ref", "A" * 40), ("config_sha256", "b" * 63),
            ("config_path", ""), ("config_path", "x" * 1025), ("config_path", 1),
        )
        with mock.patch.object(self.client, "_captured", new=self._captured):
            for field, value in invalid_pins:
                with self.subTest(field=field, value=value):
                    with self.assertRaises(CircularQuayError) as invalid:
                        await self.client.bootstrap(**{**defaults, field: value})
                    self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)
            for pointer in ("relative", "/\x00", "x" * 4097, None):
                with self.subTest(pointer=pointer):
                    with self.assertRaises(CircularQuayError):
                        await self.client.config_get(pointer)
            for handle in ("", ".", "..", "x" * 129, "a/b", None):
                with self.subTest(handle=handle):
                    with self.assertRaises(CircularQuayError):
                        await self.client.status(handle)
        self.assertEqual([], self.calls)
        for argument in ("\x00", "x" * 4097, None):
            with self.subTest(argument=argument):
                with self.assertRaises(CircularQuayError):
                    await self.client.run("alpha", override=(argument,), stdio=RunStdio.INHERIT)

    def test_remote_pins_are_either_absent_or_complete(self) -> None:
        complete = ("https://example.test/policy", "a" * 40, "policy.json", "b" * 64)
        for mask in range(16):
            pins = tuple(value if mask & (1 << index) else None
                         for index, value in enumerate(complete))
            with self.subTest(present=mask):
                if mask == 0:
                    self.assertIsNone(local._pinned_config_arguments(*pins, "flow_pins"))
                elif mask == 15:
                    self.assertIsNotNone(local._pinned_config_arguments(*pins, "flow_pins"))
                else:
                    with self.assertRaises(CircularQuayError) as invalid:
                        local._pinned_config_arguments(*pins, "flow_pins")
                    self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)

    def test_executable_identity_rejects_invalid_digest_alias_and_nonregular_file(self) -> None:
        executable = self.root / "cq"
        for digest in ("", "a" * 63, "A" * 64, None):
            with self.subTest(digest=digest):
                with self.assertRaises(CircularQuayError) as invalid:
                    LocalCLIClient(executable, digest, self.root / "policy", self.root / "state")
                self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)
        directory = self.root / "directory"
        directory.mkdir()
        for path in (directory, directory / ".." / "cq"):
            with self.subTest(path=path):
                with self.assertRaises(CircularQuayError) as invalid:
                    LocalCLIClient(path, "a" * 64, self.root / "policy", self.root / "state")
                self.assertEqual(ErrorCategory.INTEGRITY, invalid.exception.category)

    def test_environment_accepts_only_explicit_safe_entries(self) -> None:
        executable = self.root / "cq"
        digest = hashlib.sha256(executable.read_bytes()).hexdigest()
        for environment in (None, {"SYNTHETIC_CQ_OPTION": "enabled"}):
            client = LocalCLIClient(
                executable, digest, self.root / "policy", self.root / "state",
                environment=environment,
            )
            client.verify_identity()
        for environment in (
            {1: "value"}, {"NAME": 1}, {"": "value"}, {"x" * 257: "value"},
            {"NAME": "x" * 4097}, {"A=B": "value"}, {"A\x00": "value"},
            {"NAME": "\x00"}, {"vault_token": "synthetic"},
        ):
            with self.subTest(environment=environment):
                with self.assertRaises(CircularQuayError) as invalid:
                    LocalCLIClient(
                        executable, digest, self.root / "policy", self.root / "state",
                        environment=environment,
                    )
                self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)

    async def test_json_input_rejects_nonportable_values_before_capture(self) -> None:
        deep: object = None
        for _ in range(34):
            deep = [deep]
        invalid_values = (
            deep, object(), {"n": math.inf}, {"x": "\ud800"},
            {"x": "a" * 1_048_577}, {str(index): None for index in range(1025)},
            {1: "nonstring-key"}, [None] * 1025,
        )
        with mock.patch.object(self.client, "_captured", new=self._captured):
            for value in invalid_values:
                with self.subTest(kind=type(value).__name__):
                    with self.assertRaises(CircularQuayError) as invalid:
                        await self.client.config_replace(value)
                    self.assertEqual(ErrorCategory.INVALID, invalid.exception.category)
        self.assertEqual([], self.calls)

    async def test_native_json_rejects_unsafe_values_and_nonobject_command_results(self) -> None:
        deep: object = None
        for _ in range(34):
            deep = [deep]
        for value in (deep, math.nan, {str(index): None for index in range(1025)}):
            transport = mock.AsyncMock(return_value=(json.dumps(value).encode(), 0, None))
            with mock.patch.object(self.client, "_captured", new=transport):
                with self.assertRaises(CircularQuayError) as invalid:
                    await self.client.providers()
                self.assertEqual(ErrorCategory.PROTOCOL, invalid.exception.category)
        for value in (None, True, 1, 1.5, [], ["safe"]):
            transport = mock.AsyncMock(return_value=(json.dumps(value).encode(), 0, None))
            with mock.patch.object(self.client, "_captured", new=transport):
                self.assertEqual(value, await self.client.providers())
                with self.assertRaises(CircularQuayError):
                    await self.client.plan()

    async def test_version_requires_text_metadata_and_strict_semver(self) -> None:
        metadata: dict[str, object] = {
            "version": "1.0.0", "commit": "synthetic", "go": "1.27", "maturity": "stable",
        }
        for field, value in (
            ("commit", None), ("version", 1), ("version", "1.0.0\ud800"),
            ("version", "1.0.0+a+b"), ("version", "1.0.0+"),
            ("version", "1.0.0+bad build"),
        ):
            transport = mock.AsyncMock(
                return_value=(json.dumps({**metadata, field: value}).encode(), 0, None)
            )
            with mock.patch.object(self.client, "_captured", new=transport):
                with self.assertRaises(CircularQuayError) as invalid:
                    await self.client.version()
                self.assertEqual(ErrorCategory.PROTOCOL, invalid.exception.category)

    def test_safe_json_and_cli_error_parsers_reject_untyped_shapes(self) -> None:
        self.assertFalse(local._safe_json(object()))
        self.assertFalse(local._safe_json({1: "value"}))
        self.assertFalse(local._safe_json("a" * 1_048_577))
        self.assertEqual(ErrorCategory.PROTOCOL, local._cli_failure_category(b"x" * 513))
        self.assertEqual(ErrorCategory.PROTOCOL, local._cli_failure_category(b'{"error":null}'))
