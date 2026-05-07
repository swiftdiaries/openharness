# agent

First-depth walkthrough of `agent/`.

## Package summary

Package `agent` defines openharness agent execution primitives at the **seam level only**: shared types, four consumer-side interfaces, and zero-value no-op implementations. Per its own doc-comment ([`types.go:1-6`](../../agent/types.go)), the agent loop itself is deferred to Plan 5; this package ships the contracts that downstream packages compile against.

The package is intentionally decoupled from `providers/` — `LLMCallInfo` re-defines token/cost fields rather than reusing `providers.Usage` so the two packages can move independently ([`types.go:11-15`](../../agent/types.go)).

**Importers (today):** `tools/core/memory.go` (to satisfy `MemoryStore`) and its tests. No agent loop is wired yet.

## Files at depth 1

- `agent/interfaces.go` (44 lines) — Four consumer-side interfaces.
- `agent/types.go` (57 lines) — Shared data types (`LLMCallInfo`, `Block`, `MemoryEntry`, `ContextFile`).
- `agent/defaults.go` (52 lines) — Zero-value no-op implementations of every interface.
- `agent/interfaces_test.go` (98 lines) — Compile-time interface assertions and basic behavior tests for the no-ops.

## Subdirectories at depth 1

None.

## Public surface

### Interfaces

- `TraceRecorder` — [`interfaces.go:12-17`](../../agent/interfaces.go) — LLM-call/tool-call/run-lifecycle telemetry. Methods: `StartRun(ctx, runID, agentID, sessionID) context.Context`, `RecordLLMCall(ctx, info)`, `RecordToolCall(ctx, tool, args, result, durationMS, err)`, `EndRun(ctx, status, runErr)`. Layer 3 will ship a real impl (SQLite+DuckDB / Postgres+OTLP per the doc-comment).
- `KnowledgeStore` — [`interfaces.go:25-28`](../../agent/interfaces.go) — Markdown/block knowledge repo. Methods: `Search(ctx, query, limit) ([]Block, error)`, `Get(ctx, id) (Block, error)`. Doc-comment notes ghostfin's `internal/notes` will be moved into openharness in Layer 4.
- `MemoryStore` — [`interfaces.go:34-36`](../../agent/interfaces.go) — KV memory injection. Method: `Search(ctx, query, limit) ([]MemoryEntry, error)`.
- `PromptContextProvider` — [`interfaces.go:42-44`](../../agent/interfaces.go) — Dynamic context files for system-prompt builder. Method: `Files(ctx) ([]ContextFile, error)`.

### Concrete types

- `LLMCallInfo` — [`types.go:16-24`](../../agent/types.go) — Model, token counts (incl. cache creation/read), CostUSD, DurationMS.
- `Block` — [`types.go:30-36`](../../agent/types.go) — ID, Title, Content, Path, Metadata.
- `MemoryEntry` — [`types.go:41-45`](../../agent/types.go) — Key, Value, UpdatedAt.
- `ContextFile` — [`types.go:53-57`](../../agent/types.go) — Name, Role, Content.

### Exported functions

None. Surface is types + interfaces only.

### No-op default impls

- `NoOpTraceRecorder` — [`defaults.go:9-20`](../../agent/defaults.go).
- `NoOpKnowledgeStore` — [`defaults.go:26-34`](../../agent/defaults.go).
- `NoOpMemoryStore` — [`defaults.go:39-43`](../../agent/defaults.go).
- `NoOpPromptContextProvider` — [`defaults.go:48-52`](../../agent/defaults.go).

## Pluggability hooks

| Interface | Concrete impl in `agent/` | Where downstream impl lives |
|---|---|---|
| `TraceRecorder` | No-op only | Deferred — Layer 3 |
| `KnowledgeStore` | No-op only | Deferred — Layer 4 (ghostfin `internal/notes` to be extracted) |
| `MemoryStore` | No-op only | `tools/core/memory.go` (file-backed) satisfies it today |
| `PromptContextProvider` | No-op only | Verticals supply their own |

The package shape — interface + no-op default — is a textbook extension-point pattern. The no-ops let downstream code compile and run without telemetry/knowledge/memory wired up.

## External deps used

stdlib only: `context`, `time`, `errors`, `testing`.

## Notable tests

- `TestNoOpsSatisfyInterfaces` — [`interfaces_test.go:12`](../../agent/interfaces_test.go) — Compile-time assertion that no-op types still match interface contracts; signature drift breaks the build.
- `TestNoOpTraceRecorder_DoesNotPanic` — [`interfaces_test.go:23`](../../agent/interfaces_test.go) — Exercises every method; agent loop must be safe to run with no-op.
- `TestNoOpKnowledgeStore_ReturnsEmpty` — [`interfaces_test.go:50`](../../agent/interfaces_test.go).
- `TestNoOpMemoryStore_ReturnsEmpty` — [`interfaces_test.go:72`](../../agent/interfaces_test.go).
- `TestNoOpPromptContextProvider_ReturnsEmpty` — [`interfaces_test.go:87`](../../agent/interfaces_test.go).

## Open questions / gaps

1. **No agent loop.** There is no `Run(...)`, `NewAgent(...)`, or other entry point at this depth. The package is purely contracts; the loop lands in Plan 5.
2. **No assembly/factory.** No `Config` struct or constructor wires the four interfaces together. Unclear how downstream code will inject all four at once when the loop arrives.
3. **`TraceRecorder` is a complete stub.** Doc-comment defers a real impl to Layer 3; verticals must supply their own until then.
4. **Decoupling from `providers/` adds friction.** Every caller must convert `providers.Usage → LLMCallInfo` at the call site. Likely fine; may want a converter helper later.
5. **Reviewer follow-up:** confirm Plan 5 (agent loop) lands in this package or a sibling, and whether `Config{Trace, Knowledge, Memory, Prompt}` is the wiring shape.
