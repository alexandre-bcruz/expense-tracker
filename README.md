# Expense Tracker

An event-sourced expense tracker written in Go, built to demonstrate
**Event Sourcing** and **CQRS** in a small but realistic domain.

State is never stored directly: every change is an immutable domain event
appended to a stream, and an aggregate's current state is the fold of its
events. This is deliberately more machinery than a CRUD expense tracker needs —
the goal is to show these patterns hands-on. The trade-off is documented
explicitly in [ADR-001](docs/adr/ADR-001.md).

## Architecture

The write side is modeled as an event-sourced aggregate behind a command
handler; the read side (projections and queries) is on the roadmap.

```
internal/
├── domain/expense/   Expense aggregate, domain events, Money value object
├── eventstore/       Store[E] contract + in-memory implementation
└── command/          CQRS write side: command handler over the store
```

### Design notes

- **No floating point for money.** `Money` holds an integer amount in the
  currency's minor unit (e.g. cents), avoiding rounding errors.
- **Deterministic replay.** State mutates only in the aggregate's `apply` step,
  free of validation and side effects, so replaying a stream always rebuilds an
  identical aggregate.
- **Optimistic concurrency.** Appends carry the expected stream version and are
  rejected on conflict.
- **Typed event store.** `Store[E]` is generic over the event type, so callers
  keep full type safety instead of asserting on an untyped interface.

## Requirements

- Go 1.26+

## Running the tests

```bash
go test ./...
```

## Roadmap

- [x] Expense aggregate and domain events
- [x] Generic event store (in-memory)
- [x] Command handler (write side)
- [ ] Read side: projections and queries
- [ ] Durable event store (PostgreSQL)
- [ ] HTTP API

## Contributing

Commit conventions and the commit-msg hook are described in
[CONTRIBUTING.md](CONTRIBUTING.md).
