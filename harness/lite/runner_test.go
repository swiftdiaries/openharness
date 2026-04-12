package lite

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time check.
var _ harness.HarnessRunner = (*LiteRunner)(nil)

func TestLiteRunnerDispatchAndComplete(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		events <- harness.Event{Type: harness.EventChunk, Content: "hello"}
		return nil
	}
	runner := NewLiteRunner(factory)

	stream, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}

	// run.started
	ev, err := stream.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if ev.Type != harness.EventRunStarted {
		t.Fatalf("Type = %q, want run.started", ev.Type)
	}

	// chunk from factory
	ev, err = stream.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if ev.Type != harness.EventChunk || ev.Content != "hello" {
		t.Fatalf("got %+v, want chunk 'hello'", ev)
	}

	// run.completed
	ev, err = stream.Next()
	if err != nil {
		t.Fatalf("Next error: %v", err)
	}
	if ev.Type != harness.EventRunCompleted {
		t.Fatalf("Type = %q, want run.completed", ev.Type)
	}

	// EOF
	_, err = stream.Next()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestLiteRunnerDuplicateRunID(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		<-ctx.Done()
		return ctx.Err()
	}
	runner := NewLiteRunner(factory)
	_, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("first Dispatch error: %v", err)
	}
	_, err = runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err == nil {
		t.Fatal("expected error for duplicate RunID")
	}
}

func TestLiteRunnerCancel(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		<-ctx.Done()
		return ctx.Err()
	}
	runner := NewLiteRunner(factory)
	_, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}

	err = runner.Cancel(context.Background(), "r1")
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	status, err := runner.Status(context.Background(), "r1")
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status.Status != "cancelled" {
		t.Fatalf("Status = %q, want cancelled", status.Status)
	}
}

func TestLiteRunnerStatusNotFound(t *testing.T) {
	runner := NewLiteRunner(nil)
	_, err := runner.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
}

func TestLiteRunnerResumeUnsupported(t *testing.T) {
	runner := NewLiteRunner(nil)
	_, err := runner.Resume(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error: resume not supported in Lite")
	}
}

func TestLiteRunnerFactoryError(t *testing.T) {
	factory := func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error {
		return io.ErrUnexpectedEOF
	}
	runner := NewLiteRunner(factory)
	stream, err := runner.Dispatch(context.Background(), harness.RunConfig{RunID: "r1"})
	if err != nil {
		t.Fatalf("Dispatch error: %v", err)
	}

	// run.started
	ev, _ := stream.Next()
	if ev.Type != harness.EventRunStarted {
		t.Fatalf("expected run.started, got %q", ev.Type)
	}

	// error event
	ev, _ = stream.Next()
	if ev.Type != harness.EventError {
		t.Fatalf("expected error, got %q", ev.Type)
	}

	// Verify status is error
	time.Sleep(10 * time.Millisecond)
	status, _ := runner.Status(context.Background(), "r1")
	if status.Status != "error" {
		t.Fatalf("Status = %q, want error", status.Status)
	}
}
