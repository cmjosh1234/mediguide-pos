#!/usr/bin/env python3
"""Verify deployment environment coverage and running containers without exposing values."""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import subprocess


def read_assignments(path):
    spec = importlib.util.spec_from_file_location("organize_env", Path(__file__).with_name("organize-env.py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.assignments(path.read_bytes())


def validate(path, compose_file):
    values = read_assignments(path)
    # Release metadata is supplied by release.env; build-only and development
    # settings are intentionally excluded from the production runtime schema.
    references = set(re.findall(r"(?<!\$)\$\{([A-Z][A-Z0-9_]*)", compose_file.read_text()))
    references -= {"API_IMAGE", "AI_WORKER_IMAGE", "DASHBOARD_IMAGE", "GUIDELINES_IMAGE", "BUILD_VERSION", "BUILD_REVISION"}
    missing = references - values.keys()
    if missing:
        raise ValueError("Missing production environment keys: " + ", ".join(sorted(missing)))
    required = {"DATABASE_URL", "AI_DATABASE_URL", "POSTGRES_USER", "POSTGRES_DB", "POSTGRES_PASSWORD", "JWT_SECRET", "MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD", "AI_WORKER_SECRET", "DEFAULT_ADMIN_PASSWORD"}
    driver = values.get("MAIL_DRIVER", b"").strip(b"\"'")
    if driver == b"resend":
        required |= {"MAIL_FROM", "RESEND_API_KEY"}
    elif driver == b"smtp":
        required |= {"MAIL_FROM", "SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD"}
    for key in sorted(required):
        if not values.get(key, b"").strip(b"\"'"):
            raise ValueError("Required production environment key is empty: " + key)
    if values.get("APP_ENV") != b"production":
        raise ValueError("APP_ENV must be production")


def compare(config, containers):
    errors = []
    checked = 0
    for name, service in config["services"].items():
        expected = service.get("environment", {})
        matches = [c for c in containers if c.get("Config", {}).get("Labels", {}).get("com.docker.compose.service") == name]
        if not matches:
            errors.append(name + ": container missing")
        for container in matches:
            actual = dict(item.split("=", 1) for item in container["Config"].get("Env", []) if "=" in item)
            for key, value in expected.items():
                if value is not None and actual.get(key) != str(value):
                    errors.append(name + ": environment mismatch for " + key)
                checked += 1
    if errors:
        raise ValueError("; ".join(errors))
    return checked


def run_json(command):
    result = subprocess.run(command, capture_output=True, text=True)
    if result.returncode:
        # Docker diagnostics can include interpolated credentials.
        raise ValueError("Docker environment verification command failed; output suppressed")
    return json.loads(result.stdout)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("file", type=Path)
    parser.add_argument("--runtime", action="store_true")
    args = parser.parse_args()
    compose_file = args.file.parent / "docker-compose.yml"
    try:
        validate(args.file, compose_file)
        print("Production environment covers all production Compose settings; required credentials are present")
        if args.runtime:
            compose = ["docker", "compose", "--env-file", str(args.file), "--env-file", str(args.file.parent / "release.env"), "-f", str(compose_file)]
            config = run_json(compose + ["config", "--format", "json"])
            result = subprocess.run(compose + ["ps", "--all", "--quiet"], capture_output=True, text=True)
            if result.returncode or not result.stdout.split():
                raise ValueError("Unable to identify production containers")
            count = compare(config, run_json(["docker", "inspect", *result.stdout.split()]))
            print(f"Verified {count} configured environment values across all production containers; no values printed")
    except (ValueError, OSError) as error:
        raise SystemExit(str(error) if isinstance(error, ValueError) else "Production environment verification failed")


if __name__ == "__main__":
    main()
