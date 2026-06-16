# Sprint 21 — On-Prem Kubernetes

**Theme:** k3s, Helm, KEDA, local storage, full stack on a single node or VPS

---

## Goal

Deploy the full data platform stack on a self-managed Kubernetes cluster without any cloud provider. By the end of this sprint Airflow, Trino, Nessie, MinIO, and the control plane API are all running on k3s — reachable via Ingress, orchestrating real pipelines, reading and writing Iceberg tables to local object storage. No AWS account required.

---

## Concepts

### Why On-Prem First

- Running the full stack locally proves the architecture works before paying for cloud infrastructure.
- Forces you to understand every component — you cannot rely on AWS managed services to hide complexity.
- A working on-prem deployment is a strong portfolio demo: spin up on any machine, no account needed.
- Migration to AWS (Sprint 23) becomes a controlled variable swap: MinIO → S3, local Postgres → RDS, k3s → EKS.

### k3s vs k8s vs kind vs minikube

- **k3s** — lightweight production-grade Kubernetes distribution by Rancher. Single binary, SQLite or etcd backend. Runs on Linux VMs, Raspberry Pi, or your dev machine. Closest to real Kubernetes in behavior.
- **kind (Kubernetes in Docker)** — runs cluster nodes as Docker containers. Great for CI testing. Less suitable for persistent workloads (Airflow, MinIO) because storage is inside containers.
- **minikube** — single-node local cluster. Good for learning, less suitable for multi-service stacks due to resource limits.
- **Why k3s**: persistent volumes work naturally, Helm charts behave identically to EKS, and you can run it on a $20/mo VPS for a publicly accessible demo.

### k3s Architecture

- **Server node** — runs the Kubernetes control plane (API server, scheduler, controller manager) plus an embedded SQLite or etcd. For a single-node setup, the server is also a worker.
- **Agent node** — runs only the kubelet and kube-proxy. Add agent nodes to scale out.
- k3s includes: Traefik ingress, CoreDNS, local-path-provisioner (dynamic PV provisioning backed by host filesystem), Flannel CNI.
- Default storage class is `local-path` — creates PersistentVolumes as directories on the host node. Sufficient for a single-node dev setup.

### Helm

- **Helm** is the Kubernetes package manager. A **chart** is a templated collection of Kubernetes manifests.
- `helm repo add` — register a chart repository (e.g., Apache Airflow, Trino).
- `helm upgrade --install` — installs if not present, upgrades if already installed. Idempotent.
- `helm upgrade --atomic` — rolls back automatically if new pods fail to become ready within timeout.
- **Values files** — each service has a `values.yaml` that overrides chart defaults. Keep values files in `infra/helm/<service>/values.yaml`.
- `helm list` — show installed releases. `helm history <release>` — show all revisions. `helm rollback <release> <revision>` — roll back.

### MinIO as S3-Compatible Object Storage

- **MinIO** is an open-source object store that implements the S3 API. Drop-in replacement for AWS S3 in local and on-prem environments.
- Your Spark jobs, Trino, and Nessie all speak S3 protocol — point them at MinIO and they work without code changes.
- MinIO runs as a Kubernetes Deployment with a PersistentVolume for data. In a single-node setup, this is a local-path PV.
- **MinIO Console** — web UI for browsing buckets and objects. Expose via Ingress for local access.
- Buckets to create: `raw`, `iceberg`, `logs`.

### Nessie as Iceberg Catalog

- **Project Nessie** is a transactional catalog for Iceberg (and Delta) tables. It provides Git-like branching: create a branch, write tables, merge to main.
- Nessie stores catalog metadata (table locations, schemas, snapshots) in a backend database — use the in-cluster Postgres.
- Trino and Spark connect to Nessie via the Iceberg REST catalog interface or the Nessie-specific catalog implementation.
- **Why Nessie over Hive Metastore**: no separate Hive services needed, branching support, REST-native, simpler to deploy on Kubernetes.

### Trino on Kubernetes

