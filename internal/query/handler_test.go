package query

import (
	"context"
	"errors"
	"testing"
	"time"
)

func view(id, category string, minor int64) ExpenseView {
	return ExpenseView{ID: id, AmountMinor: minor, Currency: "BRL", Category: category, IncurredOn: time.Unix(0, 0)}
}

func TestGetExpense(t *testing.T) {
	store := NewExpenseStore()
	store.Save(view("e1", "food", 1500))
	h := NewHandler(store)

	got, err := h.GetExpense(context.Background(), "e1")
	if err != nil {
		t.Fatalf("GetExpense: %v", err)
	}
	if got.AmountMinor != 1500 || got.Category != "food" {
		t.Fatalf("unexpected view: %+v", got)
	}
}

func TestGetExpenseNotFound(t *testing.T) {
	h := NewHandler(NewExpenseStore())
	if _, err := h.GetExpense(context.Background(), "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListExpensesOrderedByID(t *testing.T) {
	store := NewExpenseStore()
	store.Save(view("e2", "travel", 500))
	store.Save(view("e1", "food", 1500))
	h := NewHandler(store)

	got, err := h.ListExpenses(context.Background())
	if err != nil {
		t.Fatalf("ListExpenses: %v", err)
	}
	if len(got) != 2 || got[0].ID != "e1" || got[1].ID != "e2" {
		t.Fatalf("list not ordered by id: %+v", got)
	}
}

func TestStoreMutations(t *testing.T) {
	store := NewExpenseStore()
	store.Save(view("e1", "food", 1500))

	store.SetAmount("e1", 1800)
	store.SetCategory("e1", "groceries")
	if v, _ := store.Get("e1"); v.AmountMinor != 1800 || v.Category != "groceries" {
		t.Fatalf("mutations not applied: %+v", v)
	}

	store.Delete("e1")
	if _, ok := store.Get("e1"); ok {
		t.Fatal("expense should be gone after Delete")
	}

	// Mutating an unknown ID is a no-op, not a panic.
	store.SetAmount("ghost", 99)
}
