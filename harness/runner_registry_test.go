package harness

import (
	"testing"
)

func TestRunnerRegistryGetUnregistered(t *testing.T) {
	reg := NewRunnerRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered runner")
	}
}

func TestRunnerRegistryRegisterAndGet(t *testing.T) {
	reg := NewRunnerRegistry()
	// Use a nil runner just to verify registration works
	// (LiteRunner not available yet in this package)
	reg.Register("test", nil)
	runner, err := reg.Get("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner != nil {
		t.Fatal("expected nil runner")
	}
}
