import ast
import importlib.util
from pathlib import Path
import re
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / "infra" / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


organizer = load("organize_env", "organize-env.py")
checker = load("check_public_storage", "check-public-storage.py")


class ProductionEnvironmentToolsTest(unittest.TestCase):
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

    def test_validates_https_browser_facing_storage(self):
        self.assertEqual(self.check("assets.example.org"), "https://assets.example.org")
        self.assertEqual(self.check("assets.example.org:9443"), "https://assets.example.org:9443")

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
