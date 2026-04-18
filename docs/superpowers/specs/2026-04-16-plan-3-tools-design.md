# Layer 2 Plan 3: Tools — Design Spec

**Parent spec:** `docs/superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md`
**Execution context:** `docs/superpowers/plans/layer-2/2026-04-13-execution-order.md` (Wave 2 — depends on Plan 1 merged)

**Goal:** Move the generic tool system from `ghostfin/desktop/internal/tools/` into `openharness/tools/` (core) and `openharness/tools/core/` (9 built-in tool implementations). Add `Effects` and `Visibility` fields to `ToolDefinition` for loop-detection, mode-filtering classification, and UIBridge visibility. Invert `knowledge_graph` and `subagent` dependencies so they're vertical-agnostic.

**Branch:** `feat/layer-2-plan-3-tools`

**Review status (2026-04-18):** The adversarial review at the bottom of this spec has been processed. All Blockers (B1–B5) and Hard Questions (Q1–Q5) are resolved inline; design decisions below reflect the chosen resolutions. See the annotated review section for the mapping.

---

## Design Decisions

### D1: Knowledge graph inversion — local rich interface (Option A)

The `agent.KnowledgeStore` interface (Plan 1) has only `Search(ctx, query, limit)` and `Get(ctx, id)` — intentionally narrow for `retrieval.go` (Plan 5).

The `knowledge_graph` tool uses 8 methods on `notes.Store`: `Search`, `SearchInPage`, `GetPage`, `GetPageByID`, `GetBlockTree`, `GetBlockSnippets`, `GetRelatedPages`, `GetPagesWithSharedTags`.

**Decision:** Define a richer `core.KnowledgeGraphStore` interface locally in `openharness/tools/core/knowledge_graph.go`. The `agent.KnowledgeStore` stays narrow. Ghostfin's `notes.Store` satisfies both — the narrow one structurally, the rich one via adapter methods added in Plan 7.

**Rationale:** Different consumers want different contracts. Polluting `agent/interfaces.go` with tool-specific surface would force every future vertical to implement 8 methods when `retrieval.go` only needs 2.

**Interface shape:**

```go
package core

type KnowledgeGraphStore interface {
    Search(ctx context.Context, query string, limit int) ([]KGBlock, error)
    SearchInPage(ctx context.Context, pageID, query string, limit int) ([]KGBlock, error)
    GetPage(ctx context.Context, name string) (*KGPage, error)
    GetPageByID(ctx context.Context, id string) (*KGPage, error)
    GetBlockTree(ctx context.Context, pageID string) ([]KGBlock, error)
    GetBlockSnippets(ctx context.Context, pageID string, limit int) ([]string, error)
    GetRelatedPages(ctx context.Context, pageID string, depth int) ([]KGPage, []KGRef, error)
    GetPagesWithSharedTags(ctx context.Context, pageID string, limit int) ([]KGPage, error)
}
```

All methods take `ctx` for cancellation. Ghostfin's underlying `notes.Store` methods today have no ctx; Plan 7's adapter adds the ctx parameter at the interface boundary.

**Concrete types — flat, full field set** (`core/knowledge_graph_types.go`):

```go
type KGBlock struct {
    ID         string
    ParentID   *string
    PageID     string
    Title      string
    Order      int
    Collapsed  bool
    Properties map[string]string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type KGPage struct {
    // Block fields flattened (no embedding) — 9 fields
    ID         string
    ParentID   *string
    PageID     string
    Title      string
    Order      int
    Collapsed  bool
    Properties map[string]string
    CreatedAt  time.Time
    UpdatedAt  time.Time
    // Page-only fields — 3 fields
    Name       string
    JournalDay *string
    Icon       string
}

type KGRef struct {
    SourceBlockID string
    TargetPageID  string
    RefType       string
}
```

**Flatten rationale:** openharness's knowledge model should not assume `Page` specializes `Block` structurally. Future verticals whose knowledge schema diverges can still populate these types by field assignment without mirroring ghostfin's embedding choice. Ghostfin's Plan 7 adapter becomes a pure field-for-field copy: `core.KGPage{ID: p.ID, ..., Name: p.Name, ...}`.

**Field-set rationale:** Mirror ghostfin's current `notes.Page/Block/Ref` field sets exactly. Pruning unused fields later is cheap; discovering Plan 7 needs a field we did not include is expensive.

### D2: Memory tool — migration in ghostfin (Option C), `category` dropped (Option i)

The existing `tools/memory.go` defines its own `MemoryEntry` struct with fields `{Key, Content, Category, CreatedAt: string}`. Plan 1 shipped `agent.MemoryEntry{Key, Value, UpdatedAt: time.Time}`. On-disk JSON format differs; types don't round-trip.

