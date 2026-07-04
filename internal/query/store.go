package query

import (
	"sort"
	"sync"
)

// ExpenseStore is a goroutine-safe in-memory read model. The projection writes
// to it via the mutating methods; query handlers read through Get and List. A
// durable implementation would back this with a database.
type ExpenseStore struct {
	mu    sync.RWMutex
	views map[string]ExpenseView
}

func NewExpenseStore() *ExpenseStore {
	return &ExpenseStore{views: make(map[string]ExpenseView)}
}

// --- write side (driven by the projection) ---

func (s *ExpenseStore) Save(v ExpenseView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.views[v.ID] = v
}

func (s *ExpenseStore) SetAmount(id string, amountMinor int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.views[id]; ok {
		v.AmountMinor = amountMinor
		s.views[id] = v
	}
}

func (s *ExpenseStore) SetCategory(id, category string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.views[id]; ok {
		v.Category = category
		s.views[id] = v
	}
}

func (s *ExpenseStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.views, id)
}

// --- read side (driven by query handlers) ---

func (s *ExpenseStore) Get(id string) (ExpenseView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.views[id]
	return v, ok
}

// List returns all views ordered by ID for a stable, deterministic result.
func (s *ExpenseStore) List() []ExpenseView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ExpenseView, 0, len(s.views))
	for _, v := range s.views {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
