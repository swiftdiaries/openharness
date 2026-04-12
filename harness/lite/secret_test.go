package lite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

func TestLiteSecretStoreGetFromEnv(t *testing.T) {
	t.Setenv("GHOSTFIN_TEST_SECRET", "my-api-key")

	store := NewLiteSecretStore()
	ctx := context.Background()

	creds, err := store.GetCredentials(ctx, "tenant-1", "GHOSTFIN_TEST_SECRET")
	if err != nil {
		t.Fatalf("GetCredentials returned error: %v", err)
	}

	val, ok := creds["GHOSTFIN_TEST_SECRET"]
	if !ok {
		t.Fatal("expected key GHOSTFIN_TEST_SECRET in credentials map")
	}
	if val != "my-api-key" {
		t.Errorf("expected %q, got %q", "my-api-key", val)
	}
}

func TestLiteSecretStoreMintScopedToken(t *testing.T) {
	store := NewLiteSecretStore()
	ctx := context.Background()

	token, err := store.MintScopedToken(ctx, "tenant-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("MintScopedToken returned error: %v", err)
	}
	if token != "lite-passthrough" {
		t.Errorf("expected %q, got %q", "lite-passthrough", token)
	}
}

func TestLiteSecretStoreStoreAndGet(t *testing.T) {
	store := NewLiteSecretStore()
	ctx := context.Background()

	creds := map[string]string{
		"api_key":    "sk-12345",
		"api_secret": "supersecret",
	}

	if err := store.StoreCredentials(ctx, "tenant-1", "addepar", creds); err != nil {
		t.Fatalf("StoreCredentials returned error: %v", err)
	}

	got, err := store.GetCredentials(ctx, "tenant-1", "addepar")
	if err != nil {
		t.Fatalf("GetCredentials returned error: %v", err)
	}

	if len(got) != len(creds) {
		t.Fatalf("expected %d entries, got %d", len(creds), len(got))
	}
	for k, want := range creds {
		if got[k] != want {
			t.Errorf("key %q: expected %q, got %q", k, want, got[k])
		}
	}

	// Verify returned map is a copy (mutation should not affect stored data).
	got["api_key"] = "mutated"
	got2, _ := store.GetCredentials(ctx, "tenant-1", "addepar")
	if got2["api_key"] != "sk-12345" {
		t.Error("returned credentials map should be a copy; mutation leaked into store")
	}
}

func TestLiteSecretStoreNotFound(t *testing.T) {
	store := NewLiteSecretStore()
	ctx := context.Background()

	_, err := store.GetCredentials(ctx, "tenant-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent credential, got nil")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestLiteSecretStoreInMemoryOverridesEnv(t *testing.T) {
	t.Setenv("OVERRIDE_KEY", "env-value")

	store := NewLiteSecretStore()
	ctx := context.Background()

	// Store in-memory value for the same name.
	if err := store.StoreCredentials(ctx, "tenant-1", "OVERRIDE_KEY", map[string]string{"OVERRIDE_KEY": "mem-value"}); err != nil {
		t.Fatalf("StoreCredentials returned error: %v", err)
	}

	got, err := store.GetCredentials(ctx, "tenant-1", "OVERRIDE_KEY")
	if err != nil {
		t.Fatalf("GetCredentials returned error: %v", err)
	}

	// In-memory should take precedence over env.
	if got["OVERRIDE_KEY"] != "mem-value" {
		t.Errorf("expected in-memory value %q, got %q", "mem-value", got["OVERRIDE_KEY"])
	}
}
