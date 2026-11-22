"""Owned synthetic producer, authority, complete-inventory and XML oracles."""
from __future__ import annotations

import contextlib
import copy
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import stat
import tempfile
from types import SimpleNamespace
import unittest
from unittest import mock
import zipfile

SCRIPT = Path(__file__).resolve().parents[1] / "python_coverage.py"
SPEC = importlib.util.spec_from_file_location("cq_native_python_contract", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
P = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(P)
COMMIT, TREE, HEAD = "a" * 40, "b" * 40, "c" * 40
SOURCE = {"identity": {"commit": COMMIT, "tree": TREE, "tracked_clean": True,
          "locks": {"go.mod": "1" * 64, "go.sum": "2" * 64}},
          "bindings": {name: "3" * 64 for name in P.BINDINGS}, "platform_blob": "d" * 40,
          "module_blobs": {target: "d" * 40 for target in P.TARGETS}}
TOOL = {"python": "3.14.10", "implementation": "cpython", "platform": "win32",
        "binary_sha256": "4" * 64, "coverage_version": "7.17.0", "coverage_module_sha256": "5" * 64}
ORIGIN = {"GITHUB_REPOSITORY": P.SI.REPOSITORY, "GITHUB_REPOSITORY_ID": str(P.SI.REPOSITORY_ID),
          "GITHUB_RUN_ID": "101", "GITHUB_RUN_ATTEMPT": "1", "GITHUB_JOB": "python-static-security",
          "GITHUB_EVENT_NAME": "pull_request", "CQ_COVERAGE_PR_HEAD": HEAD}
LINUX = {"lines": [1, 2, 3], "hit_lines": [1, 2], "excluded": [],
         "branches": [[1, 2], [1, 3]], "hit_branches": [[1, 2]]}
NATIVE = {**LINUX, "hit_lines": [1, 3], "hit_branches": [[1, 3]]}
XML = b'''<?xml version="1.0"?><coverage version="7.17.0" lines-valid="4" lines-covered="3" line-rate="0.75" branches-valid="2" branches-covered="1" branch-rate="0.5"><sources><source>.</source></sources><packages><package name="owned" line-rate="0.75" branch-rate="0.5"><classes><class name="platform_coverage.py" filename="platform_coverage.py" line-rate="0.6667" branch-rate="0.5"><methods/><lines><line number="1" hits="1" branch="true" condition-coverage="50% (1/2)" missing-branches="3"/><line number="2" hits="1"/><line number="3" hits="0"/></lines></class><class name="unchanged.py" filename="unchanged.py" line-rate="1" branch-rate="1"><methods/><lines><line number="9" hits="1"/></lines></class></classes></package></packages></coverage>'''


def official(counter=LINUX):
    lines, hits, excluded, branches, taken = P.validate_counter(counter)
    return P.PC.encoded({"meta": {"format": 3, "version": "7.17.0", "branch_coverage": True, "show_contexts": False},
        "files": {P.TARGET: {"executed_lines": sorted(hits), "missing_lines": sorted(lines - hits),
        "excluded_lines": sorted(excluded), "executed_branches": sorted(map(list, taken)),
        "missing_branches": sorted(map(list, branches - taken)), "summary": {
        "num_statements": len(lines), "covered_lines": len(hits), "missing_lines": len(lines - hits),
        "excluded_lines": len(excluded), "num_branches": len(branches), "covered_branches": len(taken),
        "missing_branches": len(branches - taken)}}}})


def official_all(counter=LINUX):
    report = json.loads(official(counter))
    record = report["files"][P.TARGET]
    report["files"] = {target: copy.deepcopy(record) for target in P.TARGETS}
    return P.PC.encoded(report)


def native_counters(counter=NATIVE):
    return {"version": 2, "files": {target: copy.deepcopy(counter) for target in P.TARGETS}}


def two_class_xml():
    selected = next(P.CLASS.finditer(XML)).group()
    second = selected.replace(b"platform_coverage.py", b"browser_coverage.py")
    return (XML.replace(b"</classes>", second + b"</classes>")
            .replace(b'lines-valid="4"', b'lines-valid="7"')
            .replace(b'lines-covered="3"', b'lines-covered="5"')
            .replace(b'line-rate="0.75"', b'line-rate="0.7143"')
            .replace(b'branches-valid="2"', b'branches-valid="4"')
            .replace(b'branches-covered="1"', b'branches-covered="2"'))


def run(identity, workflow):
    return {"id": int(identity), "repository": {"id": P.SI.REPOSITORY_ID, "full_name": P.SI.REPOSITORY},
            "head_repository": {"id": P.SI.REPOSITORY_ID}, "event": "pull_request", "head_sha": HEAD,
            "path": ".github/workflows/" + workflow + ".yml", "run_attempt": 1,
            "pull_requests": [{"number": 7, "head": {"sha": HEAD}}], "status": "in_progress",
            "created_at": "2026-11-18T00:00:00Z"}


def job():
    return {"id": 303, "name": P.JOB, "status": "completed", "conclusion": "failure",
            "steps": [{"name": P.STEP, "status": "completed", "conclusion": "success"}]}


def zipdata(members):
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", compression=zipfile.ZIP_STORED) as archive:
        for name, content in members.items():
            item = zipfile.ZipInfo(name)
            item.external_attr = (stat.S_IFREG | 0o600) << 16
            archive.writestr(item, content)
    return stream.getvalue()


def evidence():
    metadata = {"version": 2, "kind": "platform-python-counters", "source": SOURCE,
        "tool": TOOL, "origin": ORIGIN, "suites": list(P.TARGETS.values()), "test_exit": 0,
        "test_counts": {"selected": 110, "executed": 101, "skipped": 9},
        "source_before_after_equal": True, "official_json_sha256": "6" * 64,
        "counter_sha256": P.PC.digest(P.PC.encoded(native_counters())), "global_gate_passed": False,
        "production_qualified": False}
    data = zipdata({"coverage.json": P.PC.encoded(native_counters()), "metadata.json": P.PC.encoded(metadata)})
    pin = {"id": "202", "sha256": P.PC.digest(data), "bytes": len(data), "name": P.ARTIFACT + COMMIT}
    sdk, sonar = run("101", "sdk"), run("102", "sonar")
    binding = {"version": 1, "source": SOURCE, "head": HEAD, "pr": "7",
               "sonar": P.SI.run_matches(sonar, HEAD, "7", "sonar"),
               "run": P.SI.run_matches(sdk, HEAD, "7", "sdk"),
               "job": {"id": "303", "name": P.JOB, "status": "completed", "conclusion": "failure"}, "artifact": pin}
    return data, pin, binding, metadata


class CounterContract(unittest.TestCase):
    def test_actual_stdlib_counts_are_bounded_not_static_native_pass_claims(self):
        raw = b"owned test body\r\nRan 110 tests in 0.124s\r\n\r\nOK (skipped=9)\r\n"
        self.assertEqual(P.suite_counts(raw), {"selected": 110, "executed": 101, "skipped": 9})
        self.assertEqual(P.suite_counts(b"Ran 1 test in 0.1s\n\nOK\n"), {"selected": 1, "executed": 1, "skipped": 0})
        for value in (b"PRIVATE_DATA", b"Ran 0 tests in 0.1s\n\nOK\n", b"Ran 1 test in 0.1s\n\nOK (skipped=2)\n"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                P.suite_counts(value)
    def test_both_fixed_native_modules_require_complete_official_inventories(self):
        report = P.normalized_reports(official_all(), native=True)
        self.assertEqual(report, {"version": 2, "files": {target: LINUX for target in P.TARGETS}})
        for target in P.TARGETS:
            value = json.loads(official_all())
            del value["files"][target]
            with self.subTest(target=target), self.assertRaises(ValueError):
                P.normalized_reports(P.PC.encoded(value), native=True)
    def test_partial_rates_never_round_to_a_strict_pass_or_zero(self):
        self.assertEqual(P.rate(19999, 20000), "0.9999")
        self.assertEqual(P.rate(1, 20000), "0.0001")
        self.assertEqual(P.rate(0, 2), "0")
        self.assertEqual(P.rate(0, 0), "1")
        self.assertEqual(P.rate(2, 2), "1")
        with self.assertRaises(ValueError):
            P.rate(3, 2)

    def test_official_json_and_observed_native_union_preserve_zeros(self):
        self.assertEqual(P.normalized_report(official()), LINUX)
        self.assertEqual(P.normalized_report(official(NATIVE), native=True), NATIVE)
        union = P.merge_counters(LINUX, NATIVE)
        self.assertEqual(union["hit_lines"], [1, 2, 3])
        self.assertEqual(union["hit_branches"], [[1, 2], [1, 3]])
        self.assertEqual(P.merge_counters(LINUX, LINUX), LINUX)
        self.assertEqual(LINUX["hit_lines"], [1, 2])

    def test_complete_coordinate_inventory_mismatch_refuses_patch_aliases(self):
        for key, value in (("lines", [1, 2, 3, 4]), ("branches", [[1, 2], [1, 4]]), ("excluded", [9])):
            candidate = copy.deepcopy(NATIVE)
            candidate[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                P.merge_counters(LINUX, candidate)

    def test_report_requires_pinned_branch_format_and_unique_fixed_path(self):
        original = json.loads(official())
        for key, value in (("format", True), ("version", "7.16.0"), ("branch_coverage", False), ("show_contexts", True)):
            data = copy.deepcopy(original)
            data["meta"][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                P.normalized_report(P.PC.encoded(data))
        data = copy.deepcopy(original)
        data["files"][P.TARGET.replace("/", "\\")] = data["files"][P.TARGET]
        with self.assertRaises(ValueError):
            P.normalized_report(P.PC.encoded(data))
        data = copy.deepcopy(original)
        data["files"]["other.py"] = data["files"][P.TARGET]
        with self.assertRaises(ValueError):
            P.normalized_report(P.PC.encoded(data), native=True)
        self.assertEqual(P.normalized_report(P.PC.encoded(data)), LINUX)

    def test_summary_cannot_hide_missing_statement_or_condition(self):
        original = json.loads(official())
        for key in original["files"][P.TARGET]["summary"]:
            data = copy.deepcopy(original)
            data["files"][P.TARGET]["summary"][key] += 1
            with self.subTest(key=key), self.assertRaises(ValueError):
                P.normalized_report(P.PC.encoded(data))

    def test_coordinate_bounds_disjointness_and_function_exit_destinations(self):
        for value in (None, [True], [0], [-1], [1, 1], [100001]):
            with self.subTest(value=value), self.assertRaises(ValueError):
                P.integers(value)
        for value in (None, [[1]], [[1, 0]], [[True, 2]], [[1, 2], [1, 2]]):
            with self.subTest(value=value), self.assertRaises(ValueError):
                P.pairs(value)
        self.assertEqual(P.pairs([[1, -1]]), {(1, -1)})
        for key, value in (("hit_lines", [9]), ("hit_branches", [[1, 9]]), ("branches", [[1, 2]])):
            data = copy.deepcopy(LINUX)
            data[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                P.validate_counter(data)

    def test_xml_preserves_every_other_class_and_original_input(self):
        before = bytes(XML)
        result, count = P.merge_xml(XML, LINUX, P.merge_counters(LINUX, NATIVE))
        self.assertTrue(count["passed"])
        self.assertEqual(count, {"lines": 4, "covered_lines": 4, "branches": 2, "covered_branches": 2, "passed": True})
        self.assertEqual(list(P.CLASS.finditer(result))[1].group(), list(P.CLASS.finditer(XML))[1].group())
        self.assertEqual(XML, before)
        self.assertIn(b'condition-coverage="100% (2/2)"', result)
        self.assertNotIn(b'missing-branches=', result)
        unchanged, count = P.merge_xml(XML, LINUX, LINUX)
        self.assertFalse(count["passed"])
        self.assertIn(b'hits="0"', unchanged)

    def test_xml_rejects_entities_missing_duplicate_or_inconsistent_class(self):
        selected = next(P.CLASS.finditer(XML)).group()
        for data in (b'<!DOCTYPE coverage>' + XML, XML.replace(selected, b''),
                     XML.replace(selected, selected + selected), XML.replace(b'number="3"', b'number="4"'),
                     XML.replace(b'condition-coverage="50% (1/2)"', b'condition-coverage="100% (2/2)"')):
            with self.subTest(data=data[:40]), self.assertRaises(ValueError):
                P.merge_xml(data, LINUX, NATIVE)

    def test_archive_is_provider_hash_bound_exact_two_regular_members(self):
        data, pin, _, _ = evidence()
        self.assertEqual(set(P.archive_bytes(data, pin)), {"coverage.json", "metadata.json"})
        for changed in ({**pin, "bytes": len(data) + 1}, {**pin, "sha256": "0" * 64}):
            with self.assertRaises(ValueError):
                P.archive_bytes(data, changed)
        for names in ({"../coverage.json": b'{}', "metadata.json": b'{}'},
                      {"coverage.json": b'{}', "metadata.json": b'{}', "extra": b'{}'}):
            data = zipdata(names)
            with self.assertRaises(ValueError):
                P.archive_bytes(data, {"bytes": len(data), "sha256": P.PC.digest(data)})


class ProducerContract(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.source = self.root / "source"
        self.source.mkdir()
        self.output = self.root / "output"

    def test_begin_and_retain_measure_before_after_and_only_two_public_members(self):
        with mock.patch.object(P, "source_state", return_value=SOURCE), mock.patch.object(P, "python_tool", return_value=TOOL), mock.patch.object(P, "origin", return_value=ORIGIN):
            P.begin(self.source, COMMIT, self.output)
            self.assertIn(b'branch = True', (self.output / "coverage.ini").read_bytes())
            (self.output / "raw" / "official.json").write_bytes(official_all(NATIVE))
            (self.output / "raw" / "test.log").write_bytes(b"Ran 110 tests in 0.1s\n\nOK (skipped=9)\n")
            P.retain(self.source, COMMIT, self.output, 0)
        self.assertEqual({p.name for p in (self.output / "retained").iterdir()}, {"coverage.json", "metadata.json"})
        metadata = json.loads((self.output / "retained" / "metadata.json").read_bytes())
        self.assertEqual(metadata["test_exit"], 0)
        self.assertFalse(metadata["production_qualified"])

    def test_source_state_binds_raw_tracked_source_and_declared_lock_bytes(self):
        for name in P.BINDINGS:
            path = self.source / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"owned-public-source\n")
        blob = hashlib.sha1(b"blob 20\0owned-public-source\n").hexdigest()
        with mock.patch.object(P.PC, "source_identity", return_value=(SOURCE["identity"], set())), mock.patch.object(P.PC, "git", return_value=(blob + "\n").encode()):
            state = P.source_state(self.source, COMMIT)
        self.assertEqual(state["platform_blob"], blob)
        self.assertEqual(state["module_blobs"], {target: blob for target in P.TARGETS})
        self.assertEqual(state["bindings"], {name: P.PC.digest(b"owned-public-source\n") for name in P.BINDINGS})

    def test_tool_identity_hashes_owned_files_and_records_actual_platform(self):
        binary, module = self.root / "python", self.root / "coverage.py"
        binary.write_bytes(b"not-executed-owned-binary")
        module.write_bytes(b"owned-package")
        package = SimpleNamespace(__version__="7.17.0", __file__=str(module))
        selected = SimpleNamespace(platform=P.sys.platform, implementation=SimpleNamespace(name="cpython"),
                                   executable=str(binary), version_info=(3, 14, 10))
        with mock.patch.dict(P.sys.modules, {"coverage": package}), mock.patch.object(P, "sys", selected):
            measured = P.python_tool()
            self.assertEqual(measured["platform"], P.sys.platform)
            self.assertEqual(measured["binary_sha256"], P.PC.digest(binary.read_bytes()))
            package.__version__ = "7.16.0"
            with self.assertRaises(ValueError):
                P.python_tool()

    def test_retain_source_or_interpreter_change_and_failed_test_cannot_be_admitted(self):
        with mock.patch.object(P, "source_state", return_value=SOURCE), mock.patch.object(P, "python_tool", return_value=TOOL), mock.patch.object(P, "origin", return_value=ORIGIN):
            P.begin(self.source, COMMIT, self.output)
            (self.output / "raw" / "official.json").write_bytes(official_all())
            (self.output / "raw" / "test.log").write_bytes(b"Ran 110 tests in 0.1s\n\nFAILED (failures=1, skipped=9)\n")
            with mock.patch.object(P, "source_state", return_value={**SOURCE, "platform_blob": "0" * 40}), self.assertRaises(ValueError):
                P.retain(self.source, COMMIT, self.output, 0)
            P.retain(self.source, COMMIT, self.output, 1)
        self.assertEqual(json.loads((self.output / "retained" / "metadata.json").read_bytes())["test_exit"], 1)

    def test_origin_requires_exact_original_windows_job_not_an_ambient_account(self):
        with mock.patch.dict(os.environ, ORIGIN, clear=True):
            self.assertEqual(P.origin(), ORIGIN)
            for key, value in (("GITHUB_RUN_ATTEMPT", "2"), ("GITHUB_JOB", "other"), ("GITHUB_EVENT_NAME", "push"), ("CQ_COVERAGE_PR_HEAD", "bad")):
                with self.subTest(key=key), mock.patch.dict(os.environ, {key: value}), self.assertRaises(ValueError):
                    P.origin()

    def test_linux_measurement_identity_is_fenced_without_inventing_native_hits(self):
        linux = {**TOOL, "platform": "linux", "python": "3.14.3"}
        with mock.patch.object(P, "source_state", return_value=SOURCE), mock.patch.object(P, "python_tool", return_value=linux):
            P.linux_begin(self.source, COMMIT, self.output)
            P.linux_check(self.source, COMMIT, self.output)
            with mock.patch.object(P, "python_tool", return_value={**linux, "binary_sha256": "0" * 64}), self.assertRaises(ValueError):
                P.linux_check(self.source, COMMIT, self.output)
        self.assertEqual(json.loads((self.output / "linux-tool.json").read_bytes())["tool"]["python"], "3.14.3")
        self.assertEqual((self.output / "linux-tool.json").read_bytes(), (self.output / "linux-tool-verified.json").read_bytes())


class AdmissionContract(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.source = self.root / "source"
        self.source.mkdir()
        self.output = self.root / "output"
        self.data, self.pin, self.binding, self.metadata = evidence()
        self.archives = self.root / "archives"
        self.archives.mkdir()
        (self.archives / "owned.zip").write_bytes(self.data)
        self.linux_json, self.linux_xml = self.root / "linux.json", self.root / "linux.xml"
        self.linux_json.write_bytes(official_all())
        self.original_xml = two_class_xml()
        self.linux_xml.write_bytes(self.original_xml)
        self.linux_tool = self.root / "linux-tool.json"
        self.linux_tool.write_bytes(P.PC.encoded({"source": SOURCE, "tool": {**TOOL, "platform": "linux", "python": "3.14.3"}}))
        self.linux_tool.chmod(0o600)

    def api(self, path):
        if path.endswith('/runs/101'):
            return run("101", "sdk")
        if path.endswith('/runs/102'):
            return run("102", "sonar")
        if '/workflows/sdk.yml/runs?' in path:
            return {"total_count": 1, "workflow_runs": [run("101", "sdk")]}
        if '/jobs?' in path:
            return {"total_count": 1, "jobs": [job()]}
        artifact = {"id": 202, "expired": False, "digest": "sha256:" + self.pin["sha256"],
                    "size_in_bytes": self.pin["bytes"], "name": self.pin["name"], "workflow_run": {"id": 101}}
        if path.endswith('/artifacts/202'):
            return artifact
        if '/artifacts?' in path:
            return {"total_count": 1, "artifacts": [artifact]}
        raise AssertionError("unexpected metadata route")

    def consume(self, binding=None):
        raw = P.PC.encoded(self.binding if binding is None else binding)
        with mock.patch.object(P, "source_state", return_value=SOURCE):
            return P.consume(self.api, self.source, COMMIT, raw, P.PC.digest(raw), self.archives,
                             self.linux_json, self.linux_xml, self.linux_tool, self.output)

    def test_completed_selected_step_not_overall_sdk_pass_controls_admission(self):
        sleeps = []
        binding = P.discover(self.api, SOURCE, HEAD, "7", "102", sleeps.append)
        self.assertEqual(binding, self.binding)
        self.assertEqual(sleeps, [])
        summary = self.consume()
        self.assertTrue(summary["passed"])
        self.assertEqual(self.linux_xml.read_bytes(), self.original_xml)
        receipt = json.loads((self.output / "python-composite.json").read_bytes())
        self.assertEqual(receipt["original_sdk_job"]["admission"]["conclusion"], "failure")
        self.assertFalse(receipt["raw_linux_status_promoted"])

    def test_one_native_class_cannot_hide_other_classes_uncovered_coordinates(self):
        native = native_counters()
        native["files"][".github/scripts/browser_coverage.py"] = copy.deepcopy(LINUX)
        self.metadata["counter_sha256"] = P.PC.digest(P.PC.encoded(native))
        data = zipdata({"coverage.json": P.PC.encoded(native), "metadata.json": P.PC.encoded(self.metadata)})
        self.pin = {**self.pin, "bytes": len(data), "sha256": P.PC.digest(data)}
        self.binding["artifact"] = self.pin
        (self.archives / "owned.zip").write_bytes(data)
        summary = self.consume()
        self.assertFalse(summary["passed"])
        self.assertEqual((summary["lines"], summary["covered_lines"]), (7, 6))
        before = list(P.CLASS.finditer(self.original_xml))[1].group()
        after = list(P.CLASS.finditer((self.output / "python-coverage.xml").read_bytes()))[1].group()
        self.assertEqual(before, after)

    def test_pending_selected_job_with_present_archive_waits_before_binding(self):
        observations, sleeps = [], []
        def api(path):
            result = self.api(path)
            if '/jobs?' in path:
                observations.append(1)
                if len(observations) == 1:
                    result["jobs"][0]["status"] = "in_progress"
                    result["jobs"][0]["conclusion"] = None
                    result["jobs"][0]["steps"][0]["status"] = "in_progress"
            return result
        self.assertEqual(P.discover(api, SOURCE, HEAD, "7", "102", sleeps.append), self.binding)
        self.assertEqual(sleeps, [10])

    def test_wrong_step_failure_duplicate_job_or_artifact_never_qualifies(self):
        for changed in ({**job(), "steps": []}, {**job(), "steps": [{"name": P.STEP, "status": "completed", "conclusion": "failure"}]}):
            with self.subTest(changed=changed), self.assertRaises(ValueError):
                P.selected_job([changed])
        with self.assertRaises(ValueError):
            P.selected_job([job(), job()])
        self.assertIsNone(P.selected_job([]))
        self.assertIsNone(P.selected_job([{**job(), "status": "queued"}]))

    def test_completed_producer_can_be_bound_while_unrelated_sdk_steps_run(self):
        pending = {**job(), "status": "in_progress", "conclusion": None}
        observation = P.selected_job([pending])
        self.assertEqual(observation["status"], "in_progress")
        self.binding["job"] = observation
        self.assertTrue(self.consume()["passed"])
        receipt = json.loads((self.output / "python-composite.json").read_bytes())
        self.assertEqual(receipt["original_sdk_job"]["discovery"], observation)
        self.assertEqual(receipt["original_sdk_job"]["admission"]["conclusion"], "failure")

    def test_original_identity_source_and_provider_artifact_must_still_match(self):
        for field, value in (("run", {**self.binding["run"], "created_at": "2026-11-18T00:00:01Z"}),
                             ("source", {**SOURCE, "platform_blob": "0" * 40}),
                             ("artifact", {**self.pin, "sha256": "0" * 64})):
            binding = copy.deepcopy(self.binding)
            binding[field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                self.consume(binding)
        self.assertFalse(self.output.exists())

    def test_failed_test_wrong_tool_or_promoted_metadata_refuses_before_write(self):
        for field, value in (("test_exit", 1), ("source_before_after_equal", False), ("global_gate_passed", True),
                             ("counter_sha256", "0" * 64), ("suites", ["unselected.py"]),
                             ("tool", {**TOOL, "platform": "linux"})):
            metadata = {**self.metadata, field: value}
            data = zipdata({"coverage.json": P.PC.encoded(native_counters()), "metadata.json": P.PC.encoded(metadata)})
            self.data = data
            self.pin = {**self.pin, "bytes": len(data), "sha256": P.PC.digest(data)}
            self.binding["artifact"] = self.pin
            (self.archives / "owned.zip").write_bytes(data)
            with self.subTest(field=field), self.assertRaises(ValueError):
                self.consume()
        self.assertFalse(self.output.exists())

    def test_cli_discover_and_consume_use_one_api_closure_and_fixed_output(self):
        raw = P.PC.encoded(self.binding)
        binding = self.root / "binding.json"
        binding.write_bytes(raw)
        binding.chmod(0o600)
        args = ["consume", "--source", str(self.source), "--expected-commit", COMMIT, "--output", str(self.output),
                "--binding", str(binding), "--binding-sha256", P.PC.digest(raw), "--archives", str(self.archives),
                "--linux-json", str(self.linux_json), "--linux-xml", str(self.linux_xml), "--linux-tool", str(self.linux_tool)]
        with mock.patch.object(P.SI, "GitHubRead", return_value=self.api) as factory, mock.patch.object(P, "source_state", return_value=SOURCE):
            self.assertEqual(P.main(args), 0)
            self.assertEqual(factory.call_count, 1)
        with mock.patch.object(P.SI, "GitHubRead", side_effect=ValueError("SYNTHETIC_PRIVATE")), contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(P.main(args), 2)
        self.assertEqual(output.getvalue(), "native-python-coverage-refused\n")

    def test_cli_producer_and_linux_fences_forward_only_explicit_arguments(self):
        for command, function, additional in (("begin", "begin", []), ("retain", "retain", ["--test-exit", "0"]),
                                               ("linux-begin", "linux_begin", []), ("linux-check", "linux_check", [])):
            args = [command, "--source", str(self.source), "--expected-commit", COMMIT, "--output", str(self.output), *additional]
            with self.subTest(command=command), mock.patch.object(P, function) as target, mock.patch.object(P.SI, "GitHubRead", side_effect=AssertionError("producer must not access Actions")):
                self.assertEqual(P.main(args), 0)
                expected = (self.source, COMMIT, self.output, 0) if command == "retain" else (self.source, COMMIT, self.output)
                target.assert_called_once_with(*expected)

    @unittest.skipUnless(hasattr(os, "getuid"), "Only the existing Ubuntu Sonar Actions-output fence is POSIX")
    def test_cli_discovery_writes_binding_digest_only_after_same_source_recheck(self):
        destination = self.root / "discovered"
        step = self.root / "step-output"
        step.write_bytes(b"")
        with mock.patch.object(P.SI, "GitHubRead", return_value=self.api), mock.patch.object(P, "source_state", return_value=SOURCE):
            result = P.main(["discover", "--source", str(self.source), "--expected-commit", COMMIT,
                             "--output", str(destination), "--head", HEAD, "--pr", "7", "--sonar-run", "102", "--step-output", str(step)])
        self.assertEqual(result, 0)
        raw = (destination / "binding.json").read_bytes()
        self.assertEqual(json.loads(raw), self.binding)
        self.assertEqual(step.read_text(), "ready=true\nrun_id=101\nartifact_id=202\nbinding_sha256=" + P.PC.digest(raw) + "\n")


if __name__ == "__main__":
    unittest.main()
