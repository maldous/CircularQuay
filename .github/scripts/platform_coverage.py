"""Retain real default-platform Go counters and join authenticated same-run inputs.

The workflow, not artifact metadata, supplies the expected producer descriptors,
artifact IDs/digests and original job outcomes. A pins digest only binds bytes;
it does not authenticate their author. No network, SDK enrollment or service
activation occurs here. The existing Linux-only SDK join is a separate contract.
An aggregate coverage pass never rewrites an original Source/platform result.
Selected executable SHA/version/platform and observed clean checkout states are
producer measurements, not publisher, whole-GOROOT or atomic hash-to-exec
attestation. Windows reparse/descriptor/hash checks do not assert Unix ownership
or independently attest NTFS ACLs. Uploaded members contain only public-source
coordinates and whitelisted provenance, never environment/configuration bodies.
"""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import math
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import zipfile
import zlib


REPOSITORY = "maldous/CircularQuay"
REPOSITORY_ID = "1444116753"
GO_VERSION = "go1.27.4"
MODULE = "github.com/maldous/CircularQuay/"
ROLES = {"linux": "tests", "macos": "native-macos", "windows": "native-windows", "source": "assurance"}
PLATFORMS = {"linux": {"linux/amd64"}, "macos": {"darwin/amd64", "darwin/arm64"},
             "windows": {"windows/amd64", "windows/arm64"}, "source": {"linux/amd64"}}
MAX_PROFILE = 16 << 20
MAX_METADATA = 1 << 20
MAX_ARCHIVE = 32 << 20
MAX_TOOL = 128 << 20
BLOCK = re.compile(r"(.+):([0-9]+)\.([0-9]+),([0-9]+)\.([0-9]+) ([0-9]+) ([0-9]+)")
STATUSES = {"passed", "failed", "not_run", "skipped", "cancelled"}
TOKEN = re.compile(r"[A-Za-z0-9_.:-]{1,128}")


class InvalidEvidence(ValueError):
    """A fixed input cannot be authenticated as the requested counter evidence."""


def require(value):
    if not value:
        raise InvalidEvidence("invalid bounded platform coverage evidence")


def keys(value, expected):
    require(type(value) is dict and set(value) == set(expected))


def hex_value(value, length=64):
    require(type(value) is str and re.fullmatch(r"[0-9a-f]{" + str(length) + "}", value) is not None)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def encoded(value):
    return (json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False) + "\n").encode("utf-8")


def unique(items):
    value = {}
    for name, item in items:
        require(name not in value)
        value[name] = item
    return value


def invalid_constant(_):
    raise InvalidEvidence("invalid bounded platform coverage evidence")


def bounded_integer(text):
    require(len(text.lstrip("-")) <= 20)
    return int(text)


def bounded_float(text):
    require(len(text) <= 64)
    value = float(text)
    require(math.isfinite(value))
    return value


def strict_json(data):
    value = json.loads(data, object_pairs_hook=unique, parse_constant=invalid_constant,
                       parse_int=bounded_integer, parse_float=bounded_float)
    require(type(value) is dict)
    return value


def file_identity(info):
    return (info.st_dev, info.st_ino, info.st_mode, info.st_nlink,
            info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def safe_file(path, maximum, private=False):
    path = path.absolute()
    require(path.resolve(strict=True) == path)
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_NONBLOCK", 0)
    descriptor = os.open(path, flags)
    with os.fdopen(descriptor, "rb") as stream:
        before = os.fstat(stream.fileno())
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1)
        require(not getattr(before, "st_file_attributes", 0) & 0x400)
        require(0 < before.st_size <= maximum)
        if os.name == "posix":
            require(before.st_uid in (0, os.geteuid()) and not before.st_mode & 0o022)
            require(not private or (before.st_uid == os.geteuid() and not before.st_mode & 0o077))
        data = stream.read(maximum + 1)
        require(len(data) == before.st_size)
        require(file_identity(before) == file_identity(os.fstat(stream.fileno())))
        require(file_identity(before) == file_identity(os.stat(path, follow_symlinks=False)))
        return data


def write_new(path, data):
    require(len(data) <= MAX_PROFILE)
    flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0)
    descriptor = os.open(path, flags, 0o600)
    with os.fdopen(descriptor, "wb") as stream:
        stream.write(data)


