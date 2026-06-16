# Sprint 23 — AWS Cloud Migration

**Theme:** Terraform, EKS, Karpenter, KEDA, S3, RDS, ECR, OIDC, environment promotion

---

## Goal

Migrate the platform from the on-prem k3s cluster to AWS. By the end of this sprint every infrastructure resource is declared in Terraform, the full stack runs on EKS, and the CI/CD pipeline deploys to AWS using GitHub OIDC — no static credentials. MinIO becomes S3, local Postgres becomes Aurora, the local registry becomes ECR, and k3s becomes EKS. Application code changes nothing — only config and infrastructure change.

---

## Concepts

### Migration Philosophy: Controlled Variable Swap

- The on-prem sprint was deliberate: build confidence that the architecture works before paying for cloud.
- Migration is a swap of infrastructure primitives, not a rewrite:
  - MinIO → AWS S3 (same API, same bucket names, update endpoint config)
  - Local Postgres → Amazon Aurora Postgres (same schema, run migrations on new host)
  - Local registry → Amazon ECR (same Docker images, different registry URL)
  - k3s → Amazon EKS (same Helm charts, same values files with minor overrides)
  - Traefik Ingress → AWS Load Balancer Controller + ALB
  - Local kubeconfig → EKS kubeconfig via `aws eks update-kubeconfig`
- Application code, DAGs, and Helm chart templates do not change. Only values files and Terraform configs are new.

### Terraform Core Concepts

- **Provider** — a plugin that knows how to talk to an API (AWS, Kubernetes, Helm).
- **Resource** — a single infrastructure object (`aws_eks_cluster`, `aws_s3_bucket`). The unit of management.
- **Data source** — reads existing infrastructure Terraform does not manage (current AWS account ID, existing VPC).
- **Variable** — an input to a module. Declared in `variables.tf`, set in `terraform.tfvars`.
- **Output** — a value exported from a module, consumed by another module (e.g., EKS cluster endpoint → Kubernetes provider).
- **State** — Terraform's mapping of config to real resources. Stored remotely in S3 + DynamoDB lock. Never edit manually.
- **`terraform plan`** — shows what will change without changing anything. Always review before applying.
- **`terraform apply`** — executes the plan. Use `--auto-approve` only in CI.
- **`terraform destroy`** — destroys all managed resources. In dev: use after testing to stop billing.

### Remote State Backend

- Store state in S3 with DynamoDB for locking — prevents two concurrent applies from corrupting state.
  ```hcl
  terraform {
    backend "s3" {
      bucket         = "my-platform-tfstate"
      key            = "envs/dev/terraform.tfstate"
      region         = "ap-southeast-1"
      dynamodb_table = "terraform-state-lock"
      encrypt        = true
    }
  }
  ```
- Each environment (`dev`, `staging`, `prod`) has its own state file under a different S3 key.
- Bootstrap: create the S3 bucket and DynamoDB table once manually (chicken-and-egg problem — Terraform cannot manage its own backend bucket).

### Module Structure

```
infra/terraform/
  modules/
    vpc/          → VPC, subnets, NAT gateway, security groups
    eks/          → EKS cluster, managed node groups, add-ons, IRSA
    karpenter/    → Karpenter Helm install, NodePool CRDs
    rds/          → Aurora Postgres Serverless v2
    s3/           → S3 buckets (raw, iceberg, logs) with lifecycle policies
    ecr/          → ECR repositories (api, orchestrator, spark-base)
    iam/          → IAM roles, policies, IRSA trust policies, GitHub OIDC
  envs/
    dev/
      main.tf           → calls modules with dev-specific sizing
      terraform.tfvars  → dev variable values
    staging/
    prod/
```

### VPC and Networking

- **VPC** — isolated network. All resources live inside it.
- **Public subnets** — route to Internet Gateway. ALB load balancers live here.
- **Private subnets** — no direct internet route. EKS nodes, RDS, pods live here. Egress via NAT Gateway.
- **NAT Gateway** — allows private subnet resources to pull Docker images, call APIs, without being reachable from the internet. Costs ~$32/month plus data transfer — use one NAT Gateway in dev (not one per AZ) to save cost.
- **Security groups** — stateful firewall at resource level. Separate groups for: ALB (443 from internet), EKS nodes (node-to-node + ALB), RDS (5432 from EKS nodes only).

### EKS and IRSA

- **EKS** — AWS-managed Kubernetes control plane. You manage worker nodes; AWS manages etcd, API server.
- **Managed node groups** — EC2 instances managed by AWS (patching, drain on termination). Simpler than self-managed.
- **IRSA (IAM Roles for Service Accounts)** — assigns an IAM role to a Kubernetes service account. Pods using that service account assume the role — no AWS credentials in pod environment variables.
  - Airflow workers: S3 read/write on `iceberg` and `raw` buckets.
  - Trino: S3 read on `iceberg` bucket.
  - Nessie: S3 read/write on `iceberg` bucket.
  - API: Secrets Manager read (for DB password, encryption key).
