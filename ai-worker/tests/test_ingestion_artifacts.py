from copy import deepcopy
import pytest
from app.services.ingestion_artifacts import IngestionArtifacts, encode_document
from app.document_processing.types import ExtractedDocument, ExtractedContentBlock, ExtractedAsset


class Repository:
    def __init__(self):
        self.rows = {}

    def get_many(self, kind, keys):
        return {key: deepcopy(self.rows[kind, key]) for key in keys if (kind, key) in self.rows}

    def put_many(self, kind, values):
        self.rows.update({(kind, key): deepcopy(value) for key, value in values.items()})


class Storage:
    def __init__(self):
        self.objects = {}

    def upload_bytes(self, data, key, content_type):
        self.objects[key] = data

    def download_file(self, key, path):
        path.write_bytes(self.objects[key])


class Provider:
    def __init__(self, fail_on=None):
        self.calls = []
        self.fail_on = fail_on

    def embed(self, texts):
        self.calls.append(texts)
        if len(self.calls) == self.fail_on:
            raise RuntimeError("temporary provider failure")
        return [[float(len(text)), 1.0] for text in texts]


def document():
    return ExtractedDocument("Title", 1, "<p>text</p>", "text", "text", [], [],
        blocks=[ExtractedContentBlock("paragraph", 0, {"text": "text"}, "fingerprint")],
        assets=[ExtractedAsset("figure", "page-image", "image-fingerprint", "image/png", "checksum", 3, data=b"png")])


def test_extraction_reuse_is_durable_and_does_not_copy_version_enrichment():
    repo, storage = Repository(), Storage()
    first = IngestionArtifacts(repo, storage, {}).extraction("key", document)
    original = encode_document(first)
    first.blocks[0].provenance["ingestion_job_id"] = "job-A"
    first.assets[0].storage_key = "version-A/image.png"
    first.assets[0].data = None
    metrics = {}
    second = IngestionArtifacts(repo, storage, metrics).extraction("key", lambda: pytest.fail("extraction repeated"))
    assert encode_document(second) == original
    assert metrics["extraction_cache_hits"] == 1


def test_corrupt_or_missing_extraction_is_recomputed():
    repo, storage = Repository(), Storage()
    cache = IngestionArtifacts(repo, storage, {})
    cache.extraction("key", document)
    storage.objects.clear()
    assert cache.extraction("key", document).text == "text"
    assert cache.metrics["extraction_cache_misses"] == 2


def test_ocr_settings_and_page_identity_and_failed_checkpoint():
    cache = IngestionArtifacts(Repository(), Storage(), {})
    assert cache.ocr("page", {"revision": "one"}, lambda: "text") == "text"
    assert cache.ocr("page", {"revision": "one"}, lambda: pytest.fail("OCR repeated")) == "text"
    assert cache.ocr("page", {"revision": "two"}, lambda: "changed") == "changed"
    with pytest.raises(RuntimeError):
        cache.ocr("new-page", {}, lambda: (_ for _ in ()).throw(RuntimeError()))
    assert cache.ocr("new-page", {}, lambda: "recovered") == "recovered"


def embed(cache, provider, texts, revision="one"):
    return cache.embeddings(texts, provider, {"revision": revision}, 2, 1, lambda *_: None)


def test_embeddings_checkpoint_retry_revision_and_exact_changed_inputs():
    repo, storage = Repository(), Storage()
    cache = IngestionArtifacts(repo, storage, {})
    with pytest.raises(RuntimeError):
        embed(cache, Provider(fail_on=2), ["one", "two", "one"])
    provider = Provider()
    result = embed(IngestionArtifacts(repo, storage, {}), provider, ["one", "two", "one"])
    assert provider.calls == [["two"]]
    assert result == [[3.0, 1.0]] * 3
    provider = Provider()
    embed(cache, provider, ["one", "two", "one"])
    assert provider.calls == []
    embed(cache, provider, ["one", "two changed"])
    assert provider.calls == [["two changed"]]
    embed(cache, provider, ["one"], revision="new-model")
    assert provider.calls[-1] == ["one"]


def test_invalid_vectors_and_shorter_input_fallback_not_cached():
    cache = IngestionArtifacts(Repository(), Storage(), {})
    provider = Provider()
    provider.embed = lambda texts: [[float("nan"), 1.0]]
    with pytest.raises(ValueError):
        embed(cache, provider, ["text"])
    assert not cache.repository.rows
    provider = Provider()
    provider.last_batch_cacheable = False
    embed(cache, provider, ["text"])
    embed(cache, provider, ["text"])
    assert len(provider.calls) == 2
    assert not cache.repository.rows


def test_unresolved_model_revision_does_not_persist_embeddings():
    cache = IngestionArtifacts(Repository(), Storage(), {})
    cache.embeddings(["text"], Provider(), None, 2, 1, lambda *_: None)
    assert not cache.repository.rows


