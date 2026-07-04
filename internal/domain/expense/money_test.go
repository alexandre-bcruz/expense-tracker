package expense

import (
	"encoding/json"
	"testing"
)

func TestMoneyJSONRoundTrip(t *testing.T) {
	m, err := NewMoney(1599, "BRL")
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Money
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != m {
		t.Fatalf("round trip = %v, want %v", got, m)
	}
}

// Money's unexported fields must survive serialization when nested in an event,
// which is how the durable store persists amounts.
func TestMoneySurvivesInEvent(t *testing.T) {
	m, _ := NewMoney(1500, "BRL")
	data, err := json.Marshal(ExpenseRecorded{ExpenseID: "e1", Amount: m, Category: "food"})
	if err != nil {
		t.Fatalf("Marshal event: %v", err)
	}

	var got ExpenseRecorded
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal event: %v", err)
	}
	if got.Amount != m {
		t.Fatalf("amount lost in round trip: %v, want %v", got.Amount, m)
	}
}
