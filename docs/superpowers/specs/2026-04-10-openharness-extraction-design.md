# OpenHarness Extraction — Design Spec

**Date:** 2026-04-10
**Status:** Draft
**Depends on:** ghostfin harness interfaces (already implemented in `desktop/pkg/harness/`)

## Context

GhostFin's agent harness and desktop framework are domain-agnostic — the agent loop, tool system, provider abstraction, cost tracking, tracing, notes/knowledge graph, and Wails app scaffold contain no finance-specific logic. This design extracts these components into `openharness`, a standalone open-source Go framework for building agentic desktop and cloud applications across any vertical (finance: GhostFin, legal: Harvey, etc.).

### Problem

Today, building a new vertical means forking the entire ghostfin desktop codebase and removing finance-specific code. The generic infrastructure (23 internal packages) is tightly coupled to a single vertical.

### Goal

`openharness` is a Go module that any vertical imports to get:
- A complete Wails-based desktop app scaffold with agent chat + knowledge graph
- A harness boundary layer with Lite (local/embedded) and generic Enterprise (Postgres/S3/Vault/K8s) implementations
- An agent execution loop, tool system, LLM provider abstraction
- Cost tracking, dual-mode tracing, session management
- A registration-based API for vertical customization

### Non-Goals

- Python pipeline framework (verticals own their pipelines, connected via HTTP tools)
- Vertical-specific enterprise features (Nile multi-tenancy, billing, RBAC policies — these stay in vertical-specific enterprise repos)
- Breaking changes to existing ghostfin functionality during migration

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module structure | Standalone Go module | Clean dependency graph, independent versioning, `go get` consumption |
| Scope | Full framework (harness + agent + app scaffold) | Verticals should get a complete, runnable app by importing openharness and registering domain tools |
| Enterprise code split | Generic enterprise in openharness, vertical-specific in ghostfin-enterprise | Postgres/S3/Vault/K8s are infrastructure, not competitive advantage |
| Lite implementations | Included in openharness | Critical for driving Wails desktop apps — the open-source local experience |
| Migration approach | Interface-first, incremental (5 layers) | Each layer is a reviewable PR with tests passing |
| Trace architecture | Two-layer: relational metadata + columnar span data | SQLite/Postgres for metadata index, DuckDB+Parquet / OTLP for span analytics |
| Lambda runner | S3-triggered chain, snapshot-based execution | Pure functions, crash-consistent, no timeout concerns |
| Vertical customization | Registration APIs at app startup | Verticals register tools, agents, views, trace attributes — no framework forking |

---

## System Architecture

### Three-Repo Model

```
                    openharness (open source)
                   ┌─────────────────────────┐
                   │  harness/               │
                   │  agent/                 │
                   │  tools/                 │
                   │  providers/             │
                   │  cost/ telemetry/       │
                   │  notes/ gateway/        │
                   │  app/ config/ edition/  │
                   │  scheduler/ bootstrap/  │
                   │  harness/lite/          │
                   │  harness/enterprise/    │
                   └──────────┬──────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
     ghostfin (open)    harvey (open)    ghostfin-enterprise
    ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐
    │ desktop/     │  │ desktop/     │  │ Nile multi-tenant │
    │  - fund tools│  │  - legal     │  │ Billing / RBAC    │
    │  - fund UI   │  │    tools     │  │ Mgmt server       │
    │ pipeline/    │  │  - case UI   │  │ Vertical-specific  │
    │  (Python)    │  │ pipeline/    │  │  enterprise feat.  │
    └──────────────┘  └──────────────┘  └──────────────────┘
```

### Import Rules

1. `openharness` imports nothing from any vertical
2. Verticals (`ghostfin/desktop`) import `openharness` for all framework code, add domain tools/agents/UI
3. Vertical enterprise repos (`ghostfin-enterprise`) import both `openharness` and the vertical for vertical-specific enterprise features
4. New verticals import only `openharness`

### Go Module Dependencies

