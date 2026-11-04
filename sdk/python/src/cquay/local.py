"""Integrity-pinned adapter for the existing native ``cquay`` CLI."""

from __future__ import annotations

import asyncio
import hashlib
import json
import math
import os
import re
import stat
import time
from collections.abc import Awaitable, Callable, Mapping, Sequence
from enum import StrEnum
from pathlib import Path
from typing import Any, TypeGuard, TypeVar
from urllib.parse import urlsplit

from .errors import CircularQuayError, ErrorCategory
from .types import JSONValue, Observation, Observer

_SHA256 = re.compile(r"^[0-9a-f]{64}$")
_MAX_INPUT_BYTES = 1_048_576
_MAX_OUTPUT_BYTES = 1_048_576
_MAX_ERROR_BYTES = 512
_MAX_VERSION_LENGTH = 256
_CHUNK_BYTES = 65_536
_TERM_GRACE_SECONDS = 5
_IJSON_SAFE_INTEGER = 9_007_199_254_740_991
_MAX_IJSON_DEPTH = 32
_MAX_IJSON_COLLECTION = 1024
_MAX_IJSON_NODES = 16_384
_DENIED_ENV_PREFIXES = ("AWS_", "AZURE_", "GOOGLE_", "VAULT_", "OP_", "BW_", "CQUAY_CONFIG_")
_RemotePinArguments = tuple[str, str, str, str, str, str, str, str]
_T = TypeVar("_T")


class RunStdio(StrEnum):
    """The only supported SDK policy for a real workload's standard streams."""

    INHERIT = "inherit"


