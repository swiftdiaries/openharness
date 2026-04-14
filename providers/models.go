package providers

// ModelContextWindows maps model identifiers to their context window size in tokens.
var ModelContextWindows = map[string]int{
	"anthropic/claude-sonnet-4-6":          200000,
	"anthropic/claude-opus-4-6":            200000,
	"anthropic/claude-haiku-4-5":           200000,
	"anthropic/claude-sonnet-4-5-20250514": 200000,
	"openai/gpt-4o":                        128000,
	"openai/gpt-4o-mini":                   128000,
	"google/gemini-2.5-pro":                1000000,
	"google/gemini-2.5-flash":              1000000,
	"google/gemma-4-31b-it:free":           131072,
	"z-ai/glm-4.5-air:free":               131072,
	"anthropic/claude-haiku-4.5":           200000,
}

// ContextWindowSize returns the context window for a model, defaulting to 128000.
func ContextWindowSize(model string) int {
	if size, ok := ModelContextWindows[model]; ok {
		return size
	}
	return 128000
}