```go
// openharness/go.mod
module github.com/swiftdiaries/openharness
// No vertical imports.
// Dependencies: wails, pgx, otel, modernc sqlite, marcboeker go-duckdb,
//               hashicorp vault/api, aws-sdk-go-v2, k8s client-go

// ghostfin/desktop/go.mod
module github.com/swiftdiaries/ghostfin/desktop
require github.com/swiftdiaries/openharness v0.1.0

// ghostfin-enterprise/go.mod
module github.com/swiftdiaries/ghostfin-enterprise
require (
    github.com/swiftdiaries/openharness v0.1.0
    github.com/swiftdiaries/ghostfin/desktop v0.1.0
)
```

Local development via `go.work`:

```go
// ghostfin/go.work
go 1.25.0
use (
    ./desktop
    ../openharness
)
```

---

## Package Layout

```
openharness/
├── go.mod
├── LICENSE
├── README.md
│
├── harness/                        # Layer 1: Boundary interfaces + implementations
│   ├── interfaces.go               # 8 interfaces (HarnessRunner, SessionStore, etc.)
│   ├── types.go                    # RunConfig, Event, AgentDefinition, Snapshot, etc.
│   ├── errors.go                   # Sentinel errors
│   ├── runner_registry.go          # RunnerRegistry: maps names → HarnessRunner impls
│   ├── stream_store.go             # StoreBackedEventStream: polls SessionStore
│   ├── lite/                       # Lite implementations (for Wails desktop apps)
│   │   ├── runner.go               # LiteRunner (goroutine-based, LoopFactory pattern)
│   │   ├── agent.go                # LiteAgentStore (in-memory with seed data)
│   │   ├── session.go              # LiteSessionStore (file-based JSONL)
│   │   ├── secret.go               # LiteSecretStore (env vars / keychain)
│   │   ├── artifact.go             # LiteArtifactStore (filesystem)
│   │   ├── registry.go             # LiteToolRegistry (in-memory)
│   │   ├── skill.go                # LiteSkillStore (filesystem-based)
│   │   └── channel.go              # LiteChannelRouter (outbound notification-only)
│   └── enterprise/                 # Generic enterprise implementations
│       ├── session_pg.go           # Postgres SessionStore (append-only event log)
│       ├── agent_pg.go             # Postgres AgentStore (versioned, optimistic concurrency)
│       ├── skill_pg.go             # Postgres + S3 SkillStore
│       ├── registry_pg.go          # Postgres ToolRegistry (global + tenant overlay)
│       ├── secret_vault.go         # Vault AppRole SecretStore
│       ├── artifact_s3.go          # S3/MinIO ArtifactStore
│       ├── runner_k8s.go           # K8s Job HarnessRunner
│       ├── runner_lambda.go        # Lambda HarnessRunner (S3-triggered snapshot chain)
│       ├── snapshot.go             # Snapshot type + S3 read/write
│       └── testutil/               # Testcontainers helpers (Postgres, Vault, MinIO)
│           ├── pg.go
│           ├── vault.go
│           └── minio.go
│
├── agent/                          # Layer 2: Agent primitives
│   ├── loop.go                     # Think→Act→Observe loop (budget, max iterations)
│   ├── loop_test.go
│   └── detection.go                # 3-layer loop detection
│
├── tools/                          # Layer 2: Tool system
│   ├── tool.go                     # Tool interface + ToolDefinition
│   ├── registry.go                 # In-memory tool registry
│   ├── builtin/                    # Built-in tools
│   │   ├── web_search.go
│   │   ├── filesystem.go
│   │   ├── exec.go
│   │   └── memory.go
│   └── mcp/                        # MCP server bridge
│       └── bridge.go
│
├── providers/                      # Layer 2: LLM provider abstraction
│   ├── types.go                    # Provider interface, Message, Response
│   ├── anthropic.go
│   ├── openai.go
│   └── openrouter.go
│
├── cost/                           # Layer 3: Cost tracking
│   ├── tracker.go                  # Per-model token billing + budget limits
│   └── models.go                   # Model pricing data
│
├── telemetry/                      # Layer 3: Dual-mode tracing
│   ├── tracer.go                   # OTel span creation
│   ├── metadata.go                 # TraceMetadataStore interface (relational index)
│   ├── spans.go                    # SpanSink interface (span data destination)
│   ├── composite.go                # Wires metadata store + span sink
│   ├── lite/
│   │   ├── sqlite_metadata.go      # SQLite TraceMetadataStore
│   │   ├── duckdb_sink.go          # DuckDB SpanSink
│   │   └── parquet.go              # Parquet file management
│   └── enterprise/
│       ├── pg_metadata.go          # Postgres TraceMetadataStore
│       └── otlp_sink.go            # OTLP SpanSink → Honeycomb/Datadog
│
├── sessions/                       # Layer 3: Session management
│   └── store.go                    # File-based session history
│
├── config/                         # Layer 3: Configuration
│   ├── config.go                   # JSON config loading
│   └── workspace.go                # Workspace settings
│
├── edition/                        # Layer 3: Edition gating
│   └── edition.go                  # Lite vs Enterprise feature flags
│
├── scheduler/                      # Layer 3: Cron scheduling
│   └── scheduler.go
│
├── notes/                          # Layer 4: Knowledge management
│   ├── store.go                    # Block-based outliner + FTS
│   ├── graph.go                    # Knowledge graph
│   └── sqlite.go                   # SQLite implementation
│
├── gateway/                        # Layer 4: HTTP gateway + SSE
│   ├── server.go
│   ├── sse.go
│   └── routes.go
│
├── bootstrap/                      # Layer 4: Agent persona system
│   └── context.go                  # AGENTS.md seeding, persona files
│
└── app/                            # Layer 4: Wails app scaffold
    ├── app.go                      # Lifecycle (init stores, start gateway, registration APIs)
    ├── main.go                     # Wails entry template
    └── frontend/                   # React UI (Claw chat + Journal knowledge graph)
        ├── src/
        │   ├── components/         # Shared UI components (chat, editor, graph)
        │   ├── views/
        │   │   ├── claw/           # Agent chat view
        │   │   └── journal/        # Knowledge graph / notes view
        │   └── hooks/              # Wails bindings, SSE hooks
        └── package.json
```

