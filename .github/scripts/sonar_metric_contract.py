"""Validate pinned Sonar measure shapes without inventing unavailable evidence."""

from __future__ import annotations

import json
import os
import re
import stat
import sys
from decimal import Decimal
from typing import Any, NamedTuple

MAX_INPUT_BYTES = 64 << 10
PERCENT = re.compile(r"(?:0|[1-9][0-9]*)(?:\.[0-9]+)?")
COUNT = re.compile(r"0|[1-9][0-9]*")
INVALID_EVIDENCE = "sonar-metric-evidence-invalid"
ARGUMENT_STAGE = "sonar-metric-stage-arguments-refused"
READ_STAGE = "sonar-metric-stage-read-refused"
SCHEMA_STAGE = "sonar-metric-stage-schema-refused"
OPTIONAL_COUNTS = frozenset({"violations", "security_hotspots"})
REQUIRED_METRICS = frozenset({
    "coverage", "new_coverage", "duplicated_lines_density", "new_duplicated_lines_density",
})
ALLOWED_METRICS = REQUIRED_METRICS | OPTIONAL_COUNTS

# SonarSource/sonarqube 26.10.0.133078, MeasureDtoToWsMeasure, writes
# new_* measures as PeriodValue. No baseline or applicability is inferred here.


class InvalidEvidence(ValueError):
    """The bounded response does not satisfy the pinned measure contract."""


class Family(NamedTuple):
    """One unchanged overall-and-new-code threshold and its fixed diagnostics."""

    metrics: tuple[str, str]
    expected: Decimal
    missing: str
    invalid: str
    failed: str


FAMILIES = (
    Family(("coverage", "new_coverage"), Decimal(100),
           "sonar-coverage-evidence-missing", "sonar-coverage-evidence-invalid", "sonar-coverage-not-100"),
    Family(("duplicated_lines_density", "new_duplicated_lines_density"), Decimal(0),
           "sonar-duplication-evidence-missing", "sonar-duplication-evidence-invalid", "sonar-duplication-not-zero"),
)
EVIDENCE_FAILURES = frozenset({INVALID_EVIDENCE}) | frozenset(
    code for family in FAMILIES for code in (family.missing, family.invalid)
)


def require(condition: bool) -> None:
    """Refuse malformed evidence without echoing any supplied field or value."""
    if not condition:
        raise InvalidEvidence(INVALID_EVIDENCE)


def unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    """Reject duplicate JSON members rather than accepting a last-writer value."""
    result: dict[str, Any] = {}
    for key, value in pairs:
        require(key not in result)
        result[key] = value
    return result


def reject_constant(_value: str) -> None:
    """Reject JSON extensions such as NaN and Infinity."""
    raise InvalidEvidence(INVALID_EVIDENCE)


def metric_index(document: Any) -> dict[str, dict[str, Any]]:
    """Bind only the six requested, unique metric identities to project cq."""
    require(isinstance(document, dict))
    component = document.get("component")
    require(isinstance(component, dict) and component.get("key") == "cq")
    measures = component.get("measures")
    require(isinstance(measures, list) and len(measures) <= len(ALLOWED_METRICS))
    result: dict[str, dict[str, Any]] = {}
    for measure in measures:
        require(isinstance(measure, dict))
        name = measure.get("metric")
        require(isinstance(name, str) and name in ALLOWED_METRICS and name not in result)
        if name in OPTIONAL_COUNTS:
            validate_count(measure)
        result[name] = measure
    return result


def validate_count(measure: dict[str, Any]) -> None:
    """Validate optional count shape without replacing the separate issue gates."""
    value = measure.get("value")
    require("period" not in measure and isinstance(value, str))
    require(len(value) <= 32 and COUNT.fullmatch(value) is not None)


def percentage(measure: dict[str, Any], on_new_code: bool) -> Decimal:
    """Read current value or pinned new-code period.value, never a fallback."""
    if on_new_code:
        require("value" not in measure)
        period = measure.get("period")
        require(isinstance(period, dict))
        require("index" not in period or type(period["index"]) is int and period["index"] == 1)
        value = period.get("value")
    else:
        require("period" not in measure)
        value = measure.get("value")
    require(isinstance(value, str))
    require(len(value) <= 32 and PERCENT.fullmatch(value) is not None)
    number = Decimal(value)
    require(Decimal(0) <= number <= Decimal(100))
    return number


def family_failure(measures: dict[str, dict[str, Any]], family: Family) -> str | None:
    """Keep unavailable or invalid evidence distinct from a measured failure."""
    if any(name not in measures for name in family.metrics):
        return family.missing
    try:
        values = [percentage(measures[name], name.startswith("new_")) for name in family.metrics]
    except InvalidEvidence:
        return family.invalid
    if any(value != family.expected for value in values):
        return family.failed
    return None


def evaluate_metrics(data: bytes) -> tuple[str, ...]:
    """Return fixed failure codes; an empty tuple requires all four strict metrics."""
    try:
        require(isinstance(data, bytes) and 0 < len(data) <= MAX_INPUT_BYTES)
        document = json.loads(data.decode("utf-8"), object_pairs_hook=unique_object, parse_constant=reject_constant)
        measures = metric_index(document)
    except (ValueError, RecursionError):
        return (INVALID_EVIDENCE,)
    failures = [family_failure(measures, family) for family in FAMILIES]
    return tuple(code for code in failures if code is not None)


def read_evidence(path: str) -> bytes:
    """Read one bounded regular Linux workflow file without following its final link."""
    descriptor = os.open(path, os.O_RDONLY | os.O_NONBLOCK | os.O_NOFOLLOW)
    with os.fdopen(descriptor, "rb") as stream:
        before = os.fstat(stream.fileno())
        require(stat.S_ISREG(before.st_mode) and 0 < before.st_size <= MAX_INPUT_BYTES)
        data = stream.read(MAX_INPUT_BYTES + 1)
        after = os.fstat(stream.fileno())
        require(len(data) == before.st_size and len(data) <= MAX_INPUT_BYTES)
        require((before.st_size, before.st_mtime_ns) == (after.st_size, after.st_mtime_ns))
    return data


def main(arguments: list[str] | None = None) -> int:
    """Exit 0 for a strict pass, 1 for measured failure, 2 for unavailable evidence."""
    arguments = sys.argv[1:] if arguments is None else arguments
    failures = (INVALID_EVIDENCE,)
    stage = ARGUMENT_STAGE
    if len(arguments) == 1:
        stage = READ_STAGE
        try:
            data = read_evidence(arguments[0])
            stage = SCHEMA_STAGE
            failures = evaluate_metrics(data)
        except (OSError, ValueError):
            failures = (INVALID_EVIDENCE,)
    for code in failures:
        print(code)
    # The original generic failure and exit remain mandatory. This fixed label
    # reports only the last attempted boundary, never an error, value or path.
    if failures == (INVALID_EVIDENCE,):
        print(stage)
    if any(code in EVIDENCE_FAILURES for code in failures):
        return 2
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
