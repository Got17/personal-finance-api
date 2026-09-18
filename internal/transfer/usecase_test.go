package transfer_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/errs"

	"github.com/Got17/personal-finance-api/internal/financialrecord"
	"github.com/Got17/personal-finance-api/internal/fxquote"
	"github.com/Got17/personal-finance-api/internal/transfer"
)

func TestCreateTransfer_SameCurrency_Success(t *testing.T) {
	uc, _, _, _ := setupTransferTest()
	destAmount := int64(1000)

	result, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:        "acct-usd-1",
		DestinationAccountID:   "acct-usd-2",
		SourceAmountMinor:      1000,
		DestinationAmountMinor: &destAmount,
		Date:                   time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC),
		Note:                   "Same-currency transfer",
	})
	if err != nil {
		t.Fatalf("CreateTransfer() unexpected error: %v", err)
	}

	if result.Kind != "transfer" {
		t.Errorf("Kind = %q, want 'transfer'", result.Kind)
	}
	if result.SourceAmountMinor != 1000 || result.DestinationAmountMinor != 1000 {
		t.Errorf("Amounts = (%d, %d), want (1000, 1000)", result.SourceAmountMinor, result.DestinationAmountMinor)
	}
	if result.SourceCurrency != "USD" || result.DestinationCurrency != "USD" {
		t.Errorf("Currencies = (%s, %s), want (USD, USD)", result.SourceCurrency, result.DestinationCurrency)
	}
	if result.HistoricalFXQuoteID != nil {
		t.Errorf("HistoricalFXQuoteID = %v, want nil for same-currency transfer", result.HistoricalFXQuoteID)
	}
}

func TestCreateTransfer_SameCurrency_DifferentAmounts_Fails(t *testing.T) {
	uc, _, _, _ := setupTransferTest()
	destAmount := int64(1200)

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:        "acct-usd-1",
		DestinationAccountID:   "acct-usd-2",
		SourceAmountMinor:      1000,
		DestinationAmountMinor: &destAmount,
		Date:                   time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for mismatched same-currency amounts, got nil")
	}
	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateTransfer_SameAccount_Fails(t *testing.T) {
	uc, _, _, _ := setupTransferTest()

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-1",
		SourceAmountMinor:    1000,
		Date:                 time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for identical source and destination accounts, got nil")
	}
	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateTransfer_CrossUserAccount_Forbidden(t *testing.T) {
	uc, _, _, _ := setupTransferTest()

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-other-user",
		SourceAmountMinor:    1000,
		Date:                 time.Now(),
	})
	if err == nil {
		t.Fatal("expected forbidden error for other user's account, got nil")
	}
	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 403 {
		t.Fatalf("expected 403 AppError, got %#v", err)
	}
}

func TestCreateTransfer_InactiveAccount_Fails(t *testing.T) {
	uc, _, _, _ := setupTransferTest()

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-inactive",
		DestinationAccountID: "acct-usd-1",
		SourceAmountMinor:    1000,
		Date:                 time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for inactive account, got nil")
	}
	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateTransfer_CrossCurrency_AutomaticQuote_Success(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	destAmount := int64(920) // 1000 * 0.92 = 920

	result, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:        "acct-usd-1",
		DestinationAccountID:   "acct-eur-1",
		SourceAmountMinor:      1000,
		DestinationAmountMinor: &destAmount,
		Date:                   time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateTransfer() unexpected error: %v", err)
	}

	if result.HistoricalFXQuoteID == nil {
		t.Fatal("expected HistoricalFXQuoteID to be populated")
	}
	quote, ok := repo.quotes[*result.HistoricalFXQuoteID]
	if !ok {
		t.Fatalf("quote not found in repo: %s", *result.HistoricalFXQuoteID)
	}
	if quote.FromCurrency != "USD" || quote.ToCurrency != "EUR" {
		t.Errorf("Quote pair = %s/%s, want USD/EUR", quote.FromCurrency, quote.ToCurrency)
	}
	if quote.Rate != 0.92 {
		t.Errorf("Quote rate = %v, want 0.92", quote.Rate)
	}
	if quote.Provenance != fxquote.ProvenanceProvider {
		t.Errorf("Quote provenance = %q, want %q", quote.Provenance, fxquote.ProvenanceProvider)
	}
	if quote.RecordID != result.ID {
		t.Errorf("Quote RecordID = %q, want %q", quote.RecordID, result.ID)
	}
}

