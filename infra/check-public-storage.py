#!/usr/bin/env python3
"""Validate browser-facing production storage without printing env values."""
from __future__ import annotations

import argparse
import ipaddress
from pathlib import Path
import re
from urllib.parse import urlsplit
from urllib.request import urlopen


def storage_url(path: Path) -> str:
    settings = {}
    for line in path.read_text().splitlines():
        key, separator, value = line.partition("=")
        if separator and key in {"S3_PUBLIC_ENDPOINT", "S3_PUBLIC_SSL"}:
            settings[key] = value.strip().strip('"\'')
    endpoint = settings.get("S3_PUBLIC_ENDPOINT", "")
    if not endpoint:
        raise ValueError("S3_PUBLIC_ENDPOINT is required: configure the public S3 API hostname")
    if settings.get("S3_PUBLIC_SSL", "").lower() != "true":
        raise ValueError("S3_PUBLIC_SSL must be true in production")
    parsed = urlsplit("https://" + endpoint)
    if parsed.username or parsed.password or parsed.path or parsed.query or parsed.fragment:
        raise ValueError("S3_PUBLIC_ENDPOINT must be hostname[:port], with no scheme, path or credentials")
    host = parsed.hostname or ""
    if not host or host == "localhost" or host.endswith((".localhost", ".local", ".internal")):
        raise ValueError("S3_PUBLIC_ENDPOINT cannot be an internal storage address")
    try:
        address = ipaddress.ip_address(host)
    except ValueError:
        if "." not in host or not re.fullmatch(r"[A-Za-z0-9.-]+", host):
            raise ValueError("S3_PUBLIC_ENDPOINT must be a public hostname") from None
    else:
        if not address.is_global:
            raise ValueError("S3_PUBLIC_ENDPOINT cannot be a private or loopback address")
    try:
        port = parsed.port
    except ValueError:
        raise ValueError("S3_PUBLIC_ENDPOINT has an invalid port") from None
    if port is not None and not 1 <= port <= 65535:
        raise ValueError("S3_PUBLIC_ENDPOINT has an invalid port")
    return "https://" + endpoint


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("file", type=Path)
    parser.add_argument("--check-network", action="store_true")
    args = parser.parse_args()
    try:
        base = storage_url(args.file)
        if args.check_network:
            with urlopen(base + "/minio/health/live", timeout=15) as response:
                if response.status != 200 or urlsplit(response.url).scheme != "https" or urlsplit(response.url).netloc != urlsplit(base).netloc:
                    raise ValueError("Public storage health must return 200 without redirecting to another host")
    except (ValueError, OSError) as error:
        # Network exceptions may contain URLs; do not print their content.
        reason = str(error) if isinstance(error, ValueError) else "Public storage is unreachable; check DNS, TLS and the S3 API reverse proxy"
        parser.exit(1, reason + "\n")
    print("Public storage configuration valid" + ("; HTTPS S3 API health passed" if args.check_network else ""))


if __name__ == "__main__":
    main()
