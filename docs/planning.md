# Data Platform Project Plan

## 1. Project Goal

Build a visual data platform that supports:

- drag-and-drop pipeline design
- batch execution first
- later support for streaming
- multiple source/sink connectors
- warehouse-oriented load patterns
- gradual evolution from simple data movement to advanced historical modeling

The platform should grow in this order:

1. control plane
2. batch movement
3. batch transforms
4. warehouse semantics
5. governance and observability
6. streaming and CDC

---

## 2. Product Vision

The platform has 3 main layers:

### A. Control Plane
Responsible for:

- visual pipeline builder
- pipeline config storage
- versioning
- validation
- scheduling
- secrets management
- run history
- deployment orchestration

### B. Execution Plane
Responsible for:

- running batch jobs
- later running streaming jobs
- retries
- logging
- metrics
- failure handling

### C. Connector Plane
Responsible for:

- source connectors
- sink connectors
- schema discovery
- connection testing
- previewing data
- standardized config handling

---

## 3. Recommended Tech Stack

### Frontend
- Next.js
- React
- TypeScript
- React Flow
- Tailwind CSS
- shadcn/ui

### Backend / Control Plane
- Go
- PostgreSQL
- Redis
- OpenAPI / gRPC for internal services

### Orchestration / Execution
- Prefect for batch orchestration
- Docker or Kubernetes workers
- later Kafka + Flink for streaming

### Storage / Metadata
- PostgreSQL for metadata
- MinIO or S3 for object storage
- later Iceberg for analytical tables

### Observability
- Prometheus
- Grafana
- OpenLineage
- structured logs

---

## 4. Product Principles

- batch first, streaming later
- support a few connectors very well before adding many
- every pipeline must be versioned
- every run must be observable
- every node must have validation rules
- warehouse concepts should be first-class platform concepts, not hidden implementation details
- internal graph must compile into a runtime execution plan

---

## 5. Internal Domain Model

These concepts should exist early in the platform:

- `Project`
- `Environment`
- `Pipeline`
- `PipelineVersion`
- `Node`
- `Edge`
- `Connection`
- `SecretReference`
- `Dataset`
- `Run`
- `RunStep`
- `Deployment`
- `ConnectorDefinition`

For warehouse semantics, also add:

- `table_role`
- `grain`
- `load_semantic`
- `history_semantic`
- `freshness_mode`
- `time_semantic`

Suggested values:

### `table_role`
- fact
- dimension
- reference
- bridge

### `load_semantic`
- full_refresh
- append
- upsert
- periodic_snapshot
- accumulating_snapshot

### `history_semantic`
- none
- scd1
- scd2
- scd3

### `freshness_mode`
- batch
- micro_batch
- stream

---

## 6. Delivery Strategy

Use gradual phases:

1. foundation
2. movement
3. transforms
4. warehouse patterns
5. production readiness
6. streaming

Recommended sprint size:

- 2 weeks per sprint

---

# 7. Sprint Plan

---

## Sprint 0 — Architecture and Scope Freeze

### Goal
Set the technical direction and prevent rework later.

### Features
- define product scope for v1
- define core architecture
- define domain model
- define API boundaries
- define pipeline graph JSON format
- define runtime compiler concept
- define execution strategy for batch jobs
- define metadata schema in PostgreSQL
- define connector config standard
- define naming conventions

### Deliverables
- architecture document
- entity relationship diagram
- pipeline JSON schema
- initial repo structure
- sprint backlog for next phase

### Learning Focus
- control plane vs execution plane
- DAG modeling
- pipeline versioning
- data platform boundaries

---

## Sprint 1 — Project Setup and Platform Skeleton

### Goal
Create the first usable internal platform shell.

### Features
- monorepo or multi-repo setup
- frontend app scaffold
- backend service scaffold
- PostgreSQL integration
- authentication
- organization / workspace model
- project creation
- environment creation
- base layout for pipeline builder
- health checks
- audit log foundation

### Deliverables
- user can log in
- user can create project
- user can create environment
- empty pipeline page loads

### Learning Focus
- multi-tenant metadata design
- auth and environment isolation

---

## Sprint 2 — Visual Pipeline Builder MVP

### Goal
Let users create and save graph-based pipelines.

### Features
- React Flow canvas
- add/remove nodes
- add/remove edges
- node selection
- side panel for node configuration
- save draft pipeline
- load saved pipeline
- graph validation basics
- cycle detection
- pipeline version draft model

### Deliverables
- drag and drop graph editor
- pipeline saved in database
- basic validation errors shown in UI

### Learning Focus
- DAG constraints
- JSON schema driven config
- graph persistence

---

## Sprint 3 — Connection and Secret Management

### Goal
Make connectors manageable and reusable.

### Features
- create connection definitions
- store connection metadata
- secret references
- connection test API
- credential masking
- environment-specific connections
- source/sink capability flags
- simple sample preview endpoint

### Deliverables
- user can create a database connection
- user can test a connection
- pipeline node can reference a saved connection

### Learning Focus
- secure secrets design
- reusable connector abstraction
- environment-specific runtime config

---

## Sprint 4 — Batch Execution Foundation

### Goal
Run a pipeline end-to-end in batch mode.

### Features
- compile graph to internal execution plan
- Prefect integration
- job submission service
- worker bootstrap
- run status tracking
- logs collection
- retries
- cancellation
- run history page

### Deliverables
- one pipeline can be executed manually
- run status visible in UI
- logs visible in UI

### Learning Focus
- orchestration vs execution
- task retries
- idempotent job launching

---

## Sprint 5 — Batch Data Movement MVP

### Goal
Support basic source-to-target movement without heavy transforms.

### Features
- source: PostgreSQL
- source: MySQL
- source: REST API
- source: SFTP/file
- sink: PostgreSQL
- sink: object storage
- extract to memory/file staging
- schema capture
- row count tracking
- error handling for failed extracts/loads

### Deliverables
- source to sink data movement works
- schema and row counts are stored
- pipeline run shows source and sink step details

