"""Caller-owned MCP client adapter for the four CircularQuay v1 tools."""

from __future__ import annotations

import asyncio
import json
import re
import time
from collections.abc import Awaitable, Callable, Mapping, Sequence
from datetime import datetime
from typing import Any, Protocol, TypeGuard, TypeVar

from .errors import CircularQuayError, ErrorCategory
from .types import (
    Observation,
    Observer,
    SecretDisclosure,
    ToolCatalog,
    ToolName,
    WorkloadRun,
    WorkloadStatus,
)

_MAX_WORKLOADS = 4096
_MAX_HANDLE_LENGTH = 128
_MAX_PROVIDER_VERSION_BYTES = 4_096
_MAX_SECRET_BYTES = 65_536
_MAX_STRUCTURED_BYTES = 1_048_576
_RFC3339 = re.compile(
    r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$"
)
_MODES = frozenset({"offline", "online", "hybrid"})
_STATES = frozenset(
    {
        "unprovisioned",
        "ready",
        "pending",
        "error",
        "running",
        "stopped",
        "unknown",
        "not_provisioned",
        "blocked",
        "resolved",
        "snapshotted",
        "applied",
        "activated",
        "apply_failed",
        "pending_activation",
        "partial_resolved",
        "partial_applied",
        "partial_activated",
        "partial_apply_failed",
        "partial_pending_activation",
    }
)
_RUN_STATES = frozenset({"completed", "failed", "running", "cancelled"})
_KNOWN_TOOLS = frozenset(ToolName)
_T = TypeVar("_T")


class ToolSession(Protocol):
    """The small, stable portion of the official ``mcp.ClientSession`` used here."""

    async def list_tools(self, **kwargs: Any) -> Any: ...

    async def call_tool(self, name: str, arguments: Mapping[str, Any]) -> Any: ...


class MCPClientSessionAdapter:
    """Adapt a caller-owned official :mod:`mcp` ``ClientSession``.

    The adapter does not import or construct a transport, session, credential,
    token cache, subprocess, or HTTP client.  The application owns setup and
    teardown of its official MCP session and can install ``mcp`` through this
    package's optional ``mcp`` extra only when it needs that transport.
    """

    def __init__(self, session: ToolSession) -> None:
        if session is None:
            raise CircularQuayError(ErrorCategory.INVALID, "mcp_session")
        self._session = session

    async def list_tools(self) -> Any:
        return await self._session.list_tools()

    async def call_tool(self, name: ToolName, arguments: Mapping[str, str]) -> Any:
        return await self._session.call_tool(str(name), arguments=dict(arguments))


class CircularQuayClient:
    """Typed access to non-disclosure CircularQuay MCP operations.

    The client does not retry, cache, discover credentials, or derive a profile
    from its environment.  Cancellation propagates to the caller-owned MCP
    session.  To read plaintext, make the separate deliberate transition with
    :meth:`disclosures`.
    """

    def __init__(
        self, adapter: MCPClientSessionAdapter, *, observer: Observer | None = None
    ) -> None:
        if adapter is None:
            raise CircularQuayError(ErrorCategory.INVALID, "mcp_adapter")
        self._adapter = adapter
        self._observer = observer

    def disclosures(self) -> DisclosureClient:
        """Return the explicit plaintext-disclosure interface."""

        return DisclosureClient(self)

    async def discover_tools(self) -> ToolCatalog:
        """Return exactly the current v1 tool names visible to this principal."""

        return await self._perform("discover_tools", self._adapter.list_tools, _parse_catalog)

    async def list_workloads(self) -> tuple[str, ...]:
        """List authorized workload handles without source or secret metadata."""

        return await self._tool(ToolName.LIST_WORKLOADS, {}, _parse_workloads)

    async def get_status(self, workload: str) -> WorkloadStatus:
        """Return safe state for one exact workload."""

        _validate_handle(workload, "workload")
        result = await self._tool(
            ToolName.GET_STATUS, {"workload": workload}, _parse_status
        )
        if result.workload != workload:
            raise CircularQuayError(ErrorCategory.PROTOCOL, "get_status")
        return result

    async def run_workload(self, workload: str) -> WorkloadRun:
        """Invoke a server-enrolled action for one exact workload."""

        _validate_handle(workload, "workload")
        return await self._tool(
            ToolName.RUN_WORKLOAD, {"workload": workload}, _parse_run
        )

    async def _read_secret(self, workload: str, key: str) -> SecretDisclosure:
        _validate_handle(workload, "workload")
        _validate_handle(key, "key")
        result = await self._tool(
            ToolName.READ_SECRET,
            {"workload": workload, "key": key},
            _parse_secret,
        )
        if result.alias != key:
            raise CircularQuayError(ErrorCategory.PROTOCOL, "read_secret")
        return result

    async def _tool(
        self,
        tool: ToolName,
        arguments: Mapping[str, str],
        parser: Callable[[Any], _T],
    ) -> _T:
        operation = str(tool)
        return await self._perform(
            operation,
            lambda: self._adapter.call_tool(tool, arguments),
            lambda raw: parser(_result_data(raw, operation)),
        )

    async def _perform(
        self,
        operation: str,
        invoke: Callable[[], Awaitable[Any]],
        parser: Callable[[Any], _T],
    ) -> _T:
        started = time.monotonic()
        outcome = "success"
        try:
            raw = await invoke()
            return parser(raw)
        except asyncio.CancelledError:
            outcome = "cancelled"
            raise
        except CircularQuayError as exc:
            outcome = exc.category
            raise
        except (TimeoutError, OSError):
            outcome = ErrorCategory.UNAVAILABLE
            raise CircularQuayError(ErrorCategory.UNAVAILABLE, operation) from None
        except Exception:
            outcome = ErrorCategory.PROTOCOL
            raise CircularQuayError(ErrorCategory.PROTOCOL, operation) from None
        finally:
            _observe(
                self._observer,
                operation,
                str(outcome),
                int((time.monotonic() - started) * 1000),
            )


