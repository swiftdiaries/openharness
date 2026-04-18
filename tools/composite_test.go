package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

// stubTool is a minimal Tool implementation for testing.
type stubTool struct {
	defs   []tools.ToolDefinition
	called string
}

func (s *stubTool) Definitions() []tools.ToolDefinition { return s.defs }

func (s *stubTool) Execute(_ context.Context, name string, _ json.RawMessage) (json.RawMessage, error) {
	s.called = name
	return json.RawMessage(`{"tool":"` + name + `"}`), nil
}

func TestCompositeDefinitions(t *testing.T) {
	a := &stubTool{defs: []tools.ToolDefinition{{Name: "tool_a", Description: "a"}}}
	b := &stubTool{defs: []tools.ToolDefinition{{Name: "tool_b", Description: "b"}}}

	c := tools.NewComposite(a, b)
	defs := c.Definitions()

	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
	names := map[string]bool{defs[0].Name: true, defs[1].Name: true}
	if !names["tool_a"] || !names["tool_b"] {
		t.Errorf("unexpected definition names: %v", names)
	}
}

func TestCompositeExecute(t *testing.T) {
	a := &stubTool{defs: []tools.ToolDefinition{{Name: "tool_a"}}}
	b := &stubTool{defs: []tools.ToolDefinition{{Name: "tool_b"}}}

	c := tools.NewComposite(a, b)

	// Execute tool_b — should delegate to b, not a.
	_, err := c.Execute(context.Background(), "tool_b", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.called != "tool_b" {
		t.Errorf("expected b.called=tool_b, got %q", b.called)
	}
	if a.called != "" {
		t.Errorf("expected a not called, got %q", a.called)
	}
}

func TestCompositeExecuteNotFound(t *testing.T) {
	a := &stubTool{defs: []tools.ToolDefinition{{Name: "tool_a"}}}
	c := tools.NewComposite(a)

	_, err := c.Execute(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}
