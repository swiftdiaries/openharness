# Layer 2 Plan 3 — Tools: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

## Pipeline Tracking

| Field | Value |
|-------|-------|
| Epic | `openharness-pol.2.3` |
| Status | in_progress |
| Worktree | `.worktrees/layer-2-plan-3-tools` (branch: `feat/layer-2-plan-3-tools`) |

| Task | Beads ID | Status |
|------|----------|--------|
| T1: Establish branch + skeleton | `openharness-0bx` | closed |
| T2: Tool interface + enums | `openharness-wl1` | closed |
| T3: Registry with effects-aware registration | `openharness-7z7` | closed |
| T4: Composite | `openharness-z1d` | closed |
| T5: External content guard | `openharness-cz1` | closed |
| T6: Security helpers (SSRF, shell_deny, scrub) | `openharness-vet` | closed |
| T7: web_search built-in | `openharness-bgy` | closed |
| T8: web_fetch built-in | `openharness-66i` | closed |
| T9: filesystem built-in | `openharness-im2` | closed |
| T10: exec built-in | `openharness-lc7` | closed |
| T11: ask_user built-in | `openharness-kb7` | closed |
| T12: tasks built-in | `openharness-bf1` | closed |
| T13: memory built-in (interface reconciliation) | `openharness-t89` | closed |
| T14: knowledge_graph types | `openharness-8y8` | closed |
| T15: knowledge_graph built-in | `openharness-npc` | closed |
| T16: subagent built-in | `openharness-h53` | closed |
| T17: core.Register + Config | `openharness-3an` | open |
| T18: Integration test — D4 matrix + contract assertions | `openharness-8tv` | open |
| T19 (V1): Dependency-direction verification | `openharness-fun` | open |
| T20 (V2): Full test suite green + vet clean | `openharness-7p5` | open |
| T21: Update Progress table in extraction spec | `openharness-2ft` | open |
| T22: Close beads issue + push | `openharness-zt6` | open |

---

**Goal:** Move ghostfin's generic tool system into `openharness/tools/` (core) and `openharness/tools/core/` (9 built-in implementations). Add `Effects` + `Visibility` fields on `ToolDefinition`, invert the `knowledge_graph` and `subagent` tools to vertical-agnostic local interfaces, reconcile the `memory` tool with `agent.MemoryEntry`, and register everything through a new `core.Register(r, cfg)` entrypoint with strict Effects validation.

**Architecture:** A thin `tools` package holds the generic `Tool` interface, `Registry` with a precomputed `EffectsFor` map, and the three security helpers (SSRF, shell denylist, secret scrubbing). A nested `tools/core` package holds 9 built-in tools that only import `tools` + `agent`. Knowledge-graph access uses a new rich `core.KnowledgeGraphStore` interface; subagent registration takes a `SubagentConfig` map so ghostfin's finance presets move out to the registration callsite.

**Tech Stack:** Go 1.25.0, stdlib + `golang.org/x/net/html` (web tools) + `github.com/google/uuid` (tasks), standard `testing` package.

**Spec:** `docs/superpowers/specs/2026-04-16-plan-3-tools-design.md`
**Parent:** `docs/superpowers/specs/2026-04-13-openharness-layer-2-agent-primitives-design.md`
**Beads:** `openharness-pol.2.3`
**Branch:** `feat/layer-2-plan-3-tools`

---

## Prerequisites

- Plan 1 is merged: `openharness/agent/` contains `MemoryEntry{Key, Value, UpdatedAt time.Time}`, `Block`, `MemoryStore{Search(ctx, query, limit) ([]MemoryEntry, error)}`, `KnowledgeStore{Search, Get}`.
- Plan 2 is merged: `openharness/providers/` exists; Plan 3 does not import it.
- Ghostfin source at `/Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/` is the copy source. No edits happen in ghostfin during this plan — migration of ghostfin's own tree lands later in Plan 7.
- `golang.org/x/net` and `github.com/google/uuid` are added to `openharness/go.mod` as new direct dependencies when the first file that needs them lands (web_search/web_fetch for `x/net/html`, tasks for `uuid`).

---

## File Structure

New tree under `openharness/tools/`:

```
openharness/tools/
├── tool.go                 # Tool + ToolDefinition + ToolEffect + ToolVisibility
├── tool_test.go            # Enum zero-value + serialization sanity
├── registry.go             # Registry + Register (validates Effects) + EffectsFor
├── composite.go            # Multi-tool wrapper (verbatim from ghostfin)
├── composite_test.go       # verbatim
├── external_content.go     # verbatim
├── ssrf.go                 # verbatim
├── ssrf_test.go            # verbatim
├── shell_deny.go           # verbatim
├── shell_deny_test.go      # verbatim
├── scrub.go                # verbatim
├── scrub_test.go           # verbatim
└── core/
    ├── core.go                     # Config + Register(r, cfg) + Effects validation
    ├── core_test.go                # D4 matrix + omission + unknown-name contract
    ├── web_search.go               # Effects: Read
    ├── web_fetch.go                # Effects: Read
    ├── filesystem.go               # per-def Effects (Read/Mutate)
    ├── filesystem_test.go
    ├── exec.go                     # Effects: Neutral
    ├── exec_test.go
    ├── memory.go                   # implements agent.MemoryStore; per-def Effects
    ├── memory_test.go              # NEW
    ├── ask_user.go                 # Effects: Interactive
    ├── ask_user_test.go
    ├── tasks.go                    # sqliteonly tag stripped
    ├── tasks_test.go
    ├── knowledge_graph.go          # depends on local KnowledgeGraphStore interface
    ├── knowledge_graph_types.go    # KGPage / KGBlock / KGRef (flat, full field set)
    └── subagent.go                 # SubagentConfig + dynamic JSON schema enum
```

`tools` has no test for `registry.go` — `EffectsFor` coverage lives in `core/core_test.go` where the classification matrix is in scope.

**Dependency direction** (enforced by verification in Task V2):
- `openharness/tools` → stdlib + `golang.org/x/net/html` (the three HTML-parsing files)
- `openharness/tools/core` → `openharness/tools` + `openharness/agent` + `github.com/google/uuid`
- Neither may import `openharness/providers`, `openharness/sessions`, `openharness/cost`, or anything under `github.com/swiftdiaries/ghostfin`.

---

## Task Sequencing Map

| Wave | Tasks | Parallelism |
|---|---|---|
| A — Core foundation | T1–T6 | sequential (each depends on prior) |
| B — Security helpers | T7–T10 | independent after T1; runnable in parallel |
| C — Simple built-ins | T11–T16 | all depend on T2 + T7–T10; runnable in parallel |
| D — Inverted built-ins | T17–T19 | sequential (memory → knowledge_graph → subagent) |
| E — Registration + verification | T20–T22 | sequential |

Total: 22 tasks. Every task ends with a `go test` + `git commit` pair.

---

## Task 1: Establish branch + skeleton

**Files:**
- Create: `openharness/tools/` (directory)
- Create: `openharness/tools/core/` (directory)
- Modify: `openharness/go.mod` (add direct dependencies)

- [ ] **Step 1.1: Create the feature branch**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
git checkout -b feat/layer-2-plan-3-tools
```

- [ ] **Step 1.2: Create empty package directories**

```bash
mkdir -p openharness/tools/core
```

- [ ] **Step 1.3: Add new Go dependencies to go.mod**

Append under `require ( ... )` block in `go.mod`:

```go
require (
    github.com/anthropics/anthropic-sdk-go v1.35.1
    github.com/google/uuid v1.6.0
    go.opentelemetry.io/otel v1.43.0
    go.opentelemetry.io/otel/trace v1.43.0
    golang.org/x/net v0.38.0
)
```

Then:

```bash
go mod tidy
```

Expected: go.sum updates with `golang.org/x/net` + transitive `golang.org/x/text`. `uuid` is already in go.sum via indirect.

- [ ] **Step 1.4: Commit the skeleton**

```bash
git add go.mod go.sum
git commit -m "tools: add golang.org/x/net direct dep for Plan 3

Plan 3 moves web_search/web_fetch/external_content into openharness/tools
which all import golang.org/x/net/html. Promote from indirect to direct."
```

---

## Task 2: Tool interface + enums (`tools/tool.go`)

**Files:**
- Create: `openharness/tools/tool.go`
- Create: `openharness/tools/tool_test.go`

- [ ] **Step 2.1: Write the failing enum-zero-value test**

`openharness/tools/tool_test.go`:

```go
package tools

import (
	"encoding/json"
	"testing"
)

func TestToolEffectZeroValueIsUnspecified(t *testing.T) {
	var e ToolEffect
	if e != ToolEffectUnspecified {
		t.Fatalf("zero value of ToolEffect should be ToolEffectUnspecified, got %d", int(e))
	}
}

func TestToolVisibilityZeroValueIsDefault(t *testing.T) {
	var v ToolVisibility
	if v != ToolVisibilityDefault {
		t.Fatalf("zero value of ToolVisibility should be ToolVisibilityDefault, got %d", int(v))
	}
}

