package sessions

import (
	"testing"
)

func TestCreateAndLoadSession(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	sess := store.CreateSession("analyst")
	if sess.ID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if sess.AgentName != "analyst" {
		t.Errorf("expected agent 'analyst', got %q", sess.AgentName)
	}

	// Append messages.
	store.AppendMessage(sess.ID, Message{Role: "user", Content: "hello"})
	store.AppendMessage(sess.ID, Message{Role: "assistant", Content: "hi there"})

	// Load and verify.
	loaded, msgs, err := store.LoadSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != sess.ID {
		t.Errorf("session ID mismatch")
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("msg[0]: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi there" {
		t.Errorf("msg[1]: %+v", msgs[1])
	}
}

func TestListSessions(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.CreateSession("analyst")
	store.CreateSession("scribe")

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestDeleteSession(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	sess := store.CreateSession("analyst")
	if err := store.DeleteSession(sess.ID); err != nil {
		t.Fatal(err)
	}

	sessions, _ := store.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after delete, got %d", len(sessions))
	}
}
