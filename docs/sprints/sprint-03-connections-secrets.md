# Sprint 3 — Connections and Secret Management

**Theme:** Connector abstraction, credential encryption, environment-scoped config

---

## Goal

Make external system connections reusable and secure. By the end of this sprint a user can create a database connection, test that the connection works, and reference it from a pipeline node. Credentials are stored encrypted — no plaintext passwords anywhere in the database, logs, or API responses.

---

## Concepts

### Secret Management Patterns

- **Never store plaintext credentials.** If your database is compromised, connection passwords must not be readable.
- **Encrypt at rest** — credentials are AES-256-GCM encrypted before INSERT and decrypted only in-memory at runtime. The encryption key lives in an environment variable (or a secrets manager in production), never in the database.
- **Separation of concerns** — a `Connection` row has two parts:
  1. **Config** (JSONB, plaintext) — non-sensitive metadata: hostname, port, database name, schema, SSL mode.
  2. **Encrypted credentials** (BYTEA) — sensitive values: password, API key, private key. AES-256-GCM encrypted blob.
- **Credential masking** — the GET endpoint for a connection returns config but never the decrypted credentials. It may return a `has_credentials: true` flag so the UI can show that credentials are saved.
- **Secret references** — rather than embedding credentials in the pipeline graph JSON, the graph stores a `connection_id` pointer. At execution time, the execution plane asks the control plane to resolve the connection for the current environment.

### AES-256-GCM Encryption in Go

- **AES (Advanced Encryption Standard)** — symmetric block cipher. The key size determines security: AES-128, AES-192, AES-256. Use AES-256.
- **GCM (Galois/Counter Mode)** — authenticated encryption mode. Provides both confidentiality (encrypted) and integrity (tamper detection via authentication tag).
- **Nonce** — a 12-byte random value that must never be reused with the same key. Generate a new nonce for every encryption. Prepend the nonce to the ciphertext so you can extract it at decryption time.
- Go implementation:
  ```go
  func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
      block, _ := aes.NewCipher(key)  // key must be 32 bytes for AES-256
      gcm, _ := cipher.NewGCM(block)
      nonce := make([]byte, gcm.NonceSize())
      rand.Read(nonce)
      ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)  // prepend nonce
      return ciphertext, nil
  }

  func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
      block, _ := aes.NewCipher(key)
      gcm, _ := cipher.NewGCM(block)
      nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
      return gcm.Open(nil, nonce, ciphertext, nil)
  }
  ```
- **Key rotation** — if the encryption key needs to change, you must re-encrypt all existing credentials with the new key. Plan for this in the schema: add a `key_version` column to connections.

### Connector Abstraction Design

- A **connector type** is not a runtime interface in the early sprints — it is a metadata concept. The connector type name (e.g., `"postgres"`) is stored in the connection row and in node configs.
- Each connector type has a known set of **config fields** (host, port, database) and **credential fields** (password, private_key).
- Each connector type has **capabilities**: `can_source` (can extract data from), `can_sink` (can load data into), `sample_preview_supported`.
- Hardcode these in a Go registry map for now:
  ```go
  var ConnectorRegistry = map[string]ConnectorDefinition{
      "postgres": {
          ConfigFields:   []FieldDef{...},
          CredentialFields: []FieldDef{...},
          CanSource: true,
          CanSink:   true,
      },
  }
  ```
- In Sprint 15 this hardcoded map becomes a manifest-driven plugin system. But the interface stays the same.

### Environment-Specific Connections

- A connection belongs to a specific **environment** within a project.
- The same logical data source (e.g., "orders database") will have different connection details in dev vs production.
- When a pipeline is deployed to an environment, its source/sink nodes resolve their connections from that environment.
- The pipeline graph stores a **connection alias** (e.g., `"orders_db"`) in the node config. The deployment maps aliases to environment-specific connection IDs.
- Alternative (simpler): the pipeline node stores a `connection_id` directly. Then when you deploy to a different environment, you swap the `connection_id` at the deployment level.

### Connection Testing

