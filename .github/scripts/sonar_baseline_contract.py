"""Bind a genuine unqualified ancestor analysis; never qualify current code.

Only fixed Git metadata and protected JSON files are handled here. HTTP/scanner
calls remain explicit in the workflow. A digest or these data-only validators
are not authentication of a caller-supplied API response.
"""

from __future__ import annotations

import datetime as dt
import hashlib
import json
import os
import re
import stat
import subprocess
import sys
from pathlib import Path

REVISION = "4a9aae67eedfd161c4b23ecf6db1c767421447c3"
TREE = "bc30c289d672d60070279d1113869c9a02eebe95"
MAX_JSON = 128 << 10
SHA = re.compile(r"[a-f0-9]{40}")
DIGEST = re.compile(r"[a-f0-9]{64}")
UUID = re.compile(r"[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}")
BRANCH = re.compile(r"[A-Za-z0-9_./-]{1,128}")
REFUSED = "sonar-baseline-evidence-refused"


def require(condition: bool) -> None:
    if not condition:
        raise ValueError(REFUSED)


def unique(pairs: list[tuple]) -> dict:
    result = {}
    for key, value in pairs:
        require(key not in result)
        result[key] = value
    return result


def nonfinite(_value: str) -> None:
    raise ValueError(REFUSED)


def document(data: bytes) -> dict:
    require(isinstance(data, bytes) and 0 < len(data) <= MAX_JSON)
    result = json.loads(data.decode("utf-8"), object_pairs_hook=unique, parse_constant=nonfinite)
    require(isinstance(result, dict))
    return result


def read_owned(path: str) -> bytes:
    descriptor = os.open(path, os.O_RDONLY | os.O_NONBLOCK | os.O_NOFOLLOW)
    with os.fdopen(descriptor, "rb") as stream:
        before = os.fstat(stream.fileno())
        require(stat.S_ISREG(before.st_mode) and before.st_uid == os.getuid())
        require(before.st_mode & 0o077 == 0 and before.st_nlink == 1)
        require(0 < before.st_size <= MAX_JSON)
        data = stream.read(MAX_JSON + 1)
        after = os.fstat(stream.fileno())
        require(len(data) == before.st_size)
        require((before.st_size, before.st_mtime_ns, before.st_ctime_ns) ==
                (after.st_size, after.st_mtime_ns, after.st_ctime_ns))
        return data


def write_new(path: str, data: bytes) -> None:
    require(0 < len(data) <= MAX_JSON)
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, "wb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def save(path: str, value: dict) -> None:
    write_new(path, (json.dumps(value, sort_keys=True) + "\n").encode())


def git(root: str, *arguments: str, data: bytes | None = None) -> bytes:
    closure = {"PATH": os.environ.get("PATH", os.defpath), "HOME": os.environ["HOME"],
               "TMPDIR": os.environ.get("TMPDIR", "/tmp"), "LANG": "C.UTF-8",
               "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null", "GIT_TERMINAL_PROMPT": "0"}
    result = subprocess.run(
        ["git", "--no-replace-objects", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "log.showSignature=false", "-C", root, *arguments],
        input=data, env=closure, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=30, check=False,
    )
    require(result.returncode == 0 and len(result.stdout) <= 4 << 20)
    return result.stdout


def source_entries(raw: bytes) -> tuple[int, list[str]]:
    require(0 < len(raw) <= 4 << 20)
    entries = [entry for entry in raw.split(b"\0") if entry]
    require(0 < len(entries) <= 5000)
    blobs = []
    seen = set()
    for entry in entries:
        header, name = entry.split(b"\t", 1)
        mode, kind, blob = header.decode().split()
        path = name.decode("utf-8")
        require(mode in {"100644", "100755"} and kind == "blob" and SHA.fullmatch(blob) is not None)
        require(not path.startswith("/") and "\\" not in path)
        require(path not in seen and all(ord(character) >= 32 and ord(character) != 127 for character in path))
        seen.add(path)
        require(all(part not in {"", ".", "..", ".git"} for part in path.split("/")))
        blobs.append(blob)
    return len(entries), blobs


def object_storage(raw: bytes) -> int:
    require(0 < len(raw) <= 4096)
    values = {}
    for line in raw.decode("ascii").splitlines():
        name, value = line.split(":", 1)
        require(name not in values and name != "alternate")
        values[name] = value.strip()
    required = ("size", "size-pack", "size-garbage")
    require(all(name in values and values[name].isdecimal() for name in required))
    total = sum(int(values[name]) * 1024 for name in required)
    require(total <= 512 << 20)
    return total


