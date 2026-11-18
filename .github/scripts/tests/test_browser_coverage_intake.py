"""Owned authority/pins oracles, not native Node or coverage measurements."""
from __future__ import annotations

import copy
import importlib.util
import io
import json
import os
from pathlib import Path
import runpy
import shlex
import stat
import subprocess
import tempfile
import unittest
from unittest import mock
import zipfile

SCRIPT = Path(__file__).resolve().parents[1] / "browser_coverage_intake.py"
SPEC = importlib.util.spec_from_file_location("cq_browser_intake_contract", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
I = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(I)
COMMIT, TREE, HEAD, ALIAS = "a" * 40, "b" * 40, "c" * 40, "d" * 40
SOURCE = {"commit": COMMIT, "tree": TREE, "inventory_sha256": "1" * 64,
          "package_sha256": "2" * 64, "implementation_sha256": "3" * 64}
ENVIRONMENT = {"GITHUB_WORKSPACE": "", "GITHUB_SHA": COMMIT, "GITHUB_REPOSITORY": I.SI.REPOSITORY,
               "GITHUB_RUN_ID": "102", "GITHUB_RUN_ATTEMPT": "1", "GITHUB_JOB": "sonar",
               "GITHUB_EVENT_NAME": "pull_request", "CQ_PR_HEAD_SHA": HEAD}


def workflow_run(identity, workflow):
    return {"id": identity, "repository": {"id": I.SI.REPOSITORY_ID, "full_name": I.SI.REPOSITORY},
            "head_repository": {"id": I.SI.REPOSITORY_ID}, "event": "pull_request", "head_sha": HEAD,
            "path": ".github/workflows/" + workflow + ".yml", "run_attempt": 1,
            "pull_requests": [{"number": 7, "head": {"sha": HEAD}}], "status": "in_progress",
            "created_at": "2026-11-18T00:00:00Z"}


def producer(role):
    return {"repository": I.SI.REPOSITORY, "run": "102" if role == "baseline" else "101",
            "attempt": "1", "job": "sonar" if role == "baseline" else "browser", "event": "pull_request",
            "head": HEAD, "platform": "linux" if role == "baseline" else role,
            "suite": "full-browser" if role == "baseline" else "web", "node": I.BC.NODE_VERSION}


def members(role, commit=COMMIT):
    metadata = {"source_after": {**SOURCE, "commit": commit}, "producer": producer(role),
                "node_after": {"sha256": "4" * 64}}
    # These are authority fixtures only. Actual Node/V8/report correspondence
    # remains the separately reviewed helper union's mandatory next boundary.
    return {"lcov.info": b"owned-original-not-a-measured-profile\n", "counters.json": b"{}\n",
            "metadata.json": I.PC.encoded(metadata)}


def archive(values):
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", compression=zipfile.ZIP_STORED) as handle:
        for name, data in values.items():
            item = zipfile.ZipInfo(name)
            item.external_attr = (stat.S_IFREG | 0o600) << 16
            handle.writestr(item, data)
    return stream.getvalue()


class OwnedCase(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name).resolve()
        self.source = self.directory / "source"
        self.source.mkdir()
        self.environment = {**ENVIRONMENT, "GITHUB_WORKSPACE": str(self.source), "RUNNER_TEMP": str(self.directory)}
        self.archives = self.directory / "archives"
        self.archives.mkdir()
        self.rows = {}
        self.raw = {}
        for index, role in enumerate(I.ROLES):
            data = archive(members(role))
            self.raw[role] = data
            row = {"id": 201 + index, "expired": False, "size_in_bytes": len(data),
                   "digest": "sha256:" + I.PC.digest(data), "workflow_run": {"id": 101},
                   "name": "browser-coverage-" + COMMIT + "-web-" + role}
            self.rows[role] = row
            path = self.archives / role
            path.mkdir()
            (path / "owned.zip").write_bytes(data)
        self.jobs = [{"id": 301 + index, "name": I.JOB_NAMES[role], "status": "completed", "conclusion": "success",
                      "steps": [{"name": I.STEP, "status": "completed", "conclusion": "success"}]} for index, role in enumerate(I.ROLES)]
        self.current_job = {"id": 401, "name": "Hosted ephemeral SonarQube gate", "status": "in_progress"}
        I.COMMIT_CACHE.clear()

    def api(self, path):
        if path.endswith('/runs/102'):
            return workflow_run(102, "sonar")
        if path.endswith('/runs/101'):
            return workflow_run(101, "browser")
        if '/runs/102/jobs?' in path:
            return {"total_count": 1, "jobs": [copy.deepcopy(self.current_job)]}
        if '/runs/101/jobs?' in path:
            return {"total_count": len(self.jobs), "jobs": copy.deepcopy(self.jobs)}
        if '/workflows/browser.yml/runs?' in path:
            return {"total_count": 1, "workflow_runs": [workflow_run(101, "browser")]}
        if '/runs/101/artifacts?' in path:
            return {"total_count": len(self.rows), "artifacts": copy.deepcopy(list(self.rows.values()))}
        for row in self.rows.values():
            if path.endswith('/artifacts/' + str(row["id"])):
                return copy.deepcopy(row)
        raise AssertionError("unexpected authority route")

    def binding(self):
        return I.discover(self.api, SOURCE, HEAD, "7", "102", sleep=lambda _: self.fail("unexpected readiness wait"))


class ReadinessContracts(OwnedCase):
    def test_missing_original_uses_finite_readiness_observations_not_a_new_run(self):
        waits = []
        def absent(path):
            if '/workflows/browser.yml/runs?' in path:
                return {"total_count": 0, "workflow_runs": []}
            return self.api(path)
        with self.assertRaises(ValueError):
            I.discover(absent, SOURCE, HEAD, "7", "102", sleep=waits.append)
        self.assertEqual(waits, [10] * 29)

    def test_other_repo_head_attempt_or_workflow_is_not_the_current_original(self):
        for key, value in (("run_attempt", 2), ("head_sha", ALIAS), ("path", ".github/workflows/other.yml")):
            def changed(path):
                result = self.api(path)
                if '/workflows/browser.yml/runs?' in path:
                    result["workflow_runs"][0][key] = value
                return result
            with self.subTest(key=key), self.assertRaises(ValueError):
                I.discover(changed, SOURCE, HEAD, "7", "102")
    def test_three_distinct_original_roles_are_bound_without_current_monolith_success(self):
        binding = self.binding()
        self.assertEqual(set(binding["entries"]), set(I.ROLES))
        self.assertEqual(binding["sonar_job"], {"id": "401", "status": "in_progress"})
        self.assertEqual(binding["source"], SOURCE)
        self.assertEqual({entry["job"]["id"] for entry in binding["entries"].values()}, {"301", "302", "303"})

    def test_each_pending_native_role_waits_with_archive_already_present(self):
        for role in I.ROLES:
            observations, sleeps = [], []
            def api(path):
                result = self.api(path)
                if '/runs/101/jobs?' in path:
                    observations.append(1)
                    if len(observations) == 1:
                        selected = next(job for job in result["jobs"] if job["name"] == I.JOB_NAMES[role])
                        selected.update(status="in_progress", conclusion=None)
                        selected["steps"][0]["status"] = "in_progress"
                return result
            with self.subTest(role=role):
                self.assertEqual(I.discover(api, SOURCE, HEAD, "7", "102", sleep=sleeps.append)["head"], HEAD)
                self.assertEqual(sleeps, [10])

    def test_failed_selected_step_or_duplicate_role_never_qualifies(self):
        self.jobs[0]["steps"][0]["conclusion"] = "failure"
        with self.assertRaises(ValueError):
            self.binding()
        self.jobs[0]["steps"][0]["conclusion"] = "success"
        self.jobs.append(copy.deepcopy(self.jobs[0]))
        with self.assertRaises(ValueError):
            self.binding()

    def test_current_sonar_job_is_owned_in_progress_not_a_fabricated_completed_step(self):
        self.current_job["status"] = "completed"
        with self.assertRaises(ValueError):
            self.binding()

    def test_ambiguous_original_or_reused_artifact_refuses(self):
        def ambiguous(path):
            result = self.api(path)
            if '/workflows/browser.yml/runs?' in path:
                result = {"total_count": 2, "workflow_runs": [workflow_run(101, "browser"), workflow_run(103, "browser")]}
            return result
        with self.assertRaises(ValueError):
            I.discover(ambiguous, SOURCE, HEAD, "7", "102")
        self.rows["macos"]["id"] = self.rows["linux"]["id"]
        with self.assertRaises(ValueError):
            self.binding()

    def test_expired_wrong_provider_digest_and_wrong_generated_source_name_refuse(self):
        for field, value in (("expired", True), ("digest", "sha256:BAD"), ("name", "browser-coverage-wrong-web-linux")):
            old = self.rows["linux"][field]
            self.rows["linux"][field] = value
            try:
                with self.subTest(field=field), self.assertRaises(ValueError):
                    def terminal(path):
                        result = self.api(path)
                        if '/workflows/browser.yml/runs?' in path:
                            result["workflow_runs"][0]["status"] = "completed"
                        return result
                    I.discover(terminal, SOURCE, HEAD, "7", "102")
            finally:
                self.rows["linux"][field] = old


class SourceAndPinsContracts(OwnedCase):
    def test_real_help_entry_returns_before_git_network_or_node_probes(self):
        with mock.patch("sys.argv", [str(SCRIPT), "--help"]), mock.patch("subprocess.run", side_effect=AssertionError("no process probe before help")), mock.patch("urllib.request.OpenerDirector.open", side_effect=AssertionError("no API before help")), mock.patch("sys.stdout", new_callable=io.StringIO), self.assertRaises(SystemExit) as result:
            runpy.run_path(str(SCRIPT), run_name="__main__")
        self.assertEqual(result.exception.code, 0)
    def test_wrong_alias_tree_and_head_parent_refuse_even_with_identical_labels(self):
        for tree, parents in ((ALIAS, [{"sha": HEAD}]), (TREE, [{"sha": COMMIT}])):
            I.COMMIT_CACHE.clear()
            with self.subTest(tree=tree, parents=parents), mock.patch.object(I, "github_commit", return_value={"tree": {"sha": tree}, "parents": parents}), self.assertRaises(ValueError):
                I.source_alias(self.api, ALIAS, SOURCE, HEAD)
    def test_same_checkout_needs_no_alias_lookup_but_different_commit_needs_head_and_tree(self):
        with mock.patch.object(I, "github_commit", side_effect=AssertionError("same commit uses local current source")):
            self.assertEqual(I.source_alias(self.api, COMMIT, SOURCE, HEAD)["authority"], "same-current-checkout")
        record = {"tree": {"sha": TREE}, "parents": [{"sha": HEAD}]}
        with mock.patch.object(I, "github_commit", return_value=record) as read:
            self.assertEqual(I.source_alias(self.api, ALIAS, SOURCE, HEAD)["tree"], TREE)
            I.source_alias(self.api, ALIAS, SOURCE, HEAD)
            self.assertEqual(read.call_count, 1)
        I.COMMIT_CACHE.clear()
        with mock.patch.object(I, "github_commit", return_value={"tree": {"sha": TREE}, "parents": []}), self.assertRaises(ValueError):
            I.source_alias(self.api, ALIAS, SOURCE, HEAD)

    def test_git_metadata_read_is_fixed_same_repo_bounded_and_never_follows_redirects(self):
        response = mock.MagicMock()
        response.__enter__.return_value = response
        response.status = 200
        response.read.return_value = I.PC.encoded({"sha": ALIAS, "tree": {"sha": TREE}})
        client = mock.Mock(token="synthetic", calls=0, deadline=100)
        client.opener.open.return_value = response
        with mock.patch.object(I.time, "monotonic", return_value=1):
            self.assertEqual(I.github_commit(client, ALIAS)["sha"], ALIAS)
        request = client.opener.open.call_args.args[0]
        self.assertEqual(request.full_url, "https://api.github.com/repos/maldous/CircularQuay/git/commits/" + ALIAS)
        self.assertEqual(request.get_method(), "GET")
        self.assertEqual(client.calls, 1)
        client.calls = 128
        with mock.patch.object(I.time, "monotonic", return_value=1), self.assertRaises(ValueError):
            I.github_commit(client, ALIAS)

    def test_native_archive_must_match_api_digest_and_exact_producer_role(self):
        binding = self.binding()
        entry, raw = I.authenticated_native(self.api, binding, "linux", self.archives / "linux", SOURCE)
        self.assertEqual(raw, self.raw["linux"])
        self.assertEqual(entry["artifact_id"], "201")
        self.assertEqual(entry["producer"], producer("linux"))
        (self.archives / "linux" / "owned.zip").write_bytes(self.raw["linux"] + b"changed")
        with self.assertRaises(ValueError):
            I.authenticated_native(self.api, binding, "linux", self.archives / "linux", SOURCE)

    def test_authenticated_bytes_still_require_the_actual_role_and_generated_source(self):
        for field, value in (("producer", producer("macos")),
                             ("source_after", {**SOURCE, "commit": ALIAS})):
            raw = members("linux")
            metadata = json.loads(raw["metadata.json"])
            metadata[field] = value
            raw["metadata.json"] = I.PC.encoded(metadata)
            data = archive(raw)
            self.rows["linux"].update(size_in_bytes=len(data), digest="sha256:" + I.PC.digest(data))
            (self.archives / "linux" / "owned.zip").write_bytes(data)
            binding = self.binding()
            with self.subTest(field=field), self.assertRaises(ValueError):
                I.authenticated_native(self.api, binding, "linux", self.archives / "linux", SOURCE)

    def test_opaque_download_refuses_symlink_or_multiple_members_without_extraction(self):
        path = self.archives / "linux"
        (path / "unexpected.zip").write_bytes(b"not selected")
        with self.assertRaises(ValueError):
            I.downloaded_archive(path)
        (path / "unexpected.zip").unlink()
        self.assertEqual(I.downloaded_archive(path), self.raw["linux"])
        if hasattr(os, "symlink") and os.name == "posix":
            alias = self.directory / "download-alias"
            alias.symlink_to(path, target_is_directory=True)
            with self.assertRaises(ValueError):
                I.downloaded_archive(alias)

    def test_typed_local_baseline_has_no_provider_or_archive_identity(self):
        binding = self.binding()
        local = self.directory / "cq-browser-full-browser-linux"
        local.mkdir()
        for name, raw in members("baseline").items():
            (local / name).write_bytes(raw)
        with mock.patch.dict(os.environ, self.environment, clear=True):
            entry = I.local_baseline(binding, SOURCE)
        self.assertEqual(entry["kind"], "owned-current-sonar")
        self.assertNotIn("artifact_id", entry)
        self.assertNotIn("archive_sha256", entry)
        self.assertEqual(entry["job_id"], "401")

    def test_local_baseline_wrong_source_or_producer_cannot_construct_pins(self):
        binding = self.binding()
        for field, value in (("source_after", {**SOURCE, "commit": ALIAS}), ("producer", producer("windows"))):
            raw = members("baseline")
            metadata = json.loads(raw["metadata.json"])
            metadata[field] = value
            raw["metadata.json"] = I.PC.encoded(metadata)
            with self.subTest(field=field), mock.patch.object(I.BC, "read_local_members", return_value=raw), mock.patch.dict(os.environ, self.environment, clear=True), self.assertRaises(ValueError):
                I.local_baseline(binding, SOURCE)

    def test_pins_are_constructed_after_api_authority_not_by_a_self_authenticating_hash(self):
        binding = self.binding()
        output = self.directory / "pins"
        with mock.patch.object(I.BC, "source_identity", return_value=(SOURCE, {})), mock.patch.object(I, "local_baseline", return_value={"kind": "owned-current-sonar"}):
            I.consume(self.api, self.source, SOURCE, binding, self.archives, output)
        pins = json.loads((output / "pins.json").read_bytes())
        self.assertEqual(pins["schema"], "cq.native-browser.authorized-pins/2")
        self.assertEqual(set(pins["entries"]), {"baseline", "linux", "macos", "windows"})
        receipt = json.loads((output / "admission.json").read_bytes())
        self.assertFalse(receipt["global_release_qualified"])
        self.assertEqual(receipt["pins_sha256"], I.BC.digest((output / "pins.json").read_bytes()))

    def test_provider_pin_or_selected_step_change_prevents_output(self):
        binding = self.binding()
        original = self.rows["linux"]["digest"]
        self.rows["linux"]["digest"] = "sha256:" + "0" * 64
        with mock.patch.object(I.BC, "source_identity", return_value=(SOURCE, {})), self.assertRaises(ValueError):
            I.consume(self.api, self.source, SOURCE, binding, self.archives, self.directory / "refused")
        self.assertFalse((self.directory / "refused").exists())
        self.rows["linux"]["digest"] = original
        self.jobs[0]["steps"][0]["conclusion"] = "failure"
        with mock.patch.object(I.BC, "source_identity", return_value=(SOURCE, {})), self.assertRaises(ValueError):
            I.consume(self.api, self.source, SOURCE, binding, self.archives, self.directory / "refused")
        self.assertFalse((self.directory / "refused").exists())

    @unittest.skipUnless(hasattr(os, "getuid"), "Owned Ubuntu Actions-output file fence only")
    def test_emit_binding_keeps_one_owned_source_and_numeric_outputs(self):
        binding = self.binding()
        step = self.directory / "step"
        step.write_bytes(b"")
        output = self.directory / "binding"
        I.emit_binding(binding, self.source, output, step)
        self.assertEqual(json.loads((output / "binding.json").read_bytes()), binding)
        self.assertIn("artifact_linux=201\n", step.read_text())
        self.assertIn(I.PC.digest((output / "binding.json").read_bytes()), step.read_text())

    def test_main_refuses_a_mismatched_source_context_before_actions_access(self):
        args = ["discover", "--source", str(self.source), "--sonar-run", "102", "--expected-commit", ALIAS,
                "--head", HEAD, "--pr", "7", "--output", str(self.directory / "refused")]
        with mock.patch.dict(os.environ, self.environment, clear=True), mock.patch.object(I.SI, "GitHubRead", side_effect=AssertionError("no Actions access before source context")), mock.patch("sys.stdout", new_callable=io.StringIO) as output:
            self.assertEqual(I.main(args), 2)
        self.assertEqual(output.getvalue(), "current-browser-evidence-refused\n")
        self.assertFalse((self.directory / "refused").exists())

    @unittest.skipUnless(hasattr(os, "getuid"), "Actual workflow operator is Ubuntu only")
    def test_main_discovery_and_consume_form_one_fixed_authority_lifecycle(self):
        destination = self.directory / "discovered"
        step = self.directory / "step"
        step.write_bytes(b"")
        common = ["--source", str(self.source), "--sonar-run", "102", "--expected-commit", COMMIT, "--head", HEAD, "--pr", "7"]
        with mock.patch.dict(os.environ, self.environment, clear=True), mock.patch.object(I.SI, "GitHubRead", return_value=self.api), mock.patch.object(I.BC, "source_identity", return_value=(SOURCE, {})):
            self.assertEqual(I.main(["discover", *common, "--output", str(destination), "--step-output", str(step)]), 0)
            raw = (destination / "binding.json").read_bytes()
            with mock.patch.object(I, "local_baseline", return_value={"kind": "owned-current-sonar"}):
                self.assertEqual(I.main(["consume", *common, "--output", str(self.directory / "pins"),
                                         "--binding", str(destination / "binding.json"), "--binding-sha256", I.PC.digest(raw),
                                         "--archives", str(self.archives)]), 0)
        self.assertTrue((self.directory / "pins" / "pins.json").is_file())

    @unittest.skipUnless(hasattr(os, "getuid"), "Actual Actions-output operator is Ubuntu only")
    def test_only_the_api_client_receives_token_before_every_actual_probe_environment(self):
        probes, clients = [], []
        credential = "owned-synthetic-actions-token"
        def bounded(argv, cwd, environment, _limit):
            self.assertEqual(argv[0], "git")
            self.assertEqual(cwd, self.source)
            self.assertNotIn("GITHUB_TOKEN", environment)
            probes.append(environment.copy())
            return b"owned-probe"
        def source(root, _expected):
            # Exercise unchanged BC.git -> BC.command's real environment
            # construction. Only the process primitive is replaced; no Git
            # command, child or network is executed by this owned oracle.
            self.assertEqual(I.BC.git(root, "rev-parse", "HEAD"), b"owned-probe")
            return SOURCE, {}
        def client(token, _deadline):
            self.assertEqual(token, credential)
            self.assertNotIn("GITHUB_TOKEN", os.environ)
            clients.append(token)
            return self.api
        step = self.directory / "token-step"
        step.write_bytes(b"")
        destination = self.directory / "token-binding"
        common = ["--source", str(self.source), "--sonar-run", "102", "--expected-commit", COMMIT,
                  "--head", HEAD, "--pr", "7"]
        with mock.patch.dict(os.environ, self.environment, clear=True), mock.patch.object(I.SI, "GitHubRead", side_effect=client), mock.patch.object(I.BC, "bounded_probe", side_effect=bounded), mock.patch.object(I.BC, "source_identity", side_effect=source), mock.patch.object(I, "local_baseline", return_value={"kind": "owned-current-sonar"}):
            os.environ["GITHUB_TOKEN"] = credential
            self.assertEqual(I.main(["discover", *common, "--output", str(destination), "--step-output", str(step)]), 0)
            raw = (destination / "binding.json").read_bytes()
            os.environ["GITHUB_TOKEN"] = credential
            self.assertEqual(I.main(["consume", *common, "--output", str(self.directory / "token-pins"),
                                     "--binding", str(destination / "binding.json"), "--binding-sha256", I.PC.digest(raw),
                                     "--archives", str(self.archives)]), 0)
            self.assertNotIn("GITHUB_TOKEN", os.environ)
        self.assertEqual(clients, [credential, credential])
        self.assertEqual(len(probes), 5)


class WorkflowContracts(unittest.TestCase):
    @unittest.skipUnless(os.name == "posix" and Path("/bin/bash").is_file(), "Existing Ubuntu Sonar Bash operator only")
    def test_source_derived_native_status_control_records_and_reaches_scanner_under_errexit(self):
        workflow = (SCRIPT.parents[2] / ".github/workflows/sonar.yml").read_text()
        command = next(row for row in workflow.splitlines() if "python_coverage.py consume" in row)
        capture = command[command.index(" || "):]
        self.assertEqual(capture, " || native_python_status=$?")
        start = workflow.index('case "$native_python_status" in')
        end = workflow.index("esac", start) + len("esac")
        control = "\n".join(row.strip() for row in workflow[start:end].splitlines())
        # Restrict the source-derived snippet to fixed builtins before running
        # it. Never execute a consumer, network command or arbitrary workflow.
        expected = "\n".join([
            'case "$native_python_status" in', '0) : ;;',
            '1) fail "python-native-coverage-not-100" ;;', '*)',
            'fail "python-native-evidence-unavailable-or-refused"',
            "printf '%s\\n' '{\"kind\":\"same-source-native-python-composite\",\"admitted\":false,\"selected\":\"raw-linux-fallback\",\"global_gate_passed\":false}' >\"$evidence/python-composite.json\"",
            ';;', 'esac'])
        self.assertEqual(control, expected)
        script = ('evidence=$1\nstatus=$2\nfailed=0\n'
                  'fail() { printf "failure:%s\\n" "$1"; failed=1; }\n'
                  'owned_consume() { return "$1"; }\n'
                  'native_python_status=0\nowned_consume "$status"' + capture + '\n' + control + '\n'
                  'printf "recorded:%s\\nscanner-reachable\\n" "$native_python_status"\nexit "$failed"\n')
        self.assertLess(len(script), 4096)
        with tempfile.TemporaryDirectory() as temporary:
            for status in (0, 1, 2, 137):
                evidence = Path(temporary).resolve() / str(status)
                evidence.mkdir()
                with self.subTest(status=status):
                    result = subprocess.run(['/bin/bash', '--noprofile', '--norc', '-euo', 'pipefail', '-c',
                                             script, 'owned-status-contract', str(evidence), str(status)],
                                            cwd=evidence, env={'PATH': os.defpath, 'LANG': 'C.UTF-8'},
                                            capture_output=True, timeout=5, check=False)
                    self.assertEqual(result.returncode, 0 if status == 0 else 1)
                    self.assertEqual(result.stderr, b'')
                    failure = ('' if status == 0 else 'failure:python-native-coverage-not-100\n'
                               if status == 1 else 'failure:python-native-evidence-unavailable-or-refused\n')
                    self.assertEqual(result.stdout, (failure + f'recorded:{status}\nscanner-reachable\n').encode())
                    fallback = evidence / 'python-composite.json'
                    self.assertEqual(fallback.exists(), status not in (0, 1))
                    if fallback.exists():
                        self.assertFalse(json.loads(fallback.read_bytes())['admitted'])
                        self.assertFalse(json.loads(fallback.read_bytes())['global_gate_passed'])

    def test_native_python_shortfall_is_captured_before_browser_and_scanner_evidence(self):
        workflow = (SCRIPT.parents[2] / ".github/workflows/sonar.yml").read_text()
        rows = [row for row in workflow.splitlines() if "python_coverage.py consume" in row]
        self.assertEqual(len(rows), 1)
        self.assertTrue(rows[0].endswith(" || native_python_status=$?"))
        self.assertIn("native_python_status=0\n" + rows[0], workflow)
        self.assertLess(workflow.index(rows[0]), workflow.index("native_intake_exit="))
        self.assertLess(workflow.index("native_intake_exit="), workflow.index("run_browser_once()"))
        self.assertIn('fail "python-native-evidence-unavailable-or-refused"', workflow)
        self.assertIn('fail "python-coverage-not-100:$canonical_python_status"', workflow)
        self.assertIn('"$evidence/python-linux-original.xml"', workflow)

    def test_expected_nonzero_calls_capture_status_before_inherited_errexit(self):
        workflow = (SCRIPT.parents[2] / ".github/workflows/sonar.yml").read_text()
        commands = (("browser_coverage.py retain", "baseline_retain_status"),
                    ("browser_coverage_intake.py consume", "browser_admission_status"),
                    ("browser_coverage.py union", "browser_union_status"))
        for command, status in commands:
            rows = [row for row in workflow.splitlines() if command in row]
            with self.subTest(command=command):
                self.assertEqual(len(rows), 1)
                self.assertTrue(rows[0].endswith(" || " + status + "=$?"))
                self.assertIn(status + "=0\n" + rows[0], workflow)
        self.assertLess(workflow.index("browser_coverage.py union"), workflow.index("canonical_union_exit="))
        self.assertIn('1) fail "browser-canonical-coverage-not-100"', workflow)
        self.assertIn('*) fail "browser-native-evidence-unavailable-or-refused"', workflow)
        self.assertIn('cp "$evidence/browser/lcov-linux-original.info" "$evidence/browser/lcov.info"', workflow)

    def test_node24_web_flags_precede_the_exact_declared_suite_once(self):
        root = SCRIPT.parents[2]
        workflow = (root / ".github/workflows/browser.yml").read_text()
        body = workflow.split('NODE_V8_COVERAGE="$output/raw" ', 1)[1].split(' || original_exit=$?', 1)[0]
        arguments = shlex.split(body.replace("\\\n", " "))
        declared = shlex.split(json.loads((root / "web/package.json").read_bytes())["scripts"]["test"])
        files = [word for word in arguments if word.endswith('.test.mjs') and not word.startswith('--')]
        self.assertEqual(files, declared[3:])
        self.assertEqual(arguments.count('--test'), 1)
        self.assertIn('--test-concurrency=1', arguments)
        self.assertLess(arguments.index('--experimental-test-coverage'), arguments.index(files[0]))
        self.assertIn('--test-reporter=lcov', arguments)
        self.assertNotIn('NODE_OPTIONS=', body)

    def test_every_existing_windows_check_and_non24_or_manual_suite_is_preserved(self):
        root = SCRIPT.parents[2]
        workflow = (root / ".github/workflows/browser.yml").read_text()
        body = workflow.split('      - name: Validate dependency-free browser preview\n', 1)[1]
        self.assertLess(body.index('npm run check'), body.index("if test '${{ matrix.node }}'"))
        self.assertIn('test "$GITHUB_EVENT_NAME" = workflow_dispatch', body)
        self.assertIn('npm test\n            exit 0', body)
        self.assertIn('npm test || original_exit=$?', body)
        self.assertIn('test "$original_exit" = 0 || exit "$original_exit"', body)
        self.assertIn('test_build_explore.py -v', workflow)

    def test_sonar_keeps_original_reports_and_requires_distinct_canonical_evidence(self):
        root = SCRIPT.parents[2]
        workflow = (root / ".github/workflows/sonar.yml").read_text()
        self.assertEqual(workflow.count('run_browser_once() {'), 1)
        self.assertIn('"$browser_node" --test --test-concurrency=1', workflow)
        self.assertIn('lcov-linux-original.info', workflow)
        self.assertIn('fail "browser-canonical-coverage-not-100"', workflow)
        self.assertIn('fail "browser-native-evidence-unavailable-or-refused"', workflow)
        self.assertIn('--baseline "$baseline"', workflow)
        self.assertIn('browser_coverage_intake.py consume', workflow)
        self.assertIn('test_browser_coverage_intake >>', workflow)
        self.assertIn('".github/scripts/browser_coverage_intake.py"', workflow)
        source_workflow = (root / ".github/workflows/source.yml").read_text()
        step = source_workflow.split("      - name: Check bounded evidence retention contracts\n", 1)[1].split("      # This job retains", 1)[0]
        selected = [line.strip() for line in step.splitlines() if " -m unittest discover " in line]
        expected = ["python3 -B -m unittest discover -s .github/scripts/tests -p " + name for name in (
            "test_retain_audit_evidence.py", "test_go_mcp_coverage.py", "test_platform_coverage.py",
            "test_source_gate.py", "test_browser_coverage_intake.py")]
        self.assertEqual(selected, expected)
        self.assertIn("set -euo pipefail", step)
        self.assertEqual(workflow.count('actions: read'), 1)
        self.assertNotIn('actions: write', workflow)


if __name__ == "__main__":
    unittest.main()
