# Layer 2 Plans: Dependency Graph & Execution Strategy

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement each wave's plans task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Execute 8 Layer 2 implementation plans in optimal dependency order, maximizing parallelism across Claude Code sessions.

**Spec:** `docs/superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md`

**Scope:** Extract agent loop, tool system, LLM providers, MCP integration, and four thin Layer 3 concretes (cost/sessions/edition/config) from `ghostfin/desktop/internal/` into the `openharness` module. Ship a native Anthropic provider with streaming + prompt caching, MCP OAuth via `SecretStore`, a `UIBridge` for interactive tool UIs, tool self-classification via `Effects`, and a `/create-openharness-app` scaffolding skill.

---

## Dependency Graph

```
                    ┌───────────────────────────┐   ┌──────────────────────┐
                    │ Plan 1                    │   │ Plan 2               │
                    │ Interfaces + Concretes    │   │ Providers            │
                    │ (cost/sessions/edition/   │   │ (port + Anthropic    │
                    │  config + agent iface)    │   │  native)             │
                    └─────────────┬─────────────┘   └──────────┬───────────┘
                                  │                            │
                                  │                            │
                         ┌────────▼────────┐                   │
                         │ Plan 3          │                   │
                         │ Tools (core +   │                   │
                         │ built-ins +     │                   │
                         │ inversion)      │                   │
                         └─────┬───────┬───┘                   │
                               │       │                       │
                   ┌───────────▼───┐   │                       │
                   │ Plan 4        │   │                       │
                   │ MCP outbound  │   │                       │
                   │ + UIBridge    │   │                       │
                   └───────────────┘   │                       │
                                       │                       │
                                   ┌───▼───────────────────────▼───┐
                                   │ Plan 5                        │
                                   │ Agent loop                    │
                                   └──────────────┬────────────────┘
                                                  │
                                   ┌──────────────▼────────────────┐
                                   │ Plan 6                        │
                                   │ openharness/app registration  │
                                   │ surface                       │
                                   └──────────────┬────────────────┘
                                                  │
                                          ━━━━━━━━▼━━━━━━━━
                                          TAG: v0.2.0-rc1
                                          ━━━━━━━━┳━━━━━━━━
                                                  │
                    ┌─────────────────────────────┼─────────────────────────┐
                    │                             │                         │
           ┌────────▼────────────┐   ┌────────────▼───────────┐              │
           │ Plan 7              │   │ Plan 8                 │              │
           │ Ghostfin import     │   │ Example vertical +     │              │
           │ rewrite             │   │ scaffolder skill       │              │
           └────────┬────────────┘   └────────────┬───────────┘              │
                    │                             │                         │
                    └─────────────────┬───────────┘                          │
                                      │                                     │
                              ━━━━━━━━▼━━━━━━━━                              │
                              TAG: v0.2.0                                    │
                              ━━━━━━━━━━━━━━━━━                              │
```

**Key insight:** Plans 1 & 2 are fully independent (agent interfaces + concretes vs providers). Plans 3 & 4 share only the `tools/` directory but split between `tools/` and `tools/mcp/`, so they're sequential to avoid merge conflicts on `tools/tool.go`. Plans 7 & 8 touch separate repos/modules (ghostfin vs openharness/examples) and can run in parallel after the rc tag.

---

## Execution Order (8 plans, 6 waves + 2 tag cuts)

| Wave | Plan(s) | Session Allocation | Parallelism |
|------|---------|-------------------|-------------|
| **1** | Plan 1 (Interfaces + Concretes) + Plan 2 (Providers) | CLI worktree + CLI worktree | Zero file overlap — `agent/`+`cost/`+`sessions/`+`edition/`+`config/` vs `providers/` |
| **2** | Plan 3 (Tools core + built-ins + inversion) | CLI worktree | Depends on Plan 1 (`agent/interfaces.go`) |
| **3** | Plan 4 (MCP outbound + UIBridge) | CLI worktree | Depends on Plan 3 (`tools/tool.go`) |
| **4** | Plan 5 (Agent loop) | CLI worktree | Depends on Plans 1, 2, 3 |
| **5** | Plan 6 (`openharness/app` registration surface) | CLI worktree | Depends on Plan 5 |
| **Tag** | `v0.2.0-rc1` pre-release cut from `main` | — | — |
| **6** | Plan 7 (Ghostfin import rewrite) + Plan 8 (Example vertical + scaffolder) | CLI (ghostfin repo) + CLI (openharness repo) | Separate repos |
| **Tag** | `v0.2.0` cut from `main` | — | — |

