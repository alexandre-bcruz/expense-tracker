package projection

import (
	"testing"
	"time"

	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/query"
)

func brl(t *testing.T, minor int64) expense.Money {
	t.Helper()
	m, err := expense.NewMoney(minor, "BRL")
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

// The projector should turn a stream of events into the current read view,
// mirroring how the aggregate folds the same events into its state.
func TestProjectBuildsAndUpdatesView(t *testing.T) {
	store := query.NewExpenseStore()
	p := NewExpenseProjector(store)
	now := time.Unix(0, 0)

	p.Project(expense.ExpenseRecorded{
		ExpenseID: "e1", Amount: brl(t, 1500), Category: "food",
		Description: "lunch", IncurredOn: now, At: now,
	})
	if v, ok := store.Get("e1"); !ok || v.AmountMinor != 1500 || v.Category != "food" {
		t.Fatalf("after record: %+v (ok=%v)", v, ok)
	}

	p.Project(expense.ExpenseAmountCorrected{ExpenseID: "e1", NewAmount: brl(t, 1800), At: now})
	p.Project(expense.ExpenseRecategorized{ExpenseID: "e1", NewCategory: "groceries", At: now})
	if v, _ := store.Get("e1"); v.AmountMinor != 1800 || v.Category != "groceries" {
		t.Fatalf("after corrections: %+v", v)
	}

	p.Project(expense.ExpenseDeleted{ExpenseID: "e1", At: now})
	if _, ok := store.Get("e1"); ok {
		t.Fatal("deleted expense must not remain in the read model")
	}
}

func TestProjectReplayReconstructsView(t *testing.T) {
	// A fresh store fed the whole history must reach the same state.
	now := time.Unix(0, 0)
	history := []expense.Event{
		expense.ExpenseRecorded{ExpenseID: "e1", Amount: brl(t, 1000), Category: "food", IncurredOn: now, At: now},
		expense.ExpenseAmountCorrected{ExpenseID: "e1", NewAmount: brl(t, 1200), At: now},
	}

	store := query.NewExpenseStore()
	NewExpenseProjector(store).Project(history...)

	if v, _ := store.Get("e1"); v.AmountMinor != 1200 {
		t.Fatalf("replay amount = %d, want 1200", v.AmountMinor)
	}
}
