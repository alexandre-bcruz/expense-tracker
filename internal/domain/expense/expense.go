// Package expense contains the Expense aggregate: the write model, whose state
// is derived by replaying the domain events it has emitted (Event Sourcing).
package expense

import "time"

// Expense is the aggregate root; its state is the fold of its events.
type Expense struct {
	id          string
	amount      Money
	category    string
	description string
	incurredOn  time.Time
	deleted     bool

	version int
	changes []Event
}

// Record creates a new expense, emitting ExpenseRecorded.
func Record(id string, amount Money, category, description string, incurredOn, now time.Time) (*Expense, error) {
	if id == "" {
		return nil, ErrMissingID
	}
	if category == "" {
		return nil, ErrMissingCategory
	}
	if !amount.IsPositive() {
		return nil, ErrNonPositiveAmount
	}

	e := &Expense{}
	e.raise(ExpenseRecorded{
		ExpenseID:   id,
		Amount:      amount,
		Category:    category,
		Description: description,
		IncurredOn:  incurredOn,
		At:          now,
	})
	return e, nil
}

// Load rebuilds an expense by replaying its history; it raises no new events.
func Load(history []Event) *Expense {
	e := &Expense{}
	for _, evt := range history {
		e.apply(evt)
	}
	return e
}

// CorrectAmount fixes the amount. It must be positive and in the same currency;
// a no-op raises no event.
func (e *Expense) CorrectAmount(newAmount Money, now time.Time) error {
	if e.deleted {
		return ErrAlreadyDeleted
	}
	if !newAmount.IsPositive() {
		return ErrNonPositiveAmount
	}
	if !newAmount.SameCurrency(e.amount) {
		return ErrCurrencyMismatch
	}
	if newAmount == e.amount {
		return nil
	}
	e.raise(ExpenseAmountCorrected{ExpenseID: e.id, OldAmount: e.amount, NewAmount: newAmount, At: now})
	return nil
}

// Recategorize moves the expense to a new category; a no-op raises no event.
func (e *Expense) Recategorize(newCategory string, now time.Time) error {
	if e.deleted {
		return ErrAlreadyDeleted
	}
	if newCategory == "" {
		return ErrMissingCategory
	}
	if newCategory == e.category {
		return nil
	}
	e.raise(ExpenseRecategorized{ExpenseID: e.id, OldCategory: e.category, NewCategory: newCategory, At: now})
	return nil
}

// Delete logically removes the expense; deleting twice is an error.
func (e *Expense) Delete(reason string, now time.Time) error {
	if e.deleted {
		return ErrAlreadyDeleted
	}
	e.raise(ExpenseDeleted{ExpenseID: e.id, Reason: reason, At: now})
	return nil
}

func (e *Expense) raise(evt Event) {
	e.apply(evt)
	e.changes = append(e.changes, evt)
}

// apply must stay free of validation and side effects so replaying history is
// deterministic.
func (e *Expense) apply(evt Event) {
	switch ev := evt.(type) {
	case ExpenseRecorded:
		e.id = ev.ExpenseID
		e.amount = ev.Amount
		e.category = ev.Category
		e.description = ev.Description
		e.incurredOn = ev.IncurredOn
	case ExpenseAmountCorrected:
		e.amount = ev.NewAmount
	case ExpenseRecategorized:
		e.category = ev.NewCategory
	case ExpenseDeleted:
		e.deleted = true
	}
	e.version++
}

// UncommittedChanges returns events raised since load; the store persists them
// and then calls ClearUncommittedChanges.
func (e *Expense) UncommittedChanges() []Event { return e.changes }

// ClearUncommittedChanges marks pending events as persisted.
func (e *Expense) ClearUncommittedChanges() { e.changes = nil }

// Version is the number of applied events, used as the expected version for
// optimistic concurrency.
func (e *Expense) Version() int { return e.version }

func (e *Expense) ID() string            { return e.id }
func (e *Expense) Amount() Money         { return e.amount }
func (e *Expense) Category() string      { return e.category }
func (e *Expense) Description() string   { return e.description }
func (e *Expense) IncurredOn() time.Time { return e.incurredOn }
func (e *Expense) IsDeleted() bool       { return e.deleted }
