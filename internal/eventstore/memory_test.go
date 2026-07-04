package eventstore

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeEvent is a minimal Event used to test the store in isolation from any
// domain package.
type fakeEvent struct {
	name string
	at   time.Time
}

func (f fakeEvent) EventName() string     { return f.name }
func (f fakeEvent) OccurredAt() time.Time { return f.at }

func ev(name string) fakeEvent { return fakeEvent{name: name, at: time.Unix(0, 0)} }

func TestAppendThenLoad(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory[fakeEvent]()

	if err := s.Append(ctx, "stream-1", 0, ev("a"), ev("b")); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := s.Load(ctx, "stream-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 || got[0].name != "a" || got[1].name != "b" {
		t.Fatalf("Load = %+v, want [a b] in order", got)
	}
}

func TestAppendIsSequential(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory[fakeEvent]()

	if err := s.Append(ctx, "s", 0, ev("a")); err != nil {
		t.Fatalf("first Append: %v", err)
	}
	if err := s.Append(ctx, "s", 1, ev("b")); err != nil {
		t.Fatalf("second Append: %v", err)
	}

	got, _ := s.Load(ctx, "s")
	if len(got) != 2 || got[0].name != "a" || got[1].name != "b" {
		t.Fatalf("Load = %+v, want [a b]", got)
	}
}

func TestAppendConcurrencyConflict(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory[fakeEvent]()
	if err := s.Append(ctx, "s", 0, ev("a")); err != nil {
		t.Fatalf("seed Append: %v", err)
	}

	// Stream is at version 1; appending with a stale expectedVersion fails.
	err := s.Append(ctx, "s", 0, ev("b"))
	if !errors.Is(err, ErrConcurrencyConflict) {
		t.Fatalf("err = %v, want ErrConcurrencyConflict", err)
	}

	// And nothing was written.
	got, _ := s.Load(ctx, "s")
	if len(got) != 1 {
		t.Fatalf("stream length = %d, want 1 (conflict must not write)", len(got))
	}
}

func TestLoadUnknownStream(t *testing.T) {
	s := NewInMemory[fakeEvent]()
	got, err := s.Load(context.Background(), "nope")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Load = %+v, want empty", got)
	}
}

func TestAppendEmptyIsNoOpButChecksVersion(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory[fakeEvent]()

	if err := s.Append(ctx, "s", 0); err != nil {
		t.Fatalf("empty Append at v0: %v", err)
	}
	if err := s.Append(ctx, "s", 5); !errors.Is(err, ErrConcurrencyConflict) {
		t.Fatalf("empty Append at wrong version err = %v, want conflict", err)
	}
}

func TestLoadReturnsCopy(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory[fakeEvent]()
	_ = s.Append(ctx, "s", 0, ev("a"))

	got, _ := s.Load(ctx, "s")
	got[0] = ev("tampered")

	again, _ := s.Load(ctx, "s")
	if again[0].name != "a" {
		t.Fatalf("store was mutated through returned slice: %+v", again)
	}
}

func TestSubscribeReceivesAppendedEvents(t *testing.T) {
	ctx := context.Background()
	s := NewInMemory[fakeEvent]()

	var got []string
	s.Subscribe(func(events []fakeEvent) {
		for _, e := range events {
			got = append(got, e.name)
		}
	})

	_ = s.Append(ctx, "s", 0, ev("a"), ev("b"))
	_ = s.Append(ctx, "s", 2, ev("c"))
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("subscriber received %v, want [a b c]", got)
	}

	// A conflicting append must not notify subscribers.
	_ = s.Append(ctx, "s", 0, ev("x"))
	if len(got) != 3 {
		t.Fatalf("subscriber notified on conflict: %v", got)
	}
}

func TestContextCancellation(t *testing.T) {
	s := NewInMemory[fakeEvent]()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.Append(ctx, "s", 0, ev("a")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Append err = %v, want context.Canceled", err)
	}
	if _, err := s.Load(ctx, "s"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Load err = %v, want context.Canceled", err)
	}
}
