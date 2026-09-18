package financialrecord_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/validator"

	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"
	"github.com/Got17/personal-finance-api/internal/messages"
)

func setupValidationUsecase() (financialrecord.FinancialRecordUsecase, *editableRecords) {
	repo := &editableRecords{items: map[string]*financialrecord.FinancialRecord{
		"rec-1": {
			ID:          "rec-1",
			UserID:      "user-1",
			Kind:        financialrecord.KindExpense,
			AccountID:   "acct-1",
			CategoryID:  strPtr("cat-expense"),
			AmountMinor: 5000,
			Currency:    "USD",
			Date:        time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
			IsActive:    true,
		},
	}}

	uc := financialrecord.NewFinancialRecordUsecase(repo,
		accounts{
			"acct-1": {ID: "acct-1", UserID: "user-1", Currency: "USD", IsActive: true},
		},
		categories{
			"cat-expense":  {ID: "cat-expense", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true},
			"cat-income":   {ID: "cat-income", UserID: "user-1", Type: category.CategoryTypeIncome, IsActive: true},
			"cat-inactive": {ID: "cat-inactive", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: false},
		},
	)
	return uc, repo
}

func TestValidation_CategoryRules(t *testing.T) {
	uc, _ := setupValidationUsecase()
	ctx := context.Background()

	t.Run("create with inactive category rejected", func(t *testing.T) {
		_, err := uc.CreateFinancialRecord(ctx, "user-1", &financialrecord.CreateFinancialRecordInput{
			Kind:        "expense",
			AccountID:   "acct-1",
			CategoryID:  "cat-inactive",
			AmountMinor: 1000,
			Currency:    "USD",
			Date:        time.Now().UTC(),
		})
		assertFieldError(t, err, "category_id", messages.MsgCategoryMustBeActive)
	})

	t.Run("create with kind mismatch category rejected", func(t *testing.T) {
		_, err := uc.CreateFinancialRecord(ctx, "user-1", &financialrecord.CreateFinancialRecordInput{
			Kind:        "expense",
			AccountID:   "acct-1",
			CategoryID:  "cat-income",
			AmountMinor: 1000,
			Currency:    "USD",
			Date:        time.Now().UTC(),
		})
		assertFieldError(t, err, "category_id", messages.MsgCategoryKindMismatch)
	})

	t.Run("create with empty category rejected", func(t *testing.T) {
		_, err := uc.CreateFinancialRecord(ctx, "user-1", &financialrecord.CreateFinancialRecordInput{
			Kind:        "expense",
			AccountID:   "acct-1",
			CategoryID:  "   ",
			AmountMinor: 1000,
			Currency:    "USD",
			Date:        time.Now().UTC(),
		})
		if err == nil {
			t.Fatal("expected error for empty category")
		}
		appErr, ok := errs.IsAppError(err)
		if !ok || appErr.Status != 422 {
			t.Fatalf("expected 422, got %#v", err)
		}
	})

	t.Run("update with kind mismatch category rejected", func(t *testing.T) {
		catIncome := "cat-income"
		_, err := uc.UpdateFinancialRecord(ctx, "user-1", "rec-1", &financialrecord.UpdateFinancialRecordInput{
			CategoryID: &catIncome,
		})
		assertFieldError(t, err, "category_id", messages.MsgCategoryKindMismatch)
	})

	t.Run("update with empty category rejected", func(t *testing.T) {
		emptyCat := "   "
		_, err := uc.UpdateFinancialRecord(ctx, "user-1", "rec-1", &financialrecord.UpdateFinancialRecordInput{
			CategoryID: &emptyCat,
		})
		if err == nil {
			t.Fatal("expected error for empty category update")
		}
		appErr, ok := errs.IsAppError(err)
		if !ok || appErr.Status != 422 {
			t.Fatalf("expected 422, got %#v", err)
		}
	})
}