---

## Trace Instrumentation — Two-Layer Architecture

Traces have two distinct storage concerns: **metadata** (relational, for indexing and filtering) and **span data** (columnar/streaming, for analytics and export).

### Layer 1: Trace Metadata (Relational)

**Purpose:** Powers trace list UI, filtering by agent/session/time, cost summaries.
**Lite:** SQLite. **Enterprise:** Postgres.

```go
// TraceMetadataStore is the relational index for trace metadata.
type TraceMetadataStore interface {
    RecordTrace(ctx context.Context, meta TraceMeta) error
    GetTrace(ctx context.Context, traceID string) (TraceMeta, error)
    ListTraces(ctx context.Context, opts TraceListOpts) ([]TraceMeta, error)
    UpdateCost(ctx context.Context, traceID string, cost CostSummary) error
}

type TraceMeta struct {
    TraceID   string
    SessionID string
    AgentID   string
    RunID     string
    Status    string        // running, completed, error
    StartedAt time.Time
    EndedAt   *time.Time
    SpanCount int
    Cost      CostSummary
    Metadata  map[string]string  // domain-specific attributes
}
```

### Layer 2: Span Data (Columnar / Streaming)

**Purpose:** Actual span events for analytics, eval, and external observability.
**Lite:** DuckDB + Parquet files (local analytical power). **Enterprise:** OTLP exporter → Honeycomb/Datadog.

```go
// SpanSink receives raw span data for storage or export.
type SpanSink interface {
    WriteSpans(ctx context.Context, spans []Span) error
    Flush(ctx context.Context) error
    Shutdown(ctx context.Context) error
}
```

### Data Flow

```
Agent Loop → OTel SDK → openharness tracer
                            │
                ┌───────────┴───────────┐
                ▼                       ▼
        TraceMetadataStore          SpanSink
        (relational index)      (span data)
                │                       │
        ┌───────┴───────┐       ┌───────┴───────┐
        │               │       │               │
    SQLite (Lite)   Postgres   DuckDB+Parquet  OTLP exporter
                   (Enterprise)   (Lite)     (Enterprise)
```

