from __future__ import annotations
from pathlib import Path
import hashlib
import json
import mimetypes
import tempfile
import time
from datetime import datetime, timezone
import structlog
from app.core.config import get_settings
from app.core.storage import ObjectStorage
from app.document_processing.pdf_extractor import extract_pdf
from app.document_processing.markdown_extractor import extract_markdown
from app.document_processing.chunker import chunk_blocks, chunk_sections
from app.document_processing.original_text import (
    chunk_page_texts,
    extract_docx_text,
    extract_pdf_page_texts,
)
from app.document_processing.types import ExtractedAsset
from app.embeddings.factory import get_embedding_provider
from app.repositories.guideline_repo import (
    GuidelineRepository,
    GuidelineSourceSupersededError,
)
from app.repositories.ingestion_repo import IngestionRepository
from app.repositories.artifact_cache_repo import ArtifactCacheRepository
from app.services.ingestion_artifacts import IngestionArtifacts
from app.document_processing.artifact_identity import extraction_identity, embedding_identity, ocr_identity, identity

log = structlog.get_logger()


class IngestionCanceled(Exception):
    pass


class IngestionSuperseded(Exception):
    pass


# Documents published as uploaded (for example forms) only get their original
# file's text indexed for search and RAG.
ORIGINAL_TEXT_INDEX_JOB = "original_text_index"


