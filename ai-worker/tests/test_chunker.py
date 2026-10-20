import pytest

from app.document_processing.chunker import chunk_blocks, chunk_sections
from app.document_processing.types import ExtractedContentBlock, ExtractedSection


def test_chunk_sections_basic():
    section = ExtractedSection(title="Malaria", level=1, text="word " * 1200, sort_order=0)
    chunks = chunk_sections([section])
    assert len(chunks) >= 1
    assert chunks[0].title == "Malaria"


@pytest.mark.parametrize("kind", [
    "recommendation", "warning", "caution", "key_point", "contraindication", "dosage",
    "evidence", "definition", "procedure", "algorithm_reference", "clinical_note",
    "referral_criteria",
])
def test_clinical_callouts_have_search_chunks(kind):
    block = ExtractedContentBlock(
        type=kind, sort_order=7, section_order=3, source_fingerprint="test",
        content={"title": "Safety", "content": "Verify the authoritative source."},
    )
    chunks = chunk_blocks([block])
    assert len(chunks) == 1
    assert chunks[0].content == "Safety Verify the authoritative source."
    assert chunks[0].block_order == 7
    assert chunks[0].section_order == 3


def test_figure_indexes_description_not_asset_identifier():
    block = ExtractedContentBlock(
        type="figure", sort_order=1, source_fingerprint="test",
        content={"asset_id": "private-id", "caption": "Source chart", "alternative_text": "Source chart"},
    )
    assert chunk_blocks([block])[0].content == "Source chart"


def test_undescribed_figure_does_not_fabricate_search_text():
    block = ExtractedContentBlock(type="figure", sort_order=1, source_fingerprint="test", content={"asset_id": "private-id"})
    assert chunk_blocks([block]) == []