### Learning Focus
- extraction patterns
- staging
- schema drift basics
- load reliability

---

## Sprint 6 — Basic Batch Transformations

### Goal
Add the most common transformation blocks.

### Features
- select columns
- rename columns
- cast types
- derive column
- filter rows
- sort
- limit
- deduplicate
- schema propagation
- node-level preview

### Deliverables
- users can build simple ETL pipelines visually
- schema updates automatically through pipeline
- preview shows transformed sample data

### Learning Focus
- schema propagation
- transformation graph compilation
- reproducible transformations

---

## Sprint 7 — Load Semantics v1: Full Refresh and Append

### Goal
Support the first two core warehouse loading patterns.

### Features
- full refresh load mode
- append-only load mode
- partitioning options
- write disposition settings
- append validation
- duplicate detection options
- audit columns
- load summary metrics

### Deliverables
- user can choose load strategy per target node
- full refresh and append behavior works consistently
- audit columns added automatically when enabled

### Learning Focus
- transaction fact pattern
- append-only pipelines
- auditability
- grain definition

### Data Warehouse Concepts
- grain
- transaction facts
- additive measures
- load time vs event time

---

## Sprint 8 — Load Semantics v2: Upsert / Merge

### Goal
Support mutable current-state tables.

### Features
- primary key / business key selection
- merge/upsert load mode
- update vs insert rules
- delete handling options
- conflict resolution
- idempotency checks
- generated SQL strategy per sink
- restatement support foundation

### Deliverables
- current-state target table can be maintained with upsert
- merge run summary shows inserted/updated/deleted counts

### Learning Focus
- current-state modeling
- mutable facts
- business keys vs surrogate keys
- idempotent merges

### Data Warehouse Concepts
- latest state tables
- mutable facts
- merge semantics
- key design

---

## Sprint 9 — Dimension Modeling: SCD1 and SCD2

### Goal
Make dimension history a first-class feature.

### Features
- table role = dimension
- SCD1 overwrite mode
- SCD2 history mode
- surrogate key generation
- valid_from
- valid_to
- is_current flag
- hash-based change detection
- natural key mapping
- late change handling basics

### Deliverables
- user can define a dimension table
- user can choose SCD1 or SCD2
- platform maintains dimension history automatically

### Learning Focus
- dimensions vs facts
- natural key vs surrogate key
- historical tracking
- slowly changing dimensions

### Data Warehouse Concepts
- conformed dimensions
- SCD1
- SCD2
- effective dating

---

## Sprint 10 — Snapshot Modeling

### Goal
Support snapshot-style fact patterns.

### Features
- periodic snapshot load mode
- snapshot key definition
- snapshot frequency metadata
- current snapshot option
- snapshot partitioning
- snapshot retention rules
- compare current vs previous snapshot
- snapshot completeness checks

### Deliverables
- daily or periodic snapshot tables can be built
- snapshot runs can be backfilled
- user sees snapshot grain and frequency in config

### Learning Focus
- periodic snapshots
- semi-additive measures
- historical periodic reporting

### Data Warehouse Concepts
- periodic snapshot fact
- single latest snapshot vs multiple snapshots
- semi-additivity

---

## Sprint 11 — Join, Aggregate, and Reusable Dataset Layer

### Goal
Move from movement platform to modeling platform.

### Features
- inner join
- left join
- union
- group by
- aggregate functions
- reusable dataset node
- schema compatibility checks
- join key validation
- explain plan preview
- basic SQL node

### Deliverables
- users can build dimensional models from multiple datasets
- join pipelines are validated before execution
- reusable transformed datasets are supported

### Learning Focus
- star schema basics
- fact-dimension joins
- aggregation correctness
- join explosion risks

### Data Warehouse Concepts
- fact/dimension integration
- conformed joins
- aggregation grain
- bridge risks

---

## Sprint 12 — Accumulating Snapshot and Workflow Facts

### Goal
Support process-oriented history tables.

### Features
- accumulating snapshot load mode
- milestone field definitions
- process state transitions
- row update across lifecycle
- duration metrics
- SLA metric generation
- milestone completeness validation

### Deliverables
- platform supports workflows like order lifecycle or claim lifecycle
- one row can evolve across stages with tracked milestones

### Learning Focus
- process modeling
- workflow facts
- event lifecycle metrics

### Data Warehouse Concepts
- accumulating snapshot fact
- process latency
- milestone facts

---

## Sprint 13 — Data Quality, Validation, and Reconciliation

### Goal
Improve trust in pipelines and datasets.

### Features
- row count checks
- null checks
- uniqueness checks
- referential checks
- freshness checks
- schema drift alerts
- source vs target reconciliation
- run quality summary
- configurable failure thresholds

### Deliverables
- pipeline can fail or warn based on validation rules
- users can view data quality results per run

### Learning Focus
- operational trust
- data contracts
- reconciliation design

### Data Warehouse Concepts
- late-arriving facts
- reconciliation
- trust boundaries
- quality gates

---

## Sprint 14 — Observability and Governance

### Goal
Make the platform operationally useful for teams.

### Features
- OpenLineage integration
- dataset lineage graph
- run metrics dashboard
- cost estimation metadata
- pipeline ownership
- tags and search
- approval workflow
- role-based access control
- environment promotion flow

### Deliverables
- lineage available for datasets and runs
- governance metadata visible in UI
- role-based controls active

### Learning Focus
- lineage
- governance
- operational metadata
- production controls

---

## Sprint 15 — Connector Framework v2

### Goal
Make connectors pluggable instead of hardcoded.

### Features
- connector manifest format
- config schema per connector
- UI generated from schema
- capability registry
- connector test interface
- sample data preview interface
- versioned connector definitions
- connector packaging pattern

### Deliverables
- new connectors can be added with minimal backend UI rewrites
- connector behavior becomes standardized

### Learning Focus
- plugin architecture
- contract-first connector development
- platform extensibility

---

## Sprint 16 — CDC Foundation

### Goal
Prepare for incremental near-real-time data movement.

