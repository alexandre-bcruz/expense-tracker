// Package projection builds read models from domain events. A projector
// consumes the events an aggregate emits and updates the query side's views,
// keeping the read model eventually consistent with the write model.
package projection

import (
	"github.com/alexandre-bcruz/expense-tracker/internal/domain/expense"
	"github.com/alexandre-bcruz/expense-tracker/internal/query"
)

// ExpenseWriteModel is the write-facing view store the projector updates.
type ExpenseWriteModel interface {
	Save(query.ExpenseView)
	SetAmount(id string, amountMinor int64)
	SetCategory(id, category string)
	Delete(id string)
}

// ExpenseProjector translates expense events into read-model updates.
type ExpenseProjector struct {
	model ExpenseWriteModel
}

func NewExpenseProjector(model ExpenseWriteModel) *ExpenseProjector {
	return &ExpenseProjector{model: model}
}

// Project applies events to the read model in order. Feeding the full history
// of a stream rebuilds its view from scratch.
func (p *ExpenseProjector) Project(events ...expense.Event) {
	for _, e := range events {
		switch ev := e.(type) {
		case expense.ExpenseRecorded:
			p.model.Save(query.ExpenseView{
				ID:          ev.ExpenseID,
				AmountMinor: ev.Amount.Amount(),
				Currency:    ev.Amount.Currency(),
				Category:    ev.Category,
				Description: ev.Description,
				IncurredOn:  ev.IncurredOn,
			})
		case expense.ExpenseAmountCorrected:
			p.model.SetAmount(ev.ExpenseID, ev.NewAmount.Amount())
		case expense.ExpenseRecategorized:
			p.model.SetCategory(ev.ExpenseID, ev.NewCategory)
		case expense.ExpenseDeleted:
			p.model.Delete(ev.ExpenseID)
		}
	}
}
