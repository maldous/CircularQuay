"""Retain public-source audit evidence, not logs or live/provider audit material."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import stat
import subprocess
from pathlib import Path

MAX_COVERAGE = 16 << 20
MAX_REPORT = 4 << 20
MAX_LINE = 1 << 20
MODULE = "github.com/maldous/CircularQuay/"
HEX40 = re.compile(r"[0-9a-f]{40}")
HEX64 = re.compile(r"[0-9a-f]{64}")
TOKEN = re.compile(r"[A-Za-z0-9_.:-]{1,128}")
BLOCK = re.compile(r"(.+):([0-9]+)\.([0-9]+),([0-9]+)\.([0-9]+) ([0-9]+) ([0-9]+)")
STATUSES = {"passed", "failed", "not_run", "skipped", "cancelled"}


class InvalidEvidence(ValueError):
    """An input cannot be safely retained as this source's bounded evidence."""


def require(condition: bool) -> None:
    if not condition:
        raise InvalidEvidence("invalid bounded audit evidence")


def private_directory(path: Path) -> Path:
    absolute = path.absolute()
    require(absolute.resolve(strict=True) == absolute)
    info = absolute.stat()
    require(stat.S_ISDIR(info.st_mode))
    require(info.st_uid == os.geteuid() and info.st_mode & 0o077 == 0)
    return absolute


def bounded_file(path: Path, maximum: int) -> bytes:
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(descriptor, "rb") as stream:
        before = os.fstat(stream.fileno())
        require(stat.S_ISREG(before.st_mode))
        require(before.st_uid == os.geteuid() and before.st_mode & 0o077 == 0)
        require(before.st_nlink == 1 and 0 < before.st_size <= maximum)
        content = stream.read(maximum + 1)
        after = os.fstat(stream.fileno())
        require(len(content) == before.st_size and len(content) <= maximum)
        require((before.st_size, before.st_mtime_ns) == (after.st_size, after.st_mtime_ns))
    return content


