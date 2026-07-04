// Command api runs the expense tracker's HTTP server. It selects an event store
// (PostgreSQL when DATABASE_URL is set, otherwise in-memory), wires a projection
// that feeds the query read model, and serves the REST API.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	httpapi "github.com/alexandre-bcruz/expense-tracker/internal/api/http"
	"github.com/alexandre-bcruz/expense-tracker/internal/command"
	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
	"github.com/alexandre-bcruz/expense-tracker/internal/infrastructure/postgres"
	"github.com/alexandre-bcruz/expense-tracker/internal/projection"
	"github.com/alexandre-bcruz/expense-tracker/internal/query"
)

// store is the subset of an event store the wiring needs: the Store contract
// plus in-process subscriptions for the projection.
type store interface {
	eventstore.Store[expense.Event]
	Subscribe(listener func(events []expense.Event))
}

func main() {
	ctx := context.Background()

	readModel := query.NewExpenseStore()
	projector := projection.NewExpenseProjector(readModel)

	es := openStore(ctx)
	es.Subscribe(func(events []expense.Event) { projector.Project(events...) })
	rebuildReadModel(ctx, es, projector)

	commands := command.NewHandler(es, time.Now)
	queries := query.NewHandler(readModel)
	server := httpapi.NewServer(commands, queries)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("expense-tracker listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

// openStore returns a durable PostgreSQL store when DATABASE_URL is set, and a
// non-durable in-memory store otherwise.
func openStore(ctx context.Context) store {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Print("using in-memory event store (data is not persisted)")
		return eventstore.NewInMemory[expense.Event]()
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	es := postgres.NewEventStore[expense.Event](pool, postgres.ExpenseCodec{})
	if err := es.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Print("using PostgreSQL event store")
	return es
}

// rebuildReadModel replays the durable event log into the projection so the read
// model is warm on startup. The in-memory store has no history to replay.
func rebuildReadModel(ctx context.Context, es store, projector *projection.ExpenseProjector) {
	replayer, ok := es.(interface {
		LoadAll(ctx context.Context) ([]expense.Event, error)
	})
	if !ok {
		return
	}
	events, err := replayer.LoadAll(ctx)
	if err != nil {
		log.Fatalf("rebuild read model: %v", err)
	}
	projector.Project(events...)
	log.Printf("read model rebuilt from %d events", len(events))
}
