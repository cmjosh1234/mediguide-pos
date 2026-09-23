"""Cold extraction benchmark with semantic digest and container peak memory.

Run each configuration in a fresh, equally limited container. Compare digests
before comparing speed. No source modifications or publication writes.
"""
import argparse
from dataclasses import asdict
import hashlib
import json
from pathlib import Path
import resource
import statistics
import time

from app.document_processing.pdf_extractor import extract_pdf
from app.document_processing.chunker import chunk_blocks, chunk_sections
from app.document_processing.artifact_identity import identity


def measure(args):
    started = time.perf_counter()
    document = extract_pdf(args.source, page_workers=args.page_workers, ocr_workers=args.ocr_workers)
    duration = time.perf_counter() - started
    chunks = chunk_blocks(document.blocks) or chunk_sections(document.sections)
    canonical = asdict(document)
    canonical["metadata"].pop("extraction_timings", None)
    for asset in canonical["assets"]:
        if asset["data"] is not None:
            asset["data"] = hashlib.sha256(asset["data"]).hexdigest()
    memory = Path("/sys/fs/cgroup/memory.peak")
    return {"benchmark": "cold_processing", "source_sha256": hashlib.sha256(args.source.read_bytes()).hexdigest(),
        "page_workers": args.page_workers, "ocr_workers": args.ocr_workers, "extraction_seconds": round(duration, 3),
        "content_digest": identity(canonical), "chunks_digest": identity([asdict(chunk) for chunk in chunks]),
        "pages": document.pages, "blocks": len(document.blocks), "tables": len(document.tables), "assets": len(document.assets),
        "chunks": len(chunks), "stage_timings": document.metadata.get("extraction_timings", {}),
        "container_peak_memory_bytes": int(memory.read_text()) if memory.exists() else None,
        "parent_peak_rss_platform_units": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("--page-workers", type=int, choices=range(1, 5), default=1)
    parser.add_argument("--ocr-workers", type=int, choices=range(1, 5), default=1)
    parser.add_argument("--repeats", type=int, choices=range(1, 11), default=1)
    args = parser.parse_args()
    runs = [measure(args) for _ in range(args.repeats)]
    assert len({(row["content_digest"], row["chunks_digest"]) for row in runs}) == 1, "Output changed across runs"
    print(json.dumps({"runs": runs, "median_seconds": statistics.median(row["extraction_seconds"] for row in runs),
        "slowest_seconds": max(row["extraction_seconds"] for row in runs)}, indent=2))


if __name__ == "__main__":
    main()
