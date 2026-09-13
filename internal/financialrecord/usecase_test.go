package financialrecord_test

import (
	"context"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/errs"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"
)

type records struct {
	items map[string]*financialrecord.FinancialRecord
}

func (r *records) Create(_ context.Context, record *financialrecord.FinancialRecord) error {
	if r.items == nil {
		r.items = make(map[string]*financialrecord.FinancialRecord)
	}
	r.items[record.ID] = record
	return nil
}

func (r *records) FindByID(_ context.Context, id string) (*financialrecord.FinancialRecord, error) {
	record, found := r.items[id]
	if !found {
		return nil, financialrecord.ErrFinancialRecordNotFound
	}
	return record, nil
}

func (r *records) FindByUserID(_ context.Context, userID string, filter financialrecord.ListFilter) ([]*financialrecord.FinancialRecord, error) {
	var result []*financialrecord.FinancialRecord
	for _, record := range r.items {
		if record.UserID == userID && (filter.IncludeArchived || record.IsActive) {
			result = append(result, record)
		}
	}
	return result, nil
}

type accounts map[string]*account.Account

func (a accounts) Create(context.Context, *account.Account) error                   { return nil }
func (a accounts) FindByUserID(context.Context, string) ([]*account.Account, error) { return nil, nil }
func (a accounts) Update(context.Context, *account.Account) error                   { return nil }
func (a accounts) FindByID(_ context.Context, id string) (*account.Account, error) {
	entity, found := a[id]
	if !found {
		return nil, account.ErrAccountNotFound
	}
	return entity, nil
}

type categories map[string]*category.Category

func (c categories) Create(context.Context, *category.Category) error { return nil }
func (c categories) FindByUserID(context.Context, string) ([]*category.Category, error) {
	return nil, nil
}
func (c categories) Update(context.Context, *category.Category) error { return nil }
func (c categories) FindByID(_ context.Context, id string) (*category.Category, error) {
	entity, found := c[id]
	if !found {
		return nil, category.ErrCategoryNotFound
	}
	return entity, nil
}

func TestCreateFinancialRecord_StoresValidIncomeInAccountCurrency(t *testing.T) {
	repo := &records{}
	uc := financialrecord.NewFinancialRecordUsecase(repo,
		accounts{"account-1": {ID: "account-1", UserID: "user-1", Currency: "USD", IsActive: true}},
		categories{"category-1": {ID: "category-1", UserID: "user-1", Type: category.CategoryTypeIncome, IsActive: true}},
	)

	date := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)
	record, err := uc.CreateFinancialRecord(context.Background(), "user-1", &financialrecord.CreateFinancialRecordInput{
		Kind:        "income",
		AccountID:   "account-1",
		CategoryID:  "category-1",
		AmountMinor: 123_45,
		Currency:    "usd",
		Date:        date,
	})
	if err != nil {
		t.Fatalf("CreateFinancialRecord() error = %v", err)
	}
	if record.Kind != financialrecord.KindIncome || record.Currency != "USD" || record.AmountMinor != 123_45 || record.Date != date {
		t.Fatalf("record = %#v", record)
	}
}

func TestCreateFinancialRecord_RejectsInactiveOrMismatchedReferences(t *testing.T) {
	uc := financialrecord.NewFinancialRecordUsecase(&records{},
		accounts{"account-1": {ID: "account-1", UserID: "user-1", Currency: "USD", IsActive: true}},
		categories{"category-1": {ID: "category-1", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true}},
	)

	_, err := uc.CreateFinancialRecord(context.Background(), "user-1", &financialrecord.CreateFinancialRecordInput{
		Kind: "income", AccountID: "account-1", CategoryID: "category-1", AmountMinor: 1, Currency: "USD", Date: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	appError, ok := errs.IsAppError(err)
	if !ok || appError.Status != 422 {
		t.Fatalf("error = %#v, want 422 application error", err)
	}
}
