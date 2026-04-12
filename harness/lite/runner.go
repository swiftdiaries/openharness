package lite

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/swiftdiaries/openharness/harness"
)

// Compile-time check that LiteRunner implements HarnessRunner.
var _ harness.HarnessRunner = (*LiteRunner)(nil)

// LoopFactory constructs and runs an agent loop. The function should block
// until the loop completes and send intermediate events to the events channel.
// The caller (LiteRunner) handles sending EventRunStarted and EventRunCompleted/
// EventError — the factory should only send events that occur during the loop.
type LoopFactory func(ctx context.Context, cfg harness.RunConfig, events chan<- harness.Event) error

// runState tracks an active or completed run inside LiteRunner.
type runState struct {
	mu        sync.Mutex // guards events channel sends after close
	status    string
	cancel    context.CancelFunc
	events    chan harness.Event
	closed    bool // true after channel is closed
	startedAt time.Time
	endedAt   *time.Time
	err       string
}

// LiteRunner implements HarnessRunner using embedded goroutines.
// This is the runner used by the desktop (Lite edition) app — it wraps
// agent loop execution into the harness abstraction without requiring
// any external infrastructure.
type LiteRunner struct {
	mu      sync.RWMutex
	runs    map[string]*runState
	factory LoopFactory
}

// NewLiteRunner creates a LiteRunner that uses the given LoopFactory to
// execute agent loops.
func NewLiteRunner(factory LoopFactory) *LiteRunner {
	return &LiteRunner{
		runs:    make(map[string]*runState),
		factory: factory,
	}
}

// Dispatch spawns a goroutine that runs the agent loop via the LoopFactory.
// It immediately returns a chanEventStream that the caller can read events from.
func (r *LiteRunner) Dispatch(ctx context.Context, cfg harness.RunConfig) (harness.EventStream, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runs[cfg.RunID]; exists {
		return nil, fmt.Errorf("%w: run %q already exists", harness.ErrConflict, cfg.RunID)
	}

	runCtx, cancel := context.WithCancel(ctx)
	ch := make(chan harness.Event, 64)
	now := time.Now()

	rs := &runState{
		status:    "running",
		cancel:    cancel,
		events:    ch,
		startedAt: now,
	}
	r.runs[cfg.RunID] = rs

	go r.executeRun(runCtx, cfg, rs, ch)

	return &chanEventStream{ch: ch}, nil
}

// executeRun is the goroutine body: sends EventRunStarted, calls the factory,
// then sends EventRunCompleted or EventError based on the outcome.
func (r *LiteRunner) executeRun(ctx context.Context, cfg harness.RunConfig, rs *runState, ch chan harness.Event) {
	defer func() {
		rs.mu.Lock()
		rs.closed = true
		close(ch)
		rs.mu.Unlock()
	}()

	// Send run.started
	ch <- harness.Event{
		Type:      harness.EventRunStarted,
		ID:        cfg.RunID,
		Timestamp: time.Now(),
	}

	// Call the user-provided loop factory.
	err := r.factory(ctx, cfg, ch)

	r.mu.Lock()
	now := time.Now()
	rs.endedAt = &now

	if err != nil {
		// Check if this was a cancellation.
		if ctx.Err() == context.Canceled && rs.status == "cancelled" {
			// Already marked cancelled by Cancel(); just send the error event.
			r.mu.Unlock()
			ch <- harness.Event{
				Type:      harness.EventError,
				Content:   "run cancelled",
				ID:        cfg.RunID,
				Timestamp: time.Now(),
			}
			return
		}
		rs.status = "error"
		rs.err = err.Error()
		r.mu.Unlock()
		ch <- harness.Event{
			Type:      harness.EventError,
			Content:   err.Error(),
			ID:        cfg.RunID,
			Timestamp: time.Now(),
		}
		return
	}

	rs.status = "completed"
	r.mu.Unlock()
	ch <- harness.Event{
		Type:      harness.EventRunCompleted,
		ID:        cfg.RunID,
		Timestamp: time.Now(),
	}
}

// Resume returns ErrNotFound for Lite — there are no durable sessions in
// embedded mode.
func (r *LiteRunner) Resume(_ context.Context, _ string) (harness.EventStream, error) {
	return nil, fmt.Errorf("%w: resume not supported in Lite runner", harness.ErrNotFound)
}

// Interrupt appends an event to the run's event channel if the run is active.
// It holds the runState lock to prevent sending on a closed channel.
func (r *LiteRunner) Interrupt(_ context.Context, runID string, event harness.Event) error {
	r.mu.RLock()
	rs, ok := r.runs[runID]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("%w: run %q", harness.ErrNotFound, runID)
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.closed {
		return fmt.Errorf("%w: run %q is no longer active", harness.ErrRunNotActive, runID)
	}

	if rs.status != "running" {
		return fmt.Errorf("%w: run %q status is %q", harness.ErrRunNotActive, runID, rs.status)
	}

	// Non-blocking send; if the channel is full we drop. The buffer (64) should
	// be large enough for typical interrupt traffic.
	select {
	case rs.events <- event:
		return nil
	default:
		return fmt.Errorf("event channel full for run %q", runID)
	}
}

// Cancel cancels the run's context and marks it as cancelled.
func (r *LiteRunner) Cancel(_ context.Context, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rs, ok := r.runs[runID]
	if !ok {
		return fmt.Errorf("%w: run %q", harness.ErrNotFound, runID)
	}

	if rs.status != "running" {
		return fmt.Errorf("%w: run %q status is %q", harness.ErrRunNotActive, runID, rs.status)
	}

	rs.status = "cancelled"
	rs.cancel()
	return nil
}

// Status returns the RunStatus for the given runID.
func (r *LiteRunner) Status(_ context.Context, runID string) (harness.RunStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rs, ok := r.runs[runID]
	if !ok {
		return harness.RunStatus{}, fmt.Errorf("%w: run %q", harness.ErrNotFound, runID)
	}

	return harness.RunStatus{
		RunID:     runID,
		Status:    rs.status,
		StartedAt: rs.startedAt,
		EndedAt:   rs.endedAt,
		Error:     rs.err,
	}, nil
}

// chanEventStream implements EventStream by wrapping a chan Event.
type chanEventStream struct {
	ch <-chan harness.Event
}

// Compile-time check that chanEventStream implements EventStream.
var _ harness.EventStream = (*chanEventStream)(nil)

// Next reads the next event from the channel. Returns io.EOF when the
// channel is closed.
func (s *chanEventStream) Next() (harness.Event, error) {
	ev, ok := <-s.ch
	if !ok {
		return harness.Event{}, io.EOF
	}
	return ev, nil
}

// Close is a no-op on the consumer side. The channel is closed by the
// producer (the goroutine in LiteRunner.executeRun).
func (s *chanEventStream) Close() error {
	return nil
}