class LocalCLIClient:
    """Run the installed native CLI without a shell or ambient environment.

    ``executable``, ``config``, and ``state_dir`` must be operator-provided
    absolute paths.  ``expected_sha256`` pins the executable bytes and is
    verified before every child process starts.  This protects against ordinary
    replacement, not a privileged attacker capable of racing executable opens or
    changing the host filesystem after verification; use platform code signing,
    ACLs, and release provenance as well.

    The client accepts only an explicit environment mapping.  It never copies
    ``os.environ``.  Provider/bootstrap credential variables are rejected so a
    language SDK cannot become an ambient credential broker.
    """

    def __init__(
        self,
        executable: str | Path,
        expected_sha256: str,
        config: str | Path,
        state_dir: str | Path,
        *,
        environment: Mapping[str, str] | None = None,
        observer: Observer | None = None,
    ) -> None:
        self._executable = _absolute_path(executable, "executable")
        self._config = _absolute_path(config, "config")
        self._state_dir = _absolute_path(state_dir, "state_dir")
        if not isinstance(expected_sha256, str) or not _SHA256.fullmatch(expected_sha256):
            raise CircularQuayError(ErrorCategory.INVALID, "expected_sha256")
        self._expected_sha256 = expected_sha256
        self._environment = _validate_environment(environment)
        self._observer = observer
        self.verify_identity()

    def verify_identity(self) -> None:
        """Verify the exact regular executable path and expected SHA-256 digest."""

        try:
            if self._executable.resolve(strict=True) != self._executable:
                raise OSError
            before = os.lstat(self._executable)
            if stat.S_ISLNK(before.st_mode) or not stat.S_ISREG(before.st_mode):
                raise OSError
            digest, opened = _file_sha256(self._executable)
            after = os.lstat(self._executable)
            identities_match = (
                _identity(before) == _identity(opened)
                and _identity(before) == _identity(after)
            )
            if not identities_match or digest != self._expected_sha256:
                raise OSError
        except (OSError, ValueError):
            raise CircularQuayError(ErrorCategory.INTEGRITY, "cli_identity") from None

    async def version(self) -> dict[str, JSONValue]:
        """Return the native executable's v1 version metadata."""

        value = await self._json_object("version", ("version",))
        major = _version_major(value)
        if major is None:
            raise CircularQuayError(ErrorCategory.PROTOCOL, "version")
        if major != "1":
            raise CircularQuayError(ErrorCategory.UNSUPPORTED, "version")
        return value

    async def assert_runtime_compatibility(self) -> dict[str, JSONValue]:
        """Verify that the pinned executable reports a compatible v1 runtime."""

        return await self.version()

    async def providers(self) -> JSONValue:
        """Return native provider capability metadata without provider access."""

        return await self._json("providers", ("providers",))

    async def update_check(self) -> JSONValue:
        """Perform one policy-configured, structured update-awareness check.

        This method runs only when explicitly called.  It never schedules a
        check, prompts, downloads, installs, or activates software.
        """

        return await self._json("update_check", ("update", "check", "--json"))

    async def init(self, *, output: str | Path | None = None) -> dict[str, JSONValue]:
        """Create the native CLI's synthetic policy fixture, refusing overwrite."""

        args: list[str] = ["init"]
        if output is not None:
            args.extend(("--output", str(_absolute_path(output, "output"))))
        return await self._json_object("init", tuple(args))

    async def config_init(self, *, output: str | Path | None = None) -> dict[str, JSONValue]:
        """Create the same protected synthetic policy via ``config init``."""

        args: list[str] = ["config", "init"]
        if output is not None:
            args.extend(("--output", str(_absolute_path(output, "output"))))
        return await self._json_object("config_init", tuple(args))

    async def doctor(
        self, *, fix: bool = False, identity_only: bool = False
    ) -> dict[str, JSONValue]:
        """Run local readiness checks; ``fix`` only requests native JSON formatting."""

        if fix and identity_only:
            raise CircularQuayError(ErrorCategory.INVALID, "doctor")
        args = ["doctor"]
        if fix:
            args.append("--fix")
        if identity_only:
            args.append("--identity-only")
        return await self._json_object("doctor", tuple(args), accepted_codes={0, 1})

    async def config_validate(self) -> dict[str, JSONValue]:
        return await self._json_object("config_validate", ("config", "validate"))

    async def config_get(self, pointer: str) -> dict[str, JSONValue]:
        _pointer(pointer)
        return await self._json_object(
            "config_get", ("config", "get", "--pointer", pointer)
        )

    async def config_list(self, pointer: str = "") -> dict[str, JSONValue]:
        _pointer(pointer)
        args = ["config", "list"]
        if pointer:
            args.extend(("--pointer", pointer))
        return await self._json_object("config_list", tuple(args))

    async def config_set(
        self,
        pointer: str,
        value: JSONValue,
        *,
        expect_sha256: str | None = None,
    ) -> dict[str, JSONValue]:
        """Set one JSON pointer value through stdin, never through process argv."""

        _pointer(pointer)
        body = _json_input(value, "config_set")
        args = ["config", "set", "--pointer", pointer, "--value-file", "-"]
        _append_expected_digest(args, expect_sha256)
        return await self._json_object("config_set", tuple(args), stdin=body)

    async def config_delete(
        self, pointer: str, *, expect_sha256: str | None = None
    ) -> dict[str, JSONValue]:
        _pointer(pointer)
        args = ["config", "delete", "--pointer", pointer]
        _append_expected_digest(args, expect_sha256)
        return await self._json_object("config_delete", tuple(args))

    async def config_replace(
        self, policy: JSONValue, *, expect_sha256: str | None = None
    ) -> dict[str, JSONValue]:
        """Replace policy JSON through stdin after native full-policy validation."""

        body = _json_input(policy, "config_replace")
        args = ["config", "replace", "--file", "-"]
        _append_expected_digest(args, expect_sha256)
        return await self._json_object("config_replace", tuple(args), stdin=body)

    async def plan(self) -> dict[str, JSONValue]:
        return await self._json_object("plan", ("plan",))

    async def status(self, workload: str) -> dict[str, JSONValue]:
        _handle(workload, "workload")
        return await self._json_object(
            "status", ("status", "--workload", workload, "--json")
        )

    async def snapshot(self, workload: str, *, target: str | None = None) -> dict[str, JSONValue]:
        return await self._workload_json("snapshot", workload, target=target)

    async def apply(self, workload: str, *, target: str) -> dict[str, JSONValue]:
        if target is None:
            raise CircularQuayError(ErrorCategory.INVALID, "apply_target")
        return await self._workload_json("apply", workload, target=target)

    async def sync(self, workload: str, *, target: str | None = None) -> dict[str, JSONValue]:
        return await self._workload_json("sync", workload, target=target, once=True)

    async def bootstrap(
        self,
        *,
        config_source: str | None = None,
        config_ref: str | None = None,
        config_path: str | None = None,
        config_sha256: str | None = None,
        output: str | Path | None = None,
    ) -> dict[str, JSONValue]:
        """Use the CLI's pinned configuration bootstrap contract.

        A remote source requires all immutable pin fields.  The SDK does not read
        ``CQUAY_CONFIG_*`` environment defaults or discover a repository.
        """

        args = ["bootstrap", "--non-interactive"]
        pins = _pinned_config_arguments(
            config_source, config_ref, config_path, config_sha256, "bootstrap_pins"
        )
        if pins is not None:
            args.extend(pins)
        if output is not None:
            args.extend(("--output", str(_absolute_path(output, "output"))))
        return await self._json_object("bootstrap", tuple(args))

    async def flow(
        self,
        workload: str,
        *,
        config_source: str | None = None,
        config_ref: str | None = None,
        config_path: str | None = None,
        config_sha256: str | None = None,
    ) -> int:
        """Run the native CI flow and return its native exit code.

        Flow can dispatch a workload that writes values to standard streams, so
        it follows :meth:`run`: output is inherited rather than captured.
        """

        _handle(workload, "workload")
        args = ["flow", "run", "--workload", workload, "--non-interactive"]
        pins = _pinned_config_arguments(
            config_source, config_ref, config_path, config_sha256, "flow_pins"
        )
        if pins is not None:
            args.extend(pins)
        return await self._inherit("flow", tuple(args))

    async def run(self, workload: str, *, override: Sequence[str] = (), stdio: RunStdio) -> int:
        """Run an enrolled workload with explicitly inherited standard streams.

        Child output can contain values.  This method therefore never captures,
        parses, logs, or returns stdout/stderr.  ``stdio`` is deliberately
        required so callers make the visibility decision at each invocation.
        """

        _handle(workload, "workload")
        if stdio is not RunStdio.INHERIT:
            raise CircularQuayError(ErrorCategory.INVALID, "run_stdio")
        program = tuple(_argument(item, "run_override") for item in override)
        if program and not os.path.isabs(program[0]):
            raise CircularQuayError(ErrorCategory.INVALID, "run_override")
        separator_and_program = ("--", *program) if program else ()
        args = ("run", "--workload", workload, *separator_and_program)
        return await self._inherit("run", args)

    async def _workload_json(
        self,
        command: str,
        workload: str,
        *,
        target: str | None = None,
        once: bool = False,
    ) -> dict[str, JSONValue]:
        _handle(workload, "workload")
        args = [command, "--workload", workload]
        if target is not None:
            _handle(target, "target")
            args.extend(("--target", target))
        if once:
            args.append("--once")
        return await self._json_object(command, tuple(args))

    async def _json(
        self,
        operation: str,
        arguments: tuple[str, ...],
        *,
        stdin: bytes | None = None,
        accepted_codes: set[int] | None = None,
    ) -> JSONValue:
        output, code, failure = await self._captured(operation, arguments, stdin=stdin)
        if code not in (accepted_codes or {0}):
            _raise_exit(operation, failure or ErrorCategory.PROCESS)
        try:
            value = json.loads(output)
        except (TypeError, ValueError):
            raise CircularQuayError(ErrorCategory.PROTOCOL, operation) from None
        if not _safe_json(value):
            raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
        return value

    async def _json_object(
        self,
        operation: str,
        arguments: tuple[str, ...],
        *,
        stdin: bytes | None = None,
        accepted_codes: set[int] | None = None,
    ) -> dict[str, JSONValue]:
        value = await self._json(
            operation,
            arguments,
            stdin=stdin,
            accepted_codes=accepted_codes,
        )
        if not isinstance(value, dict):
            raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
        return value

    async def _captured(
        self,
        operation: str,
        arguments: tuple[str, ...],
        *,
        stdin: bytes | None = None,
    ) -> tuple[bytes, int, ErrorCategory | None]:
        return await self._run_owned_process(
            operation,
            lambda: self._start_captured(arguments, stdin),
            lambda process: _captured_result(process, operation),
        )

    async def _run_owned_process(
        self,
        operation: str,
        start: Callable[[], Awaitable[asyncio.subprocess.Process]],
        consume: Callable[
            [asyncio.subprocess.Process], Awaitable[tuple[_T, str]]
        ],
    ) -> _T:
        """Run one verified child while preserving its primary outcome."""

        started = time.monotonic()
        outcome = "success"
        process: asyncio.subprocess.Process | None = None
        try:
            self.verify_identity()
            process = await start()
            value, outcome = await consume(process)
            return value
        except asyncio.CancelledError:
            outcome = "cancelled"
            if process is not None:
                await _reap(process)
            raise
        except CircularQuayError as exc:
            outcome = exc.category
            raise
        except TimeoutError:
            outcome = ErrorCategory.UNAVAILABLE
            raise CircularQuayError(ErrorCategory.UNAVAILABLE, operation) from None
        except OSError:
            outcome = ErrorCategory.PROCESS
            raise CircularQuayError(ErrorCategory.PROCESS, operation) from None
        except Exception:
            outcome = ErrorCategory.PROCESS
            raise CircularQuayError(ErrorCategory.PROCESS, operation) from None
        finally:
            if process is not None and process.returncode is None:
                await _reap(process)
            _observe(
                self._observer,
                operation,
                str(outcome),
                int((time.monotonic() - started) * 1000),
            )

    async def _start_captured(
        self, arguments: tuple[str, ...], stdin: bytes | None
    ) -> asyncio.subprocess.Process:
        process = await asyncio.create_subprocess_exec(
            str(self._executable),
            "--config",
            str(self._config),
            "--state-dir",
            str(self._state_dir),
            *arguments,
            stdin=asyncio.subprocess.PIPE if stdin is not None else asyncio.subprocess.DEVNULL,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            env=dict(self._environment),
        )
        if stdin is not None and process.stdin is not None:
            process.stdin.write(stdin)
            await process.stdin.drain()
            process.stdin.close()
        return process

    async def _inherit(self, operation: str, arguments: tuple[str, ...]) -> int:
        return await self._run_owned_process(
            operation,
            lambda: self._start_inherited(arguments),
            _inherited_result,
        )

    async def _start_inherited(
        self, arguments: tuple[str, ...]
    ) -> asyncio.subprocess.Process:
        return await asyncio.create_subprocess_exec(
            str(self._executable),
            "--config",
            str(self._config),
            "--state-dir",
            str(self._state_dir),
            *arguments,
            stdin=None,
            stdout=None,
            stderr=None,
            env=dict(self._environment),
        )