- IRSA replaces node-level IAM roles — far safer, least-privilege per workload.
- **EKS add-ons**: `vpc-cni`, `coredns`, `kube-proxy`, `aws-ebs-csi-driver` (for PVCs on EKS).

### Karpenter

- **Karpenter** — node autoprovisioner. Watches for unschedulable pods, provisions EC2 instances to fit them. Replaces Cluster Autoscaler.
- **NodePool** — CRD defining allowed instance types, OS, architecture, capacity type (Spot vs On-Demand).
- **Consolidation** — Karpenter terminates underused nodes and reschedules pods to fewer, fuller nodes. Significant cost saving.
- **Spot interruption** — Karpenter listens for EC2 Spot interruption notices via EventBridge and drains the node within the 2-minute warning window.
- Two NodePools for this platform:
  - General: `t3.medium`, `t3.large`, On-Demand — for Airflow scheduler, API, Trino coordinator.
  - Compute: `r6i.xlarge`, `r6a.xlarge`, Spot preferred — for Spark and Trino workers.
- **Why Karpenter over Cluster Autoscaler**: selects optimal instance type per workload, provisions in ~30s vs 2–3 minutes, handles Spot natively.

### KEDA on EKS

- KEDA works identically on EKS as on k3s — same Helm chart, same ScaledObject CRDs.
- The only change: Airflow's metadata DB is now Aurora instead of local Postgres. Update the KEDA PostgreSQL scaler connection string.
- KEDA + Karpenter interaction: KEDA creates pods (workload scaling) → pending pods trigger Karpenter to provision nodes (infrastructure scaling) → both scale to zero when idle.

### GitHub OIDC → AWS IAM

- Static AWS credentials (`AWS_ACCESS_KEY_ID`) in GitHub Secrets rotate manually and can leak.
- **OIDC** — GitHub Actions exchanges a short-lived JWT for temporary AWS credentials via IAM federation. Zero long-lived secrets.
- How it works:
  1. GitHub generates a signed JWT for the workflow run.
  2. Workflow calls AWS STS `AssumeRoleWithWebIdentity` with the JWT.
  3. AWS validates JWT against GitHub's OIDC endpoint, returns temporary credentials (15 min TTL).
- IAM trust policy restricts which repo and branch can assume the role:
  ```json
  "Condition": {
    "StringLike": {
      "token.actions.githubusercontent.com:sub": "repo:my-org/data-platform:ref:refs/heads/main"
    }
  }
  ```
- CI deploy jobs use `aws-actions/configure-aws-credentials` with `role-to-assume` — no key storage anywhere.

### ECR

- **ECR (Elastic Container Registry)** — AWS-managed Docker registry. Images are scanned on push for CVEs.
- EKS nodes pull from ECR automatically within the same account — no registry credentials needed on nodes.
- The self-hosted runner (or GitHub-hosted runner with OIDC) pushes to ECR: `docker push <account>.dkr.ecr.<region>.amazonaws.com/api:sha-<sha>`.
- Update Helm values files: replace `registry.local:5000/api` with the ECR URI.

### Aurora Postgres Serverless v2

- **Aurora Serverless v2** — scales compute capacity in fine-grained increments (ACUs) based on load. No idle compute cost when traffic is zero (with auto-pause enabled in dev).
- Same Postgres wire protocol — no application code changes, only the connection string changes.
- Runs inside the VPC, private subnet. Not publicly reachable. Connect via a bastion pod or `kubectl port-forward` for admin tasks.
- **Migration**: `pg_dump` from local Postgres → `pg_restore` to Aurora. Run migrations (`golang-migrate`) after restore to catch up any schema changes.

### Cost Management

- **Destroy when done**: `terraform destroy` the dev environment after testing sessions. EKS control plane alone costs $2.40/day.
- **Spot instances**: Karpenter's compute NodePool uses Spot — 60–80% cheaper than On-Demand for Spark/Trino workers.
- **Single NAT Gateway in dev**: one per AZ is HA but expensive. One is enough for dev.
- **Aurora auto-pause**: set `min_capacity = 0` in dev — Aurora pauses after 5 minutes of inactivity, no ACU cost while paused.
- **S3 lifecycle policies**: move old Iceberg snapshots to S3 Glacier after 30 days, expire expired snapshots after 90 days.

---

## Tasks

### Bootstrap
- [ ] Create S3 bucket and DynamoDB table for Terraform state (one-time manual step)
- [ ] Add `aws_iam_openid_connect_provider` for GitHub OIDC to `infra/terraform/modules/iam/`
- [ ] Add IAM role for CI: trust policy scoped to `main` branch of this repo; permissions: ECR push, S3 state, DynamoDB lock, EKS describe, Helm apply via EKS API

