"""Bounded SDK conformance retention and offline, externally pinned coverage join.

The caller authenticates original producer/artifact/source receipts and supplies
their pins' digest separately. A digest binds bytes; it does NOT authenticate
their author. Generating pins from the untrusted inputs is not authentication.
Input metadata is never its own authority. This script does not fetch
artifacts, grant rights, execute tests, or promote an original Source report. A
coverage-only result is NOT acceptance by the CI/global tagged-integration gate.
Matching selected Go executable/version/platform and locks is recorded tool
identity, not proof of the entire OS, GOROOT, publisher or transport trust chain.
"""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import stat
import subprocess

# Reuse only the reviewed local file/JSON/source fences, not Source's report schema.
SPEC = importlib.util.spec_from_file_location(
    "cq_source_retention", Path(__file__).with_name("retain_audit_evidence.py"))
assert SPEC is not None and SPEC.loader is not None
BASE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(BASE)
InvalidEvidence = BASE.InvalidEvidence
require = BASE.require

MODULE = BASE.MODULE
SDK = MODULE + "sdk/go/"
PACKAGE = "github.com/maldous/CircularQuay/sdk/go"
TEST = "TestOfficialMCPClientSessionAgainstSyntheticCq"
STEP = "official_go_mcp_conformance"
REPOSITORY = "maldous/CircularQuay"
MAX_METADATA = 1 << 20
MAX_BINARY = 128 << 20
PLATFORM = "linux/amd64"
GO_VERSION = "go1.27.4"


def digest(content: bytes) -> str:
    return hashlib.sha256(content).hexdigest()


def encoded(value: object) -> bytes:
    return (json.dumps(value, sort_keys=True, indent=2, allow_nan=False) + "\n").encode()


def strict_json(content: bytes) -> dict:
    value = json.loads(content, object_pairs_hook=BASE.unique_object,
                      parse_constant=BASE.reject_constant)
    require(isinstance(value, dict))
    return value


def keys(value: object, expected: set[str]) -> None:
    require(type(value) is dict and set(value) == expected)


def hex_value(value: object, length: int = 64) -> None:
    require(isinstance(value, str) and re.fullmatch(r"[0-9a-f]{" + str(length) + "}", value) is not None)


def identity(info: os.stat_result) -> tuple:
    return (info.st_dev, info.st_ino, info.st_mode, info.st_uid, info.st_gid,
            info.st_nlink, info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def regular_digest(path: Path) -> tuple[str, int]:
    """Hash a bounded no-alias regular source/tool/binary without exporting bytes."""
    require(path.is_absolute() and path.resolve(strict=True) == path)
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(descriptor, "rb") as stream:
        before = os.fstat(stream.fileno())
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1)
        require(not before.st_mode & 0o022 and 0 < before.st_size <= MAX_BINARY)
        require(before.st_uid in {0, os.geteuid()})
        value = hashlib.sha256()
        count = 0
        for block in iter(lambda: stream.read(65536), b""):
            count += len(block)
            require(count <= MAX_BINARY)
            value.update(block)
        require(count == before.st_size)
        require(identity(before) == identity(os.fstat(stream.fileno())))
        require(identity(before) == identity(path.stat(follow_symlinks=False)))
    return value.hexdigest(), count


def private_read(path: Path, maximum: int) -> bytes:
    BASE.private_directory(path.parent)
    return BASE.bounded_file(path, maximum)


def source_identity(source: Path, commit: str) -> tuple[dict, set[str]]:
    tracked = BASE.tracked_source(source, commit)
    tree = BASE.git_output(source, "rev-parse", "HEAD^{tree}").decode().strip()
    hex_value(tree, 40)
    # An ignored/untracked Go file in the selected package can change its build.
    for selector in ([], ["--ignored"]):
        extra = BASE.git_output(source, "ls-files", "-z", "--others", *selector,
                                "--exclude-standard", "--", "sdk/go/*.go")
        require(not extra)
    locks = {name: regular_digest(source / name)[0] for name in ("go.mod", "go.sum")}
    return {"commit": commit, "tree": tree, "tracked_clean": True, "locks": locks}, tracked


