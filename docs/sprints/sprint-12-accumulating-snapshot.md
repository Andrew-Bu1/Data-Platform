# Sprint 12 — Accumulating Snapshot Facts

**Theme:** Process modeling, workflow lifecycle facts, milestone tracking, SLA metrics

---

## Goal

Support process-oriented history tables where one row represents an entity's full lifecycle. By the end of this sprint a user can configure a sink node as an accumulating snapshot, defining milestone columns and SLA thresholds. The platform correctly updates rows as milestones are completed and computes duration metrics automatically.

---

## Concepts

### Accumulating Snapshot vs Periodic Snapshot vs Transaction Fact

| Type | Rows per entity | Row lifecycle | Use case |
|------|----------------|---------------|----------|
| Transaction fact | Many (one per event) | Insert only, never updated | Sales, clicks, log entries |
| Periodic snapshot | Many (one per period) | Insert only, never updated | Daily balances, weekly inventory |
| Accumulating snapshot | One | Inserted once, updated many times | Order lifecycle, claim processing, ticket resolution |

- An **accumulating snapshot** has one row per process instance (one row per order, one row per insurance claim, one row per support ticket).
- That single row is updated each time the process advances to the next milestone.
- Final state: the row holds timestamps for every milestone, from the first event to the last.

### When to Use Accumulating Snapshot

Classic use cases:
- **Order lifecycle**: `order_placed_at`, `payment_confirmed_at`, `warehouse_picked_at`, `shipped_at`, `delivered_at`, `return_initiated_at`, `return_completed_at`.
- **Loan application**: `application_submitted_at`, `credit_check_at`, `underwriting_at`, `approved_at`, `funded_at`, `first_payment_at`.
- **Support ticket**: `opened_at`, `first_response_at`, `escalated_at`, `resolved_at`, `closed_at`.

The question to ask: "Does this entity go through a defined sequence of stages? Does leadership want metrics about how long each stage takes?" If yes, accumulating snapshot.

### Milestone Columns

- Each milestone is a nullable `TIMESTAMPTZ` column. NULL means "not yet reached."
- `is_complete BOOLEAN` — set to true when the process reaches its final milestone.
- The surrogate key of the row stays the same across updates.
- Natural key: the process identifier (e.g., `order_id`).
- Measure columns can also be updated as the process progresses (e.g., `total_amount` updated when items are added, `return_amount` updated when a return is processed).

### Update Logic

```
For each incoming row (identified by natural_key):

1. If no row exists → INSERT with natural_key and whatever milestones are present in the source.

2. If row exists:
   - For each milestone column: if incoming value is not NULL and existing value IS NULL → update it.
     (Never overwrite an existing milestone timestamp — once reached, a milestone is permanent.)
   - Update non-milestone measure columns.
   - Recompute duration metrics.
   - Update is_complete if final milestone is now set.
```

Key invariant: **milestone timestamps are immutable once set.** An order that was shipped on Tuesday cannot be un-shipped on Wednesday's pipeline run.

### Duration Metrics

Between any two milestones, compute a duration column:
- `time_to_payment_hours = EXTRACT(EPOCH FROM (payment_confirmed_at - order_placed_at)) / 3600`
- `time_to_shipment_days = EXTRACT(EPOCH FROM (shipped_at - order_placed_at)) / 86400`
- `total_cycle_time_days = EXTRACT(EPOCH FROM (delivered_at - order_placed_at)) / 86400`

The platform can auto-generate duration columns between any two configured milestones. Name them `duration_{from_milestone}_to_{to_milestone}_{unit}`.

### SLA Metrics

