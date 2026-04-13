# Layer 2 Plan 1: Interfaces + Concretes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move four thin Layer 3 concretes (`cost`, `sessions`, `edition`, `config`) from `ghostfin/desktop/internal/` into `openharness/`, and create `openharness/agent/` with consumer-side interfaces and no-op defaults so subsequent Layer 2 plans (providers, tools, MCP, agent loop) can compile against a stable foundation.

**Architecture:** Verbatim file moves for the four concretes (no logic changes). New `openharness/agent/` package gets four interfaces (`TraceRecorder`, `KnowledgeStore`, `MemoryStore`, `PromptContextProvider`), four shared types (`LLMCallInfo`, `Block`, `MemoryEntry`, `ContextFile`), and zero-value no-op implementations of each interface. The agent loop itself moves in Plan 5; Plan 1 only ships the consumer-side seams. `sessions/pg_store.go` is intentionally NOT moved — Postgres session storage is deferred to Layer 5 to keep Plan 1 free of `pgx` dependency.

**Tech Stack:** Go 1.25, stdlib only (no new third-party deps in Plan 1).

**Spec:** `docs/superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md`

**Execution context:** `docs/superpowers/plans/layer-2/2026-04-13-execution-order.md` (Wave 1A, runs in parallel with Plan 2 — Providers).

---

## File Inventory

### Files to copy from ghostfin (with optional build-tag stripping)

| Source | Destination | Notes |
|--------|-------------|-------|
| `ghostfin/desktop/internal/cost/tracker.go` | `openharness/cost/tracker.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/cost/pricing.go` | `openharness/cost/pricing.go` | Strip `//go:build sqliteonly` and any `// +build sqliteonly` line |
| `ghostfin/desktop/internal/cost/tracker_test.go` | `openharness/cost/tracker_test.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/cost/pricing_test.go` | `openharness/cost/pricing_test.go` | Strip `sqliteonly` tag |
| `ghostfin/desktop/internal/sessions/store.go` | `openharness/sessions/store.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/sessions/store_test.go` | `openharness/sessions/store_test.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/edition/edition.go` | `openharness/edition/edition.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/edition/edition_test.go` | `openharness/edition/edition_test.go` | Strip `sqliteonly` tag |
| `ghostfin/desktop/internal/config/config.go` | `openharness/config/config.go` | Strip `sqliteonly` tag |
| `ghostfin/desktop/internal/config/discovery.go` | `openharness/config/discovery.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/config/store.go` | `openharness/config/store.go` | Strip `sqliteonly` tag |
| `ghostfin/desktop/internal/config/config_test.go` | `openharness/config/config_test.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/config/discovery_test.go` | `openharness/config/discovery_test.go` | No tag, copy verbatim |
| `ghostfin/desktop/internal/config/store_test.go` | `openharness/config/store_test.go` | Strip `sqliteonly` tag |

### Files NOT moved in Plan 1 (deferred to Layer 5)

- `internal/sessions/pg_store.go` (`!sqliteonly`-gated, requires `pgx`)
- `internal/sessions/pg_store_test.go`

### Files to create from scratch

| Path | Purpose |
|------|---------|
| `openharness/agent/types.go` | `LLMCallInfo`, `Block`, `MemoryEntry`, `ContextFile` shared types |
| `openharness/agent/interfaces.go` | `TraceRecorder`, `KnowledgeStore`, `MemoryStore`, `PromptContextProvider` |
| `openharness/agent/defaults.go` | `NoOp*` implementations of each interface |
| `openharness/agent/interfaces_test.go` | Compile-time assertions + smoke tests for no-ops |

---

## Spec Discrepancies Corrected in This Plan

1. **`CostTracker` interface omitted.** The spec sketch in section "agent/interfaces.go" included `CostTracker`. Decision Q1 in the spec's key-decisions table commits to moving `cost` concretely, which makes that interface redundant. `openharness/agent/loop.go` (Plan 5) will import `openharness/cost` directly.
2. **`TraceRecorder` uses `LLMCallInfo` instead of `providers.Usage`.** Plans 1 and 2 run in parallel, so Plan 1 cannot reference Plan 2 types. `LLMCallInfo` is defined locally in `agent/types.go` with primitive fields; Plan 5's loop will convert `providers.Usage` → `LLMCallInfo` at the call site.

