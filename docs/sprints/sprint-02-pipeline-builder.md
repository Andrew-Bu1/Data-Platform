# Sprint 2 — Visual Pipeline Builder

**Theme:** React Flow canvas, graph persistence, DAG validation

---

## Goal

Build the drag-and-drop pipeline editor. By the end of this sprint a user can open a pipeline, add source/transform/sink nodes to a canvas, draw edges between them, save the graph to the database, and see validation errors when the graph is invalid (cycle, disconnected node). The pipeline does not run yet — this sprint is about the visual design tool.

---

## Concepts

### React Flow

- **React Flow** is a library for building node-based editors. The core concepts: `ReactFlow` component, `nodes` array, `edges` array, `onNodesChange` / `onEdgesChange` callbacks.
- **Node types** — you define custom node components and register them with the `nodeTypes` prop. Each node type renders its own UI (icon, label, port handles).
- **Handles** — the connection points on a node. `<Handle type="source" position={Position.Right} />` for output, `<Handle type="target" position={Position.Left} />` for input. Multi-input nodes (like join) need multiple target handles with distinct IDs.
- **Edge types** — custom edge components for styled connectors. Default edges are fine for MVP.
- **`useReactFlow` hook** — imperative access to add nodes, zoom to fit, get the current viewport.
- **`useNodesState` / `useEdgesState`** — managed state hooks that return `[nodes, setNodes, onNodesChange]` and the equivalent for edges.
- **Background, Controls, MiniMap** — built-in UI helpers for a polished canvas.
- **Persisting positions** — node positions are part of the graph JSON. Always save `position: { x, y }` with each node.

### Graph Validation

- **Cycle detection** — given nodes and edges as adjacency lists, run DFS. Mark each node as white (unvisited), gray (in current path), black (fully processed). If you reach a gray node, a cycle exists.
  ```
  function hasCycle(nodes, edges):
    state = { nodeId: "white" for all nodes }
    for each node:
      if state[node] == "white":
        if dfs(node, state, edges): return true
    return false
  
  function dfs(node, state, edges):
    state[node] = "gray"
    for each neighbor of node:
      if state[neighbor] == "gray": return true  // back edge = cycle
      if state[neighbor] == "white":
        if dfs(neighbor, state, edges): return true
    state[node] = "black"
    return false
  ```
- **Disconnected nodes** — a node with no edges is a warning: the user probably forgot to connect it.
- **Missing required ports** — a join node needs exactly two inputs. A sink node needs exactly one input. Validate port cardinality.
- **Source-only and sink-only pipelines** — a valid pipeline must have at least one source and at least one sink.
- Run validation on every graph change (debounced) and display inline error badges on offending nodes.

### JSON Schema-Driven UI

- Each node type has a **config schema** — a JSON Schema document describing what fields the node's side panel should show.
  ```json
  {
    "type": "object",
    "properties": {
      "table": { "type": "string", "title": "Table name" },
      "connection_id": { "type": "string", "title": "Connection", "x-widget": "connection-selector" }
    },
    "required": ["table", "connection_id"]
  }
  ```
- The side panel reads the schema and renders fields. This avoids writing a new panel component for every node type.
- `x-widget` custom extension: for fields that need a special input component (dropdown of connections, column picker), mark them with a custom annotation.
- This pattern is important — in Sprint 15 you will make connectors fully pluggable by driving everything from these schemas.

### Pipeline Versioning — Draft Model

- A pipeline has many `PipelineVersions`. The canvas always edits a **draft** version.
- On "Save", the draft is updated (the same row in the DB, `status = 'draft'`).
- On "Publish" (later, in Sprint 4), the draft is frozen (status = `'published'`) and a new draft is created from it.
- The UI shows whether the current draft has unsaved changes (dirty state in Zustand).
- Why immutable versions: every run records which version it executed. If you need to debug a run from three months ago, you can view the exact graph that was active.

### Frontend State Management with Zustand

- **Zustand** is a minimal state management library. A store is just a function that returns state + actions.
  ```ts
  const usePipelineStore = create<PipelineStore>((set) => ({
    nodes: [],
    edges: [],
    isDirty: false,
    setNodes: (nodes) => set({ nodes, isDirty: true }),
    setEdges: (edges) => set({ edges, isDirty: true }),
    loadPipeline: (pipeline) => set({ nodes: pipeline.nodes, edges: pipeline.edges, isDirty: false }),
  }))
  ```
