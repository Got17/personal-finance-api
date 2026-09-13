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

type editableRecords struct {
	items map[string]*financialrecord.FinancialRecord
}

func (r *editableRecords) Create(_ context.Context, record *financialrecord.FinancialRecord) error {
	r.items[record.ID] = record
	return nil
}

func (r *editableRecords) FindByID(_ context.Context, id string) (*financialrecord.FinancialRecord, error) {
	record, found := r.items[id]
	if !found {
		return nil, financialrecord.ErrFinancialRecordNotFound
	}
	return record, nil
}

func (r *editableRecords) FindByUserID(_ context.Context, _ string, _ financialrecord.ListFilter) ([]*financialrecord.FinancialRecord, error) {
	return nil, nil
}

func (r *editableRecords) Update(_ context.Context, record *financialrecord.FinancialRecord) error {
	r.items[record.ID] = record
	return nil
}

func TestFinancialRecordLifecycle_UpdatesArchivesAndProtectsOwner(t *testing.T) {
	repo := &editableRecords{items: map[string]*financialrecord.FinancialRecord{
		"record-1": {ID: "record-1", UserID: "user-1", Kind: financialrecord.KindExpense, AccountID: "account-1", CategoryID: "category-1", AmountMinor: 1200, Currency: "USD", Date: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), IsActive: true},
	}}
	uc := financialrecord.NewFinancialRecordUsecase(repo,
		accounts{"account-1": {ID: "account-1", UserID: "user-1", Currency: "USD", IsActive: true}},
		categories{"category-1": {ID: "category-1", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true}},
	)

	amount := int64(2500)
	note := "Corrected amount"
	updated, err := uc.UpdateFinancialRecord(context.Background(), "user-1", "record-1", &financialrecord.UpdateFinancialRecordInput{AmountMinor: &amount, Note: &note})
	if err != nil {
		t.Fatalf("UpdateFinancialRecord() error = %v", err)
	}
	if updated.AmountMinor != amount || updated.Note != note {
		t.Fatalf("updated record = %#v", updated)
	}

	archived, err := uc.ArchiveFinancialRecord(context.Background(), "user-1", "record-1")
	if err != nil {
		t.Fatalf("ArchiveFinancialRecord() error = %v", err)
	}
	if archived.IsActive {
		t.Fatal("expected archived record to be inactive")
	}
	_, err = uc.UpdateFinancialRecord(context.Background(), "user-1", "record-1", &financialrecord.UpdateFinancialRecordInput{Note: &note})
	if err == nil {
		t.Fatal("expected archived record to reject updates")
	}
	appError, ok := errs.IsAppError(err)
	if !ok || appError.Status != 422 {
		t.Fatalf("archived update error = %#v, want 422", err)
	}

	_, err = uc.ArchiveFinancialRecord(context.Background(), "user-2", "record-1")
	if err == nil {
		t.Fatal("expected cross-user archive to be forbidden")
	}
	appError, ok = errs.IsAppError(err)
	if !ok || appError.Status != 403 {
		t.Fatalf("cross-user archive error = %#v, want 403", err)
	}
}

var _ account.AccountRepository = accounts{}
