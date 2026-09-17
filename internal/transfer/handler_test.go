package transfer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/fxquote"
	"github.com/Got17/personal-finance-api/internal/transfer"
)

type transferHandlerFixture struct {
	app   *fiber.App
	token contract.Token
	repo  *mockTransferRepo
}

func setupTransferHandlerFixture() *transferHandlerFixture {
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := newMockTransferRepo()
	acctReader := &mockAccountReader{
		accounts: map[string]*account.Account{
			"acct-usd-1":  {ID: "acct-usd-1", UserID: "user-1", Currency: "USD", IsActive: true},
			"acct-usd-2":  {ID: "acct-usd-2", UserID: "user-1", Currency: "USD", IsActive: true},
			"acct-eur-1":  {ID: "acct-eur-1", UserID: "user-1", Currency: "EUR", IsActive: true},
			"acct-user-2": {ID: "acct-user-2", UserID: "user-2", Currency: "USD", IsActive: true},
		},
	}
	catReader := &mockCategoryReader{
		categories: map[string]*category.Category{
			"cat-expense": {ID: "cat-expense", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true},
		},
	}
	rateProvider := fxquote.NewReferenceRateProvider()
	uc := transfer.NewTransferUsecase(repo, acctReader, catReader, rateProvider)
	handler := transfer.NewTransferHandler(uc)

	app := fiber.New()
	v1 := app.Group("/v1", middleware.JWT(token))
	handler.RegisterRoutes(v1)

	return &transferHandlerFixture{
		app:   app,
		token: token,
		repo:  repo,
	}
}

func (f *transferHandlerFixture) issueToken(userID string) string {
	t, _ := f.token.Sign(contract.Claims{"sub": userID}, time.Hour)
	return t
}

func TestTransferHTTP_CreateSameCurrency_Success(t *testing.T) {
	f := setupTransferHandlerFixture()
	authToken := f.issueToken("user-1")

	body := `{"source_account_id":"acct-usd-1","destination_account_id":"acct-usd-2","source_amount_minor":5000,"date":"2026-09-10T12:00:00Z","note":"Rent split"}`
	req := httptest.NewRequest("POST", "/v1/transfers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusCreated)
	}

	var res struct {
		Success bool               `json:"success"`
		Data    *transfer.Transfer `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !res.Success || res.Data == nil {
		t.Fatal("expected success with data")
	}
	if res.Data.SourceAmountMinor != 5000 || res.Data.DestinationAmountMinor != 5000 {
		t.Errorf("amounts = (%d, %d), want (5000, 5000)", res.Data.SourceAmountMinor, res.Data.DestinationAmountMinor)
	}
}

func TestTransferHTTP_CreateCrossCurrency_WithFee_Success(t *testing.T) {
	f := setupTransferHandlerFixture()
	authToken := f.issueToken("user-1")

	body := `{
		"source_account_id":"acct-usd-1",
		"destination_account_id":"acct-eur-1",
		"source_amount_minor":1000,
		"destination_amount_minor":920,
		"date":"2026-09-10T12:00:00Z",
		"fee": {
			"account_id": "acct-usd-1",
			"category_id": "cat-expense",
			"amount_minor": 150,
			"note": "FX fee"
		}
	}`
	req := httptest.NewRequest("POST", "/v1/transfers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusCreated)
	}

	var res struct {
		Success bool               `json:"success"`
		Data    *transfer.Transfer `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if res.Data.HistoricalFXQuote == nil {
		t.Fatal("expected HistoricalFXQuote in response")
	}
	if res.Data.HistoricalFXQuote.Rate != 0.92 {
		t.Errorf("rate = %v, want 0.92", res.Data.HistoricalFXQuote.Rate)
	}
	if res.Data.TransferFee == nil {
		t.Fatal("expected TransferFee in response")
	}
	if res.Data.TransferFee.AmountMinor != 150 {
		t.Errorf("fee amount = %d, want 150", res.Data.TransferFee.AmountMinor)
	}
}

func TestTransferHTTP_GetTransfer_SuccessAndForbidden(t *testing.T) {
	f := setupTransferHandlerFixture()
	tokenUser1 := f.issueToken("user-1")
	tokenUser2 := f.issueToken("user-2")

	// Pre-populate a transfer for user-1
	t1 := &transfer.Transfer{
		ID:                     "transfer-100",
		UserID:                 "user-1",
		Kind:                   "transfer",
		SourceAccountID:        "acct-usd-1",
		DestinationAccountID:   "acct-usd-2",
		SourceAmountMinor:      2000,
		DestinationAmountMinor: 2000,
		SourceCurrency:         "USD",
		DestinationCurrency:    "USD",
		Date:                   time.Now().UTC(),
		IsActive:               true,
	}
	_ = f.repo.CreateTransferWithLegsAndFee(context.Background(), t1, nil, nil)

	// User-1 can retrieve it
	req1 := httptest.NewRequest("GET", "/v1/transfers/transfer-100", nil)
	req1.Header.Set("Authorization", "Bearer "+tokenUser1)
	resp1, err := f.app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("status = %d, want %d", resp1.StatusCode, fiber.StatusOK)
	}

	// User-2 is forbidden
	req2 := httptest.NewRequest("GET", "/v1/transfers/transfer-100", nil)
	req2.Header.Set("Authorization", "Bearer "+tokenUser2)
	resp2, err := f.app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d", resp2.StatusCode, fiber.StatusForbidden)
	}
}

func TestTransferHTTP_Unauthorized(t *testing.T) {
	f := setupTransferHandlerFixture()

	req := httptest.NewRequest("GET", "/v1/transfers/transfer-100", nil)
	resp, err := f.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
	}
}