- **Trino** is a distributed SQL query engine. It has a **coordinator** (parses queries, plans execution, manages workers) and one or more **workers** (execute query fragments).
- For a local setup: one coordinator + one worker is sufficient for dev/testing.
- Trino connects to Iceberg tables via a catalog config pointing at Nessie and MinIO.
- Trino is stateless — workers can be scaled up/down freely. No persistent storage needed.

### Airflow with KubernetesExecutor

- **KubernetesExecutor** — each Airflow task runs in its own ephemeral pod. No persistent workers sitting idle.
- Airflow components on k8s: `scheduler` (reads DAGs, creates task pods), `webserver` (UI), `triggerer` (async sensors).
- **DAG sync**: mount DAGs via a git-sync sidecar (polls a git repo and syncs files to a shared volume) or a ConfigMap (simpler for dev).
- **KEDA ScaledObject** — scales the Airflow scheduler or worker pods based on queue depth. On k3s, KEDA can watch the Airflow metadata DB task queue.

### KEDA (Kubernetes Event-Driven Autoscaler)

- **KEDA** adds event-driven scaling to Kubernetes. It extends the standard HPA with custom scalers (database query, Redis list length, message queue depth).
- **ScaledObject** — a CRD that defines: what to scale (a Deployment), the trigger (e.g., Airflow DB pending task count), min/max replicas, cooldown.
- **ScaledJob** — like ScaledObject but creates Jobs (run-to-completion) instead of scaling a Deployment. Useful for Airflow KubernetesExecutor where each task is a Job.
- On k3s, KEDA installs via Helm and works identically to EKS — this is a direct skill transfer to the cloud sprint.
- KEDA scales to zero: when no tasks are queued, worker replica count drops to 0. When tasks arrive, pods spin up within seconds.

### Ingress and Local DNS

- **Traefik** — k3s ships with Traefik as the default ingress controller. Exposes services via HTTP/HTTPS on the host's port 80/443.
- For local dev: add entries to `/etc/hosts` to map hostnames to `127.0.0.1`:
  ```
  127.0.0.1  airflow.local trino.local minio.local nessie.local api.local
  ```
- For a VPS: point a real domain or subdomain to the VPS IP. Traefik can auto-provision Let's Encrypt TLS.

### PersistentVolumes on k3s

- k3s ships with `local-path-provisioner` — dynamically creates PVs as directories under `/var/lib/rancher/k3s/storage/`.
- `StorageClass: local-path` is the default. PVCs using it get a directory on the host node.
- For production-grade on-prem: use **Longhorn** (distributed block storage, also by Rancher) which replicates data across nodes. Out of scope for this sprint but worth knowing.

---

## Tasks

### Cluster Setup
- [ ] Install k3s on local machine or VPS: `curl -sfL https://get.k3s.io | sh -`
- [ ] Configure `kubectl` to use k3s kubeconfig: `export KUBECONFIG=/etc/rancher/k3s/k3s.yaml`
- [ ] Verify cluster: `kubectl get nodes` returns one node in `Ready` state
- [ ] Add `/etc/hosts` entries for local ingress hostnames

### Namespace and Base Config
- [ ] Create namespaces: `platform`, `data`, `monitoring`
- [ ] Write `infra/k8s/namespaces.yaml` — namespace manifests
- [ ] Install KEDA via Helm: `helm install keda kedacore/keda -n keda --create-namespace`

### MinIO
- [ ] Install MinIO via Helm (`minio/minio` chart) into `data` namespace
- [ ] Configure PVC: 20Gi local-path volume
- [ ] Create buckets via MinIO init job: `raw`, `iceberg`, `logs`
- [ ] Expose MinIO API and Console via Traefik Ingress (`minio.local`, `minio-console.local`)
- [ ] Write `infra/helm/minio/values.yaml`

### Postgres (in-cluster)
- [ ] Install Postgres via Helm (`bitnami/postgresql`) into `platform` namespace
- [ ] Two databases: `platform` (control plane), `nessie` (catalog metadata)
- [ ] Store credentials in a Kubernetes Secret
- [ ] Write `infra/helm/postgres/values.yaml`

