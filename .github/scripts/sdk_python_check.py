#!/usr/bin/env python3
"""Run one bounded, fixed Python SDK qualification check with safe evidence."""

from __future__ import annotations

import argparse
import ast
import json
import os
import re
import signal
import subprocess
import sys
import tempfile
import threading
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


FORMAT = "cq-sdk-python-check-v1"
MAX_CAPTURE_BYTES = 1_048_576
MAX_DIAGNOSTICS = 1_024
MAX_COVERAGE_ROWS = 1_024
MAX_COVERAGE_MISSING_ITEMS = 1_024
MAX_MESSAGE_BYTES = 240
MAX_LINE_BYTES = 16_384
MAX_PROGRESS_EVENTS = 4_096
MAX_TEST_FAILURES = 64
HOOK_MAX_EVENTS = 1_024
HOOK_MAX_TEST_ID_BYTES = 256
MAX_TIMEOUT_SECONDS = 1_200
SDK_PATH = "sdk/python/"
WINDOWS_SDK_PATH = "sdk\\python\\"
PYTHON_SDK_SOURCE = "sdk/python/src"
# The fixed report may retain coverage only for tracked SDK modules, never a tool-supplied path.
OWNED_COVERAGE_PATHS = frozenset(
    {
        "sdk/python/src/cq/__init__.py",
        "sdk/python/src/cq/client.py",
        "sdk/python/src/cq/errors.py",
        "sdk/python/src/cq/local.py",
        "sdk/python/src/cq/types.py",
    }
)
DIAGNOSTIC = re.compile(
    r"^(?P<path>sdk/python/(?:src|tests)/[^:]+\.py):(?P<line>[1-9]\d{0,8})"
    r"(?::(?P<column>[1-9]\d{0,8}))?:\s*(?P<message>.+)$"
)
BANDIT_LOCATION = re.compile(
    r"^\s*Location: (?P<path>sdk/python/(?:src|tests)/[^:]+\.py):"
    r"(?P<line>[1-9]\d{0,8})(?::(?P<column>0|[1-9]\d{0,8}))?$"
)
RULE = re.compile(r"^[A-Za-z][A-Za-z0-9_.-]{0,63}$")
UNITTEST_RESULT = re.compile(
    r"^test_[A-Za-z0-9_]{1,127} \((?P<test>[^\r\n()]{1,256})\) \.\.\. (?P<result>FAIL|ERROR)$"
)
UNITTEST_IDENTIFIER = re.compile(
    r"^test_(?:boundaries|client|local|types)\.[A-Z][A-Za-z0-9_]{0,127}\.test_[a-z][A-Za-z0-9_]{0,127}$"
)
RUNTIME_ENVIRONMENT = (
    "LANG",
    "LC_ALL",
    "SYSTEMROOT",
    "WINDIR",
)


class CheckError(Exception):
    """Raised when fixed qualification inputs or captures are unsafe."""


@dataclass(frozen=True, slots=True)
class Diagnostic:
    """One bounded public-source diagnostic retained in the report."""

    rule: str
    path: str
    line: int
    column: int | None
    message: str


@dataclass(frozen=True, slots=True)
class CoverageRow:
    """One validated coverage-table row with no captured tool text."""

    path: str
    statements: int
    missed_statements: int
    branches: int
    partial_branches: int
    percent: int
    missing_lines: tuple[tuple[int, int], ...]
    missing_branches: tuple[tuple[int, int | None], ...]


