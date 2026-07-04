package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
)

var fixedNow = time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)

func newHandler() (*Handler, eventstore.Store[expense.Event]) {
	store := eventstore.NewInMemory[expense.Event]()
	h := NewHandler(store, func() time.Time { return fixedNow })
	return h, store
}

func seedExpense(t *testing.T, h *Handler) {
	t.Helper()
	err := h.RecordExpense(context.Background(), RecordExpense{
		ExpenseID:   "exp-1",
		AmountMinor: 1500,
		Currency:    "BRL",
		Category:    "food",
		Description: "lunch",
		IncurredOn:  fixedNow,
	})
	if err != nil {
		t.Fatalf("seed RecordExpense: %v", err)
	}
}

func TestRecordExpensePersistsEvent(t *testing.T) {
	h, store := newHandler()
	seedExpense(t, h)

	history, _ := store.Load(context.Background(), "exp-1")
	if len(history) != 1 {
		t.Fatalf("stream length = %d, want 1", len(history))
	}
	if history[0].EventName() != "expense.recorded" {
		t.Fatalf("event = %q, want expense.recorded", history[0].EventName())
	}
}

func TestRecordExpenseRejectsDuplicate(t *testing.T) {
	h, _ := newHandler()
	seedExpense(t, h)

	err := h.RecordExpense(context.Background(), RecordExpense{
		ExpenseID: "exp-1", AmountMinor: 100, Currency: "BRL", Category: "food",
	})
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("err = %v, want ErrAlreadyExists", err)
	}
}

func TestRecordExpenseValidationPropagates(t *testing.T) {
	h, store := newHandler()

	// Invalid currency is rejected by the Money value object.
	err := h.RecordExpense(context.Background(), RecordExpense{
		ExpenseID: "exp-1", AmountMinor: 100, Currency: "reais", Category: "food",
	})
	if !errors.Is(err, expense.ErrInvalidCurrency) {
		t.Fatalf("err = %v, want ErrInvalidCurrency", err)
	}
	if got, _ := store.Load(context.Background(), "exp-1"); len(got) != 0 {
		t.Fatalf("stream length = %d, want 0 (rejected command writes nothing)", len(got))
	}
}

func TestCorrectAmountAppendsToStream(t *testing.T) {
	h, store := newHandler()
	seedExpense(t, h)

	err := h.CorrectExpenseAmount(context.Background(), CorrectExpenseAmount{
		ExpenseID: "exp-1", AmountMinor: 1800, Currency: "BRL",
	})
	if err != nil {
		t.Fatalf("CorrectExpenseAmount: %v", err)
	}

	history, _ := store.Load(context.Background(), "exp-1")
	if len(history) != 2 {
		t.Fatalf("stream length = %d, want 2", len(history))
	}
	// Rebuilding from the stream must reflect the correction.
	e := expense.Load(history)
	if e.Amount().Amount() != 1800 {
		t.Fatalf("amount = %d, want 1800", e.Amount().Amount())
	}
}

func TestCommandsOnMissingExpense(t *testing.T) {
	h, _ := newHandler()
	ctx := context.Background()

	tests := map[string]func() error{
		"correct": func() error {
			return h.CorrectExpenseAmount(ctx, CorrectExpenseAmount{ExpenseID: "ghost", AmountMinor: 1, Currency: "BRL"})
		},
		"recategorize": func() error {
			return h.RecategorizeExpense(ctx, RecategorizeExpense{ExpenseID: "ghost", NewCategory: "x"})
		},
		"delete": func() error { return h.DeleteExpense(ctx, DeleteExpense{ExpenseID: "ghost"}) },
	}
	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, ErrNotFound) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestDomainRuleViolationDoesNotWrite(t *testing.T) {
	h, store := newHandler()
	seedExpense(t, h)

	// Correcting to a different currency violates a domain rule.
	err := h.CorrectExpenseAmount(context.Background(), CorrectExpenseAmount{
		ExpenseID: "exp-1", AmountMinor: 1800, Currency: "USD",
	})
	if !errors.Is(err, expense.ErrCurrencyMismatch) {
		t.Fatalf("err = %v, want ErrCurrencyMismatch", err)
	}
	if got, _ := store.Load(context.Background(), "exp-1"); len(got) != 1 {
		t.Fatalf("stream length = %d, want 1 (violation must not append)", len(got))
	}
}

func TestDeleteThenCorrectFails(t *testing.T) {
	h, store := newHandler()
	seedExpense(t, h)
	ctx := context.Background()

	if err := h.DeleteExpense(ctx, DeleteExpense{ExpenseID: "exp-1", Reason: "dup"}); err != nil {
		t.Fatalf("DeleteExpense: %v", err)
	}
	err := h.CorrectExpenseAmount(ctx, CorrectExpenseAmount{ExpenseID: "exp-1", AmountMinor: 200, Currency: "BRL"})
	if !errors.Is(err, expense.ErrAlreadyDeleted) {
		t.Fatalf("err = %v, want ErrAlreadyDeleted", err)
	}

	history, _ := store.Load(ctx, "exp-1")
	if len(history) != 2 { // recorded + deleted
		t.Fatalf("stream length = %d, want 2", len(history))
	}
}

func TestNoOpCommandWritesNothing(t *testing.T) {
	h, store := newHandler()
	seedExpense(t, h)

	// Recategorizing to the same category is a domain no-op.
	err := h.RecategorizeExpense(context.Background(), RecategorizeExpense{
		ExpenseID: "exp-1", NewCategory: "food",
	})
	if err != nil {
		t.Fatalf("RecategorizeExpense: %v", err)
	}
	if got, _ := store.Load(context.Background(), "exp-1"); len(got) != 1 {
		t.Fatalf("stream length = %d, want 1 (no-op writes nothing)", len(got))
	}
}
