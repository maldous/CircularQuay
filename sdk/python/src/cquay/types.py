"""Public typed values for the CircularQuay v1 SDK contract."""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from datetime import datetime
from enum import StrEnum
from typing import TypeAlias

_MCP_READ_OPERATION = "read_secret"

JSONScalar: TypeAlias = str | int | float | bool | None
JSONValue: TypeAlias = JSONScalar | list["JSONValue"] | dict[str, "JSONValue"]


class ToolName(StrEnum):
    """The complete v1 CircularQuay MCP tool catalog."""

    LIST_WORKLOADS = "list_workloads"
    GET_STATUS = "get_status"
    RUN_WORKLOAD = "run_workload"
    READ_SECRET = _MCP_READ_OPERATION


@dataclass(frozen=True, slots=True)
class ToolCatalog:
    """The tools currently advertised to the authenticated MCP principal."""

    available: tuple[ToolName, ...]

    def includes(self, tool: ToolName | str) -> bool:
        """Return whether the server advertised *tool* for this principal."""

        try:
            return ToolName(tool) in self.available
        except ValueError:
            return False


@dataclass(frozen=True, slots=True)
class WorkloadStatus:
    """Safe metadata returned for one authorized workload."""

    workload: str
    mode: str
    state: str


@dataclass(frozen=True, slots=True)
class WorkloadRun:
    """Sanitized outcome of a locally enrolled workload action."""

    identifier: str
    status: str
    exit_code: int


@dataclass(frozen=True, slots=True, repr=False)
class SecretDisclosure:
    """One intentionally released plaintext value.

    Constructed only by :class:`cquay.client.DisclosureClient`.  ``repr()``,
    ``str()``, and :meth:`redacted_dict` never contain plaintext.  Python and
    downstream libraries can copy strings, so this type makes no zeroization
    promise.  Call :meth:`reveal` only at the narrowly scoped hand-off point.
    """

    alias: str
    version: str | None
    expires_at: datetime | None
    _value: str

    def __repr__(self) -> str:
        return f"SecretDisclosure(alias={self.alias!r}, value='[REDACTED]')"

    __str__ = __repr__

    def reveal(self) -> str:
        """Return plaintext after the caller has explicitly opted into disclosure."""

        return self._value

    def redacted_dict(self) -> dict[str, JSONValue]:
        """Return serializable metadata with the plaintext replaced by a marker."""

        value: dict[str, JSONValue] = {"alias": self.alias, "value": "[REDACTED]"}
        if self.version is not None:
            value["version"] = self.version
        if self.expires_at is not None:
            value["expires_at"] = self.expires_at.isoformat()
        return value


@dataclass(frozen=True, slots=True)
class Observation:
    """Non-identifying operation telemetry emitted after an SDK operation.

    It intentionally excludes workload IDs, argument values, URLs, secret values,
    response bodies, token material, and raw errors.
    """

    operation: str
    outcome: str
    duration_ms: int


Observer: TypeAlias = Callable[[Observation], None]
