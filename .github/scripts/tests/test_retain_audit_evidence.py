"""Synthetic disclosure, identity, file-boundary and retention-only contracts."""

from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "retain_audit_evidence.py"
SPEC = importlib.util.spec_from_file_location("retain_audit_evidence", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
RETAIN = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(RETAIN)


class RetentionContracts(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.source = self.root / "source"
        self.source.mkdir(mode=0o700)
        (self.source / "owned.go").write_text("package example\n")
        self.git("init", "-q")
        self.git("add", "owned.go")
        self.git("-c", "user.name=Synthetic", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "commit", "-qm", "Synthetic source")
        self.commit = self.git("rev-parse", "HEAD").decode().strip()
        self.audit = self.root / "audit"
        self.audit.mkdir(mode=0o700)
        self.output = self.root / "retained"
        self.coverage = b"mode: atomic\ngithub.com/maldous/CircularQuay/owned.go:1.1,1.9 1 0\n"
        self.report = {
            "version": 1, "profile": "source", "commit": self.commit,
            "passed": False, "production_qualified": False,
            "checks": [{"name": "coverage", "status": "failed", "required": True,
                        "code": "coverage_below_threshold", "diagnostic": "must-not-retain"}],
            "source": {"commit": self.commit, "tree": self.git("rev-parse", "HEAD^{tree}").decode().strip(),
                       "manifest_sha256": "b" * 64, "files": ["must-not-retain"]},
            "tool_manifest": {"path": "/private/must-not-retain"},
            "limitations": ["must-not-retain"],
        }
        self.write_inputs()

    def git(self, *arguments: str) -> bytes:
        return subprocess.check_output(
            ["git", "-c", "core.hooksPath=" + os.devnull, "-C", str(self.source), *arguments],
            stderr=subprocess.DEVNULL,
        )

    def write_inputs(self) -> None:
        (self.audit / "coverage.out").write_bytes(self.coverage)
        (self.audit / "report.json").write_text(json.dumps(self.report))
        for path in self.audit.iterdir():
            path.chmod(0o600)

    def retain(self) -> dict:
        return RETAIN.retain(self.source, self.audit, self.output, self.commit)

    def test_retains_failed_result_without_raw_diagnostics_or_private_paths(self) -> None:
        result = self.retain()
        self.assertFalse(result["audit_passed"])
        self.assertEqual({"coverage.out", "metadata.json"}, {p.name for p in self.output.iterdir()})
        data = (self.output / "metadata.json").read_bytes()
        self.assertNotIn(b"must-not-retain", data)
        metadata = json.loads(data)
        self.assertTrue(metadata["retention_only"])
        self.assertFalse(metadata["production_qualified"])
        self.assertEqual(self.commit, metadata["commit"])
        self.assertEqual(64, len(metadata["coverage_sha256"]))
        self.assertEqual(self.coverage, (self.output / "coverage.out").read_bytes())
        self.assertEqual(0o700, self.output.stat().st_mode & 0o777)
        self.assertEqual(0o600, (self.output / "coverage.out").stat().st_mode & 0o777)

    def test_does_not_recompute_or_promote_original_pass_flag(self) -> None:
        self.report["passed"] = True
        self.report["checks"][0]["status"] = "passed"
        self.write_inputs()
        self.assertTrue(self.retain()["audit_passed"])

    def test_rejects_mismatched_commit(self) -> None:
        self.report["commit"] = "c" * 40
        self.write_inputs()
        with self.assertRaises(RETAIN.InvalidEvidence):
            self.retain()
        self.assertFalse(self.output.exists())

    def test_rejects_dirty_source_and_wrong_source_revision(self) -> None:
        (self.source / "owned.go").write_text("package changed\n")
        with self.assertRaises(RETAIN.InvalidEvidence):
            self.retain()
        with self.assertRaises(RETAIN.InvalidEvidence):
            RETAIN.tracked_source(self.source, "c" * 40)

    def test_foreign_git_environment_cannot_hide_dirty_selected_source(self) -> None:
        other = self.root / "other"
        subprocess.run(["git", "clone", "-q", str(self.source), str(other)], check=True)
        (self.source / "owned.go").write_text("package changed\n")
        with mock.patch.dict(os.environ, {"GIT_DIR": str(other / ".git"),
                                         "GIT_WORK_TREE": str(other),
                                         "GIT_INDEX_FILE": str(other / ".git/index")}):
            with self.assertRaises(RETAIN.InvalidEvidence):
                self.retain()
        self.assertFalse(self.output.exists())

    def test_rejects_report_tree_unrelated_to_verified_commit(self) -> None:
        self.report["source"]["tree"] = "a" * 40
        self.write_inputs()
        with self.assertRaises(RETAIN.InvalidEvidence):
            self.retain()
        self.assertFalse(self.output.exists())

    def test_rejects_pass_claim_with_a_required_failed_check(self) -> None:
        self.report["passed"] = True
        with self.assertRaises(RETAIN.InvalidEvidence):
            RETAIN.report_metadata(json.dumps(self.report).encode(), self.commit)

    def test_rejects_unowned_or_untracked_profile_paths(self) -> None:
        for name in ("/private/secret.go", "example.invalid/outside.go",
                     RETAIN.MODULE + "../owned.go", RETAIN.MODULE + "absent.go"):
            with self.subTest(name=name):
                self.coverage = f"mode: atomic\n{name}:1.1,1.9 1 0\n".encode()
                self.write_inputs()
                with self.assertRaises(RETAIN.InvalidEvidence):
                    self.retain()

    def test_rejects_bad_coverage_grammar_and_bounds(self) -> None:
        rows = (b"mode: other\n", b"mode: set\n", b"mode: atomic\nraw diagnostic\n",
                ("mode: atomic\n" + RETAIN.MODULE + "owned.go:0.1,1.9 1 0\n").encode(),
                ("mode: atomic\n" + RETAIN.MODULE + "owned.go:2.1,1.9 1 0\n").encode(),
                ("mode: atomic\n" + RETAIN.MODULE + "owned.go:1.1,1.9 1 18446744073709551616\n").encode())
        for content in rows:
            with self.subTest(content=content):
                with self.assertRaises(RETAIN.InvalidEvidence):
                    RETAIN.validate_coverage(content, {"owned.go"})

    def test_rejects_duplicate_keys_and_nonfinite_report_values(self) -> None:
        for content in (b'{"version":1,"version":1}', b'{"value":NaN}'):
            with self.subTest(content=content):
                with self.assertRaises(RETAIN.InvalidEvidence):
                    RETAIN.report_metadata(content, self.commit)

    def test_rejects_invalid_report_types_and_qualification_claims(self) -> None:
        for field, value in (("passed", 1), ("version", True), ("production_qualified", True),
                             ("profile", "enterprise"), ("checks", "raw diagnostic")):
            with self.subTest(field=field):
                changed = dict(self.report, **{field: value})
                with self.assertRaises(RETAIN.InvalidEvidence):
                    RETAIN.report_metadata(json.dumps(changed).encode(), self.commit)

    def test_rejects_invalid_check_and_source_identity_fields(self) -> None:
        for check in ({"name": "private/path", "status": "failed", "required": True},
                      {"name": "coverage", "status": "unknown", "required": True},
                      {"name": "coverage", "status": [], "required": True},
                      {"name": "coverage", "status": "failed", "required": 1}):
            with self.subTest(check=check):
                with self.assertRaises(RETAIN.InvalidEvidence):
                    RETAIN.safe_check(check)
        with self.assertRaises(RETAIN.InvalidEvidence):
            RETAIN.source_identity({"commit": self.commit, "tree": "private/path"}, self.commit)

    def test_bounds_files_without_creating_output(self) -> None:
        with mock.patch.object(RETAIN, "MAX_REPORT", 16):
            with self.assertRaises(RETAIN.InvalidEvidence):
                self.retain()
        with mock.patch.object(RETAIN, "MAX_COVERAGE", 16):
            with self.assertRaises(RETAIN.InvalidEvidence):
                self.retain()
        self.assertFalse(self.output.exists())

    def test_rejects_symlink_and_hardlink_inputs(self) -> None:
        path = self.audit / "coverage.out"
        path.unlink()
        target = self.root / "original"
        target.write_bytes(self.coverage)
        target.chmod(0o600)
        path.symlink_to(target)
        with self.assertRaises(OSError):
            self.retain()
        path.unlink()
        os.link(target, path)
        with self.assertRaises(RETAIN.InvalidEvidence):
            self.retain()

    def test_rejects_fifo_without_blocking(self) -> None:
        path = self.audit / "coverage.out"
        path.unlink()
        os.mkfifo(path, 0o600)
        with self.assertRaises(RETAIN.InvalidEvidence):
            self.retain()

    def test_rejects_shared_permissions_and_symlink_directory(self) -> None:
        self.audit.chmod(0o755)
        with self.assertRaises(RETAIN.InvalidEvidence):
            self.retain()
        self.audit.chmod(0o700)
        alias = self.root / "alias"
        alias.symlink_to(self.audit, target_is_directory=True)
        with self.assertRaises(RETAIN.InvalidEvidence):
            RETAIN.retain(self.source, alias, self.output, self.commit)

    def test_never_overwrites_output_or_writes_inside_source(self) -> None:
        self.output.mkdir(mode=0o700)
        sentinel = self.output / "keep"
        sentinel.write_text("unchanged")
        with self.assertRaises(FileExistsError):
            self.retain()
        self.assertEqual("unchanged", sentinel.read_text())
        with self.assertRaises(RETAIN.InvalidEvidence):
            RETAIN.retain(self.source, self.audit, self.source / "new", self.commit)

    def test_cli_preserves_retention_only_success_and_constant_failure(self) -> None:
        command = [sys.executable, str(SCRIPT), "--source", str(self.source),
                   "--audit", str(self.audit), "--output", str(self.output),
                   "--expected-commit", self.commit]
        result = subprocess.run(command, capture_output=True, text=True, check=False)
        self.assertEqual(0, result.returncode)
        self.assertFalse(json.loads(result.stdout)["audit_passed"])
        failed = subprocess.run(command, capture_output=True, text=True, check=False)
        self.assertEqual(1, failed.returncode)
        self.assertEqual("Bounded audit evidence was not retained.\n", failed.stdout)
        self.assertEqual("", failed.stderr)


if __name__ == "__main__":
    unittest.main()