func TestToolDefinitionEffectsNotSerialized(t *testing.T) {
	def := ToolDefinition{
		Name:        "example",
		Description: "x",
		Parameters:  json.RawMessage(`{}`),
		Effects:     ToolEffectMutate,
		Visibility:  ToolVisibilityModelOnly,
	}
	data, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if containsAny(s, []string{"Effects", "Visibility", "effects", "visibility"}) {
		t.Fatalf("Effects/Visibility must not appear in JSON output, got: %s", s)
	}
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		for i := 0; i+len(n) <= len(s); i++ {
			if s[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 2.2: Run the test to verify it fails**

Run: `go test ./tools/...`
Expected: FAIL, package does not compile (`ToolEffect` etc. undefined).

- [ ] **Step 2.3: Write `tools/tool.go` to make it pass**

`openharness/tools/tool.go`:

```go
// Package tools defines the generic tool system used by the openharness
// agent loop. It ships:
//   - the Tool interface and ToolDefinition struct
//   - a Registry with effects-aware lookup
//   - three security helpers (SSRF validation, shell denylist, secret scrubbing)
//
// Built-in tool implementations live in the nested tools/core package;
// framework consumers (MCP, vertical-specific tools) implement the Tool
// interface directly.
package tools

import (
	"context"
	"encoding/json"
)

// ToolEffect classifies a ToolDefinition for loop-detection and mode filtering.
// The agent loop (Plan 5) reads Effects to decide whether a call resets or
// increments the read-streak, and whether the definition is allowed in
// ModePlan. Every definition registered with a Registry MUST carry a
// non-Unspecified Effects value — Register returns an error otherwise.
type ToolEffect int

const (
	// ToolEffectUnspecified is the zero value. Registry.Register treats it
	// as a classification error and refuses to register the tool. Plan 5's
	// loop treats it as Mutate at runtime as a defensive fallback.
	ToolEffectUnspecified ToolEffect = iota
	// ToolEffectRead increments the read-streak counter and is allowed in
	// ModePlan. Reads never mutate the workspace, memory, or knowledge graph.
	ToolEffectRead
	// ToolEffectMutate resets the read-streak counter and is blocked in
	// ModePlan. Writes, deletions, and sub-agent delegations are Mutate.
	ToolEffectMutate
	// ToolEffectNeutral neither resets nor increments the read-streak and
	// is allowed in ModePlan. This preserves ghostfin's tool_loop.go:161
	// behavior for `exec`, whose real gate is the Layer 6 SDK admission
	// rule, not the Plan 3 mode filter.
	ToolEffectNeutral
	// ToolEffectInteractive resets the read-streak (a user reply typically
	// redirects the task) and is allowed in ModePlan (asking for
	// clarification is non-destructive).
	ToolEffectInteractive
)

// ToolVisibility reserves space for UIBridge-facing semantics wired in Plan 4.
// Plan 3 declares the enum but does not act on it; Plan 4 decides which
// definitions render in the app UI vs. stay hidden from the LLM.
type ToolVisibility int

const (
	ToolVisibilityDefault   ToolVisibility = iota // visible to both LLM and app UI
	ToolVisibilityModelOnly                       // LLM sees it; app UI does not
	ToolVisibilityAppOnly                         // app UI surfaces it; LLM does not
)

// ToolDefinition is a single callable surface a Tool exposes to the LLM. A
// Tool may publish several definitions (e.g. the filesystem tool publishes
// read_file, write_file, list_files, edit_file). Effects and Visibility are
// per-definition, never per-Tool, because tools commonly mix read and
// mutate surfaces.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	// Effects and Visibility are framework-internal metadata and MUST NOT
	// be serialized to the LLM — the `json:"-"` tag is load-bearing.
	Effects    ToolEffect     `json:"-"`
	Visibility ToolVisibility `json:"-"`
}

// Tool is the handler for one or more ToolDefinitions. Execute is dispatched
// by name; implementations typically switch on name and delegate to private
// per-definition handlers.
type Tool interface {
	Definitions() []ToolDefinition
	Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error)
}
```

- [ ] **Step 2.4: Run the tests to verify they pass**

Run: `go test ./tools/...`
Expected: PASS (3 tests in `tools`).

- [ ] **Step 2.5: Commit**

```bash
git add tools/tool.go tools/tool_test.go
git commit -m "tools: add Tool interface + ToolEffect/ToolVisibility enums

Ships the 5-state ToolEffect enum (Unspecified as zero value, Read, Mutate,
Neutral, Interactive) and the 3-state ToolVisibility enum. Both are marked
json:\"-\" so they never leak to the LLM."
```

---

## Task 3: Registry with effects-aware registration (`tools/registry.go`)

**Files:**
- Create: `openharness/tools/registry.go`

No dedicated `registry_test.go` — `EffectsFor`, validation errors, and unknown-name semantics are asserted from `core/core_test.go` in Task 20 where the D4 matrix and real tools live together.

- [ ] **Step 3.1: Write `tools/registry.go`**

Copy from `ghostfin/desktop/internal/tools/registry.go` with these changes:
1. Strip `//go:build sqliteonly` header.
2. Add precomputed `effects map[string]ToolEffect`.
3. Make `Register` return `error` and validate every definition's `Effects`.
4. Add `EffectsFor(name) ToolEffect`.

Full contents of `openharness/tools/registry.go`:

```go
package tools

import (
	"fmt"
	"sort"
)

// ToolCategory groups tools under a named category. Retained from ghostfin
// for Plan 4 (MCP) compatibility; Plan 3 callers do not use it yet.
type ToolCategory struct {
	Name  string           `json:"name"`
	Count int              `json:"count"`
	Tools []ToolDefinition `json:"tools"`
}

// Registry holds registered tools and their category/effects mappings.
// Registration is Effects-aware: every definition must carry a
// non-Unspecified Effects value or Register returns an error. Effects are
// precomputed into an O(1) map because Plan 5's loop-detection path
// calls EffectsFor on every tool invocation.
type Registry struct {
	tools             map[string]Tool
	categories        map[string]string    // def name -> category
	effects           map[string]ToolEffect // def name -> effects
	deferredActivator func(name string) bool
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:      make(map[string]Tool),
		categories: make(map[string]string),
		effects:    make(map[string]ToolEffect),
	}
}

// Register adds a tool under the given category. Every definition returned
// by t.Definitions() must carry a non-Unspecified Effects value, otherwise
// Register returns an error naming the offending definition and nothing is
// added to the registry. If a later definition fails validation, earlier
// definitions in the same call are rolled back.
func (r *Registry) Register(category string, t Tool) error {
	defs := t.Definitions()
	for _, def := range defs {
		if def.Effects == ToolEffectUnspecified {
			return fmt.Errorf("tools: definition %q has Effects: ToolEffectUnspecified; every registered definition must classify its effects", def.Name)
		}
	}
	for _, def := range defs {
		r.tools[def.Name] = t
		r.categories[def.Name] = category
		r.effects[def.Name] = def.Effects
	}
	return nil
}

// Get returns the Tool that handles the given tool name, or nil.
func (r *Registry) Get(name string) Tool {
	return r.tools[name]
}

// EffectsFor returns the precomputed ToolEffect for a definition name.
// Unknown names return ToolEffectUnspecified with no error — the caller
// decides how to handle unclassified tools.
func (r *Registry) EffectsFor(name string) ToolEffect {
	return r.effects[name]
}

// Unregister removes a tool by name from the registry.
func (r *Registry) Unregister(name string) {
	delete(r.categories, name)
	delete(r.tools, name)
	delete(r.effects, name)
}

// Categories returns all categories with their tool counts and definitions.
func (r *Registry) Categories() []ToolCategory {
	catTools := make(map[string][]ToolDefinition)
	for name, cat := range r.categories {
		t := r.tools[name]
		for _, def := range t.Definitions() {
			if def.Name == name {
				catTools[cat] = append(catTools[cat], def)
			}
		}
	}

	var result []ToolCategory
	for cat, defs := range catTools {
		result = append(result, ToolCategory{Name: cat, Count: len(defs), Tools: defs})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// ToolsByCategory returns all Tool implementations in the given category.
func (r *Registry) ToolsByCategory(cat string) []Tool {
	seen := make(map[Tool]bool)
	var result []Tool
	for name, c := range r.categories {
		if c == cat {
			t := r.tools[name]
			if !seen[t] {
				seen[t] = true
				result = append(result, t)
			}
		}
	}
	return result
}

// SetDeferredActivator sets a callback that can lazily activate deferred tools by name.
func (r *Registry) SetDeferredActivator(fn func(name string) bool) {
	r.deferredActivator = fn
}

// TryActivateDeferred attempts to activate a deferred tool by name.
// Returns the tool if activation succeeded, nil otherwise.
func (r *Registry) TryActivateDeferred(name string) Tool {
	if r.deferredActivator != nil && r.deferredActivator(name) {
		return r.tools[name]
	}
	return nil
}

// ToolsByNames returns the Tools wrapping the named definitions, deduplicated.
func (r *Registry) ToolsByNames(names ...string) []Tool {
	seen := make(map[Tool]bool)
	var result []Tool
	for _, name := range names {
		if t, ok := r.tools[name]; ok && !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}
```

- [ ] **Step 3.2: Verify it compiles cleanly**

Run: `go build ./tools/...`
Expected: Success, no output.

- [ ] **Step 3.3: Commit**

```bash
git add tools/registry.go
git commit -m "tools: port Registry with Effects-aware Register + EffectsFor

Register now validates every definition has Effects != Unspecified and
returns an error if any does not. Adds precomputed effects map for O(1)
EffectsFor lookups in Plan 5's loop-detection hot path. RegisterTool
removed (category-only callsites can re-add later; none exist now)."
```

---

## Task 4: Composite (`tools/composite.go`)

**Files:**
- Create: `openharness/tools/composite.go`
- Create: `openharness/tools/composite_test.go`

Both copied verbatim from ghostfin (source: `ghostfin/desktop/internal/tools/composite.go` + `composite_test.go`). Composite does not touch Effects — it only dispatches by name.

- [ ] **Step 4.1: Copy `composite.go` verbatim**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/composite.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/composite.go
```

Verify the file header says `package tools`. No edits needed.

- [ ] **Step 4.2: Copy `composite_test.go` verbatim**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/composite_test.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/composite_test.go
```

- [ ] **Step 4.3: Run the tests**

Run: `go test ./tools/...`
Expected: PASS. Previous `tool_test.go` tests still green; new composite tests green.

- [ ] **Step 4.4: Commit**

```bash
git add tools/composite.go tools/composite_test.go
git commit -m "tools: port Composite verbatim from ghostfin"
```

---

## Task 5: External content guard (`tools/external_content.go`)

**Files:**
- Create: `openharness/tools/external_content.go`

No existing test in ghostfin. Copy verbatim.

- [ ] **Step 5.1: Copy the file**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/external_content.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/external_content.go
```

- [ ] **Step 5.2: Build + commit**

```bash
go build ./tools/...
git add tools/external_content.go
git commit -m "tools: port WrapExternalContent helper verbatim"
```

---

## Task 6: Security helpers — SSRF, shell_deny, scrub (`tools/*.go`)

**Files:**
- Create: `openharness/tools/ssrf.go`, `ssrf_test.go`
- Create: `openharness/tools/shell_deny.go`, `shell_deny_test.go`
- Create: `openharness/tools/scrub.go`, `scrub_test.go`

All six files copy verbatim. None touch the tool system; they exist here so Plan 4 (MCP) and Plan 5 (agent loop) don't have to reach into ghostfin.

- [ ] **Step 6.1: Copy the six files verbatim**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools
for f in ssrf.go ssrf_test.go shell_deny.go shell_deny_test.go scrub.go scrub_test.go; do
  cp "$f" /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/"$f"
done
```

- [ ] **Step 6.2: Run tests**

Run: `go test ./tools/...`
Expected: PASS. All copied tests green.

- [ ] **Step 6.3: Commit**

```bash
git add tools/ssrf.go tools/ssrf_test.go tools/shell_deny.go tools/shell_deny_test.go tools/scrub.go tools/scrub_test.go
git commit -m "tools: port SSRF + shell_deny + scrub helpers verbatim"
```

---

## Task 7: web_search built-in (`tools/core/web_search.go`)

**Files:**
- Create: `openharness/tools/core/web_search.go`

No existing test in ghostfin. `web_search` makes a network call — we add a small definitions-only test for coverage but do not hit the network in CI.

- [ ] **Step 7.1: Copy `web_search.go`**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/web_search.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/web_search.go
```

- [ ] **Step 7.2: Edit package declaration + imports + Effects**

Change the first two lines:

```go
package tools
```

to:

```go
package core

import "github.com/swiftdiaries/openharness/tools"
```

(You'll merge the new import with the existing import block — the existing file already has an `import ( ... )` block, so insert `"github.com/swiftdiaries/openharness/tools"` into that block rather than adding a second `import` statement.)

Then update `Definitions()` to set Effects. Replace the existing `Definitions` body:

```go
func (w *WebSearch) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "web_search",
			Description: "Search the web using DuckDuckGo. Returns titles, URLs, and snippets.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query"},
					"limit": {"type": "integer", "description": "Max results (default 5)"}
				},
				"required": ["query"]
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}
```

- [ ] **Step 7.3: Write a definitions-only test**

Create `openharness/tools/core/web_search_test.go`:

```go
package core

import (
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestWebSearchDefinitions(t *testing.T) {
	w := NewWebSearch()
	defs := w.Definitions()
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "web_search" {
		t.Errorf("name = %q, want web_search", defs[0].Name)
	}
	if defs[0].Effects != tools.ToolEffectRead {
		t.Errorf("Effects = %v, want Read", defs[0].Effects)
	}
}
```

- [ ] **Step 7.4: Build + test**

Run: `go test ./tools/core/...`
Expected: PASS.

- [ ] **Step 7.5: Commit**

```bash
git add tools/core/web_search.go tools/core/web_search_test.go
git commit -m "tools/core: port web_search with Effects: Read"
```

---

## Task 8: web_fetch built-in (`tools/core/web_fetch.go`)

**Files:**
- Create: `openharness/tools/core/web_fetch.go`, `web_fetch_test.go`

- [ ] **Step 8.1: Copy + re-package**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/web_fetch.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/web_fetch.go
```

Change `package tools` → `package core`. Add import `"github.com/swiftdiaries/openharness/tools"`.

Change the `Definitions()` return type to `[]tools.ToolDefinition` and each literal to `tools.ToolDefinition{...}`. Set `Effects: tools.ToolEffectRead` on every definition returned.

Any unqualified reference to `WrapExternalContent` inside `web_fetch.go` must become `tools.WrapExternalContent` (it was moved in Task 5). Same for `ScrubCredentials` (Task 6) and any SSRF helpers imported from `ssrf.go` — prefix with `tools.`.

- [ ] **Step 8.2: Write a definitions-only test**

`openharness/tools/core/web_fetch_test.go`:

```go
package core

import (
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestWebFetchDefinitions(t *testing.T) {
	w := NewWebFetch()
	defs := w.Definitions()
	if len(defs) == 0 {
		t.Fatal("want at least 1 definition, got 0")
	}
	for _, d := range defs {
		if d.Effects != tools.ToolEffectRead {
			t.Errorf("%s: Effects = %v, want Read", d.Name, d.Effects)
		}
	}
}
```

- [ ] **Step 8.3: Build + test**

Run: `go test ./tools/core/...`
Expected: PASS.

- [ ] **Step 8.4: Commit**

```bash
git add tools/core/web_fetch.go tools/core/web_fetch_test.go
git commit -m "tools/core: port web_fetch with Effects: Read

WrapExternalContent + ScrubCredentials references updated to point at
the tools package where they were moved in Tasks 5 & 6."
```

---

## Task 9: filesystem built-in (`tools/core/filesystem.go`)

**Files:**
- Create: `openharness/tools/core/filesystem.go`, `filesystem_test.go`

Ghostfin's filesystem publishes four definitions: `read_file`, `write_file`, `list_files`, `edit_file`. Effects per the D4 table:
- `read_file`, `list_files` → `ToolEffectRead`
- `write_file`, `edit_file` → `ToolEffectMutate`

(The spec's D4 table mentions `search_files`, `delete_file`, `make_directory` — these do not exist in ghostfin's current filesystem tool. Only classify the four definitions that actually exist. Effects for the speculative three are out of scope.)

- [ ] **Step 9.1: Copy + re-package**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/filesystem.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/filesystem.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/filesystem_test.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/filesystem_test.go
```

Change `package tools` → `package core` in both files. Add `"github.com/swiftdiaries/openharness/tools"` import. Change `ToolDefinition` → `tools.ToolDefinition`.

- [ ] **Step 9.2: Add Effects to each definition**

In `filesystem.go` Definitions(), annotate as:

```go
func (f *Filesystem) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read the contents of a file in the workspace.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectRead,
		},
		{
			Name:        "write_file",
			Description: "Write content to a file in the workspace.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectMutate,
		},
		{
			Name:        "list_files",
			Description: "List files and directories at a path in the workspace.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectRead,
		},
		{
			Name:        "edit_file",
			Description: "Replace a substring in a file within the workspace.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectMutate,
		},
	}
}
```

(The body JSON blobs are identical to ghostfin's; only annotate each literal with Effects.)

- [ ] **Step 9.3: Append an Effects assertion to `filesystem_test.go`**

Add at the end of `filesystem_test.go`:

```go
func TestFilesystemEffects(t *testing.T) {
	f := NewFilesystem(t.TempDir())
	want := map[string]tools.ToolEffect{
		"read_file":  tools.ToolEffectRead,
		"write_file": tools.ToolEffectMutate,
		"list_files": tools.ToolEffectRead,
		"edit_file":  tools.ToolEffectMutate,
	}
	for _, d := range f.Definitions() {
		if got := d.Effects; got != want[d.Name] {
			t.Errorf("%s: Effects = %v, want %v", d.Name, got, want[d.Name])
		}
	}
}
```

- [ ] **Step 9.4: Run tests**

Run: `go test ./tools/core/...`
Expected: PASS (existing filesystem tests + new Effects test).

- [ ] **Step 9.5: Commit**

```bash
git add tools/core/filesystem.go tools/core/filesystem_test.go
git commit -m "tools/core: port filesystem with per-def Read/Mutate Effects"
```

---

## Task 10: exec built-in (`tools/core/exec.go`)

**Files:**
- Create: `openharness/tools/core/exec.go`, `exec_test.go`

Per D4: `exec` is `ToolEffectNeutral` — preserves ghostfin's `tool_loop.go:161` behavior. Layer 6 SDK admission is its real gate.

- [ ] **Step 10.1: Copy + re-package**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/exec.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/exec.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/exec_test.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/exec_test.go
```

Change `package tools` → `package core`. Add import `"github.com/swiftdiaries/openharness/tools"`.

Note: `exec.go` uses the `matchesDenyPattern` helper from `shell_deny.go` (now in the `tools` package). Prefix that reference as `tools.MatchesDenyPattern` — check whether the helper is currently exported. If it is still lowercase (`matchesDenyPattern`) in `tools/shell_deny.go`, you must export it by renaming in that file plus its test file, committing as a separate small change BEFORE continuing this task. The edit is: rename `matchesDenyPattern` → `MatchesDenyPattern` throughout `tools/shell_deny.go` + `tools/shell_deny_test.go`, then `go test ./tools/...` green, then commit with message "tools: export MatchesDenyPattern for cross-package use in core/exec". Then return to this task.

- [ ] **Step 10.2: Annotate Definitions with Effects**

```go
func (e *Exec) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "exec",
			Description: "Execute a shell command in the workspace. Dangerous commands are blocked.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectNeutral,
		},
	}
}
```

- [ ] **Step 10.3: Append an Effects assertion to `exec_test.go`**

```go
func TestExecEffectsIsNeutral(t *testing.T) {
	e := NewExec(t.TempDir())
	defs := e.Definitions()
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if defs[0].Effects != tools.ToolEffectNeutral {
		t.Errorf("Effects = %v, want Neutral (preserves tool_loop.go:161 behavior)", defs[0].Effects)
	}
}
```

(Import `"github.com/swiftdiaries/openharness/tools"` in the test file if not present.)

- [ ] **Step 10.4: Run tests**

Run: `go test ./tools/core/...`
Expected: PASS.

- [ ] **Step 10.5: Commit**

```bash
git add tools/core/exec.go tools/core/exec_test.go
git commit -m "tools/core: port exec with Effects: Neutral

Neutral preserves ghostfin's tool_loop.go:161 behavior — exec neither
increments nor resets the read-streak. Layer 6 SDK admission rule is
the real gate."
```

---

## Task 11: ask_user built-in (`tools/core/ask_user.go`)

**Files:**
- Create: `openharness/tools/core/ask_user.go`, `ask_user_test.go`

Per D4: `ask_user` is `ToolEffectInteractive`. Definition name is `ask_user_question` (matching ghostfin's actual source, not the spec's shorthand).

- [ ] **Step 11.1: Copy + re-package**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/ask_user.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/ask_user.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/ask_user_test.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/ask_user_test.go
```

Change `package tools` → `package core`. Add import for `"github.com/swiftdiaries/openharness/tools"`. Change `[]ToolDefinition` → `[]tools.ToolDefinition`. Set `Effects: tools.ToolEffectInteractive` on the one definition.

- [ ] **Step 11.2: Append an Effects assertion**

```go
func TestAskUserEffectsIsInteractive(t *testing.T) {
	a := NewAskUser()
	defs := a.Definitions()
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if defs[0].Effects != tools.ToolEffectInteractive {
		t.Errorf("Effects = %v, want Interactive", defs[0].Effects)
	}
}
```

- [ ] **Step 11.3: Run tests**

Run: `go test ./tools/core/...`
Expected: PASS.

- [ ] **Step 11.4: Commit**

```bash
git add tools/core/ask_user.go tools/core/ask_user_test.go
git commit -m "tools/core: port ask_user with Effects: Interactive"
```

---

## Task 12: tasks built-in (`tools/core/tasks.go`)

**Files:**
- Create: `openharness/tools/core/tasks.go`, `tasks_test.go`

Ghostfin's `tasks.go` and `tasks_test.go` carry `//go:build sqliteonly` build tags. Per the File Inventory note in the spec, this tag is a ghostfin-wide feature-flag artifact, not a dependency gate — `tasks.go` uses `os.WriteFile` + `json.MarshalIndent`, no SQLite driver. Stripping the tag introduces zero new dependencies.

Per D4: `task_get`, `task_list` → `ToolEffectRead`; `task_create`, `task_update` → `ToolEffectMutate`.

- [ ] **Step 12.1: Copy + re-package**

```bash
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/tasks.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/tasks.go
cp /Users/adhita/projects/python/src/github.com/swiftdiaries/ghostfin/desktop/internal/tools/tasks_test.go \
   /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness/tools/core/tasks_test.go
```

Then:
1. Delete the two-line `//go:build sqliteonly` + blank line header from both files.
2. Change `package tools` → `package core`.
3. Add import `"github.com/swiftdiaries/openharness/tools"` (test file imports it too).
4. Replace `[]ToolDefinition` → `[]tools.ToolDefinition`.
5. Add Effects to each literal (see table below).

```go
func (t *TaskCRUD) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "task_create",
			Description: "Create a new task. Returns the task with its assigned ID.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectMutate,
		},
		{
			Name:        "task_update",
			Description: "Update an existing task's status, subject, description, or activeForm.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectMutate,
		},
		{
			Name:        "task_get",
			Description: "Get a single task by ID.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectRead,
		},
		{
			Name:        "task_list",
			Description: "List all tasks. Returns the full task list with current statuses.",
			Parameters:  json.RawMessage(`{ ... }`),
			Effects:     tools.ToolEffectRead,
		},
	}
}
```

- [ ] **Step 12.2: Append an Effects assertion**

```go
func TestTaskCRUDEffects(t *testing.T) {
	tc := NewTaskCRUD(t.TempDir())
	want := map[string]tools.ToolEffect{
		"task_create": tools.ToolEffectMutate,
		"task_update": tools.ToolEffectMutate,
		"task_get":    tools.ToolEffectRead,
		"task_list":   tools.ToolEffectRead,
	}
	for _, d := range tc.Definitions() {
		if got := d.Effects; got != want[d.Name] {
			t.Errorf("%s: Effects = %v, want %v", d.Name, got, want[d.Name])
		}
	}
}
```

- [ ] **Step 12.3: Run tests**

Run: `go test ./tools/core/...`
Expected: PASS (no build tag required in openharness).

- [ ] **Step 12.4: Commit**

```bash
git add tools/core/tasks.go tools/core/tasks_test.go
git commit -m "tools/core: port tasks with per-def Read/Mutate Effects

Strip the sqliteonly build tag — it's a ghostfin-wide feature-flag
artifact, not a dependency gate. tasks.go uses os.WriteFile/json.Marshal,
zero SQLite coupling."
```

---

## Task 13: memory built-in (`tools/core/memory.go`) — interface reconciliation

**Files:**
- Create: `openharness/tools/core/memory.go`
- Create: `openharness/tools/core/memory_test.go` (**new file** — no ghostfin equivalent)

This is the riskiest task. The memory tool must:
1. Replace the local `MemoryEntry` with `agent.MemoryEntry{Key, Value, UpdatedAt time.Time}`.
2. Rename `SearchEntries(query, limit)` → `Search(ctx, query, limit) ([]agent.MemoryEntry, error)` to satisfy `agent.MemoryStore`.
3. Drop `category` from the `memory_store` JSON schema (field does not exist on `agent.MemoryEntry`; silent acceptance would be confusing).
4. Remove `DefaultMemoryPath()` — caller always passes the path.
5. Preserve substring-match semantics across `Key` and `Value`.
6. Use the new on-disk format only (no legacy-format reader — migration of `~/.ghostfin/data/memory.json` lives in ghostfin Plan 7).

- [ ] **Step 13.1: Write `memory_test.go` first (test-driven)**

`openharness/tools/core/memory_test.go`:

```go
package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/swiftdiaries/openharness/agent"
	"github.com/swiftdiaries/openharness/tools"
)

// Compile-time assertion that *Memory satisfies agent.MemoryStore.
// If the interface grows or Search's signature drifts, this line
// won't compile and the test file will fail to build.
var _ agent.MemoryStore = (*Memory)(nil)

func newMemory(t *testing.T) *Memory {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory.json")
	return NewMemory(path)
}

func TestMemoryStoreAndSearch_RoundTrip(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()
	// Store via Execute so we exercise the JSON schema path.
	args, _ := json.Marshal(map[string]string{
		"key":   "fav_color",
		"value": "blue",
	})
	if _, err := m.Execute(ctx, "memory_store", args); err != nil {
		t.Fatal(err)
	}
	// New Memory on same file → Search still finds it.
	m2 := NewMemory(m.path)
	results, err := m2.Search(ctx, "blue", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Key != "fav_color" || results[0].Value != "blue" {
		t.Errorf("got %+v", results[0])
	}
	if results[0].UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set by memory_store")
	}
}

func TestMemoryOnDiskFormat(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()
	args, _ := json.Marshal(map[string]string{"key": "k", "value": "v"})
	if _, err := m.Execute(ctx, "memory_store", args); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(m.path)
	if err != nil {
		t.Fatal(err)
	}
	// On-disk format must be agent.MemoryEntry — Key/Value/UpdatedAt only.
	var decoded map[string]struct {
		Key       string `json:"Key"`
		Value     string `json:"Value"`
		UpdatedAt string `json:"UpdatedAt"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("on-disk file not decodable with agent.MemoryEntry shape: %v\nraw: %s", err, raw)
	}
	e, ok := decoded["k"]
	if !ok || e.Key != "k" || e.Value != "v" || e.UpdatedAt == "" {
		t.Errorf("decoded entry: %+v", e)
	}
}

func TestMemorySearchSubstringSemantics(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()

	// Populate via Execute.
	for _, kv := range [][2]string{
		{"fruit_apple", "red and round"},
		{"fruit_banana", "yellow and curved"},
		{"veg_carrot", "orange and long"},
		{"color_red", "rhymes with bed"},
	} {
		args, _ := json.Marshal(map[string]string{"key": kv[0], "value": kv[1]})
		if _, err := m.Execute(ctx, "memory_store", args); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		query string
		want  []string // keys expected in the result set (order-independent)
	}{
		{"red", []string{"fruit_apple", "color_red"}}, // matches Value "red and round" AND Key "color_red"
		{"APPLE", []string{"fruit_apple"}},             // case-insensitive against Key
		{"yellow", []string{"fruit_banana"}},            // matches Value
		{"fruit", []string{"fruit_apple", "fruit_banana"}},
		{"nomatch", nil},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			got, err := m.Search(ctx, c.query, 10)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("query %q: got %d results, want %d: %+v", c.query, len(got), len(c.want), got)
			}
			seen := map[string]bool{}
			for _, e := range got {
				seen[e.Key] = true
			}
			for _, k := range c.want {
				if !seen[k] {
					t.Errorf("query %q: expected key %q in results", c.query, k)
				}
			}
		})
	}
}

func TestMemorySearchHonorsLimit(t *testing.T) {
	m := newMemory(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		args, _ := json.Marshal(map[string]string{
			"key":   "k" + string(rune('0'+i)),
			"value": "hello world",
		})
		m.Execute(ctx, "memory_store", args)
	}
	got, err := m.Search(ctx, "hello", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("want 3 results, got %d", len(got))
	}
}

func TestMemoryExecuteRejectsCategoryParameter(t *testing.T) {
	// memory_store's JSON schema must NOT accept `category` — the field
	// does not exist on agent.MemoryEntry. Silently accepting and
	// discarding the parameter would be confusing.
	m := newMemory(t)
	defs := m.Definitions()
	for _, d := range defs {
		if d.Name == "memory_store" {
			var schema struct {
				Properties map[string]any `json:"properties"`
			}
			if err := json.Unmarshal(d.Parameters, &schema); err != nil {
				t.Fatal(err)
			}
			if _, has := schema.Properties["category"]; has {
				t.Error("memory_store schema must not include 'category'")
			}
		}
	}
}

func TestMemoryDefinitionsEffects(t *testing.T) {
	m := newMemory(t)
	want := map[string]tools.ToolEffect{
		"memory_store":  tools.ToolEffectMutate,
		"memory_search": tools.ToolEffectRead,
		"memory_get":    tools.ToolEffectRead,
	}
	for _, d := range m.Definitions() {
		if got := d.Effects; got != want[d.Name] {
			t.Errorf("%s: Effects = %v, want %v", d.Name, got, want[d.Name])
		}
	}
}
```

- [ ] **Step 13.2: Run the test — expect build failure**

Run: `go test ./tools/core/...`
Expected: FAIL with "cannot find `Memory` in package core" (not yet written).

- [ ] **Step 13.3: Write `memory.go`**

`openharness/tools/core/memory.go`:

```go
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/swiftdiaries/openharness/agent"
	"github.com/swiftdiaries/openharness/tools"
)

// Memory provides key-value storage backed by a JSON file. It implements
// agent.MemoryStore so the agent loop's retrieval layer (Plan 5) can
// consume it via the narrow interface.
type Memory struct {
	path    string
	mu      sync.Mutex
	entries map[string]agent.MemoryEntry
}

// NewMemory creates a Memory tool whose on-disk state lives at storePath.
// Missing files are treated as empty; the directory is created lazily on
// the first save.
func NewMemory(storePath string) *Memory {
	m := &Memory{path: storePath, entries: make(map[string]agent.MemoryEntry)}
	m.load()
	return m
}

func (m *Memory) load() {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	// Tolerate an empty file (decoded as nil map).
	decoded := map[string]agent.MemoryEntry{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return
	}
	m.entries = decoded
}

func (m *Memory) save() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0o644)
}

func (m *Memory) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "memory_store",
			Description: "Store a piece of information in agent memory.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"key":   {"type": "string", "description": "Unique key for this memory"},
					"value": {"type": "string", "description": "Value to store"}
				},
				"required": ["key", "value"]
			}`),
			Effects: tools.ToolEffectMutate,
		},
		{
			Name:        "memory_search",
			Description: "Search agent memory by case-insensitive substring match against key and value.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query (substring match)"},
					"limit": {"type": "integer", "description": "Max results (default 10)"}
				},
				"required": ["query"]
			}`),
			Effects: tools.ToolEffectRead,
		},
		{
			Name:        "memory_get",
			Description: "Retrieve a specific memory entry by key.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"key": {"type": "string", "description": "Key to retrieve"}
				},
				"required": ["key"]
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}

func (m *Memory) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	switch name {
	case "memory_store":
		return m.execStore(args)
	case "memory_search":
		return m.execSearch(ctx, args)
	case "memory_get":
		return m.execGet(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (m *Memory) execStore(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[params.Key] = agent.MemoryEntry{
		Key:       params.Key,
		Value:     params.Value,
		UpdatedAt: time.Now().UTC(),
	}
	if err := m.save(); err != nil {
		return nil, fmt.Errorf("save memory: %w", err)
	}
	return json.Marshal(map[string]string{"status": "stored", "key": params.Key})
}

func (m *Memory) execSearch(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}
	results, err := m.Search(ctx, params.Query, params.Limit)
	if err != nil {
		return nil, err
	}
	return json.Marshal(results)
}

// Search satisfies agent.MemoryStore. Semantics preserved verbatim from
// ghostfin's SearchEntries: case-insensitive substring match against Key
// and Value (the old Category field was dropped in the agent.MemoryEntry
// reconciliation). Limit <= 0 defaults to 10. Results stop at limit.
// Map iteration is unordered, so callers must not rely on a stable order.
func (m *Memory) Search(ctx context.Context, query string, limit int) ([]agent.MemoryEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	q := strings.ToLower(query)
	var results []agent.MemoryEntry
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.Key), q) ||
			strings.Contains(strings.ToLower(e.Value), q) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *Memory) execGet(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[params.Key]
	if !ok {
		return json.Marshal(map[string]string{"error": "not found"})
	}
	return json.Marshal(entry)
}
```

- [ ] **Step 13.4: Run tests**

Run: `go test ./tools/core/...`
Expected: PASS. If `TestMemorySearchSubstringSemantics` fails on the "red" case, the fix is to ensure both Key and Value substring-check fire (the implementation above matches either, which is correct).

- [ ] **Step 13.5: Commit**

```bash
git add tools/core/memory.go tools/core/memory_test.go
git commit -m "tools/core: reconcile memory with agent.MemoryEntry

- Local MemoryEntry deleted; uses agent.MemoryEntry{Key, Value, UpdatedAt}.
- SearchEntries renamed to Search(ctx, q, n) to satisfy agent.MemoryStore.
- 'category' parameter + Category field dropped from memory_store schema.
- DefaultMemoryPath() removed; caller always passes the path.
- On-disk format is new-shape only; ghostfin Plan 7 handles legacy migration.
- Substring semantics pinned by memory_test.go (Key + Value, case-insensitive).
- var _ agent.MemoryStore = (*Memory)(nil) compile-time assertion."
```

---

## Task 14: knowledge_graph types (`tools/core/knowledge_graph_types.go`)

**Files:**
- Create: `openharness/tools/core/knowledge_graph_types.go`

Per D1: flat (no embedding), full ghostfin `notes.Block` + `Page` + `Ref` field set.

- [ ] **Step 14.1: Write the types file**

`openharness/tools/core/knowledge_graph_types.go`:

```go
package core

import "time"

// KGBlock is the atomic content unit in an openharness knowledge graph.
// The field set mirrors ghostfin's internal/notes.Block exactly so Plan 7's
// adapter is a pure field-for-field copy. Future verticals whose knowledge
// schema diverges can still populate these types without embedding.
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

// KGPage is a page in the knowledge graph. Fields are flattened rather than
// embedded so openharness does not assume Page structurally specializes Block.
type KGPage struct {
	// Block-level fields (flattened from KGBlock).
	ID         string
	ParentID   *string
	PageID     string
	Title      string
	Order      int
	Collapsed  bool
	Properties map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// Page-only fields.
	Name       string
	JournalDay *string
	Icon       string
}

// KGRef is a reference edge between blocks. RefType values in ghostfin today
// are "page_ref", "tag", "block_ref"; the field is an opaque string so other
// verticals can define their own taxonomies.
type KGRef struct {
	SourceBlockID string
	TargetPageID  string
	RefType       string
}
```

- [ ] **Step 14.2: Build**

Run: `go build ./tools/core/...`
Expected: Success.

- [ ] **Step 14.3: Commit**

```bash
git add tools/core/knowledge_graph_types.go
git commit -m "tools/core: add KGBlock/KGPage/KGRef flat types

Mirrors ghostfin notes.Block/Page/Ref field sets exactly so Plan 7's
adapter is a pure field-for-field copy. Flattened (no embedding) so
non-ghostfin verticals need not mirror the Logseq embedding choice."
```

---

## Task 15: knowledge_graph built-in (`tools/core/knowledge_graph.go`)

**Files:**
- Create: `openharness/tools/core/knowledge_graph.go`
- Create: `openharness/tools/core/knowledge_graph_test.go`

Per D1: local rich `KnowledgeGraphStore` interface with 8 methods, all taking `ctx`. Test uses an in-memory fake `KnowledgeGraphStore` rather than a real ghostfin notes store.

- [ ] **Step 15.1: Write the test first**

`openharness/tools/core/knowledge_graph_test.go`:

```go
package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/swiftdiaries/openharness/tools"
)

// fakeKG is an in-memory KnowledgeGraphStore for tests.
type fakeKG struct {
	pages  map[string]KGPage           // keyed by Name (lowercase)
	byID   map[string]KGPage           // keyed by ID
	blocks map[string][]KGBlock        // keyed by PageID
}

func newFakeKG() *fakeKG {
	now := time.Now()
	p := KGPage{
		ID: "p1", PageID: "p1", Title: "stocks_watchlist",
		CreatedAt: now, UpdatedAt: now, Name: "stocks_watchlist",
	}
	return &fakeKG{
		pages: map[string]KGPage{"stocks_watchlist": p},
		byID:  map[string]KGPage{"p1": p},
		blocks: map[string][]KGBlock{
			"p1": {
				{ID: "b1", PageID: "p1", Title: "AAPL 150 shares at $185", Order: 0, CreatedAt: now, UpdatedAt: now},
				{ID: "b2", PageID: "p1", Title: "GOOGL 50 shares at $142", Order: 1, CreatedAt: now, UpdatedAt: now},
				{ID: "b3", PageID: "p1", Title: "Review Q1 performance", Order: 2, CreatedAt: now, UpdatedAt: now},
			},
		},
	}
}

func (f *fakeKG) Search(ctx context.Context, query string, limit int) ([]KGBlock, error) {
	q := strings.ToLower(query)
	var out []KGBlock
	for _, blks := range f.blocks {
		for _, b := range blks {
			if strings.Contains(strings.ToLower(b.Title), q) {
				out = append(out, b)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}
func (f *fakeKG) SearchInPage(ctx context.Context, pageID, query string, limit int) ([]KGBlock, error) {
	q := strings.ToLower(query)
	var out []KGBlock
	for _, b := range f.blocks[pageID] {
		if strings.Contains(strings.ToLower(b.Title), q) {
			out = append(out, b)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
func (f *fakeKG) GetPage(ctx context.Context, name string) (*KGPage, error) {
	if p, ok := f.pages[name]; ok {
		return &p, nil
	}
	return nil, nil
}
func (f *fakeKG) GetPageByID(ctx context.Context, id string) (*KGPage, error) {
	if p, ok := f.byID[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (f *fakeKG) GetBlockTree(ctx context.Context, pageID string) ([]KGBlock, error) {
	return f.blocks[pageID], nil
}
func (f *fakeKG) GetBlockSnippets(ctx context.Context, pageID string, limit int) ([]string, error) {
	var out []string
	for _, b := range f.blocks[pageID] {
		out = append(out, b.Title)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
func (f *fakeKG) GetRelatedPages(ctx context.Context, pageID string, depth int) ([]KGPage, []KGRef, error) {
	return nil, nil, nil
}
func (f *fakeKG) GetPagesWithSharedTags(ctx context.Context, pageID string, limit int) ([]KGPage, error) {
	return nil, nil
}

func TestKnowledgeGraphDefinitions(t *testing.T) {
	kg := NewKnowledgeGraph(newFakeKG())
	defs := kg.Definitions()
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "knowledge_graph_search" {
		t.Errorf("name = %q, want knowledge_graph_search", defs[0].Name)
	}
	if defs[0].Effects != tools.ToolEffectRead {
		t.Errorf("Effects = %v, want Read", defs[0].Effects)
	}
}

func TestKnowledgeGraphSearch_SearchMode(t *testing.T) {
	kg := NewKnowledgeGraph(newFakeKG())
	args, _ := json.Marshal(map[string]any{"query": "AAPL", "mode": "search"})
	result, err := kg.Execute(context.Background(), "knowledge_graph_search", args)
	if err != nil {
		t.Fatal(err)
	}
	var text string
	json.Unmarshal(result, &text)
	if !strings.Contains(text, "AAPL") {
		t.Fatalf("expected AAPL in result: %s", text)
	}
	if !strings.Contains(text, "stocks_watchlist") {
		t.Fatalf("expected stocks_watchlist in result: %s", text)
	}
}

func TestKnowledgeGraphSearch_PageMode(t *testing.T) {
	kg := NewKnowledgeGraph(newFakeKG())
	args, _ := json.Marshal(map[string]any{"query": "stocks_watchlist", "mode": "page"})
	result, err := kg.Execute(context.Background(), "knowledge_graph_search", args)
	if err != nil {
		t.Fatal(err)
	}
	var text string
	json.Unmarshal(result, &text)
	if !strings.Contains(text, "AAPL") || !strings.Contains(text, "GOOGL") {
		t.Fatalf("expected page content: %s", text)
	}
}
```

- [ ] **Step 15.2: Run the test — expect build failure**

Run: `go test ./tools/core/...`
Expected: FAIL (NewKnowledgeGraph, KnowledgeGraphStore undefined).

- [ ] **Step 15.3: Write `knowledge_graph.go`**

`openharness/tools/core/knowledge_graph.go`:

```go
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/swiftdiaries/openharness/tools"
)

// KnowledgeGraphStore is the rich interface the knowledge_graph tool calls
// against. It is intentionally broader than agent.KnowledgeStore (which
// ships a two-method Search/Get surface for Plan 5's retrieval layer).
// Different consumers want different contracts — keeping them separate
// avoids forcing every future vertical to implement 8 methods when the
// retrieval layer only needs 2.
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

// KnowledgeGraph is the built-in tool that exposes a user's markdown/block
// knowledge store to the agent as a single knowledge_graph_search tool with
// four modes: search (full-text), traverse (connected pages via refs),
// page (fetch content), related (shared-tag pages).
type KnowledgeGraph struct {
	store KnowledgeGraphStore
}

// NewKnowledgeGraph creates a KnowledgeGraph tool backed by the given store.
func NewKnowledgeGraph(store KnowledgeGraphStore) *KnowledgeGraph {
	return &KnowledgeGraph{store: store}
}

type kgArgs struct {
	Query      string `json:"query"`
	Page       string `json:"page,omitempty"`
	Mode       string `json:"mode,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

func (kg *KnowledgeGraph) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "knowledge_graph_search",
			Description: "Search the user's notes knowledge graph. Modes: 'search' for full-text keyword search, 'traverse' to find connected pages via refs, 'page' to fetch a page's content, 'related' for pages sharing tags.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query or page name"},
					"page":  {"type": "string", "description": "Optional page name to scope search to"},
					"mode":  {"type": "string", "enum": ["search", "traverse", "page", "related"], "description": "Search mode (default: search)"},
					"max_results": {"type": "integer", "description": "Max results to return (default 10, max 100)"}
				},
				"required": ["query"]
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}

func (kg *KnowledgeGraph) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name != "knowledge_graph_search" {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	var params kgArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if params.Mode == "" {
		params.Mode = "search"
	}
	if params.MaxResults <= 0 {
		if params.Mode == "page" {
			params.MaxResults = 100
		} else {
			params.MaxResults = 10
		}
	}
	if params.MaxResults > 100 {
		params.MaxResults = 100
	}

	var (
		result string
		err    error
	)
	switch params.Mode {
	case "search":
		result, err = kg.execSearch(ctx, params)
	case "traverse":
		result, err = kg.execTraverse(ctx, params)
	case "page":
		result, err = kg.execPage(ctx, params)
	case "related":
		result, err = kg.execRelated(ctx, params)
	default:
		return nil, fmt.Errorf("unknown mode: %s", params.Mode)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (kg *KnowledgeGraph) execSearch(ctx context.Context, params kgArgs) (string, error) {
	var (
		blocks []KGBlock
		err    error
	)
	if params.Page != "" {
		page, err := kg.store.GetPage(ctx, strings.ToLower(params.Page))
		if err != nil {
			return "", fmt.Errorf("get page: %w", err)
		}
		if page == nil {
			return fmt.Sprintf("Page %q not found.", params.Page), nil
		}
		blocks, err = kg.store.SearchInPage(ctx, page.ID, params.Query, params.MaxResults)
		if err != nil {
			return "", fmt.Errorf("search in page: %w", err)
		}
	} else {
		blocks, err = kg.store.Search(ctx, params.Query, params.MaxResults)
		if err != nil {
			return "", fmt.Errorf("search: %w", err)
		}
	}

	if len(blocks) == 0 {
		return fmt.Sprintf("No results for %q.", params.Query), nil
	}

	grouped := map[string][]KGBlock{}
	var pageOrder []string
	for _, b := range blocks {
		if _, seen := grouped[b.PageID]; !seen {
			pageOrder = append(pageOrder, b.PageID)
		}
		grouped[b.PageID] = append(grouped[b.PageID], b)
	}

	var sb strings.Builder
	for _, pid := range pageOrder {
		page, _ := kg.store.GetPageByID(ctx, pid)
		pageName := pid
		if page != nil {
			pageName = page.Name
		}
		fmt.Fprintf(&sb, "## %s\n", pageName)
		for _, b := range grouped[pid] {
			fmt.Fprintf(&sb, "- %s\n", b.Title)
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

func (kg *KnowledgeGraph) execTraverse(ctx context.Context, params kgArgs) (string, error) {
	pageName := strings.TrimSuffix(strings.ToLower(params.Query), ".md")
	page, err := kg.store.GetPage(ctx, pageName)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page == nil {
		return fmt.Sprintf("Page %q not found.", params.Query), nil
	}

	depth := 2
	if params.MaxResults <= 5 {
		depth = 1
	}

	related, refs, err := kg.store.GetRelatedPages(ctx, page.ID, depth)
	if err != nil {
		return "", fmt.Errorf("get related pages: %w", err)
	}
	if len(related) == 0 {
		return fmt.Sprintf("No pages connected to %q.", pageName), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Pages connected to %q:\n", pageName)
	pageRefs := map[string][]string{}
	for _, r := range refs {
		pageRefs[r.TargetPageID] = append(pageRefs[r.TargetPageID], r.RefType)
	}

	count := 0
	for _, p := range related {
		if count >= params.MaxResults {
			break
		}
		refTypes := uniqueStrings(pageRefs[p.ID])
		if len(refTypes) > 0 {
			fmt.Fprintf(&sb, "## %s (via %s)\n", p.Name, strings.Join(refTypes, ", "))
		} else {
			fmt.Fprintf(&sb, "## %s\n", p.Name)
		}
		snippets, _ := kg.store.GetBlockSnippets(ctx, p.ID, 3)
		for _, s := range snippets {
			if len(s) > 200 {
				s = s[:200] + "..."
			}
			fmt.Fprintf(&sb, "- %s\n", s)
		}
		sb.WriteString("\n")
		count++
	}
	return strings.TrimSpace(sb.String()), nil
}

func (kg *KnowledgeGraph) execPage(ctx context.Context, params kgArgs) (string, error) {
	pageName := strings.TrimSuffix(strings.ToLower(params.Query), ".md")
	page, err := kg.store.GetPage(ctx, pageName)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page == nil {
		return fmt.Sprintf("Page %q not found.", pageName), nil
	}

	blocks, err := kg.store.GetBlockTree(ctx, page.ID)
	if err != nil {
		return "", fmt.Errorf("get block tree: %w", err)
	}

	type node struct {
		block    KGBlock
		children []*node
	}
	nodeMap := map[string]*node{}
	var roots []*node
	for i := range blocks {
		nodeMap[blocks[i].ID] = &node{block: blocks[i]}
	}
	for i := range blocks {
		n := nodeMap[blocks[i].ID]
		if blocks[i].ParentID != nil {
			if parent, ok := nodeMap[*blocks[i].ParentID]; ok {
				parent.children = append(parent.children, n)
				continue
			}
		}
		roots = append(roots, n)
	}
	if params.MaxResults < len(roots) {
		roots = roots[:params.MaxResults]
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n", page.Name)
	var writeNode func(n *node, depth int)
	writeNode = func(n *node, depth int) {
		indent := strings.Repeat("  ", depth)
		fmt.Fprintf(&sb, "%s- %s\n", indent, n.block.Title)
		for _, c := range n.children {
			writeNode(c, depth+1)
		}
	}
	for _, r := range roots {
		writeNode(r, 0)
	}
	return strings.TrimSpace(sb.String()), nil
}

func (kg *KnowledgeGraph) execRelated(ctx context.Context, params kgArgs) (string, error) {
	pageName := strings.TrimSuffix(strings.ToLower(params.Query), ".md")
	page, err := kg.store.GetPage(ctx, pageName)
	if err != nil {
		return "", fmt.Errorf("get page: %w", err)
	}
	if page == nil {
		return fmt.Sprintf("Page %q not found.", params.Query), nil
	}
	related, err := kg.store.GetPagesWithSharedTags(ctx, page.ID, params.MaxResults)
	if err != nil {
		return "", fmt.Errorf("get related pages: %w", err)
	}
	if len(related) == 0 {
		return fmt.Sprintf("No pages share tags with %q.", pageName), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Pages sharing tags with %q:\n\n", pageName)
	for _, p := range related {
		fmt.Fprintf(&sb, "## %s\n", p.Name)
		snippets, _ := kg.store.GetBlockSnippets(ctx, p.ID, 3)
		for _, s := range snippets {
			if len(s) > 200 {
				s = s[:200] + "..."
			}
			fmt.Fprintf(&sb, "- %s\n", s)
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
```

- [ ] **Step 15.4: Run tests**

Run: `go test ./tools/core/...`
Expected: PASS.

- [ ] **Step 15.5: Commit**

```bash
git add tools/core/knowledge_graph.go tools/core/knowledge_graph_test.go
git commit -m "tools/core: knowledge_graph against local KnowledgeGraphStore

Replaces ghostfin's notes.Store dependency with a local 8-method
interface. All methods take ctx. Test uses an in-memory fake; Plan 7
adapts ghostfin's notes.Store to satisfy the same interface."
```

---

## Task 16: subagent built-in (`tools/core/subagent.go`)

**Files:**
- Create: `openharness/tools/core/subagent.go`
- Create: `openharness/tools/core/subagent_test.go`

Per D3: `SubagentConfig{Subagents map[string]SubagentSpec}`. Empty map → `Definitions()` returns nil so `Register` omits the tool. Dynamic JSON schema enum built from `maps.Keys(cfg.Subagents)`.

- [ ] **Step 16.1: Write the test first**

`openharness/tools/core/subagent_test.go`:

```go
package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestSubagentConfigured_Definitions(t *testing.T) {
	cfg := SubagentConfig{
		Subagents: map[string]SubagentSpec{
			"analyst": {Name: "analyst", Description: "Financial analyst"},
			"scribe":  {Name: "scribe", Description: "Report writer"},
		},
	}
	st := NewSubagentTool(cfg)
	defs := st.Definitions()
	if len(defs) != 1 {
		t.Fatalf("want 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "agent" {
		t.Errorf("name = %q, want agent", defs[0].Name)
	}
	if defs[0].Effects != tools.ToolEffectMutate {
		t.Errorf("Effects = %v, want Mutate", defs[0].Effects)
	}
	// Verify the enum includes both configured types.
	var schema struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(defs[0].Parameters, &schema); err != nil {
		t.Fatal(err)
	}
	enum := schema.Properties["subagent_type"].Enum
	seen := map[string]bool{}
	for _, v := range enum {
		seen[v] = true
	}
	if !seen["analyst"] || !seen["scribe"] {
		t.Errorf("enum missing entries: %v", enum)
	}
	// Description should mention both types.
	if !strings.Contains(defs[0].Description, "analyst") ||
		!strings.Contains(defs[0].Description, "scribe") {
		t.Errorf("description should mention configured types: %s", defs[0].Description)
	}
}

func TestSubagentEmpty_DefinitionsReturnsNil(t *testing.T) {
	st := NewSubagentTool(SubagentConfig{})
	defs := st.Definitions()
	if len(defs) != 0 {
		t.Errorf("want 0 definitions for empty SubagentConfig, got %d", len(defs))
	}
}

func TestSubagentExecute_Stub(t *testing.T) {
	st := NewSubagentTool(SubagentConfig{
		Subagents: map[string]SubagentSpec{"general": {Name: "general", Description: "generalist"}},
	})
	args, _ := json.Marshal(map[string]string{
		"description":   "do the thing",
		"prompt":        "please do",
		"subagent_type": "general",
	})
	out, err := st.Execute(t.Context(), "agent", args)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	json.Unmarshal(out, &got)
	if got["status"] != "delegated" {
		t.Errorf("status = %q, want delegated", got["status"])
	}
	if got["subagent_type"] != "general" {
		t.Errorf("subagent_type = %q, want general", got["subagent_type"])
	}
}
```

(`t.Context()` is Go 1.24+; we're on 1.25 so that's fine. If it causes trouble, replace with `context.Background()`.)

- [ ] **Step 16.2: Run the test — expect failure**

Run: `go test ./tools/core/...`
Expected: FAIL (NewSubagentTool/SubagentConfig undefined).

- [ ] **Step 16.3: Write `subagent.go`**

`openharness/tools/core/subagent.go`:

```go
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/swiftdiaries/openharness/tools"
)

// SubagentSpec describes a single registrable subagent type. AllowedTools is
// reserved — Plan 3 stores it; Layer 4's gateway will enforce the allowlist
// when actually executing subagents.
type SubagentSpec struct {
	Name         string
	Description  string
	SystemPrompt string
	AllowedTools []string
}

// SubagentConfig enumerates which subagent types the tool advertises to the
// LLM. The map key is the type name exposed in the subagent_type JSON enum.
// Plan 3 stores configuration only; Layer 4 will extend the public surface
// with registry/runner/tracer dependencies to actually execute subagents.
type SubagentConfig struct {
	Subagents map[string]SubagentSpec
}

// SubagentTool publishes a single "agent" definition that lets the LLM
// delegate a task to a typed subagent. Plan 3's implementation is a stub
// returning {"status":"delegated"}; Layer 4 wires real execution.
type SubagentTool struct {
	cfg SubagentConfig
}

// NewSubagentTool returns a SubagentTool. An empty cfg.Subagents causes
// Definitions() to return nil so Register omits the tool entirely — a JSON
// schema enum with zero entries is invalid.
func NewSubagentTool(cfg SubagentConfig) *SubagentTool {
	return &SubagentTool{cfg: cfg}
}

func (s *SubagentTool) Definitions() []tools.ToolDefinition {
	if len(s.cfg.Subagents) == 0 {
		return nil
	}

	// Sorted keys so description + enum are deterministic across runs.
	keys := make([]string, 0, len(s.cfg.Subagents))
	for k := range s.cfg.Subagents {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var descLines []string
	for _, k := range keys {
		spec := s.cfg.Subagents[k]
		d := spec.Description
		if d == "" {
			d = k
		}
		descLines = append(descLines, fmt.Sprintf("- %s: %s", k, d))
	}
	description := "Delegate a task to a specialized subagent. Use when the task needs domain expertise.\n\nAvailable types:\n" + strings.Join(descLines, "\n")

	// Build the JSON schema with a dynamic enum.
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{
				"type":        "string",
				"description": "Short description of what this subagent will do (3-5 words)",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task for the subagent to perform",
			},
			"subagent_type": map[string]any{
				"type":        "string",
				"enum":        keys,
				"description": "Type of subagent to delegate to",
			},
		},
		"required": []string{"description", "prompt", "subagent_type"},
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		// json.Marshal on this map cannot fail in practice; preserve
		// the tool anyway with an empty schema so Register doesn't skip it.
		raw = []byte(`{"type":"object"}`)
	}

	return []tools.ToolDefinition{{
		Name:        "agent",
		Description: description,
		Parameters:  json.RawMessage(raw),
		Effects:     tools.ToolEffectMutate,
	}}
}

func (s *SubagentTool) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name != "agent" {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	var input struct {
		Description  string `json:"description"`
		Prompt       string `json:"prompt"`
		SubagentType string `json:"subagent_type"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	// Plan 3 does not execute subagents — Layer 4's gateway wires the
	// real runner. Return a stub payload the agent loop can round-trip.
	return json.Marshal(map[string]string{
		"status":        "delegated",
		"subagent_type": input.SubagentType,
		"description":   input.Description,
		"note":          "Subagent execution will be wired in Layer 4 (gateway)",
	})
}
```

- [ ] **Step 16.4: Run tests**

Run: `go test ./tools/core/...`
Expected: PASS.

- [ ] **Step 16.5: Commit**

```bash
git add tools/core/subagent.go tools/core/subagent_test.go
git commit -m "tools/core: SubagentTool with SubagentConfig + dynamic schema enum

Ghostfin's hardcoded finance subagent enum becomes data passed at
registration time. Empty SubagentConfig causes Definitions() to return
nil so Register omits the tool (JSON schema enum needs >=1 entry).
Plan 3 stores config only; Layer 4 adds execution wiring."
```

---

## Task 17: core.Register + Config (`tools/core/core.go`)

**Files:**
- Create: `openharness/tools/core/core.go`

- [ ] **Step 17.1: Write `core.go`**

`openharness/tools/core/core.go`:

```go
// Package core holds built-in tool implementations for openharness: web,
// filesystem, exec, memory, tasks, ask_user, knowledge_graph, and subagent.
// Callers bring their own Registry and call Register(r, cfg) — the
// package does not construct a Registry on their behalf so verticals can
// mix built-ins with their own tools.
package core

import (
	"github.com/swiftdiaries/openharness/tools"
)

// Config captures what the built-in set needs to construct. All fields are
// optional where noted; omitted optional fields cause the corresponding
// tool to be skipped rather than erroring.
type Config struct {
	// WorkspacePath is the root for filesystem + exec + tasks (required
	// for those three tools to function; an empty string still succeeds
	// but scopes tools to the empty workspace).
	WorkspacePath string
	// MemoryPath is the JSON file backing the memory tool. An empty
	// string skips registration of memory_*.
	MemoryPath string
	// KnowledgeStore, if non-nil, enables the knowledge_graph_search tool.
	// nil skips registration — matches ghostfin's "notes optional" path.
	KnowledgeStore KnowledgeGraphStore
	// Subagent enumerates the subagent types advertised to the LLM. An
	// empty map (or zero value) skips registration of the agent tool.
	Subagent SubagentConfig
}

// Register populates r with built-in tools according to cfg. Every
// registered definition must carry a non-Unspecified Effects value;
// Register returns the underlying tools.Registry.Register error if any
// tool fails that contract — the partial registration state is
// whatever r.Register left behind.
//
// Skips:
//   - memory_* if cfg.MemoryPath == ""
//   - knowledge_graph_search if cfg.KnowledgeStore == nil
//   - agent if cfg.Subagent.Subagents is empty
func Register(r *tools.Registry, cfg Config) error {
	if err := r.Register("web", NewWebSearch()); err != nil {
		return err
	}
	if err := r.Register("web", NewWebFetch()); err != nil {
		return err
	}
	if err := r.Register("filesystem", NewFilesystem(cfg.WorkspacePath)); err != nil {
		return err
	}
	if err := r.Register("exec", NewExec(cfg.WorkspacePath)); err != nil {
		return err
	}
	if err := r.Register("interactive", NewAskUser()); err != nil {
		return err
	}
	if err := r.Register("interactive", NewTaskCRUD(cfg.WorkspacePath)); err != nil {
		return err
	}
	if cfg.MemoryPath != "" {
		if err := r.Register("memory", NewMemory(cfg.MemoryPath)); err != nil {
			return err
		}
	}
	if cfg.KnowledgeStore != nil {
		if err := r.Register("knowledge", NewKnowledgeGraph(cfg.KnowledgeStore)); err != nil {
			return err
		}
	}
	// Subagent's Definitions() returns nil when the map is empty, so
	// Register would iterate zero definitions — but that still succeeds
	// and leaves behind no tool. We skip the call explicitly for clarity.
	if len(cfg.Subagent.Subagents) > 0 {
		if err := r.Register("agent", NewSubagentTool(cfg.Subagent)); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 17.2: Build**

Run: `go build ./tools/core/...`
Expected: Success.

- [ ] **Step 17.3: Commit**

```bash
git add tools/core/core.go
git commit -m "tools/core: Config + Register(r, cfg) entrypoint

Callers bring their own Registry. nil KnowledgeStore skips
knowledge_graph_search; empty SubagentConfig omits the agent tool;
empty MemoryPath omits memory_*. Every call propagates the underlying
Effects validation error from tools.Registry."
```

---

## Task 18: Integration test — D4 matrix + contract assertions (`tools/core/core_test.go`)

**Files:**
- Create: `openharness/tools/core/core_test.go`

This test is the spec's primary verification gate — the 12-row D4 matrix, `EffectsFor("unknown")` semantics, `Register` Effects-validation behavior, and omission semantics (nil KG, empty subagent, empty memory path).

- [ ] **Step 18.1: Write the integration test**

`openharness/tools/core/core_test.go`:

```go
package core

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

// fullConfig returns a Config that exercises every opt-in tool.
func fullConfig(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	return Config{
		WorkspacePath:  dir,
		MemoryPath:     filepath.Join(dir, "memory.json"),
		KnowledgeStore: newFakeKG(),
		Subagent: SubagentConfig{
			Subagents: map[string]SubagentSpec{
				"general": {Name: "general", Description: "generalist"},
			},
		},
	}
}

func TestRegister_WiresAllTools(t *testing.T) {
	r := tools.NewRegistry()
	if err := Register(r, fullConfig(t)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	want := []string{
		"web_search",
		"web_fetch",
		"read_file", "write_file", "list_files", "edit_file",
		"exec",
		"ask_user_question",
		"task_create", "task_update", "task_get", "task_list",
		"memory_store", "memory_search", "memory_get",
		"knowledge_graph_search",
		"agent",
	}
	for _, name := range want {
		if r.Get(name) == nil {
			t.Errorf("expected tool %q registered, missing", name)
		}
	}
}

func TestRegister_D4EffectsMatrix(t *testing.T) {
	r := tools.NewRegistry()
	if err := Register(r, fullConfig(t)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Every row from the D4 table in the Plan 3 spec — asserted at the
	// definition level. If this matrix drifts from the tool code, Plan 5's
	// loop-detection parity invariant will fail.
	cases := []struct {
		name string
		want tools.ToolEffect
	}{
		// web
		{"web_search", tools.ToolEffectRead},
		{"web_fetch", tools.ToolEffectRead},
		// filesystem
		{"read_file", tools.ToolEffectRead},
		{"list_files", tools.ToolEffectRead},
		{"write_file", tools.ToolEffectMutate},
		{"edit_file", tools.ToolEffectMutate},
		// exec — neutral (preserves tool_loop.go:161 behavior)
		{"exec", tools.ToolEffectNeutral},
		// memory
		{"memory_search", tools.ToolEffectRead},
		{"memory_get", tools.ToolEffectRead},
		{"memory_store", tools.ToolEffectMutate},
		// ask_user
		{"ask_user_question", tools.ToolEffectInteractive},
		// tasks
		{"task_get", tools.ToolEffectRead},
		{"task_list", tools.ToolEffectRead},
		{"task_create", tools.ToolEffectMutate},
		{"task_update", tools.ToolEffectMutate},
		// knowledge_graph
		{"knowledge_graph_search", tools.ToolEffectRead},
		// subagent
		{"agent", tools.ToolEffectMutate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := r.EffectsFor(c.name); got != c.want {
				t.Errorf("EffectsFor(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestRegister_EffectsForUnknownReturnsUnspecified(t *testing.T) {
	r := tools.NewRegistry()
	if err := Register(r, fullConfig(t)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if got := r.EffectsFor("does_not_exist"); got != tools.ToolEffectUnspecified {
		t.Errorf("EffectsFor(unknown) = %v, want Unspecified", got)
	}
}

// unclassifiedTool returns a single definition with Effects left as the zero
// value (Unspecified). Register must refuse to register it.
type unclassifiedTool struct{}

func (unclassifiedTool) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{{
		Name:        "oops_unclassified",
		Description: "a forgetful tool author",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		// Effects deliberately unset → Unspecified.
	}}
}
func (unclassifiedTool) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	return nil, nil
}

func TestRegister_RejectsUnspecifiedEffects(t *testing.T) {
	r := tools.NewRegistry()
	err := r.Register("misc", unclassifiedTool{})
	if err == nil {
		t.Fatal("expected Register to reject Unspecified Effects")
	}
	if !strings.Contains(err.Error(), "oops_unclassified") {
		t.Errorf("error should name the offending definition, got: %v", err)
	}
	// Registry must not leak the partial registration — Get must return nil.
	if r.Get("oops_unclassified") != nil {
		t.Errorf("Registry should not contain rejected definition")
	}
}

func TestRegister_NilKnowledgeStoreOmitsKG(t *testing.T) {
	cfg := fullConfig(t)
	cfg.KnowledgeStore = nil
	r := tools.NewRegistry()
	if err := Register(r, cfg); err != nil {
		t.Fatal(err)
	}
	if r.Get("knowledge_graph_search") != nil {
		t.Error("nil KnowledgeStore must omit knowledge_graph_search")
	}
}

func TestRegister_EmptySubagentOmitsAgent(t *testing.T) {
	cfg := fullConfig(t)
	cfg.Subagent = SubagentConfig{}
	r := tools.NewRegistry()
	if err := Register(r, cfg); err != nil {
		t.Fatal(err)
	}
	if r.Get("agent") != nil {
		t.Error("empty Subagent.Subagents must omit agent")
	}
}

func TestRegister_EmptyMemoryPathOmitsMemory(t *testing.T) {
	cfg := fullConfig(t)
	cfg.MemoryPath = ""
	r := tools.NewRegistry()
	if err := Register(r, cfg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"memory_store", "memory_search", "memory_get"} {
		if r.Get(name) != nil {
			t.Errorf("empty MemoryPath must omit %s", name)
		}
	}
}
```

- [ ] **Step 18.2: Run tests**

Run: `go test ./tools/core/... -v -run TestRegister`
Expected: PASS (all 17 matrix rows green + omission cases).

- [ ] **Step 18.3: Commit**

```bash
git add tools/core/core_test.go
git commit -m "tools/core: integration + D4 matrix verification

Asserts all 17 definition-level Effects rows from the Plan 3 spec,
EffectsFor(unknown) contract, Register rejection of Unspecified
Effects, and omission semantics for nil KnowledgeStore / empty
SubagentConfig / empty MemoryPath."
```

---

## Task 19 (V1): Dependency-direction verification

**Files:** none (read-only check)

This is an automated guard against a regression the spec cares about: `openharness/tools/` must not import ghostfin, providers, sessions, cost, or any enterprise package.

- [ ] **Step 19.1: Run the import audit**

```bash
cd /Users/adhita/projects/python/src/github.com/swiftdiaries/openharness
go list -deps -f '{{.ImportPath}}: {{.Imports}}' ./tools/... ./tools/core/... \
  | grep -E '(ghostfin|openharness/providers|openharness/sessions|openharness/cost|openharness/edition)' \
  || echo "PASS: no forbidden imports"
```

Expected output: `PASS: no forbidden imports`.

If the grep matches something, fix the import before continuing.

- [ ] **Step 19.2: Run the build-tag audit**

```bash
grep -rE '^//go:build\s+sqliteonly' tools/ && echo "FAIL: sqliteonly tag present" || echo "PASS: no sqliteonly tags"
```

Expected: `PASS: no sqliteonly tags`.

- [ ] **Step 19.3: Run the package-import-direction check**

```bash
# core may import tools + agent + stdlib + google/uuid
go list -deps ./tools/core/... | grep -E '^github.com/swiftdiaries/openharness/' | sort -u
```

Expected lines (exact set):
```
github.com/swiftdiaries/openharness/agent
github.com/swiftdiaries/openharness/tools
github.com/swiftdiaries/openharness/tools/core
```

Any additional `openharness/*` line indicates scope creep — investigate.

- [ ] **Step 19.4: (No commit needed — read-only verification.)**

---

## Task 20 (V2): Full test suite green + vet clean

- [ ] **Step 20.1: Run all tests with race detection**

```bash
go test -race ./tools/... ./tools/core/...
```

Expected: all tests PASS, no race warnings.

- [ ] **Step 20.2: Run `go vet`**

```bash
go vet ./tools/... ./tools/core/...
```

Expected: no output (clean).

- [ ] **Step 20.3: Run `go build` on the whole module**

```bash
go build ./...
```

Expected: Success — no unrelated packages broken by the new imports.

- [ ] **Step 20.4: Commit a green-check marker (optional but recommended)**

If anything regressed in this task, fix it and commit. Otherwise skip — no new files.

---

## Task 21: Update the Progress table in the extraction spec

**Files:**
- Modify: `openharness/docs/superpowers/specs/2026-04-10-openharness-extraction-design.md` (Progress section)
- Modify: `openharness/docs/superpowers/plans/layer-2/2026-04-13-execution-order.md` (Wave 2 status)

- [ ] **Step 21.1: Flip the Layer 2 status in the Progress table**

In `docs/superpowers/specs/2026-04-10-openharness-extraction-design.md` around line 52, update the Layer 2 row's Status cell:

old:
```
| Layer 2 — agent primitives | in progress (Plans 1–2 merged; Plan 3 next) | ...
```

new:
```
| Layer 2 — agent primitives | in progress (Plans 1–3 merged; Plan 4 next) | ...
```

Also update the "Next unblocked work" line (around line 58):

old:
```
**Next unblocked work:** Layer 2 Plan 3 — write implementation plan from `specs/2026-04-16-plan-3-tools-design.md` (resolve the 5 adversarial-review blockers first), then execute.
```

new:
```
**Next unblocked work:** Layer 2 Plan 4 — MCP outbound + UIBridge. Plan 3 (tools) merged.
```

- [ ] **Step 21.2: Update the Layer 2 execution-order doc**

Append a status marker under the Wave 2 entry in `plans/layer-2/2026-04-13-execution-order.md` indicating Plan 3 is complete, with a link to this implementation-plan file.

- [ ] **Step 21.3: Commit the docs update**

```bash
git add docs/superpowers/specs/2026-04-10-openharness-extraction-design.md \
        docs/superpowers/plans/layer-2/2026-04-13-execution-order.md
git commit -m "docs: mark Layer 2 Plan 3 (tools) as merged"
```

---

## Task 22: Close the beads issue + push

- [ ] **Step 22.1: Push to remote**

```bash
git push -u origin feat/layer-2-plan-3-tools
```

- [ ] **Step 22.2: Open the PR** (via `gh pr create` or the superpowers:finishing-a-development-branch flow)

- [ ] **Step 22.3: Once merged, close the beads issue**

```bash
bd close openharness-pol.2.3 --reason "Plan 3 tools merged — $(git rev-parse --short HEAD)"
bd dolt push
```

---

## Spec-Coverage Self-Review

| Spec section | Covered by task(s) |
|---|---|
| D1 — KnowledgeGraphStore interface (8 methods, ctx everywhere) | T14, T15 |
| D1 — flat KGBlock/KGPage/KGRef with full ghostfin field set | T14 |
| D2 — agent.MemoryEntry everywhere, rename SearchEntries → Search | T13 |
| D2 — drop `category` JSON param | T13 (`TestMemoryExecuteRejectsCategoryParameter`) |
| D2 — `var _ agent.MemoryStore = (*Memory)(nil)` | T13 (compile-time assertion in test file) |
| D2 — DefaultMemoryPath removed | T13 (omitted from file) |
| D2 — JSON file persistence preserved, new format only | T13 (`TestMemoryOnDiskFormat`) |
| D3 — SubagentConfig map, dynamic enum, empty-map omission | T16, T17, T18 |
| D4 — 5-state ToolEffect + Unspecified error | T2, T3, T18 |
| D4 — Visibility enum shipped in Plan 3 | T2 |
| D4 — Interactive semantics | T2 (doc), T11 (applied), T18 (asserted) |
| D4 — 12-row matrix (17 rows including all real-definitions) | T18 |
| D4 — EffectsFor precomputed + O(1) | T3, T18 |
| D5 — `Register(r, cfg) error` pattern with per-call validation | T17, T18 |
| D5 — package renamed `builtin` → `core` | T17 and throughout |
| File inventory — verbatim copies (composite, ssrf, shell_deny, scrub, external_content) | T4, T5, T6 |
| File inventory — `sqliteonly` stripped from tasks | T12 |
| File inventory — no registry_test.go; coverage lives in core_test.go | T18 |
| File inventory — memory_test.go authored | T13 |
| Dependency direction (tools → stdlib + x/net; core → tools + agent) | T19 |
| Verification checkboxes — `go test` green, `go vet` clean | T20 |
| Verification — loop-detection parity invariant declaration | Covered by the matrix in T18; Plan 5 runs the runtime fixture |

All spec sections map to tasks; no gaps.

---

## Risk Notes for Reviewers

- **Exec's `matchesDenyPattern`** may need to be exported (`MatchesDenyPattern`) for cross-package use — Task 10 Step 10.1 flags this as an inline prerequisite. If the function is already exported, skip that sub-step.
- **Memory map iteration order** is Go-random. `TestMemorySearchSubstringSemantics` uses set membership assertions to avoid depending on iteration order.
- **Subagent JSON schema `required`** now lists `subagent_type` — ghostfin's original marked it optional with `default: "general"`. This is a deliberate tightening: callers must always declare the type; Register omits the tool when no types are configured, so the field is never nullable at call time.
- **`x/net` promotion from indirect to direct** in Task 1 is the only `go.mod` edit. If `go mod tidy` prunes other existing direct deps, stop and investigate rather than committing the churn.

---

## Out of Scope (re-stated)

- MCP tools (`tools/mcp/`) — Plan 4.
- Agent loop wiring (`tool_loop.go`, `mode.go` reading Effects) — Plan 5.
- `notes.Store` adapter methods satisfying `KnowledgeGraphStore` — Plan 7 (ghostfin repo).
- Subagent runtime execution — Layer 4 (gateway).
- Storage backend abstraction for Memory — future Layer 3+.