def _absolute_path(value: str | Path, operation: str) -> Path:
    path = Path(value)
    if not path.is_absolute():
        raise CircularQuayError(ErrorCategory.INVALID, operation)
    return path


def _version_major(value: dict[str, JSONValue]) -> str | None:
    if not isinstance(value, dict) or set(value) != {"version", "commit", "go", "maturity"}:
        return None
    version = value["version"]
    if not all(isinstance(value[field], str) for field in ("commit", "go", "maturity")):
        return None
    if not isinstance(version, str):
        return None
    return _semver_major(version)


def _semver_major(version: str) -> str | None:
    """Return a strict SemVer major or ``None`` without retaining malformed input."""

    try:
        encoded_length = len(version.encode("utf-8"))
    except UnicodeEncodeError:
        return None
    if not version or encoded_length > _MAX_VERSION_LENGTH:
        return None
    if version.startswith("v"):
        version = version[1:]
    core_and_prerelease, *build_parts = version.split("+")
    if len(build_parts) > 1:
        return None
    if build_parts and not _valid_semver_identifiers(build_parts[0], numeric=False):
        return None
    core, separator, prerelease = core_and_prerelease.partition("-")
    if separator and not _valid_semver_identifiers(prerelease, numeric=True):
        return None
    parts = core.split(".")
    if len(parts) != 3 or not all(_valid_semver_number(part) for part in parts):
        return None
    return parts[0]


