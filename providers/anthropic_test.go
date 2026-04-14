package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicProvider_NameAndDefaultModel(t *testing.T) {
	p := NewAnthropicProvider("anthropic", "test-key", "claude-opus-4-6")
	if got := p.Name(); got != "anthropic" {
		t.Errorf("Name() = %q, want %q", got, "anthropic")
	}
	if got := p.DefaultModel(); got != "claude-opus-4-6" {
		t.Errorf("DefaultModel() = %q, want %q", got, "claude-opus-4-6")
	}
}

func TestAnthropicProvider_Chat_Text(t *testing.T) {
	body := `{
		"id": "msg_test_123",
		"type": "message",
		"role": "assistant",
		"model": "claude-opus-4-6",
		"content": [{"type": "text", "text": "Hello from mock Claude"}],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 12, "output_tokens": 7}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" && r.Header.Get("X-Api-Key") == "" {
			t.Errorf("expected x-api-key header set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "claude-opus-4-6", srv.URL)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello from mock Claude" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from mock Claude")
	}
	if resp.ID != "msg_test_123" {
		t.Errorf("ID = %q, want %q", resp.ID, "msg_test_123")
	}
	if resp.Usage == nil || resp.Usage.PromptTokens != 12 || resp.Usage.CompletionTokens != 7 {
		t.Errorf("Usage = %+v, want prompt=12 completion=7", resp.Usage)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "end_turn")
	}
}

func TestAnthropicProvider_ChatStream_Text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		if err := writeAnthropicTextStream(w, []string{"Hello ", "from ", "mock ", "Claude"}, 10, 4); err != nil {
			t.Fatalf("write SSE: %v", err)
		}
	}))
	defer srv.Close()

	p := newAnthropicProviderWithBaseURL("anthropic", "test-key", "claude-opus-4-6", srv.URL)

	var received []string
	resp, err := p.ChatStream(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
	}, func(chunk StreamChunk) {
		if chunk.Content != "" {
			received = append(received, chunk.Content)
		}
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	joined := strings.Join(received, "")
	if joined != "Hello from mock Claude" {
		t.Errorf("chunks joined = %q, want %q", joined, "Hello from mock Claude")
	}
	if resp.Content != "Hello from mock Claude" {
		t.Errorf("final content = %q, want %q", resp.Content, "Hello from mock Claude")
	}
	if resp.Usage == nil || resp.Usage.PromptTokens != 10 || resp.Usage.CompletionTokens != 4 {
		t.Errorf("Usage = %+v, want prompt=10 completion=4", resp.Usage)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "end_turn")
	}
}
