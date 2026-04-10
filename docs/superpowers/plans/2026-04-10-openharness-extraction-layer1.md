# OpenHarness Extraction — Layer 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap the `openharness` Go module and extract the harness boundary layer (8 interfaces + 7 Lite implementations + RunnerRegistry + StoreBackedEventStream) from `ghostfin/desktop/pkg/harness/` into `openharness/harness/`. Update ghostfin to import from openharness. This is Layer 1 of a 5-layer incremental extraction.

**Architecture:** The harness package has zero dependencies on ghostfin's internal packages — it's entirely self-contained (stdlib only: context, encoding/json, time, sync, io, fmt, errors). Extraction is a move + repackage: interfaces and shared types stay in `openharness/harness/`, Lite implementations move to `openharness/harness/lite/` (separate sub-package for clean isolation). ghostfin/desktop then imports openharness and removes its local copy.

**Tech Stack:** Go 1.25, go.work for local development, modernc.org/sqlite (build tag: sqliteonly)

**Reference:** [Design Spec](../docs/superpowers/specs/2026-04-10-openharness-extraction-design.md)

---

## File Structure

### New files in openharness

| File | Responsibility |
|------|---------------|
| `go.mod` | Module definition: `github.com/swiftdiaries/openharness` |
| `harness/interfaces.go` | 8 boundary interfaces (HarnessRunner, SessionStore, SecretStore, ArtifactStore, ToolRegistry, ChannelRouter, AgentStore, SkillStore) |
| `harness/types.go` | All shared types (Event, RunConfig, AgentDefinition, Snapshot, etc.) |
| `harness/errors.go` | Sentinel errors |
| `harness/runner_registry.go` | RunnerRegistry: maps runner names → HarnessRunner implementations |
| `harness/stream_store.go` | StoreBackedEventStream: polls SessionStore for events |
| `harness/interfaces_test.go` | Compile-time interface satisfaction checks |
| `harness/runner_registry_test.go` | RunnerRegistry unit tests |
| `harness/stream_store_test.go` | StoreBackedEventStream unit tests |
| `harness/lite/runner.go` | LiteRunner (goroutine-based HarnessRunner) |
| `harness/lite/runner_test.go` | LiteRunner tests |
| `harness/lite/agent.go` | LiteAgentStore (in-memory) |
| `harness/lite/agent_test.go` | LiteAgentStore tests |
| `harness/lite/session.go` | LiteSessionStore (file-based) — stub (full impl in Layer 3) |
| `harness/lite/secret.go` | LiteSecretStore (env vars) |
| `harness/lite/secret_test.go` | LiteSecretStore tests |
| `harness/lite/artifact.go` | LiteArtifactStore (filesystem) |
| `harness/lite/artifact_test.go` | LiteArtifactStore tests |
| `harness/lite/registry.go` | LiteToolRegistry (in-memory) |
| `harness/lite/registry_test.go` | LiteToolRegistry tests |
| `harness/lite/skill.go` | LiteSkillStore (filesystem-based) |
| `harness/lite/skill_test.go` | LiteSkillStore tests |
| `harness/lite/channel.go` | LiteChannelRouter (notification-only) |
| `harness/lite/channel_test.go` | LiteChannelRouter tests |

### Modified files in ghostfin

| File | Change |
|------|--------|
| `desktop/go.mod` | Add `require github.com/swiftdiaries/openharness` |
| `go.work` | Add `../openharness` to workspace |
| All files importing `pkg/harness` | Update import path to `github.com/swiftdiaries/openharness/harness` and `github.com/swiftdiaries/openharness/harness/lite` |

### Deleted files in ghostfin (after migration verified)

| File | Reason |
|------|--------|
| `desktop/pkg/harness/*.go` | Moved to openharness |
| `desktop/pkg/harness/*_test.go` | Moved to openharness |

---

## Task 1: Bootstrap openharness Go module

**Files:**
- Create: `openharness/go.mod`

- [ ] **Step 1: Initialize the Go module**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go mod init github.com/swiftdiaries/openharness
```

Expected: `go.mod` created with `module github.com/swiftdiaries/openharness` and `go 1.25.0`

- [ ] **Step 2: Verify the module compiles**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go build ./...
```

