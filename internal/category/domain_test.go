package category_test

import (
	"testing"

	"github.com/Got17/personal-finance-api/internal/category"
)

func TestCategoryMatchesKind(t *testing.T) {
	incomeCat := &category.Category{
		Type: category.CategoryTypeIncome,
	}
	expenseCat := &category.Category{
		Type: category.CategoryTypeExpense,
	}

	tests := []struct {
		name     string
		cat      *category.Category
		kind     string
		expected bool
	}{
		{"income matches income", incomeCat, "income", true},
		{"income matches uppercase INCOME", incomeCat, "INCOME", true},
		{"income does not match expense", incomeCat, "expense", false},
		{"income does not match empty", incomeCat, "", false},
		{"expense matches expense", expenseCat, "expense", true},
		{"expense matches uppercase EXPENSE", expenseCat, "EXPENSE", true},
		{"expense does not match income", expenseCat, "income", false},
		{"nil category returns false", nil, "income", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cat.MatchesKind(tc.kind)
			if got != tc.expected {
				t.Errorf("MatchesKind(%q) = %v, want %v", tc.kind, got, tc.expected)
			}
		})
	}
}
