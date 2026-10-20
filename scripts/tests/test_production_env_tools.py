import ast
import io
import importlib.util
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[2]


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / "infra" / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


organizer = load("organize_env", "organize-env.py")
checker = load("check_public_storage", "check-public-storage.py")
production = load("check_production_env", "check-production-env.py")


class ProductionEnvironmentToolsTest(unittest.TestCase):
    def test_production_requires_complete_mapping_and_mail_credentials(self):
        source = (ROOT / "infra/production.env.example").read_bytes()
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "production.env"
            path.write_bytes(source.replace(b"RESEND_API_KEY=\n", b"RESEND_API_KEY=private-fixture\n"))
            production.validate(path, ROOT / "infra/docker-compose.yml")
            path.write_bytes(path.read_bytes().replace(b"RESEND_API_KEY=private-fixture", b"RESEND_API_KEY="))
            with self.assertRaisesRegex(ValueError, "RESEND_API_KEY"):
                production.validate(path, ROOT / "infra/docker-compose.yml")
            path.write_bytes(path.read_bytes().replace(b"CACHE_ENABLED=true\n", b""))
            with self.assertRaisesRegex(ValueError, "CACHE_ENABLED"):
                production.validate(path, ROOT / "infra/docker-compose.yml")

    def test_runtime_environment_comparison_reports_names_only(self):
        config = {"services": {"api": {"environment": {"JWT_SECRET": "private-fixture", "HTTP_PORT": "8080"}}}}
        containers = [{"Config": {"Labels": {"com.docker.compose.service": "api"}, "Env": ["JWT_SECRET=private-fixture", "HTTP_PORT=8080", "PATH=/bin"]}}]
        self.assertEqual(production.compare(config, containers), 2)
        containers[0]["Config"]["Env"][0] = "JWT_SECRET=wrong-private-fixture"
        with self.assertRaises(ValueError) as context:
            production.compare(config, containers)
        self.assertIn("JWT_SECRET", str(context.exception))
        self.assertNotIn("private-fixture", str(context.exception))
        with self.assertRaisesRegex(ValueError, "container missing"):
            production.compare(config, [])

    def test_fills_missing_keys_without_overwriting_existing_credentials_or_blanks(self):
        source = b'JWT_SECRET="private $value=#"\nSMTP_PASSWORD=\n'
        defaults = b'JWT_SECRET=template-placeholder\nSMTP_PASSWORD=template-placeholder\nACCOUNT_ACTION_URL=https://example.test/admin\n'
        result = organizer.assignments(organizer.organize(source, defaults))
        self.assertEqual(result["JWT_SECRET"], b'"private $value=#"')
        self.assertEqual(result["SMTP_PASSWORD"], b"")
        self.assertEqual(result["ACCOUNT_ACTION_URL"], b"https://example.test/admin")

    def test_backend_template_covers_runtime_configuration(self):
        source = (ROOT / "backend/internal/config/config.go").read_text()
        keys = organizer.assignments((ROOT / "backend/.env.example").read_bytes())
        for key in re.findall(r'\bget(?:Int|Bool|CSV|CSVWithFallback)?\("([A-Z][A-Z0-9_]*)"', source):
            self.assertIn(key, keys, f"Backend template is missing {key}")
        for aliases in re.findall(r'get(?:Int)?Any\(\[\]string\{([^}]+)\}', source):
            self.assertTrue(set(re.findall(r'"([A-Z][A-Z0-9_]*)"', aliases)) & keys.keys())

    def test_worker_template_covers_pydantic_settings(self):
        source = ast.parse((ROOT / "ai-worker/app/core/config.py").read_text())
        settings = next(node for node in source.body if isinstance(node, ast.ClassDef) and node.name == "Settings")
        keys = organizer.assignments((ROOT / "ai-worker/.env.example").read_bytes())
        for field in settings.body:
            if not isinstance(field, ast.AnnAssign) or field.target.id == "model_config":
                continue
            choices = {field.target.id.upper()}
            if isinstance(field.value, ast.Call):
                alias = next((arg.value for arg in field.value.keywords if arg.arg == "validation_alias"), None)
                if isinstance(alias, ast.Constant):
                    choices = {alias.value}
                elif isinstance(alias, ast.Call):
                    choices = {arg.value for arg in alias.args}
            self.assertTrue(choices & keys.keys(), f"Worker template is missing one of {sorted(choices)}")

    def test_preserves_opaque_credential_bytes_and_effective_duplicate_values(self):
        source = b'JWT_SECRET="opaque $value=#=\\n"\nSMTP_PASSWORD=opaque=credential\nS3_PUBLIC_SSL=false\nS3_PUBLIC_SSL=true\n'
        result = organizer.organize(source)
        self.assertEqual(organizer.assignments(source), organizer.assignments(result))
        self.assertEqual(sum(line.startswith(b"S3_PUBLIC_SSL=") for line in result.splitlines()), 1)
        self.assertIn(b'JWT_SECRET="opaque $value=#=\\n"', result)
        self.assertEqual(organizer.organize(result), result)

    def test_rejects_unsupported_syntax_without_disclosing_its_contents(self):
        with self.assertRaises(ValueError) as context:
            organizer.organize(b"not-an-assignment confidential-fixture\n")
        self.assertNotIn("confidential-fixture", str(context.exception))

    def check(self, endpoint, ssl="true"):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "production.env"
            path.write_text(f"SMTP_PASSWORD=opaque-fixture\nS3_PUBLIC_ENDPOINT={endpoint}\nS3_PUBLIC_SSL={ssl}\n")
            return checker.storage_url(path)

    def test_deployment_requires_local_storage_health_without_public_network_probe(self):
        script = (ROOT / "infra/deploy-production.sh").read_text()
        bootstrap = script.index('up --no-build -d --no-deps --wait --wait-timeout 120 minio')
        replacement = script.index('down --remove-orphans')
        self.assertLess(script.index('config --quiet'), bootstrap)
        self.assertLess(script.index('Using immutable release images preloaded'), bootstrap)
        self.assertLess(bootstrap, replacement)
        self.assertNotIn('--check-network', script)
        self.assertIn('python3 "${infra_dir}/check-public-storage.py" "${production_env}"', script)

    def test_legacy_release_filter_preserves_every_other_deployment_check(self):
        workflow = (ROOT / ".github/workflows/deploy-production.yml").read_text()
        start = workflow.index('# Older immutable tags still contain the retired public HTTPS probe.')
        program = re.search(r"awk '\n(.*?)\n\s*'", workflow[start:], re.DOTALL).group(1)
        probe = 'python3 "${infra_dir}/check-public-storage.py" "${production_env}" --check-network\n'
        before = 'python3 "${infra_dir}/check-public-storage.py" "${production_env}"\n'
        after = '"${compose[@]}" up --no-build -d --no-deps --wait --wait-timeout 120 minio\nverify_public_route "API" "${public_api_base_url%/}/api/readyz"\n'
        result = subprocess.run(['awk', program], input=before + probe + after, text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, before + after)
        # Fail closed if the legacy script unexpectedly contains duplicate probes.
        result = subprocess.run(['awk', program], input=before + probe + probe + after, text=True, capture_output=True)
        self.assertNotEqual(result.returncode, 0)

    def test_preflight_and_workflow_no_longer_require_public_probe_override(self):
        preflight = (ROOT / ".github/workflows/production-preflight.yml").read_text()
        workflow = (ROOT / ".github/workflows/deploy-production.yml").read_text()
        self.assertNotIn('--check-network', preflight)
        self.assertIn('python3 infra/check-public-storage.py infra/production.env', preflight)
        self.assertNotIn('skip_public_storage_check', workflow)
        self.assertNotIn('SKIP_PUBLIC_STORAGE_CHECK', workflow)

    def test_validates_https_browser_facing_storage(self):
        self.assertEqual(self.check("assets.example.org"), "https://assets.example.org")
        self.assertEqual(self.check("assets.example.org:9443"), "https://assets.example.org:9443")

    def test_health_rejects_spa_fallback_errors_and_redirects(self):
        base = "https://mediguide.example.org"
        for status, url, body in [(200, base + "/minio/health/live", b"<html>website</html>"),
                                  (502, base + "/minio/health/live", b""),
                                  (200, base + "/", b""),
                                  (200, "https://other.example.org/minio/health/live", b"")]:
            response = SimpleNamespace(status=status, url=url, read=io.BytesIO(body).read)
            with self.subTest(status=status, url=url), self.assertRaises(ValueError):
                checker.validate_health_response(base, response)
        response = SimpleNamespace(status=200, url=base + "/minio/health/live", read=io.BytesIO(b"").read)
        checker.validate_health_response(base, response)

    def test_rejects_internal_endpoints_path_prefixes_and_credentials(self):
        for endpoint in ["", "minio:9000", "localhost:9000", "127.0.0.1:9000", "10.1.2.3:9000", "storage.internal", "https://assets.example.org", "assets.example.org/storage", "name:opaque-fixture@assets.example.org", "assets.example.org?token=opaque-fixture"]:
            with self.subTest(endpoint=endpoint):
                with self.assertRaises(ValueError) as context:
                    self.check(endpoint)
                self.assertNotIn("opaque-fixture", str(context.exception))
        with self.assertRaises(ValueError):
            self.check("assets.example.org", "false")


if __name__ == "__main__":
    unittest.main()
