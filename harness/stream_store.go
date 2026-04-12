package harness

import (
	"context"
	"io"
	"sync"
	"time"
)

const defaultPollInterval = 200 * time.Millisecond

// StoreBackedEventStream polls a SessionStore for new events.
// It is used by remote runners (K8s, Lambda) where events are written to the
// store by the remote process and read back by the gateway via polling.
type StoreBackedEventStream struct {
	store        SessionStore
	sessionID    string
	lastPos      int64
	pollInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc

	mu   sync.Mutex
	done bool
}

// NewStoreBackedEventStream creates a new polling event stream.
// The stream polls the given SessionStore for events on the specified session.
func NewStoreBackedEventStream(ctx context.Context, store SessionStore, sessionID string) *StoreBackedEventStream {
	ctx, cancel := context.WithCancel(ctx)
	return &StoreBackedEventStream{
		store:        store,
		sessionID:    sessionID,
		lastPos:      0,
		pollInterval: defaultPollInterval,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Next returns the next event from the stream. It blocks until an event is
// available or the stream is closed. Returns io.EOF when the stream is done
// (either closed or a terminal event was received).
func (s *StoreBackedEventStream) Next() (Event, error) {
	for {
		s.mu.Lock()
		if s.done {
			s.mu.Unlock()
			return Event{}, io.EOF
		}
		s.mu.Unlock()

		events, err := s.store.GetEvents(s.ctx, s.sessionID, s.lastPos)
		if err != nil {
			return Event{}, err
		}

		if len(events) > 0 {
			ev := events[0]
			s.lastPos = ev.Position + 1

			// Terminal events signal the end of the stream.
			if ev.Type == EventRunCompleted || ev.Type == EventError {
				s.mu.Lock()
				s.done = true
				s.mu.Unlock()
			}

			return ev, nil
		}

		// No events available; wait before polling again.
		select {
		case <-s.ctx.Done():
			return Event{}, s.ctx.Err()
		case <-time.After(s.pollInterval):
			// continue polling
		}
	}
}

// Close stops the stream and releases resources.
func (s *StoreBackedEventStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
	s.cancel()
	return nil
}