@dataclass(slots=True)
class Capture:
    """Streaming, bounded parser that never writes raw tool output."""

    check: str
    diagnostics: list[Diagnostic] = field(default_factory=list)
    coverage_rows: list[CoverageRow] = field(default_factory=list)
    bytes_seen: int = 0
    malformed: bool = False
    unparsed_sdk_lines: int = 0
    bandit_rule: str | None = None
    bandit_message: str = ""
    pending: bytes = b""
    discarding_line: bool = False
    unittest_failures: list[str] = field(default_factory=list)
    unittest_started: bool = False
    unittest_exit_code: int | None = None
    unittest_timed_out: bool = False
    coverage_report_started: bool = False
    coverage_report_exit_code: int | None = None
    coverage_report_timed_out: bool = False

    def consume(self, raw: bytes) -> None:
        """Consume a bounded read without allocating an unbounded output line."""

        self.bytes_seen += len(raw)
        if self.bytes_seen > MAX_CAPTURE_BYTES:
            self.malformed = True
            return
        for fragment in raw.splitlines(keepends=True):
            self._consume_fragment(fragment)

    def finish(self) -> None:
        if self.pending:
            self._consume_complete_line(self.pending)
            self.pending = b""
        if self.bandit_rule is not None:
            self.malformed = True

    def _consume_fragment(self, fragment: bytes) -> None:
        complete = fragment.endswith((b"\n", b"\r"))
        if self.discarding_line:
            if complete:
                self.discarding_line = False
            return
        if len(self.pending) + len(fragment) > MAX_LINE_BYTES:
            self.malformed = True
            self.pending = b""
            self.discarding_line = not complete
            return
        self.pending += fragment
        if complete:
            self._consume_complete_line(self.pending)
            self.pending = b""

    def _consume_complete_line(self, raw: bytes) -> None:
        try:
            line = raw.decode("utf-8").rstrip("\r\n")
        except UnicodeDecodeError:
            self.malformed = True
            return
        self._consume_line(line)

    def _consume_line(self, line: str) -> None:
        unittest = UNITTEST_RESULT.fullmatch(line)
        if unittest is not None:
            self._add_unittest_failure(unittest.group("test"))
            return
        issue = _bandit_issue(line)
        if issue is not None:
            self.bandit_rule, self.bandit_message = issue
            return
        location = BANDIT_LOCATION.fullmatch(line)
        if location is not None and self.bandit_rule is not None:
            self._add(
                self.bandit_rule,
                location.group("path"),
                int(location.group("line")),
                _integer(location.group("column")),
                self.bandit_message,
            )
            self.bandit_rule = None
            self.bandit_message = ""
            return
        if self.check == "coverage" and _coverage_candidate(line):
            row = _coverage_row(line)
            if row is None or any(item.path == row.path for item in self.coverage_rows):
                self.unparsed_sdk_lines += 1
                self.malformed = True
                return
            if len(self.coverage_rows) >= MAX_COVERAGE_ROWS:
                self.malformed = True
                return
            self.coverage_rows.append(row)
            return
        match = DIAGNOSTIC.fullmatch(line)
        if match is None:
            if line.startswith(SDK_PATH):
                self.unparsed_sdk_lines += 1
                self.malformed = True
            return
        parsed = _diagnostic_fields(self.check, match.group("message"))
        if parsed is None:
            self.unparsed_sdk_lines += 1
            return
        rule, message = parsed
        self._add(rule, match.group("path"), int(match.group("line")), _integer(match.group("column")), message)

    def _add(self, rule: str, path: str, line: int, column: int | None, message: str) -> None:
        if len(self.diagnostics) >= MAX_DIAGNOSTICS:
            self.malformed = True
            return
        safe = _safe_message(message)
        if safe is None or RULE.fullmatch(rule) is None or not _safe_sdk_path(path):
            self.malformed = True
            return
        self.diagnostics.append(Diagnostic(rule, path, line, column, safe))

    def _add_unittest_failure(self, test: str) -> None:
        if len(self.unittest_failures) >= MAX_TEST_FAILURES or _safe_test_identifier(test) is None:
            self.malformed = True
            return
        self.unittest_failures.append(test)


def _integer(value: str | None) -> int | None:
    return int(value) if value is not None else None


def _diagnostic_fields(check: str, text: str) -> tuple[str, str] | None:
    if check == "ruff":
        fields = text.split(None, 1)
        if len(fields) != 2 or not _ruff_rule(fields[0]):
            return None
        return fields[0], fields[1]
    if check == "mypy":
        return _mypy_diagnostic(text)
    return None


def _bandit_issue(line: str) -> tuple[str, str] | None:
    """Parse Bandit's fixed issue header without a backtracking expression."""

    prefix = ">> Issue: ["
    if not line.startswith(prefix):
        return None
    closing = line.find("]", len(prefix))
    if closing < 0:
        return None
    rule, separator, kind = line[len(prefix) : closing].partition(":")
    if not separator or not kind or not rule.startswith("B") or not rule[1:].isdigit():
        return None
    return rule, line[closing + 1 :].lstrip()