---

## Pre-flight Check

```bash
# Verify Layer 1 is shipped and tests pass
cd openharness && go test ./harness/... ./harness/lite/... -v

# Verify ghostfin main is at a clean, buildable state
cd ghostfin/desktop && go test -tags sqliteonly ./internal/... -v
cd ghostfin/desktop && wails build -tags sqliteonly
```

If any of these fail, fix before starting Layer 2 — the migration assumes a green baseline.

---

## Wave 1: Foundation (Parallel)

### 1A: Plan 1 — Interfaces + Concretes (CLI session, worktree)

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-1-interfaces-and-concretes.md`
- **Branch:** `feat/layer-2-plan-1-interfaces-and-concretes`
- **Scope:** Move `cost`, `sessions`, `edition`, `config` packages from ghostfin into openharness verbatim. Create `openharness/agent/interfaces.go` with `TraceRecorder`, `KnowledgeStore`, `MemoryStore`, `PromptContextProvider` interfaces plus shared types (`LLMCallInfo`, `Block`, `MemoryEntry`, `ContextFile`). Create `openharness/agent/defaults.go` with no-op implementations so openharness compiles and tests standalone.
- **Key files created:**
  - `openharness/cost/tracker.go`, `pricing.go`, `tracker_test.go`, `pricing_test.go`
  - `openharness/sessions/store.go`, `pg_store.go`, `store_test.go`, `pg_store_test.go`
  - `openharness/edition/edition.go`, `edition_test.go`
  - `openharness/config/config.go`, `discovery.go`, `store.go`, `config_test.go`, `discovery_test.go`, `store_test.go`
  - `openharness/agent/interfaces.go`, `defaults.go`, `types.go`, `interfaces_test.go`
- **Skill:** `superpowers:subagent-driven-development`

### 1B: Plan 2 — Providers (CLI session, worktree)

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-2-providers.md` *(written after Plan 1 starts)*
- **Branch:** `feat/layer-2-plan-2-providers`
- **Scope:** Move `internal/providers/*.go` (openai_compat, openrouter, registry, types) from ghostfin verbatim, strip `sqliteonly` build tag. Add a new `providers/anthropic.go` using `github.com/anthropics/anthropic-sdk-go` with real SSE streaming, prompt caching headers, and cache-token accounting.
- **Key files created:**
  - `openharness/providers/types.go`, `openai_compat.go`, `openrouter.go`, `registry.go`
  - `openharness/providers/anthropic.go` (NEW, native SDK)
  - `openharness/providers/*_test.go` (`httptest.NewServer` mocks)
- **Skill:** `superpowers:subagent-driven-development`

**Why parallel is safe:** Plan 1 touches `openharness/{cost,sessions,edition,config,agent}/`. Plan 2 touches `openharness/providers/`. Zero file overlap.

### Wave 1 Verification Gate

- [ ] Merge Plan 1 to main: `git merge --no-ff feat/layer-2-plan-1-interfaces-and-concretes`
- [ ] Run: `cd openharness && go test ./... -v` — all passing
- [ ] Merge Plan 2 to main: `git merge --no-ff feat/layer-2-plan-2-providers`
- [ ] Run: `cd openharness && go test ./... -v` — all passing
- [ ] Tag: `git tag layer-2-wave-1`

---

