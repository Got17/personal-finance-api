package currency

type Currency struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	DecimalDigits int    `json:"decimal_digits"`
}

// supported is the fixed ordered list of currencies accepted by this system.
// Ordering is intentional: LAK first (home currency), then regional peers,
// then major international currencies.
var supported = []Currency{
	{Code: "LAK", Name: "Lao Kip", Symbol: "₭", DecimalDigits: 0},
	{Code: "THB", Name: "Thai Baht", Symbol: "฿", DecimalDigits: 2},
	{Code: "USD", Name: "US Dollar", Symbol: "$", DecimalDigits: 2},
	{Code: "CNY", Name: "Chinese Yuan", Symbol: "¥", DecimalDigits: 2},
	{Code: "EUR", Name: "Euro", Symbol: "€", DecimalDigits: 2},
}

// supportedSet is a fast-lookup index built from the supported slice.
var supportedSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(supported))
	for _, c := range supported {
		m[c.Code] = struct{}{}
	}
	return m
}()

// IsValid reports whether code is an accepted currency code.
// Comparison is case-insensitive; code is normalised to upper-case before lookup.
func IsValid(code string) bool {
	_, ok := supportedSet[normalize(code)]
	return ok
}

// SupportedCurrencies returns a copy of the ordered supported-currency slice.
func SupportedCurrencies() []Currency {
	out := make([]Currency, len(supported))
	copy(out, supported)
	return out
}

// DecimalDigits returns the minor-unit decimal digits for code, or false if unsupported.
func DecimalDigits(code string) (int, bool) {
	norm := normalize(code)
	for _, c := range supported {
		if c.Code == norm {
			return c.DecimalDigits, true
		}
	}
	return 0, false
}

func normalize(code string) string {
	out := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		b := code[i]
		if b == ' ' || b == '\t' {
			continue
		}
		if b >= 'a' && b <= 'z' {
			b -= 32
		}
		out = append(out, b)
	}
	return string(out)
}
