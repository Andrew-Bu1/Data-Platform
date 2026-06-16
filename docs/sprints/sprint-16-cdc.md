# Sprint 16 — CDC Foundation

**Theme:** Change Data Capture, changelog model, offsets, replay

---

## Goal

Prepare the platform for incremental near-real-time data movement using Change Data Capture. By the end of this sprint a CDC connector can read the PostgreSQL write-ahead log, emit structured change events (insert/update/delete), store offsets, and apply those changes to a target table. The platform supports replaying from any past offset.

---

## Concepts

### CDC vs Polling

- **Polling (high watermark)** — query the source on a schedule: `SELECT * FROM orders WHERE updated_at > :last_run`. Simple, but:
  - Misses deletes entirely.
  - Misses rows where `updated_at` is not reliably updated.
  - Touches the source database on every poll (adds load).
  - Latency = polling interval (minimum minutes).
- **CDC (Change Data Capture)** — reads the database's **write-ahead log (WAL)** or binary log. The database writes all changes to the log before they are applied. CDC reads the log as a stream of events.
  - Captures inserts, updates, AND deletes.
  - Low source impact (WAL is written anyway).
  - Sub-second to second latency.
  - Requires database configuration (WAL level, replication slots).

### Write-Ahead Log (WAL) and Logical Replication

- PostgreSQL's WAL is a sequential log of all data changes. Originally for crash recovery and physical replication.
- **Logical replication** decodes the WAL into logical change events (not physical binary diffs). Each event is: `{operation: insert|update|delete, table, before: {}, after: {}}`.
- To enable: set `wal_level = logical` in PostgreSQL config. Create a replication slot.
- **Replication slot** — a named slot that PostgreSQL uses to track how far a consumer has read. PostgreSQL will not discard WAL segments until all replication slots have consumed them. This prevents data loss but can cause disk usage to grow if a consumer falls behind.
- **Debezium** — the most widely used open-source CDC tool. It reads WAL (or MySQL binlog, MongoDB oplog, etc.) and emits change events to Kafka topics. You do not need Kafka for this sprint — implement a simpler direct WAL reader.
- For this sprint: use **`psycopg2` / `psycopg3`** logical replication API to read the PostgreSQL WAL directly (no Kafka yet). Kafka is introduced in Sprint 17.

### Change Event Model

Every change event has the same envelope:
```python
@dataclass
class ChangeEvent:
    event_id: str           # UUID for this event
    operation: str          # "insert" | "update" | "delete"
    table_name: str
    schema_name: str
    before: dict | None     # Row state before the change (None for inserts)
    after: dict | None      # Row state after the change (None for deletes)
    transaction_id: int     # PostgreSQL transaction ID (xid)
    lsn: str                # Log Sequence Number (position in WAL)
    committed_at: datetime  # Transaction commit timestamp
    source_system: str      # Connection ID / source identifier
```

- `lsn` (Log Sequence Number) — the offset in the WAL. Used to resume from a checkpoint.
- `before` is only populated if the table has `REPLICA IDENTITY FULL` (expensive) or if the changed columns include the primary key.

### Offset Management

- **Offset** — the last WAL position successfully processed. Persisted in the control plane database.
- On startup, the CDC connector reads the stored offset and starts reading from that LSN.
- On successful processing of a batch of events, the offset is updated.
- If the connector crashes between processing and updating the offset, events will be re-delivered on restart — the consumer must be idempotent (handle duplicate events).
- Offset storage: `cdc_offsets` table with `connection_id`, `slot_name`, `last_lsn`, `last_processed_at`.

### Applying Change Events to Targets

Three modes for applying CDC changes to a target table:

1. **Append mode** — insert every change event as a new row (useful for building a changelog or event log).
2. **Current-state mode** — apply insert/update/delete to maintain a current-state copy of the source table. This is essentially an online upsert/delete.
3. **Audit log mode** — insert every change event with a `_cdc_operation` column and `_cdc_timestamp`. Preserves full history.

