import base64
import importlib.util
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


def load(name):
    spec = importlib.util.spec_from_file_location(name, ROOT / "scripts" / (name + ".py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


changes = load("ci-changes")
credentials = load("ci-mobile-preflight")


class ChangesTest(unittest.TestCase):
    def test_docs_only_skips_apps(self):
        self.assertFalse(any(changes.classify(["docs/readme.md", "README.md"]).values()))

    def test_worker_only(self):
        self.assertEqual(["worker"], [key for key, value in changes.classify(["ai-worker/app/worker.py"]).items() if value])

    def test_backend_invalidates_consumers(self):
        result = changes.classify(["backend/internal/api/handlers.go"])
        for name in ("backend", "dashboard", "guidelines", "mobile"):
            self.assertTrue(result[name])

    def test_legacy_assets_invalidate_backend_image(self):
        self.assertTrue(changes.classify(["dashboard/samples/tool.html"])["backend"])
        self.assertTrue(changes.classify(["clinical-tools/tool.json"])["backend"])

    def test_shared_policy_checks_everything(self):
        for path in ("scripts/ci-changes.py", "proto/service.proto", ".github/workflows/mobile-release.yml", "VERSION", ".dockerignore", "Makefile", ".gitattributes"):
            self.assertTrue(all(changes.classify([path]).values()))

    def test_full_release_checks_everything(self):
        self.assertTrue(all(changes.classify([], full=True).values()))

    def test_component_scopes(self):
        for prefix, name in (("user_app", "mobile"), ("dashboard", "dashboard"), ("guidelines-platform", "guidelines"), ("infra", "infra")):
            self.assertTrue(changes.classify([prefix + "/file"])[name])


class CredentialsTest(unittest.TestCase):
    def android_env(self):
        env = {key: "test" for key in credentials.required_fields("firebase-android")}
        env["ANDROID_UPLOAD_KEYSTORE_BASE64"] = base64.b64encode(b"test-keystore").decode()
        account = {"project_id": "test", "client_email": "test@example.org", "private_key": "test"}
        env["FIREBASE_APP_DISTRIBUTION_SERVICE_ACCOUNT_BASE64"] = base64.b64encode(json.dumps(account).encode()).decode()
        env["FIREBASE_MOBILE_CONFIG_JSON"] = json.dumps({"MEDIGUIDE_FLAVOR": "staging", **{key: "test" for key in ("FIREBASE_PROJECT_ID", "FIREBASE_API_KEY", "FIREBASE_MESSAGING_SENDER_ID", "FIREBASE_ANDROID_APP_ID")}})
        return env

    def test_android_needs_no_apple_credentials(self):
        self.assertEqual([], credentials.validate("firebase-android", self.android_env()))

    def test_missing_secrets_only_report_names(self):
        self.assertIn("ANDROID_UPLOAD_KEYSTORE_BASE64", credentials.validate("firebase-android", {}))

    def test_invalid_base64_rejected(self):
        env = self.android_env()
        env["ANDROID_UPLOAD_KEYSTORE_BASE64"] = "not base64"
        self.assertEqual(["ANDROID_UPLOAD_KEYSTORE_BASE64"], credentials.validate("firebase-android", env))

    def test_wrong_flavor_rejected(self):
        env = self.android_env()
        env["MOBILE_FLAVOR"] = "production"
        self.assertIn("FIREBASE_MOBILE_CONFIG_JSON", credentials.validate("firebase-android", env))

    def test_destination_specific_apple_requirements(self):
        flight = credentials.required_fields("testflight")
        self.assertIn("APP_STORE_CONNECT_PRIVATE_KEY_BASE64", flight)
        self.assertNotIn("IOS_FIREBASE_PROVISIONING_PROFILE_BASE64", flight)
        self.assertNotIn("FIREBASE_APP_DISTRIBUTION_SERVICE_ACCOUNT_BASE64", flight)
        self.assertIn("IOS_APP_STORE_PROVISIONING_PROFILE_BASE64", credentials.required_fields("app-store"))

    def test_unknown_destination_fails(self):
        with self.assertRaises(ValueError):
            credentials.required_fields("unknown")

    def test_production_all_is_not_tester_all(self):
        required = credentials.required_fields("production-all")
        self.assertIn("GOOGLE_PLAY_SERVICE_ACCOUNT_BASE64", required)
        self.assertIn("IOS_APP_STORE_PROVISIONING_PROFILE_BASE64", required)
        self.assertNotIn("IOS_FIREBASE_PROVISIONING_PROFILE_BASE64", required)
        self.assertNotIn("IOS_TESTFLIGHT_PROVISIONING_PROFILE_BASE64", required)

    def test_malformed_config_json_is_rejected(self):
        env = self.android_env()
        env["FIREBASE_MOBILE_CONFIG_JSON"] = "not json"
        self.assertEqual(["FIREBASE_MOBILE_CONFIG_JSON"], credentials.validate("firebase-android", env))

    def test_malformed_export_plist_is_rejected(self):
        env = {key: "test" for key in credentials.required_fields("testflight")}
        env["IOS_TESTFLIGHT_EXPORT_OPTIONS_PLIST_BASE64"] = base64.b64encode(b"not a plist").decode()
        self.assertIn("IOS_TESTFLIGHT_EXPORT_OPTIONS_PLIST_BASE64", credentials.validate("testflight", env))


class WorkflowPolicyTest(unittest.TestCase):
    def workflow(self, name):
        return (ROOT / ".github/workflows" / (name + ".yml")).read_text()

    def test_pr_checks_do_not_use_trigger_path_filters(self):
        for name in ("container-images", "mobile-release"):
            self.assertNotIn("    paths:", self.workflow(name))
            self.assertIn("    if: always()", self.workflow(name))

    def test_images_are_canonical_only(self):
        workflow = self.workflow("container-images")
        build = workflow.split("  build:\n")[1].split("  release-images:\n")[0]
        self.assertIn("github.repository == 'mohuganda/mediguide-pos'", build)
        self.assertIn("github.event_name == 'push'", build)
        self.assertNotIn("github.event_name == 'pull_request'", build)
        self.assertIn("needs.changes.outputs.build_images == 'true'", build)

    def test_default_mobile_destination_is_android(self):
        for name in ("mobile-alpha", "mobile-beta", "mobile-distribution"):
            self.assertIn("default: firebase-android", self.workflow(name))
            self.assertNotIn("default: all", self.workflow(name))

    def test_tags_do_not_override_distribution_destination(self):
        workflow = self.workflow("mobile-distribution")
        self.assertNotIn("startsWith(github.ref, 'refs/tags/v') ||", workflow)
        self.assertIn("    needs: preflight", workflow)

    def test_optional_targets_do_not_block_tag_release(self):
        workflow = self.workflow("mobile-release")
        release = workflow.split("  release:\n")[1]
        self.assertIn("    needs: android", release)
        self.assertNotIn("      - apple", release)
        self.assertNotIn('test -n "${ios}"', release)
        self.assertIn("inputs.build_unsigned_ios", workflow)
        self.assertIn("inputs.build_macos", workflow)
        self.assertIn("inputs.build_web", workflow)

    def test_critical_workflows_are_not_auto_cancelled(self):
        for name in ("deploy-production", "mobile-alpha", "mobile-distribution", "mobile-production"):
            self.assertIn("cancel-in-progress: false", self.workflow(name))

    def test_production_workflow_supports_android_firebase_distribution(self):
        workflow = self.workflow("mobile-production")
        self.assertIn("          - firebase-android", workflow)
        self.assertIn("uses: ./.github/workflows/mobile-distribution.yml", workflow)
        self.assertIn("      flavor: production", workflow)
        self.assertIn("      github_environment: production", workflow)
        self.assertIn("      source_ref: ${{ inputs.release_tag }}", workflow)

    def test_no_overlapping_gradle_caches(self):
        for name in ("mobile-release", "mobile-distribution", "mobile-production"):
            self.assertIn("gradle/actions/setup-gradle@v6", self.workflow(name))
            self.assertNotIn("cache: gradle", self.workflow(name))


if __name__ == "__main__":
    unittest.main()
