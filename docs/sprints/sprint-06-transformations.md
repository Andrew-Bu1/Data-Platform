# Sprint 6 — Batch Transformations

**Theme:** Transform node implementations, schema propagation, data preview

---

## Goal

Add the most common data transformation operations as first-class nodes. By the end of this sprint a user can build a full ETL pipeline visually: extract from a source, apply select/filter/rename/cast/derive/deduplicate transforms, and load into a sink. The schema updates automatically as nodes are configured, and a preview button shows sample data at any point in the pipeline.

---

## Concepts

### Schema Propagation

- Every node in a pipeline has an **input schema** and an **output schema**.
- Transforms modify the schema: `select` removes columns, `rename` changes column names, `cast` changes data types, `derive` adds a new column.
- **Schema propagation** means computing each node's output schema from its input schema and its config, then passing that output schema as the input schema to the next node.
- This computation runs client-side (TypeScript, for instant feedback in the UI) and server-side (Go, for validation before execution).
- Why it matters: if node B expects column `amount` but node A was configured to drop it, we want a validation error before the run, not a runtime crash.
- Schema propagation rules by transform type:
  - `select`: output = subset of input columns
  - `rename`: output = input with column names changed
  - `cast`: output = input with data types changed
  - `derive`: output = input + one new column
  - `filter`: output = same schema as input (filters rows, not columns)
  - `deduplicate`: output = same schema as input
  - `sort` / `limit`: output = same schema as input

### Expression Evaluation for Derived Columns

- A `derive` node lets users write an expression to create a new column. Examples:
  - `amount * 0.2` (arithmetic)
  - `upper(name)` (string function)
  - `case when status = 'active' then 1 else 0 end` (conditional)
- The expression language must be simple enough for users to write, and safe enough that it cannot execute arbitrary code.
- **Option 1: SQL expressions** — the expression is a SQL fragment (`amount * 0.2`). The execution plane wraps it in `SELECT *, {expr} AS {new_col} FROM ...`. Works well when your execution layer is SQL-based (DuckDB).
- **Option 2: Formula language** — a restricted expression parser (e.g., `jmespath`, or a simple arithmetic parser). Safer but limited.
- For this project: use SQL expressions evaluated by **DuckDB** in the transform layer. DuckDB runs in-process (no server needed), supports full SQL, and is extremely fast on DataFrames.

### DuckDB for In-Process Transforms

- **DuckDB** is an embedded analytical SQL database. It runs in the same process as your Python worker.
- You can query Polars DataFrames directly from DuckDB:
  ```python
  import duckdb
  import polars as pl
  
  df = pl.DataFrame({"name": ["alice", "bob"], "amount": [100, 200]})
  result = duckdb.sql("SELECT upper(name) AS name, amount * 1.1 AS amount_adjusted FROM df").pl()
  ```
- Each transform node becomes a SQL `SELECT` or `SELECT DISTINCT` statement generated from the node config. No hand-written Python per transform.
- DuckDB handles type casting, string functions, date arithmetic, window functions — everything a transform layer needs.

### Transform Implementation Strategy

Rather than writing a Python class per transform type, generate SQL:

```python
def compile_transform_to_sql(node: TransformNode, input_table: str) -> str:
    if node.type == "select":
        cols = ", ".join(node.config["columns"])
        return f"SELECT {cols} FROM {input_table}"
    elif node.type == "filter":
        return f"SELECT * FROM {input_table} WHERE {node.config['expression']}"
    elif node.type == "derive":
        return f"SELECT *, {node.config['expression']} AS {node.config['column_name']} FROM {input_table}"
    elif node.type == "rename":
        renames = ", ".join(f"{old} AS {new}" for old, new in node.config["mappings"].items())
        return f"SELECT * EXCLUDE ({','.join(node.config['mappings'].keys())}), {renames} FROM {input_table}"
    ...
```

### Node-Level Data Preview

- The preview endpoint runs the pipeline partially: extract a sample (first 100 rows) from the source, then run all transforms up to and including the requested node.
- The preview must be fast (< 2 seconds). Use a low `LIMIT` on extraction. Run transforms in DuckDB (in-process, no network roundtrip).
- The preview result is a JSON array of rows returned to the frontend. Show it in a data table component.
- Cache the preview result on the server for 30 seconds — if the same node is previewed twice with the same config, return the cached result.

