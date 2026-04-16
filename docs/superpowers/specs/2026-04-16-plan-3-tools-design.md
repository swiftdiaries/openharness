# Layer 2 Plan 3: Tools — Design Spec

**Parent spec:** `docs/superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md`
**Execution context:** `docs/superpowers/plans/layer-2/2026-04-13-execution-order.md` (Wave 2 — depends on Plan 1 merged)

**Goal:** Move the generic tool system from `ghostfin/desktop/internal/tools/` into `openharness/tools/` (core) and `openharness/tools/builtin/` (9 built-in tool implementations). Add the `Effects` field to `ToolDefinition` for loop-detection and mode-filtering classification. Invert `knowledge_graph` and `subagent` dependencies so they're vertical-agnostic.

**Branch:** `feat/layer-2-plan-3-tools`

---

## Design Decisions

### D1: Knowledge graph inversion — local rich interface (Option A)

The `agent.KnowledgeStore` interface (Plan 1) has only `Search(ctx, query, limit)` and `Get(ctx, id)` — intentionally narrow for `retrieval.go` (Plan 5).

The `knowledge_graph` tool uses 8 methods on `notes.Store`: `Search`, `SearchInPage`, `GetPage`, `GetPageByID`, `GetBlockTree`, `GetBlockSnippets`, `GetRelatedPages`, `GetPagesWithSharedTags`.

**Decision:** Define a richer `builtin.KnowledgeGraphStore` interface locally in `openharness/tools/builtin/knowledge_graph.go`. The `agent.KnowledgeStore` stays narrow. Ghostfin's `notes.Store` satisfies both — the narrow one structurally, the rich one via adapter methods added in Plan 7.

**Rationale:** Different consumers want different contracts. Polluting `agent/interfaces.go` with tool-specific surface would force every future vertical to implement 8 methods when `retrieval.go` only needs 2.

**Types:** A `builtin/knowledge_graph_types.go` file defines minimal `KGPage`, `KGBlock`, `KGRef` structs — field-compatible with `notes.Page/Block/Ref`. Ghostfin's Plan 7 adds adapter methods on `notes.Store` (e.g., `func (s *Store) KGSearch(q, n) ([]builtin.KGBlock, error)`) to satisfy the interface.

### D2: Memory tool — keep JSON-file persistence, reconcile types (Option A)

The existing `tools/memory.go` defines its own `MemoryEntry` struct. Plan 1 already landed `agent.MemoryEntry`.

**Decision:**
- Delete the tool's local `MemoryEntry`; use `agent.MemoryEntry` throughout.
- The built-in `Memory` tool implements `agent.MemoryStore` structurally. Compile-time assertion: `var _ agent.MemoryStore = (*Memory)(nil)`.
- JSON-file persistence stays as-is. Storage backend abstraction is a future concern (Layer 3+).
- `DefaultMemoryPath()` is removed. Constructor is `NewMemory(storePath string)` — caller always passes the path.

**Rationale:** YAGNI on storage abstraction. No vertical has asked for non-file memory. The tool is ~200 lines; a future refactor is trivial.

### D3: Subagent tool — map-based SubagentConfig (Option A)

Current `NewSubagentTool(registry, tracker any, traceStore any)` is a stub returning `{"status":"delegated"}`. The JSON schema hardcodes ghostfin-specific subagent types.

**Decision:**
```go
type SubagentSpec struct {
    Name         string
    Description  string
    SystemPrompt string
    AllowedTools []string // reserved; Plan 3 stores, Layer 4 wires
}

type SubagentConfig struct {
    Subagents map[string]SubagentSpec // key = type name
}

func NewSubagentTool(cfg SubagentConfig) *SubagentTool
```

The tool's JSON schema builds the `subagent_type` enum dynamically from `maps.Keys(cfg.Subagents)`. Per-type descriptions feed into the tool description.

**Rationale:** Ghostfin's 5 finance presets become data passed at registration time. Generic verticals pass one "general" entry or an empty map (which omits the tool). `AllowedTools` is carried but not enforced until Layer 4 wires actual subagent execution.

### D4: Effects field classification

`Effects` lives on `ToolDefinition` (per-call, not per-tool), since tools like `filesystem` and `memory` have mixed read/mutate definitions.

