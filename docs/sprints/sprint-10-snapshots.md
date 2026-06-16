# Sprint 10 — Snapshot Modeling

**Theme:** Periodic snapshot facts, semi-additive measures, backfill, partitioning

---

## Goal

Support snapshot-style fact patterns. By the end of this sprint a user can configure a sink node with periodic snapshot mode, defining the snapshot grain and frequency. The platform correctly handles daily/weekly snapshots, manages partitioning, and supports backfill for historical periods.

---

## Concepts

### Periodic Snapshot Fact Tables

- A **periodic snapshot fact** captures the state of something at regular intervals — daily, weekly, monthly.
- One row per entity per period. The key difference from transaction facts: these rows are planned, not event-driven.
- Examples:
  - Daily account balances: one row per account per day.
  - Weekly inventory levels: one row per product per week.
  - Monthly customer metrics: one row per customer per month.
- Unlike transaction facts, periodic snapshots often need to produce rows even when there was no activity (zero balances, zero inventory). This means the pipeline cannot just extract changed rows — it must produce the full set of rows for every snapshot period.

### Semi-Additive Measures

- **Additive measures** can be summed across all dimensions: `SUM(revenue)` across all customers and all dates makes sense.
- **Semi-additive measures** can be summed across some dimensions but not others. The classic example: **balance**.
  - `SUM(balance) across customers` on a given date = total balance across all customers. Makes sense.
  - `SUM(balance) across dates` for one customer = nonsense. A customer's balance on Monday + Tuesday is not a meaningful number.
- Semi-additive measures require special handling in BI tools. The correct aggregation is `MAX`, `MIN`, `AVG`, or `LAST_VALUE` across the date dimension.
- The platform should flag semi-additive measures in node config so downstream queries are generated correctly.

### Snapshot Grain and Frequency

- Snapshot grain = entity + period. Example: `(account_id, snapshot_date)` uniquely identifies one row.
- Snapshot frequency = how often a snapshot is taken: `daily`, `weekly`, `monthly`.
- The snapshot pipeline must know: "I am producing the snapshot for date X." This comes from either:
  - The pipeline's scheduled run date.
  - An explicit `snapshot_date` parameter passed to the run.
- For backfill: the pipeline must accept a date range and produce one batch of rows per date in the range.

### Snapshot vs Accumulating Snapshot

- **Periodic snapshot** — produces one row per entity per period. The row is inserted and never updated.
- **Accumulating snapshot** — one row per entity (e.g., one row per order). The row is updated as the entity moves through lifecycle stages. Covered in Sprint 12.

### Partitioning for Snapshot Tables

- Snapshot tables grow one day's worth of rows at a time. Partitioning by snapshot date is natural.
- In PostgreSQL: declarative partitioning by `snapshot_date`:
  ```sql
  CREATE TABLE fact_daily_balances (
      account_id UUID NOT NULL,
      snapshot_date DATE NOT NULL,
      balance NUMERIC NOT NULL,
      _loaded_at TIMESTAMPTZ
  ) PARTITION BY RANGE (snapshot_date);
  
  CREATE TABLE fact_daily_balances_2025_01
      PARTITION OF fact_daily_balances
      FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
  ```
- The loader must create the partition for each snapshot date before inserting.
- Benefit: queries like `WHERE snapshot_date = '2025-01-15'` only scan one partition. Old partitions can be archived or dropped easily.
- Partition creation in Python: check if the partition exists; if not, CREATE TABLE PARTITION before loading.

### Backfill Design

- **Backfill** = running a pipeline for a past date range to produce historical snapshots.
- The pipeline must be parameterized by date: `run_for_date: date`.
- Backfill loop: call the pipeline for each date in the range, in order.
- Idempotency: if the snapshot for a date already exists, re-running should replace it, not duplicate. Use:
  ```sql
  DELETE FROM fact_daily_balances WHERE snapshot_date = :run_for_date;
  INSERT INTO fact_daily_balances SELECT ..., :run_for_date AS snapshot_date FROM source;
  ```
