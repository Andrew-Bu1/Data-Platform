from __future__ import annotations

from typing import Annotated, Literal

from pydantic import BaseModel, Field, model_validator

from .common import ColumnMapping


class DbmsExtractionConfig(BaseModel):
    mode: Literal["FULL", "INCREMENTAL"] = "FULL"
    tracking_column: str | None = Field(default=None)
    custom_sql: str | None = Field(default=None)
    lookback_minutes: int = Field(default=60, ge=0)

    @model_validator(mode="after")
    def require_tracking_for_incremental(self) -> DbmsExtractionConfig:
        if (
            self.mode == "INCREMENTAL"
            and not self.tracking_column
            and not self.custom_sql
        ):
            raise ValueError(
                "INCREMENTAL extraction requires tracking_column or custom_sql"
            )
        return self


class DbmsSlicingConfig(BaseModel):
    strategy: Literal["DIRECT", "NUMERIC_RANGE", "EXACT_BOUNDARY", "HASH_MODULO"] = (
        "DIRECT"
    )
    pk_columns: list[str] = Field(default_factory=list)
    target_partitions: int = Field(default=8, ge=1)
    max_concurrency: int = Field(default=4, ge=1)
    fetch_size: int = Field(default=10_000, le=100_000, ge=100)

    @model_validator(mode="after")
    def validate_slicing_caps(self) -> DbmsSlicingConfig:
        if self.strategy == "HASH_MODULO" and self.target_partitions > 16:
            raise ValueError("HASH_MODULO target_partitions must be <= 16")
        if self.strategy == "EXACT_BOUNDARY" and self.target_partitions > 64:
            raise ValueError("EXACT_BOUNDARY target_partitions must be <= 64")
        if self.strategy != "DIRECT" and not self.pk_columns:
            raise ValueError(f"{self.strategy} requires non-empty pk_columns")
        return self


class BaseSource(BaseModel):
    connection_ref: str = Field(...)
    columns: list[ColumnMapping] = Field(default_factory=list)


class DbmsSourceConfig(BaseSource):
    type: Literal["dbms"] = "dbms"
    dialect: Literal["oracle", "mssql", "postgres", "mysql"]
    database: str | None = Field(default=None)
    schema_name: str = Field(..., description="Schema name")
    table: str = Field(..., min_length=1)
    extraction: DbmsExtractionConfig = Field(default_factory=DbmsExtractionConfig)
    slicing: DbmsSlicingConfig = Field(default_factory=DbmsSlicingConfig)


class RestApiSourceConfig(BaseSource):
    type: Literal["rest_api"] = "rest_api"
    endpoint: str = Field(..., min_length=1)
    method: Literal["GET", "POST"] = "GET"
    pagination: Literal["none", "page_number", "cursor", "time_window"] = "page_number"


class SftpSourceConfig(BaseSource):
    type: Literal["sftp"] = "sftp"
    remote_dir: str = Field(..., min_length=1)
    file_pattern: str = Field(default="*.csv")
    file_format: Literal["csv", "json"] = "csv"
    decrypt_key_ref: str | None = Field(default=None)


SourceConfig = Annotated[
    DbmsSourceConfig | RestApiSourceConfig | SftpSourceConfig,
    Field(discriminator="type"),
]
