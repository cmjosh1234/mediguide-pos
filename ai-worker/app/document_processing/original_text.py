"""Plain-text indexing for documents published as their uploaded file.

Documents such as forms keep the uploaded PDF or Word file as the published
document. Their text is extracted only so search and RAG can find them; it is
never turned into editable Markdown, sections or blocks.
"""

from __future__ import annotations

import html
import zipfile
from pathlib import Path
from xml.etree import ElementTree

import fitz

from app.document_processing.chunker import Chunk
from app.document_processing.pdf_extractor import (
    _clean_text,
    _is_low_signal_page_text,
    _ocr_page_text,
)

_WORD_NAMESPACE = "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}"
# A Word body larger than this is not a form; refuse it rather than inflate it.
_MAX_DOCX_XML_BYTES = 20 * 1024 * 1024

PageText = tuple[int | None, str]


def extract_pdf_page_texts(path: Path) -> list[PageText]:
    """Return each page's text, using OCR for scanned pages as extract_pdf does."""
    pages: list[PageText] = []
    with fitz.open(str(path)) as document:
        for page_number, page in enumerate(document, start=1):
            text = _clean_text(page.get_text("text"))
            if _is_low_signal_page_text(text):
                ocr_text = _ocr_page_text(page)
                if len(ocr_text) > len(text):
                    text = ocr_text
            pages.append((page_number, text))
    return pages


def extract_docx_text(path: Path) -> str:
    """Return the text of a .docx body, one line per paragraph or table cell."""
    with zipfile.ZipFile(path) as archive:
        try:
            entry = archive.getinfo("word/document.xml")
        except KeyError as exc:
            raise ValueError("Word document has no body (word/document.xml)") from exc
        if entry.file_size > _MAX_DOCX_XML_BYTES:
            raise ValueError("Word document body is too large to index")
        content = archive.read(entry)
    # Word bodies never declare a DTD; refusing one blocks entity-expansion attacks.
    if b"<!DOCTYPE" in content or b"<!ENTITY" in content:
        raise ValueError("Word document body contains a DTD")
    root = ElementTree.fromstring(content)
    lines: list[str] = []
    for paragraph in root.iter(f"{_WORD_NAMESPACE}p"):
        parts: list[str] = []
        for node in paragraph.iter():
            if node.tag == f"{_WORD_NAMESPACE}t" and node.text:
                parts.append(node.text)
            elif node.tag == f"{_WORD_NAMESPACE}tab":
                parts.append(" ")
            elif node.tag in {f"{_WORD_NAMESPACE}br", f"{_WORD_NAMESPACE}cr"}:
                parts.append("\n")
        line = "".join(parts).strip()
        if line:
            lines.append(line)
    return "\n".join(lines)


def chunk_page_texts(
    pages: list[PageText], *, title: str, chunk_size: int, overlap: int
) -> list[Chunk]:
    """Split page texts into overlapping word windows that keep their page range.

    Short documents still produce one chunk: a form with a few fields must stay
    findable even when its text is below the usual minimum chunk length.
    """
    words: list[tuple[str, int | None]] = [
        (word, page_number) for page_number, text in pages for word in text.split()
    ]
    if not words:
        return []
    size = max(1, chunk_size)
    step = max(1, size - max(0, overlap))
    chunks: list[Chunk] = []
    start = 0
    while True:
        window = words[start : start + size]
        pages_in_window = [page for _, page in window if page is not None]
        content = " ".join(word for word, _ in window)
        chunks.append(
            Chunk(
                title=title or None,
                content=content,
                html=f"<p>{html.escape(content)}</p>",
                page_start=min(pages_in_window) if pages_in_window else None,
                page_end=max(pages_in_window) if pages_in_window else None,
                section_order=0,
                chunk_order=len(chunks),
            )
        )
        if start + size >= len(words):
            return chunks
        start += step
