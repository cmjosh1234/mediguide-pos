import httpx

from app.core.config import get_settings
from app.embeddings.base import EmbeddingProvider


class OllamaEmbeddingProvider(EmbeddingProvider):
    def __init__(self):
        settings = get_settings()
        self.base_url = settings.ollama_base_url.rstrip("/")
        self.model = settings.ollama_embedding_model
        self.dim = settings.embedding_dim

    def embed(self, texts: list[str]) -> list[list[float]]:
        self.last_batch_cacheable = True
        if not texts:
            return []
        with httpx.Client(timeout=120) as client:
            return self._embed_many(client, texts)

    def _embed_many(self, client: httpx.Client, texts: list[str]) -> list[list[float]]:
        payload = {
            "model": self.model,
            "input": texts,
            "truncate": True,
        }
        response = client.post(f"{self.base_url}/api/embed", json=payload)
        if response.status_code < 400:
            embeddings = response.json().get("embeddings") or []
            self._validate_embeddings(embeddings, len(texts))
            return embeddings

        if response.status_code in {400, 404} and len(texts) > 1:
            self.last_batch_cacheable = False
            embeddings: list[list[float]] = []
            for text in texts:
                embeddings.extend(self._embed_single(client, text))
            self._validate_embeddings(embeddings, len(texts))
            return embeddings

        if response.status_code in {400, 404}:
            self.last_batch_cacheable = False
            return self._embed_single(client, texts[0])

        self._raise_with_body(response)
        raise RuntimeError("unreachable")

    def _embed_single(self, client: httpx.Client, text: str) -> list[list[float]]:
        last_error: Exception | None = None
        for candidate in self._single_input_candidates(text):
            response = client.post(
                f"{self.base_url}/api/embed",
                json={
                    "model": self.model,
                    "input": candidate,
                    "truncate": True,
                },
            )
            if response.status_code < 400:
                embeddings = response.json().get("embeddings") or []
                self._validate_embeddings(embeddings, 1)
                return embeddings

            if response.status_code in {400, 404} or self._is_context_length_error(response):
                try:
                    embedding = self._legacy_embed_single(client, candidate)
                    self._validate_embeddings([embedding], 1)
                    return [embedding]
                except Exception as exc:
                    last_error = exc
                    continue

            try:
                self._raise_with_body(response)
            except Exception as exc:
                last_error = exc
                continue

        if last_error is not None:
            raise last_error
        raise RuntimeError("single embedding request failed")

    def _legacy_embed_single(self, client: httpx.Client, text: str) -> list[float]:
        # Older Ollama builds use /api/embeddings with a single prompt only.
        # Retry with shorter prompts if the legacy endpoint cannot handle the full text.
        candidates = self._legacy_prompt_candidates(text)
        last_error: Exception | None = None
        for prompt in candidates:
            response = client.post(
                f"{self.base_url}/api/embeddings",
                json={
                    "model": self.model,
                    "prompt": prompt,
                },
            )
            if response.status_code < 400:
                embedding = response.json().get("embedding")
                if not embedding:
                    raise RuntimeError(
                        "Ollama legacy embeddings endpoint returned no embedding "
                        f"for model={self.model}"
                    )
                return embedding

            try:
                self._raise_with_body(response)
            except Exception as exc:
                last_error = exc
                continue

        if last_error is not None:
            raise last_error
        raise RuntimeError("legacy embedding request failed")

    def _legacy_prompt_candidates(self, text: str) -> list[str]:
        normalized = " ".join((text or "").split())
        candidates: list[str] = []
        for candidate in [
            normalized,
            normalized[:4000],
            normalized[:2500],
            normalized[:1500],
            normalized[:1000],
            normalized[:700],
            normalized[:500],
            normalized[:350],
            normalized[:250],
            normalized[:180],
        ]:
            candidate = candidate.strip()
            if candidate and candidate not in candidates:
                candidates.append(candidate)
        return candidates

    def _single_input_candidates(self, text: str) -> list[str]:
        normalized = " ".join((text or "").split())
        candidates: list[str] = []
        for candidate in [
            normalized,
            normalized[:4000],
            normalized[:2500],
            normalized[:1500],
            normalized[:1000],
            normalized[:700],
            normalized[:500],
            normalized[:350],
            normalized[:250],
            normalized[:180],
        ]:
            candidate = candidate.strip()
            if candidate and candidate not in candidates:
                candidates.append(candidate)
        return candidates

    def _validate_embeddings(self, embeddings: list[list[float]], expected_count: int) -> None:
        if len(embeddings) != expected_count:
            raise RuntimeError(
                "Ollama returned an unexpected embedding count "
                f"for model={self.model}: expected={expected_count}, returned={len(embeddings)}"
            )
        for embedding in embeddings:
            if len(embedding) != self.dim:
                raise RuntimeError(
                    "Configured EMBEDDING_DIM does not match the Ollama embedding output "
                    f"dimension: EMBEDDING_DIM={self.dim}, model={self.model}, "
                    f"returned_dim={len(embedding)}"
                )

    @staticmethod
    def _raise_with_body(response: httpx.Response) -> None:
        detail = response.text.strip()
        if detail:
            raise RuntimeError(
                f"Ollama request failed: status={response.status_code}, body={detail}"
            )
        response.raise_for_status()

    @staticmethod
    def _is_context_length_error(response: httpx.Response) -> bool:
        detail = (response.text or "").lower()
        return "context length" in detail or "input length exceeds" in detail
