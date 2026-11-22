"""Exact policy/parser/entry boundaries using only owned public-shaped JSON."""

from __future__ import annotations

import contextlib
import importlib.util
import io
import json
from pathlib import Path
import runpy
import sys
import tempfile
import unittest
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "sdk_python_darwin_source_policy.py"
SPEC = importlib.util.spec_from_file_location("darwin_policy_completion", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
POLICY = importlib.util.module_from_spec(SPEC)
with mock.patch.dict(sys.modules, {SPEC.name: POLICY}):
    SPEC.loader.exec_module(POLICY)

VALID = {"format": POLICY.POLICY_FORMAT, "platform": "darwin", "architecture": "x86_64",
         "package": "synthetic-package", "version": "1.2.3",
         "url": "https://files.pythonhosted.org/packages/owned/synthetic-package-1.2.3.tar.gz",
         "sha256": "a" * 64}


class DarwinPolicyCompletionContracts(unittest.TestCase):
    def setUp(self) -> None:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve(strict=True)
        self.path = self.root / "policy.json"
        self.write(VALID)

    def write(self, value: object) -> None:
        self.path.write_text(json.dumps(value), encoding="utf-8")

    def argv(self) -> list[str]:
        arguments = [str(SCRIPT), "--policy", str(self.path)]
        for name in ("platform", "architecture", "package", "version", "url", "sha256"):
            arguments += ["--" + name, VALID[name]]
        return arguments

    def test_each_scalar_validator_rejects_types_and_grammar_before_normalization(self) -> None:
        for check, valid, invalid in (
            (POLICY._checked_name, "SYNTHETIC__Package", (None, 1, "", "/private", "-name")),
            (POLICY._checked_version, "1.2.3+local", (None, True, "", "1/2", " version")),
            (POLICY._checked_digest, "a" * 64, (None, 3, "", "A" * 64, "a" * 63)),
        ):
            with self.subTest(check=check.__name__):
                self.assertIsInstance(check(valid), str)
                for value in invalid:
                    with self.subTest(value=value), self.assertRaises(POLICY.ValidationError):
                        check(value)
        self.assertEqual("synthetic-package", POLICY._checked_name("SYNTHETIC__Package"))

    def test_source_url_rejects_parse_errors_and_all_nonexact_authorities(self) -> None:
        urls = (None, "https://files.pythonhosted.org:invalid/pkg/synthetic-package-1.2.3.tar.gz",
                "https://[broken/pkg/synthetic-package-1.2.3.tar.gz",
                "http://files.pythonhosted.org/pkg/synthetic-package-1.2.3.tar.gz",
                "https://other.invalid/pkg/synthetic-package-1.2.3.tar.gz",
                "https://files.pythonhosted.org:443/pkg/synthetic-package-1.2.3.tar.gz",
                "https://user:password@files.pythonhosted.org/pkg/synthetic-package-1.2.3.tar.gz",
                VALID["url"] + "?query=1", VALID["url"] + "#fragment", VALID["url"] + ".other")
        for value in urls:
            with self.subTest(value=value), self.assertRaises(POLICY.ValidationError):
                POLICY._checked_source_url(value, VALID["package"], VALID["version"])
        self.assertEqual(VALID["url"], POLICY._checked_source_url(VALID["url"], VALID["package"], VALID["version"]))

    def test_closed_policy_schema_platform_and_format_are_not_inferred(self) -> None:
        variants = [None, [], dict(VALID, extra=True), {key: value for key, value in VALID.items() if key != "sha256"},
                    dict(VALID, format="other"), dict(VALID, platform="linux"),
                    dict(VALID, architecture="arm64")]
        for value in variants:
            self.write(value)
            with self.subTest(value=value), self.assertRaises(POLICY.ValidationError):
                POLICY.load_policy(self.path)
        self.write(VALID)
        policy = POLICY.load_policy(self.path)
        self.assertEqual(POLICY.SourcePolicy("darwin", "x86_64", VALID["package"], VALID["version"],
                                             VALID["url"], VALID["sha256"]), policy)

    def test_policy_read_errors_invalid_json_and_actual_size_growth_refuse(self) -> None:
        with self.assertRaisesRegex(POLICY.ValidationError, "policy is invalid"):
            POLICY.load_policy(self.root / "absent.json")
        for value in (b"{", b"\xff"):
            self.path.write_bytes(value)
            with self.subTest(value=value), self.assertRaisesRegex(POLICY.ValidationError, "policy is invalid"):
                POLICY.load_policy(self.path)
        with self.path.open("wb") as stream:
            stream.truncate(POLICY.MAX_POLICY_BYTES + 1)
        with self.assertRaisesRegex(POLICY.ValidationError, "policy exceeds size limit"):
            POLICY.load_policy(self.path)
        self.write(VALID)
        original_read = Path.read_bytes

        def grow_then_read(path):
            if path == self.path:
                with path.open("ab") as stream:
                    stream.write(b" " * (POLICY.MAX_POLICY_BYTES + 1))
            return original_read(path)

        with mock.patch.object(Path, "read_bytes", grow_then_read):
            with self.assertRaisesRegex(POLICY.ValidationError, "policy exceeds size limit"):
                POLICY.load_policy(self.path)
        self.assertGreater(self.path.stat().st_size, POLICY.MAX_POLICY_BYTES)

    def test_exact_proof_short_circuit_and_mismatch_preserve_frozen_policy(self) -> None:
        policy = POLICY.load_policy(self.path)
        proof = {key: VALID[key] for key in VALID if key != "format"}
        original = self.path.read_bytes()
        for field, value in (("platform", "linux"), ("architecture", "arm64"),
                             ("package", "other-package"), ("version", "1.2.4"),
                             ("url", VALID["url"].replace("/owned/", "/other/")), ("sha256", "b" * 64)):
            with self.subTest(field=field), self.assertRaisesRegex(POLICY.ValidationError, "source evidence does not match"):
                POLICY.validate_source_proof(policy, **dict(proof, **{field: value}))
        POLICY.validate_source_proof(policy, **proof)
        self.assertEqual(original, self.path.read_bytes())
        with self.assertRaises(AttributeError):
            policy.platform = "linux"

    def test_real_cli_success_and_validation_failure_keep_fixed_nonsecret_diagnostics(self) -> None:
        captured = io.StringIO()
        with mock.patch.object(sys, "argv", self.argv()), contextlib.redirect_stderr(captured):
            self.assertEqual(0, POLICY.main())
        self.assertEqual("", captured.getvalue())
        self.write(dict(VALID, format="unsupported"))
        with mock.patch.object(sys, "argv", self.argv()), contextlib.redirect_stderr(captured):
            self.assertEqual(2, POLICY.main())
        self.assertEqual("error: policy format is unsupported\n", captured.getvalue())
        self.assertNotIn(str(self.root), captured.getvalue())

    def test_script_entry_runs_only_owned_policy_and_no_fetch_or_build(self) -> None:
        original = self.path.read_bytes()
        with mock.patch.object(sys, "argv", self.argv()), self.assertRaises(SystemExit) as stopped:
            runpy.run_path(str(SCRIPT), run_name="__main__")
        self.assertEqual(0, stopped.exception.code)
        self.assertEqual(original, self.path.read_bytes())
        self.assertEqual({"policy.json"}, {path.name for path in self.root.iterdir()})


if __name__ == "__main__":
    unittest.main()