---

## Task 1: Worktree setup + sanity check

**Files:** none (setup only)

- [ ] **Step 1: Create the worktree**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git worktree add .worktrees/plan-1-interfaces-and-concretes -b feat/layer-2-plan-1-interfaces-and-concretes main
cd .worktrees/plan-1-interfaces-and-concretes
```

Expected: worktree created, branch `feat/layer-2-plan-1-interfaces-and-concretes` checked out.

- [ ] **Step 2: Verify baseline**

```bash
go test ./... -v
go build ./...
```

Expected: all Layer 1 tests pass (`harness/`, `harness/lite/`); build succeeds.

- [ ] **Step 3: Verify ghostfin source is accessible**

```bash
ls /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/cost/
ls /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/sessions/
ls /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/edition/
ls /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/config/
```

Expected: all four directories list with the file inventory above.

---

## Task 2: Move cost package

**Files:**
- Create: `openharness/cost/tracker.go`, `openharness/cost/pricing.go`, `openharness/cost/tracker_test.go`, `openharness/cost/pricing_test.go`

- [ ] **Step 1: Create the package directory and copy files**

```bash
mkdir -p cost
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/cost/tracker.go cost/tracker.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/cost/pricing.go cost/pricing.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/cost/tracker_test.go cost/tracker_test.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/cost/pricing_test.go cost/pricing_test.go
```

- [ ] **Step 2: Strip `sqliteonly` build tag from `pricing.go`**

Open `cost/pricing.go`. Find the lines:

```go
//go:build sqliteonly
```

(and possibly an old-style `// +build sqliteonly` directly under it, plus any blank line that follows the directive group).

Delete them so the file starts directly with the `package cost` declaration.

Use Edit tool with `old_string`:
```
//go:build sqliteonly

package cost
```
and `new_string`:
```
package cost
```

If the file uses the old-style two-line form (`//go:build sqliteonly\n// +build sqliteonly`), include both lines in `old_string`.

- [ ] **Step 3: Strip `sqliteonly` build tag from `pricing_test.go`**

Same procedure as Step 2 but for `cost/pricing_test.go`.

- [ ] **Step 4: Verify package builds and tests pass**

```bash
go build ./cost/...
go test ./cost/... -v
```

Expected: `ok  github.com/swiftdiaries/openharness/cost`. All `TestTracker*` and `TestCalculate*` tests pass.

If a test fails because it references `defaultPricing` or `pricingOverrides` (private vars), that's expected — they exist in the original file. The tests are in the same package so they can access privates.

- [ ] **Step 5: Verify no stray `sqliteonly` references remain**

```bash
grep -rn "sqliteonly" cost/
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add cost/
git commit -m "$(cat <<'EOF'
feat(cost): extract cost tracking package from ghostfin

Verbatim move of internal/cost/{tracker,pricing}.go from ghostfin
into openharness. Strips sqliteonly build tag — openharness ships
Lite + Enterprise in one module via runtime edition gating, not
build-time tag separation.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Move sessions package (Lite only)

**Files:**
- Create: `openharness/sessions/store.go`, `openharness/sessions/store_test.go`

**Note:** `pg_store.go` and `pg_store_test.go` are intentionally NOT moved. They depend on `pgx` and are gated by `!sqliteonly`. Postgres session storage moves to `openharness/sessions/` (or `openharness/harness/enterprise/`) in Layer 5 along with the rest of the enterprise backend extraction.

- [ ] **Step 1: Copy the Lite store files**

```bash
mkdir -p sessions
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/sessions/store.go sessions/store.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/sessions/store_test.go sessions/store_test.go
```

- [ ] **Step 2: Verify package builds and tests pass**

```bash
go build ./sessions/...
go test ./sessions/... -v
```

Expected: `ok  github.com/swiftdiaries/openharness/sessions`. All `TestStore*` tests pass. `google/uuid` is the only third-party dep introduced; `go build` will print "no required module provides..." if it's not in `go.mod` yet.

- [ ] **Step 3: Add `google/uuid` dependency if missing**

```bash
go get github.com/google/uuid
go mod tidy
```

Expected: `go.mod` now has a `require github.com/google/uuid v...` line.

- [ ] **Step 4: Re-run tests**

```bash
go test ./sessions/... -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add sessions/ go.mod go.sum
git commit -m "$(cat <<'EOF'
feat(sessions): extract Lite session store from ghostfin

