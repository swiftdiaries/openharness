package core

import (
	"encoding/json"
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestAskUserDefinitions(t *testing.T) {
	a := NewAskUser()
	defs := a.Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "ask_user_question" {
		t.Errorf("expected name 'ask_user_question', got %q", defs[0].Name)
	}

	var schema map[string]any
	json.Unmarshal(defs[0].Parameters, &schema)
	props := schema["properties"].(map[string]any)
	if _, ok := props["question"]; !ok {
		t.Error("schema missing 'question' property")
	}
}

func TestAskUserSchemaHasSuggestions(t *testing.T) {
	a := NewAskUser()
	defs := a.Definitions()
	var schema map[string]any
	json.Unmarshal(defs[0].Parameters, &schema)
	props := schema["properties"].(map[string]any)
	sg, ok := props["suggestions"]
	if !ok {
		t.Fatal("schema missing 'suggestions' property")
	}
	sgMap := sg.(map[string]any)
	if sgMap["type"] != "array" {
		t.Errorf("expected suggestions type 'array', got %v", sgMap["type"])
	}
}

// TODO(plan-4): this test asserts the stubbed "waiting_for_user" response.
// When Plan 4 wires the UIBridge channel, rewrite this test to exercise the
// real interactive flow. See docs/superpowers/plans/layer-2/*-plan-4-*.md
// (or the beads issue once filed).
func TestAskUserExecuteReturnsPlaceholder(t *testing.T) {
	a := NewAskUser()
	result, err := a.Execute(nil, "ask_user_question", json.RawMessage(`{"question":"test?"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp map[string]string
	json.Unmarshal(result, &resp)
	if resp["status"] != "waiting_for_user" {
		t.Errorf("expected status 'waiting_for_user', got %q", resp["status"])
	}
}

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