class IngestionService:
    def __init__(self):
        self.settings = get_settings()
        self.jobs = IngestionRepository()
        self.guidelines = GuidelineRepository()
        self.storage = ObjectStorage()
        self.embedder = get_embedding_provider()
        self.artifact_repository = ArtifactCacheRepository()

    def _source_is_current(
        self,
        *,
        version_id: str,
        version: dict | None,
        source_format: str,
        source_key: str,
        revision_id: str | None,
    ) -> bool:
        if not version:
            return False
        if source_format == "markdown":
            return self.guidelines.is_current_markdown_source(
                version_id, revision_id, source_key
            )
        return str(version.get("original_file_key") or "").strip() == source_key

    def run_job(self, job_id: str) -> None:
        job = self.jobs.get_job(job_id)
        if not job:
            raise ValueError(f"Ingestion job not found: {job_id}")
        if job.get("status") in {"completed", "canceled"}:
            return
        if job.get("status") == "cancel_requested":
            self.jobs.mark_canceled(job_id)
            return
        # For API-triggered runs the job may not be in 'running' state yet.
        self.jobs.mark_running(job_id)
        self._metrics = {"stage_seconds": {}}
        created = job.get("created_at")
        if isinstance(created, datetime):
            self._metrics["queue_seconds"] = round(max(0, (datetime.now(timezone.utc) - created.replace(tzinfo=created.tzinfo or timezone.utc)).total_seconds()), 3)
        self._stage_name = None
        self._stage_started = time.perf_counter()
        job_started = self._stage_started
        try:
            self._process(job)
            self.jobs.mark_completed(job_id)
        except IngestionCanceled:
            self.jobs.mark_canceled(job_id)
            log.info("ingestion_job_canceled", job_id=job_id)
        except (IngestionSuperseded, GuidelineSourceSupersededError):
            self.jobs.mark_superseded(job_id)
            log.info("ingestion_job_superseded", job_id=job_id)
        except Exception as exc:
            log.exception("ingestion_job_failed", job_id=job_id, error=str(exc))
            self.jobs.mark_failed(job_id, str(exc))
            raise
        finally:
            self._finish_stage()
            self._metrics["processing_seconds"] = round(time.perf_counter() - job_started, 3)
            log.info("ingestion_benchmark", job_id=job_id, **self._metrics)
            try:
                self.jobs.record_metrics(job_id, self._metrics)
            except Exception:
                log.exception("ingestion_metrics_save_failed", job_id=job_id)

    def _finish_stage(self):
        if getattr(self, "_stage_name", None):
            stages = self._metrics["stage_seconds"]
            stages[self._stage_name] = round(stages.get(self._stage_name, 0) + time.perf_counter() - self._stage_started, 3)
        self._stage_started = time.perf_counter()

    def _process(self, job: dict) -> None:
        job_id = str(job["id"])

        def stage(name: str, percent: int) -> None:
            if hasattr(self, "_metrics") and name != self._stage_name:
                self._finish_stage()
                self._stage_name = name
            if self.jobs.cancellation_requested(job_id):
                raise IngestionCanceled()
            self.jobs.set_progress(job_id, name, percent)

        version_id = str(job["version_id"])
        version = self.guidelines.get_version_with_document(version_id)
        if not version:
            raise ValueError(f"Guideline version not found: {version_id}")
        payload = self._job_payload(job)
        if job.get("job_type") == ORIGINAL_TEXT_INDEX_JOB:
            self._index_original_text(job, version, payload, stage)
            return
        source_format = str(payload.get("source_format") or "").strip().lower()
        if not source_format:
            source_format = "markdown" if job.get("job_type") == "markdown_ingestion" else "pdf"
        if source_format not in {"pdf", "markdown"}:
            raise ValueError(f"Unsupported guideline source format: {source_format}")
        revision_id = str(payload.get("revision_id") or "").strip() or None
        source_field = "original_file_key"
        source_key = str(payload.get("file_key") or version.get(source_field) or "").strip()
        if not source_key:
            raise ValueError(f"Guideline version has no {source_format} source file")
        source_is_current = self._source_is_current(
            version_id=version_id,
            version=version,
            source_format=source_format,
            source_key=source_key,
            revision_id=revision_id,
        )
        if not source_is_current:
            log.info(
                "ingestion_skipped_superseded_source",
                job_id=str(job["id"]),
                version_id=version_id,
                source_format=source_format,
            )
            raise IngestionSuperseded()

        with tempfile.TemporaryDirectory(prefix="mediguide-ingest-") as tmp:
            tmp_path = Path(tmp)
            source_path = tmp_path / ("source.md" if source_format == "markdown" else "source.pdf")

            stage("downloading", 5)
            started = time.perf_counter()
            self.storage.download_file(source_key, source_path)
            document_checksum = self._file_checksum(source_path)
            if hasattr(self, "_metrics"):
                self._metrics.update(source_bytes=source_path.stat().st_size, source_format=source_format, checksum=document_checksum)
            log.info(
                "ingestion_download_completed",
                job_id=str(job["id"]),
                version_id=version_id,
                seconds=round(time.perf_counter() - started, 2),
                file_key=source_key,
                source_format=source_format,
                checksum=document_checksum,
            )

            reuse = getattr(self.settings, "ingestion_artifact_reuse", False)
            artifacts = IngestionArtifacts(self.artifact_repository, self.storage, getattr(self, "_metrics", {})) if reuse else None
            extraction_key = extraction_identity(document_checksum, source_format, self.settings, version.get("document_title") or "Guideline") if reuse else None
            embedding_key = embedding_identity(self.settings, self.embedder) if reuse else None
            processing_identity = identity({"extraction": extraction_key, "embedding": embedding_key, "chunking": [self.settings.chunk_size, self.settings.chunk_overlap, self.settings.min_chunk_chars]}) if extraction_key and embedding_key else None
            previous_metadata = version.get("extraction_metadata_json") or {}
            if isinstance(previous_metadata, str):
                previous_metadata = json.loads(previous_metadata)
            same_pipeline = not reuse or (processing_identity is not None and previous_metadata.get("processing_identity") == processing_identity and previous_metadata.get("source_file_key") == source_key and previous_metadata.get("markdown_revision_id") == revision_id)
            if same_pipeline and self.guidelines.is_extraction_current(version_id, document_checksum):
                if hasattr(self, "_metrics"):
                    self._metrics["checksum_noop"] = True
                log.info(
                    "ingestion_skipped_current_checksum",
                    job_id=str(job["id"]),
                    version_id=version_id,
                    checksum=document_checksum,
                    extraction_schema_version=self.guidelines.EXTRACTION_SCHEMA_VERSION,
                )
                self.jobs.complete_noop_comparison(job_id)
                return

            stage("parsing", 20)
            started = time.perf_counter()
            last_parsing_check = 0.0
            def check_parsing_cancel():
                nonlocal last_parsing_check
                now = time.perf_counter()
                if now - last_parsing_check >= 1:
                    stage("parsing", 20)
                    last_parsing_check = now
            def compute_extraction():
                if source_format == "markdown":
                    return extract_markdown(source_path, fallback_title=version.get("document_title") or "Guideline")
                concurrent = getattr(self.settings, "ingestion_processing_concurrency", False)
                parallel_options = {"page_workers": self.settings.pdf_page_workers, "ocr_workers": self.settings.ocr_workers,
                    "check_cancel": check_parsing_cancel} if concurrent else {}
                if artifacts is not None:
                    ocr = ocr_identity()
                    return extract_pdf(source_path, artifacts=artifacts, ocr_settings={"epoch": self.settings.artifact_cache_epoch, **ocr} if ocr else None, **parallel_options)
                return extract_pdf(source_path, **parallel_options)
            # Save raw extraction BEFORE enriching with revision/job IDs,
            # version-owned storage keys or the original PDF attachment.
            extracted = artifacts.extraction(extraction_key, compute_extraction) if artifacts else compute_extraction()
            if artifacts is not None and not artifacts.extraction_complete:
                processing_identity = None
            for block in extracted.blocks:
                block.provenance = {
                    **block.provenance,
                    "markdown_revision_id": revision_id,
                    "ingestion_job_id": job_id,
                }
            if hasattr(self, "_metrics"):
                self._metrics.update(pages=extracted.pages, blocks=len(extracted.blocks), sections=len(extracted.sections), tables=len(extracted.tables), assets=len(extracted.assets), extraction=extracted.metadata.get("extraction_timings", {}))
            markdown_bytes = extracted.markdown.encode("utf-8")
            markdown_checksum = hashlib.sha256(markdown_bytes).hexdigest()
            log.info(
                "ingestion_extract_completed",
                job_id=str(job["id"]),
                version_id=version_id,
                seconds=round(time.perf_counter() - started, 2),
                pages=extracted.pages,
                source_format=source_format,
                sections=len(extracted.sections),
                tables=len(extracted.tables),
            )

            stage("building_structure", 40)
            started = time.perf_counter()
            chunks = chunk_blocks(extracted.blocks)
            if not chunks:
                chunks = chunk_sections(extracted.sections)
            if hasattr(self, "_metrics"):
                self._metrics["chunks"] = len(chunks)
            log.info(
                "ingestion_chunking_completed",
                job_id=str(job["id"]),
                version_id=version_id,
                seconds=round(time.perf_counter() - started, 2),
                chunks=len(chunks),
            )

            schema_version = self.guidelines.EXTRACTION_SCHEMA_VERSION
            html_key = (
                f"guidelines/{version_id}/extracted/{document_checksum}.v{schema_version}.html"
            )
            markdown_key = (
                f"guidelines/{version_id}/extracted/{document_checksum}.v{schema_version}.md"
            )

            started = time.perf_counter()
            stage("uploading_assets", 50)
            self.storage.upload_bytes(
                extracted.html.encode("utf-8"), html_key, "text/html; charset=utf-8"
            )
            self.storage.upload_bytes(markdown_bytes, markdown_key, "text/markdown; charset=utf-8")
            uploaded_asset_keys: set[str] = set()
            for asset in extracted.assets:
                if not asset.data:
                    continue
                extension = self._extension_for_asset(asset)
                asset.storage_key = f"guidelines/{version_id}/assets/{asset.checksum}.{extension}"
                if asset.storage_key not in uploaded_asset_keys:
                    self.storage.upload_bytes(asset.data, asset.storage_key, asset.mime_type)
                    uploaded_asset_keys.add(asset.storage_key)
                asset.data = None

            if source_format == "pdf":
                original_asset = ExtractedAsset(
                    type="original_pdf",
                    source_key="original-pdf",
                    source_fingerprint=document_checksum,
                    mime_type="application/pdf",
                    checksum=document_checksum,
                    size_bytes=source_path.stat().st_size,
                    storage_key=source_key,
                    original_filename=Path(source_key).name,
                    page_start=1,
                    page_end=extracted.pages,
                    provenance={
                        "source": "uploaded_original",
                        "immutable": True,
                        "review_required": False,
                    },
                )
                extracted.assets.insert(0, original_asset)
            log.info(
                "ingestion_asset_upload_completed",
                job_id=str(job["id"]),
                version_id=version_id,
                seconds=round(time.perf_counter() - started, 2),
                extracted_assets=len(extracted.assets),
                unique_stored_assets=len(uploaded_asset_keys)
                + (1 if source_format == "pdf" else 0),
            )

            stage("chunking", 55)
            texts = [c.content for c in chunks]
            embeddings = []
            batch_size = max(1, self.settings.embedding_request_batch_size)
            started = time.perf_counter()
            stage("embeddings", 65)
            concurrent = getattr(self.settings, "ingestion_processing_concurrency", False)
            embedding_artifacts = artifacts or (IngestionArtifacts(self.artifact_repository, self.storage, getattr(self, "_metrics", {})) if concurrent else None)
            if embedding_artifacts:
                workers = self.settings.embedding_workers if concurrent else 1
                # Never multiply in-process transformer model weights/GPU memory.
                if self.settings.embedding_provider.lower() == "sentence_transformers":
                    workers = 1
                embeddings = embedding_artifacts.embeddings(texts, self.embedder, embedding_key, self.settings.embedding_dim, batch_size,
                    lambda done, total: stage("embeddings", min(84, 65 + int((done / max(1, total)) * 19))),
                    workers=workers, provider_factory=get_embedding_provider if workers > 1 else None)
            for i in range(0, len(texts) if not embedding_artifacts else 0, batch_size):
                stage("embeddings", min(84, 65 + int((i / max(1, len(texts))) * 19)))
                batch_started = time.perf_counter()
                batch = texts[i : i + batch_size]
                embeddings.extend(self.embedder.embed(batch))
                log.info(
                    "ingestion_embedding_batch_completed",
                    job_id=str(job["id"]),
                    version_id=version_id,
                    batch_index=(i // batch_size) + 1,
                    batch_count=((len(texts) + batch_size - 1) // batch_size) if texts else 0,
                    batch_size=len(batch),
                    seconds=round(time.perf_counter() - batch_started, 2),
                )
            log.info(
                "ingestion_embedding_completed",
                job_id=str(job["id"]),
                version_id=version_id,
                seconds=round(time.perf_counter() - started, 2),
                chunks=len(chunks),
            )

            # Cancellation is deliberately no longer accepted after persistence
            # starts: replacement is one database transaction.
            stage("persisting", 90)
            started = time.perf_counter()
            latest = self.guidelines.get_version_with_document(version_id)
            source_is_current = self._source_is_current(
                version_id=version_id,
                version=latest,
                source_format=source_format,
                source_key=source_key,
                revision_id=revision_id,
            )
            if not source_is_current:
                log.info(
                    "ingestion_skipped_superseded_source",
                    job_id=str(job["id"]),
                    version_id=version_id,
                    source_format=source_format,
                )
                raise IngestionSuperseded()
            self.guidelines.replace_extraction(
                version_id=version_id,
                version=version,
                sections=extracted.sections,
                tables=extracted.tables,
                chunks=chunks,
                embeddings=embeddings,
                html_key=html_key,
                markdown_key=markdown_key,
                blocks=extracted.blocks,
                assets=extracted.assets,
                checksum=document_checksum,
                metadata={
                    **extracted.metadata,
                    "processing_identity": processing_identity,
                    "source_format": source_format,
                    "source_file_key": source_key,
                    "markdown_checksum": markdown_checksum,
                    "markdown_size_bytes": len(markdown_bytes),
                    "toc_entries": extracted.toc_entries,
                    "structured_block_count": len(extracted.blocks),
                    "asset_count": len(extracted.assets),
                    "unique_asset_count": len(uploaded_asset_keys)
                    + (1 if source_format == "pdf" else 0),
                    "markdown_revision_id": revision_id,
                    "ingestion_job_id": job_id,
                },
                warnings=extracted.warnings,
                markdown_revision_id=revision_id,
                ingestion_job_id=str(job["id"]),
            )
            self.jobs.set_progress(job_id, "review_required", 98)
            log.info(
                "ingestion_persist_completed",
                job_id=str(job["id"]),
                version_id=version_id,
                seconds=round(time.perf_counter() - started, 2),
                sections=len(extracted.sections),
                chunks=len(chunks),
                tables=len(extracted.tables),
                blocks=len(extracted.blocks),
                assets=len(extracted.assets),
                source_format=source_format,
            )
            log.info(
                "ingestion_job_completed",
                job_id=str(job["id"]),
                version_id=version_id,
                sections=len(extracted.sections),
                chunks=len(chunks),
                tables=len(extracted.tables),
                blocks=len(extracted.blocks),
                assets=len(extracted.assets),
                source_format=source_format,
            )

    def _index_original_text(self, job: dict, version: dict, payload: dict, stage) -> None:
        """Index the text of a published-as-uploaded file without creating editable content."""
        job_id = str(job["id"])
        version_id = str(job["version_id"])
        source_format = str(payload.get("source_format") or "pdf").strip().lower()
        if source_format not in {"pdf", "docx"}:
            raise ValueError(f"Unsupported original file format: {source_format}")
        source_key = str(payload.get("file_key") or version.get("original_file_key") or "").strip()
        if not source_key:
            raise ValueError("Guideline version has no original file")
        if str(version.get("original_file_key") or "").strip() != source_key:
            raise IngestionSuperseded()

        with tempfile.TemporaryDirectory(prefix="mediguide-original-") as tmp:
            source_path = Path(tmp) / f"source.{source_format}"
            stage("downloading", 5)
            self.storage.download_file(source_key, source_path)
            stage("extracting", 25)
            if source_format == "pdf":
                pages = extract_pdf_page_texts(source_path)
            else:
                pages = [(None, extract_docx_text(source_path))]

        stage("chunking", 55)
        chunks = chunk_page_texts(
            pages,
            title=str(version.get("document_title") or ""),
            chunk_size=self.settings.chunk_size,
            overlap=self.settings.chunk_overlap,
        )
        texts = [chunk.content for chunk in chunks]
        embeddings: list[list[float]] = []
        batch_size = max(1, self.settings.embedding_request_batch_size)
        for start in range(0, len(texts), batch_size):
            stage("embeddings", min(84, 65 + int((start / max(1, len(texts))) * 19)))
            embeddings.extend(self.embedder.embed(texts[start : start + batch_size]))

        stage("saving", 90)
        # The repository re-checks the original under a row lock before writing.
        self.guidelines.replace_original_text_chunks(
            version=version, source_key=source_key, chunks=chunks, embeddings=embeddings
        )
        log.info(
            "original_text_indexed",
            job_id=job_id,
            version_id=version_id,
            source_format=source_format,
            pages=len(pages),
            chunks=len(chunks),
        )

    @staticmethod
    def _job_payload(job: dict) -> dict:
        value = job.get("payload_json") or {}
        if isinstance(value, dict):
            return value
        try:
            parsed = json.loads(str(value))
        except (TypeError, ValueError) as exc:
            raise ValueError("Ingestion job payload is invalid JSON") from exc
        if not isinstance(parsed, dict):
            raise ValueError("Ingestion job payload must be an object")
        return parsed

    @staticmethod
    def _file_checksum(path: Path) -> str:
        digest = hashlib.sha256()
        with path.open("rb") as source:
            for chunk in iter(lambda: source.read(1024 * 1024), b""):
                digest.update(chunk)
        return digest.hexdigest()

    @staticmethod
    def _extension_for_asset(asset: ExtractedAsset) -> str:
        if asset.original_filename and "." in asset.original_filename:
            return asset.original_filename.rsplit(".", 1)[-1].lower()
        guessed = mimetypes.guess_extension(asset.mime_type) or ".bin"
        return guessed.lstrip(".")
