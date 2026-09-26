from app.core.config import Settings, get_settings
from app.embeddings.base import EmbeddingProvider


class OpenAIEmbeddingProvider(EmbeddingProvider):
    def __init__(self, settings: Settings | None = None):
        try:
            from openai import OpenAI
        except ImportError as exc:
            raise RuntimeError("Install openai to use EMBEDDING_PROVIDER=openai") from exc
        settings = settings or get_settings()
        self.client = OpenAI(api_key=settings.openai_api_key)
        self.model = settings.openai_embedding_model
        self.dim = settings.embedding_dim

    def embed(self, texts: list[str]) -> list[list[float]]:
        response = self.client.embeddings.create(model=self.model, input=texts)
        embeddings = [item.embedding for item in response.data]
        for embedding in embeddings:
            if len(embedding) != self.dim:
                raise RuntimeError(
                    "Configured EMBEDDING_DIM does not match the OpenAI embedding output "
                    f"dimension: EMBEDDING_DIM={self.dim}, model={self.model}, "
                    f"returned_dim={len(embedding)}"
                )
        return embeddings