Moves internal/sessions/store.go (JSONL-backed) verbatim. The
Postgres session store (pg_store.go) is deferred to Layer 5
enterprise extraction along with the rest of the enterprise
backend implementations.

Adds github.com/google/uuid as a transitive dependency.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Move edition package

**Files:**
- Create: `openharness/edition/edition.go`, `openharness/edition/edition_test.go`

- [ ] **Step 1: Copy the files**

```bash
mkdir -p edition
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/edition/edition.go edition/edition.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/edition/edition_test.go edition/edition_test.go
```

- [ ] **Step 2: Strip `sqliteonly` tag from `edition_test.go`**

Open `edition/edition_test.go`. Find the build directive at the top of the file (e.g. `//go:build sqliteonly` and the blank line after) and remove it so the file starts directly with `package edition`.

- [ ] **Step 3: Verify package builds and tests pass**

```bash
go build ./edition/...
go test ./edition/... -v
```

Expected: `ok  github.com/swiftdiaries/openharness/edition`. `TestLite`, `TestEnterprise`, `TestSetEdition`, `TestAgentLimitReached`, `TestIsLimited` all pass.

- [ ] **Step 4: Verify no stray `sqliteonly` references**

```bash
grep -rn "sqliteonly" edition/
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add edition/
git commit -m "$(cat <<'EOF'
feat(edition): extract edition gating package from ghostfin

Verbatim move of internal/edition/edition.go. Strips sqliteonly
build tag from the test file (the implementation file has no tag).

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Move config package

**Files:**
- Create: `openharness/config/config.go`, `openharness/config/discovery.go`, `openharness/config/store.go`, `openharness/config/config_test.go`, `openharness/config/discovery_test.go`, `openharness/config/store_test.go`

- [ ] **Step 1: Copy all six files**

```bash
mkdir -p config
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/config/config.go config/config.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/config/discovery.go config/discovery.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/config/store.go config/store.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/config/config_test.go config/config_test.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/config/discovery_test.go config/discovery_test.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/config/store_test.go config/store_test.go
```

- [ ] **Step 2: Strip `sqliteonly` tag from `config.go`**

Edit `config/config.go`: remove the `//go:build sqliteonly` line (and `// +build sqliteonly` if present) plus any blank line directly under it, so the file starts with `package config`.

- [ ] **Step 3: Strip `sqliteonly` tag from `store.go`**

Same procedure for `config/store.go`.

- [ ] **Step 4: Strip `sqliteonly` tag from `store_test.go`**

Same procedure for `config/store_test.go`.

- [ ] **Step 5: Verify package builds and tests pass**

```bash
go build ./config/...
go test ./config/... -v
```

Expected: `ok  github.com/swiftdiaries/openharness/config`. All `TestDefault*`, `TestLoad*`, `TestSave*`, `TestDiscoverInstructionFiles*`, `TestResolveModel*` tests pass.

- [ ] **Step 6: Verify no stray `sqliteonly` references**

```bash
grep -rn "sqliteonly" config/
```

Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add config/
git commit -m "$(cat <<'EOF'
feat(config): extract config package from ghostfin

Verbatim move of internal/config/{config,discovery,store}.go.
Strips sqliteonly build tags; config is generic and ships in
both Lite and Enterprise editions.

The MCPServerConfig type lives here and will be referenced by
Plan 4 (MCP) when that lands.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Create `openharness/agent/types.go`

**Files:**
- Create: `openharness/agent/types.go`