### Terraform Modules
- [ ] Write `infra/terraform/modules/vpc/` — VPC, public/private subnets (2 AZs), single NAT Gateway, security groups
- [ ] Write `infra/terraform/modules/s3/` — `raw`, `iceberg`, `logs` buckets with versioning and lifecycle policies
- [ ] Write `infra/terraform/modules/ecr/` — repositories: `api`, `orchestrator`
- [ ] Write `infra/terraform/modules/iam/` — IRSA roles for Airflow, Trino, Nessie, API; GitHub OIDC role
- [ ] Write `infra/terraform/modules/rds/` — Aurora Postgres Serverless v2, auto-pause in dev, Secrets Manager for password
- [ ] Write `infra/terraform/modules/eks/` — EKS cluster, managed node group (On-Demand, general), EKS add-ons, IRSA setup
- [ ] Write `infra/terraform/modules/karpenter/` — Karpenter Helm install, NodePool (general On-Demand + compute Spot)

### Environments
- [ ] Write `infra/terraform/envs/dev/main.tf` — dev sizing: small node group, single NAT, Aurora min 0.5 ACU auto-pause
- [ ] `terraform plan` passes for dev with zero errors
- [ ] `terraform apply` creates working EKS cluster — `kubectl get nodes` shows ready nodes

### Data Plane Migration
- [ ] Create S3 buckets via Terraform, verify accessible from the cluster (pod with AWS CLI)
- [ ] `pg_dump` local Postgres → restore to Aurora → run `golang-migrate` to verify schema is current
- [ ] Update Nessie Helm values: swap Postgres endpoint to Aurora, swap MinIO endpoint to S3
- [ ] Update Trino Helm values: swap MinIO endpoint to S3, use IRSA service account
- [ ] Update Airflow Helm values: swap MinIO endpoint to S3, Aurora as metadata DB, IRSA service account
- [ ] Update API Helm values: Aurora connection string from Secrets Manager, IRSA service account
- [ ] Helm upgrade all services to EKS cluster — all pods Running

### CI/CD Update
- [ ] Update `deploy.yml`: use GitHub OIDC (`aws-actions/configure-aws-credentials` with `role-to-assume`)
- [ ] Update `deploy.yml`: push images to ECR instead of local registry
- [ ] Update `deploy.yml`: run `helm upgrade` against EKS cluster (update kubeconfig step)
- [ ] Remove self-hosted runner dependency for deploy jobs — GitHub-hosted runner can now reach EKS via OIDC
- [ ] Add `tf-plan.yml` — runs `terraform plan` on PR, posts output as PR comment
- [ ] Add `tf-apply.yml` — runs `terraform apply` on merge to main (dev only, auto-approve)

### Cost Controls
- [ ] Add `terraform destroy` Makefile target with confirmation prompt
- [ ] Aurora auto-pause configured in dev Terraform — verify it pauses after idle period
- [ ] S3 lifecycle policy: transition `iceberg/` prefix to Intelligent-Tiering after 30 days
- [ ] Set up AWS Budget alert: email if monthly spend exceeds $50

### Validation
- [ ] End-to-end on EKS: trigger an Airflow DAG that writes an Iceberg table to S3, query via Trino
- [ ] KEDA + Karpenter: submit 10 Airflow tasks, observe pods scale up (KEDA), then new nodes provision (Karpenter); idle → pods scale to zero → nodes terminate
- [ ] IRSA validation: Airflow worker pod can write to S3, has no `AWS_ACCESS_KEY_ID` in its environment
- [ ] `terraform destroy` on dev — all resources removed, zero billing after destroy

---

## Interview Topics

- **Walk through the migration from MinIO to S3.** What changed in the application? What changed in infrastructure?
- **What is IRSA?** Why is it better than putting AWS credentials in a Kubernetes Secret?
- **Explain GitHub OIDC authentication to AWS.** Why does this eliminate the need for stored access keys?
- **What is the EKS control plane cost?** How do you minimize total cluster cost in a dev environment?
- **Explain Karpenter consolidation.** When would you disable it, and why?
- **What is Aurora Serverless v2 auto-pause?** What is the tradeoff versus always-on?
- **How does KEDA interact with Karpenter during a traffic spike?** What happens at each layer?
- **What is `terraform plan` and why should it always run before `terraform apply` in production?**
- **If `terraform apply` fails halfway through, what is the state?** How do you recover?
- **What S3 lifecycle policies would you put on an Iceberg data lake?** Why?

---

## Definition of Done

- [ ] `terraform apply` on dev environment succeeds — EKS cluster, Aurora, S3, ECR all provisioned
- [ ] `kubectl get nodes` shows nodes in `Ready` state on EKS
- [ ] All services deployed via Helm to EKS — `kubectl get pods -A` shows all Running
- [ ] End-to-end pipeline: Airflow DAG writes Iceberg table to S3, Trino queries it successfully
- [ ] IRSA confirmed: Airflow worker has no AWS credentials in environment, can still write to S3
- [ ] GitHub OIDC: deploy workflow uses `role-to-assume`, no `AWS_ACCESS_KEY_ID` in GitHub Secrets
- [ ] Karpenter provisions a new node within 60 seconds of a pending pod with unmet resource requests
- [ ] KEDA scales Airflow workers to zero within 5 minutes of task queue being empty
- [ ] AWS Budget alert is configured — you receive a test email
- [ ] `terraform destroy` removes all resources cleanly — AWS console shows nothing remaining in the VPC
