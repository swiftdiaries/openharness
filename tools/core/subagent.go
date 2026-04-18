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
	return json.Marshal(map[string]string{
		"status":        "delegated",
		"subagent_type": input.SubagentType,
		"description":   input.Description,
		"note":          "Subagent execution will be wired in Layer 4 (gateway)",
	})
}
