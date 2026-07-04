package expense

import "errors"

// Sentinel errors from the Expense aggregate; match with errors.Is.
var (
	ErrMissingID         = errors.New("expense: id is required")
	ErrMissingCategory   = errors.New("expense: category is required")
	ErrNonPositiveAmount = errors.New("expense: amount must be positive")
	ErrInvalidCurrency   = errors.New("expense: invalid ISO 4217 currency code")
	ErrCurrencyMismatch  = errors.New("expense: currency does not match the expense")
	ErrAlreadyDeleted    = errors.New("expense: expense is already deleted")
)
