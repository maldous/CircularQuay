"""Synthetic positive/negative provenance, denominator and privacy oracles only."""

from __future__ import annotations

import argparse
import copy
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
from types import SimpleNamespace
import unittest
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "go_mcp_coverage.py"
SPEC = importlib.util.spec_from_file_location("go_mcp_coverage", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
COVER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COVER)


class GoMCPContracts(unittest.TestCase):
    def setUp(self) -> None:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve(strict=True)
        self.source = self.root / "source"
        self.source.mkdir(mode=0o700)
        for path in ("sdk/go/client.go", "sdk/go/local.go", "cmd/cq/main.go"):
            target = self.source / path
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text("package example\n")
        (self.source / "go.mod").write_text("module github.com/maldous/CircularQuay\n")
        (self.source / "go.sum").write_text("synthetic-lock\n")
        self.git("init", "-q")
        self.git("add", ".")
        self.git("-c", "user.name=Synthetic", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "commit", "-qm", "Synthetic source")
        self.commit = self.git("rev-parse", "HEAD").decode().strip()
        self.state, self.tracked = COVER.source_identity(self.source, self.commit)
        self.tool = {"version": COVER.GO_VERSION, "platform": COVER.PLATFORM, "sha256": "d" * 64}
        self.binary = self.root / "synthetic-cq"
        self.binary.write_bytes(b"synthetic-not-a-real-cq-binary")
        self.binary.chmod(0o700)
        binary_sha, binary_size = COVER.regular_digest(self.binary)
        self.binary_identity = {"sha256": binary_sha, "bytes": binary_size}
        self.origin = {"repository": COVER.REPOSITORY, "pr_head": "f" * 40, "run_id": "100",
                       "attempt": 1, "job": "typescript", "matrix": "ubuntu-24.04/node24.22.2",
                       "step": COVER.STEP}
        self.paths = {}
        for label in ("baseline", "sdk", "pins", "preflight"):
            (self.root / label).mkdir(mode=0o700)
        for name in ("baseline_coverage", "baseline_metadata", "sdk_coverage", "sdk_metadata"):
            self.paths[name] = self.root / name.split("_")[0] / name
        self.pins_path = self.root / "pins" / "pins.json"
        self.output = self.root / "composite"
        self.baseline_bytes = ("mode: atomic\n" + COVER.SDK + "client.go:1.1,1.9 2 0\n" +
                               COVER.SDK + "local.go:1.1,1.9 3 1\n" +
                               COVER.MODULE + "cmd/cq/main.go:1.1,1.9 4 1\n").encode()
        self.sdk_bytes = ("mode: atomic\n" + COVER.SDK + "client.go:1.1,1.9 2 7\n" +
                          COVER.SDK + "local.go:1.1,1.9 3 0\n").encode()
        names = ("format", "unit_tests", "go_vet", "go_toolchain", "source_revision",
                 "source_identity_before_checks", "source_identity_after_checks")
        checks = [{"name": name, "status": "passed", "required": True} for name in names]
        checks += [{"name": "statement_coverage", "status": "failed", "required": True},
                   {"name": "tagged_integration", "status": "not_run", "required": True}]
        self.baseline = {"version": 1, "profile": "source", "commit": self.commit, "passed": False,
                         "production_qualified": False, "retention_only": True, "checks": checks,
                         "source": {"commit": self.commit, "tree": self.state["tree"], "manifest_sha256": "b" * 64},
                         "original_report_sha256": "c" * 64, "coverage_sha256": COVER.digest(self.baseline_bytes),
                         "coverage_semantics": "Original audit result; not recomputed or promoted by retention",
                         "go_version": COVER.GO_VERSION, "platform": COVER.PLATFORM, "go_tool_sha256": "d" * 64}
        self.sdk = {"version": 1, "kind": "sdk-go-mcp", "source": self.state, "toolchain": self.tool,
                    "binary": self.binary_identity, "producer": self.origin,
                    "source_before_after_equal": True, "preflight_sha256": "a" * 64,
                    "test": {"package": COVER.PACKAGE, "name": COVER.TEST, "tags": "integration", "count": 1, "exit_code": 0},
                    "coverage": {"mode": "atomic", "bytes": len(self.sdk_bytes), "sha256": COVER.digest(self.sdk_bytes)},
                    "production_qualified": False}
        self.pins = {"version": 1, "baseline_commit": self.commit, "sdk_commit": self.commit,
                     "tree": self.state["tree"], "producer": self.origin, "toolchain": self.tool,
                     "binary": self.binary_identity, "locks": self.state["locks"], "artifacts": {}}
        self.write_inputs()

    def git(self, *arguments: str) -> bytes:
        return subprocess.check_output(["git", "-c", "core.hooksPath=" + os.devnull,
                                        "-C", str(self.source), *arguments], stderr=subprocess.DEVNULL)

    def write(self, path: Path, content: bytes) -> None:
        path.write_bytes(content)
        path.chmod(0o600)

    def write_inputs(self) -> None:
        self.baseline["coverage_sha256"] = COVER.digest(self.baseline_bytes)
        self.sdk["coverage"] = {"mode": "atomic", "bytes": len(self.sdk_bytes), "sha256": COVER.digest(self.sdk_bytes)}
        contents = {"baseline_coverage": self.baseline_bytes, "sdk_coverage": self.sdk_bytes,
                    "baseline_metadata": COVER.encoded(self.baseline), "sdk_metadata": COVER.encoded(self.sdk)}
        for name, content in contents.items():
            self.write(self.paths[name], content)
        self.pins["artifacts"] = {name: COVER.digest(content) for name, content in contents.items()}
        self.write(self.pins_path, COVER.encoded(self.pins))
        self.pins_sha = COVER.digest(self.pins_path.read_bytes())

    def join(self) -> dict:
        return COVER.join(self.source, self.paths, self.output, self.pins_path, self.pins_sha)

    def test_union_preserves_every_baseline_coordinate_denominator_and_old_result(self) -> None:
        old_metadata = self.paths["baseline_metadata"].read_bytes()
        result = self.join()
        self.assertEqual(old_metadata, self.paths["baseline_metadata"].read_bytes())
        self.assertEqual({"coverage.out", "composite.json"}, {p.name for p in self.output.iterdir()})
        self.assertEqual(9, result["coverage"]["total_statements"])
        self.assertEqual(9, result["coverage"]["covered_statements"])
        self.assertTrue(result["coverage"]["coverage_gate_passed"])
        self.assertFalse(result["baseline_passed"])
        self.assertFalse(result["global_gate_passed"])
        self.assertFalse(result["source_report_promoted"])
        self.assertEqual(self.baseline["checks"], result["baseline_checks_unchanged"])
        merged = COVER.profile((self.output / "coverage.out").read_bytes(), self.tracked)
        original = COVER.profile(self.baseline_bytes, self.tracked)
        self.assertEqual(set(original), set(merged))
        self.assertEqual(0o600, (self.output / "composite.json").stat().st_mode & 0o777)

    def test_duplicate_coordinates_use_max_not_addition_and_do_not_double_denominator(self) -> None:
        self.sdk_bytes += (COVER.SDK + "client.go:1.1,1.9 2 3\n").encode()
        self.write_inputs()
        result = self.join()
        self.assertEqual(9, result["coverage"]["total_statements"])
        self.assertIn(b"client.go:1.1,1.9 2 7\n", (self.output / "coverage.out").read_bytes())

    def test_missing_pins_digest_never_defaults_to_untrusted_input_hash(self) -> None:
        for value in ("", "e" * 64):
            with self.subTest(value=value):
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.read_pins(self.pins_path, value)
        self.assertFalse(self.output.exists())

    def test_artifact_digest_mismatch_refuses_before_output(self) -> None:
        self.paths["sdk_coverage"].write_bytes(self.sdk_bytes + b"unexpected\n")
        with self.assertRaises(COVER.InvalidEvidence):
            self.join()
        self.assertFalse(self.output.exists())

    def test_wrong_source_tree_locks_or_toolchain_refuses_even_with_fresh_artifact_hashes(self) -> None:
        for field, value in (("tree", "a" * 40), ("toolchain", dict(self.tool, sha256="e" * 64)),
                             ("locks", {"go.mod": "e" * 64, "go.sum": "e" * 64})):
            with self.subTest(field=field):
                pins = copy.deepcopy(self.pins)
                pins[field] = value
                self.write(self.pins_path, COVER.encoded(pins))
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.join(self.source, self.paths, self.output, self.pins_path,
                               COVER.digest(self.pins_path.read_bytes()))
        self.assertFalse(self.output.exists())

    def test_failed_skipped_or_wrong_test_identity_never_supplies_coverage(self) -> None:
        original = copy.deepcopy(self.sdk)
        for field, value in (("exit_code", 1), ("exit_code", False), ("count", True),
                             ("name", "OtherTest"), ("tags", "")):
            with self.subTest(field=field, value=value):
                self.sdk = copy.deepcopy(original)
                self.sdk["test"][field] = value
                self.write_inputs()
                with self.assertRaises(COVER.InvalidEvidence):
                    self.join()
        self.assertFalse(self.output.exists())

    def test_wrong_producer_attempt_matrix_or_boolean_identity_refuses(self) -> None:
        for field, value in (("attempt", True), ("run_id", "101"), ("matrix", "windows/node24"),
                             ("repository", "other/repo"), ("step", "other_step")):
            with self.subTest(field=field):
                changed = copy.deepcopy(self.sdk)
                changed["producer"][field] = value
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.validate_sdk(changed, self.pins, self.sdk_bytes)

    def test_absent_runtime_or_source_before_after_identity_refuses(self) -> None:
        for field in ("go_version", "platform", "go_tool_sha256", "source"):
            with self.subTest(field=field):
                changed = copy.deepcopy(self.baseline)
                del changed[field]
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.validate_baseline(changed, self.pins)
        for value in (False, 1):
            changed = dict(self.sdk, source_before_after_equal=value)
            with self.assertRaises(COVER.InvalidEvidence):
                COVER.validate_sdk(changed, self.pins, self.sdk_bytes)

    def test_baseline_identity_format_unit_vet_and_go_checks_must_pass(self) -> None:
        for name in ("format", "unit_tests", "go_vet", "go_toolchain", "source_identity_before_checks", "source_identity_after_checks"):
            with self.subTest(name=name):
                changed = copy.deepcopy(self.baseline)
                next(c for c in changed["checks"] if c["name"] == name)["status"] = "not_run"
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.validate_baseline(changed, self.pins)

    def test_extra_missing_shifted_or_numstmt_changed_sdk_coordinates_refuse(self) -> None:
        original = COVER.profile(self.baseline_bytes, self.tracked)
        sdk = COVER.profile(self.sdk_bytes, self.tracked, sdk_only=True)
        key = next(iter(sdk))
        variants = [dict(sdk)]
        variants[0].pop(key)
        shifted = dict(sdk)
        shifted[(key[0], 2, 1, 2, 9)] = shifted.pop(key)
        variants.append(shifted)
        changed = dict(sdk)
        changed[key] = (99, 1)
        variants.append(changed)
        for rows in variants:
            with self.subTest(rows=rows):
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.joined_profile(original, rows)

    def test_rejects_foreign_test_paths_modes_counts_and_duplicate_numstmt(self) -> None:
        rows = [self.sdk_bytes.replace(b"mode: atomic", b"mode: set"),
                ("mode: atomic\n" + COVER.MODULE + "cmd/cq/main.go:1.1,1.9 4 1\n").encode(),
                ("mode: atomic\n" + COVER.SDK + "client_test.go:1.1,1.9 1 1\n").encode(),
                self.sdk_bytes + (COVER.SDK + "client.go:1.1,1.9 3 1\n").encode(),
                self.sdk_bytes.replace(b"2 7", b"2 18446744073709551616")]
        for row in rows:
            with self.subTest(row=row):
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.profile(row, self.tracked, sdk_only=True)

    def test_duplicate_json_nonfinite_unknown_fields_and_raw_diagnostics_refuse(self) -> None:
        for content in (b'{"version":1,"version":1}', b'{"value":NaN}'):
            with self.assertRaises(COVER.InvalidEvidence):
                COVER.strict_json(content)
        changed = copy.deepcopy(self.baseline)
        changed["checks"][0]["diagnostic"] = "/private/must-not-retain"
        with self.assertRaises(COVER.InvalidEvidence):
            COVER.validate_baseline(changed, self.pins)
        changed = dict(self.sdk, raw="must-not-retain")
        with self.assertRaises(COVER.InvalidEvidence):
            COVER.validate_sdk(changed, self.pins, self.sdk_bytes)

    def test_private_file_symlink_hardlink_permission_and_size_fences(self) -> None:
        path = self.paths["sdk_coverage"]
        target = self.root / "sdk" / "target"
        self.write(target, self.sdk_bytes)
        path.unlink()
        path.symlink_to(target)
        with self.assertRaises(OSError):
            self.join()
        path.unlink()
        os.link(target, path)
        with self.assertRaises(COVER.InvalidEvidence):
            self.join()
        path.unlink()
        self.write(path, self.sdk_bytes)
        path.chmod(0o644)
        with self.assertRaises(COVER.InvalidEvidence):
            self.join()
        path.chmod(0o600)
        with self.assertRaises(COVER.InvalidEvidence):
            COVER.private_read(path, 4)
        self.assertFalse(self.output.exists())

    def test_dirty_wrong_or_ignored_sdk_source_is_not_a_verified_checkout(self) -> None:
        with self.assertRaises(COVER.InvalidEvidence):
            COVER.source_identity(self.source, "a" * 40)
        (self.source / ".git/info/exclude").write_text("sdk/go/ignored.go\n")
        ignored = self.source / "sdk/go/ignored.go"
        ignored.write_text("package ignored\n")
        with self.assertRaises(COVER.InvalidEvidence):
            COVER.source_identity(self.source, self.commit)
        ignored.unlink()
        (self.source / "sdk/go/client.go").write_text("package changed\n")
        with self.assertRaises(COVER.InvalidEvidence):
            self.join()
        self.assertFalse(self.output.exists())

    def test_output_is_new_private_and_never_inside_inputs_or_source(self) -> None:
        self.output.mkdir(mode=0o700)
        sentinel = self.output / "keep"
        sentinel.write_text("unchanged")
        with self.assertRaises(FileExistsError):
            self.join()
        self.assertEqual("unchanged", sentinel.read_text())
        for output in (self.source / "output", self.pins_path.parent / "output"):
            with self.assertRaises(COVER.InvalidEvidence):
                COVER.join(self.source, self.paths, output, self.pins_path, self.pins_sha)

    def test_retainer_keeps_failed_conformance_failed_without_private_material(self) -> None:
        preflight = self.root / "preflight" / "preflight.json"
        with mock.patch.object(COVER, "go_identity", return_value=self.tool):
            before = COVER.measurement(self.source, self.commit, self.binary, self.binary, self.origin)
            self.write(preflight, COVER.encoded(before))
            output = self.root / "retained"
            result = COVER.retain(self.source, self.paths["sdk_coverage"], output, self.commit,
                                  self.binary, self.binary, self.origin, 1, preflight)
        self.assertEqual(1, result["conformance_exit_code"])
        metadata = json.loads((output / "sdk-metadata.json").read_bytes())
        self.assertTrue(metadata["source_before_after_equal"])
        with self.assertRaises(COVER.InvalidEvidence):
            COVER.validate_sdk(metadata, self.pins, self.sdk_bytes)
        self.assertNotIn(str(self.root), json.dumps(metadata))
        self.assertEqual({"coverage.out", "sdk-metadata.json"}, {p.name for p in output.iterdir()})

    def test_retainer_rejects_preflight_change_before_any_output(self) -> None:
        preflight = self.root / "preflight" / "preflight.json"
        with mock.patch.object(COVER, "go_identity", return_value=self.tool):
            before = COVER.measurement(self.source, self.commit, self.binary, self.binary, self.origin)
            before["source"]["tree"] = "a" * 40
            self.write(preflight, COVER.encoded(before))
            with self.assertRaises(COVER.InvalidEvidence):
                COVER.retain(self.source, self.paths["sdk_coverage"], self.output, self.commit,
                             self.binary, self.binary, self.origin, 0, preflight)
        self.assertFalse(self.output.exists())

    def test_go_version_probe_refuses_wrong_version_or_stderr(self) -> None:
        for stdout, stderr in ((b"go version go1.26.0 linux/amd64\n", b""),
                               (f"go version {COVER.GO_VERSION} {COVER.PLATFORM}\n".encode(), b"private")):
            with mock.patch.object(COVER.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=stdout, stderr=stderr)):
                with self.assertRaises(COVER.InvalidEvidence):
                    COVER.go_identity(self.binary)

    def test_incomplete_coverage_never_emits_a_green_global_gate(self) -> None:
        self.sdk_bytes = self.sdk_bytes.replace(b"2 7", b"2 0")
        self.write_inputs()
        result = self.join()
        self.assertEqual(9, result["coverage"]["total_statements"])
        self.assertEqual(7, result["coverage"]["covered_statements"])
        self.assertFalse(result["coverage"]["coverage_gate_passed"])
        self.assertFalse(result["global_gate_passed"])

    def test_cli_failure_is_constant_and_does_not_disclose_paths_or_metadata(self) -> None:
        arguments = argparse.Namespace(action="join", source=self.source, output=self.output,
                                      pins=self.pins_path, expected_pins_sha256="e" * 64, **self.paths)
        output = io.StringIO()
        with mock.patch.object(COVER, "arguments", return_value=arguments), mock.patch("sys.stdout", output):
            self.assertEqual(2, COVER.main())
        self.assertEqual("Bounded Go-MCP evidence refused.\n", output.getvalue())
        self.assertFalse(self.output.exists())

    def fake_go_environment(self, label: str) -> tuple[dict, Path, Path]:
        directory = self.root / label
        directory.mkdir(mode=0o700)
        executable = directory / "go"
        executable.write_text('#!/bin/sh\nprintf "%s\\n" invocation "$CQ_GO_MCP_CONFORMANCE" "$@" >> "$ARGV_CAPTURE"\n')
        executable.chmod(0o700)
        capture = directory / "argv"
        environment = {"PATH": str(directory) + os.pathsep + os.defpath, "ARGV_CAPTURE": str(capture),
                       "CQ_EXECUTABLE": "/synthetic/cq", "CQ_CONFIG": "/synthetic/policy.json",
                       "CQ_STATE_DIRECTORY": "/synthetic/state"}
        return environment, executable, capture

    def test_go_mcp_legacy_and_instrumented_routes_each_invoke_one_exact_fixture(self) -> None:
        script = Path(__file__).resolve().parents[3] / "sdk/conformance/go-mcp.sh"
        original = ["test", "-tags=integration", "./sdk/go", "-run", "^" + COVER.TEST + "$", "-count=1"]
        for instrumented in (False, True):
            with self.subTest(instrumented=instrumented):
                environment, executable, capture = self.fake_go_environment("route-" + str(instrumented))
                expected = ["invocation", "1", *original]
                if instrumented:
                    profile = str(capture.parent / "coverage.out")
                    environment.update(CQ_GO_BINARY=str(executable), CQ_GO_MCP_COVERPROFILE=profile)
                    expected += ["-covermode=atomic", "-coverpkg=" + COVER.PACKAGE, "-coverprofile=" + profile]
                result = subprocess.run(["/bin/bash", str(script)], env=environment,
                                        capture_output=True, check=False, timeout=5)
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertEqual(expected, capture.read_text().splitlines())

    def test_go_mcp_partial_empty_and_relative_instrumentation_refuse_before_fixture(self) -> None:
        script = Path(__file__).resolve().parents[3] / "sdk/conformance/go-mcp.sh"
        for case in ("binary-only", "profile-only", "both-empty", "relative-binary", "relative-profile"):
            with self.subTest(case=case):
                environment, executable, capture = self.fake_go_environment(case)
                values = {"CQ_GO_BINARY": str(executable), "CQ_GO_MCP_COVERPROFILE": str(capture.parent / "coverage.out")}
                if case == "binary-only":
                    del values["CQ_GO_MCP_COVERPROFILE"]
                elif case == "profile-only":
                    del values["CQ_GO_BINARY"]
                elif case == "both-empty":
                    values = {name: "" for name in values}
                elif case == "relative-binary":
                    values["CQ_GO_BINARY"] = "relative-go"
                else:
                    values["CQ_GO_MCP_COVERPROFILE"] = "relative-coverage"
                environment.update(values)
                result = subprocess.run(["/bin/bash", str(script)], env=environment,
                                        capture_output=True, check=False, timeout=5)
                self.assertNotEqual(0, result.returncode)
                self.assertFalse(capture.exists())


if __name__ == "__main__":
    unittest.main()
