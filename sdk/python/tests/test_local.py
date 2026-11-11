"""Source-level behavioral contracts for the integrity-pinned CLI adapter."""

from __future__ import annotations

import asyncio
import hashlib
import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from cq import CircularQuayError, ErrorCategory, LocalCLIClient, RunStdio, local


class _OverflowReader:
    def __init__(self) -> None:
        self.sent = False

    async def read(self, _: int) -> bytes:
        if self.sent:
            return b""
        self.sent = True
        return b"x" * (local._MAX_OUTPUT_BYTES + 1)


class _BlockingReader:
    def __init__(self) -> None:
        self.cancelled = False

    async def read(self, _: int) -> bytes:
        try:
            await asyncio.Event().wait()
        except asyncio.CancelledError:
            self.cancelled = True
            raise
        return b""


class _HostileProcess:
    def __init__(self) -> None:
        self.stdin = None
        self.stdout = _OverflowReader()
        self.stderr = _BlockingReader()
        self.returncode: int | None = None
        self.exited = asyncio.Event()
        self.terminated = False
        self.killed = False

    def terminate(self) -> None:
        self.terminated = True  # Deliberately ignores TERM.

    def kill(self) -> None:
        self.killed = True
        self.returncode = -9
        self.exited.set()

    async def wait(self) -> int:
        await self.exited.wait()
        return self.returncode or -9


class _FinishedProcess:
    def __init__(self, code: int) -> None:
        self.returncode = code

    async def wait(self) -> int:
        return self.returncode


class _CapturedProcess(_FinishedProcess):
    def __init__(self, code: int, output: bytes, error: bytes) -> None:
        super().__init__(code)
        self.stdin = None
        self.stdout = asyncio.StreamReader()
        self.stdout.feed_data(output)
        self.stdout.feed_eof()
        self.stderr = asyncio.StreamReader()
        self.stderr.feed_data(error)
        self.stderr.feed_eof()


class _InputWriter:
    def __init__(self, failure: BaseException | None = None) -> None:
        self.failure = failure
        self.writes: list[bytes] = []
        self.closed = False
        self.draining = asyncio.Event()
        self.release = asyncio.Event()

    def write(self, data: bytes) -> None:
        self.writes.append(data)

    async def drain(self) -> None:
        self.draining.set()
        if self.failure is not None:
            raise self.failure
        await self.release.wait()

    def close(self) -> None:
        self.closed = True


class _InputProcess:
    def __init__(self, writer: _InputWriter) -> None:
        self.stdin = writer
        self.stdout = None
        self.stderr = None
        self.returncode: int | None = None
        self.terminated = False
        self.waited = False
        self.exited = asyncio.Event()

    def terminate(self) -> None:
        self.terminated = True
        self.returncode = -15
        self.exited.set()

    def kill(self) -> None:
        self.returncode = -9
        self.exited.set()

    async def wait(self) -> int:
        self.waited = True
        await self.exited.wait()
        return self.returncode or -15


class LocalCLIClientTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.directory = tempfile.TemporaryDirectory()
        # The production identity check intentionally requires a canonical
        # executable path; temporary roots can pass through platform aliases.
        self.root = Path(self.directory.name).resolve()
        root = self.root
        self.executable = root / "cq"
        self.executable.write_bytes(b"synthetic-cq-binary")
        os.chmod(self.executable, 0o700)
        digest = hashlib.sha256(self.executable.read_bytes()).hexdigest()
        self.client = LocalCLIClient(
            self.executable,
            digest,
            root / "policy.json",
            root / "state",
            environment={},
        )

    def tearDown(self) -> None:
        self.directory.cleanup()

    def test_requires_absolute_identity_pinned_paths(self) -> None:
        with self.assertRaises(CircularQuayError) as captured:
            LocalCLIClient(
                "relative-cq", "0" * 64, "/config.json", "/state", environment={}
            )
        self.assertEqual(ErrorCategory.INVALID, captured.exception.category)

    def test_detects_binary_replacement_before_process_start(self) -> None:
        self.executable.write_bytes(b"replacement")
        with self.assertRaises(CircularQuayError) as captured:
            self.client.verify_identity()
        self.assertEqual(ErrorCategory.INTEGRITY, captured.exception.category)

    async def test_rejects_ambient_provider_environment_and_invalid_run_stdio(self) -> None:
        with self.assertRaises(CircularQuayError):
            LocalCLIClient(
                self.executable,
                hashlib.sha256(self.executable.read_bytes()).hexdigest(),
                self.root / "policy.json",
                self.root / "state",
                environment={"AWS_SECRET_ACCESS_KEY": "synthetic"},
            )
        with self.assertRaises(CircularQuayError):
            await self.client.run("alpha", stdio="capture")  # type: ignore[arg-type]

    def test_run_stdio_requires_explicit_enum(self) -> None:
        self.assertEqual(RunStdio.INHERIT, RunStdio("inherit"))

    async def test_update_check_uses_bounded_json_without_a_prompt_or_install(self) -> None:
        calls: list[tuple[str, tuple[str, ...], bytes | None]] = []

        async def fake_transport(
            operation: str, arguments: tuple[str, ...], *, stdin: bytes | None = None
        ) -> tuple[bytes, int, ErrorCategory | None]:
            calls.append((operation, arguments, stdin))
            return (
                b'{"current_version":"1.0.0","latest_version":"1.0.1",'
                b'"update_available":true}',
                0,
                None,
            )

        with mock.patch.object(self.client, "_captured", new=fake_transport):
            update = await self.client.update_check()

        self.assertEqual(
            {
                "current_version": "1.0.0",
                "latest_version": "1.0.1",
                "update_available": True,
            },
            update,
        )
        self.assertEqual([("update_check", ("update", "check", "--json"), None)], calls)

    async def test_version_classifies_malformed_dto_and_incompatible_major(self) -> None:
        async def malformed(_: str, __: tuple[str, ...], **___: object) -> object:
            return {"version": "1.0.0"}

        with mock.patch.object(self.client, "_json_object", new=malformed):
            with self.assertRaises(CircularQuayError) as captured:
                await self.client.version()
        self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)

        async def incompatible(_: str, __: tuple[str, ...], **___: object) -> object:
            return {"version": "2.0.0", "commit": "synthetic", "go": "1.27", "maturity": "stable"}

        with mock.patch.object(self.client, "_json_object", new=incompatible):
            with self.assertRaises(CircularQuayError) as captured:
                await self.client.version()
        self.assertEqual(ErrorCategory.UNSUPPORTED, captured.exception.category)

    def test_version_semver_gate_matches_cross_sdk_contract(self) -> None:
        metadata = {"commit": "synthetic", "go": "1.27", "maturity": "stable"}
        for version in (
            "1.0.0",
            "v1.0.0",
            "1.0.0-dev",
            "1.0.0-alpha.1",
            "1.0.0-0",
            "1.0.0+build.1",
            "1.0.0-alpha.1+build.1",
        ):
            with self.subTest(version=version):
                self.assertEqual("1", local._version_major({"version": version, **metadata}))
        self.assertEqual("2", local._version_major({"version": "2.0.0", **metadata}))
        for version in (
            "1.invalid",
            "1",
            "1.2",
            "2",
            "v1",
            "1.01.0",
            "1.0.0-alpha..1",
            "1.0.0-01",
            "1.0.0-alpha.01",
            "",
            " 1.0.0",
            "1.0.0\n",
        ):
            with self.subTest(version=version):
                self.assertIsNone(local._version_major({"version": version, **metadata}))

    async def test_stdout_overflow_kills_term_ignoring_child_and_cleans_reader_tasks(self) -> None:
        process = _HostileProcess()

        async def create_process(*_: object, **__: object) -> _HostileProcess:
            return process

        with (
            mock.patch(
                "cq.local.asyncio.create_subprocess_exec", new=create_process
            ),
            mock.patch.object(local, "_TERM_GRACE_SECONDS", 0.001),
        ):
            with self.assertRaises(CircularQuayError) as captured:
                await self.client._captured("synthetic_overflow", ())
        self.assertEqual(ErrorCategory.PROTOCOL, captured.exception.category)
        self.assertTrue(process.terminated)
        self.assertTrue(process.killed)
        self.assertTrue(process.stderr.cancelled)

    async def test_captured_commands_classify_exit_without_disclosing_stderr(self) -> None:
        events: list[object] = []
        self.client._observer = events.append
        cases = (
            (0, b'{"workload":"alpha"}', b"", None),
            (1, b"", b'{"error":"access_denied"}', ErrorCategory.DENIED),
            (1, b"", b"", ErrorCategory.PROCESS),
            (1, b"", b"synthetic-private-diagnostic", ErrorCategory.PROTOCOL),
        )
        for code, output, error, category in cases:
            with self.subTest(category=category):
                process = _CapturedProcess(code, output, error)
                with mock.patch(
                    "cq.local.asyncio.create_subprocess_exec",
                    new=mock.AsyncMock(return_value=process),
                ):
                    if category is None:
                        self.assertEqual({"workload": "alpha"}, await self.client.status("alpha"))
                    else:
                        with self.assertRaises(CircularQuayError) as captured:
                            await self.client.status("alpha")
                        self.assertEqual(category, captured.exception.category)
                        self.assertNotIn("synthetic-private", str(captured.exception))
        self.assertEqual(4, len(events))

    async def test_launch_failures_have_safe_categories_and_observations(self) -> None:
        events: list[object] = []
        self.client._observer = events.append
        for failure, category in (
            (TimeoutError("synthetic-private"), ErrorCategory.UNAVAILABLE),
            (OSError("synthetic-private"), ErrorCategory.PROCESS),
            (RuntimeError("synthetic-private"), ErrorCategory.PROCESS),
        ):
            with self.subTest(category=category):
                with mock.patch(
                    "cq.local.asyncio.create_subprocess_exec",
                    new=mock.AsyncMock(side_effect=failure),
                ):
                    with self.assertRaises(CircularQuayError) as captured:
                        await self.client.status("alpha")
                self.assertEqual(category, captured.exception.category)
                self.assertNotIn("synthetic-private", str(captured.exception))
                self.assertEqual(str(category), events[-1].outcome)

    async def test_cancellation_during_launch_propagates_without_an_owned_child(self) -> None:
        with mock.patch(
            "cq.local.asyncio.create_subprocess_exec",
            new=mock.AsyncMock(side_effect=asyncio.CancelledError),
        ):
            with self.assertRaises(asyncio.CancelledError):
                await self.client.status("alpha")

    async def test_missing_stdin_pipe_fails_and_reaps_owned_child(self) -> None:
        process = _InputProcess(_InputWriter())
        process.stdin = None
        with mock.patch(
            "cq.local.asyncio.create_subprocess_exec",
            new=mock.AsyncMock(return_value=process),
        ):
            with self.assertRaises(CircularQuayError) as captured:
                await self.client.config_replace({"version": 1})
        self.assertEqual(ErrorCategory.PROCESS, captured.exception.category)
        self.assertTrue(process.terminated)
        self.assertTrue(process.waited)

    async def test_absent_output_stream_and_empty_cleanup_are_safe(self) -> None:
        self.assertEqual((b"", False), await local._read_bounded(None))
        await local._cleanup_reader_tasks(None)

    async def test_overflow_with_unreapable_child_is_a_process_failure(self) -> None:
        process = _HostileProcess()
        with mock.patch.object(local, "_reap", new=mock.AsyncMock(return_value=False)):
            with self.assertRaises(CircularQuayError) as captured:
                await local._capture_process_output(process, "status")
        self.assertEqual(ErrorCategory.PROCESS, captured.exception.category)
        self.assertTrue(process.stderr.cancelled)

    async def test_reader_failure_reaps_child_before_propagating(self) -> None:
        process = _InputProcess(_InputWriter())
        reader = mock.AsyncMock()
        reader.read.side_effect = OSError("synthetic")
        process.stdout = reader
        with self.assertRaises(OSError):
            await local._capture_process_output(process, "status")
        self.assertTrue(process.terminated)
        self.assertTrue(process.waited)

    async def test_reaping_handles_exit_races_and_unresponsive_children(self) -> None:
        self.assertTrue(await local._reap(_FinishedProcess(0)))
        # A transport wait can time out after the exit code has already arrived.
        # Cleanup must wait again without signalling an exited PID.
        exited = _FinishedProcess(0)
        with mock.patch(
            "cq.local.asyncio.wait_for",
            new=mock.AsyncMock(side_effect=[TimeoutError, 0]),
        ):
            self.assertTrue(await local._reap(exited))
        for missing_on_kill in (False, True):
            with self.subTest(missing_on_kill=missing_on_kill):
                process = _HostileProcess()
                process.terminate = mock.Mock(side_effect=ProcessLookupError)
                if missing_on_kill:
                    process.kill = mock.Mock(side_effect=ProcessLookupError)
                with mock.patch.object(local, "_TERM_GRACE_SECONDS", 0.001):
                    self.assertEqual(not missing_on_kill, await local._reap(process))
                process.terminate.assert_called_once()

    def test_identity_hash_rejects_nonregular_opened_descriptor(self) -> None:
        # Model a descriptor whose opened object is not the checked regular file.
        # A real directory open is rejected earlier by Windows itself.
        with mock.patch("cq.local.os.fstat", return_value=self.root.stat()):
            with self.assertRaises(OSError):
                local._file_sha256(self.executable)

    async def test_cancelled_captured_input_closes_and_reaps_owned_child(self) -> None:
        for operation, expected_body, invoke in (
            (
                "config_set",
                b'{"enabled":true}',
                lambda: self.client.config_set("/alpha", {"enabled": True}),
            ),
            (
                "config_replace",
                b'{"workloads":[]}',
                lambda: self.client.config_replace({"workloads": []}),
            ),
        ):
            with self.subTest(operation=operation):
                writer = _InputWriter()
                process = _InputProcess(writer)

                with mock.patch(
                    "cq.local.asyncio.create_subprocess_exec",
                    new=mock.AsyncMock(return_value=process),
                ):
                    task = asyncio.create_task(invoke())
                    try:
                        await asyncio.wait_for(writer.draining.wait(), timeout=1)
                        task.cancel()
                        with self.assertRaises(asyncio.CancelledError):
                            await task
                    finally:
                        if not task.done():
                            task.cancel()
                            await asyncio.gather(task, return_exceptions=True)

                self.assertEqual([expected_body], writer.writes)
                self.assertTrue(writer.closed)
                self.assertTrue(process.terminated)
                self.assertTrue(process.waited)

    async def test_captured_input_failure_closes_and_reaps_for_each_config_writer(self) -> None:
        for operation, invoke in (
            ("config_set", lambda: self.client.config_set("/alpha", {"enabled": True})),
            ("config_replace", lambda: self.client.config_replace({"workloads": []})),
        ):
            with self.subTest(operation=operation):
                writer = _InputWriter(BrokenPipeError("synthetic"))
                process = _InputProcess(writer)

                with mock.patch(
                    "cq.local.asyncio.create_subprocess_exec",
                    new=mock.AsyncMock(return_value=process),
                ):
                    with self.assertRaises(CircularQuayError) as captured:
                        await invoke()
                self.assertEqual(ErrorCategory.PROCESS, captured.exception.category)
                self.assertTrue(writer.closed)
                self.assertTrue(process.terminated)
                self.assertTrue(process.waited)

    async def test_inherited_run_returns_native_exit_code_with_native_streams(self) -> None:
        process = _FinishedProcess(7)
        calls: list[tuple[tuple[object, ...], dict[str, object]]] = []

        async def create_process(*arguments: object, **options: object) -> _FinishedProcess:
            calls.append((arguments, options))
            return process

        with mock.patch("cq.local.asyncio.create_subprocess_exec", new=create_process):
            code = await self.client._inherit("synthetic_inherit", ("run", "--workload", "alpha"))

        self.assertEqual(7, code)
        self.assertEqual(1, len(calls))
        options = calls[0][1]
        self.assertIsNone(options["stdin"])
        self.assertIsNone(options["stdout"])
        self.assertIsNone(options["stderr"])

    def test_rejects_malformed_json_pointer_escapes(self) -> None:
        with self.assertRaises(CircularQuayError) as captured:
            local._pointer("/workloads/~2bad")
        self.assertEqual(ErrorCategory.INVALID, captured.exception.category)

    def test_config_convenience_values_use_i_json_integer_bounds(self) -> None:
        self.assertTrue(local._is_i_json({"limit": 9_007_199_254_740_991}))
        self.assertFalse(local._is_i_json({"limit": 9_007_199_254_740_992}))

    def test_config_convenience_values_enforce_i_json_global_node_bound(self) -> None:
        document = [[0] * 1024 for _ in range(15)] + [[0] * 1007]
        # root + 16 arrays + 16,367 scalars = 16,384 nodes.
        self.assertTrue(local._is_i_json(document))
        document[-1].append(0)
        self.assertFalse(local._is_i_json(document))
        # A separate per-container bound.
        self.assertFalse(local._is_i_json([0] * 1025))

    def test_remote_config_source_rejects_userinfo_query_and_fragment_before_spawn(self) -> None:
        ref = "a" * 40
        digest = "b" * 64
        valid = local._pinned_config_arguments(
            "https://github.com/acme/cq-config", ref, "cq.json", digest, "bootstrap_pins"
        )
        if valid is None:
            self.fail("complete remote pin unexpectedly omitted its arguments")
        self.assertEqual("https://github.com/acme/cq-config", valid[1])
        for source in (
            "https://token@example.test/cq.json",
            "https://example.test/cq.json?token=synthetic",
            "https://example.test/cq.json#token",
            "https://example.test/cq.json?",
            "https://example.test/cq.json#",
            "http://example.test/cq.json",
        ):
            with self.subTest(source=source):
                with self.assertRaises(CircularQuayError) as captured:
                    local._pinned_config_arguments(
                        source,
                        ref,
                        "cq.json",
                        digest,
                        "bootstrap_pins",
                    )
                self.assertEqual(ErrorCategory.INVALID, captured.exception.category)

    def test_allowlisted_native_error_categories_do_not_retain_diagnostics(self) -> None:
        self.assertEqual(ErrorCategory.PROCESS, local._cli_failure_category(b""))
        self.assertEqual(
            ErrorCategory.EXPIRED,
            local._cli_failure_category(b'{"error":"expired_value"}'),
        )
        self.assertEqual(
            ErrorCategory.UNSUPPORTED,
            local._cli_failure_category(b'{"error":"unsupported_capability"}'),
        )
        self.assertEqual(
            ErrorCategory.PROTOCOL,
            local._cli_failure_category(b'{"error":"unknown","detail":"synthetic"}'),
        )
        self.assertEqual(
            ErrorCategory.PROTOCOL,
            local._cli_failure_category(
                b'{"error":"invalid_policy","error":"access_denied"}'
            ),
        )

    def test_advisory_observer_failure_does_not_replace_outcome(self) -> None:
        def broken_observer(_: object) -> None:
            raise RuntimeError("synthetic observer failure")

        local._observe(broken_observer, "status", "success", 0)