def source_identity(root: str, target: str) -> dict:
    require(SHA.fullmatch(target) is not None and target != REVISION)
    root = str(Path(root).resolve(strict=True))
    require(git(root, "rev-parse", "--show-toplevel").decode().strip() == root)
    require(git(root, "rev-parse", "HEAD").decode().strip() == target)
    require(not git(root, "status", "--porcelain", "--untracked-files=all"))
    require(git(root, "rev-parse", REVISION + "^{tree}").decode().strip() == TREE)
    git(root, "merge-base", "--is-ancestor", REVISION, target)
    epoch = int(git(root, "show", "-s", "--format=%ct", REVISION).decode().strip())
    target_epoch = int(git(root, "show", "-s", "--format=%ct", target).decode().strip())
    require(0 < epoch < target_epoch)
    timestamp = git(root, "show", "-s", "--format=%cI", REVISION).decode().strip()
    commit_date = dt.datetime.fromisoformat(timestamp)
    require(commit_date.tzinfo is not None and int(commit_date.timestamp()) == epoch)
    target_tree = git(root, "rev-parse", "HEAD^{tree}").decode().strip()
    require(SHA.fullmatch(target_tree) is not None)
    storage = object_storage(git(root, "count-objects", "-v"))
    raw = git(root, "ls-tree", "-rz", REVISION)
    count, blobs = source_entries(raw)
    sizes = git(root, "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)",
                data=("\n".join(blobs) + "\n").encode()).decode().splitlines()
    require(len(sizes) == len(blobs))
    total = 0
    for expected, record in zip(blobs, sizes):
        blob, kind, size = record.split()
        require(blob == expected and kind == "blob" and size.isdecimal())
        size = int(size)
        require(size <= 16 << 20)
        total += size
    require(total <= 512 << 20)
    day = dt.datetime.fromtimestamp(epoch, dt.timezone.utc).date().isoformat()
    return {"schema": "cq-sonar-ancestor-source/1", "repository": "maldous/CircularQuay",
            "revision": REVISION, "tree": TREE, "committer_timestamp": timestamp,
            "committer_epoch": epoch, "project_date": day, "date_granularity": "UTC calendar day; conservative floor",
            "target_revision": target, "target_tree": target_tree, "target_committer_epoch": target_epoch,
            "source_entries": count, "source_bytes": total, "repository_object_bytes": storage, "listing_sha256": hashlib.sha256(raw).hexdigest()}


