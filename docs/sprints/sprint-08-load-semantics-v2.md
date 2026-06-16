# Sprint 8 — Load Semantics v2: Upsert and Merge

**Theme:** MERGE SQL, business keys, current-state tables, idempotent loads

---

## Goal

Support mutable current-state tables. By the end of this sprint a sink node can be configured with upsert mode, specifying a business key. The loader will insert new rows, update changed rows, and optionally soft-delete rows missing from the source. The merge summary (inserted/updated/deleted counts) is visible in the run detail.

---

## Concepts

### Current-State Tables

- A **current-state table** holds one row per entity, always reflecting its latest known state. Not history — just the most recent values.
- Examples: `customers` (current address, current status), `products` (current price, current inventory), `accounts` (current balance).
- Unlike transaction facts (append-only), current-state tables are mutable — rows are updated when the source changes.
- The upsert load semantic maintains current-state tables automatically.

### Business Key vs Surrogate Key

- **Business key (natural key)** — the identifier that comes from the source system. Examples: `customer_id` from the CRM, `order_number` from the order system, `sku` from the product catalog. It has business meaning.
- **Surrogate key** — a platform-generated identifier with no business meaning. Usually a UUID or auto-increment integer. Added by the warehouse, not by the source.
- Why add a surrogate key if you already have a business key?
  - Business keys can change (a customer might be reassigned a new ID after a system migration).
  - Business keys from different source systems might collide.
  - Foreign key joins are faster on integers than on long strings.
- **Rule**: use the business key as the match key for upsert logic. Store the surrogate key as an additional column. Downstream dimension tables reference the surrogate key, not the business key.

### MERGE SQL Statement

The standard SQL `MERGE` (also called `INSERT ... ON CONFLICT` in PostgreSQL) handles upsert atomically:

```sql
-- PostgreSQL syntax
INSERT INTO target (business_key, col1, col2, updated_at)
SELECT business_key, col1, col2, NOW()
FROM staging
ON CONFLICT (business_key) DO UPDATE SET
    col1 = EXCLUDED.col1,
    col2 = EXCLUDED.col2,
    updated_at = EXCLUDED.updated_at
WHERE target.col1 IS DISTINCT FROM EXCLUDED.col1
   OR target.col2 IS DISTINCT FROM EXCLUDED.col2;
-- The WHERE clause prevents no-op updates (don't update if nothing changed)
```

```sql
-- Standard SQL MERGE (PostgreSQL 15+, also supported in most other databases)
MERGE INTO target AS t
USING staging AS s ON t.business_key = s.business_key
WHEN MATCHED AND (t.col1 IS DISTINCT FROM s.col1 OR t.col2 IS DISTINCT FROM s.col2) THEN
    UPDATE SET col1 = s.col1, col2 = s.col2, updated_at = NOW()
WHEN NOT MATCHED THEN
    INSERT (business_key, col1, col2, updated_at) VALUES (s.business_key, s.col1, s.col2, NOW());
```

- **`IS DISTINCT FROM`** — handles NULL comparisons correctly. `NULL != NULL` in SQL, but `NULL IS DISTINCT FROM NULL` is false.
- Always use `IS DISTINCT FROM` in merge conditions to avoid unnecessary updates.

### Counting Inserts vs Updates

PostgreSQL does not directly return insert/update counts from `INSERT ON CONFLICT`. To get counts:
1. Use `xmax` system column trick: a row with `xmax = 0` after upsert was inserted; `xmax != 0` was updated.
2. Or: stage to a temp table, then run explicit separate INSERT for non-matching rows and UPDATE for matching.
3. Or: use the standard `MERGE ... RETURNING merge_action()` (PostgreSQL 17+).

### Delete Handling Options

When a source table is full-refreshed, rows in the target that no longer exist in the source must be handled:

- **Ignore** — leave orphan rows in the target. Simple, no data loss, but target grows stale.
- **Soft delete** — set a `deleted_at` column on orphan rows. Query with `WHERE deleted_at IS NULL`. History preserved.
- **Hard delete** — `DELETE FROM target WHERE business_key NOT IN (SELECT business_key FROM staging)`. Permanent, irreversible.

Default: **soft delete** is the safest choice. Hard delete requires explicit user opt-in.

### Idempotency of Upsert Loads

