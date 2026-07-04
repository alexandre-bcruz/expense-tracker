// Package command is the write side of the CQRS split: it turns commands into
// state changes by loading an aggregate, invoking domain behavior, and
// appending the resulting events under optimistic concurrency control.
package command

import (
	"context"
	"errors"
	"time"

	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
)

var (
	ErrAlreadyExists = errors.New("command: expense already exists")
	ErrNotFound      = errors.New("command: expense not found")
)

// Clock returns the current time; injected so tests can pin it.
type Clock func() time.Time

// Handler executes expense commands against an event store.
type Handler struct {
	store eventstore.Store[expense.Event]
	now   Clock
}

// NewHandler wires a handler to a store, defaulting to time.Now when clock is nil.
func NewHandler(store eventstore.Store[expense.Event], clock Clock) *Handler {
	if clock == nil {
		clock = time.Now
	}
	return &Handler{store: store, now: clock}
}

type RecordExpense struct {
	ExpenseID   string
	AmountMinor int64
	Currency    string
	Category    string
	Description string
	IncurredOn  time.Time
}

func (h *Handler) RecordExpense(ctx context.Context, cmd RecordExpense) error {
	history, err := h.store.Load(ctx, cmd.ExpenseID)
	if err != nil {
		return err
	}
	if len(history) > 0 {
		return ErrAlreadyExists
	}

	amount, err := expense.NewMoney(cmd.AmountMinor, cmd.Currency)
	if err != nil {
		return err
	}
	e, err := expense.Record(cmd.ExpenseID, amount, cmd.Category, cmd.Description, cmd.IncurredOn, h.now())
	if err != nil {
		return err
	}
	return h.save(ctx, e)
}

type CorrectExpenseAmount struct {
	ExpenseID   string
	AmountMinor int64
	Currency    string
}

func (h *Handler) CorrectExpenseAmount(ctx context.Context, cmd CorrectExpenseAmount) error {
	e, err := h.load(ctx, cmd.ExpenseID)
	if err != nil {
		return err
	}
	amount, err := expense.NewMoney(cmd.AmountMinor, cmd.Currency)
	if err != nil {
		return err
	}
	if err := e.CorrectAmount(amount, h.now()); err != nil {
		return err
	}
	return h.save(ctx, e)
}

type RecategorizeExpense struct {
	ExpenseID   string
	NewCategory string
}

func (h *Handler) RecategorizeExpense(ctx context.Context, cmd RecategorizeExpense) error {
	e, err := h.load(ctx, cmd.ExpenseID)
	if err != nil {
		return err
	}
	if err := e.Recategorize(cmd.NewCategory, h.now()); err != nil {
		return err
	}
	return h.save(ctx, e)
}

type DeleteExpense struct {
	ExpenseID string
	Reason    string
}

func (h *Handler) DeleteExpense(ctx context.Context, cmd DeleteExpense) error {
	e, err := h.load(ctx, cmd.ExpenseID)
	if err != nil {
		return err
	}
	if err := e.Delete(cmd.Reason, h.now()); err != nil {
		return err
	}
	return h.save(ctx, e)
}

func (h *Handler) load(ctx context.Context, id string) (*expense.Expense, error) {
	history, err := h.store.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, ErrNotFound
	}
	return expense.Load(history), nil
}

// save appends uncommitted events using the pre-command version as the expected
// version, then marks them committed. A no-op command writes nothing.
func (h *Handler) save(ctx context.Context, e *expense.Expense) error {
	changes := e.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}
	expectedVersion := e.Version() - len(changes)
	if err := h.store.Append(ctx, e.ID(), expectedVersion, changes...); err != nil {
		return err
	}
	e.ClearUncommittedChanges()
	return nil
}
