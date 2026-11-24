from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, model_validator


class Metadata(BaseModel):
    id: str = Field(..., min_length=3, description="Pipeline ID")
    domain: str = Field(..., min_length=2)
    owner: str = Field(..., min_length=2)
    tags: list[str] = Field(default_factory=list)


class ScheduleConfig(BaseModel):
    cron: str | None = Field(
        default=None, description="Cron expression, e.g. '0 2 * * *'"
    )
    timezone: str = Field(default="Asia/Ho_Chi_Minh")
    assets: list[str] = Field(default_factory=list, description="Airflow Asset URI")
    catchup: bool = Field(default=False)


class ChecksourceConfig(BaseModel):
    enabled: bool = Field(default=False)
    method: Literal["api", "table_log", "flag_file"] = "table_log"
    connection_ref: str | None = Field(default=None)
    query_or_endpoint: str | None = Field(default=None)
    expected_value: str | None = Field(default=None)
    poke_interval: int = Field(default=60, ge=10)
    timeout: int = Field(default=120, ge=1)

    @model_validator(mode="after")
    def validate_checksource(self) -> ChecksourceConfig:
        if self.enabled:
            if not self.query_or_endpoint:
                raise ValueError(
                    "checksource.query_or_endpoint is required when checksource is true"
                )

            if not self.connection_ref:
                raise ValueError("connection ref is required when enable checksource")
        return self


class ColumnMapping(BaseModel):
    source: str = Field(description="source column name")
    target: str = Field(description="target column name")
    source_type: str = Field(description="Source column type")
    target_type: str = Field(description="Target column type")
    cast_expression: str | None = Field(default=None, description="SQL override")


class SecurityConfig(BaseModel):
    pii_columns: list[str] = Field(default_factory=list)
    # encrypt_inplace overwrites col
    # encrypt_and_mask replace col with col_encrypted + col_masked
    mode: Literal["encrypt_inplace", "encrypt_and_mask"] | None = Field(default=None)
    key_ref: str | None = Field(default=None, description="env ref from secret")

    @model_validator(mode="after")
    def validate_security_fields(self) -> SecurityConfig:
        has_pii = bool(self.pii_columns)
        if has_pii and (not self.mode or not self.key_ref):
            raise ValueError("pii_columns requires both mode and key_ref")
        if not has_pii and (self.mode is not None or self.key_ref is not None):
            raise ValueError("mode and key_ref cannot be set when pii_columns is empty")
        return self
