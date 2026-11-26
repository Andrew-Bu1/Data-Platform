from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, model_validator

from .common import (
    ChecksourceConfig,
    Metadata,
    ScheduleConfig,
    SecurityConfig,
)
from .source import SourceConfig
from .target import LandingConfig, RawConfig, StagingConfig


class ComputeConfig(BaseModel):
    engine: Literal["duckdb", "spark"] = "duckdb"
    # Later cpu, core, ram


class Pipeline(BaseModel):
    metadata: Metadata
    schedule: ScheduleConfig


class DeclarativePipelineConfig(Pipeline):
    kind: Literal["DeclarativePipeline"] = "DeclarativePipeline"
    checksource: ChecksourceConfig = Field(default_factory=ChecksourceConfig)
    compute: ComputeConfig = Field(default_factory=ComputeConfig)
    source: SourceConfig
    landing: LandingConfig = Field(default_factory=LandingConfig)
    raw: RawConfig
    staging: StagingConfig | None = Field(default=None)
    security: SecurityConfig = Field(default_factory=SecurityConfig)

    @model_validator(mode="after")
    def validate_rules(self) -> DeclarativePipelineConfig:
        if (
            self.source.type == "dbms"
            and self.source.extraction.mode == "INCREMENTAL"
            and self.staging is not None
            and self.staging.delete_policy != "ignore"
        ):
            raise ValueError(
                f"INCREMENTAL DBMS extraction cannot use {self.staging.delete_policy}"
            )

        if self.security.pii_columns and self.source.columns:
            col_types = {c.target: c.target_type for c in self.source.columns}

            for pii_col in self.security.pii_columns:
                if pii_col in col_types and col_types[pii_col] != "string":
                    raise ValueError(
                        f"PII column `{pii_col}` must have target_type = `string`"
                    )

        return self
