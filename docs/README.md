# Data Platform — Data Engineering Mastery Roadmap

A visual, low-code data platform built to learn data engineering in production depth.
21 sprints across batch, warehouse modeling, streaming, and governance. Each sprint builds on the previous one.

---

## Project Overview

**Platform:** Visual drag-and-drop data pipeline platform — build, execute, and monitor data flows without writing orchestration code by hand.
**Goal:** Cover every major data engineering topic through real platform work. Inspired by tools like Fivetran, dbt, and Airbyte but built from scratch.
**Stack:** Go · Python · TypeScript · Next.js · React Flow · Prefect · PostgreSQL · Redis · Kafka · Apache Flink · MinIO · Docker

---

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                      Frontend (Next.js)                         │
│         React Flow canvas · Pipeline Builder · Run History      │
└────────────────────────┬───────────────────────────────────────┘
                         │ REST / WebSocket
┌────────────────────────▼───────────────────────────────────────┐
│                    Control Plane (Go)                            │
│     Projects · Pipelines · Connections · Secrets · Scheduler    │
└────────┬───────────────────────────────────────┬───────────────┘
         │ Execution Plan (JSON)                  │ Metadata
┌────────▼────────────────┐            ┌──────────▼──────────────┐
│   Execution Plane        │            │   Connector Plane        │
│   Prefect Workers        │            │   Source/Sink Connectors │
│   Batch / Streaming      │            │   Schema Discovery       │
└─────────────────────────┘            └─────────────────────────┘
         │
┌────────▼────────────────────────────────────────────────────┐
│                    Shared Infrastructure                       │
│        PostgreSQL · Redis · MinIO · Kafka · Prometheus        │
└────────────────────────────────────────────────────────────┘
```

Three independent planes. The control plane holds all state. The execution plane runs batch jobs. The connector plane handles I/O with external systems.

---

## Domain Model

```
Organization
    └── Project
            ├── Environment (dev / staging / prod)
            ├── Connection ──── SecretReference
            ├── Pipeline
            │       └── PipelineVersion
            │               ├── Node (source / transform / sink)
            │               └── Edge
            ├── Deployment
            ├── Run
            │       └── RunStep
            └── Dataset
