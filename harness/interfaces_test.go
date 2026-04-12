package harness

import (
	"testing"
	"time"
)

// Compile-time interface satisfaction checks.
var (
	_ HarnessRunner = (HarnessRunner)(nil)
	_ SessionStore  = (SessionStore)(nil)
	_ SecretStore   = (SecretStore)(nil)
	_ ArtifactStore = (ArtifactStore)(nil)
	_ ToolRegistry  = (ToolRegistry)(nil)
	_ ChannelRouter = (ChannelRouter)(nil)
	_ AgentStore    = (AgentStore)(nil)
	_ SkillStore    = (SkillStore)(nil)
	_ EventStream   = (EventStream)(nil)
)

func TestInterfacesDefined(t *testing.T) {
	// These assignments verify that each interface type is defined and usable.
	// If any interface is missing or malformed, this file will not compile.
	var _ HarnessRunner
	var _ SessionStore
	var _ SecretStore
	var _ ArtifactStore
	var _ ToolRegistry
	var _ ChannelRouter
	var _ AgentStore
	var _ SkillStore
	t.Log("all 8 boundary interfaces compile successfully")
}

func TestRunConfigFields(t *testing.T) {
	cfg := RunConfig{
		RunID:          "run-1",
		SessionID:      "sess-1",
		TenantID:       "tenant-1",
		HarnessVersion: "v1",
		VaultToken:     "tok",
		ArtifactBucket: "bucket",
		Mode:           "auto",
	}

	if cfg.RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", cfg.RunID, "run-1")
	}
	if cfg.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", cfg.SessionID, "sess-1")
	}
	if cfg.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", cfg.TenantID, "tenant-1")
	}
	if cfg.HarnessVersion != "v1" {
		t.Errorf("HarnessVersion = %q, want %q", cfg.HarnessVersion, "v1")
	}
	if cfg.Mode != "auto" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "auto")
	}
}

func TestAgentDefinitionVersion(t *testing.T) {
	def := AgentDefinition{
		ID:      "agent-1",
		Name:    "test-agent",
		Version: 3,
	}

	if def.Version != 3 {
		t.Errorf("Version = %d, want 3", def.Version)
	}
}

func TestEventTypes(t *testing.T) {
	expected := map[string]string{
		"EventRunStarted":   EventRunStarted,
		"EventChunk":        EventChunk,
		"EventToolCall":     EventToolCall,
		"EventToolResult":   EventToolResult,
		"EventRunCompleted": EventRunCompleted,
		"EventError":        EventError,
		"EventAskUser":      EventAskUser,
		"EventActivity":     EventActivity,
		"EventModeSwitch":   EventModeSwitch,
		"EventCheckpoint":   EventCheckpoint,
	}

	vals := map[string]bool{}
	for name, val := range expected {
		if val == "" {
			t.Errorf("%s is empty", name)
		}
		if vals[val] {
			t.Errorf("%s has duplicate value %q", name, val)
		}
		vals[val] = true
	}

	if len(expected) != 10 {
		t.Errorf("expected 10 event types, got %d", len(expected))
	}
}

func TestAgentDefinitionRunnerField(t *testing.T) {
	def := AgentDefinition{
		Runner:         "lite",
		FallbackRunner: "k8s",
	}

	if def.Runner != "lite" {
		t.Errorf("Runner = %q, want %q", def.Runner, "lite")
	}
	if def.FallbackRunner != "k8s" {
		t.Errorf("FallbackRunner = %q, want %q", def.FallbackRunner, "k8s")
	}
}

func TestRunnerRegistryGetUnregistered(t *testing.T) {
	reg := NewRunnerRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered runner")
	}
}

func TestSnapshotFields(t *testing.T) {
	now := time.Now()
	snap := Snapshot{
		SessionID:  "sess-1",
		RunID:      "run-1",
		Sequence:   5,
		Checkpoint: 42,
		AgentConfig: AgentDefinition{
			ID:   "agent-1",
			Name: "test",
		},
		Status: "active",
	}

	if snap.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", snap.SessionID, "sess-1")
	}
	if snap.RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", snap.RunID, "run-1")
	}
	if snap.Sequence != 5 {
		t.Errorf("Sequence = %d, want 5", snap.Sequence)
	}
	if snap.Checkpoint != 42 {
		t.Errorf("Checkpoint = %d, want 42", snap.Checkpoint)
	}
	if snap.AgentConfig.ID != "agent-1" {
		t.Errorf("AgentConfig.ID = %q, want %q", snap.AgentConfig.ID, "agent-1")
	}
	if snap.Status != "active" {
		t.Errorf("Status = %q, want %q", snap.Status, "active")
	}

	_ = now // suppress unused variable if we need it later
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