def _ruff_rule(value: str) -> bool:
    """Recognize Ruff's uppercase-letter and decimal-code format directly."""

    prefix = 0
    while prefix < len(value) and "A" <= value[prefix] <= "Z":
        prefix += 1
    return 0 < prefix < len(value) and value[prefix:].isdigit()


def _mypy_diagnostic(text: str) -> tuple[str, str]:
    """Separate mypy's optional bracketed rule from its public diagnostic."""

    message = text.removeprefix("error: ")
    if message.startswith("[") and message.endswith("]"):
        # A rule alone has no public diagnostic message and is malformed.
        # Return an empty message so _add applies the common safe-text gate.
        return "mypy", ""
    if not message.endswith("]"):
        return "mypy", message
    opening = message.rfind("[")
    if opening == 0 or not message[opening - 1].isspace():
        return "mypy", message
    rule = message[opening + 1 : -1]
    if not rule or "[" in rule or "]" in rule:
        return "mypy", message
    return rule, message[:opening].rstrip()


def _safe_message(value: str) -> str | None:
    if any(ord(character) < 0x20 or ord(character) == 0x7F for character in value):
        return None
    collapsed = " ".join(value.split())
    if not collapsed:
        return None
    encoded = collapsed.encode("utf-8")
    return encoded[:MAX_MESSAGE_BYTES].decode("utf-8", "ignore")


def _safe_sdk_path(value: str) -> bool:
    if not value.startswith(SDK_PATH) or not value.endswith(".py") or "\\" in value:
        return False
    parts = value.split("/")
    return all(
        part not in {"", ".", ".."}
        and all(character.isascii() and (character.isalnum() or character in "_.-") for character in part)
        for part in parts
    )


def _safe_test_identifier(value: str) -> str | None:
    if len(value.encode("utf-8")) > 256 or UNITTEST_IDENTIFIER.fullmatch(value) is None:
        return None
    return value


def _coverage_row(line: str) -> CoverageRow | None:
    """Parse one fixed-format coverage row without retaining raw output."""

    columns = line.split(maxsplit=6)
    if len(columns) not in {6, 7}:
        return None
    path = _coverage_path(columns[0])
    if path is None:
        return None
    numbers = _coverage_numbers(columns[1:6])
    if numbers is None:
        return None
    statement_count, missed_count, branch_count, partial_count, percentage = numbers
    missing = _coverage_missing(columns[6] if len(columns) == 7 else "", missed_count)
    if missing is None:
        return None
    ranges, branches = missing
    return CoverageRow(
        path=path,
        statements=statement_count,
        missed_statements=missed_count,
        branches=branch_count,
        partial_branches=partial_count,
        percent=percentage,
        missing_lines=ranges,
        missing_branches=branches,
    )


def _coverage_candidate(line: str) -> bool:
    """Recognize only the two relative separator forms emitted by supported runners."""

    return line.startswith(SDK_PATH) or line.startswith(WINDOWS_SDK_PATH)


def _coverage_path(value: str) -> str | None:
    """Normalize an expected relative Windows path before the manifest check."""

    if value.startswith(WINDOWS_SDK_PATH):
        value = value.replace("\\", "/")
    return value if value in OWNED_COVERAGE_PATHS else None


def _coverage_numbers(values: list[str]) -> tuple[int, int, int, int, int] | None:
    """Validate the four integer columns and the displayed percentage."""

    if len(values) != 5 or not values[4].endswith("%"):
        return None
    counts = _coverage_counts(values[:4])
    if counts is None:
        return None
    percentage = _bounded_count(values[4][:-1])
    if percentage is None or percentage > 100:
        return None
    statements, missed, branches, partial = counts
    if missed > statements or partial > branches:
        return None
    return statements, missed, branches, partial, percentage


def _coverage_counts(values: list[str]) -> tuple[int, int, int, int] | None:
    """Parse the four non-negative count columns with a fixed upper bound."""

    if len(values) != 4:
        return None
    counts: list[int] = []
    for value in values:
        count = _bounded_count(value)
        if count is None:
            return None
        counts.append(count)
    return counts[0], counts[1], counts[2], counts[3]


