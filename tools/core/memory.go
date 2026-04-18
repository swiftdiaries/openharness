package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/swiftdiaries/openharness/agent"
	"github.com/swiftdiaries/openharness/tools"
)

// Memory provides key-value storage backed by a JSON file. It implements
// agent.MemoryStore so the agent loop's retrieval layer (Plan 5) can
// consume it via the narrow interface.
type Memory struct {
	path    string
	mu      sync.Mutex
	entries map[string]agent.MemoryEntry
}

// NewMemory creates a Memory tool whose on-disk state lives at storePath.
// Missing files are treated as empty; the directory is created lazily on
// the first save.
func NewMemory(storePath string) *Memory {
	m := &Memory{path: storePath, entries: make(map[string]agent.MemoryEntry)}
	m.load()
	return m
}

func (m *Memory) load() {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	decoded := map[string]agent.MemoryEntry{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return
	}
	m.entries = decoded
}

func (m *Memory) save() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0o644)
}

func (m *Memory) Definitions() []tools.ToolDefinition {
	return []tools.ToolDefinition{
		{
			Name:        "memory_store",
			Description: "Store a piece of information in agent memory.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"key":   {"type": "string", "description": "Unique key for this memory"},
					"value": {"type": "string", "description": "Value to store"}
				},
				"required": ["key", "value"]
			}`),
			Effects: tools.ToolEffectMutate,
		},
		{
			Name:        "memory_search",
			Description: "Search agent memory by case-insensitive substring match against key and value.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query (substring match)"},
					"limit": {"type": "integer", "description": "Max results (default 10)"}
				},
				"required": ["query"]
			}`),
			Effects: tools.ToolEffectRead,
		},
		{
			Name:        "memory_get",
			Description: "Retrieve a specific memory entry by key.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"key": {"type": "string", "description": "Key to retrieve"}
				},
				"required": ["key"]
			}`),
			Effects: tools.ToolEffectRead,
		},
	}
}

func (m *Memory) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	switch name {
	case "memory_store":
		return m.execStore(args)
	case "memory_search":
		return m.execSearch(ctx, args)
	case "memory_get":
		return m.execGet(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (m *Memory) execStore(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[params.Key] = agent.MemoryEntry{
		Key:       params.Key,
		Value:     params.Value,
		UpdatedAt: time.Now().UTC(),
	}
	if err := m.save(); err != nil {
		return nil, fmt.Errorf("save memory: %w", err)
	}
	return json.Marshal(map[string]string{"status": "stored", "key": params.Key})
}

func (m *Memory) execSearch(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}
	results, err := m.Search(ctx, params.Query, params.Limit)
	if err != nil {
		return nil, err
	}
	return json.Marshal(results)
}

// Search satisfies agent.MemoryStore. Semantics preserved verbatim from
// ghostfin's SearchEntries: case-insensitive substring match against Key
// and Value. Limit <= 0 defaults to 10. Results stop at limit. Map
// iteration is unordered, so callers must not rely on a stable order.
func (m *Memory) Search(ctx context.Context, query string, limit int) ([]agent.MemoryEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	q := strings.ToLower(query)
	var results []agent.MemoryEntry
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.Key), q) ||
			strings.Contains(strings.ToLower(e.Value), q) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *Memory) execGet(args json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[params.Key]
	if !ok {
		return json.Marshal(map[string]string{"error": "not found"})
	}
	return json.Marshal(entry)
}
