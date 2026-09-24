from pathlib import Path

import fitz
import pytest

from app.document_processing.chunker import chunk_blocks
from app.document_processing.pdf_extractor import (
    _page_text_from_blocks,
    _remove_repeated_margin_lines,
    extract_pdf,
)
from app.document_processing.structured_blocks import build_structured_blocks
from app.document_processing.types import ExtractedSection, ExtractedTable
from app.services.ingestion_service import IngestionService


def test_structured_blocks_preserve_lists_callouts_tables_and_provenance():
    section = ExtractedSection(
        title="3. Diagnosis and Assessment",
        level=2,
        page_start=4,
        page_end=5,
        sort_order=0,
        text="\n".join(
            [
                "Clinical assessment must retain 2.4 mg/kg and SpO₂ ≥ 94%.",
                "• Check airway",
                "• Check breathing",
                "Recommendation: Refer patients with danger signs.",
                "Warning: Do not alter the prescribed dose.",
            ]
        ),
    )
    table = ExtractedTable(
        title="Performance of RDT",
        page=5,
        html="<table></table>",
        data=[["Test", "Sensitivity"], ["HRP2-based", "95–98%"]],
    )

    blocks = build_structured_blocks(
        [section],
        [table],
        [],
        page_methods={4: "embedded_text", 5: "embedded_text"},
    )

    assert [block.type for block in blocks] == [
        "heading",
        "paragraph",
        "unordered_list",
        "recommendation",
        "warning",
        "table",
    ]
    paragraph = next(block for block in blocks if block.type == "paragraph")
    assert "2.4 mg/kg" in paragraph.content["text"]
    assert "SpO₂ ≥ 94%" in paragraph.content["text"]
    assert paragraph.page_start == 4
    assert paragraph.provenance["review_required"] is True
    assert all(len(block.source_fingerprint) == 64 for block in blocks)
    assert len({block.source_fingerprint for block in blocks}) == len(blocks)
    table_block = blocks[-1]
    assert table_block.content["columns"] == ["Test", "Sensitivity"]
    assert table_block.content["rows"] == [["HRP2-based", "95–98%"]]


def test_block_chunks_keep_block_and_page_association():
    section = ExtractedSection(
        title="Assessment",
        level=1,
        page_start=7,
        page_end=7,
        sort_order=3,
        text="A sufficiently detailed clinical assessment paragraph for retrieval and citation.",
    )
    blocks = build_structured_blocks([section], [], [])

    chunks = chunk_blocks(blocks)

    assert chunks
    assert all(chunk.block_order is not None for chunk in chunks)
    assert all(chunk.page_start == 7 and chunk.page_end == 7 for chunk in chunks)


def test_repeated_margin_lines_are_removed_without_changing_body():
    pages = [
        (1, "Ministry Clinical Guideline\nPage one clinical body\n1"),
        (2, "Ministry Clinical Guideline\nPage two clinical body\n2"),
        (3, "Ministry Clinical Guideline\nPage three clinical body\n3"),
    ]

    cleaned = _remove_repeated_margin_lines(pages)

    assert all("Ministry Clinical Guideline" not in text for _, text in cleaned)
    assert "Page two clinical body" in cleaned[1][1]


def test_multi_column_reading_order_is_left_column_then_right_column():
    blocks = [
        (
            320.0,
            40.0,
            580.0,
            100.0,
            "Right first paragraph with enough clinical detail to count.",
            0,
            0,
        ),
        (
            20.0,
            200.0,
            280.0,
            260.0,
            "Left second paragraph with enough clinical detail to count.",
            1,
            0,
        ),
        (
            20.0,
            40.0,
            280.0,
            100.0,
            "Left first paragraph with enough clinical detail to count.",
            2,
            0,
        ),
        (
            320.0,
            200.0,
            580.0,
            260.0,
            "Right second paragraph with enough clinical detail to count.",
            3,
            0,
        ),
    ]

    text = _page_text_from_blocks(blocks, [], 600.0)

    assert text.index("Left first") < text.index("Left second") < text.index("Right first")


