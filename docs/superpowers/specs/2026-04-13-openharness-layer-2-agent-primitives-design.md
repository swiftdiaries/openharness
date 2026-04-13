# OpenHarness Layer 2 — Agent Primitives Design

**Date:** 2026-04-13
**Status:** Draft
**Depends on:** Layer 1 (harness boundary + Lite stores, shipped as `v0.1.0`)
**Upstream:** `2026-04-10-openharness-extraction-design.md`

## Context

OpenHarness Layer 1 shipped the harness boundary (eight interfaces) and Lite implementations (in-memory / filesystem / keychain-backed stores + `LiteRunner`) as `v0.1.0`. Layer 2 extracts the agent execution loop, tool system, LLM provider abstraction, and MCP integration from `ghostfin/desktop/internal/` into the `openharness` module. When Layer 2 lands, any vertical can `import github.com/swiftdiaries/openharness`, register domain tools + agents + views at startup, and get a runnable agentic Go application.

The original extraction spec framed Layer 2 as "move four packages" (`agent`, `tools`, `providers`, `mcp`). Exploration of the ghostfin source tree surfaced three complications that reshape the scope:

1. **Circular dependency with Layer 3.** `internal/agent` directly imports packages scheduled for Layer 3 (`cost`, `telemetry`, `traces`, `sessions`, `config`) and Layer 4 (`bootstrap`, `notes`). A clean Layer 2 requires either pulling dependencies forward or inverting them.
2. **Finance-specific content in generic-looking files.** `internal/agent/agent.go` defines 5 fund-analysis preset agents; `internal/agent/harness.go` names them in routing rules; `internal/tools/` contains three finance-API tool files (Addepar, Tamarac, GhostFin pipeline).
3. **The registration API must be committed, not implied.** A framework without a scaffolder is just a collection of packages. A scaffolder forces the public API to be real, documented, and stable.

This spec resolves those by pulling `cost`/`sessions`/`edition`/`config` into Layer 2 concretely, inverting telemetry/traces/notes behind small consumer-side interfaces, adding a native Anthropic provider with real streaming and prompt caching, and shipping a `/create-openharness-app` scaffolding skill as part of the same PR chain.

## Goals

- `openharness/agent`, `openharness/tools`, `openharness/providers`, `openharness/tools/mcp` exist as first-class packages with tests green in pure-Go CI (no Docker, no API keys).
- Ghostfin's desktop app builds and runs against a published openharness pre-release tag with zero behavior change.
- A developer can run `/create-openharness-app`, answer a few prompts, and end up with a Go module that builds, passes tests, and boots a running agent chat loop.
- Native Anthropic support with real SSE streaming and prompt-caching headers populated into the existing `Usage` cache-token fields.
- MCP OAuth tokens stop being persisted as filesystem JSON and start flowing through the `SecretStore` interface from Layer 1.

## Non-goals

- OpenAI-compatible provider streaming (stays stubbed; tracked as fast-follow).
- Telemetry two-layer refactor (SQLite+DuckDB / Postgres+OTLP) — Layer 3.
- Wails frontend scaffold, HTTP gateway, SSE endpoints — Layer 4.
- `notes` package extraction — Layer 4 (Layer 2 interfaces against it via `KnowledgeStore`).
- Scheduler extraction — Layer 3 (the `schedule_cron` tool stays in ghostfin until then).
- Enterprise backends — Layer 5.

