package providers

import (
	"context"
	"encoding/json"
)

// Provider is the interface all LLM providers must implement.
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error)
	DefaultModel() string
	Name() string
}

// ChatRequest contains the input for a Chat/ChatStream call.
type ChatRequest struct {
	Messages []Message        `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
	Model    string           `json:"model,omitempty"`
	Options  map[string]any   `json:"options,omitempty"`
}

// ChatResponse is the result from an LLM call.
type ChatResponse struct {
	ID           string     `json:"id"`
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
	Usage        *Usage     `json:"usage,omitempty"`
}

// StreamChunk is a piece of a streaming response.
type StreamChunk struct {
	Content string `json:"content,omitempty"`
	Done    bool   `json:"done,omitempty"`
}

// ToolDefinition describes a tool available to the LLM.
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function ToolFunctionSchema `json:"function"`
}

// ToolFunctionSchema is the schema for a function tool.
type ToolFunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Message represents a conversation message.
type Message struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// NewTextMessage creates a message with a plain text content string.
func NewTextMessage(role, text string) Message {
	raw, _ := json.Marshal(text)
	return Message{Role: role, Content: raw}
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall is the function name and arguments in a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Usage tracks token consumption including cache and thinking tokens.
type Usage struct {
	PromptTokens        int      `json:"prompt_tokens"`
	CompletionTokens    int      `json:"completion_tokens"`
	TotalTokens         int      `json:"total_tokens"`
	CacheCreationTokens int      `json:"cache_creation_input_tokens,omitempty"`
	CacheReadTokens     int      `json:"cache_read_input_tokens,omitempty"`
	CostUSD             *float64 `json:"cost,omitempty"`
}

// ParseModelString normalises a raw model string (e.g. "openrouter/anthropic/claude-sonnet-4-6")
// by stripping a routing-provider prefix and extracting the upstream AI system.
// Returns (system, model) where system is the vendor (e.g. "anthropic") and model
// is the identifier to send to the API (e.g. "anthropic/claude-sonnet-4-6").
func ParseModelString(providerName, raw string) (system, model string) {
	model = raw
	if providerName != "" {
		// Strip routing prefix: "openrouter/anthropic/..." → "anthropic/..."
		trimmed := model
		prefix := providerName + "/"
		if len(model) > len(prefix) && model[:len(prefix)] == prefix {
			trimmed = model[len(prefix):]
		}
		model = trimmed
	}
	// Extract vendor from first path segment
	for i := 0; i < len(model); i++ {
		if model[i] == '/' {
			system = model[:i]
			return system, model
		}
	}
	// No slash — can't determine vendor, use provider name
	system = providerName
	return system, model
}

// ToolDef is an alias for backward compatibility.
type ToolDef = ToolDefinition

// ToolDefFunction is an alias for backward compatibility.
type ToolDefFunction = ToolFunctionSchema
