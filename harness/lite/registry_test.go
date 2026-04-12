package lite

import (
	"context"
	"errors"
	"testing"

	"github.com/swiftdiaries/openharness/harness"
)

func TestLiteToolRegistryListGlobal(t *testing.T) {
	reg := NewLiteToolRegistry()
	reg.AddGlobalTool(harness.ToolDefinition{Name: "read_file", Description: "Read a file", Category: "filesystem", BuiltIn: true})
	reg.AddGlobalTool(harness.ToolDefinition{Name: "write_file", Description: "Write a file", Category: "filesystem", BuiltIn: true})

	got, err := reg.ListGlobal(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d tools, want 2", len(got))
	}

	names := make(map[string]bool)
	for _, def := range got {
		names[def.Name] = true
	}
	if !names["read_file"] || !names["write_file"] {
		t.Errorf("expected read_file and write_file, got %v", names)
	}
}

func TestLiteToolRegistryRegisterAndResolve(t *testing.T) {
	reg := NewLiteToolRegistry()
	ctx := context.Background()

	def := harness.ToolDefinition{Name: "custom_tool", Description: "A custom tool", Category: "custom"}
	if err := reg.RegisterTool(ctx, "tenant-1", def); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	resolved, err := reg.Resolve(ctx, "tenant-1", []string{"custom_tool"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("got %d tools, want 1", len(resolved))
	}
	if resolved[0].Name != "custom_tool" {
		t.Errorf("name = %q, want %q", resolved[0].Name, "custom_tool")
	}
	if resolved[0].TenantID != "tenant-1" {
		t.Errorf("tenant_id = %q, want %q", resolved[0].TenantID, "tenant-1")
	}
	// ID should have been auto-generated.
	if resolved[0].ID == "" {
		t.Error("expected auto-generated ID, got empty string")
	}
}

func TestLiteToolRegistryResolveOverride(t *testing.T) {
	reg := NewLiteToolRegistry()
	ctx := context.Background()

	// Add global "foo".
	reg.AddGlobalTool(harness.ToolDefinition{
		Name:        "foo",
		Description: "global foo",
		Category:    "test",
		BuiltIn:     true,
	})

	// Register tenant "foo" which should override.
	if err := reg.RegisterTool(ctx, "tenant-1", harness.ToolDefinition{
		Name:        "foo",
		Description: "tenant foo",
		Category:    "test",
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	resolved, err := reg.Resolve(ctx, "tenant-1", []string{"foo"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("got %d tools, want 1", len(resolved))
	}
	if resolved[0].Description != "tenant foo" {
		t.Errorf("description = %q, want %q (tenant should override global)", resolved[0].Description, "tenant foo")
	}
}

func TestLiteToolRegistryResolveNotFound(t *testing.T) {
	reg := NewLiteToolRegistry()
	ctx := context.Background()

	_, err := reg.Resolve(ctx, "tenant-1", []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestLiteToolRegistryVerifyValid(t *testing.T) {
	reg := NewLiteToolRegistry()
	ctx := context.Background()

	result, err := reg.VerifyTool(ctx, harness.ToolDefinition{
		Name:        "good_tool",
		Description: "Does good things",
		Category:    "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected Valid=true, got errors: %v", result.Errors)
	}
}

func TestLiteToolRegistryVerifyInvalid(t *testing.T) {
	reg := NewLiteToolRegistry()
	ctx := context.Background()

	result, err := reg.VerifyTool(ctx, harness.ToolDefinition{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected Valid=false for empty tool definition")
	}
	if len(result.Errors) != 3 {
		t.Fatalf("got %d errors, want 3 (name, description, category)", len(result.Errors))
	}
}

func TestLiteToolRegistryDeleteNotFound(t *testing.T) {
	reg := NewLiteToolRegistry()
	ctx := context.Background()

	err := reg.DeleteTool(ctx, "tenant-1", "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestLiteToolRegistryUpdateAndDelete(t *testing.T) {
	reg := NewLiteToolRegistry()
	ctx := context.Background()

	// Register a tool.
	def := harness.ToolDefinition{Name: "my_tool", Description: "Original", Category: "test"}
	if err := reg.RegisterTool(ctx, "tenant-1", def); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	// Find the assigned ID.
	tools, _ := reg.ListTenant(ctx, "tenant-1")
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(tools))
	}
	toolID := tools[0].ID

	// Update it.
	updated := harness.ToolDefinition{Name: "my_tool", Description: "Updated", Category: "test"}
	if err := reg.UpdateTool(ctx, "tenant-1", toolID, updated); err != nil {
		t.Fatalf("UpdateTool: %v", err)
	}

	resolved, _ := reg.Resolve(ctx, "tenant-1", []string{"my_tool"})
	if resolved[0].Description != "Updated" {
		t.Errorf("description = %q, want %q", resolved[0].Description, "Updated")
	}

	// Delete it.
	if err := reg.DeleteTool(ctx, "tenant-1", toolID); err != nil {
		t.Fatalf("DeleteTool: %v", err)
	}

	// Should not be resolvable anymore.
	_, err := reg.Resolve(ctx, "tenant-1", []string{"my_tool"})
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("after delete: error = %v, want ErrNotFound", err)
	}
}

func TestLiteToolRegistryListTenantEmpty(t *testing.T) {
	reg := NewLiteToolRegistry()

	got, err := reg.ListTenant(context.Background(), "no-such-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d tools, want 0 for unknown tenant", len(got))
	}
}
