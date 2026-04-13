package cost

import (
	"fmt"
	"strings"
)

// ModelPrice holds per-token costs in USD.
type ModelPrice struct {
	InputPerToken  float64
	OutputPerToken float64
}

var defaultPricing = map[string]ModelPrice{
	"anthropic/claude-sonnet-4-6": {
		InputPerToken:  3.0 / 1_000_000,
		OutputPerToken: 15.0 / 1_000_000,
	},
	"anthropic/claude-opus-4-6": {
		InputPerToken:  15.0 / 1_000_000,
		OutputPerToken: 75.0 / 1_000_000,
	},
	"anthropic/claude-haiku-4-5": {
		InputPerToken:  0.80 / 1_000_000,
		OutputPerToken: 4.0 / 1_000_000,
	},
	"anthropic/claude-3-5-sonnet": {
		InputPerToken:  3.0 / 1_000_000,
		OutputPerToken: 15.0 / 1_000_000,
	},
	"anthropic/claude-3-5-haiku": {
		InputPerToken:  1.0 / 1_000_000,
		OutputPerToken: 5.0 / 1_000_000,
	},
	"anthropic/claude-3-opus": {
		InputPerToken:  15.0 / 1_000_000,
		OutputPerToken: 75.0 / 1_000_000,
	},
	"anthropic/claude-3-haiku": {
		InputPerToken:  0.25 / 1_000_000,
		OutputPerToken: 1.25 / 1_000_000,
	},
	"google/gemini-2.5-pro": {
		InputPerToken:  1.25 / 1_000_000,
		OutputPerToken: 10.0 / 1_000_000,
	},
	"google/gemini-2.5-flash": {
		InputPerToken:  0.15 / 1_000_000,
		OutputPerToken: 0.60 / 1_000_000,
	},
	"google/gemma-4-31b-it:free": {
		InputPerToken:  0,
		OutputPerToken: 0,
	},
	"z-ai/glm-4.5-air:free": {
		InputPerToken:  0,
		OutputPerToken: 0,
	},
	"anthropic/claude-haiku-4.5": {
		InputPerToken:  0.80 / 1_000_000,
		OutputPerToken: 4.0 / 1_000_000,
	},
}

var pricingOverrides map[string]ModelPrice

func SetPricingOverrides(overrides map[string]ModelPrice) {
	pricingOverrides = overrides
}

func lookupPrice(model string) ModelPrice {
	model = strings.TrimPrefix(model, "openrouter/")

	// Check overrides first (exact)
	if pricingOverrides != nil {
		if p, ok := pricingOverrides[model]; ok {
			return p
		}
	}

	if p, ok := defaultPricing[model]; ok {
		return p
	}

	// Substring match in overrides
	if pricingOverrides != nil {
		for key, p := range pricingOverrides {
			if strings.Contains(key, model) || strings.Contains(model, key) {
				return p
			}
		}
	}
	for key, p := range defaultPricing {
		if strings.Contains(key, model) || strings.Contains(model, key) {
			return p
		}
	}

	return defaultPricing["anthropic/claude-sonnet-4-6"]
}

// Calculate returns the total USD cost for the given model and token counts.
func Calculate(model string, inputTokens, outputTokens int) float64 {
	p := lookupPrice(model)
	return p.InputPerToken*float64(inputTokens) + p.OutputPerToken*float64(outputTokens)
}

// FormatCost formats a USD cost for display.
func FormatCost(cost float64) string {
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	if cost < 1 {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}
