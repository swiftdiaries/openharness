package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestChat_NoProviderSpan(t *testing.T) {
	// The Chat method must NOT create its own span via telemetry.Tracer().
	// It should rely on the parent span from loop.go via trace.SpanFromContext(ctx).
	// We verify this by checking that the source file does not import the telemetry package
	// and does not call telemetry.Tracer().
	src, err := os.ReadFile("openai_compat.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	content := string(src)
	// openharness has no telemetry package; this import should never appear.
	if strings.Contains(content, `/internal/telemetry"`) {
		t.Error("openai_compat.go still imports a telemetry package; Chat() should use trace.SpanFromContext(ctx) instead")
	}
	if strings.Contains(content, "telemetry.Tracer(") {
		t.Error("openai_compat.go still calls telemetry.Tracer(); provider should not create its own span")
	}
}

func TestOpenAICompatProvider_Chat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		resp := openAIResponse{
			ID: "resp-123",
			Choices: []openAIChoice{{
				Message:      openAIMessage{Role: "assistant", Content: json.RawMessage(`"Hello!"`)},
				FinishReason: "stop",
			}},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider("test-provider", srv.URL, "test-key", "default-model")

	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
		Model:    "test-model",
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("expected Hello!, got %q", resp.Content)
	}
	if resp.ID != "resp-123" {
		t.Errorf("expected resp-123, got %q", resp.ID)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected stop, got %q", resp.FinishReason)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
}

func TestOpenAICompatProvider_Name(t *testing.T) {
	p := NewOpenAICompatProvider("openrouter", "https://example.com", "key", "model")
	if p.Name() != "openrouter" {
		t.Errorf("expected openrouter, got %s", p.Name())
	}
}

func TestOpenAICompatProvider_DefaultModel(t *testing.T) {
	p := NewOpenAICompatProvider("openrouter", "https://example.com", "key", "claude-sonnet")
	if p.DefaultModel() != "claude-sonnet" {
		t.Errorf("expected claude-sonnet, got %s", p.DefaultModel())
	}
}

func TestOpenAICompatProvider_Chat_StripsProviderPrefix(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		json.NewDecoder(r.Body).Decode(&req)
		gotModel = req.Model
		resp := openAIResponse{
			Choices: []openAIChoice{{
				Message:      openAIMessage{Role: "assistant", Content: json.RawMessage(`"ok"`)},
				FinishReason: "stop",
			}},
			Usage: Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	tests := []struct {
		providerName string
		requestModel string
		wantModel    string
	}{
		{"openrouter", "openrouter/anthropic/claude-sonnet-4-6", "anthropic/claude-sonnet-4-6"},
		{"openrouter", "openrouter/anthropic/claude-haiku-4-5", "anthropic/claude-haiku-4-5"},
		{"openai", "openai/gpt-4o", "gpt-4o"},
		{"openrouter", "anthropic/claude-sonnet-4-6", "anthropic/claude-sonnet-4-6"}, // no prefix to strip
		{"custom", "custom/my-model", "my-model"},
	}
	for _, tt := range tests {
		t.Run(tt.providerName+"/"+tt.requestModel, func(t *testing.T) {
			p := NewOpenAICompatProvider(tt.providerName, srv.URL, "key", "")
			_, err := p.Chat(context.Background(), ChatRequest{
				Messages: []Message{NewTextMessage("user", "hi")},
				Model:    tt.requestModel,
			})
			if err != nil {
				t.Fatalf("Chat: %v", err)
			}
			if gotModel != tt.wantModel {
				t.Errorf("model sent to API = %q, want %q", gotModel, tt.wantModel)
			}
		})
	}
}

func TestParseModelString(t *testing.T) {
	tests := []struct {
		provider   string
		raw        string
		wantSystem string
		wantModel  string
	}{
		{"openrouter", "openrouter/anthropic/claude-sonnet-4-6", "anthropic", "anthropic/claude-sonnet-4-6"},
		{"openrouter", "openrouter/anthropic/claude-haiku-4-5", "anthropic", "anthropic/claude-haiku-4-5"},
		{"openrouter", "anthropic/claude-sonnet-4-6", "anthropic", "anthropic/claude-sonnet-4-6"},
		{"openrouter", "openrouter/google/gemini-2.5-pro", "google", "google/gemini-2.5-pro"},
		{"openai", "openai/gpt-4o", "openai", "gpt-4o"},
		{"openai", "gpt-4o", "openai", "gpt-4o"},
		{"anthropic", "claude-sonnet-4-6", "anthropic", "claude-sonnet-4-6"},
		{"", "anthropic/claude-sonnet-4-6", "anthropic", "anthropic/claude-sonnet-4-6"},
	}
	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.raw, func(t *testing.T) {
			system, model := ParseModelString(tt.provider, tt.raw)
			if system != tt.wantSystem {
				t.Errorf("system = %q, want %q", system, tt.wantSystem)
			}
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
		})
	}
}

func TestOpenAICompatProvider_Chat_CostUSD(t *testing.T) {
	costVal := 0.00014
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			ID: "resp-cost",
			Choices: []openAIChoice{{
				Message:      openAIMessage{Role: "assistant", Content: json.RawMessage(`"Hi"`)},
				FinishReason: "stop",
			}},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, CostUSD: &costVal},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider("test-provider", srv.URL, "test-key", "default-model")
	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
		Model:    "test-model",
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Usage.CostUSD == nil {
		t.Fatal("expected CostUSD to be non-nil")
	}
	if *resp.Usage.CostUSD != 0.00014 {
		t.Errorf("expected CostUSD 0.00014, got %f", *resp.Usage.CostUSD)
	}
}

func TestOpenAICompatProvider_Chat_Retry429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(429)
			w.Write([]byte("rate limited"))
			return
		}
		resp := openAIResponse{
			Choices: []openAIChoice{{
				Message:      openAIMessage{Role: "assistant", Content: json.RawMessage(`"OK"`)},
				FinishReason: "stop",
			}},
			Usage: Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider("test", srv.URL, "key", "model")
	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{NewTextMessage("user", "hi")},
	})
	if err != nil {
		t.Fatalf("Chat after retries: %v", err)
	}
	if resp.Content != "OK" {
		t.Errorf("expected OK, got %q", resp.Content)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
