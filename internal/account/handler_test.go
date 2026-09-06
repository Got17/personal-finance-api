package account_test

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
	"github.com/Got17/personal-finance-api/internal/httputil"
)

type contractToken interface {
	Sign(claims contract.Claims, exp time.Duration) (string, error)
}

func setupTestApp() (*fiber.App, contractToken, *mockAccountRepo) {
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	repo := newMockAccountRepo()
	uc := account.NewAccountUsecase(repo)
	handler := account.NewAccountHandler(uc)

	app := fiber.New()
	api := app.Group("/v1", middleware.JWT(tokenAdapter))
	handler.RegisterRoutes(api)

	return app, tokenAdapter, repo
}

func TestCreateAccount_HTTP_201Created(t *testing.T) {
	app, tokenAdapter, _ := setupTestApp()

	token, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-123"}, time.Hour)

	payload := map[string]any{
		"name":        "My Wallet",
		"type":        "cash",
		"currency":    "USD",
		"description": "Pocket cash",
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/accounts", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("status = %d, want 201 Created", resp.StatusCode)
	}

	var body struct {
		Success bool            `json:"success"`
		Data    account.Account `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !body.Success || body.Data.Name != "My Wallet" || body.Data.UserID != "user-123" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestCreateAccount_HTTP_401Unauthorized(t *testing.T) {
	app, _, _ := setupTestApp()

	payload := map[string]any{
		"name":     "My Wallet",
		"type":     "cash",
		"currency": "USD",
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/accounts", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401 Unauthorized", resp.StatusCode)
	}
}

func TestCreateAccount_HTTP_422UnprocessableEntity(t *testing.T) {
	app, tokenAdapter, _ := setupTestApp()
	token, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-123"}, time.Hour)

	payload := map[string]any{
		"name":     "Bad Currency",
		"type":     "savings",
		"currency": "INVALID",
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/accounts", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 422 {
		t.Fatalf("status = %d, want 422 Unprocessable Entity", resp.StatusCode)
	}

	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Success || body.Error != httputil.ErrCodeBadRequest && body.Error != "UNPROCESSABLE_ENTITY" {
		t.Logf("response body = %#v", body)
	}
}

func TestListAccounts_HTTP_200OK_And_CrossUserIsolation(t *testing.T) {
	app, tokenAdapter, repo := setupTestApp()

	_ = repo.Create(context.Background(), &account.Account{
		ID: "acc-1", UserID: "user-1", Name: "User 1 Checking", Type: "checking", Currency: "USD", IsActive: true,
	})
	_ = repo.Create(context.Background(), &account.Account{
		ID: "acc-2", UserID: "user-2", Name: "User 2 Savings", Type: "savings", Currency: "EUR", IsActive: true,
	})

	tokenUser1, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	req := httptest.NewRequest("GET", "/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+tokenUser1)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 OK", resp.StatusCode)
	}

	var body struct {
		Success bool              `json:"success"`
		Data    []account.Account `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !body.Success {
		t.Fatalf("expected success true, got %#v", body)
	}

	if len(body.Data) != 1 {
		t.Fatalf("got %d accounts for user-1, want 1", len(body.Data))
	}
	if body.Data[0].ID != "acc-1" {
		t.Fatalf("got account ID %q, want acc-1", body.Data[0].ID)
	}
}
