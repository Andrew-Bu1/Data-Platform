# Sprint 9 — Dimension Modeling: SCD1 and SCD2

**Theme:** Slowly changing dimensions, surrogate keys, historical tracking

---

## Goal

Make dimension history a first-class platform feature. By the end of this sprint a user can configure a sink node as a dimension with SCD1 (overwrite) or SCD2 (history) behavior. The platform manages surrogate key generation, effective dating, and change detection automatically.

---

## Concepts

### Dimensions vs Facts

- **Fact tables** — measure things that happen. Rows represent events or transactions. They are append-only, high-volume, and contain foreign keys to dimensions.
- **Dimension tables** — describe the context around facts. Who placed the order (customer), what was ordered (product), when (date), where (geography). Dimensions are joined to facts at query time.
- **Conformed dimensions** — dimensions that are shared across multiple fact tables. A `dim_customer` table can be joined to `fact_orders`, `fact_returns`, and `fact_support_tickets`. If each fact had its own customer table, the warehouse would produce different answers for the same question.

### Slowly Changing Dimensions (SCD)

Dimensions change over time — a customer moves to a new city, a product changes its category, an employee changes their department. How you handle these changes determines what questions you can answer.

#### SCD Type 1 — Overwrite

- Simply overwrite the changed columns. No history preserved.
- After the change, it appears as if the dimension always had the new value.
- Use when: the old value was wrong (data correction), or historical context is not needed.
- Problem: if you run a historical report "sales by customer city last year", and a customer moved, SCD1 will show their new city — even for last year's transactions. The history is lost.
- Implementation: same as an upsert load with `last_write_wins` conflict resolution.

#### SCD Type 2 — Add a New Row

- When an attribute changes, **close the old row** and **insert a new row**.
- Each row has:
  - `valid_from TIMESTAMPTZ` — when this version became effective
  - `valid_to TIMESTAMPTZ` — when this version was superseded (NULL for the current row)
  - `is_current BOOLEAN` — convenience flag for `WHERE is_current = true`
  - `surrogate_key UUID` — unique per version (not per entity)
  - `natural_key TEXT` — the business identifier (same across all versions of the same entity)
- After the change, you can answer: "What city was this customer in when they placed this order in 2023?" Join on surrogate key, not natural key.
- Problem: dimension table grows over time. Queries must filter on `is_current` or use effective date joins.
- Implementation is the most complex of the SCD types — see below.

#### SCD Type 3 — Add a Column

- Store both the current and previous value in the same row as separate columns (`city`, `previous_city`, `city_changed_at`).
- Simple to query but only tracks one historical change — after two moves, the previous_previous value is lost.
- Rarely used. Only when you need exactly "current and one prior value" and historical completeness is not required.

### SCD2 Implementation Algorithm

```
For each incoming row (identified by natural_key):

1. Find the current row in the target: WHERE natural_key = X AND is_current = true

2a. If no current row exists → INSERT new row:
    - surrogate_key = new UUID
    - valid_from = NOW()
    - valid_to = NULL
    - is_current = true

2b. If current row exists and attributes CHANGED (use hash comparison):
    - UPDATE old row: SET valid_to = NOW(), is_current = false
    - INSERT new row with updated attributes:
      - surrogate_key = new UUID
      - valid_from = NOW()
      - valid_to = NULL
      - is_current = true

2c. If current row exists and attributes UNCHANGED → do nothing.

3. For rows in target that do not appear in source (natural_key missing):
    - Soft close: UPDATE SET valid_to = NOW(), is_current = false
```

### Hash-Based Change Detection

- Comparing column-by-column for changes is verbose and slow for wide tables.
- Instead, compute an MD5 or SHA-256 hash of all tracked columns and store it on the dimension row (`_row_hash`).
- On the next run, hash the incoming row's tracked columns and compare to `_row_hash`.
- If hashes differ → the row changed. If equal → no change.
- Hash computation:
  ```python
  import hashlib, json
  def row_hash(row: dict, tracked_columns: list[str]) -> str:
      values = {k: row[k] for k in sorted(tracked_columns)}
      return hashlib.md5(json.dumps(values, sort_keys=True, default=str).encode()).hexdigest()
  ```
