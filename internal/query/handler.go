package query

import "context"

// Handler serves read queries over the expense read model.
type Handler struct {
	store *ExpenseStore
}

func NewHandler(store *ExpenseStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) GetExpense(ctx context.Context, id string) (ExpenseView, error) {
	if v, ok := h.store.Get(id); ok {
		return v, nil
	}
	return ExpenseView{}, ErrNotFound
}

func (h *Handler) ListExpenses(ctx context.Context) ([]ExpenseView, error) {
	return h.store.List(), nil
}
