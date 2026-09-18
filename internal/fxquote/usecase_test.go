package fxquote_test

import (
	"context"
	"testing"
	"time"

	"github.com/Got17/personal-finance-api/internal/fxquote"
)

type mockFXQuoteRepo struct {
	quotes map[string]*fxquote.HistoricalFXQuote
}

func (m *mockFXQuoteRepo) Create(_ context.Context, quote *fxquote.HistoricalFXQuote) error {
	if m.quotes == nil {
		m.quotes = make(map[string]*fxquote.HistoricalFXQuote)
	}
	m.quotes[quote.ID] = quote
	return nil
}

func (m *mockFXQuoteRepo) FindByID(_ context.Context, id string) (*fxquote.HistoricalFXQuote, error) {
	q, ok := m.quotes[id]
	if !ok {
		return nil, fxquote.ErrFXQuoteNotFound
	}
	return q, nil
}

func (m *mockFXQuoteRepo) FindByRecordID(_ context.Context, recordID string) (*fxquote.HistoricalFXQuote, error) {
	for _, q := range m.quotes {
		if q.RecordID == recordID {
			return q, nil
		}
	}
	return nil, fxquote.ErrFXQuoteNotFound
}

func TestFXQuoteUsecase_GetRate_SameCurrency(t *testing.T) {
	uc := fxquote.NewFXQuoteUsecase(&mockFXQuoteRepo{}, nil)
	rate, err := uc.GetRate(context.Background(), "USD", "USD", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 1.0 {
		t.Errorf("rate = %v, want 1.0", rate)
	}
}

func TestFXQuoteUsecase_GetRate_ValidPair(t *testing.T) {
	uc := fxquote.NewFXQuoteUsecase(&mockFXQuoteRepo{}, nil)
	rate, err := uc.GetRate(context.Background(), "USD", "EUR", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 0.92 {
		t.Errorf("rate = %v, want 0.92", rate)
	}
}

func TestFXQuoteUsecase_GetRate_InvalidCurrency(t *testing.T) {
	uc := fxquote.NewFXQuoteUsecase(&mockFXQuoteRepo{}, nil)
	_, err := uc.GetRate(context.Background(), "INVALID", "EUR", time.Now())
	if err == nil {
		t.Fatal("expected error for invalid currency, got nil")
	}
}

func TestFXQuoteUsecase_GetQuote(t *testing.T) {
	repo := &mockFXQuoteRepo{quotes: make(map[string]*fxquote.HistoricalFXQuote)}
	uc := fxquote.NewFXQuoteUsecase(repo, nil)

	q := &fxquote.HistoricalFXQuote{
		ID:            "quote-1",
		FromCurrency:  "USD",
		ToCurrency:    "EUR",
		Rate:          0.92,
		EffectiveDate: time.Now().UTC(),
		Provenance:    fxquote.ProvenanceProvider,
		RecordID:      "rec-1",
	}
	_ = repo.Create(context.Background(), q)

	got, err := uc.GetQuote(context.Background(), "quote-1")
	if err != nil {
		t.Fatalf("GetQuote() unexpected error: %v", err)
	}
	if got.Rate != 0.92 {
		t.Errorf("Rate = %v, want 0.92", got.Rate)
	}

	_, err = uc.GetQuote(context.Background(), "non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent quote, got nil")
	}
}
