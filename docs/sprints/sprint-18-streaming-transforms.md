# Sprint 18 — Streaming Transformations

**Theme:** Window operations, streaming deduplication, stream materialization

---

## Goal

Add stateful streaming transformations. By the end of this sprint users can configure tumbling and hopping window aggregations, streaming deduplication, and materialize streaming results to a sink table updated in near real-time.

---

## Concepts

### Window Types

Windows group a stream of events into finite sets for aggregation. Without windows, you can only compute totals over the entire history of the stream — not "revenue in the last hour."

#### Tumbling Windows

- Fixed-size, non-overlapping windows. Each event belongs to exactly one window.
- Example: one window per hour. Events at 9:01, 9:30, 9:59 all go in the 9:00-10:00 window.
- Use case: "total revenue per hour," "error count per 5-minute period."
- Defined by: window size (duration).

```
Events: [9:01 $10] [9:30 $20] [10:05 $15] [10:30 $5]
Tumbling 1h:  |---9:00-10:00--$30---|---10:00-11:00--$20---|
```

#### Hopping (Sliding) Windows

- Fixed-size, overlapping windows. Defined by size AND slide interval.
- A window is emitted every `slide` interval covering the last `size` duration.
- Example: size = 1 hour, slide = 15 minutes → a new window result every 15 minutes covering the last hour. Each event appears in `size/slide = 4` windows.
- Use case: "rolling 1-hour revenue updated every 15 minutes."

```
Events: [9:01 $10] [9:30 $20] [10:00 $15]
Hopping size=1h, slide=30m:
  8:30-9:30:  $10
  9:00-10:00: $30
  9:30-10:30: $35
```

#### Session Windows

- Windows triggered by activity gaps. A session ends when no events arrive for a defined idle duration.
- Example: a user session ends after 30 minutes of inactivity.
- Each session window has variable length. No fixed duration.
- Use case: user session analytics, IoT sensor activity bursts.

### Flink Window API

```python
# PyFlink example
stream \
    .key_by(lambda event: event['user_id']) \
    .window(TumblingEventTimeWindows.of(Time.hours(1))) \
    .aggregate(SumAggregate(), WindowResultFunction())
```

- `key_by` — group the stream by a key. Each key gets its own independent window.
- `window` — define the window type and size.
- `aggregate` / `reduce` — the aggregation function applied to each window.
- Window triggers: by default, a window fires when the watermark passes its end time.

### Streaming Deduplication

- Events in a stream may be duplicated (network retries, at-least-once delivery).
- Streaming deduplication: for each event, check if an event with the same dedup key has already been seen.
- In Flink: use a keyed state (a `ValueState` per key) to track seen event IDs.
  ```python
  class DeduplicateFunction(KeyedProcessFunction):
      def open(self, ctx):
          self.seen = self.runtime_context.get_state(ValueStateDescriptor("seen", Types.BOOLEAN()))
      
      def process_element(self, event, ctx, out):
          if self.seen.value() is None:
              self.seen.update(True)
              out.collect(event)
          # else: duplicate, discard
  ```
- **State TTL** — keep dedup state for only a bounded time window (e.g., 24 hours). After TTL, the state is garbage-collected, and a re-delivery of an old event would pass through again. This is acceptable — dedup is best-effort for long-running streams.

### Stream Materialization

- **Materialization** — writing streaming results to a target table that is queryable by BI tools or downstream pipelines.
- The sink table is updated continuously (or in micro-batches). Unlike batch pipelines, it is never fully refreshed — it is always the current aggregation.
- Two patterns:
  1. **Upsert sink** — each window result is an upsert to the target table keyed by the window key + window start. The row is inserted on first emission and updated if the window is re-triggered by late data.
  2. **Append sink** — each window emission is a new row. The target table grows indefinitely and downstream queries must take the latest row per key.
- Use **upsert sink** for materialization. It is more complex but produces a clean, queryable table.
- For PostgreSQL sink: use `INSERT ON CONFLICT (key, window_start) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`.

