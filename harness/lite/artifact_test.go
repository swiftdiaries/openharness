package lite

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

func TestLiteArtifactStorePutAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteArtifactStore(dir)
	ctx := context.Background()

	data := []byte("hello, artifacts")
	key, err := store.Put(ctx, "tenant-1", "run-1", "output.csv", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	wantKey := filepath.Join("tenant-1", "run-1", "output.csv")
	if key != wantKey {
		t.Errorf("Put key = %q, want %q", key, wantKey)
	}

	// Verify file exists on disk.
	fpath := filepath.Join(dir, key)
	if _, err := os.Stat(fpath); err != nil {
		t.Fatalf("file not found at %s: %v", fpath, err)
	}

	// Get the artifact back and verify content.
	rc, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Errorf("Get content = %q, want %q", got, data)
	}
}

func TestLiteArtifactStoreList(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteArtifactStore(dir)
	ctx := context.Background()

	files := map[string]string{
		"report.pdf":  "pdf-content-here",
		"summary.txt": "summary of the run",
		"data.json":   `{"key":"value"}`,
	}

	for name, content := range files {
		_, err := store.Put(ctx, "tenant-2", "run-42", name, strings.NewReader(content))
		if err != nil {
			t.Fatalf("Put(%s): %v", name, err)
		}
	}

	metas, err := store.List(ctx, "tenant-2", "run-42")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(metas) != len(files) {
		t.Fatalf("List returned %d items, want %d", len(metas), len(files))
	}

	// Build a map for easy lookup.
	metaMap := make(map[string]harness.ArtifactMeta)
	for _, m := range metas {
		metaMap[m.Name] = m
	}

	for name, content := range files {
		m, ok := metaMap[name]
		if !ok {
			t.Errorf("List missing artifact %q", name)
			continue
		}

		wantKey := filepath.Join("tenant-2", "run-42", name)
		if m.Key != wantKey {
			t.Errorf("artifact %q: Key = %q, want %q", name, m.Key, wantKey)
		}
		if m.Size != int64(len(content)) {
			t.Errorf("artifact %q: Size = %d, want %d", name, m.Size, len(content))
		}
		if m.CreatedAt.IsZero() {
			t.Errorf("artifact %q: CreatedAt is zero", name)
		}
	}
}

func TestLiteArtifactStoreListEmpty(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteArtifactStore(dir)
	ctx := context.Background()

	metas, err := store.List(ctx, "no-tenant", "no-run")
	if err != nil {
		t.Fatalf("List on nonexistent dir: %v", err)
	}
	if metas != nil {
		t.Errorf("List on nonexistent dir = %v, want nil", metas)
	}
}

func TestLiteArtifactStorePresign(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteArtifactStore(dir)
	ctx := context.Background()

	key, err := store.Put(ctx, "tenant-1", "run-1", "chart.png", strings.NewReader("png-bytes"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	url, err := store.Presign(ctx, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("Presign: %v", err)
	}

	if !strings.HasPrefix(url, "file://") {
		t.Errorf("Presign URL = %q, want file:// prefix", url)
	}

	// The URL should contain the absolute path to the file.
	absDir, _ := filepath.Abs(dir)
	if !strings.Contains(url, absDir) {
		t.Errorf("Presign URL %q does not contain base dir %q", url, absDir)
	}
}

func TestLiteArtifactStoreGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteArtifactStore(dir)
	ctx := context.Background()

	_, err := store.Get(ctx, filepath.Join("no-tenant", "no-run", "missing.txt"))
	if err == nil {
		t.Fatal("Get on nonexistent key: expected error, got nil")
	}

	if !strings.Contains(err.Error(), harness.ErrNotFound.Error()) {
		t.Errorf("Get error = %q, want it to wrap ErrNotFound", err)
	}
}

func TestLiteArtifactStorePresignNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewLiteArtifactStore(dir)
	ctx := context.Background()

	_, err := store.Presign(ctx, filepath.Join("no-tenant", "no-run", "missing.txt"), time.Minute)
	if err == nil {
		t.Fatal("Presign on nonexistent key: expected error, got nil")
	}

	if !strings.Contains(err.Error(), harness.ErrNotFound.Error()) {
		t.Errorf("Presign error = %q, want it to wrap ErrNotFound", err)
	}
}
