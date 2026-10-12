import importlib.util
from pathlib import Path
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("android_abis", ROOT / "scripts/check-android-abis.py")
abis = importlib.util.module_from_spec(spec)
spec.loader.exec_module(abis)


def elf(abi):
    header = bytearray(20)
    header[:4] = b"\x7fELF"
    header[4], machine = abis.ABIS[abi]
    header[5] = 1
    header[18:20] = machine.to_bytes(2, "little")
    return header


class AndroidAbiTest(unittest.TestCase):
    def archive(self, root, suffix, entries):
        path = Path(root) / ("release" + suffix)
        prefix = "base/" if suffix == ".aab" else ""
        with zipfile.ZipFile(path, "w") as archive:
            for abi, name, actual_abi in entries:
                archive.writestr(f"{prefix}lib/{abi}/{name}", elf(actual_abi))
        return path

    def test_valid_split_apks_and_complete_bundle(self):
        with tempfile.TemporaryDirectory() as root:
            for abi in abis.ABIS:
                entries = [(abi, name, abi) for name in ("libflutter.so", "libapp.so")]
                abis.validate(self.archive(root, ".apk", entries), {abi})
            entries = [(abi, name, abi) for abi in abis.ABIS for name in ("libflutter.so", "libapp.so")]
            abis.validate(self.archive(root, ".aab", entries), set(abis.ABIS))

    def test_arm64_library_in_x64_directory_is_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            path = self.archive(root, ".apk", [("x86_64", "libflutter.so", "arm64-v8a")])
            with self.assertRaisesRegex(ValueError, "wrong ELF architecture"):
                abis.validate(path, {"x86_64"})

    def test_plugin_only_abi_cannot_advertise_flutter_support(self):
        with tempfile.TemporaryDirectory() as root:
            entries = [("arm64-v8a", name, "arm64-v8a") for name in ("libflutter.so", "libapp.so")]
            entries.append(("x86_64", "libplugin.so", "x86_64"))
            with self.assertRaisesRegex(ValueError, "x86_64 is missing"):
                abis.validate(self.archive(root, ".aab", entries), {"arm64-v8a", "x86_64"})

    def test_missing_release_target_is_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            entries = [("arm64-v8a", name, "arm64-v8a") for name in ("libflutter.so", "libapp.so")]
            with self.assertRaisesRegex(ValueError, "expected ABIs"):
                abis.validate(self.archive(root, ".aab", entries), set(abis.ABIS))


if __name__ == "__main__":
    unittest.main()