**Decision:**
- Delete the tool's local `MemoryEntry`; use `agent.MemoryEntry` throughout.
- The built-in `Memory` tool implements `agent.MemoryStore` structurally. Compile-time assertion: `var _ agent.MemoryStore = (*Memory)(nil)`.
- **Rename `SearchEntries(query, limit)` → `Search(ctx, query, limit)`** to satisfy `agent.MemoryStore`. Matching semantics preserved verbatim from ghostfin: case-insensitive substring match against `Key` and `Value` (previously `Key`/`Content`/`Category`), stop at `limit`. Plan 5's `retrieval.go` depends on this scoring; behavior pinned by `core/memory_test.go`.
- **`category` parameter dropped from the `memory_store` JSON schema.** `agent.MemoryEntry` has no Category field; silently accepting the parameter and discarding it would be confusing. The search quality delta is small — substring match still hits `Key` and `Value`.
- JSON-file persistence stays. Storage backend abstraction is a future concern (Layer 3+).
- `DefaultMemoryPath()` is removed. Constructor is `NewMemory(storePath string)` — caller always passes the path.

**Migration (Option C — lives in ghostfin Plan 7, not openharness):**
- `openharness/tools/core/memory.go` only understands the new JSON format: `{Key, Value, UpdatedAt}`.
- Ghostfin's Plan 7 adds a one-time reader that converts `~/.ghostfin/data/memory.json` (old shape: `{key, content, category, created_at}`) → new shape on first boot, then the built-in tool operates normally.
- Dropped fields (`Category`, original `CreatedAt` string) are discarded; the migration note in the ghostfin release documents this minor data loss and the `category` parameter removal from the tool.

**Rationale:** Keeps openharness free of vertical-specific legacy shapes. Migration is ~30 LoC in ghostfin, runs once per user, then never again. YAGNI on storage abstraction.

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

**Empty-map semantics:** If `len(cfg.Subagents) == 0`, `Register` omits the subagent tool entirely. A JSON schema enum with zero entries is invalid, so registration is conditional on at least one spec.

**Forward-looking scope:** Plan 3's `SubagentTool` stores configuration only and is not a frozen contract. Layer 4's gateway will extend the public surface with registry/runner/tracer dependencies to actually execute subagents. The `AllowedTools` field on `SubagentSpec` is carried forward but not enforced until Layer 4.

**Rationale:** Ghostfin's 5 finance presets become data passed at registration time. Generic verticals pass one "general" entry or an empty map (which omits the tool).

### D4: Effects and Visibility classification

Both `Effects` and `Visibility` live on `ToolDefinition` (per-definition, not per-tool), since tools like `filesystem` and `memory` have mixed read/mutate definitions. Landing both fields together in Plan 3 matches the parent spec's key-decision #6 commitment and avoids re-churning `ToolDefinition` in Plan 4 when UIBridge semantics get wired.

```go
type ToolEffect int

const (
    ToolEffectUnspecified ToolEffect = iota // zero value; Register errors if any definition has this
    ToolEffectRead                          // increments read-streak; allowed in ModePlan
    ToolEffectMutate                        // resets read-streak; blocked in ModePlan
    ToolEffectNeutral                       // neither increments nor resets; allowed in ModePlan
    ToolEffectInteractive                   // resets read-streak; allowed in ModePlan
)

type ToolVisibility int // reserved; Plan 4 populates UIBridge-facing semantics
const (
    ToolVisibilityDefault ToolVisibility = iota
    ToolVisibilityModelOnly
    ToolVisibilityAppOnly
)

type ToolDefinition struct {
    Name        string
    Description string
    Parameters  json.RawMessage
    Effects     ToolEffect     `json:"-"` // never serialized to the LLM
    Visibility  ToolVisibility `json:"-"` // reserved; Plan 4 wires semantics
}
```

**Neutral category preserves ghostfin's `tool_loop.go:161` behavior.** Today `exec` is the only tool that neither resets the read-streak nor increments it, because its effect is ambiguous (could be `ls` or `rm`). The 5-state enum keeps this semantics rather than silently collapsing it into Mutate. `ModePlan` allows `Neutral` because `exec`'s real gate is the Layer 6 SDK admission rule (see `docs/superpowers/specs/2026-04-17-layer-6-sdk-design.md`: `exec` permitted only if `sandbox.type ∈ {runloop, firecracker, gvisor, lambda}` ∧ `trust_mode == "safe"`), not Plan 3's mode filter.

