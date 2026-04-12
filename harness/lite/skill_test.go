package lite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/swiftdiaries/openharness/harness"
)

func TestLiteSkillStoreUploadAndResolve(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteSkillStore(dir)
	ctx := context.Background()

	promptContent := "You are a financial analyst skill.\n\nAnalyze fund holdings."

	meta, err := store.Upload(ctx, "tenant-1", harness.SkillMeta{
		Name:        "fund-analysis",
		Category:    "finance",
		Description: "Analyze fund holdings",
		Triggers:    []string{"analyze fund", "fund analysis"},
	}, strings.NewReader(promptContent))
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if meta.ID == "" {
		t.Fatal("Upload() returned empty ID")
	}
	if meta.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", meta.TenantID, "tenant-1")
	}
	if meta.Version != 1 {
		t.Errorf("Version = %d, want 1", meta.Version)
	}
	if meta.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if meta.ArtifactKey == "" {
		t.Error("ArtifactKey is empty")
	}

	// Resolve the uploaded skill.
	skills, err := store.Resolve(ctx, []harness.SkillRef{{SkillID: meta.ID}})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Resolve() returned %d skills, want 1", len(skills))
	}
	if skills[0].Prompt != promptContent {
		t.Errorf("Prompt = %q, want %q", skills[0].Prompt, promptContent)
	}
	if skills[0].Meta.Name != "fund-analysis" {
		t.Errorf("Meta.Name = %q, want %q", skills[0].Meta.Name, "fund-analysis")
	}

	// Resolve unknown skill should return ErrSkillNotFound.
	_, err = store.Resolve(ctx, []harness.SkillRef{{SkillID: "nonexistent"}})
	if !errors.Is(err, harness.ErrSkillNotFound) {
		t.Errorf("Resolve(unknown) error = %v, want ErrSkillNotFound", err)
	}
}

func TestLiteSkillStoreListTenant(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteSkillStore(dir)
	ctx := context.Background()

	// Upload skills in different tenants.
	_, err := store.Upload(ctx, "tenant-a", harness.SkillMeta{
		Name:        "skill-1",
		Description: "First skill",
	}, strings.NewReader("prompt 1"))
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	_, err = store.Upload(ctx, "tenant-a", harness.SkillMeta{
		Name:        "skill-2",
		Description: "Second skill",
	}, strings.NewReader("prompt 2"))
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	_, err = store.Upload(ctx, "tenant-b", harness.SkillMeta{
		Name:        "skill-3",
		Description: "Third skill (different tenant)",
	}, strings.NewReader("prompt 3"))
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	// ListTenant for tenant-a should return 2 skills.
	listA, err := store.ListTenant(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("ListTenant(tenant-a) error: %v", err)
	}
	if len(listA) != 2 {
		t.Errorf("ListTenant(tenant-a) returned %d skills, want 2", len(listA))
	}

	// ListTenant for tenant-b should return 1 skill.
	listB, err := store.ListTenant(ctx, "tenant-b")
	if err != nil {
		t.Fatalf("ListTenant(tenant-b) error: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("ListTenant(tenant-b) returned %d skills, want 1", len(listB))
	}

	// ListTenant for unknown tenant should return empty slice.
	listC, err := store.ListTenant(ctx, "tenant-c")
	if err != nil {
		t.Fatalf("ListTenant(tenant-c) error: %v", err)
	}
	if len(listC) != 0 {
		t.Errorf("ListTenant(tenant-c) returned %d skills, want 0", len(listC))
	}

	// ListPlatform should return empty slice.
	platform, err := store.ListPlatform(ctx)
	if err != nil {
		t.Fatalf("ListPlatform() error: %v", err)
	}
	if len(platform) != 0 {
		t.Errorf("ListPlatform() returned %d skills, want 0", len(platform))
	}
}

func TestLiteSkillStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteSkillStore(dir)
	ctx := context.Background()

	meta, err := store.Upload(ctx, "tenant-1", harness.SkillMeta{
		Name:        "to-delete",
		Description: "Will be deleted",
	}, strings.NewReader("delete me"))
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	// Verify skill is listed.
	list, err := store.ListTenant(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("ListTenant() error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListTenant() returned %d skills, want 1", len(list))
	}

	// Delete the skill.
	err = store.Delete(ctx, "tenant-1", meta.ID)
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// ListTenant should now return empty.
	list, err = store.ListTenant(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("ListTenant() after delete error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ListTenant() after delete returned %d skills, want 0", len(list))
	}

	// Resolve should return ErrSkillNotFound.
	_, err = store.Resolve(ctx, []harness.SkillRef{{SkillID: meta.ID}})
	if !errors.Is(err, harness.ErrSkillNotFound) {
		t.Errorf("Resolve(deleted) error = %v, want ErrSkillNotFound", err)
	}

	// Delete again should return ErrSkillNotFound.
	err = store.Delete(ctx, "tenant-1", meta.ID)
	if !errors.Is(err, harness.ErrSkillNotFound) {
		t.Errorf("Delete(deleted) error = %v, want ErrSkillNotFound", err)
	}
}
