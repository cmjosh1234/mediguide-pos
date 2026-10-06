import contextlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("release_notes", ROOT / "scripts/generate-release-notes.py")
notes = importlib.util.module_from_spec(spec)
spec.loader.exec_module(notes)


class ReleaseNotesTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.git("init", "-q")
        self.git("config", "user.name", "Release test")
        self.git("config", "user.email", "release@example.org")
        self.write("user_app/pubspec.yaml", "version: 1.0.0+10\n")
        self.commit("Initial release")
        self.git("tag", "v1.0.0")
        self.write("backend/public.go", "public tools\n")
        self.write("user_app/tools.dart", "public tools\n")
        self.commit("Allow public tools access")
        self.write("dashboard/runtime.txt", "updated\n")
        self.commit("Fix dashboard runtime")
        self.write("user_app/pubspec.yaml", "version: 1.0.1+11\n")

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.root), *args], stderr=subprocess.STDOUT, text=True)

    def write(self, path, value):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(value)

    def commit(self, message):
        self.git("add", ".")
        self.git("commit", "-qm", message)

    def generate(self, check=False):
        with contextlib.redirect_stdout(io.StringIO()):
            notes.generate(self.root, "v1.0.1", check)

    def test_changes_are_scoped_and_unchanged_services_are_explicit(self):
        self.generate()
        source = json.loads((self.root / "docs/releases/v1.0.1/notes.json").read_text())
        self.assertEqual(["Allow public tools access"], source["services"]["api"])
        self.assertEqual(["Allow public tools access"], source["services"]["mobile"])
        self.assertEqual(["Fix dashboard runtime"], source["services"]["dashboard"])
        self.assertIn("No service-specific changes", source["services"]["ai-worker"][0])
        self.assertTrue((self.root / "user_app/fastlane/metadata/android/en-US/changelogs/11.txt").exists())
        self.generate(check=True)

    def test_check_rejects_missing_or_stale_notes(self):
        with self.assertRaisesRegex(ValueError, "Missing release notes source"):
            self.generate(check=True)
        self.generate()
        self.write("docs/releases/v1.0.1/mobile.md", "stale")
        with self.assertRaisesRegex(ValueError, "stale release notes"):
            self.generate(check=True)

    def test_reviewed_source_is_preserved_and_outputs_can_be_regenerated(self):
        self.generate()
        path = self.root / "docs/releases/v1.0.1/notes.json"
        source = json.loads(path.read_text())
        source["play_store"] = "Use clinical tools without signing in."
        path.write_text(json.dumps(source))
        with self.assertRaises(ValueError):
            self.generate(check=True)
        self.generate()
        self.assertEqual(source, json.loads(path.read_text()))
        self.assertEqual(source["play_store"] + "\n", (self.root / "user_app/fastlane/metadata/android/en-US/changelogs/11.txt").read_text())

    def test_store_limits_missing_service_and_wrong_build_block_release(self):
        source = notes.draft(self.root, "v1.0.1", 11)
        source["play_store"] = "x" * 501
        with self.assertRaisesRegex(ValueError, "1–500"):
            notes.validate(source, "v1.0.1", 11)
        source["play_store"] = "Updates"
        with self.assertRaisesRegex(ValueError, "build number"):
            notes.validate(source, "v1.0.1", 12)
        del source["services"]["api"]
        with self.assertRaisesRegex(ValueError, "five services"):
            notes.validate(source, "v1.0.1", 11)

    def test_backfill_uses_tagged_build_after_main_version_changes(self):
        self.commit("chore(release): prepare v1.0.1")
        self.git("tag", "v1.0.1")
        self.write("user_app/pubspec.yaml", "version: 1.0.2+12\n")
        self.generate()
        self.generate(check=True)
        self.assertTrue((self.root / "user_app/fastlane/metadata/android/en-US/changelogs/11.txt").exists())


if __name__ == "__main__":
    unittest.main()
