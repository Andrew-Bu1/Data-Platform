# Sprint 19 — Streaming Joins and Enrichment

**Theme:** Temporal joins, dimension lookup, state retention, exactly-once vs at-least-once

---

## Goal

Support advanced real-time enrichment pipelines. By the end of this sprint users can join two event streams with a time-bounded join, look up streaming events against a slowly changing dimension table, and choose between exactly-once and at-least-once delivery semantics. The platform exposes state retention configuration to manage resource costs.

---

## Concepts

### Stream-to-Stream Joins

Joining two streams is harder than joining two tables because the data arrives continuously and asynchronously.

**Interval Join** — the most common stream-to-stream join. Match events from stream A and stream B if their event times are within a bounded interval.

```
For each event A with event_time = T:
  Match events B where T - lower_bound ≤ B.event_time ≤ T + upper_bound
```

Example: match "order placed" events to "payment received" events within 30 minutes.

Flink API:
```python
orders_stream \
    .interval_join(payments_stream) \
    .between(Time.minutes(-1), Time.minutes(30)) \
    .process(JoinFunction())
```

**Window Join** — both streams are windowed (same window definition), and events in the same window are joined. Simpler than interval join but less flexible.

**State implications**: Flink must buffer events from both sides until the match window passes. For a 30-minute join window, Flink holds up to 30 minutes of events in state for each side. Large join windows = large state = more memory.

### Stream-to-Dimension Lookup (Temporal Table Join)

- A common pattern: enrich each streaming event with dimension attributes from a slowly changing dimension table.
- Example: each "sale" event carries a `product_id`. Look up the product's current category and price from `dim_product`.
- **Point-in-time lookup** — the key insight is that you want the dimension version that was current at the time of the event, not the current version today. Use the event's timestamp to find the correct SCD2 version.
- In Flink: use a **temporal table join** (versioned table or changelog table as the build side).
  ```sql
  -- Flink SQL
  SELECT s.sale_id, s.amount, p.category, p.price
  FROM fact_sales_stream s
  JOIN dim_product FOR SYSTEM_TIME AS OF s.event_time AS p
    ON s.product_id = p.natural_key
  ```
- **`FOR SYSTEM_TIME AS OF`** — the temporal join syntax. It finds the dimension version where `valid_from ≤ s.event_time < valid_to`.
- Dimension table source: can be a Kafka changelog topic (emitted by CDC pipeline from Sprint 16), or read from PostgreSQL using a JDBC lookup source (queries DB on each event — simpler but adds DB load).

### State Retention and RocksDB

- Streaming jobs accumulate state over time (window buffers, join buffers, dedup state). Without cleanup, state grows without bound.
- **State TTL** — configure how long Flink retains state per key before clearing it. After TTL, the state is garbage-collected.
  - Window state: cleared automatically when the window fires.
  - Join state: cleared after `join_window + allowed_lateness`.
  - Dedup state: cleared after configured TTL.
- **RocksDB state backend** — for large-scale jobs, Flink uses RocksDB (an embedded key-value store) instead of memory for state storage. RocksDB spills to disk. Configured at the Flink job manager level.
- **State size monitoring** — monitor `flink_taskmanager_job_task_operator_numberOfElements` and state size metrics. An unexpectedly growing state is a sign of a missing TTL configuration or a join window that is too large.

### Exactly-Once vs At-Least-Once Delivery

| | At-Least-Once | Exactly-Once |
|--|--------------|-------------|
| Definition | Every event is processed ≥ 1 times | Every event is processed exactly 1 time |
| On failure | May reprocess (replay) and produce duplicates | Reprocess but deduplicate via transactional checkpointing |
| Implementation cost | Simple | Complex: requires 2-phase commit (2PC) with sinks |
| Throughput | Higher | Lower (2PC adds overhead) |
| When to use | Idempotent sinks (upsert), low sensitivity to duplicates | Financial data, billing, anything where a duplicate record causes a real problem |

