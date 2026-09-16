package financialrecord_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"
	"github.com/gofiber/fiber/v2"
)

func TestFinancialRecordHTTP_CreatesListsAndProtectsOwnerRecords(t *testing.T) {
	repo := &records{}
	usecase := financialrecord.NewFinancialRecordUsecase(repo,
		accounts{"account-1": {ID: "account-1", UserID: "user-1", Currency: "USD", IsActive: true}},
		categories{"category-1": {ID: "category-1", UserID: "user-1", Type: category.CategoryTypeIncome, IsActive: true}},
	)
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	app := fiber.New()
	financialrecord.NewFinancialRecordHandler(usecase).RegisterRoutes(app.Group("/v1", middleware.JWT(tokenAdapter)))
	ownerToken, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)
	otherToken, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-2"}, time.Hour)

	request := httptest.NewRequest("POST", "/v1/financial-records", bytes.NewBufferString(`{"kind":"income","account_id":"account-1","category_id":"category-1","amount_minor":12345,"currency":"USD","date":"2026-09-10T00:00:00Z"}`))
	request.Header.Set("Authorization", "Bearer "+ownerToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create status = %d, want 201", response.StatusCode)
	}
	var created struct {
		Data financialrecord.FinancialRecord `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	listRequest := httptest.NewRequest("GET", "/v1/financial-records?kind=income", nil)
	listRequest.Header.Set("Authorization", "Bearer "+ownerToken)
	listResponse, err := app.Test(listRequest)
	if err != nil {
		t.Fatalf("list request: %v", err)
	}
	defer listResponse.Body.Close()
	if listResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("list status = %d, want 200", listResponse.StatusCode)
	}

	getRequest := httptest.NewRequest("GET", "/v1/financial-records/"+created.Data.ID, nil)
	getRequest.Header.Set("Authorization", "Bearer "+otherToken)
	getResponse, err := app.Test(getRequest)
	if err != nil {
		t.Fatalf("cross-user get request: %v", err)
	}
	defer getResponse.Body.Close()
	if getResponse.StatusCode != fiber.StatusForbidden {
		t.Fatalf("cross-user get status = %d, want 403", getResponse.StatusCode)
	}
}

func TestFinancialRecordHTTP_RejectsInvalidAndForeignCreation(t *testing.T) {
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	usecase := financialrecord.NewFinancialRecordUsecase(&records{}, accounts{
		"owned-account":   {ID: "owned-account", UserID: "user-1", Currency: "USD", IsActive: true},
		"foreign-account": {ID: "foreign-account", UserID: "user-2", Currency: "USD", IsActive: true},
	}, categories{"category-1": {ID: "category-1", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true}})
	app := fiber.New()
	financialrecord.NewFinancialRecordHandler(usecase).RegisterRoutes(app.Group("/v1", middleware.JWT(tokenAdapter)))
	token, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)

	for _, test := range []struct {
		name, accountID, categoryID, amount, date string
		wantStatus                                int
	}{
		{name: "non-positive zero amount", accountID: "owned-account", categoryID: "category-1", amount: "0", date: "2026-09-10T00:00:00Z", wantStatus: fiber.StatusUnprocessableEntity},
		{name: "negative amount", accountID: "owned-account", categoryID: "category-1", amount: "-100", date: "2026-09-10T00:00:00Z", wantStatus: fiber.StatusUnprocessableEntity},
		{name: "empty category", accountID: "owned-account", categoryID: "   ", amount: "100", date: "2026-09-10T00:00:00Z", wantStatus: fiber.StatusUnprocessableEntity},
		{name: "zero date", accountID: "owned-account", categoryID: "category-1", amount: "100", date: "0001-01-01T00:00:00Z", wantStatus: fiber.StatusUnprocessableEntity},
		{name: "foreign account", accountID: "foreign-account", categoryID: "category-1", amount: "1", date: "2026-09-10T00:00:00Z", wantStatus: fiber.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := `{"kind":"expense","account_id":"` + test.accountID + `","category_id":"` + test.categoryID + `","amount_minor":` + test.amount + `,"currency":"USD","date":"` + test.date + `"}`
			request := httptest.NewRequest("POST", "/v1/financial-records", bytes.NewBufferString(body))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Content-Type", "application/json")
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
		})
	}
}
