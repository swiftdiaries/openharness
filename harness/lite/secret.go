package lite

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time check that LiteSecretStore implements SecretStore.
var _ harness.SecretStore = (*LiteSecretStore)(nil)

// LiteSecretStore implements SecretStore using environment variable lookup
// with an in-memory override layer. Suitable for single-user desktop (Lite
// edition) where there is no Vault or external secret manager.
//
// Lookup order for GetCredentials:
//  1. In-memory store (populated via StoreCredentials)
//  2. Environment variable via os.Getenv
//
// In-memory credentials are lost on restart, which is acceptable for the
// single-user desktop use case.
type LiteSecretStore struct {
	mu    sync.RWMutex
	store map[string]map[string]string // name -> credentials map
}

// NewLiteSecretStore creates a new LiteSecretStore with an empty in-memory store.
func NewLiteSecretStore() *LiteSecretStore {
	return &LiteSecretStore{
		store: make(map[string]map[string]string),
	}
}

// GetCredentials looks up credentials by name. It first checks the in-memory
// store, then falls back to os.Getenv. The tenantID parameter is accepted for
// interface compatibility but ignored in Lite edition (single-tenant).
//
// Returns ErrNotFound if the name is not found in either location.
func (s *LiteSecretStore) GetCredentials(_ context.Context, _, name string) (map[string]string, error) {
	// Check in-memory store first.
	s.mu.RLock()
	creds, ok := s.store[name]
	s.mu.RUnlock()

	if ok {
		// Return a copy to prevent callers from mutating internal state.
		out := make(map[string]string, len(creds))
		for k, v := range creds {
			out[k] = v
		}
		return out, nil
	}

	// Fall back to environment variable.
	if val := os.Getenv(name); val != "" {
		return map[string]string{name: val}, nil
	}

	return nil, fmt.Errorf("%w: credential %q", harness.ErrNotFound, name)
}

// MintScopedToken returns a static passthrough token. In Lite edition there
// is no Vault, so scoped tokens are not meaningful.
func (s *LiteSecretStore) MintScopedToken(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "lite-passthrough", nil
}

// StoreCredentials writes credentials to the in-memory store. These are lost
// on process restart, which is acceptable for single-user desktop.
func (s *LiteSecretStore) StoreCredentials(_ context.Context, _, name string, creds map[string]string) error {
	// Store a copy to prevent callers from mutating internal state.
	copied := make(map[string]string, len(creds))
	for k, v := range creds {
		copied[k] = v
	}

	s.mu.Lock()
	s.store[name] = copied
	s.mu.Unlock()

	return nil
}
