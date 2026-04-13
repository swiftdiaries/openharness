package agent

import "context"

// NoOpTraceRecorder is a zero-value TraceRecorder that does nothing.
// Use it in tests or as the default when no telemetry backend is
// wired up. The agent loop should be safe to run with this
// implementation — no panics, no nil dereferences.
type NoOpTraceRecorder struct{}

func (NoOpTraceRecorder) StartRun(ctx context.Context, runID, agentID, sessionID string) context.Context {
	return ctx
}

func (NoOpTraceRecorder) RecordLLMCall(ctx context.Context, info LLMCallInfo) {}

func (NoOpTraceRecorder) RecordToolCall(ctx context.Context, tool string, args, result []byte, durationMS int64, err error) {
}

func (NoOpTraceRecorder) EndRun(ctx context.Context, status string, runErr error) {}

// NoOpKnowledgeStore is a zero-value KnowledgeStore that returns
// empty results for every query. Use it when a vertical does not
// have a knowledge backend (the knowledge_graph built-in tool
// becomes a no-op).
type NoOpKnowledgeStore struct{}

func (NoOpKnowledgeStore) Search(ctx context.Context, query string, limit int) ([]Block, error) {
	return nil, nil
}

func (NoOpKnowledgeStore) Get(ctx context.Context, id string) (Block, error) {
	return Block{}, nil
}

// NoOpMemoryStore is a zero-value MemoryStore that returns empty
// results for every query. Use it when memory injection is
// undesirable or when no memory backend is available.
type NoOpMemoryStore struct{}

func (NoOpMemoryStore) Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error) {
	return nil, nil
}

// NoOpPromptContextProvider is a zero-value PromptContextProvider
// that supplies no context files. The system prompt builder falls
// back to its built-in defaults.
type NoOpPromptContextProvider struct{}

func (NoOpPromptContextProvider) Files(ctx context.Context) ([]ContextFile, error) {
	return nil, nil
}
