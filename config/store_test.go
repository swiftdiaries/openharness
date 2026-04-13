package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := AppConfig{
		DefaultModel: "claude-sonnet-4-20250514",
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file contains valid JSON matching the config.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	var got AppConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}
	if got.DefaultModel != cfg.DefaultModel {
		t.Errorf("DefaultModel = %q, want %q", got.DefaultModel, cfg.DefaultModel)
	}

	// Verify no temp files are left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "config.json" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestSaveOverwritePreservesOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write an initial config.
	original := AppConfig{DefaultModel: "original-model"}
	if err := Save(path, original); err != nil {
		t.Fatalf("initial Save failed: %v", err)
	}

	// Attempt Save to a read-only directory so temp file creation fails.
	badDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(badDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	badPath := filepath.Join(badDir, "config.json")

	// Write a file there first, then make dir read-only.
	if err := Save(badPath, original); err != nil {
		t.Fatalf("setup Save failed: %v", err)
	}
	if err := os.Chmod(badDir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(badDir, 0o700) })

	// This should fail because we can't create a temp file in a read-only dir.
	err := Save(badPath, AppConfig{DefaultModel: "new-model"})
	if err == nil {
		t.Fatal("expected error writing to read-only dir, got nil")
	}

	// Original file should still be intact.
	data, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	var got AppConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("original corrupted: %v", err)
	}
	if got.DefaultModel != "original-model" {
		t.Errorf("original changed to %q", got.DefaultModel)
	}
}

func TestSaveFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := Save(path, Default()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}
