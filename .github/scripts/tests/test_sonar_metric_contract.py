"""Offline strict Sonar schema, threshold, privacy and bounded-input contracts."""

from __future__ import annotations

import contextlib
import copy
import importlib.util
import io
import json
import os
import runpy
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

SCRIPT = Path(__file__).parents[1] / "sonar_metric_contract.py"
SPEC = importlib.util.spec_from_file_location("sonar_metric_contract", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
METRICS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(METRICS)


def response() -> dict:
    """Return fabricated values in the pinned current and period-only shapes."""
    return {"component": {"key": "cq", "measures": [
        {"metric": "coverage", "value": "100.0"},
        {"metric": "new_coverage", "period": {"index": 1, "value": "100.0"}},
        {"metric": "duplicated_lines_density", "value": "0.0"},
        {"metric": "new_duplicated_lines_density", "period": {"index": 1, "value": "0.0"}},
        {"metric": "violations", "value": "0"},
        {"metric": "security_hotspots", "value": "0"},
    ]}}


def encoded(document: object) -> bytes:
    return json.dumps(document, separators=(",", ":")).encode("utf-8")


class SonarMetricContracts(unittest.TestCase):
    def assert_codes(self, document: object, *codes: str) -> None:
        self.assertEqual(codes, METRICS.evaluate_metrics(encoded(document)))

    def test_pinned_period_values_and_exact_thresholds_pass(self) -> None:
        document = response()
        original = copy.deepcopy(document)
        self.assert_codes(document)
        self.assertEqual(original, document)
        document["component"]["measures"][1]["period"].pop("index")
        document["component"]["measures"][3]["period"].pop("index")
        self.assert_codes(document)  # The shipped example permits an omitted index.
        document["component"]["measures"] = document["component"]["measures"][:4]
        self.assert_codes(document)  # Separate issue/hotspot gates remain separate.

    def test_each_current_and_new_threshold_is_mandatory(self) -> None:
        for index, value, code in [
            (0, "99.9", "sonar-coverage-not-100"),
            (1, "99.99", "sonar-coverage-not-100"),
            (2, "0.1", "sonar-duplication-not-zero"),
            (3, "0.01", "sonar-duplication-not-zero"),
        ]:
            with self.subTest(metric=index):
                document = response()
                self.set_percentage(document, index, value)
                self.assert_codes(document, code)
        document = response()
        document["component"]["measures"][0]["value"] = "73.9"
        document["component"]["measures"][2]["value"] = "1.0"
        self.assert_codes(document, "sonar-coverage-not-100", "sonar-duplication-not-zero")

    def test_missing_baseline_and_each_ratio_remain_failures(self) -> None:
        for index, code in [(0, "sonar-coverage-evidence-missing"), (1, "sonar-coverage-evidence-missing"),
                            (2, "sonar-duplication-evidence-missing"), (3, "sonar-duplication-evidence-missing")]:
            with self.subTest(metric=index):
                document = response()
                document["component"]["measures"].pop(index)
                self.assert_codes(document, code)
        document = response()
        document["component"]["measures"] = document["component"]["measures"][::2]
        self.assert_codes(document, "sonar-coverage-evidence-missing", "sonar-duplication-evidence-missing")

    def test_duplicate_and_unknown_metric_identities_are_not_cardinality_proofs(self) -> None:
        for replacement in [response()["component"]["measures"][0],
                            {"metric": "private-unrequested-metric", "value": "0"},
                            {"metric": ["coverage"], "value": "100"},
                            {"value": "100"}, "private-non-object-measure"]:
            with self.subTest(shape=type(replacement).__name__):
                document = response()
                document["component"]["measures"][1] = replacement
                self.assert_codes(document, METRICS.INVALID_EVIDENCE)
        document = response()
        document["component"]["measures"][5] = document["component"]["measures"][4]
        self.assert_codes(document, METRICS.INVALID_EVIDENCE)

    def test_percentages_refuse_nonfinite_coerced_or_out_of_range_values(self) -> None:
        for value in [None, True, 100, 100.0, "NaN", "Infinity", "-Infinity", "1e2", " 100", "+100",
                      "-0", "01", "100.1", "9" * 33, "", "0x0"]:
            with self.subTest(value_type=type(value).__name__):
                document = response()
                self.set_percentage(document, 1, value)
                self.assert_codes(document, "sonar-coverage-evidence-invalid")
                document = response()
                self.set_percentage(document, 2, value)
                self.assert_codes(document, "sonar-duplication-evidence-invalid")

    def test_current_and_period_shapes_cannot_fall_back_or_conflict(self) -> None:
        for replacement in [
            {"metric": "new_coverage", "value": "100"},
            {"metric": "new_coverage", "value": "100", "period": {"value": "100"}},
            {"metric": "new_coverage", "period": []},
            {"metric": "new_coverage", "period": {}},
            {"metric": "new_coverage", "period": {"index": True, "value": "100"}},
            {"metric": "new_coverage", "period": {"index": 2, "value": "100"}},
        ]:
            with self.subTest(shape=replacement):
                document = response()
                document["component"]["measures"][1] = replacement
                self.assert_codes(document, "sonar-coverage-evidence-invalid")
        document = response()
        document["component"]["measures"][0]["period"] = {"value": "100"}
        self.assert_codes(document, "sonar-coverage-evidence-invalid")
        document["component"]["measures"][0].pop("value")
        self.assert_codes(document, "sonar-coverage-evidence-invalid")

    def test_optional_count_shapes_cannot_smuggle_invalid_evidence(self) -> None:
        for replacement in [{"metric": "violations", "value": True},
                            {"metric": "violations", "value": "0.0"},
                            {"metric": "violations", "value": "-1"},
                            {"metric": "violations", "value": "1" * 33},
                            {"metric": "violations", "value": "0", "period": {"value": "0"}}]:
            with self.subTest(shape=replacement):
                document = response()
                document["component"]["measures"][4] = replacement
                self.assert_codes(document, METRICS.INVALID_EVIDENCE)

    def test_document_identity_shape_and_size_are_bounded(self) -> None:
        for document in [None, [], {}, {"component": None},
                         {"component": {"key": "private-other-project", "measures": []}},
                         {"component": {"key": "cq", "measures": {}}},
                         {"component": {"key": "cq", "measures": [None] * 7}}]:
            with self.subTest(shape=type(document).__name__):
                self.assert_codes(document, METRICS.INVALID_EVIDENCE)
        for raw in [b"", "not-bytes", b"\xff", b"{", b'{"component":{},"component":{}}', b'{"private":NaN}',
                    b'{"private":Infinity}', b'{"private":-Infinity}', encoded(response()) + b" null"]:
            with self.subTest(raw_type=type(raw).__name__):
                self.assertEqual((METRICS.INVALID_EVIDENCE,), METRICS.evaluate_metrics(raw))
        padded = encoded(response()).ljust(METRICS.MAX_INPUT_BYTES, b" ")
        self.assertEqual((), METRICS.evaluate_metrics(padded))
        self.assertEqual((METRICS.INVALID_EVIDENCE,), METRICS.evaluate_metrics(padded + b" "))
        with mock.patch.object(METRICS.json, "loads", side_effect=RecursionError("private-parser-detail")):
            self.assertEqual((METRICS.INVALID_EVIDENCE,), METRICS.evaluate_metrics(b"{}"))

    def test_cli_pass_threshold_and_evidence_exits_are_sanitized(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "private-metrics.json"
            path.write_bytes(encoded(response()))
            self.assert_main([str(path)], 0, "")
            document = response()
            document["component"]["measures"][0]["value"] = "73.9"
            path.write_bytes(encoded(document))
            self.assert_main([str(path)], 1, "sonar-coverage-not-100\n")
            document["component"]["measures"].pop(1)
            path.write_bytes(encoded(document))
            self.assert_main([str(path)], 2, "sonar-coverage-evidence-missing\n")
            path.write_bytes(b'{"private-api-body":"must-not-echo"}')
            self.assert_main([str(path)], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.SCHEMA_STAGE + "\n")
            self.assert_main([str(Path(directory) / "private-missing-file")], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")
            self.assert_main([directory], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")
        self.assert_main([], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.ARGUMENT_STAGE + "\n")
        self.assert_main(["private-argument", "private-extra"], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.ARGUMENT_STAGE + "\n")
        self.assert_main(["private\x00path"], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")

    def test_stage_markers_distinguish_real_read_refusal_from_schema_refusal(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "private-metrics.json"
            raw = b'{"private-api-body":"must-not-echo"}'
            path.write_bytes(raw)
            with mock.patch.object(METRICS, "evaluate_metrics", wraps=METRICS.evaluate_metrics) as evaluate:
                self.assert_main([str(Path(directory) / "private-missing-file")], 2,
                                 METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")
                evaluate.assert_not_called()
                self.assert_main([str(path)], 2,
                                 METRICS.INVALID_EVIDENCE + "\n" + METRICS.SCHEMA_STAGE + "\n")
                evaluate.assert_called_once_with(raw)
            with mock.patch.object(METRICS, "read_evidence", wraps=METRICS.read_evidence) as read:
                self.assert_main([], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.ARGUMENT_STAGE + "\n")
                read.assert_not_called()

    def test_current_only_shape_still_requires_both_new_ratios_without_a_stage_failure(self) -> None:
        document = {"component": {"key": "cq", "name": "CircularQuay", "qualifier": "TRK", "measures": [
            {"metric": "coverage", "value": "74.6", "bestValue": False},
            {"metric": "duplicated_lines_density", "value": "0.0", "bestValue": True},
            {"metric": "violations", "value": "1", "bestValue": False},
            {"metric": "security_hotspots", "value": "0", "bestValue": True},
        ]}}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "current-metrics.json"
            path.write_bytes(encoded(document))
            self.assert_main([str(path)], 2,
                             "sonar-coverage-evidence-missing\nsonar-duplication-evidence-missing\n")

    def test_owned_file_bounds_final_link_and_metadata_drift_refuse(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "metrics.json"
            padded = encoded(response()).ljust(METRICS.MAX_INPUT_BYTES, b" ")
            path.write_bytes(padded)
            self.assert_main([str(path)], 0, "")
            path.write_bytes(padded + b" ")
            self.assert_main([str(path)], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")
            path.write_bytes(b"")
            self.assert_main([str(path)], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")
            path.write_bytes(encoded(response()))
            link = Path(directory) / "private-link"
            link.symlink_to(path)
            self.assert_main([str(link)], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")
            self.assert_mutation_refused(path, "mtime")
            self.assertEqual(encoded(response()), path.read_bytes())

    def test_owned_descriptor_short_and_grown_reads_refuse(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "metrics.json"
            path.write_bytes(encoded(response()))
            self.assert_mutation_refused(path, "shorten")
            path.write_bytes(encoded(response()))
            self.assert_mutation_refused(path, "grow")

    def test_script_entry_point_uses_only_the_owned_fixture(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "metrics.json"
            path.write_bytes(encoded(response()))
            with mock.patch.object(sys, "argv", [str(SCRIPT), str(path)]):
                with (contextlib.redirect_stdout(io.StringIO()) as output,
                      contextlib.redirect_stderr(io.StringIO()) as errors):
                    with self.assertRaises(SystemExit) as result:
                        runpy.run_path(str(SCRIPT), run_name="__main__")
            self.assertEqual(0, result.exception.code)
            self.assertEqual("", output.getvalue())
            self.assertEqual("", errors.getvalue())

    def set_percentage(self, document: dict, index: int, value: object) -> None:
        measure = document["component"]["measures"][index]
        if index in (1, 3):
            measure["period"]["value"] = value
        else:
            measure["value"] = value

    def assert_mutation_refused(self, path: Path, mutation: str) -> None:
        real_fstat = os.fstat
        calls = 0

        def observe_and_mutate(descriptor: int):
            nonlocal calls
            observed = real_fstat(descriptor)
            calls += 1
            if calls == 1:
                self.mutate_owned_file(path, mutation, observed.st_mtime_ns)
            return observed

        with mock.patch.object(METRICS.os, "fstat", side_effect=observe_and_mutate):
            self.assert_main([str(path)], 2, METRICS.INVALID_EVIDENCE + "\n" + METRICS.READ_STAGE + "\n")
        self.assertEqual(2, calls)

    def mutate_owned_file(self, path: Path, mutation: str, modified: int) -> None:
        if mutation == "mtime":
            os.utime(path, ns=(modified, modified + 2_000_000_000))
        elif mutation == "shorten":
            path.write_bytes(path.read_bytes()[:-1])
        else:
            self.assertEqual("grow", mutation)
            path.write_bytes(path.read_bytes() + b" ")

    def assert_main(self, arguments: list[str], status: int, text: str) -> None:
        with contextlib.redirect_stdout(io.StringIO()) as output, contextlib.redirect_stderr(io.StringIO()) as errors:
            self.assertEqual(status, METRICS.main(arguments))
        self.assertEqual(text, output.getvalue())
        self.assertEqual("", errors.getvalue())


if __name__ == "__main__":
    unittest.main()
