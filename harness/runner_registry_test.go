package harness

import (
	"context"
	"errors"
	"testing"
)

// stubRunner is a minimal HarnessRunner for testing the registry.
type stubRunner struct{}

func (s *stubRunner) Dispatch(_ context.Context, _ RunConfig) (EventStream, error) {
	return nil, nil
}
func (s *stubRunner) Resume(_ context.Context, _ string) (EventStream, error) { return nil, nil }
func (s *stubRunner) Interrupt(_ context.Context, _ string, _ Event) error    { return nil }
func (s *stubRunner) Cancel(_ context.Context, _ string) error                { return nil }
func (s *stubRunner) Status(_ context.Context, _ string) (RunStatus, error) {
	return RunStatus{}, nil
}

func TestRunnerRegistryRegisterAndGet(t *testing.T) {
	reg := NewRunnerRegistry()
	runner := &stubRunner{}

	reg.Register("lite", runner)

	got, err := reg.Get("lite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != runner {
		t.Error("returned runner does not match registered runner")
	}
}

func TestRunnerRegistryGetUnregisteredReturnsError(t *testing.T) {
	reg := NewRunnerRegistry()

	_, err := reg.Get("k8s")
	if err == nil {
		t.Fatal("expected error for unregistered runner")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}
