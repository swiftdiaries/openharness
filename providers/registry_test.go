package providers

import (
	"context"
	"testing"
)

type stubProvider struct {
	name         string
	defaultModel string
}

func (s *stubProvider) Chat(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "hello"}, nil
}
func (s *stubProvider) ChatStream(_ context.Context, _ ChatRequest, _ func(StreamChunk)) (*ChatResponse, error) {
	return &ChatResponse{Content: "hello"}, nil
}
func (s *stubProvider) DefaultModel() string { return s.defaultModel }
func (s *stubProvider) Name() string         { return s.name }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubProvider{name: "openrouter", defaultModel: "claude-sonnet"})

	p, err := r.Get("openrouter")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Name() != "openrouter" {
		t.Errorf("expected openrouter, got %s", p.Name())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing provider")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubProvider{name: "openrouter"})
	r.Register(&stubProvider{name: "anthropic"})

	names := r.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(names))
	}
}

func TestRegistry_ReplaceProvider(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubProvider{name: "openrouter", defaultModel: "old"})
	r.Register(&stubProvider{name: "openrouter", defaultModel: "new"})

	p, _ := r.Get("openrouter")
	if p.DefaultModel() != "new" {
		t.Errorf("expected replaced provider, got model %s", p.DefaultModel())
	}
}
