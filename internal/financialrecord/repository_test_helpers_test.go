package financialrecord_test

import (
	"context"

	"github.com/Got17/personal-finance-api/internal/financialrecord"
)

func (r *records) Update(_ context.Context, record *financialrecord.FinancialRecord) error {
	if r.items == nil {
		r.items = make(map[string]*financialrecord.FinancialRecord)
	}
	r.items[record.ID] = record
	return nil
}

func (r *filteredRecords) Update(_ context.Context, record *financialrecord.FinancialRecord) error {
	for index, existing := range r.items {
		if existing.ID == record.ID {
			r.items[index] = record
			return nil
		}
	}
	r.items = append(r.items, record)
	return nil
}
