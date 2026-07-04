// Command api runs the expense tracker's HTTP server. It wires an in-memory
// event store to the command handler and a projection that feeds the query
// read model, then serves the REST API.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	httpapi "github.com/alexandre-bcruz/expense-tracker/internal/api/http"
	"github.com/alexandre-bcruz/expense-tracker/internal/command"
	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/eventstore"
	"github.com/alexandre-bcruz/expense-tracker/internal/projection"
	"github.com/alexandre-bcruz/expense-tracker/internal/query"
)

func main() {
	store := eventstore.NewInMemory[expense.Event]()
	readModel := query.NewExpenseStore()
	projector := projection.NewExpenseProjector(readModel)
	store.Subscribe(func(events []expense.Event) { projector.Project(events...) })

	commands := command.NewHandler(store, time.Now)
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
