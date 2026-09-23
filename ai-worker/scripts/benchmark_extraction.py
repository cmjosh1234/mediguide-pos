"""Read-only PDF/Markdown extraction benchmark. Does not queue or publish content."""
import argparse
import hashlib
import json
import time
from pathlib import Path

from app.document_processing.pdf_extractor import extract_pdf
from app.document_processing.markdown_extractor import extract_markdown
from app.document_processing.chunker import chunk_blocks, chunk_sections


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    args = parser.parse_args()
    digest = hashlib.sha256()
    with args.source.open("rb") as stream:
        for data in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(data)
    start = time.perf_counter()
    document = extract_pdf(args.source) if args.source.suffix.lower() == ".pdf" else extract_markdown(args.source)
    extract_seconds = time.perf_counter() - start
    start = time.perf_counter()
    chunks = chunk_blocks(document.blocks) or chunk_sections(document.sections)
    print(json.dumps({"benchmark": "extraction_only", "sha256": digest.hexdigest(), "source_bytes": args.source.stat().st_size,
                      "extraction_seconds": round(extract_seconds, 3), "chunking_seconds": round(time.perf_counter()-start, 3),
                      "pages": document.pages, "sections": len(document.sections), "blocks": len(document.blocks), "chunks": len(chunks),
                      "tables": len(document.tables), "assets": len(document.assets), "detail": document.metadata.get("extraction_timings", {})}, indent=2))


if __name__ == "__main__":
    main()