def go_identity(binary: Path) -> dict:
    require(bool(binary.stat().st_mode & stat.S_IXUSR))
    before = regular_digest(binary)
    environment = {"PATH": os.defpath, "GOTOOLCHAIN": "local", "GOENV": "off"}
    result = subprocess.run([str(binary), "version"], env=environment, check=False,
                            capture_output=True, timeout=30)
    require(result.returncode == 0 and result.stderr == b"")
    require(result.stdout == f"go version {GO_VERSION} {PLATFORM}\n".encode())
    require(regular_digest(binary) == before)
    return {"version": GO_VERSION, "platform": PLATFORM, "sha256": before[0]}


def profile(content: bytes, tracked: set[str], sdk_only: bool = False) -> dict:
    require(0 < len(content) <= BASE.MAX_COVERAGE)
    BASE.validate_coverage(content, tracked)
    rows = content.decode().splitlines()
    require(rows[0] == "mode: atomic")
    blocks = {}
    for row in rows[1:]:
        match = BASE.BLOCK.fullmatch(row)
        assert match is not None
        name = match[1]
        require(not name.endswith("_test.go"))
        require(not sdk_only or name.startswith(SDK))
        coordinate = (name, *(int(match[i]) for i in range(2, 6)))
        statements, count = int(match[6]), int(match[7])
        previous = blocks.get(coordinate, (statements, 0))
        require(previous[0] == statements)
        blocks[coordinate] = (statements, max(previous[1], count))
    require(0 < sum(value[0] for value in blocks.values()) <= 100_000_000)
    return blocks


def producer(value: dict) -> None:
    keys(value, {"repository", "pr_head", "run_id", "attempt", "job", "matrix", "step"})
    require(value["repository"] == REPOSITORY and value["step"] == STEP)
    hex_value(value["pr_head"], 40)
    require(isinstance(value["run_id"], str) and re.fullmatch(r"[1-9][0-9]{0,29}", value["run_id"]) is not None)
    require(type(value["attempt"]) is int and 1 <= value["attempt"] <= 1_000_000)
    require(value["job"] == "typescript" and value["matrix"] == "ubuntu-24.04/node24.22.2")


