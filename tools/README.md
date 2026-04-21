# tools

## What is this package

The `tools` package defines the generic tool surface the openharness agent loop calls. The `tools/core` subpackage ships built-in implementations verticals can use unmodified.

## The Tool interface

`ToolDefinition` (see `tools/tool.go:62`) describes one callable surface the LLM sees. A single Tool may publish several definitions — effects are per-definition, not per-Tool.

```go
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	// Effects is framework-internal metadata and MUST NOT be serialized to the LLM — the `json:"-"` tag is load-bearing.
	Effects ToolEffect `json:"-"`
}
```

`Tool` (see `tools/tool.go:75`) is the handler dispatched by name:

```go
type Tool interface {
	Definitions() []ToolDefinition
	Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error)
}
```

## Effects taxonomy

Every definition registered with a `Registry` must carry a non-Unspecified `ToolEffect`. The agent loop reads this value to decide whether a call increments or resets the read-streak, and whether the definition is allowed in `ModePlan`.

| Effect | Meaning | Example tool |
|--------|---------|--------------|
| Read | Inspects state without modifying it; increments read-streak; allowed in ModePlan | `read_file`, `web_fetch` |
| Mutate | Modifies workspace/memory/KG state; resets read-streak; blocked in ModePlan | `write_file`, `memory_store` |
| Neutral | Neither read nor mutate for loop-counting; allowed in ModePlan | `exec` |
| Interactive | Yields to user input; resets read-streak; allowed in ModePlan | `ask_user` |

`ToolEffectUnspecified` is the zero value. `Registry.Register` (see `tools/registry.go:42`) refuses any definition that still carries it and returns an error naming the offending definition; nothing is added to the registry in that case. Each definition — not each Tool — carries its own Effect. A single tool like `filesystem` publishes multiple definitions with different effects (`read_file` is Read, `write_file` is Mutate).

## Registering built-ins

Callers bring their own `Registry` and hand it to `core.Register` along with a `core.Config`. See `tools/core/core.go:15` for the full `Config` struct and `tools/core/core.go:41` for `Register`.

```go
r := tools.NewRegistry()
if err := core.Register(r, core.Config{
    WorkspacePath: "/abs/path",
    MemoryPath:    "/abs/path/memory.json",
}); err != nil { ... }
fmt.Println(r.EffectsFor("read_file")) // -> Read
```

Omitted optional fields cause the corresponding tool to be skipped rather than failing:

- `MemoryPath == ""` skips `memory_*`.
- `KnowledgeStore == nil` skips `knowledge_graph_search`.
- `Subagent.Subagents` empty skips `agent`.

## Authoring a new tool

Implement the `Tool` interface, classify every definition with a non-Unspecified Effect, and register it under a category. A minimal Read-effect tool:

```go
type MyTool struct{}

func (MyTool) Definitions() []tools.ToolDefinition {
    return []tools.ToolDefinition{{
        Name: "my_op", Description: "...",
        Parameters: json.RawMessage(`{"type":"object","properties":{"arg":{"type":"string"}},"required":["arg"]}`),
        Effects: tools.ToolEffectRead,
    }}
}

func (MyTool) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
    // parse args, do work, return JSON
    return json.Marshal(map[string]string{"ok": "true"})
}
```

Register it under a category of your choice:

```go
if err := r.Register("my_category", MyTool{}); err != nil { ... }
```

## External-content wrapping

Any tool whose output includes content that originated outside the process — web pages, search snippets, knowledge-graph entries — should wrap its string outputs with `SanitizeExternal(...)` from `tools/core/external_output.go` (or with `tools.WrapExternalContent` + `tools.ScrubCredentials` directly). Tools that return process-local content (filesystem reads, exec output, memory reads) should use `SanitizeRead(...)`. NOTE: `external_output.go` is being introduced in Task 4 of this plan; if it does not yet exist in your working copy, treat this section as aspirational and pair it with Task 4 when wiring a new external-content tool.

## Testing conventions

Every built-in tool ships with a `*_test.go` file that exercises `Definitions()` (shape, effects), `Execute` (happy path), and security edges (SSRF, write-guards, shell metachars). The canonical cross-tool D4 matrix lives in `tools/core/core_test.go` — new tools should extend that matrix with their definition+effects row so the single-source-of-truth stays in sync.
