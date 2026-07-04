package eventstore

import (
	"context"
	"fmt"
	"sync"
)

// InMemory is a goroutine-safe, non-durable Store for tests and local runs.
type InMemory[E Event] struct {
	mu          sync.RWMutex
	streams     map[string][]E
	subscribers []func(events []E)
}

func NewInMemory[E Event]() *InMemory[E] {
	return &InMemory[E]{streams: make(map[string][]E)}
}

// Subscribe registers a listener invoked with the events of each successful
// Append, in order. It lets projections follow the store without the write side
// knowing they exist.
func (s *InMemory[E]) Subscribe(listener func(events []E)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers = append(s.subscribers, listener)
}

func (s *InMemory[E]) Append(ctx context.Context, streamID string, expectedVersion int, events ...E) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	current := s.streams[streamID]
	if len(current) != expectedVersion {
		s.mu.Unlock()
		return fmt.Errorf("%w: stream %q is at version %d, expected %d",
			ErrConcurrencyConflict, streamID, len(current), expectedVersion)
	}
	if len(events) == 0 {
		s.mu.Unlock()
		return nil
	}

	// Copy so the store never aliases the caller's slice.
	appended := make([]E, len(current), len(current)+len(events))
	copy(appended, current)
	appended = append(appended, events...)
	s.streams[streamID] = appended
	listeners := s.subscribers
	s.mu.Unlock()

	// Notify outside the lock so a listener may read the store without deadlock.
	for _, notify := range listeners {
		notify(events)
	}
	return nil
}

func (s *InMemory[E]) Load(ctx context.Context, streamID string) ([]E, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	stream := s.streams[streamID]
	out := make([]E, len(stream))
	copy(out, stream)
	return out, nil
}
