package providers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ResolveAPIKey resolves an API key value.
//   - "$ENV_VAR" → reads from environment
//   - ""         → falls back to os.Getenv(fallbackEnv)
//   - anything else → returned as-is (literal key)
//
// NOTE: the ghostfin-era "keychain:<account>" scheme is not supported here;
// secret-store integration is tracked as a fast-follow (see the task filed at
// the end of Plan 2). In the interim, vertical apps should pass resolved
// literal keys or $ENV_VAR references at registration time.
func ResolveAPIKey(key, fallbackEnv string) string {
	if key == "" {
		return os.Getenv(fallbackEnv)
	}
	if strings.HasPrefix(key, "$") {
		return os.Getenv(key[1:])
	}
	return key
}

// VerifyAPIKey checks whether the resolved API key is accepted by the provider.
func VerifyAPIKey(apiKey, fallbackEnv string) (resolved bool, err error) {
	key := ResolveAPIKey(apiKey, fallbackEnv)
	if key == "" {
		return false, nil
	}
	req, _ := http.NewRequest(http.MethodGet, "https://openrouter.ai/api/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("connection failed: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return false, fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	return true, nil
}

// CallOpenRouter is the legacy function signature. It wraps OpenAICompatProvider.Chat
// for backward compatibility with code that hasn't migrated to the Provider interface.
func CallOpenRouter(ctx context.Context, model string, messages []Message, tools []ToolDefinition, apiKey string) (*ChatResponse, error) {
	apiKey = ResolveAPIKey(apiKey, "OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not set: provide via config or environment variable")
	}
	model = strings.TrimPrefix(model, "openrouter/")

	p := NewOpenAICompatProvider("openrouter", "https://openrouter.ai/api/v1", apiKey, model)
	return p.Chat(ctx, ChatRequest{
		Messages: messages,
		Tools:    tools,
		Model:    model,
	})
}