def _bounded_count(value: str) -> int | None:
    if not value or len(value) > 9 or not value.isascii() or not value.isdecimal():
        return None
    if len(value) > 1 and value.startswith("0"):
        return None
    return int(value)


def _coverage_missing(
    value: str, missed_count: int
) -> tuple[tuple[tuple[int, int], ...], tuple[tuple[int, int | None], ...]] | None:
    if not value:
        return ((), ())
    parts = value.split(",")
    if not parts or len(parts) > MAX_COVERAGE_MISSING_ITEMS:
        return None
    ranges: list[tuple[int, int]] = []
    branches: list[tuple[int, int | None]] = []
    for part in parts:
        if not _append_missing_item(part.strip(), ranges, branches):
            return None
    if missed_count == 0 and ranges:
        return None
    return tuple(ranges), tuple(branches)


def _append_missing_item(
    value: str, ranges: list[tuple[int, int]], branches: list[tuple[int, int | None]]
) -> bool:
    """Append one unique branch arc or ordered statement range."""

    branch = _branch_arc(value)
    if branch is not None:
        if branch in branches:
            return False
        branches.append(branch)
        return True
    return _append_statement_range(value, ranges)


def _append_statement_range(value: str, ranges: list[tuple[int, int]]) -> bool:
    """Append a statement range only when it preserves strict source ordering."""

    line_range = _statement_range(value)
    if line_range is None:
        return False
    if ranges and line_range[0] <= ranges[-1][1]:
        return False
    ranges.append(line_range)
    return True


def _branch_arc(value: str) -> tuple[int, int | None] | None:
    """Parse one documented missing-branch arrow, including an exit target."""

    source_text, arrow, destination_text = value.partition("->")
    if not arrow:
        return None
    source = _positive_count(source_text)
    destination = None if destination_text == "exit" else _positive_count(destination_text)
    if source is None or (destination_text != "exit" and destination is None):
        return None
    return source, destination


def _statement_range(value: str) -> tuple[int, int] | None:
    """Parse one positive missing physical-line number or inclusive range."""

    start_text, separator, end_text = value.partition("-")
    start = _positive_count(start_text)
    end = _positive_count(end_text) if separator else start
    if start is None or end is None or start > end:
        return None
    return start, end


def _positive_count(value: str) -> int | None:
    count = _bounded_count(value)
    return count if count is not None and count > 0 else None


def fixed_command(check: str) -> tuple[str, ...]:
    """Return the only supported commands, run by the current locked interpreter."""

    commands = {
        "ruff": (sys.executable, "-m", "ruff", "check", "--output-format=concise", "sdk/python"),
        "mypy": (
            sys.executable,
            "-m",
            "mypy",
            "--config-file",
            "sdk/python/pyproject.toml",
            "sdk/python/src",
        ),
        "bandit": (sys.executable, "-m", "bandit", "-r", "sdk/python/src"),
        "coverage": (
            sys.executable,
            "-m",
            "coverage",
            "run",
            "--rcfile",
            "sdk/python/pyproject.toml",
            "--branch",
            "-m",
            "unittest",
            "discover",
            "-s",
            "sdk/python/tests",
            "-v",
        ),
    }
    try:
        return commands[check]
    except KeyError as error:
        raise CheckError("unknown check") from error


def coverage_report_command() -> tuple[str, ...]:
    """Return the fixed coverage threshold command required after test coverage."""

    return (
        sys.executable,
        "-m",
        "coverage",
        "report",
        "--rcfile",
        "sdk/python/pyproject.toml",
        "--fail-under=100",
    )


def protected_json(path: Path, value: dict[str, Any]) -> None:
    """Atomically write bounded evidence with owner-only Unix permissions."""

    contents = json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode("utf-8")
    if len(contents) > MAX_CAPTURE_BYTES:
        raise CheckError("report exceeds bound")
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    descriptor, temporary = tempfile.mkstemp(prefix=".cq-", dir=path.parent)
    try:
        with os.fdopen(descriptor, "wb") as output:
            output.write(contents)
            output.flush()
            os.fsync(output.fileno())
        os.chmod(temporary, 0o600)
        os.replace(temporary, path)
        try:
            os.chmod(path, 0o600)
        except OSError:
            pass
    except OSError as error:
        try:
            os.unlink(temporary)
        except OSError:
            pass
        raise CheckError("could not write protected report") from error


