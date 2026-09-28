#!/usr/bin/env python3
"""Check Play source prerequisites and export non-secret store preparation files.

This does not certify a binary, policy compliance, or device behavior.
"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import struct
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[1]
METADATA = ROOT / "fastlane/metadata/android/en-US"

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--submission", action="store_true",
                        help="Also reject known account-deletion placeholder")
    parser.add_argument("--export", action="store_true")
    args = parser.parse_args()
    errors = []
    if args.submission:
        required = ("deletion_fulfilment", "privacy_policy", "data_safety",
                    "signed_aab", "device_acceptance", "store_assets",
                    "console_declarations")
        try:
            readiness = json.loads((ROOT / "play-store/release-readiness.json").read_text())
            checks = readiness["checks"]
            for name in required:
                check = checks.get(name, {})
                if check.get("status") != "passed" or not str(check.get("evidence", "")).strip():
                    errors.append(f"Release evidence outstanding: {name}")
        except (OSError, ValueError, KeyError, TypeError, AttributeError):
            errors.append("Release readiness evidence is missing or invalid")
    for name, limit in (("title.txt", 30), ("short_description.txt", 80),
                        ("full_description.txt", 4000), ("changelogs/default.txt", 500)):
        path = METADATA / name
        value = path.read_text(encoding="utf-8").strip() if path.is_file() else ""
        if not value or len(value) > limit:
            errors.append(f"{name}: required, maximum {limit} characters")
    icon = ROOT / "android/app/src/production/ic_launcher-playstore.png"
    header = icon.read_bytes()[:24] if icon.is_file() else b""
    if len(header) != 24 or header[:8] != b"\x89PNG\r\n\x1a\n" or struct.unpack(">II", header[16:24]) != (512, 512):
        errors.append("Production Play icon must be a 512x512 PNG")
    profile = ROOT / "lib/features/profile/presentation/screens/profile_page.dart"
    unavailable = "Account deletion is not available in this version."
    if unavailable in profile.read_text(encoding="utf-8"):
        message = "Account deletion is a placeholder; submission is blocked"
        if args.submission:
            errors.append(message)
        else:
            print("BLOCKER:", message)
    if errors:
        for error in errors:
            print("ERROR:", error, file=sys.stderr)
        return 1
    print("Store text/icon checks passed. Build, signing, policies and device QA remain separate gates.")
    if args.export:
        # Recreate only this dedicated generated-output folder; never copy credentials.
        output = ROOT / "build/play-store"
        if output.exists():
            shutil.rmtree(output)
        output.mkdir(parents=True)
        shutil.copytree(METADATA, output / "metadata/en-US")
        shutil.copy2(icon, output / "icon.png")
        for name in ("README.md", "data-safety-worksheet.md", "assets-and-review.md", "acceptance.csv",
                     "account-deletion-operations.md", "release-readiness.json"):
            shutil.copy2(ROOT / "play-store" / name, output / name)
        manifest = {
            "package": "com.mediguide.ug",
            "status": "PREPARATION_ONLY_NOT_SUBMISSION_APPROVAL",
            "missing": ["signed verified AAB", "feature graphic", "actual screenshots",
                        "published legal URLs", "completed console declarations",
                        "functional account deletion", "device acceptance evidence"],
            "sha256": {str(p.relative_to(output)): hashlib.sha256(p.read_bytes()).hexdigest()
                       for p in sorted(output.rglob("*")) if p.is_file()},
        }
        (output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        archive = ROOT / "build/mediguide-play-store-preparation.zip"
        with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as bundle:
            for path in sorted(output.rglob("*")):
                if path.is_file():
                    bundle.write(path, path.relative_to(output))
        print(f"Exported {archive}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