def source_state(root: str) -> dict:
    """Count checkout drift without exporting paths or reading untracked bodies."""
    raw = git(root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
    entries = [entry for entry in raw.split(b"\0") if entry]
    require(len(entries) <= 10000)
    tracked = untracked = bytecode = 0
    index = 0
    while index < len(entries):
        entry = entries[index]
        require(len(entry) >= 4 and entry[2:3] == b" ")
        code, path = entry[:2], entry[3:]
        if code == b"??":
            untracked += 1
            if b"__pycache__" in path.split(b"/") and path.endswith((b".pyc", b".pyo")):
                bytecode += 1
        else:
            tracked += 1
            if b"R" in code or b"C" in code:
                index += 1
                require(index < len(entries))
        index += 1
    return {"schema": "cq-sonar-checkout-state-counts/1", "clean": not raw,
            "tracked_changes": tracked, "untracked_entries": untracked,
            "python_bytecode_entries": bytecode}


def verify_worktree(root: str) -> None:
    require(git(root, "rev-parse", "HEAD").decode().strip() == REVISION)
    require(git(root, "rev-parse", "HEAD^{tree}").decode().strip() == TREE)
    require(not git(root, "status", "--porcelain", "--untracked-files=all"))


def analysis_list(value: dict, count: int) -> list:
    paging = value.get("paging", {})
    analyses = value.get("analyses")
    require(isinstance(paging, dict) and type(paging.get("total")) is int and paging["total"] == count)
    require(type(paging.get("pageIndex")) is int and paging["pageIndex"] == 1 and type(paging.get("pageSize")) is int and 1 <= paging["pageSize"] <= 100)
    require(isinstance(analyses, list) and len(analyses) == count)
    return analyses


def main_branch(value: dict) -> str:
    branches = value.get("branches")
    require(isinstance(branches, list) and len(branches) == 1)
    branch = branches[0]
    require(isinstance(branch, dict) and branch.get("isMain") is True and branch.get("type") == "BRANCH")
    name = branch.get("name")
    require(isinstance(name, str) and BRANCH.fullmatch(name) is not None)
    return name


def task_id(data: bytes) -> str:
    require(0 < len(data) <= 4096)
    values = [line.split("=", 1)[1] for line in data.decode().splitlines() if line.startswith("ceTaskId=")]
    require(len(values) == 1 and re.fullmatch(r"[A-Za-z0-9_-]{1,128}", values[0]) is not None)
    return values[0]


def server_identity(value: dict) -> dict:
    require(isinstance(value, dict) and value.get("status") == "UP" and value.get("version") == "26.10.0.133078")
    identifier = value.get("id")
    require(isinstance(identifier, str) and re.fullmatch(r"[A-Za-z0-9_-]{8,128}", identifier) is not None)
    return {"id": identifier, "version": value["version"], "status": "UP"}


def validate_identity(value: dict) -> None:
    require(isinstance(value, dict))
    require(value.get("schema") == "cq-sonar-ancestor-source/1" and value.get("repository") == "maldous/CircularQuay")
    require(value.get("revision") == REVISION and value.get("tree") == TREE)
    epoch = value.get("committer_epoch")
    require(type(epoch) is int and epoch > 0 and type(value.get("target_committer_epoch")) is int)
    require(value["target_committer_epoch"] > epoch)
    require(isinstance(value.get("committer_timestamp"), str))
    instant = dt.datetime.fromisoformat(value["committer_timestamp"])
    require(instant.tzinfo is not None and int(instant.timestamp()) == epoch)
    require(value.get("project_date") == dt.datetime.fromtimestamp(epoch, dt.timezone.utc).date().isoformat())
    require(value.get("date_granularity") == "UTC calendar day; conservative floor")
    require(isinstance(value.get("target_revision"), str) and SHA.fullmatch(value["target_revision"]) is not None and value["target_revision"] != REVISION)
    require(isinstance(value.get("target_tree"), str) and SHA.fullmatch(value["target_tree"]) is not None)
    require(type(value.get("source_entries")) is int and 0 < value["source_entries"] <= 5000)
    require(type(value.get("source_bytes")) is int and 0 <= value["source_bytes"] <= 512 << 20)
    require(type(value.get("repository_object_bytes")) is int and 0 <= value["repository_object_bytes"] <= 512 << 20)
    server_identity(value.get("server", {}))
    for key in ("scanner_sha256", "analyzer_sha256", "listing_sha256"):
        require(isinstance(value.get(key), str) and DIGEST.fullmatch(value[key]) is not None)


def observed_baseline(task_doc: dict, analyses: dict, gate: dict) -> dict:
    """Retain safe observed facts even when adoption correctly refuses them."""
    task = task_doc.get("task")
    task = task if isinstance(task, dict) else {}
    rows = analyses.get("analyses")
    analysis = rows[0] if isinstance(rows, list) and len(rows) == 1 and isinstance(rows[0], dict) else {}
    quality = gate.get("projectStatus")
    quality = quality if isinstance(quality, dict) else {}
    def text(value, pattern):
        return value if isinstance(value, str) and pattern.fullmatch(value) is not None else None
    date = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.+-]{8,32}Z?")
    warning_count = task.get("warningCount")
    return {"schema": "cq-sonar-unqualified-baseline-observation/1", "qualified": False,
            "ce_task_id": text(task.get("id"), re.compile(r"[A-Za-z0-9_-]{1,128}")),
            "ce_status": text(task.get("status"), re.compile(r"SUCCESS|PENDING|IN_PROGRESS|FAILED|CANCELED")),
            "ce_analysis_uuid": text(task.get("analysisId"), UUID),
            "analysis_uuid": text(analysis.get("key"), UUID), "revision": text(analysis.get("revision"), SHA),
            "effective_analysis_date": text(analysis.get("date"), date),
            "execution_times": {key: text(task.get(key), date) for key in ("submittedAt", "startedAt", "executedAt")},
            "warning_count": warning_count if type(warning_count) is int and 0 <= warning_count <= 1000 else None,
            "quality_gate_status": text(quality.get("status"), re.compile(r"OK|ERROR|NONE"))}


