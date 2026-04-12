package harness

import (
	"testing"
)

func TestInterfacesDefined(t *testing.T) {
	// Verify each interface type is usable as a variable type.
	var _ HarnessRunner
	var _ SessionStore
	var _ SecretStore
	var _ ArtifactStore
	var _ ToolRegistry
	var _ ChannelRouter
	var _ AgentStore
	var _ SkillStore
}

func TestRunConfigFields(t *testing.T) {
	cfg := RunConfig{
		RunID:     "run-1",
		SessionID: "sess-1",
		TenantID:  "tenant-1",
		Mode:      "auto",
	}
	if cfg.RunID != "run-1" {
		t.Fatalf("RunID = %q", cfg.RunID)
	}
}

func TestEventTypes(t *testing.T) {
	e := Event{Type: EventRunStarted}
	if e.Type != "run.started" {
		t.Fatalf("Type = %q", e.Type)
	}
}

func TestAgentDefinitionRunnerField(t *testing.T) {
	def := AgentDefinition{
		ID:     "agent-1",
		Runner: "lambda",
	}
	if def.Runner != "lambda" {
		t.Fatalf("Runner = %q, want lambda", def.Runner)
	}
}

func TestSnapshotFields(t *testing.T) {
	s := Snapshot{
		SessionID: "sess-1",
		RunID:     "run-1",
		Sequence:  3,
		Status:    "running",
	}
	if s.Sequence != 3 {
		t.Fatalf("Sequence = %d", s.Sequence)
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrNotFound == nil {
		t.Fatal("ErrNotFound should not be nil")
	}
	if ErrConflict == nil {
		t.Fatal("ErrConflict should not be nil")
	}
	if ErrRunNotActive == nil {
		t.Fatal("ErrRunNotActive should not be nil")
	}
}

func TestSandboxPolicyDefaults(t *testing.T) {
	// Nil sandbox = unrestricted (backward compatible)
	cfg := RunConfig{RunID: "r1", Mode: "auto"}
	if cfg.Sandbox != nil {
		t.Fatal("default Sandbox should be nil")
	}

	// With policy
	cfg.Sandbox = &SandboxPolicy{
		AllowedPaths:   []string{"/tmp/workspace"},
		DeniedCommands: []string{"rm -rf /"},
		TimeoutSeconds: 30,
		NetworkAccess:  false,
	}
	if len(cfg.Sandbox.AllowedPaths) != 1 {
		t.Fatalf("AllowedPaths = %d, want 1", len(cfg.Sandbox.AllowedPaths))
	}
}
