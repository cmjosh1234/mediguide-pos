from app.core.config import Settings, get_settings
from app.embeddings.base import EmbeddingProvider


class SentenceTransformersProvider(EmbeddingProvider):
    def __init__(self, settings: Settings | None = None):
        try:
            from sentence_transformers import SentenceTransformer
        except ImportError as exc:
            raise RuntimeError(
                "Install sentence-transformers to use EMBEDDING_PROVIDER=sentence_transformers"
            ) from exc
        settings = settings or get_settings()
        self.model = SentenceTransformer(settings.embedding_model)
        self.dim = self.model.get_sentence_embedding_dimension()
        if self.dim != settings.embedding_dim:
            raise RuntimeError(
                "Configured EMBEDDING_DIM does not match the SentenceTransformers model "
                f"dimension: EMBEDDING_DIM={settings.embedding_dim}, model_dim={self.dim}, "
                f"EMBEDDING_MODEL={settings.embedding_model}"
            )

    def embed(self, texts: list[str]) -> list[list[float]]:
        embeddings = self.model.encode(texts, normalize_embeddings=True, show_progress_bar=False)
        return embeddings.tolist()
