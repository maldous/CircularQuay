"""Admit one authenticated current Source union into the same-checkout Sonar scan.

Only the already-declared Source workflow may supply the counter archive. An
Actions API digest authenticates that artifact's bytes, not arbitrary JSON's
claims. Trust in the reviewed same-revision Source workflow and its existing
producer validators remains explicit. Original Source and Linux outcomes are
retained, never promoted; this module neither runs tests nor starts a scan.
"""
from __future__ import annotations

import argparse
from datetime import datetime
import hashlib
import importlib.util
import io
import json
import math
import os
from pathlib import Path
import re
import stat
import time
import urllib.error
import urllib.request
import zipfile

SPEC = importlib.util.spec_from_file_location("cq_platform_intake_contract", Path(__file__).with_name("platform_coverage.py"))
assert SPEC is not None and SPEC.loader is not None
PC = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PC)
REPOSITORY = "maldous/CircularQuay"
REPOSITORY_ID = 1444116753
PREFIX = "/repos/" + REPOSITORY
MAX_JSON = 1 << 20
MAX_ARCHIVE = 32 << 20
MAX_CALLS = 128
JOBS = {"linux": "Linux synthetic tests", "macos": "macOS synthetic tests",
        "windows": "Windows synthetic tests", "source": "Source assurance gate"}
UNION_JOB = "Authoritative complete Source qualification"


class Refused(ValueError):
    """Only fixed public stage labels, never private diagnostics."""


def require(value, stage="source-union-contract-refused"):
    if not value:
        raise Refused(stage)


def number(value):
    require(type(value) is str and re.fullmatch(r"[1-9][0-9]{0,19}", value) is not None)
    return value


def sha(value):
    PC.hex_value(value)
    return value