**Mapping (preserves ghostfin's current loop-detection behavior):**

| Tool | Definition(s) | Effects |
|---|---|---|
| `web_search` | `web_search` | Read |
| `web_fetch` | `web_fetch` | Read |
| `filesystem` | `read_file`, `list_directory`, `search_files` | Read |
| `filesystem` | `write_file`, `edit_file`, `delete_file`, `make_directory` | Mutate |
| `exec` | `exec` / `shell` | Mutate |
| `memory` | `memory_search`, `memory_get` | Read |
| `memory` | `memory_store` | Mutate |
| `ask_user` | `ask_user` | Interactive |
| `tasks` | `task_get`, `task_list` | Read |
| `tasks` | `task_create`, `task_update` | Mutate |
| `knowledge_graph` | `knowledge_graph_search` | Read |
| `subagent` | `agent` | Mutate |

`ToolEffectRead` is the zero value — unclassified tools default to Read (conservative: doesn't reset read streaks).

`Effects` is `json:"-"` — never serialized to the LLM.

A `Registry.EffectsFor(name string) ToolEffect` helper is added for `agent/tool_loop.go` and `agent/mode.go` to query without walking definitions.

### D5: BuiltinTools constructor — Register(r, cfg) pattern

```go
package builtin

type Config struct {
    WorkspacePath   string
    MemoryPath      string
    KnowledgeStore  KnowledgeGraphStore // optional; nil skips knowledge_graph
    Subagent        SubagentConfig      // optional; zero-value omits subagent tool
}

func Register(r *tools.Registry, cfg Config) error
```

**Rationale:**
- `Register(r, cfg)` not `NewRegistry(cfg)` — callers bring their own registry to mix built-ins with vertical-specific tools.
- No `notes.WorkspaceSync` return — that's a ghostfin/notes concern.
- `KnowledgeStore` optional: nil means `knowledge_graph` isn't registered (matches today's `if notesStore != nil` branch).
- Error return for future validation without breaking changes.

---

## Package Layout

```
openharness/tools/
├── tool.go              # Tool interface + ToolDefinition (with Effects) + ToolEffect enum
├── registry.go          # Registry (sqliteonly tag stripped) + EffectsFor helper
├── composite.go         # Multi-tool wrapper
├── external_content.go  # External content guard helpers
├── ssrf.go              # SSRF URL validator
├── shell_deny.go        # Shell command denylist
├── scrub.go             # Secret scrubbing for tool output
└── builtin/
    ├── builtin.go                  # Config + Register(r, cfg)
    ├── web_search.go
    ├── web_fetch.go
    ├── filesystem.go
    ├── exec.go
    ├── memory.go                   # Implements agent.MemoryStore
    ├── ask_user.go
    ├── tasks.go
    ├── knowledge_graph.go          # KnowledgeGraphStore interface
    ├── knowledge_graph_types.go    # KGPage, KGBlock, KGRef structs
    ├── subagent.go                 # SubagentConfig + SubagentSpec
    └── *_test.go
```

**Dependency direction:**
- `tools` → stdlib only
- `tools/builtin` → `tools` + `agent` (for `MemoryEntry`, `Block`, `MemoryStore`)
- Neither depends on `providers`, `sessions`, `cost`, etc.

---

## File Inventory

### Files to copy from ghostfin (strip `//go:build sqliteonly` + `// +build sqliteonly`, otherwise verbatim unless noted)

| Source | Destination | Notes |
|---|---|---|
| `internal/tools/tool.go` | `openharness/tools/tool.go` | Add `ToolEffect` enum + `Effects` field to `ToolDefinition`. Add `json:"-"` tag. |
| `internal/tools/registry.go` | `openharness/tools/registry.go` | Strip `sqliteonly`. Add `EffectsFor(name) ToolEffect`. |
| `internal/tools/composite.go` | `openharness/tools/composite.go` | Verbatim. |
| `internal/tools/external_content.go` | `openharness/tools/external_content.go` | Verbatim. |
| `internal/tools/ssrf.go` | `openharness/tools/ssrf.go` | Verbatim. |
| `internal/tools/shell_deny.go` | `openharness/tools/shell_deny.go` | Verbatim. |
| `internal/tools/scrub.go` | `openharness/tools/scrub.go` | Verbatim. |
| `internal/tools/web_search.go` | `openharness/tools/builtin/web_search.go` | Change package to `builtin`. Add `Effects: tools.ToolEffectRead`. |
| `internal/tools/web_fetch.go` | `openharness/tools/builtin/web_fetch.go` | Change package to `builtin`. Add `Effects: tools.ToolEffectRead`. |
| `internal/tools/filesystem.go` | `openharness/tools/builtin/filesystem.go` | Change package. Per-definition Effects (Read/Mutate). |
| `internal/tools/exec.go` | `openharness/tools/builtin/exec.go` | Change package. `Effects: tools.ToolEffectMutate`. |
| `internal/tools/memory.go` | `openharness/tools/builtin/memory.go` | Change package. Delete local `MemoryEntry`, use `agent.MemoryEntry`. Remove `DefaultMemoryPath()`. Add `agent.MemoryStore` assertion. |
| `internal/tools/ask_user.go` | `openharness/tools/builtin/ask_user.go` | Change package. `Effects: tools.ToolEffectInteractive`. |
| `internal/tools/tasks.go` | `openharness/tools/builtin/tasks.go` | Change package. Per-definition Effects. Strip `sqliteonly`. |
| `internal/tools/knowledge_graph.go` | `openharness/tools/builtin/knowledge_graph.go` | Rewrite: replace `notes.Store` with local `KnowledgeGraphStore` interface. Replace `notes.Block`/`Page`/`Ref` with `KGBlock`/`KGPage`/`KGRef`. |
| `internal/tools/subagent.go` | `openharness/tools/builtin/subagent.go` | Rewrite: replace `(registry, tracker any, traceStore any)` with `SubagentConfig`. Dynamic JSON schema enum. |
| `internal/tools/builtin.go` | `openharness/tools/builtin/builtin.go` | Rewrite: `Register(r, cfg)` pattern. Drop `notes.WorkspaceSync` return. |

### Test files to move

All `*_test.go` files move alongside their source. Specific adjustments:
- `registry_test.go`: strip `sqliteonly`.
- `tasks_test.go`: strip `sqliteonly`.
- `knowledge_graph_test.go`: rewrite to use mock `KnowledgeGraphStore` instead of `notes.Store`.
- `subagent_test.go`: rewrite to use `SubagentConfig`.
- New: `builtin/builtin_test.go` — integration test verifying `Register` wires all tools with correct Effects.

### Files NOT moved (stay in ghostfin)

- `addepar_api.go` / `addepar_api_test.go` — finance-specific
- `tamarac_api.go` / `tamarac_api_test.go` — finance-specific
- `ghostfin_api.go` / `ghostfin_api_test.go` — finance-specific
- `schedule_cron.go` / `schedule_cron_test.go` — stays until Layer 3 scheduler extraction

---

## Task Sequencing (4 internal waves)

### Wave A — Core (sequential, foundation)
1. `tool.go` — interface + `ToolEffect` + `Effects` field
2. `registry.go` — strip sqliteonly, add `EffectsFor`
3. `composite.go`
4. `ssrf.go`, `shell_deny.go`, `scrub.go`, `external_content.go` (parallel subagents)

### Wave B — Simple built-ins (6 parallel subagents)
After core lands: `web_search`, `web_fetch`, `filesystem`, `exec`, `ask_user`, `tasks`. Each subagent copies file, changes package, adds Effects, moves test, runs `go test`.

### Wave C — Inversion built-ins (sequential)
1. `memory.go` — reconcile types, implement `MemoryStore`
2. `knowledge_graph.go` + `knowledge_graph_types.go` — local `KnowledgeGraphStore` interface, rewrite types
3. `subagent.go` — `SubagentConfig`, dynamic schema

### Wave D — Registration + verification
1. `builtin/builtin.go` — `Config` + `Register(r, cfg)`
2. `builtin/builtin_test.go` — integration test: all tools registered, effects match D4 table
3. Final `go test ./tools/... ./tools/builtin/...` green

---

## Verification

- [ ] `go test ./tools/...` — all core tool tests pass
- [ ] `go test ./tools/builtin/...` — all built-in tests pass
- [ ] `go vet ./tools/... ./tools/builtin/...` — clean
- [ ] Integration test: `Register(r, fullConfig)` → `r.Get("web_search")` exists, `r.EffectsFor("memory_store") == ToolEffectMutate`, etc.
- [ ] No imports of `ghostfin` anywhere in `openharness/tools/`
- [ ] No `//go:build sqliteonly` tags in any moved file

---

## Out of Scope

- MCP tools (`tools/mcp/`) — that's Plan 4
- Agent loop wiring (`tool_loop.go`, `mode.go` reading Effects) — that's Plan 5
- notes.Store adapter methods — that's Plan 7
- Actual subagent execution — that's Layer 4 (gateway)
- Storage backend abstraction for Memory tool — future Layer 3+
