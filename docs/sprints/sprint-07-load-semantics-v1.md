# Sprint 7 — Load Semantics v1: Full Refresh and Append

**Theme:** Warehouse loading patterns, grain, transaction facts, audit columns

---

## Goal

Make the platform warehouse-aware. By the end of this sprint a user can configure a sink node with a load mode (full refresh or append), a table role, and a grain definition. Both load modes work reliably and produce correct results on re-runs. The run detail shows a load summary (rows inserted, mode used).

---

## Concepts

### Grain — The Most Important Concept in Data Warehousing

- The **grain** of a table is the definition of what one row represents. It is the most precise level of detail stored in the table.
- Examples:
  - "One row per order line item" (fact table grain)
  - "One row per customer as of today" (snapshot dimension grain)
  - "One row per daily account balance" (periodic snapshot grain)
- **Declaring the grain explicitly** forces you to think clearly about what your table contains. It prevents joining two tables at incompatible grains (a classic source of fanout/double-counting bugs).
- Grain is stored as a list of column names that together form the natural key at that level of detail.
- **Never violate the grain.** If a fact table is at the order-line-item grain, do not add a "one summary row per order" to the same table. Create a separate table.

### Transaction Fact Tables

- A **transaction fact table** is the most common type of fact table. One row per business event (one row per sale, one row per click, one row per payment).
- Characteristics:
  - **High row count** — often the largest tables in a warehouse.
  - **Append-only** — new events are added; existing rows are never updated.
  - **Additive measures** — measures like `amount`, `quantity`, `duration_ms` can be summed across any dimension. This is what makes fact tables useful for aggregation.
  - **Foreign keys to dimensions** — each fact row carries dimension keys (customer_id, product_id, date_key) used for joining.
- When to use transaction facts: any event that happens at a discrete point in time with measurable quantities.

### Full Refresh Load

- **Full refresh** replaces the entire target table on every run.
- Two implementation strategies:
  1. `TRUNCATE` then `INSERT` — fast, but leaves a window where the table is empty (visible to concurrent queries).
  2. Write to a new table, then `ALTER TABLE ... RENAME` — atomic swap, no empty window.
  3. `CREATE TABLE new AS SELECT ...; DROP TABLE old; ALTER TABLE new RENAME TO old` — works for PostgreSQL.
- When to use: small reference tables, tables where correctness requires seeing the full picture (e.g., detecting deletes at source), daily snapshots.
- When NOT to use: large tables (too slow), tables where you need to preserve rows not in the current extract.
- **Atomicity** — full refresh must be atomic. Use a transaction: BEGIN, TRUNCATE, INSERT, COMMIT. If the INSERT fails, ROLLBACK and the old data remains intact.

### Append Load

- **Append** adds new rows to the existing target table. Never deletes or updates.
- Correct for transaction facts (sales, events, log entries) where history must be preserved.
- The risk: **duplicate rows**. If the same pipeline runs twice with the same source data, rows are doubled.
- Mitigation strategies:
  - **Watermark filter** — only extract rows newer than the last run's watermark. If you only extract new rows, you only append new rows.
  - **Deduplication on read** — downstream queries use `SELECT DISTINCT` or `ROW_NUMBER()` to deduplicate.
  - **Deduplication on write** — before inserting, delete matching rows from the target (using a batch key or event ID). Safer but slower.
- **Audit columns** — columns added automatically by the platform, not sourced from the input:
  - `_loaded_at TIMESTAMPTZ` — when the row was loaded.
  - `_pipeline_run_id UUID` — which run produced this row. Essential for tracing and removing a bad load.
  - `_source_batch_id TEXT` — the batch identifier from the source (if available).

### Write Disposition

- **Write disposition** tells the loader what to do when data already exists:
  - `truncate_insert` — always replace everything (full refresh).
  - `append` — always add new rows.
  - `fail_if_nonempty` — fail the run if the target table already has rows (useful for first-load validation).
- Store write disposition on the sink node config. The loader reads it at runtime.

### Data Layers (Bronze / Silver / Gold)

