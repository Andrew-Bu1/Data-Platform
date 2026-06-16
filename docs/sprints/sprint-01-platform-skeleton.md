# Sprint 1 — Platform Skeleton

**Theme:** Go backend, Next.js frontend, authentication, multi-tenant foundation

---

## Goal

Stand up the full platform shell: a running Go API with PostgreSQL, a Next.js frontend, and working authentication. By the end of this sprint a user can register, log in, create a project, create an environment, and see an empty pipeline list. No pipelines yet — just the scaffold that everything else will sit on.

---

## Concepts

### Multi-Tenant Metadata Design

- **Multi-tenancy** means multiple organizations/users share the same application and database. Their data must be completely isolated.
- Two common approaches: **row-level tenancy** (all tables have an `org_id` or `project_id` column, queries always filter by it) vs **schema-per-tenant** (each tenant gets their own PostgreSQL schema). Row-level is simpler to start; schema-per-tenant has stronger isolation but is operationally heavier.
- In this platform: row-level tenancy scoped to `project_id` and `environment_id`.
- **Every query in the repository layer must include the tenant filter** — this is a source of serious security bugs if forgotten. Consider middleware that injects the current project ID into the Go context and a repository helper that reads it.
- **Audit logs** — every create/update/delete should record who did it and when. This is not optional in a data platform where pipelines change data.

### Go HTTP Patterns (review from Cloud-Platform)

- `net/http` ServeMux with Go 1.22+ method+path patterns: `mux.HandleFunc("POST /v1/projects", handler)`
- Middleware chaining as `func(http.Handler) http.Handler`
- Reading the authenticated user from context: define a typed context key to avoid collisions
- Structured logging: use `log/slog` (stdlib since Go 1.21) — always log with fields, never with `fmt.Sprintf` inside log calls
- Config loading: read from environment variables, use `godotenv` for local dev, never hardcode secrets

### PostgreSQL Schema Design Fundamentals

- **UUID primary keys** — use `uuid_generate_v4()` or `gen_random_uuid()` (pg 13+). UUIDs avoid enumeration attacks, scale across shards, and can be generated client-side.
- **`TIMESTAMPTZ` not `TIMESTAMP`** — always store timestamps with timezone. `TIMESTAMP` silently drops timezone info.
- **`JSONB` for flexible config** — use it for node configs, connection configs, anything that varies by type. Add a GIN index if you query inside the JSON.
- **Soft deletes** — add `deleted_at TIMESTAMPTZ` and filter with `WHERE deleted_at IS NULL`. Recoverable and auditable.
- **Foreign key constraints** — always define them. They prevent orphan rows and document relationships. Use `ON DELETE CASCADE` sparingly — prefer application-level deletion so you control what happens.
- **`golang-migrate`** — migration files named `000001_create_users.up.sql` / `000001_create_users.down.sql`. Run up migrations at startup programmatically. Never run down migrations in production without a plan.

### Next.js App Router

- `app/` directory with route segments as folders. `page.tsx` is the page component. `layout.tsx` wraps children.
- **Server Components vs Client Components** — server components run on the server, can fetch data directly, cannot use hooks or browser APIs. Client components use `'use client'` directive.
- **Route Groups** — `(auth)` and `(dashboard)` are route group folders. They affect layout nesting but not the URL.
- **API Client pattern** — create a typed HTTP client in `lib/api/` that wraps `fetch`. Centralize error handling and auth token injection there.
- **Auth state** — store JWT in an httpOnly cookie (more secure) or localStorage (simpler for a learning project). Use a context provider or Zustand store for the current user.
- **Route guards** — a layout component that checks auth state and redirects to `/login` if unauthenticated. In Next.js app router, this is done in a middleware (`middleware.ts`) or a layout server component.

### shadcn/ui and Tailwind

- shadcn/ui is not a component library you install — it is a CLI that copies component source code into your project. You own the components and can modify them.
- Components are built on Radix UI primitives (accessible, unstyled) + Tailwind CSS.
- `npx shadcn@latest add button card form input select` — add only what you need.
- Tailwind: utility-first CSS. No custom CSS files. Everything is class names. Learn `flex`, `grid`, `gap`, `p-`, `m-`, `text-`, `bg-`, `border-`, `rounded-`, `shadow-`.

### Session and Token Management

