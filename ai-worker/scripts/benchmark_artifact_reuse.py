"""Isolated artifact-reuse benchmark; no publication writes or network calls.

Uses temporary SQLite/filesystem checkpoint adapters and deterministic development
embeddings. Measures reuse/content equivalence, NOT production provider latency.
"""
import argparse
import hashlib
import json
import resource
import sqlite3
import tempfile
import time
from pathlib import Path

from app.core.config import Settings
from app.document_processing.artifact_identity import extraction_identity, embedding_identity, ocr_identity
from app.document_processing.pdf_extractor import extract_pdf
from app.document_processing.chunker import chunk_blocks, chunk_sections
from app.embeddings.hash_provider import HashEmbeddingProvider
from app.services.ingestion_artifacts import IngestionArtifacts, encode_document


class Checkpoints:
    def __init__(self, path):
        self.db = sqlite3.connect(path)
        self.db.execute("CREATE TABLE IF NOT EXISTS artifacts(kind TEXT,key TEXT,payload TEXT,PRIMARY KEY(kind,key))")

    def get_many(self, kind, keys):
        wanted = set(keys)
        return {key: json.loads(payload) for key, payload in self.db.execute("SELECT key,payload FROM artifacts WHERE kind=?", (kind,)) if key in wanted}

    def put_many(self, kind, values):
        self.db.executemany("INSERT OR REPLACE INTO artifacts VALUES (?,?,?)", [(kind, key, json.dumps(value)) for key, value in values.items()])
        self.db.commit()


class Files:
    def __init__(self, root):
        self.root = root

    def upload_bytes(self, data, key, content_type):
        path = self.root / key
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)

    def download_file(self, key, destination):
        destination.write_bytes((self.root / key).read_bytes())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    args = parser.parse_args()
    settings = Settings(embedding_provider="hash", embedding_dim=64)
    provider = HashEmbeddingProvider(64)
    checksum = hashlib.sha256(args.source.read_bytes()).hexdigest()
    key = extraction_identity(checksum, "pdf", settings)
    assert key is not None, "OCR identity unavailable; cannot benchmark full extraction reuse"
    engine = {"epoch": settings.artifact_cache_epoch, **ocr_identity()}
    model = embedding_identity(settings, provider)
    reports = []
    original_digest = None
    with tempfile.TemporaryDirectory(prefix="mediguide-reuse-benchmark-") as tmp:
        root = Path(tmp)
        for label in ("cold", "identical", "one_changed_embedding_input"):
            repository = Checkpoints(root / "checkpoints.sqlite")
            metrics = {}
            cache = IngestionArtifacts(repository, Files(root), metrics)
            start = time.perf_counter()
            document = cache.extraction(key, lambda: extract_pdf(args.source, artifacts=cache, ocr_settings=engine))
            extraction_seconds = time.perf_counter() - start
            digest = hashlib.sha256(encode_document(document)).hexdigest()
            if original_digest is None:
                original_digest = digest
            assert digest == original_digest, "Cached extraction differs from cold extraction"
            chunks = chunk_blocks(document.blocks) or chunk_sections(document.sections)
            texts = [chunk.content for chunk in chunks]
            if label == "one_changed_embedding_input":
                texts[0] += "\nBenchmark-only revision."
            start = time.perf_counter()
            cache.embeddings(texts, provider, model, 64, 64, lambda *_: None)
            reports.append({"scenario": label, "extraction_seconds": round(extraction_seconds, 3),
                "embedding_seconds": round(time.perf_counter() - start, 3), "metrics": metrics,
                "pages": document.pages, "sections": len(document.sections), "blocks": len(document.blocks),
                "tables": len(document.tables), "assets": len(document.assets), "chunks": len(chunks),
                "extraction_digest": digest})
            repository.db.close()
    print(json.dumps({"benchmark": "isolated_artifact_reuse", "sha256": checksum,
        "peak_rss_platform_units": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
        "runs": reports}, indent=2))


if __name__ == "__main__":
    main()
