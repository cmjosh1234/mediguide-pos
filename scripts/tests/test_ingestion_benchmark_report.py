import importlib.util
import json
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("report", Path(__file__).parents[1] / "ingestion-benchmark-report.py")
report = importlib.util.module_from_spec(spec)
spec.loader.exec_module(report)


class BenchmarkReportTests(unittest.TestCase):
    def test_filters_sensitive_fields_and_separates_noop_runs(self):
        rows = report.records(json.dumps([
            {"data": {"id": "job-1", "metrics": {"processing_seconds": 12, "stage_seconds": {"parsing": 10}, "secret": "omit"}, "payload_json": "omit"}},
            {"event": "ingestion_benchmark", "job_id": "job-2", "processing_seconds": 1, "checksum_noop": True},
        ]))
        summary = report.summarize(rows)
        self.assertEqual(summary["median_processing_seconds"], 12)
        self.assertEqual(summary["noop_runs"], 1)
        self.assertNotIn("omit", json.dumps(summary))

    def test_ignores_unrelated_log_lines(self):
        self.assertEqual(report.records('not json\n{"event":"unrelated"}'), [])


if __name__ == "__main__":
    unittest.main()