This file defines shared types used by the `openharness/agent` interfaces and (later) by the agent loop in Plan 5. Keeping types separate from interfaces makes it easy for verticals to construct value objects without importing the interface file.

- [ ] **Step 1: Create the package directory**

```bash
mkdir -p agent
```

- [ ] **Step 2: Write `agent/types.go`**

Create file `agent/types.go` with this content:

```go
// Package agent defines the openharness agent execution primitives:
// shared types, consumer-side interfaces (TraceRecorder, KnowledgeStore,
// MemoryStore, PromptContextProvider), and zero-value no-op
// implementations. The agent loop itself lands in Plan 5; Plan 1 only
// ships the seams that downstream packages compile against.
package agent

import "time"

// LLMCallInfo carries token + cost data for a single LLM call. It is
// intentionally defined with primitive fields rather than reusing
// providers.Usage so that the agent package does not depend on the
// providers package — the two packages move in parallel plans and
// must remain decoupled at the type level. The agent loop converts
// providers.Usage → LLMCallInfo at the call site.
type LLMCallInfo struct {
	Model               string
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	CostUSD             float64
	DurationMS          int64
}

// Block is a single content unit in a knowledge store. Verticals
// supply concrete blocks via their KnowledgeStore implementation.
// The Path field is optional and refers to a file or document
// location when the underlying store is filesystem-backed.
type Block struct {
	ID       string
	Title    string
	Content  string
	Path     string
	Metadata map[string]string
}

// MemoryEntry is a single entry from the memory store. Used by
// agent/retrieval.go (Plan 5) to inject relevant memories into
// the prompt before each LLM call.
type MemoryEntry struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

// ContextFile carries a named context document (persona, AGENTS.md,
// onboarding instructions) for inclusion in the system prompt. This
// type was previously defined in ghostfin's internal/bootstrap
// package; it is inlined here so the agent package has zero
// dependency on bootstrap. ghostfin/internal/bootstrap will
// re-export this type as an alias in Plan 7.
type ContextFile struct {
	Name    string
	Role    string
	Content string
}
```

- [ ] **Step 3: Verify the file compiles**

```bash
go build ./agent/...
```

Expected: builds successfully (no test files yet).

- [ ] **Step 4: Commit**

```bash
git add agent/types.go
git commit -m "$(cat <<'EOF'
feat(agent): add shared types for openharness/agent package

LLMCallInfo decouples token/cost data from providers.Usage so
that agent and providers can be developed in parallel plans
without a type-level dependency.

Block and MemoryEntry are the units of content that flow through
the KnowledgeStore and MemoryStore interfaces (added in the next
commit).

ContextFile is inlined from ghostfin's internal/bootstrap so the
agent package has zero dependency on bootstrap. ghostfin's
bootstrap package will re-export this type as an alias when
Plan 7 lands.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Create `openharness/agent/interfaces.go`

**Files:**
- Create: `openharness/agent/interfaces.go`

- [ ] **Step 1: Write the interfaces file**

Create file `agent/interfaces.go` with this content:

```go
package agent

import "context"

// TraceRecorder is the consumer-side interface the agent loop uses to
// record LLM call spans, tool-call spans, and run lifecycle events.
// Layer 3's telemetry refactor (out of scope for Layer 2) will provide
// a concrete implementation backed by SQLite + DuckDB (Lite) or
// Postgres + OTLP (Enterprise). Until then, verticals can pass
// NoOpTraceRecorder{} or implement this interface against their own
// telemetry stack.
type TraceRecorder interface {
	StartRun(ctx context.Context, runID, agentID, sessionID string) context.Context
	RecordLLMCall(ctx context.Context, info LLMCallInfo)
	RecordToolCall(ctx context.Context, tool string, args, result []byte, durationMS int64, err error)
	EndRun(ctx context.Context, status string, runErr error)
}

// KnowledgeStore is the framework-level interface for a user-owned
// markdown/block knowledge repo. Used by the knowledge_graph built-in
// tool (Plan 3) and by agent/retrieval.go (Plan 5). Ghostfin's
// internal/notes package implements this interface in Plan 7. When
// notes itself is extracted in Layer 4, the implementation moves
// into openharness.
type KnowledgeStore interface {
	Search(ctx context.Context, query string, limit int) ([]Block, error)
	Get(ctx context.Context, id string) (Block, error)
}

