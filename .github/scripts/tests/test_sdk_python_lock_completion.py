"""Owned-input completion contracts; no resolver, installation or network."""
from __future__ import annotations

import contextlib
import copy
import hashlib
import importlib.util
import io
import json
import os
import runpy
import socket
import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "sdk_python_lock.py"
SPEC = importlib.util.spec_from_file_location("cq_lock_completion", SCRIPT)
assert SPEC and SPEC.loader
LOCK = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(LOCK)
HASH = "a" * 64


def entry():
    return {"is_yanked": False, "metadata": {"name": "example", "version": "1.2"},
            "download_info": {"url": "https://files.pythonhosted.org/example.whl",
                              "archive_info": {"hashes": {"sha256": HASH}}}}


class LockCompletion(unittest.TestCase):
    def setUp(self):
        self.stack = contextlib.ExitStack()
        self.addCleanup(self.stack.close)
        self.root = Path(self.stack.enter_context(tempfile.TemporaryDirectory()))
        for target in ("subprocess.Popen", "subprocess.run", "socket.create_connection", "socket.getaddrinfo"):
            self.stack.enter_context(mock.patch(target, side_effect=AssertionError("unexpected external effect")))

    def write(self, name, data):
        path = self.root / name
        path.write_bytes(data if isinstance(data, bytes) else data.encode())
        return path

    def refuses(self, message, function, *args):
        with self.assertRaises(LOCK.ValidationError) as caught:
            function(*args)
        self.assertEqual(message, str(caught.exception))

    def test_names_versions_and_public_wheel_shapes(self):
        for value in (None, "", "../escape", "a b"):
            self.refuses("invalid package name", LOCK.canonical_name, value)
            self.refuses("invalid package version", LOCK.checked_version, value)
        self.assertEqual("a-b-c", LOCK.canonical_name("A_B.C"))
        for download, message in ((None, "package download information is missing"),
                                  ({"url": None}, "package download URL is invalid"),
                                  ({"url": "https://files.pythonhosted.org:bad/a.whl"}, "package download URL is invalid")):
            self.refuses(message, LOCK.wheel_hash, {"download_info": download})
        for url in ("http://files.pythonhosted.org/a.whl", "https://files.pythonhosted.org:443/a.whl",
                    "https://u:p@files.pythonhosted.org/a.whl", "https://files.pythonhosted.org/a.whl?q=1",
                    "https://files.pythonhosted.org/a.whl#x", "https://files.pythonhosted.org/a.tar.gz"):
            item = entry(); item["download_info"]["url"] = url
            self.refuses("package must be a public PyPI wheel", LOCK.wheel_hash, item)
        for archive in (None, {}, {"hashes": []}, {"hashes": {"sha256": "A" * 64}}):
            item = entry(); item["download_info"]["archive_info"] = archive
            self.refuses("package SHA-256 is invalid", LOCK.wheel_hash, item)

    def test_report_and_requirements_real_files(self):
        report = {"version": "1", "pip_version": "25.0", "install": [entry()]}
        raw = json.dumps(report).encode(); path = self.write("report", raw)
        self.assertEqual((report, hashlib.sha256(raw).hexdigest()), LOCK.read_json_report(path))
        for raw, message in ((b"{", "report is invalid"), (b"\xff", "report is invalid"),
                             (b"[]", "report must be an object"), (b'{"version":"2"}', "report version is unsupported")):
            path.write_bytes(raw); self.refuses(message, LOCK.read_json_report, path)
        self.refuses("report is invalid", LOCK.read_json_report, self.root / "absent")
        path.write_bytes(b"x" * (LOCK.MAX_REPORT_BYTES + 1))
        self.refuses("report exceeds size limit", LOCK.read_json_report, path)
        req = self.write("requirements", "# comment\nExample==1.2 # exact\n\n")
        self.assertEqual({"example": "1.2"}, LOCK.requested_pins(req))
        for raw, message in ((b"\xff", "requirements are invalid"), (b"", "requirements contain no pins"),
                             (b"example>=1", "requirements must use exact pins"),
                             (b"example==1\nExample==1", "requirements contain duplicate packages")):
            req.write_bytes(raw); self.refuses(message, LOCK.requested_pins, req)
        self.refuses("requirements are invalid", LOCK.requested_pins, self.root / "absent")
        req.write_bytes(b"x" * (LOCK.MAX_REQUIREMENTS_BYTES + 1))
        self.refuses("requirements exceed size limit", LOCK.requested_pins, req)

    def test_read_growth_between_real_stat_and_read(self):
        for name, function, limit, message in (
            ("report", LOCK.read_json_report, LOCK.MAX_REPORT_BYTES, "report exceeds size limit"),
            ("requirements", LOCK.requested_pins, LOCK.MAX_REQUIREMENTS_BYTES, "requirements exceed size limit"),
            ("lock", LOCK.verify_lock, LOCK.MAX_REPORT_BYTES, "lock exceeds size limit")):
            path = self.write(name, b"x")
            original = Path.read_bytes
            def grow(owned):
                self.assertEqual(path, owned)
                owned.write_bytes(b"x" * (limit + 1))
                return original(owned)
            with mock.patch.object(Path, "read_bytes", grow):
                args = (path, {}) if name == "lock" else (path,)
                self.refuses(message, function, *args)

    def test_install_list_package_and_pin_refusals(self):
        for value in (None, [], [entry()] * (LOCK.MAX_PACKAGES + 1)):
            self.refuses("report install list is invalid", LOCK.lock_entries, {"install": value}, {})
        for item, message in ((None, "report package is invalid"), ({}, "report contains a yanked package"),
                              ({"is_yanked": False}, "report package metadata is invalid")):
            self.refuses(message, LOCK.lock_entries, {"install": [item]}, {})
        self.refuses("report contains duplicate packages", LOCK.lock_entries, {"install": [entry(), entry()]}, {})
        self.refuses("report does not contain every requested exact pin", LOCK.lock_entries, {"install": [entry()]}, {"other": "1"})
        self.refuses("report does not contain every requested exact pin", LOCK.lock_entries, {"install": [entry()]}, {"example": "2"})
        intake = {"python": "3.14.10", "wheel_uri": "file:///synthetic", "wheel_sha256": HASH}
        environment = {"sys_platform": "darwin", "platform_machine": "x86_64", "implementation_name": "cpython", "python_full_version": "3.14.10"}
        self.refuses("Darwin closure omits qualified cryptography wheel", LOCK.lock_entries, {"environment": environment, "install": [entry()]}, {}, intake)
        for item in ({"metadata": None}, {"metadata": {}, "download_info": None}, {"metadata": {}, "download_info": {"archive_info": None}}):
            self.refuses("local wheel differs from reviewed Darwin intake", LOCK.local_wheel_hash, item, intake)
        self.refuses("local wheel requires its qualified native target report", LOCK.verify_native_target, {"environment": None}, intake)

    def test_verify_lock_complete_owned_matrix(self):
        path = self.write("lock", "example==1.2 --hash=sha256:" + HASH + "\n")
        self.assertIsNone(LOCK.verify_lock(path, {"example": "1.2"}))
        for raw, message in ((b"", "lock is invalid"), (b"\xff", "lock is invalid"),
                             (b"invalid", "lock has an invalid entry"),
                             (("z==1 --hash=sha256:" + HASH + "\na==1 --hash=sha256:" + HASH).encode(), "lock is not uniquely sorted"),
                             (b"\n" * (LOCK.MAX_PACKAGES + 1), "lock is invalid")):
            path.write_bytes(raw); self.refuses(message, LOCK.verify_lock, path, {})
        path.write_text("example==1.2 --hash=sha256:" + HASH)
        self.refuses("lock does not contain every requested exact pin", LOCK.verify_lock, path, {"example": "2"})
        self.refuses("lock is invalid", LOCK.verify_lock, self.root / "absent", {})
        path.write_bytes(b"x" * (LOCK.MAX_REPORT_BYTES + 1))
        self.refuses("lock exceeds size limit", LOCK.verify_lock, path, {})

    def native_fixture(self):
        source = self.root / "sdk/python"; source.mkdir(parents=True)
        policy = b"owned policy"; (source / "darwin-source-policy.json").write_bytes(policy)
        wheel = self.write("qualified.whl", b"owned tiny wheel")
        archive = self.root / "archive.zip"
        with zipfile.ZipFile(archive, "w") as output:
            for case in ("one", "two"):
                output.writestr(f"build/{case}/dist/{wheel.name}", wheel.read_bytes())
        selected = {"python": "3.14.10", "wheel": wheel.name, "custody": "fixture:immutable",
                    "artifact_sha256": hashlib.sha256(archive.read_bytes()).hexdigest(),
                    "wheel_sha256": hashlib.sha256(wheel.read_bytes()).hexdigest()}
        qualified = {"source_policy_sha256": hashlib.sha256(policy).hexdigest(), "run_id": 1,
                     "workflow_head": "1" * 40, "feature": "2" * 40, "qualified_repeat_builds": [selected]}
        manifest = source / "darwin-qualified-wheels.json"
        manifest.write_text(json.dumps(qualified))
        self.stack.enter_context(mock.patch.object(LOCK, "__file__", str(self.root / ".github/scripts/sdk_python_lock.py")))
        return archive, wheel, qualified, manifest

    def test_native_policy_choice_name_and_member_bounds(self):
        archive, wheel, qualified, manifest = self.native_fixture()
        self.refuses("reviewed immutable custody identifier is required", LOCK.darwin_intake, archive, wheel, "3.14.10", "bad value")
        for change, message in ((lambda q: q.update(source_policy_sha256="0" * 64), "Darwin source policy changed"),
                                (lambda q: q.update(qualified_repeat_builds=[]), "Python patch has no qualified wheel"),
                                (lambda q: q["qualified_repeat_builds"][0].update(wheel="other.whl"), "wheel filename differs from qualified input")):
            value = copy.deepcopy(qualified); change(value); manifest.write_text(json.dumps(value))
            self.refuses(message, LOCK.darwin_intake, archive, wheel, "3.14.10", "fixture:immutable")
        manifest.write_text(json.dumps(qualified))
        digest = hashlib.sha256(wheel.read_bytes()).hexdigest()
        with zipfile.ZipFile(archive, "w") as output:
            output.writestr("build/one/dist/qualified.whl", b"short")
            output.writestr("build/two/dist/qualified.whl", wheel.read_bytes())
        self.refuses("archive member size differs from supplied wheel", LOCK.verify_archive_member, archive, wheel, digest)

    def test_atomic_write_success_and_real_parent_failure(self):
        path = self.root / "nested/output"
        LOCK.atomic_write(path, b"owned bytes")
        self.assertEqual(b"owned bytes", path.read_bytes())
        if os.name != "nt": self.assertEqual(0o600, path.stat().st_mode & 0o777)
        parent = self.write("regular-parent", b"unchanged")
        self.refuses("could not write output", LOCK.atomic_write, parent / "child", b"x")
        self.assertEqual(b"unchanged", parent.read_bytes())
        with mock.patch.object(LOCK.os, "replace", side_effect=OSError("synthetic replace refusal")):
            self.refuses("could not write output", LOCK.atomic_write, path, b"replacement")
        self.assertEqual(b"owned bytes", path.read_bytes()); self.assertEqual([], list(path.parent.glob(".cq-*")))
        with mock.patch.object(LOCK.os, "replace", side_effect=OSError), mock.patch.object(LOCK.os, "unlink", side_effect=OSError):
            self.refuses("could not write output", LOCK.atomic_write, path, b"replacement")
        self.assertEqual(b"owned bytes", path.read_bytes())
        # Owned TemporaryDirectory removes the deliberately unremovable staged file after patches unwind.

    def invoke(self, args, expected):
        with mock.patch.object(sys, "argv", [str(SCRIPT), *map(str, args)]), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()) as errors:
            status = LOCK.main()
        self.assertEqual(expected, status)
        return errors.getvalue()

    def test_cli_report_verify_and_native_provenance(self):
        req = self.write("req", "example==1.2")
        report = self.write("report", json.dumps({"version": "1", "pip_version": "25.0", "install": [entry()]}))
        output, proof = self.root / "output", self.root / "proof"
        args = ["--report", report, "--requirements", req, "--output", output, "--receipt", proof, "--revision", "a" * 40]
        self.assertEqual("", self.invoke(args, 0)); self.assertIn("packages=1\n", proof.read_text())
        self.assertNotIn("pythonhosted", proof.read_text())
        self.assertEqual("", self.invoke(["--verify-lock", output, "--requirements", req], 0))
        self.assertIn("report lock arguments are incomplete", self.invoke(["--report", report, "--requirements", req], 2))
        self.assertIn("Darwin intake arguments are incomplete", self.invoke(args + ["--darwin-python", "3.14.10"], 2))
        self.refuses("revision is invalid", LOCK.receipt, HASH, "bad", 1, "25.0")
        archive, wheel, qualified, manifest = self.native_fixture()
        row = entry(); row["metadata"] = {"name": "cryptography", "version": "50.0.2"}
        row["download_info"] = {"url": wheel.resolve().as_uri(), "archive_info": {"hashes": {"sha256": qualified["qualified_repeat_builds"][0]["wheel_sha256"]}}}
        document = {"version": "1", "pip_version": "25.0", "environment": {"sys_platform": "darwin", "platform_machine": "x86_64", "implementation_name": "cpython", "python_full_version": "3.14.10"}, "install": [row]}
        report.write_text(json.dumps(document)); req.write_text("cryptography==50.0.2")
        self.assertEqual("", self.invoke(args + ["--darwin-archive", archive, "--darwin-wheel", wheel, "--darwin-python", "3.14.10", "--darwin-custody", "fixture:immutable"], 0))
        self.assertIn("darwin_intake=", proof.read_text()); self.assertNotIn("file:", proof.read_text())

    def test_owned_script_entry(self):
        req = self.write("req", "example==1.2")
        lock = self.write("lock", "example==1.2 --hash=sha256:" + HASH)
        with mock.patch.object(sys, "argv", [str(SCRIPT), "--verify-lock", str(lock), "--requirements", str(req)]):
            with self.assertRaises(SystemExit) as status:
                runpy.run_path(str(SCRIPT), run_name="__main__")
        self.assertEqual(0, status.exception.code)
