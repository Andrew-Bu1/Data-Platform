# Sprint 14 — Observability and Governance

**Theme:** Data lineage, RBAC, data catalog, environment promotion

---

## Goal

Make the platform usable for teams, not just individual users. By the end of this sprint data lineage is queryable (which pipelines read from and write to which datasets), RBAC is enforced on all API endpoints, tags and search work across all entities, and pipeline versions can be promoted from dev to staging to production through an approval workflow.

---

## Concepts

### Data Lineage

- **Dataset-level lineage** — which pipeline produced dataset X? Which pipelines consume it? Expressed as a directed graph of datasets and pipelines.
- **Column-level lineage** — where does column X in dataset Y come from? Which source columns contributed to it, and what transforms were applied? Much harder to compute but much more valuable for debugging.
- **OpenLineage** — an open standard for capturing and exchanging lineage metadata. Defines a JSON event format that tools like dbt, Spark, and Airflow can emit. The format is `RunEvent` (pipeline started/completed) → `InputDataset` → `OutputDataset`.
- How to emit: at run completion, the execution plane emits an OpenLineage `RunEvent` to a collector endpoint. The collector stores it; the control plane can query lineage from the collector.
- For MVP: emit simplified lineage events to a lineage table in PostgreSQL (no external OpenLineage server required). Store: `source_dataset_id`, `target_dataset_id`, `pipeline_version_id`, `run_id`, `created_at`.

### Lineage Graph Query

Given a dataset D, find:
- **Upstream** — all datasets that D was built from (recursive: upstream of upstream).
- **Downstream** — all datasets that consume D (recursive: downstream of downstream).

This is a recursive graph traversal. In PostgreSQL:
```sql
WITH RECURSIVE upstream AS (
    SELECT source_dataset_id, target_dataset_id, 1 AS depth
    FROM dataset_lineage
    WHERE target_dataset_id = :dataset_id
    UNION ALL
    SELECT l.source_dataset_id, l.target_dataset_id, u.depth + 1
    FROM dataset_lineage l
    JOIN upstream u ON l.target_dataset_id = u.source_dataset_id
    WHERE u.depth < 10  -- prevent infinite recursion
)
SELECT * FROM upstream;
```

### RBAC — Role-Based Access Control

- **Roles**: `admin`, `editor`, `viewer`.
  - `admin` — full access: manage users, manage environments, promote to production, delete pipelines.
  - `editor` — create and edit pipelines, run pipelines, manage connections in non-production environments.
  - `viewer` — read-only: view pipelines, runs, datasets, lineage.
- **Scope**: roles are assigned per project (a user can be an editor in project A and a viewer in project B).
- **Implementation**: add a `project_members` table with `user_id`, `project_id`, `role`. Every API endpoint checks the current user's role for the relevant project before executing.
- **Resource ownership** — pipelines, connections, and datasets have an `owner_id` field. Owners always have edit access regardless of project role (optional, adds complexity).
- Middleware approach in Go:
  ```go
  func requireRole(minRole Role) func(http.Handler) http.Handler {
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              user := userFromContext(r.Context())
              projectID := projectIDFromPath(r)
              if !hasRole(user, projectID, minRole) {
                  writeJSON(w, http.StatusForbidden, ErrorResponse{"insufficient permissions"})
                  return
              }
              next.ServeHTTP(w, r)
          })
      }
  }
  ```

### Environment Promotion Workflow

- A pipeline is developed in `dev`, tested in `staging`, and promoted to `prod` when ready.
- **Promotion** = deploy the same pipeline version (or a new one) to the next environment.
- In each environment, the pipeline version's connections are resolved against that environment's connection registry.
- Promotion steps:
  1. Developer marks a pipeline version as "ready for promotion."
  2. (Optional) An approver (admin or senior editor) reviews and approves.
  3. The platform deploys the version to the target environment.
- **Approval workflow**: `pending_approval → approved → deployed` or `pending_approval → rejected`. Store in a `deployment_requests` table.
- Why this matters: prevents accidental pipeline changes from reaching production without review. A data platform that can corrupt production data needs guardrails.

### Tags and Search