def _valid_semver_identifiers(value: str, *, numeric: bool) -> bool:
    return bool(value) and all(
        _valid_semver_identifier(identifier, numeric=numeric)
        for identifier in value.split(".")
    )


def _valid_semver_identifier(value: str, *, numeric: bool) -> bool:
    valid_characters = all(
        (character.isascii() and character.isalnum()) or character == "-"
        for character in value
    )
    if not value or not valid_characters:
        return False
    if numeric and all("0" <= character <= "9" for character in value):
        return _valid_semver_number(value)
    return True


def _valid_semver_number(value: str) -> bool:
    return value == "0" or (value[:1] in "123456789" and value.isascii() and value.isdecimal())


def _pinned_config_arguments(
    source: str | None,
    ref: str | None,
    path: str | None,
    digest: str | None,
    incomplete_operation: str,
) -> _RemotePinArguments | None:
    pins = (source, ref, path, digest)
    if any(item is not None for item in pins) and not all(item is not None for item in pins):
        raise CircularQuayError(ErrorCategory.INVALID, incomplete_operation)
    if source is None:
        return None
    if ref is None or path is None or digest is None:
        raise CircularQuayError(ErrorCategory.INVALID, incomplete_operation)
    if not _valid_remote_source(source):
        raise CircularQuayError(ErrorCategory.INVALID, "config_source")
    _sha1(ref, "config_ref")
    _sha256(digest, "config_sha256")
    if not isinstance(path, str) or not path or len(path) > 1024:
        raise CircularQuayError(ErrorCategory.INVALID, "config_path")
    return (
        "--config-source",
        source,
        "--config-ref",
        ref,
        "--config-path",
        path,
        "--config-sha256",
        digest,
    )


