package expense

import (
	"errors"
	"testing"
	"time"
)

func brl(cents int64) Money {
	m, err := NewMoney(cents, "BRL")
	if err != nil {
		panic(err)
	}
	return m
}

var (
	incurred = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	now      = time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
)

func mustRecord(t *testing.T) *Expense {
	t.Helper()
	e, err := Record("exp-1", brl(1500), "food", "lunch", incurred, now)
	if err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}
	return e
}

func TestRecord(t *testing.T) {
	e := mustRecord(t)

	if e.ID() != "exp-1" || e.Category() != "food" || e.Amount() != brl(1500) {
		t.Fatalf("unexpected state: %+v", e)
	}
	if e.Version() != 1 {
		t.Fatalf("version = %d, want 1", e.Version())
	}

	changes := e.UncommittedChanges()
	if len(changes) != 1 {
		t.Fatalf("uncommitted changes = %d, want 1", len(changes))
	}
	if _, ok := changes[0].(ExpenseRecorded); !ok {
		t.Fatalf("first event = %T, want ExpenseRecorded", changes[0])
	}
}

func TestRecordValidation(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		amount   Money
		category string
		want     error
	}{
		{"missing id", "", brl(100), "food", ErrMissingID},
		{"missing category", "exp-1", brl(100), "", ErrMissingCategory},
		{"zero amount", "exp-1", brl(0), "food", ErrNonPositiveAmount},
		{"negative amount", "exp-1", brl(-100), "food", ErrNonPositiveAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Record(tt.id, tt.amount, tt.category, "", incurred, now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCorrectAmount(t *testing.T) {
	e := mustRecord(t)

	if err := e.CorrectAmount(brl(1800), now); err != nil {
		t.Fatalf("CorrectAmount: %v", err)
	}
	if e.Amount() != brl(1800) {
		t.Fatalf("amount = %v, want 18.00 BRL", e.Amount())
	}
	if e.Version() != 2 {
		t.Fatalf("version = %d, want 2", e.Version())
	}

	last := e.UncommittedChanges()[1].(ExpenseAmountCorrected)
	if last.OldAmount != brl(1500) || last.NewAmount != brl(1800) {
		t.Fatalf("unexpected correction event: %+v", last)
	}
}

func TestCorrectAmountRejectsBadInput(t *testing.T) {
	usd, _ := NewMoney(1800, "USD")
	tests := []struct {
		name   string
		amount Money
		want   error
	}{
		{"non-positive", brl(0), ErrNonPositiveAmount},
		{"currency mismatch", usd, ErrCurrencyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := mustRecord(t)
			if err := e.CorrectAmount(tt.amount, now); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if e.Version() != 1 {
				t.Fatalf("version = %d, want 1 (no event on rejection)", e.Version())
			}
		})
	}
}

func TestNoOpTransitionsRaiseNoEvent(t *testing.T) {
	e := mustRecord(t)

	if err := e.CorrectAmount(brl(1500), now); err != nil {
		t.Fatalf("CorrectAmount: %v", err)
	}
	if err := e.Recategorize("food", now); err != nil {
		t.Fatalf("Recategorize: %v", err)
	}
	if e.Version() != 1 {
		t.Fatalf("version = %d, want 1 (no-ops raise no events)", e.Version())
	}
}

func TestDelete(t *testing.T) {
	e := mustRecord(t)

	if err := e.Delete("duplicate", now); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !e.IsDeleted() {
		t.Fatal("expense should be deleted")
	}
	if e.Version() != 2 {
		t.Fatalf("version = %d, want 2", e.Version())
	}
	if _, ok := e.UncommittedChanges()[1].(ExpenseDeleted); !ok {
		t.Fatalf("last event = %T, want ExpenseDeleted", e.UncommittedChanges()[1])
	}
}

func TestDeleteGuards(t *testing.T) {
	e := mustRecord(t)
	if err := e.Delete("first", now); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := e.Delete("again", now); !errors.Is(err, ErrAlreadyDeleted) {
		t.Fatalf("second Delete err = %v, want ErrAlreadyDeleted", err)
	}
	if err := e.CorrectAmount(brl(200), now); !errors.Is(err, ErrAlreadyDeleted) {
		t.Fatalf("CorrectAmount after delete err = %v, want ErrAlreadyDeleted", err)
	}
	if err := e.Recategorize("travel", now); !errors.Is(err, ErrAlreadyDeleted) {
		t.Fatalf("Recategorize after delete err = %v, want ErrAlreadyDeleted", err)
	}
}

func TestLoadReplaysHistory(t *testing.T) {
	// Build a stream by exercising the aggregate, then replay it fresh.
	src := mustRecord(t)
	_ = src.CorrectAmount(brl(2000), now)
	_ = src.Recategorize("groceries", now)
	history := src.UncommittedChanges()

	loaded := Load(history)

	if loaded.Amount() != brl(2000) {
		t.Fatalf("amount = %v, want 20.00 BRL", loaded.Amount())
	}
	if loaded.Category() != "groceries" {
		t.Fatalf("category = %q, want groceries", loaded.Category())
	}
	if loaded.Version() != len(history) {
		t.Fatalf("version = %d, want %d", loaded.Version(), len(history))
	}
	if len(loaded.UncommittedChanges()) != 0 {
		t.Fatal("a loaded aggregate must have no uncommitted changes")
	}
}
