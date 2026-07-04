// Package postgres provides a durable, PostgreSQL-backed event store that
// satisfies eventstore.Store. Events are stored as JSON with a per-stream
// version guarded by a unique constraint for optimistic concurrency.
package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
)

//go:embed schema.sql
var schema string

// EventStore persists event streams in PostgreSQL. It is generic over the event
// type and delegates (de)serialization to a Codec.
type EventStore[E eventstore.Event] struct {
	pool  *pgxpool.Pool
	codec Codec[E]

	mu          sync.RWMutex
	subscribers []func(events []E)
}

func NewEventStore[E eventstore.Event](pool *pgxpool.Pool, codec Codec[E]) *EventStore[E] {
	return &EventStore[E]{pool: pool, codec: codec}
}

// Migrate creates the events table if it does not exist.
func (s *EventStore[E]) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schema)
	return err
}

// Subscribe registers a listener invoked with the events of each successful
// Append, letting an in-process projection follow the store.
func (s *EventStore[E]) Subscribe(listener func(events []E)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers = append(s.subscribers, listener)
}

func (s *EventStore[E]) Append(ctx context.Context, streamID string, expectedVersion int, events ...E) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM events WHERE stream_id = $1`, streamID).Scan(&count); err != nil {
		return err
	}
	if count != expectedVersion {
		return fmt.Errorf("%w: stream %q is at version %d, expected %d",
			eventstore.ErrConcurrencyConflict, streamID, count, expectedVersion)
	}

	for i, e := range events {
		eventType, payload, err := s.codec.Marshal(e)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO events (stream_id, version, event_type, payload, occurred_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			streamID, expectedVersion+i+1, eventType, payload, e.OccurredAt())
		if err != nil {
			return mapWriteErr(err, streamID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return mapWriteErr(err, streamID)
	}

	if len(events) > 0 {
		s.notify(events)
	}
	return nil
}

func (s *EventStore[E]) Load(ctx context.Context, streamID string) ([]E, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT event_type, payload FROM events WHERE stream_id = $1 ORDER BY version`, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scan(rows)
}

// LoadAll returns every event across all streams in global append order, used
// to rebuild projections from scratch.
func (s *EventStore[E]) LoadAll(ctx context.Context) ([]E, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT event_type, payload FROM events ORDER BY global_seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scan(rows)
}

func (s *EventStore[E]) scan(rows pgx.Rows) ([]E, error) {
	out := make([]E, 0)
	for rows.Next() {
		var eventType string
		var payload []byte
		if err := rows.Scan(&eventType, &payload); err != nil {
			return nil, err
		}
		e, err := s.codec.Unmarshal(eventType, payload)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *EventStore[E]) notify(events []E) {
	s.mu.RLock()
	listeners := s.subscribers
	s.mu.RUnlock()
	for _, notify := range listeners {
		notify(events)
	}
}

// mapWriteErr turns a unique-violation on (stream_id, version) — the sign of a
// concurrent append — into ErrConcurrencyConflict.
func mapWriteErr(err error, streamID string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: stream %q was modified concurrently", eventstore.ErrConcurrencyConflict, streamID)
	}
	return err
}