- Zustand integrates well with React Flow: the store holds nodes and edges, React Flow reads them and calls `onNodesChange` which updates the store.
- Keep API calls out of Zustand. Use React Query or `useEffect` for data fetching; Zustand for UI state only.

### Go: JSONB Storage for Graph Data

- Store the pipeline graph as `JSONB` in PostgreSQL. Do not normalize nodes and edges into separate tables at this stage — JSONB is simpler and the entire graph is always read/written as one unit.
- Validate the JSONB against the pipeline graph JSON Schema before storing.
- Use `pgtype.JSONB` (pgx v5) or `json.RawMessage` for the Go struct field.
- Later (if you need to query individual nodes), add JSONB extraction operators (`->`, `->>`).

---

## Tasks

### Backend
- [ ] Create `pipelines` table migration (`id`, `project_id`, `environment_id`, `name`, `description`, `created_at`, `updated_at`, `deleted_at`)
- [ ] Create `pipeline_versions` table migration (`id`, `pipeline_id`, `version_number`, `status` (draft/published), `graph` JSONB, `created_at`)
- [ ] `POST /v1/projects/:id/pipelines` — create pipeline, auto-create first draft version
- [ ] `GET /v1/projects/:id/pipelines` — list pipelines
- [ ] `GET /v1/pipelines/:id` — get pipeline with latest draft graph
- [ ] `PUT /v1/pipelines/:id/draft` — save draft graph (validate against JSON Schema first)
- [ ] `DELETE /v1/pipelines/:id` — soft delete
- [ ] Graph validation service: cycle detection, disconnected node detection, port cardinality rules
- [ ] Return validation errors from the save endpoint (not a 400 — return 200 with a `validation_errors` field so the UI can show inline errors without blocking the save)

### Frontend
- [ ] Install React Flow (`@xyflow/react`)
- [ ] Create pipeline builder page (`/pipelines/[id]/builder`)
- [ ] Implement React Flow canvas with background grid, controls, minimap
- [ ] Create `SourceNode` component — left-side icon, label, one output handle
- [ ] Create `TransformNode` component — centered, one input handle, one output handle
- [ ] Create `SinkNode` component — right-side icon, label, one input handle
- [ ] Node toolbar (left sidebar or top bar) — drag node types onto canvas
- [ ] Implement drag-and-drop from toolbar to canvas (React Flow's `onDrop` + `onDragOver`)
- [ ] Implement add/remove edges between nodes
- [ ] Node click → open side panel with node config form
- [ ] Side panel: render form fields from node config schema (basic version: hardcoded fields per type)
- [ ] Save button → `PUT /v1/pipelines/:id/draft` → show success/error toast
- [ ] Load pipeline on page mount → `GET /v1/pipelines/:id` → render graph on canvas
- [ ] Show validation errors as red badge icons on nodes
- [ ] Zustand store for pipeline canvas state (nodes, edges, isDirty)
- [ ] Pipeline list page with "New pipeline" button and pipeline cards

---

## Interview Topics

- **Explain Kahn's algorithm for topological sort.** How does it detect cycles?
- **What is the difference between cycle detection with DFS white/gray/black coloring vs Kahn's algorithm (in-degree counting)?** When would you prefer each?
- **Why store a pipeline graph as JSONB instead of normalized rows?** What are the tradeoffs?
- **What is a dirty state in a UI and how do you prevent accidental data loss?** Describe the `isDirty` flag pattern and how to show an "unsaved changes" warning on navigation.
- **Explain React Flow's node + edge data model.** How would you add a node type that has two input handles (for a join node)?
- **Why is pipeline versioning important?** Explain the draft vs published state machine.
- **What is JSON Schema and what problem does it solve?** Compare it to TypeScript types — both describe structure, but JSON Schema is runtime-validatable and language-agnostic.

---

## Definition of Done

- [ ] User can create a new pipeline from the UI
- [ ] User can drag source, transform, and sink node types onto the canvas
- [ ] User can draw edges between nodes and remove them
- [ ] User can click a node to open a config side panel
- [ ] Graph is saved to the database and reloads correctly on page refresh
- [ ] Cycle in the graph shows a validation error badge on the offending node(s)
- [ ] Disconnected nodes show a warning badge
- [ ] Canvas is pannable, zoomable, and shows a minimap
- [ ] `isDirty` flag shows "Unsaved changes" in the UI
- [ ] Go validation service tests: cycle detection, valid DAG, disconnected node all have unit tests