def test_extract_pdf_uses_ocr_for_scanned_page(monkeypatch: pytest.MonkeyPatch, tmp_path: Path):
    path = tmp_path / "scanned.pdf"
    document = fitz.open()
    document.new_page()
    document.save(path)
    document.close()
    monkeypatch.setattr(
        "app.document_processing.pdf_extractor._ocr_page_text",
        lambda _page: "1 EMERGENCIES AND TRAUMA\n1.1 Triage Assessment\nAssess airway breathing circulation and danger signs before urgent referral.",
    )

    extracted = extract_pdf(path)

    assert extracted.ocr_pages == [1]
    assert extracted.metadata["text_mode"] == "ocr_required"
    assert extracted.blocks
    assert any("OCR-derived text" in warning for warning in extracted.warnings)


def test_extract_pdf_rejects_malformed_pdf(tmp_path: Path):
    path = tmp_path / "malformed.pdf"
    path.write_bytes(b"this is not a pdf")

    with pytest.raises(Exception):
        extract_pdf(path)


def test_document_checksum_and_asset_extension_are_deterministic(tmp_path: Path):
    path = tmp_path / "source.pdf"
    path.write_bytes(b"clinical source")

    first = IngestionService._file_checksum(path)
    second = IngestionService._file_checksum(path)

    assert first == second
    assert len(first) == 64


def test_extract_pdf_font_styles_reject_numbered_dosing_lines(tmp_path: Path):
    path = tmp_path / "styled.pdf"
    with fitz.open() as document:
        page = document.new_page()
        page.insert_text((50, 60), "1.3.1 General Management", fontname="hebo", fontsize=11)
        page.insert_text((50, 90), "Give diazepam rectally, repeated if necessary, for convulsions.", fontsize=10)
        page.insert_text((50, 110), "0.5 mg/kg per dose for children under two years", fontsize=10)
        page.insert_text((50, 140), "1.3.2 Organophosphate Poisoning", fontname="hebo", fontsize=11)
        page.insert_text((50, 170), "Give atropine and provide supportive care until stable.", fontsize=10)
        document.save(path)

    extracted = extract_pdf(path)

    assert [section.title for section in extracted.sections] == [
        "1.3.1 General Management",
        "1.3.2 Organophosphate Poisoning",
    ]
    assert extracted.sections[0].provenance["heading_detection"] == "heuristic+font"


def test_extract_pdf_renders_vector_diagrams_but_not_tables(tmp_path: Path):
    path = tmp_path / "diagram.pdf"
    with fitz.open() as document:
        page = document.new_page()
        page.insert_text((50, 50), "1 Surveillance Overview", fontname="hebo")
        # A flowchart: boxes joined by arrows, drawn as vectors.
        for index in range(4):
            top = 80 + index * 70
            page.draw_rect(fitz.Rect(150, top, 400, top + 40))
            page.draw_line((275, top + 40), (275, top + 70))
            page.draw_line((270, top + 64), (275, top + 70))
            page.draw_line((280, top + 64), (275, top + 70))
        page.draw_circle((275, 390), 20)
        page.insert_text((50, 440), "Figure 1: Flow chart of the reporting pathway")
        # A ruled table further down the page.
        for x in (50, 250, 450):
            page.draw_line((x, 480), (x, 600))
        for y in (480, 520, 560, 600):
            page.draw_line((50, y), (450, y))
        for row, y in enumerate((505, 545, 585)):
            page.insert_text((60, y), f"Indicator {row}")
            page.insert_text((260, y), f"Value {row}")
        document.save(path)

    extracted = extract_pdf(path)

    vectors = [asset for asset in extracted.assets if asset.provenance.get("source") == "vector_drawing"]
    assert len(vectors) == 1
    assert vectors[0].type == "diagram"
    assert vectors[0].mime_type == "image/png"
    assert vectors[0].provenance["caption"].startswith("Figure 1")
    assert vectors[0].provenance["bbox"][3] < 480
    assert extracted.tables


def test_ocr_renders_pages_at_300_dpi(monkeypatch: pytest.MonkeyPatch):
    import app.document_processing.pdf_extractor as module

    captured = []
    monkeypatch.setattr(module, "_ocr_image_text", lambda image, *_: captured.append(image) or "")
    with fitz.open() as document:
        page = document.new_page(width=595, height=842)
        module._ocr_page_text(page)

    assert abs(fitz.Pixmap(captured[0]).width - 595 * 300 / 72) <= 1
