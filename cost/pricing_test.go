package cost

import "testing"

func TestCalculateKnownModel(t *testing.T) {
	got := Calculate("openrouter/anthropic/claude-sonnet-4-6", 1000, 500)
	want := 0.0105
	if diff := got - want; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("Calculate sonnet: got %f, want %f", got, want)
	}
}

func TestCalculateSubstringMatch(t *testing.T) {
	got := Calculate("openrouter/anthropic/claude-3-5-sonnet-20241022", 1_000_000, 0)
	want := 3.0
	if diff := got - want; diff > 0.01 || diff < -0.01 {
		t.Errorf("Calculate substring match: got %f, want %f", got, want)
	}
}

func TestCalculateUnknownFallsBackToSonnet(t *testing.T) {
	got := Calculate("some-unknown-model", 1_000_000, 0)
	want := 3.0
	if diff := got - want; diff > 0.01 || diff < -0.01 {
		t.Errorf("Calculate unknown: got %f, want %f", got, want)
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		cost float64
		want string
	}{
		{0.0042, "$0.0042"},
		{0.123, "$0.123"},
		{1.50, "$1.50"},
		{0, "$0.0000"},
	}
	for _, tt := range tests {
		got := FormatCost(tt.cost)
		if got != tt.want {
			t.Errorf("FormatCost(%f) = %q, want %q", tt.cost, got, tt.want)
		}
	}
}

func TestCalculateFreeModels(t *testing.T) {
	got := Calculate("google/gemma-4-31b-it:free", 1_000_000, 1_000_000)
	if got != 0.0 {
		t.Errorf("expected 0.0 for free model, got %f", got)
	}
	got = Calculate("z-ai/glm-4.5-air:free", 1_000_000, 1_000_000)
	if got != 0.0 {
		t.Errorf("expected 0.0 for free model, got %f", got)
	}
}

func TestCalculateHaiku45(t *testing.T) {
	got := Calculate("anthropic/claude-haiku-4.5", 1_000_000, 0)
	want := 0.80
	if diff := got - want; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("expected %f, got %f", want, got)
	}
}

func TestCalculateWithConfigOverride(t *testing.T) {
	SetPricingOverrides(map[string]ModelPrice{
		"custom/my-model": {InputPerToken: 1.0 / 1_000_000, OutputPerToken: 2.0 / 1_000_000},
	})
	defer SetPricingOverrides(nil)

	got := Calculate("custom/my-model", 1_000_000, 1_000_000)
	want := 3.0
	if diff := got - want; diff > 0.01 || diff < -0.01 {
		t.Errorf("Calculate with override: got %f, want %f", got, want)
	}
}