def _valid_remote_source(value: Any) -> bool:
    if not isinstance(value, str) or not value or len(value) > 4096 or "\x00" in value:
        return False
    try:
        source = urlsplit(value)
        _ = source.port
    except ValueError:
        return False
    return (
        source.scheme == "https"
        and bool(source.netloc)
        and "@" not in source.netloc
        and "?" not in value
        and "#" not in value
        and not source.query
        and not source.fragment
    )


def _file_sha256(path: Path) -> tuple[str, os.stat_result]:
    hasher = hashlib.sha256()
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    descriptor = os.open(path, flags)
    try:
        opened = os.fstat(descriptor)
        if not stat.S_ISREG(opened.st_mode):
            raise OSError
        with os.fdopen(descriptor, "rb", closefd=False) as handle:
            while chunk := handle.read(_CHUNK_BYTES):
                hasher.update(chunk)
        return hasher.hexdigest(), opened
    finally:
        os.close(descriptor)


def _identity(info: os.stat_result) -> tuple[int, int, int, int]:
    return (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns)


def _validate_environment(environment: Mapping[str, str] | None) -> dict[str, str]:
    if environment is None:
        return {}
    clean: dict[str, str] = {}
    for name, value in environment.items():
        if not _safe_environment_entry(name, value):
            raise CircularQuayError(ErrorCategory.INVALID, "environment")
        clean[name] = value
    return clean


def _safe_environment_entry(name: Any, value: Any) -> bool:
    if not isinstance(name, str) or not isinstance(value, str):
        return False
    if not name or len(name) > 256 or len(value) > 4096:
        return False
    if "=" in name or "\x00" in name or "\x00" in value:
        return False
    return not name.upper().startswith(_DENIED_ENV_PREFIXES)


def _argument(value: str, operation: str) -> str:
    if not isinstance(value, str) or "\x00" in value or len(value) > 4096:
        raise CircularQuayError(ErrorCategory.INVALID, operation)
    return value


def _handle(value: str, operation: str) -> None:
    if (
        not isinstance(value, str)
        or not value
        or len(value) > 128
        or value in {".", ".."}
        or any(
            not (item.isascii() and (item.isalnum() or item in "_.-"))
            for item in value
        )
    ):
        raise CircularQuayError(ErrorCategory.INVALID, operation)


def _pointer(value: str) -> None:
    if (
        not isinstance(value, str)
        or len(value) > 4096
        or (value and not value.startswith("/"))
        or "\x00" in value
    ):
        raise CircularQuayError(ErrorCategory.INVALID, "json_pointer")
    index = 0
    while index < len(value):
        if value[index] != "~":
            index += 1
            continue
        if index + 1 == len(value) or value[index + 1] not in {"0", "1"}:
            raise CircularQuayError(ErrorCategory.INVALID, "json_pointer")
        index += 2


def _sha256(value: str | None, operation: str) -> None:
    if not isinstance(value, str) or not _SHA256.fullmatch(value):
        raise CircularQuayError(ErrorCategory.INVALID, operation)


def _sha1(value: str | None, operation: str) -> None:
    if not isinstance(value, str) or re.fullmatch(r"[0-9a-f]{40}", value) is None:
        raise CircularQuayError(ErrorCategory.INVALID, operation)


def _append_expected_digest(arguments: list[str], digest: str | None) -> None:
    if digest is not None:
        _sha256(digest, "expect_sha256")
        arguments.extend(("--expect-sha256", digest))


