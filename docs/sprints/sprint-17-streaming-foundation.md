# Sprint 17 — Streaming Foundation

**Theme:** Kafka architecture, Apache Flink, event time, watermarks, checkpoints

---

## Goal

Add streaming pipeline infrastructure. By the end of this sprint the platform can deploy a streaming job to Apache Flink that reads from a Kafka topic, applies a stateless transform, and writes to a sink. Streaming job state (running/paused/failed) is tracked in the control plane and visible in the UI.

---

## Concepts

### Kafka Architecture

- **Kafka** is a distributed event streaming platform. Think of it as a durable, ordered, replayable message bus.
- **Topic** — a named, append-only log of events. Multiple producers write to a topic; multiple consumers read from it.
- **Partition** — a topic is split into N partitions. Each partition is an ordered log. Partitions allow horizontal scaling: multiple consumers can each read a different partition in parallel.
- **Offset** — the position of a message within a partition. Kafka consumers track their offset. Rewinding the offset replays past messages.
- **Consumer group** — a group of consumers that collectively read a topic. Each partition is assigned to exactly one consumer in the group at a time. Adding consumers to the group scales throughput (more partitions = more consumers can work in parallel).
- **Retention** — Kafka retains messages for a configurable time (default: 7 days) regardless of whether they were consumed. This is fundamentally different from a queue (which deletes on acknowledgment).
- **Replication** — each partition has N replicas across brokers. The leader handles reads/writes; followers replicate for fault tolerance.

### Apache Flink

- **Apache Flink** is a stateful stream processing engine. It consumes event streams, applies transformations with state, and writes to sinks.
- Key concepts:
  - **DataStream API** — the lower-level API for building stateful streaming jobs in Java/Python.
  - **Table API / Flink SQL** — higher-level SQL-like API. Write `SELECT ... FROM kafka_source GROUP BY ...` to do streaming aggregations.
  - **Sources and sinks** — Flink reads from Kafka (source) and writes to PostgreSQL, Kafka, or other sinks.
  - **Task manager** — the worker process that runs tasks. Multiple task managers = more parallelism.
  - **Job manager** — the coordinator that deploys and monitors jobs.
  - **Parallelism** — how many parallel instances of each operator run. Set per job or per operator.

### Event Time vs Processing Time

- **Processing time** — the wall clock time when the event is processed by Flink. Simple, but wrong for late data.
- **Ingestion time** — when Kafka received the event.
- **Event time** — when the event actually occurred (the timestamp embedded in the event payload). Most semantically correct for business metrics.
- Example: an app log event was generated at 9:00 AM but not delivered to Kafka until 9:05 AM due to mobile offline batching. Processing time = 9:05 AM. Event time = 9:00 AM.
- **Always use event time for business metrics.** Processing time gives wrong aggregation results for late-arriving data.

### Watermarks

- With event time, how does Flink know when to close a time window and emit results?
- It cannot wait forever — what if the last event in a window is never delivered?
- **Watermark** — a timestamp W that means "no event with event time < W will arrive in the future." When a watermark advances past the end of a window, the window is closed and results are emitted.
- **Watermark strategy** — how to generate watermarks:
  - **Bounded out-of-order** (`WatermarkStrategy.forBoundedOutOfOrderness(Duration.ofSeconds(5))`) — assume events are at most 5 seconds late. The watermark is `max_seen_event_time - 5s`.
  - **Monotonous** — assume events arrive in perfect order. The watermark is `max_seen_event_time`.
- **Late data** — events that arrive after the watermark has already passed their window. Options:
  - Drop silently
  - Emit to a side output (a separate stream for late events)
  - Re-trigger the window with the late event (allowed lateness)

### Checkpointing

- Streaming jobs run indefinitely. If a job crashes, it must restart from a consistent state — not from the beginning of time.
- **Checkpoint** — a consistent snapshot of all Flink operator states and the corresponding Kafka offsets. Stored in a durable store (HDFS, S3, or local filesystem for dev).
- On restart, Flink recovers from the latest checkpoint and replays events from the corresponding Kafka offsets.
- Checkpoint interval: lower = faster recovery, higher cost. Default: every 10 seconds to 1 minute.
- **Savepoint** — a manually triggered checkpoint. Used for planned upgrades or job migrations.

