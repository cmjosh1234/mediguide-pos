"""Read-only Make smoke checks and dependency installer regression coverage."""

import importlib.util
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location(
    "install_ai_deps", ROOT / "scripts" / "install-ai-deps.py"
)
installer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(installer)


class DependencyInstallTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(prefix="ai deps ")
        self.addCleanup(self.directory.cleanup)
        self.requirements = Path(self.directory.name) / "requirements.txt"
        self.requirements.write_text("example==1.0\n")
        self.stamp = Path(self.directory.name) / ".requirements.sha256"

    @patch.object(installer.subprocess, "run")
    def test_install_cache_and_changed_requirements(self, run):
        installer.install(self.requirements, self.stamp)
        run.assert_called_once_with(
            [sys.executable, "-m", "pip", "install", "-r", "requirements.txt"],
            cwd=self.requirements.resolve().parent,
            check=True,
        )
        installer.install(self.requirements, self.stamp)
        self.assertEqual(run.call_count, 1)
        self.requirements.write_text("example==2.0\n")
        installer.install(self.requirements, self.stamp)
        self.assertEqual(run.call_count, 2)

    @patch.object(installer.subprocess, "run")
    def test_failed_install_does_not_update_stamp(self, run):
        self.stamp.write_text("previous-install")
        run.side_effect = subprocess.CalledProcessError(1, "pip")
        with self.assertRaises(subprocess.CalledProcessError):
            installer.install(self.requirements, self.stamp)
        self.assertEqual(self.stamp.read_text(), "previous-install")

    @patch.object(installer.subprocess, "run")
    def test_new_interpreter_reinstalls(self, run):
        installer.install(self.requirements, self.stamp)
        with patch.object(installer.sys, "executable", "another-python"):
            installer.install(self.requirements, self.stamp)
        self.assertEqual(run.call_count, 2)


class MakeSmokeTest(unittest.TestCase):
    def make(self, *arguments, success=True):
        result = subprocess.run(
            ["make", "--no-print-directory", *arguments],
            cwd=ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        )
        self.assertEqual(result.returncode == 0, success, result.stdout)
        return result.stdout

    def test_help(self):
        self.assertIn("Available targets:", self.make("help"))

    def test_environment_check_uses_working_bash(self):
        self.assertIn("schemas are aligned", self.make("env-check"))

    def test_backend_build_metadata_and_extension(self):
        output = self.make("-n", "backend-build")
        version = (ROOT / "VERSION").read_text().strip()
        self.assertIn(f"Version={version}", output)
        extension = ".exe" if os.name == "nt" else ""
        self.assertIn(f"bin/api{extension} ./cmd/api", output)
        self.assertIn(f"bin/worker{extension} ./cmd/worker", output)

    def test_missing_production_env_stops_before_docker(self):
        with tempfile.TemporaryDirectory() as directory:
            missing = Path(directory) / "missing.env"
            output = self.make("prod-up", f"PRODUCTION_ENV_FILE={missing.as_posix()}", success=False)
        self.assertIn("Missing", output)
        self.assertNotIn("docker compose", output)

    def test_production_env_path_with_spaces(self):
        with tempfile.TemporaryDirectory(prefix="make env ") as directory:
            env = Path(directory) / "production.env"
            env.touch()
            self.make("production-env-check", f"PRODUCTION_ENV_FILE={env.as_posix()}")

    def test_missing_actor_stops_before_import(self):
        output = self.make("clinical-tools-import", "ACTOR_ID=", success=False)
        self.assertIn("ACTOR_ID is required", output)
        self.assertNotIn("go run", output)


if __name__ == "__main__":
    unittest.main()
