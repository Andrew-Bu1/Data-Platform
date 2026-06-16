# Sprint 0 — Architecture and Scope Freeze

**Theme:** System design, domain modeling, API boundaries

---

## Goal

Lock the technical direction before writing any production code. By the end of this sprint you can explain every major architectural decision, draw the domain model from memory, describe how a visual pipeline graph compiles into a runtime execution plan, and defend why the three-plane architecture was chosen over alternatives.

---

## Concepts

### Control Plane vs Execution Plane vs Connector Plane

- **Control plane** — owns all persistent state: projects, pipelines, connections, secrets, runs, deployments. It never touches user data. Think of it as the brain.
- **Execution plane** — receives a compiled execution plan from the control plane and runs it. It touches user data (extracts, transforms, loads). Think of it as the hands.
- **Connector plane** — knows how to talk to external systems: PostgreSQL, REST APIs, S3. It is the adapter between the generic execution model and the specific protocol of each system.
- Why separate them: each plane has different scaling, security, and failure characteristics. The control plane runs always. The execution plane runs on demand. The connector plane can be extended without touching the control plane.

### DAG Theory — Directed Acyclic Graphs

- A **DAG** is a graph with directed edges and no cycles. Every data pipeline is a DAG.
- **Topological sort** — an ordering of nodes such that for every directed edge A→B, A appears before B. This is the execution order.
- **Cycle detection** — if a cycle exists, no topological ordering is possible. Use DFS with a visited/in-progress set (Kahn's algorithm or DFS coloring).
- Why acyclic: a cycle in a pipeline means node A depends on node B which depends on A — the pipeline can never start.
- **Fan-out** — one node feeds multiple downstream nodes (broadcast).
- **Fan-in** — multiple upstream nodes feed one node (join, union).
- **Critical path** — the longest path through the DAG, which determines the minimum total execution time.

### Pipeline as Intermediate Representation (IR)

- The visual canvas produces a **pipeline graph** — a JSON document describing nodes and edges.
- The graph is not directly executable. It must be **compiled** into an execution plan.
- The execution plan is a topologically sorted list of steps, each with concrete config (connection credentials resolved, schema resolved, execution parameters filled in).
- This separation is important: the graph is what the user designed; the plan is what the runtime runs. Versioning the graph lets you replay any historical run exactly.
- Analogy: source code (graph) → compiled binary (execution plan). The compiler validates and transforms.

### Domain Model Design

- **Project** — the top-level namespace. All resources belong to a project.
- **Environment** — a project can have multiple environments (dev, staging, prod). Connections are environment-specific. Pipelines are deployed per environment.
- **Pipeline** — the logical entity. Has a name, description, owner.
- **PipelineVersion** — an immutable snapshot of the graph at a point in time. Draft vs published. Execution always runs a version, not the live pipeline.
- **Node** — a vertex in the graph. Has a type (source, sink, transform), a connector reference, and a config JSON blob.
- **Edge** — a directed connection between two nodes. Records which output port connects to which input port (for nodes with multiple inputs like join).
- **Connection** — a saved set of credentials for an external system. Belongs to a project + environment.
- **SecretReference** — a pointer to an encrypted credential, never the plaintext value.
- **Run** — one execution of a pipeline version. Has status, start time, end time.
- **RunStep** — one node's execution within a run. Records row counts, schema captured, error messages.
- **Deployment** — the act of promoting a pipeline version to an environment. Tracks which version is active per environment.
- **Dataset** — a referenceable output of a pipeline node that other pipelines can consume.
- **ConnectorDefinition** — the metadata about a connector type (what fields it needs, what capabilities it has: source? sink? both?).

### Warehouse Semantic Types

These belong on the sink node config, not the pipeline itself:

- **table_role** — `fact`, `dimension`, `reference`, `bridge`. Describes what the table represents in the warehouse.
- **load_semantic** — `full_refresh`, `append`, `upsert`, `periodic_snapshot`, `accumulating_snapshot`. Describes how new data is written to the target.
- **history_semantic** — `none`, `scd1`, `scd2`, `scd3`. Describes how historical changes to rows are tracked.
- **freshness_mode** — `batch`, `micro_batch`, `stream`. Describes how often the table is updated.
- **grain** — the set of columns that uniquely identifies one row at the table's level of detail.

### API Boundary Design

- The control plane exposes a **REST API** consumed by the frontend and by the execution plane (callbacks).
- The execution plane exposes a minimal **internal API** for job status callbacks.
- The connector plane is not a separate service early on — it is a library used by the execution plane.
- OpenAPI 3.0 is the contract format. Generate types from specs (do not write them by hand on both sides).
- gRPC is worth knowing conceptually — understand protobuf, service definition, streaming vs unary — but REST is fine for this project until you need high-throughput internal communication.

### Pipeline Graph JSON Schema

The canonical format for a stored pipeline graph:

```json
{
  "schema_version": "1",
  "nodes": [
    {
      "id": "node-1",
      "type": "source",
      "connector_type": "postgres",
      "label": "Orders table",
      "position": { "x": 100, "y": 200 },
      "config": { "table": "orders", "connection_id": "conn-abc" }
    },
    {
      "id": "node-2",
      "type": "transform",
      "transform_type": "filter",
      "label": "Active only",
      "position": { "x": 300, "y": 200 },
      "config": { "column": "status", "operator": "eq", "value": "active" }
    }
  ],
  "edges": [
    { "id": "edge-1", "source": "node-1", "target": "node-2", "source_handle": "out", "target_handle": "in" }
  ]
}
```

### Monorepo Structure Decisions

- One repo, three planes. Each plane is independently deployable.
- `apps/` — runnable services (api, orchestrator, frontend).
- `packages/` — shared code (types, schemas, config).
- Top-level `Makefile` for `make dev`, `make migrate`, `make test`.
- Docker Compose for local development. Kubernetes for production (later).

---

## Tasks

- [ ] Write architecture document (`docs/architecture.md`) covering the three planes, data flow, and deployment topology
- [ ] Draw entity relationship diagram for all domain models
- [ ] Define pipeline graph JSON schema (`docs/schemas/pipeline-graph.json`)
- [ ] Define OpenAPI specs for control plane, execution plane, and connector plane (`docs/api/`)
- [ ] Define naming conventions: DB table names (snake_case), Go packages (short, lowercase), API routes (`/v1/projects/:id`)
- [ ] Define how graph compiles to execution plan — write a one-page design doc
- [ ] Define connector config standard (required fields, type system, validation rules)
- [ ] Create initial repo structure (`apps/`, `packages/`, `infra/`, `docs/`)
- [ ] Set up top-level Makefile with `dev`, `test`, `migrate`, `lint` targets
- [ ] Set up `.gitignore` and CI workflow skeleton (`.github/workflows/ci.yml`)
- [ ] Create sprint backlog for Sprint 1 as GitHub issues or a task list

---

## Interview Topics

- **What is a DAG and why must a data pipeline be acyclic?** Explain topological sort and why a cycle makes execution impossible.
- **Explain the difference between a pipeline graph and an execution plan.** Why is compiling from one to the other valuable?
- **What is a control plane?** Compare to Kubernetes: the API server is the control plane, kubelets are the execution plane.
- **How would you version a pipeline?** Describe immutable versions, draft vs published, and why you always run a version rather than the live config.
- **What is the grain of a table?** Give an example of a fact table with a clear grain.
- **Why separate secrets from connection config?** Describe the encrypted_creds pattern — config is readable metadata, credentials are encrypted blobs decrypted only at runtime.
- **What is the difference between load_semantic and history_semantic?** Load semantic describes how rows are written; history semantic describes whether and how previous values are tracked.

---

## Definition of Done

- [ ] Architecture document exists and describes all three planes with a diagram
- [ ] Entity relationship diagram covers all 13 core domain entities
- [ ] Pipeline graph JSON schema is valid JSON Schema draft-07 or later
- [ ] OpenAPI specs define at minimum the core control plane endpoints (projects, pipelines, runs)
- [ ] Naming conventions document exists and is agreed
- [ ] Repo structure exists with correct folder hierarchy
- [ ] Makefile has working `dev` and `test` targets (even if they just echo for now)
- [ ] You can explain from memory how a drag-and-drop pipeline graph becomes a running Prefect flow