### Features
- CDC connector abstraction
- change event model
- insert/update/delete event handling
- offset storage
- checkpoint metadata
- replay capability
- deduplication support
- target changelog application mode

### Deliverables
- platform can ingest change streams as structured change events
- CDC metadata is stored and inspectable

### Learning Focus
- changelog thinking
- source-of-truth updates
- replay safety
- offset management

### Data Warehouse Concepts
- change capture
- late updates
- corrections
- event ordering

---

## Sprint 17 — Streaming Foundation

### Goal
Add the platform building blocks for streaming.

### Features
- streaming job deployment service
- Kafka topic abstraction
- message schema registry integration
- event-time metadata support
- watermark configuration
- streaming run state model
- long-running job status page
- checkpoint monitoring

### Deliverables
- streaming jobs can be deployed and monitored
- event-time and checkpoint settings are configurable

### Learning Focus
- streaming runtime concepts
- event time
- watermarks
- checkpoints

---

## Sprint 18 — Streaming Transformations v1

### Goal
Support useful real-time transformations.

### Features
- streaming filter
- streaming derive
- streaming map
- streaming deduplication
- tumbling windows
- hopping windows
- window aggregates
- upsert sink
- stream to current-state materialization

### Deliverables
- users can build simple streaming aggregations
- results can be materialized into sink tables

### Learning Focus
- windows
- real-time aggregation
- event-time correctness
- stream materialization

### Data Warehouse Concepts
- real-time facts
- stateful aggregates
- streaming snapshots

---

## Sprint 19 — Streaming Joins and History-Aware Enrichment

### Goal
Support more advanced real-time warehouse behavior.

### Features
- stream-to-stream join
- stream-to-dimension lookup
- history-aware enrichment
- temporal lookup behavior
- state retention policies
- late event handling
- exactly-once vs at-least-once runtime option

### Deliverables
- streaming enrichment pipelines are possible
- correctness behavior is visible in configuration

### Learning Focus
- stateful streaming
- temporal semantics
- exactly-once vs at-least-once
- retention and state cost

---

## Sprint 20 — Production Hardening

### Goal
Make the platform ready for real team usage.

### Features
- backfill framework
- rerun with version pinning
- deployment rollback
- blue/green deployment for pipelines
- quota management
- concurrency controls
- tenancy hardening
- disaster recovery plan
- metadata backup/restore
- support playbook

### Deliverables
- platform supports safer production operations
- recovery and rollback paths are documented and tested

### Learning Focus
- operational maturity
- reproducibility
- safe rollout patterns

---

# 8. MVP Definition

## MVP Scope
The smallest version worth building should include:

- pipeline builder
- pipeline save/load
- connection management
- batch execution
- PostgreSQL/MySQL/API/SFTP sources
- PostgreSQL/object storage sinks
- select/rename/filter/cast/derive/deduplicate transforms
- full refresh
- append
- upsert
- SCD1
- SCD2
- run history
- logs
- basic validation

## Explicitly Not in MVP
- streaming joins
- advanced Flink features
- connector marketplace
- enterprise approval workflows
- advanced lineage
- accumulating snapshot
- temporal streaming enrichment

---

# 9. Suggested Learning Track for You

Follow this order while building:

1. DAG and orchestration basics
2. source/sink connector design
3. batch ETL execution
4. grain and dimensional modeling
5. append vs upsert
6. dimensions and SCD
7. snapshots and semi-additive facts
8. joins and aggregation correctness
9. data quality and lineage
10. CDC and changelog thinking
11. event time and stateful streaming

---

# 10. Feature Priority by Business Value

## Highest Priority
- visual pipeline builder
- pipeline save/versioning
- batch execution
- connection management
- full refresh
- append
- upsert
- logs and run history

## Medium Priority
- SCD1
- SCD2
- periodic snapshots
- joins
- aggregates
- data quality
- lineage

## Later Priority
- accumulating snapshots
- CDC
- streaming windows

---

# 11. Folder Structure

