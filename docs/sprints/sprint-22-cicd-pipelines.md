# Sprint 22 — CI/CD Pipelines (On-Prem)

**Theme:** GitHub Actions, Docker, container registry, Helm deployments to k3s

---

## Goal

Automate every step from code push to deployment on the on-prem k3s cluster. By the end of this sprint a merged pull request automatically builds Docker images, pushes them to a self-hosted registry, and deploys to the cluster via Helm. No deployment is triggered by running a local command — everything flows through GitHub Actions.

---

## Concepts

### CI/CD Fundamentals

- **CI (Continuous Integration)** — automatically build and test every code change. Catches regressions before they merge.
- **CD (Continuous Delivery)** — automatically deploy every merged change to a lower environment. Humans approve production promotion.
- **Pipeline as code** — CI/CD config lives in the repository alongside application code. Changes to the pipeline go through the same PR review as application changes.
- For this sprint: CI runs on every PR (lint, test, build). CD runs on merge to `main` and deploys to the k3s cluster.

### GitHub Actions Architecture

- **Workflow** — a YAML file in `.github/workflows/`. Triggered by events (`push`, `pull_request`, `workflow_dispatch`).
- **Job** — a unit of work that runs on a runner. Jobs in the same workflow run in parallel by default. Use `needs:` to sequence them.
- **Step** — a single command or action within a job. Steps within a job run sequentially.
- **Runner** — the machine that executes jobs. Two options for this sprint:
  - **GitHub-hosted** (`ubuntu-latest`) — free for public repos, 2000 minutes/month for private. Cannot reach your local k3s directly.
  - **Self-hosted runner** — a process running on your VPS or dev machine. Has direct access to k3s. Use this for deploy jobs.
- **Action** — a reusable step package (e.g., `actions/checkout`, `docker/build-push-action`).
- **Secret** — an encrypted value stored in GitHub repository settings. Injected as environment variables at runtime.

### Self-Hosted Runner

- Install the GitHub Actions runner agent on the same machine as k3s.
- The runner polls GitHub for jobs and executes them locally — no inbound firewall rule needed.
- Label the runner (e.g., `self-hosted, k3s`) and target it in workflows with `runs-on: [self-hosted, k3s]`.
- The runner has direct `kubectl` access to the cluster — no need to expose the k3s API publicly.
- **Security**: the runner runs with the permissions of the OS user it is installed under. Use a dedicated low-privilege user. Never use root.

### Self-Hosted Container Registry

- For on-prem: run a private Docker registry inside k3s instead of using ECR or Docker Hub.
- **Distribution** (formerly Docker Registry v2) — lightweight registry, runs as a single container.
- Or use **Gitea Container Registry** / **Harbor** for a more feature-complete option.
- The self-hosted runner pushes to the registry. k3s pulls from it. Both are on the same network — no internet transfer.
- k3s needs to trust the registry. For HTTP (no TLS): configure k3s `registries.yaml`:
  ```yaml
  mirrors:
    registry.local:5000:
      endpoint:
        - "http://registry.local:5000"
  ```
- Tag strategy: `registry.local:5000/api:sha-<git-sha>`.

### Docker Multi-Stage Builds

- **Multi-stage builds** — use a builder stage (with full SDK) and a runtime stage (minimal image). Keeps the final image small and secure.
  ```dockerfile
  FROM golang:1.23 AS builder
  WORKDIR /app
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -o api ./cmd/api

  FROM gcr.io/distroless/static
  COPY --from=builder /app/api /api
  ENTRYPOINT ["/api"]
  ```
- **Layer caching** — copy dependency files (`go.mod`, `pyproject.toml`) before source code. If dependencies don't change, Docker reuses the cached download layer.
- **Image tagging**:
  - `sha-<git-sha>` — immutable. Tied to a specific commit. Always use this for deployments.
  - `latest` — mutable. Never use in Helm values or deployment manifests.

### Helm in CI

- Deploy step: build new image → push to registry → `helm upgrade --install` with the new image tag.
  ```bash
  helm upgrade --install api ./charts/api \
    --set image.tag=sha-${{ github.sha }} \
    --values infra/helm/api/values.yaml \
    --atomic --timeout 3m
  ```
- `--atomic` — rolls back automatically if pods fail to become ready. CI fails fast with a clear error.
- `--wait` — waits until all pods are ready before the step succeeds (implied by `--atomic`).
- **Helm release history** — `helm history api` shows all revisions. CI posts the new revision number to the job summary.

### Workflow Structure

```
.github/workflows/
  ci.yml          → runs on every PR: lint, test, build (no push)
  deploy.yml      → runs on merge to main: build, push, helm upgrade
  rollback.yml    → manual trigger: helm rollback to previous revision
```

### Branch Protection and Merge Gates

- Configure GitHub branch protection on `main`:
  - Require CI (`ci.yml`) to pass before merge.
  - Require at least one PR review.
  - No direct pushes to `main`.
- This ensures broken code never reaches the cluster.

### Makefile as Local CI Mirror

- The `Makefile` should mirror CI commands so developers can run the same checks locally before pushing.
- `make lint` → same linter commands as CI.
- `make test` → same test commands as CI.
- `make build` → same Docker build commands as CI.
- If it passes locally it should pass in CI — no "works on my machine" surprises.