def coverage_hook(directory: Path) -> Path:
    """Create an ephemeral unittest hook that flushes test start/end evidence."""

    hook = directory / "sitecustomize.py"
    source = "\n".join(
        (
            "import json, os, tempfile, unittest",
            "_path = os.environ['CQ_SDK_PYTHON_PROGRESS']",
            "_events = []",
            "_max_events = 1024",
            "_max_test_id_bytes = 256",
            "def _write():",
            "    data = json.dumps({'format': 'cq-sdk-python-progress-v1', 'events': _events}, sort_keys=True, separators=(',', ':')).encode()",
            "    fd, temporary = tempfile.mkstemp(prefix='.cq-', dir=os.path.dirname(_path))",
            "    with os.fdopen(fd, 'wb') as output:",
            "        output.write(data)",
            "        output.flush()",
            "        os.fsync(output.fileno())",
            "    os.chmod(temporary, 0o600)",
            "    os.replace(temporary, _path)",
            "_start, _stop = unittest.TestResult.startTest, unittest.TestResult.stopTest",
            "def start(self, test):",
            "    _start(self, test)",
            "    test_id = test.id()",
            "    if len(_events) >= _max_events or len(test_id.encode('utf-8')) > _max_test_id_bytes:",
            "        raise RuntimeError('progress event bound exceeded')",
            "    _events.append({'phase': 'start', 'test': test_id})",
            "    _write()",
            "def stop(self, test):",
            "    test_id = test.id()",
            "    if len(_events) >= _max_events or len(test_id.encode('utf-8')) > _max_test_id_bytes:",
            "        raise RuntimeError('progress event bound exceeded')",
            "    _events.append({'phase': 'end', 'test': test_id})",
            "    _write()",
            "    _stop(self, test)",
            "unittest.TestResult.startTest = start",
            "unittest.TestResult.stopTest = stop",
            "",
        )
    )
    try:
        ast.parse(source, filename=str(hook))
    except SyntaxError as error:
        raise CheckError("coverage progress hook is invalid") from error
    hook.write_text(source, encoding="utf-8")
    os.chmod(hook, 0o600)
    return hook


def _child_environment(
    root: Path, hook_directory: Path | None, progress: Path | None, private_home: Path
) -> dict[str, str]:
    environment = {name: os.environ[name] for name in RUNTIME_ENVIRONMENT if name in os.environ}
    environment["PYTHONDONTWRITEBYTECODE"] = "1"
    environment["PYTHONNOUSERSITE"] = "1"
    for name in ("HOME", "LOCALAPPDATA", "TEMP", "TMP", "TMPDIR", "USERPROFILE"):
        environment[name] = str(private_home)
    paths = [str(root / PYTHON_SDK_SOURCE)]
    if hook_directory is not None:
        paths.insert(0, str(hook_directory))
    environment["PYTHONPATH"] = os.pathsep.join(paths)
    if progress is not None:
        environment["CQ_SDK_PYTHON_PROGRESS"] = str(progress)
    if hook_directory is not None:
        environment["COVERAGE_FILE"] = str(hook_directory / ".coverage")
    return environment


def _terminate(process: subprocess.Popen[bytes]) -> bool:
    try:
        _signal_tree(process, force=False)
        if _wait_for_exit(process, 10):
            return True
    except (OSError, subprocess.TimeoutExpired):
        pass
    try:
        _signal_tree(process, force=True)
    except (OSError, subprocess.TimeoutExpired):
        pass
    return _wait_for_exit(process, 10)


def _signal_tree(process: subprocess.Popen[bytes], *, force: bool) -> None:
    if os.name == "nt":
        try:
            subprocess.run(
                ("taskkill", "/PID", str(process.pid), "/T", "/F"),
                check=False,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                timeout=10,
            )
        except (OSError, subprocess.TimeoutExpired):
            pass
        if process.poll() is None:
            (process.kill if force else process.terminate)()
        return
    os.killpg(process.pid, signal.SIGKILL if force else signal.SIGTERM)


def _wait_for_exit(process: subprocess.Popen[bytes], timeout_seconds: int) -> bool:
    try:
        process.wait(timeout=timeout_seconds)
        return True
    except subprocess.TimeoutExpired:
        return False


