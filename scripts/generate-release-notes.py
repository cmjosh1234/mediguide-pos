#!/usr/bin/env python3
"""Generate reviewable, versioned service notes and store changelogs."""
import argparse
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
SERVICES = {
    "mobile": ("Mobile application", "user_app/"),
    "api": ("API", "backend/"),
    "ai-worker": ("AI worker", "ai-worker/"),
    "dashboard": ("Dashboard", "dashboard/"),
    "guidelines": ("Public guidelines portal", "guidelines-platform/"),
}


def git(root, *args):
    return subprocess.check_output(["git", "-C", str(root), *args], text=True).strip()


def version_parts(tag):
    match = re.fullmatch(r"v(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z.-]+)?", tag)
    if not match:
        raise ValueError("Release notes require a vMAJOR.MINOR.PATCH tag")
    return tuple(map(int, match.groups()))


def mobile_build(root, tag):
    exists = subprocess.run(
        ["git", "-C", str(root), "rev-parse", "--verify", "--quiet", "refs/tags/" + tag],
        capture_output=True,
    ).returncode == 0
    text = git(root, "show", tag + ":user_app/pubspec.yaml") if exists else (root / "user_app/pubspec.yaml").read_text()
    match = re.search(r"^version:\s*(\S+)\+([1-9]\d*)$", text, re.MULTILINE)
    if not match or match[1] != tag[1:]:
        raise ValueError("Release notes tag does not match the mobile version")
    return int(match[2])


def draft(root, tag, build):
    target = tag if tag in git(root, "tag", "--list").splitlines() else "HEAD"
    tags = [t for t in git(root, "tag", "--merged", target).splitlines()
            if re.fullmatch(r"v\d+\.\d+\.\d+", t) and version_parts(t) < version_parts(tag)]
    if not tags:
        raise ValueError("No previous stable tag found; create notes.json with an explicit baseline")
    previous = max(tags, key=version_parts)
    changes = {name: [] for name in SERVICES}
    operations = []
    for line in git(root, "log", "--reverse", "--no-merges", "--format=%H%x09%s", previous + ".." + target).splitlines():
        sha, subject = line.split("\t", 1)
        if re.search(r"(?:chore\(release\)|release: prepare|prepare v\d)", subject, re.I):
            continue
        paths = git(root, "diff-tree", "--no-commit-id", "--name-only", "-r", sha).splitlines()
        for name, (_, prefix) in SERVICES.items():
            if any(p.startswith(prefix) for p in paths) and subject not in changes[name]:
                changes[name].append(subject)
        if any(p.startswith(("infra/", ".github/", "scripts/")) for p in paths) and subject not in operations:
            operations.append(subject)
    for name in changes:
        if not changes[name]:
            changes[name] = ["No service-specific changes since " + previous + "."]
    store = []
    for item in changes["mobile"]:
        candidate = "\n".join(store + ["- " + item])
        if len(candidate) > 500:
            break
        store.append("- " + item)
    if not store:
        store = ["MediGuide " + tag[1:] + ": mobile updates. Review mobile.md for the full change list."]
    return {"schema_version": 1, "release_tag": tag, "previous_tag": previous,
            "mobile_build_number": build, "services": changes,
            "operations": operations, "play_store": "\n".join(store)}


def validate(notes, tag, build):
    if notes.get("schema_version") != 1 or notes.get("release_tag") != tag:
        raise ValueError("notes.json schema or release tag is invalid")
    if notes.get("mobile_build_number") != build:
        raise ValueError("notes.json mobile build number does not match the release")
    if version_parts(notes.get("previous_tag", "")) >= version_parts(tag):
        raise ValueError("Release notes baseline must precede the release")
    services = notes.get("services", {})
    if set(services) != set(SERVICES):
        raise ValueError("Release notes must cover all five services")
    for name, entries in {**services, "operations": notes.get("operations", [])}.items():
        if not isinstance(entries, list) or (name != "operations" and not entries):
            raise ValueError("Missing release notes for " + name)
        if any(not isinstance(s, str) or not s.strip() or "\n" in s for s in entries):
            raise ValueError("Each release note must be a nonempty single-line string")
    store = notes.get("play_store")
    if not isinstance(store, str) or not store.strip() or len(store) > 500:
        raise ValueError("Google Play notes must contain 1–500 characters")


def render(notes):
    tag = notes["release_tag"]
    build = notes["mobile_build_number"]
    header = f"# MediGuide {tag}\n\nChanges since {notes['previous_tag']}. Mobile version: {tag[1:]}+{build}.\n"
    outputs = {}
    combined = header
    for name, (label, _) in SERVICES.items():
        section = f"\n## {label}\n\n" + "\n".join("- " + item for item in notes["services"][name]) + "\n"
        outputs[name + ".md"] = header + section
        combined += section
    if notes["operations"]:
        combined += "\n## Deployment and operations\n\n" + "\n".join("- " + item for item in notes["operations"]) + "\n"
    outputs["release-notes.md"] = combined
    outputs["play-store.txt"] = notes["play_store"].strip() + "\n"
    return outputs


def generate(root, tag, check=False):
    version_parts(tag)
    directory = root / "docs/releases" / tag
    source = directory / "notes.json"
    build = mobile_build(root, tag)
    if not source.exists():
        if check:
            raise ValueError("Missing release notes source: " + str(source))
        notes = draft(root, tag, build)
        directory.mkdir(parents=True, exist_ok=True)
        source.write_text(json.dumps(notes, indent=2) + "\n")
    notes = json.loads(source.read_text())
    validate(notes, tag, build)
    outputs = {directory / name: contents for name, contents in render(notes).items()}
    outputs[root / f"user_app/fastlane/metadata/android/en-US/changelogs/{build}.txt"] = notes["play_store"].strip() + "\n"
    for path, contents in outputs.items():
        if check:
            if not path.exists() or path.read_text() != contents:
                raise ValueError("Missing or stale release notes: " + str(path))
        else:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents)
    print(f"Release notes {'verified' if check else 'generated'} for {tag}: all five services / Play build {build}.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag")
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    try:
        generate(ROOT, args.tag, args.check)
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        parser.exit(1, str(error) + "\n")


if __name__ == "__main__":
    main()