- Define SLA thresholds per milestone pair. Example: "orders should be shipped within 2 days of placement."
- Store: `sla_threshold_hours`, `is_sla_met BOOLEAN`, `sla_breach_hours NUMERIC` (how many hours over/under).
  ```sql
  UPDATE orders_lifecycle
  SET
    is_sla_met = (shipped_at - order_placed_at) <= INTERVAL '48 hours',
    sla_breach_hours = CASE 
        WHEN (shipped_at - order_placed_at) > INTERVAL '48 hours'
        THEN EXTRACT(EPOCH FROM ((shipped_at - order_placed_at) - INTERVAL '48 hours')) / 3600
        ELSE NULL
    END
  WHERE shipped_at IS NOT NULL AND order_placed_at IS NOT NULL;
  ```

### Milestone Completeness Validation

- For a given run, how many rows advanced at least one milestone? How many were untouched?
- Alert if an unusually high percentage of rows had no milestone updates — could indicate a pipeline or source issue.

---

## Tasks

### Orchestrator
- [ ] Implement `AccumulatingSnapshotLoader`:
  - Identify new entities (not yet in target) → INSERT
  - Identify existing entities → SELECT current milestones
  - For each milestone column: update only if incoming is NOT NULL and existing IS NULL
  - Recompute configured duration columns after update
  - Recompute SLA columns for all milestone pairs with configured thresholds
  - Update `is_complete` when final milestone is set
  - Return result: new_rows, milestone_updates (per milestone column), unchanged_rows
- [ ] Duration column auto-generation from milestone pair config
- [ ] SLA computation from threshold config
- [ ] Milestone completeness check: warn if >20% of rows had no updates

### Backend
- [ ] Add `load_semantic = accumulating_snapshot` to compiler
- [ ] Accumulating snapshot node config schema:
  - `milestones`: ordered list of milestone column names
  - `final_milestone`: which milestone marks completion
  - `sla_pairs`: list of `{from_milestone, to_milestone, threshold_hours, label}`
  - `measure_columns`: columns that can be overwritten (not milestone-immutable)
- [ ] Accumulating snapshot result in run step metadata: `{new_rows, updated_rows, milestone_updates: {col: count}}`

### Frontend
- [ ] Sink node config: `Load mode` — add `accumulating_snapshot` option
- [ ] Sink node config: milestone editor — ordered list of column names with up/down reorder
- [ ] Sink node config: final milestone selector
- [ ] Sink node config: SLA pairs editor — from/to milestone + threshold hours + label
- [ ] Run detail: accumulating snapshot summary (new rows, updated rows, per-milestone update counts)

### SQL Practice
- [ ] Create an order lifecycle table manually and run three update batches simulating order progression
- [ ] Write the SLA breach query from scratch (order shipped > 48h after placement)
- [ ] Write a query to find orders stuck in "payment_confirmed" for more than 24h with no shipment
- [ ] Write a query to compute average cycle time by product category using accumulated milestone data

---

## Interview Topics

- **What is an accumulating snapshot fact?** Give a real-world example of when you would use it instead of a transaction fact.
- **Why are milestone timestamps immutable once set?** What would go wrong if you allowed a milestone to be updated retroactively?
- **Explain the update logic for an accumulating snapshot.** Walk through what happens when an order advances from "shipped" to "delivered."
- **How do you compute an SLA breach?** Write the SQL.
- **What is the difference between a duration metric and an SLA metric?** Which one requires a threshold to be configured?
- **What kind of queries are well-suited to accumulating snapshot tables?** Compare to how you would answer the same questions with a transaction fact approach.
- **What does "milestone completeness" mean and why should you monitor it?**

---

## Definition of Done

- [ ] New process instances are inserted with whatever milestones are present
- [ ] Existing instances advance milestones correctly — a non-NULL incoming value fills a NULL column
- [ ] Existing milestone timestamps are never overwritten (immutability test: send conflicting milestone timestamps, confirm the original is kept)
- [ ] Duration columns are auto-generated and computed correctly
- [ ] SLA columns are computed: `is_sla_met`, `sla_breach_hours` correct for all cases
- [ ] `is_complete` is set when the final milestone is filled
- [ ] Accumulating snapshot summary is visible in run detail UI
- [ ] SQL tests: all five update scenarios covered (new, milestone advance, no update, final milestone set, SLA breach)
