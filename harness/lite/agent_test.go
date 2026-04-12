package lite

import (
	"context"
	"errors"
	"testing"

	"github.com/swiftdiaries/openharness/harness"
)

func TestLiteAgentStoreCreate(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	def, err := store.Create(ctx, "tenant-1", harness.AgentDefinition{
		Name:  "my-agent",
		Model: "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if def.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if def.Version != 1 {
		t.Errorf("Version = %d, want 1", def.Version)
	}
	if def.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", def.TenantID, "tenant-1")
	}
	if def.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if def.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestLiteAgentStoreGet(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	created, err := store.Create(ctx, "tenant-1", harness.AgentDefinition{
		Name:  "get-agent",
		Model: "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != "get-agent" {
		t.Errorf("Name = %q, want %q", got.Name, "get-agent")
	}
	if got.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", got.Model, "claude-sonnet-4-20250514")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}

	// Get non-existent agent.
	_, err = store.Get(ctx, "no-such-agent")
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("Get non-existent: got %v, want ErrNotFound", err)
	}
}

func TestLiteAgentStoreUpdate(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	created, err := store.Create(ctx, "tenant-1", harness.AgentDefinition{
		Name:  "original",
		Model: "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "updated-name"
	updated, err := store.Update(ctx, created.ID, 1, harness.AgentPatch{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Version != 2 {
		t.Errorf("Version = %d, want 2", updated.Version)
	}
	if updated.Name != "updated-name" {
		t.Errorf("Name = %q, want %q", updated.Name, "updated-name")
	}
	// Model should be preserved (nil in patch).
	if updated.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q (should be preserved)", updated.Model, "claude-sonnet-4-20250514")
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) || updated.UpdatedAt.Equal(created.UpdatedAt) {
		// UpdatedAt should be >= created time (may be same if very fast).
		// At minimum, it should not be before.
		if updated.UpdatedAt.Before(created.UpdatedAt) {
			t.Error("UpdatedAt should not be before original UpdatedAt")
		}
	}
}

func TestLiteAgentStoreVersionConflict(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	created, err := store.Create(ctx, "tenant-1", harness.AgentDefinition{
		Name:  "conflict-agent",
		Model: "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update to version 2.
	newName := "v2-name"
	_, err = store.Update(ctx, created.ID, 1, harness.AgentPatch{Name: &newName})
	if err != nil {
		t.Fatalf("Update to v2: %v", err)
	}

	// Try updating with stale version 1 — should fail.
	staleName := "stale-name"
	_, err = store.Update(ctx, created.ID, 1, harness.AgentPatch{Name: &staleName})
	if !errors.Is(err, harness.ErrConflict) {
		t.Errorf("stale update: got %v, want ErrConflict", err)
	}
}

func TestLiteAgentStoreArchive(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	created, err := store.Create(ctx, "tenant-1", harness.AgentDefinition{
		Name: "archive-me",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	archived, err := store.Archive(ctx, created.ID)
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Fatal("ArchivedAt should be set after archive")
	}

	// Archive again — should fail.
	_, err = store.Archive(ctx, created.ID)
	if !errors.Is(err, harness.ErrArchived) {
		t.Errorf("double archive: got %v, want ErrArchived", err)
	}

	// Archive non-existent — should fail.
	_, err = store.Archive(ctx, "no-such-agent")
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("archive non-existent: got %v, want ErrNotFound", err)
	}

	// Update archived agent — should fail.
	newName := "new-name"
	_, err = store.Update(ctx, created.ID, archived.Version, harness.AgentPatch{Name: &newName})
	if !errors.Is(err, harness.ErrArchived) {
		t.Errorf("update archived: got %v, want ErrArchived", err)
	}
}

func TestLiteAgentStoreList(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	// Create agents in two tenants.
	for i := 0; i < 3; i++ {
		_, err := store.Create(ctx, "tenant-a", harness.AgentDefinition{
			Name: "agent-a",
		})
		if err != nil {
			t.Fatalf("Create tenant-a agent %d: %v", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := store.Create(ctx, "tenant-b", harness.AgentDefinition{
			Name: "agent-b",
		})
		if err != nil {
			t.Fatalf("Create tenant-b agent %d: %v", i, err)
		}
	}

	// List tenant-a — expect 3.
	listA, err := store.List(ctx, "tenant-a", harness.ListOpts{})
	if err != nil {
		t.Fatalf("List tenant-a: %v", err)
	}
	if len(listA) != 3 {
		t.Errorf("tenant-a count = %d, want 3", len(listA))
	}

	// List tenant-b — expect 2.
	listB, err := store.List(ctx, "tenant-b", harness.ListOpts{})
	if err != nil {
		t.Fatalf("List tenant-b: %v", err)
	}
	if len(listB) != 2 {
		t.Errorf("tenant-b count = %d, want 2", len(listB))
	}

	// Archived agents should not appear.
	if len(listA) > 0 {
		_, err = store.Archive(ctx, listA[0].ID)
		if err != nil {
			t.Fatalf("Archive: %v", err)
		}
		listA2, err := store.List(ctx, "tenant-a", harness.ListOpts{})
		if err != nil {
			t.Fatalf("List after archive: %v", err)
		}
		if len(listA2) != 2 {
			t.Errorf("tenant-a count after archive = %d, want 2", len(listA2))
		}
	}

	// List with limit.
	limited, err := store.List(ctx, "tenant-b", harness.ListOpts{Limit: 1})
	if err != nil {
		t.Fatalf("List with limit: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("limited count = %d, want 1", len(limited))
	}
}

func TestLiteAgentStoreClone(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	original, err := store.Create(ctx, "tenant-1", harness.AgentDefinition{
		Name:     "original",
		Model:    "claude-sonnet-4-20250514",
		Autonomy: "auto",
		Tools:    []string{"tool-a", "tool-b"},
		Metadata: map[string]string{"key": "value"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cloned, err := store.Clone(ctx, original.ID, "cloned-agent")
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	if cloned.ID == original.ID {
		t.Error("cloned ID should differ from original")
	}
	if cloned.Name != "cloned-agent" {
		t.Errorf("Name = %q, want %q", cloned.Name, "cloned-agent")
	}
	if cloned.Version != 1 {
		t.Errorf("Version = %d, want 1", cloned.Version)
	}
	if cloned.Model != original.Model {
		t.Errorf("Model = %q, want %q", cloned.Model, original.Model)
	}
	if cloned.Autonomy != original.Autonomy {
		t.Errorf("Autonomy = %q, want %q", cloned.Autonomy, original.Autonomy)
	}
	if cloned.ArchivedAt != nil {
		t.Error("cloned ArchivedAt should be nil")
	}

	// Verify deep copy — mutating clone tools should not affect original.
	if len(cloned.Tools) != 2 {
		t.Fatalf("cloned Tools len = %d, want 2", len(cloned.Tools))
	}
	cloned.Tools[0] = "mutated"
	refetched, _ := store.Get(ctx, original.ID)
	if refetched.Tools[0] != "tool-a" {
		t.Errorf("original tools mutated via clone: %v", refetched.Tools)
	}

	// Clone non-existent — should fail.
	_, err = store.Clone(ctx, "no-such-agent", "nope")
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("clone non-existent: got %v, want ErrNotFound", err)
	}
}

func TestLiteAgentStoreSeedData(t *testing.T) {
	store := NewLiteAgentStoreWithSeeds()
	ctx := context.Background()

	list, err := store.List(ctx, "lite", harness.ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) < 5 {
		t.Errorf("seed agent count = %d, want >= 5", len(list))
	}

	// Verify all seeds have expected fields.
	names := make(map[string]bool)
	for _, a := range list {
		names[a.Name] = true
		if a.Model != "claude-sonnet-4-20250514" {
			t.Errorf("agent %q: Model = %q, want claude-sonnet-4-20250514", a.Name, a.Model)
		}
		if a.TenantID != "lite" {
			t.Errorf("agent %q: TenantID = %q, want lite", a.Name, a.TenantID)
		}
		if a.Version != 1 {
			t.Errorf("agent %q: Version = %d, want 1", a.Name, a.Version)
		}
	}

	expectedNames := []string{"harness", "fund-analyst", "analyst", "scribe", "executive-assistant"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("missing seed agent %q", name)
		}
	}
}

func TestLiteAgentStoreListVersions(t *testing.T) {
	store := NewLiteAgentStore()
	ctx := context.Background()

	// ListVersions always returns ErrNotFound for Lite.
	_, err := store.ListVersions(ctx, "any-id")
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("ListVersions: got %v, want ErrNotFound", err)
	}
}
