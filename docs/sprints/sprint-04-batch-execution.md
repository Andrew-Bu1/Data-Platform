# Sprint 4 — Batch Execution Foundation

**Theme:** Prefect orchestration, graph compiler, run lifecycle

---

## Goal

Run a pipeline end-to-end in batch mode. By the end of this sprint a user can click "Run pipeline" in the UI, the control plane compiles the graph into an execution plan, submits it to Prefect, and the run status and logs appear in real time in the UI. The pipeline does not move real data yet — that is Sprint 5. This sprint is about the execution infrastructure.

---

## Concepts

### Orchestration Theory — What an Orchestrator Does

- An **orchestrator** manages: scheduling (when to run), dependency resolution (in what order), retries (what to do on failure), and observability (what happened).
- **Orchestrator vs executor** — the orchestrator decides what to run and when. The executor actually runs the code (a worker process). Prefect separates these: the Prefect server is the orchestrator; Prefect workers are the executors.
- **Task** — a single unit of work in Prefect. Decorated with `@task`. Has its own retry config, timeout, caching.
- **Flow** — a Python function decorated with `@flow` that calls tasks. The flow defines the dependency graph between tasks. Prefect infers the graph from Python function calls.
- **Work pool and worker** — a work pool is a queue of flow runs. A worker polls the work pool for runs to execute. For local development, use a `Process` work pool (runs in a subprocess on the same machine).
- **Flow run** — one execution of a flow. Has a unique ID, state (scheduled, running, completed, failed), and run logs.

### Prefect vs Airflow (know both)

| Concept | Prefect | Airflow |
|---------|---------|---------|
| DAG definition | Python functions calling other functions | Python file with a `DAG` object, tasks connected with `>>` |
| When DAG is parsed | At runtime (dynamic) | At schedule time (static file scan) |
| Dynamic tasks | Native — loops, conditionals in Python | Limited — `dynamic_task_mapping` added later |
| Trigger method | Python API / REST / UI | `airflow dags trigger` / REST |
| UI | Prefect Cloud or self-hosted server | Airflow webserver |
| Scheduler | Pull model (worker polls) | Push model (scheduler pushes) |
| Interview frequency | Growing | Very high — most enterprise data teams use Airflow |

Know Airflow's core concepts for interviews even though this project uses Prefect: `DAG`, `Operator`, `Sensor`, `XCom`, `TaskInstance`, `DagRun`, scheduler internals.

### Graph Compilation — Pipeline Graph → Execution Plan

The compilation step transforms the visual graph into something the execution plane can run:

