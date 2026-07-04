# Expense Tracker

An event-sourced expense tracker written in Go, built to demonstrate
**Event Sourcing** and **CQRS** in a small but realistic domain.

State is never stored directly: every change is an immutable domain event
appended to a stream, and an aggregate's current state is the fold of its
events. This is deliberately more machinery than a CRUD expense tracker needs —
the goal is to show these patterns hands-on. The trade-off is documented
explicitly in [ADR-001](docs/adr/ADR-001.md).

## Architecture

The write side is an event-sourced aggregate behind a command handler. Writes
append events to the store, a projection subscribed to the store folds them into
a read model, and the query side serves reads from that model.

```
cmd/api/              HTTP server entrypoint (wires everything together)
internal/
├── domain/expense/   Expense aggregate, domain events, Money value object
├── eventstore/       Store[E] contract + in-memory implementation
├── command/          CQRS write side: command handler over the store
├── projection/       Folds events into the read model
├── query/            CQRS read side: read model + query handler
├── api/http/         REST transport over the command and query handlers
└── infrastructure/
    └── postgres/     Durable PostgreSQL event store
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

## Running

```bash
make run                 # start the API on :8080 (ADDR=:9090 to override)
make test                # run all tests
make help                # list all targets
```

By default the store is in-memory, so data resets on restart.

### Persistence (PostgreSQL)

Set `DATABASE_URL` to use the durable event store. On startup the app migrates
the schema and rebuilds the read model by replaying the event log.

```bash
make db-up                                    # start Postgres via docker compose
export DATABASE_URL=postgres://expense:expense@localhost:5432/expense?sslmode=disable
make run                                       # now persists across restarts
make test-integration                          # run the Postgres integration tests
make db-down                                    # stop and remove the database
```

### API

| Method | Path                        | Action                     |
|--------|-----------------------------|----------------------------|
| POST   | `/expenses`                 | record an expense          |
| GET    | `/expenses`                 | list expenses              |
| GET    | `/expenses/{id}`            | get one expense            |
| PATCH  | `/expenses/{id}/amount`     | correct the amount         |
| PATCH  | `/expenses/{id}/category`   | recategorize               |
| DELETE | `/expenses/{id}`            | delete an expense          |

```bash
curl -X POST localhost:8080/expenses \
  -d '{"amount_minor":1599,"currency":"BRL","category":"food","description":"lunch"}'
curl localhost:8080/expenses
```

Amounts are integers in the currency's minor unit (`1599` = 15.99 BRL).

## Roadmap

- [x] Expense aggregate and domain events
- [x] Generic event store (in-memory)
- [x] Command handler (write side)
- [x] Read side: projection and queries
- [x] HTTP API
- [x] Durable event store (PostgreSQL)

## Contributing

Commit conventions and the commit-msg hook are described in
[CONTRIBUTING.md](CONTRIBUTING.md).
