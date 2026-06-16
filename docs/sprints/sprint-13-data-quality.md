# Sprint 13 — Data Quality and Reconciliation

**Theme:** Data contracts, quality rules, quality gates, reconciliation

---

## Goal

Add trust to pipelines and datasets. By the end of this sprint every node can have quality rules attached to it. Rules run automatically after each step and can warn or fail the run. Users can see a quality results summary per run, and a quality history per dataset.

---

## Concepts

### Data Contracts

- A **data contract** is a formal agreement about what shape and quality a dataset will have. It specifies:
  - Schema (column names, types, required vs optional)
  - Statistical expectations (row count range, null rate limit, uniqueness requirements)
  - Freshness (how recently was this data loaded)
  - Referential integrity (foreign keys exist in reference tables)
- Contracts are defined by the producer (the pipeline that creates the dataset) and consumed by downstream pipelines.
- Why they matter: without contracts, a schema change upstream silently breaks downstream pipelines. With contracts, the break is caught at the producer before it propagates.
- Contracts live on the Dataset entity in the platform's metadata, not just in comments or tribal knowledge.

### Quality Rule Types

**Schema rules**
- Column must exist
- Column must be of type X
- Column must not be added without review (schema drift alert)

**Completeness rules**
- `row_count` — total rows must be between min and max
- `null_rate` — percentage of NULL values in a column must be below threshold
- `not_null` — a column must have zero NULLs

**Uniqueness rules**
- `unique` — all values in a column (or combination of columns) must be distinct
- `unique_rate` — percentage of distinct values among total must be above threshold

**Validity rules**
- `value_in_set` — column values must be in a defined set (e.g., `status IN ('active', 'inactive')`)
- `range` — numeric column must be within min/max bounds
- `regex` — string column must match a pattern

**Referential rules**
- `referential_integrity` — values in a column must exist in another table (soft FK check)
- `freshness` — the most recent timestamp in a column must be within X hours of now

### Quality Severity Levels

- `warn` — log the failure, continue the run. The pipeline finishes. A warning is visible in the UI.
- `fail` — abort the run at this step. Downstream steps do not execute. A failure is visible in the UI.
- Users configure severity per rule. `row_count_min` might be a `fail` (no data = serious problem); `null_rate` might be a `warn`.

### Quality Gate Pattern

A quality gate is a special node type (or a post-step hook) that evaluates quality rules before allowing the pipeline to continue. If a rule fails:

```
[Source] → [Transform] → [Quality Gate] → [Sink]
                              ↓ fail
                         Stop run, report results
```

Alternatively, implement quality checks as part of each step's completion (not a separate node). The step runs, data lands in staging, checks run on staging, then either gate passes (load to target) or gate fails (discard staging, mark step failed).

### Source vs Target Reconciliation

- **Row count reconciliation** — after loading N rows to the target, confirm the target has at least N rows (accounting for any deduplication).
- **Checksum reconciliation** — compute `SUM(hash(row))` for all rows in source and target. If equal, the data transferred without corruption. Expensive for large tables — use on critical datasets only.
- **Column sum reconciliation** — for numeric columns, sum them in source and target and compare. Fast and catches most corruption.
- **Audit log reconciliation** — compare `_pipeline_run_id` counts in target to expected row count from run metadata.

### Schema Drift Detection

- Run schema comparison: `{columns in this run's extract schema}` vs `{columns in last run's schema}`.
- Changes to alert on:
  - Column added
  - Column removed
  - Column type changed
  - Column nullability changed
- Store schema snapshots in the RunStep metadata so you can diff between runs.
- Response options: warn and continue, fail the run, auto-adapt (add new column to target, drop removed column).

---

## Tasks

### Orchestrator
- [ ] Quality rule engine: given a Polars DataFrame and a list of `QualityRule` configs, return a list of `QualityResult(rule, status: pass/warn/fail, details)`
- [ ] Implement rule evaluators: `row_count`, `not_null`, `null_rate`, `unique`, `unique_rate`, `value_in_set`, `range`, `regex`, `freshness`, `referential_integrity`
- [ ] Schema drift detection: compare current schema to previous run's schema, return list of changes
- [ ] Column sum reconciliation: compute and compare column sums between source extract and target post-load
- [ ] Quality gate: after each step, run quality rules; if any `fail`-severity rule fails, abort run with quality failure error

### Backend
- [ ] Create `data_quality_rules` table migration — attached to pipeline node versions
- [ ] Create `data_quality_results` table migration — one result per rule per run step
- [ ] `POST /v1/nodes/:id/quality-rules` — add quality rule to a node
- [ ] `GET /v1/nodes/:id/quality-rules` — list rules
- [ ] `DELETE /v1/quality-rules/:id` — delete rule
- [ ] `GET /v1/run-steps/:id/quality-results` — get quality results for a step
- [ ] `GET /v1/datasets/:id/quality-history` — quality result trends over time

### Frontend
- [ ] Quality rules editor per node (accessible from side panel)
- [ ] Add rule form: rule type dropdown, configuration fields per type, severity selector
- [ ] Run detail: quality results table per step (rule, status badge, details)
- [ ] Run detail: overall quality summary (all pass / N warnings / M failures)
- [ ] Pipeline list: last run quality status badge on each pipeline card
- [ ] Dataset detail page: quality history chart (pass/warn/fail per run over time)

### SQL Practice
- [ ] Write a `NOT EXISTS` referential integrity check in raw SQL
- [ ] Write a query that computes null rate per column for a given table
- [ ] Write a query that detects duplicate rows using `GROUP BY` + `HAVING COUNT(*) > 1`
- [ ] Write a column sum reconciliation query: sum the same column in two tables and compare

---

## Interview Topics

- **What is a data contract?** Who defines it, who enforces it, and what happens when it is violated?
- **What is schema drift and how do you detect it automatically?**
- **Explain the difference between a `warn` and `fail` quality severity.** When would you use each?
- **What is row count reconciliation vs checksum reconciliation?** Which is more reliable and why is the more reliable one also more expensive?
- **How would you test referential integrity without a real foreign key constraint?** Write the SQL.
- **What is the fan-out risk when computing null rates on a very wide table?** Discuss query performance.
- **How would you build a quality history chart in a BI tool?** Describe the schema and the query.
- **Why should quality rules run against staging data before writing to the target?** What would go wrong if you checked quality after loading to the target?

---

## Definition of Done

- [ ] Quality rules are configurable per pipeline node
- [ ] A `fail`-severity rule failing aborts the run before loading to the target
- [ ] A `warn`-severity rule logs a warning but does not abort the run
- [ ] Quality results (pass/warn/fail + details) are visible in run detail UI
- [ ] Schema drift is detected and logged as a warning when a column is added or removed
- [ ] Column sum reconciliation confirms data integrity between source extract and loaded target
- [ ] Quality history API returns results for at least the last 10 runs
- [ ] Test: pipeline with a `unique` rule fails when source data has duplicates on the key
