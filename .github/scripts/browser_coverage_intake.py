"""Admit current original Browser counters, without a new test or scan.

The caller authenticates one same-repository original run and its three native
Node24 steps through bounded Actions reads. Provider digests bind opaque ZIP
bytes; reviewed workflow/source and selected-step correspondence remain trust
assumptions, not hash self-authentication or process attestation. The fourth
input is explicitly the owned current Sonar subsection, never a provider
artifact or an independently completed monolithic job. Counter/reporter math
and strict canonical coverage remain the separate browser_coverage.py boundary.
"""
import argparse
import importlib.util
import os
from pathlib import Path
import re
import stat
import time
import urllib.request


def module(name, path):
    specification = importlib.util.spec_from_file_location(name, path)
    if specification is None or specification.loader is None:
        raise ValueError("browser-intake-refused")
    result = importlib.util.module_from_spec(specification)
    specification.loader.exec_module(result)
    return result


SCRIPTS = Path(__file__).resolve().parent
SI = module("cq_browser_actions", SCRIPTS / "source_union_intake.py")
BC = module("cq_browser_counter", SCRIPTS / "browser_coverage.py")
PC = SI.PC
ROLES = {"linux": "ubuntu-24.04", "macos": "macos-15-intel", "windows": "windows-2025"}
JOB_NAMES = {role: "Browser preview (Node 24.22.2, " + platform + ")" for role, platform in ROLES.items()}
STEP = "Validate dependency-free browser preview"
COMMIT_CACHE = {}


def require(value):
    if not value:
        raise ValueError("browser-intake-refused")


def native_artifact(row, run_id):
    require(type(row) is dict and row.get("expired") is False)
    SI.number(str(row.get("id")))
    require(type(row.get("size_in_bytes")) is int and 0 < row["size_in_bytes"] <= BC.MAX_ARCHIVE)
    digest = row.get("digest")
    require(type(digest) is str and re.fullmatch(r"sha256:[0-9a-f]{64}", digest) is not None)
    require(row.get("workflow_run", {}).get("id") == int(run_id))
    return {"id": str(row["id"]), "sha256": digest[7:], "bytes": row["size_in_bytes"], "name": row["name"]}


def downloaded_archive(directory):
    directory = directory.absolute()
    require(directory.resolve(strict=True) == directory)
    if directory.is_file():
        return BC.private_read(directory, BC.MAX_ARCHIVE)
    require(directory.is_dir())
    entries = list(directory.iterdir())
    require(len(entries) == 1 and entries[0].is_file() and not entries[0].is_symlink())
    return BC.private_read(entries[0], BC.MAX_ARCHIVE)


def github_commit(api, commit):
    """The only additional contents-read endpoint; never arbitrary host/path."""
    PC.hex_value(commit, 40)
    api.calls += 1
    remaining = api.deadline - time.monotonic()
    require(api.calls <= SI.MAX_CALLS and remaining > 0)
    route = SI.PREFIX + "/git/commits/" + commit
    request = urllib.request.Request("https://api.github.com" + route, method="GET", headers={
        "Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28",
        "Authorization": "Bearer " + api.token, "User-Agent": "cq-current-browser-counters"})
    with api.opener.open(request, timeout=min(10, remaining)) as response:
        require(response.status == 200)
        raw = response.read(SI.MAX_JSON + 1)
    require(0 < len(raw) <= SI.MAX_JSON)
    value = PC.strict_json(raw)
    require(value.get("sha") == commit)
    return value


def source_alias(api, commit, source, head):
    if commit == source["commit"]:
        return {"commit": commit, "tree": source["tree"], "authority": "same-current-checkout"}
    if commit not in COMMIT_CACHE:
        COMMIT_CACHE[commit] = github_commit(api, commit)
    value = COMMIT_CACHE[commit]
    require(value.get("tree", {}).get("sha") == source["tree"])
    parents = value.get("parents")
    require(type(parents) is list and len(parents) <= 2)
    for parent in parents:
        PC.hex_value(parent.get("sha"), 40)
    require(commit == head or head in [parent.get("sha") for parent in parents])
    return {"commit": commit, "tree": source["tree"], "authority": "current-head-and-GitHub-commit-tree",
            "parents": [parent["sha"] for parent in parents]}


def current_sonar(api, head, pr, sonar_run):
    identity = SI.run_matches(api(SI.PREFIX + "/actions/runs/" + SI.number(sonar_run)), head, pr, "sonar")
    jobs = SI.paged(api, SI.PREFIX + "/actions/runs/" + identity["id"] + "/jobs", "jobs")
    selected = [job for job in jobs if job.get("name") == "Hosted ephemeral SonarQube gate"]
    require(len(selected) == 1 and selected[0].get("status") == "in_progress")
    return identity, {"id": SI.number(str(selected[0]["id"])), "status": "in_progress"}


