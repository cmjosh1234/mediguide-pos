"""Reusable computation only; never cache document IDs or editorial approval."""
import base64
import gzip
import hashlib
import json
import math
import tempfile
import threading
from dataclasses import asdict
from pathlib import Path
import structlog
from app.document_processing.artifact_identity import identity
from app.document_processing.bounded_work import ordered_work
from app.document_processing.types import ExtractedDocument, ExtractedSection, ExtractedTable, ExtractedContentBlock, ExtractedAsset

log = structlog.get_logger()
MAX_ARTIFACT_BYTES = 256 << 20


def encode_document(document):
    value = asdict(document)
    for asset in value["assets"]:
        if asset["storage_key"] is not None:
            raise ValueError("Only raw extraction may be cached")
        asset["data"] = base64.b64encode(asset["data"]).decode() if asset["data"] is not None else None
    return json.dumps(value, ensure_ascii=False, allow_nan=False).encode()


def decode_document(data):
    value = json.loads(data)
    value["sections"] = [ExtractedSection(**row) for row in value["sections"]]
    value["tables"] = [ExtractedTable(**{**row, "bbox": tuple(row["bbox"]) if row["bbox"] else None}) for row in value["tables"]]
    value["blocks"] = [ExtractedContentBlock(**row) for row in value["blocks"]]
    value["assets"] = [ExtractedAsset(**{**row, "data": base64.b64decode(row["data"], validate=True) if row["data"] is not None else None}) for row in value["assets"]]
    if any(asset.storage_key is not None for asset in value["assets"]):
        raise ValueError("Artifact contains document-specific references")
    return ExtractedDocument(**value)


class IngestionArtifacts:
    def __init__(self, repository, storage, metrics):
        self.repository, self.storage, self.metrics = repository, storage, metrics
        self._extraction_complete = True
        self._metrics_lock = threading.Lock()

    def _increment(self, name):
        with self._metrics_lock:
            self.metrics[name] = self.metrics.get(name, 0) + 1

    def extraction_incomplete(self):
        # Preserve best-effort extraction, but never freeze a transient failure
        # into a reusable full-document result.
        self._extraction_complete = False
        self.metrics["extraction_cache_incomplete"] = True

    @property
    def extraction_complete(self):
        return self._extraction_complete

    def extraction(self, key, compute):
        if key is None:
            return compute()
        entry = self.repository.get_many("extraction", [key]).get(key)
        if entry:
            try:
                digest = entry["sha256"]
                if not isinstance(digest, str) or len(digest) != 64 or any(c not in "0123456789abcdef" for c in digest):
                    raise ValueError("Invalid artifact checksum")
                object_key = f"ingestion-cache/v1/extraction/{key}/{digest}.json.gz"
                with tempfile.TemporaryDirectory(prefix="mediguide-cache-") as tmp:
                    path = Path(tmp) / "artifact.gz"
                    self.storage.download_file(object_key, path)
                    with gzip.open(path, "rb") as stream:
                        data = stream.read(MAX_ARTIFACT_BYTES + 1)
                if len(data) > MAX_ARTIFACT_BYTES or hashlib.sha256(data).hexdigest() != entry["sha256"]:
                    raise ValueError("Artifact integrity mismatch")
                document = decode_document(data)
                self.metrics["extraction_cache_hits"] = self.metrics.get("extraction_cache_hits", 0) + 1
                return document
            except Exception:
                log.warning("extraction_cache_invalid_or_unavailable")
        self.metrics["extraction_cache_misses"] = self.metrics.get("extraction_cache_misses", 0) + 1
        self._extraction_complete = True
        document = compute()
        if not self._extraction_complete:
            return document
        try:
            data = encode_document(document)
            if len(data) <= MAX_ARTIFACT_BYTES:
                digest = hashlib.sha256(data).hexdigest()
                object_key = f"ingestion-cache/v1/extraction/{key}/{digest}.json.gz"
                self.storage.upload_bytes(gzip.compress(data, mtime=0), object_key, "application/gzip")
                self.repository.put_many("extraction", {key: {"sha256": digest}})
        except Exception:
            log.warning("extraction_cache_checkpoint_unavailable")
        return document

    def ocr(self, page_hash, engine_identity, compute):
        if engine_identity is None:
            return compute()
        key = identity({"kind": "ocr-v1", "page": page_hash, "engine": engine_identity})
        value = self.repository.get_many("ocr", [key]).get(key)
        if isinstance(value, dict) and isinstance(value.get("text"), str):
            self._increment("ocr_cache_hits")
            return value["text"]
        self._increment("ocr_cache_misses")
        text = compute()  # exceptions are not cached as empty successful OCR
        self.repository.put_many("ocr", {key: {"text": text}})
        return text

    def embeddings(self, texts, provider, provider_identity, dimensions, batch_size, progress, workers=1, provider_factory=None):
        if workers > 1 and provider_factory is None:
            raise ValueError("Concurrent embeddings require an isolated provider factory")
        local = threading.local()
        def valid(vector):
            return isinstance(vector, list) and len(vector) == dimensions and all(type(n) in (int, float) and math.isfinite(n) for n in vector)
        keys = [identity({"input": text, "provider": provider_identity}) for text in texts]
        unique = dict(zip(keys, texts))
        cached = self.repository.get_many("embedding", list(unique)) if provider_identity else {}
        vectors = {key: value["vector"] for key, value in cached.items() if isinstance(value, dict) and valid(value.get("vector"))}
        self.metrics["embedding_cache_hits"] = sum(key in vectors for key in keys)
        missing = [key for key in unique if key not in vectors]
        self.metrics["embedding_unique_misses"] = len(missing)
        self.metrics["embedding_cache_enabled"] = provider_identity is not None
        def batches():
            for offset in range(0, len(missing), max(1, batch_size)):
                progress(offset, len(missing))  # cancellation before bounded submission
                yield missing[offset:offset + max(1, batch_size)]

        def compute(batch):
            if workers > 1:
                if not hasattr(local, "provider"):
                    local.provider = provider_factory()
                active = local.provider
            else:
                active = provider
            self._increment("embedding_provider_calls")
            result = active.embed([unique[key] for key in batch])
            if len(result) != len(batch) or not all(valid(vector) for vector in result):
                raise ValueError("Embedding provider returned invalid vectors or count")
            completed = dict(zip(batch, result))
            # A successful batch is durable before starting the next one. Do not
            # cache a provider's dynamic shorter-input fallback as an exact input.
            if provider_identity and getattr(active, "last_batch_cacheable", True):
                self.repository.put_many("embedding", {key: {"vector": vector} for key, vector in completed.items()})
            return completed

        for completed in ordered_work(compute, batches(), workers):
            vectors.update(completed)
        progress(len(missing), len(missing))
        return [list(vectors[key]) for key in keys]