- The test endpoint must actually open a TCP connection and run a lightweight check (e.g., `SELECT 1` for PostgreSQL).
- Do not trust the test endpoint as a security gate — it only confirms connectivity. Authorization checks happen at execution time.
- Connection test results should be stored (`last_tested_at`, `last_test_status`, `last_test_error`) so the UI can show a status badge without re-testing on every load.
- Timeout the test aggressively (3–5 seconds). A slow connection is almost as bad as a broken one.

### Python Type Annotations (for orchestrator layer)

- From this sprint onward, all Python code in the `apps/orchestrator/` layer must use type hints.
- `from typing import Protocol` — use `Protocol` instead of abstract base classes for connector interfaces. Structural typing (duck typing) is idiomatic Python.
  ```python
  class Connector(Protocol):
      def test_connection(self) -> bool: ...
      def get_schema(self, table: str) -> list[ColumnDef]: ...
  ```
- `@dataclass` for config objects. Use `frozen=True` for immutable configs.
- Return `Result` types instead of raising exceptions where possible — callers should handle errors explicitly.

---

## Tasks

### Backend
- [ ] Create `connections` table migration (`id`, `project_id`, `environment_id`, `connector_type`, `display_name`, `config` JSONB, `encrypted_creds` BYTEA, `key_version`, `last_tested_at`, `last_test_status`, `last_test_error`, `created_at`, `updated_at`)
- [ ] Implement AES-256-GCM encryption/decryption utility (`internal/crypto/`)
- [ ] `POST /v1/projects/:id/connections` — create connection (encrypt credentials before storing)
- [ ] `GET /v1/projects/:id/connections` — list connections (never return decrypted creds)
- [ ] `GET /v1/connections/:id` — get connection details (config only, no creds)
- [ ] `PUT /v1/connections/:id` — update connection
- [ ] `DELETE /v1/connections/:id` — delete connection
- [ ] `POST /v1/connections/:id/test` — test connection (decrypt creds, open real connection, return result)
- [ ] Store test result on the connection row after each test
- [ ] Hardcoded connector registry with capabilities for: `postgres`, `mysql`, `rest_api`, `sftp`
- [ ] `GET /v1/connector-types` — list available connector types with their config schemas

### Frontend
- [ ] Connections list page — show all connections for current project/environment
- [ ] Connection status badge (tested/untested/failed)
- [ ] Connection creation form — connector type selector, then dynamic fields per type
- [ ] Password/key fields: masked input, show/hide toggle, never pre-filled after save
- [ ] Test connection button — show spinner, then success/failure result with error message
- [ ] Environment selector on connection form
- [ ] In pipeline builder: source/sink node config panel — connection dropdown (filtered by capability)

### Orchestrator
- [ ] Create `connectors/` package with `base.py` defining the `Connector` Protocol
- [ ] Stub connection test function for PostgreSQL (`psycopg2.connect` + `SELECT 1`)
- [ ] All Python files use type hints (enforce with `mypy`)

---

## Interview Topics

- **Why use AES-256-GCM instead of AES-256-CBC?** GCM provides authenticated encryption — it detects tampering, CBC does not.
- **What is a nonce and why must it never be reused?** Nonce reuse with GCM completely breaks confidentiality.
- **Explain the difference between encrypting at rest and encrypting in transit.** When do you need both?
- **How do you handle key rotation for encrypted credentials?** Describe the key_version approach and the re-encryption migration.
- **Why should the GET connection endpoint never return the decrypted password?** Discuss over-fetching data and the principle of least privilege in API design.
- **What is a Protocol in Python?** How does it differ from an abstract base class? Explain structural subtyping (duck typing) vs nominal subtyping.
- **How would you design environment-specific connection management?** Describe connection aliases vs direct connection ID references on pipeline nodes.

---

## Definition of Done

- [ ] User can create connections for postgres, mysql, rest_api, sftp connector types
- [ ] Credentials are AES-256-GCM encrypted in the database — confirmed by inspecting the `encrypted_creds` column directly and seeing binary data, not plaintext
- [ ] GET connection endpoint never returns decrypted credentials
- [ ] Test connection actually opens a real connection (test against a local PostgreSQL)
- [ ] Test result is persisted and shown as a status badge in the UI
- [ ] Pipeline builder source/sink node panels show a connection dropdown
- [ ] Connections are scoped to environment — a connection in "dev" does not appear in "prod" dropdown
- [ ] `mypy` passes on all orchestrator Python files
