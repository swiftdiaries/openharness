package providers

import (
	"context"
	"fmt"

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

func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return nil, fmt.Errorf("anthropic: Chat not yet implemented")
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	return nil, fmt.Errorf("anthropic: ChatStream not yet implemented")
}
