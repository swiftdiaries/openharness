package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// DefaultAnthropicModel is the fallback model used when a caller does not specify one.
const DefaultAnthropicModel = "claude-opus-4-6"

// AnthropicProvider implements Provider using the official Anthropic Go SDK.
// It supports real SSE streaming, tool-use reassembly, and prompt caching.
type AnthropicProvider struct {
	name         string
	apiKey       string
	defaultModel string
	client       anthropic.Client
}

// NewAnthropicProvider constructs a provider backed by the Anthropic SDK.
// If defaultModel is empty, DefaultAnthropicModel is used.
func NewAnthropicProvider(name, apiKey, defaultModel string) *AnthropicProvider {
	if defaultModel == "" {
		defaultModel = DefaultAnthropicModel
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{
		name:         name,
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       client,
	}
}

// newAnthropicProviderWithBaseURL is the test-only constructor that points the
// SDK at an httptest server.
func newAnthropicProviderWithBaseURL(name, apiKey, defaultModel, baseURL string) *AnthropicProvider {
	if defaultModel == "" {
		defaultModel = DefaultAnthropicModel
	}
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &AnthropicProvider{
		name:         name,
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       client,
	}
}

func (p *AnthropicProvider) Name() string         { return p.name }
func (p *AnthropicProvider) DefaultModel() string { return p.defaultModel }

// Chat sends a non-streaming Messages request and returns a ChatResponse.
// System prompts are drawn from any message with role "system" (concatenated
// with blank-line separators) and passed via MessageNewParams.System.
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	params, err := p.buildMessageParams(req)
	if err != nil {
		return nil, err
	}

	msg, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat: %w", err)
	}

	return p.messageToChatResponse(msg), nil
}

// ChatStream sends a streaming Messages request. Each text delta is forwarded
// via onChunk; the running message is reassembled server-side and returned as
// a ChatResponse. A final Done:true chunk is emitted once the stream completes.
func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	params, err := p.buildMessageParams(req)
	if err != nil {
		return nil, err
	}

	stream := p.client.Messages.NewStreaming(ctx, params)

	var accumulated anthropic.Message
	for stream.Next() {
		event := stream.Current()
		if err := accumulated.Accumulate(event); err != nil {
			return nil, fmt.Errorf("anthropic stream accumulate: %w", err)
		}

		// Forward text deltas to the caller as they arrive.
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			if onChunk != nil && event.Delta.Text != "" {
				onChunk(StreamChunk{Content: event.Delta.Text})
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("anthropic stream: %w", err)
	}

	if onChunk != nil {
		onChunk(StreamChunk{Done: true})
	}
	return p.messageToChatResponse(&accumulated), nil
}

// buildMessageParams translates a ChatRequest into anthropic.MessageNewParams.
// Messages with role == "system" are hoisted into params.System; all others
// become MessageParam entries (user or assistant).
func (p *AnthropicProvider) buildMessageParams(req ChatRequest) (anthropic.MessageNewParams, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	// Strip any routing prefix like "openrouter/anthropic/".
	_, model = ParseModelString(p.name, model)

	var systemText string
	userOrAssistant := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		text, err := textFromRawContent(m.Content)
		if err != nil {
			return anthropic.MessageNewParams{}, err
		}
		switch m.Role {
		case "system":
			if systemText != "" {
				systemText += "\n\n"
			}
			systemText += text
		case "user":
			userOrAssistant = append(userOrAssistant, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		case "assistant":
			userOrAssistant = append(userOrAssistant, anthropic.NewAssistantMessage(anthropic.NewTextBlock(text)))
		default:
			return anthropic.MessageNewParams{}, fmt.Errorf("anthropic: unsupported message role %q", m.Role)
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 4096,
		Messages:  userOrAssistant,
	}
	if systemText != "" {
		params.System = []anthropic.TextBlockParam{{Text: systemText}}
	}
	return params, nil
}

// textFromRawContent unmarshals a Message.Content json.RawMessage into a
// plain string. If the RawMessage is already a JSON string, we use it
// directly; otherwise we fall back to the raw bytes.
func textFromRawContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	return string(raw), nil
}

// messageToChatResponse translates an anthropic.Message into our ChatResponse shape.
func (p *AnthropicProvider) messageToChatResponse(msg *anthropic.Message) *ChatResponse {
	resp := &ChatResponse{
		ID:           msg.ID,
		FinishReason: string(msg.StopReason),
	}

	var textParts []string
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}
	if len(textParts) > 0 {
		resp.Content = strings.Join(textParts, "")
	}

	resp.Usage = &Usage{
		PromptTokens:        int(msg.Usage.InputTokens),
		CompletionTokens:    int(msg.Usage.OutputTokens),
		TotalTokens:         int(msg.Usage.InputTokens + msg.Usage.OutputTokens),
		CacheCreationTokens: int(msg.Usage.CacheCreationInputTokens),
		CacheReadTokens:     int(msg.Usage.CacheReadInputTokens),
	}
	return resp
}
