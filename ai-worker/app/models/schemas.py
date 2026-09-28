from pydantic import BaseModel, Field


class HealthResponse(BaseModel):
    status: str = "ok"
    service: str = "mediguide-ai-worker"
    version: str = "development"
    revision: str = "unknown"


class ExtractionPreviewResponse(BaseModel):
    title: str | None = None
    pages: int
    sections: int
    chunks: int
    tables: int
    blocks: int = 0
    assets: int = 0
    ocr_pages: list[int] = Field(default_factory=list)
    multi_column_pages: list[int] = Field(default_factory=list)
    warnings: list[str] = Field(default_factory=list)
    markdown_sample: str


class RetrievedChunk(BaseModel):
    """Safe projection of a retrieved guideline chunk for API consumers.
    Intentionally excludes internal fields such as embedding_text."""

    id: str
    document_id: str | None = None
    version_id: str | None = None
    section_id: str | None = None
    block_id: str | None = None
    title: str | None = None
    content: str | None = None
    page_start: int | None = None
    page_end: int | None = None
    language: str | None = None
    program_area: str | None = None
    country: str | None = None
    source_name: str | None = None
    source_version: str | None = None
    similarity: float | None = None


class ReadinessResponse(BaseModel):
    status: str  # "ok" or "degraded"
    checks: dict[str, str]
