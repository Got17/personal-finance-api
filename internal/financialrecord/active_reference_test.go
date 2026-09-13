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

func TestCreateFinancialRecord_RejectsInactiveCategory(t *testing.T) {
	uc := financialrecord.NewFinancialRecordUsecase(&filteredRecords{}, accounts{
		"account-1": {ID: "account-1", UserID: "user-1", Currency: "USD", IsActive: true},
	}, categories{
		"category-1": {ID: "category-1", UserID: "user-1", Type: category.CategoryTypeIncome, IsActive: false},
	})
	_, err := uc.CreateFinancialRecord(context.Background(), "user-1", &financialrecord.CreateFinancialRecordInput{
		Kind: "income", AccountID: "account-1", CategoryID: "category-1", AmountMinor: 1, Currency: "USD", Date: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected inactive category error")
	}
	appError, ok := errs.IsAppError(err)
	if !ok || appError.Status != 422 {
		t.Fatalf("error = %#v, want 422", err)
	}
}

var _ account.AccountRepository = accounts{}
