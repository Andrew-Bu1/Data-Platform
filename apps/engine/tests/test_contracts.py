from pathlib import Path

import pytest
import yaml
from pydantic import ValidationError

from engine.cli import main as cli_main
from engine.models import DeclarativePipelineConfig

RESOURCES_DIR = Path(__file__).parent / "resources"
VALID_YAMLS = sorted((RESOURCES_DIR / "valid").glob("*.yaml"))
INVALID_YAMLS = sorted((RESOURCES_DIR / "invalid").glob("*.yaml"))


@pytest.mark.parametrize("yaml_path", VALID_YAMLS, ids=lambda p: p.name)
def test_valid_yaml_contracts(yaml_path: Path) -> None:
    data = yaml.safe_load(yaml_path.read_text(encoding="utf-8"))
    cfg = DeclarativePipelineConfig.model_validate(data)
    assert cfg.metadata.id
    assert cli_main(["validate", str(yaml_path)]) == 0


@pytest.mark.parametrize("yaml_path", INVALID_YAMLS, ids=lambda p: p.name)
def test_invalid_yaml_contracts(yaml_path: Path) -> None:
    data = yaml.safe_load(yaml_path.read_text(encoding="utf-8"))
    with pytest.raises((ValidationError, TypeError)):
        if not isinstance(data, dict):
            raise TypeError("YAML document must be an object")
        DeclarativePipelineConfig.model_validate(data)

    assert cli_main(["validate", str(yaml_path)]) == 1


def test_cli_rejects_duplicate_ids(tmp_path: Path) -> None:
    src = VALID_YAMLS[0].read_text(encoding="utf-8")
    (tmp_path / "one.yaml").write_text(src, encoding="utf-8")
    (tmp_path / "two.yaml").write_text(src, encoding="utf-8")
    assert cli_main(["validate", str(tmp_path)]) == 1

def test_cli_rejects_empty_yaml(tmp_path: Path) -> None:
    empty_file = tmp_path / "empty.yaml"
    empty_file.write_text("", encoding="utf-8")
    assert cli_main(["validate", str(empty_file)]) == 1
