package lite

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time interface check.
var _ harness.AgentStore = (*LiteAgentStore)(nil)

// LiteAgentStore is an in-memory AgentStore for the Lite edition.
// It stores only the current version of each agent (no version history).
type LiteAgentStore struct {
	mu      sync.RWMutex
	agents  map[string]harness.AgentDefinition
	counter atomic.Int64
}

// NewLiteAgentStore returns an empty LiteAgentStore.
func NewLiteAgentStore() *LiteAgentStore {
	return &LiteAgentStore{
		agents: make(map[string]harness.AgentDefinition),
	}
}

// NewLiteAgentStoreWithSeeds returns a LiteAgentStore pre-populated with the
// five default Lite agents.
func NewLiteAgentStoreWithSeeds() *LiteAgentStore {
	s := NewLiteAgentStore()
	now := time.Now()
	model := "claude-sonnet-4-20250514"
	tenantID := "lite"

	seeds := []harness.AgentDefinition{
		{
			Name:      "harness",
			AgentType: "open",
			Autonomy:  "auto",
			Model:     model,
		},
		{
			Name:      "fund-analyst",
			AgentType: "predefined",
			Autonomy:  "auto",
			Model:     model,
		},
		{
			Name:      "analyst",
			AgentType: "predefined",
			Autonomy:  "auto",
			Model:     model,
		},
		{
			Name:      "scribe",
			AgentType: "predefined",
			Autonomy:  "auto",
			Model:     model,
		},
		{
			Name:      "executive-assistant",
			AgentType: "predefined",
			Autonomy:  "auto",
			Model:     model,
		},
	}

	for _, seed := range seeds {
		id := fmt.Sprintf("agent-%d", s.counter.Add(1))
		seed.ID = id
		seed.TenantID = tenantID
		seed.Version = 1
		seed.CreatedAt = now
		seed.UpdatedAt = now
		s.agents[id] = seed
	}

	return s
}

// Create adds a new agent definition to the store.
func (s *LiteAgentStore) Create(_ context.Context, tenantID string, def harness.AgentDefinition) (harness.AgentDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	def.ID = fmt.Sprintf("agent-%d", s.counter.Add(1))
	def.TenantID = tenantID
	def.Version = 1
	def.CreatedAt = now
	def.UpdatedAt = now
	def.ArchivedAt = nil

	s.agents[def.ID] = def
	return def, nil
}

// Get retrieves an agent definition by ID.
func (s *LiteAgentStore) Get(_ context.Context, agentID string) (harness.AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	def, ok := s.agents[agentID]
	if !ok {
		return harness.AgentDefinition{}, harness.ErrNotFound
	}
	return def, nil
}

// Update applies a partial update to an agent definition with optimistic
// concurrency control. The caller must supply the expected current version;
// if it does not match, ErrConflict is returned. Nil pointer fields in the
// patch are skipped, preserving the existing value.
func (s *LiteAgentStore) Update(_ context.Context, agentID string, version int, patch harness.AgentPatch) (harness.AgentDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	def, ok := s.agents[agentID]
	if !ok {
		return harness.AgentDefinition{}, harness.ErrNotFound
	}
	if def.ArchivedAt != nil {
		return harness.AgentDefinition{}, harness.ErrArchived
	}
	if def.Version != version {
		return harness.AgentDefinition{}, harness.ErrConflict
	}

	// Apply non-nil patch fields.
	if patch.Name != nil {
		def.Name = *patch.Name
	}
	if patch.Description != nil {
		def.Description = *patch.Description
	}
	if patch.Model != nil {
		def.Model = *patch.Model
	}
	if patch.HarnessVersion != nil {
		def.HarnessVersion = *patch.HarnessVersion
	}
	if patch.SystemPrompt != nil {
		def.SystemPrompt = *patch.SystemPrompt
	}
	if patch.Persona != nil {
		def.Persona = *patch.Persona
	}
	if patch.Tools != nil {
		def.Tools = *patch.Tools
	}
	if patch.Skills != nil {
		def.Skills = *patch.Skills
	}
	if patch.Autonomy != nil {
		def.Autonomy = *patch.Autonomy
	}
	if patch.Metadata != nil {
		def.Metadata = *patch.Metadata
	}

	def.Version++
	def.UpdatedAt = time.Now()

	s.agents[agentID] = def
	return def, nil
}

// Archive sets the ArchivedAt timestamp on an agent, soft-deleting it.
func (s *LiteAgentStore) Archive(_ context.Context, agentID string) (harness.AgentDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	def, ok := s.agents[agentID]
	if !ok {
		return harness.AgentDefinition{}, harness.ErrNotFound
	}
	if def.ArchivedAt != nil {
		return harness.AgentDefinition{}, harness.ErrArchived
	}

	now := time.Now()
	def.ArchivedAt = &now
	def.UpdatedAt = now

	s.agents[agentID] = def
	return def, nil
}

// List returns non-archived agents for the given tenant, respecting pagination.
func (s *LiteAgentStore) List(_ context.Context, tenantID string, opts harness.ListOpts) ([]harness.AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []harness.AgentDefinition
	for _, def := range s.agents {
		if def.TenantID == tenantID && def.ArchivedAt == nil {
			filtered = append(filtered, def)
		}
	}

	// Apply offset.
	if opts.Offset > 0 {
		if opts.Offset >= len(filtered) {
			return nil, nil
		}
		filtered = filtered[opts.Offset:]
	}

	// Apply limit.
	if opts.Limit > 0 && opts.Limit < len(filtered) {
		filtered = filtered[:opts.Limit]
	}

	return filtered, nil
}

// ListVersions is not supported in the Lite edition (only the current version
// is stored in memory). It always returns ErrNotFound.
func (s *LiteAgentStore) ListVersions(_ context.Context, _ string) ([]harness.AgentDefinition, error) {
	return nil, harness.ErrNotFound
}

// Clone deep-copies an existing agent with a new ID and name.
func (s *LiteAgentStore) Clone(_ context.Context, agentID string, newName string) (harness.AgentDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	src, ok := s.agents[agentID]
	if !ok {
		return harness.AgentDefinition{}, harness.ErrNotFound
	}

	now := time.Now()
	clone := src
	clone.ID = fmt.Sprintf("agent-%d", s.counter.Add(1))
	clone.Name = newName
	clone.Version = 1
	clone.CreatedAt = now
	clone.UpdatedAt = now
	clone.ArchivedAt = nil

	// Deep-copy slices and maps to avoid aliasing.
	if src.Tools != nil {
		clone.Tools = make([]string, len(src.Tools))
		copy(clone.Tools, src.Tools)
	}
	if src.Skills != nil {
		clone.Skills = make([]harness.SkillRef, len(src.Skills))
		copy(clone.Skills, src.Skills)
	}
	if src.MCPServers != nil {
		clone.MCPServers = make([]harness.MCPServerConfig, len(src.MCPServers))
		copy(clone.MCPServers, src.MCPServers)
	}
	if src.CallableAgents != nil {
		clone.CallableAgents = make([]string, len(src.CallableAgents))
		copy(clone.CallableAgents, src.CallableAgents)
	}
	if src.Metadata != nil {
		clone.Metadata = make(map[string]string, len(src.Metadata))
		for k, v := range src.Metadata {
			clone.Metadata[k] = v
		}
	}

	s.agents[clone.ID] = clone
	return clone, nil
}