def baseline_receipt(identity: dict, metadata: bytes, task_doc: dict, analyses: dict,
                     gate: dict, exit_status: str, branch: str) -> dict:
    validate_identity(identity)
    require(exit_status in {"0", "3"} and BRANCH.fullmatch(branch) is not None)
    task = task_doc.get("task", {})
    require(isinstance(task, dict) and task.get("id") == task_id(metadata))
    require(task.get("status") == "SUCCESS" and task.get("componentKey") == "cq")
    analysis = analysis_list(analyses, 1)[0]
    require(isinstance(analysis, dict))
    uuid = analysis.get("key")
    require(isinstance(uuid, str) and UUID.fullmatch(uuid) is not None and task.get("analysisId") == uuid)
    require(analysis.get("revision") == REVISION)
    date = analysis.get("date")
    require(isinstance(date, str))
    instant = dt.datetime.fromisoformat(date)
    require(instant.tzinfo is not None and instant.astimezone(dt.timezone.utc).isoformat() == identity["project_date"] + "T00:00:00+00:00")
    warnings = task.get("warningCount")
    require(type(warnings) is int and 0 <= warnings <= 1000)
    gate_status = gate.get("projectStatus", {})
    require(isinstance(gate_status, dict))
    status = gate_status.get("status")
    require(status in {"OK", "ERROR"})
    execution = {key: task.get(key) for key in ("submittedAt", "startedAt", "executedAt")}
    require(all(isinstance(value, str) for value in execution.values()))
    instants = [dt.datetime.fromisoformat(value) for value in execution.values()]
    require(all(value.tzinfo is not None for value in instants))
    require(instants == sorted(instants) and instants[0].timestamp() >= identity["target_committer_epoch"])
    return {"schema": "cq-sonar-unqualified-baseline/1", "qualified": False, "project": "cq",
            "branch": branch, "analysis_uuid": uuid, "ce_task_id": task["id"],
            "source": identity, "effective_analysis_date": date, "execution_times": execution,
            "scanner_exit": int(exit_status), "quality_gate_status": status, "warning_count": warnings,
            "meaning": "Historical source-day comparand executed now, not historical qualification or current success",
            "new_line_scope": "SCM dates after ancestor UTC day floor; may include earlier same-day changes, not an exact Git diff"}


def period_receipt(value: dict, baseline: dict) -> dict:
    require(baseline.get("schema") == "cq-sonar-unqualified-baseline/1" and baseline.get("qualified") is False)
    validate_identity(baseline.get("source", {}))
    uuid = baseline.get("analysis_uuid")
    require(isinstance(uuid, str) and UUID.fullmatch(uuid) is not None)
    require(value.get("projectKey") == "cq" and value.get("branchKey") == baseline.get("branch"))
    require(value.get("type") == "SPECIFIC_ANALYSIS" and value.get("value") == uuid and value.get("inherited") is False)
    updated = value.get("updatedAt")
    require(type(updated) is int or isinstance(updated, str) and re.fullmatch(r"[1-9][0-9]{0,19}", updated) is not None)
    require(int(updated) > 0)
    return {"schema": "cq-sonar-specific-period-readback/1", "project": "cq", "branch": value["branchKey"],
            "analysis_uuid": uuid, "updated_at": int(updated), "verified": True, "baseline_qualified": False}


def current_receipt(baseline: dict, task_doc: dict, analyses: dict, server: dict) -> dict:
    require(baseline.get("schema") == "cq-sonar-unqualified-baseline/1" and baseline.get("qualified") is False)
    identity = baseline["source"]
    validate_identity(identity)
    require(server_identity(server) == identity["server"])
    current, previous = analysis_list(analyses, 2)
    require(isinstance(current, dict) and isinstance(previous, dict))
    require(previous.get("key") == baseline["analysis_uuid"] and previous.get("revision") == REVISION)
    task = task_doc.get("task", {})
    require(isinstance(task, dict) and task.get("status") == "SUCCESS" and task.get("componentKey") == "cq")
    uuid = current.get("key")
    require(isinstance(uuid, str) and UUID.fullmatch(uuid) is not None)
    require(uuid == task.get("analysisId") and uuid != baseline["analysis_uuid"])
    require(current.get("revision") == identity["target_revision"])
    instant = dt.datetime.fromisoformat(current.get("date", ""))
    require(instant.tzinfo is not None and instant.timestamp() >= identity["target_committer_epoch"])
    require(instant >= dt.datetime.fromisoformat(baseline["execution_times"]["executedAt"]))
    return {"schema": "cq-sonar-current-baseline-correspondence/1", "current_qualified": False,
            "current_analysis_uuid": uuid, "baseline_analysis_uuid": baseline["analysis_uuid"],
            "current_revision": current["revision"], "current_tree": identity["target_tree"],
            "server": identity["server"]}