```
Data-Platform/
├── docs/
│   ├── planning.md
│   ├── architecture.md                # Sprint 0 deliverable
│   ├── api/                           # OpenAPI specs
│   │   ├── control-plane.yaml
│   │   ├── connector-plane.yaml
│   │   └── execution-plane.yaml
│   ├── schemas/
│   │   └── pipeline-graph.json        # Pipeline JSON schema (Sprint 0)
│   └── erd/                           # Entity relationship diagrams
│
├── frontend/                          # Next.js app
│   ├── public/
│   ├── src/
│   │   ├── app/                       # Next.js app router
│   │   │   ├── (auth)/                # Login / signup routes
│   │   │   ├── (dashboard)/
│   │   │   │   ├── projects/
│   │   │   │   │   └── [id]/
│   │   │   │   │       └── settings/
│   │   │   │   │           └── data-layers/
│   │   │   │   ├── connections/
│   │   │   │   ├── pipelines/
│   │   │   │   │   └── [id]/
│   │   │   │   │       ├── builder/   # React Flow canvas
│   │   │   │   │       └── runs/
│   │   │   │   ├── datasets/
│   │   │   │   ├── lineage/
│   │   │   │   └── settings/
│   │   │   └── layout.tsx
│   │   ├── components/
│   │   │   ├── ui/                    # shadcn/ui primitives
│   │   │   ├── pipeline-builder/      # React Flow nodes, edges, panels
│   │   │   │   ├── nodes/             # Source, sink, transform node components
│   │   │   │   ├── edges/
│   │   │   │   ├── panels/            # Side config panels per node type
│   │   │   │   └── toolbar/
│   │   │   ├── connections/
│   │   │   ├── settings/
│   │   │   │   └── data-layer-editor/
│   │   │   ├── runs/
│   │   │   ├── data-quality/
│   │   │   └── lineage/
│   │   ├── hooks/
│   │   ├── lib/
│   │   │   ├── api/                   # API client
│   │   │   ├── graph/                 # DAG validation, cycle detection
│   │   │   └── schema/               # Schema propagation (client-side)
│   │   ├── stores/                    # State management
│   │   └── types/                     # TypeScript types mirroring domain model
│   ├── tailwind.config.ts
│   ├── next.config.ts
│   ├── package.json
│   └── tsconfig.json
│
├── backend/                           # Go services
│   ├── cmd/
│   │   ├── control-plane/
│   │   │   └── main.go
│   │   ├── execution-plane/
│   │   │   └── main.go
│   │   └── connector-plane/
│   │       └── main.go
│   ├── internal/
│   │   ├── domain/                    # Core domain models (no external deps)
│   │   │   ├── project.go
│   │   │   ├── environment.go
│   │   │   ├── pipeline.go
│   │   │   ├── pipeline_version.go
│   │   │   ├── node.go
│   │   │   ├── edge.go
│   │   │   ├── connection.go
│   │   │   ├── secret.go
│   │   │   ├── dataset.go
│   │   │   ├── run.go
│   │   │   ├── run_step.go
│   │   │   ├── deployment.go
│   │   │   ├── connector_definition.go
│   │   │   └── warehouse/
│   │   │       ├── table_role.go
│   │   │       ├── load_semantic.go
│   │   │       ├── history_semantic.go
│   │   │       ├── freshness_mode.go
│   │   │       └── grain.go
│   │   ├── controlplane/
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   └── repository/
│   │   ├── executionplane/
│   │   │   ├── compiler/              # Graph → execution plan compiler
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   └── repository/
│   │   ├── connectorplane/
│   │   │   ├── registry/              # Connector definitions & capabilities
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   └── repository/
│   │   ├── auth/
│   │   ├── middleware/
│   │   ├── config/
│   │   └── platform/
│   │       ├── database/
│   │       ├── redis/
│   │       └── observability/
│   ├── migrations/
│   ├── go.mod
│   └── go.sum
│
├── orchestration/                     # Prefect flows (Python)
│   ├── flows/
│   │   ├── batch_pipeline.py
│   │   ├── extract.py
│   │   ├── transform.py
│   │   └── load.py
│   ├── connectors/
│   │   ├── sources/
│   │   │   ├── postgres.py
│   │   │   ├── mysql.py
│   │   │   ├── rest_api.py
│   │   │   └── sftp.py
│   │   ├── sinks/
│   │   │   ├── postgres.py
│   │   │   └── object_storage.py
│   │   └── base.py
│   ├── transforms/
│   │   ├── select.py
│   │   ├── rename.py
│   │   ├── cast.py
│   │   ├── derive.py
│   │   ├── filter.py
│   │   ├── deduplicate.py
│   │   ├── join.py
│   │   ├── aggregate.py
│   │   └── base.py
│   ├── loaders/
│   │   ├── full_refresh.py
│   │   ├── append.py
│   │   ├── upsert.py
│   │   ├── scd1.py
│   │   ├── scd2.py
│   │   ├── periodic_snapshot.py
│   │   ├── accumulating_snapshot.py
│   │   └── base.py
│   ├── quality/
│   │   ├── row_count.py
│   │   ├── null_check.py
│   │   ├── uniqueness.py
│   │   ├── freshness.py
│   │   └── reconciliation.py
│   ├── utils/
│   ├── pyproject.toml
│   └── requirements.txt
│
├── streaming/                         # Future: Kafka + Flink (Sprint 16+)
│   ├── jobs/
│   └── connectors/
│
├── connector-specs/                   # Connector manifests (Sprint 15+)
│   ├── postgres/
│   │   ├── manifest.yaml
│   │   └── config-schema.json
│   ├── mysql/
│   ├── rest-api/
│   └── sftp/
│
├── deploy/
│   ├── docker/
│   │   ├── Dockerfile.frontend
│   │   ├── Dockerfile.backend
│   │   ├── Dockerfile.orchestration
│   │   └── docker-compose.yml
│   ├── k8s/
│   └── terraform/
│
├── scripts/
│   ├── seed.sh
│   └── migrate.sh
│
├── .github/
│   └── workflows/
│
├── .gitignore
├── Makefile
└── README.md
```

---

# 12. Architecture Decisions

## 12.1 Monorepo

All three planes (control, execution, connector) live in one repo. A top-level `Makefile` ties build/test/dev commands together. Each plane can run as a separate service or be composed into one binary during early development.

## 12.2 Secrets Strategy

| Phase | Approach |
|-------|----------|
| MVP (Sprint 3–5) | AES-256 encrypted column in PostgreSQL, encryption key in env var |
| Production (Sprint 14+) | Migrate to HashiCorp Vault or cloud secret manager |

Never store plaintext passwords in metadata DB, logs, or job configs. Decryption happens in-memory at runtime only.

## 12.3 Connection Storage

Connector **definitions** (what fields a connector needs, what capabilities it has) are hardcoded in early sprints, then move to declarative `connector-specs/` files in Sprint 15.

Connection **instances** (actual user-provided values) are always stored in the database:

```sql
CREATE TABLE connections (
    id              UUID PRIMARY KEY,
    project_id      UUID REFERENCES projects(id),
    environment_id  UUID REFERENCES environments(id),
    connector_type  TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    config          JSONB NOT NULL,       -- non-sensitive: host, port, database
    encrypted_creds BYTEA,                -- AES-256 encrypted: password, tokens
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);
```

## 12.4 Data Layers (Bronze / Silver / Gold)

Users configure logical data layers at the **project level** (not per-pipeline). Each layer maps to a connection + schema/path. The pipeline graph inherently creates the layers — extract lands in Bronze, transforms produce Silver, load semantics write to Gold.

Skip explicit layer config for Sprints 1–6. Add lightweight project-level layer settings from Sprint 7+ when users start having multiple destinations.

```json
{
  "data_layers": [
    { "name": "raw",     "default_connection_id": "conn-s3",       "default_schema": "raw" },
    { "name": "curated", "default_connection_id": "conn-s3",       "default_schema": "curated" },
    { "name": "serving", "default_connection_id": "conn-redshift", "default_schema": "analytics" }
  ]
}
```

