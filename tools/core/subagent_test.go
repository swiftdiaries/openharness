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
