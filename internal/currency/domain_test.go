package currency_test

import (
	"testing"

	"github.com/Got17/personal-finance-api/internal/currency"
)

func TestIsValid(t *testing.T) {
	supported := []string{"LAK", "THB", "USD", "CNY", "EUR"}
	for _, code := range supported {
		if !currency.IsValid(code) {
			t.Errorf("IsValid(%q) = false, want true", code)
		}
	}
}

func TestIsValid_CaseInsensitive(t *testing.T) {
	cases := []string{"lak", "thb", "usd", "cny", "eur", "Usd", " USD "}
	for _, code := range cases {
		if !currency.IsValid(code) {
			t.Errorf("IsValid(%q) = false, want true", code)
		}
	}
}

func TestIsValid_Rejected(t *testing.T) {
	cases := []string{"GBP", "JPY", "AUD", "KRW", "", "US", "USDX", "123"}
	for _, code := range cases {
		if currency.IsValid(code) {
			t.Errorf("IsValid(%q) = true, want false", code)
		}
	}
}

func TestSupportedCurrencies_Order(t *testing.T) {
	want := []string{"LAK", "THB", "USD", "CNY", "EUR"}
	got := currency.SupportedCurrencies()
	if len(got) != len(want) {
		t.Fatalf("len(SupportedCurrencies()) = %d, want %d", len(got), len(want))
	}
	for i, c := range got {
		if c.Code != want[i] {
			t.Errorf("SupportedCurrencies()[%d].Code = %q, want %q", i, c.Code, want[i])
		}
	}
}

func TestSupportedCurrencies_LAKDecimalDigits(t *testing.T) {
	for _, c := range currency.SupportedCurrencies() {
		if c.Code == "LAK" {
			if c.DecimalDigits != 0 {
				t.Errorf("LAK decimal_digits = %d, want 0", c.DecimalDigits)
			}
			return
		}
	}
	t.Fatal("LAK not found in SupportedCurrencies")
}

func TestSupportedCurrencies_IsMutable(t *testing.T) {
	// Mutating the returned slice must not affect subsequent calls.
	first := currency.SupportedCurrencies()
	first[0].Code = "MUTATED"
	second := currency.SupportedCurrencies()
	if second[0].Code != "LAK" {
		t.Errorf("SupportedCurrencies returned the internal slice (mutation leaked)")
	}
}
