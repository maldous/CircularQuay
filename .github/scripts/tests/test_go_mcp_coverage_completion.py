"""Complete reachable producer/CLI edges with owned bytes, never a real Go call.

Source and Go identity observations are supplied at their existing per-call
seams.  These tests do not authenticate a real producer or qualify a join.
"""

from __future__ import annotations

import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import runpy
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "go_mcp_coverage.py"
SPEC = importlib.util.spec_from_file_location("go_mcp_completion", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
COVER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COVER)


class GoMCPCompletionContracts(unittest.TestCase):
    def setUp(self) -> None:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve(strict=True)
        self.source = self.root / "source"
        self.source.mkdir(mode=0o700)
        self.binary = self.root / "synthetic-binary"
        self.binary.write_bytes(b"synthetic opaque bytes; never executed")
        self.binary.chmod(0o700)
        self.commit = "1" * 40
        self.state = {"commit": self.commit, "tree": "2" * 40,
                      "tracked_clean": True, "locks": {"go.mod": "3" * 64, "go.sum": "4" * 64}}
        self.tracked = {"sdk/go/client.go", "cmd/cq/main.go"}
        self.tool = {"version": COVER.GO_VERSION, "platform": COVER.PLATFORM, "sha256": "5" * 64}
        self.origin = {"repository": COVER.REPOSITORY, "pr_head": "6" * 40,
                       "run_id": "123", "attempt": 1, "job": "typescript",
                       "matrix": "ubuntu-24.04/node24.22.2", "step": COVER.STEP}

    def write(self, path: Path, value: bytes) -> None:
        path.write_bytes(value)
        path.chmod(0o600)

    def producer_argv(self, action: str, output: Path) -> list[str]:
        return [str(SCRIPT), action, "--source", str(self.source), "--output", str(output),
                "--go-binary", str(self.binary), "--cq-binary", str(self.binary),
                "--expected-commit", self.commit, "--pr-head", self.origin["pr_head"],
                "--run-id", "123", "--job-key", "typescript", "--matrix", self.origin["matrix"],
                "--attempt", "1"]

    def test_go_probe_exact_environment_success_and_real_post_probe_drift(self) -> None:
        response = SimpleNamespace(returncode=0, stderr=b"", stdout=(
            f"go version {COVER.GO_VERSION} {COVER.PLATFORM}\n").encode())
        before = COVER.regular_digest(self.binary)
        with mock.patch.object(COVER.subprocess, "run", return_value=response) as probe:
            result = COVER.go_identity(self.binary)
        self.assertEqual(dict(self.tool, sha256=before[0]), result)
        probe.assert_called_once_with([str(self.binary), "version"],
                                      env={"PATH": os.defpath, "GOTOOLCHAIN": "local", "GOENV": "off"},
                                      check=False, capture_output=True, timeout=30)

        def mutate(*_args, **_kwargs):
            self.binary.write_bytes(b"different real owned bytes")
            return response

        with mock.patch.object(COVER.subprocess, "run", side_effect=mutate):
            with self.assertRaises(COVER.InvalidEvidence):
                COVER.go_identity(self.binary)
        self.assertEqual(b"different real owned bytes", self.binary.read_bytes())

    def test_new_bundle_rolls_back_only_its_files_on_first_or_later_open_failure(self) -> None:
        original_open = os.open
        sentinel = self.root / "keep"
        sentinel.write_bytes(b"unchanged")
        for failed_name in ("first", "second"):
            output = self.root / ("failed-" + failed_name)

            def fail_selected(path, flags, mode=0o777):
                if Path(path) == output / failed_name:
                    raise OSError("synthetic private failure; must not be retained")
                return original_open(path, flags, mode)

            with self.subTest(failed_name=failed_name):
                with mock.patch.object(COVER.os, "open", side_effect=fail_selected):
                    with self.assertRaisesRegex(OSError, "synthetic private failure"):
                        COVER.write_bundle(output, {"first": b"one", "second": b"two"}, (self.source,))
                self.assertFalse(output.exists())
                self.assertEqual(b"unchanged", sentinel.read_bytes())

    def test_pass_claim_requires_all_required_checks_but_not_optional_success(self) -> None:
        names = ("format", "unit_tests", "go_vet", "go_toolchain", "source_revision",
                 "source_identity_before_checks", "source_identity_after_checks")
        checks = [{"name": name, "status": "passed", "required": True} for name in names]
        checks.append({"name": "optional", "status": "failed", "required": False})
        self.assertEqual(checks, COVER.baseline_checks(checks, True))
        failed = checks + [{"name": "statement_coverage", "status": "failed", "required": True}]
        with self.assertRaises(COVER.InvalidEvidence):
            COVER.baseline_checks(failed, True)
        self.assertEqual(failed, COVER.baseline_checks(failed, False))

    def test_real_begin_and_retain_cli_keep_failed_conformance_status(self) -> None:
        preflight_output = self.root / "preflight"
        retained_output = self.root / "retained"
        coverage_directory = self.root / "coverage"
        coverage_directory.mkdir(mode=0o700)
        coverage = coverage_directory / "coverage.out"
        self.write(coverage, ("mode: atomic\n" + COVER.SDK + "client.go:1.1,1.9 1 1\n").encode())
        with mock.patch.object(COVER, "source_identity", return_value=(self.state, self.tracked)), \
                mock.patch.object(COVER, "go_identity", return_value=self.tool), \
                mock.patch.object(COVER.subprocess, "run", side_effect=AssertionError("no process")):
            for action, output in (("begin", preflight_output), ("retain", retained_output)):
                arguments = self.producer_argv(action, output)
                if action == "retain":
                    arguments += ["--coverage", str(coverage), "--preflight",
                                  str(preflight_output / "preflight.json"), "--exit-code", "7"]
                captured = io.StringIO()
                with self.subTest(action=action), mock.patch.object(sys, "argv", arguments), \
                        contextlib.redirect_stdout(captured):
                    self.assertEqual(0, COVER.main())
                self.assertEqual({"action": action, "evidence_written": True, "global_gate_passed": False},
                                 json.loads(captured.getvalue()))
        before = json.loads((preflight_output / "preflight.json").read_bytes())
        self.assertEqual("sdk-go-mcp-preflight", before["kind"])
        metadata = json.loads((retained_output / "sdk-metadata.json").read_bytes())
        self.assertEqual(7, metadata["test"]["exit_code"])
        self.assertFalse(metadata["production_qualified"])
        self.assertEqual(COVER.digest((preflight_output / "preflight.json").read_bytes()),
                         metadata["preflight_sha256"])
        self.assertEqual(coverage.read_bytes(), (retained_output / "coverage.out").read_bytes())
        self.assertEqual({"sdk-metadata.json", "coverage.out"}, {p.name for p in retained_output.iterdir()})

    def join_fixture(self, covered: bool) -> tuple[list[str], Path]:
        inputs = self.root / ("inputs-" + str(covered))
        inputs.mkdir(mode=0o700)
        baseline = ("mode: atomic\n" + COVER.SDK + "client.go:1.1,1.9 2 0\n" +
                    COVER.MODULE + "cmd/cq/main.go:1.1,1.9 3 1\n").encode()
        sdk = ("mode: atomic\n" + COVER.SDK + f"client.go:1.1,1.9 2 {int(covered)}\n").encode()
        names = ("format", "unit_tests", "go_vet", "go_toolchain", "source_revision",
                 "source_identity_before_checks", "source_identity_after_checks")
        checks = [{"name": name, "status": "passed", "required": True} for name in names]
        original = {"version": 1, "profile": "source", "commit": self.commit, "passed": False,
                    "production_qualified": False, "checks": checks, "retention_only": True,
                    "source": {"commit": self.commit, "tree": self.state["tree"], "manifest_sha256": "7" * 64},
                    "original_report_sha256": "8" * 64, "coverage_sha256": COVER.digest(baseline),
                    "coverage_semantics": "Original audit result; not recomputed or promoted by retention",
                    "go_version": self.tool["version"], "platform": self.tool["platform"],
                    "go_tool_sha256": self.tool["sha256"]}
        binary_sha, binary_bytes = COVER.regular_digest(self.binary)
        binary = {"sha256": binary_sha, "bytes": binary_bytes}
        supplement = {"version": 1, "kind": "sdk-go-mcp", "source": self.state,
                      "toolchain": self.tool, "binary": binary, "producer": self.origin,
                      "source_before_after_equal": True, "preflight_sha256": "9" * 64,
                      "test": {"package": COVER.PACKAGE, "name": COVER.TEST, "tags": "integration",
                               "count": 1, "exit_code": 0},
                      "coverage": {"mode": "atomic", "bytes": len(sdk), "sha256": COVER.digest(sdk)},
                      "production_qualified": False}
        contents = {"baseline_coverage": baseline, "baseline_metadata": COVER.encoded(original),
                    "sdk_coverage": sdk, "sdk_metadata": COVER.encoded(supplement)}
        pins = {"version": 1, "baseline_commit": self.commit, "sdk_commit": self.commit,
                "tree": self.state["tree"], "producer": self.origin, "toolchain": self.tool,
                "binary": binary, "locks": self.state["locks"],
                "artifacts": {name: COVER.digest(value) for name, value in contents.items()}}
        output = self.root / ("joined-" + str(covered))
        arguments = [str(SCRIPT), "join", "--source", str(self.source), "--output", str(output)]
        for name, value in contents.items():
            self.write(inputs / name, value)
            arguments += ["--" + name.replace("_", "-"), str(inputs / name)]
        pins_path = inputs / "pins.json"
        self.write(pins_path, COVER.encoded(pins))
        arguments += ["--pins", str(pins_path), "--expected-pins-sha256", COVER.digest(pins_path.read_bytes())]
        return arguments, output

    def test_real_join_cli_distinguishes_strict_shortfall_without_promoting_original(self) -> None:
        for covered in (False, True):
            arguments, output = self.join_fixture(covered)
            captured = io.StringIO()
            with self.subTest(covered=covered), \
                    mock.patch.object(COVER, "source_identity", return_value=(self.state, self.tracked)), \
                    mock.patch.object(COVER.subprocess, "run", side_effect=AssertionError("no process")), \
                    mock.patch.object(sys, "argv", arguments), contextlib.redirect_stdout(captured):
                self.assertEqual(0 if covered else 1, COVER.main())
            result = json.loads((output / "composite.json").read_bytes())
            self.assertEqual(5, result["coverage"]["total_statements"])
            self.assertEqual(5 if covered else 3, result["coverage"]["covered_statements"])
            self.assertFalse(result["global_gate_passed"])
            self.assertFalse(result["baseline_passed"])
            self.assertFalse(result["production_qualified"])
            self.assertEqual({"action": "join", "evidence_written": True, "global_gate_passed": False},
                             json.loads(captured.getvalue()))

    def test_script_entry_missing_required_arguments_refuses_before_any_process(self) -> None:
        captured = io.StringIO()
        with mock.patch.object(sys, "argv", [str(SCRIPT), "join"]), \
                mock.patch.object(COVER.subprocess, "run", side_effect=AssertionError("no process")) as child, \
                contextlib.redirect_stderr(captured):
            with self.assertRaises(SystemExit) as stopped:
                runpy.run_path(str(SCRIPT), run_name="__main__")
        self.assertEqual(2, stopped.exception.code)
        self.assertIn("required", captured.getvalue())
        child.assert_not_called()
        self.assertFalse(any(path.name.startswith("joined-") for path in self.root.iterdir()))


if __name__ == "__main__":
    unittest.main()