## 12.5 Execution Migration Path

| Layer | Start Simple | Migrate Later |
|-------|-------------|---------------|
| Secrets | Encrypted column in PostgreSQL | HashiCorp Vault |
| Orchestration | Prefect with Docker workers | Kubernetes workers |
| Execution | Prefect tasks (Python) | Spark / Flink |
| Connectors | Hardcoded in code | Pluggable manifest system |
| Storage | PostgreSQL + MinIO | Iceberg analytical tables |
| Data movement | Batch only | CDC → Streaming |
| Load patterns | Full refresh → Append | Upsert → SCD → Snapshots |
| Observability | Structured logs | Prometheus + Grafana + OpenLineage |
| Auth | Basic auth/session | RBAC + environment isolation |
| Deployment | Docker Compose | Kubernetes + Terraform |

---

# 13. Feature Task List

Checklist of every task per sprint. Use this to track progress.

---

## Sprint 0 — Architecture and Scope Freeze

### Architecture
- [ ] Write architecture document (control plane / execution plane / connector plane)
- [ ] Create entity relationship diagram for all domain models
- [ ] Define pipeline graph JSON schema
- [ ] Define API boundaries between planes (OpenAPI specs)
- [ ] Define naming conventions for DB tables, API routes, Go packages

### Domain Model
- [ ] Define all 13 core entities (Project, Environment, Pipeline, PipelineVersion, Node, Edge, Connection, SecretReference, Dataset, Run, RunStep, Deployment, ConnectorDefinition)
- [ ] Define warehouse semantic types (table_role, grain, load_semantic, history_semantic, freshness_mode)
- [ ] Design PostgreSQL metadata schema (initial migration files)

### Runtime Design
- [ ] Define how graph compiles to execution plan
- [ ] Define execution strategy for batch jobs (Prefect flow structure)
- [ ] Define connector config standard (what fields, what types)

### Repo Setup
- [ ] Create initial repo structure (frontend/, backend/, orchestration/, deploy/, docs/)
- [ ] Set up Makefile with build/test/dev/migrate targets
- [ ] Set up .gitignore
- [ ] Set up CI pipeline (.github/workflows)

---

## Sprint 1 — Project Setup and Platform Skeleton

### Frontend
- [ ] Scaffold Next.js app with TypeScript
- [ ] Install and configure Tailwind CSS + shadcn/ui
- [ ] Create app layout (sidebar, header, content area)
- [ ] Create login page
- [ ] Create signup page
- [ ] Create project list page
- [ ] Create project creation form
- [ ] Create environment list page (inside project)
- [ ] Create environment creation form
- [ ] Create empty pipeline list page
- [ ] Set up API client (lib/api/)
- [ ] Set up auth state management (stores/)
- [ ] Set up route guards (redirect to login if unauthenticated)

### Backend
- [ ] Scaffold Go project with module structure (cmd/, internal/)
- [ ] Set up HTTP router and middleware
- [ ] Set up PostgreSQL connection pool
- [ ] Set up Redis connection
- [ ] Set up config loading (env vars / config file)
- [ ] Set up structured logging
- [ ] Create health check endpoint
- [ ] Implement auth: registration endpoint
- [ ] Implement auth: login endpoint
- [ ] Implement auth: session/token management
- [ ] Implement auth: middleware (validate token on every request)
- [ ] Create projects table migration
- [ ] Create environments table migration
- [ ] Create audit_logs table migration
- [ ] Implement CRUD API: projects
- [ ] Implement CRUD API: environments
- [ ] Implement audit log writes on create/update/delete
- [ ] Multi-tenant filtering (all queries scoped by project/org)

### Infrastructure
- [ ] Create docker-compose.yml (PostgreSQL, Redis, frontend, backend)
- [ ] Create Dockerfile.frontend
- [ ] Create Dockerfile.backend
- [ ] Write seed script (scripts/seed.sh)
- [ ] Write migrate script (scripts/migrate.sh)

---

## Sprint 2 — Visual Pipeline Builder MVP

### Frontend
- [ ] Install React Flow
- [ ] Create pipeline builder page (/pipelines/[id]/builder)
- [ ] Implement canvas with React Flow (zoom, pan, grid)
- [ ] Create source node component
- [ ] Create sink node component
- [ ] Create transform node component (generic)
- [ ] Implement drag-and-drop to add nodes from toolbar
- [ ] Implement add/remove edges between nodes
- [ ] Implement node selection (click to select)
- [ ] Create side panel that opens on node select
- [ ] Side panel: show node type and basic config fields
- [ ] Implement save pipeline button → POST to API
- [ ] Implement load pipeline → GET from API and render on canvas
- [ ] Implement graph validation: show cycle detection errors
- [ ] Implement graph validation: show disconnected node warnings
- [ ] Show validation errors as inline badges on nodes
- [ ] Create pipeline list page with saved pipelines

### Backend
- [ ] Create pipelines table migration
- [ ] Create pipeline_versions table migration
- [ ] Create nodes table migration (or store as JSONB in pipeline_version)
- [ ] Create edges table migration (or store as JSONB in pipeline_version)
- [ ] Implement API: create pipeline
- [ ] Implement API: get pipeline (with latest version graph)
- [ ] Implement API: update pipeline (save new draft version)
- [ ] Implement API: list pipelines (for project)
- [ ] Implement API: delete pipeline
- [ ] Implement graph validation service (cycle detection, orphan checks)
- [ ] Pipeline version draft model (draft vs published)

---

## Sprint 3 — Connection and Secret Management

### Frontend
- [ ] Create connections list page
- [ ] Create connection form (connector type selector)
- [ ] Hardcoded form fields for PostgreSQL (host, port, database, user, password)
- [ ] Hardcoded form fields for MySQL
- [ ] Hardcoded form fields for REST API (base_url, auth_type, headers)
- [ ] Hardcoded form fields for SFTP (host, port, username, key/password)
- [ ] Password fields: masked input, never displayed after save
- [ ] Test connection button → call API and show result
- [ ] Connection list: show status (tested/untested)
- [ ] Environment selector on connection form (which env this belongs to)
- [ ] In pipeline builder: source/sink node config panel → connection dropdown
- [ ] In pipeline builder: show connection capabilities (source/sink flags)

