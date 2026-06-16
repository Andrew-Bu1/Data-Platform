# Sprint 20 — Production Hardening

**Theme:** Backfill, rollback, multi-tenancy hardening, disaster recovery

---

## Goal

Make the platform safe and recoverable for real team usage. By the end of this sprint the platform supports backfill runs, pipeline version rollback, concurrency controls, and has documented disaster recovery procedures. This sprint is about operational maturity — turning a feature-complete platform into a production-ready one.

---

## Concepts

### Backfill Framework

- **Backfill** = re-running a pipeline for a past date range, typically to:
  - Populate historical data for a newly created pipeline.
  - Reprocess past data after a bug fix in the pipeline logic.
  - Re-ingest after a source system recovered from an outage.
- Backfill design requirements:
  1. **Parameterized by date** — the pipeline must accept `run_date` as a parameter (not read `datetime.now()`).
  2. **Idempotent** — running a backfill for the same date twice must produce the same result. Use full-refresh, upsert, or snapshot replace-by-date semantics.
  3. **Parallelizable** — for large date ranges, run multiple dates in parallel (up to a concurrency limit).
  4. **Resumable** — if a backfill job fails midway through a date range, it should be able to restart from the last successful date.
- Backfill execution: create one `Run` per backfill date. Group them under a `BackfillJob` entity. Track progress: how many dates completed, how many failed, which date is in progress.

### Pipeline Version Rollback

- Every run executes a specific `PipelineVersion`. The deployment records which version is "active" per environment.
- **Rollback** = set a previous version as the active deployment. Future runs will use the rolled-back version.
- Rollback is safe because pipeline versions are immutable — you cannot change a published version. Rolling back simply points the deployment to an earlier version.
- Rollback procedure:
  1. Find the previous successful deployment version ID.
  2. Create a new deployment record pointing to that version with status = `active`.
  3. Mark the current deployment as `rolled_back`.
- **Blue/green pipeline deployment** — instead of atomically switching versions, run both versions briefly in parallel and compare outputs. If outputs match (or the new version looks better), complete the cutover. If not, discard the green version.

### Concurrency Controls

- Pipelines may have multiple triggers (scheduled + manual). Prevent overlapping runs:
  - **Pipeline-level concurrency lock** — while a run is in progress for pipeline X, a new trigger for X is rejected (or queued).
  - **Environment-level lock** — a production environment may allow only one pipeline run at a time.
- Implementation: use a `UNIQUE` constraint or `SELECT FOR UPDATE SKIP LOCKED` advisory lock in PostgreSQL.
  ```sql
  -- Acquire a pipeline-level advisory lock before starting a run
  SELECT pg_try_advisory_xact_lock(:pipeline_id_hash);
  -- Returns TRUE if lock acquired, FALSE if another run holds it
  ```
- **Max concurrent runs** — a quota per project: "no more than 5 runs in progress simultaneously." Track in-progress run count and reject if over limit.

### Multi-Tenancy Hardening

- Review every query in the codebase: does it always filter by `project_id`?
- **Cross-tenant query test** — create two projects with identical pipeline names. Query the API as user A and confirm user B's pipeline is never returned.
- **Row-level security (RLS)** in PostgreSQL — enforce tenant isolation at the database level, not just the application level. Even if a bug in the application code forgets a WHERE clause, RLS prevents data leakage.
  ```sql
  ALTER TABLE pipelines ENABLE ROW LEVEL SECURITY;
  CREATE POLICY pipelines_isolation ON pipelines
      USING (project_id = current_setting('app.current_project_id')::UUID);
  ```
  Set `app.current_project_id` in each database session from the application layer.
- **Secret isolation** — confirm that decrypting credentials for project A never accidentally uses a key from project B. Key version must be tied to the project.

### Quota Management

- **Quotas** protect shared infrastructure from runaway usage:
  - Max concurrent runs per project
  - Max pipeline count per project
  - Max run duration (auto-cancel runs that exceed N hours)
  - Max rows processed per run
- Store quotas in a `project_quotas` table with defaults. Allow admins to override per project.
- Quota enforcement: check before launching a run; auto-cancel runs that exceed duration.

### Metadata Backup and Restore

- The PostgreSQL metadata database (projects, pipelines, connections, runs) is the single source of truth. If lost, the platform is unusable.
- **Backup strategy**:
  - Continuous WAL archiving to S3/MinIO (Point-in-Time Recovery).
  - Daily full `pg_dump` to object storage. Retain 30 days.
  - Test restore monthly (a backup you have never tested is not a backup).
