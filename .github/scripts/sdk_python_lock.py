#!/usr/bin/env python3
"""Write a reviewed, hash-pinned Python CI lock from a pip JSON report.

The resolver runs in a hosted GitHub Actions job.  This program never invokes
pip or contacts an index; it turns that bounded report into a fail-closed lock
that can be reviewed and installed with ``--require-hashes``.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import tempfile
import zipfile
from pathlib import Path
from typing import Any
from urllib.parse import unquote, urlsplit

MAX_REPORT_BYTES = 4 * 1024 * 1024
MAX_REQUIREMENTS_BYTES = 256 * 1024
MAX_PACKAGES = 4096
SHA256 = re.compile(r"[0-9a-f]{64}\Z")
PACKAGE_NAME = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]*\Z")
VERSION = re.compile(r"[A-Za-z0-9][A-Za-z0-9.!+_-]*\Z")
NORMALIZE_NAME = re.compile(r"[-_.]+")
LOCK_LINE = re.compile(
    r"(?P<name>[A-Za-z0-9][A-Za-z0-9._-]*)=="
    r"(?P<version>[A-Za-z0-9][A-Za-z0-9.!+_-]*) "
    r"--hash=sha256:(?P<digest>[0-9a-f]{64})\Z"
)


class ValidationError(Exception):
    """Raised when an untrusted resolver report cannot become a safe lock."""


def canonical_name(value: object) -> str:
    """Return a PEP 503 normalized name, rejecting untrusted non-names."""
    if not isinstance(value, str) or not PACKAGE_NAME.fullmatch(value):
        raise ValidationError("invalid package name")
    return NORMALIZE_NAME.sub("-", value).lower()


def checked_version(value: object) -> str:
    """Accept one bounded, printable package version without whitespace."""
    if not isinstance(value, str) or not VERSION.fullmatch(value):
        raise ValidationError("invalid package version")
    return value


def read_json_report(path: Path) -> tuple[dict[str, Any], str]:
    """Read one size-bounded JSON object without surfacing report contents."""
    try:
        if path.stat().st_size > MAX_REPORT_BYTES:
            raise ValidationError("report exceeds size limit")
        contents = path.read_bytes()
        if len(contents) > MAX_REPORT_BYTES:
            raise ValidationError("report exceeds size limit")
        value = json.loads(contents)
    except (OSError, UnicodeDecodeError, json.JSONDecodeError, RecursionError) as error:
        raise ValidationError("report is invalid") from error
    if not isinstance(value, dict):
        raise ValidationError("report must be an object")
    if value.get("version") != "1":
        raise ValidationError("report version is unsupported")
    checked_version(value.get("pip_version"))
    return value, hashlib.sha256(contents).hexdigest()


def requested_pins(path: Path) -> dict[str, str]:
    """Read direct exact pins; index URLs and unconstrained requirements fail."""
    try:
        if path.stat().st_size > MAX_REQUIREMENTS_BYTES:
            raise ValidationError("requirements exceed size limit")
        contents = path.read_bytes()
        if len(contents) > MAX_REQUIREMENTS_BYTES:
            raise ValidationError("requirements exceed size limit")
        lines = contents.decode("utf-8").splitlines()
    except (OSError, UnicodeDecodeError) as error:
        raise ValidationError("requirements are invalid") from error

    pins: dict[str, str] = {}
    for line in lines:
        candidate = line.split("#", 1)[0].strip()
        if not candidate:
            continue
        if candidate.count("==") != 1:
            raise ValidationError("requirements must use exact pins")
        raw_name, raw_version = candidate.split("==", 1)
        name = canonical_name(raw_name)
        version = checked_version(raw_version)
        if name in pins:
            raise ValidationError("requirements contain duplicate packages")
        pins[name] = version
    if not pins:
        raise ValidationError("requirements contain no pins")
    return pins


def wheel_hash(entry: dict[str, Any]) -> str:
    """Require an immutable public PyPI wheel URL and a SHA-256 digest."""
    download = entry.get("download_info")
    if not isinstance(download, dict):
        raise ValidationError("package download information is missing")
    url = download.get("url")
    if not isinstance(url, str):
        raise ValidationError("package download URL is invalid")
    try:
        parsed = urlsplit(url)
        port = parsed.port
    except ValueError as error:
        raise ValidationError("package download URL is invalid") from error
    if (
        parsed.scheme != "https"
        or parsed.hostname != "files.pythonhosted.org"
        or port is not None
        or parsed.username is not None
        or parsed.password is not None
        or parsed.query
        or parsed.fragment
        or not unquote(parsed.path).endswith(".whl")
    ):
        raise ValidationError("package must be a public PyPI wheel")
    archive = download.get("archive_info")
    hashes = archive.get("hashes") if isinstance(archive, dict) else None
    digest = hashes.get("sha256") if isinstance(hashes, dict) else None
    if not isinstance(digest, str) or not SHA256.fullmatch(digest):
        raise ValidationError("package SHA-256 is invalid")
    return digest


def darwin_intake(
    archive: Path, wheel: Path, python: str, custody: str
) -> dict[str, Any]:
    """Bind retained bytes to a reviewed native receipt; never fetch or extract."""
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9:/._@+-]{0,511}", custody):
        raise ValidationError("reviewed immutable custody identifier is required")
    source = Path(__file__).resolve().parents[2] / "sdk/python"
    qualified = json.loads((source / "darwin-qualified-wheels.json").read_bytes())
    policy = hashlib.sha256(
        (source / "darwin-source-policy.json").read_bytes()
    ).hexdigest()
    if qualified["source_policy_sha256"] != policy:
        raise ValidationError("Darwin source policy changed")
    choices = [
        row for row in qualified["qualified_repeat_builds"] if row["python"] == python
    ]
    if len(choices) != 1:
        raise ValidationError("Python patch has no qualified wheel")
    selected = choices[0]
    if not selected.get("custody") or custody != selected["custody"]:
        raise ValidationError(
            "native artifact has no matching reviewed immutable custody binding"
        )
    if wheel.name != selected["wheel"]:
        raise ValidationError("wheel filename differs from qualified input")
    if (
        archive.stat().st_size > 128 * 1024 * 1024
        or wheel.stat().st_size > 128 * 1024 * 1024
    ):
        raise ValidationError("native input exceeds size limit")
    if hashlib.sha256(archive.read_bytes()).hexdigest() != selected["artifact_sha256"]:
        raise ValidationError("native archive digest differs")
    digest = hashlib.sha256(wheel.read_bytes()).hexdigest()
    if digest != selected["wheel_sha256"]:
        raise ValidationError("native wheel digest differs")
    verify_archive_member(archive, wheel, digest)
    return {
        **selected,
        "custody": custody,
        "source_policy_sha256": policy,
        "run_id": qualified["run_id"],
        "workflow_head": qualified["workflow_head"],
        "feature": qualified["feature"],
        "wheel_uri": wheel.resolve().as_uri(),
    }


def verify_archive_member(archive: Path, wheel: Path, digest: str) -> None:
    """Check membership without extracting untrusted archive paths."""
    with zipfile.ZipFile(archive) as bundle:
        members = [
            item
            for item in bundle.infolist()
            if Path(item.filename).name == wheel.name and not item.is_dir()
        ]
        if len(members) != 1 or members[0].file_size != wheel.stat().st_size:
            raise ValidationError(
                "native archive must contain the exact qualified wheel once"
            )
        if hashlib.sha256(bundle.read(members[0])).hexdigest() != digest:
            raise ValidationError("archive member differs from supplied wheel")


def local_wheel_hash(entry: dict[str, Any], intake: dict[str, Any]) -> str:
    """Admit only the byte-verified cryptography wheel at its explicit local URI."""
    metadata = entry.get("metadata", {})
    download = entry.get("download_info", {})
    archive = download.get("archive_info") if isinstance(download, dict) else None
    hashes = archive.get("hashes") if isinstance(archive, dict) else None
    if (
        not isinstance(metadata, dict)
        or not isinstance(download, dict)
        or not isinstance(hashes, dict)
        or metadata.get("name") != "cryptography"
        or metadata.get("version") != "50.0.2"
        or download.get("url") != intake["wheel_uri"]
        or hashes.get("sha256") != intake["wheel_sha256"]
    ):
        raise ValidationError("local wheel differs from reviewed Darwin intake")
    return str(intake["wheel_sha256"])


def verify_native_target(report: dict[str, Any], intake: dict[str, Any] | None) -> None:
    """Keep the local-wheel exception confined to the qualified native target."""
    if intake is not None:
        environment = report.get("environment", {})
        if (
            not isinstance(environment, dict)
            or environment.get("sys_platform") != "darwin"
            or environment.get("platform_machine") != "x86_64"
            or environment.get("implementation_name") != "cpython"
            or environment.get("python_full_version") != intake["python"]
        ):
            raise ValidationError(
                "local wheel requires its qualified native target report"
            )


def lock_entries(
    report: dict[str, Any], pins: dict[str, str], intake: dict[str, Any] | None = None
) -> list[tuple[str, str, str]]:
    """Validate report packages and return normalized sorted lock tuples."""
    verify_native_target(report, intake)
    install = report.get("install")
    if not isinstance(install, list) or not install or len(install) > MAX_PACKAGES:
        raise ValidationError("report install list is invalid")
    packages: dict[str, tuple[str, str]] = {}
    for entry in install:
        if not isinstance(entry, dict):
            raise ValidationError("report package is invalid")
        if entry.get("is_yanked") is not False:
            raise ValidationError("report contains a yanked package")
        metadata = entry.get("metadata")
        if not isinstance(metadata, dict):
            raise ValidationError("report package metadata is invalid")
        name = canonical_name(metadata.get("name"))
        version = checked_version(metadata.get("version"))
        if name in packages:
            raise ValidationError("report contains duplicate packages")
        digest = (
            local_wheel_hash(entry, intake)
            if intake is not None and name == "cryptography"
            else wheel_hash(entry)
        )
        packages[name] = (version, digest)
    if intake is not None and "cryptography" not in packages:
        raise ValidationError("Darwin closure omits qualified cryptography wheel")
    for name, version in pins.items():
        actual = packages.get(name)
        if actual is None or actual[0] != version:
            raise ValidationError("report does not contain every requested exact pin")
    return [(name, *packages[name]) for name in sorted(packages)]


def verify_lock(path: Path, pins: dict[str, str]) -> None:
    """Reject a committed lock that differs from the authoritative direct pins."""
    try:
        if path.stat().st_size > MAX_REPORT_BYTES:
            raise ValidationError("lock exceeds size limit")
        contents = path.read_bytes()
        if len(contents) > MAX_REPORT_BYTES:
            raise ValidationError("lock exceeds size limit")
        lines = contents.decode("utf-8").splitlines()
    except (OSError, UnicodeDecodeError) as error:
        raise ValidationError("lock is invalid") from error
    if not lines or len(lines) > MAX_PACKAGES:
        raise ValidationError("lock is invalid")
    packages: dict[str, str] = {}
    previous_name = ""
    for line in lines:
        match = LOCK_LINE.fullmatch(line)
        if match is None:
            raise ValidationError("lock has an invalid entry")
        name = canonical_name(match.group("name"))
        version = checked_version(match.group("version"))
        if name in packages or name <= previous_name:
            raise ValidationError("lock is not uniquely sorted")
        packages[name] = version
        previous_name = name
    for name, version in pins.items():
        if packages.get(name) != version:
            raise ValidationError("lock does not contain every requested exact pin")


def atomic_write(path: Path, contents: bytes) -> None:
    """Replace one output with restrictive permissions after complete writing."""
    temporary: str | None = None
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        descriptor, temporary = tempfile.mkstemp(prefix=".cq-", dir=path.parent)
        with os.fdopen(descriptor, "wb") as output:
            output.write(contents)
            output.flush()
            os.fsync(output.fileno())
        os.chmod(temporary, 0o600)
        os.replace(temporary, path)
    except OSError as error:
        if temporary is not None:
            try:
                os.unlink(temporary)
            except OSError:
                pass
        raise ValidationError("could not write output") from error


def receipt(
    report_digest: str, revision: str, packages: int, pip_version: object
) -> bytes:
    """Produce provenance without report URLs, environment values, or secrets."""
    if not re.fullmatch(r"[0-9a-f]{40,64}", revision):
        raise ValidationError("revision is invalid")
    lines = [
        "format=cq-python-lock-v1",
        f"revision={revision}",
        f"report_sha256={report_digest}",
        f"packages={packages}",
        f"python={sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}",
        f"platform={sys.platform}",
        f"implementation={sys.implementation.name}",
        f"pip={checked_version(pip_version)}",
    ]
    return ("\n".join(lines) + "\n").encode("utf-8")


def arguments() -> argparse.Namespace:
    """Parse the intentionally small, non-secret command-line interface."""
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--report", type=Path)
    mode.add_argument("--verify-lock", type=Path)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--receipt", type=Path)
    parser.add_argument("--revision")
    parser.add_argument("--darwin-archive", type=Path)
    parser.add_argument("--darwin-wheel", type=Path)
    parser.add_argument("--darwin-python")
    parser.add_argument("--darwin-custody")
    parser.add_argument(
        "--requirements", type=Path, default=Path("sdk/python/requirements-ci.txt")
    )
    return parser.parse_args()


def main() -> int:
    """Validate a hosted resolver report and write its lock and receipt."""
    try:
        options = arguments()
        pins = requested_pins(options.requirements)
        if options.verify_lock is not None:
            verify_lock(options.verify_lock, pins)
            return 0
        if (
            options.report is None
            or options.output is None
            or options.receipt is None
            or options.revision is None
        ):
            raise ValidationError("report lock arguments are incomplete")
        report, report_digest = read_json_report(options.report)
        native = [
            options.darwin_archive,
            options.darwin_wheel,
            options.darwin_python,
            options.darwin_custody,
        ]
        intake = None
        if any(item is not None for item in native):
            if not all(item is not None for item in native):
                raise ValidationError("Darwin intake arguments are incomplete")
            intake = darwin_intake(*native)
        packages = lock_entries(report, pins, intake)
        lines = [
            f"{name}=={version} --hash=sha256:{digest}"
            for name, version, digest in packages
        ]
        atomic_write(options.output, ("\n".join(lines) + "\n").encode("utf-8"))
        provenance = receipt(
            report_digest, options.revision, len(packages), report.get("pip_version")
        )
        if intake is not None:
            # A runner path locates bytes only; it is not the durable artifact identity.
            provenance += (
                "darwin_intake="
                + json.dumps(
                    {key: value for key, value in intake.items() if key != "wheel_uri"},
                    sort_keys=True,
                    separators=(",", ":"),
                )
                + "\n"
            ).encode("utf-8")
        atomic_write(options.receipt, provenance)
    except (ValidationError, OSError, ValueError, zipfile.BadZipFile) as error:
        print(f"error: {error}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
