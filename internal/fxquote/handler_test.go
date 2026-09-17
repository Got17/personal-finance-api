package fxquote_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/fxquote"
)

func TestFXQuoteHTTP_GetRate_Success(t *testing.T) {
	uc := fxquote.NewFXQuoteUsecase(&mockFXQuoteRepo{}, nil)
	handler := fxquote.NewFXQuoteHandler(uc)
	app := fiber.New()
	handler.RegisterRoutes(app.Group("/v1"))

	req := httptest.NewRequest("GET", "/v1/fx-quotes?from=USD&to=EUR&date=2026-09-10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			FromCurrency string  `json:"from_currency"`
			ToCurrency   string  `json:"to_currency"`
			Rate         float64 `json:"rate"`
			Date         string  `json:"date"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !res.Success || res.Data.Rate != 0.92 {
		t.Errorf("rate = %v, want 0.92", res.Data.Rate)
	}
}

func TestFXQuoteHTTP_GetRate_InvalidCurrency(t *testing.T) {
	uc := fxquote.NewFXQuoteUsecase(&mockFXQuoteRepo{}, nil)
	handler := fxquote.NewFXQuoteHandler(uc)
	app := fiber.New()
	handler.RegisterRoutes(app.Group("/v1"))

	req := httptest.NewRequest("GET", "/v1/fx-quotes?from=INVALID&to=EUR", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusUnprocessableEntity)
	}
}
