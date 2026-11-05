"""Synthetic unit tests for the hosted Python lock-report validator."""

from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).parents[1] / "sdk_python_lock.py"
SPEC = importlib.util.spec_from_file_location("sdk_python_lock", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
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

