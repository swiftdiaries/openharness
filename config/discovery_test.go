package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverInstructionFiles(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "project", "subdir")
	os.MkdirAll(child, 0o755)

	os.WriteFile(filepath.Join(root, "GHOSTFIN.md"), []byte("# Root instructions"), 0o644)
	os.WriteFile(filepath.Join(root, "project", "GHOSTFIN.md"), []byte("# Project instructions"), 0o644)

	files := DiscoverInstructionFiles(child)
	if len(files) != 2 {
		t.Fatalf("expected 2 instruction files, got %d", len(files))
	}
	if files[0].Content != "# Project instructions" {
		t.Errorf("expected project instructions first, got: %s", files[0].Content)
	}
	if files[1].Content != "# Root instructions" {
		t.Errorf("expected root instructions second, got: %s", files[1].Content)
	}
}

func TestDiscoverInstructionFiles_SizeLimit(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, 5000)
	for i := range big {
		big[i] = 'x'
	}
	os.WriteFile(filepath.Join(root, "GHOSTFIN.md"), big, 0o644)

	files := DiscoverInstructionFiles(root)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if len(files[0].Content) > MaxInstructionFileChars+20 {
		t.Errorf("expected truncated content, got %d chars", len(files[0].Content))
	}
}

func TestDiscoverInstructionFiles_NoneFound(t *testing.T) {
	root := t.TempDir()
	files := DiscoverInstructionFiles(root)
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}