def original_browser(api, head, pr, sonar):
    value = api(SI.PREFIX + "/actions/workflows/browser.yml/runs?event=pull_request&head_sha=" + head + "&per_page=100")
    require(type(value.get("total_count")) is int and 0 <= value["total_count"] <= 100)
    require(type(value.get("workflow_runs")) is list and len(value["workflow_runs"]) == value["total_count"])
    candidates = []
    for run in value["workflow_runs"]:
        identity = SI.run_matches(run, head, pr, "browser")
        if abs((SI.stamp(identity["created_at"]) - SI.stamp(sonar["created_at"])).total_seconds()) <= 600:
            candidates.append((run, identity))
    require(len(candidates) <= 1)
    return candidates[0] if candidates else None


def selected_producer(jobs, role):
    matches = [job for job in jobs if job.get("name") == JOB_NAMES[role]]
    require(len(matches) <= 1)
    if not matches:
        return None
    job = matches[0]
    require(job.get("status") in {"queued", "in_progress", "completed"})
    if job["status"] == "queued":
        return None
    steps = [step for step in job.get("steps", []) if step.get("name") == STEP]
    require(len(steps) <= 1)
    if not steps or steps[0].get("status") != "completed":
        require(job["status"] != "completed")
        return None
    require(steps[0].get("conclusion") == "success")
    require(job.get("conclusion") in ({"success", "failure"} if job["status"] == "completed" else {None}))
    return {"id": SI.number(str(job["id"])), "name": JOB_NAMES[role], "status": job["status"],
            "conclusion": job.get("conclusion")}


def role_binding(api, rows, jobs, role, run, source, head):
    pattern = r"browser-coverage-([0-9a-f]{40})-web-" + role
    selected = [(row, re.fullmatch(pattern, row.get("name", ""))) for row in rows]
    selected = [(row, match) for row, match in selected if match]
    require(len(selected) <= 1)
    job = selected_producer(jobs, role)
    if not selected or job is None:
        return None
    row, match = selected[0]
    archive = native_artifact(row, run["id"])
    alias = source_alias(api, match[1], source, head)
    return {"artifact": archive, "job": job, "source_alias": alias}


def ready_browser_binding(api, candidate, source, head, pr, sonar, sonar_job):
    run, identity = candidate
    require(run.get("status") in {"queued", "in_progress", "completed"})
    jobs = SI.paged(api, SI.PREFIX + "/actions/runs/" + identity["id"] + "/jobs", "jobs")
    rows = SI.paged(api, SI.PREFIX + "/actions/runs/" + identity["id"] + "/artifacts", "artifacts", 100)
    roles = {role: role_binding(api, rows, jobs, role, identity, source, head) for role in ROLES}
    if all(roles.values()):
        require(len({entry["artifact"]["id"] for entry in roles.values()}) == 3)
        require(len({entry["job"]["id"] for entry in roles.values()}) == 3)
        return {"version": 1, "source": source, "head": head, "pr": pr, "sonar": sonar,
                "sonar_job": sonar_job, "browser": identity, "entries": roles}
    require(run.get("status") != "completed")
    return None


def discover(api, source, head, pr, sonar_run, sleep=time.sleep):
    sonar, sonar_job = current_sonar(api, head, pr, sonar_run)
    for observation in range(30):
        candidate = original_browser(api, head, pr, sonar)
        if candidate:
            binding = ready_browser_binding(api, candidate, source, head, pr, sonar, sonar_job)
            if binding:
                return binding
        if observation < 29:
            sleep(10)
    raise ValueError("browser-intake-unavailable")


def emit_binding(binding, root, output, step_output):
    PC.new_directory(output, root)
    raw = PC.encoded(binding)
    PC.write_new(output / "binding.json", raw)
    require(step_output.resolve(strict=True) == step_output)
    info = step_output.stat()
    require(stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid() and info.st_nlink == 1)
    with step_output.open("a") as stream:
        stream.write("ready=true\nrun_id=" + binding["browser"]["id"] + "\nbinding_sha256=" + PC.digest(raw) + "\n")
        for role, entry in binding["entries"].items():
            stream.write("artifact_" + role + "=" + entry["artifact"]["id"] + "\n")


def authenticated_native(api, binding, role, directory, source):
    expected = binding["entries"][role]
    actual = native_artifact(api(SI.PREFIX + "/actions/artifacts/" + expected["artifact"]["id"]), binding["browser"]["id"])
    require(actual == expected["artifact"])
    alias = source_alias(api, expected["source_alias"]["commit"], source, binding["head"])
    require(alias == expected["source_alias"])
    raw = downloaded_archive(directory)
    require(len(raw) == actual["bytes"] and PC.digest(raw) == actual["sha256"])
    members = BC.read_archive(raw)
    metadata = BC.strict_json(members["metadata.json"])
    producer = {"repository": SI.REPOSITORY, "run": binding["browser"]["id"], "attempt": "1", "job": "browser",
                "event": "pull_request", "head": binding["head"], "platform": role, "suite": "web", "node": BC.NODE_VERSION}
    require(metadata["producer"] == producer and metadata["source_after"]["commit"] == alias["commit"])
    entry = {"archive_sha256": actual["sha256"], "metadata_sha256": PC.digest(members["metadata.json"]),
            "lcov_sha256": PC.digest(members["lcov.info"]), "counters_sha256": PC.digest(members["counters.json"]),
            "producer": producer, "source_commit": alias["commit"], "node_sha256": metadata["node_after"]["sha256"],
            "artifact_id": actual["id"], "run_id": binding["browser"]["id"], "job_id": expected["job"]["id"]}
    return entry, raw


