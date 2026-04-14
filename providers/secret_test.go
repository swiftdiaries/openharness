package providers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/swiftdiaries/openharness/harness"
	"github.com/swiftdiaries/openharness/harness/lite"
)

func TestResolveAPIKeyWithStore_DelegatesNonSecretSchemes(t *testing.T) {
	ctx := context.Background()
	store := lite.NewLiteSecretStore()

	cases := []struct {
		name        string
		key         string
		fallbackEnv string
		envVars     map[string]string
		want        string
	}{
		{name: "literal", key: "my-key", want: "my-key"},
		{name: "env var expansion", key: "$FOO_KEY", envVars: map[string]string{"FOO_KEY": "foo-value"}, want: "foo-value"},
		{name: "empty falls back to env", key: "", fallbackEnv: "FALLBACK_KEY", envVars: map[string]string{"FALLBACK_KEY": "fallback-value"}, want: "fallback-value"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			got, err := ResolveAPIKeyWithStore(ctx, store, tc.key, tc.fallbackEnv)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveAPIKeyWithStore_NilStoreAllowedForNonSecretScheme(t *testing.T) {
	got, err := ResolveAPIKeyWithStore(context.Background(), nil, "plain-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "plain-key" {
		t.Errorf("got %q, want plain-key", got)
	}
}

func TestResolveAPIKeyWithStore_SecretScheme_UsesAPIKeyField(t *testing.T) {
	ctx := context.Background()
	store := lite.NewLiteSecretStore()
	if err := store.StoreCredentials(ctx, "", "openrouter", map[string]string{
		"api_key": "sk-from-store",
		"org_id":  "acme",
	}); err != nil {
		t.Fatalf("store creds: %v", err)
	}

	got, err := ResolveAPIKeyWithStore(ctx, store, "secret:openrouter", "UNUSED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-from-store" {
		t.Errorf("got %q, want sk-from-store", got)
	}
}

func TestResolveAPIKeyWithStore_SecretScheme_FallsBackToNameKey(t *testing.T) {
	// LiteSecretStore's env fallback returns {name: value} maps when it
	// resolves via os.Getenv. Exercise that code path.
	t.Setenv("OPENROUTER_SECRET", "sk-from-env")
	ctx := context.Background()
	store := lite.NewLiteSecretStore()

	got, err := ResolveAPIKeyWithStore(ctx, store, "secret:OPENROUTER_SECRET", "UNUSED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-from-env" {
		t.Errorf("got %q, want sk-from-env", got)
	}
}

func TestResolveAPIKeyWithStore_SecretScheme_NilStore(t *testing.T) {
	_, err := ResolveAPIKeyWithStore(context.Background(), nil, "secret:openrouter", "")
	if err == nil {
		t.Fatal("expected error for secret: scheme with nil store")
	}
	if !strings.Contains(err.Error(), "SecretStore") {
		t.Errorf("error should mention SecretStore, got: %v", err)
	}
}

func TestResolveAPIKeyWithStore_SecretScheme_EmptyName(t *testing.T) {
	store := lite.NewLiteSecretStore()
	_, err := ResolveAPIKeyWithStore(context.Background(), store, "secret:", "")
	if err == nil {
		t.Fatal("expected error for empty secret name")
	}
}

func TestResolveAPIKeyWithStore_SecretScheme_NotFound(t *testing.T) {
	store := lite.NewLiteSecretStore()
	_, err := ResolveAPIKeyWithStore(context.Background(), store, "secret:missing", "")
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("expected wrapped harness.ErrNotFound, got: %v", err)
	}
}
