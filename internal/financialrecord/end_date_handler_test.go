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

func TestFinancialRecordHTTP_EndDateIncludesEntireRequestedDay(t *testing.T) {
	repo := &filteredRecords{items: []*financialrecord.FinancialRecord{
		{ID: "included", UserID: "user-1", Kind: financialrecord.KindIncome, Date: time.Date(2026, 9, 10, 23, 59, 59, 0, time.UTC), IsActive: true},
		{ID: "excluded", UserID: "user-1", Kind: financialrecord.KindIncome, Date: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC), IsActive: true},
	}}
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	app := fiber.New()
	financialrecord.NewFinancialRecordHandler(financialrecord.NewFinancialRecordUsecase(repo, accounts{}, categories{})).RegisterRoutes(app.Group("/v1", middleware.JWT(tokenAdapter)))
	token, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	request := httptest.NewRequest("GET", "/v1/financial-records?end_date=2026-09-10", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request: %v", err)
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
	if len(body.Data) != 1 || body.Data[0].ID != "included" {
		t.Fatalf("records = %#v", body.Data)
	}
}