- Store the hash as `_row_hash TEXT` on the dimension table. Do not include it in the hash computation itself.

### Surrogate Key Generation

- Use UUID v4 (`uuid_generate_v4()` in PostgreSQL, `uuid.uuid4()` in Python).
- Generate the surrogate key at load time in the Python loader, not in the database trigger — more portable and testable.
- Foreign keys in fact tables reference the **surrogate key** of the dimension at the time the fact was recorded, not the natural key. This is what preserves historical accuracy.

### Effective Date Joins

To find the correct dimension version for a historical fact:
```sql
SELECT f.*, d.customer_city
FROM fact_orders f
JOIN dim_customer d
  ON d.natural_key = f.customer_natural_key
  AND f.order_date >= d.valid_from
  AND (f.order_date < d.valid_to OR d.valid_to IS NULL);
```

This is why SCD2 enables historical accuracy: you can find the exact dimension row that was "current" at the time of the fact.

---

## Tasks

### Orchestrator
- [ ] Implement `SCD1Loader` — upsert with overwrite behavior for all tracked columns
- [ ] Implement `SCD2Loader`:
  - Compute `_row_hash` for all tracked columns per incoming row
  - Identify: new entities, changed entities, unchanged entities, deleted entities
  - Batch UPDATE (close old rows: `valid_to = NOW(), is_current = false`)
  - Batch INSERT (new rows with new `surrogate_key`, `valid_from = NOW()`)
  - Soft-close rows for deleted entities
  - Return SCD2 result: new_versions, closed_versions, unchanged, deleted
- [ ] Hash computation utility with deterministic serialization (handle NULLs, dates)
- [ ] Surrogate key generation (UUID v4) at load time
- [ ] `valid_from`, `valid_to`, `is_current`, `_row_hash` columns injected automatically

### Backend
- [ ] Add `history_semantic` field to sink node config: none, scd1, scd2
- [ ] Add `tracked_columns` field: which columns trigger a new SCD2 version when changed
- [ ] Add `natural_key` field: the business identifier column(s)
- [ ] Validate: if `history_semantic = scd2`, `natural_key` and `tracked_columns` must be set

### Frontend
- [ ] Sink node config: `History semantic` dropdown (none, scd1, scd2)
- [ ] Sink node config: (shown when scd2) `Natural key` multi-select
- [ ] Sink node config: (shown when scd2) `Tracked columns` multi-select
- [ ] Run detail: SCD2 summary card (new_versions / closed_versions / unchanged / deleted)

### SQL Practice
- [ ] Write the SCD2 effective date join query manually and test it against real data
- [ ] Run a dimension load, change one attribute in the source, run again — verify two versions exist with correct valid_from/valid_to
- [ ] Write a query that uses `is_current = true` and compare performance to the effective date range join
- [ ] Test the hash function: same values in different column order must produce the same hash

---

## Interview Topics

- **Explain SCD1, SCD2, and SCD3.** When would you choose each?
- **What is a conformed dimension?** Why does it matter for cross-fact reporting?
- **How does SCD2 enable historical accuracy in fact-dimension joins?** Walk through the effective date join.
- **What is the difference between a natural key and a surrogate key?** Why does SCD2 need both?
- **Explain the hash-based change detection approach.** What are the edge cases (NULLs, type coercion)?
- **What does "closing a row" mean in SCD2?** What columns are updated and to what values?
- **What is the performance cost of SCD2?** How does the growing dimension size affect join performance?
- **Why is `valid_to IS NULL` equivalent to `is_current = true` but slower for some query planners?** Discuss index design.

---

## Definition of Done

- [ ] SCD1 load overwrites changed rows — confirmed by running load twice with changed source data
- [ ] SCD2 load: changing one attribute creates a new row and closes the old one with correct `valid_from`/`valid_to`
- [ ] SCD2 load: unchanged rows produce no new versions (unchanged count equals total unchanged rows)
- [ ] SCD2 load: new entities get a new row with `valid_from = load_time` and `valid_to = NULL`
- [ ] SCD2 load: entities missing from source get soft-closed
- [ ] Effective date join query returns correct dimension version for historical facts
- [ ] SCD2 summary visible in run detail UI
- [ ] Hash tests: hash is stable across reruns for the same data, different data produces different hash