**Flink Exactly-Once mechanism**:
1. Flink checkpoints all operator state AND Kafka consumer offsets atomically.
2. For sink writes, Flink uses a 2-phase commit: pre-commit (write to a temp location), then at checkpoint, commit all pre-committed writes atomically.
3. The PostgreSQL JDBC sink supports exactly-once via transactions. The Kafka producer supports exactly-once via `enable.idempotence = true` + `transactional.id`.

**For most data warehouse pipelines**: at-least-once + idempotent upsert sink is simpler and sufficient. Exactly-once is overkill unless the sink cannot tolerate duplicates.

### Late Event Handling Strategies

- **Drop** — silently ignore events that arrive after the watermark has passed their window. Simple, loses data.
- **Side output** — route late events to a separate Kafka topic for later analysis or re-injection. No data loss.
- **Allowed lateness** — configure Flink to keep window state open for an additional `allowed_lateness` duration after the watermark fires. The window re-triggers if a late event arrives within this window. After `allowed_lateness` expires, state is cleared. Cost: state memory held for longer.

---

## Tasks

### Orchestrator / Streaming
- [ ] Implement `IntervalJoin` streaming node: two input streams, lower_bound and upper_bound duration, join key
- [ ] Implement `TemporalTableLookup` streaming node: event stream + dimension table source + temporal join SQL
- [ ] Dimension source: JDBC lookup (query PostgreSQL on each event, cache with TTL)
- [ ] Configure state TTL per operator type (join buffer, dedup, window)
- [ ] Add `exactly_once` toggle to streaming job config: when true, enable Flink 2-phase commit + idempotent Kafka producer
- [ ] Late event side output: route to `{topic}_late_events` Kafka topic
- [ ] Configure RocksDB state backend for jobs with large state

### Backend
- [ ] Streaming join node config schema: `join_type` (interval/window), `lower_bound_ms`, `upper_bound_ms`, `join_key_columns`
- [ ] Temporal lookup node config schema: `dimension_table`, `lookup_key`, `lookup_columns`, `cache_ttl_seconds`
- [ ] Add `delivery_guarantee` field to streaming job config: `at_least_once` or `exactly_once`
- [ ] `GET /v1/streaming-jobs/:id/state-metrics` — current state size per operator

### Frontend
- [ ] Interval join node: join key selector, lower/upper bound duration inputs
- [ ] Temporal lookup node: dimension table selector, lookup key, output columns selector
- [ ] Delivery guarantee selector on streaming pipeline config (with explanation tooltip: exactly-once vs at-least-once)
- [ ] State metrics panel on streaming job status page: per-operator state size

### SQL Practice
- [ ] Write a Flink SQL temporal join from scratch: event stream joined to a versioned dimension
- [ ] Write a Flink SQL interval join: match order and payment events within 30 minutes
- [ ] Write a query that reads from the late events Kafka topic and counts late events per source topic per hour

---

## Interview Topics

- **What is an interval join in stream processing?** What state does Flink maintain to support it?
- **What is a temporal table join?** How does `FOR SYSTEM_TIME AS OF` work?
- **Explain exactly-once delivery semantics.** Why does it require 2-phase commit with the sink?
- **What is the difference between at-least-once + idempotent sink vs exactly-once?** In practice, which do most data platforms use and why?
- **What is the state retention problem in streaming joins?** How does state TTL help?
- **What is RocksDB and why does Flink use it for large-scale state?**
- **What are the three strategies for handling late events?** Give an example where you would choose each.
- **What is allowed lateness?** What is the cost of setting it to a large value?

---

## Definition of Done

- [ ] Interval join matches events from two streams within a configurable time window
- [ ] Temporal lookup enriches each event with the correct dimension version at event time
- [ ] State TTL is configured on all stateful operators — confirmed by monitoring state metrics stabilizing over time
- [ ] At-least-once mode: job restarts from checkpoint after simulated failure, duplicate events handled by upsert sink
- [ ] Exactly-once mode: job restarts from checkpoint, no duplicate rows in the target table
- [ ] Late events are routed to a side output topic, not dropped
- [ ] State metrics (operator state size) are visible in the streaming job status UI
- [ ] Tests: interval join with events on both sides, interval join with unmatched events (verify no output), temporal lookup with multiple dimension versions