### Backend
- [ ] Create connections table migration (with encrypted_creds BYTEA column)
- [ ] Implement AES-256 encryption/decryption utility for credentials
- [ ] Implement API: create connection (encrypt sensitive fields before storing)
- [ ] Implement API: get connection (return config without decrypted creds)
- [ ] Implement API: update connection
- [ ] Implement API: delete connection
- [ ] Implement API: list connections (for project + environment)
- [ ] Implement API: test connection (decrypt creds, attempt real connection, return result)
- [ ] Define source/sink capability flags per connector type
- [ ] Implement simple data preview endpoint (first N rows from source)
- [ ] Scope connections to project + environment

---

## Sprint 4 — Batch Execution Foundation

### Backend
- [ ] Implement graph-to-execution-plan compiler (topological sort, dependency resolution)
- [ ] Create runs table migration
- [ ] Create run_steps table migration
- [ ] Implement API: trigger pipeline run (manual)
- [ ] Implement API: get run status
- [ ] Implement API: get run logs
- [ ] Implement API: list runs (for pipeline)
- [ ] Implement API: cancel run
- [ ] Implement job submission service → sends execution plan to Prefect

### Orchestration
- [ ] Set up Prefect project (pyproject.toml, requirements.txt)
- [ ] Create base batch_pipeline flow (receives execution plan, runs steps in order)
- [ ] Implement step executor (runs one step, reports status back)
- [ ] Implement retry logic (configurable retries per step)
- [ ] Implement log collection (capture stdout/stderr per step)
- [ ] Implement status callbacks to backend (running, success, failed, cancelled)
- [ ] Set up Docker worker for local execution
- [ ] Implement cancellation handling (stop running flow)

### Frontend
- [ ] Create run history page (/pipelines/[id]/runs)
- [ ] Show run list with status (pending, running, success, failed)
- [ ] Create run detail page (show steps with individual status)
- [ ] Show logs per step (streaming or polling)
- [ ] Add "Run pipeline" button on builder page
- [ ] Show run progress indicator
- [ ] Add cancel run button

---

## Sprint 5 — Batch Data Movement MVP

### Orchestration — Source Connectors
- [ ] Implement base source connector interface (base.py)
- [ ] Implement PostgreSQL source (JDBC/psycopg2: full table extract)
- [ ] Implement MySQL source (full table extract)
- [ ] Implement REST API source (paginated GET, JSON parsing)
- [ ] Implement SFTP source (download file, parse CSV/JSON)
- [ ] Each source: schema discovery (return column names + types)
- [ ] Each source: row count tracking

### Orchestration — Sink Connectors
- [ ] Implement base sink connector interface (base.py)
- [ ] Implement PostgreSQL sink (write to target table)
- [ ] Implement object storage sink (write to S3/MinIO as Parquet/CSV)
- [ ] Each sink: write confirmation (rows written, bytes)

### Orchestration — Pipeline Integration
- [ ] Wire source connectors into extract flow step
- [ ] Wire sink connectors into load flow step
- [ ] Implement staging (extract to local file/memory, then load)
- [ ] Implement error handling for failed extracts (retry, skip, fail)
- [ ] Implement error handling for failed loads
- [ ] Store schema metadata after successful extract
- [ ] Store row counts in run_step results

### Frontend
- [ ] Source node config panel: table selector (after connection is picked)
- [ ] Sink node config panel: target table name input
- [ ] Run detail page: show source schema captured
- [ ] Run detail page: show row counts per step
- [ ] Run detail page: show extract/load step details separately

---

## Sprint 6 — Basic Batch Transformations

### Orchestration — Transform Implementations
- [ ] Implement base transform interface (base.py)
- [ ] Implement select columns transform
- [ ] Implement rename columns transform
- [ ] Implement cast types transform
- [ ] Implement derive column transform (expression-based)
- [ ] Implement filter rows transform
- [ ] Implement sort transform
- [ ] Implement limit transform
- [ ] Implement deduplicate transform
- [ ] Each transform: input schema → output schema propagation

### Frontend
- [ ] Create transform node subtypes in toolbar (select, rename, cast, derive, filter, sort, limit, deduplicate)
- [ ] Select columns panel: checkbox list of available columns
- [ ] Rename columns panel: old name → new name mapping
- [ ] Cast types panel: column → target type mapping
- [ ] Derive column panel: new column name + expression input
- [ ] Filter panel: column + operator + value
- [ ] Sort panel: column + direction
- [ ] Limit panel: number input
- [ ] Deduplicate panel: select key columns
- [ ] Schema propagation: show output columns on each node (auto-calculated)
- [ ] Node preview button → call preview endpoint → show sample data table

### Backend
- [ ] Implement schema propagation service (given input schema + transform config → output schema)
- [ ] Implement preview endpoint (run partial pipeline up to a node, return sample rows)

---

## Sprint 7 — Load Semantics v1: Full Refresh and Append

### Backend
- [ ] Add load_semantic field to sink node config
- [ ] Add table_role field to sink node config
- [ ] Add grain field to sink node config
- [ ] Validate load_semantic per sink node before execution

### Orchestration — Load Strategies
- [ ] Implement base loader interface (base.py)
- [ ] Implement full_refresh loader (truncate + insert, or drop + create + insert)
- [ ] Implement append loader (insert only, no dedup by default)
- [ ] Implement partitioning options (partition by column/date)
- [ ] Implement write disposition settings (fail if exists, replace, append)
- [ ] Implement duplicate detection option for append mode
- [ ] Implement audit columns injection (load_timestamp, batch_id)
- [ ] Implement load summary metrics (rows inserted, duration)

