"""Direct CLI/report boundary contracts; no Git command or external audit runs."""

from __future__ import annotations

import contextlib
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import runpy
import sys
import tempfile
import unittest
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "retain_audit_evidence.py"
SPEC = importlib.util.spec_from_file_location("retain_completion", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
RETAIN = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(RETAIN)


class RetentionCompletionContracts(unittest.TestCase):
    def setUp(self) -> None:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve(strict=True)
        self.source = self.root / "source"
        self.source.mkdir(mode=0o700)
        self.audit = self.root / "audit"
        self.audit.mkdir(mode=0o700)
        self.output = self.root / "retained"
        self.commit, self.tree = "1" * 40, "2" * 40
        self.coverage = ("mode: atomic\n" + RETAIN.MODULE + "owned.go:1.1,1.9 1 0\n").encode()
        self.report = {"version": 1, "profile": "source", "commit": self.commit,
                       "passed": False, "production_qualified": False,
                       "checks": [{"name": "statement_coverage", "status": "failed", "required": True}]}
        self.write_inputs()

    def write_inputs(self) -> None:
        for name, value in (("coverage.out", self.coverage), ("report.json", json.dumps(self.report).encode())):
            target = self.audit / name
            target.write_bytes(value)
            target.chmod(0o600)

    def argv(self) -> list[str]:
        return [str(SCRIPT), "--source", str(self.source), "--audit", str(self.audit),
                "--output", str(self.output), "--expected-commit", self.commit]

    def git_observation(self, source: Path, *arguments: str) -> bytes:
        self.assertEqual(self.source, source)
        responses = {("rev-parse", "--show-toplevel"): str(self.source).encode(),
                     ("rev-parse", "HEAD"): self.commit.encode(), ("rev-parse", "HEAD^{tree}"): self.tree.encode(),
                     ("status", "--porcelain", "--untracked-files=no"): b"",
                     ("ls-files", "-z", "--", "*.go"): b"owned.go\0"}
        self.assertIn(arguments, responses)
        return responses[arguments]

    def test_unavailable_probe_shapes_do_not_fabricate_a_tool_digest(self) -> None:
        for value in (None, [], {}, {"tools": None}, {"tools": "not-a-list"}):
            with self.subTest(value=value):
                self.assertIsNone(RETAIN.selected_go_probe(value))
                result = RETAIN.source_runtime({"go_version": "go1.27.4", "platform": "linux/amd64",
                                                "tool_manifest": value})
                self.assertEqual({"go_version": "go1.27.4", "platform": "linux/amd64"}, result)
        self.assertIsNone(RETAIN.selected_go_probe({"tools": []}))
        self.assertIsNone(RETAIN.selected_go_probe({"tools": [{"id": "other"}]}))

    def test_report_without_source_keeps_original_failure_without_inventing_lineage(self) -> None:
        metadata = RETAIN.report_metadata(json.dumps(self.report).encode(), self.commit)
        self.assertNotIn("source", metadata)
        self.assertNotIn("go_tool_sha256", metadata)
        self.assertEqual(False, metadata["passed"])
        self.assertEqual(self.report["checks"], metadata["checks"])

    def test_direct_main_retains_real_owned_bytes_and_constant_repeat_refusal(self) -> None:
        report_bytes = (self.audit / "report.json").read_bytes()
        captured = io.StringIO()
        with mock.patch.object(RETAIN, "git_output", side_effect=self.git_observation) as git, \
                mock.patch.object(RETAIN.subprocess, "run", side_effect=AssertionError("no process")) as process, \
                mock.patch.object(sys, "argv", self.argv()), contextlib.redirect_stdout(captured):
            self.assertEqual(0, RETAIN.main())
            self.assertEqual(1, RETAIN.main())
        lines = captured.getvalue().splitlines()
        self.assertEqual({"retained_files": 2, "commit": self.commit, "audit_passed": False}, json.loads(lines[0]))
        self.assertEqual("Bounded audit evidence was not retained.", lines[1])
        self.assertGreaterEqual(git.call_count, 9)
        process.assert_not_called()
        metadata = json.loads((self.output / "metadata.json").read_bytes())
        self.assertEqual(hashlib.sha256(report_bytes).hexdigest(), metadata["original_report_sha256"])
        self.assertEqual(hashlib.sha256(self.coverage).hexdigest(), metadata["coverage_sha256"])
        self.assertEqual(self.coverage, (self.output / "coverage.out").read_bytes())
        self.assertTrue(metadata["retention_only"])
        self.assertFalse(metadata["passed"])
        self.assertFalse(metadata["production_qualified"])
        self.assertNotIn("source", metadata)
        self.assertEqual(report_bytes, (self.audit / "report.json").read_bytes())
        self.assertNotIn(str(self.root), captured.getvalue())

    def test_script_entry_invalid_commit_refuses_before_subprocess_or_output(self) -> None:
        arguments = self.argv()
        arguments[-1] = "invalid-commit"
        captured = io.StringIO()
        with mock.patch.object(sys, "argv", arguments), \
                mock.patch.object(RETAIN.subprocess, "run", side_effect=AssertionError("no process")) as child, \
                contextlib.redirect_stdout(captured), self.assertRaises(SystemExit) as stopped:
            runpy.run_path(str(SCRIPT), run_name="__main__")
        self.assertEqual(1, stopped.exception.code)
        self.assertEqual("Bounded audit evidence was not retained.\n", captured.getvalue())
        child.assert_not_called()
        self.assertFalse(self.output.exists())
        self.assertEqual(self.coverage, (self.audit / "coverage.out").read_bytes())


if __name__ == "__main__":
    unittest.main()
