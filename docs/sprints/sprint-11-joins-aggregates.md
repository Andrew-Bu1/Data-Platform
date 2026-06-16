# Sprint 11 — Joins, Aggregates, and Reusable Datasets

**Theme:** Star schema modeling, fan-out risks, aggregation correctness, reusable datasets

---

## Goal

Move the platform from a data movement tool to a data modeling tool. By the end of this sprint a user can build multi-input pipelines with join and aggregate nodes, reference previously saved datasets from other pipelines, and preview an execution plan before running. The platform validates join key compatibility before accepting the pipeline.

---

## Concepts

### Star Schema

- The most common dimensional model layout: one central **fact table** surrounded by **dimension tables**, connected by foreign keys.
- The shape resembles a star: fact in the center, dimensions radiating outward.
- A query joins the fact to any combination of dimensions to answer business questions:
  ```sql
  SELECT d.customer_city, d.product_category, SUM(f.revenue)
  FROM fact_orders f
  JOIN dim_customer d ON f.customer_sk = d.surrogate_key AND d.is_current = true
  JOIN dim_product p ON f.product_sk = p.surrogate_key AND p.is_current = true
  GROUP BY d.customer_city, d.product_category;
  ```
- **Snowflake schema** — a variation where dimensions are further normalized (e.g., `dim_product` joins to `dim_category`). More normalized but harder to query. Prefer star schema for analytical workloads.

### Join Types

- **INNER JOIN** — returns rows where the key exists in both tables. Drops rows from either table with no match. Appropriate when you want only facts with a known dimension.
- **LEFT JOIN** — returns all rows from the left table plus matching rows from the right. Non-matching right-side columns are NULL. Use when you want all facts, even those with unknown dimensions.
- **FULL OUTER JOIN** — returns all rows from both tables, NULLs on either side when no match.
- **CROSS JOIN** — Cartesian product. Every row from left × every row from right. Almost never intentional in a data pipeline.

### Fan-Out — The Most Common Modeling Bug

- **Fan-out** (also called "row explosion") happens when a join multiplies rows instead of enriching them.
- Cause: joining a fact table to a table that has multiple rows per key.
- Example: `fact_orders` (one row per order) joined to `order_items` (multiple rows per order) → each order row is duplicated for each item. If you then SUM revenue, you count each order's revenue multiple times.
- Prevention:
  1. Always check the grain of each table before joining.
  2. Aggregate the many-side table before joining: `SELECT order_id, SUM(amount) FROM order_items GROUP BY order_id`.
  3. Validate join key cardinality in the platform: before executing, count `SELECT COUNT(*), COUNT(DISTINCT key)` on the join table. If they differ, warn the user.

### Aggregation Correctness

- **Aggregation grain** — the set of columns in the GROUP BY clause. Every column in SELECT must be either in GROUP BY or inside an aggregate function.
- Common aggregate functions:
  - `COUNT(*)` — count all rows; `COUNT(col)` — count non-NULL values
  - `SUM`, `AVG`, `MIN`, `MAX`
  - `COUNT(DISTINCT col)` — count unique non-NULL values; expensive on large datasets
  - `APPROX_COUNT_DISTINCT` — approximate count via HyperLogLog, much faster for large cardinalities
- **Aggregation after join vs aggregation before join** — when possible, aggregate before joining (reduce the dataset first, then join to dimensions). This is called **pre-aggregation** and dramatically reduces the work done by the join.
- `HAVING` vs `WHERE`: `WHERE` filters rows before aggregation; `HAVING` filters groups after aggregation.

### Window Functions vs GROUP BY

- `GROUP BY` collapses rows — you lose individual row identity.
- Window functions (`OVER`) perform calculations across rows while preserving individual rows.
- When to use window functions in a data pipeline:
  - Rank rows within a group: `ROW_NUMBER() OVER (PARTITION BY customer_id ORDER BY order_date DESC)`
  - Running totals: `SUM(amount) OVER (PARTITION BY customer_id ORDER BY order_date ROWS UNBOUNDED PRECEDING)`
  - Lead/lag comparisons: `LAG(balance, 1) OVER (PARTITION BY account_id ORDER BY snapshot_date)`
  - Deduplication: take the latest row per key using `ROW_NUMBER()`.

### Reusable Datasets

- A **reusable dataset** is the output of a pipeline node that other pipelines can reference as an input.
- In the UI: a "Dataset" node type that represents a previously computed and stored dataset.
- Under the hood: a named table in the warehouse (e.g., `analytics.dim_customer`) that other pipelines can read from.
- Why important: avoid re-computing the same transformation in every pipeline that needs it. DRY for data.
- Platform behavior: a Dataset node creates a `Dataset` record in the control plane metadata. Other pipelines can add a "Dataset source" node and pick from the registry.
- The Dataset node injects a source connector step that reads from the named table.

