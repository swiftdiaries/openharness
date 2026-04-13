// Package sessions provides local JSONL-based session persistence.
package sessions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        string    `json:"id"`
	AgentName string    `json:"agent"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Timestamp time.Time       `json:"ts"`
	Usage     *Usage          `json:"usage,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"in"`
	OutputTokens int `json:"out"`
}

type jsonlLine struct {
	Type string `json:"type"`

	// Meta fields
	ID        string    `json:"id,omitempty"`
	Agent     string    `json:"agent,omitempty"`
	CreatedAt time.Time `json:"created_at,omitzero"`

	// Message fields
	Role       string          `json:"role,omitempty"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Timestamp  time.Time       `json:"ts,omitzero"`
	MsgUsage   *Usage          `json:"usage,omitempty"`
}

type Store struct {
	dir string
}

// Compile-time assertion: *Store satisfies store.SessionsStore.
// (Inline to avoid circular import with the store package.)
var _ interface {
	CreateSession(agentName string) Session
	AppendMessage(sessionID string, msg Message) error
	LoadSession(sessionID string) (Session, []Message, error)
	ListSessions() ([]Session, error)
	DeleteSession(sessionID string) error
} = (*Store)(nil)

func NewStore(dir string) *Store {
	os.MkdirAll(dir, 0o755)
	return &Store{dir: dir}
}

func (s *Store) CreateSession(agentName string) Session {
	sess := Session{
		ID:        uuid.New().String(),
		AgentName: agentName,
		CreatedAt: time.Now().UTC(),
	}
	line := jsonlLine{
		Type:      "meta",
		ID:        sess.ID,
		Agent:     sess.AgentName,
		CreatedAt: sess.CreatedAt,
	}
	data, _ := json.Marshal(line)
	path := filepath.Join(s.dir, sess.ID+".jsonl")
	os.WriteFile(path, append(data, '\n'), 0o644)
	return sess
}

func (s *Store) AppendMessage(sessionID string, msg Message) error {
	msg.Type = "message"
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}
	line := jsonlLine{
		Type:       "message",
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCalls:  msg.ToolCalls,
		ToolCallID: msg.ToolCallID,
		Timestamp:  msg.Timestamp,
		MsgUsage:   msg.Usage,
	}
	data, _ := json.Marshal(line)
	data = append(data, '\n')

	path := filepath.Join(s.dir, sessionID+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (s *Store) LoadSession(sessionID string) (Session, []Message, error) {
	path := filepath.Join(s.dir, sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return Session{}, nil, fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	var sess Session
	var msgs []Message
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var line jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		switch line.Type {
		case "meta":
			sess = Session{ID: line.ID, AgentName: line.Agent, CreatedAt: line.CreatedAt}
		case "message":
			msgs = append(msgs, Message{
				Role:       line.Role,
				Content:    line.Content,
				ToolCalls:  line.ToolCalls,
				ToolCallID: line.ToolCallID,
				Timestamp:  line.Timestamp,
				Usage:      line.MsgUsage,
			})
		}
	}
	return sess, msgs, scanner.Err()
}

func (s *Store) ListSessions() ([]Session, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var sessions []Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".jsonl")
		sess, _, err := s.LoadSession(id)
		if err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *Store) DeleteSession(sessionID string) error {
	path := filepath.Join(s.dir, sessionID+".jsonl")
	return os.Remove(path)
}
