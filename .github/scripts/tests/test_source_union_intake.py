"""Owned metadata/ZIP/coordinate contracts; no GitHub/network/Git/test execution."""
from __future__ import annotations

import copy
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock
import urllib.error
import zipfile

PATH = Path(__file__).parents[1] / "source_union_intake.py"
SPEC = importlib.util.spec_from_file_location("cq_source_union_contract", PATH)
assert SPEC is not None and SPEC.loader is not None
INTAKE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(INTAKE)
PC = INTAKE.PC
SOURCE = {"commit": "a" * 40, "tree": "b" * 40, "tracked_clean": True,
          "locks": {"go.mod": "c" * 64, "go.sum": "d" * 64}}
HEAD = "e" * 40
PROFILE = b"mode: atomic\ngithub.com/maldous/CircularQuay/internal/owned.go:1.1,2.1 2 1\n"
TRACKED = {"internal/owned.go"}


def run(identifier, workflow="source", status="completed"):
    return {"id": identifier, "repository": {"id": INTAKE.REPOSITORY_ID, "full_name": INTAKE.REPOSITORY},
            "head_repository": {"id": INTAKE.REPOSITORY_ID}, "event": "pull_request", "head_sha": HEAD,
            "path": ".github/workflows/" + workflow + ".yml", "run_attempt": 1,
            "pull_requests": [{"number": 1, "head": {"sha": HEAD}}],
            "created_at": "2026-11-17T23:40:00Z", "status": status, "conclusion": "failure"}


def artifact(identifier, name, digest="f" * 64, size=1):
    return {"id": identifier, "name": name, "expired": False, "size_in_bytes": size,
            "digest": "sha256:" + digest, "workflow_run": {"id": 10}}


def fixture():
    checks = [{"name": n, "required": True, "status": "passed"} for n in
              ("format", "unit_tests", "go_vet", "go_toolchain", "source_revision", "source_identity_before_checks", "source_identity_after_checks")]
    checks.append({"name": "coverage", "required": True, "status": "failed"})
    original = {"version": 1, "profile": "source", "commit": SOURCE["commit"], "passed": False,
                "production_qualified": False, "checks": checks,
                "source": {"commit": SOURCE["commit"], "tree": SOURCE["tree"], "manifest_sha256": "1" * 64},
                "original_report_sha256": "2" * 64, "coverage_sha256": PC.digest(PROFILE), "retention_only": True,
                "coverage_semantics": "Original audit result; not recomputed or promoted by retention",
                "go_version": PC.GO_VERSION, "platform": "linux/amd64", "go_tool_sha256": "3" * 64}
    entries, api_artifacts = {}, {}
    for index, (role, job) in enumerate(PC.ROLES.items(), 100):
        metadata = (json.dumps(original, indent=2) + "\n").encode() if role == "source" else b"synthetic metadata"
        descriptor = {"version": 1, "role": role, "source": copy.deepcopy(SOURCE),
                      "toolchain": {"version": PC.GO_VERSION, "platform": sorted(PC.PLATFORMS[role])[0], "sha256": "3" * 64, "bytes": 100},
                      "producer": {"repository": INTAKE.REPOSITORY, "repository_id": str(INTAKE.REPOSITORY_ID), "run_id": "10", "attempt": "1", "job": job,
                                   "event": "pull_request", "pr_head": HEAD, "step": "platform_counters"},
                      "coverage": {"mode": "atomic", "bytes": len(PROFILE), "sha256": PC.digest(PROFILE)},
                      "metadata_sha256": PC.digest(metadata), "metadata_bytes": len(metadata)}
        name = ("source-audit-" if role == "source" else "platform-go-" + role + "-") + SOURCE["commit"]
        api_artifacts[str(index)] = artifact(index, name)
        entries[role] = {"descriptor": descriptor, "artifact_id": str(index), "archive_sha256": "f" * 64}
    outcomes = {job: "failure" for job in PC.ROLES.values()}
    pins = {"version": 1, "kind": "same-run-platform-coverage-pins", "source": SOURCE,
            "entries": entries, "original_job_results": outcomes}
    _, coverage, gaps = PC.merged_profiles([PC.profile(PROFILE, TRACKED)])
    value = {"version": 1, "kind": "same-revision-platform-source-coverage", "source": copy.deepcopy(SOURCE),
             "pins_sha256": PC.digest(PC.encoded(pins)), "producers": entries, "coverage": coverage,
             "coverage_sha256": PC.digest(PROFILE), "original_source_report_unchanged": {"passed": False, "checks": checks,
             "original_report_sha256": "2" * 64, "production_qualified": False}, "original_source_metadata_unchanged": original,
             "original_job_results_unchanged": outcomes,
             "denominator": "all retained Source and Linux/macOS/Windows coordinate sets; OS-only zeros included",
             "global_gate_passed": False, "production_qualified": False, "source_report_promoted": False}
    binding = {"version": 1, "repository": INTAKE.REPOSITORY, "repository_id": INTAKE.REPOSITORY_ID,
               "source": copy.deepcopy(SOURCE), "head": HEAD, "pr": "1", "sonar": {"id": "20", "attempt": 1, "created_at": "2026-11-17T23:40:00Z"},
               "run": {"id": "10", "attempt": 1, "created_at": "2026-11-17T23:40:00Z"},
               "original_jobs": {name: "failure" for name in [*INTAKE.JOBS.values(), INTAKE.UNION_JOB]}}
    return value, binding, api_artifacts, {"coverage": coverage, "gaps": gaps}


