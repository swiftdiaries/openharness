// Package agent defines the openharness agent execution primitives:
// shared types, consumer-side interfaces (TraceRecorder, KnowledgeStore,
// MemoryStore, PromptContextProvider), and zero-value no-op
// implementations. The agent loop itself lands in Plan 5; Plan 1 only
// ships the seams that downstream packages compile against.
package agent

import "time"

// LLMCallInfo carries token + cost data for a single LLM call. It is
// intentionally defined with primitive fields rather than reusing
// providers.Usage so that the agent package does not depend on the
// providers package — the two packages move in parallel plans and
// must remain decoupled at the type level. The agent loop converts
// providers.Usage → LLMCallInfo at the call site.
type LLMCallInfo struct {
	Model               string
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	CostUSD             float64
	DurationMS          int64
}

// Block is a single content unit in a knowledge store. Verticals
// supply concrete blocks via their KnowledgeStore implementation.
// The Path field is optional and refers to a file or document
// location when the underlying store is filesystem-backed.
type Block struct {
	ID       string
	Title    string
	Content  string
	Path     string
	Metadata map[string]string
}

// MemoryEntry is a single entry from the memory store. Used by
// agent/retrieval.go (Plan 5) to inject relevant memories into
// the prompt before each LLM call.
type MemoryEntry struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

// ContextFile carries a named context document (persona, AGENTS.md,
// onboarding instructions) for inclusion in the system prompt. This
// type was previously defined in ghostfin's internal/bootstrap
// package; it is inlined here so the agent package has zero
// dependency on bootstrap. ghostfin/internal/bootstrap will
// re-export this type as an alias in Plan 7.
type ContextFile struct {
	Name    string
	Role    string
	Content string
}
