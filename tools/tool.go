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
