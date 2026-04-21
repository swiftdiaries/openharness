package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeTool struct {
	name   string
	effect ToolEffect
}

func (f fakeTool) Definitions() []ToolDefinition {
	return []ToolDefinition{{Name: f.name, Description: "fake", Effects: f.effect}}
}

func (f fakeTool) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	return nil, nil
}

func TestRegister_RejectsNameCollision(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("cat", fakeTool{name: "foo", effect: ToolEffectRead}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := r.Register("cat", fakeTool{name: "foo", effect: ToolEffectRead})
	if err == nil {
		t.Fatal("expected collision error")
	}
}