## Wave 2: Plan 3 — Tools

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-3-tools.md` *(written after Wave 1)*
- **Branch:** `feat/layer-2-plan-3-tools`
- **Scope:** Move `internal/tools/{tool,registry,builtin,composite,ssrf,shell_deny,scrub,external_content}.go` plus the 7 generic built-ins (web_search, web_fetch, filesystem, exec, memory, ask_user, tasks). Add `Effects` field (`ToolEffectRead`/`Mutate`/`Interactive`) to every `ToolDefinition`. Move `knowledge_graph.go` and `subagent.go` with interface-based injection: `knowledge_graph` uses `agent.KnowledgeStore`; `subagent` uses a pluggable `SubagentAllowlist` config. Leave `addepar_api.go`, `tamarac_api.go`, `ghostfin_api.go`, `schedule_cron.go` in ghostfin.
- **Key files created:**
  - `openharness/tools/tool.go` (with `Effects` field), `registry.go`, `builtin.go`, `composite.go`, `ssrf.go`, `shell_deny.go`, `scrub.go`, `external_content.go`
  - `openharness/tools/builtin/{web_search,web_fetch,filesystem,exec,memory,ask_user,tasks,knowledge_graph,subagent}.go`
- **Skill:** `superpowers:subagent-driven-development`
- **Internal parallelism:** The 9 built-in tool files are independent after `tool.go`/`registry.go`/`composite.go` land. Dispatch them as parallel subagents.

### Wave 2 Verification Gate

- [ ] Merge Plan 3 to main
- [ ] Run: `cd openharness && go test ./... -v` — all passing
- [ ] Tag: `git tag layer-2-wave-2`

---

## Wave 3: Plan 4 — MCP

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-4-mcp.md` *(written after Wave 2)*
- **Branch:** `feat/layer-2-plan-4-mcp`
- **Scope:** Move `internal/mcp/{client,manager,bridge_tool,oauth,bm25_index}.go` from ghostfin. Rewire OAuth token storage from filesystem JSON to `SecretStore` (Layer 1 interface) with keys namespaced as `mcp/oauth/<serverName>`. Make OAuth and search mode opt-in via `MCPConfig`. Add `openharness/tools/mcp/app_bridge.go` — the host-side `UIBridge` for interactive tool UIs (mirrors `modelcontextprotocol/ext-apps/src/app-bridge.ts` protocol verbatim).
- **Key files created:**
  - `openharness/tools/mcp/client.go`, `manager.go`, `bridge_tool.go`, `oauth.go`, `bm25_index.go`
  - `openharness/tools/mcp/app_bridge.go` (NEW: UIBridge + Transport interface)
  - `openharness/tools/mcp/*_test.go` (mock MCP server + mock OAuth issuer + mock transport)
- **Skill:** `superpowers:subagent-driven-development`

### Wave 3 Verification Gate

- [ ] Merge Plan 4 to main
- [ ] Run: `cd openharness && go test ./... -v` — all passing
- [ ] Tag: `git tag layer-2-wave-3`

---

## Wave 4: Plan 5 — Agent Loop

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-5-agent-loop.md` *(written after Wave 3)*
- **Branch:** `feat/layer-2-plan-5-agent-loop`
- **Scope:** Move `internal/agent/{loop,tool_loop,systemprompt,compact,retrieval,mode,events,input_guard,pruning,security_log}.go` into `openharness/agent/`. Wire `tool_loop.go` to use `ToolDefinition.Effects` for loop-detection read-streak resets and mode filtering (removes two previously hardcoded allowlists). Wire `retrieval.go` to use `KnowledgeStore` + `MemoryStore` interfaces from Plan 1. Inline `ContextFile` into `systemprompt.go` (drops dependency on `internal/bootstrap.ContextFile`). Keep `internal/agent/agent.go` (preset agents) and finance-specific routing in `harness.go` inside ghostfin.
- **Key files created:**
  - `openharness/agent/loop.go`, `tool_loop.go`, `systemprompt.go`, `compact.go`, `retrieval.go`, `mode.go`, `events.go`, `input_guard.go`, `pruning.go`, `security_log.go`
  - `openharness/agent/config.go` (`AgentConfig` + `LoopDetectionConfig`)
- **Skill:** `superpowers:subagent-driven-development`

### Wave 4 Verification Gate

- [ ] Merge Plan 5 to main
- [ ] Run: `cd openharness && go test ./... -v` — all passing
- [ ] Tag: `git tag layer-2-wave-4`

---

## Wave 5: Plan 6 — Registration Surface

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-6-app-registration-surface.md` *(written after Wave 4)*
- **Branch:** `feat/layer-2-plan-6-app-registration`
- **Scope:** Create `openharness/app/app.go` with `NewApp`, `Tools()`, `Agents()`, `Telemetry()`, `Frontend()`, `Run()`. Layer 2 ships only the registration contract — the full app lifecycle (Wails wiring, gateway startup, SSE) lands in Layer 4. `Run()` errors clearly if no agent is registered ("no agents registered; see docs/getting-started").
- **Key files created:**
  - `openharness/app/app.go`
  - `openharness/app/config.go` (`AppConfig` type — name, data dir, edition, default provider)
  - `openharness/app/app_test.go`
