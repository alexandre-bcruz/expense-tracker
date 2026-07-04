package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexandre-bcruz/expense-tracker/internal/command"
	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
	"github.com/alexandre-bcruz/expense-tracker/internal/projection"
	"github.com/alexandre-bcruz/expense-tracker/internal/query"
)

// newTestServer wires the full stack: event store, command handler, and a
// projection subscribed to the store that feeds the query read model.
func newTestServer() http.Handler {
	store := eventstore.NewInMemory[expense.Event]()
	readModel := query.NewExpenseStore()
	projector := projection.NewExpenseProjector(readModel)
	store.Subscribe(func(events []expense.Event) { projector.Project(events...) })

	return NewServer(command.NewHandler(store, time.Now), query.NewHandler(readModel)).Routes()
}

func do(t *testing.T, srv http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, r)
	return rec
}

// A record then a list proves the write flows through the projection into the
// read model end to end.
func TestRecordThenListReflectsWrite(t *testing.T) {
	srv := newTestServer()

	rec := do(t, srv, "POST", "/expenses",
		`{"amount_minor":1500,"currency":"BRL","category":"food","description":"lunch"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201 (%s)", rec.Code, rec.Body)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.ID == "" {
		t.Fatal("POST did not return an id")
	}

	rec = do(t, srv, "GET", "/expenses", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rec.Code)
	}
	var list []expenseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0].AmountMinor != 1500 || list[0].Category != "food" {
		t.Fatalf("list = %+v, want one food/1500 expense", list)
	}
}

func TestCorrectAmountUpdatesReadModel(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, "POST", "/expenses", `{"amount_minor":1500,"currency":"BRL","category":"food"}`)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	rec = do(t, srv, "PATCH", "/expenses/"+created.ID+"/amount", `{"amount_minor":1800,"currency":"BRL"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %d, want 204", rec.Code)
	}

	rec = do(t, srv, "GET", "/expenses/"+created.ID, "")
	var got expenseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.AmountMinor != 1800 {
		t.Fatalf("amount = %d, want 1800", got.AmountMinor)
	}
}

func TestDeleteRemovesFromReadModel(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, "POST", "/expenses", `{"amount_minor":500,"currency":"BRL","category":"food"}`)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	if rec = do(t, srv, "DELETE", "/expenses/"+created.ID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", rec.Code)
	}
	if rec = do(t, srv, "GET", "/expenses/"+created.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("GET after delete = %d, want 404", rec.Code)
	}
}

func TestErrorStatuses(t *testing.T) {
	srv := newTestServer()

	if rec := do(t, srv, "GET", "/expenses/ghost", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown GET = %d, want 404", rec.Code)
	}
	// Invalid currency is a domain validation error -> 400.
	if rec := do(t, srv, "POST", "/expenses", `{"amount_minor":100,"currency":"reais","category":"food"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad currency = %d, want 400", rec.Code)
	}
	// Non-positive amount -> 400.
	if rec := do(t, srv, "POST", "/expenses", `{"amount_minor":0,"currency":"BRL","category":"food"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("zero amount = %d, want 400", rec.Code)
	}
	// Malformed JSON -> 400.
	if rec := do(t, srv, "POST", "/expenses", `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json = %d, want 400", rec.Code)
	}
}
