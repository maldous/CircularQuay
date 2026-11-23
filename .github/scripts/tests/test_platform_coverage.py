"""Fabricated counter/provenance oracles; never execute Go or contact a provider."""

from __future__ import annotations

import contextlib
import copy
import hashlib
import importlib.util
import io
import json
import os
import re
import runpy
import socket
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import textwrap
import unittest
from unittest import mock
import zipfile
import warnings


SCRIPT = Path(__file__).resolve().parents[1] / "platform_coverage.py"
SPEC = importlib.util.spec_from_file_location("cq_platform_coverage", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
P = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(P)

COMMIT = "a" * 40
TREE = "b" * 40
HEAD = "c" * 40
SOURCE = {"commit": COMMIT, "tree": TREE, "tracked_clean": True,
          "locks": {"go.mod": "1" * 64, "go.sum": "2" * 64}}
TRACKED = {"shared.go", "linux.go", "darwin.go", "windows.go"}


def profile(*rows, mode="atomic"):
    return ("mode: " + mode + "\n" + "\n".join(P.MODULE + row for row in rows) + "\n").encode()


def producer(role):
    return {"repository": P.REPOSITORY, "repository_id": P.REPOSITORY_ID, "run_id": "12345",
            "attempt": "1", "job": P.ROLES[role], "event": "pull_request", "pr_head": HEAD,
            "step": "platform_counters"}


def tool(role):
    platform = {"linux": "linux/amd64", "source": "linux/amd64", "macos": "darwin/arm64", "windows": "windows/amd64"}[role]
    return {"version": P.GO_VERSION, "platform": platform,
            "sha256": hashlib.sha256(role.encode()).hexdigest(), "bytes": 12345}


def native_metadata(role, content):
    return {"version": 1, "kind": "platform-counters", "role": role, "source": copy.deepcopy(SOURCE),
            "toolchain": tool(role), "producer": producer(role),
            "build": {"go_flags": "", "cgo_selector": "default", "scope": "default-all-packages-no-tags"},
            "source_before_after_equal": True,
            "coverage": {"mode": "atomic", "bytes": len(content), "sha256": P.digest(content)},
            "test": {"package_pattern": "./...", "coverpkg": "./...", "count": 1,
                     "package_parallelism": 1, "tags": [], "exit_code": 0}, "production_qualified": False}


def source_metadata(content):
    required = ["format", "unit_tests", "go_vet", "go_toolchain", "source_revision",
                "source_identity_before_checks", "source_identity_after_checks"]
    checks = [{"name": name, "status": "passed", "required": True} for name in required]
    checks += [{"name": "statement_coverage", "status": "failed", "required": True, "code": "coverage_below_100_percent"},
               {"name": "tagged_integration", "status": "not_run", "required": False, "code": "requires_qualified_tagged_integration_lane"}]
    return {"version": 1, "profile": "source", "commit": COMMIT, "passed": False,
            "production_qualified": False, "checks": checks,
            "source": {"commit": COMMIT, "tree": TREE, "manifest_sha256": "3" * 64},
            "original_report_sha256": "4" * 64, "coverage_sha256": P.digest(content), "retention_only": True,
            "coverage_semantics": "Original audit result; not recomputed or promoted by retention",
            "go_version": P.GO_VERSION, "platform": "linux/amd64", "go_tool_sha256": tool("source")["sha256"]}


def zip_bytes(files):
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", compression=zipfile.ZIP_STORED) as archive:
        for name, content in files.items():
            entry = zipfile.ZipInfo(name)
            entry.create_system = 3
            entry.external_attr = (stat.S_IFREG | 0o600) << 16
            archive.writestr(entry, content)
    return stream.getvalue()


def fixture(role, content):
    metadata = source_metadata(content) if role == "source" else native_metadata(role, content)
    encoded = P.encoded(metadata)
    record = {"version": 1, "role": role, "source": copy.deepcopy(SOURCE), "toolchain": tool(role),
              "producer": producer(role), "coverage": {"mode": "atomic", "bytes": len(content), "sha256": P.digest(content)}}
    descriptor = P.descriptor(record, encoded)
    name = "metadata.json" if role == "source" else "platform-metadata.json"
    archive = zip_bytes({"coverage.out": content, name: encoded})
    entry = {"descriptor": descriptor, "artifact_id": str(100 + list(P.ROLES).index(role)), "archive_sha256": P.digest(archive)}
    return archive, metadata, entry


def four_inputs(full=False):
    hit = int(full)
    contents = {"source": profile("shared.go:1.1,2.2 2 0", "linux.go:3.1,4.2 3 0"),
                "linux": profile("shared.go:1.1,2.2 2 1", f"linux.go:3.1,4.2 3 {hit}"),
                "macos": profile("shared.go:1.1,2.2 2 0", f"darwin.go:5.1,6.2 5 {hit}"),
                "windows": profile("shared.go:1.1,2.2 2 4", f"windows.go:7.1,8.2 7 {hit}")}
    values = {role: fixture(role, contents[role]) for role in P.ROLES}
    pins = {"version": 1, "kind": "same-run-platform-coverage-pins", "source": copy.deepcopy(SOURCE),
            "entries": {role: value[2] for role, value in values.items()},
            "original_job_results": {"tests": "success", "native-macos": "success", "native-windows": "success", "assurance": "failure"}}
    return values, pins


class PrimitiveContracts(unittest.TestCase):
    def test_strict_objects_reject_duplicates_nonfinite_and_nonobjects(self):
        for data in (b'{"a":1,"a":2}', b'{"a":NaN}', b'{"a":Infinity}', b'{"a":1e999}',
                     b'{"a":123456789012345678901}', b'[]', b'null', b'1'):
            with self.subTest(data=data), self.assertRaises(ValueError):
                P.strict_json(data)
        self.assertEqual(P.strict_json(b'{"a":1}'), {"a": 1})
        self.assertEqual(P.strict_json(b'{"a":1.5}'), {"a": 1.5})

    def test_key_hex_and_predicate_boundaries(self):
        for value in (None, [], {"a": 1, "b": 2}):
            with self.subTest(value=value), self.assertRaises(ValueError):
                P.keys(value, {"a"})
        for value in (None, True, "A" * 64, "g" * 64, "a" * 63):
            with self.subTest(value=value), self.assertRaises(ValueError):
                P.hex_value(value)
        P.hex_value("a" * 64)

    def test_fixed_producer_context_is_scoped_and_consumer_is_not_a_producer(self):
        environment = {"GITHUB_REPOSITORY": P.REPOSITORY, "GITHUB_REPOSITORY_ID": P.REPOSITORY_ID,
                       "GITHUB_RUN_ID": "12345", "GITHUB_RUN_ATTEMPT": "1", "GITHUB_JOB": "tests",
                       "GITHUB_EVENT_NAME": "pull_request", "CQ_COVERAGE_PR_HEAD": HEAD,
                       "UNRELATED_SECRET": "SYNTHETIC_PRIVATE"}
        with mock.patch.dict(os.environ, environment, clear=True):
            self.assertEqual(P.origin("linux"), producer("linux"))
            with self.assertRaises(ValueError):
                P.consumer_origin()
            os.environ["GITHUB_JOB"] = "platform-coverage"
            self.assertEqual(P.consumer_origin()["job"], "platform-coverage")
            with self.assertRaises(ValueError):
                P.origin("source")
        self.assertNotIn("SYNTHETIC_PRIVATE", json.dumps(producer("linux")))

    def test_producer_identity_refuses_wrong_repo_role_run_or_event(self):
        for field, value in (("repository", "other/repo"), ("repository_id", "123"), ("run_id", "0"),
                             ("attempt", True), ("job", "native-windows"), ("event", "workflow_run"),
                             ("step", "other"), ("pr_head", "branch")):
            with self.subTest(field=field), self.assertRaises(ValueError):
                P.validate_origin(dict(producer("linux"), **{field: value}), "linux")

    def test_build_options_refuse_hidden_tags_overlay_and_crosscompile(self):
        with mock.patch.dict(os.environ, {"GOENV": "off", "GOWORK": "off", "GOTOOLCHAIN": "local"}, clear=True):
            self.assertEqual(P.build_options()["scope"], "default-all-packages-no-tags")
            for name, value in (("GOFLAGS", "-tags=integration"), ("GOENV", "user"), ("GOWORK", "workspace"),
                                ("GOOS", "windows"), ("GOARCH", "arm64"), ("CGO_ENABLED", "bad"), ("GOTOOLCHAIN", "auto")):
                with self.subTest(name=name), mock.patch.dict(os.environ, {name: value}), self.assertRaises(ValueError):
                    P.build_options()

    def test_tracked_index_rejects_aliases_nonregular_or_unmerged_records(self):
        blob = b"a" * 40
        self.assertEqual(P.tracked_go(b"100644 " + blob + b" 0\tshared.go\0"), {"shared.go"})
        for record in (b"120000 " + blob + b" 0\tshared.go\0", b"100644 " + blob + b" 1\tshared.go\0",
                       b"100644 " + blob + b" 0\t../shared.go\0", b"100644 " + blob + b" 0\t/shared.go\0", b""):
            with self.subTest(record=record), self.assertRaises(ValueError):
                P.tracked_go(record)


class ProfileContracts(unittest.TestCase):
    def test_duplicate_max_union_keeps_zero_coordinates_and_all_numstmt(self):
        data = profile("shared.go:1.1,2.2 2 0", "shared.go:1.1,2.2 2 4", "linux.go:3.1,4.2 3 0")
        blocks = P.profile(data, TRACKED)
        self.assertEqual(len(blocks), 2)
        self.assertEqual(blocks[(P.MODULE + "shared.go", 1, 1, 2, 2)], (2, 4))
        _, value, gaps = P.merged_profiles([blocks])
        self.assertEqual((value["covered_statements"], value["total_statements"]), (2, 5))
        self.assertEqual(gaps[0]["statements"], 3)

    def test_true_sixteen_mib_bound_rejects_hidden_tail(self):
        content = profile("shared.go:1.1,2.2 1 1")
        with self.assertRaises(ValueError):
            P.profile(content + b" " * (P.MAX_PROFILE + 1 - len(content)), TRACKED)

    def test_profile_requires_atomic_nonempty_owned_real_coordinates(self):
        rows = ["shared.go:0.1,2.2 1 1", "shared.go:2.1,1.2 1 1", "shared.go:1.1,2.2 10000001 1",
                "shared.go:1.1,2.2 1 18446744073709551616", "unknown.go:1.1,2.2 1 1",
                "../shared.go:1.1,2.2 1 1", "shared_test.go:1.1,2.2 1 1"]
        for row in rows:
            with self.subTest(row=row), self.assertRaises(ValueError):
                P.profile(profile(row), TRACKED)
        for data in (b"mode: atomic\n", profile("shared.go:1.1,2.2 0 0"), profile("shared.go:1.1,2.2 1 1", mode="count")):
            with self.subTest(data=data), self.assertRaises(ValueError):
                P.profile(data, TRACKED)

    def test_numstmt_mismatch_refuses_within_and_across_producers(self):
        with self.assertRaises(ValueError):
            P.profile(profile("shared.go:1.1,2.2 1 0", "shared.go:1.1,2.2 2 1"), TRACKED)
        first = P.profile(profile("shared.go:1.1,2.2 1 0"), TRACKED)
        second = P.profile(profile("shared.go:1.1,2.2 2 1"), TRACKED)
        with self.assertRaises(ValueError):
            P.merged_profiles([first, second])

    def test_line_endings_match_go_scanlines_not_unicode_separators(self):
        content = profile("shared.go:1.1,2.2 1 1")
        expected = P.profile(content, TRACKED)
        self.assertEqual(P.profile(content.replace(b"\n", b"\r\n"), TRACKED), expected)
        self.assertEqual(P.profile(content[:-1], TRACKED), expected)
        with self.assertRaises(ValueError):
            P.profile(content.replace(b"\n", b"\v"), TRACKED)

    def test_os_only_zero_blocks_expand_denominator_instead_of_disappearing(self):
        values, _ = four_inputs()
        blocks = [P.profile(P.archive_members(data, role, entry)[0], TRACKED) for role, (data, _, entry) in values.items()]
        content, coverage, gaps = P.merged_profiles(blocks)
        self.assertEqual((coverage["covered_statements"], coverage["total_statements"]), (2, 17))
        self.assertFalse(coverage["coverage_gate_passed"])
        self.assertEqual(coverage["threshold_percent"], 100)
        self.assertEqual({row["path"] for row in gaps}, {"linux.go", "windows.go", "darwin.go"})
        self.assertEqual(len(P.profile(content, TRACKED)), 4)


class ProvenanceContracts(unittest.TestCase):
    def test_descriptor_allows_actual_arm64_darwin_not_assumed_amd64(self):
        _, _, value = fixture("macos", profile("darwin.go:1.1,2.2 1 1"))
        P.validate_descriptor(value["descriptor"], "macos", SOURCE)
        self.assertEqual(value["descriptor"]["toolchain"]["platform"], "darwin/arm64")

    def test_descriptor_refuses_wrong_source_version_platform_mode_and_hash(self):
        _, _, value = fixture("linux", profile("linux.go:1.1,2.2 1 1"))
        base = value["descriptor"]
        changes = [("source", "tree", "d" * 40), ("toolchain", "version", "go1.0.0"),
                   ("toolchain", "platform", "windows/amd64"), ("coverage", "mode", "set"),
                   ("coverage", "bytes", True), ("coverage", "sha256", "not-a-hash")]
        for group, key, item in changes:
            bad = copy.deepcopy(base)
            bad[group][key] = item
            with self.subTest(group=group, key=key), self.assertRaises(ValueError):
                P.validate_descriptor(bad, "linux", SOURCE)

    def test_platform_refuses_failed_skipped_tagged_or_mutated_test_metadata(self):
        content = profile("linux.go:1.1,2.2 1 1")
        _, metadata, entry = fixture("linux", content)
        P.validate_platform(metadata, entry["descriptor"], content)
        cases = [("exit_code", 1), ("exit_code", True), ("tags", ["integration"]),
                 ("package_pattern", "./internal/source"), ("package_parallelism", True)]
        for field, value in cases:
            bad = copy.deepcopy(metadata)
            bad["test"][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                P.validate_platform(bad, entry["descriptor"], content)
        for field in ("source_before_after_equal", "production_qualified"):
            bad = copy.deepcopy(metadata)
            bad[field] = not bad[field]
            with self.subTest(field=field), self.assertRaises(ValueError):
                P.validate_platform(bad, entry["descriptor"], content)

    def test_baseline_preserves_false_and_every_failed_or_not_run_check(self):
        content = profile("shared.go:1.1,2.2 1 0")
        value = source_metadata(content)
        self.assertEqual(P.validate_baseline(value, SOURCE, content), value["checks"])
        self.assertFalse(value["passed"])
        self.assertEqual(value["checks"][-1]["status"], "not_run")
        bad = copy.deepcopy(value)
        bad["passed"] = True
        with self.assertRaises(ValueError):
            P.validate_baseline(bad, SOURCE, content)

    def test_baseline_required_source_checks_cannot_be_missing_or_failed(self):
        content = profile("shared.go:1.1,2.2 1 0")
        for index in range(7):
            value = source_metadata(content)
            value["checks"][index]["status"] = "failed"
            with self.subTest(index=index), self.assertRaises(ValueError):
                P.validate_baseline(value, SOURCE, content)
        value = source_metadata(content)
        value["checks"].append(value["checks"][0])
        with self.assertRaises(ValueError):
            P.validate_baseline(value, SOURCE, content)

    def test_pins_refuse_missing_platform_mixed_run_attempt_head_or_locks(self):
        _, base = four_inputs()
        P.validate_pins(base, SOURCE)
        for field, value in (("run_id", "999"), ("attempt", "2"), ("pr_head", "d" * 40)):
            bad = copy.deepcopy(base)
            bad["entries"]["windows"]["descriptor"]["producer"][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                P.validate_pins(bad, SOURCE)
        bad = copy.deepcopy(base)
        bad["entries"]["macos"]["descriptor"]["source"]["locks"]["go.sum"] = "9" * 64
        with self.assertRaises(ValueError):
            P.validate_pins(bad, SOURCE)
        missing = copy.deepcopy(base)
        del missing["entries"]["windows"]
        with self.assertRaises(ValueError):
            P.validate_pins(missing, SOURCE)
        duplicate = copy.deepcopy(base)
        duplicate["entries"]["windows"]["artifact_id"] = duplicate["entries"]["linux"]["artifact_id"]
        with self.assertRaises(ValueError):
            P.validate_pins(duplicate, SOURCE)

    def test_expected_hashes_come_from_needs_not_archive_metadata(self):
        values, expected = four_inputs()
        needs = {job: {"result": expected["original_job_results"][job], "outputs": {
            "descriptor": json.dumps(values[role][2]["descriptor"]), "artifact_id": values[role][2]["artifact_id"],
            "artifact_sha256": values[role][2]["archive_sha256"]}} for role, job in P.ROLES.items()}
        with tempfile.TemporaryDirectory() as name:
            output = Path(name) / "pins.json"
            with mock.patch.object(P, "source_identity", return_value=(SOURCE, TRACKED)), mock.patch.object(P, "consumer_origin", return_value=dict(producer("source"), job="platform-coverage")):
                value = P.pins_from_needs(Path(name), COMMIT, needs, output)
            self.assertEqual(value, expected)
            self.assertEqual(P.strict_json(output.read_bytes()), expected)

    def test_blank_artifact_id_refuses_before_any_download_can_select_all(self):
        values, expected = four_inputs()
        needs = {job: {"result": "success", "outputs": {"descriptor": json.dumps(values[role][2]["descriptor"]),
            "artifact_id": values[role][2]["artifact_id"], "artifact_sha256": values[role][2]["archive_sha256"]}} for role, job in P.ROLES.items()}
        needs["tests"]["outputs"]["artifact_id"] = ""
        with tempfile.TemporaryDirectory() as name, mock.patch.object(P, "source_identity", return_value=(SOURCE, TRACKED)), mock.patch.object(P, "consumer_origin", return_value=dict(producer("source"), job="platform-coverage")):
            with self.assertRaises(ValueError):
                P.pins_from_needs(Path(name), COMMIT, needs, Path(name) / "pins.json")


class ArchiveContracts(unittest.TestCase):
    def test_exact_two_members_are_hash_bound_before_use(self):
        content = profile("linux.go:1.1,2.2 1 1")
        data, metadata, entry = fixture("linux", content)
        self.assertEqual(P.archive_members(data, "linux", entry), (content, metadata))
        wrong = copy.deepcopy(entry)
        wrong["archive_sha256"] = "0" * 64
        with self.assertRaises(ValueError):
            P.archive_members(data, "linux", wrong)

    def test_extra_and_symlink_members_refuse(self):
        content = profile("linux.go:1.1,2.2 1 1")
        _, metadata, entry = fixture("linux", content)
        data = zip_bytes({"coverage.out": content, "platform-metadata.json": P.encoded(metadata), "extra": b"private"})
        entry["archive_sha256"] = P.digest(data)
        with self.assertRaises(ValueError):
            P.archive_members(data, "linux", entry)
        stream = io.BytesIO()
        with zipfile.ZipFile(stream, "w") as archive:
            row = zipfile.ZipInfo("coverage.out")
            row.external_attr = (stat.S_IFLNK | 0o777) << 16
            archive.writestr(row, content)
            archive.writestr("platform-metadata.json", P.encoded(metadata))
        entry["archive_sha256"] = P.digest(stream.getvalue())
        with self.assertRaises(ValueError):
            P.archive_members(stream.getvalue(), "linux", entry)

    def test_duplicate_and_encrypted_headers_refuse_before_open(self):
        content = profile("linux.go:1.1,2.2 1 1")
        data, metadata, entry = fixture("linux", content)
        repeated = io.BytesIO()
        with warnings.catch_warnings(), zipfile.ZipFile(repeated, "w") as archive:
            warnings.simplefilter("ignore", UserWarning)
            archive.writestr("coverage.out", content)
            archive.writestr("coverage.out", content)
            archive.writestr("platform-metadata.json", P.encoded(metadata))
        duplicate = repeated.getvalue()
        entry["archive_sha256"] = P.digest(duplicate)
        with self.assertRaises(ValueError):
            P.archive_members(duplicate, "linux", entry)
        changed = bytearray(data)
        central = changed.index(b"PK\x01\x02")
        changed[central + 8] |= 1
        entry["archive_sha256"] = P.digest(changed)
        with self.assertRaises(ValueError):
            P.archive_members(bytes(changed), "linux", entry)

    def test_parser_implementation_errors_are_constant_and_unchained(self):
        import zlib
        for error in (EOFError("PRIVATE_DATA"), RuntimeError("PRIVATE_DATA"), zlib.error("PRIVATE_DATA")):
            with self.subTest(error=type(error).__name__), mock.patch.object(P, "checked_archive", side_effect=error):
                with self.assertRaisesRegex(P.InvalidEvidence, "^invalid bounded platform coverage archive$") as caught:
                    P.archive_members(b"owned", "linux", {})
                self.assertIsNone(caught.exception.__cause__)
                self.assertTrue(caught.exception.__suppress_context__)

    def test_local_header_filename_cannot_escape_in_refusal_message(self):
        data, _, entry = fixture("linux", profile("linux.go:1.1,2.2 1 1"))
        changed = bytearray(data)
        start = changed.index(b"coverage.out")
        changed[start:start + len(b"coverage.out")] = b"PRIVATE_DATA"
        entry["archive_sha256"] = P.digest(changed)
        with self.assertRaisesRegex(P.InvalidEvidence, "^invalid bounded platform coverage archive$") as error:
            P.archive_members(bytes(changed), "linux", entry)
        self.assertNotIn("PRIVATE_DATA", str(error.exception))
        self.assertIsNone(error.exception.__cause__)
        self.assertTrue(error.exception.__suppress_context__)

    def test_crc_read_error_is_sanitized(self):
        data, _, entry = fixture("linux", profile("linux.go:1.1,2.2 1 1"))
        changed = bytearray(data)
        position = changed.index(b"mode: atomic")
        changed[position] ^= 1
        entry["archive_sha256"] = P.digest(changed)
        with self.assertRaisesRegex(P.InvalidEvidence, "^invalid bounded platform coverage archive$"):
            P.archive_members(bytes(changed), "linux", entry)

    def test_malformed_zip_and_changed_member_hash_refuse(self):
        _, _, entry = fixture("linux", profile("linux.go:1.1,2.2 1 1"))
        entry["archive_sha256"] = P.digest(b"not a zip")
        with self.assertRaises(ValueError):
            P.archive_members(b"not a zip", "linux", entry)
        data, _, entry = fixture("linux", profile("linux.go:1.1,2.2 1 1"))
        entry["descriptor"]["coverage"]["sha256"] = "0" * 64
        with self.assertRaises(ValueError):
            P.archive_members(data, "linux", entry)


class OwnedFileContracts(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.root.chmod(0o700)

    def put(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(content)
        path.chmod(0o600)
        return path

    def test_private_regular_file_exact_bound_and_exclusive_output(self):
        path = self.put("input", b"fabricated")
        self.assertEqual(P.safe_file(path, 10, True), b"fabricated")
        with self.assertRaises(ValueError):
            P.safe_file(path, 9, True)
        output = self.root / "output"
        P.write_new(output, b"owned")
        with self.assertRaises(FileExistsError):
            P.write_new(output, b"replace")

    def test_metadata_comparison_classifies_refusal_without_reporting_values(self):
        original = self.put("private", b"fabricated").stat()
        changed = type("StatView", (), {
            name: getattr(original, name) for name in (
                "st_dev", "st_ino", "st_mode", "st_nlink", "st_size", "st_mtime_ns", "st_ctime_ns"
            )
        })()
        P.require_same_identity(original, changed, "pathname identity")
        changed.st_ctime_ns += 1
        with self.assertRaisesRegex(P.FileIdentityMismatch, r"^pathname identity: ctime$"):
            P.require_same_identity(original, changed, "pathname identity")

    def test_windows_file_binding_allows_only_cross_view_projection(self):
        owned = self.put("windows-model", b"fabricated").stat()
        fields = {
            name: getattr(owned, name) for name in (
                "st_dev", "st_ino", "st_mode", "st_nlink", "st_size", "st_mtime_ns", "st_ctime_ns"
            )
        }

        def info(**changes):
            return type("StatView", (), fields | changes)()

        pathname = info(st_mode=owned.st_mode | 0o111, st_ctime_ns=owned.st_ctime_ns + 1)
        P.require_same_windows_file(info(), pathname)
        with self.assertRaises(P.FileIdentityMismatch):
            P.require_same_identity(info(), pathname, "descriptor stability")
        for change in (
            {"st_dev": owned.st_dev + 1}, {"st_ino": owned.st_ino + 1}, {"st_ino": 0},
            {"st_size": owned.st_size + 1}, {"st_mtime_ns": owned.st_mtime_ns + 1},
            {"st_nlink": 2}, {"st_file_attributes": 0x400}, {"st_mode": 0o040700},
        ):
            with self.subTest(change=change), self.assertRaises(ValueError):
                P.require_same_windows_file(info(), info(**change))

    @unittest.skipUnless(os.name == "posix", "POSIX protection oracle only; Windows ACLs are not inferred")
    def test_symlink_hardlink_and_public_private_file_refuse(self):
        path = self.put("private", b"fabricated")
        link = self.root / "link"
        link.symlink_to(path)
        with self.assertRaises(ValueError):
            P.safe_file(link, 99)
        hard = self.root / "hard"
        os.link(path, hard)
        with self.assertRaises(ValueError):
            P.safe_file(path, 99)
        hard.unlink()
        path.chmod(0o644)
        with self.assertRaises(ValueError):
            P.safe_file(path, 99, True)

    def test_archive_transport_accepts_only_one_fixed_owned_file(self):
        folder = self.root / "download"
        folder.mkdir()
        first = self.put("download/opaque.zip", b"fixed")
        self.assertEqual(P.archive_file(folder), b"fixed")
        self.assertEqual(P.archive_file(first), b"fixed")
        self.put("download/extra", b"other")
        with self.assertRaises(ValueError):
            P.archive_file(folder)

    def test_new_output_cannot_be_inside_source_or_existing_directory(self):
        source = self.root / "source"
        source.mkdir()
        with self.assertRaises(ValueError):
            P.new_directory(source / "output", source)
        other = self.root / "other"
        self.assertEqual(P.new_directory(other, source), other)
        with self.assertRaises(FileExistsError):
            P.new_directory(other, source)

    def test_retention_writes_real_measured_bundle_and_preserves_failed_exit(self):
        content = profile("linux.go:1.1,2.2 1 1")
        before = {"version": 1, "kind": "platform-counter-preflight", "role": "linux", "source": SOURCE,
                  "toolchain": tool("linux"), "producer": producer("linux"),
                  "build": {"go_flags": "", "cgo_selector": "default", "scope": "default-all-packages-no-tags"}}
        self.put("before.json", P.encoded(before))
        self.put("coverage.out", content)
        step = self.root / "step-output"
        with mock.patch.object(P, "source_identity", return_value=(SOURCE, TRACKED)), mock.patch.object(P, "tool_identity", return_value=tool("linux")), mock.patch.object(P, "origin", return_value=producer("linux")), mock.patch.object(P, "build_options", return_value=before["build"]):
            result = P.retain(self.root / "source", COMMIT, self.root / "fake-go", "linux", self.root, 7, step)
        metadata = P.strict_json((self.root / "bundle/platform-metadata.json").read_bytes())
        self.assertEqual(metadata["test"]["exit_code"], 7)
        self.assertEqual((self.root / "bundle/coverage.out").read_bytes(), content)
        self.assertEqual(result["metadata_sha256"], P.digest((self.root / "bundle/platform-metadata.json").read_bytes()))
        self.assertTrue(step.read_text().startswith("descriptor="))

    def test_begin_and_post_identity_mismatch_refuse_without_running_go_tests(self):
        destination = self.root / "counter"
        build = {"go_flags": "", "cgo_selector": "default", "scope": "default-all-packages-no-tags"}
        with mock.patch.object(P, "source_identity", return_value=(SOURCE, TRACKED)), mock.patch.object(P, "tool_identity", return_value=tool("linux")), mock.patch.object(P, "origin", return_value=producer("linux")), mock.patch.object(P, "build_options", return_value=build):
            P.begin(self.root / "source", COMMIT, self.root / "fake-go", "linux", destination)
        self.put("counter/coverage.out", profile("linux.go:1.1,2.2 1 1"))
        with mock.patch.object(P, "source_identity", return_value=({**SOURCE, "tree": "d" * 40}, TRACKED)):
            with self.assertRaises(ValueError):
                P.retain(self.root / "source", COMMIT, self.root / "fake-go", "linux", destination, 0, self.root / "step")

    def test_baseline_binding_is_read_only_and_preserves_original_bytes(self):
        content = profile("shared.go:1.1,2.2 1 0")
        metadata = P.encoded(source_metadata(content))
        self.put("coverage.out", content)
        self.put("metadata.json", metadata)
        with mock.patch.object(P, "source_identity", return_value=(SOURCE, TRACKED)), mock.patch.object(P, "tool_identity", return_value=tool("source")), mock.patch.object(P, "origin", return_value=producer("source")):
            value = P.bind_baseline(self.root / "source", COMMIT, self.root / "fake-go", self.root, self.root / "step")
        self.assertEqual(value["metadata_sha256"], P.digest(metadata))
        self.assertEqual((self.root / "metadata.json").read_bytes(), metadata)
        self.assertEqual((self.root / "coverage.out").read_bytes(), content)


class ProcessMetadataContracts(unittest.TestCase):
    def test_git_uses_only_fixed_metadata_argv_and_no_credential_environment(self):
        complete = subprocess.CompletedProcess([], 0, b"safe metadata", b"")
        source = Path("/owned")
        with mock.patch.dict(os.environ, {"PATH": "/trusted/bin", "HOME": "/private", "PRIVATE_TOKEN": "synthetic"}, clear=True), mock.patch.object(P.subprocess, "run", return_value=complete) as run:
            self.assertEqual(P.git(source, "rev-parse", "HEAD"), b"safe metadata")
            arguments, options = run.call_args
            self.assertEqual(arguments[0][-3:], [str(source), "rev-parse", "HEAD"])
            self.assertNotIn("PRIVATE_TOKEN", options["env"])
            self.assertNotIn("HOME", options["env"])
            self.assertEqual(options["env"]["GIT_CONFIG_GLOBAL"], os.devnull)
            self.assertEqual(options["timeout"], 30)

    def test_git_errors_are_not_relayed(self):
        failed = subprocess.CompletedProcess([], 1, b"", b"PRIVATE_DATA")
        with mock.patch.object(P.subprocess, "run", return_value=failed):
            with self.assertRaisesRegex(P.InvalidEvidence, "^invalid bounded platform coverage evidence$"):
                P.git(Path("/owned"), "status", "--porcelain=v1")

    def test_tool_version_probe_hashes_real_owned_bytes_without_executing_go(self):
        with tempfile.TemporaryDirectory() as name:
            binary = Path(name).resolve() / "fake-go"
            binary.write_bytes(b"OWNED_SYNTHETIC_NOT_EXECUTED")
            binary.chmod(0o700)
            complete = subprocess.CompletedProcess([], 0, b"go version go1.27.4 darwin/arm64\n", b"")
            with mock.patch.object(P.subprocess, "run", return_value=complete) as run:
                value = P.tool_identity(binary, "macos")
                run.assert_called_once()
                self.assertEqual(run.call_args.args[0], [str(binary), "version"])
                self.assertEqual(run.call_args.kwargs["env"]["GOTOOLCHAIN"], "local")
            self.assertEqual(value["sha256"], P.digest(binary.read_bytes()))
            self.assertEqual(value["bytes"], binary.stat().st_size)
            self.assertEqual(value["platform"], "darwin/arm64")

    def test_wrong_version_platform_stderr_or_changed_tool_refuses(self):
        with tempfile.TemporaryDirectory() as name:
            binary = Path(name).resolve() / "fake-go"
            binary.write_bytes(b"OWNED_SYNTHETIC")
            binary.chmod(0o700)
            cases = [(b"go version go1.27.3 linux/amd64\n", b""),
                     (b"go version go1.27.4 windows/amd64\n", b""),
                     (b"go version go1.27.4 linux/amd64\n", b"PRIVATE_DATA")]
            for out, error in cases:
                with self.subTest(out=out, error=error), mock.patch.object(P.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, out, error)), self.assertRaises(ValueError):
                    P.tool_identity(binary, "linux")
            def changed(*_args, **_kwargs):
                binary.write_bytes(b"CHANGED_SYNTHETIC")
                return subprocess.CompletedProcess([], 0, b"go version go1.27.4 linux/amd64\n", b"")
            with mock.patch.object(P.subprocess, "run", side_effect=changed), self.assertRaises(ValueError):
                P.tool_identity(binary, "linux")

    def test_source_identity_checks_actual_owned_root_and_finite_fake_git(self):
        with tempfile.TemporaryDirectory() as name:
            source = Path(name).resolve()
            for file in ("go.mod", "go.sum"):
                (source / file).write_bytes(("synthetic-" + file).encode())
                (source / file).chmod(0o600)
            def command(_source, *arguments):
                if arguments == ("rev-parse", "--show-toplevel"):
                    return str(source).encode() + b"\n"
                if arguments == ("rev-parse", "HEAD"):
                    return COMMIT.encode() + b"\n"
                if arguments == ("rev-parse", "HEAD^{tree}"):
                    return TREE.encode() + b"\n"
                if arguments[:2] == ("ls-files", "--stage"):
                    return b"100644 " + b"a" * 40 + b" 0\tshared.go\0"
                return b""
            with mock.patch.object(P, "git", side_effect=command):
                value, files = P.source_identity(source, COMMIT)
            self.assertEqual(value["tree"], TREE)
            self.assertEqual(value["locks"]["go.sum"], P.digest((source / "go.sum").read_bytes()))
            self.assertEqual(files, {"shared.go"})
            with mock.patch.object(P, "git", return_value=b"wrong root"), self.assertRaises(ValueError):
                P.source_identity(source, COMMIT)

    def test_windows_process_environment_adds_only_required_systemroot(self):
        environment = {"PATH": "trusted", "SystemRoot": "C:\\Windows", "PRIVATE_TOKEN": "synthetic"}
        with mock.patch.dict(os.environ, environment, clear=True), mock.patch.object(P.os, "name", "nt"):
            self.assertEqual(P.child_environment(), {"PATH": "trusted", "SystemRoot": "C:\\Windows"})


class CLIContracts(unittest.TestCase):
    def test_actual_entry_help_exits_before_evidence_or_process_work(self):
        original = SCRIPT.read_bytes()
        output = io.StringIO()
        errors = io.StringIO()
        with contextlib.ExitStack() as stack:
            probes = []
            for owner, name in ((subprocess, "run"), (subprocess, "Popen"),
                                (os, "system"), (socket, "create_connection"),
                                (socket, "getaddrinfo"), (socket.socket, "connect"),
                                (socket.socket, "connect_ex")):
                probe = stack.enter_context(mock.patch.object(
                    owner, name, side_effect=AssertionError("unexpected external operation")))
                probes.append(probe)
            stack.enter_context(mock.patch.object(sys, "argv", [str(SCRIPT), "--help"]))
            stack.enter_context(contextlib.redirect_stdout(output))
            stack.enter_context(contextlib.redirect_stderr(errors))
            with self.assertRaises(SystemExit) as stopped:
                runpy.run_path(str(SCRIPT), run_name="__main__")
            self.assertEqual(stopped.exception.code, 0)
            for probe in probes:
                probe.assert_not_called()
        help_text = output.getvalue()
        self.assertIn("usage:", help_text)
        self.assertIn("{begin,retain,baseline,pins,union}", help_text)
        self.assertIn("--source", help_text)
        self.assertIn("--expected-commit", help_text)
        self.assertEqual(errors.getvalue(), "")
        self.assertEqual(SCRIPT.read_bytes(), original)

    def test_exact_actions_delegate_without_running_any_go_or_provider(self):
        routes = {"begin": "begin", "retain": "retain", "baseline": "bind_baseline", "pins": "pins_from_needs", "union": "union"}
        for action, function in routes.items():
            argv = ["platform_coverage.py", action, "--source", "/owned", "--expected-commit", COMMIT]
            with self.subTest(action=action), mock.patch.object(P.sys, "argv", argv), mock.patch.object(P, function, return_value=1 if action == "union" else {}), mock.patch.object(P, "safe_file", return_value=b"{}"), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(P.main(), 1 if action == "union" else 0)

    def test_malformed_input_uses_refusal_two_without_raw_diagnostic(self):
        argv = ["platform_coverage.py", "begin", "--source", "/owned", "--expected-commit", COMMIT]
        stream = io.StringIO()
        with mock.patch.object(P.sys, "argv", argv), mock.patch.object(P, "begin", side_effect=ValueError("PRIVATE_DATA")), contextlib.redirect_stdout(stream):
            self.assertEqual(P.main(), 2)
        self.assertEqual(stream.getvalue(), "Bounded same-run platform evidence refused.\n")
        self.assertNotIn("PRIVATE_DATA", stream.getvalue())


class CompleteUnionContracts(unittest.TestCase):
    def run_union(self, full=False, mutate=None):
        values, pins = four_inputs(full)
        if mutate:
            mutate(values, pins)
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name).resolve()
        root.chmod(0o700)
        artifacts = root / "artifacts"
        artifacts.mkdir()
        for role, (archive, _, _) in values.items():
            directory = artifacts / role
            directory.mkdir()
            path = directory / "opaque.zip"
            path.write_bytes(archive)
            path.chmod(0o600)
        pinfile = root / "pins.json"
        pinfile.write_bytes(P.encoded(pins))
        pinfile.chmod(0o600)
        output = root / "union"
        stream = io.StringIO()
        with mock.patch.object(P, "source_identity", return_value=(SOURCE, TRACKED)), contextlib.redirect_stdout(stream):
            code = P.union(root / "source", COMMIT, artifacts, pinfile, P.digest(pinfile.read_bytes()), output)
        return code, P.strict_json((output / "platform-union.json").read_bytes()), P.profile((output / "coverage.out").read_bytes(), TRACKED)

    def test_actual_shortfall_returns_one_and_preserves_every_original_result(self):
        code, result, blocks = self.run_union()
        self.assertEqual(code, 1)
        self.assertEqual((result["coverage"]["covered_statements"], result["coverage"]["total_statements"]), (2, 17))
        self.assertEqual(len(blocks), 4)
        self.assertFalse(result["original_source_report_unchanged"]["passed"])
        self.assertEqual(result["original_source_metadata_unchanged"], source_metadata(profile("shared.go:1.1,2.2 2 0", "linux.go:3.1,4.2 3 0")))
        self.assertEqual(result["original_source_report_unchanged"]["checks"][-1]["status"], "not_run")
        self.assertEqual(result["original_job_results_unchanged"]["assurance"], "failure")
        self.assertFalse(result["source_report_promoted"])

    def test_complete_union_is_not_old_source_or_global_production_pass(self):
        code, result, _ = self.run_union(True)
        self.assertEqual(code, 0)
        self.assertEqual(result["coverage"]["total_statements"], 17)
        self.assertTrue(result["coverage"]["coverage_gate_passed"])
        self.assertFalse(result["original_source_report_unchanged"]["passed"])
        self.assertFalse(result["global_gate_passed"])
        self.assertFalse(result["production_qualified"])

    def test_other_original_linux_quality_failure_remains_explicit(self):
        def change(_values, pins):
            pins["original_job_results"]["tests"] = "failure"
        code, result, _ = self.run_union(True, change)
        self.assertEqual(code, 0)
        self.assertEqual(result["original_job_results_unchanged"]["tests"], "failure")
        self.assertFalse(result["global_gate_passed"])

    def test_failed_unit_producer_refuses_instead_of_becoming_partial_success(self):
        def change(values, pins):
            content = profile("windows.go:7.1,8.2 7 1")
            metadata = native_metadata("windows", content)
            metadata["test"]["exit_code"] = 1
            raw = P.encoded(metadata)
            archive = zip_bytes({"coverage.out": content, "platform-metadata.json": raw})
            old = pins["entries"]["windows"]
            old["descriptor"]["coverage"] = metadata["coverage"]
            old["descriptor"]["metadata_sha256"] = P.digest(raw)
            old["descriptor"]["metadata_bytes"] = len(raw)
            old["archive_sha256"] = P.digest(archive)
            values["windows"] = (archive, metadata, old)
        with self.assertRaises(ValueError):
            self.run_union(True, change)

    def test_corrupt_archive_and_source_mismatch_never_write_aggregate(self):
        def change(values, _pins):
            archive, metadata, entry = values["macos"]
            values["macos"] = (archive + b"changed", metadata, entry)
        with self.assertRaises(ValueError):
            self.run_union(True, change)


class WorkflowContracts(unittest.TestCase):
    def source(self):
        return SCRIPT.parents[1].joinpath("workflows/source.yml").read_text()

    def job(self, name):
        value = self.source().split("\n  " + name + ":\n", 1)[1]
        return re.split(r"\n  (?=\S)", value, maxsplit=1)[0]

    def test_all_three_existing_native_calls_keep_full_default_selection(self):
        value = self.source()
        self.assertEqual(value.count("test -count=1 -p=1 -covermode=atomic"), 3)
        for role, job in (("linux", "tests"), ("macos", "native-macos"), ("windows", "native-windows")):
            body = self.job(job)
            self.assertIn("--platform " + role, body)
            self.assertIn("-coverpkg=./...", body)
            self.assertIn("-coverprofile=", body)
            self.assertIn("--test-exit", body)
            self.assertIn("steps.native_counters.outputs.descriptor", body)
            self.assertIn("retention-days: 7", body)
        self.assertIn('if test "$cq_test_exit" -ne 0; then exit "$cq_test_exit"; fi', value)
        self.assertIn('if ($cqTestExit -ne 0) { exit $cqTestExit }', value)

    def test_windows_selects_one_native_path_identity_for_probe_test_and_retention(self):
        body = self.job("native-windows")
        selection = "$cqGo = (Get-Command go -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source"
        refusal = "if ([string]::IsNullOrWhiteSpace($cqGo)) { throw 'Selected Go executable is unavailable' }"
        self.assertEqual(body.count(selection), 1)
        self.assertLess(body.index(selection), body.index(refusal))
        self.assertLess(body.index(refusal), body.index("platform_coverage.py begin "))
        # The exact scalar selected above is passed to BOTH identity probes and
        # the real test invocation. No older path, version fallback or second
        # command discovery can enter one of the three stages independently.
        self.assertEqual(body.count("Get-Command "), 1)
        self.assertEqual(body.count("--go $cqGo --platform windows"), 2)
        self.assertEqual(body.count("& $cqGo test -count=1 -p=1 -covermode=atomic"), 1)
        self.assertIn("go-version: '1.27.4'", body)
        self.assertIn("$cqTestExit = $LASTEXITCODE", body)
        self.assertIn("$cqRetainExit = $LASTEXITCODE", body)
        self.assertLess(body.index("if ($cqTestExit -ne 0)"), body.index("exit $cqRetainExit"))

    def test_linux_race_vet_fuzz_and_complexity_controls_are_not_replaced(self):
        value = self.source()
        for required in ("go test -race -count=1 -p 1 ./...", "go vet -p 1 ./...", "-fuzz=FuzzCoverage",
                         "-fuzz=FuzzDecodeNeverLeaksInput", "-fuzz=FuzzEndpointNeverAcceptsCredentialURL",
                         '"$GOBIN/gocyclo" -over 10', '"$GOBIN/gocognit" -exact-recursion -over 10'):
            self.assertIn(required, value)
        self.assertNotIn("continue-on-error:", value)

    @unittest.skipUnless(os.name == "posix", "POSIX Bash retention contract; remains mandatory in Linux Source")
    def test_bash_retention_preserves_original_go_failure_and_independent_refusal(self):
        # Select the ACTUAL narrow control span, replacing only its two external
        # commands with shell return functions. No Go/helper command can remain.
        for job in ("tests", "native-macos"):
            body = self.job(job).split("          set +e\n", 1)[1]
            body = body.split("\n      - name:", 1)[0].split("          go test -race", 1)[0]
            lines = textwrap.dedent(body).splitlines()
            self.assertEqual(sum(line.startswith('"$cq_go" test ') for line in lines), 1)
            self.assertEqual(sum(line.startswith('python3 -I -B ') for line in lines), 1)
            mapped = []
            for line in lines:
                if line.startswith('"$cq_go" test '):
                    line = 'synthetic_go'
                elif line.startswith('python3 -I -B '):
                    self.assertIn('platform_coverage.py retain ', line)
                    line = 'synthetic_retain'
                mapped.append(line)
            span = "\n".join(mapped)
            self.assertNotIn('python3', span)
            self.assertNotIn('"$cq_go"', span)
            prefix = 'set -e\nsynthetic_go() { return "$TEST_GO_EXIT"; }\nsynthetic_retain() { return "$TEST_RETAIN_EXIT"; }\nset +e\n'
            for go_exit, retain_exit in ((7, 2), (7, 0), (0, 2), (0, 0)):
                environment = {"TEST_GO_EXIT": str(go_exit), "TEST_RETAIN_EXIT": str(retain_exit)}
                with self.subTest(job=job, go_exit=go_exit, retain_exit=retain_exit):
                    result = subprocess.run(['/bin/bash', '--noprofile', '--norc', '-c',
                                             prefix + span + '\nprintf "LATER_GATE"\n'],
                                            env=environment, capture_output=True, check=False, timeout=5)
                    self.assertEqual(result.returncode, go_exit or retain_exit)
                    self.assertEqual(result.stderr, b'')
                    self.assertEqual(result.stdout, b'LATER_GATE' if job == 'tests' and not (go_exit or retain_exit) else b'')

    def test_source_retainer_and_sdk_contract_tests_stay_mandatory(self):
        value = self.source()
        for name in ("test_retain_audit_evidence.py", "test_go_mcp_coverage.py", "test_platform_coverage.py"):
            self.assertIn("python3 -B -m unittest discover -s .github/scripts/tests -p " + name, value)
        self.assertIn("name: source-audit-${{ github.sha }}", value)
        self.assertIn(".github/scripts/retain_audit_evidence.py", value)
        self.assertIn("platform_coverage.py baseline", value)

    def test_formatter_capture_precedes_suites_and_upload_is_always_guarded(self):
        value = self.source()
        self.assertLess(value.index("id: gofmt_diagnostic"), value.index("id: native_counters"))
        self.assertIn("always() && steps.gofmt_diagnostic.outputs.retained == 'true'", value)
        self.assertIn("name: gofmt-diagnostic-${{ github.sha }}", value)
        self.assertIn("go mod tidy\n          git diff --exit-code -- go.mod go.sum\n          go mod verify", value)
        self.assertNotIn('test -z "$(gofmt -l', value)
        self.assertIn("go version go1.27.4 linux/amd64", value)

    def test_literal_current_run_downloads_are_pinned_and_never_auto_extract(self):
        body = self.job("platform-coverage")
        pin = "actions/download-artifact@9000827ccba6bdab643e8b6fd33ac0654aef8333"
        self.assertEqual(body.count(pin), 4)
        self.assertEqual(body.count("skip-decompress: true"), 4)
        self.assertEqual(body.count("digest-mismatch: error"), 4)
        self.assertEqual(body.count("artifact-ids:"), 4)
        self.assertNotIn("github-token:", body)
        self.assertNotIn("run-id:", body)
        self.assertNotIn("repository:", body)
        self.assertLess(body.index("platform_coverage.py pins"), body.index(pin))
        self.assertIn("expected-pins-sha256", body)
        self.assertIn("original", body)
        self.assertIn("contents: read", self.source())


def formatter_namespace():
    workflow = SCRIPT.parents[1].joinpath("workflows/source.yml").read_text()
    marker = "python3 -I - <<'CQ_FORMATTER'\n"
    body = workflow.split(marker, 1)[1].split("\n          CQ_FORMATTER", 1)[0]
    source = textwrap.dedent(body)
    namespace = {"__name__": "cq_formatter_contract"}
    # These are reviewed repository-owned function definitions. The __main__
    # entry is deliberately not selected; tests never execute a Go formatter.
    exec(compile(source, "<source-workflow-formatter>", "exec"), namespace)
    return namespace


@unittest.skipUnless(os.name == "posix", "POSIX Source formatter pipe/group contract; mandatory on Linux")
class FormatterVisibilityContracts(unittest.TestCase):
    def capture(self, body, limits=None, deadline=2):
        namespace = formatter_namespace()
        streams = {"stdout": io.BytesIO(), "stderr": io.BytesIO()}
        result = namespace["capture"]([sys.executable, "-I", "-c", body], streams,
                                      limits or {"stdout": 1024, "stderr": 1024}, deadline)
        return result, {name: stream.getvalue() for name, stream in streams.items()}

    def test_empty_success_is_the_only_formatter_pass(self):
        result, streams = self.capture("pass")
        self.assertTrue(result["complete"])
        self.assertTrue(result["format_passed"])
        self.assertEqual(result["original_exit_code"], 0)
        self.assertEqual(streams, {"stdout": b"", "stderr": b""})

    def test_full_diff_and_stderr_retained_even_original_exit_is_zero(self):
        result, streams = self.capture("import sys;sys.stdout.write('FULL_DIFF\\n');sys.stderr.write('FULL_ERROR\\n')")
        self.assertTrue(result["complete"])
        self.assertFalse(result["format_passed"])
        self.assertTrue(result["has_diff"])
        self.assertEqual(result["original_exit_code"], 0)
        self.assertEqual(streams, {"stdout": b"FULL_DIFF\n", "stderr": b"FULL_ERROR\n"})
        self.assertEqual(result["stdout"]["sha256"], P.digest(streams["stdout"]))
        self.assertEqual(result["stderr"]["hash_scope"], "whole_stream")

    def test_original_formatter_error_code_is_not_relabelled_clean(self):
        result, streams = self.capture("import sys;sys.stderr.write('SYNTAX_ERROR');sys.exit(2)")
        self.assertTrue(result["complete"])
        self.assertEqual(result["original_exit_code"], 2)
        self.assertFalse(result["format_passed"])
        self.assertEqual(streams["stderr"], b"SYNTAX_ERROR")

    def test_overflow_retains_prefix_but_never_calls_it_complete(self):
        result, streams = self.capture("import sys;sys.stdout.write('x'*4096)", {"stdout": 32, "stderr": 32})
        self.assertTrue(result["overflow"])
        self.assertFalse(result["complete"])
        self.assertFalse(result["format_passed"])
        self.assertIsNone(result["original_exit_code"])
        self.assertEqual(streams["stdout"], b"x" * 32)
        self.assertEqual(result["stdout"]["hash_scope"], "retained_prefix")

    def test_deadline_reaps_owned_synthetic_group_and_refuses_complete(self):
        result, _ = self.capture("import time;time.sleep(5)", deadline=0.05)
        self.assertTrue(result["timed_out"])
        self.assertFalse(result["complete"])
        self.assertFalse(result["format_passed"])
        self.assertIsNone(result["original_exit_code"])

    def test_preflight_failure_still_writes_refused_metadata_and_empty_streams(self):
        namespace = formatter_namespace()
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            environment = {"RUNNER_TEMP": str(root), "GITHUB_OUTPUT": str(root / "step"), "GITHUB_SHA": COMMIT}
            with mock.patch.dict(os.environ, environment), mock.patch.dict(namespace, {"tracked_files": mock.Mock(side_effect=ValueError("PRIVATE_DATA"))}), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(namespace["main"](), 1)
            metadata = json.loads((root / "cq-gofmt/formatter.json").read_bytes())
            self.assertFalse(metadata["complete"])
            self.assertEqual(metadata["status"], "refused")
            self.assertEqual((root / "cq-gofmt/stdout.diff").read_bytes(), b"")
            self.assertEqual((root / "cq-gofmt/stderr.txt").read_bytes(), b"")
            self.assertEqual((root / "step").read_text(), "retained=true\n")
            self.assertNotIn("PRIVATE_DATA", json.dumps(metadata))

    def test_qualified_main_uses_one_fixed_readonly_formatter_call(self):
        namespace = formatter_namespace()
        capture = mock.Mock(return_value={"complete": True, "format_passed": True, "original_exit_code": 0})
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            environment = {"RUNNER_TEMP": str(root), "GITHUB_OUTPUT": str(root / "step"), "GITHUB_SHA": COMMIT}
            functions = {"tracked_files": mock.Mock(return_value=["./owned.go"]),
                         "selected_tools": mock.Mock(return_value=(Path("/pinned/gofmt"), "a" * 64)), "capture": capture}
            with mock.patch.dict(os.environ, environment), mock.patch.dict(namespace, functions), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(namespace["main"](), 0)
            capture.assert_called_once()
            self.assertEqual(capture.call_args.args[0], ["/pinned/gofmt", "-d", "./owned.go"])
            self.assertEqual(capture.call_args.args[2], {"stdout": 8 << 20, "stderr": 1 << 20})
            self.assertEqual(json.loads((root / "cq-gofmt/formatter.json").read_bytes())["status"], "passed")


if __name__ == "__main__":
    unittest.main()