- **Skill:** `superpowers:subagent-driven-development`

### Wave 5 Verification Gate

- [ ] Merge Plan 6 to main
- [ ] Run: `cd openharness && go test ./... -v` — all passing
- [ ] Tag: `git tag layer-2-wave-5`

---

## Tag Cut: `v0.2.0-rc1` pre-release

```bash
cd openharness
git checkout main
git pull
git tag -a v0.2.0-rc1 -m "Layer 2 pre-release: agent primitives + providers + tools + MCP + registration"
git push origin main v0.2.0-rc1
```

This tag exists specifically so Plans 7 and 8 can `require` a real version without `replace` directives or `go.work` `use` lines.

---

## Wave 6: Ghostfin Rewrite + Example Vertical (Parallel)

### 6A: Plan 7 — Ghostfin Import Rewrite (CLI session, ghostfin repo)

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-7-ghostfin-rewrite.md` *(written after Wave 5)*
- **Repo:** `ghostfin/desktop/`
- **Branch:** `feat/layer-2-ghostfin-rewrite`
- **Scope:** Rewrite every `internal/{agent,tools,providers,mcp,cost,sessions,edition,config}` import in ghostfin to the corresponding `openharness/...` path. Delete the old `internal/` packages that have moved. Move the 5 preset agent definitions from `internal/agent/agent.go` to a new `ghostfin/desktop/agents/` package that imports `openharness/agent` and registers them via `app.Agents().Seed(...)` at startup. Move finance-specific subagent routing (`internal/agent/harness.go`) to an allowlist config passed to `openharness/tools/builtin.NewSubagentTool(...)`. `internal/notes` gains a `KnowledgeStore` adapter method satisfying the openharness interface. `internal/bootstrap.ContextFile` becomes a type alias to `openharness/agent.ContextFile`. Ghostfin's `main.go` switches to `openharness.NewApp(...)` for app lifecycle.
- **Skill:** `superpowers:subagent-driven-development`
- **Internal parallelism:** Import rewrites for independent internal/ callers can be dispatched as parallel subagents.

### 6B: Plan 8 — Example Vertical + Scaffolder Skill (CLI session, openharness repo)

- **Plan:** `docs/superpowers/plans/layer-2/2026-04-13-plan-8-example-and-scaffolder.md` *(written after Wave 5)*
- **Repo:** `openharness/`
- **Branch:** `feat/layer-2-example-and-scaffolder`
- **Scope:** Create `openharness/examples/minimal-vertical/` — a real, buildable Go module that `require`s `github.com/swiftdiaries/openharness v0.2.0-rc1` and demonstrates `NewApp(...)`, `Tools().Register(...)`, `Agents().Seed(...)`. Create `openharness/skills/create-openharness-app/SKILL.md` — a Claude scaffolding skill that prompts for vertical name, data dir, default edition, and default provider, then copies from `examples/minimal-vertical/`, rewrites the module path, and emits a README. CI runs scaffold → `go mod tidy` → `go build` → `go test` in the generated project.
- **Key files created:**
  - `openharness/examples/minimal-vertical/{go.mod,main.go,agents/default.go,tools/hello.go,README.md}`
  - `openharness/skills/create-openharness-app/SKILL.md`
  - `openharness/.github/workflows/scaffolder-ci.yml` (or equivalent)
- **Skill:** `superpowers:subagent-driven-development`

**Why parallel is safe:** Plan 7 touches only `ghostfin/desktop/`. Plan 8 touches only `openharness/examples/` and `openharness/skills/`. Different repos.

### Wave 6 Verification Gate

- [ ] Merge Plan 7 to ghostfin main; run `cd ghostfin/desktop && go test -tags sqliteonly ./... -v && wails build -tags sqliteonly`
- [ ] Merge Plan 8 to openharness main; run `cd openharness/examples/minimal-vertical && go test ./...`
- [ ] End-to-end: run the scaffolder skill against a scratch dir, verify `go mod tidy && go build && go test` in the generated project
- [ ] Tag: `git tag layer-2-wave-6`

---

## Tag Cut: `v0.2.0` release

```bash
cd openharness
git checkout main
git pull
git tag -a v0.2.0 -m "Layer 2 release: agent primitives + providers + tools + MCP + registration + scaffolder"
git push origin main v0.2.0
```

Ghostfin's `go.mod` bumps from `v0.2.0-rc1` to `v0.2.0` in a follow-up commit.

---

## High-Risk Files (merge-conflict hotspots)

| File | Touched By Plans | Mitigation |
|------|-----------------|------------|
| `openharness/tools/tool.go` | 3, 4 | Sequential waves (3 before 4) |
| `openharness/tools/registry.go` | 3, 4, 5 | Sequential waves |
| `openharness/agent/interfaces.go` | 1, 5 | Plan 1 creates it with finalized shape; Plan 5 adds no new methods |
| `openharness/app/app.go` | 6 | Single plan; no contention |
| `ghostfin/desktop/main.go` | 7 | Single plan; no contention |
| `ghostfin/desktop/internal/*` | 7 | Import rewrites across many files; Plan 7 internal subagents partition by top-level package |

The wave structure ensures no two plans modify the same file concurrently.

---

## Spec Discrepancies Corrected in Plans

Documented here so executors don't get confused by the spec saying one thing and the plan saying another:

1. **`CostTracker` interface dropped.** Spec Section "agent/interfaces.go — consumer-side inversion seams" included a `CostTracker` interface. Q1 in the spec's key-decisions table committed to moving `cost` concretely, which makes the interface redundant. Plan 1 omits `CostTracker` from `openharness/agent/interfaces.go`; `openharness/agent/loop.go` in Plan 5 imports `openharness/cost` directly.
2. **`TraceRecorder` uses primitive-typed `LLMCallInfo`, not `providers.Usage`.** Plans 1 and 2 run in parallel; Plan 1 cannot depend on Plan 2's types. `LLMCallInfo` is defined in `openharness/agent/types.go` and holds token counts as `int64` + cost as `float64`. Plan 5's agent loop converts `providers.Usage` → `LLMCallInfo` at the call site.
3. **`sessions` package preserves build tags.** `internal/sessions/pg_store.go` is `!sqliteonly`-gated in ghostfin. Plan 1 moves it verbatim with the tag intact. A future Layer 3 refactor can revisit.

---

## Session Launch Commands

### Wave 1 Setup (parallel)

```bash
cd openharness
git worktree add .worktrees/plan-1-interfaces-and-concretes -b feat/layer-2-plan-1-interfaces-and-concretes main
git worktree add .worktrees/plan-2-providers -b feat/layer-2-plan-2-providers main
```

### Wave 2 Setup

```bash
cd openharness
git worktree remove .worktrees/plan-1-interfaces-and-concretes
git worktree remove .worktrees/plan-2-providers
git worktree add .worktrees/plan-3-tools -b feat/layer-2-plan-3-tools main
```

### Wave 3 Setup

```bash
cd openharness
git worktree remove .worktrees/plan-3-tools
git worktree add .worktrees/plan-4-mcp -b feat/layer-2-plan-4-mcp main
```

### Wave 4 Setup

```bash
cd openharness
git worktree remove .worktrees/plan-4-mcp
git worktree add .worktrees/plan-5-agent-loop -b feat/layer-2-plan-5-agent-loop main
```

### Wave 5 Setup

```bash
cd openharness
git worktree remove .worktrees/plan-5-agent-loop
git worktree add .worktrees/plan-6-app-registration -b feat/layer-2-plan-6-app-registration main
```

### Wave 6 Setup (parallel, across two repos)

```bash
cd openharness
git worktree remove .worktrees/plan-6-app-registration
git worktree add .worktrees/plan-8-example-and-scaffolder -b feat/layer-2-example-and-scaffolder main

cd ../ghostfin
git worktree add .worktrees/plan-7-ghostfin-rewrite -b feat/layer-2-ghostfin-rewrite main
```

---

## Session Prompt Template

When starting each session, use this prompt:

```
Execute the plan at docs/superpowers/plans/layer-2/<plan-file>.md
using superpowers:subagent-driven-development.

Branch: feat/layer-2-<branch-name>
Use opus model for all subagents.
Run verification commands after each task.
Commit after each task passes.
```
