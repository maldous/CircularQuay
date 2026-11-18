"""Attainable-input contracts for the fixed SDK row set and literal hook."""

from __future__ import annotations

import ast
import hashlib
import importlib.util
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SCRIPT = Path(__file__).resolve().parents[1] / "sdk_python_check.py"
SPEC = importlib.util.spec_from_file_location("cq_sdk_check_invariant_contracts", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
CHECK = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = CHECK
SPEC.loader.exec_module(CHECK)

EXPECTED_PATHS = frozenset({
    "sdk/python/src/cq/__init__.py",
    "sdk/python/src/cq/client.py",
    "sdk/python/src/cq/errors.py",
    "sdk/python/src/cq/local.py",
    "sdk/python/src/cq/types.py",
})


class FixedSDKInvariantContracts(unittest.TestCase):
    def test_complete_fixed_row_set_and_normalized_duplicates(self):
        self.assertEqual(CHECK.OWNED_COVERAGE_PATHS, EXPECTED_PATHS)
        self.assertLess(len(EXPECTED_PATHS), CHECK.MAX_COVERAGE_ROWS)
        capture = CHECK.Capture("coverage")
        for path in sorted(EXPECTED_PATHS):
            capture.consume((path + " 1 0 0 0 100%\n").encode())
        capture.finish()
        self.assertFalse(capture.malformed)
        self.assertEqual({row.path for row in capture.coverage_rows}, EXPECTED_PATHS)
        self.assertEqual(len(capture.coverage_rows), 5)
        admitted = tuple(capture.coverage_rows)
        for path in sorted(EXPECTED_PATHS):
            for spelling in (path, path.replace("/", "\\")):
                capture.consume((spelling + " 1 0 0 0 100%\n").encode())
                capture.finish()
                self.assertEqual(tuple(capture.coverage_rows), admitted)
        self.assertTrue(capture.malformed)
        self.assertEqual(capture.unparsed_sdk_lines, 10)

    def test_unlisted_sixth_path_is_refused_without_retention(self):
        capture = CHECK.Capture("coverage")
        for path in sorted(EXPECTED_PATHS):
            capture.consume((path + " 1 0 0 0 100%\n").encode())
        admitted = tuple(capture.coverage_rows)
        capture.consume(b"sdk/python/src/cq/unlisted.py 1 0 0 0 100%\n")
        capture.finish()
        self.assertTrue(capture.malformed)
        self.assertEqual(capture.unparsed_sdk_lines, 1)
        self.assertEqual(tuple(capture.coverage_rows), admitted)

    def test_literal_hook_ast_and_bytes_are_independent_of_owned_destination(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name).resolve()
            contents = []
            with mock.patch.object(subprocess, "run", side_effect=AssertionError("unexpected child")), \
                 mock.patch.object(subprocess, "Popen", side_effect=AssertionError("unexpected child")):
                for leaf in ("first", "second with spaces"):
                    directory = root / leaf
                    directory.mkdir(mode=0o700)
                    hook = CHECK.coverage_hook(directory)
                    self.assertEqual(hook, directory / "sitecustomize.py")
                    data = hook.read_bytes()
                    contents.append(data)
                    parsed = ast.parse(data, filename=str(hook))
                    self.assertIsInstance(parsed, ast.Module)
                    self.assertEqual({item.name for item in parsed.body if isinstance(item, ast.FunctionDef)},
                                     {"_write", "start", "stop"})
                    self.assertNotIn(str(root).encode(), data)
                    if os.name == "posix":
                        self.assertEqual(hook.stat().st_mode & 0o777, 0o600)
                    self.assertEqual(list(directory.iterdir()), [hook])
            self.assertEqual(contents[0], contents[1])
            self.assertEqual(hashlib.sha256(contents[0]).hexdigest(), "9ff2b8b64722d1c36d42df88b1bb6bf5c2ab44135aaafe9a3174efe943037ee1")


if __name__ == "__main__":
    unittest.main()
