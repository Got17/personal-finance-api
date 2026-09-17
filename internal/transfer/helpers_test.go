package transfer_test

import (
	"context"
	"errors"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"
	"github.com/Got17/personal-finance-api/internal/fxquote"
	"github.com/Got17/personal-finance-api/internal/transfer"
)

type mockAccountReader struct {
	accounts map[string]*account.Account
}

func (m *mockAccountReader) GetAccount(_ context.Context, userID string, accountID string) (*account.Account, error) {
	acct, ok := m.accounts[accountID]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	if acct.UserID != userID {
		return nil, account.ErrAccessDenied
	}
	return acct, nil
}

type mockCategoryReader struct {
	categories map[string]*category.Category
}

func (m *mockCategoryReader) GetCategory(_ context.Context, userID string, categoryID string) (*category.Category, error) {
	cat, ok := m.categories[categoryID]
	if !ok {
		return nil, category.ErrCategoryNotFound
	}
	if cat.UserID != userID {
		return nil, category.ErrAccessDenied
	}
	return cat, nil
}

type mockTransferRepo struct {
	transfers    map[string]*transfer.Transfer
	quotes       map[string]*fxquote.HistoricalFXQuote
	feeRecords   map[string]*financialrecord.FinancialRecord
	failOnCreate bool
}

func newMockTransferRepo() *mockTransferRepo {
	return &mockTransferRepo{
		transfers:  make(map[string]*transfer.Transfer),
		quotes:     make(map[string]*fxquote.HistoricalFXQuote),
		feeRecords: make(map[string]*financialrecord.FinancialRecord),
	}
}

func (m *mockTransferRepo) CreateTransferWithLegsAndFee(_ context.Context, t *transfer.Transfer, q *fxquote.HistoricalFXQuote, fee *financialrecord.FinancialRecord) error {
	if m.failOnCreate {
		return errors.New("database transaction failed")
	}
	m.transfers[t.ID] = t
	if q != nil {
		m.quotes[q.ID] = q
	}
	if fee != nil {
		m.feeRecords[fee.ID] = fee
	}
	return nil
}

func (m *mockTransferRepo) FindByID(_ context.Context, id string) (*transfer.Transfer, error) {
	t, ok := m.transfers[id]
	if !ok {
		return nil, transfer.ErrTransferNotFound
	}
	res := *t
	if t.HistoricalFXQuoteID != nil {
		if q, ok := m.quotes[*t.HistoricalFXQuoteID]; ok {
			res.HistoricalFXQuote = q
		}
	}
	if t.TransferFeeRecordID != nil {
		if f, ok := m.feeRecords[*t.TransferFeeRecordID]; ok {
			res.TransferFee = &transfer.TransferFeeResult{
				ID:          f.ID,
				UserID:      f.UserID,
				Kind:        string(f.Kind),
				AccountID:   f.AccountID,
				CategoryID:  "",
				AmountMinor: f.AmountMinor,
				Currency:    f.Currency,
				Date:        f.Date,
				Note:        f.Note,
				IsActive:    f.IsActive,
				CreatedAt:   f.CreatedAt,
				UpdatedAt:   f.UpdatedAt,
			}
			if f.CategoryID != nil {
				res.TransferFee.CategoryID = *f.CategoryID
			}
		}
	}
	return &res, nil
}

func (m *mockTransferRepo) FindByUserID(_ context.Context, userID string, filter transfer.ListTransferFilter) ([]*transfer.Transfer, error) {
	var list []*transfer.Transfer
	for _, t := range m.transfers {
		if t.UserID == userID {
			if !filter.IncludeArchived && !t.IsActive {
				continue
			}
			list = append(list, t)
		}
	}
	return list, nil
}

func setupTransferTest() (*transfer.TransferUsecase, *mockTransferRepo, *mockAccountReader, *mockCategoryReader) {
	acctReader := &mockAccountReader{
		accounts: map[string]*account.Account{
			"acct-usd-1":      {ID: "acct-usd-1", UserID: "user-1", Currency: "USD", IsActive: true},
			"acct-usd-2":      {ID: "acct-usd-2", UserID: "user-1", Currency: "USD", IsActive: true},
			"acct-eur-1":      {ID: "acct-eur-1", UserID: "user-1", Currency: "EUR", IsActive: true},
			"acct-lak-1":      {ID: "acct-lak-1", UserID: "user-1", Currency: "LAK", IsActive: true},
			"acct-inactive":   {ID: "acct-inactive", UserID: "user-1", Currency: "USD", IsActive: false},
			"acct-other-user": {ID: "acct-other-user", UserID: "user-2", Currency: "USD", IsActive: true},
		},
	}
	catReader := &mockCategoryReader{
		categories: map[string]*category.Category{
			"cat-expense":  {ID: "cat-expense", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true},
			"cat-income":   {ID: "cat-income", UserID: "user-1", Type: category.CategoryTypeIncome, IsActive: true},
			"cat-inactive": {ID: "cat-inactive", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: false},
		},
	}
	repo := newMockTransferRepo()
	rateProvider := fxquote.NewReferenceRateProvider()
	uc := transfer.NewTransferUsecase(repo, acctReader, catReader, rateProvider)
	return &uc, repo, acctReader, catReader
}