- Tags are free-form string labels on pipelines, datasets, and connections. Example: `["finance", "pii", "critical"]`.
- Store tags as a `TEXT[]` column in PostgreSQL. Index with GIN: `CREATE INDEX idx_pipelines_tags ON pipelines USING GIN(tags)`.
- Search: full-text search using PostgreSQL `tsvector` and `tsquery`, or a simple `ILIKE %term%` for MVP.
- Search should cover: pipeline names, dataset names, connection names, and tags.

### Prometheus + Grafana for Platform Metrics

- Emit metrics from the Go API using `prometheus/client_golang`:
  - `data_platform_runs_total{status}` — counter per run status
  - `data_platform_run_duration_seconds{pipeline_id}` — histogram of run durations
  - `data_platform_rows_processed_total{pipeline_id, step_type}` — counter of rows
- Expose metrics on `/metrics` endpoint.
- Add Prometheus + Grafana to docker-compose. Create a dashboard showing: run success rate, P95 run duration, rows processed per hour.

---

## Tasks

### Backend
- [ ] Create `dataset_lineage` table migration (`source_dataset_id`, `target_dataset_id`, `pipeline_version_id`, `run_id`, `created_at`)
- [ ] Emit lineage events at run completion (source nodes → target nodes)
- [ ] `GET /v1/datasets/:id/lineage` — upstream and downstream graph using recursive CTE
- [ ] Create `project_members` table migration (if not already from Sprint 1) with `role` enum
- [ ] `POST /v1/projects/:id/members` — add member with role
- [ ] `PUT /v1/projects/:id/members/:userId` — change role
- [ ] `DELETE /v1/projects/:id/members/:userId` — remove member
- [ ] RBAC middleware: `requireRole` applied to all pipeline/connection/run endpoints
- [ ] Tags: add `tags TEXT[]` to pipelines, datasets, connections with GIN index
- [ ] `GET /v1/search?q=&entity_type=` — full-text search across entities
- [ ] Create `deployment_requests` table migration and approval endpoints
- [ ] `POST /v1/pipelines/:id/promote` — create deployment request for environment promotion
- [ ] `PUT /v1/deployment-requests/:id/approve` / `reject` (admin only)
- [ ] Prometheus metrics endpoint (`/metrics`) — run counters, duration histograms, row counters

### Frontend
- [ ] Lineage graph page — visual DAG of datasets using React Flow (read-only)
- [ ] Dataset detail page: upstream/downstream dataset list
- [ ] Settings → Members page: add/remove/change role for project members
- [ ] Tags editor: inline tag input on pipeline, dataset, and connection cards
- [ ] Search page: text search box, entity type filter, results list
- [ ] Environment promotion UI: "Promote to staging/prod" button, shows current deployment status per environment
- [ ] Deployment approval UI: pending requests list, approve/reject buttons (admin only)
- [ ] Add Grafana to docker-compose and create a basic platform metrics dashboard

---

## Interview Topics

- **What is data lineage and why does a data platform need it?** Give an example of a bug you would find with lineage but not without it.
- **What is the difference between dataset-level and column-level lineage?** What makes column-level lineage harder to compute?
- **Explain RBAC vs ABAC (attribute-based access control).** When would you move from RBAC to ABAC?
- **Write a recursive CTE in PostgreSQL to traverse a graph.**
- **Why does an environment promotion workflow exist?** What can go wrong without one?
- **How do GIN indexes work in PostgreSQL?** What types of queries benefit from them?
- **What is OpenLineage?** Why is an open standard for lineage valuable?
- **Explain the four golden signals (latency, traffic, errors, saturation) and how each maps to a data platform.**

---

## Definition of Done

- [ ] Lineage graph is populated after every run and queryable via API
- [ ] Lineage graph page renders the upstream/downstream graph visually
- [ ] Viewer cannot trigger a run or edit a pipeline — 403 returned
- [ ] Editor cannot promote to production — 403 returned for admin-only operations
- [ ] Tags can be added to pipelines and are returned in search results
- [ ] Search returns results across pipeline names, dataset names, and tags
- [ ] Environment promotion creates a deployment request; admin can approve/reject
- [ ] Prometheus metrics are scraped by Prometheus and visible in Grafana dashboard
- [ ] RBAC tests: all three roles tested against all major endpoint categories
