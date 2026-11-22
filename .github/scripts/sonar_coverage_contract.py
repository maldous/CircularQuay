"""Preserve measured Cobertura bytes while supplying an explicit checkout root."""

from __future__ import annotations

import os
import xml.etree.ElementTree as ET
from pathlib import Path
from xml.sax.saxutils import escape

MAX_COVERAGE_BYTES = 16 << 20
INVALID_COVERAGE = "sonar-python-coverage-roots-invalid"


class InvalidCoverage(ValueError):
    """The generated report cannot retain an unambiguous source mapping."""


def require(condition: bool) -> None:
    """Refuse without exposing a supplied path or XML value."""
    if not condition:
        raise InvalidCoverage(INVALID_COVERAGE)


def coverage_document(data: bytes) -> ET.Element:
    """Read only the bounded, entity-free generated coverage envelope."""
    require(isinstance(data, bytes) and 0 < len(data) <= MAX_COVERAGE_BYTES)
    require(b"<!DOCTYPE" not in data)
    try:
        report = ET.fromstring(data)
    except ET.ParseError:
        raise InvalidCoverage(INVALID_COVERAGE) from None
    require(report.tag == "coverage" and len(report.findall("./sources")) == 1)
    return report


def repository_root(repository: str) -> Path:
    """Require an explicitly selected, existing absolute checkout directory."""
    require(isinstance(repository, str) and "\x00" not in repository and Path(repository).is_absolute())
    root = Path(repository)
    require(root.is_dir())
    return root


def with_repository_source(data: bytes, repository: str) -> bytes:
    """Insert one escaped absolute root; every prior byte stays unchanged."""
    report = coverage_document(data)
    root = str(repository_root(repository))
    require(data.count(b"</sources>") == 1)
    if any(source.text == root for source in report.findall("./sources/source")):
        return data
    addition = b"<source>" + escape(root).encode("ascii", "xmlcharrefreplace") + b"</source>\n\t"
    require(len(data) + len(addition) <= MAX_COVERAGE_BYTES)
    return data.replace(b"</sources>", addition + b"</sources>", 1)


def explicit_source_roots(report: ET.Element, checkout: str) -> list[str]:
    """Keep valid explicit roots; blank or normalized-dot text adds no root."""
    roots = []
    for source in report.findall("./sources/source"):
        value = source.text or ""
        if not value.strip() or not os.path.isabs(value) and os.path.normpath(value) == ".":
            continue
        root = os.path.normpath(os.path.join(checkout, value))
        require(Path(root).is_dir())
        roots.append(root)
    require(bool(roots))
    return roots


def resolve_filename(name: str, roots: list[str]) -> str:
    """Refuse absent or multiply resolved files rather than dropping measures."""
    require(bool(name))
    candidates = [name] if os.path.isabs(name) else [os.path.join(root, name) for root in roots]
    matches = [os.path.normpath(path) for path in candidates if Path(path).is_file()]
    require(len(matches) == 1)
    return matches[0]


def resolved_file_paths(data: bytes, repository: str) -> set[str]:
    """Check generated-report resolution without inventing a checkout fallback.

    Pinned CoberturaParser ignores blank roots and requires each relative file
    to exist below exactly one explicit root. Matching metadata is not proof of
    Sonar indexing/import and is not a filesystem sandbox or authorization.
    """
    report = coverage_document(data)
    roots = explicit_source_roots(report, str(repository_root(repository)))
    resolved = {resolve_filename(item.get("filename", ""), roots) for item in report.findall(".//class")}
    require(bool(resolved))
    return resolved
