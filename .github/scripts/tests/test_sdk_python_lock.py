"""Synthetic unit tests for the hosted Python lock-report validator."""

from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).parents[1] / "sdk_python_lock.py"
SPEC = importlib.util.spec_from_file_location("sdk_python_lock", SCRIPT)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("lock validator module is unavailable")
LOCK = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(LOCK)


HASH = "a" * 64


def package(
    name: str, version: str, url: str, *, yanked: bool = False
) -> dict[str, object]:
    """Return a minimal synthetic pip-report package."""
    return {
        "is_yanked": yanked,
        "metadata": {"name": name, "version": version},
        "download_info": {
            "url": url,
            "archive_info": {"hashes": {"sha256": HASH}},
        },
    }


class LockEntriesTest(unittest.TestCase):
    """Exercise the authority checks before CI trusts a generated lock."""

    def test_accepts_public_wheel_and_requested_pin(self) -> None:
        report = {
            "install": [
                package(
                    "Example_Package",
                    "1.2.3",
                    "https://files.pythonhosted.org/packages/"
                    "example_package-1.2.3-py3-none-any.whl",
                )
            ]
        }
        self.assertEqual(
            LOCK.lock_entries(report, {"example-package": "1.2.3"}),
            [("example-package", "1.2.3", HASH)],
        )

    def test_rejects_non_public_wheel_host(self) -> None:
        report = {
            "install": [
                package(
                    "example-package",
                    "1.2.3",
                    "https://packages.example.invalid/"
                    "example-package-1.2.3-py3-none-any.whl",
                )
            ]
        }
        with self.assertRaises(LOCK.ValidationError):
            LOCK.lock_entries(report, {"example-package": "1.2.3"})

    def test_rejects_yanked_or_unrequested_version(self) -> None:
        report = {
            "install": [
                package(
                    "example-package",
                    "1.2.4",
                    "https://files.pythonhosted.org/packages/"
                    "example_package-1.2.4-py3-none-any.whl",
                    yanked=True,
                )
            ]
        }
        with self.assertRaises(LOCK.ValidationError):
            LOCK.lock_entries(report, {"example-package": "1.2.3"})

    def test_rejects_duplicate_direct_lock_entry(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            lock = Path(directory) / "ci.txt"
            lock.write_text(
                "example-package==1.2.3 --hash=sha256:" + HASH + "\n"
                "example_package==1.2.3 --hash=sha256:" + HASH + "\n",
                encoding="utf-8",
            )
            with self.assertRaises(LOCK.ValidationError):
                LOCK.verify_lock(lock, {"example-package": "1.2.3"})


class DarwinIntakeTest(unittest.TestCase):
    """Prove that the exception cannot admit an arbitrary local dependency."""

    def setUp(self) -> None:
        self.intake = {
            "python": "3.14.10",
            "wheel_uri": "file:///reviewed/cryptography.whl",
            "wheel_sha256": HASH,
        }
        self.report = {
            "environment": {
                "sys_platform": "darwin",
                "platform_machine": "x86_64",
                "implementation_name": "cpython",
                "python_full_version": "3.14.10",
            },
            "install": [package("cryptography", "50.0.2", self.intake["wheel_uri"])],
        }

    def test_exact_local_wheel_and_native_target(self) -> None:
        self.assertEqual(
            LOCK.lock_entries(self.report, {}, self.intake),
            [("cryptography", "50.0.2", HASH)],
        )

    def test_rejects_other_target_version_location_and_digest(self) -> None:
        import copy

        for section, field, value in [
            ("environment", "sys_platform", "linux"),
            ("environment", "platform_machine", "arm64"),
            ("environment", "implementation_name", "pypy"),
            ("environment", "python_full_version", "3.14.9"),
            ("metadata", "version", "50.0.3"),
            ("download_info", "url", "file:///unreviewed/cryptography.whl"),
        ]:
            with self.subTest(field=field):
                report = copy.deepcopy(self.report)
                target = (
                    report[section]
                    if section == "environment"
                    else report["install"][0][section]
                )
                target[field] = value
                with self.assertRaises(LOCK.ValidationError):
                    LOCK.lock_entries(report, {}, self.intake)
        report = copy.deepcopy(self.report)
        report["install"][0]["download_info"]["archive_info"]["hashes"]["sha256"] = (
            "b" * 64
        )
        with self.assertRaises(LOCK.ValidationError):
            LOCK.lock_entries(report, {}, self.intake)

    def test_other_local_dependencies_and_public_crypto_substitution_fail(self) -> None:
        for entry in [
            package("other", "1.0", "file:///reviewed/other.whl"),
            package(
                "cryptography", "50.0.2", "https://files.pythonhosted.org/crypto.whl"
            ),
        ]:
            with self.subTest(entry=entry):
                report = {**self.report, "install": [entry]}
                with self.assertRaises(LOCK.ValidationError):
                    LOCK.lock_entries(report, {}, self.intake)

    def test_archive_and_wheel_bytes_must_match_reviewed_receipt(self) -> None:
        import hashlib
        import json
        import zipfile
        from unittest.mock import patch

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "sdk/python"
            source.mkdir(parents=True)
            policy = b"synthetic policy"
            (source / "darwin-source-policy.json").write_bytes(policy)
            wheel = root / "cryptography-50.0.2-cp314-abi3-macosx_10_15_x86_64.whl"
            wheel.write_bytes(b"synthetic wheel bytes")
            archive = root / "receipt.zip"
            with zipfile.ZipFile(archive, "w") as output:
                output.writestr(wheel.name, wheel.read_bytes())
            fixture = {
                "source_policy_sha256": hashlib.sha256(policy).hexdigest(),
                "run_id": 1,
                "workflow_head": "1" * 40,
                "feature": "2" * 40,
                "qualified_repeat_builds": [
                    {
                        "python": "3.14.10",
                        "wheel": wheel.name,
                        "custody": "approved:immutable-object-version",
                        "artifact_sha256": hashlib.sha256(
                            archive.read_bytes()
                        ).hexdigest(),
                        "wheel_sha256": hashlib.sha256(wheel.read_bytes()).hexdigest(),
                    }
                ],
            }
            (source / "darwin-qualified-wheels.json").write_text(json.dumps(fixture))
            with patch.object(
                LOCK, "__file__", str(root / ".github/scripts/sdk_python_lock.py")
            ):
                result = LOCK.darwin_intake(
                    archive, wheel, "3.14.10", "approved:immutable-object-version"
                )
                self.assertEqual(result["wheel_uri"], wheel.as_uri())
                with self.assertRaises(LOCK.ValidationError):
                    LOCK.darwin_intake(archive, wheel, "3.14.10", "unreviewed:object")
                fixture["qualified_repeat_builds"][0]["custody"] = None
                (source / "darwin-qualified-wheels.json").write_text(
                    json.dumps(fixture)
                )
                with self.assertRaises(LOCK.ValidationError):
                    LOCK.darwin_intake(
                        archive, wheel, "3.14.10", "approved:immutable-object-version"
                    )
                fixture["qualified_repeat_builds"][0]["custody"] = (
                    "approved:immutable-object-version"
                )
                (source / "darwin-qualified-wheels.json").write_text(
                    json.dumps(fixture)
                )
                wheel.write_bytes(b"tampered")
                with self.assertRaises(LOCK.ValidationError):
                    LOCK.darwin_intake(
                        archive, wheel, "3.14.10", "approved:immutable-object-version"
                    )
                archive.write_bytes(b"tampered archive")
                with self.assertRaises(LOCK.ValidationError):
                    LOCK.darwin_intake(
                        archive, wheel, "3.14.10", "approved:immutable-object-version"
                    )