def _json_input(value: JSONValue, operation: str) -> bytes:
    if not _is_i_json(value):
        raise CircularQuayError(ErrorCategory.INVALID, operation)
    try:
        body = json.dumps(
            value,
            separators=(",", ":"),
            ensure_ascii=False,
            allow_nan=False,
        ).encode("utf-8")
    except (TypeError, ValueError):
        raise CircularQuayError(ErrorCategory.INVALID, operation) from None
    if len(body) > _MAX_INPUT_BYTES:
        raise CircularQuayError(ErrorCategory.INVALID, operation)
    return body


def _is_i_json(value: Any, *, depth: int = 0) -> bool:
    """Accept portable I-JSON values for SDK convenience configuration calls."""

    return _is_i_json_node(value, depth, [0])


def _is_i_json_node(value: Any, depth: int, nodes: list[int]) -> bool:
    if depth > _MAX_IJSON_DEPTH:
        return False
    nodes[0] += 1
    if nodes[0] > _MAX_IJSON_NODES:
        return False
    if isinstance(value, list):
        return _is_i_json_list(value, depth, nodes)
    if isinstance(value, dict):
        return _is_i_json_object(value, depth, nodes)
    return _is_i_json_scalar(value)


def _is_i_json_scalar(value: Any) -> bool:
    if value is None or isinstance(value, (bool, str)):
        return True
    if isinstance(value, int):
        return -_IJSON_SAFE_INTEGER <= value <= _IJSON_SAFE_INTEGER
    if isinstance(value, float):
        return math.isfinite(value)
    return False


def _is_i_json_list(value: list[Any], depth: int, nodes: list[int]) -> bool:
    return len(value) <= _MAX_IJSON_COLLECTION and all(
        _is_i_json_node(item, depth + 1, nodes) for item in value
    )


def _is_i_json_object(value: dict[Any, Any], depth: int, nodes: list[int]) -> bool:
    if len(value) > _MAX_IJSON_COLLECTION:
        return False
    return all(
        isinstance(key, str) and _is_i_json_node(item, depth + 1, nodes)
        for key, item in value.items()
    )


def _safe_json(value: Any, *, depth: int = 0) -> TypeGuard[JSONValue]:
    if depth > 32:
        return False
    if value is None or isinstance(value, bool):
        return True
    if isinstance(value, str):
        return len(value) <= _MAX_OUTPUT_BYTES
    if isinstance(value, int):
        return not isinstance(value, bool)
    if isinstance(value, float):
        return math.isfinite(value)
    if isinstance(value, list):
        return _safe_json_items(value, depth)
    if isinstance(value, dict):
        return _safe_json_object(value, depth)
    return False


def _safe_json_items(items: list[Any], depth: int) -> bool:
    return len(items) <= 1024 and all(_safe_json(item, depth=depth + 1) for item in items)


def _safe_json_object(items: dict[Any, Any], depth: int) -> bool:
    if len(items) > 1024:
        return False
    return all(
        _safe_json_key(key) and _safe_json(value, depth=depth + 1)
        for key, value in items.items()
    )


def _safe_json_key(value: Any) -> bool:
    return isinstance(value, str) and len(value) <= 512


async def _read_bounded(
    stream: asyncio.StreamReader | None,
    limit: int = _MAX_OUTPUT_BYTES,
) -> tuple[bytes, bool]:
    if stream is None:
        return b"", False
    parts: list[bytes] = []
    total = 0
    exceeded = False
    while chunk := await stream.read(_CHUNK_BYTES):
        if not exceeded and total + len(chunk) <= limit:
            parts.append(chunk)
            total += len(chunk)
        else:
            return b"", True
    return b"".join(parts), exceeded


async def _reap(process: asyncio.subprocess.Process) -> bool:
    """Terminate an owned child in bounded time and report whether it reaped."""

    waiter = asyncio.create_task(process.wait())
    if process.returncode is None:
        try:
            process.terminate()
        except ProcessLookupError:
            pass
    try:
        await asyncio.wait_for(asyncio.shield(waiter), timeout=_TERM_GRACE_SECONDS)
        return True
    except TimeoutError:
        if process.returncode is None:
            try:
                process.kill()
            except ProcessLookupError:
                pass
        try:
            await asyncio.wait_for(asyncio.shield(waiter), timeout=_TERM_GRACE_SECONDS)
            return True
        except TimeoutError:
            return False
    finally:
        if not waiter.done():
            waiter.cancel()
        await asyncio.gather(waiter, return_exceptions=True)


