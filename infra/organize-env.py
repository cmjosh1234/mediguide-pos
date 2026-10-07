#!/usr/bin/env python3
"""Group dotenv assignments without interpreting or printing their values."""
from __future__ import annotations

import argparse
import os
from pathlib import Path
import re
import tempfile

SECTIONS = [
    ("Runtime and network bindings", ("APP_NAME", "APP_ENV", "HTTP_PORT", "API_HOST", "GRPC_HOST", "WORKER_GRPC_PORT", "LOG_LEVEL", "LEGACY_CLINICAL_TOOLS_DIR", "PUBLIC_BIND_ADDRESS", "DEV_DATA_BIND_ADDRESS", "TRUSTED_PROXIES")),
    ("Container images", ("API_IMAGE", "AI_WORKER_IMAGE", "DASHBOARD_IMAGE", "GUIDELINES_IMAGE", "DASHBOARD_DEV_IMAGE", "GUIDELINES_DEV_IMAGE")),
    ("Public web routes", ("PUBLIC_SITE_URL", "PUBLIC_API_BASE_URL", "PUBLIC_APP_URL", "DASHBOARD_PUBLIC_URL", "DASHBOARD_BASE_PATH", "ALLOWED_ORIGINS", "API_PUBLIC_PORT", "DASHBOARD_PUBLIC_PORT", "GUIDELINES_PUBLIC_PORT")),
    ("Object storage and browser asset URLs", ("STORAGE_DRIVER", "MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD", "S3_", "MINIO_API_CORS_ALLOW_ORIGIN", "GUIDELINE_DIRECT_UPLOADS", "MAX_UPLOAD_MB", "MAX_UPLOAD_BYTES", "MINIO_PUBLIC_PORT", "MINIO_CONSOLE_PUBLIC_PORT")),
    ("Database connections", ("POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_PUBLIC_PORT", "DATABASE_URL", "AI_DATABASE_URL", "DB_STATEMENT_TIMEOUT_MS")),
    ("Authentication and initial administrator", ("JWT_SECRET", "JWT_ISSUER", "JWT_ACCESS_TTL_MINUTES", "JWT_REFRESH_TTL_MINUTES", "DEFAULT_ADMIN_NAME", "DEFAULT_ADMIN_EMAIL", "DEFAULT_ADMIN_PASSWORD", "ACCOUNT_ACTION_URL")),
    ("Account email delivery", ("MAIL_DRIVER", "MAIL_FROM", "RESEND_API_KEY", "SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD")),
    ("Firebase and notification worker", ("FIREBASE_", "NOTIFICATION_")),
    ("Redis and response caching", ("REDIS_", "CACHE_")),
    ("Rate limits", ("RATE_LIMIT_",)),
    ("AI, search and model configuration", ("AI_RAG_PROVIDER", "AI_WORKER_", "WORKER_API_SECRET", "RAG_", "OLLAMA_", "OPENAI_", "LLM_PROVIDER")),
    ("Document ingestion and processing", ("INGESTION_", "WORKER_ENABLED", "WORKER_POLL_INTERVAL_SECONDS", "WORKER_BATCH_SIZE", "WORKER_MAX_ATTEMPTS", "WORKER_RETRY_BACKOFF_SECONDS", "PDF_PAGE_WORKERS", "OCR_WORKERS", "EMBEDDING_", "ARTIFACT_CACHE_EPOCH", "CHUNK_SIZE", "CHUNK_OVERLAP", "MIN_CHUNK_CHARS")),
    ("Governed guideline asset import", ("GUIDELINE_ASSET_IMPORT_",)),
    ("Development-only file watching", ("CHOKIDAR_USEPOLLING",)),
    ("Build metadata", ("BUILD_VERSION", "BUILD_REVISION")),
]


def assignments(raw: bytes) -> dict[str, bytes]:
    result: dict[str, bytes] = {}
    for number, line in enumerate(raw.splitlines(), 1):
        if not line.strip() or line.lstrip().startswith(b"#"):
            continue
        match = re.fullmatch(rb"([A-Za-z_][A-Za-z0-9_]*)=(.*)", line)
        if not match:
            # Do not echo malformed lines: they can contain credentials.
            raise ValueError(f"Unsupported dotenv syntax at line {number}; file unchanged")
        result[match[1].decode("ascii")] = match[2]
    return result


def organize(raw: bytes, defaults: bytes | None = None) -> bytes:
    values = assignments(raw)
    original = values.copy()
    if defaults is not None:
        for key, value in assignments(defaults).items():
            values.setdefault(key, value)
    remaining = set(values)
    blocks = ["# Environment settings grouped by responsibility.\n# Credential values are preserved verbatim; do not commit populated private env files.".encode()]
    for title, selectors in SECTIONS:
        keys = [key for key in values if any(key.startswith(s) if s.endswith("_") else key == s for s in selectors)]
        keys = sorted(set(keys) & remaining)
        if not keys:
            continue
        block = [f"# {title}".encode()]
        if title.startswith("Object storage"):
            block += [b"# S3_PUBLIC_ENDPOINT is a browser-reachable hostname[:port], without scheme or path.",
                      b"# Production requires working DNS, TLS and an S3 API reverse proxy preserving Host/path/query.",
                      b"# Keep S3_PUBLIC_SSL=true in production. Internal API storage traffic still uses minio:9000."]
        if title.startswith("Firebase"):
            block += [b"# Service-account credentials stay backend-only; use encoded JSON, not a filename."]
        if title == "Account email delivery":
            block += [b"# MAIL_DRIVER: disabled, development (non-production), smtp, or resend (HTTPS API).",
                      b"# Resend requires a verified sender in MAIL_FROM and a private RESEND_API_KEY."]
        block.extend(key.encode() + b"=" + values[key] for key in keys)
        remaining.difference_update(keys)
        blocks.append(b"\n".join(block))
    if remaining:
        blocks.append(b"# Additional settings\n" + b"\n".join(key.encode() + b"=" + values[key] for key in sorted(remaining)))
    output = b"\n\n".join(blocks) + b"\n"
    parsed_output = assignments(output)
    if parsed_output != values or any(parsed_output[key] != value for key, value in original.items()):
        raise ValueError("Environment preservation check failed; file unchanged")
    return output


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("file", type=Path)
    parser.add_argument("--private", action="store_true", help="Restrict file permissions to its owner")
    parser.add_argument("--defaults-from", type=Path, help="Add absent keys from a matching environment template; existing values are preserved")
    args = parser.parse_args()
    path = args.file.resolve()
    defaults = args.defaults_from.read_bytes() if args.defaults_from else None
    output = organize(path.read_bytes(), defaults)
    mode = 0o600 if args.private else path.stat().st_mode & 0o777
    fd, temporary = tempfile.mkstemp(prefix=".organized-env-", dir=path.parent)
    try:
        os.fchmod(fd, mode)
        with os.fdopen(fd, "wb") as stream:
            stream.write(output)
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    print(f"Organized {args.file}: unique keys, effective values preserved; no values printed")


if __name__ == "__main__":
    main()