### Streaming Job State Model

Streaming jobs are not like batch runs — they never complete (unless cancelled or paused):

```
SUBMITTED → STARTING → RUNNING → PAUSED
                     → FAILED
                     → CANCELLED
```

- **RUNNING** — normal state. The job is processing events continuously.
- **PAUSED** — explicitly paused. Can be resumed. Flink saves a savepoint on pause.
- **FAILED** — job crashed. Control plane records the error and attempts restart (configurable).
- Monitor job health via Flink REST API: `/jobs/:id` returns current status and metrics.

---

## Tasks

### Infrastructure
- [ ] Add Kafka (KRaft mode, no ZooKeeper) to docker-compose
- [ ] Add Apache Flink (job manager + task manager) to docker-compose
- [ ] Create Kafka topic management helper scripts
- [ ] Configure Flink checkpointing to local filesystem volume

### Orchestrator / Streaming
- [ ] Implement `StreamingJobDeployer`: submits a Flink JAR or PyFlink script to the Flink job manager REST API
- [ ] Create base PyFlink streaming job template: reads from Kafka topic → applies transforms from execution plan → writes to sink
- [ ] Implement Kafka source connector for Flink (using Flink Kafka connector)
- [ ] Implement stateless transform step in streaming: filter, derive, rename (no state needed)
- [ ] Configure event time + bounded out-of-order watermark strategy
- [ ] Configure checkpointing (interval: 30 seconds, storage: filesystem)
- [ ] Streaming job status poller: call Flink REST API every 10s, report status to control plane

### Backend
- [ ] Create `streaming_jobs` table migration (`id`, `pipeline_version_id`, `flink_job_id`, `status`, `checkpoint_location`, `started_at`, `last_checkpoint_at`, `error_message`)
- [ ] `POST /v1/pipelines/:id/streaming-jobs` — deploy a streaming job
- [ ] `GET /v1/streaming-jobs/:id` — get streaming job status
- [ ] `POST /v1/streaming-jobs/:id/pause` — pause (save savepoint, stop job)
- [ ] `POST /v1/streaming-jobs/:id/resume` — resume from savepoint
- [ ] `DELETE /v1/streaming-jobs/:id` — cancel and delete job

### Frontend
- [ ] Pipeline builder: streaming pipeline mode toggle (batch vs streaming)
- [ ] Streaming job status page: current status, last checkpoint time, events/sec metric
- [ ] Watermark config panel: bounded out-of-order duration input
- [ ] Pause/resume/cancel buttons on streaming job status page

---

## Interview Topics

- **Explain Kafka topics, partitions, and consumer groups.** How does adding partitions affect consumer parallelism?
- **What is the difference between a Kafka topic and a traditional message queue?** What does retention enable that a queue cannot?
- **Explain event time vs processing time.** Give an example where they would produce different aggregation results.
- **What is a watermark in Flink?** Why do we need them for event-time windowing?
- **What is the bounded out-of-order watermark strategy?** What does the duration parameter control?
- **What is a Flink checkpoint?** How does it enable fault tolerance?
- **What is the difference between a checkpoint and a savepoint?** When would you use a savepoint?
- **Explain Kafka consumer group rebalancing.** When does it happen and what is the impact?

---

## Definition of Done

- [ ] Kafka and Flink are running in docker-compose and accessible locally
- [ ] A streaming job can be deployed to Flink via the control plane API
- [ ] The job reads events from a Kafka topic and logs them (stateless pass-through)
- [ ] Filter and derive transforms work in the streaming job
- [ ] Event time is used with bounded out-of-order watermarks (configurable duration)
- [ ] Checkpoints are saved every 30 seconds to local filesystem
- [ ] Job restart recovers from the latest checkpoint — test by killing the task manager
- [ ] Streaming job status (RUNNING/FAILED/PAUSED) is tracked and shown in the UI
- [ ] Pause/resume cycle works via savepoint: pause saves checkpoint, resume restarts from it
