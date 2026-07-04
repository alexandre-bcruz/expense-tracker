// Package httpapi exposes the expense tracker over HTTP, translating REST
// requests into commands and queries.
package httpapi

import (
	"net/http"
	"time"

	"github.com/alexandre-bcruz/expense-tracker/internal/command"
	"github.com/alexandre-bcruz/expense-tracker/internal/query"
)

// Server wires the HTTP transport to the command (write) and query (read)
// handlers.
type Server struct {
	commands *command.Handler
	queries  *query.Handler
	newID    func() string
}

func NewServer(commands *command.Handler, queries *query.Handler) *Server {
	return &Server{commands: commands, queries: queries, newID: newID}
}

// Routes registers every expense endpoint and returns the HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /expenses", s.record)
	mux.HandleFunc("GET /expenses", s.list)
	mux.HandleFunc("GET /expenses/{id}", s.get)
	mux.HandleFunc("PATCH /expenses/{id}/amount", s.correctAmount)
	mux.HandleFunc("PATCH /expenses/{id}/category", s.recategorize)
	mux.HandleFunc("DELETE /expenses/{id}", s.delete)
	return mux
}

type recordRequest struct {
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	IncurredOn  time.Time `json:"incurred_on"`
}

func (s *Server) record(w http.ResponseWriter, r *http.Request) {
	var req recordRequest
	if !decode(w, r, &req) {
		return
	}
	id := s.newID()
	err := s.commands.RecordExpense(r.Context(), command.RecordExpense{
		ExpenseID:   id,
		AmountMinor: req.AmountMinor,
		Currency:    req.Currency,
		Category:    req.Category,
		Description: req.Description,
		IncurredOn:  req.IncurredOn,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/expenses/"+id)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	views, err := s.queries.ListExpenses(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]expenseResponse, len(views))
	for i, v := range views {
		out[i] = toResponse(v)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	v, err := s.queries.GetExpense(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(v))
}

type amountRequest struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func (s *Server) correctAmount(w http.ResponseWriter, r *http.Request) {
	var req amountRequest
	if !decode(w, r, &req) {
		return
	}
	err := s.commands.CorrectExpenseAmount(r.Context(), command.CorrectExpenseAmount{
		ExpenseID:   r.PathValue("id"),
		AmountMinor: req.AmountMinor,
		Currency:    req.Currency,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type categoryRequest struct {
	Category string `json:"category"`
}

func (s *Server) recategorize(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if !decode(w, r, &req) {
		return
	}
	err := s.commands.RecategorizeExpense(r.Context(), command.RecategorizeExpense{
		ExpenseID:   r.PathValue("id"),
		NewCategory: req.Category,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	err := s.commands.DeleteExpense(r.Context(), command.DeleteExpense{
		ExpenseID: r.PathValue("id"),
		Reason:    r.URL.Query().Get("reason"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