- **Restore procedure**: documented runbook, tested in staging. Includes: restore from dump, run any outstanding migrations, verify data integrity.
- **Encrypted credentials** — if the encryption key is lost, all credentials are permanently inaccessible. Treat the encryption key with the same backup rigor as the database.

### Disaster Recovery Concepts

- **RPO (Recovery Point Objective)** — how much data can be lost? If RPO = 1 hour, the backup must be at most 1 hour old.
- **RTO (Recovery Time Objective)** — how long can the platform be down? If RTO = 30 minutes, recovery must complete in 30 minutes.
- For a data platform: RPO = 0 for run metadata (use WAL streaming replication). RPO for actual data depends on source system — if sources are intact, re-running pipelines recovers the data.
- **Data loss vs service loss**: losing the control plane database means you cannot see run history or manage pipelines. The underlying data in sources and sinks is unaffected. Distinguish between these.

---

## Tasks

### Backend
- [ ] Create `backfill_jobs` table migration (`id`, `pipeline_id`, `start_date`, `end_date`, `status`, `completed_dates`, `failed_dates`, `created_at`)
- [ ] `POST /v1/pipelines/:id/backfill` — create and start a backfill job (start_date, end_date, concurrency)
- [ ] `GET /v1/backfill-jobs/:id` — get backfill progress
- [ ] `POST /v1/backfill-jobs/:id/cancel` — cancel in-progress backfill
- [ ] Pipeline deployment rollback: `POST /v1/deployments/:id/rollback` — roll back to previous version
- [ ] Deployment history page: `GET /v1/pipelines/:id/deployments` — list all versions deployed per environment
- [ ] Pipeline advisory lock: `pg_try_advisory_xact_lock` before triggering a run; return 409 if locked
- [ ] Create `project_quotas` table with defaults; enforce on run trigger
- [ ] Auto-cancel runs exceeding max duration (cron job or scheduled Prefect check)
- [ ] PostgreSQL Row-Level Security policies for `pipelines`, `connections`, `runs` tables
- [ ] Cross-tenant query tests: automated test that confirms project B data is never returned to project A user

### Orchestrator
- [ ] Backfill flow: given date range, run batch pipeline for each date with `run_date` parameter
- [ ] Resumable backfill: track completed dates; on restart, skip already-completed dates
- [ ] Parallel backfill: configurable concurrency (default: 3 concurrent dates)
- [ ] `run_date` parameter threading: all source extracts and snapshot loaders use `run_date` instead of `datetime.now()`

### Frontend
- [ ] Backfill UI: date range picker, concurrency slider, progress display (N/M dates completed)
- [ ] Deployment history page: list of deployed versions per environment with rollback button
- [ ] Rollback confirmation dialog: "This will roll back environment X from version Y to version Z"
- [ ] Quota usage display on project settings page
- [ ] Admin settings page: edit project quotas

### Documentation
- [ ] Write `docs/runbooks/backup-restore.md` — step-by-step PostgreSQL restore procedure
- [ ] Write `docs/runbooks/incident-response.md` — what to do when a pipeline fails in production
- [ ] Write `docs/runbooks/key-rotation.md` — how to rotate the encryption key
- [ ] Disaster recovery plan: RPO/RTO targets, which components need replication, recovery order

---

## Interview Topics

- **What is a backfill?** How do you design a pipeline to be backfill-safe?
- **What makes a pipeline idempotent?** Give examples for full refresh, upsert, and snapshot patterns.
- **Explain RPO and RTO.** If you had to choose only one to optimize, which would you pick for a data platform and why?
- **What is PostgreSQL Row-Level Security?** How does it differ from application-level tenant filtering?
- **Explain `SELECT FOR UPDATE SKIP LOCKED`.** What problem does it solve in a job queue?
- **What is `pg_try_advisory_xact_lock`?** How would you use it to prevent concurrent pipeline runs?
- **What is the difference between a blue/green deployment for an application vs a blue/green deployment for a data pipeline?**
- **If the encryption key for stored credentials was lost, what can you recover?** What cannot be recovered?

---

## Definition of Done

- [ ] Backfill runs a date range with parallel execution and tracks progress
- [ ] Backfill is resumable: cancelled and restarted, it skips already-completed dates
- [ ] Pipeline rollback sets a previous version as active and subsequent runs use it
- [ ] Advisory lock prevents two simultaneous runs for the same pipeline — test by triggering twice quickly
- [ ] Row-Level Security: automated test confirms cross-tenant query returns empty results
- [ ] Quota enforcement: project that exceeds max concurrent runs gets 429 on additional trigger
- [ ] Backup runbook is written and the restore procedure has been tested in a local environment
- [ ] `run_date` parameter works correctly: backfill for past date does not use `datetime.now()` anywhere
