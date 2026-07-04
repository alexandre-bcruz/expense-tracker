package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
)

// Codec serializes events to and from their stored form — a type name and a
// JSON payload — so the generic EventStore can persist domain events without
// importing any specific aggregate.
type Codec[E eventstore.Event] interface {
	Marshal(e E) (eventType string, payload []byte, err error)
	Unmarshal(eventType string, payload []byte) (E, error)
}

// ExpenseCodec is the Codec for expense domain events.
type ExpenseCodec struct{}

func (ExpenseCodec) Marshal(e expense.Event) (string, []byte, error) {
	payload, err := json.Marshal(e)
	if err != nil {
		return "", nil, err
	}
	return e.EventName(), payload, nil
}

func (ExpenseCodec) Unmarshal(eventType string, payload []byte) (expense.Event, error) {
	switch eventType {
	case "expense.recorded":
		return unmarshal[expense.ExpenseRecorded](payload)
	case "expense.amount_corrected":
		return unmarshal[expense.ExpenseAmountCorrected](payload)
	case "expense.recategorized":
		return unmarshal[expense.ExpenseRecategorized](payload)
	case "expense.deleted":
		return unmarshal[expense.ExpenseDeleted](payload)
	default:
		return nil, fmt.Errorf("postgres: unknown event type %q", eventType)
	}
}

func unmarshal[T expense.Event](payload []byte) (expense.Event, error) {
	var ev T
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	return ev, nil
}