// MemoryStore is the framework-level interface for the key-value
// memory tool. Used by agent/retrieval.go to inject relevant
// memories into the prompt before each LLM call. The built-in
// memory tool (Plan 3) writes through this interface.
type MemoryStore interface {
	Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error)
}

// PromptContextProvider supplies dynamic context files (AGENTS.md,
// persona files, onboarding instructions) to the system prompt
// builder. Verticals implement this against whatever document
// source they prefer — filesystem, database, in-memory.
type PromptContextProvider interface {
	Files(ctx context.Context) ([]ContextFile, error)
}
```

- [ ] **Step 2: Verify the file compiles**

```bash
go build ./agent/...
```

Expected: builds successfully.

- [ ] **Step 3: Commit**

```bash
git add agent/interfaces.go
git commit -m "$(cat <<'EOF'
feat(agent): add consumer-side interfaces for openharness/agent

Adds four interfaces that downstream Layer 2 plans depend on:
- TraceRecorder: telemetry inversion seam (Layer 3 will provide
  the concrete SQLite/Postgres-backed impl)
- KnowledgeStore: framework-level interface for user-owned
  markdown/block knowledge repos
- MemoryStore: key-value memory store interface
- PromptContextProvider: dynamic context file supplier for the
  system prompt builder

These interfaces let Plans 3, 4, 5, and 6 develop against a
stable foundation. No-op implementations land in the next commit.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Create `openharness/agent/defaults.go`

**Files:**
- Create: `openharness/agent/defaults.go`

- [ ] **Step 1: Write the defaults file**

Create file `agent/defaults.go` with this content:

```go
package agent

import "context"

// NoOpTraceRecorder is a zero-value TraceRecorder that does nothing.
// Use it in tests or as the default when no telemetry backend is
// wired up. The agent loop should be safe to run with this
// implementation — no panics, no nil dereferences.
type NoOpTraceRecorder struct{}

func (NoOpTraceRecorder) StartRun(ctx context.Context, runID, agentID, sessionID string) context.Context {
	return ctx
}

func (NoOpTraceRecorder) RecordLLMCall(ctx context.Context, info LLMCallInfo) {}

func (NoOpTraceRecorder) RecordToolCall(ctx context.Context, tool string, args, result []byte, durationMS int64, err error) {
}

func (NoOpTraceRecorder) EndRun(ctx context.Context, status string, runErr error) {}

// NoOpKnowledgeStore is a zero-value KnowledgeStore that returns
// empty results for every query. Use it when a vertical does not
// have a knowledge backend (the knowledge_graph built-in tool
// becomes a no-op).
type NoOpKnowledgeStore struct{}

func (NoOpKnowledgeStore) Search(ctx context.Context, query string, limit int) ([]Block, error) {
	return nil, nil
}

func (NoOpKnowledgeStore) Get(ctx context.Context, id string) (Block, error) {
	return Block{}, nil
}

// NoOpMemoryStore is a zero-value MemoryStore that returns empty
// results for every query. Use it when memory injection is
// undesirable or when no memory backend is available.
type NoOpMemoryStore struct{}

func (NoOpMemoryStore) Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error) {
	return nil, nil
}

// NoOpPromptContextProvider is a zero-value PromptContextProvider
// that supplies no context files. The system prompt builder falls
// back to its built-in defaults.
type NoOpPromptContextProvider struct{}

func (NoOpPromptContextProvider) Files(ctx context.Context) ([]ContextFile, error) {
	return nil, nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./agent/...
```

Expected: builds successfully.

- [ ] **Step 3: Commit**

