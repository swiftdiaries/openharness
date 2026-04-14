package providers

import "testing"

func TestAnthropicProvider_NameAndDefaultModel(t *testing.T) {
	p := NewAnthropicProvider("anthropic", "test-key", "claude-opus-4-6")
	if got := p.Name(); got != "anthropic" {
		t.Errorf("Name() = %q, want %q", got, "anthropic")
	}
	if got := p.DefaultModel(); got != "claude-opus-4-6" {
		t.Errorf("DefaultModel() = %q, want %q", got, "claude-opus-4-6")
	}
}
