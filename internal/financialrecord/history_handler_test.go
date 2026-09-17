package financialrecord_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/financialrecord"
)

func TestFinancialRecordHTTP_ListsFilteredOwnerHistory(t *testing.T) {
	date := func(day int) time.Time { return time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC) }
	repo := &filteredRecords{items: []*financialrecord.FinancialRecord{
		{ID: "income", UserID: "user-1", Kind: financialrecord.KindIncome, AccountID: "account-a", CategoryID: strPtr("income-category"), Date: date(10), IsActive: true},
		{ID: "expense", UserID: "user-1", Kind: financialrecord.KindExpense, AccountID: "account-b", CategoryID: strPtr("expense-category"), Date: date(11), IsActive: true},
		{ID: "archived", UserID: "user-1", Kind: financialrecord.KindIncome, AccountID: "account-a", CategoryID: strPtr("income-category"), Date: date(12), IsActive: false},
		{ID: "foreign", UserID: "user-2", Kind: financialrecord.KindIncome, AccountID: "account-a", CategoryID: strPtr("income-category"), Date: date(13), IsActive: true},
	}}
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	app := fiber.New()
	financialrecord.NewFinancialRecordHandler(financialrecord.NewFinancialRecordUsecase(repo, accounts{}, categories{})).RegisterRoutes(app.Group("/v1", middleware.JWT(tokenAdapter)))
	token, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	request := httptest.NewRequest("GET", "/v1/financial-records?start_date=2026-09-01&end_date=2026-09-11&kind=income&account_id=account-a&category_id=income-category", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("list request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var body struct {
		Data []financialrecord.FinancialRecord `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != "income" {
		t.Fatalf("filtered history = %#v", body.Data)
	}
}