1. **Topological sort** the nodes (Kahn's algorithm: repeatedly remove nodes with in-degree 0).
2. For each node, **resolve config**: look up the connection, decrypt credentials, substitute environment variables.
3. Build a list of **execution steps** in order. Each step has:
   - `step_id` (maps to a node ID)
   - `step_type` (source / transform_filter / transform_join / sink / etc.)
   - `connector_type` (for source/sink nodes)
   - `config` (fully resolved, no aliases)
   - `depends_on` (list of upstream step IDs — for parallel execution later)
4. The execution plan JSON is passed to Prefect as a flow parameter. The Prefect flow reads the plan and executes steps in order.

The key insight: the control plane knows nothing about how to extract data or run transforms. It only knows how to compile graphs and track state. The execution plane knows how to run steps but does not hold configuration.

### Idempotency in Job Launching

- **Idempotent** means running the same operation multiple times produces the same result as running it once.
- A run trigger must be idempotent: if the user clicks "Run" twice, or if the webhook retries, only one run should execute.
- Use a **job key** (a hash of pipeline_version_id + trigger_time + idempotency_key). Check for an existing run with the same key before creating a new one.
- Prefect supports `idempotency_key` on flow run creation — use it.

### Run Status Lifecycle

```
PENDING → RUNNING → COMPLETED
                  → FAILED
                  → CANCELLED
```

- The control plane creates a `Run` row in status `PENDING` immediately when the user triggers.
- The execution plane calls back to the control plane API to update status as it progresses.
- Each `RunStep` follows the same state machine independently.
- The UI polls `GET /v1/runs/:id` every few seconds (or uses WebSocket / SSE for real-time updates).

### Python Patterns — Prefect Flows

- `@flow` and `@task` decorators. Tasks are cached by default — disable for data movement tasks.
- Pass the execution plan as a flow parameter (a Pydantic model). Prefect serializes it automatically.
- Use `get_run_logger()` inside flows/tasks to log structured messages that appear in the Prefect UI.
- Handle errors with try/except inside tasks — always call back to the control plane with the error before re-raising.
- `prefect.deployments.run_deployment` or direct Python API for programmatic triggering from Go (via HTTP to the Prefect API).

### Callbacks from Execution Plane to Control Plane

- The execution plane does not share a database with the control plane.
- Status updates flow via HTTP: the Prefect flow calls `POST /v1/internal/runs/:id/status` and `POST /v1/internal/run-steps/:id/status`.
- Use a shared secret (HMAC header) to authenticate internal callbacks — the execution plane is not a user and must not go through the JWT auth middleware.
- Design the callback endpoint to be idempotent: if the same status update arrives twice, it is a no-op.

---

## Tasks

### Backend
- [ ] Create `runs` table migration (`id`, `pipeline_id`, `pipeline_version_id`, `environment_id`, `status`, `triggered_by`, `idempotency_key`, `started_at`, `completed_at`, `error_message`, `created_at`)
- [ ] Create `run_steps` table migration (`id`, `run_id`, `node_id`, `step_type`, `status`, `started_at`, `completed_at`, `rows_in`, `rows_out`, `error_message`, `metadata` JSONB)
- [ ] Implement graph compiler service (`internal/compiler/`) — topological sort + config resolution
- [ ] `POST /v1/pipelines/:id/runs` — trigger a run (validate, compile, submit to Prefect, return run ID)
- [ ] `GET /v1/runs/:id` — get run with all steps and their statuses
- [ ] `GET /v1/pipelines/:id/runs` — list runs for a pipeline
- [ ] `POST /v1/runs/:id/cancel` — request cancellation
- [ ] Internal callback endpoints (HMAC-authenticated):
  - `POST /v1/internal/runs/:id/status`
  - `POST /v1/internal/run-steps/:id/status`
  - `POST /v1/internal/run-steps/:id/logs`
- [ ] Idempotency key check — if run with same key exists and is not failed, return existing run

### Orchestrator
- [ ] Set up Prefect project (`pyproject.toml`, configure Prefect API URL)
- [ ] Create `ExecutionPlan` Pydantic model (mirrors what the compiler produces)
- [ ] Create base `batch_pipeline` flow — receives `ExecutionPlan`, iterates steps in order
- [ ] Implement step dispatcher — routes each step to the correct task function by `step_type`
- [ ] Implement stub task for `source` — logs "extracting from {connector_type}" and sleeps 1s
- [ ] Implement stub task for `transform` — logs "transforming with {transform_type}"
- [ ] Implement stub task for `sink` — logs "loading to {connector_type}"
- [ ] Status callbacks to control plane API after each step (running → completed/failed)
- [ ] Run-level callbacks (run started, run completed/failed)
- [ ] Cancellation handling — poll for cancellation flag, raise `CancelledError`
- [ ] Set up Docker-based Prefect worker for local execution
- [ ] Log collection: all `get_run_logger()` messages persisted via callback

### Frontend
- [ ] "Run pipeline" button on builder page
- [ ] Run history page (`/pipelines/[id]/runs`) — list of runs with status badge and timestamps
- [ ] Run detail page — show steps with individual status, duration, row counts
- [ ] Log viewer per step — poll `GET /v1/run-steps/:id/logs`
- [ ] Cancel run button (shown for PENDING/RUNNING runs)
- [ ] Auto-refresh run status (poll every 3 seconds while status is PENDING or RUNNING)

---

## Interview Topics

- **Explain Prefect's work pool and worker model.** How does a worker know what to run?
- **What is the difference between an orchestrator and an executor?** Give examples of each.
- **How do you make a job trigger idempotent?** Describe the idempotency key approach.
- **Explain Kahn's algorithm for topological sort.** What is the time complexity?
- **What is the difference between Prefect `@task` and `@flow`?** When does a task get retried vs a full flow rerun?
- **How would you track run status from a Python worker to a Go API?** Describe the callback pattern with authentication.
- **In Airflow, what is an XCom?** Compare to passing data between Prefect tasks via return values.
- **What is a DAG run in Airflow vs a flow run in Prefect?** Compare terminology.

---

## Definition of Done

- [ ] Clicking "Run pipeline" creates a run in `PENDING` status and submits to Prefect
- [ ] Run transitions through `PENDING → RUNNING → COMPLETED` with correct timestamps
- [ ] Individual steps appear in the run detail page with their status
- [ ] Stub tasks log messages that appear in the step log viewer
- [ ] Cancelling a run transitions it to `CANCELLED` status
- [ ] Triggering the same pipeline twice quickly results in one run (idempotency key deduplication)
- [ ] Graph compiler tests: valid DAG produces correct topological order, circular graph raises error
- [ ] Callback endpoints reject requests without the correct HMAC header
