"""Join authenticated native Windows Python counters into current Sonar evidence.

Only the fixed platform/browser coverage modules are supplemented. Coverage7.17
line and branch-pair inventories must agree completely across interpreters;
neither OS guesses nor numeric branch aliases are evidence. The reviewed SDK
producer and authenticated Actions artifact remain explicit trust boundaries.
Raw Linux evidence and original job outcomes are never rewritten or promoted.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import io
import os
from pathlib import Path
import re
import stat
import sys
import time
import xml.etree.ElementTree as ET
import zipfile
import zlib

SPEC = importlib.util.spec_from_file_location("cq_python_intake_source", Path(__file__).with_name("source_union_intake.py"))
assert SPEC is not None and SPEC.loader is not None
SI = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SI)
PC = SI.PC
TARGET = ".github/scripts/platform_coverage.py"
SUITE = ".github/scripts/tests/test_platform_coverage.py"
TARGETS = {TARGET: SUITE,
           ".github/scripts/browser_coverage.py": ".github/scripts/tests/test_browser_coverage.py"}
SELF = ".github/scripts/python_coverage.py"
LOCK = "sdk/python/locks/ci-win32-py314.txt"
BINDINGS = (*TARGETS, *TARGETS.values(), SELF, LOCK, "sdk/python/requirements-ci.txt")
JOB = "Python static and security (3.14, windows-2025)"
STEP = "Measure native Windows platform-helper counters"
ARTIFACT = "native-python-platform-coverage-"
MAX_JSON = 16 << 20
MAX_XML = 16 << 20
MAX_ARCHIVE = 32 << 20
CLASS = re.compile(rb"<class\b[^>]*>.*?</class>", re.S)
LINE = re.compile(rb"<line\b[^>]*/>")


def require(value):
    if not value:
        raise ValueError("native-python-coverage-refused")


def integers(value, signed=False):
    require(type(value) is list and len(value) <= 20000)
    require(all(type(n) is int and (-100000 <= n <= 100000 if signed else 0 < n <= 100000) for n in value))
    require(len(set(value)) == len(value))
    return set(value)


def pairs(value):
    require(type(value) is list and len(value) <= 20000)
    result = set()
    for pair in value:
        require(type(pair) is list and len(pair) == 2)
        require(type(pair[0]) is int and 0 < pair[0] <= 100000)
        require(type(pair[1]) is int and -100000 <= pair[1] <= 100000 and pair[1] != 0)
        result.add(tuple(pair))
    require(len(result) == len(value))
    return result


def validate_counter(counter):
    PC.keys(counter, {"lines", "hit_lines", "excluded", "branches", "hit_branches"})
    lines = integers(counter["lines"])
    hits = integers(counter["hit_lines"])
    excluded = integers(counter["excluded"])
    branches, taken = pairs(counter["branches"]), pairs(counter["hit_branches"])
    # This first profile admits no source exclusions. With X empty, the pinned
    # reporter's disjoint executed/missing pairs are subsets of eligible branch
    # exits; exact summary cardinality below proves they exhaust that inventory.
    # This is not a claim to capture single-exit control-flow arcs.
    require(lines and hits <= lines and not excluded and taken <= branches)
    exits = {}
    for start, _ in branches:
        require(start in lines)
        exits[start] = exits.get(start, 0) + 1
    require(all(count >= 2 for count in exits.values()))
    return lines, hits, excluded, branches, taken


def official_files(data):
    """Validate the fixed coverage.py envelope and unique relative paths."""
    require(isinstance(data, bytes) and 0 < len(data) <= MAX_JSON)
    report = PC.strict_json(data)
    meta = report["meta"]
    require(meta["format"] == 3 and type(meta["format"]) is int)
    require(meta["version"] == "7.17.0" and meta["branch_coverage"] is True and meta["show_contexts"] is False)
    files = report["files"]
    require(type(files) is dict and 0 < len(files) <= 256)
    names = [name.replace("\\", "/") for name in files]
    require(len(names) == len(set(names)))
    return {name.replace("\\", "/"): record for name, record in files.items()}


def official_counter(record):
    """Retain observed coordinates only after exact official summary checks."""
    executed, missing = integers(record["executed_lines"]), integers(record["missing_lines"])
    excluded = integers(record["excluded_lines"])
    # Refuse unexplained inventory differences, including any future exclusion
    # that would invalidate the complete eligible-branch cardinality proof.
    hits, lines = executed - excluded, (executed | missing) - excluded
    taken, absent = pairs(record["executed_branches"]), pairs(record["missing_branches"])
    require(not executed & missing and not taken & absent)
    summary = record["summary"]
    expected = {"num_statements": len(lines), "covered_lines": len(hits), "missing_lines": len(missing),
                "excluded_lines": len(excluded), "num_branches": len(taken | absent),
                "covered_branches": len(taken), "missing_branches": len(absent)}
    require(all(type(summary.get(key)) is int and summary[key] == value for key, value in expected.items()))
    counter = {"lines": sorted(lines), "hit_lines": sorted(hits), "excluded": sorted(excluded),
               "branches": sorted(map(list, taken | absent)), "hit_branches": sorted(map(list, taken))}
    validate_counter(counter)
    return counter


def normalized_report(data, native=False, target=TARGET):
    files = official_files(data)
    require(target in TARGETS and target in files)
    require(not native or list(files) == [target])
    return official_counter(files[target])


def normalized_reports(data, native=False):
    files = official_files(data)
    if native:
        require(len(files) == len(TARGETS) and set(files) == set(TARGETS))
    require(set(TARGETS) <= set(files))
    return {"version": 2, "files": {target: official_counter(files[target]) for target in TARGETS}}


def merge_counters(linux, native):
    first, second = validate_counter(linux), validate_counter(native)
    require(all(first[index] == second[index] for index in (0, 2, 3)))
    return {**linux, "hit_lines": sorted(first[1] | second[1]),
            "hit_branches": sorted(map(list, first[4] | second[4]))}


def source_state(source, expected):
    identity, _ = PC.source_identity(source, expected)
    raw = {name: PC.safe_file(source / name, MAX_JSON) for name in BINDINGS}
    bindings = {name: PC.digest(data) for name, data in raw.items()}
    blobs = {}
    for target in TARGETS:
        blob = PC.git(source, "rev-parse", "HEAD:" + target).decode("ascii").strip()
        PC.hex_value(blob, 40)
        data = raw[target]
        require(hashlib.sha1(("blob " + str(len(data)) + "\0").encode() + data).hexdigest() == blob)
        blobs[target] = blob
    return {"identity": identity, "bindings": bindings, "platform_blob": blobs[TARGET], "module_blobs": blobs}


def python_tool():
    """Measure this selected interpreter, not a PATH-selected alternate process."""
    import coverage  # Only the existing hash-locked producer environment imports it.

    require(sys.implementation.name == "cpython" and sys.version_info[:2] == (3, 14))
    require(coverage.__version__ == "7.17.0")
    binary = Path(sys.executable).resolve(strict=True)
    module = Path(coverage.__file__).resolve(strict=True)
    # Record reality, not a platform override. Only the authenticated Windows
    # job plus the consumer's exact platform fence can admit native counters.
    return {"python": ".".join(map(str, sys.version_info[:3])), "implementation": "cpython", "platform": sys.platform,
            "binary_sha256": PC.digest(PC.safe_file(binary, 128 << 20)),
            "coverage_version": coverage.__version__, "coverage_module_sha256": PC.digest(PC.safe_file(module, 1 << 20))}


def origin():
    result = {name: os.environ.get(name) for name in ("GITHUB_REPOSITORY", "GITHUB_REPOSITORY_ID",
              "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_JOB", "GITHUB_EVENT_NAME", "CQ_COVERAGE_PR_HEAD")}
    require(result["GITHUB_REPOSITORY"] == SI.REPOSITORY and result["GITHUB_REPOSITORY_ID"] == str(SI.REPOSITORY_ID))
    require(result["GITHUB_JOB"] == "python-static-security" and result["GITHUB_EVENT_NAME"] == "pull_request")
    require(result["GITHUB_RUN_ATTEMPT"] == "1")
    SI.number(result["GITHUB_RUN_ID"])
    PC.hex_value(result["CQ_COVERAGE_PR_HEAD"], 40)
    return result


def linux_begin(source, expected, output):
    state = {"source": source_state(source, expected), "tool": python_tool()}
    require(state["tool"]["platform"] == "linux")
    PC.new_directory(output, source)
    PC.write_new(output / "linux-tool.json", PC.encoded(state))


def linux_check(source, expected, output):
    state = PC.strict_json(PC.safe_file(output / "linux-tool.json", SI.MAX_JSON, True))
    require(state == {"source": source_state(source, expected), "tool": python_tool()})
    require(state["tool"]["platform"] == "linux")
    PC.write_new(output / "linux-tool-verified.json", PC.encoded(state))


def begin(source, expected, output):
    snapshot = {"source": source_state(source, expected), "tool": python_tool(), "origin": origin()}
    PC.new_directory(output, source)
    PC.write_new(output / "snapshot.json", PC.encoded(snapshot))
    (output / "raw").mkdir(mode=0o700)
    # Own configuration, never inherit a project/user coverage setting. No
    # native-only denominator pass is asserted; the full composite is checked.
    configuration = "[run]\nbranch = True\nrelative_files = True\ninclude =\n" + "".join("    " + target + "\n" for target in TARGETS) + "[report]\nfail_under = 0\n"
    PC.write_new(output / "coverage.ini", configuration.encode("ascii"))


def retain(source, expected, output, test_exit):
    require(type(test_exit) is int and 0 <= test_exit <= 255)
    snapshot = PC.strict_json(PC.safe_file(output / "snapshot.json", SI.MAX_JSON))
    require(snapshot == {"source": source_state(source, expected), "tool": python_tool(), "origin": origin()})
    raw = PC.safe_file(output / "raw" / "official.json", MAX_JSON)
    counter = normalized_reports(raw, native=True)
    actual_counts = suite_counts(PC.safe_file(output / "raw" / "test.log", 4 << 20))
    retained = output / "retained"
    PC.new_directory(retained, source)
    data = PC.encoded(counter)
    metadata = {"version": 2, "kind": "platform-python-counters", **snapshot,
                "suites": list(TARGETS.values()), "test_exit": test_exit, "test_counts": actual_counts,
                "source_before_after_equal": True,
                "official_json_sha256": PC.digest(raw), "counter_sha256": PC.digest(data),
                "global_gate_passed": False, "production_qualified": False}
    PC.write_new(retained / "coverage.json", data)
    PC.write_new(retained / "metadata.json", PC.encoded(metadata))
    print("native-python-counts:" + PC.encoded(actual_counts).decode("ascii").strip())


def suite_counts(raw):
    """Retain only the actual stdlib runner totals, never its private body."""
    require(isinstance(raw, bytes) and 0 < len(raw) <= 4 << 20)
    tail = raw[-4096:].replace(b"\r\n", b"\n")
    matches = re.findall(rb"(?:^|\n)Ran ([0-9]{1,5}) tests? in [0-9.]+s\n\n(?:OK|FAILED)(?: \(([^\n]*)\))?(?:\n|$)", tail)
    require(len(matches) == 1)
    selected, detail = matches[0]
    selected = int(selected)
    skips = re.findall(rb"(?:^|, )skipped=([0-9]{1,5})(?:, |$)", detail)
    require(len(skips) <= 1)
    skipped = int(skips[0]) if skips else 0
    require(0 < selected <= 10000 and 0 <= skipped <= selected)
    return {"selected": selected, "executed": selected - skipped, "skipped": skipped}


def selected_job(jobs):
    matching = [job for job in jobs if job.get("name") == JOB]
    require(len(matching) <= 1)
    if not matching:
        return None
    job = matching[0]
    status = job.get("status")
    require(status in {"queued", "in_progress", "completed"})
    if status == "queued":
        return None
    require(job.get("conclusion") in ({"success", "failure"} if status == "completed" else {None}))
    steps = [step for step in job.get("steps", []) if step.get("name") == STEP]
    require(len(steps) <= 1)
    if not steps or steps[0].get("status") != "completed":
        require(status != "completed")
        return None
    require(steps[0].get("conclusion") == "success")
    return {"id": SI.number(str(job["id"])), "name": JOB, "status": status, "conclusion": job["conclusion"]}


def original_sdk_candidate(api, head, pr, sonar):
    value = api(SI.PREFIX + "/actions/workflows/sdk.yml/runs?event=pull_request&head_sha=" + head + "&per_page=100")
    require(type(value.get("total_count")) is int and 0 <= value["total_count"] <= 100)
    require(type(value.get("workflow_runs")) is list and len(value["workflow_runs"]) == value["total_count"])
    candidates = []
    for run in value["workflow_runs"]:
        identity = SI.run_matches(run, head, pr, "sdk")
        if abs((SI.stamp(identity["created_at"]) - SI.stamp(sonar["created_at"])).total_seconds()) <= 600:
            candidates.append((run, identity))
    require(len(candidates) <= 1)
    return candidates[0] if candidates else None


def ready_sdk_binding(api, candidate, source, head, pr, sonar):
    run, identity = candidate
    require(run.get("status") in {"queued", "in_progress", "completed"})
    jobs = SI.paged(api, SI.PREFIX + "/actions/runs/" + identity["id"] + "/jobs", "jobs")
    job = selected_job(jobs)
    rows = SI.paged(api, SI.PREFIX + "/actions/runs/" + identity["id"] + "/artifacts", "artifacts", 100)
    archives = [row for row in rows if row.get("name") == ARTIFACT + source["identity"]["commit"]]
    require(len(archives) <= 1)
    if job and archives:
        return {"version": 1, "source": source, "head": head, "pr": pr, "sonar": sonar,
                "run": identity, "job": job, "artifact": SI.artifact(archives[0], identity["id"])}
    require(run.get("status") != "completed")
    return None


def discover(api, source, head, pr, sonar_run, sleep=time.sleep):
    sonar = SI.run_matches(api(SI.PREFIX + "/actions/runs/" + SI.number(sonar_run)), head, pr, "sonar")
    for observation in range(30):
        candidate = original_sdk_candidate(api, head, pr, sonar)
        if candidate:
            binding = ready_sdk_binding(api, candidate, source, head, pr, sonar)
            if binding:
                return binding
        if observation < 29:
            sleep(10)
    raise ValueError("native-python-coverage-unavailable")


def archive_bytes(data, pin):
    require(0 < len(data) <= MAX_ARCHIVE and len(data) == pin["bytes"] and PC.digest(data) == pin["sha256"])
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        members = archive.infolist()
        require(len(members) == 2 and {m.filename for m in members} == {"coverage.json", "metadata.json"})
        result = {}
        for member in members:
            mode = member.external_attr >> 16
            require(not member.is_dir() and not member.flag_bits & 1)
            require(member.compress_type in (zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED))
            require(stat.S_IFMT(mode) in (0, stat.S_IFREG) and 0 < member.file_size <= MAX_JSON)
            with archive.open(member) as stream:
                payload = stream.read(MAX_JSON + 1)
            require(len(payload) == member.file_size)
            result[member.filename] = PC.strict_json(payload)
    return result


def attribute(tag, name, value):
    """Replace one generated attribute, preserving all other lexical bytes."""
    pattern = rb'\b' + name.encode() + rb'="[^"]*"'
    require(len(re.findall(pattern, tag)) == 1)
    return re.sub(pattern, name.encode() + b'="' + str(value).encode("ascii") + b'"', tag, count=1)


def counts(element):
    lines, hits, branches, taken = 0, 0, 0, 0
    for line in element.findall(".//line"):
        lines += 1
        require(line.get("hits") in {"0", "1"})
        hits += int(line.get("hits"))
        if line.get("branch") == "true":
            match = re.fullmatch(r"[0-9]+% \(([0-9]+)/([0-9]+)\)", line.get("condition-coverage", ""))
            require(match is not None)
            a, b = map(int, match.groups())
            require(0 <= a <= b and b > 0)
            taken, branches = taken + a, branches + b
    return lines, hits, branches, taken


def rates(tag, count):
    lines, hits, branches, taken = count
    tag = attribute(tag, "line-rate", rate(hits, lines))
    return attribute(tag, "branch-rate", rate(taken, branches))


def rate(hit, total):
    # Match the pinned XML reporter: rounding must never turn a partial ratio
    # into an apparent zero or one. Canonical admission also uses exact counts.
    require(type(hit) is int and type(total) is int and 0 <= hit <= total)
    if total == 0 or hit == total:
        return "1"
    if hit == 0:
        return "0"
    return format(min(max(hit / total, 0.0001), 0.9999), ".4g")


def selected_xml_class(raw, target):
    require(target in TARGETS and 0 < len(raw) <= MAX_XML and b"<!DOCTYPE" not in raw)
    root = ET.fromstring(raw)
    require(root.tag == "coverage" and root.get("version") == "7.17.0")
    candidates = []
    for match in CLASS.finditer(raw):
        element = ET.fromstring(match.group())
        if element.get("filename") in {target, target.rsplit("/", 1)[1]}:
            candidates.append((match, element))
    require(len(candidates) == 1)
    return candidates[0]


def missing_destinations(choices, taken):
    return ",".join("exit" if n < 0 else str(n) for n in sorted(pair[1] for pair in choices - taken))


def validate_xml_branch(line, choices, taken):
    if not choices:
        require(line.get("branch") is None)
        return
    covered = len(choices & taken)
    require(line.get("branch") == "true")
    require(line.get("condition-coverage") == f"{100 * covered // len(choices)}% ({covered}/{len(choices)})")
    require(line.get("missing-branches", "") == missing_destinations(choices, taken))


def validate_xml_class(element, linux):
    lines, hits, _, branches, taken = validate_counter(linux)
    wanted = {int(line.get("number")): line for line in element.findall("./lines/line")}
    require(set(wanted) == lines and {n for n, line in wanted.items() if line.get("hits") == "1"} == hits)
    require(counts(element) == (len(lines), len(hits), len(branches), len(taken)))
    for number, line in wanted.items():
        choices = {pair for pair in branches if pair[0] == number}
        validate_xml_branch(line, choices, taken)
    return lines, hits, branches


def updated_xml_line(old, final):
    line = ET.fromstring(old)
    number = int(line.get("number"))
    updated = attribute(old, "hits", int(number in final[1]))
    choices = {pair for pair in final[3] if pair[0] == number}
    if not choices:
        return updated
    require(line.get("branch") == "true")
    absent = choices - final[4]
    total, covered = len(choices), len(choices) - len(absent)
    updated = attribute(updated, "condition-coverage", f"{100 * covered // total}% ({covered}/{total})")
    updated = re.sub(rb' missing-branches="[^"]*"', b"", updated)
    if absent:
        value = missing_destinations(choices, final[4])
        updated = updated[:-2] + b' missing-branches="' + value.encode() + b'"/>'
    return updated


def updated_xml_package(found):
    data = found.group()
    opening = re.match(rb"<package\b[^>]*>", data)
    return rates(opening.group(), counts(ET.fromstring(data))) + data[opening.end():]


def updated_xml_aggregates(output):
    # Aggregates reflect ALL classes. No class is serialized or dropped.
    output = re.sub(rb"<package\b[^>]*>.*?</package>", updated_xml_package, output, flags=re.S)
    count = counts(ET.fromstring(output))
    opening = re.search(rb"<coverage\b[^>]*>", output)
    updated = rates(opening.group(), count)
    for name, value in zip(("lines-valid", "lines-covered", "branches-valid", "branches-covered"), count):
        updated = attribute(updated, name, value)
    return output[:opening.start()] + updated + output[opening.end():], count


def merge_xml(raw, linux, composite, target=TARGET):
    """Modify one selected class and aggregate attributes, never siblings."""
    match, element = selected_xml_class(raw, target)
    lines, _, branches = validate_xml_class(element, linux)
    final = validate_counter(composite)
    require(final[0] == lines and final[2] == validate_counter(linux)[2] and final[3] == branches)
    changed = LINE.sub(lambda found: updated_xml_line(found.group(), final), match.group())
    header = re.match(rb"<class\b[^>]*>", changed)
    changed = rates(header.group(), (len(lines), len(final[1]), len(branches), len(final[4]))) + changed[header.end():]
    output, count = updated_xml_aggregates(raw[:match.start()] + changed + raw[match.end():])
    before_other = [m.group() for m in CLASS.finditer(raw) if m.group() != match.group()]
    names = {target, target.rsplit("/", 1)[1]}
    after_other = [m.group() for m in CLASS.finditer(output) if ET.fromstring(m.group()).get("filename") not in names]
    require(before_other == after_other and len(output) <= MAX_XML)
    return output, {"lines": count[0], "covered_lines": count[1], "branches": count[2], "covered_branches": count[3],
                    "passed": count[0] > 0 and count[0] == count[1] and count[2] == count[3]}


def validate_tool(tool, platform):
    PC.keys(tool, {"python", "implementation", "platform", "binary_sha256", "coverage_version", "coverage_module_sha256"})
    require(tool["coverage_version"] == "7.17.0" and tool["platform"] == platform and tool["implementation"] == "cpython")
    require(re.fullmatch(r"3\.14\.[0-9]{1,3}", tool["python"]) is not None)
    for name in ("binary_sha256", "coverage_module_sha256"):
        PC.hex_value(tool[name])


def authenticated_members(api, binding, expected, archives):
    run = SI.run_matches(api(SI.PREFIX + "/actions/runs/" + SI.number(binding["run"]["id"])), binding["head"], binding["pr"], "sdk")
    require(run == binding["run"])
    pin = SI.artifact(api(SI.PREFIX + "/actions/artifacts/" + SI.number(binding["artifact"]["id"])), run["id"])
    require(pin == binding["artifact"] and pin["name"] == ARTIFACT + expected)
    admitted_job = selected_job(SI.paged(api, SI.PREFIX + "/actions/runs/" + run["id"] + "/jobs", "jobs"))
    require(admitted_job is not None and all(admitted_job[key] == binding["job"][key] for key in ("id", "name")))
    members = archive_bytes(PC.archive_file(archives), pin)
    return members, pin, admitted_job, run


def validated_suite_counts(actual_counts):
    PC.keys(actual_counts, {"selected", "executed", "skipped"})
    require(all(type(n) is int for n in actual_counts.values()))
    require(0 < actual_counts["selected"] <= 10000 and 0 <= actual_counts["skipped"] <= actual_counts["selected"])
    require(actual_counts["executed"] == actual_counts["selected"] - actual_counts["skipped"])
    return actual_counts


def validated_native_metadata(metadata, binding, run):
    PC.keys(metadata, {"version", "kind", "source", "tool", "origin", "suites", "test_exit", "test_counts",
                       "source_before_after_equal", "official_json_sha256", "counter_sha256",
                       "global_gate_passed", "production_qualified"})
    require(type(metadata["version"]) is int and metadata["version"] == 2 and metadata["kind"] == "platform-python-counters")
    require(metadata["source"] == binding["source"] and metadata["source_before_after_equal"] is True)
    require(metadata["suites"] == list(TARGETS.values()) and type(metadata["test_exit"]) is int and metadata["test_exit"] == 0)
    actual_counts = validated_suite_counts(metadata["test_counts"])
    require(metadata["global_gate_passed"] is False and metadata["production_qualified"] is False)
    producer = metadata["origin"]
    require(producer == {"GITHUB_REPOSITORY": SI.REPOSITORY, "GITHUB_REPOSITORY_ID": str(SI.REPOSITORY_ID),
                       "GITHUB_RUN_ID": run["id"], "GITHUB_RUN_ATTEMPT": "1", "GITHUB_JOB": "python-static-security",
                       "GITHUB_EVENT_NAME": "pull_request", "CQ_COVERAGE_PR_HEAD": binding["head"]})
    tool = metadata["tool"]
    validate_tool(tool, "win32")
    return tool, actual_counts


def consume(api, source, expected, binding_data, binding_sha, archives, linux_json, linux_xml, linux_tool, output):
    require(PC.digest(binding_data) == binding_sha)
    binding = PC.strict_json(binding_data)
    PC.keys(binding, {"version", "source", "head", "pr", "sonar", "run", "job", "artifact"})
    require(binding["version"] == 1 and binding["source"] == source_state(source, expected))
    members, pin, admitted_job, run = authenticated_members(api, binding, expected, archives)
    metadata, native = members["metadata.json"], members["coverage.json"]
    tool, actual_counts = validated_native_metadata(metadata, binding, run)
    linux_identity = PC.strict_json(PC.safe_file(linux_tool, SI.MAX_JSON, True))
    PC.keys(linux_identity, {"source", "tool"})
    require(linux_identity["source"] == binding["source"])
    validate_tool(linux_identity["tool"], "linux")
    require(PC.digest(PC.encoded(native)) == metadata["counter_sha256"])
    PC.hex_value(metadata["official_json_sha256"])
    original_json, original_xml = PC.safe_file(linux_json, MAX_JSON), PC.safe_file(linux_xml, MAX_XML)
    linux = normalized_reports(original_json)
    PC.keys(native, {"version", "files"})
    require(type(native["version"]) is int and native["version"] == 2)
    PC.keys(native["files"], TARGETS)
    result = original_xml
    for target in TARGETS:
        first, second = linux["files"][target], native["files"][target]
        composite = merge_counters(first, second)
        result, summary = merge_xml(result, first, composite, target)
    require(source_state(source, expected) == binding["source"])
    PC.new_directory(output, source)
    PC.write_new(output / "python-coverage.xml", result)
    receipt = {"version": 1, "kind": "same-source-native-python-composite", "source": binding["source"],
               "original_sdk_job": {"discovery": binding["job"], "admission": admitted_job}, "artifact": pin,
               "python": {"windows": tool, "linux": linux_identity["tool"]}, "native_test_counts": actual_counts,
               "linux_json_sha256": PC.digest(original_json), "linux_xml_sha256": PC.digest(original_xml),
               "composite_xml_sha256": PC.digest(result), "summary": summary,
               "raw_linux_status_promoted": False, "global_gate_passed": False, "production_qualified": False}
    PC.write_new(output / "python-composite.json", PC.encoded(receipt))
    return summary


def producer_command(args):
    if args.command == "begin":
        begin(args.source, args.expected_commit, args.output)
    elif args.command == "retain":
        retain(args.source, args.expected_commit, args.output, args.test_exit)
    elif args.command == "linux-begin":
        linux_begin(args.source, args.expected_commit, args.output)
    elif args.command == "linux-check":
        linux_check(args.source, args.expected_commit, args.output)
    else:
        return False
    return True


def store_discovered_binding(args, api):
    state = source_state(args.source, args.expected_commit)
    binding = discover(api, state, args.head, args.pr, args.sonar_run)
    require(state == source_state(args.source, args.expected_commit))
    PC.new_directory(args.output, args.source)
    raw = PC.encoded(binding)
    PC.write_new(args.output / "binding.json", raw)
    require(args.step_output is not None and args.step_output.resolve(strict=True) == args.step_output)
    info = args.step_output.stat()
    require(stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid() and info.st_nlink == 1)
    with args.step_output.open("a") as stream:
        stream.write("ready=true\nrun_id=" + binding["run"]["id"] + "\nartifact_id=" + binding["artifact"]["id"] + "\nbinding_sha256=" + PC.digest(raw) + "\n")


def api_command(args):
    api = SI.GitHubRead(os.environ.get("GITHUB_TOKEN"), time.monotonic() + (360 if args.command == "discover" else 120))
    if args.command == "discover":
        store_discovered_binding(args, api)
        return 0
    summary = consume(api, args.source, args.expected_commit, PC.safe_file(args.binding, SI.MAX_JSON, True),
                      args.binding_sha256, args.archives, args.linux_json, args.linux_xml, args.linux_tool, args.output)
    return 0 if summary["passed"] else 1


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("begin", "retain", "discover", "consume", "linux-begin", "linux-check"))
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--expected-commit", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--test-exit", type=int)
    parser.add_argument("--head")
    parser.add_argument("--pr")
    parser.add_argument("--sonar-run")
    parser.add_argument("--step-output", type=Path)
    parser.add_argument("--binding", type=Path)
    parser.add_argument("--binding-sha256")
    parser.add_argument("--archives", type=Path)
    parser.add_argument("--linux-json", type=Path)
    parser.add_argument("--linux-xml", type=Path)
    parser.add_argument("--linux-tool", type=Path)
    args = parser.parse_args(argv)
    try:
        if producer_command(args):
            return 0
        return api_command(args)
    except (ValueError, OSError, KeyError, TypeError, AttributeError, RecursionError,
            ET.ParseError, zipfile.BadZipFile, EOFError, RuntimeError, zlib.error):
        print("native-python-coverage-refused")
        return 2



if __name__ == "__main__":
    raise SystemExit(main())
