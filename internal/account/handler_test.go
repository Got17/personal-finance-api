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

	if body.Success {
		t.Fatalf("expected success false, got %#v", body)
	}
	if body.Error != "UNPROCESSABLE" {
		t.Fatalf("expected UNPROCESSABLE error code, got %#v", body)
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

func TestUpdateAccount_HTTP_200OK(t *testing.T) {
	app, tokenAdapter, repo := setupTestApp()

	_ = repo.Create(context.Background(), &account.Account{
		ID: "acc-1", UserID: "user-1", Name: "User 1 Checking", Type: "checking", Currency: "USD", IsActive: true,
	})

	tokenUser1, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	payload := map[string]any{
		"name": "Updated Checking",
		"type": "savings",
	}
	bodyBytes, _ := json.Marshal(payload)

	// Test PATCH
	reqPatch := httptest.NewRequest("PATCH", "/v1/accounts/acc-1", bytes.NewReader(bodyBytes))
	reqPatch.Header.Set("Authorization", "Bearer "+tokenUser1)
	reqPatch.Header.Set("Content-Type", "application/json")

	respPatch, err := app.Test(reqPatch, 5000)
	if err != nil {
		t.Fatalf("patch request failed: %v", err)
	}
	defer respPatch.Body.Close()

	if respPatch.StatusCode != 200 {
		t.Fatalf("patch status = %d, want 200 OK", respPatch.StatusCode)
	}

	var bodyPatch struct {
		Success bool            `json:"success"`
		Data    account.Account `json:"data"`
	}
	if err := json.NewDecoder(respPatch.Body).Decode(&bodyPatch); err != nil {
		t.Fatalf("decode patch response: %v", err)
	}

	if !bodyPatch.Success || bodyPatch.Data.Name != "Updated Checking" || bodyPatch.Data.Type != "savings" {
		t.Fatalf("unexpected patch response data: %#v", bodyPatch)
	}

	// Test PUT
	reqPut := httptest.NewRequest("PUT", "/v1/accounts/acc-1", bytes.NewReader(bodyBytes))
	reqPut.Header.Set("Authorization", "Bearer "+tokenUser1)
	reqPut.Header.Set("Content-Type", "application/json")

	respPut, err := app.Test(reqPut, 5000)
	if err != nil {
		t.Fatalf("put request failed: %v", err)
	}
	defer respPut.Body.Close()

	if respPut.StatusCode != 200 {
		t.Fatalf("put status = %d, want 200 OK", respPut.StatusCode)
	}
}

func TestUpdateAccount_HTTP_403Forbidden(t *testing.T) {
	app, tokenAdapter, repo := setupTestApp()

	_ = repo.Create(context.Background(), &account.Account{
		ID: "acc-1", UserID: "user-1", Name: "User 1 Checking", Type: "checking", Currency: "USD", IsActive: true,
	})

	tokenUser2, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-2"}, time.Hour)

	payload := map[string]any{"name": "Hacked Account"}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/v1/accounts/acc-1", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+tokenUser2)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Fatalf("status = %d, want 403 Forbidden", resp.StatusCode)
	}
}

func TestUpdateAccount_HTTP_404NotFound(t *testing.T) {
	app, tokenAdapter, _ := setupTestApp()
	tokenUser1, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	payload := map[string]any{"name": "New Name"}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("PATCH", "/v1/accounts/non-existent", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+tokenUser1)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404 Not Found", resp.StatusCode)
	}
}

func TestDeactivateAccount_HTTP_200OK(t *testing.T) {
	app, tokenAdapter, repo := setupTestApp()

	_ = repo.Create(context.Background(), &account.Account{
		ID: "acc-1", UserID: "user-1", Name: "User 1 Account", Type: "checking", Currency: "USD", IsActive: true,
	})

	tokenUser1, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	req := httptest.NewRequest("DELETE", "/v1/accounts/acc-1", nil)
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
		Success bool            `json:"success"`
		Data    account.Account `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !body.Success || body.Data.IsActive != false {
		t.Fatalf("expected account to be deactivated, got %#v", body)
	}
}

func TestDeactivateAccount_HTTP_403Forbidden(t *testing.T) {
	app, tokenAdapter, repo := setupTestApp()

	_ = repo.Create(context.Background(), &account.Account{
		ID: "acc-1", UserID: "user-1", Name: "User 1 Account", Type: "checking", Currency: "USD", IsActive: true,
	})

	tokenUser2, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-2"}, time.Hour)

	req := httptest.NewRequest("DELETE", "/v1/accounts/acc-1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenUser2)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Fatalf("status = %d, want 403 Forbidden", resp.StatusCode)
	}
}

func TestDeactivateAccount_HTTP_404NotFound(t *testing.T) {
	app, tokenAdapter, _ := setupTestApp()

	tokenUser1, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	req := httptest.NewRequest("DELETE", "/v1/accounts/non-existent", nil)
	req.Header.Set("Authorization", "Bearer "+tokenUser1)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404 Not Found", resp.StatusCode)
	}
}

