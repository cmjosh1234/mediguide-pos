#!/usr/bin/env python3
"""Group dotenv assignments without interpreting or printing their values."""
from __future__ import annotations

import argparse
import os
from pathlib import Path
import re
import tempfile

SECTIONS = [
    ("Runtime and network bindings", ("APP_ENV", "PUBLIC_BIND_ADDRESS", "DEV_DATA_BIND_ADDRESS", "TRUSTED_PROXIES")),
    ("Container images", ("API_IMAGE", "AI_WORKER_IMAGE", "DASHBOARD_IMAGE", "GUIDELINES_IMAGE", "DASHBOARD_DEV_IMAGE", "GUIDELINES_DEV_IMAGE")),
    ("Public web routes", ("PUBLIC_SITE_URL", "PUBLIC_API_BASE_URL", "DASHBOARD_PUBLIC_URL", "DASHBOARD_BASE_PATH", "ALLOWED_ORIGINS", "API_PUBLIC_PORT", "DASHBOARD_PUBLIC_PORT", "GUIDELINES_PUBLIC_PORT")),
    ("Object storage and browser asset URLs", ("MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD", "S3_BUCKET", "S3_PUBLIC_ENDPOINT", "S3_PUBLIC_SSL", "MINIO_API_CORS_ALLOW_ORIGIN", "GUIDELINE_DIRECT_UPLOADS", "MINIO_PUBLIC_PORT", "MINIO_CONSOLE_PUBLIC_PORT")),
    ("Database connections", ("POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_PUBLIC_PORT", "DATABASE_URL", "AI_DATABASE_URL")),
    ("Authentication and initial administrator", ("JWT_SECRET", "JWT_ISSUER", "JWT_ACCESS_TTL_MINUTES", "JWT_REFRESH_TTL_MINUTES", "DEFAULT_ADMIN_NAME", "DEFAULT_ADMIN_EMAIL", "DEFAULT_ADMIN_PASSWORD", "ACCOUNT_ACTION_URL")),
    ("Account email delivery", ("MAIL_DRIVER", "MAIL_FROM", "SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD")),
    ("Firebase and notification worker", ("FIREBASE_", "NOTIFICATION_")),
    ("Redis and response caching", ("REDIS_", "CACHE_")),
    ("Rate limits", ("RATE_LIMIT_",)),
    ("AI, search and model configuration", ("AI_WORKER_", "RAG_", "OLLAMA_")),
    ("Document ingestion and processing", ("INGESTION_", "PDF_PAGE_WORKERS", "OCR_WORKERS", "EMBEDDING_", "ARTIFACT_CACHE_EPOCH")),
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


def organize(raw: bytes) -> bytes:
    values = assignments(raw)
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
        block.extend(key.encode() + b"=" + values[key] for key in keys)
        remaining.difference_update(keys)
        blocks.append(b"\n".join(block))
    if remaining:
        blocks.append(b"# Additional settings\n" + b"\n".join(key.encode() + b"=" + values[key] for key in sorted(remaining)))
    output = b"\n\n".join(blocks) + b"\n"
    if assignments(output) != values:
        raise ValueError("Environment preservation check failed; file unchanged")
    return output


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("file", type=Path)
    parser.add_argument("--private", action="store_true", help="Restrict file permissions to its owner")
    args = parser.parse_args()
    path = args.file.resolve()
    output = organize(path.read_bytes())
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
