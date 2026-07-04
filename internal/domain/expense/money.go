package expense

import (
	"encoding/json"
	"fmt"
)

// Money is a monetary amount in the minor unit of its currency (e.g. cents),
// held as an integer to avoid floating-point rounding — a hard requirement for
// a financial domain. It is immutable and comparable with ==.
type Money struct {
	amount   int64
	currency string // ISO 4217 alphabetic code
}

// NewMoney builds a Money, validating the currency code. Domain rules such as
// "an expense amount must be positive" belong to the aggregate, not here.
func NewMoney(amount int64, currency string) (Money, error) {
	if len(currency) != 3 {
		return Money{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
	}
	for _, r := range currency {
		if r < 'A' || r > 'Z' {
			return Money{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
		}
	}
	return Money{amount: amount, currency: currency}, nil
}

func (m Money) Amount() int64             { return m.amount }
func (m Money) Currency() string          { return m.currency }
func (m Money) IsPositive() bool          { return m.amount > 0 }
func (m Money) SameCurrency(o Money) bool { return m.currency == o.currency }

// moneyJSON is the on-the-wire shape of Money, needed because its fields are
// unexported and would otherwise not be (de)serialized.
type moneyJSON struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(moneyJSON{AmountMinor: m.amount, Currency: m.currency})
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var aux moneyJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.amount = aux.AmountMinor
	m.currency = aux.Currency
	return nil
}

// String renders the amount for logs, assuming a two-decimal currency.
func (m Money) String() string {
	return fmt.Sprintf("%d.%02d %s", m.amount/100, abs(m.amount%100), m.currency)
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
