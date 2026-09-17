package fxquote

import (
	"context"
	"strings"
	"time"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/Got17/personal-finance-api/internal/currency"
	"github.com/Got17/personal-finance-api/internal/messages"
)

type FXQuoteUsecase interface {
	GetRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (float64, error)
	GetQuote(ctx context.Context, id string) (*HistoricalFXQuote, error)
	GetQuoteByRecordID(ctx context.Context, recordID string) (*HistoricalFXQuote, error)
}

type fxQuoteUsecase struct {
	repo     FXQuoteRepository
	provider FXQuoteProvider
}

func NewFXQuoteUsecase(repo FXQuoteRepository, provider FXQuoteProvider) FXQuoteUsecase {
	if provider == nil {
		provider = NewReferenceRateProvider()
	}
	return &fxQuoteUsecase{repo: repo, provider: provider}
}

func (u *fxQuoteUsecase) GetRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (float64, error) {
	from := strings.ToUpper(strings.TrimSpace(fromCurrency))
	to := strings.ToUpper(strings.TrimSpace(toCurrency))

	if !currency.IsValid(from) || !currency.IsValid(to) {
		return 0, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{
			"currency": messages.MsgInvalidCurrencyCode,
		})
	}
	if from == to {
		return 1.0, nil
	}

	rate, err := u.provider.GetRate(ctx, from, to, date)
	if err != nil {
		return 0, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{
			"rate": messages.MsgFXRateUnavailable,
		})
	}
	return rate, nil
}

func (u *fxQuoteUsecase) GetQuote(ctx context.Context, id string) (*HistoricalFXQuote, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errs.NotFound(messages.MsgFXQuoteNotFound)
	}
	quote, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errs.NotFound(messages.MsgFXQuoteNotFound)
	}
	return quote, nil
}

func (u *fxQuoteUsecase) GetQuoteByRecordID(ctx context.Context, recordID string) (*HistoricalFXQuote, error) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return nil, errs.NotFound(messages.MsgFXQuoteNotFound)
	}
	quote, err := u.repo.FindByRecordID(ctx, recordID)
	if err != nil {
		return nil, errs.NotFound(messages.MsgFXQuoteNotFound)
	}
	return quote, nil
}

// ReferenceRateProvider provides deterministic exchange rates for the supported currencies.
type ReferenceRateProvider struct {
	baseRates map[string]float64
}

func NewReferenceRateProvider() *ReferenceRateProvider {
	return &ReferenceRateProvider{
		baseRates: map[string]float64{
			"USD": 1.0,
			"EUR": 0.92,
			"THB": 35.0,
			"CNY": 7.20,
			"LAK": 22000.0,
		},
	}
}

func (p *ReferenceRateProvider) GetRate(_ context.Context, fromCurrency, toCurrency string, _ time.Time) (float64, error) {
	from := strings.ToUpper(strings.TrimSpace(fromCurrency))
	to := strings.ToUpper(strings.TrimSpace(toCurrency))

	fromRate, okFrom := p.baseRates[from]
	toRate, okTo := p.baseRates[to]
	if !okFrom || !okTo {
		return 0, ErrCurrencyNotSupported
	}
	return toRate / fromRate, nil
}
