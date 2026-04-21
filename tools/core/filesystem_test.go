package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestFilesystem_WriteGuardBlocks(t *testing.T) {
	ws := t.TempDir()
	fs := NewFilesystem(ws)
	fs.SetWriteGuard(func(absPath string) error {
		if filepath.Base(absPath) == "BLOCKED.md" {
			return fmt.Errorf("BLOCKED.md is immutable")
		}
		return nil
	})

	args, _ := json.Marshal(map[string]string{"path": "BLOCKED.md", "content": "x"})
	_, err := fs.Execute(context.Background(), "write_file", args)
	if err == nil || !strings.Contains(err.Error(), "BLOCKED.md") {
		t.Fatalf("expected BLOCKED.md error, got %v", err)
	}

	args2, _ := json.Marshal(map[string]string{"path": "ok.md", "content": "x"})
	if _, err := fs.Execute(context.Background(), "write_file", args2); err != nil {
		t.Fatalf("unblocked write failed: %v", err)
	}
}

func TestFilesystem_NoGuardAllowsAll(t *testing.T) {
	ws := t.TempDir()
	fs := NewFilesystem(ws)
	args, _ := json.Marshal(map[string]string{"path": "anything.md", "content": "x"})
	if _, err := fs.Execute(context.Background(), "write_file", args); err != nil {
		t.Fatalf("no guard should allow write: %v", err)
	}
}