## Key decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Hybrid dependency resolution: move `cost`/`sessions`/`edition`/`config` concretely into Layer 2; interface-invert telemetry/traces, knowledge store, and memory store | Telemetry needs its own two-layer refactor; the other Layer 3 concretes are thin and vertical-agnostic, so moving them is free |
| 2 | `knowledge_graph` tool and `retrieval.go` move to openharness behind a `KnowledgeStore` interface; ghostfin's `internal/notes` implements the interface until Layer 4 | Treats the knowledge graph as a framework-level concept (users organize their work with the agent in a markdown repo) while keeping the notes package extraction deferred |
| 3 | Port existing providers verbatim AND add a native Anthropic provider with real streaming + prompt-caching headers + cache-token accounting | Fixes a latent cost-tracking gap (the `Usage` struct has cache-token fields that no existing provider populates) and anchors the framework to its primary LLM vendor |
| 4 | Move all MCP files (`client`, `manager`, `bridge_tool`, `oauth`, `bm25_index`); OAuth and search mode become opt-in via config; OAuth tokens flow through `SecretStore` | Security improvement (keychain/Vault instead of filesystem JSON) and cleaner vertical boundary |
| 5 | Add `UIBridge` for interactive tool UIs (host-side protocol handler) in `tools/mcp/` | New scope from the `modelcontextprotocol/ext-apps` reference; Layer 2 owns the Go-side protocol, Layer 4 owns the iframe renderer |
| 6 | Tool self-classification via an `Effects` field on `ToolDefinition` (`Read`, `Mutate`, `Interactive`); loop detection and approval-mode filter both read the same field; thresholds exposed via `LoopDetectionConfig` | Unifies two previously hardcoded allowlists; makes loop detection work correctly on any vertical whose tool names differ from ghostfin's |
| 7 | Ship `/create-openharness-app` scaffolding skill with reference templates under `examples/minimal-vertical/` | Forcing function on the public registration API (`NewApp`, `Tools`, `Agents`, `Telemetry`, `Frontend`, `Run`) |
| 8 | `openharness/agent` ships zero preset agent definitions; the scaffolder generates a starter agent in the vertical's own tree | Keeps `openharness/agent` a pure primitive package; prevents a "preset zoo" from forming in core |
| 9 | All tests mock HTTP / mock transport; no testcontainers in Layer 2; cut `v0.2.0-rc1` pre-release before the scaffolder PR so the scaffolder can `require` a real tag | Fast CI (runs in seconds, no Docker, no API keys); matches the documented dependency-style preference (real tags, no replace directives, no `go.work` `use` lines) |

## Target package layout

```
openharness/
├── go.mod                      # New deps:
│                               #   github.com/anthropics/anthropic-sdk-go
│                               #   github.com/mark3labs/mcp-go
│                               #   go.opentelemetry.io/otel
│
├── harness/                    # Layer 1, unchanged
│   └── lite/
│
├── agent/                      # NEW
│   ├── loop.go                 # Think→Act→Observe; uses TraceRecorder + CostTracker interfaces
│   ├── tool_loop.go            # 3-layer loop detection using ToolEffect classifier
│   ├── systemprompt.go         # Uses inlined ContextFile
│   ├── compact.go              # History compaction
│   ├── retrieval.go            # Uses KnowledgeStore + MemoryStore interfaces
│   ├── mode.go                 # Plan/Auto/Approval; uses ToolEffect for filtering
│   ├── events.go
│   ├── input_guard.go
│   ├── pruning.go
│   ├── security_log.go
│   ├── interfaces.go           # TraceRecorder, CostTracker, KnowledgeStore, MemoryStore, PromptContextProvider
│   ├── config.go               # AgentConfig, LoopDetectionConfig
│   └── *_test.go
│
├── tools/                      # NEW
│   ├── tool.go                 # Tool interface + ToolDefinition (with Effects field)
│   ├── registry.go
│   ├── builtin.go
│   ├── composite.go
│   ├── ssrf.go
│   ├── shell_deny.go
│   ├── scrub.go
│   ├── external_content.go
│   ├── builtin/
│   │   ├── web_search.go
│   │   ├── web_fetch.go
│   │   ├── filesystem.go
│   │   ├── exec.go
│   │   ├── memory.go
│   │   ├── ask_user.go
│   │   ├── tasks.go
│   │   ├── knowledge_graph.go  # Uses KnowledgeStore interface
│   │   └── subagent.go         # Uses pluggable allowlist config
│   └── mcp/
│       ├── client.go           # mark3labs/mcp-go wrapper
│       ├── manager.go          # Lifecycle + reconnect + health; takes SecretStore dep
│       ├── bridge_tool.go      # Outbound MCP tool → tools.Tool adapter
│       ├── oauth.go            # Tokens via SecretStore
│       ├── bm25_index.go       # Search mode (opt-in)
│       ├── app_bridge.go       # NEW: UIBridge (host side of ui/* protocol)
│       └── *_test.go
│
├── providers/                  # NEW
│   ├── types.go                # Provider, ChatRequest, ChatResponse, Usage (cache tokens)
│   ├── registry.go
│   ├── openai_compat.go        # Ported verbatim; ChatStream still fallback
│   ├── openrouter.go           # Thin wrapper
│   ├── anthropic.go            # NEW: native SDK, real streaming + prompt caching
│   └── *_test.go
│
├── cost/                       # NEW (moved from ghostfin/desktop/internal/cost)
├── sessions/                   # NEW (moved from ghostfin/desktop/internal/sessions)
├── edition/                    # NEW (moved from ghostfin/desktop/internal/edition)
├── config/                     # NEW (moved from ghostfin/desktop/internal/config)
│
├── app/                        # NEW: minimal registration API surface
│   └── app.go                  # NewApp, Tools(), Agents(), Telemetry(), Frontend(), Run()
│                               # Full app lifecycle lands in Layer 4
│
├── examples/
│   └── minimal-vertical/       # NEW: reference template for scaffolder
│       ├── go.mod
│       ├── main.go
│       ├── agents/default.go
│       ├── tools/hello.go
│       └── README.md
│
└── skills/
    └── create-openharness-app/
        └── SKILL.md            # NEW: scaffolder skill
```