Expected: No errors (no Go files yet, just the module)

- [ ] **Step 3: Commit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git add go.mod
git commit -m "chore: bootstrap openharness Go module"
```

---

## Task 2: Extract harness interfaces and types

**Files:**
- Create: `openharness/harness/interfaces.go`
- Create: `openharness/harness/types.go`
- Create: `openharness/harness/errors.go`
- Source: `ghostfin/desktop/pkg/harness/interfaces.go` (83 lines)
- Source: `ghostfin/desktop/pkg/harness/types.go` (244 lines)
- Source: `ghostfin/desktop/pkg/harness/errors.go` (14 lines)

- [ ] **Step 1: Write the compile-time interface test**

Create `openharness/harness/interfaces_test.go`:

```go
package harness

import (
	"testing"
)

func TestInterfacesDefined(t *testing.T) {
	// Verify each interface type is usable as a variable type.
	var _ HarnessRunner
	var _ SessionStore
	var _ SecretStore
	var _ ArtifactStore
	var _ ToolRegistry
	var _ ChannelRouter
	var _ AgentStore
	var _ SkillStore
}

func TestRunConfigFields(t *testing.T) {
	cfg := RunConfig{
		RunID:     "run-1",
		SessionID: "sess-1",
		TenantID:  "tenant-1",
		Mode:      "auto",
	}
	if cfg.RunID != "run-1" {
		t.Fatalf("RunID = %q", cfg.RunID)
	}
}

func TestEventTypes(t *testing.T) {
	e := Event{Type: EventRunStarted}
	if e.Type != "run.started" {
		t.Fatalf("Type = %q", e.Type)
	}
}

func TestAgentDefinitionRunnerField(t *testing.T) {
	def := AgentDefinition{
		ID:     "agent-1",
		Runner: "lambda",
	}
	if def.Runner != "lambda" {
		t.Fatalf("Runner = %q, want lambda", def.Runner)
	}
}