def new_denominators(value: dict, current: dict) -> dict:
    require(current.get("schema") == "cq-sonar-current-baseline-correspondence/1" and current.get("current_qualified") is False)
    require(isinstance(current.get("current_analysis_uuid"), str) and UUID.fullmatch(current["current_analysis_uuid"]) is not None)
    component = value.get("component", {})
    require(isinstance(component, dict) and component.get("key") == "cq")
    measures = component.get("measures")
    require(isinstance(measures, list) and len(measures) == 3)
    required = {"new_lines", "new_lines_to_cover", "new_conditions_to_cover"}
    counts = {}
    for measure in measures:
        require(isinstance(measure, dict) and "value" not in measure)
        name = measure.get("metric")
        require(isinstance(name, str) and name in required and name not in counts)
        period = measure.get("period", {})
        require(isinstance(period, dict) and ("index" not in period or type(period["index"]) is int and period["index"] == 1))
        raw = period.get("value")
        require(isinstance(raw, str) and re.fullmatch(r"0|[1-9][0-9]{0,19}", raw) is not None)
        counts[name] = int(raw)
    require(set(counts) == required and counts["new_lines"] > 0)
    require(counts["new_lines_to_cover"] + counts["new_conditions_to_cover"] > 0)
    return {"schema": "cq-sonar-new-code-denominators/1", "current_qualified": False,
            "analysis_uuid": current["current_analysis_uuid"], "counts": counts}


def admin_config(password: bytes) -> bytes:
    # Actual repository producer: Cq9! + hex.EncodeToString(32 random bytes).
    require(re.fullmatch(rb"Cq9![0-9a-f]{64}\n", password) is not None)
    return b'user = "admin:' + password.rstrip(b"\n") + b'"\n'


ARGUMENT_COUNTS = {"prepare": 6, "state": 2, "verify": 1, "empty": 1, "branch": 1, "admin": 2,
                   "analysis": 9, "period": 4, "current": 5, "denominators": 3, "observe": 2}


def execute(command: str, args: list[str]) -> str | None:
    if command == "prepare":
        identity = source_identity(args[0], args[1])
        identity["server"] = server_identity(document(read_owned(args[2])))
        identity.update(scanner_sha256=args[4], analyzer_sha256=args[5])
        validate_identity(identity)
        save(args[3], identity)
        return identity["project_date"]
    if command == "state":
        save(args[1], source_state(args[0]))
    elif command == "verify":
        verify_worktree(args[0])
    elif command == "empty":
        analysis_list(document(read_owned(args[0])), 0)
    elif command == "branch":
        return main_branch(document(read_owned(args[0])))
    elif command == "admin":
        write_new(args[1], admin_config(read_owned(args[0])))
    elif command == "observe":
        save(args[1], observed_baseline(document(read_owned(args[0])), {}, {}))
    elif command == "analysis":
        identity, metadata, task, analyses, gate, status, branch, output = args[:8]
        require(args[8] == "unqualified")
        task_doc, analysis_doc, gate_doc = (document(read_owned(path)) for path in (task, analyses, gate))
        save(output + ".observation.json", observed_baseline(task_doc, analysis_doc, gate_doc))
        result = baseline_receipt(document(read_owned(identity)), read_owned(metadata), task_doc,
                                  analysis_doc, gate_doc, status, branch)
        save(output, result)
        return result["analysis_uuid"]
    elif command == "period":
        baseline = document(read_owned(args[1]))
        require(server_identity(document(read_owned(args[2]))) == baseline["source"]["server"])
        save(args[3], period_receipt(document(read_owned(args[0])), baseline))
    elif command == "current":
        save(args[4], current_receipt(*(document(read_owned(path)) for path in args[:4])))
    elif command == "denominators":
        save(args[2], new_denominators(document(read_owned(args[0])), document(read_owned(args[1]))))
    else:
        raise ValueError(REFUSED)
    return None


def main(arguments: list[str] | None = None) -> int:
    arguments = sys.argv[1:] if arguments is None else arguments
    try:
        command, *args = arguments
        require(command in ARGUMENT_COUNTS and len(args) == ARGUMENT_COUNTS[command])
        output = execute(command, args)
        if output is not None:
            print(output)
    except (ValueError, OSError, TypeError, KeyError, AttributeError, UnicodeError, RecursionError, subprocess.SubprocessError):
        print(REFUSED, file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
