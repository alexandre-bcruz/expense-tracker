package expense

import "time"

// Event is the contract every domain event implements. Events are immutable
// past-tense facts; infrastructure relies on EventName to persist and route them.
type Event interface {
	EventName() string
	OccurredAt() time.Time
}

// ExpenseRecorded is emitted when a new expense enters the system.
type ExpenseRecorded struct {
	ExpenseID   string
	Amount      Money
	Category    string
	Description string
	IncurredOn  time.Time
	At          time.Time
}

func (e ExpenseRecorded) EventName() string     { return "expense.recorded" }
func (e ExpenseRecorded) OccurredAt() time.Time { return e.At }

// ExpenseAmountCorrected is emitted when a recorded amount is fixed.
type ExpenseAmountCorrected struct {
	ExpenseID string
	OldAmount Money
	NewAmount Money
	At        time.Time
}

func (e ExpenseAmountCorrected) EventName() string     { return "expense.amount_corrected" }
func (e ExpenseAmountCorrected) OccurredAt() time.Time { return e.At }

// ExpenseRecategorized is emitted when an expense moves to a new category.
type ExpenseRecategorized struct {
	ExpenseID   string
	OldCategory string
	NewCategory string
	At          time.Time
}

func (e ExpenseRecategorized) EventName() string     { return "expense.recategorized" }
func (e ExpenseRecategorized) OccurredAt() time.Time { return e.At }

// ExpenseDeleted is emitted when an expense is logically removed.
type ExpenseDeleted struct {
	ExpenseID string
	Reason    string
	At        time.Time
}

func (e ExpenseDeleted) EventName() string     { return "expense.deleted" }
func (e ExpenseDeleted) OccurredAt() time.Time { return e.At }