- Or: use `INSERT ON CONFLICT (account_id, snapshot_date) DO UPDATE SET balance = EXCLUDED.balance`.

### Snapshot Completeness Checks

- For a given snapshot date, every expected entity should have a row.
- If entity X had a balance yesterday, it should have a balance today (unless it was closed).
- Completeness check: `SELECT COUNT(*) FROM fact_daily_balances WHERE snapshot_date = :date` should match the expected entity count.
- Missing entities are a data quality problem — either the source did not produce the full set, or the pipeline logic is wrong.

---

## Tasks

### Orchestrator
- [ ] Implement `PeriodicSnapshotLoader`:
  - Accept `snapshot_date` as a loader parameter
  - Create the partition for `snapshot_date` if it does not exist
  - Delete existing rows for `snapshot_date` (idempotent re-run support)
  - Insert all rows from staging with `snapshot_date` column injected
  - Return snapshot result: rows_loaded, snapshot_date, partition_created
- [ ] Backfill flow: accepts `start_date`, `end_date`, calls snapshot loader for each date in range
- [ ] Snapshot completeness check: compare row count for snapshot date to expected entity count
- [ ] Snapshot retention: optionally drop partitions older than N days

### Backend
- [ ] Add `load_semantic = periodic_snapshot` handling to compiler
- [ ] Add snapshot fields to sink node config: `snapshot_frequency`, `snapshot_date_column`, `entity_key`, `retention_days`
- [ ] `POST /v1/runs/:id/backfill` — trigger a backfill with `start_date` and `end_date` parameters
- [ ] Backfill creates one run per snapshot date (or one run with multiple steps — design decision)

### Frontend
- [ ] Sink node config: `Load mode` — add `periodic_snapshot` option
- [ ] Sink node config: `Snapshot frequency` dropdown (daily, weekly, monthly)
- [ ] Sink node config: `Snapshot date column` — which column carries the snapshot date from source
- [ ] Sink node config: `Retention days` — optional, how many days of snapshots to keep
- [ ] Backfill dialog on pipeline page: date range picker, shows estimated number of runs
- [ ] Run detail: snapshot summary (rows loaded, snapshot_date, partition used)

### SQL Practice
- [ ] Create a partitioned table in PostgreSQL and load data into three monthly partitions
- [ ] Write a query that compares current snapshot vs previous snapshot (using LAG window function):
  ```sql
  SELECT account_id, snapshot_date, balance,
         balance - LAG(balance) OVER (PARTITION BY account_id ORDER BY snapshot_date) AS balance_change
  FROM fact_daily_balances;
  ```
- [ ] Write a completeness check query using a known entity set
- [ ] Test partition pruning: EXPLAIN a query with `WHERE snapshot_date = '2025-01-15'` and confirm only one partition is scanned

---

## Interview Topics

- **What is a periodic snapshot fact?** Give a real-world example and describe its grain.
- **What is a semi-additive measure?** Why can you not SUM a balance across dates?
- **How is a periodic snapshot fact different from a transaction fact?** Compare grain, growth pattern, and query patterns.
- **Explain PostgreSQL table partitioning by range.** What are the performance benefits and the operational overhead?
- **How do you design a backfill for a daily snapshot pipeline?** What makes it idempotent?
- **Explain LAG() and LEAD() window functions.** Write a query to compute day-over-day change in balance.
- **What is a partition prune?** When does PostgreSQL skip scanning a partition?
- **What are the tradeoffs between keeping all historical snapshots vs retaining only the last N days?**

---

## Definition of Done

- [ ] Periodic snapshot load inserts rows for a given snapshot date
- [ ] Running the same snapshot date twice replaces the rows (idempotent)
- [ ] Partitions are created automatically for each snapshot date
- [ ] Backfill runs a range of dates in order and produces one snapshot per date
- [ ] LAG-based comparison query works correctly across multiple snapshot dates
- [ ] Completeness check runs after each snapshot and reports missing entity count
- [ ] EXPLAIN query confirms partition pruning for single-date queries
- [ ] Retention: partitions older than N days are dropped when retention is configured