class DisclosureClient:
    """Explicit, narrow interface for the ``read_secret`` MCP disclosure tool."""

    def __init__(self, client: CircularQuayClient) -> None:
        self._client = client

    async def read_secret(self, workload: str, key: str) -> SecretDisclosure:
        """Request one policy-authorized plaintext value from the MCP service.

        This may place plaintext in application memory.  It does not log, cache,
        serialize, or zeroize the result, and remote revocation cannot recall a
        value already returned to a client.
        """

        return await self._client._read_secret(workload, key)


def _result_data(result: Any, operation: str) -> Mapping[str, Any]:
    if _field(result, "isError", "is_error", default=False) is True:
        raise CircularQuayError(ErrorCategory.DENIED, operation)
    found, structured = _field_present(result, "structuredContent", "structured_content")
    if not found:
        raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
    data = _object(structured, operation)
    if not _within_structured_budget(data):
        raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
    return data


def _within_structured_budget(value: Mapping[str, Any]) -> bool:
    """Bound a decoded MCP DTO without retaining a second raw response copy."""

    try:
        used = 0
        encoder = json.JSONEncoder(ensure_ascii=False, separators=(",", ":"), allow_nan=False)
        for fragment in encoder.iterencode(value):
            used += len(fragment.encode("utf-8"))
            if used > _MAX_STRUCTURED_BYTES:
                return False
        return True
    except (OverflowError, RecursionError, TypeError, ValueError, UnicodeError):
        return False


def _parse_catalog(raw: Any) -> ToolCatalog:
    # The service exposes at most the four v1 tools.  Following a cursor could
    # silently turn discovery into a partial authorization decision, so v1
    # requires the complete catalog in one response.
    next_cursor = _field(raw, "nextCursor", "next_cursor", default=None)
    if next_cursor not in (None, ""):
        raise CircularQuayError(ErrorCategory.PROTOCOL, "discover_tools")
    tools = _field(raw, "tools", default=None)
    if (
        not isinstance(tools, Sequence)
        or isinstance(tools, (str, bytes))
        or len(tools) > len(_KNOWN_TOOLS)
    ):
        raise CircularQuayError(ErrorCategory.PROTOCOL, "discover_tools")
    available: list[ToolName] = []
    for tool in tools:
        name = _field(tool, "name", default=None)
        try:
            typed = ToolName(name)
        except (TypeError, ValueError):
            raise CircularQuayError(ErrorCategory.PROTOCOL, "discover_tools") from None
        if typed in available:
            raise CircularQuayError(ErrorCategory.PROTOCOL, "discover_tools")
        available.append(typed)
    return ToolCatalog(tuple(available))


def _parse_workloads(data: Mapping[str, Any]) -> tuple[str, ...]:
    _only(data, {"workloads"}, "list_workloads")
    values = data.get("workloads")
    if not isinstance(values, list) or len(values) > _MAX_WORKLOADS:
        raise CircularQuayError(ErrorCategory.PROTOCOL, "list_workloads")
    workloads: list[str] = []
    for value in values:
        workload = _validated_handle(value, "list_workloads")
        if workload in workloads:
            raise CircularQuayError(ErrorCategory.PROTOCOL, "list_workloads")
        workloads.append(workload)
    if workloads != sorted(workloads):
        raise CircularQuayError(ErrorCategory.PROTOCOL, "list_workloads")
    return tuple(workloads)


def _parse_status(data: Mapping[str, Any]) -> WorkloadStatus:
    _only(data, {"workload", "mode", "state"}, "get_status")
    workload = _validated_handle(data.get("workload"), "get_status")
    mode = data.get("mode")
    state = data.get("state")
    if not isinstance(mode, str) or not isinstance(state, str):
        raise CircularQuayError(ErrorCategory.PROTOCOL, "get_status")
    if mode not in _MODES or state not in _STATES:
        raise CircularQuayError(ErrorCategory.PROTOCOL, "get_status")
    return WorkloadStatus(workload=workload, mode=mode, state=state)