def run_check(check: str, timeout_seconds: int, progress: Path | None) -> tuple[int, Capture, bool, int]:
    """Run one fixed command and return exit code, safe capture, timeout, duration."""

    root = Path(__file__).parents[2]
    command = fixed_command(check)
    capture = Capture(check)
    with tempfile.TemporaryDirectory(prefix="cq-sdk-python-") as temporary:
        hooks = Path(temporary) if check == "coverage" else None
        private_home = Path(temporary) / "home"
        private_home.mkdir(mode=0o700)
        if hooks is not None and progress is not None:
            protected_json(progress, {"format": "cq-sdk-python-progress-v1", "events": []})
            coverage_hook(hooks)
        environment = _child_environment(root, hooks, progress, private_home)
        started = time.monotonic()
        if check == "coverage":
            capture.unittest_started = True
        code, timed_out = _run_command(command, root, environment, capture, timeout_seconds)
        if check == "coverage":
            capture.unittest_exit_code = code
            capture.unittest_timed_out = timed_out
        if check == "coverage" and progress is not None:
            _validate_progress(progress, capture, allow_open=timed_out)
        elapsed = int((time.monotonic() - started) * 1000)
        if check == "coverage":
            remaining = timeout_seconds - int((time.monotonic() - started))
            if remaining > 0:
                capture.coverage_report_started = True
                report_code, report_timeout = _run_command(coverage_report_command(), root, environment, capture, remaining)
                capture.coverage_report_exit_code = report_code
                capture.coverage_report_timed_out = report_timeout
                if code == 0:
                    code = report_code
                timed_out = timed_out or report_timeout
            else:
                timed_out = True
            _require_coverage_manifest(capture)
            elapsed = int((time.monotonic() - started) * 1000)
        return code, capture, timed_out, elapsed


def _require_coverage_manifest(capture: Capture) -> None:
    """Fail closed unless coverage evidence covers exactly the declared SDK modules."""

    if {item.path for item in capture.coverage_rows} != OWNED_COVERAGE_PATHS:
        capture.malformed = True


def _run_command(
    command: tuple[str, ...], root: Path, environment: dict[str, str], capture: Capture, timeout_seconds: int
) -> tuple[int, bool]:
    process = _start(command, root, environment)
    reader = threading.Thread(target=_read_output, args=(process.stdout, capture), daemon=True)
    reader.start()
    try:
        code = process.wait(timeout=timeout_seconds)
        timed_out = False
    except subprocess.TimeoutExpired:
        timed_out = True
        if not _terminate(process):
            capture.malformed = True
        code = process.returncode if process.returncode is not None else 1
    reader.join(timeout=10)
    if reader.is_alive():
        capture.malformed = True
        _terminate(process)
        reader.join(timeout=10)
        if reader.is_alive():
            capture.malformed = True
    return code, timed_out


def _start(command: tuple[str, ...], root: Path, environment: dict[str, str]) -> subprocess.Popen[bytes]:
    options: dict[str, Any] = {
        "cwd": root,
        "env": environment,
        "stdout": subprocess.PIPE,
        "stderr": subprocess.STDOUT,
    }
    if os.name == "nt":
        options["creationflags"] = getattr(subprocess, "CREATE_NEW_PROCESS_GROUP", 0)
    else:
        options["start_new_session"] = True
    return subprocess.Popen(command, **options)


def _read_output(stream: Any, capture: Capture) -> None:
    if stream is None:
        capture.malformed = True
        return
    try:
        while chunk := stream.read(4096):
            capture.consume(chunk)
    except OSError:
        capture.malformed = True
    finally:
        capture.finish()
        try:
            stream.close()
        except OSError:
            capture.malformed = True


def _validate_progress(path: Path, capture: Capture, *, allow_open: bool) -> None:
    try:
        with path.open("rb") as source:
            contents = source.read(MAX_CAPTURE_BYTES + 1)
        if len(contents) > MAX_CAPTURE_BYTES:
            raise OSError
        value = json.loads(contents, object_pairs_hook=_unique_object)
    except (OSError, RecursionError, UnicodeDecodeError, json.JSONDecodeError, ValueError):
        capture.malformed = True
        return
    if not isinstance(value, dict) or set(value) != {"format", "events"}:
        capture.malformed = True
        return
    events = value.get("events")
    if value.get("format") != "cq-sdk-python-progress-v1" or not isinstance(events, list):
        capture.malformed = True
        return
    if not events or len(events) > MAX_PROGRESS_EVENTS:
        capture.malformed = True
        return
    active: str | None = None
    for event in events:
        if not _valid_progress_event(event, active):
            capture.malformed = True
            return
        phase = event["phase"]
        active = event["test"] if phase == "start" else None
    if active is not None and not allow_open:
        capture.malformed = True


