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

func TestUpdateFinancialRecord_RejectsInvalidReplacementReferences(t *testing.T) {
	repo := &editableRecords{items: map[string]*financialrecord.FinancialRecord{
		"record-1": {ID: "record-1", UserID: "user-1", Kind: financialrecord.KindExpense, AccountID: "account-1", CategoryID: "expense-category", AmountMinor: 100, Currency: "USD", Date: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), IsActive: true},
	}}
	uc := financialrecord.NewFinancialRecordUsecase(repo,
		accounts{
			"account-1":        {ID: "account-1", UserID: "user-1", Currency: "USD", IsActive: true},
			"foreign-account":  {ID: "foreign-account", UserID: "user-2", Currency: "USD", IsActive: true},
			"inactive-account": {ID: "inactive-account", UserID: "user-1", Currency: "USD", IsActive: false},
		},
		categories{
			"expense-category": {ID: "expense-category", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true},
			"income-category":  {ID: "income-category", UserID: "user-1", Type: category.CategoryTypeIncome, IsActive: true},
		},
	)

	foreignAccount, inactiveAccount, incomeCategory, wrongCurrency := "foreign-account", "inactive-account", "income-category", "EUR"
	for _, input := range []*financialrecord.UpdateFinancialRecordInput{
		{AccountID: &foreignAccount},
		{AccountID: &inactiveAccount},
		{CategoryID: &incomeCategory},
		{Currency: &wrongCurrency},
	} {
		_, err := uc.UpdateFinancialRecord(context.Background(), "user-1", "record-1", input)
		if err == nil {
			t.Fatal("expected replacement-reference validation error")
		}
		appError, ok := errs.IsAppError(err)
		if !ok || (appError.Status != 403 && appError.Status != 422) {
			t.Fatalf("error = %#v", err)
		}
	}
}

var _ account.AccountRepository = accounts{}
