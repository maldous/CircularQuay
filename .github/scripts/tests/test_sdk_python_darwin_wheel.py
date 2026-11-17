"""Owned-byte staging contracts, not a Darwin build or wheel installation.

Transport is a bounded in-memory response. The separately tested native intake
validator is mocked at its explicit seam, so these oracles do not authenticate
Git objects, attest a native build or qualify an installable wheel.
"""

from __future__ import annotations

import contextlib
import hashlib
import importlib.util
import io
import json
import os
import runpy
import stat
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest import mock

SCRIPTS = Path(__file__).parents[1]
LOCK_SPEC = importlib.util.spec_from_file_location("sdk_python_lock", SCRIPTS / "sdk_python_lock.py")
assert LOCK_SPEC is not None and LOCK_SPEC.loader is not None
LOCK = importlib.util.module_from_spec(LOCK_SPEC)
SPEC = importlib.util.spec_from_file_location("darwin_wheel_contract", SCRIPTS / "sdk_python_darwin_wheel.py")
assert SPEC is not None and SPEC.loader is not None
WHEEL = importlib.util.module_from_spec(SPEC)
# Keep the helper's real import/API, without relying on another test module's
# import order or leaving a replacement module registered for other suites.
with mock.patch.dict(sys.modules, {"sdk_python_lock": LOCK}):
    LOCK_SPEC.loader.exec_module(LOCK)
    SPEC.loader.exec_module(WHEEL)

VERSION = "3.14.10"
COMMIT = "1" * 40
NAME = "cryptography-50.0.2-cp314-abi3-macosx_10_15_x86_64.whl"


class OwnedResponse(io.BytesIO):
    """Honor the real read bound and retain only its numeric call record."""

    def __init__(self, value: bytes):
        super().__init__(value)
        self.read_sizes = []

    def read(self, size: int = -1) -> bytes:
        self.read_sizes.append(size)
        return super().read(size)


def archive_bytes(filename: str = NAME, wheel: bytes = b"synthetic opaque wheel; never installed") -> bytes:
    output = io.BytesIO()
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for case in ("one", "two"):
            archive.writestr(f"build/{case}/dist/{filename}", wheel)
    return output.getvalue()


