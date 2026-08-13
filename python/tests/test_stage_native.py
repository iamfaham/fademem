import subprocess
import sys
from pathlib import Path


def test_stage_native_copies_shared_library_into_package_resources(tmp_path: Path) -> None:
    source = tmp_path / "recolva.dll"
    target_directory = tmp_path / "package" / "recolva" / "_native"
    source.write_bytes(b"native-library")
    script = Path(__file__).parents[2] / "scripts" / "stage_native.py"

    subprocess.run(
        [
            sys.executable,
            str(script),
            "--source",
            str(source),
            "--target-directory",
            str(target_directory),
        ],
        check=True,
        capture_output=True,
        text=True,
    )

    assert (target_directory / "recolva.dll").read_bytes() == b"native-library"
