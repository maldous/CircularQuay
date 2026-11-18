"""Complete bounded diagnostics/control contracts without launching tools."""
from __future__ import annotations

import ast
import contextlib
import importlib.util
import io
import json
import os
import runpy
import socket
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "sdk_python_check.py"
SPEC = importlib.util.spec_from_file_location("cq_check_completion", SCRIPT)
assert SPEC and SPEC.loader
CHECK = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = CHECK
SPEC.loader.exec_module(CHECK)
PATH = "sdk/python/src/cq/client.py"
TEST = "test_client.ClientTests.test_owned"


class CheckCompletion(unittest.TestCase):
    def setUp(self):
        self.stack = contextlib.ExitStack(); self.addCleanup(self.stack.close)
        self.root = Path(self.stack.enter_context(tempfile.TemporaryDirectory()))
        for target in ("subprocess.Popen", "subprocess.run", "socket.create_connection", "socket.getaddrinfo", "os.kill"):
            self.stack.enter_context(mock.patch(target, side_effect=AssertionError("unexpected process/network effect")))
        if hasattr(os, "killpg"):
            self.stack.enter_context(mock.patch.object(os, "killpg", side_effect=AssertionError("unexpected process signal")))

    def test_capture_stream_bounds_and_resume(self):
        capture = CHECK.Capture("ruff"); capture.consume(b"x" * (CHECK.MAX_CAPTURE_BYTES + 1))
        self.assertTrue(capture.malformed); self.assertEqual([], capture.diagnostics)
        capture = CHECK.Capture("ruff")
        capture.consume(b"x" * (CHECK.MAX_LINE_BYTES + 1)); capture.consume(b"more"); capture.consume(b"\n")
        self.assertFalse(capture.discarding_line)
        capture.consume(f"{PATH}:1: E501 owned".encode()); capture.finish()
        self.assertEqual("owned", capture.diagnostics[0].message); self.assertEqual(b"", capture.pending)
        capture = CHECK.Capture("ruff"); capture.consume(f"{PATH}:1: invalid\n".encode())
        self.assertEqual(1, capture.unparsed_sdk_lines)
        capture = CHECK.Capture("ruff")
        for _ in range(CHECK.MAX_DIAGNOSTICS + 1): capture.consume(f"{PATH}:1: E501 owned\n".encode())
        self.assertTrue(capture.malformed); self.assertEqual(CHECK.MAX_DIAGNOSTICS, len(capture.diagnostics))
        failures = CHECK.Capture("coverage")
        for _ in range(CHECK.MAX_TEST_FAILURES + 1): failures.consume(f"test_owned ({TEST}) ... FAIL\n".encode())
        self.assertTrue(failures.malformed); self.assertEqual(CHECK.MAX_TEST_FAILURES, len(failures.unittest_failures))

    def test_diagnostic_grammar_complete_negative_controls(self):
        for line in (">> Issue: [B1", ">> Issue: [X1:type] text", ">> Issue: [B1:] text"):
            self.assertIsNone(CHECK._bandit_issue(line))
        for text, expected in (("plain", ("mypy", "plain")), ("plain[x]", ("mypy", "plain[x]")),
                               ("plain []", ("mypy", "plain []")), ("plain [[x]", ("mypy", "plain [[x]"))):
            self.assertEqual(expected, CHECK._mypy_diagnostic(text))
        self.assertIsNone(CHECK._diagnostic_fields("other", "E501 owned"))
        self.assertIsNone(CHECK._safe_message(" \t "))
        self.assertEqual("a b", CHECK._safe_message("a  b"))
        self.assertLessEqual(len(CHECK._safe_message("é" * 200).encode()), CHECK.MAX_MESSAGE_BYTES)
        for path in ("outside.py", "sdk/python/../x.py", "sdk/python//x.py", "sdk/python/é.py", "sdk\\python\\x.py"):
            self.assertFalse(CHECK._safe_sdk_path(path))
        capture = CHECK.Capture("ruff"); capture._add("bad rule", PATH, 1, None, "owned")
        self.assertTrue(capture.malformed)
        with self.assertRaisesRegex(CHECK.CheckError, "unknown check"): CHECK.fixed_command("unrecognized")

    def test_coverage_parser_all_numeric_and_order_fences(self):
        for values in ([], ["1", "0", "0", "0", "100"], ["1", "bad", "0", "0", "100%"],
                       ["1", "0", "0", "0", "101%"], ["1", "2", "0", "0", "0%"], ["1", "0", "0", "1", "0%"]):
            self.assertIsNone(CHECK._coverage_numbers(values))
        self.assertIsNone(CHECK._coverage_counts([]))
        for value in ("", "01", "-1", "１２", "1234567890"):
            self.assertIsNone(CHECK._bounded_count(value))
        self.assertIsNone(CHECK._coverage_row(PATH + " 1 0"))
        self.assertIsNone(CHECK._coverage_row(PATH + " 1 0 0 0 101%"))
        self.assertIsNone(CHECK._coverage_missing(",".join("1" for _ in range(1025)), 1))
        for value in ("1->2,1->2", "0->2", "2->0", "3-2", "1-0", "1,1"):
            self.assertIsNone(CHECK._coverage_missing(value, 1))
        self.assertEqual((((1, 2),), ((3, None),)), CHECK._coverage_missing("1-2,3->exit", 2))
        self.assertEqual((1, 0, 0, 0, 100), CHECK._coverage_numbers(["1", "0", "0", "0", "100%"]))

    def test_progress_owned_input_complete_shapes(self):
        path = self.root / "progress"
        start = {"phase": "start", "test": "owned.Case.test_one"}; end = {"phase": "end", "test": start["test"]}
        def document(events): return {"format": "cq-sdk-python-progress-v1", "events": events}
        cases = [None, {}, {"format": "wrong", "events": []}, document(None), document([]),
                 document([start] * (CHECK.MAX_PROGRESS_EVENTS + 1)), document([start, start]),
                 document([end]), document([{"phase": "start", "test": "bad\x00"}]),
                 document([{"phase": "start", "test": "x" * 513}]), document([{"phase": "start", "test": 1}]),
                 document([{"phase": "other", "test": "owned"}]), document([None]), document([start])]
        for value in cases:
            path.write_text(json.dumps(value)); capture = CHECK.Capture("coverage")
            CHECK._validate_progress(path, capture, allow_open=False); self.assertTrue(capture.malformed)
        for raw in (b"x" * (CHECK.MAX_CAPTURE_BYTES + 1), b'{"format":1,"format":2}', b"\xff"):
            path.write_bytes(raw); capture = CHECK.Capture("coverage")
            CHECK._validate_progress(path, capture, allow_open=False); self.assertTrue(capture.malformed)
        for events, allow in (([start, end], False), ([start], True)):
            path.write_text(json.dumps(document(events))); capture = CHECK.Capture("coverage")
            CHECK._validate_progress(path, capture, allow_open=allow); self.assertFalse(capture.malformed)
        self.assertFalse(CHECK._valid_progress_event({"phase": "start", "test": ""}, None))
        self.assertEqual("not_run", CHECK._phase_status(True, None, False))

    def test_protected_report_refusals_preserve_old_bytes(self):
        path = self.root / "report"; path.write_bytes(b"original")
        with self.assertRaisesRegex(CHECK.CheckError, "report exceeds bound"):
            CHECK.protected_json(path, {"large": "x" * CHECK.MAX_CAPTURE_BYTES})
        with mock.patch.object(CHECK.os, "replace", side_effect=OSError):
            with self.assertRaisesRegex(CHECK.CheckError, "could not write protected report"):
                CHECK.protected_json(path, {"safe": True})
        self.assertEqual(b"original", path.read_bytes()); self.assertEqual([], list(self.root.glob(".cq-*")))
        with mock.patch.object(CHECK.os, "replace", side_effect=OSError), mock.patch.object(CHECK.os, "unlink", side_effect=OSError):
            with self.assertRaises(CHECK.CheckError): CHECK.protected_json(path, {"safe": True})
        real_chmod = os.chmod
        def chmod(name, mode):
            if Path(name) == path: raise OSError("owned final chmod refusal")
            return real_chmod(name, mode)
        with mock.patch.object(CHECK.os, "chmod", side_effect=chmod): CHECK.protected_json(path, {"safe": True})
        self.assertEqual({"safe": True}, json.loads(path.read_bytes()))

    def test_read_output_none_read_error_and_close_error(self):
        capture = CHECK.Capture("ruff"); CHECK._read_output(None, capture); self.assertTrue(capture.malformed)
        class Stream(io.BytesIO):
            def read(self, count=-1): raise OSError("owned read refusal")
            def close(self): super().close(); raise OSError("owned close refusal")
        capture = CHECK.Capture("ruff"); stream = Stream()
        CHECK._read_output(stream, capture); self.assertTrue(capture.malformed); self.assertTrue(stream.closed)

    def test_start_and_signal_use_only_explicit_finite_process_seams(self):
        root = self.root
        for platform in ("posix", "nt"):
            process = mock.Mock(); process.poll.return_value = None
            with mock.patch.object(CHECK.os, "name", platform), mock.patch.object(CHECK.subprocess, "Popen", return_value=process) as popen:
                self.assertIs(process, CHECK._start(("fixed",), root, {"HOME": str(root)}))
            options = popen.call_args.kwargs
            self.assertEqual(root, options["cwd"]); self.assertEqual({"HOME": str(root)}, options["env"])
            self.assertEqual(platform == "posix", "start_new_session" in options)
            if platform == "nt":
                with mock.patch.object(CHECK.os, "name", "nt"), mock.patch.object(CHECK.subprocess, "run", return_value=None) as taskkill:
                    CHECK._signal_tree(process, force=True)
                process.kill.assert_called_once(); self.assertIn("/T", taskkill.call_args.args[0])
                process.poll.return_value = 0
                with mock.patch.object(CHECK.os, "name", "nt"), mock.patch.object(CHECK.subprocess, "run", side_effect=subprocess.TimeoutExpired("fixed", 10)):
                    CHECK._signal_tree(process, force=False)
                process.terminate.assert_not_called()
            else:
                with mock.patch.object(CHECK.os, "name", "posix"), mock.patch.object(CHECK.signal, "SIGKILL", 9, create=True), mock.patch.object(CHECK.os, "killpg", create=True) as signal_tree:
                    CHECK._signal_tree(process, force=False); CHECK._signal_tree(process, force=True)
                    self.assertEqual([mock.call(process.pid, CHECK.signal.SIGTERM), mock.call(process.pid, CHECK.signal.SIGKILL)], signal_tree.call_args_list)

    def test_termination_escalation_preserves_failure(self):
        process = mock.Mock()
        with mock.patch.object(CHECK, "_signal_tree", side_effect=[OSError(), OSError()]) as signals, mock.patch.object(CHECK, "_wait_for_exit", return_value=False):
            self.assertFalse(CHECK._terminate(process)); self.assertEqual(2, signals.call_count)
        with mock.patch.object(CHECK, "_signal_tree") as signals, mock.patch.object(CHECK, "_wait_for_exit", side_effect=[False, True]):
            self.assertTrue(CHECK._terminate(process)); self.assertEqual([mock.call(process, force=False), mock.call(process, force=True)], signals.call_args_list)
        process.wait.side_effect = subprocess.TimeoutExpired("fixed", 1)
        self.assertFalse(CHECK._wait_for_exit(process, 1))
        process.wait.side_effect = None
        self.assertTrue(CHECK._wait_for_exit(process, 1))

    def test_command_reader_timeout_and_stuck_reader_are_not_success(self):
        for wait_error, returncode, alive in ((False, 0, [False]), (True, None, [True, True]), (True, 7, [True, False])):
            process = mock.Mock(); process.returncode = returncode
            process.wait.side_effect = subprocess.TimeoutExpired("fixed", 1) if wait_error else None
            process.wait.return_value = 0
            reader = mock.Mock(); reader.is_alive.side_effect = alive
            capture = CHECK.Capture("ruff")
            with mock.patch.object(CHECK, "_start", return_value=process), mock.patch.object(CHECK.threading, "Thread", return_value=reader), mock.patch.object(CHECK, "_terminate", return_value=False):
                code, timed = CHECK._run_command(("fixed",), self.root, {}, capture, 1)
            self.assertEqual(wait_error, timed); self.assertEqual(returncode if returncode is not None else 1, code)
            self.assertEqual(wait_error, capture.malformed); reader.start.assert_called_once()

    def test_run_check_complete_finite_phase_matrix(self):
        for check, initial, report_code, timeout, clock in (
            ("ruff", 0, 0, False, [0, 0]), ("coverage", 0, 1, False, [0, 0, 0, 0]),
            ("coverage", 3, 0, False, [0, 0, 0, 0]), ("coverage", 0, 0, True, [0, 0, 0, 0]),
            ("coverage", 0, 0, False, [0, 2, 2, 2])):
            progress = self.root / "progress.json"
            calls = []
            def command(argv, root, environment, capture, seconds):
                calls.append(argv)
                self.assertNotIn("AWS_SECRET_ACCESS_KEY", environment)
                if len(calls) == 1 and check == "coverage":
                    CHECK.protected_json(progress, {"format": "cq-sdk-python-progress-v1", "events": [{"phase": "start", "test": "owned"}, {"phase": "end", "test": "owned"}]})
                if check == "coverage":
                    for path in sorted(CHECK.OWNED_COVERAGE_PATHS):
                        if not capture.coverage_rows or path not in {row.path for row in capture.coverage_rows}:
                            capture.consume(f"{path} 1 0 0 0 100%\n".encode())
                return (initial, timeout) if len(calls) == 1 else (report_code, False)
            with mock.patch.object(CHECK, "__file__", str(self.root / ".github/scripts/sdk_python_check.py")), mock.patch.object(CHECK, "_run_command", side_effect=command), mock.patch.object(CHECK.time, "monotonic", side_effect=clock):
                code, capture, timed, elapsed = CHECK.run_check(check, 1, progress if check == "coverage" else None)
            self.assertEqual(initial or (report_code if len(calls) == 2 else 0), code)
            self.assertEqual(timeout or clock[-1] >= 2, timed)
            self.assertFalse(capture.malformed)
            self.assertEqual(2 if check == "coverage" and clock[-1] == 0 else 1, len(calls))

    def test_cli_real_arguments_owned_report_and_script_entry(self):
        report = self.root / "report"; progress = self.root / "progress"
        for args, expected, malformed, timed, code in (
            (["--check", "ruff", "--report", str(report)], 0, False, False, 0),
            (["--check", "ruff", "--report", str(report)], 2, True, False, 0),
            (["--check", "ruff", "--report", str(report)], 7, False, True, 7),
            (["--check", "ruff", "--report", str(report), "--timeout-seconds", "-1"], 2, False, False, 0),
            (["--check", "coverage", "--report", str(report)], 2, False, False, 0),
            (["--check", "coverage", "--report", str(report), "--progress", str(progress)], 0, False, False, 0)):
            capture = CHECK.Capture("coverage"); capture.malformed = malformed
            with mock.patch.object(sys, "argv", [str(SCRIPT), *args]), mock.patch.object(CHECK, "run_check", return_value=(code, capture, timed, 1)), contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(expected, CHECK.main())
        # Fresh entry point is stopped at argparse's help branch, before any check can launch.
        with mock.patch.object(sys, "argv", [str(SCRIPT), "--help"]), contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaises(SystemExit) as status: runpy.run_path(str(SCRIPT), run_name="__main__")
        self.assertEqual(0, status.exception.code)

    def test_output_reader_consumes_actual_nonempty_stream_and_final_fragment(self):
        raw = (f"{PATH}:3: E501 complete\n{PATH}:4: E502 final").encode()
        class Stream(io.BytesIO):
            def __init__(self, data):
                super().__init__(data)
                self.requested = []
            def read(self, count=-1):
                self.requested.append(count)
                return super().read(count)
        stream = Stream(raw)
        capture = CHECK.Capture("ruff")
        CHECK._read_output(stream, capture)
        self.assertTrue(stream.closed)
        self.assertEqual([4096, 4096], stream.requested)
        self.assertFalse(capture.malformed)
        self.assertEqual(b"", capture.pending)
        self.assertEqual([(3, "E501", "complete"), (4, "E502", "final")],
                         [(item.line, item.rule, item.message) for item in capture.diagnostics])
