package account_test

import (
	"context"
	"testing"

	"github.com/BounkhongDev/bkgo/errs"

	"github.com/Got17/personal-finance-api/internal/account"
)

type mockAccountRepo struct {
	accounts map[string]*account.Account
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{
		accounts: make(map[string]*account.Account),
	}
}

func (m *mockAccountRepo) Create(_ context.Context, entity *account.Account) error {
	m.accounts[entity.ID] = entity
	return nil
}

func (m *mockAccountRepo) FindByUserID(_ context.Context, userID string) ([]*account.Account, error) {
	var res []*account.Account
	for _, acct := range m.accounts {
		if acct.UserID == userID {
			res = append(res, acct)
		}
	}
	return res, nil
}

func (m *mockAccountRepo) FindByID(_ context.Context, id string) (*account.Account, error) {
	acct, ok := m.accounts[id]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return acct, nil
}

func TestCreateAccount_Success(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	input := &account.CreateAccountInput{
		Name:        "Checking Account",
		Type:        "checking",
		Currency:    "USD",
		Description: "Primary daily checking",
	}

	result, err := uc.CreateAccount(context.Background(), "user-123", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID == "" {
		t.Error("expected non-empty account ID")
	}
	if result.UserID != "user-123" {
		t.Errorf("got UserID %q, want %q", result.UserID, "user-123")
	}
	if result.Name != "Checking Account" {
		t.Errorf("got Name %q, want %q", result.Name, "Checking Account")
	}
	if result.Type != account.AccountTypeChecking {
		t.Errorf("got Type %q, want %q", result.Type, account.AccountTypeChecking)
	}
	if result.Currency != "USD" {
		t.Errorf("got Currency %q, want %q", result.Currency, "USD")
	}
	if result.Description != "Primary daily checking" {
		t.Errorf("got Description %q, want %q", result.Description, "Primary daily checking")
	}
	if !result.IsActive {
		t.Error("expected IsActive to default to true")
	}
}

func TestCreateAccount_CustomActiveState(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	active := false
	input := &account.CreateAccountInput{
		Name:     "Old Account",
		Type:     "savings",
		Currency: "EUR",
		IsActive: &active,
	}

	result, err := uc.CreateAccount(context.Background(), "user-123", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsActive {
		t.Error("expected IsActive to be false")
	}
}

func TestCreateAccount_InvalidType(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	input := &account.CreateAccountInput{
		Name:     "Crypto Account",
		Type:     "crypto_wallet_invalid",
		Currency: "USD",
	}

	_, err := uc.CreateAccount(context.Background(), "user-123", input)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateAccount_InvalidCurrency(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	input := &account.CreateAccountInput{
		Name:     "Foreign Account",
		Type:     "savings",
		Currency: "INVALID",
	}

	_, err := uc.CreateAccount(context.Background(), "user-123", input)
	if err == nil {
		t.Fatal("expected error for invalid currency code, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateAccount_WhitespaceOnlyName(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	input := &account.CreateAccountInput{
		Name:     "   ",
		Type:     "savings",
		Currency: "USD",
	}

	_, err := uc.CreateAccount(context.Background(), "user-123", input)
	if err == nil {
		t.Fatal("expected error for whitespace-only name, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateAccount_EmptyUserID(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	input := &account.CreateAccountInput{
		Name:     "Checking Account",
		Type:     "checking",
		Currency: "USD",
	}

	_, err := uc.CreateAccount(context.Background(), "", input)
	if err == nil {
		t.Fatal("expected error for empty userID, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 401 {
		t.Fatalf("expected 401 AppError, got %#v", err)
	}
}

func TestListAccounts_Isolation(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	_, _ = uc.CreateAccount(context.Background(), "user-1", &account.CreateAccountInput{
		Name: "User 1 Account", Type: "checking", Currency: "USD",
	})
	_, _ = uc.CreateAccount(context.Background(), "user-1", &account.CreateAccountInput{
		Name: "User 1 Savings", Type: "savings", Currency: "USD",
	})
	_, _ = uc.CreateAccount(context.Background(), "user-2", &account.CreateAccountInput{
		Name: "User 2 Account", Type: "credit_card", Currency: "EUR",
	})

	user1Accounts, err := uc.ListAccounts(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user1Accounts) != 2 {
		t.Fatalf("got %d accounts for user-1, want 2", len(user1Accounts))
	}

	user2Accounts, err := uc.ListAccounts(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user2Accounts) != 1 {
		t.Fatalf("got %d accounts for user-2, want 1", len(user2Accounts))
	}

	user3Accounts, err := uc.ListAccounts(context.Background(), "user-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user3Accounts == nil || len(user3Accounts) != 0 {
		t.Fatalf("expected non-nil empty slice for user-3, got %#v", user3Accounts)
	}
}

func (m *mockAccountRepo) Update(_ context.Context, entity *account.Account) error {
	m.accounts[entity.ID] = entity
	return nil
}

func TestUpdateAccount_Success(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	created, err := uc.CreateAccount(context.Background(), "user-123", &account.CreateAccountInput{
		Name:        "Old Name",
		Type:        "checking",
		Currency:    "USD",
		Description: "Old desc",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	newName := "New Name"
	newType := "savings"
	newCurr := "EUR"
	newDesc := "New desc"
	active := false

	updated, err := uc.UpdateAccount(context.Background(), "user-123", created.ID, &account.UpdateAccountInput{
		Name:        &newName,
		Type:        &newType,
		Currency:    &newCurr,
		Description: &newDesc,
		IsActive:    &active,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if updated.Name != newName || updated.Type != account.AccountType(newType) || updated.Currency != newCurr || updated.Description != newDesc || updated.IsActive != false {
		t.Fatalf("unexpected updated entity: %#v", updated)
	}
}

func TestUpdateAccount_NotFound(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	newName := "New Name"
	_, err := uc.UpdateAccount(context.Background(), "user-123", "non-existent-id", &account.UpdateAccountInput{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error for non-existent account, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 404 {
		t.Fatalf("expected 404 AppError, got %#v", err)
	}
}

func TestUpdateAccount_AccessDenied(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	created, err := uc.CreateAccount(context.Background(), "user-1", &account.CreateAccountInput{
		Name: "User 1 Account", Type: "checking", Currency: "USD",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	newName := "Hacked Name"
	_, err = uc.UpdateAccount(context.Background(), "user-2", created.ID, &account.UpdateAccountInput{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error when user-2 tries to update user-1 account, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 403 {
		t.Fatalf("expected 403 AppError, got %#v", err)
	}
}

func TestUpdateAccount_ValidationFailed(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	created, err := uc.CreateAccount(context.Background(), "user-123", &account.CreateAccountInput{
		Name: "Valid Account", Type: "checking", Currency: "USD",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	invalidType := "invalid_type_xyz"
	_, err = uc.UpdateAccount(context.Background(), "user-123", created.ID, &account.UpdateAccountInput{
		Type: &invalidType,
	})
	if err == nil {
		t.Fatal("expected 422 error for invalid type, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}

	invalidCurrency := "BAD"
	_, err = uc.UpdateAccount(context.Background(), "user-123", created.ID, &account.UpdateAccountInput{
		Currency: &invalidCurrency,
	})
	if err == nil {
		t.Fatal("expected 422 error for invalid currency, got nil")
	}

	emptyName := "   "
	_, err = uc.UpdateAccount(context.Background(), "user-123", created.ID, &account.UpdateAccountInput{
		Name: &emptyName,
	})
	if err == nil {
		t.Fatal("expected 422 error for empty name, got nil")
	}
}

func TestDeactivateAccount_Success(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	created, err := uc.CreateAccount(context.Background(), "user-123", &account.CreateAccountInput{
		Name: "Active Account", Type: "checking", Currency: "USD",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	deactivated, err := uc.DeactivateAccount(context.Background(), "user-123", created.ID)
	if err != nil {
		t.Fatalf("deactivate failed: %v", err)
	}

	if deactivated.IsActive != false {
		t.Fatalf("expected IsActive to be false, got true")
	}
}

func TestDeactivateAccount_AccessDenied(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	created, err := uc.CreateAccount(context.Background(), "user-1", &account.CreateAccountInput{
		Name: "User 1 Account", Type: "checking", Currency: "USD",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = uc.DeactivateAccount(context.Background(), "user-2", created.ID)
	if err == nil {
		t.Fatal("expected error for user-2 deactivating user-1 account, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 403 {
		t.Fatalf("expected 403 AppError, got %#v", err)
	}
}

func TestDeactivateAccount_NotFound(t *testing.T) {
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)

	_, err := uc.DeactivateAccount(context.Background(), "user-123", "non-existent-id")
	if err == nil {
		t.Fatal("expected error for non-existent account deactivation, got nil")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 404 {
		t.Fatalf("expected 404 AppError, got %#v", err)
	}
}