func TestCreateTransfer_CrossCurrency_ManualOverride_Success(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	manualRate := 0.95
	destAmount := int64(950) // 1000 * 0.95 = 950

	result, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:        "acct-usd-1",
		DestinationAccountID:   "acct-eur-1",
		SourceAmountMinor:      1000,
		DestinationAmountMinor: &destAmount,
		Rate:                   &manualRate,
		Date:                   time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateTransfer() unexpected error: %v", err)
	}

	quote := repo.quotes[*result.HistoricalFXQuoteID]
	if quote.Rate != 0.95 {
		t.Errorf("Quote rate = %v, want 0.95", quote.Rate)
	}
	if quote.Provenance != fxquote.ProvenanceManualOverride {
		t.Errorf("Quote provenance = %q, want %q", quote.Provenance, fxquote.ProvenanceManualOverride)
	}
}

func TestCreateTransfer_CrossCurrency_PrecisionMismatch_Fails(t *testing.T) {
	uc, _, _, _ := setupTransferTest()
	destAmount := int64(950) // Rate is 0.92 so 1000 * 0.92 should be 920, not 950

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:        "acct-usd-1",
		DestinationAccountID:   "acct-eur-1",
		SourceAmountMinor:      1000,
		DestinationAmountMinor: &destAmount,
		Date:                   time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for destination amount precision mismatch, got nil")
	}
	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateTransfer_WithLinkedFee_Success(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()

	result, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    1000,
		Date:                 time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC),
		Fee: &transfer.TransferFeeInput{
			AccountID:   "acct-usd-1",
			CategoryID:  "cat-expense",
			AmountMinor: 50,
			Note:        "Wire transfer fee",
		},
	})
	if err != nil {
		t.Fatalf("CreateTransfer() unexpected error: %v", err)
	}

	if result.TransferFeeRecordID == nil {
		t.Fatal("expected TransferFeeRecordID to be populated")
	}
	feeRecord, ok := repo.feeRecords[*result.TransferFeeRecordID]
	if !ok {
		t.Fatalf("fee record not found: %s", *result.TransferFeeRecordID)
	}
	if feeRecord.Kind != financialrecord.KindExpense {
		t.Errorf("fee Kind = %q, want 'expense'", feeRecord.Kind)
	}
	if feeRecord.AmountMinor != 50 {
		t.Errorf("fee AmountMinor = %d, want 50", feeRecord.AmountMinor)
	}
	if feeRecord.Currency != "USD" {
		t.Errorf("fee Currency = %q, want USD", feeRecord.Currency)
	}
	if feeRecord.LinkedTransferID == nil || *feeRecord.LinkedTransferID != result.ID {
		t.Errorf("fee LinkedTransferID = %v, want %q", feeRecord.LinkedTransferID, result.ID)
	}
}

func TestCreateTransfer_WithFee_CategoryNotExpense_Fails(t *testing.T) {
	uc, _, _, _ := setupTransferTest()

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    1000,
		Date:                 time.Now(),
		Fee: &transfer.TransferFeeInput{
			AccountID:   "acct-usd-1",
			CategoryID:  "cat-income", // income category!
			AmountMinor: 50,
		},
	})
	if err == nil {
		t.Fatal("expected error for fee category not being expense, got nil")
	}
	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
}

func TestCreateTransfer_AtomicityFailure_Rollback(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	repo.failOnCreate = true

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    1000,
		Date:                 time.Now(),
	})
	if err == nil {
		t.Fatal("expected error on db transaction failure, got nil")
	}

	if len(repo.transfers) != 0 {
		t.Errorf("expected 0 transfers saved on failure, got %d", len(repo.transfers))
	}
}

func TestGetTransfer_Success(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()

	created, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    1000,
		Date:                 time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateTransfer() unexpected error: %v", err)
	}

	got, err := (*uc).GetTransfer(context.Background(), "user-1", created.ID)
	if err != nil {
		t.Fatalf("GetTransfer() unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("got ID %q, want %q", got.ID, created.ID)
	}

	// Unauthorized access from another user
	_, err = (*uc).GetTransfer(context.Background(), "user-2", created.ID)
	if err == nil {
		t.Fatal("expected 403 error for another user, got nil")
	}
	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 403 {
		t.Fatalf("expected 403 AppError, got %#v", err)
	}
	_ = repo
}

func TestCreateTransfer_ZeroBalance_Fails(t *testing.T) {
	uc, _, _, _ := setupTransferTest()

	// acct-usd-2 starts with 0 balance
	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-2",
		DestinationAccountID: "acct-usd-1",
		SourceAmountMinor:    5000,
		Date:                 time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for zero balance account, got nil")
	}

	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
	fieldsMap, ok := appErr.Data.(map[string]string)
	if !ok || fieldsMap["source_account_id"] != "insufficient account balance" {
		t.Errorf("expected source_account_id error 'insufficient account balance', got %v", appErr.Data)
	}
}

func TestCreateTransfer_ExceedingBalance_Fails(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	repo.SetAccountBalance("acct-usd-1", 5000)

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    5001,
		Date:                 time.Now(),
	})
	if err == nil {
		t.Fatal("expected error when transfer exceeds balance, got nil")
	}

	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
	fieldsMap, ok := appErr.Data.(map[string]string)
	if !ok || fieldsMap["source_account_id"] != "insufficient account balance" {
		t.Errorf("expected source_account_id error 'insufficient account balance', got %v", appErr.Data)
	}
}

