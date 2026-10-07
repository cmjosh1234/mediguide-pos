import importlib.util
from pathlib import Path
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