def test_best_effort_extraction_failure_is_not_frozen_in_cache():
    cache = IngestionArtifacts(Repository(), Storage(), {})
    def incomplete():
        cache.extraction_incomplete()
        return document()
    cache.extraction("key", incomplete)
    assert not cache.repository.rows
    assert not cache.storage.objects
    cache.extraction("key", document)
    assert cache.repository.rows


def test_jobs_reuse_raw_pdf_but_keep_version_references_separate(monkeypatch):
    from unittest.mock import Mock
    from app.core.config import Settings
    from app.embeddings.hash_provider import HashEmbeddingProvider
    from app.services.ingestion_service import IngestionService
    import app.services.ingestion_service as module

    repo, storage = Repository(), Storage()
    storage.objects["source.pdf"] = b"test PDF bytes"
    extraction = Mock(side_effect=lambda *args, **kwargs: document())
    monkeypatch.setattr(module, "extract_pdf", extraction)
    monkeypatch.setattr(module, "ocr_identity", lambda: {"engine": "test"})
    monkeypatch.setattr(module, "extraction_identity", lambda *args: "same-pdf")
    provider = HashEmbeddingProvider(2)
    provider.embed = Mock(wraps=provider.embed)
    saved = []
    for target in ("version-A", "version-B"):
        service = IngestionService.__new__(IngestionService)
        service.settings = Settings(
            INGESTION_ARTIFACT_REUSE=True,
            EMBEDDING_WORKERS=1,
            embedding_provider="hash",
            embedding_dim=2,
        )
        service.artifact_repository = repo
        service.storage = storage
        service.embedder = provider
        service.jobs = Mock()
        service.jobs.cancellation_requested.return_value = False
        service.guidelines = Mock()
        service.guidelines.EXTRACTION_SCHEMA_VERSION = 1
        service.guidelines.get_version_with_document.return_value = {"original_file_key": "source.pdf", "document_title": target}
        service.guidelines.is_extraction_current.return_value = False
        service._process({"id": target + "-job", "version_id": target, "payload_json": {}})
        saved.append(service.guidelines.replace_extraction.call_args.kwargs)
    assert extraction.call_count == 1
    assert provider.embed.call_count == 1
    for target, row in zip(("version-A", "version-B"), saved):
        assert row["version_id"] == target
        assert row["blocks"][0].provenance["ingestion_job_id"] == target + "-job"
        assert row["assets"][1].storage_key.startswith("guidelines/" + target + "/")
        assert row["metadata"]["ingestion_job_id"] == target + "-job"
    assert saved[0]["embeddings"] == saved[1]["embeddings"]


def test_embedding_provider_factory_uses_explicit_job_settings():
    from app.core.config import Settings
    from app.embeddings.factory import get_embedding_provider
    from app.embeddings.hash_provider import HashEmbeddingProvider

    provider = get_embedding_provider(
        Settings(embedding_provider="hash", embedding_dim=3)
    )

    assert isinstance(provider, HashEmbeddingProvider)
    assert len(provider.embed(["text"])[0]) == 3


def test_identity_invalidates_on_settings_revision_and_exact_source(monkeypatch):
    from app.core.config import Settings
    from app.embeddings.hash_provider import HashEmbeddingProvider
    import app.document_processing.artifact_identity as module
    settings = Settings(embedding_provider="hash", embedding_dim=2)
    provider = HashEmbeddingProvider(2)
    original = module.embedding_identity(settings, provider)
    settings.embedding_dim = 3
    assert module.embedding_identity(settings, provider) != original
    settings.embedding_dim = 2
    settings.artifact_cache_epoch = "new"
    assert module.embedding_identity(settings, provider) != original
    monkeypatch.setattr(module, "ocr_identity", lambda: {"engine": "one"})
    pdf = module.extraction_identity("checksum", "pdf", settings)
    assert module.extraction_identity("changed", "pdf", settings) != pdf
    monkeypatch.setattr(module, "ocr_identity", lambda: {"engine": "two"})
    assert module.extraction_identity("checksum", "pdf", settings) != pdf
    monkeypatch.setattr(module, "ocr_identity", lambda: None)
    assert module.extraction_identity("checksum", "pdf", settings) is None


def test_ollama_mutable_tag_uses_digest_and_unknown_revision_disables_cache(monkeypatch):
    from unittest.mock import Mock
    from app.core.config import Settings
    from app.embeddings.hash_provider import HashEmbeddingProvider
    import app.document_processing.artifact_identity as module
    settings = Settings(embedding_provider="ollama")
    response = Mock()
    response.json.return_value = {"models": [{"name": settings.ollama_embedding_model, "digest": "first"}]}
    monkeypatch.setattr(module.httpx, "get", lambda *args, **kwargs: response)
    first = module.embedding_identity(settings, HashEmbeddingProvider(2))
    response.json.return_value["models"][0]["digest"] = "second"
    assert module.embedding_identity(settings, HashEmbeddingProvider(2)) != first
    response.json.return_value = {"models": []}
    assert module.embedding_identity(settings, HashEmbeddingProvider(2)) is None
