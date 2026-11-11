"""Unit tests for the narrow Darwin Intel source-policy validator."""

from __future__ import annotations

import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).parents[1] / "sdk_python_darwin_source_policy.py"
SPEC = importlib.util.spec_from_file_location("darwin_source_policy", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
POLICY = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = POLICY
SPEC.loader.exec_module(POLICY)


VALID = {
    "format": "cq-python-darwin-source-policy-v1",
    "platform": "darwin",
    "architecture": "x86_64",
    "package": "cryptography",
    "version": "50.0.2",
    "url": (
        "https://files.pythonhosted.org/packages/9d/af/"
        "182eb91b0df3fe75c4d9f26fe70684569566745f6ba7e5c9c73a862c5252/"
        "cryptography-50.0.2.tar.gz"
    ),
    "sha256": "7b46165bb56eb4704e2eaaf86f3c940d19154535d9b0ca7d6d590b04060e00d5",
}


def policy_from(value: dict[str, object]) -> object:
    with tempfile.TemporaryDirectory() as directory:
        path = Path(directory) / "policy.json"
        path.write_text(json.dumps(value), encoding="utf-8")
        return POLICY.load_policy(path)


class DarwinSourcePolicyTest(unittest.TestCase):
    """Keep the exception exact, platform-scoped and source-only."""

    def test_accepts_exact_evidence(self) -> None:
        policy = policy_from(VALID)
        POLICY.validate_source_proof(policy, **{key: VALID[key] for key in VALID if key != "format"})

    def test_rejects_other_platform_or_package(self) -> None:
        policy = policy_from(VALID)
        proof = {key: VALID[key] for key in VALID if key != "format"}
        for key, value in (("platform", "linux"), ("package", "other-package")):
            changed = dict(proof)
            changed[key] = value
            with self.subTest(key=key), self.assertRaises(POLICY.ValidationError):
                POLICY.validate_source_proof(policy, **changed)

    def test_rejects_changed_url_hash_and_unknown_schema(self) -> None:
        proof = {key: VALID[key] for key in VALID if key != "format"}
        policy = policy_from(VALID)
        for key, value in (("url", VALID["url"] + "?x=1"), ("sha256", "a" * 64)):
            changed = dict(proof)
            changed[key] = value
            with self.subTest(key=key), self.assertRaises(POLICY.ValidationError):
                POLICY.validate_source_proof(policy, **changed)
        malformed = dict(VALID)
        malformed["extra"] = True
        with self.assertRaises(POLICY.ValidationError):
            policy_from(malformed)
