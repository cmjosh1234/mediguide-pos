"""Install AI requirements only when they change for the current interpreter."""

import hashlib
from pathlib import Path
import subprocess
import sys


def install(requirements: Path, stamp: Path) -> None:
    requirements = requirements.resolve()
    stamp = stamp.resolve()
    # A different virtualenv/interpreter must not reuse another one's stamp.
    identity = f"{sys.executable}\n{sys.prefix}\n{sys.version}\n".encode()
    fingerprint = hashlib.sha256(identity + requirements.read_bytes()).hexdigest()
    if stamp.is_file() and stamp.read_text().strip() == fingerprint:
        print("ai-worker requirements unchanged")
        return

    print("Installing ai-worker requirements", flush=True)
    subprocess.run(
        [sys.executable, "-m", "pip", "install", "-r", requirements.name],
        cwd=requirements.parent,
        check=True,
    )
    # Do not mark a failed install as successful.
    stamp.write_text(fingerprint)


if __name__ == "__main__":
    try:
        install(Path(sys.argv[1]), Path(sys.argv[2]))
    except subprocess.CalledProcessError as error:
        sys.exit(error.returncode)
