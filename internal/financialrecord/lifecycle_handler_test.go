package financialrecord_test

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"
)

func TestFinancialRecordHTTP_UpdatesAndArchivesOwnedRecord(t *testing.T) {
	repo := &editableRecords{items: map[string]*financialrecord.FinancialRecord{
		"record-1": {ID: "record-1", UserID: "user-1", Kind: financialrecord.KindExpense, AccountID: "account-1", CategoryID: "category-1", AmountMinor: 1200, Currency: "USD", Date: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), IsActive: true},
	}}
	usecase := financialrecord.NewFinancialRecordUsecase(repo,
		accounts{"account-1": {ID: "account-1", UserID: "user-1", Currency: "USD", IsActive: true}},
		categories{"category-1": {ID: "category-1", UserID: "user-1", Type: category.CategoryTypeExpense, IsActive: true}},
	)
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	app := fiber.New()
	financialrecord.NewFinancialRecordHandler(usecase).RegisterRoutes(app.Group("/v1", middleware.JWT(tokenAdapter)))
	ownerToken, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-1"}, time.Hour)
	otherToken, _ := tokenAdapter.Sign(contract.Claims{"sub": "user-2"}, time.Hour)

	updateRequest := httptest.NewRequest("PATCH", "/v1/financial-records/record-1", bytes.NewBufferString(`{"amount_minor":2500,"note":"Corrected"}`))
	updateRequest.Header.Set("Authorization", "Bearer "+ownerToken)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResponse, err := app.Test(updateRequest)
	if err != nil {
		t.Fatalf("update request: %v", err)
	}
	defer updateResponse.Body.Close()
	if updateResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("update status = %d, want 200", updateResponse.StatusCode)
	}

	archiveRequest := httptest.NewRequest("DELETE", "/v1/financial-records/record-1", nil)
	archiveRequest.Header.Set("Authorization", "Bearer "+ownerToken)
	archiveResponse, err := app.Test(archiveRequest)
	if err != nil {
		t.Fatalf("archive request: %v", err)
	}
	defer archiveResponse.Body.Close()
	if archiveResponse.StatusCode != fiber.StatusOK || repo.items["record-1"].IsActive {
		t.Fatalf("archive status = %d, record = %#v", archiveResponse.StatusCode, repo.items["record-1"])
	}

	secondUpdateRequest := httptest.NewRequest("PATCH", "/v1/financial-records/record-1", bytes.NewBufferString(`{"note":"Not allowed"}`))
	secondUpdateRequest.Header.Set("Authorization", "Bearer "+ownerToken)
	secondUpdateRequest.Header.Set("Content-Type", "application/json")
	secondUpdateResponse, err := app.Test(secondUpdateRequest)
	if err != nil {
		t.Fatalf("terminal update request: %v", err)
	}
	defer secondUpdateResponse.Body.Close()
	if secondUpdateResponse.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("terminal update status = %d, want 422", secondUpdateResponse.StatusCode)
	}

	foreignArchiveRequest := httptest.NewRequest("DELETE", "/v1/financial-records/record-1", nil)
	foreignArchiveRequest.Header.Set("Authorization", "Bearer "+otherToken)
	foreignArchiveResponse, err := app.Test(foreignArchiveRequest)
	if err != nil {
		t.Fatalf("foreign archive request: %v", err)
	}
	defer foreignArchiveResponse.Body.Close()
	if foreignArchiveResponse.StatusCode != fiber.StatusForbidden {
		t.Fatalf("foreign archive status = %d, want 403", foreignArchiveResponse.StatusCode)
	}
}

var _ account.AccountRepository = accounts{}
