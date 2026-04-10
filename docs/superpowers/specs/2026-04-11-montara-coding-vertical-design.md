# Montara — Coding Agent Vertical Design Spec

**Date:** 2026-04-11
**Status:** Draft
**Depends on:** openharness extraction design spec (2026-04-10)

## Context

Montara is a minimal coding agent vertical that proves openharness can support domains beyond finance. It lives at `openharness/examples/montara/` and exercises the framework's registration APIs — tools, agents, frontend views, and trace attributes — to build a functional pair-programming agent with both desktop (Wails) and CLI interfaces.

The name comes from Montara Mountain, a coastal ridge hike near Half Moon Bay.

### Problem

The openharness design spec defines a vertical customization model (registration APIs at app startup), but the only planned consumer is GhostFin (finance). Without a second vertical, there's no proof the framework is truly domain-agnostic — hidden finance assumptions could be baked into interfaces.

### Goal

Montara validates that:
1. A coding agent maps cleanly onto openharness's registration APIs
2. The harness boundary layer supports both desktop and CLI interaction patterns
3. Framework gaps are identified early (before Layers 2-4 are extracted)

### Non-Goals

- Competing with Goose, Claude Code, or aider as a product
- Implementing LSP/tree-sitter integration (specced but not built)
- Building a full sandbox/container isolation system (specced but not built)
- Supporting MCP extension loading (openharness provides the bridge; Montara doesn't need custom MCP servers)

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Location | `examples/montara/` in openharness repo | Compiles on every CI run, catches breakage immediately, discoverable |
| Scope | 4 tools, 1 agent, 1 frontend view | Minimum viable set to prove the coding loop works |
| Tool pattern | Search/replace editing (not line-number patching) | More reliable for LLMs — no line number drift across multi-edit sessions |
| Sandbox | SandboxPolicy type in harness, unsandboxed in PoC | Surfaces the interface gap without blocking implementation |
| CLI mode | Specced as CLIAdapter, not implemented | Proves the design supports it; implementation is a Layer 4 concern |

---

## Architecture

### How Montara Fits the Three-Repo Model

```
                    openharness (open source)
                   ┌─────────────────────────┐
                   │  harness/ agent/ tools/  │
                   │  providers/ cost/        │
                   │  telemetry/ app/ ...     │
                   │                          │
                   │  examples/               │
                   │    └── montara/           │  ← coding agent PoC
                   └──────────┬───────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
     ghostfin (finance)  $YOUR_VERTICAL   production coding agent
                         (import openharness, (would follow ghostfin's
                          register tools)     separate-repo pattern)
```

Montara is an **in-repo example**, not a separate module. A production coding vertical would follow the ghostfin pattern (separate repo, `go get openharness`). Montara proves the APIs work so that path is viable.

### Import Rules

Montara follows the same import rules as any vertical:
1. Imports `openharness/harness`, `openharness/tools`, `openharness/app`, etc.
2. Registers domain-specific tools, agents, and views at startup
3. Never imported by openharness framework code

---

## File Layout

```
openharness/
├── examples/
│   └── montara/
│       ├── main.go              # Wires openharness + registers coding tools/agent
│       ├── tools/
│       │   ├── file_read.go     # Context-aware file reading with line numbers
│       │   ├── file_edit.go     # Search/replace diff-based editing
│       │   ├── shell_exec.go    # Shell command execution with timeout
│       │   └── git.go           # Git operations (status, diff, log, commit)
│       ├── agents/
│       │   └── pair.go          # Pair programmer agent definition + system prompt
│       └── frontend/
│           └── diff_view.go     # Diff viewer component registration
```

---

## Tool Implementations

Each tool implements the openharness `tools.Tool` interface. All tools receive the working directory from `RunConfig.Metadata["working_dir"]`.

### file_read

**Purpose:** Read files with line numbers, supporting offset/limit for large files.

**Behavior:**
- Returns content with line number prefixes (`  1\t<line>`) so the LLM can reference specific lines
- Truncates files beyond a configurable limit (default: 2000 lines) and reports total line count
- The agent learns to request slices of large files rather than reading everything

**Parameters:**
```go
type FileReadParams struct {
    Path   string `json:"path"`             // Relative to working dir
    Offset int    `json:"offset,omitempty"` // Start line (0-based)
    Limit  int    `json:"limit,omitempty"`  // Max lines to return
}
```

**Returns:** File content with line numbers, total line count, truncation indicator.

### file_edit

**Purpose:** Apply search/replace edits to files. Fails if the search string isn't unique.

**Behavior:**
- Takes `old_string` (text to find) and `new_string` (replacement)
- Fails with an error if `old_string` appears zero times or more than once (unless `replace_all` is set)
- The LLM must read a file before editing it — the tool checks that the old_string matches exactly
- This pattern avoids line-number drift: after an edit changes line counts, subsequent edits still work because they match on content, not position

**Parameters:**
```go
type FileEditParams struct {
    Path       string `json:"path"`
    OldString  string `json:"old_string"`
    NewString  string `json:"new_string"`
    ReplaceAll bool   `json:"replace_all,omitempty"`
}
```

**Returns:** Success confirmation with a unified diff of the change.

### shell_exec

**Purpose:** Execute shell commands with timeout and output capture.

**Behavior:**
- Runs the command in the working directory via `sh -c`
- Captures stdout, stderr, exit code
- Enforces a per-command timeout (default: 120s, configurable via SandboxPolicy)
- In the PoC, runs unsandboxed — the spec section below describes how SandboxPolicy would constrain this

**Parameters:**
```go
type ShellExecParams struct {
    Command string `json:"command"`
    Timeout int    `json:"timeout,omitempty"` // Seconds, overrides default
}
```

**Returns:** stdout, stderr, exit code, whether the command was killed by timeout.

### git

**Purpose:** Structured git operations that return parsed output.

**Behavior:**
- Wraps common git operations: `status`, `diff`, `log`, `add`, `commit`, `branch`
- Returns structured data (e.g., changed files as a list with status indicators) rather than raw git output
- Separate from shell_exec because structured output gives the agent better signal for decision-making

**Parameters:**
```go
type GitParams struct {
    Operation string   `json:"operation"`          // status, diff, log, add, commit, branch
    Args      []string `json:"args,omitempty"`     // Operation-specific args
    Message   string   `json:"message,omitempty"`  // For commit
}
```

**Returns:** Structured output specific to each operation (file list for status, diff text for diff, commit list for log).

---

## Agent Persona

A single `AgentDefinition` registered at startup.

```go
func PairProgrammerAgent() harness.AgentDefinition {
    return harness.AgentDefinition{
        ID:          "montara-pair",
        Name:        "Pair Programmer",
        Description: "A coding assistant that reads, edits, and runs code",
        Runner:      "lite",
        SystemPrompt: pairProgrammerPrompt,
        Tools:       []string{"file_read", "file_edit", "shell_exec", "git"},
    }
}
```

**System prompt principles:**
- Read files before editing — don't guess at content
- Prefer minimal diffs over full rewrites
- Explain changes briefly, don't narrate every step
- Ask for clarification on ambiguous requests
- Don't add features, refactor, or "improve" beyond what was asked

**Specced but not implemented — additional personas:**
- **Code Reviewer** — reads diffs, checks for bugs/security issues, suggests improvements
- **Debugger** — systematic hypothesis-test loop, reads errors, adds instrumentation, narrows root cause

---

## Frontend View

A diff viewer registered via `app.Frontend().RegisterView("diff", montara.DiffViewerView())`.

**Purpose:** Shows file changes the agent has made during the session. Each file_edit tool call produces a unified diff that accumulates in the view.

**Implementation:** A React component that receives diffs via the SSE event stream and renders them with syntax highlighting. Minimal — this is proving the frontend registration API works, not building a full IDE.

---

## Entry Point

```go
// examples/montara/main.go
package main

import (
    "github.com/swiftdiaries/openharness/app"
    "github.com/swiftdiaries/openharness/edition"
    "github.com/swiftdiaries/openharness/examples/montara/agents"
    "github.com/swiftdiaries/openharness/examples/montara/tools"
    "github.com/swiftdiaries/openharness/examples/montara/frontend"
)

func main() {
    a := app.NewApp(app.AppConfig{
        Name:    "Montara",
        DataDir: "~/.montara/data",
        Edition: edition.Lite,
    })

    // Coding tools
    a.Tools().Register("file_read", tools.NewFileReadTool())
    a.Tools().Register("file_edit", tools.NewFileEditTool())
    a.Tools().Register("shell_exec", tools.NewShellExecTool())
    a.Tools().Register("git", tools.NewGitTool())

    // Agent persona
    a.Agents().Seed(agents.PairProgrammerAgent())

    // Coding-specific trace attributes
    a.Telemetry().AddAttributes(map[string]string{
        "vertical":  "coding",
        "language":  "", // populated per-session
        "repo":      "", // populated per-session
    })

    // Frontend view
    a.Frontend().RegisterView("diff", frontend.DiffViewerView())

    a.Run()
}
```

---

## Framework Gaps Surfaced

These are openharness interface changes that Montara's design reveals. Each is tagged with the extraction layer it affects.

### Gap 1: SandboxPolicy (Layer 1 — harness/types.go)

**Problem:** `shell_exec` runs arbitrary commands. A production coding agent needs filesystem scope restrictions, command deny-lists, and optional container isolation. There's no place in `RunConfig` to express these constraints.

**Proposed addition to `harness/types.go`:**

```go
// SandboxPolicy constrains tool execution within a run.
// Nil policy means unrestricted (default for Lite desktop use).
type SandboxPolicy struct {
    AllowedPaths   []string `json:"allowed_paths,omitempty"`   // Filesystem scope (empty = unrestricted)
    DeniedCommands []string `json:"denied_commands,omitempty"` // Command blocklist patterns
    TimeoutSeconds int      `json:"timeout_seconds,omitempty"` // Per-command timeout (0 = default 120s)
    NetworkAccess  bool     `json:"network_access"`            // Allow outbound network calls
}
```

**Addition to `RunConfig`:**

```go
type RunConfig struct {
    RunID     string            `json:"run_id"`
    SessionID string            `json:"session_id"`
    TenantID  string            `json:"tenant_id,omitempty"`
    AgentID   string            `json:"agent_id,omitempty"`
    Mode      string            `json:"mode"`
    Input     string            `json:"input,omitempty"`
    Metadata  map[string]string `json:"metadata,omitempty"`
    Sandbox   *SandboxPolicy    `json:"sandbox,omitempty"`  // NEW — nil means unrestricted
}
```

**Impact:** This is a backward-compatible addition (pointer field, nil = no change in behavior). Tools that execute external commands check `cfg.Sandbox` and enforce constraints. The harness runner passes the policy through to tool execution context.

**Who uses this beyond Montara:**
- Any vertical that runs user-provided code (data science, DevOps)
- Enterprise K8s runner could map SandboxPolicy → Pod SecurityContext
- Lambda runner could map it → Lambda execution role permissions

### Gap 2: CLI Adapter (Layer 4 — app/)

**Problem:** The app scaffold assumes a Wails desktop entry point. A CLI-first coding agent needs an alternative that renders to the terminal while using the same gateway/SSE backend.

**Proposed design:**

```go
// app/adapter.go
type AppAdapter interface {
    // Start launches the UI (Wails window, terminal TUI, or headless)
    Start(ctx context.Context, gateway *gateway.Server) error
    // Shutdown gracefully stops the UI
    Shutdown(ctx context.Context) error
}

// app/wails_adapter.go — existing behavior
type WailsAdapter struct { ... }

// app/cli_adapter.go — new for CLI-first verticals
type CLIAdapter struct { ... }
```

**AppConfig gains an Adapter field:**

```go
type AppConfig struct {
    Name    string
    DataDir string
    Edition edition.Edition
    Adapter AppAdapter  // nil defaults to WailsAdapter
}
```

**Impact:** This is a Layer 4 concern. The gateway, SSE, agent loop, and tool execution are all adapter-agnostic — the adapter only handles rendering. Montara's PoC uses the default WailsAdapter; the CLI path is specced for future implementation.

### Gap 3: Streaming Command Output (Layer 4 — gateway/)

**Problem:** Long-running commands (builds, test suites) produce output over minutes. The `StoreBackedEventStream` polls the session store, adding latency. Coding agents need lower-latency feedback.

**Proposed design:** Add a direct-push option alongside polling:

```go
// harness/interfaces.go — extend EventStream
type EventStream interface {
    Next() (Event, error)
    Close() error
}

// For Lite (in-process), LiteRunner already uses a channel internally.
// Expose a DirectEventStream that wraps the channel without polling:
type DirectEventStream struct {
    ch     <-chan Event
    closed bool
}
```

**Impact:** No interface change needed — `EventStream` is already an interface. `DirectEventStream` is a new implementation alongside `StoreBackedEventStream`. The gateway SSE handler consumes either transparently.

### Gap 4: Working Directory Context (No framework change)

Coding agents operate on a specific directory. This is handled by `RunConfig.Metadata["working_dir"]` — no interface change required. Tools read it from context. This is the intended use of the `Metadata` map.

---

## Comparison with Goose

Goose (block/goose) is a Rust-based coding agent under the Agentic AI Foundation. This comparison shows how Goose's architecture maps to openharness, validating the framework's generality.

| Goose Component | openharness Equivalent | Gap? |
|----------------|----------------------|------|
| Rust core + Think→Act→Observe loop | `agent/loop.go` | No |
| MCP tool extensions (70+) | `tools/mcp/bridge.go` | No |
| Provider registry (15+ LLMs) | `providers/registry.go` | No |
| Recipe system (YAML config) | Registration APIs (Go code) | Different philosophy, compatible |
| OpenTelemetry tracing | `telemetry/` dual-mode | No |
| Electron desktop app | Wails desktop app | No |
| CLI interface | CLIAdapter (Gap 2) | Specced |
| Tool confirmation routing | SandboxPolicy (Gap 1) | Specced |
| Session context (Tokio task-local) | `sessions/store.go` | No |
| V8 embedded scripting | Not needed (Go tools) | N/A |

**Key takeaway:** Goose's domain-specific surface is almost entirely tools and system prompts — the same pattern Montara follows. The framework infrastructure (agent loop, providers, tracing, session management) is generic. This validates openharness's bet that verticals are thin wrappers around a shared core.

---

## Implementation Scope

### What Gets Built (PoC)

1. `examples/montara/main.go` — entry point wiring openharness
2. `examples/montara/tools/file_read.go` — file reading with line numbers
3. `examples/montara/tools/file_edit.go` — search/replace editing
4. `examples/montara/tools/shell_exec.go` — command execution with timeout
5. `examples/montara/tools/git.go` — structured git operations
6. `examples/montara/agents/pair.go` — pair programmer agent definition
7. `examples/montara/frontend/diff_view.go` — diff viewer registration stub
8. `SandboxPolicy` type in `harness/types.go` (framework addition)

### What Gets Specced Only

1. CLI adapter (`app/cli_adapter.go`)
2. Direct-push event streaming (`DirectEventStream`)
3. Additional agent personas (code reviewer, debugger)
4. LSP bridge tool (go-to-definition, references, diagnostics)
5. Tree-sitter tool (AST-level code search)
6. Container-based sandbox execution
7. Separate-repo production vertical pattern

### When to Build

- **SandboxPolicy type:** Add to Layer 1 (it's a types.go addition, zero implementation cost)
- **Montara tools + agent:** After Layer 2 lands (needs `tools.Tool` interface and `agent/loop.go`)
- **Frontend view:** After Layer 4 lands (needs `app.Frontend().RegisterView()`)
- **CLI adapter, streaming:** Future Layer 4 enhancements

---

## Verification

### PoC Verification (Post Layer 2)

1. `cd openharness && go build ./examples/montara/...` — compiles
2. Register tools, seed agent, verify `app.Tools().List()` includes all 4
3. Run the agent loop with a simple prompt ("read main.go and add a comment") — verify Think→Act→Observe produces correct file_read → file_edit sequence
4. Verify trace spans include `vertical=coding` attribute

### Framework Gap Verification (Post Layer 1)

1. `SandboxPolicy` type exists in `harness/types.go`
2. `RunConfig.Sandbox` field is available
3. Existing tests still pass (backward compatible — nil Sandbox = no change)

### Integration Verification (Post Layer 4)

1. Montara Wails app starts and renders chat + diff viewer
2. Agent edits a file, diff appears in the diff viewer via SSE
3. GhostFin still builds and runs (no regression from SandboxPolicy addition)