```sql
-- Current-state application
-- For INSERT or UPDATE events:
INSERT INTO target (pk, col1, col2, _cdc_lsn, _cdc_ts)
VALUES (:pk, :col1, :col2, :lsn, :ts)
ON CONFLICT (pk) DO UPDATE SET col1 = EXCLUDED.col1, col2 = EXCLUDED.col2, _cdc_lsn = EXCLUDED._cdc_lsn;

-- For DELETE events:
DELETE FROM target WHERE pk = :pk;
-- Or soft delete: UPDATE target SET deleted_at = :ts WHERE pk = :pk;
```

### Replay Capability

- **Replay** — re-read events from a past LSN position. Use cases:
  - The target table was accidentally corrupted — replay from last-known-good LSN to rebuild.
  - A bug in event processing was fixed — replay to re-process affected events with the correct logic.
- Implementation: reset the `last_lsn` in `cdc_offsets` to a past position. The connector will re-read from there.
- Requirement: events must be processed idempotently for replay to be safe.

### Deduplication for At-Least-Once Delivery

- CDC connectors typically guarantee **at-least-once delivery** — events may be delivered more than once (on crash/restart).
- Deduplication strategies:
  - Track `event_id` or `lsn` in a seen-events table. Skip if already processed.
  - Use `INSERT ON CONFLICT DO NOTHING` with `(pk, lsn)` as the conflict key.
  - Design the consumer to be naturally idempotent (upsert by PK, delete is idempotent).

---

## Tasks

### Orchestrator
- [ ] Implement `PostgreSQLCDCConnector` using `psycopg3` logical replication protocol
  - Create a replication slot if it does not exist
  - Stream change events from the slot
  - Decode `pgoutput` format into `ChangeEvent` objects
  - Acknowledge LSN after successful batch processing
- [ ] Implement change event applier: `append`, `current_state`, `audit_log` modes
- [ ] Deduplication: skip events with already-seen LSN (configurable, adds overhead)
- [ ] Offset commit: update `last_lsn` in control plane after each batch via callback

### Backend
- [ ] Create `cdc_offsets` table migration (`connection_id`, `slot_name`, `last_lsn`, `last_processed_at`)
- [ ] `GET /v1/connections/:id/cdc-status` — current LSN, lag estimate, last processed timestamp
- [ ] `POST /v1/connections/:id/cdc-reset` — reset offset to a given LSN (admin only)
- [ ] `POST /v1/connections/:id/cdc-slots` — create a replication slot
- [ ] `DELETE /v1/connections/:id/cdc-slots/:name` — drop a replication slot

### Frontend
- [ ] Source node config: `CDC mode` toggle (polling vs CDC)
- [ ] CDC status panel: current LSN, lag, last event timestamp
- [ ] CDC history page: recent change events for a connection (operation, table, timestamp)

### SQL Practice
- [ ] Enable `wal_level = logical` in local PostgreSQL (add to docker-compose init script)
- [ ] Create a replication slot manually using `pg_create_logical_replication_slot`
- [ ] Use `pg_logical_slot_get_changes` to manually read change events from the WAL
- [ ] Write the idempotent current-state applier SQL for all three operations (insert, update, delete)

---

## Interview Topics

- **What is CDC?** How is it different from polling? What can CDC capture that polling cannot?
- **What is the PostgreSQL WAL?** What is `wal_level = logical`? What is a replication slot?
- **What is a Log Sequence Number (LSN)?** What is it used for in a CDC pipeline?
- **Explain at-least-once vs exactly-once delivery.** Why is exactly-once hard to achieve?
- **How do you make a CDC consumer idempotent?** Give three approaches.
- **What is the risk of a replication slot that is not consumed?** What happens to PostgreSQL disk usage?
- **When would you use append mode vs current-state mode for applying CDC events?**
- **What is event replay?** Under what circumstances would you replay from a past offset?

---

## Definition of Done

- [ ] PostgreSQL CDC connector reads INSERT, UPDATE, DELETE events from the WAL
- [ ] Each event is correctly decoded into `ChangeEvent` with `before`/`after` values
- [ ] LSN offset is stored after each batch and used to resume on restart
- [ ] Crash and restart resumes from the last committed LSN — no events lost
- [ ] Current-state applier correctly applies all three operation types to the target
- [ ] Duplicate event delivery (simulated by replaying the same LSN range) is handled idempotently
- [ ] CDC reset API allows replaying from a past LSN
- [ ] Tests: all three operation types with both current-state and append application modes