## Component designs

### agent/interfaces.go — consumer-side inversion seams

Layer 2 depends on three concepts that aren't yet extracted into openharness: telemetry/traces, knowledge store, and memory store. Rather than wait for those extractions, Layer 2 defines narrow consumer-side interfaces with only the methods the agent loop actually calls. Ghostfin provides concrete implementations via its existing packages.

```go
package agent

// TraceRecorder is the minimal surface the agent loop uses to record
// LLM call spans, tool-call spans, and cost attribution. The Layer 3
// telemetry refactor will provide a concrete implementation backed by
// SQLite+DuckDB (Lite) or Postgres+OTLP (Enterprise).
type TraceRecorder interface {
    StartRun(ctx context.Context, runID, agentID, sessionID string) context.Context
    RecordLLMCall(ctx context.Context, model string, usage Usage, durationMS int64)
    RecordToolCall(ctx context.Context, tool string, args, result []byte, durationMS int64, err error)
    EndRun(ctx context.Context, status string, err error)
}

// CostTracker enforces per-run and per-window budget limits.
type CostTracker interface {
    CheckBudget(ctx context.Context, runID string) error
    Record(ctx context.Context, runID, model string, usage Usage) error
}

// KnowledgeStore is the framework-level interface for a user-owned
// markdown/block knowledge repo. Used by the knowledge_graph tool and
// agent/retrieval.go. Ghostfin's internal/notes implements this.
type KnowledgeStore interface {
    Search(ctx context.Context, query string, limit int) ([]Block, error)
    Get(ctx context.Context, id string) (Block, error)
}

// MemoryStore is the framework-level interface for the key-value
// memory tool. Used by agent/retrieval.go for pre-query memory injection.
type MemoryStore interface {
    Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error)
}

// PromptContextProvider supplies dynamic context files (AGENTS.md,
// persona files) to the system prompt builder.
type PromptContextProvider interface {
    Files(ctx context.Context) ([]ContextFile, error)
}

// ContextFile is inlined from what was internal/bootstrap.ContextFile in
// ghostfin. Intentionally small and stable so Layer 4 bootstrap can
// alias back to this type without a breaking change.
type ContextFile struct {
    Name    string
    Role    string
    Content string
}
```

All five interfaces have no-op default implementations in a `agent/defaults.go` file, so openharness compiles and tests standalone without any ghostfin dependency.

### providers/anthropic.go — native SDK provider

The existing ghostfin provider surface is a single polymorphic `openai_compat.go` that speaks OpenAI chat-completions over stdlib `net/http`. It powers OpenAI, OpenRouter, Azure, and Databricks via a `ParseModelString` prefix-stripper. `ChatStream` is stubbed everywhere — it falls back to `Chat` and emits a single chunk at the end.

Layer 2 keeps all of that as-is (no behavior change for ghostfin) and adds a new `providers/anthropic.go` using the official `github.com/anthropics/anthropic-sdk-go`:

- Implements the existing `Provider` interface (`Chat`, `ChatStream`, `DefaultModel`, `Name`).
- **Real streaming**: wraps the SDK's streaming API, translates `content_block_delta` events into `StreamChunk` calls on the caller's callback, reassembles tool-call deltas server-side.
- **Prompt caching**: honors `cache_control` markers on system prompts and long context messages. Populates `Usage.CacheCreationTokens` and `Usage.CacheReadTokens` from the response — fixing a latent gap where ghostfin's `Usage` struct has those fields but nothing fills them.
- **Default model constant**: `claude-opus-4-6`. Available alternates: `claude-sonnet-4-6`, `claude-haiku-4-5-20251001`.

Testing uses `httptest.NewServer` returning canned SSE streams. A helper emits `event: message_start`, `event: content_block_delta`, `event: message_delta`, `event: message_stop` sequences. Table-driven tests cover streaming chunks, tool-call reassembly, cache-token accounting, and error handling. No API key is required in CI. Wire-format drift is handled by a separate weekly smoke job (not a Layer 2 deliverable).