def archive(value, gaps, profile=PROFILE, extra=None):
    data = io.BytesIO()
    with zipfile.ZipFile(data, "w", compression=zipfile.ZIP_DEFLATED) as target:
        for name, raw in {"coverage.out": profile, "platform-union.json": PC.encoded(value), "gaps.json": PC.encoded(gaps)}.items():
            target.writestr(name, raw)
        if extra is not None:
            target.writestr(extra, b"unexpected")
    return data.getvalue()


def bind_archive(binding, raw):
    binding["artifact"] = {"id": "200", "name": "platform-go-union-" + SOURCE["commit"], "bytes": len(raw), "sha256": PC.digest(raw)}


class SourceUnionIntakeContracts(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.source = self.root / "source"
        self.source.mkdir()
        self.value, self.binding, self.artifacts, self.gaps = fixture()
        self.raw = archive(self.value, self.gaps)
        bind_archive(self.binding, self.raw)
        self.api_calls = []

    def api(self, path):
        self.api_calls.append(path)
        if path == INTAKE.PREFIX + "/actions/runs/10":
            return run(10)
        identifier = path.rsplit("/", 1)[-1]
        if identifier == "200":
            a = self.binding["artifact"]
            return artifact(200, a["name"], a["sha256"], a["bytes"])
        return copy.deepcopy(self.artifacts[identifier])

    def check(self, value=None, gaps=None, extra=None):
        data = archive(value or self.value, gaps or self.gaps, extra=extra)
        binding = copy.deepcopy(self.binding)
        bind_archive(binding, data)
        return INTAKE.checked_union(data, binding, TRACKED, self.api)

    def test_real_coordinate_union_preserves_original_failed_source_and_provider_pins(self):
        blocks, value = INTAKE.checked_union(self.raw, self.binding, TRACKED, self.api)
        self.assertEqual(PC.profile(PROFILE, TRACKED), blocks)
        self.assertFalse(value["original_source_report_unchanged"]["passed"])
        self.assertFalse(value["global_gate_passed"])
        self.assertEqual(4, len(self.api_calls))
        self.assertNotEqual(PC.digest(PC.encoded(value["original_source_metadata_unchanged"])),
                            value["producers"]["source"]["descriptor"]["metadata_sha256"])
        self.assertTrue(all(path.startswith(INTAKE.PREFIX + "/actions/artifacts/") for path in self.api_calls))

    def test_archive_member_mode_identity_digest_and_bounds_fail_closed(self):
        for extra in ("../escaped", "coverage.out", "unexpected"):
            with self.subTest(extra=extra), self.assertRaises((INTAKE.Refused, PC.InvalidEvidence)):
                self.check(extra=extra)
        binding = copy.deepcopy(self.binding)
        binding["artifact"]["sha256"] = "0" * 64
        with self.assertRaises(INTAKE.Refused):
            INTAKE.checked_union(self.raw, binding, TRACKED, self.api)
        with self.assertRaises(INTAKE.Refused):
            INTAKE.checked_union(b"", self.binding, TRACKED, self.api)

    def test_same_tree_different_commit_or_locks_is_not_an_alias(self):
        for key, changed in (("commit", "f" * 40), ("tree", "f" * 40), ("locks", {"go.mod": "f" * 64, "go.sum": "d" * 64})):
            value = copy.deepcopy(self.value)
            value["source"][key] = changed
            with self.subTest(key=key), self.assertRaises(INTAKE.Refused):
                self.check(value)

    def test_no_producer_or_job_failure_is_relabelled_and_no_raw_metadata_hash_is_reconstructed(self):
        self.value["original_source_metadata_unchanged"]["checks"][0]["status"] = "failed"
        with self.assertRaises(PC.InvalidEvidence):
            self.check()
        self.value, _, _, _ = fixture()
        self.value["original_source_report_unchanged"]["passed"] = True
        with self.assertRaises(INTAKE.Refused):
            self.check()
        self.value, _, _, _ = fixture()
        self.value["production_qualified"] = True
        with self.assertRaises(INTAKE.Refused):
            self.check()

    def test_provider_metadata_not_self_declared_hash_authenticates_each_producer(self):
        for field, changed in (("digest", "sha256:" + "0" * 64), ("expired", True), ("workflow_run", {"id": 11}), ("name", "different-source")):
            original = copy.deepcopy(self.artifacts["100"])
            self.artifacts["100"][field] = changed
            with self.subTest(field=field), self.assertRaises(INTAKE.Refused):
                self.check()
            self.artifacts["100"] = original

    def test_counts_coordinates_gap_inventory_and_hash_must_all_reproduce(self):
        for field, changed in (("coverage_sha256", "0" * 64), ("coverage", {}), ("pins_sha256", "0" * 64)):
            value = copy.deepcopy(self.value)
            value[field] = changed
            with self.subTest(field=field), self.assertRaises(INTAKE.Refused):
                self.check(value)
        with self.assertRaises(INTAKE.Refused):
            self.check(gaps={"coverage": self.gaps["coverage"], "gaps": [{"fake": True}]})

    def test_consume_keeps_linux_hits_and_outputs_only_fixed_public_source_provenance(self):
        linux = self.root / "linux.out"
        linux.write_bytes(PROFILE.replace(b" 2 1\n", b" 2 4\n"))
        archives = self.root / "archives"
        archives.mkdir()
        (archives / "200.zip").write_bytes(self.raw)
        binding = PC.encoded(self.binding)
        output = self.root / "output"
        with mock.patch.object(PC, "source_identity", return_value=(SOURCE, TRACKED)) as identity:
            receipt = INTAKE.consume(self.api, self.source, SOURCE["commit"], binding, PC.digest(binding), archives, linux, output)
        self.assertEqual(2, identity.call_count)
        self.assertEqual(linux.read_bytes(), (output / "coverage.out").read_bytes())
        self.assertEqual(PC.digest(linux.read_bytes()), receipt["linux_sha256"])
        self.assertFalse(receipt["source_report_promoted"])
        self.assertNotIn(str(self.root), (output / "source-union-intake.json").read_text())
        self.assertEqual({"coverage.out", "source-union-intake.json"}, {p.name for p in output.iterdir()})

    def test_consume_refuses_changed_binding_source_and_provider_before_output(self):
        binding = PC.encoded(self.binding)
        with self.assertRaises(INTAKE.Refused):
            INTAKE.consume(self.api, self.source, SOURCE["commit"], binding, "0" * 64, self.root, self.root, self.root / "output")
        with mock.patch.object(PC, "source_identity", return_value=({**SOURCE, "commit": "f" * 40}, TRACKED)), self.assertRaises(INTAKE.Refused):
            INTAKE.consume(self.api, self.source, SOURCE["commit"], binding, PC.digest(binding), self.root, self.root, self.root / "output")
        self.assertFalse((self.root / "output").exists())

    def discovery_api(self, path):
        if path.endswith("/runs/20"):
            return run(20, "sonar")
        if "/workflows/source.yml/runs?" in path:
            return {"total_count": 1, "workflow_runs": [run(10)]}
        if "/artifacts?" in path:
            outer = self.binding["artifact"]
            return {"total_count": 1, "artifacts": [artifact(200, outer["name"], outer["sha256"], outer["bytes"])]}
        if "/jobs?" in path:
            names = [*INTAKE.JOBS.values(), INTAKE.UNION_JOB]
            return {"total_count": len(names), "jobs": [{"id": i, "name": name, "status": "completed", "conclusion": "failure"} for i, name in enumerate(names, 1)]}
        raise AssertionError("unexpected fixed API path")

    def test_discover_binds_one_original_same_event_current_source_artifact_without_scan(self):
        sleep = mock.Mock(side_effect=AssertionError("terminal source does not poll"))
        binding = INTAKE.discover(self.discovery_api, SOURCE, HEAD, "1", "20", sleep=sleep)
        self.assertEqual(self.binding, binding)
        sleep.assert_not_called()

    def test_completed_producers_are_ready_even_while_unrelated_source_jobs_still_run(self):
        def progressing(path):
            result = self.discovery_api(path)
            if "/workflows/" in path:
                result["workflow_runs"][0]["status"] = "in_progress"
                result["workflow_runs"][0]["conclusion"] = None
            return result
        sleep = mock.Mock(side_effect=AssertionError("completed producers do not wait for unrelated jobs"))
        self.assertEqual(self.binding, INTAKE.discover(progressing, SOURCE, HEAD, "1", "20", sleep=sleep))
        sleep.assert_not_called()

    def test_pending_producers_wait_without_readiness_and_later_exact_artifact_is_admitted(self):
        for pending in ["missing-archive", *INTAKE.JOBS.values(), INTAKE.UNION_JOB]:
            with self.subTest(pending=pending):
                observations = 0
                pending_seen = []
                def progressing(path):
                    nonlocal observations
                    result = self.discovery_api(path)
                    if "/workflows/" in path:
                        observations += 1
                        result["workflow_runs"][0]["status"] = "in_progress"
                        result["workflow_runs"][0]["conclusion"] = None
                    if observations == 1 and "/artifacts?" in path and pending == "missing-archive":
                        pending_seen.append(pending)
                        return {"total_count": 0, "artifacts": []}
                    if observations == 1 and "/jobs?" in path and pending != "missing-archive":
                        for job in result["jobs"]:
                            if job["name"] == pending:
                                job["status"], job["conclusion"] = "in_progress", None
                                pending_seen.append(pending)
                    return result
                def wait(seconds):
                    self.assertEqual(20, seconds)
                    self.assertEqual(1, observations)
                    self.assertEqual([pending], pending_seen)
                sleep = mock.Mock(side_effect=wait)
                self.assertEqual(self.binding, INTAKE.discover(progressing, SOURCE, HEAD, "1", "20", sleep=sleep))
                self.assertEqual(2, observations)
                sleep.assert_called_once_with(20)

    def test_discovery_wait_is_bounded_and_ambiguous_originals_refuse(self):
        def pending(path):
            if "/workflows/" in path:
                return {"total_count": 0, "workflow_runs": []}
            return self.discovery_api(path)
        sleep = mock.Mock()
        with self.assertRaisesRegex(INTAKE.Refused, "original-not-terminal"):
            INTAKE.discover(pending, SOURCE, HEAD, "1", "20", sleep=sleep)
        self.assertEqual(39, sleep.call_count)
        def ambiguous(path):
            if "/workflows/" in path:
                return {"total_count": 2, "workflow_runs": [run(10), run(11)]}
            return self.discovery_api(path)
        with self.assertRaisesRegex(INTAKE.Refused, "ambiguous"):
            INTAKE.discover(ambiguous, SOURCE, HEAD, "1", "20", sleep=mock.Mock())

    def test_wrong_repository_event_head_attempt_pr_and_workflow_are_refused(self):
        for field, value in (("repository", {"id": 1}), ("head_repository", {"id": 1}), ("event", "push"),
                             ("head_sha", "f" * 40), ("run_attempt", 2), ("path", ".github/workflows/untrusted.yml"), ("pull_requests", [])):
            row = run(10)
            row[field] = value
            with self.subTest(field=field), self.assertRaises(INTAKE.Refused):
                INTAKE.run_matches(row, HEAD, "1", "source")

    def test_pagination_must_be_complete_stable_unique_and_bounded(self):
        rows = [{"id": i} for i in range(1, 102)]
        def api(path):
            return {"total_count": 101, "jobs": rows[:100] if path.endswith("page=1") else rows[100:]}
        self.assertEqual(rows, INTAKE.paged(api, INTAKE.PREFIX + "/actions/runs/10/jobs", "jobs"))
        with self.assertRaises(INTAKE.Refused):
            INTAKE.paged(lambda _: {"total_count": 501, "jobs": []}, "/path", "jobs")
        with self.assertRaises(INTAKE.Refused):
            INTAKE.paged(lambda _: {"total_count": 2, "jobs": [{"id": 1}, {"id": 1}]}, "/path", "jobs")

    def test_api_get_has_fixed_host_no_redirect_token_leak_and_bounded_body(self):
        client = INTAKE.GitHubRead("synthetic-token", 999)
        response = mock.MagicMock()
        response.__enter__.return_value = response
        response.status = 200
        response.read.return_value = b'{"owned":true}'
        client.opener = mock.Mock()
        client.opener.open.return_value = response
        with mock.patch.object(INTAKE.time, "monotonic", return_value=1):
            self.assertEqual({"owned": True}, client(INTAKE.PREFIX + "/actions/runs/20"))
        request = client.opener.open.call_args.args[0]
        self.assertEqual("GET", request.get_method())
        self.assertEqual("https://api.github.com" + INTAKE.PREFIX + "/actions/runs/20", request.full_url)
        self.assertEqual("Bearer synthetic-token", request.get_header("Authorization"))
        response.read.assert_called_once_with(INTAKE.MAX_JSON + 1)
        with self.assertRaises(INTAKE.Refused):
            client("https://other.invalid/")
        with self.assertRaises(INTAKE.Refused):
            INTAKE.NoRedirect().redirect_request(None, None, 302, "", {}, "https://other.invalid/")
        with mock.patch.object(INTAKE.time, "monotonic", return_value=1000), self.assertRaises(INTAKE.Refused):
            client(INTAKE.PREFIX + "/actions/runs/20")

    @unittest.skipUnless(hasattr(os, "getuid"), "Only this Linux workflow uid-ownership CLI oracle requires POSIX")
    def test_cli_discovery_and_consumption_use_only_owned_outputs_and_fake_api(self):
        step = self.root / "step-output"
        step.write_bytes(b"")
        step.chmod(0o600)
        location = self.root / "binding"
        args = ["discover", "--source", str(self.source), "--expected-commit", SOURCE["commit"],
                "--output", str(location), "--head", HEAD, "--pr", "1", "--sonar-run", "20", "--step-output", str(step)]
        with mock.patch.dict(os.environ, {"GITHUB_TOKEN": "synthetic-token"}), \
                mock.patch.object(INTAKE, "GitHubRead", return_value=self.discovery_api), \
                mock.patch.object(PC, "source_identity", return_value=(SOURCE, TRACKED)):
            self.assertEqual(0, INTAKE.main(args))
        raw = (location / "binding.json").read_bytes()
        self.assertEqual(self.binding, json.loads(raw))
        self.assertEqual("ready=true\nrun_id=10\nartifact_id=200\nbinding_sha256=" + PC.digest(raw) + "\n", step.read_text())
        self.assertNotIn("synthetic-token", raw.decode())
        archives = self.root / "archives"
        archives.mkdir()
        (archives / "200.zip").write_bytes(self.raw)
        linux = self.root / "linux.out"
        linux.write_bytes(PROFILE)
        args = ["consume", "--source", str(self.source), "--expected-commit", SOURCE["commit"], "--output", str(self.root / "admitted"),
                "--binding", str(location / "binding.json"), "--binding-sha256", PC.digest(raw), "--archives", str(archives), "--linux", str(linux)]
        with mock.patch.dict(os.environ, {"GITHUB_TOKEN": "synthetic-token"}), \
                mock.patch.object(INTAKE, "GitHubRead", return_value=self.api), \
                mock.patch.object(PC, "source_identity", return_value=(SOURCE, TRACKED)):
            self.assertEqual(0, INTAKE.main(args))
        self.assertEqual(PROFILE, (self.root / "admitted/coverage.out").read_bytes())

    def test_discovery_missing_artifact_or_job_cannot_claim_readiness(self):
        for collection in ("artifacts", "jobs"):
            def api(path):
                if "/" + collection + "?" in path:
                    return {"total_count": 0, collection: []}
                return self.discovery_api(path)
            with self.subTest(collection=collection), self.assertRaises(INTAKE.Refused):
                INTAKE.discover(api, SOURCE, HEAD, "1", "20", sleep=mock.Mock())
        def stale(path):
            result = self.discovery_api(path)
            if "/workflows/" in path:
                result["workflow_runs"][0]["created_at"] = "2026-11-17T20:00:00Z"
            return result
        with self.assertRaisesRegex(INTAKE.Refused, "not-terminal"):
            INTAKE.discover(stale, SOURCE, HEAD, "1", "20", sleep=mock.Mock())

    def test_api_cap_response_size_and_json_failures_do_not_export_bodies(self):
        client = INTAKE.GitHubRead("synthetic-token", 999)
        response = mock.MagicMock()
        response.__enter__.return_value = response
        client.opener = mock.Mock()
        client.opener.open.return_value = response
        for status, raw in ((503, b"private marker"), (200, b"x" * (INTAKE.MAX_JSON + 1)), (200, b"not JSON")):
            response.status, response.read.return_value = status, raw
            with self.subTest(status=status, length=len(raw)), mock.patch.object(INTAKE.time, "monotonic", return_value=1), self.assertRaises((INTAKE.Refused, ValueError)):
                client(INTAKE.PREFIX + "/actions/runs/20")
        client.calls = 128
        with mock.patch.object(INTAKE.time, "monotonic", return_value=1), self.assertRaises(INTAKE.Refused):
            client(INTAKE.PREFIX + "/actions/runs/20")

    def test_workflow_scopes_token_read_grant_and_preserves_original_reports_and_gates(self):
        root = Path(__file__).parents[3]
        workflow = (root / ".github/workflows/sonar.yml").read_text()
        self.assertEqual(1, workflow.count("  actions: read"))
        self.assertEqual(INTAKE.UNION_JOB, "Authoritative complete Source qualification")
        self.assertNotIn("actions: write", workflow)
        self.assertIn('unset CQ_SOURCE_UNION_TOKEN', workflow)
        self.assertLess(workflow.index('unset CQ_SOURCE_UNION_TOKEN'), workflow.index('go test -count=1'))
        self.assertIn('GITHUB_TOKEN="$source_union_token" python -I -S -B .github/scripts/source_union_intake.py consume', workflow)
        self.assertIn('source_union_token=\'\'', workflow)
        self.assertIn('fail "source-platform-union-unavailable-or-refused"', workflow)
        self.assertIn('coverage-linux-original.out', workflow)
        self.assertIn('if (covered != total) exit 1', workflow)
        self.assertIn('go test -count=1 -p 1 -tags=integration', workflow)
        commands = [line for line in workflow.splitlines() if ' -m unittest -v test_go_mcp_coverage ' in line]
        self.assertEqual(len(commands), 1)
        selected = commands[0].split(' -m unittest -v ', 1)[1].split(' >>', 1)[0].split()
        original = {'test_go_mcp_coverage', 'test_retain_audit_evidence', 'test_sdk_python_check',
                    'test_sdk_python_darwin_source_policy', 'test_sdk_python_lock', 'test_sdk_python_darwin_wheel',
                    'test_platform_coverage', 'test_build_explore', 'test_sdk_python_check_completion',
                    'test_sdk_python_lock_completion', 'test_go_mcp_coverage_completion',
                    'test_retain_audit_evidence_completion', 'test_sdk_python_darwin_source_policy_completion',
                    'test_source_union_intake'}
        self.assertTrue(original <= set(selected))
        self.assertEqual(len(selected), len(set(selected)))
        self.assertIn('".github/scripts/source_union_intake.py"', workflow)

    def test_main_refuses_without_authentication_and_prints_only_fixed_failure(self):
        with mock.patch.dict(os.environ, {"GITHUB_TOKEN": ""}), mock.patch("sys.stdout", new_callable=io.StringIO) as output:
            result = INTAKE.main(["discover", "--source", str(self.source), "--expected-commit", SOURCE["commit"], "--output", str(self.root / "out")])
        self.assertEqual(1, result)
        self.assertEqual("source-union-intake-refused\n", output.getvalue())
        self.assertFalse((self.root / "out").exists())


if __name__ == "__main__":
    unittest.main()