### DuckDB for Join and Aggregate Execution

- For in-memory join and aggregate execution, use DuckDB:
  ```python
  result = duckdb.sql("""
      SELECT c.city, SUM(f.amount) as total
      FROM fact_df f
      JOIN dim_df c ON f.customer_id = c.customer_id
      GROUP BY c.city
  """).pl()
  ```
- Register Polars DataFrames as DuckDB views: `duckdb.register("fact_df", fact_polars_df)`.
- DuckDB handles multi-table joins efficiently in-process, without needing a running database.

---

## Tasks

### Orchestrator
- [ ] Implement `JoinNode` transform: accepts two input DataFrames, join type, and join keys; generates DuckDB JOIN SQL
- [ ] Implement `UnionNode` transform: UNION ALL two DataFrames with schema compatibility check
- [ ] Implement `AggregateNode` transform: GROUP BY + aggregate functions; generates DuckDB GROUP BY SQL
- [ ] Pre-join cardinality check: log a warning if join key is not unique on the dimension side
- [ ] Fan-out detection: compare `COUNT(*)` before and after join; warn if output > left input × 1.1
- [ ] Implement `SqlNode` transform: user provides raw DuckDB SQL; schema is inferred from result
- [ ] Reusable dataset source: read from a named table (treated as a regular PostgreSQL source)

### Backend
- [ ] Join node config schema: `left_key`, `right_key`, `join_type` (inner/left/right/full)
- [ ] Aggregate node config schema: `group_by` columns, `aggregations` list (column, function, alias)
- [ ] Union node config schema: `match_by` (name/position), schema compatibility validation
- [ ] SQL node config schema: `sql` (raw DuckDB SQL expression)
- [ ] Dataset registry: `POST /v1/datasets` (create from pipeline run), `GET /v1/datasets` (list), `GET /v1/datasets/:id/schema`
- [ ] `GET /v1/pipelines/:id/explain` — return the execution plan in human-readable form before running

### Frontend
- [ ] Join node: left/right port handles, join type dropdown, key mapping UI
- [ ] Aggregate node: group-by columns multi-select, aggregation rows (column + function + alias)
- [ ] Union node: show schema compatibility status (columns that match and columns that don't)
- [ ] SQL node: code editor with DuckDB SQL syntax highlighting
- [ ] Dataset source node: dataset picker dropdown (from registry)
- [ ] "Explain plan" button — show execution step order before running
- [ ] Fan-out warning: yellow badge on join node if output rows > input rows × threshold

### SQL Practice
- [ ] Write a star schema query joining one fact to two dimensions with GROUP BY and SUM
- [ ] Reproduce a fan-out bug intentionally: join a fact to a non-unique key and observe the duplicated SUM
- [ ] Fix the fan-out by pre-aggregating the many-side table first
- [ ] Write a query using ROW_NUMBER to take the latest version per customer from a table with duplicates

---

## Interview Topics

- **What is a star schema?** How is it different from a snowflake schema? When would you prefer each?
- **Explain fan-out in a join.** How would you detect it? How would you prevent it?
- **When does `SUM(revenue)` give you a wrong answer?** Walk through a fan-out example with concrete numbers.
- **What is the difference between `WHERE` and `HAVING`?** Which executes first?
- **Explain window functions vs GROUP BY.** Give a scenario where you need a window function.
- **What does `ROW_NUMBER() OVER (PARTITION BY customer_id ORDER BY created_at DESC)` return?** How would you use it to deduplicate?
- **What is a reusable dataset in a data platform?** How does it prevent redundant computation?
- **What is `COUNT(DISTINCT col)` doing? Why is it expensive and what is the approximate alternative?**

---

## Definition of Done

- [ ] Inner join and left join work correctly between two in-memory DataFrames
- [ ] Fan-out warning fires when join output > input × 1.1 (test with a non-unique join key)
- [ ] Aggregate node produces correct GROUP BY results — test with known input/output pairs
- [ ] Union node detects schema incompatibility before executing
- [ ] SQL node executes arbitrary DuckDB SQL and returns results
- [ ] Dataset registry allows registering a pipeline output and reading it in another pipeline
- [ ] Explain plan endpoint returns a readable execution order before running
- [ ] Star schema query test: fact joined to two dimensions with SUM produces the correct total
