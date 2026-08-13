import importlib.util
from pathlib import Path


def test_custom_build_hook_marks_wheel_as_native() -> None:
    project_root = Path(__file__).parents[1]
    specification = importlib.util.spec_from_file_location(
        "hatch_build", project_root / "hatch_build.py"
    )
    assert specification is not None and specification.loader is not None
    module = importlib.util.module_from_spec(specification)
    specification.loader.exec_module(module)

    build_data: dict[str, object] = {}
    module.CustomBuildHook.initialize(object(), "0.2.0", build_data)

    assert build_data["tag"] == "py3-none-win_amd64"
    assert build_data["pure_python"] is False


def test_custom_build_hook_uses_target_wheel_tag_from_environment(monkeypatch) -> None:
    project_root = Path(__file__).parents[1]
    specification = importlib.util.spec_from_file_location(
        "hatch_build", project_root / "hatch_build.py"
    )
    assert specification is not None and specification.loader is not None
    module = importlib.util.module_from_spec(specification)
    specification.loader.exec_module(module)
    monkeypatch.setenv("RECOLVA_WHEEL_TAG", "py3-none-manylinux_2_28_x86_64")

    build_data: dict[str, object] = {}
    module.CustomBuildHook.initialize(object(), "0.2.0", build_data)

    assert build_data["tag"] == "py3-none-manylinux_2_28_x86_64"
