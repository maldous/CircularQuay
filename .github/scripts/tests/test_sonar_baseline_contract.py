"""Synthetic provenance/IO oracles, never an old-source or server qualification."""

from __future__ import annotations

import contextlib
import copy
import datetime as dt
import hashlib
import importlib.util
import io
import json
import os
import runpy
import stat
import subprocess
import sys
import tempfile
import types
import unittest
from pathlib import Path
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "sonar_baseline_contract.py"
SPEC = importlib.util.spec_from_file_location("sonar_baseline_contract", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
BASE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(BASE)
EPOCH = int(dt.datetime(2026, 11, 17, 13, tzinfo=dt.timezone.utc).timestamp())
TARGET = "a" * 40
TARGET_TREE = "b" * 40
BLOB = "c" * 40
ANALYSIS = "11111111-1111-4111-8111-111111111111"
CURRENT = "22222222-2222-4222-8222-222222222222"
SERVER = {"id": "synthetic-server-0001", "version": "26.10.0.133078", "status": "UP"}


def encoded(value):
    return (json.dumps(value) + "\n").encode()


def identity():
    return {"schema": "cq-sonar-ancestor-source/1", "repository": "maldous/CircularQuay",
            "revision": BASE.REVISION, "tree": BASE.TREE,
            "committer_timestamp": "2026-11-17T13:00:00+00:00", "committer_epoch": EPOCH,
            "project_date": "2026-11-17", "date_granularity": "UTC calendar day; conservative floor",
            "target_revision": TARGET, "target_tree": TARGET_TREE, "target_committer_epoch": EPOCH + 3600,
            "source_entries": 1, "source_bytes": 7, "repository_object_bytes": 2048, "listing_sha256": "d" * 64,
            "server": SERVER.copy(), "scanner_sha256": "e" * 64, "analyzer_sha256": "f" * 64}


def analyses(rows):
    return {"paging": {"pageIndex": 1, "pageSize": 100, "total": len(rows)}, "analyses": rows}


def baseline_inputs():
    task = {"task": {"id": "synthetic-task", "status": "SUCCESS", "componentKey": "cq", "analysisId": ANALYSIS,
                     "warningCount": 0, "submittedAt": "2026-11-17T22:00:00+0000",
                     "startedAt": "2026-11-17T22:00:01+0000", "executedAt": "2026-11-17T22:00:02+0000"}}
    history = analyses([{"key": ANALYSIS, "revision": BASE.REVISION, "date": "2026-11-17T00:00:00+0000"}])
    return identity(), b"ceTaskId=synthetic-task\n", task, history, {"projectStatus": {"status": "ERROR"}}, "3", "main"


def baseline():
    return BASE.baseline_receipt(*baseline_inputs())


def period():
    return {"projectKey": "cq", "branchKey": "main", "type": "SPECIFIC_ANALYSIS", "value": ANALYSIS,
            "inherited": False, "updatedAt": 1794952803000}


class SonarBaselineContracts(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()

    def file(self, name, value):
        path = self.root / name
        path.write_bytes(value if isinstance(value, bytes) else encoded(value))
        path.chmod(0o600)
        return str(path)

    def call(self, args, status=0, stdout=""):
        with contextlib.redirect_stdout(io.StringIO()) as out, contextlib.redirect_stderr(io.StringIO()) as err:
            self.assertEqual(status, BASE.main(args))
        self.assertEqual(stdout, out.getvalue())
        self.assertEqual(BASE.REFUSED + "\n" if status else "", err.getvalue())

    def fake_git(self, **changes):
        raw = f"100644 blob {BLOB}\tsrc/owned.py\0".encode()
        values = {
            ("rev-parse", "--show-toplevel"): (str(self.root) + "\n").encode(),
            ("rev-parse", "HEAD"): (TARGET + "\n").encode(),
            ("status", "--porcelain", "--untracked-files=all"): b"",
            ("rev-parse", BASE.REVISION + "^{tree}"): (BASE.TREE + "\n").encode(),
            ("merge-base", "--is-ancestor", BASE.REVISION, TARGET): b"",
            ("show", "-s", "--format=%ct", BASE.REVISION): str(EPOCH).encode(),
            ("show", "-s", "--format=%ct", TARGET): str(EPOCH + 3600).encode(),
            ("show", "-s", "--format=%cI", BASE.REVISION): b"2026-11-17T13:00:00+00:00",
            ("rev-parse", "HEAD^{tree}"): TARGET_TREE.encode(),
            ("count-objects", "-v"): b"size: 1\nsize-pack: 1\nsize-garbage: 0\n",
            ("ls-tree", "-rz", BASE.REVISION): raw,
            ("cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)"): f"{BLOB} blob 7\n".encode(),
        }
        values.update(changes.get("values", {}))
        def selected(root, *args, data=None):
            self.assertEqual(str(self.root), root)
            self.assertIn(args, values)
            value = values[args]
            if isinstance(value, Exception):
                raise value
            if args[0] == "cat-file":
                self.assertEqual((BLOB + "\n").encode(), data)
            return value
        return selected

    def test_json_is_bounded_unique_and_strict_without_echoing_payloads(self):
        self.assertEqual({"owned": 1}, BASE.document(b'{"owned":1}'))
        for raw in (b"", b"{}".ljust(BASE.MAX_JSON + 1, b" "), b"\xff", b"[]", b'{"a":1,"a":2}', b'{"a":NaN}', b"{"):
            with self.subTest(kind=raw[:8]), self.assertRaises(ValueError):
                BASE.document(raw)
        with self.assertRaises(ValueError):
            BASE.document("not bytes")

    def test_protected_owned_read_and_new_write_preserve_existing_data(self):
        path = self.file("owned", b"owned")
        self.assertEqual(b"owned", BASE.read_owned(path))
        new = str(self.root / "new")
        BASE.write_new(new, b"new")
        self.assertEqual(0o600, stat.S_IMODE(Path(new).stat().st_mode))
        self.assertEqual(b"new", Path(new).read_bytes())
        with self.assertRaises(FileExistsError):
            BASE.write_new(new, b"replacement")
        self.assertEqual(b"new", Path(new).read_bytes())
        with self.assertRaises(ValueError):
            BASE.write_new(str(self.root / "empty"), b"")
        with self.assertRaises(ValueError):
            BASE.write_new(str(self.root / "huge"), b"x" * (BASE.MAX_JSON + 1))
        BASE.save(str(self.root / "json"), {"owned": True})
        self.assertEqual({"owned": True}, json.loads((self.root / "json").read_bytes()))

    def test_read_refuses_final_link_empty_oversize_and_shared_permissions(self):
        path = Path(self.file("owned", b"owned"))
        link = self.root / "link"
        link.symlink_to(path)
        with self.assertRaises(OSError):
            BASE.read_owned(str(link))
        for raw in (b"", b"x" * (BASE.MAX_JSON + 1)):
            path.write_bytes(raw)
            with self.assertRaises(ValueError):
                BASE.read_owned(str(path))
        path.write_bytes(b"owned")
        path.chmod(0o644)
        with self.assertRaises(ValueError):
            BASE.read_owned(str(path))
        path.chmod(0o600)
        os.link(path, self.root / "hardlink")
        with self.assertRaises(ValueError):
            BASE.read_owned(str(path))

    def test_read_refuses_synthetic_type_owner_short_read_and_metadata_drift(self):
        path = self.file("owned", b"owned")
        actual = os.stat(path)
        values = {name: getattr(actual, name) for name in ("st_mode", "st_uid", "st_nlink", "st_size", "st_mtime_ns", "st_ctime_ns")}
        for key, value in (("st_mode", stat.S_IFDIR | 0o700), ("st_uid", actual.st_uid + 1), ("st_size", 7)):
            before = types.SimpleNamespace(**(values | {key: value}))
            with self.subTest(key=key), mock.patch.object(BASE.os, "fstat", return_value=before), self.assertRaises(ValueError):
                BASE.read_owned(path)
        for key in ("st_size", "st_mtime_ns", "st_ctime_ns"):
            after = types.SimpleNamespace(**(values | {key: values[key] + 1}))
            with self.subTest(key=key), mock.patch.object(BASE.os, "fstat", side_effect=[types.SimpleNamespace(**values), after]), self.assertRaises(ValueError):
                BASE.read_owned(path)

    def test_fsync_failure_is_not_reported_as_a_rolled_back_write(self):
        path = self.root / "partial"
        with mock.patch.object(BASE.os, "fsync", side_effect=OSError("synthetic failure")), self.assertRaises(OSError):
            BASE.write_new(str(path), b"owned")
        self.assertEqual(b"owned", path.read_bytes())

    def test_git_wrapper_is_bounded_noninteractive_and_excludes_ambient_config(self):
        good = types.SimpleNamespace(returncode=0, stdout=b"owned")
        with mock.patch.dict(os.environ, {"GIT_DIR": "synthetic-unselected", "PYTHONPATH": "synthetic-unselected"}), \
                mock.patch.object(BASE.subprocess, "run", return_value=good) as run:
            self.assertEqual(b"owned", BASE.git(str(self.root), "rev-parse", "HEAD", data=b"input"))
        args, kwargs = run.call_args
        self.assertIn("--no-replace-objects", args[0])
        self.assertEqual(b"input", kwargs["input"])
        self.assertEqual(30, kwargs["timeout"])
        self.assertFalse(kwargs["check"])
        self.assertEqual(subprocess.DEVNULL, kwargs["stderr"])
        self.assertNotIn("GIT_DIR", kwargs["env"])
        self.assertNotIn("PYTHONPATH", kwargs["env"])
        self.assertEqual("/dev/null", kwargs["env"]["GIT_CONFIG_GLOBAL"])
        for result in (types.SimpleNamespace(returncode=1, stdout=b""), types.SimpleNamespace(returncode=0, stdout=b"x" * ((4 << 20) + 1))):
            with mock.patch.object(BASE.subprocess, "run", return_value=result), self.assertRaises(ValueError):
                BASE.git(str(self.root), "rev-parse", "HEAD")

    def test_source_tree_guard_refuses_links_submodules_escapes_and_duplicates(self):
        valid = f"100755 blob {BLOB}\ttools/owned.py\0".encode()
        self.assertEqual((1, [BLOB]), BASE.source_entries(valid))
        rows = (b"", b"x" * ((4 << 20) + 1), valid * 5001, valid * 2,
                valid.replace(b"100755", b"120000"), valid.replace(b"blob", b"commit"),
                valid.replace(BLOB.encode(), b"invalid"), valid.replace(b"tools/owned.py", b"/escape"),
                valid.replace(b"tools/owned.py", b"a\\b"), valid.replace(b"tools/owned.py", b"a/../b"),
                valid.replace(b"tools/owned.py", b"a/.git/b"), valid.replace(b"tools/owned.py", b"a//b"),
                valid.replace(b"tools/owned.py", b"a\nb"), valid.replace(b"tools/owned.py", b"a/./b"))
        for row in rows:
            with self.subTest(prefix=row[:12]), self.assertRaises(ValueError):
                BASE.source_entries(row)

    def test_local_clone_object_storage_is_bounded_without_copying_private_state(self):
        self.assertEqual(2048, BASE.object_storage(b"size: 1\nsize-pack: 1\nsize-garbage: 0\n"))
        for raw in (b"", b"x" * 4097, b"size: 1\nsize: 2\n", b"alternate: /synthetic-unselected\nsize: 0\nsize-pack: 0\nsize-garbage: 0\n", b"size: unknown\nsize-pack: 1\nsize-garbage: 0\n",
                    b"size: 0\nsize-pack: 524289\nsize-garbage: 0\n"):
            with self.assertRaises(ValueError):
                BASE.object_storage(raw)

    def test_source_identity_binds_real_command_results_not_a_self_declared_hash(self):
        with mock.patch.object(BASE, "git", side_effect=self.fake_git()):
            result = BASE.source_identity(str(self.root), TARGET)
        self.assertEqual(BASE.REVISION, result["revision"])
        self.assertEqual(BASE.TREE, result["tree"])
        self.assertEqual(TARGET_TREE, result["target_tree"])
        self.assertEqual("2026-11-17", result["project_date"])
        self.assertEqual((1, 7), (result["source_entries"], result["source_bytes"]))
        result.update(server=SERVER, scanner_sha256="e" * 64, analyzer_sha256="f" * 64)
        BASE.validate_identity(result)

    def test_source_identity_refuses_wrong_head_dirty_ancestry_dates_and_blob_sizes(self):
        changes = [
            (("rev-parse", "--show-toplevel"), b"/synthetic-other"),
            (("rev-parse", "HEAD"), ("d" * 40).encode()),
            (("status", "--porcelain", "--untracked-files=all"), b" M owned"),
            (("rev-parse", BASE.REVISION + "^{tree}"), TARGET_TREE.encode()),
            (("merge-base", "--is-ancestor", BASE.REVISION, TARGET), ValueError("not ancestor")),
            (("show", "-s", "--format=%ct", BASE.REVISION), b"0"),
            (("show", "-s", "--format=%cI", BASE.REVISION), b"2026-11-17T13:00:01+00:00"),
            (("rev-parse", "HEAD^{tree}"), b"invalid"),
            (("cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)"), b""),
            (("cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)"), f"{BLOB} tree 7".encode()),
            (("cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)"), f"{BLOB} blob {17 << 20}".encode()),
        ]
        for key, value in changes:
            with self.subTest(key=key), mock.patch.object(BASE, "git", side_effect=self.fake_git(values={key: value})), self.assertRaises(ValueError):
                BASE.source_identity(str(self.root), TARGET)
        for target in ("invalid", BASE.REVISION):
            with self.assertRaises(ValueError):
                BASE.source_identity(str(self.root), target)
        for responses in ([BASE.REVISION.encode(), BASE.TREE.encode(), b""], [TARGET.encode()], [BASE.REVISION.encode(), b"wrong"], [BASE.REVISION.encode(), BASE.TREE.encode(), b" M owned"]):
            with mock.patch.object(BASE, "git", side_effect=responses):
                if len(responses) == 3 and not responses[-1]:
                    BASE.verify_worktree(str(self.root))
                else:
                    with self.assertRaises(ValueError):
                        BASE.verify_worktree(str(self.root))

    def test_source_identity_total_byte_bound_and_offset_day_are_explicit(self):
        raw = b"".join(f"100644 blob {BLOB}\tsrc/{index}.py\0".encode() for index in range(33))
        values = {("ls-tree", "-rz", BASE.REVISION): raw,
                  ("cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)"): (f"{BLOB} blob {16 << 20}\n" * 33).encode()}
        selected = self.fake_git(values=values)
        def repeated(root, *args, data=None):
            return values[args] if args[0] == "cat-file" else selected(root, *args, data=data)
        with mock.patch.object(BASE, "git", side_effect=repeated), self.assertRaises(ValueError):
            BASE.source_identity(str(self.root), TARGET)
        offset = {("show", "-s", "--format=%cI", BASE.REVISION): b"2026-11-17T08:00:00-05:00"}
        with mock.patch.object(BASE, "git", side_effect=self.fake_git(values=offset)):
            result = BASE.source_identity(str(self.root), TARGET)
        self.assertEqual("2026-11-17T08:00:00-05:00", result["committer_timestamp"])
        self.assertEqual("2026-11-17", result["project_date"])

    def test_fresh_analysis_list_and_actual_main_branch_are_mandatory(self):
        self.assertEqual([], BASE.analysis_list(analyses([]), 0))
        self.assertEqual("main", BASE.main_branch({"branches": [{"name": "main", "isMain": True, "type": "BRANCH"}]}))
        for value in ({}, {"paging": {}, "analyses": []}, analyses([{}]), {"paging": {"pageIndex": True, "pageSize": 100, "total": 0}, "analyses": []}):
            with self.assertRaises(ValueError):
                BASE.analysis_list(value, 0)
        for value in ({}, {"branches": []}, {"branches": [{}]}, {"branches": [{"name": "main", "isMain": False, "type": "BRANCH"}]}, {"branches": [{"name": "bad\nbranch", "isMain": True, "type": "BRANCH"}]}):
            with self.assertRaises(ValueError):
                BASE.main_branch(value)

    def test_metadata_task_id_is_exact_unique_and_bounded(self):
        self.assertEqual("owned-id", BASE.task_id(b"projectKey=cq\nceTaskId=owned-id\n"))
        for value in (b"", b"x" * 4097, b"projectKey=cq", b"ceTaskId=one\nceTaskId=two\n", b"ceTaskId=bad?query"):
            with self.assertRaises(ValueError):
                BASE.task_id(value)

    def test_refused_baseline_still_has_bounded_safe_observations_not_raw_fields(self):
        args = baseline_inputs()
        value = BASE.observed_baseline(args[2], args[3], args[4])
        self.assertFalse(value["qualified"])
        self.assertEqual(ANALYSIS, value["analysis_uuid"])
        self.call(["observe", self.file("observed-task", args[2]), str(self.root / "observed")])
        self.assertEqual("SUCCESS", json.loads((self.root / "observed").read_bytes())["ce_status"])
        self.assertEqual("ERROR", value["quality_gate_status"])
        self.assertEqual("2026-11-17T00:00:00+0000", value["effective_analysis_date"])
        malformed = BASE.observed_baseline({"task": {"analysisId": "synthetic-private-value", "warningCount": True}},
                                           {"analyses": [{"revision": "synthetic-private-value"}, {}]},
                                           {"projectStatus": "synthetic-private-value"})
        self.assertIsNone(malformed["analysis_uuid"])
        self.assertIsNone(malformed["warning_count"])
        self.assertNotIn("synthetic-private-value", json.dumps(malformed))
        self.assertIsNone(BASE.observed_baseline({"task": []}, {}, {})["ce_status"])

    def test_unqualified_error_gate_and_real_uuid_never_mean_current_success(self):
        value = baseline()
        self.assertFalse(value["qualified"])
        self.assertEqual("ERROR", value["quality_gate_status"])
        self.assertEqual(3, value["scanner_exit"])
        self.assertEqual(ANALYSIS, value["analysis_uuid"])
        self.assertEqual("2026-11-17T00:00:00+0000", value["effective_analysis_date"])
        self.assertNotEqual(value["effective_analysis_date"], value["execution_times"]["submittedAt"])
        args = list(baseline_inputs());args[4] = {"projectStatus": {"status": "OK"}};args[5] = "0"
        args[2]["task"]["warningCount"] = 1
        self.assertEqual(1, BASE.baseline_receipt(*args)["warning_count"])
        self.assertFalse(BASE.baseline_receipt(*args)["qualified"])

    def test_identity_refuses_manual_date_rebound_source_and_bad_controls(self):
        for key, value in (("revision", TARGET), ("tree", TARGET_TREE), ("project_date", "2026-11-16"),
                           ("date_granularity", "exact timestamp"), ("target_revision", BASE.REVISION),
                           ("target_tree", 1), ("committer_timestamp", "2026-11-17T13:00:01+00:00"),
                           ("target_committer_epoch", EPOCH), ("source_entries", 0), ("source_bytes", 513 << 20),
                           ("server", SERVER | {"id": "other?server"}), ("scanner_sha256", "invalid")):
            value_dict = identity();value_dict[key] = value
            with self.subTest(key=key), self.assertRaises((ValueError, TypeError)):
                BASE.validate_identity(value_dict)

    def test_baseline_refuses_incomplete_cross_project_stale_sha_uuid_or_dates(self):
        changes = [(2, "status", "FAILED"), (2, "componentKey", "other"), (2, "analysisId", CURRENT),
                   (2, "id", "different-task"), (2, "warningCount", True), (2, "submittedAt", "2026-11-16T00:00:00+0000"),
                   (2, "startedAt", "2026-11-17T22:00:03+0000"), (2, "executedAt", "2026-11-17T22:00:02"),
                   (3, "key", BASE.REVISION), (3, "revision", TARGET), (3, "date", "2026-11-17T13:00:00+0000")]
        for index, key, value in changes:
            args = list(baseline_inputs())
            (args[2]["task"] if index == 2 else args[3]["analyses"][0])[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                BASE.baseline_receipt(*args)
        args = list(baseline_inputs());args[5] = "124"
        with self.assertRaises(ValueError):
            BASE.baseline_receipt(*args)
        args = list(baseline_inputs());args[4] = {"projectStatus": {"status": "NONE"}}
        with self.assertRaises(ValueError):
            BASE.baseline_receipt(*args)

    def test_specific_period_is_main_branch_uuid_not_project_inheritance(self):
        result = BASE.period_receipt(period(), baseline())
        self.assertEqual(ANALYSIS, result["analysis_uuid"])
        self.assertFalse(result["baseline_qualified"])
        as_string = period() | {"updatedAt": "1794952803000"}
        self.assertEqual(result, BASE.period_receipt(as_string, baseline()))
        for key, value in (("projectKey", "other"), ("branchKey", "other"), ("type", "PREVIOUS_VERSION"),
                           ("value", CURRENT), ("value", BASE.REVISION), ("inherited", True),
                           ("updatedAt", True), ("updatedAt", 0), ("updatedAt", "01")):
            with self.subTest(key=key), self.assertRaises(ValueError):
                BASE.period_receipt(period() | {key: value}, baseline())
        with self.assertRaises(ValueError):
            BASE.period_receipt(period(), baseline() | {"qualified": True})

    def test_current_correspondence_keeps_distinct_revision_uuid_server_and_chronology(self):
        previous = baseline_inputs()[3]["analyses"][0]
        current = {"key": CURRENT, "revision": TARGET, "date": "2026-11-17T22:10:00+0000"}
        history = analyses([current, previous]);task = {"task": {"status": "SUCCESS", "componentKey": "cq", "analysisId": CURRENT}}
        result = BASE.current_receipt(baseline(), task, history, SERVER)
        self.assertFalse(result["current_qualified"])
        self.assertEqual(TARGET_TREE, result["current_tree"])
        for selected in (history | {"analyses": [previous, current]}, analyses([current]), analyses([current, previous, {}])):
            with self.assertRaises(ValueError):
                BASE.current_receipt(baseline(), task, selected, SERVER)
        for key, value in (("revision", BASE.REVISION), ("key", ANALYSIS), ("date", "2026-11-17T21:59:00+0000")):
            changed = copy.deepcopy(history);changed["analyses"][0][key] = value
            with self.assertRaises(ValueError):
                BASE.current_receipt(baseline(), task, changed, SERVER)
        with self.assertRaises(ValueError):
            BASE.current_receipt(baseline(), task, history, SERVER | {"id": "different-server"})

    def test_new_code_counts_require_both_positive_denominators_not_missing_as_zero(self):
        current = {"schema": "cq-sonar-current-baseline-correspondence/1", "current_qualified": False,
                   "current_analysis_uuid": CURRENT}
        value = {"component": {"key": "cq", "measures": [
            {"metric": "new_lines", "period": {"index": 1, "value": "3"}},
            {"metric": "new_lines_to_cover", "period": {"value": "2"}},
            {"metric": "new_conditions_to_cover", "period": {"value": "0"}},
        ]}}
        result = BASE.new_denominators(value, current)
        self.assertEqual({"new_lines": 3, "new_lines_to_cover": 2, "new_conditions_to_cover": 0}, result["counts"])
        self.assertFalse(result["current_qualified"])
        for index, replacement in ((0, {"metric": "new_lines", "period": {"value": "0"}}),
                                    (1, {"metric": "new_lines_to_cover", "period": {"value": "0"}}),
                                    (0, {"metric": "new_lines", "value": "3"}),
                                    (0, {"metric": "new_lines", "period": {"index": True, "value": "3"}}),
                                    (0, {"metric": "new_lines", "period": {"value": "NaN"}}),
                                    (0, {"metric": "new_lines_to_cover", "period": {"value": "3"}})):
            changed = copy.deepcopy(value);changed["component"]["measures"][index] = replacement
            with self.subTest(index=index), self.assertRaises(ValueError):
                BASE.new_denominators(changed, current)
        with self.assertRaises(ValueError):
            BASE.new_denominators({"component": {"key": "cq", "measures": []}}, current)
        with self.assertRaises(ValueError):
            BASE.new_denominators(value, current | {"current_analysis_uuid": BASE.REVISION})
        self.call(["denominators", self.file("counts", value), self.file("current-counts", current), str(self.root / "denominators")])

    def test_admin_config_is_existing_fake_protected_credential_not_a_permission_grant(self):
        fake = b"Cq9!" + b"A" * 43 + b"\n"
        self.assertEqual(b'user = "admin:' + fake.rstrip(b"\n") + b'"\n', BASE.admin_config(fake))
        for value in (b"", b"admin", fake + b"\n", fake.replace(b"A", b'"', 1)):
            with self.assertRaises(ValueError):
                BASE.admin_config(value)

    def test_cli_all_success_paths_are_owned_data_or_mocked_git_not_live_analysis(self):
        server = self.file("server", SERVER)
        source = self.file("source", identity())
        with mock.patch.object(BASE, "source_identity", return_value=identity()) as prepare:
            self.call(["prepare", str(self.root), TARGET, server, str(self.root / "prepared"), "e" * 64, "f" * 64], stdout="2026-11-17\n")
            prepare.assert_called_once_with(str(self.root), TARGET)
        with mock.patch.object(BASE, "verify_worktree") as verify:
            self.call(["verify", "owned-root"]);verify.assert_called_once_with("owned-root")
        self.call(["empty", self.file("empty", analyses([]))])
        self.call(["branch", self.file("branches", {"branches": [{"name": "main", "isMain": True, "type": "BRANCH"}]})], stdout="main\n")
        self.call(["admin", self.file("password", b"Cq9!" + b"A" * 43 + b"\n"), str(self.root / "curl-config")])
        args = baseline_inputs()
        metadata = self.file("task-metadata", args[1]);task = self.file("task", args[2]);history = self.file("history", args[3]);gate = self.file("gate", args[4]);record = str(self.root / "baseline")
        self.call(["analysis", source, metadata, task, history, gate, "3", "main", record, "unqualified"], stdout=ANALYSIS + "\n")
        self.call(["period", self.file("period", period()), record, server, str(self.root / "period-receipt")])
        now = {"key": CURRENT, "revision": TARGET, "date": "2026-11-17T22:10:00+0000"}
        current_task = self.file("current-task", {"task": {"status": "SUCCESS", "componentKey": "cq", "analysisId": CURRENT}})
        self.call(["current", record, current_task, self.file("current-history", analyses([now, args[3]["analyses"][0]])), server, str(self.root / "current")])
        self.assertFalse(json.loads((self.root / "current").read_text())["current_qualified"])

    def test_cli_refusals_are_fixed_and_script_entry_executes_only_owned_empty_check(self):
        with self.assertRaises(ValueError):
            BASE.execute("unknown", [])
        for args in ([], ["unknown", "synthetic-private-argument"], ["empty"], ["empty", "synthetic-missing-path"]):
            self.call(args, 2)
        with mock.patch.object(BASE, "read_owned", side_effect=RecursionError("synthetic-private-detail")):
            self.call(["empty", "synthetic-private-path"], 2)
        path = self.file("entry-empty", analyses([]))
        with mock.patch.object(sys, "argv", [str(SCRIPT), "empty", path]), \
                contextlib.redirect_stdout(io.StringIO()) as out, contextlib.redirect_stderr(io.StringIO()) as err:
            with self.assertRaises(SystemExit) as caught:
                runpy.run_path(str(SCRIPT), run_name="__main__")
        self.assertEqual(0, caught.exception.code)
        self.assertEqual("", out.getvalue())
        self.assertEqual("", err.getvalue())


if __name__ == "__main__":
    unittest.main()
