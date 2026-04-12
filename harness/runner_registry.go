package harness

import (
	"fmt"
	"sync"
)

// RunnerRegistry maps runner names (e.g. "lite", "k8s", "lambda") to
// HarnessRunner implementations. It is used by the gateway to dispatch
// agent runs to the correct backend. Thread-safe for concurrent access.
type RunnerRegistry struct {
	mu      sync.RWMutex
	runners map[string]HarnessRunner
}

// NewRunnerRegistry creates an empty runner registry.
func NewRunnerRegistry() *RunnerRegistry {
	return &RunnerRegistry{
		runners: make(map[string]HarnessRunner),
	}
}

// Register adds a named runner to the registry.
func (r *RunnerRegistry) Register(name string, runner HarnessRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runners[name] = runner
}

// Get returns the runner registered under the given name, or an error
// wrapping ErrNotFound if no runner is registered with that name.
func (r *RunnerRegistry) Get(name string) (HarnessRunner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	runner, ok := r.runners[name]
	if !ok {
		return nil, fmt.Errorf("%w: runner %q", ErrNotFound, name)
	}
	return runner, nil
}