def write_bundle(output: Path, files: dict[str, bytes], forbidden: tuple[Path, ...]) -> None:
    output = output.absolute()
    BASE.private_directory(output.parent)
    require(all(not output.is_relative_to(path.absolute()) for path in forbidden))
    output.mkdir(mode=0o700, exist_ok=False)
    created = []
    try:
        for name, content in files.items():
            descriptor = os.open(output / name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
            created.append(name)
            with os.fdopen(descriptor, "wb") as stream:
                stream.write(content)
    except BaseException:
        # Only files created by this invocation can exist in this new directory.
        for name in created:
            (output / name).unlink(missing_ok=True)
        output.rmdir()
        raise


def measurement(source: Path, commit: str, go_binary: Path, cq_binary: Path, origin: dict) -> dict:
    producer(origin)
    before, tracked = source_identity(source, commit)
    toolchain = go_identity(go_binary)
    require(bool(cq_binary.stat().st_mode & stat.S_IXUSR))
    binary_sha, binary_bytes = regular_digest(cq_binary)
    require(source_identity(source, commit) == (before, tracked))
    return {"version": 1, "kind": "sdk-go-mcp-preflight", "source": before,
            "toolchain": toolchain, "binary": {"sha256": binary_sha, "bytes": binary_bytes}, "producer": origin}


def retain(source: Path, coverage_path: Path, output: Path, commit: str,
           go_binary: Path, cq_binary: Path, origin: dict, exit_code: int, preflight: Path) -> dict:
    require(type(exit_code) is int and 0 <= exit_code <= 255)
    preflight_bytes = private_read(preflight, MAX_METADATA)
    before = strict_json(preflight_bytes)
    keys(before, {"version", "kind", "source", "toolchain", "binary", "producer"})
    require(before == measurement(source, commit, go_binary, cq_binary, origin))
    require(before["source"]["tracked_clean"] is True)
    require(type(before["version"]) is int)
    validate_binary(before["binary"])
    producer(before["producer"])
    _, tracked = source_identity(source, commit)
    content = private_read(coverage_path, BASE.MAX_COVERAGE)
    profile(content, tracked, sdk_only=True)
    require(before == measurement(source, commit, go_binary, cq_binary, origin))
    metadata = dict(before, kind="sdk-go-mcp", source_before_after_equal=True,
                    preflight_sha256=digest(preflight_bytes),
                    # count is the fixed go test -count flag, not fabricated
                    # telemetry claiming an observed number of test results.
                    test={"package": PACKAGE, "name": TEST, "tags": "integration",
                                             "count": 1, "exit_code": exit_code},
                    coverage={"mode": "atomic", "bytes": len(content), "sha256": digest(content)},
                    production_qualified=False)
    data = encoded(metadata)
    require(len(data) <= MAX_METADATA)
    write_bundle(output, {"coverage.out": content, "sdk-metadata.json": data},
                 (source, coverage_path.parent, preflight.parent))
    return {"retained_files": 2, "conformance_exit_code": exit_code, "coverage_sha256": digest(content)}


def read_pins(path: Path, expected: str) -> dict:
    hex_value(expected)
    content = private_read(path, MAX_METADATA)
    require(digest(content) == expected)
    value = strict_json(content)
    keys(value, {"version", "baseline_commit", "sdk_commit", "tree", "producer", "toolchain",
                 "binary", "locks", "artifacts"})
    require(type(value["version"]) is int and value["version"] == 1)
    for name in ("baseline_commit", "sdk_commit", "tree"):
        hex_value(value[name], 40)
    producer(value["producer"])
    validate_toolchain(value["toolchain"])
    validate_binary(value["binary"])
    validate_locks(value["locks"])
    keys(value["artifacts"], {"baseline_coverage", "baseline_metadata", "sdk_coverage", "sdk_metadata"})
    for value_hash in value["artifacts"].values():
        hex_value(value_hash)
    return value


def validate_toolchain(value: dict) -> None:
    keys(value, {"version", "platform", "sha256"})
    require(value["version"] == GO_VERSION and value["platform"] == PLATFORM)
    hex_value(value["sha256"])


def validate_binary(value: dict) -> None:
    keys(value, {"sha256", "bytes"})
    hex_value(value["sha256"])
    require(type(value["bytes"]) is int and 0 < value["bytes"] <= MAX_BINARY)


def validate_locks(value: dict) -> None:
    keys(value, {"go.mod", "go.sum"})
    for value_hash in value.values():
        hex_value(value_hash)


def validate_sdk(metadata: dict, pins: dict, content: bytes) -> None:
    keys(metadata, {"version", "kind", "source", "toolchain", "binary", "producer", "test",
                    "coverage", "production_qualified", "source_before_after_equal", "preflight_sha256"})
    require(type(metadata["version"]) is int and metadata["version"] == 1)
    require(metadata["kind"] == "sdk-go-mcp" and metadata["production_qualified"] is False)
    require(metadata["source_before_after_equal"] is True)
    hex_value(metadata["preflight_sha256"])
    keys(metadata["source"], {"commit", "tree", "tracked_clean", "locks"})
    require(metadata["source"]["tracked_clean"] is True)
    validate_locks(metadata["source"]["locks"])
    validate_toolchain(metadata["toolchain"])
    validate_binary(metadata["binary"])
    producer(metadata["producer"])
    expected_source = {"commit": pins["sdk_commit"], "tree": pins["tree"],
                       "tracked_clean": True, "locks": pins["locks"]}
    require(metadata["source"] == expected_source)
    for name in ("toolchain", "binary", "producer"):
        require(metadata[name] == pins[name])
    require(metadata["test"] == {"package": PACKAGE, "name": TEST, "tags": "integration",
                                 "count": 1, "exit_code": 0})
    # bool is not an exit/count integer, even though Python equality treats it as one.
    require(type(metadata["test"]["count"]) is int and type(metadata["test"]["exit_code"]) is int)
    require(metadata["coverage"] == {"mode": "atomic", "bytes": len(content), "sha256": digest(content)})
    require(type(metadata["coverage"]["bytes"]) is int)


def required_baseline_checks(by_name: dict) -> None:
    for name in ("format", "unit_tests", "go_vet", "go_toolchain", "source_revision",
                 "source_identity_before_checks", "source_identity_after_checks"):
        require(name in by_name and by_name[name]["required"] and by_name[name]["status"] == "passed")


def baseline_checks(checks: object, passed: bool) -> list[dict]:
    require(isinstance(checks, list) and len(checks) <= 512)
    selected = [BASE.safe_check(check) for check in checks]
    require(selected == checks)
    require(len({check["name"] for check in selected}) == len(selected))
    by_name = {check["name"]: check for check in selected}
    required_baseline_checks(by_name)
    if passed:
        require(all(not check["required"] or check["status"] == "passed" for check in selected))
    return selected


def validate_baseline(metadata: dict, pins: dict) -> list[dict]:
    keys(metadata, {"version", "profile", "commit", "passed", "production_qualified", "checks",
                    "source", "original_report_sha256", "coverage_sha256", "retention_only",
                    "coverage_semantics", "go_version", "platform", "go_tool_sha256"})
    require(metadata.get("version") == 1 and type(metadata["version"]) is int)
    require(metadata.get("profile") == "source" and metadata.get("retention_only") is True)
    require(type(metadata.get("passed")) is bool and metadata.get("production_qualified") is False)
    require(metadata.get("commit") == pins["baseline_commit"])
    source = metadata.get("source")
    keys(source, {"commit", "tree", "manifest_sha256"})
    require(source.get("commit") == pins["baseline_commit"])
    require(source.get("tree") == pins["tree"])
    hex_value(source.get("manifest_sha256"))
    hex_value(metadata.get("original_report_sha256"))
    require(metadata.get("go_version") == pins["toolchain"]["version"])
    require(metadata.get("platform") == pins["toolchain"]["platform"])
    require(metadata.get("go_tool_sha256") == pins["toolchain"]["sha256"])
    require(metadata["coverage_semantics"] == "Original audit result; not recomputed or promoted by retention")
    return baseline_checks(metadata["checks"], metadata["passed"])


def joined_profile(baseline: dict, supplement: dict) -> tuple[bytes, dict]:
    sdk_blocks = {key for key in baseline if key[0].startswith(SDK)}
    require(sdk_blocks and set(supplement) == sdk_blocks)
    result = dict(baseline)
    for key, (statements, count) in supplement.items():
        require(baseline[key][0] == statements)
        result[key] = (statements, max(baseline[key][1], count))
    total = sum(value[0] for value in baseline.values())
    require(total == sum(value[0] for value in result.values()))
    covered = sum(statements for statements, count in result.values() if count > 0)
    rows = ["mode: atomic"]
    for key, (statements, count) in sorted(result.items()):
        name, start_line, start_column, end_line, end_column = key
        rows.append(f"{name}:{start_line}.{start_column},{end_line}.{end_column} {statements} {count}")
    content = ("\n".join(rows) + "\n").encode()
    require(len(content) <= BASE.MAX_COVERAGE)
    return content, {"covered_statements": covered, "total_statements": total,
                     "statement_percent": 100 * covered / total, "threshold_percent": 100,
                     "coverage_gate_passed": covered == total}


def join(source: Path, paths: dict[str, Path], output: Path, pins_path: Path, pins_sha: str) -> dict:
    pins = read_pins(pins_path, pins_sha)
    before, tracked = source_identity(source, pins["baseline_commit"])
    require(before["tree"] == pins["tree"] and before["locks"] == pins["locks"])
    contents = {name: private_read(path, BASE.MAX_COVERAGE if name.endswith("coverage") else MAX_METADATA)
                for name, path in paths.items()}
    require(set(contents) == set(pins["artifacts"]))
    require(all(digest(contents[name]) == expected for name, expected in pins["artifacts"].items()))
    baseline = strict_json(contents["baseline_metadata"])
    sdk = strict_json(contents["sdk_metadata"])
    baseline_checks = validate_baseline(baseline, pins)
    require(baseline.get("coverage_sha256") == digest(contents["baseline_coverage"]))
    validate_sdk(sdk, pins, contents["sdk_coverage"])
    original = profile(contents["baseline_coverage"], tracked)
    supplemental = profile(contents["sdk_coverage"], tracked, sdk_only=True)
    coverage, summary = joined_profile(original, supplemental)
    require(source_identity(source, pins["baseline_commit"]) == (before, tracked))
    result = {"version": 1, "kind": "source-plus-sdk-go-mcp-coverage", "pins_sha256": pins_sha,
              "source_tree": pins["tree"], "baseline_commit": pins["baseline_commit"],
              "sdk_commit": pins["sdk_commit"], "producer": pins["producer"],
              "input_sha256": pins["artifacts"], "coverage": summary,
              "composite_coverage_sha256": digest(coverage), "baseline_passed": baseline["passed"],
              "baseline_checks_unchanged": baseline_checks, "global_gate_passed": False,
              "production_qualified": False, "source_report_promoted": False,
              "limitation": "Coverage-only composition; requires separate reviewed CI gate intake. SDK conformance is not whole-product tagged integration."}
    metadata_bytes = encoded(result)
    require(len(metadata_bytes) <= MAX_METADATA)
    write_bundle(output, {"coverage.out": coverage, "composite.json": metadata_bytes},
                 (source, *(path.parent for path in paths.values()), pins_path.parent))
    return result


def arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    for action in ("begin", "retain"):
        command = sub.add_parser(action)
        for name in ("source", "output", "go_binary", "cq_binary"):
            command.add_argument("--" + name.replace("_", "-"), type=Path, required=True)
        for name in ("expected_commit", "pr_head", "run_id", "job_key", "matrix"):
            command.add_argument("--" + name.replace("_", "-"), required=True)
        command.add_argument("--attempt", type=int, required=True)
        if action == "retain":
            command.add_argument("--coverage", type=Path, required=True)
            command.add_argument("--preflight", type=Path, required=True)
            command.add_argument("--exit-code", type=int, required=True)
    join_parser = sub.add_parser("join")
    for name in ("source", "baseline_coverage", "baseline_metadata", "sdk_coverage", "sdk_metadata", "output", "pins"):
        join_parser.add_argument("--" + name.replace("_", "-"), type=Path, required=True)
    join_parser.add_argument("--expected-pins-sha256", required=True)
    return parser.parse_args()


def main() -> int:
    args = arguments()
    try:
        if args.action in {"begin", "retain"}:
            origin = {"repository": REPOSITORY, "pr_head": args.pr_head, "run_id": args.run_id,
                      "attempt": args.attempt, "job": args.job_key, "matrix": args.matrix, "step": STEP}
            if args.action == "begin":
                result = measurement(args.source, args.expected_commit, args.go_binary, args.cq_binary, origin)
                write_bundle(args.output, {"preflight.json": encoded(result)}, (args.source,))
            else:
                result = retain(args.source, args.coverage, args.output, args.expected_commit,
                                args.go_binary, args.cq_binary, origin, args.exit_code, args.preflight)
        else:
            paths = {name: getattr(args, name) for name in
                     ("baseline_coverage", "baseline_metadata", "sdk_coverage", "sdk_metadata")}
            result = join(args.source, paths, args.output, args.pins, args.expected_pins_sha256)
        print(json.dumps({"action": args.action, "evidence_written": True,
                          "global_gate_passed": False}, sort_keys=True))
        return 1 if args.action == "join" and not result["coverage"]["coverage_gate_passed"] else 0
    except (OSError, ValueError, UnicodeError, RecursionError, subprocess.SubprocessError):
        print("Bounded Go-MCP evidence refused.")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
