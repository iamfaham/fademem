from pathlib import Path

import pytest

from recolva._native import (
    NativeLibraryUnavailableError,
    bundled_library_path,
    platform_library_filename,
)


def test_platform_library_filename_uses_expected_windows_name() -> None:
    assert platform_library_filename("Windows") == "recolva.dll"


@pytest.mark.parametrize(
    ("system_name", "expected"),
    [
        ("Linux", "librecolva.so"),
        ("Darwin", "librecolva.dylib"),
    ],
)
def test_platform_library_filename_uses_expected_unix_name(
    system_name: str, expected: str
) -> None:
    assert platform_library_filename(system_name) == expected


def test_bundled_library_path_returns_existing_platform_library(tmp_path: Path) -> None:
    native_dir = tmp_path / "_native"
    native_dir.mkdir()
    library = native_dir / "recolva.dll"
    library.write_bytes(b"native")

    assert bundled_library_path("Windows", resource_root=tmp_path) == library


def test_bundled_library_path_explains_missing_native_library(tmp_path: Path) -> None:
    with pytest.raises(NativeLibraryUnavailableError, match="recolva.dll"):
        bundled_library_path("Windows", resource_root=tmp_path)
