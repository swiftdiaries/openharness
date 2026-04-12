package lite

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time check that LiteArtifactStore implements ArtifactStore.
var _ harness.ArtifactStore = (*LiteArtifactStore)(nil)

// LiteArtifactStore implements ArtifactStore using the local filesystem.
// Files are stored at {baseDir}/{tenantID}/{runID}/{name}.
type LiteArtifactStore struct {
	baseDir string
	mu      sync.Mutex
}

// NewLiteArtifactStore creates a new LiteArtifactStore rooted at baseDir.
func NewLiteArtifactStore(baseDir string) *LiteArtifactStore {
	return &LiteArtifactStore{baseDir: baseDir}
}

// Put writes data from an io.Reader to {baseDir}/{tenantID}/{runID}/{name}.
// It returns the relative key path "{tenantID}/{runID}/{name}".
func (s *LiteArtifactStore) Put(_ context.Context, tenantID, runID, name string, data io.Reader) (string, error) {
	dir := filepath.Join(s.baseDir, tenantID, runID)
	key := filepath.Join(tenantID, runID, name)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("artifact put: mkdir %s: %w", dir, err)
	}

	fpath := filepath.Join(s.baseDir, key)
	f, err := os.Create(fpath)
	if err != nil {
		return "", fmt.Errorf("artifact put: create %s: %w", fpath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return "", fmt.Errorf("artifact put: write %s: %w", fpath, err)
	}

	return key, nil
}

// Get opens the file at {baseDir}/{key} and returns it as an io.ReadCloser.
// Returns ErrNotFound if the file does not exist.
func (s *LiteArtifactStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	fpath := filepath.Join(s.baseDir, key)
	f, err := os.Open(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: artifact %q", harness.ErrNotFound, key)
		}
		return nil, fmt.Errorf("artifact get: open %s: %w", fpath, err)
	}
	return f, nil
}

// List reads directory entries at {baseDir}/{tenantID}/{runID}/ and returns
// ArtifactMeta for each file found.
func (s *LiteArtifactStore) List(_ context.Context, tenantID, runID string) ([]harness.ArtifactMeta, error) {
	dir := filepath.Join(s.baseDir, tenantID, runID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // empty result, not an error
		}
		return nil, fmt.Errorf("artifact list: read dir %s: %w", dir, err)
	}

	var metas []harness.ArtifactMeta
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue // skip files we can't stat
		}
		key := filepath.Join(tenantID, runID, entry.Name())
		metas = append(metas, harness.ArtifactMeta{
			Key:       key,
			Name:      entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}
	return metas, nil
}

// Presign returns a file:// URL for the artifact at {baseDir}/{key}.
// In Lite edition there is no real expiry — the ttl parameter is ignored.
// Returns ErrNotFound if the file does not exist.
func (s *LiteArtifactStore) Presign(_ context.Context, key string, _ time.Duration) (string, error) {
	fpath := filepath.Join(s.baseDir, key)
	if _, err := os.Stat(fpath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: artifact %q", harness.ErrNotFound, key)
		}
		return "", fmt.Errorf("artifact presign: stat %s: %w", fpath, err)
	}

	abs, err := filepath.Abs(fpath)
	if err != nil {
		return "", fmt.Errorf("artifact presign: abs path: %w", err)
	}
	return "file://" + abs, nil
}