**Lite query path:** UI trace list → SQLite metadata. Span drill-down → DuckDB query on Parquet.
**Enterprise query path:** UI trace list → Postgres metadata. Span analytics → Honeycomb/Datadog.

---

## Vertical Customization

Verticals customize openharness via registration APIs at app startup — no framework forking.

```go
// ghostfin/desktop/main.go
func main() {
    app := openharness.NewApp(openharness.AppConfig{
        Name:    "GhostFin",
        DataDir: "~/.ghostfin/data",
        Edition: edition.Lite,
    })

    // Domain-specific tools (call Python pipeline via HTTP)
    app.Tools().Register("extract_holdings", ghostfin.NewHoldingsTool(pipelineURL))
    app.Tools().Register("classify_pages", ghostfin.NewClassifyTool(pipelineURL))
    app.Tools().Register("validate_nav", ghostfin.NewValidationTool())

    // Domain-specific agents
    app.Agents().Seed(ghostfin.FundAnalystAgent())
    app.Agents().Seed(ghostfin.ScribeAgent())

    // Domain trace attributes
    app.Telemetry().AddAttributes(ghostfin.DomainAttributes())

    // Domain frontend views
    app.Frontend().RegisterView("pipeline", ghostfin.PipelineView())

    app.Run()
}
```

### What Verticals Provide

1. **Domain tools** — HTTP tool implementations calling their pipelines
2. **Agent definitions** — preset agents with domain prompts/personas
3. **Frontend views** — React components for domain-specific UI
4. **Trace attributes** — domain-specific span attributes
5. **Validation logic** — domain-specific data checks

### What OpenHarness Provides (Verticals Don't Touch)

1. Agent loop, budget enforcement, loop detection
2. Tool execution framework + MCP bridge
3. LLM provider switching (Anthropic, OpenAI, OpenRouter)
4. Cost tracking + dual-mode tracing
5. Notes/knowledge graph (block-based outliner + FTS)
6. Session management
7. Harness boundary layer (Lite + Enterprise)
8. Wails app lifecycle, HTTP gateway, SSE streaming
9. Cron scheduling, agent personas, config management

---

## Migration Strategy — 5 Layers

Incremental extraction from `ghostfin/desktop/` → `openharness/`. Each layer is an independent PR chain: extract code, update imports in ghostfin, verify tests pass.

### Layer 1: Harness Boundary (lowest risk)

**Source:** `ghostfin/desktop/pkg/harness/`
**Destination:** `openharness/harness/`, `openharness/harness/lite/`

| Source File | Destination |
|------------|-------------|
| `pkg/harness/interfaces.go` | `harness/interfaces.go` |
| `pkg/harness/types.go` | `harness/types.go` |
| `pkg/harness/errors.go` | `harness/errors.go` |
| `pkg/harness/runner_registry.go` | `harness/runner_registry.go` |
| `pkg/harness/stream_store.go` | `harness/stream_store.go` |
| `pkg/harness/*_lite.go` | `harness/lite/*.go` |

Also move enterprise testutil from `ghostfin-enterprise/internal/testutil/` → `openharness/harness/enterprise/testutil/`.

Update `ghostfin/desktop` imports: `github.com/swiftdiaries/ghostfin/desktop/pkg/harness` → `github.com/swiftdiaries/openharness/harness`.

### Layer 2: Agent Primitives

**Source:** `ghostfin/desktop/internal/agent/`, `internal/tools/`, `internal/providers/`, `internal/mcp/`
**Destination:** `openharness/agent/`, `openharness/tools/`, `openharness/providers/`, `openharness/tools/mcp/`

ghostfin keeps only domain-specific tool implementations (fund extraction, classification, validation). Generic built-in tools (web_search, filesystem, exec, memory) move to openharness.

### Layer 3: Infrastructure