### Frontend
- [ ] Sink node config panel: load mode dropdown (full_refresh, append)
- [ ] Sink node config panel: table role dropdown (fact, dimension, reference)
- [ ] Sink node config panel: grain input (comma-separated columns)
- [ ] Sink node config panel: audit columns toggle
- [ ] Sink node config panel: partitioning config
- [ ] Run detail page: show load summary (rows inserted, load mode used)

### Data Layer Config (lightweight)
- [ ] Create project settings page for data layers
- [ ] Data layer editor: add/remove/edit layers (name, connection, schema/path)
- [ ] Sink node config panel: optional layer dropdown (auto-fills connection + schema)
- [ ] API: save data layer config per project
- [ ] API: get data layers for project

---

## Sprint 8 — Load Semantics v2: Upsert / Merge

### Orchestration
- [ ] Implement upsert loader (merge/upsert based on business key)
- [ ] Support primary key / business key selection in config
- [ ] Implement insert vs update rules (update all columns, or only changed)
- [ ] Implement delete handling options (soft delete, hard delete, ignore)
- [ ] Implement conflict resolution strategy
- [ ] Implement idempotency checks (re-run same batch safely)
- [ ] Generate merge SQL per sink type (PostgreSQL ON CONFLICT, etc.)
- [ ] Implement restatement support (re-process a date range)
- [ ] Merge summary: inserted/updated/deleted counts

### Frontend
- [ ] Sink node config panel: upsert mode option
- [ ] Sink node config panel: business key column selector
- [ ] Sink node config panel: update rule config
- [ ] Sink node config panel: delete handling option
- [ ] Run detail page: show merge summary (inserted/updated/deleted)

---

## Sprint 9 — Dimension Modeling: SCD1 and SCD2

### Orchestration
- [ ] Implement SCD1 loader (overwrite changed rows, no history)
- [ ] Implement SCD2 loader (insert new version, close old version)
- [ ] SCD2: surrogate key generation
- [ ] SCD2: valid_from / valid_to column management
- [ ] SCD2: is_current flag management
- [ ] SCD2: hash-based change detection (hash of tracked columns)
- [ ] Natural key → surrogate key mapping
- [ ] Late change handling (update to already-closed version)

### Frontend
- [ ] Sink node config panel: history_semantic dropdown (none, scd1, scd2)
- [ ] Sink node — when dimension + SCD2: show surrogate key config
- [ ] Sink node — when dimension + SCD2: show tracked columns selector
- [ ] Sink node — when dimension + SCD2: show valid_from/valid_to column names
- [ ] Run detail page: show SCD2 summary (new versions, closed versions)

---

## Sprint 10 — Snapshot Modeling

### Orchestration
- [ ] Implement periodic_snapshot loader
- [ ] Snapshot key definition (what grain + what period)
- [ ] Snapshot frequency metadata (daily, weekly, monthly)
- [ ] Current snapshot option (keep only latest, or keep all)
- [ ] Snapshot partitioning (by snapshot date)
- [ ] Snapshot retention rules (keep last N snapshots)
- [ ] Compare current vs previous snapshot (diff row counts, value changes)
- [ ] Snapshot completeness checks (all expected keys present)
- [ ] Backfill support (generate snapshots for past dates)

### Frontend
- [ ] Sink node config panel: periodic_snapshot mode
- [ ] Sink node config panel: snapshot frequency selector
- [ ] Sink node config panel: snapshot key columns
- [ ] Sink node config panel: retention policy input
- [ ] Run detail page: show snapshot comparison summary

---

## Sprint 11 — Join, Aggregate, and Reusable Dataset Layer

### Orchestration
- [ ] Implement inner join transform
- [ ] Implement left join transform
- [ ] Implement union transform
- [ ] Implement group by + aggregate transform
- [ ] Aggregate functions: count, sum, avg, min, max, count_distinct
- [ ] Implement basic SQL node (user writes raw SQL)
- [ ] Join key validation (types must match)
- [ ] Schema compatibility checks for union

### Frontend
- [ ] Join node: select left/right inputs, join keys, join type
- [ ] Aggregate node: select group-by columns, add aggregate expressions
- [ ] Union node: show schema compatibility status
- [ ] SQL node: code editor with syntax highlighting
- [ ] Reusable dataset node: select from previously saved datasets
- [ ] Explain plan preview (show execution order before running)

### Backend
- [ ] Implement reusable dataset concept (a pipeline output that can be referenced)
- [ ] API: list available datasets for a project
- [ ] API: get dataset schema

---

## Sprint 12 — Accumulating Snapshot and Workflow Facts

### Orchestration
- [ ] Implement accumulating_snapshot loader
- [ ] Milestone field definitions (configurable milestone columns)
- [ ] Process state transition tracking
- [ ] Row update across lifecycle (same row updated multiple times)
- [ ] Duration metrics calculation between milestones
- [ ] SLA metric generation (time between milestones vs threshold)
- [ ] Milestone completeness validation

### Frontend
- [ ] Sink node config: accumulating_snapshot mode
- [ ] Sink node config: milestone columns editor
- [ ] Sink node config: SLA thresholds per milestone pair
- [ ] Run detail page: show milestone update summary

---

## Sprint 13 — Data Quality, Validation, and Reconciliation

### Orchestration
- [ ] Implement row count check (min/max/exact)
- [ ] Implement null check (column must not have nulls, or max % nulls)
- [ ] Implement uniqueness check (column or composite key must be unique)
- [ ] Implement referential check (foreign key exists in reference table)
- [ ] Implement freshness check (most recent timestamp within threshold)
- [ ] Implement schema drift alert (columns added/removed/type-changed vs baseline)
- [ ] Implement source vs target reconciliation (row count match, checksum)
- [ ] Quality check result model (pass/warn/fail + details)
- [ ] Configurable failure thresholds (warn at X%, fail at Y%)

### Frontend
- [ ] Quality rules editor per node (add checks, configure thresholds)
- [ ] Run detail page: show quality results per step (pass/warn/fail badges)
- [ ] Run detail page: quality summary table with all check results
- [ ] Pipeline list: show last run quality status

### Backend
- [ ] Create data_quality_results table migration
- [ ] API: get quality results for a run
- [ ] API: get quality history for a dataset