- An upsert is inherently more idempotent than append: running the same data twice produces the same state (not doubled rows).
- But be careful: if the source data changes between two runs of the same logical "batch", the second run will update rows to the second source state. This is correct behavior, not a bug.
- **Restatement** — re-processing a historical date range to correct past errors. Upsert supports restatement naturally: re-run the pipeline for the historical period; the upsert corrects rows that were wrong.

### Conflict Resolution Strategies

- **Last write wins** — the staging row always overwrites the target row if they differ. Simple, correct for most cases.
- **Source wins only if newer** — only update if `staging.updated_at > target.updated_at`. Useful when updates can arrive out of order.
- **Manual merge rules** — different columns have different rules (always update `status`, never update `created_at`).

---

## Tasks

### Orchestrator
- [ ] Implement `UpsertLoader`:
  - Load staging data to a temp table
  - Execute `INSERT ... ON CONFLICT (business_key) DO UPDATE` with IS DISTINCT FROM guard
  - Handle delete mode: none / soft_delete (`UPDATE SET deleted_at = NOW()`) / hard_delete (`DELETE`)
  - Return `MergeResult(inserted, updated, deleted, unchanged)`
- [ ] Support composite business keys (multiple columns as the match key)
- [ ] Support conflict resolution strategy: last_write_wins, source_wins_if_newer
- [ ] Idempotency: running same staging data twice produces same target state (add a test)
- [ ] Restatement: accept a `restatement_range` in loader config, document the behavior

### Backend
- [ ] Add upsert fields to sink node config schema: `business_key` (list of columns), `update_columns` (which columns to update), `delete_mode`, `conflict_resolution`
- [ ] Validate: if `load_semantic = upsert`, `business_key` must not be empty
- [ ] Merge result callback: accept `inserted`, `updated`, `deleted`, `unchanged` counts from execution plane

### Frontend
- [ ] Sink node config: `Load mode` — add `upsert` option
- [ ] Sink node config: `Business key` — multi-select from input schema columns
- [ ] Sink node config: `Update columns` — optional; if empty, update all non-key columns
- [ ] Sink node config: `Delete mode` — dropdown (ignore, soft_delete, hard_delete) with warning on hard_delete
- [ ] Sink node config: `Conflict resolution` — dropdown (last_write_wins, source_wins_if_newer)
- [ ] Run detail: merge summary card (inserted / updated / deleted / unchanged counts)

### SQL Practice
- [ ] Write a raw `INSERT ON CONFLICT` that counts inserts vs updates using `xmax`
- [ ] Write a soft delete query: `UPDATE target SET deleted_at = NOW() WHERE business_key NOT IN (...)`
- [ ] Write a restatement: re-run a load for a past date range and verify the target reflects the corrected source data
- [ ] Test `IS DISTINCT FROM` vs `!=` with NULL values — document the difference

---

## Interview Topics

- **Write a MERGE statement that inserts new rows and updates changed rows.** Use `IS DISTINCT FROM` to avoid no-op updates.
- **What is the difference between a business key and a surrogate key?** Why use both?
- **Why use `IS DISTINCT FROM` instead of `!=` in merge conditions?** What goes wrong with `!=` and NULLs?
- **Explain the three delete handling strategies.** When would you choose each?
- **Is an upsert idempotent?** Explain carefully — running the same data twice vs running different data twice.
- **What is a restatement and how does upsert support it?**
- **How do you count inserted vs updated rows after a PostgreSQL `INSERT ON CONFLICT`?** Describe the `xmax` trick or the MERGE RETURNING approach.
- **What is a current-state table?** Compare to a transaction fact table — what is different about their grain and growth pattern?

---

## Definition of Done

- [ ] Upsert load inserts new rows and updates changed rows correctly
- [ ] Running the same data twice produces the same target state (idempotency test)
- [ ] Soft delete marks removed rows with `deleted_at`, hard delete removes them
- [ ] Merge result (inserted/updated/deleted counts) is visible in run detail UI
- [ ] Composite business keys work (test with a two-column key)
- [ ] `IS DISTINCT FROM` prevents no-op updates (test: run with unchanged data, confirm `unchanged` count equals total rows)
- [ ] SQL tests: every merge scenario (insert, update, delete, no-change) is a separate test case
