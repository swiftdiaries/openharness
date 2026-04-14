package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OpenAICompatProvider implements Provider for any OpenAI-compatible API
// (OpenRouter, OpenAI, Azure OpenAI, Databricks, etc.).
type OpenAICompatProvider struct {
	name         string
	apiBase      string
	apiKey       string
	defaultModel string
	client       *http.Client
}

// NewOpenAICompatProvider creates a provider for an OpenAI-compatible API endpoint.
func NewOpenAICompatProvider(name, apiBase, apiKey, defaultModel string) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		name:         name,
		apiBase:      strings.TrimRight(apiBase, "/"),
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAICompatProvider) Name() string        { return p.name }
func (p *OpenAICompatProvider) DefaultModel() string { return p.defaultModel }


// Chat sends a non-streaming chat completion request.
func (p *OpenAICompatProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	raw := req.Model
	if raw == "" {
		raw = p.defaultModel
	}
	_, model := ParseModelString(p.name, raw)

	span := trace.SpanFromContext(ctx)

	// Build OpenAI-format request body
	oaiReq := openAIRequest{Model: model, Messages: req.Messages, Tools: req.Tools}
	bodyBytes, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := p.apiBase + "/chat/completions"
	const maxRetries = 3
	var lastErr error

	for attempt := range maxRetries + 1 {
		if attempt > 0 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			slog.Info("retrying LLM request", "provider", p.name, "attempt", attempt+1, "backoff", backoff, "model", model)
			span.AddEvent("retry", trace.WithAttributes(
				attribute.Int("retry.attempt", attempt+1),
			))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		httpResp, err := p.client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("do request: %w", err)
			continue
		}

		respBody, readErr := io.ReadAll(io.LimitReader(httpResp.Body, 10<<20))
		httpResp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read response: %w", readErr)
			continue
		}

		if httpResp.StatusCode == 429 || httpResp.StatusCode == 502 || httpResp.StatusCode == 503 {
			lastErr = fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(respBody))
			span.AddEvent("retry", trace.WithAttributes(
				attribute.Int("http.status_code", httpResp.StatusCode),
				attribute.Int("retry.attempt", attempt+1),
			))
			continue
		}

		if httpResp.StatusCode >= 400 {
			span.SetAttributes(attribute.String("exception.slug", "err-provider-http-error"))
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(respBody))
		}

		var oaiResp openAIResponse
		if err := json.Unmarshal(respBody, &oaiResp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}

		result := oaiResp.toChatResponse()
		return result, nil
	}

	span.RecordError(lastErr)
	span.SetStatus(codes.Error, lastErr.Error())
	span.SetAttributes(attribute.String("exception.slug", "err-provider-max-retries"))
	return nil, lastErr
}

// ChatStream is not yet implemented. Falls back to non-streaming Chat.
func (p *OpenAICompatProvider) ChatStream(ctx context.Context, req ChatRequest, _ func(StreamChunk)) (*ChatResponse, error) {
	return p.Chat(ctx, req)
}

// --- OpenAI wire format types (internal, not exported) ---

type openAIRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Choices []openAIChoice `json:"choices"`
	Usage   Usage          `json:"usage"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	ToolCalls []ToolCall      `json:"tool_calls,omitempty"`
}

func (r *openAIResponse) toChatResponse() *ChatResponse {
	resp := &ChatResponse{
		ID:    r.ID,
		Usage: &r.Usage,
	}
	if len(r.Choices) > 0 {
		choice := r.Choices[0]
		resp.FinishReason = choice.FinishReason

		// Extract content as string
		var text string
		if err := json.Unmarshal(choice.Message.Content, &text); err == nil {
			resp.Content = text
		} else {
			resp.Content = string(choice.Message.Content)
		}

		// Convert tool calls
		resp.ToolCalls = choice.Message.ToolCalls
	}
	return resp
}