- A common warehouse organization pattern:
  - **Bronze (raw)** — exact copy of source data, no transforms, append-only, preserves history. The audit log of what arrived.
  - **Silver (curated)** — cleaned, deduplicated, standardized. Row-level quality is trusted.
  - **Gold (serving)** — aggregated, joined, business-logic-applied. Ready for BI tools and dashboards.
- The platform supports layers as project-level config: each layer maps to a connection + schema/prefix.
- A pipeline's bronze nodes land in the raw schema, silver nodes in the curated schema, gold nodes in the serving schema.
- This is a convention, not a technical constraint. Enforce it through platform config, not database permissions (at this stage).

---

## Tasks

### Backend
- [ ] Add `load_semantic`, `table_role`, `grain`, `write_disposition`, `audit_columns_enabled` fields to sink node config schema
- [ ] Validate these fields on pipeline save (if load_semantic is set, grain must also be set)
- [ ] `GET/PUT /v1/projects/:id/data-layers` — get/save project-level data layer config
- [ ] Create `project_settings` table migration for storing data layer config

### Orchestrator
- [ ] Implement `FullRefreshLoader` — TRUNCATE + INSERT in a transaction, with atomic rename option
- [ ] Implement `AppendLoader` — INSERT with audit columns injection
- [ ] Audit columns injection: add `_loaded_at`, `_pipeline_run_id` to every row before loading
- [ ] Load result: return rows_inserted, rows_in_target_before, rows_in_target_after, duration_ms
- [ ] Write disposition handling: truncate_insert, append, fail_if_nonempty
- [ ] Partitioning option: if `partition_column` is set, load to `target_table_{partition_value}` (e.g., `orders_2025_01`)
- [ ] Loader is selected from sink node config at runtime (not hardcoded in flow)

### Frontend
- [ ] Sink node config panel: `Load mode` dropdown (full_refresh, append)
- [ ] Sink node config panel: `Table role` dropdown (fact, dimension, reference, bridge)
- [ ] Sink node config panel: `Grain` input — comma-separated column names with autocomplete from schema
- [ ] Sink node config panel: `Audit columns` toggle
- [ ] Sink node config panel: `Partition column` optional input
- [ ] Run detail page: load summary card (rows inserted, write disposition, audit columns status)
- [ ] Project settings page: Data Layers editor (add/edit/remove layers with connection + schema)

### SQL Practice
- [ ] Write a raw SQL full refresh (TRUNCATE + INSERT) and test idempotency by running twice
- [ ] Write a raw SQL append with duplicate detection using `NOT EXISTS`:
  ```sql
  INSERT INTO target (col1, col2, _loaded_at)
  SELECT col1, col2, NOW()
  FROM staging
  WHERE NOT EXISTS (
    SELECT 1 FROM target t WHERE t.event_id = staging.event_id
  );
  ```

---

## Interview Topics

- **What is the grain of a fact table?** Why must you declare it explicitly?
- **What is the difference between full refresh and append?** Give a scenario where each is appropriate.
- **Explain the risk of double-loading in append mode.** What are three ways to prevent it?
- **What is an additive measure?** Why can you SUM revenue across any dimension but NOT SUM account balances?
- **What are audit columns?** Why is `_pipeline_run_id` more useful than just `_loaded_at`?
- **Explain the Bronze/Silver/Gold layering pattern.** What guarantees does each layer provide?
- **Why is full refresh dangerous for large tables?** What is the atomic swap technique and when would you use it?
- **What is the difference between `table_role` and `load_semantic`?** Give an example where a fact table uses upsert load semantic.

---

## Definition of Done

- [ ] Full refresh load: running the pipeline twice results in the same row count (idempotent)
- [ ] Append load: running the pipeline twice doubles the row count (expected behavior)
- [ ] Audit columns (`_loaded_at`, `_pipeline_run_id`) are present in every loaded row
- [ ] `_pipeline_run_id` in the target table matches the run ID visible in the UI
- [ ] `fail_if_nonempty` write disposition causes a run to fail if the target table has existing rows
- [ ] Load summary (rows inserted, mode, duration) is visible in run detail UI
- [ ] Data layers can be configured at the project level and a sink node can select a layer
- [ ] SQL tests: full refresh is atomic — test by killing the load midway and confirming old data is intact
