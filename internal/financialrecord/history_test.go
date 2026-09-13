package financialrecord_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/errs"

	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"

)

type filteredRecords struct {
	items []*financialrecord.FinancialRecord
}

func (r *filteredRecords) Create(_ context.Context, record *financialrecord.FinancialRecord) error {
	r.items = append(r.items, record)
	return nil
}

func (r *filteredRecords) FindByID(_ context.Context, id string) (*financialrecord.FinancialRecord, error) {
	for _, record := range r.items {
		if record.ID == id {
			return record, nil
		}
	}
	return nil, financialrecord.ErrFinancialRecordNotFound
}

func (r *filteredRecords) FindByUserID(_ context.Context, userID string, filter financialrecord.ListFilter) ([]*financialrecord.FinancialRecord, error) {
	var result []*financialrecord.FinancialRecord
	for _, record := range r.items {
		if record.UserID != userID || (!filter.IncludeArchived && !record.IsActive) {
			continue
		}
		if filter.Kind != "" && record.Kind != filter.Kind {
			continue
		}
		if filter.AccountID != "" && record.AccountID != filter.AccountID {
			continue
		}
		if filter.CategoryID != "" && record.CategoryID != filter.CategoryID {
			continue
		}
		if filter.StartDate != nil && record.Date.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && record.Date.After(*filter.EndDate) {
			continue
		}
		result = append(result, record)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date.After(result[j].Date) })
	return result, nil
}

func TestListFinancialRecords_FiltersOwnerHistoryAndDefaultsToActive(t *testing.T) {
	date := func(day int) time.Time { return time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC) }
	repo := &filteredRecords{items: []*financialrecord.FinancialRecord{
		{ID: "income", UserID: "user-1", Kind: financialrecord.KindIncome, AccountID: "account-a", CategoryID: "income-category", Date: date(10), IsActive: true},
		{ID: "expense", UserID: "user-1", Kind: financialrecord.KindExpense, AccountID: "account-b", CategoryID: "expense-category", Date: date(11), IsActive: true},
		{ID: "archived", UserID: "user-1", Kind: financialrecord.KindIncome, AccountID: "account-a", CategoryID: "income-category", Date: date(12), IsActive: false},
		{ID: "foreign", UserID: "user-2", Kind: financialrecord.KindIncome, AccountID: "account-a", CategoryID: "income-category", Date: date(13), IsActive: true},
	}}
	uc := financialrecord.NewFinancialRecordUsecase(repo, accounts{}, categories{})
	start, end := date(1), date(11)
	records, err := uc.ListFinancialRecords(context.Background(), "user-1", financialrecord.ListFilter{StartDate: &start, EndDate: &end, Kind: financialrecord.KindIncome, AccountID: "account-a", CategoryID: "income-category"})
	if err != nil {
		t.Fatalf("ListFinancialRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].ID != "income" {
		t.Fatalf("filtered records = %#v", records)
	}

	records, err = uc.ListFinancialRecords(context.Background(), "user-1", financialrecord.ListFilter{IncludeArchived: true})
	if err != nil {
		t.Fatalf("ListFinancialRecords(include archived) error = %v", err)
	}
	if len(records) != 3 || records[0].ID != "archived" || records[2].ID != "income" {
		t.Fatalf("history = %#v, want newest-first owned records", records)
	}
}

func TestListFinancialRecords_EndDateIncludesEntireRequestedDay(t *testing.T) {
	repo := &filteredRecords{items: []*financialrecord.FinancialRecord{
		{ID: "included", UserID: "user-1", Kind: financialrecord.KindIncome, Date: time.Date(2026, 9, 10, 23, 59, 59, 0, time.UTC), IsActive: true},
		{ID: "excluded", UserID: "user-1", Kind: financialrecord.KindIncome, Date: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC), IsActive: true},
	}}
	uc := financialrecord.NewFinancialRecordUsecase(repo, accounts{}, categories{})
	endDate := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	records, err := uc.ListFinancialRecords(context.Background(), "user-1", financialrecord.ListFilter{EndDate: &endDate})
	if err != nil {
		t.Fatalf("ListFinancialRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].ID != "included" {
		t.Fatalf("records = %#v, want only included record", records)
	}
}


func TestCreateFinancialRecord_RejectsForeignAndInvalidReferences(t *testing.T) {
	date := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)
	validCategory := &category.Category{ID: "category-1", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true}
	uc := financialrecord.NewFinancialRecordUsecase(&filteredRecords{}, accounts{"foreign-account": {ID: "foreign-account", UserID: "user-2", Currency: "USD", IsActive: true}, "inactive-account": {ID: "inactive-account", UserID: "user-1", Currency: "USD", IsActive: false}}, categories{"category-1": validCategory})

	for _, input := range []*financialrecord.CreateFinancialRecordInput{
		{Kind: "expense", AccountID: "foreign-account", CategoryID: "category-1", AmountMinor: 1, Currency: "USD", Date: date},
		{Kind: "expense", AccountID: "inactive-account", CategoryID: "category-1", AmountMinor: 1, Currency: "USD", Date: date},
		{Kind: "expense", AccountID: "inactive-account", CategoryID: "category-1", AmountMinor: 0, Currency: "USD", Date: date},
	} {
		_, err := uc.CreateFinancialRecord(context.Background(), "user-1", input)
		if err == nil {
			t.Fatal("expected validation or ownership error")
		}
		appError, ok := errs.IsAppError(err)
		if !ok || (appError.Status != 403 && appError.Status != 422) {
			t.Fatalf("error = %#v", err)
		}
	}
}


