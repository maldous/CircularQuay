"""Owned offline XML/root correspondence; no coverage counters are rewritten."""

from __future__ import annotations

import contextlib
import importlib.util
import io
import json
import tempfile
import unittest
import xml.etree.ElementTree as ET
from pathlib import Path
from unittest import mock
from xml.sax.saxutils import escape, quoteattr

SCRIPTS = Path(__file__).parents[1]
SPEC = importlib.util.spec_from_file_location("sonar_coverage_contract", SCRIPTS / "sonar_coverage_contract.py")
assert SPEC is not None and SPEC.loader is not None
COVERAGE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COVERAGE)

METRIC_SPEC = importlib.util.spec_from_file_location("sonar_metric_contract_shape", SCRIPTS / "sonar_metric_contract.py")
assert METRIC_SPEC is not None and METRIC_SPEC.loader is not None
METRICS = importlib.util.module_from_spec(METRIC_SPEC)
METRIC_SPEC.loader.exec_module(METRICS)

SDK_FILES = ("cq/__init__.py", "cq/client.py", "cq/errors.py", "cq/local.py", "cq/types.py")
HELPERS = (".github/scripts/sonar_metric_contract.py", ".github/scripts/sonar_coverage_contract.py")


def synthetic_report(filenames: tuple[str, ...], sources: bytes | None = None) -> bytes:
    """Include deliberately non-100 counters so preservation cannot mean coercion."""
    sources = b"<source></source><source>sdk/python/src</source>" if sources is None else sources
    classes = b"".join(
        ("<class filename=" + quoteattr(name) + ' line-rate="0.5"><lines>'
         '<line number="1" hits="0" branch="true" condition-coverage="50% (1/2)" />'
         "</lines></class>").encode("utf-8") for name in filenames
    )
    return (b'<?xml version="1.0" ?>\n<coverage lines-valid="7" lines-covered="6" '
            b'branches-valid="2" branches-covered="1" line-rate="0.5">\n<sources>' + sources +
            b"</sources>\n<packages><package><classes>" + classes +
            b"</classes></package></packages></coverage>\n")