### Micro-batch vs True Streaming

- **True streaming** (record-at-a-time): Flink emits results as each event arrives (after applying transforms). Lowest latency, highest resource cost.
- **Micro-batch**: Spark Structured Streaming model. Accumulate events for a short interval (e.g., 1 second), process as a mini batch. Higher latency (by the batch interval), more efficient.
- Flink is a true streaming engine. Windowing adds a batching-like behavior but only at the window granularity.
- For materialization, window results are emitted per window trigger, not per event — so even Flink windows behave more like micro-batches for the sink.

---

## Tasks

### Orchestrator / Streaming
- [ ] Implement `TumblingWindowAggregate` transform node in PyFlink:
  - Group by key columns, aggregate by window duration
  - Supported aggregates: count, sum, avg, min, max
  - Emit results to downstream with: key columns, window_start, window_end, aggregated values
- [ ] Implement `HoppingWindowAggregate` transform node: size + slide duration params
- [ ] Implement streaming `DeduplicateFunction` using Flink keyed state with configurable TTL
- [ ] Implement `UpsertSink` for PostgreSQL: `INSERT ON CONFLICT (key, window_start) DO UPDATE SET ...`
- [ ] Implement streaming filter and derive as stateless map/filter operations (no state needed)
- [ ] Side output for late data: route late events to a separate Kafka topic (don't silently drop)

### Backend
- [ ] Add streaming-specific node types to graph schema: `streaming_window`, `streaming_deduplicate`
- [ ] Streaming window node config: `window_type`, `window_size_seconds`, `slide_seconds` (hopping only), `key_columns`, `aggregations`
- [ ] Streaming dedup node config: `dedup_key_columns`, `state_ttl_hours`
- [ ] Compiler: detect if pipeline contains streaming nodes and route to streaming deployer

### Frontend
- [ ] Streaming window node: key columns selector, window type dropdown, duration inputs, aggregation list
- [ ] Streaming dedup node: key columns selector, TTL input
- [ ] Pipeline builder: streaming nodes have a different visual style (e.g., animated edge lines) to distinguish from batch nodes
- [ ] Streaming job detail: add events/sec and late events/sec metrics to status page

### SQL Practice
- [ ] Write a SQL query that computes hourly revenue using GROUP BY on a timestamp truncated to the hour (simulating tumbling windows in batch)
- [ ] Write the equivalent hopping query using `GENERATE_SERIES` to produce overlapping windows
- [ ] Write the upsert sink SQL for window materialization with `INSERT ON CONFLICT`

---

## Interview Topics

- **Explain tumbling vs hopping windows.** Give a real-world use case for each.
- **What is the difference between a window trigger and a watermark?** How do they interact?
- **How do you implement streaming deduplication in Flink?** What is the role of `KeyedProcessFunction` and `ValueState`?
- **What is state TTL and why is it necessary?** What happens to deduplication guarantees after the TTL expires?
- **What is an upsert sink in streaming?** Why is it preferred over append sink for materialization?
- **What does "late data" mean in event-time streaming?** Name two strategies for handling late events.
- **Explain the difference between true streaming and micro-batch.** Name one framework for each.
- **Why does `key_by` exist in Flink?** What guarantee does it provide?

---

## Definition of Done

- [ ] Tumbling window aggregation (1-minute count per key) produces correct results on a test stream
- [ ] Hopping window produces overlapping results with the correct slide interval
- [ ] Deduplication correctly drops a duplicate event with the same dedup key
- [ ] Dedup state TTL expires and a re-sent old event passes through (test with short TTL)
- [ ] Upsert sink writes window results to PostgreSQL and updates on re-emission
- [ ] Late events are routed to a side output Kafka topic, not silently dropped
- [ ] Streaming job metrics (events/sec, late events/sec) are visible in the UI
- [ ] Filter and derive nodes work correctly in the streaming pipeline
