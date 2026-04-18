package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/swiftdiaries/openharness/tools"
)

func TestValidatePath_BlocksSymlinkEscape(t *testing.T) {
	workspace := t.TempDir()
	fs := NewFilesystem(workspace)

	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "secret.txt"), []byte("secret"), 0644)
	symlink := filepath.Join(workspace, "escape")
	os.Symlink(target, symlink)

	_, err := fs.ValidatePath("escape/secret.txt")
	if err == nil {
		t.Error("expected error for symlink escape, got nil")
	}
}

func TestValidatePath_AllowsNormalPaths(t *testing.T) {
	workspace := t.TempDir()
	fs := NewFilesystem(workspace)

	os.MkdirAll(filepath.Join(workspace, "subdir"), 0755)
	os.WriteFile(filepath.Join(workspace, "subdir", "file.txt"), []byte("ok"), 0644)

	resolved, err := fs.ValidatePath("subdir/file.txt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resolved == "" {
		t.Error("expected non-empty resolved path")
	}
}

func TestValidatePath_BlocksTraversal(t *testing.T) {
	workspace := t.TempDir()
	fs := NewFilesystem(workspace)

	_, err := fs.ValidatePath("../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestValidatePath_AllowsMissingFiles(t *testing.T) {
	workspace := t.TempDir()
	fs := NewFilesystem(workspace)

	resolved, err := fs.ValidatePath("newfile.txt")
	if err != nil {
		t.Errorf("unexpected error for missing file: %v", err)
	}
	if resolved == "" {
		t.Error("expected non-empty resolved path")
	}
}

func TestWriteGuard_PredefinedBlocksIdentity(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	fs.SetWriteGuard("predefined", false)

	args, _ := json.Marshal(map[string]string{"path": "IDENTITY.md", "content": "hacked"})
	_, err := fs.Execute(context.Background(), "write_file", args)
	if err == nil {
		t.Error("expected write to IDENTITY.md to be blocked for predefined agent")
	}
}

func TestWriteGuard_PredefinedBlocksAgents(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	fs.SetWriteGuard("predefined", false)

	args, _ := json.Marshal(map[string]string{"path": "AGENTS.md", "content": "hacked"})
	_, err := fs.Execute(context.Background(), "write_file", args)
	if err == nil {
		t.Error("expected write to AGENTS.md to be blocked for predefined agent")
	}
}

func TestWriteGuard_PredefinedAllowsSoulWhenEvolve(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	fs.SetWriteGuard("predefined", true)

	args, _ := json.Marshal(map[string]string{"path": "SOUL.md", "content": "evolved"})
	_, err := fs.Execute(context.Background(), "write_file", args)
	if err != nil {
		t.Errorf("expected SOUL.md write to succeed with self_evolve=true, got %v", err)
	}
}

func TestWriteGuard_PredefinedBlocksSoulWhenNoEvolve(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	fs.SetWriteGuard("predefined", false)

	args, _ := json.Marshal(map[string]string{"path": "SOUL.md", "content": "hacked"})
	_, err := fs.Execute(context.Background(), "write_file", args)
	if err == nil {
		t.Error("expected SOUL.md write to be blocked without self_evolve")
	}
}

func TestWriteGuard_OpenAllowsAll(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	fs.SetWriteGuard("open", false)

	for _, name := range []string{"SOUL.md", "IDENTITY.md", "AGENTS.md"} {
		args, _ := json.Marshal(map[string]string{"path": name, "content": "content"})
		_, err := fs.Execute(context.Background(), "write_file", args)
		if err != nil {
			t.Errorf("open agent should be able to write %s, got %v", name, err)
		}
	}
}

func TestWriteGuard_NoGuardAllowsAll(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	// No SetWriteGuard call — agentType is ""

	for _, name := range []string{"SOUL.md", "IDENTITY.md", "AGENTS.md"} {
		args, _ := json.Marshal(map[string]string{"path": name, "content": "content"})
		_, err := fs.Execute(context.Background(), "write_file", args)
		if err != nil {
			t.Errorf("no guard should allow write to %s, got %v", name, err)
		}
	}
}

func TestFilesystemEffects(t *testing.T) {
	f := NewFilesystem(t.TempDir())
	want := map[string]tools.ToolEffect{
		"read_file":  tools.ToolEffectRead,
		"write_file": tools.ToolEffectMutate,
		"list_files": tools.ToolEffectRead,
		"edit_file":  tools.ToolEffectMutate,
	}
	for _, d := range f.Definitions() {
		if got := d.Effects; got != want[d.Name] {
			t.Errorf("%s: Effects = %v, want %v", d.Name, got, want[d.Name])
		}
	}
}