---

## Sprint 14 — Observability and Governance

### Backend
- [ ] Integrate OpenLineage (emit lineage events on run completion)
- [ ] Build dataset lineage graph from stored run metadata
- [ ] API: get lineage graph for a dataset (upstream/downstream)
- [ ] API: get run metrics (duration, rows, cost estimate)
- [ ] Implement pipeline ownership (owner user/team)
- [ ] Implement tags on pipelines and datasets
- [ ] Implement search (pipelines, datasets, connections by name/tag)
- [ ] Implement RBAC: roles (admin, editor, viewer)
- [ ] Implement RBAC: permission checks on all API endpoints
- [ ] Implement environment promotion flow (dev → staging → prod)
- [ ] Implement approval workflow for production deployments

### Frontend
- [ ] Lineage graph page (visual DAG of datasets)
- [ ] Run metrics dashboard (duration trends, row count trends)
- [ ] Pipeline detail: ownership and tags editor
- [ ] Search page: search across all entities
- [ ] Settings: role management page
- [ ] Environment promotion UI (promote pipeline version to next env)
- [ ] Approval workflow UI (request/approve/reject)

---

## Sprint 15 — Connector Framework v2

### Connector Specs
- [ ] Define connector manifest format (manifest.yaml)
- [ ] Define config schema format per connector (config-schema.json)
- [ ] Create manifest for PostgreSQL connector
- [ ] Create manifest for MySQL connector
- [ ] Create manifest for REST API connector
- [ ] Create manifest for SFTP connector
- [ ] Define capability registry structure

### Backend
- [ ] Implement connector registry (load manifests, serve to frontend)
- [ ] API: list available connector types (from registry)
- [ ] API: get connector config schema (for UI form generation)
- [ ] API: get connector capabilities
- [ ] Implement versioned connector definitions

### Frontend
- [ ] Auto-generate connection form from config-schema.json (dynamic form)
- [ ] Replace hardcoded connector forms with dynamic form renderer
- [ ] Show connector capabilities from registry

### Orchestration
- [ ] Refactor connectors to load config schema at runtime
- [ ] Implement connector packaging pattern (each connector is self-contained)

---

## Sprint 16 — CDC Foundation

### Orchestration
- [ ] Implement CDC connector abstraction
- [ ] Define change event model (insert/update/delete + before/after)
- [ ] Implement insert/update/delete event handling
- [ ] Implement offset storage (track last read position)
- [ ] Implement checkpoint metadata
- [ ] Implement replay capability (re-read from a past offset)
- [ ] Implement deduplication support for change events
- [ ] Implement target changelog application mode (apply changes to target table)

### Backend
- [ ] Create cdc_offsets table migration
- [ ] API: get CDC status for a connection
- [ ] API: reset CDC offset

### Frontend
- [ ] Source node config: CDC mode toggle
- [ ] CDC status page (current offset, lag, last event time)

---

## Sprint 17 — Streaming Foundation

### Orchestration / Streaming
- [ ] Set up Kafka infrastructure (docker-compose addition)
- [ ] Implement streaming job deployment service
- [ ] Implement Kafka topic abstraction
- [ ] Integrate message schema registry
- [ ] Implement event-time metadata support
- [ ] Implement watermark configuration
- [ ] Implement streaming run state model (running, paused, failed)
- [ ] Implement checkpoint monitoring

### Frontend
- [ ] Long-running job status page
- [ ] Streaming pipeline builder mode (real-time nodes)
- [ ] Event-time and checkpoint settings panel

---

## Sprint 18 — Streaming Transformations v1

### Streaming
- [ ] Implement streaming filter
- [ ] Implement streaming derive/map
- [ ] Implement streaming deduplication
- [ ] Implement tumbling windows
- [ ] Implement hopping windows
- [ ] Implement window aggregate functions
- [ ] Implement upsert sink for streaming
- [ ] Implement stream-to-current-state materialization

### Frontend
- [ ] Streaming transform nodes in builder
- [ ] Window configuration panel (type, size, slide)
- [ ] Materialization target config

---

## Sprint 19 — Streaming Joins and Enrichment

### Streaming
- [ ] Implement stream-to-stream join
- [ ] Implement stream-to-dimension lookup
- [ ] Implement history-aware enrichment (temporal join)
- [ ] Implement temporal lookup behavior config
- [ ] Implement state retention policies
- [ ] Implement late event handling strategies
- [ ] Implement exactly-once vs at-least-once runtime option

### Frontend
- [ ] Streaming join node config (join window, key mapping)
- [ ] Dimension lookup node config
- [ ] Late event handling config
- [ ] Delivery guarantee selector

---

## Sprint 20 — Production Hardening

### Backend
- [ ] Implement backfill framework (re-run pipeline for a date range)
- [ ] Implement rerun with version pinning (run exact version that was deployed)
- [ ] Implement deployment rollback (revert to previous pipeline version)
- [ ] Implement blue/green deployment for pipelines
- [ ] Implement quota management (max concurrent runs, max pipelines)
- [ ] Implement concurrency controls (pipeline-level locks)
- [ ] Tenancy hardening (data isolation audit)
- [ ] Implement metadata backup/restore
- [ ] Write disaster recovery plan

### Frontend
- [ ] Backfill UI (date range picker, run backfill)
- [ ] Deployment history with rollback button
- [ ] Quota usage dashboard
- [ ] Admin settings: concurrency limits

### Documentation
- [ ] Write support playbook
- [ ] Write runbook for common failure scenarios
- [ ] Document backup/restore procedures
- temporal joins
- enterprise workflows

---

# 11. Recommended Repo Structure

## Frontend
- pipeline canvas
- node config UI
- run history UI
- connection UI
- auth/project UI

## Backend
- api gateway
- pipeline service
- connection service
- compiler service
- execution service
- run metadata service
- lineage service

## Workers
- batch runner
- connector runtime
- later streaming runtime bridge

## Shared
- schemas
- event contracts
- connector contracts
- pipeline IR definitions

---
