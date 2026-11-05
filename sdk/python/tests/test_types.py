"""Contract tests for public typed values without a transport or local process."""

from __future__ import annotations

import unittest
from datetime import UTC, datetime

from cq import SecretDisclosure, ToolCatalog, ToolName, WorkloadRun, WorkloadStatus


class PublicTypesTests(unittest.TestCase):
    def test_tool_catalog_accepts_only_advertised_known_names(self) -> None:
        catalog = ToolCatalog((ToolName.LIST_WORKLOADS, ToolName.GET_STATUS))
        self.assertTrue(catalog.includes(ToolName.LIST_WORKLOADS))
        self.assertTrue(catalog.includes("get_status"))
        self.assertFalse(catalog.includes("read_secret"))
        self.assertFalse(catalog.includes("unknown"))

    def test_disclosure_keeps_plaintext_out_of_representations_and_metadata(self) -> None:
        marker = "SYNTHETIC_TYPES_SECRET"
        expiry = datetime(2027, 1, 2, 3, 4, 5, tzinfo=UTC)
        disclosure = SecretDisclosure("alias", "version-1", expiry, marker)
        self.assertEqual(marker, disclosure.reveal())
        self.assertNotIn(marker, repr(disclosure))
        self.assertEqual(
            {
                "alias": "alias",
                "value": "[REDACTED]",
                "version": "version-1",
                "expires_at": expiry.isoformat(),
            },
            disclosure.redacted_dict(),
        )

    def test_public_result_records_are_immutable_value_objects(self) -> None:
        status = WorkloadStatus("alpha", "online", "ready")
        run = WorkloadRun("run-1", "completed", 0)
        self.assertEqual(("alpha", "online", "ready"), (status.workload, status.mode, status.state))
        self.assertEqual(("run-1", "completed", 0), (run.identifier, run.status, run.exit_code))
        with self.assertRaises(AttributeError):
            status.state = "error"  # type: ignore[misc]

    def test_catalog_can_represent_the_complete_v1_enum(self) -> None:
        catalog = ToolCatalog(tuple(ToolName))
        self.assertEqual(4, len(catalog.available))
        self.assertEqual(
            ("list_workloads", "get_status", "run_workload", "read_secret"),
            tuple(str(tool) for tool in catalog.available),
        )