class SonarCoverageContracts(unittest.TestCase):
    def fixture(self) -> tuple[Path, bytes, set[str]]:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name) / "checkout & <é>"
        expected = set()
        for name in SDK_FILES:
            path = root / "sdk/python/src" / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"# owned synthetic SDK file; never imported\n")
            expected.add(str(path))
        for name in HELPERS:
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"# owned synthetic helper; never imported\n")
            expected.add(str(path))
        return root, synthetic_report(SDK_FILES + HELPERS), expected

    def assert_invalid(self, operation, *arguments) -> None:
        with self.assertRaises(COVERAGE.InvalidCoverage) as caught:
            operation(*arguments)
        self.assertEqual(COVERAGE.INVALID_COVERAGE, str(caught.exception))

    def test_explicit_root_resolves_both_helpers_and_all_five_sdk_classes(self) -> None:
        root, raw, expected = self.fixture()
        # The old blank root is ignored, and the valid SDK root prevents any
        # default checkout fallback: the old helper mapping really refuses.
        self.assert_invalid(COVERAGE.resolved_file_paths, raw, str(root))
        fixed = COVERAGE.with_repository_source(raw, str(root))
        self.assertEqual(expected, COVERAGE.resolved_file_paths(fixed, str(root)))
        texts = [item.text for item in ET.fromstring(fixed).findall("./sources/source")]
        self.assertEqual([None, "sdk/python/src", str(root)], texts)

    def test_insertion_preserves_every_old_byte_class_counter_and_source(self) -> None:
        root, raw, _ = self.fixture()
        fixed = COVERAGE.with_repository_source(raw, str(root))
        insertion = b"<source>" + escape(str(root)).encode("ascii", "xmlcharrefreplace") + b"</source>\n\t"
        self.assertEqual(raw, fixed.replace(insertion, b"", 1))
        before, after = ET.fromstring(raw), ET.fromstring(fixed)
        self.assertEqual(before.attrib, after.attrib)
        self.assertEqual([ET.tostring(item) for item in before.findall(".//class")],
                         [ET.tostring(item) for item in after.findall(".//class")])
        self.assertEqual([item.text for item in before.findall("./sources/source")],
                         [item.text for item in after.findall("./sources/source")][:-1])

    def test_existing_root_is_idempotent_without_duplicate_resolution(self) -> None:
        root, raw, expected = self.fixture()
        once = COVERAGE.with_repository_source(raw, str(root))
        self.assertEqual(once, COVERAGE.with_repository_source(once, str(root)))
        self.assertEqual(expected, COVERAGE.resolved_file_paths(once, str(root)))

    def test_dot_and_blank_roots_are_not_implicit_resolution(self) -> None:
        root, _, expected = self.fixture()
        raw = synthetic_report(SDK_FILES + HELPERS, b"<source> </source><source>.</source><source>sdk/python/src</source>")
        self.assert_invalid(COVERAGE.resolved_file_paths, raw, str(root))
        fixed = COVERAGE.with_repository_source(raw, str(root))
        self.assertEqual(expected, COVERAGE.resolved_file_paths(fixed, str(root)))

    def test_missing_and_ambiguous_files_refuse_instead_of_dropping_classes(self) -> None:
        root, raw, _ = self.fixture()
        fixed = COVERAGE.with_repository_source(raw, str(root))
        duplicate_sdk = fixed.replace(b"</sources>", b"<source>sdk/python/src</source></sources>")
        self.assert_invalid(COVERAGE.resolved_file_paths, duplicate_sdk, str(root))
        duplicate_root = fixed.replace(b"</sources>", ("<source>" + escape(str(root)) + "</source></sources>").encode())
        self.assert_invalid(COVERAGE.resolved_file_paths, duplicate_root, str(root))
        (root / HELPERS[0]).unlink()
        self.assert_invalid(COVERAGE.resolved_file_paths, fixed, str(root))

    def test_absolute_class_is_not_duplicated_across_multiple_roots(self) -> None:
        root, _, _ = self.fixture()
        name = str(root / HELPERS[0])
        raw = synthetic_report((name,))
        fixed = COVERAGE.with_repository_source(raw, str(root))
        self.assertEqual({name}, COVERAGE.resolved_file_paths(fixed, str(root)))
        missing = synthetic_report((str(root / "absent.py"),))
        self.assert_invalid(COVERAGE.resolved_file_paths, missing, str(root))

    def test_resolution_refuses_absent_roots_names_and_classes(self) -> None:
        root, _, _ = self.fixture()
        for raw in (
            synthetic_report(("cq/client.py",), b"<source>absent-root</source>"),
            synthetic_report(("cq/client.py",), b"<source> </source><source>.</source>"),
            synthetic_report(("",)),
            synthetic_report(()),
        ):
            with self.subTest(report=raw[:24]):
                self.assert_invalid(COVERAGE.resolved_file_paths, raw, str(root))

    def test_generated_envelope_refuses_malformed_entities_and_ambiguous_sources(self) -> None:
        root, raw, _ = self.fixture()
        for invalid in (
            None, b"", b"not XML", b"<other><sources></sources></other>",
            b"<!DOCTYPE coverage [<!ENTITY private 'synthetic'>]><coverage><sources /></coverage>",
            b"<coverage><sources></sources><sources></sources></coverage>",
            b"<coverage><packages /></coverage>",
            b"<coverage><sources /></coverage>",
            raw.replace(b"</sources>", b"</sources><!-- </sources> -->", 1),
        ):
            with self.subTest(report=repr(invalid)[:24]):
                self.assert_invalid(COVERAGE.with_repository_source, invalid, str(root))

    def test_repository_argument_is_explicit_absolute_and_existing(self) -> None:
        root, raw, _ = self.fixture()
        for invalid in (None, "relative-checkout", str(root / "absent"), "synthetic\x00root"):
            with self.subTest(repository=repr(invalid)):
                self.assert_invalid(COVERAGE.with_repository_source, raw, invalid)
        self.assert_invalid(COVERAGE.resolved_file_paths, raw, "relative-checkout")

    def test_input_and_output_bounds_include_the_inserted_root(self) -> None:
        root, raw, _ = self.fixture()
        fixed = COVERAGE.with_repository_source(raw, str(root))
        with mock.patch.object(COVERAGE, "MAX_COVERAGE_BYTES", len(fixed)):
            self.assertEqual(fixed, COVERAGE.with_repository_source(raw, str(root)))
            self.assertEqual(fixed, COVERAGE.with_repository_source(fixed, str(root)))
        with mock.patch.object(COVERAGE, "MAX_COVERAGE_BYTES", len(fixed) - 1):
            self.assert_invalid(COVERAGE.with_repository_source, raw, str(root))
            self.assert_invalid(COVERAGE.with_repository_source, fixed, str(root))

    def test_current_only_response_shape_still_requires_both_new_ratios(self) -> None:
        # Public field shape from the failed run, not an assumption about why
        # its unretained parser call produced a different fixed label.
        document = {"component": {"key": "cq", "name": "CircularQuay", "qualifier": "TRK", "measures": [
            {"metric": "coverage", "value": "73.6", "bestValue": False},
            {"metric": "duplicated_lines_density", "value": "0.0", "bestValue": True},
            {"metric": "violations", "value": "1", "bestValue": False},
            {"metric": "security_hotspots", "value": "0", "bestValue": True},
        ]}}
        encoded = json.dumps(document, separators=(",", ":")).encode()
        expected = ("sonar-coverage-evidence-missing", "sonar-duplication-evidence-missing")
        self.assertEqual(expected, METRICS.evaluate_metrics(encoded))
        root, _, _ = self.fixture()
        path = root / "current-metrics.json"
        path.write_bytes(encoded)
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            status = METRICS.main([str(path)])
        self.assertEqual(2, status)
        self.assertEqual("\n".join(expected) + "\n", output.getvalue())


if __name__ == "__main__":
    unittest.main()
