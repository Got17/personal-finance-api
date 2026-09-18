package transfer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/validator"
	"github.com/google/uuid"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"
	"github.com/Got17/personal-finance-api/internal/fxquote"
	"github.com/Got17/personal-finance-api/internal/messages"
)

type transferUsecase struct {
	repo         TransferRepository
	accounts     AccountReader
	categories   CategoryReader
	rateProvider FXRateProvider
}

func NewTransferUsecase(repo TransferRepository, accounts AccountReader, categories CategoryReader, rateProvider FXRateProvider) TransferUsecase {
	if rateProvider == nil {
		rateProvider = fxquote.NewReferenceRateProvider()
	}
	return &transferUsecase{
		repo:         repo,
		accounts:     accounts,
		categories:   categories,
		rateProvider: rateProvider,
	}
}

func (u *transferUsecase) CreateTransfer(ctx context.Context, userID string, input *CreateTransferInput) (*Transfer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	if input == nil {
		return nil, validationError("transfer", messages.MsgValidationFailed)
	}
	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}

	srcAcctID := strings.TrimSpace(input.SourceAccountID)
	destAcctID := strings.TrimSpace(input.DestinationAccountID)

	if srcAcctID == destAcctID {
		return nil, validationError("destination_account_id", messages.MsgTransferAccountsMustBeDistinct)
	}

	srcAcct, err := u.getOwnedActiveAccount(ctx, userID, srcAcctID, "source_account_id")
	if err != nil {
		return nil, err
	}
	destAcct, err := u.getOwnedActiveAccount(ctx, userID, destAcctID, "destination_account_id")
	if err != nil {
		return nil, err
	}

	transferID := uuid.NewString()
	transferDate := input.Date.UTC()
	var destAmount int64
	var quote *fxquote.HistoricalFXQuote

	if srcAcct.Currency == destAcct.Currency {
		if input.DestinationAmountMinor != nil && *input.DestinationAmountMinor != input.SourceAmountMinor {
			return nil, validationError("destination_amount_minor", messages.MsgTransferSameCurrencyMismatch)
		}
		destAmount = input.SourceAmountMinor
	} else {
		rate, provenance, err := u.resolveRate(ctx, srcAcct.Currency, destAcct.Currency, transferDate, input.Rate)
		if err != nil {
			return nil, err
		}

		expectedDest, err := fxquote.Convert(srcAcct.Currency, destAcct.Currency, input.SourceAmountMinor, rate)
		if err != nil {
			return nil, validationError("destination_amount_minor", err.Error())
		}

		if input.DestinationAmountMinor != nil {
			if err := fxquote.ValidatePrecision(srcAcct.Currency, destAcct.Currency, input.SourceAmountMinor, *input.DestinationAmountMinor, rate); err != nil {
				return nil, validationError("destination_amount_minor", messages.MsgTransferDestinationAmountMismatch)
			}
			destAmount = *input.DestinationAmountMinor
		} else {
			destAmount = expectedDest
		}

		quoteID := uuid.NewString()
		quote = &fxquote.HistoricalFXQuote{
			ID:            quoteID,
			FromCurrency:  srcAcct.Currency,
			ToCurrency:    destAcct.Currency,
			Rate:          rate,
			EffectiveDate: transferDate,
			Provenance:    provenance,
			RecordID:      transferID,
			CreatedAt:     time.Now().UTC(),
		}
	}

	t := &Transfer{
		ID:                     transferID,
		UserID:                 userID,
		Kind:                   "transfer",
		SourceAccountID:        srcAcct.ID,
		DestinationAccountID:   destAcct.ID,
		SourceAmountMinor:      input.SourceAmountMinor,
		DestinationAmountMinor: destAmount,
		SourceCurrency:         srcAcct.Currency,
		DestinationCurrency:    destAcct.Currency,
		Date:                   transferDate,
		Note:                   strings.TrimSpace(input.Note),
		IsActive:               true,
	}
	if quote != nil {
		t.HistoricalFXQuoteID = &quote.ID
		t.HistoricalFXQuote = quote
	}

	var feeRecord *financialrecord.FinancialRecord
	if input.Fee != nil {
		feeRecord, err = u.createFeeRecord(ctx, userID, transferID, transferDate, input.Fee)
		if err != nil {
			return nil, err
		}
		t.TransferFeeRecordID = &feeRecord.ID
		t.TransferFee = toFeeResult(feeRecord)
	}

	if err := u.repo.CreateTransferWithLegsAndFee(ctx, t, quote, feeRecord); err != nil {
		if errors.Is(err, ErrInsufficientAccountBalance) {
			return nil, errs.UnprocessableFields(
				messages.MsgInsufficientSourceAccountBalance,
				map[string]string{"source_account_id": messages.MsgInsufficientAccountBalance},
			)
		}
		if errors.Is(err, ErrInsufficientFeeAccountBalance) {
			return nil, errs.UnprocessableFields(
				messages.MsgInsufficientFeeAccountBalance,
				map[string]string{"fee.account_id": messages.MsgInsufficientAccountBalance},
			)
		}
		return nil, err
	}
	return t, nil
}

