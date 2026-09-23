#!/usr/bin/env python3
"""Summarize safe metrics exported from upload-jobs API or worker JSON logs.

Usage: python3 scripts/ingestion-benchmark-report.py job.json [job2.json ...]
Only timings/counts are emitted. Payloads, document text and signed URLs are omitted.
"""
import argparse
import json
import statistics
from pathlib import Path


def records(text):
    try:
        value = json.loads(text)
        values = value if isinstance(value, list) else [value]
    except json.JSONDecodeError:
        values = []
        for line in text.splitlines():
            try:
                values.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    result = []
    for value in values:
        if not isinstance(value, dict):
            continue
        value = value.get("data", value)
        if not isinstance(value, dict):
            continue
        metrics = value.get("metrics") or value.get("metrics_json")
        if value.get("event") == "ingestion_benchmark":
            metrics = value
        if isinstance(metrics, str):
            metrics = json.loads(metrics)
        if not isinstance(metrics, dict) or "processing_seconds" not in metrics:
            continue
        safe = {key: metrics[key] for key in ("upload_session_seconds", "verification_and_queue_seconds", "upload_source_bytes", "processing_seconds", "queue_seconds", "stage_seconds", "source_bytes", "pages", "blocks", "sections", "tables", "assets", "chunks", "extraction", "checksum_noop", "source_format") if key in metrics}
        safe["job_id"] = value.get("id", value.get("job_id"))
        result.append(safe)
    return result


def summarize(rows):
    full = [row for row in rows if not row.get("checksum_noop")]
    durations = sorted(row["processing_seconds"] for row in full)
    return {"runs": rows, "full_runs": len(full), "noop_runs": len(rows) - len(full),
            "median_processing_seconds": statistics.median(durations) if durations else None,
            "max_processing_seconds": max(durations) if durations else None}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("files", nargs="+", type=Path)
    args = parser.parse_args()
    rows = [row for path in args.files for row in records(path.read_text())]
    if not rows:
        parser.error("No ingestion metrics found. Deploy migration 00054 and the instrumented worker, then export a completed job.")
    print(json.dumps(summarize(rows), indent=2))
