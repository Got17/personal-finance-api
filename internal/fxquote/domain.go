package fxquote

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/Got17/personal-finance-api/internal/currency"
	"github.com/Got17/personal-finance-api/internal/messages"
)

var (
	ErrFXQuoteNotFound           = errors.New("fx quote not found")
	ErrCurrencyNotSupported      = errors.New("currency not supported")
	ErrRateUnavailable           = errors.New("rate unavailable")
	ErrDestinationAmountMismatch = errors.New(messages.MsgTransferDestinationAmountMismatch)
)

type Provenance string

const (
	ProvenanceProvider       Provenance = "provider"
	ProvenanceManualOverride Provenance = "manual_override"
)

// HistoricalFXQuote represents an immutable historical exchange rate snapshot linked to a record.
type HistoricalFXQuote struct {
	ID            string     `json:"id"             gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	FromCurrency  string     `json:"from_currency"  gorm:"not null;index"`
	ToCurrency    string     `json:"to_currency"    gorm:"not null;index"`
	Rate          float64    `json:"rate"           gorm:"type:decimal(18,8);not null"`
	EffectiveDate time.Time  `json:"effective_date" gorm:"not null;index"`
	Provenance    Provenance `json:"provenance"     gorm:"not null"`
	RecordID      string     `json:"record_id"      gorm:"type:uuid;not null;index"`
	CreatedAt     time.Time  `json:"created_at"     gorm:"autoCreateTime"`
}

func (HistoricalFXQuote) TableName() string { return "historical_fx_quotes" }

// FXQuoteRepository defines persistence operations for HistoricalFXQuote.
type FXQuoteRepository interface {
	Create(ctx context.Context, quote *HistoricalFXQuote) error
	FindByID(ctx context.Context, id string) (*HistoricalFXQuote, error)
	FindByRecordID(ctx context.Context, recordID string) (*HistoricalFXQuote, error)
}

// FXQuoteProvider defines an interface for fetching exchange rates for currency pairs on a date.
type FXQuoteProvider interface {
	GetRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (float64, error)
}

// Convert converts a source amount in minor units to destination minor units using rate,
// properly accounting for differing decimal digit precisions and rounding to destination minor units.
func Convert(fromCurrency, toCurrency string, sourceMinor int64, rate float64) (int64, error) {
	if rate <= 0 {
		return 0, errors.New("rate must be greater than zero")
	}
	fromDigits, ok := currency.DecimalDigits(fromCurrency)
	if !ok {
		return 0, ErrCurrencyNotSupported
	}
	toDigits, ok := currency.DecimalDigits(toCurrency)
	if !ok {
		return 0, ErrCurrencyNotSupported
	}

	powerDiff := toDigits - fromDigits
	converted := float64(sourceMinor) * rate * math.Pow10(powerDiff)
	return int64(math.Round(converted)), nil
}

// ValidatePrecision checks that destinationMinor exactly matches rate conversion precision.
func ValidatePrecision(fromCurrency, toCurrency string, sourceMinor, destMinor int64, rate float64) error {
	expected, err := Convert(fromCurrency, toCurrency, sourceMinor, rate)
	if err != nil {
		return err
	}
	if destMinor != expected {
		return ErrDestinationAmountMismatch
	}
	return nil
}