```

---

## Sprint Map

| # | Sprint | Theme | Learning Focus |
|---|--------|-------|----------------|
| 0 | [Architecture and Scope Freeze](sprints/sprint-00-architecture.md) | System design | DAGs, control planes, domain modeling |
| 1 | [Platform Skeleton](sprints/sprint-01-platform-skeleton.md) | Go + Next.js bootstrap | Multi-tenant design, Go HTTP, React app router |
| 2 | [Visual Pipeline Builder](sprints/sprint-02-pipeline-builder.md) | React Flow canvas | Graph theory, DAG validation, JSON schema |
| 3 | [Connections and Secrets](sprints/sprint-03-connections-secrets.md) | Connector config | Secret management, encryption, connector abstraction |
| 4 | [Batch Execution Foundation](sprints/sprint-04-batch-execution.md) | Prefect orchestration | Orchestration theory, topological sort, idempotency |
| 5 | [Batch Data Movement](sprints/sprint-05-data-movement.md) | Source/sink connectors | Extraction patterns, staging, schema capture |
| 6 | [Batch Transformations](sprints/sprint-06-transformations.md) | Transform nodes | Schema propagation, expression evaluation |
| 7 | [Load Semantics v1: Full Refresh and Append](sprints/sprint-07-load-semantics-v1.md) | Warehouse loading | Grain, transaction facts, audit columns |
| 8 | [Load Semantics v2: Upsert and Merge](sprints/sprint-08-load-semantics-v2.md) | Mutable tables | MERGE SQL, business keys, idempotent loads |
| 9 | [Dimension Modeling: SCD1 and SCD2](sprints/sprint-09-scd.md) | Dimension history | SCD theory, surrogate keys, change detection |
| 10 | [Snapshot Modeling](sprints/sprint-10-snapshots.md) | Periodic snapshots | Semi-additive facts, backfill, partitioning |
| 11 | [Joins, Aggregates, and Datasets](sprints/sprint-11-joins-aggregates.md) | Modeling tier | Star schema, fanout risks, aggregation correctness |
| 12 | [Accumulating Snapshot Facts](sprints/sprint-12-accumulating-snapshot.md) | Process modeling | Workflow facts, milestone tracking, SLA metrics |
| 13 | [Data Quality and Reconciliation](sprints/sprint-13-data-quality.md) | Trust and contracts | Data contracts, quality gates, reconciliation |
| 14 | [Observability and Governance](sprints/sprint-14-observability-governance.md) | Production readiness | Lineage, RBAC, data catalog, environment promotion |
| 15 | [Connector Framework v2](sprints/sprint-15-connector-framework.md) | Plugin architecture | Manifest-driven config, dynamic forms, extensibility |
| 16 | [CDC Foundation](sprints/sprint-16-cdc.md) | Change capture | Debezium, changelog model, offsets, replay |
| 17 | [Streaming Foundation](sprints/sprint-17-streaming-foundation.md) | Kafka + Flink setup | Partitions, consumer groups, event time, watermarks |
| 18 | [Streaming Transformations](sprints/sprint-18-streaming-transforms.md) | Window operations | Tumbling/hopping windows, deduplication, materialization |
| 19 | [Streaming Joins and Enrichment](sprints/sprint-19-streaming-joins.md) | Stateful streaming | Temporal joins, state retention, exactly-once semantics |
| 20 | [Production Hardening](sprints/sprint-20-production-hardening.md) | Operational maturity | Backfill, rollback, tenancy, disaster recovery |

---

## Execution Phases

### Phase 1 — Control Plane (Sprints 0–3)
Establish the platform skeleton: Go backend, Next.js frontend, React Flow canvas, connection management, and secrets. By the end of Sprint 3, a user can log in, create a pipeline with drag-and-drop nodes, and configure a connection.

### Phase 2 — Batch Pipeline Engine (Sprints 4–6)
Wire the execution plane: Prefect integration, graph compiler, source/sink connectors, and transformation nodes. By the end of Sprint 6, a real ETL pipeline runs end-to-end.

### Phase 3 — Warehouse Patterns (Sprints 7–12)
Make the platform warehouse-aware: full refresh, append, upsert, SCD dimensions, snapshots, joins, and accumulating facts. This is the data modeling mastery phase.

### Phase 4 — Production Data Platform (Sprints 13–15)
Add production-grade quality checks, lineage, governance, RBAC, and a pluggable connector framework. The platform becomes something a real team could use.

### Phase 5 — Streaming and Hardening (Sprints 16–20)
Extend to CDC and real-time pipelines with Kafka and Flink, then harden for production operations: backfill, rollback, disaster recovery.

---

## How to Use These Docs

Each sprint doc has:
1. **Goal** — what you will be able to build and explain at the end
2. **Concepts** — data engineering, systems, and language topics to learn deeply
3. **Tasks** — what to build step by step
4. **Interview Topics** — what this sprint unlocks for data engineering roles
5. **Definition of Done** — how to know you are finished

**Rule:** Do not move to the next sprint until the current one's Definition of Done is met.

### Continuous Rules

- **SQL is a first-class skill.** Every sprint that touches data should include raw SQL practice — don't hide everything behind ORM methods. Write the MERGE, the window function, the CTE yourself.
- **Python type hints everywhere.** Every function in the orchestration layer must be fully typed from Sprint 4 onward. Use `Protocol` for connector interfaces, not abstract base classes.
- **Test your pipelines.** Every connector and transform must have a test that runs against a real database (use Testcontainers or a docker-compose test fixture). Do not mock the database.
- **Schema is a contract.** Every node in a pipeline has an input schema and an output schema. Make schema mismatch a validation error, not a runtime surprise.
- **Keep labs separate.** Experiments, spike code, and benchmarks belong in `_lab/` folders. Do not pollute production paths with exploration code.

---

## Interview Readiness Tracker

- [ ] Sprint 0 — Can explain control plane vs execution plane, DAG topological sort, pipeline IR concept
- [ ] Sprint 1 — Can design a multi-tenant metadata schema, explain Go HTTP middleware chains
- [ ] Sprint 2 — Can explain cycle detection in a graph, describe JSON schema-driven UI patterns
- [ ] Sprint 3 — Can explain AES-256 encryption-at-rest, describe connector capability model
- [ ] Sprint 4 — Can explain Prefect flows vs tasks, describe idempotent job launching
- [ ] Sprint 5 — Can explain full extract vs incremental extract, schema drift handling
- [ ] Sprint 6 — Can explain schema propagation through a DAG, expression evaluation strategies
- [ ] Sprint 7 — Can explain grain, transaction facts, full refresh vs append tradeoffs
- [ ] Sprint 8 — Can write a MERGE statement, explain business key vs surrogate key
- [ ] Sprint 9 — Can explain SCD1 vs SCD2 vs SCD3, hash-based change detection
- [ ] Sprint 10 — Can explain periodic snapshot facts, semi-additive measures, backfill design
- [ ] Sprint 11 — Can explain star schema joins, fanout/explosion risk, aggregation grain
- [ ] Sprint 12 — Can explain accumulating snapshot pattern, milestone-based SLA tracking
- [ ] Sprint 13 — Can explain data contracts, reconciliation, quality gate strategies
- [ ] Sprint 14 — Can explain data lineage (column-level vs dataset-level), RBAC design
- [ ] Sprint 15 — Can explain plugin/manifest architecture, dynamic form generation from JSON Schema
- [ ] Sprint 16 — Can explain CDC vs polling, Debezium offset model, exactly-once delivery
- [ ] Sprint 17 — Can explain Kafka partitioning, consumer groups, event time vs processing time, watermarks
- [ ] Sprint 18 — Can explain tumbling vs hopping windows, streaming deduplication
- [ ] Sprint 19 — Can explain temporal joins, state retention costs, exactly-once vs at-least-once tradeoffs
- [ ] Sprint 20 — Can explain backfill strategies, blue/green pipeline deployments, multi-tenancy isolation
