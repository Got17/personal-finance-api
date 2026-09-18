package fxquote_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Got17/personal-finance-api/internal/fxquote"
)

func TestOpenExchangeRateProvider_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/USD" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result":                "success",
			"base_code":             "USD",
			"time_next_update_unix": time.Now().Add(time.Hour).Unix(),
			"rates": map[string]float64{
				"USD": 1.0,
				"EUR": 0.88,
				"LAK": 22150.0,
				"THB": 34.5,
				"CNY": 6.95,
			},
		})
	}))
	defer server.Close()

	provider := fxquote.NewOpenExchangeRateProviderWithURL(server.Client(), nil, server.URL)

	rate, err := provider.GetRate(context.Background(), "USD", "LAK", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 22150.0 {
		t.Errorf("rate = %f, want 22150.0", rate)
	}
}

func TestOpenExchangeRateProvider_CacheHit(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result":                "success",
			"base_code":             "USD",
			"time_next_update_unix": time.Now().Add(time.Hour).Unix(),
			"rates": map[string]float64{
				"EUR": 0.90,
				"THB": 35.0,
			},
		})
	}))
	defer server.Close()

	provider := fxquote.NewOpenExchangeRateProviderWithURL(server.Client(), nil, server.URL)

	// First call -> triggers network request
	rate1, err := provider.GetRate(context.Background(), "USD", "EUR", time.Now())
	if err != nil {
		t.Fatalf("call 1 unexpected error: %v", err)
	}
	if rate1 != 0.90 {
		t.Errorf("call 1 rate = %f, want 0.90", rate1)
	}

	// Second call -> should hit cache, no extra HTTP request
	rate2, err := provider.GetRate(context.Background(), "USD", "EUR", time.Now())
	if err != nil {
		t.Fatalf("call 2 unexpected error: %v", err)
	}
	if rate2 != 0.90 {
		t.Errorf("call 2 rate = %f, want 0.90", rate2)
	}

	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Errorf("expected 1 HTTP request due to caching, got %d", count)
	}
}

func TestOpenExchangeRateProvider_FallbackOnServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream service unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	fallback := fxquote.NewReferenceRateProvider()
	provider := fxquote.NewOpenExchangeRateProviderWithURL(server.Client(), fallback, server.URL)

	// Should fallback gracefully to ReferenceRateProvider (EUR reference rate = 0.92)
	rate, err := provider.GetRate(context.Background(), "USD", "EUR", time.Now())
	if err != nil {
		t.Fatalf("expected graceful fallback, got error: %v", err)
	}
	if rate != 0.92 {
		t.Errorf("rate = %f, want fallback reference rate 0.92", rate)
	}
}

func TestOpenExchangeRateProvider_SameCurrency(t *testing.T) {
	provider := fxquote.NewOpenExchangeRateProvider(nil, nil)

	rate, err := provider.GetRate(context.Background(), "USD", "USD", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 1.0 {
		t.Errorf("same currency rate = %f, want 1.0", rate)
	}
}

func TestOpenExchangeRateProvider_UnsupportedCurrency(t *testing.T) {
	provider := fxquote.NewOpenExchangeRateProvider(nil, nil)

	_, err := provider.GetRate(context.Background(), "INVALID", "USD", time.Now())
	if err != fxquote.ErrCurrencyNotSupported {
		t.Errorf("expected ErrCurrencyNotSupported, got %v", err)
	}
}
