from app.core.config import Settings, get_settings
from app.embeddings.base import EmbeddingProvider
from app.embeddings.hash_provider import HashEmbeddingProvider


def get_embedding_provider(settings: Settings | None = None) -> EmbeddingProvider:
    settings = settings or get_settings()
    provider = settings.embedding_provider.lower()
    if provider == "hash":
        return HashEmbeddingProvider(settings.embedding_dim)
    if provider == "ollama":
        from app.embeddings.ollama_provider import OllamaEmbeddingProvider

        return OllamaEmbeddingProvider(settings)
    if provider == "sentence_transformers":
        from app.embeddings.sentence_transformers_provider import SentenceTransformersProvider

        return SentenceTransformersProvider(settings)
    if provider == "openai":
        from app.embeddings.openai_provider import OpenAIEmbeddingProvider

        return OpenAIEmbeddingProvider(settings)
    raise ValueError(f"Unsupported EMBEDDING_PROVIDER={settings.embedding_provider}")


def to_pgvector(vector: list[float]) -> str:
    return "[" + ",".join(f"{x:.8f}" for x in vector) + "]"
