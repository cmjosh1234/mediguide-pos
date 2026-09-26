from dataclasses import asdict
import threading
import time

import fitz
import pytest

from app.core.config import Settings
from app.document_processing.bounded_work import ordered_work
from app.document_processing.pdf_extractor import extract_pdf
from app.services.ingestion_artifacts import IngestionArtifacts
from test_ingestion_artifacts import Repository, Storage, Provider


def test_bounded_work_preserves_order_and_does_not_eagerly_consume():
    consumed = []
    def inputs():
        for number in range(20):
            consumed.append(number)
            yield number
    def compute(number):
        time.sleep(0.002 if number % 2 else 0.01)
        return number
    results = ordered_work(compute, inputs(), workers=2)
    assert next(results) == 0
    assert consumed == [0, 1]
    assert [0, *results] == list(range(20))


def test_worker_limits_are_validated():
    for field in ("PDF_PAGE_WORKERS", "OCR_WORKERS", "EMBEDDING_WORKERS"):
        with pytest.raises(ValueError):
            Settings(**{field: 50})
    assert Settings().ingestion_processing_concurrency
    assert Settings().embedding_workers == 2
    assert Settings().worker_priority_aging_seconds == 900


def test_pdf_parallel_content_equivalence(tmp_path):
    source = tmp_path / "source.pdf"
    with fitz.open() as pdf:
        for number in range(4):
            page = pdf.new_page()
            page.insert_text((50, 50), f"Chapter {number + 1}: Testing extraction")
            page.insert_text((50, 90), "This guideline contains enough ordinary words to exercise text extraction without losing any source information.")
            for x in (50, 250, 450):
                page.draw_line((x, 140), (x, 230))
            for y in (140, 170, 200, 230):
                page.draw_line((50, y), (450, y))
            page.insert_text((60, 160), "Name")
            page.insert_text((260, 160), "Value")
            page.insert_text((60, 190), "Example")
            page.insert_text((260, 190), str(number))
        pdf.save(source)
    serial = asdict(extract_pdf(source))
    parallel = asdict(extract_pdf(source, page_workers=2, ocr_workers=2))
    for result in (serial, parallel):
        result["metadata"].pop("extraction_timings", None)
    assert serial == parallel


def test_parallel_embeddings_checkpoint_successful_out_of_order_batches():
    repo = Repository()
    cache = IngestionArtifacts(repo, Storage(), {})
    completed = threading.Event()
    class FailingProvider(Provider):
        def embed(self, texts):
            if texts == ["first"]:
                assert completed.wait(2)
                raise RuntimeError("first batch failed")
            completed.set()
            return super().embed(texts)
    with pytest.raises(RuntimeError):
        cache.embeddings(["first", "second"], Provider(), {"revision": "one"}, 2, 1,
            lambda *_: None, workers=2, provider_factory=FailingProvider)
    retry = Provider()
    vectors = cache.embeddings(["first", "second"], retry, {"revision": "one"}, 2, 1, lambda *_: None)
    assert retry.calls == [["first"]]
    assert vectors == [[5.0, 1.0], [6.0, 1.0]]


def test_parallel_ocr_is_bounded_and_pymupdf_stays_on_caller_thread(tmp_path, monkeypatch):
    import app.document_processing.pdf_extractor as module
    source = tmp_path / "scanned.pdf"
    with fitz.open() as pdf:
        for _ in range(5):
            pdf.new_page()
        pdf.save(source)
    active = 0
    peak = 0
    lock = threading.Lock()
    def recognize(image, *args):
        nonlocal active, peak
        assert isinstance(image, bytes)
        with lock:
            active += 1
            peak = max(peak, active)
        # Outlast a 300 DPI page render so concurrent recognitions overlap.
        time.sleep(0.5)
        with lock:
            active -= 1
        return "OCR result"
    monkeypatch.setattr(module, "_ocr_image_text", recognize)
    result = extract_pdf(source, ocr_workers=2)
    assert result.ocr_pages == [1, 2, 3, 4, 5]
    assert peak == 2
