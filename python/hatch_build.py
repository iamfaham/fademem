"""Hatch build hook for platform-native fademem wheels."""

from __future__ import annotations

import os
from typing import Any

from hatchling.builders.hooks.plugin.interface import BuildHookInterface


class CustomBuildHook(BuildHookInterface):
    """Mark wheels that bundle a shared library as native platform artifacts."""

    def initialize(self, version: str, build_data: dict[str, Any]) -> None:
        build_data["tag"] = os.environ.get(
            "FADEMEM_WHEEL_TAG", "py3-none-win_amd64"
        )
        build_data["pure_python"] = False
