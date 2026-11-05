#!/usr/bin/env python3
"""Validate the one reviewed Darwin Intel source-distribution exception.

This helper neither resolves nor builds packages.  It validates only evidence
for the exact source archive named by the committed policy.  Workflow and lock
integration remain deliberately separate until a hosted build closure exists.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import unquote, urlsplit


MAX_POLICY_BYTES = 16 * 1024
SHA256 = re.compile(r"[0-9a-f]{64}\Z")
PACKAGE_NAME = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]*\Z")
VERSION = re.compile(r"[A-Za-z0-9][A-Za-z0-9.!+_-]*\Z")
POLICY_FORMAT = "cq-python-darwin-source-policy-v1"
REQUIRED_KEYS = frozenset(
    {"format", "platform", "architecture", "package", "version", "url", "sha256"}
)


class ValidationError(Exception):
    """Raised when source evidence does not exactly match the reviewed policy."""


@dataclass(frozen=True)
class SourcePolicy:
    """One explicit, immutable source-distribution allowance."""

    platform: str
    architecture: str
    package: str
    version: str
    url: str
    sha256: str


def _checked_name(value: object) -> str:
    if not isinstance(value, str) or not PACKAGE_NAME.fullmatch(value):
        raise ValidationError("invalid package name")
    return re.sub(r"[-_.]+", "-", value).lower()


def _checked_version(value: object) -> str:
    if not isinstance(value, str) or not VERSION.fullmatch(value):
        raise ValidationError("invalid package version")
    return value


def _checked_digest(value: object) -> str:
    if not isinstance(value, str) or not SHA256.fullmatch(value):
        raise ValidationError("invalid SHA-256")
    return value


def _checked_source_url(value: object, package: str, version: str) -> str:
    if not isinstance(value, str):
        raise ValidationError("invalid source URL")
    try:
        parsed = urlsplit(value)
        port = parsed.port
    except ValueError as error:
        raise ValidationError("invalid source URL") from error
    filename = f"{package}-{version}.tar.gz"
    if (
        parsed.scheme != "https"
        or parsed.hostname != "files.pythonhosted.org"
        or port is not None
        or parsed.username is not None
        or parsed.password is not None
        or parsed.query
        or parsed.fragment
        or not unquote(parsed.path).endswith("/" + filename)
    ):
        raise ValidationError("source must be the exact public PyPI sdist")
    return value


def load_policy(path: Path) -> SourcePolicy:
    """Load one small, closed-schema policy from a trusted committed path."""
    try:
        if path.stat().st_size > MAX_POLICY_BYTES:
            raise ValidationError("policy exceeds size limit")
        raw = path.read_bytes()
        if len(raw) > MAX_POLICY_BYTES:
            raise ValidationError("policy exceeds size limit")
        value = json.loads(raw)
    except (OSError, UnicodeDecodeError, json.JSONDecodeError, RecursionError) as error:
        raise ValidationError("policy is invalid") from error
    if not isinstance(value, dict) or set(value) != REQUIRED_KEYS:
        raise ValidationError("policy schema is invalid")
    if value.get("format") != POLICY_FORMAT:
        raise ValidationError("policy format is unsupported")
    if value.get("platform") != "darwin" or value.get("architecture") != "x86_64":
        raise ValidationError("policy must target Darwin Intel")
    package = _checked_name(value.get("package"))
    version = _checked_version(value.get("version"))
    return SourcePolicy(
        platform="darwin",
        architecture="x86_64",
        package=package,
        version=version,
        url=_checked_source_url(value.get("url"), package, version),
        sha256=_checked_digest(value.get("sha256")),
    )


def validate_source_proof(
    policy: SourcePolicy,
    *,
    platform: object,
    architecture: object,
    package: object,
    version: object,
    url: object,
    sha256: object,
) -> None:
    """Accept only evidence that exactly matches the single policy allowance."""
    if (
        platform != policy.platform
        or architecture != policy.architecture
        or _checked_name(package) != policy.package
        or _checked_version(version) != policy.version
        or _checked_source_url(url, policy.package, policy.version) != policy.url
        or _checked_digest(sha256) != policy.sha256
    ):
        raise ValidationError("source evidence does not match the reviewed policy")


def arguments() -> argparse.Namespace:
    """Parse the non-secret evidence tuple supplied by a future hosted lane."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--policy", type=Path, required=True)
    parser.add_argument("--platform", required=True)
    parser.add_argument("--architecture", required=True)
    parser.add_argument("--package", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--url", required=True)
    parser.add_argument("--sha256", required=True)
    return parser.parse_args()


def main() -> int:
    """Validate an evidence tuple without fetching, resolving, or building."""
    try:
        options = arguments()
        validate_source_proof(
            load_policy(options.policy),
            platform=options.platform,
            architecture=options.architecture,
            package=options.package,
            version=options.version,
            url=options.url,
            sha256=options.sha256,
        )
    except ValidationError as error:
        print(f"error: {error}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
