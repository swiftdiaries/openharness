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

---

## Adversarial Review (2026-04-16)

Review conducted after brainstorming against ground-truth: Plan 1's shipped `openharness/agent/` package, the parent spec, the execution-order doc, and the actual ghostfin source files listed in the File Inventory. Findings are grouped by severity. Blockers should be resolved in this spec before implementation; hard questions should be answered before the enum/struct shapes freeze; nits should be addressed during implementation.

### Blockers

#### B1. Memory data migration is unacknowledged and types don't line up

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

#### B2. `agent.MemoryStore` is one method — the `var _` assertion is almost empty

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

#### B3. "Effects defaults to Read (conservative)" is backwards

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

#### B4. `tasks.go` build-tag handling is underspecified

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

#### B5. Test files assumed by the File Inventory don't exist

The "Test files to move" subsection lists `registry_test.go` (with sqliteonly-strip instructions) and implies `memory_test.go`. Neither exists in `ghostfin/desktop/internal/tools/` (verified 2026-04-16).

Plan 3 performs the riskiest changes (type migration, interface assertion, constructor signature change) to the `memory` tool — with zero existing test coverage. 

**Resolution required:** Update the File Inventory:
- Add `builtin/memory_test.go` as a **new file** to author. Coverage must include: old-JSON-format compatibility (per B1 resolution), `agent.MemoryStore` satisfaction, `Search` semantics preservation.
- Either drop `registry_test.go` from the "move" list or add it as a new file.

### Hard questions to resolve before enum/struct shapes freeze

#### Q1. `Registry.EffectsFor(name)` — algorithm and caching

A Tool publishes multiple Definitions (`filesystem` → 4). Ghostfin's Registry indexes every def-name to its owning Tool. `EffectsFor(name)` therefore must (a) find the owning Tool, (b) walk `Definitions()` to find the matching def, (c) return its `Effects`. Either describe this algorithm or build a `map[defName]ToolEffect` at register time — Plan 5 will call this in the loop-detection hot path.

#### Q2. `Visibility` field — in Plan 3 or Plan 4?

Parent spec (key decision 6) commits `ToolDefinition.Visibility ToolVisibility` alongside `Effects`. Plan 3 drops it. Adding it in Plan 4 churns `ToolDefinition` again — a struct that Plan 3 is supposed to freeze. Decide now: land both fields together (cheap; keeps the shape stable), or state explicitly "Visibility deferred to Plan 4" with rationale.

#### Q3. `ToolEffectInteractive` semantics for both consumers

D4 marks `ask_user` as Interactive — a third category. Does Interactive reset the read-streak? Is it allowed in `ModePlan`? Plan 5 is where these wire up, but Plan 3 freezes the enum. If Plan 5 needs a fourth value or a bitfield, the enum gets redone. Commit semantics now.

#### Q4. `SubagentTool`'s post-Layer-4 shape

`SubagentConfig` drops registry + tracker from the constructor. Layer 4's gateway will need a registry, an LLM runner, and probably a `TraceRecorder` to actually execute subagents. So `SubagentTool`'s public surface will grow again.

**Resolution:** Add a sentence to D3: "Plan 3's `SubagentTool` stores configuration only and is not a frozen contract; Layer 4 will add execution wiring."

#### Q5. `KGBlock/KGPage/KGRef` — "field-compatible" is imprecise

Go has no structural subtyping for structs. Even identical field sets require explicit conversion. D1 actually means "field-*convertible*" — Plan 7's adapter methods will do `kgb := KGBlock{ID: nb.ID, Title: nb.Title, …}`. Additionally:
- `notes.Block` has 9 fields; `notes.Page` embeds `Block` + 3 more. Do the openharness types embed or flatten?
- The minimum viable field set for each KG type is not specified. Plan 7's adapter can't be written against a contract that doesn't exist yet.

**Resolution:** List the exact field set for each of `KGBlock`, `KGPage`, `KGRef`. Decide embed-vs-flatten.

### Nits and missing rationale

- **`Register(r, cfg) error` — dead return.** D5 admits no current validation. YAGNI. Return `void` now; add an error-returning variant when needed. An unused error return invites `_ = Register(...)` at every callsite.
- **`Config.Subagent` as top-level.** Only two built-ins (Subagent, KnowledgeStore) get first-class Config fields; the rest take path strings. If future tools need config (Memory TTL, Tasks quota), the struct grows forever. Consider nested `BuiltinsConfig{Subagent, Memory, Tasks}` or option functions. Not a blocker — add one sentence of rationale.
- **Package name `builtin`.** Not reserved, but collides mentally with the stdlib `builtin` package. Consider `std` or `core`. Cheap rename; do it before any callers exist.
- **Zero-value SubagentConfig semantics.** D3 says "empty map omits the tool." A JSON schema enum with zero entries is invalid, so `Register` must conditionally skip when `len(cfg.Subagent.Subagents) == 0`. Spell this out.
- **Dependency direction claim.** "tools → stdlib only" is inaccurate: `web_search.go` and `web_fetch.go` import `golang.org/x/net/html`. Update to "stdlib + golang.org/x/net".

### Verification section gaps

Current list is too weak. Add:
- **Matrix test at the definition level** (not just tool level): every row of the D4 table → one assertion. 12 rows → 12 test cases.
- **`EffectsFor("unknown_tool")` contract test** — returns the default? Returns an error? Make it explicit.
- **Memory-tool round-trip test** — write entry, restart, read back. Gates B1's migration path.
- **Loop-detection parity invariant** — Plan 3 declares the invariant ("classifier output must match the existing hardcoded allowlist for the 4 current entries"), Plan 5 runs the real fixture test.

### What this review does NOT cover

- Plan 4 (MCP) interactions beyond the `Visibility` field question.
- Plan 5 (agent loop) wiring correctness — only Plan 3's committed contracts as inputs to Plan 5.
- Plan 7 (ghostfin rewrite) — assumed to handle the ghostfin-side migration once B1's path is chosen.
- Performance of `EffectsFor` — only structural correctness.
