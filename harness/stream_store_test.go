package harness

import (
	"context"
	"io"
	"testing"
	"time"
)

// mockSessionStore is a minimal SessionStore for testing StoreBackedEventStream.
type mockSessionStore struct {
	events []Event
}

func (m *mockSessionStore) CreateSession(_ context.Context, _ string, _ SessionMeta) (string, error) {
	return "sess-1", nil
}
func (m *mockSessionStore) AppendEvent(_ context.Context, _ string, e Event) error {
	m.events = append(m.events, e)
	return nil
}
func (m *mockSessionStore) GetEvents(_ context.Context, _ string, fromPos int64) ([]Event, error) {
	if int(fromPos) >= len(m.events) {
		return nil, nil
	}
	return m.events[fromPos:], nil
}
func (m *mockSessionStore) GetSession(_ context.Context, _ string) (Session, error) {
	return Session{}, nil
}
func (m *mockSessionStore) SetCheckpoint(_ context.Context, _ string, _ int64) error {
	return nil
}
func (m *mockSessionStore) ListSessions(_ context.Context, _ string, _ ListOpts) ([]Session, error) {
	return nil, nil
}

func TestStoreBackedEventStreamReadsEvents(t *testing.T) {
	store := &mockSessionStore{
		events: []Event{
			{Type: EventRunStarted, Position: 0, Timestamp: time.Now()},
			{Type: EventRunCompleted, Position: 1, Timestamp: time.Now()},
		},
	}
	stream := NewStoreBackedEventStream(context.Background(), store, "sess-1")
	defer stream.Close()

	ev1, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev1.Type != EventRunStarted {
		t.Fatalf("Type = %q, want %q", ev1.Type, EventRunStarted)
	}

	ev2, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev2.Type != EventRunCompleted {
		t.Fatalf("Type = %q, want %q", ev2.Type, EventRunCompleted)
	}
}

func TestStoreBackedEventStreamClose(t *testing.T) {
	store := &mockSessionStore{}
	stream := NewStoreBackedEventStream(context.Background(), store, "sess-1")
	err := stream.Close()
	if err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	_, err = stream.Next()
	if err != io.EOF {
		t.Fatalf("Next() after Close = %v, want io.EOF", err)
	}
}
