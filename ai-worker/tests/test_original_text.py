from __future__ import annotations

import zipfile
from pathlib import Path

import fitz
import pytest

from app.document_processing import original_text
from app.document_processing.original_text import (
    chunk_page_texts,
    extract_docx_text,
    extract_pdf_page_texts,
)
from app.services.ingestion_service import (
    ORIGINAL_TEXT_INDEX_JOB,
    IngestionService,
    IngestionSuperseded,
)

W = 'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"'


def write_docx(path: Path, body: str) -> Path:
    with zipfile.ZipFile(path, "w") as archive:
        archive.writestr("word/document.xml", f'<?xml version="1.0"?><w:document {W}><w:body>{body}</w:body></w:document>')
    return path


def test_extract_docx_text_reads_paragraphs_table_cells_tabs_and_breaks(tmp_path: Path):
    path = write_docx(
        tmp_path / "form.docx",
        "<w:p><w:r><w:t>HMIS 105</w:t><w:tab/><w:t>Outpatient</w:t></w:r></w:p>"
        "<w:p/>"
        "<w:tbl><w:tr><w:tc><w:p><w:r><w:t>Patient name</w:t></w:r></w:p></w:tc>"
        "<w:tc><w:p><w:r><w:t>Age</w:t><w:br/><w:t>(years)</w:t></w:r></w:p></w:tc></w:tr></w:tbl>",
    )
    assert extract_docx_text(path) == "HMIS 105 Outpatient\nPatient name\nAge\n(years)"


def test_extract_docx_text_rejects_dtds_and_missing_bodies(tmp_path: Path):
    hostile = tmp_path / "hostile.docx"
    with zipfile.ZipFile(hostile, "w") as archive:
        archive.writestr("word/document.xml", '<?xml version="1.0"?><!DOCTYPE x [<!ENTITY a "aaaa">]><x>&a;</x>')
    with pytest.raises(ValueError, match="DTD"):
        extract_docx_text(hostile)

    empty = tmp_path / "empty.docx"
    with zipfile.ZipFile(empty, "w") as archive:
        archive.writestr("word/styles.xml", "<styles/>")
    with pytest.raises(ValueError, match="no body"):
        extract_docx_text(empty)


def test_chunk_page_texts_tracks_page_ranges_with_overlap():
    pages = [(1, "a b c d"), (2, "e f g h"), (3, "i j")]
    chunks = chunk_page_texts(pages, title="Register", chunk_size=4, overlap=1)
    assert [chunk.content for chunk in chunks] == ["a b c d", "d e f g", "g h i j"]
    assert [(chunk.page_start, chunk.page_end) for chunk in chunks] == [(1, 1), (1, 2), (2, 3)]
    assert [chunk.chunk_order for chunk in chunks] == [0, 1, 2]
    assert all(chunk.title == "Register" and chunk.block_order is None for chunk in chunks)
    assert chunks[0].html == "<p>a b c d</p>"


def test_chunk_page_texts_keeps_short_forms_and_skips_empty_documents():
    short = chunk_page_texts([(None, "Name: ____ Date: ____")], title="", chunk_size=320, overlap=60)
    assert len(short) == 1
    assert short[0].content == "Name: ____ Date: ____"
    assert (short[0].page_start, short[0].page_end) == (None, None)
    assert short[0].title is None
    assert chunk_page_texts([(1, "   "), (2, "")], title="x", chunk_size=10, overlap=2) == []


def test_extract_pdf_page_texts_uses_ocr_for_pages_without_text(monkeypatch: pytest.MonkeyPatch, tmp_path: Path):
    path = tmp_path / "form.pdf"
    document = fitz.open()
    page = document.new_page()
    page.insert_text((72, 72), "Outpatient register " * 5)
    document.new_page()  # a blank, scanned-looking page
    document.save(path)
    document.close()
    monkeypatch.setattr(original_text, "_ocr_page_text", lambda page: "Signature of health worker")

    pages = extract_pdf_page_texts(path)

    assert [number for number, _ in pages] == [1, 2]
    assert "Outpatient register" in pages[0][1]
    assert pages[1][1] == "Signature of health worker"


class FakeStorage:
    def __init__(self, content: bytes):
        self.content = content
        self.downloaded: list[str] = []

    def download_file(self, key: str, dest: Path) -> None:
        self.downloaded.append(key)
        dest.write_bytes(self.content)


class FakeEmbedder:
    def embed(self, texts: list[str]) -> list[list[float]]:
        return [[float(len(text))] for text in texts]


class FakeGuidelines:
    def __init__(self, version: dict):
        self.version = version
        self.saved: dict | None = None

    def get_version_with_document(self, version_id: str) -> dict:
        return self.version

    def replace_original_text_chunks(self, **kwargs) -> None:
        self.saved = kwargs

    def replace_extraction(self, *args, **kwargs):  # pragma: no cover - must not be called
        raise AssertionError("forms must not create sections, blocks or Markdown")


class FakeJobs:
    def __init__(self):
        self.stages: list[str] = []

    def cancellation_requested(self, job_id: str) -> bool:
        return False

    def set_progress(self, job_id: str, stage: str, percent: int) -> None:
        self.stages.append(stage)


class FakeSettings:
    chunk_size = 320
    chunk_overlap = 60
    embedding_request_batch_size = 8


def original_text_service(tmp_path: Path, version: dict) -> tuple[IngestionService, FakeGuidelines, FakeJobs]:
    service = IngestionService.__new__(IngestionService)
    service.settings = FakeSettings()
    service.storage = FakeStorage(write_docx(tmp_path / "upload.docx", "<w:p><w:r><w:t>Patient name and village</w:t></w:r></w:p>").read_bytes())
    service.embedder = FakeEmbedder()
    service.guidelines = FakeGuidelines(version)
    service.jobs = FakeJobs()
    return service, service.guidelines, service.jobs


def test_original_text_job_indexes_the_uploaded_file_only(tmp_path: Path):
    key = "guidelines/v1/original/1_hmis.docx"
    version = {"id": "v1", "document_id": "d1", "original_file_key": key, "document_title": "HMIS 105"}
    service, guidelines, jobs = original_text_service(tmp_path, version)
    job = {
        "id": "j1",
        "version_id": "v1",
        "job_type": ORIGINAL_TEXT_INDEX_JOB,
        "payload_json": {"file_key": key, "source_format": "docx"},
    }

    service._process(job)

    assert service.storage.downloaded == [key]
    assert guidelines.saved is not None
    assert guidelines.saved["source_key"] == key
    assert [chunk.content for chunk in guidelines.saved["chunks"]] == ["Patient name and village"]
    assert guidelines.saved["embeddings"] == [[24.0]]
    assert jobs.stages[0] == "downloading" and jobs.stages[-1] == "saving"


def test_original_text_job_is_superseded_by_a_newer_upload(tmp_path: Path):
    version = {"id": "v1", "document_id": "d1", "original_file_key": "guidelines/v1/original/2_new.pdf"}
    service, guidelines, _ = original_text_service(tmp_path, version)
    job = {
        "id": "j1",
        "version_id": "v1",
        "job_type": ORIGINAL_TEXT_INDEX_JOB,
        "payload_json": {"file_key": "guidelines/v1/original/1_old.pdf", "source_format": "pdf"},
    }

    with pytest.raises(IngestionSuperseded):
        service._process(job)
    assert service.storage.downloaded == []
    assert guidelines.saved is None