def new_directory(path, source):
    path = path.absolute()
    require(path.parent.resolve(strict=True) == path.parent)
    require(not path.is_relative_to(source.absolute()))
    path.mkdir(mode=0o700)
    return path


def child_environment():
    value = {"PATH": os.environ.get("PATH", os.defpath)}
    # Windows subprocess startup may require the system directory. It is not
    # retained, and no token, credential, HOME or arbitrary environment is copied.
    if os.name == "nt" and "SystemRoot" in os.environ:
        value["SystemRoot"] = os.environ["SystemRoot"]
    return value


def git(source, *arguments):
    environment = child_environment()
    environment.update(GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull,
                       GIT_OPTIONAL_LOCKS="0", GIT_NO_REPLACE_OBJECTS="1")
    result = subprocess.run(["git", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=" + os.devnull,
                             "-C", str(source), *arguments], env=environment, check=False,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
    require(result.returncode == 0 and not result.stderr and len(result.stdout) <= 4 << 20)
    return result.stdout


def source_identity(source, expected):
    hex_value(expected, 40)
    source = source.absolute()
    require(source.resolve(strict=True) == source)
    require(Path(git(source, "rev-parse", "--show-toplevel").decode().strip()).resolve() == source)
    require(git(source, "rev-parse", "HEAD").decode().strip() == expected)
    tree = git(source, "rev-parse", "HEAD^{tree}").decode().strip()
    hex_value(tree, 40)
    require(not git(source, "status", "--porcelain=v1", "--untracked-files=no"))
    for ignored in ([], ["--ignored"]):
        require(not git(source, "ls-files", "--others", *ignored, "--exclude-standard", "-z", "--", "*.go"))
    tracked = tracked_go(git(source, "ls-files", "--stage", "-z", "--", "*.go"))
    locks = {name: digest(safe_file(source / name, MAX_METADATA)) for name in ("go.mod", "go.sum")}
    return {"commit": expected, "tree": tree, "tracked_clean": True, "locks": locks}, tracked


def tracked_go(data):
    tracked = set()
    for row in data.split(b"\0"):
        if not row:
            continue
        header, raw = row.split(b"\t", 1)
        mode, blob, stage = header.split()
        name = raw.decode("utf-8")
        require(mode in (b"100644", b"100755") and stage == b"0")
        hex_value(blob.decode("ascii"), 40)
        require(not name.startswith("/") and "\\" not in name and "\0" not in name)
        require(all(part not in ("", ".", "..") for part in name.split("/")))
        require(name.endswith(".go"))
        tracked.add(name)
    require(bool(tracked))
    return tracked


def raw_origin():
    result = {"repository": os.environ.get("GITHUB_REPOSITORY"),
              "repository_id": os.environ.get("GITHUB_REPOSITORY_ID"),
              "run_id": os.environ.get("GITHUB_RUN_ID"), "attempt": os.environ.get("GITHUB_RUN_ATTEMPT"),
              "job": os.environ.get("GITHUB_JOB"), "event": os.environ.get("GITHUB_EVENT_NAME"),
              "pr_head": os.environ.get("CQ_COVERAGE_PR_HEAD"), "step": "platform_counters"}
    return result


def origin(role):
    result = raw_origin()
    validate_origin(result, role)
    return result


def consumer_origin():
    result = raw_origin()
    validate_common_origin(result)
    require(result["job"] == "platform-coverage")
    return result


def validate_origin(value, role):
    validate_common_origin(value)
    require(role in ROLES and value["job"] == ROLES[role])


def validate_common_origin(value):
    keys(value, {"repository", "repository_id", "run_id", "attempt", "job", "event", "pr_head", "step"})
    require(value["repository"] == REPOSITORY and value["repository_id"] == REPOSITORY_ID)
    for field in ("run_id", "attempt"):
        require(type(value[field]) is str and re.fullmatch(r"[1-9][0-9]{0,19}", value[field]) is not None)
    require(value["event"] in {"push", "pull_request", "workflow_dispatch"})
    require(value["step"] == "platform_counters")
    hex_value(value["pr_head"], 40)


def tool_identity(binary, role):
    binary = binary.resolve(strict=True)
    before = safe_file(binary, MAX_TOOL)
    environment = child_environment()
    environment.update(GOTOOLCHAIN="local", GOENV="off", GOWORK="off")
    result = subprocess.run([str(binary), "version"], env=environment, check=False,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
    require(result.returncode == 0 and not result.stderr)
    text = result.stdout.decode("ascii")
    match = re.fullmatch(r"go version (go[0-9]+\.[0-9]+\.[0-9]+) ([a-z0-9]+/[a-z0-9]+)\r?\n", text)
    require(match is not None and match[1] == GO_VERSION and match[2] in PLATFORMS[role])
    require(digest(safe_file(binary, MAX_TOOL)) == digest(before))
    return {"version": match[1], "platform": match[2], "sha256": digest(before), "bytes": len(before)}


def build_options():
    # No ambient tag/overlay selection may turn the fixed command into a subset.
    flags = os.environ.get("GOFLAGS", "")
    require(flags in ("", "-p=1"))
    require(os.environ.get("GOENV") == "off" and os.environ.get("GOWORK") == "off")
    require(os.environ.get("GOTOOLCHAIN") == "local")
    require(not os.environ.get("GOOS") and not os.environ.get("GOARCH"))
    cgo = os.environ.get("CGO_ENABLED", "default")
    require(cgo in ("default", "0", "1"))
    return {"go_flags": flags, "cgo_selector": cgo, "scope": "default-all-packages-no-tags"}


def profile(data, tracked):
    require(0 < len(data) <= MAX_PROFILE)
    rows = data.decode("utf-8").split("\n")
    if rows[-1] == "":
        rows.pop()
    rows = [row[:-1] if row.endswith("\r") else row for row in rows]
    require(len(rows) > 1 and rows[0] == "mode: atomic")
    blocks = {}
    for row in rows[1:]:
        key, statements, count = coverage_block(row, tracked)
        previous = blocks.get(key, (statements, 0))
        require(previous[0] == statements)
        blocks[key] = (statements, max(previous[1], count))
    require(0 < sum(value[0] for value in blocks.values()) <= 100_000_000)
    return blocks


def coverage_block(row, tracked):
    require(len(row.encode("utf-8")) <= 1 << 20)
    match = BLOCK.fullmatch(row)
    require(match is not None and match[1].startswith(MODULE))
    relative = match[1][len(MODULE):]
    require(relative in tracked and not relative.endswith("_test.go"))
    require(all(len(match[index]) <= 10 for index in range(2, 6)))
    require(len(match[6]) <= 8 and len(match[7]) <= 20)
    coordinates = tuple(int(match[index]) for index in range(2, 6))
    require(all(0 < value <= 0xFFFFFFFF for value in coordinates))
    require(coordinates[:2] <= coordinates[2:])
    statements, count = int(match[6]), int(match[7])
    require(statements <= 10_000_000 and count <= 0xFFFFFFFFFFFFFFFF)
    return (match[1], *coordinates), statements, count


def begin(source, expected, binary, role, output):
    require(role in {"linux", "macos", "windows"})
    before, _ = source_identity(source, expected)
    value = {"version": 1, "kind": "platform-counter-preflight", "role": role,
             "source": before, "toolchain": tool_identity(binary, role),
             "producer": origin(role), "build": build_options()}
    output = new_directory(output, source)
    write_new(output / "before.json", encoded(value))
    return value


def descriptor(value, metadata):
    return {"version": 1, "role": value["role"], "source": value["source"],
            "toolchain": value["toolchain"], "producer": value["producer"],
            "coverage": value["coverage"], "metadata_sha256": digest(metadata),
            "metadata_bytes": len(metadata)}


def publish_descriptor(value, step_output):
    data = json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False)
    require(len(data.encode()) <= 16384 and "\n" not in data)
    # Only GitHub's runner-owned step-output destination, supplied by the fixed
    # workflow, is written. Never retain any other environment value.
    with step_output.open("a", encoding="utf-8") as stream:
        stream.write("descriptor=" + data + "\n")


def retain(source, expected, binary, role, output, exit_code, step_output):
    require(type(exit_code) is int and 0 <= exit_code <= 255)
    before = strict_json(safe_file(output / "before.json", MAX_METADATA, True))
    keys(before, {"version", "kind", "role", "source", "toolchain", "producer", "build"})
    require(type(before["version"]) is int and before["version"] == 1)
    require(before["kind"] == "platform-counter-preflight" and before["role"] == role)
    current, tracked = source_identity(source, expected)
    require(current == before["source"] and tool_identity(binary, role) == before["toolchain"])
    require(origin(role) == before["producer"] and build_options() == before["build"])
    content = safe_file(output / "coverage.out", MAX_PROFILE, True)
    profile(content, tracked)
    require(source_identity(source, expected)[0] == current)
    value = dict(before, kind="platform-counters", source_before_after_equal=True,
                 coverage={"mode": "atomic", "bytes": len(content), "sha256": digest(content)},
                 test={"package_pattern": "./...", "coverpkg": "./...", "count": 1,
                       "package_parallelism": 1, "tags": [], "exit_code": exit_code},
                 production_qualified=False)
    metadata = encoded(value)
    bundle = new_directory(output / "bundle", source)
    write_new(bundle / "coverage.out", content)
    write_new(bundle / "platform-metadata.json", metadata)
    result = descriptor(value, metadata)
    publish_descriptor(result, step_output)
    return result


def safe_checks(value):
    require(type(value) is list and len(value) <= 512)
    result = []
    for check in value:
        require(type(check) is dict and set(check) in ({"name", "status", "required"}, {"name", "status", "required", "code"}))
        require(type(check["name"]) is str and TOKEN.fullmatch(check["name"]) is not None)
        require(type(check["status"]) is str and check["status"] in STATUSES and type(check["required"]) is bool)
        require("code" not in check or (type(check["code"]) is str and TOKEN.fullmatch(check["code"]) is not None))
        result.append(check)
    require(len({check["name"] for check in result}) == len(result))
    by_name = {check["name"]: check for check in result}
    required = {"format", "unit_tests", "go_vet", "go_toolchain", "source_revision",
                "source_identity_before_checks", "source_identity_after_checks"}
    require(all(name in by_name and by_name[name]["required"] and by_name[name]["status"] == "passed" for name in required))
    return result


def validate_baseline(value, expected_source, content):
    keys(value, {"version", "profile", "commit", "passed", "production_qualified", "checks", "source",
                 "original_report_sha256", "coverage_sha256", "retention_only", "coverage_semantics",
                 "go_version", "platform", "go_tool_sha256"})
    require(type(value["version"]) is int and value["version"] == 1 and value["profile"] == "source")
    require(type(value["passed"]) is bool and value["production_qualified"] is False and value["retention_only"] is True)
    require(value["commit"] == expected_source["commit"] and value["coverage_sha256"] == digest(content))
    keys(value["source"], {"commit", "tree", "manifest_sha256"})
    require(value["source"]["commit"] == expected_source["commit"] and value["source"]["tree"] == expected_source["tree"])
    hex_value(value["source"]["manifest_sha256"])
    hex_value(value["original_report_sha256"])
    hex_value(value["go_tool_sha256"])
    require(value["go_version"] == GO_VERSION and value["platform"] == "linux/amd64")
    require(value["coverage_semantics"] == "Original audit result; not recomputed or promoted by retention")
    checks = safe_checks(value["checks"])
    require(not value["passed"] or all(not item["required"] or item["status"] == "passed" for item in checks))
    return checks


def bind_baseline(source, expected, binary, bundle, step_output):
    current, tracked = source_identity(source, expected)
    content = safe_file(bundle / "coverage.out", MAX_PROFILE, True)
    metadata = safe_file(bundle / "metadata.json", MAX_METADATA, True)
    value = strict_json(metadata)
    validate_baseline(value, current, content)
    profile(content, tracked)
    tool = tool_identity(binary, "source")
    require(tool["sha256"] == value["go_tool_sha256"])
    require(source_identity(source, expected)[0] == current)
    record = {"version": 1, "role": "source", "source": current, "toolchain": tool,
              "producer": origin("source"),
              "coverage": {"mode": "atomic", "bytes": len(content), "sha256": digest(content)}}
    result = descriptor(record, metadata)
    publish_descriptor(result, step_output)
    return result


def validate_source(value):
    keys(value, {"commit", "tree", "tracked_clean", "locks"})
    hex_value(value["commit"], 40)
    hex_value(value["tree"], 40)
    require(value["tracked_clean"] is True)
    keys(value["locks"], {"go.mod", "go.sum"})
    for item in value["locks"].values():
        hex_value(item)


def validate_descriptor(value, role, current):
    keys(value, {"version", "role", "source", "toolchain", "producer", "coverage", "metadata_sha256", "metadata_bytes"})
    require(type(value["version"]) is int and value["version"] == 1 and value["role"] == role)
    validate_source(value["source"])
    require(value["source"] == current)
    keys(value["toolchain"], {"version", "platform", "sha256", "bytes"})
    tool = value["toolchain"]
    require(tool["version"] == GO_VERSION and tool["platform"] in PLATFORMS[role])
    hex_value(tool["sha256"])
    require(type(tool["bytes"]) is int and 0 < tool["bytes"] <= MAX_TOOL)
    validate_origin(value["producer"], role)
    keys(value["coverage"], {"mode", "bytes", "sha256"})
    require(value["coverage"]["mode"] == "atomic")
    require(type(value["coverage"]["bytes"]) is int and 0 < value["coverage"]["bytes"] <= MAX_PROFILE)
    hex_value(value["coverage"]["sha256"])
    hex_value(value["metadata_sha256"])
    require(type(value["metadata_bytes"]) is int and 0 < value["metadata_bytes"] <= MAX_METADATA)


def pins_from_needs(source, expected, needs, output):
    """The caller must authenticate needs as runner context, not artifact content."""
    keys(needs, set(ROLES.values()))
    current, _ = source_identity(source, expected)
    reference = consumer_origin()
    # The union job has its own key; only repository/run/attempt/event/head are
    # compared. Callers pass the fixed job identity below rather than impersonate
    # any producer. The metadata itself can never authorize a new expectation.
    entries, results = {}, {}
    for role, job in ROLES.items():
        row = needs[job]
        keys(row, {"result", "outputs"})
        require(type(row["result"]) is str and row["result"] in {"success", "failure", "cancelled", "skipped"})
        keys(row["outputs"], {"descriptor", "artifact_id", "artifact_sha256"})
        require(type(row["outputs"]["descriptor"]) is str and 0 < len(row["outputs"]["descriptor"].encode()) <= 16384)
        descriptor_value = strict_json(row["outputs"]["descriptor"].encode())
        validate_descriptor(descriptor_value, role, current)
        require(type(row["outputs"]["artifact_id"]) is str and re.fullmatch(r"[1-9][0-9]{0,19}", row["outputs"]["artifact_id"]) is not None)
        hex_value(row["outputs"]["artifact_sha256"])
        for field in ("repository", "repository_id", "run_id", "attempt", "event", "pr_head"):
            require(descriptor_value["producer"][field] == reference[field])
        entries[role] = {"descriptor": descriptor_value, "artifact_id": row["outputs"]["artifact_id"],
                         "archive_sha256": row["outputs"]["artifact_sha256"]}
        results[job] = row["result"]
    value = {"version": 1, "kind": "same-run-platform-coverage-pins", "source": current,
             "entries": entries, "original_job_results": results}
    require(len({entry["artifact_id"] for entry in entries.values()}) == len(ROLES))
    write_new(output, encoded(value))
    return value


def archive_file(directory):
    directory = directory.absolute()
    require(directory.resolve(strict=True) == directory)
    # The official transport may return its opaque archive as the fixed path
    # itself or as the sole file below that fixed destination. Neither permits
    # a scan, extra member or caller-selected artifact path.
    if directory.is_file():
        return safe_file(directory, MAX_ARCHIVE)
    require(directory.is_dir())
    members = list(directory.iterdir())
    require(len(members) == 1 and members[0].is_file() and not members[0].is_symlink())
    return safe_file(members[0], MAX_ARCHIVE)


def archive_members(data, role, expected):
    """Normalize ZIP parser diagnostics; local headers can contain private text."""
    try:
        return checked_archive(data, role, expected)
    except (OSError, ValueError, RuntimeError, EOFError, zipfile.BadZipFile, zlib.error):
        raise InvalidEvidence("invalid bounded platform coverage archive") from None


def checked_archive(data, role, expected):
    require(digest(data) == expected["archive_sha256"])
    metadata_name = "metadata.json" if role == "source" else "platform-metadata.json"
    allowed = {"coverage.out": (MAX_PROFILE, expected["descriptor"]["coverage"]["bytes"], expected["descriptor"]["coverage"]["sha256"]),
               metadata_name: (MAX_METADATA, expected["descriptor"]["metadata_bytes"], expected["descriptor"]["metadata_sha256"])}
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        entries = archive.infolist()
        require(len(entries) == 2 and {entry.filename for entry in entries} == set(allowed))
        result = {}
        for entry in entries:
            require(not entry.is_dir() and not entry.flag_bits & 1)
            require(stat.S_IFMT(entry.external_attr >> 16) in (0, stat.S_IFREG))
            limit, size, expected_hash = allowed[entry.filename]
            require(entry.file_size == size and 0 < size <= limit)
            with archive.open(entry) as stream:
                content = stream.read(limit + 1)
            require(len(content) == size and digest(content) == expected_hash)
            result[entry.filename] = content
    return result["coverage.out"], strict_json(result[metadata_name])


def validate_platform(value, expected, content):
    keys(value, {"version", "kind", "role", "source", "toolchain", "producer", "build",
                 "source_before_after_equal", "coverage", "test", "production_qualified"})
    require(value["kind"] == "platform-counters" and value["version"] == 1 and type(value["version"]) is int)
    require(value["source_before_after_equal"] is True and value["production_qualified"] is False)
    for field in ("role", "source", "toolchain", "producer", "coverage"):
        require(value[field] == expected[field])
    require(value["coverage"] == {"mode": "atomic", "bytes": len(content), "sha256": digest(content)})
    require(value["test"] == {"package_pattern": "./...", "coverpkg": "./...", "count": 1,
                             "package_parallelism": 1, "tags": [], "exit_code": 0})
    require(all(type(value["test"][name]) is int for name in ("count", "exit_code", "package_parallelism")))
    keys(value["build"], {"go_flags", "cgo_selector", "scope"})
    require(value["build"]["go_flags"] in ("", "-p=1") and value["build"]["cgo_selector"] in ("default", "0", "1"))
    require(value["build"]["scope"] == "default-all-packages-no-tags")


def merged_profiles(profiles):
    merged = {}
    for blocks in profiles:
        for key, (statements, count) in blocks.items():
            previous = merged.get(key, (statements, 0))
            require(previous[0] == statements)
            merged[key] = (statements, max(previous[1], count))
    total = sum(row[0] for row in merged.values())
    covered = sum(row[0] for row in merged.values() if row[1] > 0)
    require(0 < total <= 100_000_000)
    rows = ["mode: atomic"]
    gaps = []
    for key, (statements, count) in sorted(merged.items()):
        name, a, b, c, d = key
        rows.append(f"{name}:{a}.{b},{c}.{d} {statements} {count}")
        if count == 0 and statements > 0:
            gaps.append({"path": name[len(MODULE):], "start": [a, b], "end": [c, d], "statements": statements})
    content = ("\n".join(rows) + "\n").encode()
    require(len(content) <= MAX_PROFILE)
    return content, {"covered_statements": covered, "total_statements": total,
                     "statement_percent": 100 * covered / total, "threshold_percent": 100,
                     "coverage_gate_passed": covered == total, "coordinates": len(merged)}, gaps


def union(source, expected, artifacts, pins_file, pins_sha, output):
    hex_value(pins_sha)
    data = safe_file(pins_file, MAX_METADATA, True)
    require(digest(data) == pins_sha)
    pins = strict_json(data)
    before, tracked = source_identity(source, expected)
    validate_pins(pins, before)
    profiles, original, original_metadata = [], None, None
    for role in ROLES:
        entry = pins["entries"][role]
        keys(entry, {"descriptor", "artifact_id", "archive_sha256"})
        validate_descriptor(entry["descriptor"], role, before)
        content, metadata = archive_members(archive_file(artifacts / role), role, entry)
        if role == "source":
            checks = validate_baseline(metadata, before, content)
            require(metadata["go_tool_sha256"] == entry["descriptor"]["toolchain"]["sha256"])
            original = {"passed": metadata["passed"], "checks": checks,
                        "original_report_sha256": metadata["original_report_sha256"], "production_qualified": False}
            original_metadata = metadata
        else:
            validate_platform(metadata, entry["descriptor"], content)
        profiles.append(profile(content, tracked))
    content, coverage, gaps = merged_profiles(profiles)
    require(source_identity(source, expected)[0] == before)
    output = new_directory(output, source)
    value = {"version": 1, "kind": "same-revision-platform-source-coverage", "source": before,
             "pins_sha256": pins_sha, "producers": pins["entries"], "coverage": coverage,
             "coverage_sha256": digest(content), "original_source_report_unchanged": original,
             "original_source_metadata_unchanged": original_metadata,
             "original_job_results_unchanged": pins["original_job_results"],
             "denominator": "all retained Source and Linux/macOS/Windows coordinate sets; OS-only zeros included",
             "global_gate_passed": False, "production_qualified": False, "source_report_promoted": False}
    write_new(output / "coverage.out", content)
    write_new(output / "platform-union.json", encoded(value))
    write_new(output / "gaps.json", encoded({"coverage": coverage, "gaps": gaps}))
    print(json.dumps({"coverage": coverage, "global_gate_passed": False, "source_report_promoted": False}, sort_keys=True))
    return 0 if coverage["coverage_gate_passed"] else 1


def validate_pins(pins, source):
    keys(pins, {"version", "kind", "source", "entries", "original_job_results"})
    require(type(pins["version"]) is int and pins["version"] == 1 and pins["kind"] == "same-run-platform-coverage-pins")
    validate_source(pins["source"])
    require(pins["source"] == source)
    keys(pins["entries"], set(ROLES))
    keys(pins["original_job_results"], set(ROLES.values()))
    reference = None
    artifact_ids = set()
    for role in ROLES:
        entry = pins["entries"][role]
        keys(entry, {"descriptor", "artifact_id", "archive_sha256"})
        require(type(entry["artifact_id"]) is str and re.fullmatch(r"[1-9][0-9]{0,19}", entry["artifact_id"]) is not None)
        require(entry["artifact_id"] not in artifact_ids)
        artifact_ids.add(entry["artifact_id"])
        hex_value(entry["archive_sha256"])
        validate_descriptor(entry["descriptor"], role, source)
        producer = entry["descriptor"]["producer"]
        common = {name: producer[name] for name in ("repository", "repository_id", "run_id", "attempt", "event", "pr_head")}
        if reference is None:
            reference = common
        require(common == reference)
        result = pins["original_job_results"][ROLES[role]]
        require(type(result) is str and result in {"success", "failure", "cancelled", "skipped"})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("begin", "retain", "baseline", "pins", "union"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--expected-commit", required=True)
    parser.add_argument("--go", type=Path)
    parser.add_argument("--platform", choices=("linux", "macos", "windows"))
    parser.add_argument("--output", type=Path)
    parser.add_argument("--test-exit", type=int)
    parser.add_argument("--step-output", type=Path)
    parser.add_argument("--bundle", type=Path)
    parser.add_argument("--needs", type=Path)
    parser.add_argument("--artifacts", type=Path)
    parser.add_argument("--pins", type=Path)
    parser.add_argument("--expected-pins-sha256")
    args = parser.parse_args()
    try:
        if args.action == "begin":
            begin(args.source, args.expected_commit, args.go, args.platform, args.output)
        elif args.action == "retain":
            retain(args.source, args.expected_commit, args.go, args.platform, args.output, args.test_exit, args.step_output)
        elif args.action == "baseline":
            bind_baseline(args.source, args.expected_commit, args.go, args.bundle, args.step_output)
        elif args.action == "pins":
            value = pins_from_needs(args.source, args.expected_commit, strict_json(safe_file(args.needs, MAX_METADATA, True)), args.output)
            print("pins_sha256=" + digest(encoded(value)))
        else:
            return union(args.source, args.expected_commit, args.artifacts, args.pins, args.expected_pins_sha256, args.output)
    except (OSError, ValueError, UnicodeError, RecursionError, subprocess.SubprocessError, TypeError, AttributeError):
        print("Bounded same-run platform evidence refused.")
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