### TypeScript Schema Propagation (Frontend)

- The frontend must compute the output schema at each node to:
  1. Show the user what columns are available in the side panel dropdowns.
  2. Show schema mismatch errors before the user saves.
- Implement `propagateSchema(nodes, edges, connections): Map<nodeId, Schema>` in TypeScript.
- This is a topological-sort + fold operation: walk nodes in topological order, compute output schema from input schema + node config.
- The schema type in TypeScript:
  ```ts
  type Column = { name: string; dataType: string; nullable: boolean }
  type Schema = { columns: Column[] }
  ```

---

## Tasks

### Orchestrator — Transform Implementations
- [ ] Implement DuckDB-based transform runner (`transforms/runner.py`)
- [ ] `select` transform — compile to `SELECT col1, col2 FROM input`
- [ ] `rename` transform — compile to `SELECT * EXCLUDE (old), old AS new FROM input`
- [ ] `cast` transform — compile to `SELECT * EXCLUDE (col), CAST(col AS type) AS col FROM input`
- [ ] `derive` transform — compile to `SELECT *, {expr} AS {name} FROM input`
- [ ] `filter` transform — compile to `SELECT * FROM input WHERE {expression}`
- [ ] `sort` transform — compile to `SELECT * FROM input ORDER BY {col} {dir}`
- [ ] `limit` transform — compile to `SELECT * FROM input LIMIT {n}`
- [ ] `deduplicate` transform — compile to `SELECT DISTINCT {key_cols}, * FROM input`
- [ ] Each transform: compute output schema from input schema + config (Python function)
- [ ] Wire transforms into batch_pipeline flow between extract and load steps

### Backend
- [ ] Schema propagation service in Go — for each node, compute output schema given input schema and node config
- [ ] Schema validation: check that downstream node input schema is compatible with upstream node output schema
- [ ] `GET /v1/pipelines/:id/preview/:nodeId` — run partial pipeline, return sample rows

### Frontend
- [ ] Add transform node subtypes to the toolbar: select, rename, cast, derive, filter, sort, limit, deduplicate
- [ ] Side panel for `select`: checkbox list of available input columns
- [ ] Side panel for `rename`: editable mapping table (old → new)
- [ ] Side panel for `cast`: column picker + target type dropdown
- [ ] Side panel for `derive`: new column name input + SQL expression input
- [ ] Side panel for `filter`: column picker + operator dropdown + value input
- [ ] Side panel for `sort`: column picker + ascending/descending toggle
- [ ] Side panel for `limit`: number input
- [ ] Side panel for `deduplicate`: checkbox list of key columns
- [ ] Schema propagation in TypeScript — compute output schema per node, show column list in side panels
- [ ] Schema mismatch error badge: if a node references a column that does not exist in its input schema
- [ ] "Preview" button on each node → call preview endpoint → show data table in side panel

---

## Interview Topics

- **What is schema propagation in a visual pipeline tool?** Why does it need to run both client-side and server-side?
- **What is DuckDB?** How does it differ from PostgreSQL? Why is it well-suited for transformation workloads?
- **Explain how you would implement a `derive` transform that is safe against SQL injection.** Describe expression validation and parameterization.
- **What is the difference between push-down optimization and in-memory transformation?** When would you push a filter down to the source query vs apply it in the transform layer?
- **Why use columnar storage (Arrow) for in-memory transformations?** Explain why column-wise operations are faster than row-wise.
- **Explain `SELECT DISTINCT` vs `ROW_NUMBER() OVER (PARTITION BY key ORDER BY ...)` for deduplication.** What does each guarantee?
- **How would you implement a data preview that is always fast regardless of source table size?**

---

## Definition of Done

- [ ] All 8 transform types work end-to-end (extract → transform → sink) with real data
- [ ] Schema propagates through the pipeline — side panels show the correct available columns after each transform
- [ ] Schema mismatch (downstream node references dropped column) shows a validation error
- [ ] Preview button returns sample data within 2 seconds for a 100-row extract
- [ ] DuckDB SQL generation tests: each transform type produces the correct SQL from its config
- [ ] Transform tests run against real data (not mocked) using DuckDB in-process
