import argparse
from pathlib import Path

import yaml
from pydantic import ValidationError

from engine.models.pipeline import DeclarativePipelineConfig


def _scan_yaml_files(target: Path) -> list[Path]:
    if not target.exists():
        raise FileNotFoundError(f"Path does not exist: {target}")

    if target.is_file():
        return [target]

    files = sorted([*target.rglob("*.yaml"), *target.rglob("*.yml")])

    if not files:
        raise FileNotFoundError(f"No .yaml files found under: {target}")

    return files


def cmd_validate(target: Path) -> int:
    try:
        files = _scan_yaml_files(target)
    except FileNotFoundError as exc:
        print(f"ERROR: {exc}")

        return 1

    has_error = False
    seen_ids: dict[str, Path] = {}

    for path in files:
        try:
            raw_data = yaml.safe_load(path.read_text(encoding="utf-8"))

            if not isinstance(raw_data, dict):
                raise TypeError("YAML document must be an object")
            cfg = DeclarativePipelineConfig.model_validate(raw_data)

            if cfg.metadata.id in seen_ids:
                raise ValueError(
                    f"Duplicate metadata.id '{cfg.metadata.id}' already defined in {seen_ids[cfg.metadata.id]}"
                )

            seen_ids[cfg.metadata.id] = path
            print(
                f"""{path} (id={cfg.metadata.id}, engine={cfg.compute.engine}) is validated successfully"""
            )
        except ValidationError as exc:
            has_error = True
            print(f"FAIL {path}")
            for err in exc.errors():
                loc = " -> ".join(str(p) for p in err["loc"]) or "<root>"
                print(f" - {loc}: {err['msg']}")

        except (yaml.YAMLError, ValueError, TypeError) as exc:
            has_error = True
            print(f"FAIL {path}: {exc}")

    return 1 if has_error else 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="engine", description="Data-Platform Engine CLI"
    )
    sub = parser.add_subparsers(dest="command", required=True)

    p_val = sub.add_parser("validate", help="Validate declarative YAML config(s)")
    p_val.add_argument("path", type=Path, help="YAML file or directory")

    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    if args.command == "validate":
        return cmd_validate(args.path)

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