### tools/mcp/app_bridge.go — interactive tool UI protocol

The original extraction design contemplated only the *outbound* MCP client (connecting to Linear, Notion, etc. as tool providers). Layer 2 adds the host-side of a second, complementary protocol: interactive tool UIs, modeled on `modelcontextprotocol/ext-apps/src/app-bridge.ts`.

The concept: a tool call can return a UI resource (HTML bundle). The host renders it in a sandboxed webview, and the view talks back to the host over a bidirectional protocol — `ui/initialize`, `ui/message`, `ui/open-link`, `ui/download-file`, `ui/update-model-context`, `ui/request-display-mode`, `ui/notifications/size-changed`, `ui/resource-teardown`, plus tool-flow notifications (`ui/notifications/tool-input-partial`, `tool-input`, `tool-result`, `tool-cancelled`).

Layer 2 ships the Go-side host protocol handler with no frontend coupling:

```go
package mcp

// UIBridge is the host-side bidirectional router for interactive tool UIs.
// It forwards tools/call, tools/list, and resources/read requests from a
// sandboxed view back through the MCP client, and manages ui/* notification
// channels out to the view.
type UIBridge struct { /* ... */ }

// Transport abstracts the postMessage-equivalent between host and view.
// Layer 4 implements this against the Wails webview; tests use a mock.
type Transport interface {
    Send(msg []byte) error
    Recv() ([]byte, error)
    Close() error
}

func NewUIBridge(client *Client, host HostCapabilities) *UIBridge
func (b *UIBridge) Connect(t Transport) error
func (b *UIBridge) OnCallTool(fn func(ctx context.Context, req CallToolRequest) (CallToolResult, error))
func (b *UIBridge) OnOpenLink(fn func(ctx context.Context, url string) error)
// ... handler registration for every ui/* method
func (b *UIBridge) SendToolInputPartial(ctx context.Context, tool string, partial []byte) error
func (b *UIBridge) SendToolResult(ctx context.Context, tool string, result []byte) error
func (b *UIBridge) TeardownResource(ctx context.Context) error

// Helpers translated verbatim from the TS reference.
func GetToolUIResourceURI(meta map[string]any) string
func IsToolVisibilityModelOnly(meta map[string]any) bool
func IsToolVisibilityAppOnly(meta map[string]any) bool
func BuildAllowAttribute(perms map[string]bool) string
```

Wire format matches `app-bridge.ts` verbatim so tool UIs built against the TypeScript bridge run unchanged against the Go bridge. Tests use a mock `Transport` and drive table-driven cases for each `ui/*` message type plus a full lifecycle test (initialize → operate → teardown). No browser needed; entirely pure-Go.

### tools.ToolDefinition — the `Effects` field

```go
type ToolEffect int

const (
    ToolEffectRead ToolEffect = iota
    ToolEffectMutate
    ToolEffectInteractive  // ask_user, human-in-the-loop tools
)

type ToolDefinition struct {
    Name        string
    Description string
    Parameters  json.RawMessage
    Effects     ToolEffect       // NEW
    Visibility  ToolVisibility   // reserved for UIBridge model-only/app-only filter
}
```

Every built-in tool declares its effect at definition time. `agent/tool_loop.go` uses `tool.Definition().Effects == ToolEffectMutate` instead of the previous hardcoded `["write_file", "edit_file", "agent", "memory_store"]` allowlist for loop-detection read-streak resets. `agent/mode.go` uses the same field to filter which tools are allowed in `ModePlan` (read-only) vs `ModeAuto`. Both systems previously had independent hardcoded lists; unifying them is part of this migration.

`LoopDetectionConfig` exposes the thresholds:

```go
type LoopDetectionConfig struct {
    MaxRepeatedCalls     int  // default 5
    MaxSameResult        int  // default 6
    MaxReadStreakStuck   int  // default 12
    MaxReadStreakExplore int  // default 36
}
```

Zero-value struct means "use defaults matching today's ghostfin behavior." Verticals override per-agent via `AgentConfig.LoopDetection`.

### MCP OAuth via SecretStore

Ghostfin's `internal/mcp/oauth.go` persists OAuth tokens as JSON files at `{tokenDir}/{sanitized-name}.json`. This is both a security issue (tokens on disk in plaintext) and a layering issue (OAuth token storage should use the same interface as every other secret in the framework).