**`Unspecified` zero value.** Unclassified definitions cause `Register` to return an error — no silent defaults. If the agent loop ever encounters `Unspecified` at runtime (shouldn't happen after a clean `Register`), it treats the call as `Mutate` for safety fallback.

**`Interactive` semantics** (wired in Plan 5):
- Resets read-streak — a user reply typically redirects the task.
- Allowed in ModePlan — asking for clarification is non-destructive.

**Mapping (preserves ghostfin's current loop-detection behavior):**

| Tool | Definition(s) | Effects |
|---|---|---|
| `web_search` | `web_search` | Read |
| `web_fetch` | `web_fetch` | Read |
| `filesystem` | `read_file`, `list_directory`, `search_files` | Read |
| `filesystem` | `write_file`, `edit_file`, `delete_file`, `make_directory` | Mutate |
| `exec` | `exec` / `shell` | **Neutral** |
| `memory` | `memory_search`, `memory_get` | Read |
| `memory` | `memory_store` | Mutate |
| `ask_user` | `ask_user` | Interactive |
| `tasks` | `task_get`, `task_list` | Read |
| `tasks` | `task_create`, `task_update` | Mutate |
| `knowledge_graph` | `knowledge_graph_search` | Read |
| `subagent` | `agent` | Mutate |

**`Registry.EffectsFor(name string) ToolEffect` algorithm.** Precomputed at `Register` time: `Registry.Register(tool Tool)` walks `tool.Definitions()` and populates `map[defName]ToolEffect`. Lookups are O(1). For unknown names, returns `ToolEffectUnspecified` (not an error; caller decides). Plan 5's `tool_loop.go` calls this on every invocation — the map is the hot path.

### D5: Built-in tools constructor — `Register(r, cfg)` pattern

Package renamed `builtin` → `core` to avoid mental collision with Go's stdlib `builtin` pseudo-package. Rename happens before any callers exist.

```go
package core

type Config struct {
    WorkspacePath   string
    MemoryPath      string
    KnowledgeStore  KnowledgeGraphStore // optional; nil skips knowledge_graph
    Subagent        SubagentConfig      // optional; empty map omits subagent tool
}

func Register(r *tools.Registry, cfg Config) error
```

**Rationale:**
- `Register(r, cfg)` not `NewRegistry(cfg)` — callers bring their own registry to mix built-ins with vertical-specific tools.
- `error` return is load-bearing: `Register` validates that every registered definition has `Effects != ToolEffectUnspecified`. Also a forward-compat seam for future config validation.
- No `notes.WorkspaceSync` return — that's a ghostfin/notes concern.
- `KnowledgeStore` optional: nil means `knowledge_graph` isn't registered (matches today's `if notesStore != nil` branch).
- `Subagent`: zero-value or empty map omits the subagent tool entirely (JSON schema enum requires at least one entry).

---

## Package Layout

```
openharness/tools/
├── tool.go              # Tool interface + ToolDefinition (Effects, Visibility) + both enums
├── registry.go          # Registry (sqliteonly tag stripped) + precomputed EffectsFor map
├── composite.go         # Multi-tool wrapper
├── external_content.go  # External content guard helpers
├── ssrf.go              # SSRF URL validator
├── shell_deny.go        # Shell command denylist
├── scrub.go             # Secret scrubbing for tool output
└── core/
    ├── core.go                     # Config + Register(r, cfg)
    ├── web_search.go
    ├── web_fetch.go
    ├── filesystem.go
    ├── exec.go
    ├── memory.go                   # Implements agent.MemoryStore
    ├── memory_test.go              # NEW: round-trip + Search semantics + MemoryStore satisfaction
    ├── ask_user.go
    ├── tasks.go
    ├── knowledge_graph.go          # KnowledgeGraphStore interface
    ├── knowledge_graph_types.go    # KGPage, KGBlock, KGRef (flat, full ghostfin field set)
    ├── subagent.go                 # SubagentConfig + SubagentSpec
    ├── core_test.go                # NEW: integration test + D4 matrix assertions
    └── *_test.go
```

**Dependency direction:**
- `tools` → stdlib + `golang.org/x/net` (HTML parsing used by `web_fetch`/`web_search`)
- `tools/core` → `tools` + `agent` (for `MemoryEntry`, `Block`, `MemoryStore`)
- Neither depends on `providers`, `sessions`, `cost`, etc.

---

## File Inventory

### Files to copy from ghostfin (strip `//go:build sqliteonly` + `// +build sqliteonly`, otherwise verbatim unless noted)

Note on build tags: the `sqliteonly` tag in ghostfin is a repo-wide feature flag, not a SQLite-driver gate. None of the tools files (including `tasks.go`) import `database/sql` or any SQLite driver — all persistence is JSON-file via `os.WriteFile`. Stripping the tag introduces no new dependencies. (`sessions/pg_store.go` keeps its `!sqliteonly` guard in Plan 1/Layer 2 because it's genuinely Postgres-dependent — different situation.)

| Source | Destination | Notes |
|---|---|---|
| `internal/tools/tool.go` | `openharness/tools/tool.go` | Add `ToolEffect` + `ToolVisibility` enums + `Effects`/`Visibility` fields to `ToolDefinition` (both `json:"-"`). |
| `internal/tools/registry.go` | `openharness/tools/registry.go` | Strip `sqliteonly`. Precompute `map[defName]ToolEffect` at `Register` time; `EffectsFor(name)` is an O(1) lookup returning `ToolEffectUnspecified` for unknown names. |
| `internal/tools/composite.go` | `openharness/tools/composite.go` | Verbatim. |
| `internal/tools/external_content.go` | `openharness/tools/external_content.go` | Verbatim. |
| `internal/tools/ssrf.go` | `openharness/tools/ssrf.go` | Verbatim. |
| `internal/tools/shell_deny.go` | `openharness/tools/shell_deny.go` | Verbatim. |
| `internal/tools/scrub.go` | `openharness/tools/scrub.go` | Verbatim. |
| `internal/tools/web_search.go` | `openharness/tools/core/web_search.go` | Change package to `core`. Add `Effects: tools.ToolEffectRead`. |
| `internal/tools/web_fetch.go` | `openharness/tools/core/web_fetch.go` | Change package to `core`. Add `Effects: tools.ToolEffectRead`. |
| `internal/tools/filesystem.go` | `openharness/tools/core/filesystem.go` | Change package. Per-definition Effects (Read/Mutate). |
| `internal/tools/exec.go` | `openharness/tools/core/exec.go` | Change package. **`Effects: tools.ToolEffectNeutral`** (preserves `tool_loop.go:161` neutral behavior; Layer 6 admission rule is the real gate). |
| `internal/tools/memory.go` | `openharness/tools/core/memory.go` | Change package. Delete local `MemoryEntry`, use `agent.MemoryEntry`. Remove `DefaultMemoryPath()`. Rename `SearchEntries` → `Search(ctx, q, n)` preserving substring-match semantics. Drop `category` from `memory_store` JSON schema. Add `var _ agent.MemoryStore = (*Memory)(nil)` assertion. Per-definition Effects (search/get = Read, store = Mutate). |
| `internal/tools/ask_user.go` | `openharness/tools/core/ask_user.go` | Change package. `Effects: tools.ToolEffectInteractive`. |
| `internal/tools/tasks.go` | `openharness/tools/core/tasks.go` | Change package. Strip `sqliteonly`. Per-definition Effects. |
| `internal/tools/knowledge_graph.go` | `openharness/tools/core/knowledge_graph.go` | Rewrite: replace `notes.Store` with local `KnowledgeGraphStore` interface. Replace `notes.Block`/`Page`/`Ref` with `KGBlock`/`KGPage`/`KGRef`. Add ctx to all methods. Effects: `Read`. |
| `internal/tools/subagent.go` | `openharness/tools/core/subagent.go` | Rewrite: replace `(registry, tracker any, traceStore any)` with `SubagentConfig`. Dynamic JSON schema enum. `Effects: Mutate`. |
| `internal/tools/builtin.go` | `openharness/tools/core/core.go` | Rewrite: `Register(r, cfg)` pattern. Drop `notes.WorkspaceSync` return. Validates `Effects != Unspecified` for every registered definition. |

### Test files

Moved tests (strip `sqliteonly` where present):
- `composite_test.go`, `scrub_test.go`, `shell_deny_test.go`, `ssrf_test.go` → `openharness/tools/`. Verbatim minus build tag.
- `ask_user_test.go`, `exec_test.go`, `filesystem_test.go`, `tasks_test.go` → `openharness/tools/core/`. Package change; strip `sqliteonly` from `tasks_test.go`.
- `knowledge_graph_test.go` → `openharness/tools/core/`. Rewrite to use mock `KnowledgeGraphStore` instead of real `notes.Store`.
- `subagent_test.go` → `openharness/tools/core/`. Rewrite to use `SubagentConfig`.

New tests authored in Plan 3:
- `openharness/tools/core/memory_test.go` — **new file.** Coverage: new-format JSON round-trip (write entry → construct fresh `Memory` on same path → `Search` returns it); `var _ agent.MemoryStore = (*Memory)(nil)` satisfaction; substring-match semantics pinned verbatim against ghostfin's `SearchEntries` behavior for `Key` and `Value` fields.
- `openharness/tools/core/core_test.go` — integration test. Coverage: `Register(r, fullConfig)` wires all tools; D4 classification matrix asserted one row at a time (12 definition-level assertions); `Register` returns error if any definition has `Effects: Unspecified`; `EffectsFor("unknown_tool")` returns `ToolEffectUnspecified` with no error; empty `SubagentConfig` omits the `agent` tool from the registry; nil `KnowledgeStore` omits `knowledge_graph_search`.

No `registry_test.go` exists in ghostfin today (removed from the move list); coverage moves to `core_test.go` where `EffectsFor` lives in context with the matrix.

### Files NOT moved (stay in ghostfin)

- `addepar_api.go` / `addepar_api_test.go` — finance-specific
- `tamarac_api.go` / `tamarac_api_test.go` — finance-specific
- `ghostfin_api.go` / `ghostfin_api_test.go` — finance-specific
- `schedule_cron.go` / `schedule_cron_test.go` — stays until Layer 3 scheduler extraction

---

## Task Sequencing (4 internal waves)

### Wave A — Core (sequential, foundation)
1. `tool.go` — `Tool` interface + `ToolEffect` enum (5 values) + `ToolVisibility` enum (3 values) + `Effects`/`Visibility` fields on `ToolDefinition`
2. `registry.go` — strip sqliteonly, precompute `EffectsFor` map at `Register` time
3. `composite.go`
4. `ssrf.go`, `shell_deny.go`, `scrub.go`, `external_content.go` (parallel subagents)

### Wave B — Simple built-ins (6 parallel subagents)
After core lands: `web_search`, `web_fetch`, `filesystem`, `exec` (Effects: Neutral), `ask_user` (Effects: Interactive), `tasks`. Each subagent copies file, changes package to `core`, adds Effects per D4 table, moves test, runs `go test`.

### Wave C — Inversion built-ins (sequential)
1. `memory.go` — rename `SearchEntries`→`Search(ctx,…)`, reconcile to `agent.MemoryEntry`, drop `category` from schema, add `MemoryStore` assertion, author `memory_test.go`
2. `knowledge_graph.go` + `knowledge_graph_types.go` — local `KnowledgeGraphStore` interface, flat KG types with full ghostfin field set
3. `subagent.go` — `SubagentConfig`, dynamic schema, empty-map omission

### Wave D — Registration + verification
1. `core/core.go` — `Config` + `Register(r, cfg)` with `Effects != Unspecified` validation
2. `core/core_test.go` — integration test: D4 matrix (12 assertions), `EffectsFor("unknown")` contract, empty `SubagentConfig` omission, nil `KnowledgeStore` omission
3. Final `go test ./tools/... ./tools/core/...` green

---

## Verification

- [ ] `go test ./tools/...` — all core tool tests pass
- [ ] `go test ./tools/core/...` — all built-in tests pass
- [ ] `go vet ./tools/... ./tools/core/...` — clean
- [ ] **D4 matrix test** in `core_test.go`: 12 rows of the Effects classification table → 12 definition-level assertions (`Registry.EffectsFor("web_search") == ToolEffectRead`, `Registry.EffectsFor("exec") == ToolEffectNeutral`, …, `Registry.EffectsFor("agent") == ToolEffectMutate`).
- [ ] **`EffectsFor("unknown_tool")` contract test**: returns `ToolEffectUnspecified`, no error.
- [ ] **`Register` validation test**: passing a `Tool` whose definition has `Effects: ToolEffectUnspecified` causes `Register` to return a non-nil error naming the offending definition.
- [ ] **Memory round-trip test** (`memory_test.go`): write entry; construct fresh `Memory` on same path; `Search` returns it. No old-format compatibility required (migration lives in ghostfin Plan 7).
- [ ] **`MemoryStore` satisfaction**: `var _ agent.MemoryStore = (*Memory)(nil)` compiles.
- [ ] **Substring-match parity**: table-driven test pins `Search` behavior against ghostfin's `SearchEntries` semantics for `Key` and `Value` fields.
- [ ] **Loop-detection parity invariant** (declared here; fixture test lives in Plan 5): `Effects` classifier output must preserve ghostfin's current `tool_loop.go` behavior — `write_file`, `edit_file`, `agent`, `memory_store` reset the streak; `exec` is neutral; every other registered definition increments.
- [ ] **Integration test**: `Register(r, fullConfig)` → `r.Get("web_search")` exists; empty `SubagentConfig` → `r.Get("agent")` absent; nil `KnowledgeStore` → `r.Get("knowledge_graph_search")` absent.
- [ ] No imports of `ghostfin` anywhere in `openharness/tools/`.
- [ ] No `//go:build sqliteonly` tags in any moved file.

---

## Out of Scope

- MCP tools (`tools/mcp/`) — that's Plan 4
- Agent loop wiring (`tool_loop.go`, `mode.go` reading Effects) — that's Plan 5
- notes.Store adapter methods — that's Plan 7
- Actual subagent execution — that's Layer 4 (gateway)
- Storage backend abstraction for Memory tool — future Layer 3+

---

## Adversarial Review (2026-04-16) — **resolved 2026-04-18**

Review conducted after brainstorming against ground-truth: Plan 1's shipped `openharness/agent/` package, the parent spec, the execution-order doc, and the actual ghostfin source files listed in the File Inventory. All Blockers and Hard Questions were resolved in a second brainstorming pass on 2026-04-18; each finding below is annotated with its resolution location in the design sections above.

### Blockers

#### B1. Memory data migration is unacknowledged and types don't line up **[resolved — see D2]**

Plan 1 (merged) shipped:

```go
type MemoryEntry struct {
    Key       string
    Value     string
    UpdatedAt time.Time
}
```

Ghostfin's live `internal/tools/memory.go` uses:

```go
type MemoryEntry struct {
    Key       string `json:"key"`
    Content   string `json:"content"`
    Category  string `json:"category"`
    CreatedAt string `json:"created_at"`
}
```

D2 says "use `agent.MemoryEntry` throughout" but does not address:
- **Existing `~/.ghostfin/data/memory.json` files cannot round-trip.** Field names and the `created_at` type (string vs `time.Time`) both differ. Any upgrading user loses their stored memory on first boot.
- **`Category` is dropped** — but the `memory_store` tool JSON schema still accepts a `category` parameter. Silent behavior change unless explicitly dropped from the schema too.
- **`CreatedAt` vs `UpdatedAt`** are different semantics (append-only vs mutable). The spec doesn't say which wins.

**Resolution required:** Add a "Memory migration" subsection that either (a) ships a one-shot JSON reader that upgrades old entries in place, or (b) declares the break explicitly in migration notes (matching how the spec treats MCP OAuth re-auth at parent-spec §"MCP OAuth via SecretStore"). Silently breaking user data is worse than either option.

**Resolution chosen (2026-04-18):** Option C — migration lives in ghostfin Plan 7, not openharness. `core/memory.go` only understands the new format. `category` parameter dropped from the `memory_store` JSON schema (agent.MemoryEntry has no Category field; Option i). See D2.

#### B2. `agent.MemoryStore` is one method — the `var _` assertion is almost empty **[resolved — see D2]**

```go
type MemoryStore interface {
    Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error)
}
```

D2 promises `var _ agent.MemoryStore = (*Memory)(nil)`. Satisfying this requires the memory tool to expose a method named *exactly* `Search(ctx, string, int) ([]agent.MemoryEntry, error)`. Ghostfin today exports `SearchEntries(query string, limit int) ([]MemoryEntry, error)` — different name, no ctx, local return type.

So the migration actually:
1. Renames `SearchEntries` → `Search` (grep callers before doing this).
2. Adds `ctx` the body currently ignores.
3. Switches the element type, forcing internal rewiring.

None of this is wrong, but "use `agent.MemoryEntry` throughout" undersells it. Additionally, the interface contract does not pin the *algorithm* — Plan 5's `retrieval.go` depends on whatever `Search` returns, and the memory tool's current substring-match semantics must be preserved verbatim or retrieval quality silently regresses.

**Resolution required:** Commit explicitly that `Search` preserves existing `SearchEntries` scoring/matching behavior. Add `builtin/memory_test.go` (new file — see B5) that pins this.

**Resolution chosen (2026-04-18):** `Search(ctx, query, limit) ([]agent.MemoryEntry, error)` preserves ghostfin's case-insensitive substring semantics verbatim against `Key` and `Value` (formerly `Key`/`Content`/`Category`). Pinned by `core/memory_test.go`. See D2 and Verification.

#### B3. "Effects defaults to Read (conservative)" is backwards **[resolved — see D4]**

D4 says:
> `ToolEffectRead` is the zero value — unclassified tools default to Read (conservative: doesn't reset read streaks).

This is the wrong direction on safety. An *unknown* tool defaulting to Read means:
- The loop detector treats it as streak-safe — the agent can invoke a tool that actually mutates many times in a row without tripping the read-streak guard.
- The mode filter allows it in `ModePlan` (which is supposed to be read-only).

Both are safety regressions for any tool the author forgets to classify. Ghostfin's current `tool_loop.go` also has an explicit "neither reset nor increment" path for `exec` (line 161) — a neutral category that Plan 3's two-state-plus-Interactive enum erases.

**Resolution required:** Pick one:
- Option A — zero value is `ToolEffectMutate`: unknown tools treated as scariest, which is correct for both consumers.
- Option B — introduce `ToolEffectUnspecified` as the zero value; `Registry.EffectsFor` returns an error (or panics at registration) on unclassified definitions, forcing explicit classification.
- Option C — preserve the "neutral" exec behavior by adding a fourth state, and commit its semantics for both `tool_loop.go` and `mode.go`.

**Resolution chosen (2026-04-18):** Hybrid of B + C — 5-state enum (`Unspecified`, `Read`, `Mutate`, `Neutral`, `Interactive`). Zero = `Unspecified`; `Register` errors on unclassified definitions. `exec: Neutral` preserves `tool_loop.go:161` behavior. Layer 6 SDK admission rule is `exec`'s real gate, not the Plan 3 mode filter. See D4 for the enum definition, neutral semantics, and ModePlan allowance.

#### B4. `tasks.go` build-tag handling is underspecified **[resolved — premise of review was incorrect]**

Ground truth from ghostfin:
- `internal/tools/tasks.go` → `//go:build sqliteonly`
- `internal/tools/tasks_test.go` → same tag
- `internal/tools/registry.go` → same tag
- `internal/tools/builtin.go` → same tag

Plan 3 says "strip sqliteonly" generically. Open questions:
- If `tasks.go` uses `database/sql` + a SQLite driver, stripping the tag makes `openharness/tools/builtin` unconditionally depend on SQLite. Is that the Layer 2 rule?
- If the tag is stripped from source but left on `tasks_test.go`, the test never runs in openharness CI (no `sqliteonly` tag in this module). Coverage silently vanishes.
- Execution-order note #3 preserves build tags for `sessions/pg_store.go` because it's platform-dependent. Why is tasks different? The inconsistency needs a rationale.

**Resolution required:** Pick a rule and apply uniformly:
- Recommended: `tasks` compiles unconditionally in openharness with a SQLite dependency (Layer 3 can re-abstract); `sessions/pg_store.go` keeps its `!sqliteonly` guard because it's genuinely platform-dependent. State this in the spec.

**Resolution chosen (2026-04-18):** The review's premise was wrong. `tasks.go` does not import `database/sql` or any SQLite driver — it uses `os.WriteFile` + `json.MarshalIndent` for persistence, identical to `memory.go`. The `sqliteonly` tag is a ghostfin-wide feature-flag artifact, not a dependency gate. Stripping the tag introduces no SQLite dependency. `sessions/pg_store.go` keeps its `!sqliteonly` guard because it genuinely is Postgres-dependent (different situation). See the File Inventory's tag-handling note.

#### B5. Test files assumed by the File Inventory don't exist **[resolved — see File Inventory § Test files]**

The "Test files to move" subsection lists `registry_test.go` (with sqliteonly-strip instructions) and implies `memory_test.go`. Neither exists in `ghostfin/desktop/internal/tools/` (verified 2026-04-16).

Plan 3 performs the riskiest changes (type migration, interface assertion, constructor signature change) to the `memory` tool — with zero existing test coverage. 

**Resolution required:** Update the File Inventory:
- Add `builtin/memory_test.go` as a **new file** to author. Coverage must include: old-JSON-format compatibility (per B1 resolution), `agent.MemoryStore` satisfaction, `Search` semantics preservation.
- Either drop `registry_test.go` from the "move" list or add it as a new file.

**Resolution chosen (2026-04-18):** `core/memory_test.go` authored new (no old-format coverage needed since migration is in ghostfin Plan 7 per B1). `registry_test.go` dropped from the move list; the `EffectsFor` + matrix coverage lives in `core/core_test.go` instead. See File Inventory § Test files.

### Hard questions to resolve before enum/struct shapes freeze

#### Q1. `Registry.EffectsFor(name)` — algorithm and caching **[resolved — see D4]**

A Tool publishes multiple Definitions (`filesystem` → 4). Ghostfin's Registry indexes every def-name to its owning Tool. `EffectsFor(name)` therefore must (a) find the owning Tool, (b) walk `Definitions()` to find the matching def, (c) return its `Effects`. Either describe this algorithm or build a `map[defName]ToolEffect` at register time — Plan 5 will call this in the loop-detection hot path.

**Resolution chosen (2026-04-18):** Precomputed `map[defName]ToolEffect` built during `Register`. O(1) lookup. Unknown names return `ToolEffectUnspecified` (not an error). See D4.

#### Q2. `Visibility` field — in Plan 3 or Plan 4? **[resolved — land in Plan 3, see D4]**

Parent spec (key decision 6) commits `ToolDefinition.Visibility ToolVisibility` alongside `Effects`. Plan 3 drops it. Adding it in Plan 4 churns `ToolDefinition` again — a struct that Plan 3 is supposed to freeze. Decide now: land both fields together (cheap; keeps the shape stable), or state explicitly "Visibility deferred to Plan 4" with rationale.

**Resolution chosen (2026-04-18):** Land both fields together in Plan 3. `Visibility` gets its 3-value enum (`Default`, `ModelOnly`, `AppOnly`) now; Plan 4 populates the UIBridge-facing semantics. See D4.

#### Q3. `ToolEffectInteractive` semantics for both consumers **[resolved — see D4]**

D4 marks `ask_user` as Interactive — a third category. Does Interactive reset the read-streak? Is it allowed in `ModePlan`? Plan 5 is where these wire up, but Plan 3 freezes the enum. If Plan 5 needs a fourth value or a bitfield, the enum gets redone. Commit semantics now.

**Resolution chosen (2026-04-18):** Interactive resets the read-streak (user reply typically redirects the task); allowed in ModePlan (asking for clarification is non-destructive). See D4 § Interactive semantics.

#### Q4. `SubagentTool`'s post-Layer-4 shape **[resolved — see D3]**

`SubagentConfig` drops registry + tracker from the constructor. Layer 4's gateway will need a registry, an LLM runner, and probably a `TraceRecorder` to actually execute subagents. So `SubagentTool`'s public surface will grow again.

**Resolution:** Add a sentence to D3: "Plan 3's `SubagentTool` stores configuration only and is not a frozen contract; Layer 4 will add execution wiring."

**Resolution chosen (2026-04-18):** Sentence added to D3. Also spelled out empty-map semantics: `len(cfg.Subagents) == 0` causes `Register` to omit the subagent tool entirely (JSON schema enum requires at least one entry).

#### Q5. `KGBlock/KGPage/KGRef` — "field-compatible" is imprecise **[resolved — see D1]**

Go has no structural subtyping for structs. Even identical field sets require explicit conversion. D1 actually means "field-*convertible*" — Plan 7's adapter methods will do `kgb := KGBlock{ID: nb.ID, Title: nb.Title, …}`. Additionally:
- `notes.Block` has 9 fields; `notes.Page` embeds `Block` + 3 more. Do the openharness types embed or flatten?
- The minimum viable field set for each KG type is not specified. Plan 7's adapter can't be written against a contract that doesn't exist yet.

**Resolution:** List the exact field set for each of `KGBlock`, `KGPage`, `KGRef`. Decide embed-vs-flatten.

**Resolution chosen (2026-04-18):** Flat (no embedding), full ghostfin field set. See D1 § Concrete types for the exact field list and rationale.

### Nits and missing rationale **[resolved 2026-04-18]**

- **`Register(r, cfg) error` — dead return.** D5 admits no current validation. YAGNI. Return `void` now; add an error-returning variant when needed. An unused error return invites `_ = Register(...)` at every callsite.
  - *Resolution:* Kept. The `error` return is no longer dead — `Register` now validates `Effects != ToolEffectUnspecified` for every registered definition. See D5.
- **`Config.Subagent` as top-level.** Only two built-ins (Subagent, KnowledgeStore) get first-class Config fields; the rest take path strings. If future tools need config (Memory TTL, Tasks quota), the struct grows forever. Consider nested `BuiltinsConfig{Subagent, Memory, Tasks}` or option functions. Not a blocker — add one sentence of rationale.
  - *Resolution:* Deferred. Nested grouping can land when the second tool-specific config field actually appears. Flat struct is fine today.
- **Package name `builtin`.** Not reserved, but collides mentally with the stdlib `builtin` package. Consider `std` or `core`. Cheap rename; do it before any callers exist.
  - *Resolution:* Renamed to `core`. See D5 and Package Layout.
- **Zero-value SubagentConfig semantics.** D3 says "empty map omits the tool." A JSON schema enum with zero entries is invalid, so `Register` must conditionally skip when `len(cfg.Subagent.Subagents) == 0`. Spell this out.
  - *Resolution:* Spelled out. See D3 § Empty-map semantics and D5 `Subagent` field comment.
- **Dependency direction claim.** "tools → stdlib only" is inaccurate: `web_search.go` and `web_fetch.go` import `golang.org/x/net/html`. Update to "stdlib + golang.org/x/net".
  - *Resolution:* Updated. See Package Layout § Dependency direction.

### Verification section gaps **[resolved — see updated Verification section]**

Current list is too weak. Add:
- **Matrix test at the definition level** (not just tool level): every row of the D4 table → one assertion. 12 rows → 12 test cases.
- **`EffectsFor("unknown_tool")` contract test** — returns the default? Returns an error? Make it explicit.
- **Memory-tool round-trip test** — write entry, restart, read back. Gates B1's migration path.
- **Loop-detection parity invariant** — Plan 3 declares the invariant ("classifier output must match the existing hardcoded allowlist for the 4 current entries"), Plan 5 runs the real fixture test.

All four items are now included in the Verification section above.

### What this review does NOT cover

- Plan 4 (MCP) interactions beyond the `Visibility` field question.
- Plan 5 (agent loop) wiring correctness — only Plan 3's committed contracts as inputs to Plan 5.
- Plan 7 (ghostfin rewrite) — assumed to handle the ghostfin-side migration once B1's path is chosen.
- Performance of `EffectsFor` — only structural correctness.
