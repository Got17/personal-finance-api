package fxquote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Got17/personal-finance-api/internal/currency"
)

const (
	DefaultOpenExchangeRateBaseURL = "https://open.er-api.com/v6/latest"
	defaultHTTPTimeout             = 5 * time.Second
	defaultCacheTTL                = 1 * time.Hour
)

type openERAPIResponse struct {
	Result             string             `json:"result"`
	BaseCode           string             `json:"base_code"`
	TimeNextUpdateUnix int64              `json:"time_next_update_unix"`
	Rates              map[string]float64 `json:"rates"`
}

type cachedRates struct {
	rates     map[string]float64
	expiresAt time.Time
}

// OpenExchangeRateProvider fetches exchange rates from open.er-api.com,
// with thread-safe in-memory caching and graceful fallback.
type OpenExchangeRateProvider struct {
	client   *http.Client
	baseURL  string
	fallback FXQuoteProvider
	cacheMu  sync.RWMutex
	cache    map[string]cachedRates
}

func NewOpenExchangeRateProvider(client *http.Client, fallback FXQuoteProvider) *OpenExchangeRateProvider {
	return NewOpenExchangeRateProviderWithURL(client, fallback, DefaultOpenExchangeRateBaseURL)
}

func NewOpenExchangeRateProviderWithURL(client *http.Client, fallback FXQuoteProvider, baseURL string) *OpenExchangeRateProvider {
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if fallback == nil {
		fallback = NewReferenceRateProvider()
	}
	return &OpenExchangeRateProvider{
		client:   client,
		baseURL:  baseURL,
		fallback: fallback,
		cache:    make(map[string]cachedRates),
	}
}

func (p *OpenExchangeRateProvider) GetRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (float64, error) {
	from := strings.ToUpper(strings.TrimSpace(fromCurrency))
	to := strings.ToUpper(strings.TrimSpace(toCurrency))

	if !currency.IsValid(from) || !currency.IsValid(to) {
		return 0, ErrCurrencyNotSupported
	}
	if from == to {
		return 1.0, nil
	}

	if rate, ok := p.getCachedRate(from, to); ok {
		return rate, nil
	}

	rates, expiresAt, err := p.fetchRates(ctx, from)
	if err != nil {
		slog.Warn("failed to fetch live exchange rate, using fallback", "from", from, "to", to, "error", err)
		if p.fallback != nil {
			return p.fallback.GetRate(ctx, from, to, date)
		}
		return 0, err
	}

	p.setCachedRates(from, rates, expiresAt)

	rate, ok := rates[to]
	if !ok || rate <= 0 {
		slog.Warn("target currency not found in live response, using fallback", "from", from, "to", to)
		if p.fallback != nil {
			return p.fallback.GetRate(ctx, from, to, date)
		}
		return 0, ErrRateUnavailable
	}

	return rate, nil
}

func (p *OpenExchangeRateProvider) fetchRates(ctx context.Context, baseCurrency string) (map[string]float64, time.Time, error) {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(p.baseURL, "/"), baseCurrency)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, time.Time{}, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp openERAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, time.Time{}, err
	}

	if apiResp.Result != "success" || len(apiResp.Rates) == 0 {
		return nil, time.Time{}, fmt.Errorf("unsuccessful response: result=%q", apiResp.Result)
	}

	expiresAt := time.Now().Add(defaultCacheTTL)
	if apiResp.TimeNextUpdateUnix > 0 {
		nextUpdate := time.Unix(apiResp.TimeNextUpdateUnix, 0)
		if nextUpdate.After(time.Now()) {
			expiresAt = nextUpdate
		}
	}

	return apiResp.Rates, expiresAt, nil
}

func (p *OpenExchangeRateProvider) getCachedRate(from, to string) (float64, bool) {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	entry, ok := p.cache[from]
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, false
	}
	rate, ok := entry.rates[to]
	return rate, ok && rate > 0
}

func (p *OpenExchangeRateProvider) setCachedRates(base string, rates map[string]float64, expiresAt time.Time) {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	p.cache[base] = cachedRates{
		rates:     rates,
		expiresAt: expiresAt,
	}
}
