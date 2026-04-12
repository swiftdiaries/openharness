package lite

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time check: LiteSkillStore implements SkillStore.
var _ harness.SkillStore = (*LiteSkillStore)(nil)

// LiteSkillStore is a filesystem-based SkillStore for Lite edition.
// Metadata is held in an in-memory map protected by a RWMutex; skill
// artifacts (prompt.md files) are stored under baseDir/{tenantID}/{skillID}/.
type LiteSkillStore struct {
	baseDir string

	mu     sync.RWMutex
	skills map[string]harness.SkillMeta // keyed by skill ID
}

// NewLiteSkillStore creates a LiteSkillStore that persists skill artifacts
// under the given base directory.
func NewLiteSkillStore(baseDir string) *LiteSkillStore {
	return &LiteSkillStore{
		baseDir: baseDir,
		skills:  make(map[string]harness.SkillMeta),
	}
}

// ListPlatform returns an empty slice — Lite has no hardcoded platform skills.
func (s *LiteSkillStore) ListPlatform(_ context.Context) ([]harness.SkillMeta, error) {
	return []harness.SkillMeta{}, nil
}

// ListTenant returns all skills belonging to the given tenant.
func (s *LiteSkillStore) ListTenant(_ context.Context, tenantID string) ([]harness.SkillMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []harness.SkillMeta
	for _, meta := range s.skills {
		if meta.TenantID == tenantID {
			result = append(result, meta)
		}
	}
	if result == nil {
		result = []harness.SkillMeta{}
	}
	return result, nil
}

// Resolve loads skills by reference. For each ref it reads metadata from the
// in-memory map and loads the prompt content from disk. Returns
// ErrSkillNotFound (wrapped) for unknown IDs.
func (s *LiteSkillStore) Resolve(_ context.Context, refs []harness.SkillRef) ([]harness.Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []harness.Skill
	for _, ref := range refs {
		meta, ok := s.skills[ref.SkillID]
		if !ok {
			return nil, fmt.Errorf("%w: %s", harness.ErrSkillNotFound, ref.SkillID)
		}

		prompt, err := os.ReadFile(meta.ArtifactKey)
		if err != nil {
			return nil, fmt.Errorf("reading skill artifact: %w", err)
		}

		out = append(out, harness.Skill{
			Meta:   meta,
			Prompt: string(prompt),
		})
	}
	return out, nil
}

// Upload stores a new skill. It generates a UUID for the skill ID, writes the
// bundle content to {baseDir}/{tenantID}/{skillID}/prompt.md, and records
// metadata in the in-memory map.
func (s *LiteSkillStore) Upload(_ context.Context, tenantID string, meta harness.SkillMeta, bundle io.Reader) (harness.SkillMeta, error) {
	id := generateUUID()

	dir := filepath.Join(s.baseDir, tenantID, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return harness.SkillMeta{}, fmt.Errorf("creating skill directory: %w", err)
	}

	artifactPath := filepath.Join(dir, "prompt.md")
	data, err := io.ReadAll(bundle)
	if err != nil {
		return harness.SkillMeta{}, fmt.Errorf("reading bundle: %w", err)
	}
	if err := os.WriteFile(artifactPath, data, 0o644); err != nil {
		return harness.SkillMeta{}, fmt.Errorf("writing skill artifact: %w", err)
	}

	meta.ID = id
	meta.TenantID = tenantID
	meta.ArtifactKey = artifactPath
	meta.Version = 1
	meta.CreatedAt = time.Now()

	s.mu.Lock()
	s.skills[id] = meta
	s.mu.Unlock()

	return meta, nil
}

// Delete removes a skill from the in-memory map and deletes its artifact
// directory from the filesystem.
func (s *LiteSkillStore) Delete(_ context.Context, tenantID, skillID string) error {
	s.mu.Lock()
	meta, ok := s.skills[skillID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", harness.ErrSkillNotFound, skillID)
	}
	if meta.TenantID != tenantID {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", harness.ErrSkillNotFound, skillID)
	}
	delete(s.skills, skillID)
	s.mu.Unlock()

	dir := filepath.Join(s.baseDir, tenantID, skillID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing skill directory: %w", err)
	}
	return nil
}

// generateUUID returns a random UUID v4 string using crypto/rand.
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	// Set version (4) and variant (RFC 4122).
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
