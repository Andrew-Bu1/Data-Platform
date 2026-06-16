# Sprint 5 — Batch Data Movement

**Theme:** Source/sink connectors, extraction patterns, schema capture, staging

---

## Goal

Move real data through the pipeline. Replace the stub tasks from Sprint 4 with real connector implementations. By the end of this sprint a source-to-sink pipeline can extract data from PostgreSQL or MySQL, stage it, and load it into a PostgreSQL target. Row counts and schema are captured and shown in the run detail UI.

---

## Concepts

### Extraction Patterns — Full vs Incremental

- **Full extract** — read the entire source table on every run. Simple and reliable. Appropriate when the table is small or when correctness requires seeing the full dataset (e.g., detecting deletes).
- **Incremental extract** — read only rows changed since the last run. More efficient for large tables but requires a reliable change indicator:
  - **High watermark** — a monotonically increasing column (e.g., `updated_at`, `id`). Store the last seen value; next run reads `WHERE updated_at > last_watermark`.
  - **Limitation**: rows with non-updated timestamps (backdated records, late arrivals) are missed.
  - **CDC (Change Data Capture)** — reads the database's write-ahead log to capture all changes. Covered in Sprint 16.
- For Sprint 5: implement full extract only. Add high watermark incremental in Sprint 5+.

### Staging — Why You Don't Load Directly

- **Staging** is a buffer between extraction and loading. You extract to staging first, then load from staging to the target.
- Why: if the load fails halfway, you can retry from staging without re-extracting. The source is not touched twice.
- Staging options:
  - **In-memory** (Python list/DataFrame) — fast, simple, limited by worker memory. Appropriate for small tables (<100MB).
  - **Local file** (Parquet on disk) — handles larger datasets, survives worker restarts.
  - **Object storage** (S3/MinIO) — durable, shareable between workers, scales to any size.
- For Sprint 5: start with in-memory staging. Add file-based staging when you hit memory limits.

### Schema Capture

- Before loading data, capture the schema of what was extracted: column names, data types, nullable flags.
- Store the schema snapshot in the `RunStep.metadata` JSONB field.
- Why schema capture matters:
  - **Schema drift detection** — if the source schema changes between runs, alert the user.
  - **Target table creation** — if the target table does not exist yet, create it from the captured schema.
  - **Documentation** — users can see exactly what shape of data flowed through each step.
- Use Python dataclasses for schema representation:
  ```python
  @dataclass
  class ColumnDef:
      name: str
      data_type: str  # normalized: "string", "integer", "float", "boolean", "timestamp", "date"
      nullable: bool
  
  @dataclass
  class Schema:
      columns: list[ColumnDef]
  ```

### Connector Interface Design (Python Protocol)

```python
class SourceConnector(Protocol):
    def connect(self) -> None: ...
    def disconnect(self) -> None: ...
    def get_schema(self, table: str) -> Schema: ...
    def extract(self, table: str, config: ExtractConfig) -> Iterator[list[dict]]: ...
    #                                                         ↑ yields batches of rows

class SinkConnector(Protocol):
    def connect(self) -> None: ...
    def disconnect(self) -> None: ...
    def load(self, rows: Iterator[list[dict]], schema: Schema, config: LoadConfig) -> LoadResult: ...
```

- `extract` yields **batches** (lists of rows), not individual rows. This allows streaming through large tables without loading everything into memory at once.
- Batch size is a config parameter (default: 1000 rows).
- Use context managers (`__enter__` / `__exit__`) for connection lifecycle, or `contextlib.contextmanager`.

### Polars vs Pandas

- **Pandas** — the classic Python DataFrame library. Mutable, uses NumPy under the hood, slower on large datasets.
- **Polars** — newer, written in Rust, significantly faster, immutable DataFrames, better for production workloads.
- **Arrow** — columnar in-memory format used by both Polars and modern data tools. Polars uses Arrow natively.
- For this project: use **Polars** for in-memory data manipulation in connectors and transforms. It is faster and its immutable API prevents accidental mutation bugs.
- Key Polars operations to know:
  - `pl.read_database` / `pl.DataFrame`
  - `df.filter(pl.col("status") == "active")`
  - `df.select(["col1", "col2"])`
  - `df.with_columns(pl.col("amount").cast(pl.Float64).alias("amount_float"))`
  - `df.write_database` / `df.to_dicts()`

