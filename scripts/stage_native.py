"""Stage a target-native shared library as Python package data."""

from __future__ import annotations

import argparse
import shutil
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Copy a native fademem library into Python package resources."
    )
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--target-directory", type=Path, required=True)
    arguments = parser.parse_args()

    if not arguments.source.is_file():
        parser.error(f"native library source does not exist: {arguments.source}")

    arguments.target_directory.mkdir(parents=True, exist_ok=True)
    destination = arguments.target_directory / arguments.source.name
    shutil.copyfile(arguments.source, destination)
    print(destination)


if __name__ == "__main__":
    main()
