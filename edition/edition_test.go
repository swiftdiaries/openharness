package edition

import "testing"

func TestEditionDefaults(t *testing.T) {
	e := Current()
	if e.Name != "Lite" {
		t.Errorf("default Name = %q, want Lite", e.Name)
	}
	if !e.IsLimited() {
		t.Error("Lite should be limited")
	}
	if !e.AgentLimitReached(5) {
		t.Error("Lite should be at limit with 5 agents")
	}
	if e.AgentLimitReached(4) {
		t.Error("Lite should not be at limit with 4 agents")
	}
	// Verify Lite factory field values.
	if e.MaxAgents != 5 {
		t.Errorf("Lite MaxAgents = %d, want 5", e.MaxAgents)
	}
	if e.MaxTeams != 1 {
		t.Errorf("Lite MaxTeams = %d, want 1", e.MaxTeams)
	}
	if e.HasRBAC {
		t.Error("Lite should not have RBAC")
	}
	if e.DBBackend != "sqlite" {
		t.Errorf("Lite DBBackend = %q, want sqlite", e.DBBackend)
	}
}

func TestEditionEnterprise(t *testing.T) {
	SetEdition(Enterprise())
	defer SetEdition(Lite())

	e := Current()
	if e.Name != "Enterprise" {
		t.Errorf("Name = %q, want Enterprise", e.Name)
	}
	if e.IsLimited() {
		t.Error("Enterprise should not be limited")
	}
	if e.AgentLimitReached(100) {
		t.Error("Enterprise should never be at agent limit")
	}
	// Verify Enterprise factory field values.
	if e.MaxAgents != 0 {
		t.Errorf("Enterprise MaxAgents = %d, want 0", e.MaxAgents)
	}
	if !e.HasRBAC {
		t.Error("Enterprise should have RBAC")
	}
	if !e.HasSync {
		t.Error("Enterprise should have Sync")
	}
	if e.DBBackend != "postgresql" {
		t.Errorf("Enterprise DBBackend = %q, want postgresql", e.DBBackend)
	}
}

func TestEditionConcurrency(t *testing.T) {
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			SetEdition(Enterprise())
			SetEdition(Lite())
		}
		close(done)
	}()
	for i := 0; i < 1000; i++ {
		e := Current()
		_ = e.Name
	}
	<-done
}

func TestLiteIsLimited(t *testing.T) {
	SetEdition(Lite())
	if !Current().IsLimited() {
		t.Fatal("Lite should be limited")
	}
	if Current().HasScheduler {
		t.Fatal("Lite should not have scheduler")
	}
}

func TestEnterpriseIsNotLimited(t *testing.T) {
	SetEdition(Enterprise())
	if Current().IsLimited() {
		t.Fatal("Enterprise should not be limited")
	}
	if !Current().HasScheduler {
		t.Fatal("Enterprise should have scheduler")
	}
}