async def _cleanup_reader_tasks(*tasks: asyncio.Task[Any] | None) -> None:
    active = [task for task in tasks if task is not None]
    for task in active:
        if not task.done():
            task.cancel()
    if active:
        await asyncio.gather(*active, return_exceptions=True)


async def _capture_process_output(
    process: asyncio.subprocess.Process, operation: str
) -> tuple[bytes, int, ErrorCategory | None]:
    """Read owned process streams, enforcing limits and deterministic cleanup."""

    stdout_task = asyncio.create_task(_read_bounded(process.stdout))
    stderr_task = asyncio.create_task(_read_bounded(process.stderr, _MAX_ERROR_BYTES))
    try:
        output, error_output, overflow = await _collect_streams(stdout_task, stderr_task)
        if overflow:
            if not await _reap(process):
                raise CircularQuayError(ErrorCategory.PROCESS, operation)
            raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
        code = await process.wait()
        if code != 0:
            failure = (
                ErrorCategory.PROCESS
                if not error_output
                else _cli_failure_category(error_output)
            )
            return output, code, failure
        return output, code, None
    finally:
        if process.returncode is None:
            await _reap(process)
        await _cleanup_reader_tasks(stdout_task, stderr_task)


async def _captured_result(
    process: asyncio.subprocess.Process, operation: str
) -> tuple[tuple[bytes, int, ErrorCategory | None], str]:
    output, code, failure = await _capture_process_output(process, operation)
    return (output, code, failure), str(failure or "success")


async def _inherited_result(
    process: asyncio.subprocess.Process,
) -> tuple[int, str]:
    code = await process.wait()
    outcome = ErrorCategory.PROCESS if code != 0 else "success"
    return code, str(outcome)


async def _collect_streams(
    stdout_task: asyncio.Task[tuple[bytes, bool]], stderr_task: asyncio.Task[tuple[bytes, bool]]
) -> tuple[bytes, bytes, bool]:
    """Consume both pipes until EOF or either bounded reader reports overflow."""

    pending: set[asyncio.Task[tuple[bytes, bool]]] = {stdout_task, stderr_task}
    stdout = b""
    stderr = b""
    while pending:
        finished, pending = await asyncio.wait(pending, return_when=asyncio.FIRST_COMPLETED)
        for task in finished:
            data, overflow = task.result()
            if overflow:
                return b"", b"", True
            if task is stdout_task:
                stdout = data
            else:
                stderr = data
    return stdout, stderr, False


def _cli_failure_category(data: bytes) -> ErrorCategory:
    if not data:
        return ErrorCategory.PROCESS
    if len(data) > _MAX_ERROR_BYTES:
        return ErrorCategory.PROTOCOL
    try:
        document = json.loads(data, object_pairs_hook=_unique_object)
    except (TypeError, ValueError):
        return ErrorCategory.PROTOCOL
    if not isinstance(document, dict) or set(document) != {"error"}:
        return ErrorCategory.PROTOCOL
    categories = {
        "invalid_policy": ErrorCategory.INVALID,
        "access_denied": ErrorCategory.DENIED,
        "trust_failure": ErrorCategory.TRUST,
        "unavailable": ErrorCategory.UNAVAILABLE,
        "activation_failed": ErrorCategory.UNAVAILABLE,
        "missing_value": ErrorCategory.UNAVAILABLE,
        "expired_value": ErrorCategory.EXPIRED,
        "unsupported_capability": ErrorCategory.UNSUPPORTED,
    }
    category = document.get("error")
    if not isinstance(category, str):
        return ErrorCategory.PROTOCOL
    return categories.get(category, ErrorCategory.PROTOCOL)


def _unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise ValueError
        result[key] = value
    return result


def _raise_exit(operation: str, category: ErrorCategory) -> None:
    raise CircularQuayError(category, operation)


def _observe(observer: Observer | None, operation: str, outcome: str, duration_ms: int) -> None:
    if observer is None:
        return
    try:
        observer(
            Observation(
                operation=operation,
                outcome=outcome,
                duration_ms=max(0, duration_ms),
            )
        )
    except Exception as error:
        # Observer callbacks are advisory; retaining their errors could leak
        # provider data and must not replace a completed operation outcome.
        del error
        return
