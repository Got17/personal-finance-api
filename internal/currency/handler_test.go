package currency_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/currency"
)

func newTestApp() *fiber.App {
	app := fiber.New()
	currency.NewCurrencyHandler().RegisterRoutes(app.Group("/v1"))
	return app
}

func TestListCurrencies_Status(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/v1/currencies", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestListCurrencies_Payload(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/v1/currencies", nil)
	resp, _ := app.Test(req)

	var body struct {
		Success bool `json:"success"`
		Data    []struct {
			Code          string `json:"code"`
			Name          string `json:"name"`
			Symbol        string `json:"symbol"`
			DecimalDigits int    `json:"decimal_digits"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if !body.Success {
		t.Error("success = false, want true")
	}

	wantOrder := []string{"LAK", "THB", "USD", "CNY", "EUR"}
	if len(body.Data) != len(wantOrder) {
		t.Fatalf("len(data) = %d, want %d", len(body.Data), len(wantOrder))
	}

	for i, code := range wantOrder {
		if body.Data[i].Code != code {
			t.Errorf("data[%d].code = %q, want %q", i, body.Data[i].Code, code)
		}
	}

	// LAK must have 0 decimal digits.
	if body.Data[0].DecimalDigits != 0 {
		t.Errorf("LAK decimal_digits = %d, want 0", body.Data[0].DecimalDigits)
	}
}

func TestListCurrencies_NoAuthRequired(t *testing.T) {
	// The endpoint must be reachable without any Authorization header.
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/v1/currencies", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unauthenticated request: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
