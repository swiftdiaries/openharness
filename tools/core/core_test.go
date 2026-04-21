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
	cases := []struct {
		name string
		want tools.ToolEffect
	}{
		{"web_search", tools.ToolEffectRead},
		{"web_fetch", tools.ToolEffectRead},
		{"read_file", tools.ToolEffectRead},
		{"list_files", tools.ToolEffectRead},
		{"write_file", tools.ToolEffectMutate},
		{"edit_file", tools.ToolEffectMutate},
		{"exec", tools.ToolEffectNeutral},
		{"memory_search", tools.ToolEffectRead},
		{"memory_get", tools.ToolEffectRead},
		{"memory_store", tools.ToolEffectMutate},
		{"ask_user_question", tools.ToolEffectInteractive},
		{"task_get", tools.ToolEffectRead},
		{"task_list", tools.ToolEffectRead},
		{"task_create", tools.ToolEffectMutate},
		{"task_update", tools.ToolEffectMutate},
		{"knowledge_graph_search", tools.ToolEffectRead},
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

func TestRegister_RejectsEmptyWorkspacePath(t *testing.T) {
	r := tools.NewRegistry()
	err := Register(r, Config{WorkspacePath: ""})
	if err == nil {
		t.Fatal("expected error on empty WorkspacePath")
	}
	if !strings.Contains(err.Error(), "WorkspacePath") {
		t.Fatalf("expected WorkspacePath in error, got %v", err)
	}
}
