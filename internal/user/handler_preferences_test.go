package user_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/user"
)

func TestGetBaseCurrency_AuthenticatedCallerRetrievesOwnBaseCurrency(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{
			ID:           "user-789",
			Email:        "currency@example.com",
			BaseCurrency: "USD",
		},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-789"}, time.Hour)

	// 1. GET /v1/users/me includes base_currency
	reqMe := httptest.NewRequest("GET", "/v1/users/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+authToken)
	respMe, err := app.Test(reqMe, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respMe.Body.Close()

	if respMe.StatusCode != 200 {
		t.Fatalf("GET /v1/users/me status = %d, want 200 OK", respMe.StatusCode)
	}

	var bodyMe struct {
		Success bool `json:"success"`
		Data    struct {
			ID           string `json:"id"`
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respMe.Body).Decode(&bodyMe); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyMe.Data.BaseCurrency != "USD" {
		t.Fatalf("base_currency = %q, want USD", bodyMe.Data.BaseCurrency)
	}

	// 2. GET /v1/users/me/preferences returns preferences object
	reqPref := httptest.NewRequest("GET", "/v1/users/me/preferences", nil)
	reqPref.Header.Set("Authorization", "Bearer "+authToken)
	respPref, err := app.Test(reqPref, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respPref.Body.Close()

	if respPref.StatusCode != 200 {
		t.Fatalf("GET /v1/users/me/preferences status = %d, want 200 OK", respPref.StatusCode)
	}

	var bodyPref struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respPref.Body).Decode(&bodyPref); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyPref.Data.BaseCurrency != "USD" {
		t.Fatalf("preferences base_currency = %q, want USD", bodyPref.Data.BaseCurrency)
	}
}

func TestUpdateBaseCurrency_AuthenticatedCallerUpdatesOwnBaseCurrency(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{
			ID:           "user-789",
			Email:        "currency@example.com",
			BaseCurrency: "USD",
		},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-789"}, time.Hour)

	// 1. PUT /v1/users/me/preferences with {"base_currency": "EUR"}
	reqPut := httptest.NewRequest("PUT", "/v1/users/me/preferences", bytes.NewBufferString(`{"base_currency":"EUR"}`))
	reqPut.Header.Set("Authorization", "Bearer "+authToken)
	reqPut.Header.Set("Content-Type", "application/json")
	respPut, err := app.Test(reqPut, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respPut.Body.Close()

	if respPut.StatusCode != 200 {
		t.Fatalf("PUT /v1/users/me/preferences status = %d, want 200 OK", respPut.StatusCode)
	}

	var bodyPut struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respPut.Body).Decode(&bodyPut); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyPut.Data.BaseCurrency != "EUR" {
		t.Fatalf("updated base_currency = %q, want EUR", bodyPut.Data.BaseCurrency)
	}

	// 2. PATCH /v1/users/me/preferences with lowercase currency {"base_currency": "gbp"}
	reqPatch := httptest.NewRequest("PATCH", "/v1/users/me/preferences", bytes.NewBufferString(`{"base_currency":"gbp"}`))
	reqPatch.Header.Set("Authorization", "Bearer "+authToken)
	reqPatch.Header.Set("Content-Type", "application/json")
	respPatch, err := app.Test(reqPatch, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respPatch.Body.Close()

	if respPatch.StatusCode != 200 {
		t.Fatalf("PATCH /v1/users/me/preferences status = %d, want 200 OK", respPatch.StatusCode)
	}

	var bodyPatch struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respPatch.Body).Decode(&bodyPatch); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyPatch.Data.BaseCurrency != "GBP" {
		t.Fatalf("normalized base_currency = %q, want GBP", bodyPatch.Data.BaseCurrency)
	}

	// 3. Verify GET /v1/users/me now reflects GBP
	reqMe := httptest.NewRequest("GET", "/v1/users/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+authToken)
	respMe, err := app.Test(reqMe, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respMe.Body.Close()

	var bodyMe struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respMe.Body).Decode(&bodyMe); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyMe.Data.BaseCurrency != "GBP" {
		t.Fatalf("GET /v1/users/me base_currency = %q, want GBP", bodyMe.Data.BaseCurrency)
	}
}

func TestUpdateBaseCurrency_InvalidCurrenciesRejected(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{
			ID:           "user-789",
			Email:        "currency@example.com",
			BaseCurrency: "USD",
		},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-789"}, time.Hour)

	invalidInputs := []string{
		`{"base_currency":"INVALID"}`,
		`{"base_currency":"US"}`,
		`{"base_currency":"123"}`,
		`{"base_currency":""}`,
		`{"base_currency":"TOOLONG"}`,
		`{}`,
	}

	for _, input := range invalidInputs {
		req := httptest.NewRequest("PUT", "/v1/users/me/preferences", bytes.NewBufferString(input))
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 422 && resp.StatusCode != 400 {
			t.Fatalf("input %s: status = %d, want 422 or 400 validation error", input, resp.StatusCode)
		}
	}
}

func TestUpdateBaseCurrency_UserCanOnlyUpdateOwnPreference(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})

	userA := &user.User{ID: "user-a", Email: "usera@example.com", BaseCurrency: "USD"}
	userB := &user.User{ID: "user-b", Email: "userb@example.com", BaseCurrency: "USD"}

	repo := &mockUserRepo{user: userA}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authTokenA, _ := token.Sign(map[string]any{"sub": "user-a"}, time.Hour)

	// User A updates preference via JWT A, including attempt to inject user_id in payload/query
	req := httptest.NewRequest("PUT", "/v1/users/me/preferences?user_id=user-b", bytes.NewBufferString(`{"base_currency":"EUR","user_id":"user-b"}`))
	req.Header.Set("Authorization", "Bearer "+authTokenA)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 OK", resp.StatusCode)
	}

	if userA.BaseCurrency != "EUR" {
		t.Fatalf("User A base_currency = %q, want EUR", userA.BaseCurrency)
	}
	if userB.BaseCurrency != "USD" {
		t.Fatalf("User B base_currency = %q, want USD (untouched)", userB.BaseCurrency)
	}
}
