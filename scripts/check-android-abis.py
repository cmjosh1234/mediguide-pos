#!/usr/bin/env python3
"""Reject APK/AAB archives with incomplete or mislabeled Flutter native ABIs."""
import argparse
from pathlib import Path
import re
import zipfile

# ELF class and e_machine for Flutter's supported Android release targets.
ABIS = {"armeabi-v7a": (1, 40), "arm64-v8a": (2, 183), "x86_64": (2, 62)}


def validate(path: Path, expected: set[str]) -> None:
    libraries: dict[str, set[str]] = {}
    with zipfile.ZipFile(path) as archive:
        prefix = "base/" if path.suffix == ".aab" else ""
        for entry in archive.infolist():
            match = re.fullmatch(re.escape(prefix) + r"lib/([^/]+)/([^/]+\.so)", entry.filename)
            if not match:
                continue
            abi, name = match.groups()
            if abi not in ABIS:
                raise ValueError(f"{path.name}: unsupported native ABI {abi}")
            with archive.open(entry) as stream:
                header = stream.read(20)
            elf_class, machine = ABIS[abi]
            if (len(header) < 20 or header[:4] != b"\x7fELF"
                    or header[4] != elf_class or header[5] != 1
                    or int.from_bytes(header[18:20], "little") != machine):
                raise ValueError(f"{path.name}: {entry.filename} has the wrong ELF architecture")
            libraries.setdefault(abi, set()).add(name)
        if set(libraries) != expected:
            raise ValueError(f"{path.name}: expected ABIs {sorted(expected)}, found {sorted(libraries)}")
        for abi, names in libraries.items():
            missing = {"libflutter.so", "libapp.so"} - names
            if missing:
                raise ValueError(f"{path.name}: {abi} is missing {sorted(missing)}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--expected", required=True, help="Comma-separated native ABI names")
    parser.add_argument("artifacts", nargs="+", type=Path)
    args = parser.parse_args()
    expected = set(args.expected.split(","))
    if not expected or not expected <= ABIS.keys():
        parser.error("Expected ABIs must be supported Flutter release architectures")
    for path in args.artifacts:
        try:
            validate(path, expected)
        except (ValueError, OSError, zipfile.BadZipFile) as error:
            parser.exit(1, f"{error}\n")
        print(f"{path.name}: native ABI validation passed ({', '.join(sorted(expected))})")


if __name__ == "__main__":
    main()
