// Package query is the read side of the CQRS split: read-optimized views of
// expenses and the handlers that serve them. Views are built by the projection
// package from domain events; queries never rebuild aggregates from the store.
package query

import (
	"errors"
	"time"
)

// ErrNotFound is returned when no view exists for an expense ID.
var ErrNotFound = errors.New("query: expense not found")

// ExpenseView is the read model of a single expense: a flat, query-optimized
// snapshot of the aggregate's current state.
type ExpenseView struct {
	ID          string
	AmountMinor int64
	Currency    string
	Category    string
	Description string
	IncurredOn  time.Time
}
