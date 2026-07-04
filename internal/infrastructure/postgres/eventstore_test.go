//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
)

// freshStore connects to DATABASE_URL, migrates, and truncates the events table
// so each test starts clean.
func freshStore(t *testing.T) *EventStore[expense.Event] {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	s := NewEventStore[expense.Event](pool, ExpenseCodec{})
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE events RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

func recorded(t *testing.T, id string, minor int64) expense.ExpenseRecorded {
	t.Helper()
	m, err := expense.NewMoney(minor, "BRL")
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return expense.ExpenseRecorded{ExpenseID: id, Amount: m, Category: "food", Description: "lunch", IncurredOn: now, At: now}
}

func TestAppendAndLoadRoundTrip(t *testing.T) {
	s := freshStore(t)
	ctx := context.Background()

	if err := s.Append(ctx, "e1", 0, recorded(t, "e1", 1500)); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := s.Load(ctx, "e1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	ev, ok := got[0].(expense.ExpenseRecorded)
	if !ok {
		t.Fatalf("event type = %T, want ExpenseRecorded", got[0])
	}
	if ev.Amount.Amount() != 1500 || ev.Amount.Currency() != "BRL" || ev.Category != "food" {
		t.Fatalf("round trip lost data: %+v", ev)
	}
}

func TestOptimisticConcurrencyConflict(t *testing.T) {
	s := freshStore(t)
	ctx := context.Background()

	if err := s.Append(ctx, "e1", 0, recorded(t, "e1", 1500)); err != nil {
		t.Fatalf("first Append: %v", err)
	}
	// Stream is at version 1; a stale expected version must conflict.
	err := s.Append(ctx, "e1", 0, recorded(t, "e1", 200))
	if !errors.Is(err, eventstore.ErrConcurrencyConflict) {
		t.Fatalf("err = %v, want ErrConcurrencyConflict", err)
	}

	got, _ := s.Load(ctx, "e1")
	if len(got) != 1 {
		t.Fatalf("stream length = %d, want 1 (conflict must not write)", len(got))
	}
}

func TestLoadUnknownStreamIsEmpty(t *testing.T) {
	s := freshStore(t)
	got, err := s.Load(context.Background(), "ghost")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestLoadAllAndSubscribe(t *testing.T) {
	s := freshStore(t)
	ctx := context.Background()

	var notified int
	s.Subscribe(func(events []expense.Event) { notified += len(events) })

	if err := s.Append(ctx, "e1", 0, recorded(t, "e1", 100)); err != nil {
		t.Fatal(err)
	}
	if err := s.Append(ctx, "e2", 0, recorded(t, "e2", 200)); err != nil {
		t.Fatal(err)
	}
	if notified != 2 {
		t.Fatalf("subscriber saw %d events, want 2", notified)
	}

	all, err := s.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("LoadAll returned %d events, want 2", len(all))
	}
}
