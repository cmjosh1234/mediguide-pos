"""Cache identities follow input bytes, implementation, dependencies and settings."""
import hashlib
import inspect
import json
import re
import subprocess
from functools import lru_cache
from importlib.metadata import version, PackageNotFoundError
from pathlib import Path
import httpx


def identity(value) -> str:
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False, allow_nan=False).encode()).hexdigest()


def package_versions(names):
    result = {}
    for name in names:
        try:
            result[name] = version(name)
        except PackageNotFoundError:
            result[name] = "unavailable"
    return result


@lru_cache(maxsize=1)
def extractor_revision():
    root = Path(__file__).parent
    return identity({"code": {path.name: hashlib.sha256(path.read_bytes()).hexdigest() for path in sorted(root.glob("*.py"))},
                     "packages": package_versions(["pymupdf", "pdfplumber", "pdfminer.six", "beautifulsoup4", "markdownify", "python-slugify", "PyYAML"])})


@lru_cache(maxsize=1)
def ocr_identity():
    # Fail closed for reuse if engine/trained-data identity cannot be established.
    try:
        engine = subprocess.run(["tesseract", "--version"], capture_output=True, text=True, check=True, timeout=10).stdout
        languages = subprocess.run(["tesseract", "--list-langs"], capture_output=True, text=True, check=True, timeout=10).stdout
        directory = Path(re.search(r'"([^"]+)"', languages).group(1))
        trained = hashlib.sha256((directory / "eng.traineddata").read_bytes()).hexdigest()
        return {"engine": engine.strip(), "eng_traineddata": trained, "language": "eng", "dpi": 300, "alpha": False, "args": ["stdout", "-l", "eng", "--dpi", "300"], "wrapper": extractor_revision()}
    except (OSError, ValueError, AttributeError, subprocess.SubprocessError):
        return None


def extraction_identity(checksum, source_format, settings, fallback_title="Guideline"):
    ocr = ocr_identity() if source_format == "pdf" else None
    if source_format == "pdf" and ocr is None:
        return None
    return identity({"artifact": "extraction-v1", "epoch": settings.artifact_cache_epoch, "checksum": checksum,
                     "format": source_format, "extractor": extractor_revision(), "ocr": ocr,
                     "fallback_title": fallback_title if source_format == "markdown" else None})


def embedding_identity(settings, provider):
    name = settings.embedding_provider.lower()
    revision = settings.embedding_cache_model_revision.strip()
    model = None
    endpoint = None
    if name == "hash":
        revision = "deterministic-hash-v1"
    elif name == "ollama":
        model = settings.ollama_embedding_model
        endpoint = settings.ollama_base_url.rstrip("/")
        # Resolve mutable tags on every job, not once per worker lifetime.
        if not revision:
            try:
                response = httpx.get(f"{endpoint}/api/tags", timeout=5)
                response.raise_for_status()
                canonical = model if ":" in model else model + ":latest"
                revision = next(item["digest"] for item in response.json()["models"] if item.get("name") == canonical or item.get("model") == canonical)
            except (httpx.HTTPError, KeyError, StopIteration, ValueError, TypeError):
                return None
    elif name == "openai":
        model = settings.openai_embedding_model
        endpoint = str(getattr(getattr(provider, "client", None), "base_url", "https://api.openai.com/v1"))
    elif name == "sentence_transformers":
        model = settings.embedding_model
    if not revision:
        return None
    try:
        implementation = Path(inspect.getfile(type(provider))).read_bytes()
    except (OSError, TypeError):
        return None
    return {"artifact": "embedding-v1", "epoch": settings.artifact_cache_epoch, "provider": name, "model": model,
            "revision": revision, "dimensions": settings.embedding_dim, "endpoint": endpoint,
            "implementation": hashlib.sha256(implementation).hexdigest(),
            "packages": package_versions(["openai", "sentence-transformers", "transformers", "httpx", "numpy"])}