**Source:** `internal/cost/`, `internal/telemetry/`, `internal/traces/`, `internal/sessions/`, `internal/config/`, `internal/edition/`, `internal/scheduler/`
**Destination:** corresponding `openharness/` packages

Telemetry gets refactored into the two-layer architecture during this move:
- Existing SQLite trace store → `telemetry/lite/sqlite_metadata.go`
- New DuckDB span sink → `telemetry/lite/duckdb_sink.go`
- New OTLP span sink → `telemetry/enterprise/otlp_sink.go`
- New Postgres metadata → `telemetry/enterprise/pg_metadata.go`

### Layer 4: App Scaffold

**Source:** `internal/notes/`, `internal/gateway/`, `internal/bootstrap/`, `internal/projects/`, `app.go`, `main.go`, `frontend/`
**Destination:** `openharness/notes/`, `openharness/gateway/`, `openharness/bootstrap/`, `openharness/app/`

ghostfin keeps domain-specific frontend views (pipeline viewer, holdings table). Generic views (Claw chat, Journal knowledge graph) move to openharness.

### Layer 5: Enterprise Implementations

**Source:** implementation plan from `ghostfin-enterprise/docs/`
**Destination:** `openharness/harness/enterprise/`

Implement the generic enterprise backends:
- `session_pg.go`, `agent_pg.go`, `skill_pg.go`, `registry_pg.go` (Postgres)
- `secret_vault.go` (Vault)
- `artifact_s3.go` (S3/MinIO)
- `runner_k8s.go` (K8s Jobs)
- `runner_lambda.go` (Lambda, S3-triggered snapshot chain)
- `pg_metadata.go`, `otlp_sink.go` (enterprise tracing)

`ghostfin-enterprise` shrinks to: Nile multi-tenant wiring, billing, finance-specific RBAC, management server.

---

## Post-Migration: What Stays Where

### openharness (open source framework)

Everything in the package layout above. Generic, vertical-agnostic.

### ghostfin (finance vertical, open source)

```
ghostfin/
├── desktop/
│   ├── main.go              # Wires openharness + registers domain tools/agents/views
│   ├── tools/               # Fund extraction, classification, validation tools
│   ├── agents/              # Fund-analyst, scribe, executive-assistant definitions
│   └── frontend/views/      # Pipeline viewer, holdings table, domain-specific UI
├── pipeline/                # Python document processing (unchanged)
├── mock/                    # Mock custodian APIs (unchanged)
├── docs/                    # Finance-specific documentation
└── go.work                  # Links desktop + openharness for local dev
```

### ghostfin-enterprise (finance enterprise, proprietary)

```
ghostfin-enterprise/
├── internal/
│   ├── nile/               # Nile multi-tenant database wiring
│   ├── billing/            # Billing integration
│   ├── rbac/               # Finance-specific RBAC policies
│   └── mgmt/               # Management server (firm/user onboarding)
├── migrations/             # Nile/billing-specific migrations
└── go.mod                  # Imports openharness + ghostfin/desktop
```

---

## Verification

### Per-Layer Verification

After each extraction layer:
1. `cd openharness && go test ./...` — all extracted tests pass
2. `cd ghostfin/desktop && go test ./internal/... -tags sqliteonly` — ghostfin tests pass with new imports
3. `cd ghostfin/desktop && wails dev -tags sqliteonly` — desktop app starts and functions
4. `cd ghostfin-enterprise && go test ./...` — enterprise tests pass

### End-to-End Verification (Post Layer 4)

1. Create a minimal "hello world" vertical that imports openharness, registers one tool and one agent
2. Build and run it as a Wails app — verify agent chat, tool execution, tracing all work
3. Verify ghostfin still builds and runs identically to pre-migration behavior
4. Run ghostfin pipeline tests: `cd pipeline && uv run pytest`

### Enterprise Verification (Post Layer 5)

1. Run enterprise integration tests with testcontainers (Postgres, Vault, MinIO)
2. Verify K8s runner dispatches jobs correctly
3. Verify OTLP exporter sends spans to a test Honeycomb/Datadog endpoint
4. Verify ghostfin-enterprise still builds against both openharness and ghostfin/desktop
