package tools

import (
	"context"
	"encoding/json"
	"reflect"
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

type fakeMultiTool struct {
	defs []ToolDefinition
}

func (f fakeMultiTool) Definitions() []ToolDefinition { return f.defs }

func (f fakeMultiTool) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
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

func TestCategories_SortsDefinitionsByName(t *testing.T) {
	r := NewRegistry()
	err := r.Register("cat", fakeMultiTool{defs: []ToolDefinition{
		{Name: "tool_z", Description: "fake", Effects: ToolEffectRead},
		{Name: "tool_g", Description: "fake", Effects: ToolEffectRead},
		{Name: "tool_d", Description: "fake", Effects: ToolEffectRead},
		{Name: "tool_a", Description: "fake", Effects: ToolEffectRead},
		{Name: "tool_j", Description: "fake", Effects: ToolEffectRead},
		{Name: "tool_c", Description: "fake", Effects: ToolEffectRead},
		{Name: "tool_m", Description: "fake", Effects: ToolEffectRead},
		{Name: "tool_b", Description: "fake", Effects: ToolEffectRead},
	}})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	cats := r.Categories()
	if len(cats) != 1 {
		t.Fatalf("len(Categories()) = %d, want 1", len(cats))
	}

	got := definitionNames(cats[0].Tools)
	want := []string{"tool_a", "tool_b", "tool_c", "tool_d", "tool_g", "tool_j", "tool_m", "tool_z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Categories()[0].Tools names = %v, want %v", got, want)
	}
}

func TestToolsByCategory_SortsByDefinitionName(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"tool_z", "tool_g", "tool_d", "tool_a", "tool_j", "tool_c", "tool_m", "tool_b"} {
		if err := r.Register("cat", fakeTool{name: name, effect: ToolEffectRead}); err != nil {
			t.Fatalf("Register(%q): %v", name, err)
		}
	}

	got := toolNames(r.ToolsByCategory("cat"))
	want := []string{"tool_a", "tool_b", "tool_c", "tool_d", "tool_g", "tool_j", "tool_m", "tool_z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToolsByCategory names = %v, want %v", got, want)
	}
}

func definitionNames(defs []ToolDefinition) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}

func toolNames(ts []Tool) []string {
	names := make([]string, 0, len(ts))
	for _, t := range ts {
		defs := t.Definitions()
		if len(defs) == 0 {
			names = append(names, "")
			continue
		}
		names = append(names, defs[0].Name)
	}
	return names
}
