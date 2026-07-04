// Package eventstore defines the contract for persisting aggregate event
// streams, with an in-memory implementation for tests and local runs.
package eventstore

import (
	"context"
	"errors"
	"time"
)

// Event is the minimal contract the store needs from a domain event.
type Event interface {
	EventName() string
	OccurredAt() time.Time
}

// ErrConcurrencyConflict is returned by Append when a stream's version differs
// from the expected one.
var ErrConcurrencyConflict = errors.New("eventstore: concurrency conflict: stream modified since last read")

// Store persists ordered event streams keyed by stream ID, generic over the
// event type so callers keep type safety without untyped assertions.
type Store[E Event] interface {
	// Append writes events to the end of a stream. On expectedVersion mismatch
	// it returns ErrConcurrencyConflict and writes nothing.
	Append(ctx context.Context, streamID string, expectedVersion int, events ...E) error

	// Load returns a stream's events in order; an unknown stream yields empty.
	Load(ctx context.Context, streamID string) ([]E, error)
}