def _unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    value: dict[str, Any] = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("duplicate JSON member")
        value[key] = item
    return value


def _valid_progress_event(event: Any, active: str | None) -> bool:
    if not isinstance(event, dict) or set(event) != {"phase", "test"}:
        return False
    phase, test = event.get("phase"), event.get("test")
    if not isinstance(test, str) or not test or len(test) > 512 or _safe_message(test) is None:
        return False
    if phase == "start":
        return active is None
    return phase == "end" and active == test


def report(check: str, code: int, capture: Capture, timed_out: bool, duration_ms: int) -> dict[str, Any]:
    """Return only bounded structured diagnostics and command outcome metadata."""

    status = "timed_out" if timed_out else "malformed_capture" if capture.malformed else "passed" if code == 0 else "failed"
    return {
        "format": FORMAT,
        "check": check,
        "status": status,
        "exit_code": code,
        "duration_ms": duration_ms,
        "counts": {
            "diagnostics": len(capture.diagnostics),
            "coverage_rows": len(capture.coverage_rows),
            "unparsed_sdk_lines": capture.unparsed_sdk_lines,
            "captured_bytes": min(capture.bytes_seen, MAX_CAPTURE_BYTES),
            "unittest_failures": len(capture.unittest_failures),
        },
        "phases": _phase_report(check, capture),
        "unittest_failures": list(capture.unittest_failures),
        "diagnostics": [
            {"rule": item.rule, "path": item.path, "line": item.line, "column": item.column, "message": item.message}
            for item in capture.diagnostics
        ],
        "coverage": [
            {
                "path": item.path,
                "statements": item.statements,
                "missed_statements": item.missed_statements,
                "branches": item.branches,
                "partial_branches": item.partial_branches,
                "percent": item.percent,
                "missing_lines": [{"start": start, "end": end} for start, end in item.missing_lines],
                "missing_branches": [
                    {"from": source, "to": destination if destination is not None else "exit"}
                    for source, destination in item.missing_branches
                ],
            }
            for item in capture.coverage_rows
        ],
    }


def _phase_report(check: str, capture: Capture) -> dict[str, str]:
    if check != "coverage":
        return {}
    return {
        "unittest": _phase_status(
            capture.unittest_started, capture.unittest_exit_code, capture.unittest_timed_out
        ),
        "coverage_report": _phase_status(
            capture.coverage_report_started, capture.coverage_report_exit_code, capture.coverage_report_timed_out
        ),
    }


def _phase_status(started: bool, code: int | None, timed_out: bool) -> str:
    if not started:
        return "not_run"
    if timed_out:
        return "timed_out"
    if code is None:
        return "not_run"
    return "passed" if code == 0 else "failed"


def arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", choices=("ruff", "mypy", "bandit", "coverage"), required=True)
    parser.add_argument("--report", type=Path, required=True)
    parser.add_argument("--progress", type=Path)
    parser.add_argument("--timeout-seconds", type=int)
    return parser.parse_args()


def main() -> int:
    try:
        options = arguments()
        timeout = options.timeout_seconds or (MAX_TIMEOUT_SECONDS if options.check == "coverage" else 300)
        if timeout < 1 or timeout > MAX_TIMEOUT_SECONDS:
            raise CheckError("timeout is outside the bounded range")
        if options.check == "coverage" and options.progress is None:
            raise CheckError("coverage requires a progress report path")
        report_path = options.report.resolve()
        progress_path = options.progress.resolve() if options.progress is not None else None
        code, capture, timed_out, duration = run_check(options.check, timeout, progress_path)
        protected_json(report_path, report(options.check, code, capture, timed_out, duration))
        if timed_out or capture.malformed:
            return code if code != 0 else 2
        return code
    except (CheckError, OSError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