def stamp(value):
    require(type(value) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", value) is not None)
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise Refused("source-union-api-redirect-refused")


class GitHubRead:
    """Fixed same-repository HTTPS GET only; token never follows a redirect."""
    def __init__(self, token, deadline):
        require(type(token) is str and 0 < len(token) <= 4096 and not any(c.isspace() for c in token))
        self.token = token
        self.deadline = deadline
        self.calls = 0
        self.opener = urllib.request.build_opener(NoRedirect())

    def __call__(self, path):
        require(type(path) is str and path.startswith(PREFIX + "/actions/") and ".." not in path and "#" not in path)
        self.calls += 1
        remaining = self.deadline - time.monotonic()
        require(self.calls <= MAX_CALLS and remaining > 0, "source-union-api-bound-refused")
        request = urllib.request.Request("https://api.github.com" + path, method="GET", headers={
            "Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28",
            "Authorization": "Bearer " + self.token, "User-Agent": "cq-current-source-union"})
        with self.opener.open(request, timeout=min(10, remaining)) as response:
            require(response.status == 200, "source-union-api-response-refused")
            raw = response.read(MAX_JSON + 1)
        require(0 < len(raw) <= MAX_JSON, "source-union-api-bound-refused")
        return PC.strict_json(raw)


def run_matches(run, head, pr, workflow):
    require(type(run) is dict and run.get("repository", {}).get("id") == REPOSITORY_ID)
    require(run.get("repository", {}).get("full_name") == REPOSITORY)
    require(run.get("head_repository", {}).get("id") == REPOSITORY_ID)
    require(run.get("event") == "pull_request" and run.get("head_sha") == head)
    require(run.get("path") == ".github/workflows/" + workflow + ".yml")
    require(type(run.get("run_attempt")) is int and run["run_attempt"] == 1)
    require(type(run.get("id")) is int and 0 < run["id"] < 10**20)
    require(type(run.get("pull_requests")) is list and len(run["pull_requests"]) == 1)
    selected = run["pull_requests"][0]
    require(selected.get("number") == int(pr) and selected.get("head", {}).get("sha") == head)
    return {"id": str(run["id"]), "attempt": 1, "created_at": run["created_at"]}


def paged(api, path, collection, maximum=500):
    first = api(path + "?per_page=100&page=1")
    total = first.get("total_count")
    require(type(total) is int and 0 <= total <= maximum)
    items = first.get(collection)
    require(type(items) is list and len(items) == min(100, total))
    for page in range(2, math.ceil(total / 100) + 1):
        result = api(path + "?per_page=100&page=" + str(page))
        require(result.get("total_count") == total and type(result.get(collection)) is list)
        expected = min(100, total - (page - 1) * 100)
        require(len(result[collection]) == expected)
        items.extend(result[collection])
    ids = [item.get("id") for item in items]
    require(all(type(item) is int and 0 < item < 10**20 for item in ids) and len(ids) == len(set(ids)))
    return items


def artifact(row, run_id):
    require(type(row) is dict and row.get("expired") is False)
    number(str(row.get("id")))
    require(type(row.get("size_in_bytes")) is int and 0 < row["size_in_bytes"] <= MAX_ARCHIVE)
    digest = row.get("digest")
    require(type(digest) is str and re.fullmatch(r"sha256:[0-9a-f]{64}", digest) is not None)
    require(row.get("workflow_run", {}).get("id") == int(run_id))
    return {"id": str(row["id"]), "sha256": digest[7:], "bytes": row["size_in_bytes"], "name": row["name"]}


def discover(api, source, head, pr, sonar_run, sleep=time.sleep):
    PC.validate_source(source)
    PC.hex_value(head, 40)
    number(pr)
    sonar = run_matches(api(PREFIX + "/actions/runs/" + number(sonar_run)), head, pr, "sonar")
    for observation in range(40):
        data = api(PREFIX + "/actions/workflows/source.yml/runs?event=pull_request&head_sha=" + head + "&per_page=100")
        require(type(data.get("total_count")) is int and 0 <= data["total_count"] <= 100)
        require(type(data.get("workflow_runs")) is list and len(data["workflow_runs"]) == data["total_count"])
        candidates = []
        for run in data["workflow_runs"]:
            candidate = run_matches(run, head, pr, "source")
            if abs((stamp(candidate["created_at"]) - stamp(sonar["created_at"])).total_seconds()) <= 600:
                candidates.append((run, candidate))
        require(len(candidates) <= 1, "source-union-original-ambiguous")
        if candidates:
            run, identity = candidates[0]
            status = run.get("status")
            require(status in {"queued", "in_progress", "completed"})
            if status == "completed":
                require(run.get("conclusion") in {"success", "failure"})
            run_id = identity["id"]
            archives = paged(api, PREFIX + "/actions/runs/" + run_id + "/artifacts", "artifacts", 100)
            selected = [row for row in archives if row.get("name") == "platform-go-union-" + source["commit"]]
            require(len(selected) <= 1, "source-union-original-ambiguous")
            # Unrelated cross-build jobs do not invalidate already completed,
            # authenticated counter producers. This never promotes Source's
            # overall result or treats a pending producer as successful.
            if selected:
                outer = artifact(selected[0], run_id)
                jobs = paged(api, PREFIX + "/actions/runs/" + run_id + "/jobs", "jobs")
                original = {}
                for name in [*JOBS.values(), UNION_JOB]:
                    matches = [job for job in jobs if job.get("name") == name]
                    require(len(matches) <= 1)
                    if not matches or matches[0].get("status") != "completed":
                        continue
                    require(matches[0].get("conclusion") in {"success", "failure"})
                    original[name] = matches[0]["conclusion"]
                if len(original) == len(JOBS) + 1:
                    return {"version": 1, "repository": REPOSITORY, "repository_id": REPOSITORY_ID,
                            "source": source, "head": head, "pr": pr, "sonar": sonar, "run": identity,
                            "artifact": outer, "original_jobs": original}
            require(status != "completed", "source-union-artifact-or-producer-unavailable")
        if observation < 39:
            sleep(20)
    raise Refused("source-union-original-not-terminal")


def checked_union(raw, binding, tracked, api):
    require(0 < len(raw) <= MAX_ARCHIVE and PC.digest(raw) == binding["artifact"]["sha256"])
    require(len(raw) == binding["artifact"]["bytes"])
    limits = {"coverage.out": PC.MAX_PROFILE, "platform-union.json": MAX_JSON, "gaps.json": 4 << 20}
    content = {}
    with zipfile.ZipFile(io.BytesIO(raw)) as archive:
        entries = archive.infolist()
        require(len(entries) == 3 and {entry.filename for entry in entries} == set(limits))
        for entry in entries:
            require(not entry.is_dir() and not entry.flag_bits & 1)
            require(stat.S_IFMT(entry.external_attr >> 16) in (0, stat.S_IFREG))
            require(0 < entry.file_size <= limits[entry.filename])
            with archive.open(entry) as stream:
                data = stream.read(limits[entry.filename] + 1)
            require(len(data) == entry.file_size)
            content[entry.filename] = data
    value = PC.strict_json(content["platform-union.json"])
    PC.keys(value, {"version", "kind", "source", "pins_sha256", "producers", "coverage", "coverage_sha256",
                    "original_source_report_unchanged", "original_source_metadata_unchanged", "original_job_results_unchanged",
                    "denominator", "global_gate_passed", "production_qualified", "source_report_promoted"})
    require(type(value["version"]) is int and value["version"] == 1)
    require(value["kind"] == "same-revision-platform-source-coverage" and value["source"] == binding["source"])
    require(value["global_gate_passed"] is False and value["production_qualified"] is False and value["source_report_promoted"] is False)
    require(value["denominator"] == "all retained Source and Linux/macOS/Windows coordinate sets; OS-only zeros included")
    sha(value["pins_sha256"])
    pins = {"version": 1, "kind": "same-run-platform-coverage-pins", "source": value["source"],
            "entries": value["producers"], "original_job_results": value["original_job_results_unchanged"]}
    PC.validate_pins(pins, binding["source"])
    require(PC.digest(PC.encoded(pins)) == value["pins_sha256"])
    ids = set()
    for role, job in PC.ROLES.items():
        entry = value["producers"][role]
        producer = entry["descriptor"]["producer"]
        require(producer["repository"] == REPOSITORY and producer["repository_id"] == str(REPOSITORY_ID))
        require(producer["run_id"] == binding["run"]["id"] and producer["attempt"] == "1")
        require(producer["pr_head"] == binding["head"] and producer["event"] == "pull_request")
        require(value["original_job_results_unchanged"][job] == binding["original_jobs"][JOBS[role]])
        require(entry["artifact_id"] not in ids and entry["artifact_id"] != binding["artifact"]["id"])
        ids.add(entry["artifact_id"])
        actual = artifact(api(PREFIX + "/actions/artifacts/" + entry["artifact_id"]), binding["run"]["id"])
        require(actual["sha256"] == entry["archive_sha256"])
        expected_name = "source-audit-" if role == "source" else "platform-go-" + role + "-"
        require(actual["name"] == expected_name + binding["source"]["commit"])
    metadata = value["original_source_metadata_unchanged"]
    # The Source baseline digest binds its OWN profile, not the union profile.
    PC.keys(metadata, {"version", "profile", "commit", "passed", "production_qualified", "checks", "source",
                       "original_report_sha256", "coverage_sha256", "retention_only", "coverage_semantics",
                       "go_version", "platform", "go_tool_sha256"})
    require(metadata["commit"] == binding["source"]["commit"] and metadata["source"]["commit"] == metadata["commit"])
    require(metadata["source"]["tree"] == binding["source"]["tree"] and metadata["production_qualified"] is False)
    require(metadata["retention_only"] is True and type(metadata["passed"]) is bool)
    require(metadata["go_version"] == PC.GO_VERSION and metadata["platform"] == "linux/amd64")
    source_entry = value["producers"]["source"]["descriptor"]
    # The union retains parsed original metadata, not its original JSON bytes.
    # Never pretend reserialization reproduces that original archive digest.
    # Raw-member hashing remains the authenticated Source producer's boundary.
    PC.keys(metadata["source"], {"commit", "tree", "manifest_sha256"})
    sha(metadata["source"]["manifest_sha256"])
    sha(metadata["original_report_sha256"])
    sha(metadata["go_tool_sha256"])
    require(metadata["go_tool_sha256"] == source_entry["toolchain"]["sha256"])
    require(metadata["coverage_semantics"] == "Original audit result; not recomputed or promoted by retention")
    require(metadata["coverage_sha256"] == source_entry["coverage"]["sha256"])
    checks = PC.safe_checks(metadata["checks"])
    require(not metadata["passed"] or all(not row["required"] or row["status"] == "passed" for row in checks))
    require(value["original_source_report_unchanged"] == {"passed": metadata["passed"], "checks": checks,
            "original_report_sha256": metadata["original_report_sha256"], "production_qualified": False})
    blocks = PC.profile(content["coverage.out"], tracked)
    normalized, coverage, gaps = PC.merged_profiles([blocks])
    require(normalized == content["coverage.out"] and coverage == value["coverage"])
    require(PC.digest(normalized) == value["coverage_sha256"])
    require(PC.strict_json(content["gaps.json"]) == {"coverage": coverage, "gaps": gaps})
    return blocks, value


def consume(api, source, expected, binding_data, expected_binding_sha, archive_directory, linux, output):
    require(PC.digest(binding_data) == sha(expected_binding_sha))
    binding = PC.strict_json(binding_data)
    PC.keys(binding, {"version", "repository", "repository_id", "source", "head", "pr", "sonar", "run", "artifact", "original_jobs"})
    require(binding["version"] == 1 and type(binding["version"]) is int)
    require(binding["repository"] == REPOSITORY and binding["repository_id"] == REPOSITORY_ID)
    before, tracked = PC.source_identity(source, expected)
    require(before == binding["source"])
    # Revalidate exact original run identity and stable provider artifact pin.
    current = run_matches(api(PREFIX + "/actions/runs/" + number(binding["run"]["id"])), binding["head"], binding["pr"], "source")
    require(current == binding["run"])
    actual = artifact(api(PREFIX + "/actions/artifacts/" + number(binding["artifact"]["id"])), binding["run"]["id"])
    require(actual == binding["artifact"])
    blocks, metadata = checked_union(PC.archive_file(archive_directory), binding, tracked, api)
    original = PC.safe_file(linux, PC.MAX_PROFILE)
    linux_blocks = PC.profile(original, tracked)
    merged, coverage, _ = PC.merged_profiles([linux_blocks, blocks])
    require(PC.source_identity(source, expected)[0] == before)
    output = PC.new_directory(output, source)
    PC.write_new(output / "coverage.out", merged)
    receipt = {"version": 1, "kind": "sonar-authenticated-source-union", "source": before,
               "source_run": binding["run"], "sonar_run": binding["sonar"], "artifact": binding["artifact"],
               "linux_sha256": PC.digest(original), "union_sha256": metadata["coverage_sha256"],
               "coverage_sha256": PC.digest(merged), "coverage": coverage,
               "original_source_report_unchanged": metadata["original_source_report_unchanged"],
               "original_jobs_unchanged": binding["original_jobs"],
               "global_gate_passed": False, "source_report_promoted": False,
               "trust": "Authenticated same-revision original Source workflow artifact; reviewed producer implementation remains an explicit trust boundary"}
    PC.write_new(output / "source-union-intake.json", PC.encoded(receipt))
    return receipt


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("discover", "consume"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--expected-commit", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--head")
    parser.add_argument("--pr")
    parser.add_argument("--sonar-run")
    parser.add_argument("--step-output", type=Path)
    parser.add_argument("--binding", type=Path)
    parser.add_argument("--binding-sha256")
    parser.add_argument("--archives", type=Path)
    parser.add_argument("--linux", type=Path)
    args = parser.parse_args(argv)
    try:
        api = GitHubRead(os.environ.get("GITHUB_TOKEN"), time.monotonic() + (900 if args.command == "discover" else 120))
        if args.command == "discover":
            source, _ = PC.source_identity(args.source, args.expected_commit)
            binding = discover(api, source, args.head, args.pr, args.sonar_run)
            require(PC.source_identity(args.source, args.expected_commit)[0] == source)
            directory = PC.new_directory(args.output, args.source)
            raw = PC.encoded(binding)
            PC.write_new(directory / "binding.json", raw)
            require(args.step_output is not None and args.step_output.resolve(strict=True) == args.step_output)
            info = args.step_output.stat()
            require(stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid() and info.st_nlink == 1)
            with args.step_output.open("a") as stream:
                stream.write("ready=true\nrun_id=" + binding["run"]["id"] + "\nartifact_id=" + binding["artifact"]["id"] + "\nbinding_sha256=" + PC.digest(raw) + "\n")
        else:
            require(all(item is not None for item in (args.binding, args.binding_sha256, args.archives, args.linux)))
            consume(api, args.source, args.expected_commit, PC.safe_file(args.binding, MAX_JSON, True),
                    args.binding_sha256, args.archives, args.linux, args.output)
        return 0
    except (Refused, PC.InvalidEvidence, OSError, ValueError, TypeError, KeyError, AttributeError,
            urllib.error.URLError, zipfile.BadZipFile, EOFError, RecursionError):
        print("source-union-intake-refused")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