class DarwinWheelStagingContracts(unittest.TestCase):
    def fixture(self, body: bytes | None = None, filename: str = NAME):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        body = archive_bytes(filename) if body is None else body
        digest = hashlib.sha256(body).hexdigest()
        path = f"archives/sha256/{digest}.zip"
        custody = f"git+https://github.com/maldous/CircularQuay.git@{COMMIT}:{path}"
        row = {"python": VERSION, "custody": custody, "artifact_sha256": digest,
               "custody_git_blob": "2" * 40, "wheel": filename}
        manifest = root / "manifest.json"
        manifest.write_text(json.dumps({"qualified_repeat_builds": [row]}))
        response = OwnedResponse(body)
        opener = mock.Mock()
        opener.open.return_value = response
        intake = mock.Mock(return_value={"wheel": filename, "kind": "synthetic",
                                         "wheel_uri": "file:///synthetic/not-retained"})
        stack = self.enterContext(contextlib.ExitStack())
        stack.enter_context(mock.patch.object(WHEEL, "MANIFEST", manifest))
        builder = stack.enter_context(mock.patch.object(WHEEL.urllib.request, "build_opener", return_value=opener))
        stack.enter_context(mock.patch.object(WHEEL, "darwin_intake", intake))
        return root, manifest, row, response, builder, opener, intake

    def rewrite(self, manifest: Path, rows: list[dict]) -> None:
        manifest.write_text(json.dumps({"qualified_repeat_builds": rows}))

    def test_success_binds_exact_public_object_read_and_downstream_tuple(self) -> None:
        root, _, row, response, builder, opener, intake = self.fixture()
        output = root / "stage"
        result = WHEEL.stage(VERSION, output)
        builder.assert_called_once_with(WHEEL.NoRedirect)
        opener.open.assert_called_once_with(
            f"https://raw.githubusercontent.com/maldous/CircularQuay/{COMMIT}/archives/sha256/{row['artifact_sha256']}.zip",
            timeout=60,
        )
        self.assertEqual([WHEEL.MAX_ARCHIVE + 1], response.read_sizes)
        self.assertTrue(response.closed)
        self.assertEqual(row["artifact_sha256"], hashlib.sha256((output / "archive.zip").read_bytes()).hexdigest())
        self.assertEqual(b"synthetic opaque wheel; never installed", (output / NAME).read_bytes())
        intake.assert_called_once_with(output / "archive.zip", output / NAME, VERSION, row["custody"])
        self.assertEqual({"wheel": NAME, "kind": "synthetic"}, result)
        self.assertEqual(result, json.loads((output / "intake.json").read_bytes()))
        self.assertEqual({"archive.zip", NAME, "intake.json"}, {p.name for p in output.iterdir()})
        if os.name == "posix":
            self.assertEqual(0o700, stat.S_IMODE(output.stat().st_mode))

    def test_supported_dual_abi_filename_keeps_exact_member_and_intake_binding(self) -> None:
        name = NAME.replace("-abi3-", "-abi3.abi3t-")
        root, _, row, _, _, _, intake = self.fixture(filename=name)
        output = root / "stage"
        result = WHEEL.stage(VERSION, output)
        self.assertEqual(name, result["wheel"])
        self.assertEqual(b"synthetic opaque wheel; never installed", (output / name).read_bytes())
        intake.assert_called_once_with(output / "archive.zip", output / name, VERSION, row["custody"])

    def test_unique_patch_and_immutable_custody_refuse_before_transport(self) -> None:
        root, manifest, row, _, builder, _, intake = self.fixture()
        cases = [
            [], [row, row], [{**row, "python": "3.13.9"}],
            [{**row, "custody": None}],
            [{**row, "custody": row["custody"].replace(COMMIT, "A" * 40)}],
            [{**row, "custody": row["custody"] + "?private=query"}],
            [{**row, "artifact_sha256": "3" * 64}],
        ]
        for rows in cases:
            with self.subTest(case=len(rows)):
                self.rewrite(manifest, rows)
                with self.assertRaises(WHEEL.ValidationError):
                    WHEEL.stage(VERSION, root / "stage")
                self.assertFalse((root / "stage").exists())
        builder.assert_not_called()
        intake.assert_not_called()

    def test_archive_size_and_digest_refuse_before_output_or_intake(self) -> None:
        for bound in (False, True):
            with self.subTest(bound=bound):
                root, manifest, row, response, _, _, intake = self.fixture()
                if not bound:
                    row["artifact_sha256"] = "4" * 64
                    # Keep the locator/path binding coherent so this reaches
                    # the actual downloaded-byte digest fence, not an earlier guard.
                    row["custody"] = f"git+https://github.com/maldous/CircularQuay.git@{COMMIT}:archives/sha256/{'4' * 64}.zip"
                    self.rewrite(manifest, [row])
                with mock.patch.object(WHEEL, "MAX_ARCHIVE", 8 if bound else WHEEL.MAX_ARCHIVE):
                    with self.assertRaises(WHEEL.ValidationError):
                        WHEEL.stage(VERSION, root / "stage")
                self.assertTrue(response.closed)
                self.assertFalse((root / "stage").exists())
                intake.assert_not_called()

    def test_blob_and_filename_scope_refuse_after_hash_without_output(self) -> None:
        for change in ({"custody_git_blob": "not-a-git-blob"},
                       {"wheel": "cryptography-50.0.2-cp314-abi3-macosx_10_15_arm64.whl"},
                       {"wheel": "../../synthetic-private.whl"}):
            with self.subTest(fields=sorted(change)):
                root, manifest, row, response, _, opener, intake = self.fixture()
                self.rewrite(manifest, [{**row, **change}])
                with self.assertRaises(WHEEL.ValidationError):
                    WHEEL.stage(VERSION, root / "stage")
                opener.open.assert_called_once()
                self.assertTrue(response.closed)
                self.assertFalse((root / "stage").exists())
                intake.assert_not_called()

    def test_real_member_size_limit_preserves_archive_without_wheel_or_receipt(self) -> None:
        body = archive_bytes(wheel=b"a" * 4097)
        self.assertLess(len(body), 4096)
        root, _, _, _, _, _, intake = self.fixture(body)
        output = root / "stage"
        with mock.patch.object(WHEEL, "MAX_ARCHIVE", 4096), self.assertRaises(WHEEL.ValidationError):
            WHEEL.stage(VERSION, output)
        self.assertEqual(body, (output / "archive.zip").read_bytes())
        self.assertFalse((output / NAME).exists())
        self.assertFalse((output / "intake.json").exists())
        intake.assert_not_called()

    def test_missing_or_corrupt_zip_has_no_qualified_receipt(self) -> None:
        empty = io.BytesIO()
        with zipfile.ZipFile(empty, "w") as archive:
            archive.writestr("unrelated-public-member", b"synthetic")
        for body, expected in [(empty.getvalue(), KeyError), (b"not a ZIP archive", zipfile.BadZipFile)]:
            with self.subTest(error=expected.__name__):
                root, _, _, _, _, _, intake = self.fixture(body)
                output = root / "stage"
                with self.assertRaises(expected):
                    WHEEL.stage(VERSION, output)
                self.assertEqual(body, (output / "archive.zip").read_bytes())
                self.assertFalse((output / "intake.json").exists())
                intake.assert_not_called()

    def test_existing_output_and_downstream_refusal_never_claim_rollback(self) -> None:
        root, _, _, _, _, _, intake = self.fixture()
        output = root / "stage"
        output.mkdir()
        (output / "marker").write_bytes(b"owned existing content")
        with self.assertRaises(FileExistsError):
            WHEEL.stage(VERSION, output)
        self.assertEqual(b"owned existing content", (output / "marker").read_bytes())
        intake.assert_not_called()

        root, _, _, _, _, _, intake = self.fixture()
        output = root / "stage"
        intake.side_effect = WHEEL.ValidationError("synthetic intake refusal")
        with self.assertRaises(WHEEL.ValidationError):
            WHEEL.stage(VERSION, output)
        self.assertTrue((output / "archive.zip").is_file())
        self.assertTrue((output / NAME).is_file())
        self.assertFalse((output / "intake.json").exists())
        intake.assert_called_once()

    def test_redirect_handler_and_cli_success_use_only_explicit_inputs(self) -> None:
        self.assertIsNone(WHEEL.NoRedirect().redirect_request(None, None, 302, "Found", {}, "https://outside.invalid"))
        output = io.StringIO()
        with mock.patch.object(sys, "argv", ["stage", "--python", VERSION, "--output", "synthetic-stage"]), \
                mock.patch.object(WHEEL, "stage", return_value={"wheel": NAME}) as stage, \
                contextlib.redirect_stdout(output):
            self.assertEqual(0, WHEEL.main())
        self.assertEqual(NAME + "\n", output.getvalue())
        stage.assert_called_once_with(VERSION, Path("synthetic-stage"))

    def test_cli_refuses_missing_arguments_and_existing_typed_failures(self) -> None:
        for failure in (KeyError("synthetic"), OSError("synthetic"), WHEEL.ValidationError("synthetic"), zipfile.BadZipFile("synthetic")):
            with self.subTest(kind=type(failure).__name__):
                error, output = io.StringIO(), io.StringIO()
                with mock.patch.object(sys, "argv", ["stage", "--python", VERSION, "--output", "synthetic-stage"]), \
                        mock.patch.object(WHEEL, "stage", side_effect=failure), \
                        contextlib.redirect_stderr(error), contextlib.redirect_stdout(output):
                    with self.assertRaises(SystemExit) as caught:
                        WHEEL.main()
                self.assertEqual(1, caught.exception.code)
                self.assertTrue(error.getvalue().startswith("Darwin wheel intake failed: "))
                self.assertEqual("", output.getvalue())
        with mock.patch.object(sys, "argv", ["stage"]), \
                mock.patch.object(WHEEL, "stage") as stage, contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as caught:
                WHEEL.main()
        self.assertEqual(2, caught.exception.code)
        stage.assert_not_called()

    def test_script_entry_stages_only_owned_bytes_and_never_reads_native_manifest(self) -> None:
        root, manifest, row, response, _, opener, _ = self.fixture()
        raw = manifest.read_bytes()
        output = root / "entry-stage"
        native_manifest = SCRIPTS.resolve().parents[1] / "sdk/python/darwin-qualified-wheels.json"

        def only_manifest(path: Path) -> bytes:
            self.assertEqual(native_manifest, path)
            return raw

        standard_output = io.StringIO()
        with mock.patch.dict(sys.modules, {"sdk_python_lock": LOCK}), \
                mock.patch.object(LOCK, "darwin_intake", return_value={"wheel": NAME, "wheel_uri": "not-retained"}) as intake, \
                mock.patch.object(Path, "read_bytes", autospec=True, side_effect=only_manifest) as read, \
                mock.patch.object(sys, "argv", ["stage", "--python", VERSION, "--output", str(output)]), \
                contextlib.redirect_stdout(standard_output):
            with self.assertRaises(SystemExit) as caught:
                runpy.run_path(str(SCRIPTS / "sdk_python_darwin_wheel.py"), run_name="__main__")
        self.assertEqual(0, caught.exception.code)
        self.assertEqual(NAME + "\n", standard_output.getvalue())
        read.assert_called_once_with(native_manifest)
        intake.assert_called_once_with(output / "archive.zip", output / NAME, VERSION, row["custody"])
        opener.open.assert_called_once()
        self.assertTrue(response.closed)
        self.assertEqual({"wheel": NAME}, json.loads((output / "intake.json").read_bytes()))


if __name__ == "__main__":
    unittest.main()
