from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, model_validator


class IcebergConfig(BaseModel):
    catalog: str = Field(default="polaris")
    namespace: str = Field(..., min_length=1)
    table: str = Field(..., min_length=1)
    partition_by: list[str] = Field(default_factory=list)
    dedup_order_by: list[str] = Field(default_factory=list)
    codec: Literal["zstd", "snappy"] = "snappy"
    pk_cols: list[str] = Field(default_factory=list)


class LandingConfig(BaseModel):
    format: Literal["parquet", "csv"] = "parquet"
    codec: Literal["snappy", "gzip"] | None = Field(default=None)
    ttl_days: int = Field(default=7, ge=1, le=90)


class RawConfig(IcebergConfig):
    write_mode: Literal["append", "merge", "overwrite_partition"] = (
        "overwrite_partition"
    )

    @model_validator(mode="after")
    def require_pk(self) -> RawConfig:
        if self.write_mode == "merge" and len(self.pk_cols) == 0:
            raise ValueError("Merge Operation require pk_cols")
        return self


class StagingConfig(IcebergConfig):
    pattern: Literal[
        "append", "upsert", "scd2", "single_snapshot", "multi_snapshot", "scd4"
    ]
    delete_policy: Literal["ignore", "soft_delete", "hard_delete"] = "hard_delete"
    wap_enabled: bool = Field(default=False)

    @model_validator(mode="after")
    def require_pk(self) -> StagingConfig:
        if self.pattern in {"upsert", "scd2", "scd4"} and len(self.pk_cols) == 0:
            raise ValueError(f"Staging pattern {self.pattern} requires primary-key")
        return self
