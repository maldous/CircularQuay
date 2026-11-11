"""Sanitized Python SDK error categories."""

from __future__ import annotations

from enum import StrEnum


class ErrorCategory(StrEnum):
    """A stable failure category that never contains remote diagnostic text.

    ``CANCELLED`` is the semantic cancellation outcome for cross-SDK telemetry.
    Python operations deliberately re-raise :class:`asyncio.CancelledError`
    after owned-child cleanup, preserving structured-concurrency semantics.
    """

    INVALID = "invalid"
    DENIED = "denied"
    TRUST = "trust"
    UNAVAILABLE = "unavailable"
    EXPIRED = "expired"
    UNSUPPORTED = "unsupported"
    PROTOCOL = "protocol"
    CANCELLED = "cancelled"
    INTEGRITY = "integrity"
    PROCESS = "process"


class CircularQuayError(Exception):
    """A deliberately detail-free error suitable for application handling.

    The SDK never retains server text, command arguments, standard error, tokens,
    request bodies, or values in this exception.  Inspect :attr:`category` rather
    than parsing the message.
    """

    def __init__(self, category: ErrorCategory, operation: str) -> None:
        self.category = category
        self.operation = operation
        super().__init__(f"{category}: {operation}")
