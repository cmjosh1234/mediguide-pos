# Release notes

`make release-prepare`, `make release-patch`, `make release-minor` and
`make release-major` create notes alongside the synchronized versions.

The generator groups commits since the previous stable tag by changed service
paths. It includes mobile, API, AI worker, dashboard and public guidelines;
services with no changes receive an explicit entry. Git subjects are a starting
draft: review them and rewrite the notes for users before tagging.

Each release has one editable source: `docs/releases/vX.Y.Z/notes.json`.
Edit its service changes, operational notes and `play_store` text, then run:

```bash
make release-notes RELEASE_TAG=vX.Y.Z
```

This renders the combined `release-notes.md`, five service Markdown files,
`play-store.txt`, and the matching Android version-code changelog in
`user_app/fastlane/metadata/android/en-US/changelogs/BUILD.txt`. Play text must
be at most 500 characters. The generator preserves existing source files so
rerunning release preparation keeps reviewed wording.

Commit the source and generated files with the version changes. Release
preflight checks the service coverage, version/build and generated output.
Missing or stale notes block CI and tagging.

The tagged release attaches all notes and includes the combined notes in its
GitHub body. Google Play delivery uploads the version-specific changelog.
Production Firebase and TestFlight delivery use mobile notes when no explicit
delivery notes are supplied. Explicit tester notes still take precedence.

For a published release whose Play draft already exists, update notes without
rebuilding or reuploading its binary:

```bash
gh workflow run play-release-notes.yml --repo mohuganda/mediguide-pos \
  --ref main -f release_tag=vX.Y.Z -f confirmation=vX.Y.Z
```

This uses the protected production credentials and changes the existing
production-track draft changelog. It does not start rollout.
