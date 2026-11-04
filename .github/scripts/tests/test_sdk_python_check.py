"""Synthetic contracts for the bounded Python SDK CI diagnostics helper."""

from __future__ import annotations

import ast
import importlib.util
import io
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock


SCRIPT = Path(__file__).parents[1] / "sdk_python_check.py"
SPEC = importlib.util.spec_from_file_location("sdk_python_check", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
CHECK = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = CHECK
SPEC.loader.exec_module(CHECK)


class FixedCommandTest(unittest.TestCase):
    def test_uses_current_interpreter_and_fixed_sdk_scopes(self) -> None:
        self.assertEqual(
            ("-m", "ruff", "check", "--output-format=concise", "sdk/python"),
            CHECK.fixed_command("ruff")[1:],
        )
        self.assertEqual("sdk/python/pyproject.toml", CHECK.fixed_command("mypy")[4])
        self.assertEqual(("-m", "bandit", "-r", "sdk/python/src"), CHECK.fixed_command("bandit")[1:])
        self.assertIn("--rcfile", CHECK.fixed_command("coverage"))
        self.assertEqual(("-m", "coverage", "report"), CHECK.coverage_report_command()[1:4])


class CaptureTest(unittest.TestCase):
    def test_keeps_only_bounded_sdk_path_diagnostics(self) -> None:
        capture = CHECK.Capture("ruff")
        capture.consume(b"sdk/python/src/cquay/client.py:19:4: E501 Line too long\n")
        capture.consume(b"outside.py:1:1: E501 ignored\n")
        self.assertEqual(1, len(capture.diagnostics))
        diagnostic = capture.diagnostics[0]
        self.assertEqual(("E501", "sdk/python/src/cquay/client.py", 19, 4), (diagnostic.rule, diagnostic.path, diagnostic.line, diagnostic.column))

    def test_marks_invalid_utf8_capture_malformed(self) -> None:
        capture = CHECK.Capture("mypy")
        capture.consume(b"\xff\n")
        self.assertTrue(capture.malformed)

    def test_marks_line_overflow_and_control_message_malformed(self) -> None:
        overflow = CHECK.Capture("ruff")
        overflow.consume(b"x" * (CHECK.MAX_LINE_BYTES + 1))
        self.assertTrue(overflow.malformed)
        self.assertIsNone(CHECK._safe_message("public\x1fsource"))

    def test_rejects_malformed_bandit_and_noncanonical_paths(self) -> None:
        bandit = CHECK.Capture("bandit")
        bandit.consume(b">> Issue: [B101:assert_used] synthetic message\n")
        bandit.finish()
        self.assertTrue(bandit.malformed)

        unsafe = CHECK.Capture("ruff")
        unsafe.consume(b"sdk/python/src/cquay/../secret.py:1:1: E501 synthetic\n")
        self.assertTrue(unsafe.malformed)

        numeric = CHECK.Capture("ruff")
        numeric.consume(b"sdk/python/src/cquay/client.py:" + b"9" * 32 + b":1: E501 synthetic\n")
        self.assertTrue(numeric.malformed)

    def test_parses_bandit_issue_with_location(self) -> None:
        capture = CHECK.Capture("bandit")
        capture.consume(b">> Issue: [B101:assert_used] synthetic message\n")
        capture.consume(b"   Location: sdk/python/src/cquay/client.py:42:0\n")
        self.assertEqual("B101", capture.diagnostics[0].rule)

    def test_parses_mypy_rule_without_backtracking_expression(self) -> None:
        capture = CHECK.Capture("mypy")
        capture.consume(
            b"sdk/python/src/cquay/client.py:19: error: synthetic type error [arg-type]\n"
        )
        self.assertEqual(
            ("arg-type", "synthetic type error"),
            (capture.diagnostics[0].rule, capture.diagnostics[0].message),
        )

        empty_message = CHECK.Capture("mypy")
        empty_message.consume(b"sdk/python/src/cquay/client.py:19: [arg-type]\n")
        self.assertTrue(empty_message.malformed)

    def test_records_bounded_coverage_rows_without_raw_output(self) -> None:
        capture = CHECK.Capture("coverage")
        capture.consume(b"sdk/python/src/cquay/client.py 12 1 4 1 75% 4-6, 9->exit, 11->7\n")
        self.assertFalse(capture.malformed)
        self.assertEqual(0, capture.unparsed_sdk_lines)
        self.assertEqual(1, len(capture.coverage_rows))
        row = capture.coverage_rows[0]
        self.assertEqual("sdk/python/src/cquay/client.py", row.path)
        self.assertEqual(((4, 6),), row.missing_lines)
        self.assertEqual(((9, None), (11, 7)), row.missing_branches)
        report = CHECK.report("coverage", 2, capture, False, 1)
        self.assertEqual(
            [{"start": 4, "end": 6}],
            report["coverage"][0]["missing_lines"],
        )
        self.assertEqual(
            [{"from": 9, "to": "exit"}, {"from": 11, "to": 7}],
            report["coverage"][0]["missing_branches"],
        )

    def test_normalizes_windows_rows_and_requires_complete_manifest(self) -> None:
        capture = CHECK.Capture("coverage")
        capture.consume(b"sdk\\python\\src\\cquay\\client.py 12 0 0 0 100%\n")
        self.assertEqual("sdk/python/src/cquay/client.py", capture.coverage_rows[0].path)
        CHECK._require_coverage_manifest(capture)
        self.assertTrue(capture.malformed)

        complete = CHECK.Capture("coverage")
        for path in sorted(CHECK.OWNED_COVERAGE_PATHS):
            complete.consume(f"{path} 12 0 0 0 100%\n".encode("utf-8"))
        CHECK._require_coverage_manifest(complete)
        self.assertFalse(complete.malformed)

    def test_rejects_malformed_or_duplicate_coverage_rows(self) -> None:
        malformed = CHECK.Capture("coverage")
        malformed.consume(b"sdk/python/src/cquay/client.py 12 0 0 0 84% 4-5\n")
        self.assertTrue(malformed.malformed)

        zero_range = CHECK.Capture("coverage")
        zero_range.consume(b"sdk/python/src/cquay/client.py 12 1 0 0 91% 0\n")
        self.assertTrue(zero_range.malformed)

        overlapping_range = CHECK.Capture("coverage")
        overlapping_range.consume(b"sdk/python/src/cquay/client.py 12 3 0 0 75% 3-4, 4\n")
        self.assertTrue(overlapping_range.malformed)

        duplicate = CHECK.Capture("coverage")
        row = b"sdk/python/src/cquay/client.py 12 0 0 0 100%\n"
        duplicate.consume(row + row)
        self.assertTrue(duplicate.malformed)

        untracked = CHECK.Capture("coverage")
        untracked.consume(b"sdk/python/src/cquay/untracked.py 12 0 0 0 100%\n")
        self.assertTrue(untracked.malformed)


class ProtectedReportTest(unittest.TestCase):
    def test_writes_machine_readable_structured_report(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "reports" / "result.json"
            CHECK.protected_json(path, {"format": CHECK.FORMAT, "status": "passed"})
            self.assertEqual({"format": CHECK.FORMAT, "status": "passed"}, json.loads(path.read_text(encoding="utf-8")))

    def test_progress_hook_is_valid_python_and_missing_progress_is_malformed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            hook = CHECK.coverage_hook(Path(directory))
            ast.parse(hook.read_text(encoding="utf-8"), filename=str(hook))
            capture = CHECK.Capture("coverage")
            CHECK._validate_progress(Path(directory) / "missing.json", capture, allow_open=False)
            self.assertTrue(capture.malformed)

    def test_progress_requires_paired_bounded_events(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "progress.json"
            CHECK.protected_json(
                path,
                {
                    "format": "cquay-sdk-python-progress-v1",
                    "events": [
                        {"phase": "start", "test": "suite.Case.test_one"},
                        {"phase": "end", "test": "suite.Case.test_one"},
                    ],
                },
            )
            capture = CHECK.Capture("coverage")
            CHECK._validate_progress(path, capture, allow_open=False)
            self.assertFalse(capture.malformed)

            CHECK.protected_json(
                path,
                {"format": "cquay-sdk-python-progress-v1", "events": [{"phase": "end", "test": "wrong"}]},
            )
            malformed = CHECK.Capture("coverage")
            CHECK._validate_progress(path, malformed, allow_open=False)
            self.assertTrue(malformed.malformed)

    def test_progress_recursion_error_is_malformed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "progress.json"
            path.write_bytes(b"{}")
            capture = CHECK.Capture("coverage")
            with mock.patch.object(CHECK.json, "loads", side_effect=RecursionError):
                CHECK._validate_progress(path, capture, allow_open=False)
            self.assertTrue(capture.malformed)

    def test_child_environment_excludes_ambient_credentials_and_uses_private_home(self) -> None:
        with tempfile.TemporaryDirectory() as directory, mock.patch.dict(
            CHECK.os.environ,
            {"AWS_SECRET_ACCESS_KEY": "synthetic", "HOME": "/ambient", "SYSTEMROOT": "C:/Windows"},
            clear=True,
        ):
            private_home = Path(directory) / "home"
            private_home.mkdir()
            environment = CHECK._child_environment(Path("/repo"), None, None, private_home)
        self.assertNotIn("AWS_SECRET_ACCESS_KEY", environment)
        self.assertEqual(str(private_home), environment["HOME"])
        self.assertEqual(str(private_home), environment["USERPROFILE"])


class TimeoutTest(unittest.TestCase):
    def test_timeout_terminates_child_and_returns_nonzero_without_raw_output(self) -> None:
        class Process:
            stdout = io.BytesIO(b"")
            returncode: int | None = None

            def poll(self) -> int | None:
                return self.returncode

            def wait(self, timeout: int | None = None) -> int:
                if self.returncode is None:
                    raise subprocess.TimeoutExpired("synthetic", timeout)
                return self.returncode

            def terminate(self) -> None:
                self.returncode = 124

            def kill(self) -> None:
                self.returncode = 137

        process = Process()
        signals: list[bool] = []

        def signal_child(child: Process, *, force: bool) -> None:
            signals.append(force)
            child.returncode = 124

        with mock.patch.object(CHECK, "_start", return_value=process), mock.patch.object(CHECK, "_signal_tree", side_effect=signal_child):
            code, timed_out = CHECK._run_command(("synthetic",), Path("/"), {}, CHECK.Capture("ruff"), 1)
        self.assertEqual(124, code)
        self.assertTrue(timed_out)
        self.assertEqual([False], signals)

    def test_windows_tree_signal_falls_back_to_direct_process(self) -> None:
        class Process:
            pid = 42
            returncode: int | None = None

            def poll(self) -> int | None:
                return self.returncode

            def terminate(self) -> None:
                self.returncode = 1

            def kill(self) -> None:
                self.returncode = 2

        process = Process()
        with mock.patch.object(CHECK.os, "name", "nt"), mock.patch.object(CHECK.subprocess, "run", side_effect=OSError):
            CHECK._signal_tree(process, force=False)
        self.assertEqual(1, process.returncode)


if __name__ == "__main__":
    unittest.main()
