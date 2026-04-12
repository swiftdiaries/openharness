package harness

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// mockSessionStore is a minimal SessionStore for testing StoreBackedEventStream.
type mockSessionStore struct {
	mu     sync.Mutex
	events []Event
}

func (m *mockSessionStore) CreateSession(_ context.Context, _ string, _ SessionMeta) (string, error) {
	return "mock-session", nil
}

func (m *mockSessionStore) AppendEvent(_ context.Context, _ string, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockSessionStore) GetEvents(_ context.Context, _ string, fromPos int64) ([]Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []Event
	for _, ev := range m.events {
		if ev.Position >= fromPos {
			result = append(result, ev)
		}
	}
	return result, nil
}

func (m *mockSessionStore) GetSession(_ context.Context, _ string) (Session, error) {
	return Session{ID: "mock-session"}, nil
}

func (m *mockSessionStore) SetCheckpoint(_ context.Context, _ string, _ int64) error {
	return nil
}

func (m *mockSessionStore) ListSessions(_ context.Context, _ string, _ ListOpts) ([]Session, error) {
	return nil, nil
}

func TestStoreBackedEventStreamNext(t *testing.T) {
	store := &mockSessionStore{
		events: []Event{
			{Type: EventRunStarted, Position: 0, Timestamp: time.Now()},
			{Type: EventChunk, Content: "hello", Position: 1, Timestamp: time.Now()},
			{Type: EventRunCompleted, Position: 2, Timestamp: time.Now()},
		},
	}

	stream := NewStoreBackedEventStream(context.Background(), store, "sess-1")
	defer stream.Close()

	// First event: run.started
	ev1, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev1.Type != EventRunStarted {
		t.Errorf("event type = %q, want %q", ev1.Type, EventRunStarted)
	}

	// Second event: chunk
	ev2, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev2.Type != EventChunk {
		t.Errorf("event type = %q, want %q", ev2.Type, EventChunk)
	}
	if ev2.Content != "hello" {
		t.Errorf("content = %q, want %q", ev2.Content, "hello")
	}

	// Third event: run.completed (terminal)
	ev3, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev3.Type != EventRunCompleted {
		t.Errorf("event type = %q, want %q", ev3.Type, EventRunCompleted)
	}
}

func TestStoreBackedEventStreamClose(t *testing.T) {
	store := &mockSessionStore{}
	stream := NewStoreBackedEventStream(context.Background(), store, "sess-1")

	err := stream.Close()
	if err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if !stream.done {
		t.Error("expected done=true after Close()")
	}

	// Next after close should return EOF.
	_, err = stream.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("Next() after Close() = %v, want io.EOF", err)
	}
}

func TestStoreBackedEventStreamTerminalEvent(t *testing.T) {
	store := &mockSessionStore{
		events: []Event{
			{Type: EventRunCompleted, Position: 0, Timestamp: time.Now()},
		},
	}

	stream := NewStoreBackedEventStream(context.Background(), store, "sess-1")
	defer stream.Close()

	// Read the terminal event.
	ev, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Type != EventRunCompleted {
		t.Errorf("event type = %q, want %q", ev.Type, EventRunCompleted)
	}

	// Stream should now report done.
	if !stream.done {
		t.Error("expected done=true after terminal event")
	}

	// Subsequent Next should return EOF.
	_, err = stream.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("Next() after terminal = %v, want io.EOF", err)
	}
}
