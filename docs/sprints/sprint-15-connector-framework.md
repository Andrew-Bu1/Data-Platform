# Sprint 15 — Connector Framework v2

**Theme:** Plugin architecture, manifest-driven config, dynamic form generation

---

## Goal

Replace hardcoded connector definitions with a pluggable manifest system. By the end of this sprint, adding a new connector type requires only writing a manifest file and a Python connector class — no changes to the control plane or frontend. The UI generates connection forms dynamically from JSON Schema.

---

## Concepts

### Plugin Architecture Patterns

- **Hardcoded registry** (current state) — connector capabilities are defined in Go source code. Adding a connector requires changing the binary.
- **Manifest-driven registry** — each connector is described by a YAML/JSON file. The registry loads manifests at startup. New connectors are added by deploying a new manifest file.
- **Dynamic loading** — connectors are loaded at runtime (Python `importlib`, Go plugins). The most flexible but also the most complex and risky.
- For this project: use the **manifest-driven** approach. Manifests describe what a connector needs; the connector implementation is a Python class in the orchestrator. The control plane loads manifests; the execution plane loads the Python class.

### Connector Manifest Format

```yaml
# connector-specs/postgres/manifest.yaml
id: postgres
display_name: PostgreSQL
version: "1.0.0"
capabilities:
  source: true
  sink: true
  sample_preview: true
  schema_discovery: true
config_schema:
  $schema: http://json-schema.org/draft-07/schema#
  type: object
  properties:
    host:
      type: string
      title: Host
      description: PostgreSQL server hostname or IP
    port:
      type: integer
      title: Port
      default: 5432
    database:
      type: string
      title: Database name
    schema:
      type: string
      title: Schema
      default: public
    ssl_mode:
      type: string
      title: SSL mode
      enum: [disable, require, verify-ca, verify-full]
      default: require
  required: [host, port, database]
credential_schema:
  type: object
  properties:
    username:
      type: string
      title: Username
      x-widget: text
    password:
      type: string
      title: Password
      x-widget: password
  required: [username, password]
```

### JSON Schema for Dynamic Form Generation

- The frontend reads the `config_schema` and `credential_schema` from the manifest and renders form fields automatically.
- Standard JSON Schema field types map to standard HTML inputs:
  - `string` → text input
  - `integer`, `number` → number input
  - `boolean` → checkbox
  - `string` with `enum` → dropdown select
  - `string` with `x-widget: password` → password input (masked)
  - `string` with `x-widget: textarea` → multi-line text area
- Custom `x-widget` extension: annotate fields that need special UI components.
- Form validation runs client-side using the JSON Schema (`ajv` library for TypeScript).
- Required fields show validation errors. Optional fields with defaults show the default as placeholder.

### Capability Registry

- The control plane loads all manifests at startup and builds a capability registry:
  ```go
  type ConnectorManifest struct {
      ID           string
      DisplayName  string
      Version      string
      Capabilities ConnectorCapabilities
      ConfigSchema json.RawMessage
      CredentialSchema json.RawMessage
  }
  
  var registry map[string]ConnectorManifest
  ```
- `GET /v1/connector-types` returns the registry so the frontend can list available connectors.
- `GET /v1/connector-types/:id/config-schema` returns the JSON Schema for the connection form.
- The execution plane loads connector Python classes by `connector_type` string at runtime using a Python registry dict: `{"postgres": PostgreSQLConnector, "mysql": MySQLConnector}`.

### Versioned Connector Definitions

- Connectors evolve. The manifest has a `version` field. When a connector's config schema changes in a breaking way (a required field added, a field renamed), bump the version.
- Existing connections store which `connector_version` they were created with.
- The execution plane must handle loading a connection config that was created with an older connector version.
- Migration strategy: each connector can have a `migrate_config(from_version, config)` function that upgrades old configs to the new format.

### Python Connector Packaging

Each connector is a self-contained Python module:
```
apps/orchestrator/connectors/
├── base.py              # Protocol definitions
├── registry.py          # {connector_type: class} mapping
├── postgres/
│   ├── __init__.py
│   ├── connector.py     # PostgreSQLConnector implementing Source/Sink protocols
│   └── tests/
│       └── test_postgres.py
├── mysql/
├── rest_api/
└── sftp/
```

The `registry.py` file is the single point of truth for which Python class handles each connector type. The orchestrator imports from here.

---

## Tasks

### Connector Specs
- [ ] Define `connector-specs/` folder with `manifest.yaml` for each connector: postgres, mysql, rest_api, sftp
- [ ] Validate all manifests against a meta-schema (JSON Schema describing a valid manifest)
- [ ] Add `version` field to each manifest

### Backend
- [ ] Manifest loader: read all `connector-specs/` manifests at startup, build in-memory registry
- [ ] `GET /v1/connector-types` — return all manifests (id, display_name, capabilities)
- [ ] `GET /v1/connector-types/:id/config-schema` — return JSON Schema for the config form
- [ ] `GET /v1/connector-types/:id/credential-schema` — return JSON Schema for the credential form
- [ ] Store `connector_version` on connection rows
- [ ] Manifest hot reload: watch the `connector-specs/` folder for changes and reload without restart (useful for development)

### Frontend
- [ ] Dynamic connection form renderer: given a JSON Schema, generate and validate a form
- [ ] Remove all hardcoded connector-specific form components — replace with the dynamic renderer
- [ ] Custom widget support: `x-widget: password` renders masked input, `x-widget: connection-selector` renders connection dropdown
- [ ] Form validation using `ajv`: show per-field errors, block submit until valid
- [ ] Connector type selector: show `display_name` and capability badges (source / sink icons)

### Orchestrator
- [ ] Restructure connectors into `connectors/{type}/connector.py` modules
- [ ] Implement `ConnectorRegistry` class: loads all connector classes, supports `get(connector_type) → class`
- [ ] Add `migrate_config(from_version: str, config: dict) → dict` method to each connector class (even if it just returns config unchanged for now)
- [ ] Write tests for each connector's `migrate_config` method

---

## Interview Topics

- **Explain plugin/manifest architecture.** What are the tradeoffs compared to hardcoded registries?
- **What is JSON Schema?** How would you use it to generate a form UI?
- **What is `ajv`?** How does client-side JSON Schema validation work?
- **How do you handle breaking changes in a connector's config schema?** Describe the version + migration approach.
- **What is the `x-extension` pattern in JSON Schema?** Why does JSON Schema allow custom properties?
- **What is the Open-Closed Principle?** How does manifest-driven connector architecture implement it?
- **How would you test a dynamic form renderer?** Describe the test cases needed.
- **What are the security risks of loading manifest files at runtime?** How would you validate them?

---

## Definition of Done

- [ ] All four connectors have valid manifest files
- [ ] Control plane loads manifests at startup and serves them via API
- [ ] Frontend renders connection forms entirely from JSON Schema — no hardcoded form components remain
- [ ] Form validation catches missing required fields and invalid enum values before submit
- [ ] Adding a hypothetical new connector (`mongodb`) requires only: manifest file + Python connector class — no Go or TypeScript changes
- [ ] Connector version is stored on connections and included in config passed to the execution plane
- [ ] `migrate_config` method exists on all connectors (even as a no-op)
- [ ] Manifest meta-schema validation: an invalid manifest fails to load with a clear error message