### Nessie
- [ ] Install Nessie via Helm (`projectnessie/nessie`) into `data` namespace
- [ ] Configure JDBC backend pointing to in-cluster Postgres `nessie` database
- [ ] Expose via Ingress (`nessie.local`)
- [ ] Write `infra/helm/nessie/values.yaml`

### Trino
- [ ] Install Trino via Helm (`trino/trino`) into `data` namespace
- [ ] Configure Iceberg catalog: Nessie catalog type, MinIO as S3 endpoint
- [ ] One coordinator + one worker
- [ ] Expose coordinator via Ingress (`trino.local`)
- [ ] Write `infra/helm/trino/values.yaml`
- [ ] Verify: `trino --server http://trino.local --catalog iceberg --schema default` connects and can `SHOW TABLES`

### Airflow
- [ ] Install Airflow via Helm (`apache-airflow/airflow`) into `platform` namespace with KubernetesExecutor
- [ ] Git-sync sidecar configured to pull DAGs from the repo
- [ ] Postgres connection: use in-cluster Postgres `platform` database
- [ ] Configure Airflow connections: MinIO (S3 hook), Trino, Nessie via environment Secrets
- [ ] Install KEDA ScaledJob for Airflow task queue
- [ ] Expose webserver via Ingress (`airflow.local`)
- [ ] Write `infra/helm/airflow/values.yaml`

### Control Plane API
- [ ] Write `Dockerfile` for the Go API (multi-stage, distroless final image)
- [ ] Write `infra/k8s/api/deployment.yaml` — Deployment, Service, Ingress for `api.local`
- [ ] Run migrations as a Kubernetes Job before API starts (init container or separate Job)
- [ ] Expose via Ingress (`api.local`)

### Validation
- [ ] End-to-end test: trigger an Airflow DAG that writes an Iceberg table to MinIO via Spark (or Trino INSERT), then query it via Trino
- [ ] KEDA test: submit 10 tasks, observe pods scaling up; wait for idle, observe pods scaling to zero
- [ ] Nessie branching test: create a branch, write a table, query from main (table not visible), merge branch, query from main (table visible)

---

## Interview Topics

- **What is k3s?** How does it differ from a full Kubernetes distribution? When would you choose it?
- **What is the KubernetesExecutor in Airflow?** How does it differ from the CeleryExecutor?
- **Explain KEDA's ScaledObject vs ScaledJob.** When would you use each for Airflow?
- **What is MinIO?** How does it implement S3 compatibility? What changes in your code when you swap MinIO for real S3?
- **What is Nessie?** What does "Git-like branching for data" mean in practice?
- **Explain how Traefik Ingress works.** What happens between a DNS lookup for `airflow.local` and the Airflow webserver responding?
- **What is a PersistentVolume and a PersistentVolumeClaim?** What is the difference between static and dynamic provisioning?
- **Why does KEDA scale to zero matter for a data platform?** What is the cost impact?
- **What is `helm upgrade --atomic`?** What is the failure mode without it?
- **If a Trino query fails because Nessie is unreachable, where do you look first?** Walk through the debugging steps.

---

## Definition of Done

- [ ] `kubectl get pods -A` shows all platform components running with no restarts
- [ ] Airflow UI accessible at `airflow.local` — can trigger a DAG manually
- [ ] Trino accessible at `trino.local` — `SHOW CATALOGS` returns `iceberg`
- [ ] MinIO Console accessible — `raw`, `iceberg`, `logs` buckets exist
- [ ] Nessie accessible — `GET /api/v1/trees` returns the default `main` branch
- [ ] End-to-end: a DAG writes an Iceberg table to MinIO, Trino can query it with `SELECT *`
- [ ] KEDA: submitting tasks creates pods; idle cluster scales to zero within 5 minutes
- [ ] API reachable at `api.local/v1/health` returning 200
- [ ] All Helm values files committed to `infra/helm/` — cluster can be fully rebuilt from these files
