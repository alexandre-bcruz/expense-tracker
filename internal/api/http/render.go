package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alexandre-bcruz/expense-tracker/internal/command"
	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
	"github.com/alexandre-bcruz/expense-tracker/internal/query"
)

type expenseResponse struct {
	ID          string `json:"id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IncurredOn  string `json:"incurred_on"`
}

func toResponse(v query.ExpenseView) expenseResponse {
	return expenseResponse{
		ID:          v.ID,
		AmountMinor: v.AmountMinor,
		Currency:    v.Currency,
		Category:    v.Category,
		Description: v.Description,
		IncurredOn:  v.IncurredOn.Format("2006-01-02"),
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

// decode reads a JSON body into dst, writing a 400 and returning false on
// failure.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid JSON body"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, statusFor(err), errorResponse{err.Error()})
}

// statusFor maps domain and application errors to HTTP status codes.
func statusFor(err error) int {
	switch {
	case errors.Is(err, command.ErrNotFound), errors.Is(err, query.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, command.ErrAlreadyExists),
		errors.Is(err, eventstore.ErrConcurrencyConflict),
		errors.Is(err, expense.ErrAlreadyDeleted):
		return http.StatusConflict
	case errors.Is(err, expense.ErrNonPositiveAmount),
		errors.Is(err, expense.ErrMissingCategory),
		errors.Is(err, expense.ErrMissingID),
		errors.Is(err, expense.ErrInvalidCurrency),
		errors.Is(err, expense.ErrCurrencyMismatch):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