func (u *transferUsecase) GetTransfer(ctx context.Context, userID string, transferID string) (*Transfer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	transferID = strings.TrimSpace(transferID)
	if transferID == "" {
		return nil, errs.NotFound(messages.MsgTransferNotFound)
	}

	t, err := u.repo.FindByID(ctx, transferID)
	if err != nil {
		return nil, errs.NotFound(messages.MsgTransferNotFound)
	}
	if t.UserID != userID {
		return nil, errs.Forbidden(messages.MsgTransferAccessDenied)
	}
	return t, nil
}

func (u *transferUsecase) ListTransfers(ctx context.Context, userID string, filter ListTransferFilter) ([]*Transfer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	return u.repo.FindByUserID(ctx, userID, filter)
}

func (u *transferUsecase) getOwnedActiveAccount(ctx context.Context, userID string, accountID string, field string) (*account.Account, error) {
	acct, err := u.accounts.GetAccount(ctx, userID, accountID)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			return nil, errs.NotFound(messages.MsgAccountNotFound)
		}
		if errors.Is(err, account.ErrAccessDenied) {
			return nil, errs.Forbidden(messages.MsgAccountAccessDenied)
		}
		return nil, err
	}
	if !acct.IsActive {
		return nil, validationError(field, messages.MsgAccountMustBeActive)
	}
	return acct, nil
}

func (u *transferUsecase) resolveRate(ctx context.Context, from, to string, date time.Time, manualRate *float64) (float64, fxquote.Provenance, error) {
	if manualRate != nil {
		if *manualRate <= 0 {
			return 0, "", validationError("rate", messages.MsgTransferRateMustBePositive)
		}
		return *manualRate, fxquote.ProvenanceManualOverride, nil
	}

	rate, err := u.rateProvider.GetRate(ctx, from, to, date)
	if err != nil {
		return 0, "", validationError("rate", messages.MsgFXRateUnavailable)
	}
	return rate, fxquote.ProvenanceProvider, nil
}

func (u *transferUsecase) createFeeRecord(ctx context.Context, userID, transferID string, date time.Time, fee *TransferFeeInput) (*financialrecord.FinancialRecord, error) {
	if fee.AmountMinor <= 0 {
		return nil, validationError("fee.amount_minor", messages.MsgAmountMustBePositive)
	}
	feeAcct, err := u.getOwnedActiveAccount(ctx, userID, strings.TrimSpace(fee.AccountID), "fee.account_id")
	if err != nil {
		return nil, err
	}

	cat, err := u.categories.GetCategory(ctx, userID, strings.TrimSpace(fee.CategoryID))
	if err != nil {
		if errors.Is(err, category.ErrCategoryNotFound) {
			return nil, errs.NotFound(messages.MsgCategoryNotFound)
		}
		if errors.Is(err, category.ErrAccessDenied) {
			return nil, errs.Forbidden(messages.MsgCategoryAccessDenied)
		}
		return nil, err
	}
	if !cat.IsActive {
		return nil, validationError("fee.category_id", messages.MsgCategoryMustBeActive)
	}
	if !cat.MatchesKind("expense") {
		return nil, validationError("fee.category_id", messages.MsgCategoryKindMismatch)
	}

	feeID := uuid.NewString()
	catID := cat.ID
	rec := &financialrecord.FinancialRecord{
		ID:               feeID,
		UserID:           userID,
		Kind:             financialrecord.KindExpense,
		AccountID:        feeAcct.ID,
		CategoryID:       &catID,
		AmountMinor:      fee.AmountMinor,
		Currency:         feeAcct.Currency,
		Date:             date,
		Note:             strings.TrimSpace(fee.Note),
		IsActive:         true,
		LinkedTransferID: &transferID,
	}
	return rec, nil
}

func toFeeResult(f *financialrecord.FinancialRecord) *TransferFeeResult {
	res := &TransferFeeResult{
		ID:          f.ID,
		UserID:      f.UserID,
		Kind:        string(f.Kind),
		AccountID:   f.AccountID,
		AmountMinor: f.AmountMinor,
		Currency:    f.Currency,
		Date:        f.Date,
		Note:        f.Note,
		IsActive:    f.IsActive,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
	if f.CategoryID != nil {
		res.CategoryID = *f.CategoryID
	}
	return res
}

func validationError(field string, message string) error {
	return errs.UnprocessableFields(message, map[string]string{field: message})
}
