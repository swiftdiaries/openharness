package lite

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// mockLoopFactory returns a LoopFactory that sends n chunk events, then
// returns nil (success).
func mockLoopFactory(n int) LoopFactory {
	return func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case events <- harness.Event{
				Type:      harness.EventChunk,
				Content:   "chunk",
				Timestamp: time.Now(),
			}:
			}
		}
		return nil
	}
}

// slowLoopFactory returns a LoopFactory that blocks until the context is
// cancelled.
func slowLoopFactory() LoopFactory {
	return func(ctx context.Context, _ harness.RunConfig, _ chan<- harness.Event) error {
		<-ctx.Done()
		return ctx.Err()
	}
}

func TestLiteRunnerDispatch(t *testing.T) {
	runner := NewLiteRunner(mockLoopFactory(3))

	cfg := harness.RunConfig{RunID: "run-1"}
	stream, err := runner.Dispatch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	var events []harness.Event
	for {
		ev, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() error: %v", err)
		}
		events = append(events, ev)
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (started + completed), got %d", len(events))
	}

	// First event must be run.started.
	if events[0].Type != harness.EventRunStarted {
		t.Errorf("first event type = %q, want %q", events[0].Type, harness.EventRunStarted)
	}

	// Last event must be run.completed.
	last := events[len(events)-1]
	if last.Type != harness.EventRunCompleted {
		t.Errorf("last event type = %q, want %q", last.Type, harness.EventRunCompleted)
	}

	// Should have 3 chunk events in between.
	chunkCount := 0
	for _, ev := range events {
		if ev.Type == harness.EventChunk {
			chunkCount++
		}
	}
	if chunkCount != 3 {
		t.Errorf("chunk events = %d, want 3", chunkCount)
	}

	// Total: run.started + 3 chunks + run.completed = 5
	if len(events) != 5 {
		t.Errorf("total events = %d, want 5", len(events))
	}
}

func TestLiteRunnerCancel(t *testing.T) {
	runner := NewLiteRunner(slowLoopFactory())

	cfg := harness.RunConfig{RunID: "run-cancel"}
	stream, err := runner.Dispatch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	// Read the run.started event first.
	ev, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Type != harness.EventRunStarted {
		t.Errorf("first event = %q, want %q", ev.Type, harness.EventRunStarted)
	}

	// Cancel the run.
	if err := runner.Cancel(context.Background(), "run-cancel"); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	// Drain remaining events — should get an error event then EOF.
	for {
		ev, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() error after cancel: %v", err)
		}
		// Accept error event from cancellation.
		_ = ev
	}

	// Verify status is cancelled.
	status, err := runner.Status(context.Background(), "run-cancel")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.Status != "cancelled" {
		t.Errorf("status = %q, want %q", status.Status, "cancelled")
	}
}

func TestLiteRunnerStatusNotFound(t *testing.T) {
	runner := NewLiteRunner(mockLoopFactory(0))

	_, err := runner.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestLiteRunnerResumeNotSupported(t *testing.T) {
	runner := NewLiteRunner(mockLoopFactory(0))

	_, err := runner.Resume(context.Background(), "some-session")
	if err == nil {
		t.Fatal("expected error from Resume")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestLiteRunnerInterrupt(t *testing.T) {
	// Use a factory that blocks until it receives an interrupt event on the
	// events channel, then returns.
	factory := func(ctx context.Context, _ harness.RunConfig, events chan<- harness.Event) error {
		// We need to read from the same channel the runner uses for interrupts.
		// But the factory only gets the write side. Instead, block on context.
		// The test will send an interrupt and then cancel.
		<-ctx.Done()
		return ctx.Err()
	}

	runner := NewLiteRunner(factory)

	cfg := harness.RunConfig{RunID: "run-interrupt"}
	stream, err := runner.Dispatch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	// Read the run.started event.
	ev, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if ev.Type != harness.EventRunStarted {
		t.Errorf("first event = %q, want %q", ev.Type, harness.EventRunStarted)
	}

	// Send an interrupt event.
	interruptEvent := harness.Event{
		Type:      harness.EventAskUser,
		Content:   "user response",
		Timestamp: time.Now(),
	}
	err = runner.Interrupt(context.Background(), "run-interrupt", interruptEvent)
	if err != nil {
		t.Fatalf("Interrupt failed: %v", err)
	}

	// The interrupt event should appear on the stream.
	ev, err = stream.Next()
	if err != nil {
		t.Fatalf("Next() after interrupt error: %v", err)
	}
	if ev.Type != harness.EventAskUser {
		t.Errorf("interrupt event type = %q, want %q", ev.Type, harness.EventAskUser)
	}
	if ev.Content != "user response" {
		t.Errorf("interrupt content = %q, want %q", ev.Content, "user response")
	}

	// Clean up: cancel the run so the goroutine exits.
	_ = runner.Cancel(context.Background(), "run-interrupt")

	// Drain remaining events.
	for {
		_, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("drain error: %v", err)
		}
	}
}

func TestLiteRunnerDispatchDuplicate(t *testing.T) {
	runner := NewLiteRunner(slowLoopFactory())

	cfg := harness.RunConfig{RunID: "run-dup"}
	_, err := runner.Dispatch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("first Dispatch failed: %v", err)
	}

	// Second dispatch with same RunID should fail.
	_, err = runner.Dispatch(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for duplicate RunID")
	}
	if !errors.Is(err, harness.ErrConflict) {
		t.Errorf("error = %v, want wrapping ErrConflict", err)
	}

	// Clean up.
	_ = runner.Cancel(context.Background(), "run-dup")
}

func TestLiteRunnerCancelNotFound(t *testing.T) {
	runner := NewLiteRunner(mockLoopFactory(0))

	err := runner.Cancel(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestLiteRunnerInterruptNotFound(t *testing.T) {
	runner := NewLiteRunner(mockLoopFactory(0))

	err := runner.Interrupt(context.Background(), "nonexistent", harness.Event{})
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
	if !errors.Is(err, harness.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestLiteRunnerStatusAfterCompletion(t *testing.T) {
	runner := NewLiteRunner(mockLoopFactory(1))

	cfg := harness.RunConfig{RunID: "run-done"}
	stream, err := runner.Dispatch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	// Drain all events.
	for {
		_, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("drain error: %v", err)
		}
	}

	status, err := runner.Status(context.Background(), "run-done")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.Status != "completed" {
		t.Errorf("status = %q, want %q", status.Status, "completed")
	}
	if status.EndedAt == nil {
		t.Error("EndedAt should be set after completion")
	}
}