Layer 2 routes MCP OAuth tokens through the Layer 1 `SecretStore` interface:

- `MCPManager` takes a `SecretStore` dependency at construction time.
- Token reads/writes use namespaced keys: `mcp/oauth/<serverName>`.
- Lite deployments get OS keychain storage via the existing `LiteSecretStore` (which wraps macOS Keychain / Windows Credential Manager / Linux Secret Service).
- Enterprise deployments will get Vault via Layer 5's `VaultSecretStore`.
- The filesystem JSON path is removed entirely — not deprecated, not a fallback.

This is a security improvement, not a regression. Ghostfin users upgrading to the Layer 2 release will need to re-authenticate their MCP servers on first launch; this is documented in the migration notes.

### Scaffolding skill (`/create-openharness-app`)

Reference: `modelcontextprotocol/ext-apps/plugins/mcp-apps/skills/create-mcp-app/SKILL.md`. The shape is a guided scaffold-and-adapt flow.

Skill workflow:

1. Prompt for vertical name, data dir, default edition (Lite / Enterprise), default LLM provider (Anthropic / OpenAI-compat).
2. Initialize a Go module with `go.mod` that `require`s the published openharness tag (no `replace` directives, no `go.work` `use` lines — matches documented dependency-style preference).
3. Generate `main.go` that calls `openharness.NewApp(...)` and demonstrates the registration API (`Tools().Register`, `Agents().Seed`, `Telemetry().AddAttributes`, `Frontend().RegisterView`).
4. Drop in one stub tool, one stub agent definition (the starter agent from Q7), and one stub frontend view.
5. Configure `wails.json` (openharness ships the Wails scaffold as part of Layer 4; Layer 2's scaffolder emits the config knowing Layer 4 will wire it).
6. Write a minimal `README.md` with the three-command run path: `go mod tidy`, `go test ./...`, `wails dev` (or `go run .` for Layer 2's pre-Wails smoke).
7. Recommend a `go test ./...` run to verify the scaffold compiles.

Reference templates live under `openharness/examples/minimal-vertical/` — a real, tested Go module that the skill copies from rather than generating from inline strings. This keeps the templates exercised by CI.

The skill is a forcing function: it pins down `openharness/app`'s public registration API (`NewApp`, `Tools`, `Agents`, `Telemetry`, `Frontend`, `Run`) as a committed contract. Layer 2 is not complete until a fresh scaffold produces a runnable Go module against the published pre-release tag.

## What stays in ghostfin

After Layer 2 lands, `ghostfin/desktop/internal/` shrinks substantially. What remains:

- **Finance-specific tools**: `addepar_api.go`, `tamarac_api.go`, `ghostfin_api.go` (and their tests).
- **Finance-specific agents**: the 5 preset agent definitions (Analyst, Scribe, ExecutiveAssistant, FundDocAnalyst, PortfolioOps). These move from `internal/agent/agent.go` to a new `ghostfin/desktop/agents/` package that imports `openharness/agent` and registers the presets via `app.Agents().Seed(...)` at startup.
- **Finance-specific subagent routing**: what was `DefaultHarnessConfig` becomes a ghostfin-owned allowlist passed to the openharness subagent tool at registration time.
- **`schedule_cron` tool**: stays in ghostfin until Layer 3 moves the scheduler package.
- **`internal/notes`**: stays in ghostfin until Layer 4. Gains a `KnowledgeStore` adapter method so it satisfies the openharness interface.
- **`internal/bootstrap`**: keeps working by re-exporting `ContextFile` as an alias to the openharness type.
- **Wails frontend + gateway**: all of Layer 4 remains in ghostfin until that extraction.

## Migration sequencing — PR chain

Each PR is reviewable independently. Tests pass at every step.

1. **PR-L2-0 — Interfaces + thin concretes.** Create `openharness/agent/interfaces.go` with no-op defaults; move `cost`, `sessions`, `edition`, `config` packages from ghostfin. No callers updated yet.
2. **PR-L2-1 — Providers (port).** Move `openai_compat.go`, `openrouter.go`, `registry.go`, `types.go`. Strip `sqliteonly` build tag. `httptest` unit tests.
3. **PR-L2-2 — Providers (Anthropic native).** Add `anthropic.go` with streaming + caching. Table-driven SSE tests. This is a separable commit within the PR chain so it can be reviewed independently.
4. **PR-L2-3 — Tools core + generic built-ins.** Move `tool.go`, `registry.go`, ssrf/shell_deny/scrub, the seven generic built-ins. Add `Effects` field to every `ToolDefinition`.
5. **PR-L2-4 — Tools (knowledge_graph + retrieval + subagent with inversion).** Add `KnowledgeStore` and `MemoryStore` interfaces. Rewire `retrieval.go`. Add the pluggable subagent allowlist.
6. **PR-L2-5 — MCP outbound bridge.** Move all five MCP files. Rewire OAuth to `SecretStore`. Opt-in search mode + OAuth via config.
7. **PR-L2-6 — MCP UIBridge.** Add `app_bridge.go` and protocol tests with mock transport.
8. **PR-L2-7 — Agent loop.** Move `loop.go`, `tool_loop.go`, `systemprompt.go`, `compact.go`, `mode.go`, `events.go`, `input_guard.go`, `pruning.go`, `security_log.go`. Inline `ContextFile`. Wire loop detection and mode filter to the `Effects` classifier.
9. **PR-L2-8 — `openharness/app` registration surface.** `NewApp`, `Tools()`, `Agents()`, `Telemetry()`, `Frontend()`, `Run()`. Full lifecycle implementation lands in Layer 4; Layer 2 ships just the registration contract.
10. **Cut `v0.2.0-rc1` pre-release tag.**
11. **PR-L2-9 — Ghostfin import rewrite.** All ghostfin callers migrate to openharness imports. Preset agents move to `ghostfin/desktop/agents/`. Ghostfin tests green against `v0.2.0-rc1`.
12. **PR-L2-10 — `examples/minimal-vertical/`.** Real reference template that builds against the pre-release tag.
13. **PR-L2-11 — Scaffolder skill.** `skills/create-openharness-app/SKILL.md`. CI runs scaffold → `go mod tidy` → `go build` → `go test`.
14. **Cut `v0.2.0` tag.**

## Verification

**Per-PR:**
- `cd openharness && go test ./...` is green.
- After PR-L2-9: `cd ghostfin/desktop && go test ./internal/... -tags sqliteonly` is green.
- After PR-L2-10: `cd openharness/examples/minimal-vertical && go test ./...` is green.

**End-to-end (after cutting `v0.2.0-rc1`):**

1. **Scaffolder end-to-end.** Fresh clone a scratch directory. Run `/create-openharness-app` with a test vertical name. Verify `go mod tidy && go build ./... && go test ./...` all succeed in the generated project.
2. **Ghostfin behavior parity.** `cd ghostfin/desktop && wails dev -tags sqliteonly`. Verify agent chat works, tool execution works, MCP servers connect, traces render. Compare to pre-migration commit for regressions.
3. **Anthropic provider smoke (manual, not CI).** Set `ANTHROPIC_API_KEY`. Run a one-shot prompt through the generated minimal vertical. Verify streaming chunks arrive at the UI. Verify `Usage.CacheReadTokens` populates on the second identical prompt (cache hit).
4. **MCP OAuth migration.** Connect to a real Linear MCP server via OAuth through ghostfin. Verify the token lands in OS keychain (not `~/.ghostfin/mcp_tokens/linear.json`). Kill and restart the app; verify the token is re-read from keychain without re-auth.
5. **MCP UIBridge protocol.** Drive a mock tool returning a UI resource through the `UIBridge` test harness. Verify the full `ui/initialize` → `tools/call` → `ui/notifications/tool-result` → `ui/resource-teardown` lifecycle round-trips correctly. (Real webview rendering is Layer 4; Layer 2 only tests the Go-side protocol.)
6. **Loop detection regression.** Run ghostfin's existing `loop_test.go` and `tool_loop_test.go` suites after the `Effects`-field migration. Verify loop detection triggers at the same thresholds against the same fixtures.

## Out of scope (explicit)

- OpenAI-compat streaming implementation — stays stubbed; fast-follow ticket when the first vertical needs OpenRouter/OpenAI as a primary path.
- Telemetry two-layer refactor (SQLite+DuckDB / Postgres+OTLP) — Layer 3.
- Wails frontend scaffold (chat view, knowledge graph view) — Layer 4.
- HTTP gateway + SSE — Layer 4.
- `notes` package extraction — Layer 4.
- Enterprise backends (Postgres, Vault, S3, K8s, Lambda) — Layer 5.
- Scheduler extraction — Layer 3 (`schedule_cron` tool stays in ghostfin).
- Preset agent libraries — explicitly deferred to an optional `openharness-presets` module if ever wanted.