- **JWT (JSON Web Token)** — three base64url-encoded parts: header.payload.signature. The payload carries claims (`sub`, `exp`, `iat`, custom fields like `user_id`).
- Sign with HS256 (symmetric, one secret key) for simplicity. Use RS256 (asymmetric) when multiple services need to verify tokens independently.
- **Access token + refresh token** — access token is short-lived (15 min – 1 hour), refresh token is long-lived (7–30 days). This limits the window of compromise if an access token is stolen.
- For this project: a single access token stored in a cookie is fine for MVP. Add refresh tokens in the security sprint if you choose to do one.

---

## Tasks

### Backend
- [ ] Set up Go module structure (`cmd/api/`, `internal/`, `migrations/`)
- [ ] Configure HTTP server with graceful shutdown (`signal.NotifyContext`)
- [ ] Set up `pgxpool` from environment variables
- [ ] Run `golang-migrate` migrations automatically at startup
- [ ] Set up `log/slog` structured logging with request ID middleware
- [ ] Create migrations: `users`, `organizations`, `projects`, `environments`, `audit_logs`
- [ ] `POST /v1/auth/register` — bcrypt password, create user, return JWT
- [ ] `POST /v1/auth/login` — verify password, return JWT
- [ ] `GET /v1/auth/me` — return current user from JWT
- [ ] JWT auth middleware — validate token, inject user into context
- [ ] `POST /v1/projects` — create project (auto-create default dev environment)
- [ ] `GET /v1/projects` — list user's projects
- [ ] `GET /v1/projects/:id` — get project
- [ ] `DELETE /v1/projects/:id` — soft delete
- [ ] `POST /v1/projects/:id/environments` — create environment
- [ ] `GET /v1/projects/:id/environments` — list environments
- [ ] `GET /v1/health` — health check endpoint
- [ ] Multi-tenant filter: all repository queries scoped by project ID from context
- [ ] Audit log writes on every create/update/delete

### Frontend
- [ ] Scaffold Next.js app with TypeScript and app router
- [ ] Install Tailwind CSS and shadcn/ui
- [ ] Create app layout: sidebar + header + main content area
- [ ] Login page with email/password form
- [ ] Signup page with email/password form
- [ ] Auth middleware (`middleware.ts`) — redirect to login if unauthenticated
- [ ] Typed API client (`lib/api/client.ts`) with auth token injection
- [ ] Project list page — show all projects with create button
- [ ] Project creation dialog
- [ ] Environment list page inside a project
- [ ] Environment creation dialog
- [ ] Empty pipeline list page (no pipelines yet)

### Infrastructure
- [ ] Update `docker-compose.yml` — PostgreSQL, Redis, API service, frontend service
- [ ] `Dockerfile` for Go API and Next.js frontend
- [ ] `scripts/migrate.sh` and `scripts/seed.sh`

---

## Interview Topics

- **How do you prevent tenant data leakage in a row-level multi-tenant system?** Describe the repository pattern with tenant filter always applied, and the risk of missing a WHERE clause.
- **Why use UUIDs instead of auto-increment integers for primary keys?** Discuss enumeration attacks, distributed generation, and merge-from-replica scenarios.
- **Explain the difference between TIMESTAMP and TIMESTAMPTZ in PostgreSQL.** When would the wrong choice cause a bug?
- **What is the difference between an access token and a refresh token?** Why are access tokens short-lived?
- **What does `ON DELETE CASCADE` do and when should you avoid it?** Explain the risk of cascading deletes propagating further than expected.
- **What is an audit log and why is it non-negotiable in a data platform?** Discuss compliance, debugging, and change history.
- **In Next.js app router, what is the difference between a server component and a client component?** When must you use each?

---

## Definition of Done

- [ ] User can register and receive a JWT
- [ ] Protected endpoints return 401 without a valid token
- [ ] User can create, list, and delete a project
- [ ] User can create and list environments inside a project
- [ ] All queries are filtered by project ID — confirm with a test that crosses project boundaries and gets zero results
- [ ] Audit log records every create/update/delete with user ID and timestamp
- [ ] `GET /v1/health` returns 200
- [ ] Frontend login/signup works end-to-end with the real API
- [ ] `go vet ./...` and TypeScript type check (`tsc --noEmit`) both pass
- [ ] README has local setup instructions (`make dev`)
