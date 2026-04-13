package agent

import (
	"context"
	"errors"
	"testing"
)

// TestNoOpsSatisfyInterfaces is a compile-time assertion: if any
// no-op type drifts away from its interface contract, this file
// fails to compile and the test target reports the mismatch.
func TestNoOpsSatisfyInterfaces(t *testing.T) {
	var _ TraceRecorder = NoOpTraceRecorder{}
	var _ KnowledgeStore = NoOpKnowledgeStore{}
	var _ MemoryStore = NoOpMemoryStore{}
	var _ PromptContextProvider = NoOpPromptContextProvider{}
}

// TestNoOpTraceRecorder_DoesNotPanic exercises every method on
// NoOpTraceRecorder to confirm none of them dereference nil or
// otherwise crash. The agent loop must be safe to run with this
// implementation.
func TestNoOpTraceRecorder_DoesNotPanic(t *testing.T) {
	ctx := context.Background()
	rec := NoOpTraceRecorder{}

	gotCtx := rec.StartRun(ctx, "run-1", "agent-1", "session-1")
	if gotCtx != ctx {
		t.Errorf("StartRun should return the same context unchanged")
	}

	rec.RecordLLMCall(ctx, LLMCallInfo{
		Model:        "claude-opus-4-6",
		InputTokens:  100,
		OutputTokens: 50,
		CostUSD:      0.0034,
		DurationMS:   1234,
	})

	rec.RecordToolCall(ctx, "filesystem", []byte(`{"path":"/tmp"}`), []byte(`{"ok":true}`), 42, nil)
	rec.RecordToolCall(ctx, "exec", nil, nil, 0, errors.New("denied"))

	rec.EndRun(ctx, "completed", nil)
	rec.EndRun(ctx, "error", errors.New("budget exceeded"))
}

// TestNoOpKnowledgeStore_ReturnsEmpty confirms Search and Get
// return empty results without errors. Tools and retrieval that
// fall back to this default should observe "no results" cleanly.
func TestNoOpKnowledgeStore_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := NoOpKnowledgeStore{}

	blocks, err := store.Search(ctx, "anything", 10)
	if err != nil {
		t.Errorf("Search returned unexpected error: %v", err)
	}
	if blocks != nil {
		t.Errorf("Search should return nil slice, got %v", blocks)
	}

	block, err := store.Get(ctx, "any-id")
	if err != nil {
		t.Errorf("Get returned unexpected error: %v", err)
	}
	if block.ID != "" || block.Title != "" || block.Content != "" || block.Path != "" || block.Metadata != nil {
		t.Errorf("Get should return zero-value Block, got %+v", block)
	}
}

// TestNoOpMemoryStore_ReturnsEmpty confirms the memory store no-op.
func TestNoOpMemoryStore_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := NoOpMemoryStore{}

	entries, err := store.Search(ctx, "query", 5)
	if err != nil {
		t.Errorf("Search returned unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("Search should return nil slice, got %v", entries)
	}
}

// TestNoOpPromptContextProvider_ReturnsEmpty confirms the prompt
// context provider no-op.
func TestNoOpPromptContextProvider_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	provider := NoOpPromptContextProvider{}

	files, err := provider.Files(ctx)
	if err != nil {
		t.Errorf("Files returned unexpected error: %v", err)
	}
	if files != nil {
		t.Errorf("Files should return nil slice, got %v", files)
	}
}