def local_baseline(binding, source):
    directory = Path(os.environ["RUNNER_TEMP"]) / "cq-browser-full-browser-linux"
    members = BC.read_local_members(directory, os.environ)
    metadata = BC.strict_json(members["metadata.json"])
    require(metadata["source_after"] == source)
    require(metadata["producer"] == BC.producer_identity(os.environ, "linux", "full-browser"))
    return {"kind": "owned-current-sonar", "metadata_sha256": PC.digest(members["metadata.json"]),
            "lcov_sha256": PC.digest(members["lcov.info"]), "counters_sha256": PC.digest(members["counters.json"]),
            "producer": metadata["producer"], "source_commit": source["commit"], "node_sha256": metadata["node_after"]["sha256"],
            "run_id": binding["sonar"]["id"], "job_id": binding["sonar_job"]["id"]}


def consume(api, root, source, binding, archives, output):
    PC.keys(binding, {"version", "source", "head", "pr", "sonar", "sonar_job", "browser", "entries"})
    PC.keys(binding["entries"], ROLES)
    require(type(binding["version"]) is int and binding["version"] == 1 and binding["source"] == source)
    sonar, sonar_job = current_sonar(api, binding["head"], binding["pr"], binding["sonar"]["id"])
    require(sonar == binding["sonar"] and sonar_job == binding["sonar_job"])
    actual_run = SI.run_matches(api(SI.PREFIX + "/actions/runs/" + binding["browser"]["id"]), binding["head"], binding["pr"], "browser")
    require(actual_run == binding["browser"])
    jobs = SI.paged(api, SI.PREFIX + "/actions/runs/" + actual_run["id"] + "/jobs", "jobs")
    for role in ROLES:
        observed = selected_producer(jobs, role)
        require(observed is not None and observed["id"] == binding["entries"][role]["job"]["id"])
    require(BC.source_identity(root, source["commit"])[0] == source)
    native = {role: authenticated_native(api, binding, role, archives / role, source) for role in ROLES}
    entries = {role: item[0] for role, item in native.items()}
    entries["baseline"] = local_baseline(binding, source)
    pins = {"schema": "cq.native-browser.authorized-pins/2", "head": binding["head"], "tree": source["tree"],
            "current_commit": source["commit"], "source_hashes": {key: source[key] for key in ("package_sha256", "implementation_sha256")},
            "entries": entries}
    require(BC.source_identity(root, source["commit"])[0] == source)
    PC.new_directory(output, root)
    for role, item in native.items():
        BC.exclusive_write(output / (role + ".zip"), item[1])
    PC.write_new(output / "pins.json", BC.encoded(pins))
    receipt = {"kind": "current-browser-api-admission", "source": source, "sonar": sonar, "browser": actual_run,
               "provider_artifacts": binding["entries"], "baseline_authority": "owned-current-sonar-not-provider-artifact",
               "pins_sha256": BC.digest(BC.encoded(pins)), "global_release_qualified": False}
    PC.write_new(output / "admission.json", BC.encoded(receipt))


def main(argv=None):
    # Only the fixed API client receives this credential. Source identity uses
    # child Git probes, so remove the exported value before any such probe and
    # never restore it to the process environment.
    token = os.environ.pop("GITHUB_TOKEN", None)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("phase", choices=("discover", "consume"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--sonar-run", required=True)
    parser.add_argument("--expected-commit", required=True)
    parser.add_argument("--head", required=True)
    parser.add_argument("--pr", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--step-output", type=Path)
    parser.add_argument("--binding", type=Path)
    parser.add_argument("--binding-sha256")
    parser.add_argument("--archives", type=Path)
    args = parser.parse_args(argv)
    try:
        root = args.source
        require(root.is_absolute() and root.resolve(strict=True) == root)
        require(root == Path(os.environ["GITHUB_WORKSPACE"]).resolve(strict=True))
        require(args.expected_commit == os.environ["GITHUB_SHA"] and args.sonar_run == os.environ["GITHUB_RUN_ID"])
        PC.hex_value(args.head, 40)
        SI.number(args.pr)
        api = SI.GitHubRead(token, time.monotonic() + (360 if args.phase == "discover" else 120))
        source, _ = BC.source_identity(root, args.expected_commit)
        if args.phase == "discover":
            binding = discover(api, source, args.head, args.pr, args.sonar_run)
            require(BC.source_identity(root, args.expected_commit)[0] == source)
            emit_binding(binding, root, args.output, args.step_output)
        else:
            raw = PC.safe_file(args.binding, SI.MAX_JSON, True)
            require(PC.digest(raw) == args.binding_sha256)
            binding = PC.strict_json(raw)
            require(binding["head"] == args.head and binding["pr"] == args.pr)
            consume(api, root, source, binding, args.archives, args.output)
        return 0
    except (ValueError, OSError, KeyError, TypeError, AttributeError, RuntimeError):
        print("current-browser-evidence-refused")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