def git_output(source: Path, *arguments: str) -> bytes:
    command = ["git", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=" + os.devnull]
    # Inherited Git selectors/configuration can redirect -C to another checkout.
    # These read-only operations need no credentials or caller Git environment.
    environment = {
        "PATH": os.environ.get("PATH", os.defpath),
        "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
        "GIT_OPTIONAL_LOCKS": "0", "GIT_NO_REPLACE_OBJECTS": "1",
    }
    result = subprocess.run(
        [*command, "-C", str(source), *arguments],
        check=True, capture_output=True, timeout=30, env=environment,
    )
    return result.stdout


def tracked_source(source: Path, expected: str) -> set[str]:
    require(HEX40.fullmatch(expected) is not None)
    require(source.resolve(strict=True) == source.absolute())
    toplevel = Path(git_output(source, "rev-parse", "--show-toplevel").decode().strip())
    require(toplevel.resolve(strict=True) == source.absolute())
    require(git_output(source, "rev-parse", "HEAD").decode().strip() == expected)
    require(not git_output(source, "status", "--porcelain", "--untracked-files=no"))
    return set(git_output(source, "ls-files", "-z", "--", "*.go").decode().split("\0")) - {""}


def validate_block(line: str, tracked: set[str]) -> None:
    require(len(line.encode()) <= MAX_LINE)
    match = BLOCK.fullmatch(line)
    require(match is not None)
    assert match is not None
    name = match[1]
    require(name.startswith(MODULE))
    relative = name[len(MODULE):]
    require(relative in tracked and not any(part in {"", ".", ".."} for part in relative.split("/")))
    coordinates = [int(match[index]) for index in range(2, 6)]
    require(all(0 < value <= 10_000_000 for value in coordinates))
    require(tuple(coordinates[:2]) <= tuple(coordinates[2:]))
    require(int(match[6]) <= 10_000_000 and int(match[7]) <= (1 << 64) - 1)


def validate_coverage(content: bytes, tracked: set[str]) -> None:
    lines = content.decode("utf-8").splitlines()
    require(len(lines) > 1 and lines[0] in {"mode: atomic", "mode: count", "mode: set"})
    for line in lines[1:]:
        validate_block(line, tracked)


def unique_object(pairs: list[tuple[str, object]]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in pairs:
        require(key not in result)
        result[key] = value
    return result


def reject_constant(_: str) -> None:
    raise InvalidEvidence("non-finite report number")


def safe_check(check: object) -> dict[str, object]:
    require(isinstance(check, dict))
    assert isinstance(check, dict)
    require(isinstance(check.get("name"), str) and TOKEN.fullmatch(check["name"]) is not None)
    require(isinstance(check.get("status"), str) and check["status"] in STATUSES)
    require(type(check.get("required")) is bool)
    result = {key: check[key] for key in ("name", "status", "required")}
    if "code" in check:
        require(isinstance(check["code"], str) and TOKEN.fullmatch(check["code"]) is not None)
        result["code"] = check["code"]
    return result


def source_identity(value: object, expected: str) -> dict[str, str]:
    require(isinstance(value, dict))
    assert isinstance(value, dict)
    require(value.get("commit") == expected)
    require(isinstance(value.get("tree"), str) and HEX40.fullmatch(value["tree"]) is not None)
    digest = value.get("manifest_sha256")
    require(isinstance(digest, str) and HEX64.fullmatch(digest) is not None)
    return {key: value[key] for key in ("commit", "tree", "manifest_sha256")}


def selected_go_probe(manifest: object) -> dict | None:
    if not isinstance(manifest, dict) or not isinstance(manifest.get("tools"), list):
        return None
    tools = manifest["tools"]
    require(len(tools) <= 32)
    selected = [tool for tool in tools if isinstance(tool, dict) and tool.get("id") == "go"]
    require(len(selected) <= 1)
    if selected and selected[0].get("probe") == {"id": "go", "status": "passed"}:
        return selected[0]
    return None


def source_runtime(report: dict[str, object]) -> dict[str, str]:
    """Retain no tool paths/stdout; incomplete old evidence cannot supply join pins."""
    if "go_version" not in report:
        return {}
    version, platform = report["go_version"], report.get("platform")
    require(isinstance(version, str) and re.fullmatch(r"go[0-9]+\.[0-9]+\.[0-9]+", version) is not None)
    require(isinstance(platform, str) and re.fullmatch(r"[a-z0-9]+/[a-z0-9]+", platform) is not None)
    result = {"go_version": version, "platform": platform}
    tool = selected_go_probe(report.get("tool_manifest"))
    if tool is not None:
        require(tool.get("declared_version") == f"go version {version} {platform}")
        require(isinstance(tool.get("sha256"), str) and HEX64.fullmatch(tool["sha256"]) is not None)
        result["go_tool_sha256"] = tool["sha256"]
    return result


def report_metadata(content: bytes, expected: str) -> dict[str, object]:
    report = json.loads(content, object_pairs_hook=unique_object, parse_constant=reject_constant)
    require(isinstance(report, dict) and type(report.get("version")) is int)
    require(report["version"] == 1 and report.get("profile") == "source")
    require(report.get("commit") == expected)
    require(type(report.get("passed")) is bool and report.get("production_qualified") is False)
    checks = report.get("checks")
    require(isinstance(checks, list) and len(checks) <= 512)
    selected_checks = [safe_check(check) for check in checks]
    if report["passed"]:
        require(all(not check["required"] or check["status"] == "passed" for check in selected_checks))
    result: dict[str, object] = {
        "version": 1, "profile": "source", "commit": expected,
        "passed": report["passed"], "production_qualified": False,
        "checks": selected_checks,
    }
    if report.get("source") is not None:
        result["source"] = source_identity(report["source"], expected)
    result.update(source_runtime(report))
    return result


def retain(source: Path, audit: Path, output: Path, expected: str) -> dict[str, object]:
    tracked = tracked_source(source, expected)
    audit = private_directory(audit)
    coverage = bounded_file(audit / "coverage.out", MAX_COVERAGE)
    report = bounded_file(audit / "report.json", MAX_REPORT)
    validate_coverage(coverage, tracked)
    metadata = report_metadata(report, expected)
    tree = git_output(source, "rev-parse", "HEAD^{tree}").decode().strip()
    if "source" in metadata:
        require(metadata["source"]["tree"] == tree)
    # Recheck the selected checkout after reading the bounded input files.
    require(tracked_source(source, expected) == tracked)
    metadata["original_report_sha256"] = hashlib.sha256(report).hexdigest()
    metadata["coverage_sha256"] = hashlib.sha256(coverage).hexdigest()
    metadata["retention_only"] = True
    metadata["coverage_semantics"] = "Original audit result; not recomputed or promoted by retention"
    encoded = (json.dumps(metadata, sort_keys=True, indent=2) + "\n").encode()
    output = output.absolute()
    require(output.parent.resolve(strict=True) == output.parent)
    require(not output.is_relative_to(source.absolute()) and not output.is_relative_to(audit))
    output.mkdir(mode=0o700, exist_ok=False)
    for name, data in (("coverage.out", coverage), ("metadata.json", encoded)):
        descriptor = os.open(output / name, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(data)
    return {"retained_files": 2, "commit": expected, "audit_passed": metadata["passed"]}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--audit", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--expected-commit", required=True)
    args = parser.parse_args()
    try:
        result = retain(args.source, args.audit, args.output, args.expected_commit)
    except (OSError, ValueError, UnicodeError, subprocess.SubprocessError):
        print("Bounded audit evidence was not retained.")
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
