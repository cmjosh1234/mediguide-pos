import httpx
import pytest

from app.core.config import get_settings
from app.embeddings.ollama_provider import OllamaEmbeddingProvider


class FakeResponse:
    def __init__(
        self,
        embeddings: list[list[float]] | None = None,
        *,
        status_code: int = 200,
        embedding: list[float] | None = None,
    ):
        self._embeddings = embeddings
        self._embedding = embedding
        self.status_code = status_code
        self.text = f"status={status_code}"

    def raise_for_status(self) -> None:
        if self.status_code >= 400:
            request = httpx.Request("POST", "http://ollama.test")
            response = httpx.Response(self.status_code, request=request)
            raise httpx.HTTPStatusError("request failed", request=request, response=response)
        return None

    def json(self) -> dict:
        if self._embedding is not None:
            return {"embedding": self._embedding}
        return {"embeddings": self._embeddings}


class FakeClient:
    def __init__(self, responses: list[FakeResponse], calls: list[tuple[str, dict]], **_: object):
        self._responses = responses
        self._calls = calls

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False

    def post(self, url: str, json: dict):
        self._calls.append((url, json))
        if self._responses:
            response = self._responses.pop(0)
            if isinstance(response, Exception):
                raise response
            return response
        return FakeResponse([])


def clear_settings_cache() -> None:
    get_settings.cache_clear()


def test_timeout_splits_full_inputs_and_reuses_smaller_batches(monkeypatch):
    from app.core.config import Settings

    calls = []
    timeouts = []
    responses = [
        httpx.ReadTimeout("timed out"),
        FakeResponse([[1.0], [2.0]]),
        FakeResponse([[3.0], [4.0]]),
        FakeResponse([[5.0], [6.0]]),
        FakeResponse([[7.0], [8.0]]),
    ]
    def client(**kwargs):
        timeouts.append(kwargs["timeout"])
        return FakeClient(responses, calls)
    monkeypatch.setattr("app.embeddings.ollama_provider.httpx.Client", client)
    provider = OllamaEmbeddingProvider(Settings(embedding_dim=1, ollama_embedding_timeout_seconds=450))
    texts = [str(i) + " complete clinical text " * 100 for i in range(8)]

    assert provider.embed(texts[:4]) == [[1.0], [2.0], [3.0], [4.0]]
    assert provider.last_batch_cacheable is True
    assert provider.embed(texts[4:]) == [[5.0], [6.0], [7.0], [8.0]]
    assert [payload["input"] for _, payload in calls] == [texts[:4], texts[:2], texts[2:4], texts[4:6], texts[6:]]
    assert all(timeout.read == 450 and timeout.connect == 10 for timeout in timeouts)


def test_single_input_timeout_retries_once_then_succeeds(monkeypatch):
    from app.core.config import Settings

    calls = []
    monkeypatch.setattr("app.embeddings.ollama_provider.httpx.Client", lambda **_: FakeClient(
        [httpx.ReadTimeout("timed out"), FakeResponse([[1.0]])], calls))
    provider = OllamaEmbeddingProvider(Settings(embedding_dim=1))
    assert provider.embed(["full input"]) == [[1.0]]
    assert [payload["input"] for _, payload in calls] == [["full input"], ["full input"]]


def test_nested_splits_preserve_odd_batch_order_and_checkpoint_exact_inputs(monkeypatch):
    from app.core.config import Settings
    from app.services.ingestion_artifacts import IngestionArtifacts
    from tests.test_ingestion_artifacts import Repository, Storage

    calls = []
    responses = [
        httpx.ReadTimeout("timed out"),
        FakeResponse([[1.0]]),
        httpx.ReadTimeout("timed out"),
        FakeResponse([[2.0]]),
        FakeResponse([[3.0]]),
    ]
    monkeypatch.setattr("app.embeddings.ollama_provider.httpx.Client", lambda **_: FakeClient(responses, calls))
    provider = OllamaEmbeddingProvider(Settings(embedding_dim=1))
    cache = IngestionArtifacts(Repository(), Storage(), {})
    texts = ["first", "second", "third", "first"]
    result = cache.embeddings(texts, provider, "model-revision", 1, 8, lambda *_: None)
    assert result == [[1.0], [2.0], [3.0], [1.0]]
    assert [payload["input"] for _, payload in calls] == [texts[:3], texts[:1], texts[1:3], texts[1:2], texts[2:3]]
    # A retry reuses these full-input vectors without contacting the provider.
    calls.clear()
    assert cache.embeddings(texts, provider, "model-revision", 1, 8, lambda *_: None) == result
    assert calls == []


@pytest.mark.parametrize("error_type", [httpx.ReadTimeout, httpx.ConnectTimeout])
def test_persistent_timeout_is_bounded_and_actionable(monkeypatch, error_type):
    from app.core.config import Settings

    calls = []
    monkeypatch.setattr("app.embeddings.ollama_provider.httpx.Client", lambda **_: FakeClient(
        [error_type("timed out"), error_type("timed out")], calls))
    provider = OllamaEmbeddingProvider(Settings(embedding_dim=1))
    with pytest.raises(RuntimeError, match="Ollama embedding request timed out.*server capacity"):
        provider.embed(["full input"])
    assert len(calls) == 2
    assert all(url.endswith("/api/embed") for url, _ in calls)


def test_connect_timeout_does_not_fan_out_batches(monkeypatch):
    from app.core.config import Settings

    calls = []
    monkeypatch.setattr("app.embeddings.ollama_provider.httpx.Client", lambda **_: FakeClient(
        [httpx.ConnectTimeout("timed out"), httpx.ConnectTimeout("timed out")], calls))
    provider = OllamaEmbeddingProvider(Settings(embedding_dim=1))
    with pytest.raises(RuntimeError, match="embedding request timed out"):
        provider.embed(["first", "second", "third"])
    assert [payload["input"] for _, payload in calls] == [["first", "second", "third"]] * 2