```bash
git add agent/defaults.go
git commit -m "$(cat <<'EOF'
feat(agent): add zero-value no-op interface implementations

NoOpTraceRecorder, NoOpKnowledgeStore, NoOpMemoryStore, and
NoOpPromptContextProvider let openharness compile and run
standalone without requiring a vertical to wire up backends.
They are the safe defaults when a feature is unused.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Test the agent interface package

**Files:**
- Create: `openharness/agent/interfaces_test.go`

This task uses test-driven verification: write the test first, run it to confirm the no-op impls satisfy the interfaces and don't panic.

- [ ] **Step 1: Write the test file**

Create `agent/interfaces_test.go`:

```go
package agent

import (
	"context"
	"errors"
	"testing"
)

// TestNoOpsSatisfyInterfaces is a compile-time assertion: if any
// no-op type drifts away from its interface contract, this file
// fails to compile and the test target reports the mismatch.
func TestNoOpsSatisfyInterfaces(t *testing.T) {
	var _ TraceRecorder = NoOpTraceRecorder{}
	var _ KnowledgeStore = NoOpKnowledgeStore{}
	var _ MemoryStore = NoOpMemoryStore{}
	var _ PromptContextProvider = NoOpPromptContextProvider{}
}

// TestNoOpTraceRecorder_DoesNotPanic exercises every method on
// NoOpTraceRecorder to confirm none of them dereference nil or
// otherwise crash. The agent loop must be safe to run with this
// implementation.
func TestNoOpTraceRecorder_DoesNotPanic(t *testing.T) {
	ctx := context.Background()
	rec := NoOpTraceRecorder{}

	gotCtx := rec.StartRun(ctx, "run-1", "agent-1", "session-1")
	if gotCtx != ctx {
		t.Errorf("StartRun should return the same context unchanged")
	}

	rec.RecordLLMCall(ctx, LLMCallInfo{
		Model:        "claude-opus-4-6",
		InputTokens:  100,
		OutputTokens: 50,
		CostUSD:      0.0034,
		DurationMS:   1234,
	})

	rec.RecordToolCall(ctx, "filesystem", []byte(`{"path":"/tmp"}`), []byte(`{"ok":true}`), 42, nil)
	rec.RecordToolCall(ctx, "exec", nil, nil, 0, errors.New("denied"))

	rec.EndRun(ctx, "completed", nil)
	rec.EndRun(ctx, "error", errors.New("budget exceeded"))
}

// TestNoOpKnowledgeStore_ReturnsEmpty confirms Search and Get
// return empty results without errors. Tools and retrieval that
// fall back to this default should observe "no results" cleanly.
func TestNoOpKnowledgeStore_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := NoOpKnowledgeStore{}

	blocks, err := store.Search(ctx, "anything", 10)
	if err != nil {
		t.Errorf("Search returned unexpected error: %v", err)
	}
	if blocks != nil {
		t.Errorf("Search should return nil slice, got %v", blocks)
	}

	block, err := store.Get(ctx, "any-id")
	if err != nil {
		t.Errorf("Get returned unexpected error: %v", err)
	}
	if block != (Block{}) {
		t.Errorf("Get should return zero-value Block, got %+v", block)
	}
}

// TestNoOpMemoryStore_ReturnsEmpty confirms the memory store no-op.
func TestNoOpMemoryStore_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := NoOpMemoryStore{}

	entries, err := store.Search(ctx, "query", 5)
	if err != nil {
		t.Errorf("Search returned unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("Search should return nil slice, got %v", entries)
	}
}

// TestNoOpPromptContextProvider_ReturnsEmpty confirms the prompt
// context provider no-op.
func TestNoOpPromptContextProvider_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	provider := NoOpPromptContextProvider{}

	files, err := provider.Files(ctx)
	if err != nil {
		t.Errorf("Files returned unexpected error: %v", err)
	}
	if files != nil {
		t.Errorf("Files should return nil slice, got %v", files)
	}
}
```

- [ ] **Step 2: Run the tests**

```bash
go test ./agent/... -v
```

Expected: 5 tests pass:
```
=== RUN   TestNoOpsSatisfyInterfaces
--- PASS: TestNoOpsSatisfyInterfaces
=== RUN   TestNoOpTraceRecorder_DoesNotPanic
--- PASS: TestNoOpTraceRecorder_DoesNotPanic
=== RUN   TestNoOpKnowledgeStore_ReturnsEmpty
--- PASS: TestNoOpKnowledgeStore_ReturnsEmpty
=== RUN   TestNoOpMemoryStore_ReturnsEmpty
--- PASS: TestNoOpMemoryStore_ReturnsEmpty
=== RUN   TestNoOpPromptContextProvider_ReturnsEmpty
--- PASS: TestNoOpPromptContextProvider_ReturnsEmpty
PASS
ok  	github.com/swiftdiaries/openharness/agent
```

- [ ] **Step 3: Commit**

```bash
git add agent/interfaces_test.go
git commit -m "$(cat <<'EOF'
test(agent): verify no-op impls satisfy interfaces and do not panic