---

## Tasks

### Self-Hosted Runner
- [ ] Create a dedicated OS user (`github-runner`) on the VPS/dev machine
- [ ] Install GitHub Actions runner agent and register it with the repo (label: `self-hosted, k3s`)
- [ ] Verify runner appears as online in GitHub → Settings → Actions → Runners
- [ ] Ensure runner user has `kubectl` access to k3s (kubeconfig at `~/.kube/config`)
- [ ] Ensure runner user has Docker access (member of `docker` group)

### Container Registry
- [ ] Deploy Docker Registry v2 as a k3s Deployment in the `platform` namespace
- [ ] Expose registry on a NodePort or via Ingress (`registry.local`)
- [ ] Configure k3s `registries.yaml` to trust the local registry
- [ ] Test: `docker push registry.local:5000/test:latest` succeeds, `kubectl run test --image=registry.local:5000/test:latest` pulls it

### Dockerfiles
- [ ] Write `apps/api/Dockerfile` — multi-stage Go build, distroless runtime image
- [ ] Write `apps/orchestrator/Dockerfile` — multi-stage Python build with `uv`, slim runtime image
- [ ] Local test: both images build and run with `docker compose up`

### CI Workflow (`ci.yml`)
- [ ] Trigger: `pull_request` to `main`, runs on `ubuntu-latest` (GitHub-hosted)
- [ ] Job: `lint-api` — `go vet ./...`, `golangci-lint run`
- [ ] Job: `test-api` — `go test ./...` with Postgres as a service container
- [ ] Job: `lint-orchestrator` — `ruff check`, `mypy`
- [ ] Job: `test-orchestrator` — `pytest` with service containers
- [ ] Job: `build-images` — `docker build` for api and orchestrator (no push, just verify build succeeds)
- [ ] Cache: Go module cache and Docker layer cache via `actions/cache`
- [ ] All jobs must pass before PR can be merged (branch protection rule)

### Deploy Workflow (`deploy.yml`)
- [ ] Trigger: `push` to `main`, runs on `[self-hosted, k3s]`
- [ ] Build `api` image and push to local registry tagged `sha-${{ github.sha }}`
- [ ] Build `orchestrator` image and push to local registry tagged `sha-${{ github.sha }}`
- [ ] `helm upgrade --install --atomic` for api with new image tag
- [ ] `helm upgrade --install --atomic` for orchestrator with new image tag
- [ ] Post deploy summary to GitHub Actions job summary: image tags deployed, Helm revision number
- [ ] Verify after deploy: `kubectl rollout status deployment/api` returns success

### Rollback Workflow (`rollback.yml`)
- [ ] Trigger: `workflow_dispatch` with input: `service` (api | orchestrator), `revision` (Helm revision number)
- [ ] Runs on `[self-hosted, k3s]`
- [ ] `helm rollback <service> <revision>`
- [ ] Post result to job summary

### Makefile
- [ ] `make lint` — run all linters (Go + Python)
- [ ] `make test` — run all tests with local service containers
- [ ] `make build` — build all Docker images locally
- [ ] `make push TAG=sha-xxx` — push images to local registry
- [ ] `make deploy TAG=sha-xxx` — helm upgrade all services with given tag
- [ ] `make rollback SERVICE=api REVISION=3` — helm rollback

### Documentation
- [ ] Write `docs/runbooks/deploy.md` — how to trigger a deploy, how to verify it succeeded
- [ ] Write `docs/runbooks/rollback.md` — how to roll back a bad deploy using the rollback workflow

---

## Interview Topics

- **What is a self-hosted GitHub Actions runner?** Why would you use one instead of GitHub-hosted runners?
- **Explain Docker multi-stage builds.** What is the security benefit of using a distroless final image?
- **Why tag Docker images with the git SHA instead of `latest`?** What goes wrong with mutable tags?
- **What does `helm upgrade --atomic` do?** What is the failure mode without it?
- **What is branch protection?** How does requiring CI to pass before merge improve reliability?
- **What is the difference between Continuous Delivery and Continuous Deployment?** Which does this setup implement?
- **If a deploy fails in CI, what is the state of the cluster?** How does `--atomic` protect you?
- **How would you debug a failing GitHub Actions step?** Walk through the process from seeing a red check to identifying the root cause.
- **What is `helm rollback`?** How is it different from re-running a deploy with an old image tag?
- **Why mirror CI commands in a Makefile?** What problem does it solve for developers?

---

## Definition of Done

- [ ] Self-hosted runner shows as online in GitHub — a test workflow dispatched manually completes successfully
- [ ] Images build in CI: every PR triggers a build job that succeeds
- [ ] Lint and test pass in CI: a PR with a failing test is blocked from merging
- [ ] Merging to `main` automatically builds and pushes images to the local registry
- [ ] Helm deploy runs after push — `kubectl get pods` shows new pods with updated image SHA
- [ ] `helm rollback` workflow runs successfully via manual dispatch and reverts the deployment
- [ ] `make lint && make test` pass locally with identical results to CI
- [ ] Branch protection is configured — direct push to `main` is rejected
- [ ] Deploy runbook has been followed once end-to-end from a fresh terminal
