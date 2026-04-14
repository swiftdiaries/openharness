package providers

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// Registry manages available LLM providers.
type Registry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewRegistry creates a provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds or replaces a provider. If the old provider implements io.Closer, it is closed.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.providers[p.Name()]; ok {
		if c, ok := old.(io.Closer); ok {
			c.Close()
		}
	}
	r.providers[p.Name()] = p
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.providers[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("provider not found: %s", name)
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Close calls Close() on all providers that implement io.Closer.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, p := range r.providers {
		if c, ok := p.(io.Closer); ok {
			if err := c.Close(); err != nil {
				slog.Warn("provider close error", "name", name, "error", err)
			}
		}
	}
}