TestNoOpsSatisfyInterfaces is a compile-time assertion via
typed nil declarations — if a no-op type drifts away from its
interface contract, this file fails to compile.

The remaining tests exercise every method to catch any nil
dereference or panic in the no-op paths.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Run the full test suite**

```bash
go test ./... -v
```

Expected: every package passes — `harness`, `harness/lite`, `cost`, `sessions`, `edition`, `config`, `agent`. No failures, no skipped suites.

- [ ] **Step 2: Build everything**

```bash
go build ./...
```

Expected: builds with no errors.

- [ ] **Step 3: Run `go vet`**

```bash
go vet ./...
```

Expected: no vet warnings.

- [ ] **Step 4: Verify `go.mod` is clean**

```bash
go mod tidy
git diff go.mod go.sum
```

Expected: empty diff (no changes from `go mod tidy` because all dep additions were committed in Task 3).

- [ ] **Step 5: Verify no `sqliteonly` references remain anywhere in the new packages**

```bash
grep -rn "sqliteonly" cost/ sessions/ edition/ config/ agent/
```

Expected: no output.

- [ ] **Step 6: Confirm package structure**

```bash
find cost sessions edition config agent -name "*.go" | sort
```

Expected output:
```
agent/defaults.go
agent/interfaces.go
agent/interfaces_test.go
agent/types.go
config/config.go
config/config_test.go
config/discovery.go
config/discovery_test.go
config/store.go
config/store_test.go
cost/pricing.go
cost/pricing_test.go
cost/tracker.go
cost/tracker_test.go
edition/edition.go
edition/edition_test.go
sessions/store.go
sessions/store_test.go
```

18 files total: 14 moved from ghostfin (cost ×4, sessions ×2, edition ×2, config ×6) + 4 new in `agent/` (types, interfaces, defaults, interfaces_test). Verify the count matches.

- [ ] **Step 7: Push the branch**

```bash
git push -u origin feat/layer-2-plan-1-interfaces-and-concretes
```

Expected: branch published to origin.

---

## Verification Summary

After Plan 1 lands on `main`, the openharness module has:

- **Five new packages**: `cost`, `sessions`, `edition`, `config`, `agent`
- **Four new interfaces** in `agent/`: `TraceRecorder`, `KnowledgeStore`, `MemoryStore`, `PromptContextProvider`
- **Four new shared types** in `agent/`: `LLMCallInfo`, `Block`, `MemoryEntry`, `ContextFile`
- **Four no-op implementations** in `agent/`: `NoOpTraceRecorder`, `NoOpKnowledgeStore`, `NoOpMemoryStore`, `NoOpPromptContextProvider`
- **One new transitive dependency**: `github.com/google/uuid`

Plans 3, 4, 5, and 6 can now compile against `openharness/agent` without waiting on Plan 5's loop implementation. Plan 2 (Providers) is fully independent and runs in parallel.

## Merge Instructions (after all tasks complete)

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git checkout main
git pull
git merge --no-ff feat/layer-2-plan-1-interfaces-and-concretes
go test ./... -v
go build ./...
git push origin main
git tag layer-2-plan-1-complete
git push origin layer-2-plan-1-complete
git worktree remove .worktrees/plan-1-interfaces-and-concretes
```

If Plan 2 has also merged at this point, tag `layer-2-wave-1` per the execution-order doc.