func TestValidation_AmountRules(t *testing.T) {
	uc, _ := setupValidationUsecase()
	ctx := context.Background()

	t.Run("create with zero amount rejected", func(t *testing.T) {
		_, err := uc.CreateFinancialRecord(ctx, "user-1", &financialrecord.CreateFinancialRecordInput{
			Kind:        "expense",
			AccountID:   "acct-1",
			CategoryID:  "cat-expense",
			AmountMinor: 0,
			Currency:    "USD",
			Date:        time.Now().UTC(),
		})
		assertFieldError(t, err, "amount_minor", messages.MsgAmountMustBePositive)
	})

	t.Run("create with negative amount rejected", func(t *testing.T) {
		_, err := uc.CreateFinancialRecord(ctx, "user-1", &financialrecord.CreateFinancialRecordInput{
			Kind:        "expense",
			AccountID:   "acct-1",
			CategoryID:  "cat-expense",
			AmountMinor: -500,
			Currency:    "USD",
			Date:        time.Now().UTC(),
		})
		assertFieldError(t, err, "amount_minor", messages.MsgAmountMustBePositive)
	})

	t.Run("update with zero amount rejected", func(t *testing.T) {
		zero := int64(0)
		_, err := uc.UpdateFinancialRecord(ctx, "user-1", "rec-1", &financialrecord.UpdateFinancialRecordInput{
			AmountMinor: &zero,
		})
		assertFieldError(t, err, "amount_minor", messages.MsgAmountMustBePositive)
	})

	t.Run("update with negative amount rejected", func(t *testing.T) {
		neg := int64(-100)
		_, err := uc.UpdateFinancialRecord(ctx, "user-1", "rec-1", &financialrecord.UpdateFinancialRecordInput{
			AmountMinor: &neg,
		})
		assertFieldError(t, err, "amount_minor", messages.MsgAmountMustBePositive)
	})
}

func TestValidation_DateRules(t *testing.T) {
	uc, _ := setupValidationUsecase()
	ctx := context.Background()

	t.Run("create with zero date rejected", func(t *testing.T) {
		_, err := uc.CreateFinancialRecord(ctx, "user-1", &financialrecord.CreateFinancialRecordInput{
			Kind:        "expense",
			AccountID:   "acct-1",
			CategoryID:  "cat-expense",
			AmountMinor: 1000,
			Currency:    "USD",
			Date:        time.Time{},
		})
		assertFieldError(t, err, "date", messages.MsgDateIsRequired)
	})

	t.Run("update with zero date rejected", func(t *testing.T) {
		zeroDate := time.Time{}
		_, err := uc.UpdateFinancialRecord(ctx, "user-1", "rec-1", &financialrecord.UpdateFinancialRecordInput{
			Date: &zeroDate,
		})
		assertFieldError(t, err, "date", messages.MsgDateIsRequired)
	})

	t.Run("create with future date allowed", func(t *testing.T) {
		futureDate := time.Now().UTC().Add(48 * time.Hour)
		record, err := uc.CreateFinancialRecord(ctx, "user-1", &financialrecord.CreateFinancialRecordInput{
			Kind:        "expense",
			AccountID:   "acct-1",
			CategoryID:  "cat-expense",
			AmountMinor: 1000,
			Currency:    "USD",
			Date:        futureDate,
		})
		if err != nil {
			t.Fatalf("expected future date to be allowed, got: %v", err)
		}
		if !record.Date.Equal(futureDate) {
			t.Fatalf("expected date %v, got %v", futureDate, record.Date)
		}
	})

	t.Run("update with future date allowed", func(t *testing.T) {
		futureDate := time.Now().UTC().Add(72 * time.Hour)
		record, err := uc.UpdateFinancialRecord(ctx, "user-1", "rec-1", &financialrecord.UpdateFinancialRecordInput{
			Date: &futureDate,
		})
		if err != nil {
			t.Fatalf("expected future date update to be allowed, got: %v", err)
		}
		if !record.Date.Equal(futureDate) {
			t.Fatalf("expected date %v, got %v", futureDate, record.Date)
		}
	})
}

func assertFieldError(t *testing.T, err error, field string, expectedMsg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error on field %q, got nil", field)
	}
	appErr, ok := errs.IsAppError(err)
	if !ok {
		t.Fatalf("expected AppError, got %#v", err)
	}
	if appErr.Status != 422 {
		t.Fatalf("expected status 422, got %d", appErr.Status)
	}
	if fieldsMap, ok := appErr.Data.(map[string]string); ok {
		val, found := fieldsMap[field]
		if !found {
			t.Fatalf("expected error field %q in %#v", field, fieldsMap)
		}
		if expectedMsg != "" && val != expectedMsg {
			t.Fatalf("expected error field %q to be %q, got %q", field, expectedMsg, val)
		}
		return
	}
	if validatorErrors, ok := appErr.Data.([]validator.FieldError); ok {
		for _, fe := range validatorErrors {
			if strings.EqualFold(fe.Field, field) || strings.EqualFold(fe.Field, strings.ReplaceAll(field, "_", "")) {
				return
			}
		}
		t.Fatalf("expected validator error for field %q in %#v", field, validatorErrors)
	}
	t.Fatalf("unexpected appErr.Data format: %#v", appErr.Data)
}