func TestSnapshotFields(t *testing.T) {
	s := Snapshot{
		SessionID: "sess-1",
		RunID:     "run-1",
		Sequence:  3,
		Status:    "running",
	}
	if s.Sequence != 3 {
		t.Fatalf("Sequence = %d", s.Sequence)
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrNotFound == nil {
		t.Fatal("ErrNotFound should not be nil")
	}
	if ErrConflict == nil {
		t.Fatal("ErrConflict should not be nil")
	}
	if ErrRunNotActive == nil {
		t.Fatal("ErrRunNotActive should not be nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./harness/ -v
```

Expected: FAIL — types not defined yet.

- [ ] **Step 3: Copy interfaces.go from ghostfin**

Copy `ghostfin/desktop/pkg/harness/interfaces.go` → `openharness/harness/interfaces.go`. The file is self-contained — no import path changes needed (package is already `harness`, imports are stdlib only: `context`, `encoding/json`, `io`, `time`).

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/pkg/harness/interfaces.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/harness/interfaces.go
```

- [ ] **Step 4: Copy types.go from ghostfin**

Copy `ghostfin/desktop/pkg/harness/types.go` → `openharness/harness/types.go`. Self-contained (stdlib only: `encoding/json`, `time`).

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/pkg/harness/types.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/harness/types.go
```

- [ ] **Step 5: Copy errors.go from ghostfin**

Copy `ghostfin/desktop/pkg/harness/errors.go` → `openharness/harness/errors.go`. Self-contained (stdlib only: `errors`).

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/pkg/harness/errors.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/harness/errors.go
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./harness/ -v
```

Expected: PASS — all interface, type, and error sentinel tests green.

- [ ] **Step 7: Commit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git add harness/interfaces.go harness/types.go harness/errors.go harness/interfaces_test.go
git commit -m "feat(harness): extract boundary interfaces, types, and errors from ghostfin"
```

---

## Task 3: Extract RunnerRegistry and StoreBackedEventStream

**Files:**
- Create: `openharness/harness/runner_registry.go`
- Create: `openharness/harness/runner_registry_test.go`
- Create: `openharness/harness/stream_store.go`
- Create: `openharness/harness/stream_store_test.go`
- Source: `ghostfin/desktop/pkg/harness/runner_registry.go` (40 lines)
- Source: `ghostfin/desktop/pkg/harness/stream_store.go` (89 lines)

- [ ] **Step 1: Write RunnerRegistry test**

Create `openharness/harness/runner_registry_test.go`:

```go
package harness

import (
	"testing"
)

func TestRunnerRegistryGetUnregistered(t *testing.T) {
	reg := NewRunnerRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered runner")
	}
}

func TestRunnerRegistryRegisterAndGet(t *testing.T) {
	reg := NewRunnerRegistry()
	// Use a nil runner just to verify registration works
	// (LiteRunner not available yet in this package)
	reg.Register("test", nil)
	runner, err := reg.Get("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner != nil {
		t.Fatal("expected nil runner")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./harness/ -v -run TestRunnerRegistry
```

Expected: FAIL — NewRunnerRegistry not defined.

- [ ] **Step 3: Copy runner_registry.go from ghostfin**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/pkg/harness/runner_registry.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/harness/runner_registry.go
```

Self-contained (imports: `fmt`, `sync`).

- [ ] **Step 4: Run RunnerRegistry tests**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./harness/ -v -run TestRunnerRegistry
```

Expected: PASS

- [ ] **Step 5: Copy stream_store.go from ghostfin**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/pkg/harness/stream_store.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/harness/stream_store.go
```

Self-contained (imports: `context`, `io`, `time`). Uses `SessionStore` and `EventStream` interfaces defined in interfaces.go.

- [ ] **Step 6: Write StoreBackedEventStream test**

Create `openharness/harness/stream_store_test.go`:

```go
package harness

import (
	"context"
	"io"
	"testing"
	"time"
)

// mockSessionStore is a minimal SessionStore for testing StoreBackedEventStream.
type mockSessionStore struct {
	events []Event
}

func (m *mockSessionStore) CreateSession(_ context.Context, _ string, _ SessionMeta) (string, error) {
	return "sess-1", nil
}
func (m *mockSessionStore) AppendEvent(_ context.Context, _ string, e Event) error {
	m.events = append(m.events, e)
	return nil
}
func (m *mockSessionStore) GetEvents(_ context.Context, _ string, fromPos int64) ([]Event, error) {
	if int(fromPos) >= len(m.events) {
		return nil, nil
	}
	return m.events[fromPos:], nil
}
func (m *mockSessionStore) GetSession(_ context.Context, _ string) (Session, error) {
	return Session{}, nil
}
func (m *mockSessionStore) SetCheckpoint(_ context.Context, _ string, _ int64) error {
	return nil
}
func (m *mockSessionStore) ListSessions(_ context.Context, _ string, _ ListOpts) ([]Session, error) {
	return nil, nil
}

func TestStoreBackedEventStreamReadsEvents(t *testing.T) {
	store := &mockSessionStore{
		events: []Event{
			{Type: EventRunStarted, Position: 0, Timestamp: time.Now()},
			{Type: EventRunCompleted, Position: 1, Timestamp: time.Now()},
		},
	}
	stream := NewStoreBackedEventStream(store, "sess-1", 10*time.Millisecond)
	defer stream.Close()

	ev1, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev1.Type != EventRunStarted {
		t.Fatalf("Type = %q, want %q", ev1.Type, EventRunStarted)
	}

	ev2, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev2.Type != EventRunCompleted {
		t.Fatalf("Type = %q, want %q", ev2.Type, EventRunCompleted)
	}
}

func TestStoreBackedEventStreamClose(t *testing.T) {
	store := &mockSessionStore{}
	stream := NewStoreBackedEventStream(store, "sess-1", 10*time.Millisecond)
	err := stream.Close()
	if err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	_, err = stream.Next()
	if err != io.EOF {
		t.Fatalf("Next() after Close = %v, want io.EOF", err)
	}
}
```

- [ ] **Step 7: Run all harness tests**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./harness/ -v
```

Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git add harness/runner_registry.go harness/runner_registry_test.go harness/stream_store.go harness/stream_store_test.go
git commit -m "feat(harness): extract RunnerRegistry and StoreBackedEventStream"
```

---

## Task 4: Extract Lite implementations into harness/lite/

**Files:**
- Create: `openharness/harness/lite/runner.go`
- Create: `openharness/harness/lite/agent.go`
- Create: `openharness/harness/lite/secret.go`
- Create: `openharness/harness/lite/artifact.go`
- Create: `openharness/harness/lite/registry.go`
- Create: `openharness/harness/lite/skill.go`
- Create: `openharness/harness/lite/channel.go`
- Source: `ghostfin/desktop/pkg/harness/*_lite.go`

The Lite implementations are currently in the `harness` package (flat). They need to move to `harness/lite` (sub-package), which means:
1. Change `package harness` → `package lite`
2. Add `import "github.com/swiftdiaries/openharness/harness"` for interface types
3. Qualify all types: `RunConfig` → `harness.RunConfig`, `Event` → `harness.Event`, etc.

- [ ] **Step 1: Write LiteRunner test in harness/lite/**

Create `openharness/harness/lite/runner_test.go`:

```go
package lite

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time check.
var _ harness.HarnessRunner = (*LiteRunner)(nil)

func TestLiteRunnerDispatchAndComplete(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		events <- harness.Event{Type: harness.EventChunk, Content: "hello"}
		return nil
	}
	runner := NewLiteRunner(factory)

	stream, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}

	// run.started
	ev, err := stream.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if ev.Type != harness.EventRunStarted {
		t.Fatalf("Type = %q, want run.started", ev.Type)
	}

	// chunk from factory
	ev, err = stream.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if ev.Type != harness.EventChunk || ev.Content != "hello" {
		t.Fatalf("got %+v, want chunk 'hello'", ev)
	}

	// run.completed
	ev, err = stream.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if ev.Type != harness.EventRunCompleted {
		t.Fatalf("Type = %q, want run.completed", ev.Type)
	}

	// EOF
	_, err = stream.Next()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestLiteRunnerDuplicateRunID(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		<-ctx.Done()
		return ctx.Err()
	}
	runner := NewLiteRunner(factory)
	_, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("first Dispatch error: %v", err)
	}
	_, err = runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err == nil {
		t.Fatal("expected error for duplicate RunID")
	}
}

func TestLiteRunnerCancel(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		<-ctx.Done()
		return ctx.Err()
	}
	runner := NewLiteRunner(factory)
	_, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}

	err = runner.Cancel(context.Background(), "r1")
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	status, err := runner.Status(context.Background(), "r1")
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status.Status != "cancelled" {
		t.Fatalf("Status = %q, want cancelled", status.Status)
	}
}

func TestLiteRunnerStatusNotFound(t *testing.T) {
	runner := NewLiteRunner(nil)
	_, err := runner.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
}

func TestLiteRunnerResumeUnsupported(t *testing.T) {
	runner := NewLiteRunner(nil)
	_, err := runner.Resume(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error: resume not supported in Lite")
	}
}

func TestLiteRunnerFactoryError(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		return io.ErrUnexpectedEOF
	}
	runner := NewLiteRunner(factory)
	stream, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}

	// run.started
	ev, _ := stream.Next()
	if ev.Type != harness.EventRunStarted {
		t.Fatalf("expected run.started, got %q", ev.Type)
	}

	// error event
	ev, _ = stream.Next()
	if ev.Type != harness.EventError {
		t.Fatalf("expected error, got %q", ev.Type)
	}

	// Verify status is error
	time.Sleep(10 * time.Millisecond)
	status, _ := runner.Status(context.Background(), "r1")
	if status.Status != "error" {
		t.Fatalf("Status = %q, want error", status.Status)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./harness/lite/ -v
```

Expected: FAIL — LiteRunner not defined.

- [ ] **Step 3: Copy and adapt runner_lite.go → harness/lite/runner.go**

Copy `ghostfin/desktop/pkg/harness/runner_lite.go` to `openharness/harness/lite/runner.go` and make these changes:
1. Change `package harness` → `package lite`
2. Add import `"github.com/swiftdiaries/openharness/harness"`
3. Qualify all harness types: `RunConfig` → `harness.RunConfig`, `Event` → `harness.Event`, `EventRunStarted` → `harness.EventRunStarted`, `EventRunCompleted` → `harness.EventRunCompleted`, `EventError` → `harness.EventError`, `RunStatus` → `harness.RunStatus`, `EventStream` → `harness.EventStream`, `ErrConflict` → `harness.ErrConflict`, `ErrNotFound` → `harness.ErrNotFound`, `ErrRunNotActive` → `harness.ErrRunNotActive`
4. Keep the `LoopFactory` type definition but update its signature to use qualified types: `func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error`

- [ ] **Step 4: Run LiteRunner tests**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./harness/lite/ -v -run TestLiteRunner
```

Expected: ALL PASS

- [ ] **Step 5: Copy and adapt remaining Lite implementations**

For each file, apply the same transformation: `package harness` → `package lite`, add `"github.com/swiftdiaries/openharness/harness"` import, qualify all harness types.

Copy and adapt these files (each one following the same pattern):

| Source (ghostfin) | Destination (openharness) |
|---|---|
| `pkg/harness/agent_lite.go` | `harness/lite/agent.go` |
| `pkg/harness/secret_lite.go` | `harness/lite/secret.go` |
| `pkg/harness/artifact_lite.go` | `harness/lite/artifact.go` |
| `pkg/harness/registry_lite.go` | `harness/lite/registry.go` |
| `pkg/harness/skill_lite.go` | `harness/lite/skill.go` |
| `pkg/harness/channel_lite.go` | `harness/lite/channel.go` |

For each file:
1. Change `package harness` → `package lite`
2. Add `import "github.com/swiftdiaries/openharness/harness"`
3. Change compile-time checks: e.g. `var _ AgentStore = (*LiteAgentStore)(nil)` → `var _ harness.AgentStore = (*LiteAgentStore)(nil)`
4. Qualify all type references from the harness package

- [ ] **Step 6: Write compile-time satisfaction tests for all Lite impls**

Create `openharness/harness/lite/satisfy_test.go`:

```go
package lite

import (
	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time interface satisfaction checks for all Lite implementations.
var (
	_ harness.HarnessRunner  = (*LiteRunner)(nil)
	_ harness.AgentStore     = (*LiteAgentStore)(nil)
	_ harness.SecretStore    = (*LiteSecretStore)(nil)
	_ harness.ArtifactStore  = (*LiteArtifactStore)(nil)
	_ harness.ToolRegistry   = (*LiteToolRegistry)(nil)
	_ harness.SkillStore     = (*LiteSkillStore)(nil)
	_ harness.ChannelRouter  = (*LiteChannelRouter)(nil)
)
```

- [ ] **Step 7: Run all tests**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./... -v
```

Expected: ALL PASS — both `harness/` and `harness/lite/` packages.

- [ ] **Step 8: Commit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git add harness/lite/
git commit -m "feat(harness/lite): extract all Lite implementations from ghostfin"
```

---

## Task 5: Copy existing tests from ghostfin

**Files:**
- Source: all `*_test.go` files from `ghostfin/desktop/pkg/harness/`
- Create: corresponding test files in `openharness/harness/` and `openharness/harness/lite/`

- [ ] **Step 1: List all test files in ghostfin harness**

```bash
ls /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/pkg/harness/*_test.go
```

- [ ] **Step 2: Copy and adapt harness-level tests**

Copy test files that test interfaces, types, RunnerRegistry, StoreBackedEventStream into `openharness/harness/`. These stay in `package harness` — no type qualification needed. Merge into the test files already created in Tasks 2-3 or create new ones.

- [ ] **Step 3: Copy and adapt Lite implementation tests**

Copy test files that test Lite implementations into `openharness/harness/lite/`. Apply the same transformation:
1. `package harness` → `package lite`  
2. Add `import "github.com/swiftdiaries/openharness/harness"`
3. Qualify harness types

Merge into the test files already created in Task 4 or create new ones.

- [ ] **Step 4: Run full test suite**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go test ./... -v -count=1
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git add harness/ 
git commit -m "test(harness): migrate existing tests from ghostfin"
```

---

## Task 6: Wire ghostfin to import from openharness

**Files:**
- Modify: `ghostfin/go.work`
- Modify: `ghostfin/desktop/go.mod`
- Modify: all files in `ghostfin/desktop/` that import `pkg/harness`

- [ ] **Step 1: Add openharness to ghostfin's go.work**

Edit `ghostfin/go.work` to add `../openharness`:

```go
go 1.25.0

use (
    ./desktop
    ../openharness
)
```

- [ ] **Step 2: Add openharness dependency to desktop/go.mod**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop
go get github.com/swiftdiaries/openharness@latest
```

If using go.work with local replace, instead add to `go.mod`:

```
require github.com/swiftdiaries/openharness v0.0.0
```

The `go.work` replace will handle resolution.

- [ ] **Step 3: Find all files importing the old harness path**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop
grep -r '"github.com/swiftdiaries/ghostfin/desktop/pkg/harness"' --include='*.go' -l
```

This lists every file that needs its import updated.

- [ ] **Step 4: Update import paths**

For each file found in Step 3, change:
- `"github.com/swiftdiaries/ghostfin/desktop/pkg/harness"` → `"github.com/swiftdiaries/openharness/harness"`

For files that reference Lite types directly (e.g., `harness.LiteRunner`), add:
- `lite "github.com/swiftdiaries/openharness/harness/lite"` 

And change references: `harness.LiteRunner` → `lite.LiteRunner`, `harness.NewLiteRunner` → `lite.NewLiteRunner`, etc.

- [ ] **Step 5: Verify ghostfin compiles**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop
go build ./... 2>&1 | head -20
```

Fix any remaining import issues. Common issues:
- Lite types now in `lite` package, not `harness`
- `harness.LoopFactory` → `lite.LoopFactory` (defined in lite/runner.go)

- [ ] **Step 6: Run ghostfin tests**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop
go test ./... -tags sqliteonly -v 2>&1 | tail -30
```

Expected: ALL PASS — same behavior, different import source.

- [ ] **Step 7: Commit in ghostfin**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin
git add go.work desktop/go.mod desktop/go.sum
git add -u  # staged modified files with updated imports
git commit -m "refactor: import harness from openharness instead of local pkg/harness"
```

---

## Task 7: Remove old harness package from ghostfin

**Files:**
- Delete: `ghostfin/desktop/pkg/harness/` (entire directory)

- [ ] **Step 1: Verify no remaining references to old path**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop
grep -r 'pkg/harness' --include='*.go' -l
```

Expected: no results (all imports updated in Task 6).

- [ ] **Step 2: Delete the old harness package**

```bash
rm -rf /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/pkg/harness/
```

- [ ] **Step 3: Verify ghostfin still compiles and tests pass**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop
go build ./... && go test ./... -tags sqliteonly -v 2>&1 | tail -20
```

Expected: ALL PASS

- [ ] **Step 4: Verify the desktop app starts**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop
wails dev -tags sqliteonly
```

Expected: App starts, gateway on 127.0.0.1:18790, agent chat works. Ctrl+C to stop.

- [ ] **Step 5: Commit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin
git add -A
git commit -m "chore: remove old pkg/harness (now in openharness)"
```

---

## Task 8: Update ghostfin-enterprise to import from openharness

**Files:**
- Modify: `ghostfin-enterprise/go.mod`

- [ ] **Step 1: Check current ghostfin-enterprise imports**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin-enterprise
grep -r 'ghostfin/desktop/pkg/harness' --include='*.go' -l
```

- [ ] **Step 2: Update go.mod to add openharness dependency**

Add to `ghostfin-enterprise/go.mod`:

```
require github.com/swiftdiaries/openharness v0.0.0
```

Add `replace` directive for local development:

```
replace github.com/swiftdiaries/openharness => ../openharness
```

- [ ] **Step 3: Update import paths in enterprise code**

Change all imports from `github.com/swiftdiaries/ghostfin/desktop/pkg/harness` → `github.com/swiftdiaries/openharness/harness`.

- [ ] **Step 4: Verify enterprise compiles**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin-enterprise
go build ./...
```

Expected: Compiles successfully.

- [ ] **Step 5: Run enterprise tests**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin-enterprise
go test ./... -v
```

Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin-enterprise
git add go.mod go.sum
git add -u
git commit -m "refactor: import harness from openharness"
```

---

## Verification (Layer 1 Complete)

After all 8 tasks:

1. **openharness** — `go test ./... -v` → all pass
2. **ghostfin** — `cd desktop && go test ./... -tags sqliteonly -v` → all pass
3. **ghostfin** — `cd desktop && wails dev -tags sqliteonly` → app starts, agent chat works
4. **ghostfin-enterprise** — `go test ./... -v` → all pass
5. **No references** to old path: `grep -r 'pkg/harness' ghostfin/desktop/ --include='*.go'` → empty

---

## Roadmap: Layers 2-5

After Layer 1 lands, each subsequent layer gets its own detailed plan. Here's the file mapping for planning:

### Layer 2: Agent Primitives

| Source (ghostfin/desktop) | Destination (openharness) | Lines | Dependencies |
|---|---|---|---|
| `internal/agent/loop.go` | `agent/loop.go` | 1,041 | tools, providers, cost, traces, bootstrap, telemetry, notes |
| `internal/agent/tool_loop.go` | `agent/detection.go` | 265 | tools, traces |
| `internal/agent/systemprompt.go` | `agent/systemprompt.go` | 312 | bootstrap |
| `internal/agent/compact.go` | `agent/compact.go` | 156 | — |
| `internal/agent/pruning.go` | `agent/pruning.go` | 90 | — |
| `internal/agent/retrieval.go` | `agent/retrieval.go` | 81 | notes |
| `internal/agent/input_guard.go` | `agent/input_guard.go` | 64 | — |
| `internal/agent/mode.go` | `agent/mode.go` | 88 | — |
| `internal/agent/events.go` | `agent/events.go` | 36 | — |
| `internal/agent/agent.go` | stays in ghostfin (domain-specific agent defs) | 137 | — |
| `internal/tools/tool.go` | `tools/tool.go` | 17 | — |
| `internal/tools/registry.go` | `tools/registry.go` | 115 | — |
| `internal/tools/filesystem.go` | `tools/builtin/filesystem.go` | 258 | — |
| `internal/tools/web_search.go` | `tools/builtin/web_search.go` | 142 | — |
| `internal/tools/web_fetch.go` | `tools/builtin/web_fetch.go` | 112 | — |
| `internal/tools/exec.go` | `tools/builtin/exec.go` | 142 | — |
| `internal/tools/memory.go` | `tools/builtin/memory.go` | 207 | — |
| `internal/tools/tasks.go` | `tools/builtin/tasks.go` | 212 | — |
| `internal/tools/ask_user.go` | `tools/builtin/ask_user.go` | 42 | — |
| `internal/tools/knowledge_graph.go` | `tools/builtin/knowledge_graph.go` | 310 | notes |
| `internal/tools/ssrf.go` | `tools/security/ssrf.go` | 87 | — |
| `internal/tools/shell_deny.go` | `tools/security/shell_deny.go` | 37 | — |
| `internal/tools/scrub.go` | `tools/security/scrub.go` | 37 | — |
| `internal/tools/composite.go` | `tools/composite.go` | 36 | — |
| `internal/tools/builtin.go` | `tools/builtin/register.go` | 42 | — |
| `internal/tools/ghostfin_api.go` | stays in ghostfin (domain-specific) | 238 | — |
| `internal/tools/addepar_api.go` | stays in ghostfin (domain-specific) | 236 | — |
| `internal/tools/tamarac_api.go` | stays in ghostfin (domain-specific) | 187 | — |
| `internal/tools/subagent.go` | `tools/builtin/subagent.go` (refactor circular dep) | 60 | agent |
| `internal/tools/schedule_cron.go` | `tools/builtin/schedule_cron.go` | 154 | — |
| `internal/providers/types.go` | `providers/types.go` | 123 | — |
| `internal/providers/openai_compat.go` | `providers/openai_compat.go` | 200 | — |
| `internal/providers/models.go` | `providers/models.go` | 80 | — |
| `internal/providers/registry.go` | `providers/registry.go` | 100 | — |
| `internal/providers/openrouter.go` | `providers/openrouter.go` | 150 | — |
| `internal/mcp/manager.go` | `tools/mcp/manager.go` | 150 | config, tools |
| `internal/mcp/client.go` | `tools/mcp/client.go` | 120 | — |
| `internal/mcp/bridge_tool.go` | `tools/mcp/bridge_tool.go` | varies | tools |
| `internal/mcp/bm25_index.go` | `tools/mcp/bm25_index.go` | 80 | — |
| `internal/mcp/oauth.go` | `tools/mcp/oauth.go` | 80 | — |

**Note:** `internal/tools/subagent.go` has a circular dependency with `internal/agent/`. This must be refactored during Layer 2 — inject the agent runner as a callback function into the tool, breaking the import cycle.

### Layer 3: Infrastructure

| Source | Destination | Lines |
|---|---|---|
| `internal/cost/tracker.go` | `cost/tracker.go` | ~100 |
| `internal/cost/pricing.go` | `cost/pricing.go` | ~100 |
| `internal/telemetry/otel.go` | `telemetry/tracer.go` | ~80 |
| `internal/traces/store.go` | `telemetry/lite/sqlite_metadata.go` (refactored) | ~150 |
| `internal/traces/pg_store.go` | `telemetry/enterprise/pg_metadata.go` (refactored) | ~100 |
| `internal/traces/types.go` | `telemetry/types.go` | ~50 |
| New file | `telemetry/metadata.go` (TraceMetadataStore interface) | ~40 |
| New file | `telemetry/spans.go` (SpanSink interface) | ~30 |
| New file | `telemetry/lite/duckdb_sink.go` | ~100 |
| New file | `telemetry/lite/parquet.go` | ~80 |
| New file | `telemetry/enterprise/otlp_sink.go` | ~80 |
| `internal/sessions/store.go` | `sessions/store.go` | ~100 |
| `internal/sessions/types.go` | `sessions/types.go` | ~50 |
| `internal/config/config.go` | `config/config.go` | ~100 |
| `internal/config/discovery.go` | `config/discovery.go` | varies |
| `internal/edition/edition.go` | `edition/edition.go` | ~100 |
| `internal/scheduler/*.go` | `scheduler/*.go` | ~400 |

### Layer 4: App Scaffold

| Source | Destination | Lines |
|---|---|---|
| `internal/store/interfaces.go` | `store/interfaces.go` | ~98 |
| `internal/store/stores.go` | `store/stores.go` | ~28 |
| `internal/notes/*.go` | `notes/*.go` | ~600 |
| `internal/bootstrap/*.go` | `bootstrap/*.go` | ~200 |
| `internal/projects/*.go` | `projects/*.go` | ~200 |
| `internal/channels/*.go` | `channels/*.go` | ~300 |
| `internal/runs/*.go` | `runs/*.go` | ~100 |
| `internal/hooks/*.go` | `hooks/*.go` | ~50 |
| `internal/gateway/*.go` | `gateway/*.go` | ~800 |
| `app.go` patterns | `app/app.go` | new |
| `main.go` patterns | `app/main.go` | new |
| `frontend/` | `app/frontend/` | varies |

### Layer 5: Enterprise Implementations

| File | Destination | Notes |
|---|---|---|
| New | `harness/enterprise/session_pg.go` | Postgres SessionStore |
| New | `harness/enterprise/agent_pg.go` | Postgres AgentStore |
| New | `harness/enterprise/skill_pg.go` | Postgres + S3 SkillStore |
| New | `harness/enterprise/registry_pg.go` | Postgres ToolRegistry |
| New | `harness/enterprise/secret_vault.go` | Vault AppRole SecretStore |
| New | `harness/enterprise/artifact_s3.go` | S3/MinIO ArtifactStore |
| New | `harness/enterprise/runner_k8s.go` | K8s Job HarnessRunner |
| New | `harness/enterprise/runner_lambda.go` | Lambda S3-triggered runner |
| New | `harness/enterprise/snapshot.go` | Snapshot for Lambda chain |
| Move from ghostfin-enterprise | `harness/enterprise/testutil/pg.go` | Testcontainers Postgres |
| Move from ghostfin-enterprise | `harness/enterprise/testutil/vault.go` | Testcontainers Vault |
| Move from ghostfin-enterprise | `harness/enterprise/testutil/minio.go` | Testcontainers MinIO |
