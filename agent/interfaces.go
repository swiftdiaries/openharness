package agent

import "context"

// TraceRecorder is the consumer-side interface the agent loop uses to
// record LLM call spans, tool-call spans, and run lifecycle events.
// Layer 3's telemetry refactor (out of scope for Layer 2) will provide
// a concrete implementation backed by SQLite + DuckDB (Lite) or
// Postgres + OTLP (Enterprise). Until then, verticals can pass
// NoOpTraceRecorder{} or implement this interface against their own
// telemetry stack.
type TraceRecorder interface {
	StartRun(ctx context.Context, runID, agentID, sessionID string) context.Context
	RecordLLMCall(ctx context.Context, info LLMCallInfo)
	RecordToolCall(ctx context.Context, tool string, args, result []byte, durationMS int64, err error)
	EndRun(ctx context.Context, status string, runErr error)
}

// KnowledgeStore is the framework-level interface for a user-owned
// markdown/block knowledge repo. Used by the knowledge_graph built-in
// tool (Plan 3) and by agent/retrieval.go (Plan 5). Ghostfin's
// internal/notes package implements this interface in Plan 7. When
// notes itself is extracted in Layer 4, the implementation moves
// into openharness.
type KnowledgeStore interface {
	Search(ctx context.Context, query string, limit int) ([]Block, error)
	Get(ctx context.Context, id string) (Block, error)
}

// MemoryStore is the framework-level interface for the key-value
// memory tool. Used by agent/retrieval.go to inject relevant
// memories into the prompt before each LLM call. The built-in
// memory tool (Plan 3) writes through this interface.
type MemoryStore interface {
	Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error)
}

// PromptContextProvider supplies dynamic context files (AGENTS.md,
// persona files, onboarding instructions) to the system prompt
// builder. Verticals implement this against whatever document
// source they prefer — filesystem, database, in-memory.
type PromptContextProvider interface {
	Files(ctx context.Context) ([]ContextFile, error)
}
