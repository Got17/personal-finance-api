package fxquote_test

import (
	"math"
	"testing"

	"github.com/Got17/personal-finance-api/internal/fxquote"
)

func TestConvert_Precision(t *testing.T) {
	tests := []struct {
		name        string
		from        string
		to          string
		sourceMinor int64
		rate        float64
		wantDest    int64
	}{
		{
			name:        "USD to EUR with 0.92 rate",
			from:        "USD",
			to:          "EUR",
			sourceMinor: 1000, // 10.00 USD
			rate:        0.92,
			wantDest:    920, // 9.20 EUR
		},
		{
			name:        "USD to LAK with 22000.0 rate (decimal diff)",
			from:        "USD",
			to:          "LAK",
			sourceMinor: 1000, // 10.00 USD
			rate:        22000.0,
			wantDest:    220000, // 220,000 LAK (0 decimals)
		},
		{
			name:        "LAK to USD with rate",
			from:        "LAK",
			to:          "USD",
			sourceMinor: 220000, // 220,000 LAK
			rate:        1.0 / 22000.0,
			wantDest:    1000, // 10.00 USD
		},
		{
			name:        "THB to EUR with rounding half to even / nearest",
			from:        "THB",
			to:          "EUR",
			sourceMinor: 3550, // 35.50 THB
			rate:        0.0263,
			wantDest:    int64(math.Round(35.50 * 0.0263 * 100)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fxquote.Convert(tt.from, tt.to, tt.sourceMinor, tt.rate)
			if err != nil {
				t.Fatalf("Convert() unexpected error: %v", err)
			}
			if got != tt.wantDest {
				t.Errorf("Convert() = %d, want %d", got, tt.wantDest)
			}
		})
	}
}

func TestValidatePrecision(t *testing.T) {
	// Valid match
	err := fxquote.ValidatePrecision("USD", "EUR", 1000, 920, 0.92)
	if err != nil {
		t.Errorf("ValidatePrecision() unexpected error: %v", err)
	}

	// Mismatched destination amount
	err = fxquote.ValidatePrecision("USD", "EUR", 1000, 950, 0.92)
	if err == nil {
		t.Error("ValidatePrecision() expected error for mismatched destination, got nil")
	}
}
