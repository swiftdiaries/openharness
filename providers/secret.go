package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/swiftdiaries/openharness/harness"
)

// SecretScheme is the prefix that triggers SecretStore resolution in
// ResolveAPIKeyWithStore. A value of the form "secret:<name>" is resolved by
// calling SecretStore.GetCredentials(ctx, "", name) and returning the first
// of the following credential fields that is non-empty:
//
//  1. "api_key"
//  2. the secret name itself (matches LiteSecretStore's env fallback shape)
//
// Other schemes accepted by ResolveAPIKey ("$ENV_VAR", literal, empty) are
// delegated to ResolveAPIKey unchanged.
const SecretScheme = "secret:"

// ResolveAPIKeyWithStore is the SecretStore-aware counterpart to ResolveAPIKey.
// It accepts the same inputs as ResolveAPIKey plus a "secret:<name>" scheme
// that looks up the credential via the supplied harness.SecretStore.
//
// Passing a nil store is allowed when the key does not use the "secret:"
// scheme; if the key does use the scheme and store is nil, an error is
// returned.
func ResolveAPIKeyWithStore(ctx context.Context, store harness.SecretStore, key, fallbackEnv string) (string, error) {
	if !strings.HasPrefix(key, SecretScheme) {
		return ResolveAPIKey(key, fallbackEnv), nil
	}
	name := strings.TrimPrefix(key, SecretScheme)
	if name == "" {
		return "", fmt.Errorf("providers: %q scheme requires a secret name", SecretScheme)
	}
	if store == nil {
		return "", fmt.Errorf("providers: %q scheme requires a SecretStore but none was provided", SecretScheme)
	}
	creds, err := store.GetCredentials(ctx, "", name)
	if err != nil {
		return "", fmt.Errorf("providers: resolve secret %q: %w", name, err)
	}
	if v, ok := creds["api_key"]; ok && v != "" {
		return v, nil
	}
	if v, ok := creds[name]; ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("providers: secret %q has no %q or %q field", name, "api_key", name)
}