func TestCreateTransfer_FeeExceedingSourceBalance_Fails(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	repo.SetAccountBalance("acct-usd-1", 1000)

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    950,
		Date:                 time.Now(),
		Fee: &transfer.TransferFeeInput{
			AccountID:   "acct-usd-1",
			CategoryID:  "cat-expense",
			AmountMinor: 60, // 950 + 60 = 1010 > 1000
		},
	})
	if err == nil {
		t.Fatal("expected error when transfer + fee exceeds balance, got nil")
	}

	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
	fieldsMap, ok := appErr.Data.(map[string]string)
	if !ok || fieldsMap["source_account_id"] != "insufficient account balance" {
		t.Errorf("expected source_account_id error 'insufficient account balance', got %v", appErr.Data)
	}
}

func TestCreateTransfer_FeeExceedingDistinctFeeAccountBalance_Fails(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	repo.SetAccountBalance("acct-usd-1", 50000)
	repo.SetAccountBalance("acct-usd-2", 10) // fee account has only 10

	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    1000,
		Date:                 time.Now(),
		Fee: &transfer.TransferFeeInput{
			AccountID:   "acct-usd-2",
			CategoryID:  "cat-expense",
			AmountMinor: 50, // 50 > 10
		},
	})
	if err == nil {
		t.Fatal("expected error when fee exceeds distinct fee account balance, got nil")
	}

	appErr, ok := errs.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Fatalf("expected 422 AppError, got %#v", err)
	}
	fieldsMap, ok := appErr.Data.(map[string]string)
	if !ok || fieldsMap["fee.account_id"] != "insufficient account balance" {
		t.Errorf("expected fee.account_id error 'insufficient account balance', got %v", appErr.Data)
	}
}

func TestCreateTransfer_FundedByInboundTransfer_CascadingTransfer_Success(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	repo.SetAccountBalance("acct-usd-1", 5000)
	repo.SetAccountBalance("acct-usd-2", 0)

	// Transfer 3000 from acct-usd-1 to acct-usd-2
	_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-1",
		DestinationAccountID: "acct-usd-2",
		SourceAmountMinor:    3000,
		Date:                 time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error on first transfer: %v", err)
	}

	// Now acct-usd-2 has 3000, transfer 2000 from acct-usd-2 back to acct-usd-1
	res, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
		SourceAccountID:      "acct-usd-2",
		DestinationAccountID: "acct-usd-1",
		SourceAmountMinor:    2000,
		Date:                 time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error on cascading transfer: %v", err)
	}
	if res.SourceAmountMinor != 2000 {
		t.Errorf("got source amount %d, want 2000", res.SourceAmountMinor)
	}

	// Check final balances: acct-usd-1: 5000 - 3000 + 2000 = 4000; acct-usd-2: 0 + 3000 - 2000 = 1000
	bal1, _ := repo.GetAccountBalance(context.Background(), "user-1", "acct-usd-1")
	if bal1 != 4000 {
		t.Errorf("acct-usd-1 balance = %d, want 4000", bal1)
	}
	bal2, _ := repo.GetAccountBalance(context.Background(), "user-1", "acct-usd-2")
	if bal2 != 1000 {
		t.Errorf("acct-usd-2 balance = %d, want 1000", bal2)
	}
}

func TestCreateTransfer_ConcurrentTransfers_PreventOverdraft(t *testing.T) {
	uc, repo, _, _ := setupTransferTest()
	repo.SetAccountBalance("acct-usd-1", 10000)
	repo.SetAccountBalance("acct-usd-2", 0)

	var wg sync.WaitGroup
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := (*uc).CreateTransfer(context.Background(), "user-1", &transfer.CreateTransferInput{
				SourceAccountID:      "acct-usd-1",
				DestinationAccountID: "acct-usd-2",
				SourceAmountMinor:    8000,
				Date:                 time.Now(),
			})
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	var successCount, failureCount int
	for err := range results {
		if err == nil {
			successCount++
		} else {
			failureCount++
			appErr, ok := errs.IsAppError(err)
			if !ok || appErr.Status != 422 {
				t.Errorf("expected 422 error on concurrent overdraft, got %v", err)
			}
		}
	}

	if successCount != 1 || failureCount != 1 {
		t.Fatalf("expected exactly 1 success and 1 failure, got %d successes and %d failures", successCount, failureCount)
	}

	bal, _ := repo.GetAccountBalance(context.Background(), "user-1", "acct-usd-1")
	if bal != 2000 {
		t.Errorf("acct-usd-1 balance = %d, want 2000 (10000 - 8000)", bal)
	}
}