def _parse_run(data: Mapping[str, Any]) -> WorkloadRun:
    _only(data, {"id", "status", "exit_code"}, "run_workload")
    identifier = _validated_handle(data.get("id"), "run_workload")
    status = data.get("status")
    exit_code = data.get("exit_code")
    if (
        not isinstance(status, str)
        or status not in _RUN_STATES
        or isinstance(exit_code, bool)
        or not isinstance(exit_code, int)
        or not 0 <= exit_code <= 255
    ):
        raise CircularQuayError(ErrorCategory.PROTOCOL, "run_workload")
    return WorkloadRun(identifier=identifier, status=status, exit_code=exit_code)


def _parse_secret(data: Mapping[str, Any]) -> SecretDisclosure:
    _only(data, {"alias", "value", "version", "expires_at"}, "read_secret")
    alias = _validated_handle(data.get("alias"), "read_secret")
    value = data.get("value")
    if not _bounded_utf8_text(value, _MAX_SECRET_BYTES):
        raise CircularQuayError(ErrorCategory.PROTOCOL, "read_secret")
    version = _secret_version(data.get("version"))
    parsed = _secret_expiry(data.get("expires_at"))
    return SecretDisclosure(alias=alias, version=version, expires_at=parsed, _value=value)


def _secret_version(value: Any) -> str | None:
    if value is None:
        return None
    if not _bounded_utf8_text(value, _MAX_PROVIDER_VERSION_BYTES) or _contains_c0_or_del(value):
        raise CircularQuayError(ErrorCategory.PROTOCOL, "read_secret")
    return value


def _bounded_utf8_text(value: Any, maximum: int) -> TypeGuard[str]:
    if not isinstance(value, str):
        return False
    try:
        return len(value.encode("utf-8")) <= maximum
    except UnicodeEncodeError:
        return False


def _contains_c0_or_del(value: str) -> bool:
    return any(ord(character) < 0x20 or ord(character) == 0x7F for character in value)


def _secret_expiry(value: Any) -> datetime | None:
    parsed = _parse_timestamp(value, "read_secret") if value is not None else None
    if _is_legacy_zero_time(parsed):
        return None
    if parsed is not None and parsed <= datetime.now(parsed.tzinfo):
        raise CircularQuayError(ErrorCategory.EXPIRED, "read_secret")
    return parsed


def _is_legacy_zero_time(value: datetime | None) -> bool:
    return (
        value is not None
        and value.year == 1
        and value.month == 1
        and value.day == 1
        and value.hour == value.minute == value.second == value.microsecond == 0
    )


def _parse_timestamp(value: Any, operation: str) -> datetime:
    if not isinstance(value, str) or len(value) > 64 or _RFC3339.fullmatch(value) is None:
        raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        raise CircularQuayError(ErrorCategory.PROTOCOL, operation) from None
    if parsed.tzinfo is None:
        raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
    return parsed


def _field(value: Any, *names: str, default: Any = None) -> Any:
    if isinstance(value, Mapping):
        for name in names:
            if name in value:
                return value[name]
        return default
    for name in names:
        if hasattr(value, name):
            return getattr(value, name)
    return default


def _field_present(value: Any, *names: str) -> tuple[bool, Any]:
    if isinstance(value, Mapping):
        for name in names:
            if name in value:
                return True, value[name]
        return False, None
    for name in names:
        if hasattr(value, name):
            return True, getattr(value, name)
    return False, None


def _object(value: Any, operation: str) -> Mapping[str, Any]:
    if (
        not isinstance(value, Mapping)
        or len(value) > 8
        or not all(isinstance(key, str) for key in value)
    ):
        raise CircularQuayError(ErrorCategory.PROTOCOL, operation)
    return value


def _only(data: Mapping[str, Any], allowed: set[str], operation: str) -> None:
    if set(data) - allowed:
        raise CircularQuayError(ErrorCategory.PROTOCOL, operation)


def _validate_handle(value: Any, operation: str) -> None:
    _validated_handle(value, operation)


def _validated_handle(value: Any, operation: str) -> str:
    category = (
        ErrorCategory.INVALID
        if operation in {"workload", "key"}
        else ErrorCategory.PROTOCOL
    )
    if (
        not isinstance(value, str)
        or not value
        or len(value) > _MAX_HANDLE_LENGTH
        or value in {".", ".."}
    ):
        raise CircularQuayError(category, operation)
    if any(
        not (character.isascii() and (character.isalnum() or character in "_.-"))
        for character in value
    ):
        raise CircularQuayError(category, operation)
    return value


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
        # The observer is advisory.  Do not retain diagnostics or alter the
        # completed operation's outcome when a caller callback fails.
        del error
        return