def test_ollama_embedding_provider_calls_embed_api(monkeypatch: pytest.MonkeyPatch):
    calls: list[tuple[str, dict]] = []
    monkeypatch.setenv("OLLAMA_BASE_URL", "http://ollama:11434")
    monkeypatch.setenv("OLLAMA_EMBEDDING_MODEL", "mxbai-embed-large:latest")
    monkeypatch.setenv("EMBEDDING_DIM", "4")
    monkeypatch.setattr(
        "app.embeddings.ollama_provider.httpx.Client",
        lambda **kwargs: FakeClient([FakeResponse([[0.1, 0.2, 0.3, 0.4]])], calls, **kwargs),
    )
    clear_settings_cache()

    try:
        provider = OllamaEmbeddingProvider()
        embeddings = provider.embed(["severe malaria treatment"])
    finally:
        clear_settings_cache()

    assert embeddings == [[0.1, 0.2, 0.3, 0.4]]
    assert calls == [
        (
            "http://ollama:11434/api/embed",
            {
                "model": "mxbai-embed-large:latest",
                "input": ["severe malaria treatment"],
                "truncate": True,
            },
        )
    ]


def test_ollama_embedding_provider_falls_back_to_legacy_embeddings_api(
    monkeypatch: pytest.MonkeyPatch,
):
    calls: list[tuple[str, dict]] = []
    monkeypatch.setenv("OLLAMA_BASE_URL", "http://ollama:11434")
    monkeypatch.setenv("OLLAMA_EMBEDDING_MODEL", "mxbai-embed-large:latest")
    monkeypatch.setenv("EMBEDDING_DIM", "4")
    monkeypatch.setattr(
        "app.embeddings.ollama_provider.httpx.Client",
        lambda **kwargs: FakeClient(
            [
                FakeResponse([], status_code=400),
                FakeResponse([], status_code=400),
                FakeResponse(embedding=[0.1, 0.2, 0.3, 0.4]),
            ],
            calls,
            **kwargs,
        ),
    )
    clear_settings_cache()

    try:
        provider = OllamaEmbeddingProvider()
        embeddings = provider.embed(["severe malaria treatment"])
    finally:
        clear_settings_cache()

    assert embeddings == [[0.1, 0.2, 0.3, 0.4]]
    assert calls == [
        (
            "http://ollama:11434/api/embed",
            {
                "model": "mxbai-embed-large:latest",
                "input": ["severe malaria treatment"],
                "truncate": True,
            },
        ),
        (
            "http://ollama:11434/api/embed",
            {
                "model": "mxbai-embed-large:latest",
                "input": "severe malaria treatment",
                "truncate": True,
            },
        ),
        (
            "http://ollama:11434/api/embeddings",
            {
                "model": "mxbai-embed-large:latest",
                "prompt": "severe malaria treatment",
            },
        ),
    ]


def test_ollama_embedding_provider_splits_failed_batch_before_fallback(
    monkeypatch: pytest.MonkeyPatch,
):
    calls: list[tuple[str, dict]] = []
    monkeypatch.setenv("OLLAMA_BASE_URL", "http://ollama:11434")
    monkeypatch.setenv("OLLAMA_EMBEDDING_MODEL", "mxbai-embed-large:latest")
    monkeypatch.setenv("EMBEDDING_DIM", "4")
    monkeypatch.setattr(
        "app.embeddings.ollama_provider.httpx.Client",
        lambda **kwargs: FakeClient(
            [
                FakeResponse([], status_code=400),
                FakeResponse([[0.1, 0.2, 0.3, 0.4]]),
                FakeResponse([[0.4, 0.3, 0.2, 0.1]]),
            ],
            calls,
            **kwargs,
        ),
    )
    clear_settings_cache()

    try:
        provider = OllamaEmbeddingProvider()
        embeddings = provider.embed(["first text", "second text"])
    finally:
        clear_settings_cache()

    assert embeddings == [[0.1, 0.2, 0.3, 0.4], [0.4, 0.3, 0.2, 0.1]]
    assert calls == [
        (
            "http://ollama:11434/api/embed",
            {
                "model": "mxbai-embed-large:latest",
                "input": ["first text", "second text"],
                "truncate": True,
            },
        ),
        (
            "http://ollama:11434/api/embed",
            {
                "model": "mxbai-embed-large:latest",
                "input": "first text",
                "truncate": True,
            },
        ),
        (
            "http://ollama:11434/api/embed",
            {
                "model": "mxbai-embed-large:latest",
                "input": "second text",
                "truncate": True,
            },
        ),
    ]


def test_ollama_embedding_provider_rejects_dimension_mismatch(monkeypatch: pytest.MonkeyPatch):
    calls: list[tuple[str, dict]] = []
    monkeypatch.setenv("OLLAMA_BASE_URL", "http://ollama:11434")
    monkeypatch.setenv("OLLAMA_EMBEDDING_MODEL", "mxbai-embed-large:latest")
    monkeypatch.setenv("EMBEDDING_DIM", "4")
    monkeypatch.setattr(
        "app.embeddings.ollama_provider.httpx.Client",
        lambda **kwargs: FakeClient([FakeResponse([[0.1, 0.2, 0.3]])], calls, **kwargs),
    )
    clear_settings_cache()

    try:
        provider = OllamaEmbeddingProvider()
        with pytest.raises(RuntimeError, match="EMBEDDING_DIM"):
            provider.embed(["severe malaria treatment"])
    finally:
        clear_settings_cache()