### Error Handling in Data Pipelines

- Two error philosophies: **fail fast** (stop the entire run on first error) vs **partial load with error rows** (skip bad rows, load the rest, report what failed).
- For source extraction: fail fast. If you cannot extract, nothing downstream can run.
- For row-level errors during load: use an **error row table** — write failed rows to a `_errors` table with the error reason. Do not silently drop them.
- Always retry transient errors (network timeout, connection reset) with exponential backoff. Do not retry logic errors (missing required column, type mismatch).
- Use Prefect's built-in retry: `@task(retries=3, retry_delay_seconds=exponential(1, max=10))`.

---

## Tasks

### Orchestrator — Source Connectors
- [ ] Implement `PostgreSQLSource` — `psycopg2`/`psycopg`: connect, `get_schema`, `extract` as batch iterator
- [ ] Implement `MySQLSource` — `pymysql`: same interface
- [ ] Implement `RestApiSource` — `httpx`: paginated GET, JSON response parsing, schema inference
- [ ] Implement `SFTPSource` — `paramiko`: download file, parse CSV/JSON, schema inference
- [ ] Each source: normalize Python types to the canonical `ColumnDef.data_type` strings
- [ ] Each source: count total rows extracted, report in step metadata

### Orchestrator — Sink Connectors
- [ ] Implement `PostgreSQLSink` — write DataFrame to target table (full refresh only for now — Sprint 7 adds load semantics)
- [ ] Implement `ObjectStorageSink` — write as Parquet to MinIO using `boto3`/`s3fs`
- [ ] Each sink: return `LoadResult` (rows_written, bytes_written, duration_ms)

### Orchestrator — Pipeline Integration
- [ ] Replace stub source task with real extraction: call source connector, stage to in-memory Polars DataFrame
- [ ] Replace stub sink task with real load: call sink connector, write from DataFrame
- [ ] Wire extract → stage → load in the batch_pipeline flow
- [ ] Schema capture after extraction — serialize schema to JSON, send to control plane callback
- [ ] Row count callback after each step (rows_in, rows_out)
- [ ] Error handling: transient retry with backoff, error row table for row-level failures

### Frontend
- [ ] Source node config panel: table name input (after connection is selected)
- [ ] Run detail: show schema captured per source step (column name + type table)
- [ ] Run detail: show rows extracted and rows loaded per step
- [ ] Run detail: show extract vs load as separate steps with individual timing

---

## Interview Topics

- **Explain full extract vs incremental extract.** What is the high watermark pattern? What does it miss?
- **Why stage data between extraction and loading?** What happens if you skip staging and the load fails halfway?
- **What is schema drift?** How would you detect it and what should happen when it occurs?
- **What is Apache Arrow?** Why do modern data tools (DuckDB, Polars, Spark) use columnar formats instead of row-oriented formats?
- **Explain the difference between Polars and Pandas.** What makes Polars faster?
- **When do you use fail-fast vs error row pattern for bad data?** Give examples of each.
- **Explain exponential backoff with jitter.** Why add jitter?
- **What is a batch size in streaming extraction?** Why not yield one row at a time?

---

## Definition of Done

- [ ] Full source-to-sink pipeline runs with real PostgreSQL source and PostgreSQL sink
- [ ] Schema is captured and displayed in run detail UI
- [ ] Row counts (extracted, loaded) are visible in run detail UI
- [ ] REST API source works against a real paginated endpoint (use a public API or mock server)
- [ ] Object storage sink writes a valid Parquet file to local MinIO — verify with `polars.read_parquet`
- [ ] Transient network error triggers automatic retry (test by temporarily blocking the DB port)
- [ ] Row-level error writes a failed row to the `_errors` table instead of aborting the run
- [ ] Connector tests use a real PostgreSQL instance via Docker (no mocking the DB layer)
