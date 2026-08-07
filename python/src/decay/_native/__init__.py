"""Location helpers for platform-specific packaged native libraries."""

from __future__ import annotations

import platform
from pathlib import Path
from typing import Optional


class NativeLibraryUnavailableError(RuntimeError):
    """Raised when a platform wheel does not contain its native library."""


def platform_library_filename(system_name: str) -> str:
    """Return the bundled library filename for an operating-system name."""
    filenames = {
        "Windows": "memorydecay.dll",
        "Linux": "libmemorydecay.so",
        "Darwin": "libmemorydecay.dylib",
    }
    try:
        return filenames[system_name]
    except KeyError as error:
        raise NativeLibraryUnavailableError(
            f"memory-decay has no native library for {system_name!r}"
        ) from error


def bundled_library_path(
    system_name: Optional[str] = None, *, resource_root: Optional[Path] = None
) -> Path:
    """Return the installed platform library path, or raise a clear error."""
    filename = platform_library_filename(system_name or platform.system())
    package_root = resource_root or Path(__file__).resolve().parent.parent
    library_path = package_root / "_native" / filename
    if not library_path.is_file():
        raise NativeLibraryUnavailableError(
            f"packaged native library {filename!r} was not found at {library_path}"
        )
    return library_path
