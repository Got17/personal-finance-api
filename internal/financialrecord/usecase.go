package financialrecord

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
	"github.com/Got17/personal-finance-api/internal/messages"
	"github.com/Got17/personal-finance-api/internal/user"
)

type CreateFinancialRecordInput struct {
	Kind        string    `json:"kind"         validate:"required"`
	AccountID   string    `json:"account_id"   validate:"required"`
	CategoryID  string    `json:"category_id"  validate:"required"`
	AmountMinor int64     `json:"amount_minor" validate:"required,gt=0"`
	Currency    string    `json:"currency"     validate:"required"`
	Date        time.Time `json:"date"         validate:"required"`
	Note        string    `json:"note"         validate:"omitempty,max=1000"`
}

type UpdateFinancialRecordInput struct {
	Kind        *string    `json:"kind"         validate:"omitempty"`
	AccountID   *string    `json:"account_id"   validate:"omitempty"`
	CategoryID  *string    `json:"category_id"  validate:"omitempty"`
	AmountMinor *int64     `json:"amount_minor" validate:"omitempty,gt=0"`
	Currency    *string    `json:"currency"     validate:"omitempty"`
	Date        *time.Time `json:"date"         validate:"omitempty"`
	Note        *string    `json:"note"         validate:"omitempty,max=1000"`
}
type financialRecordUsecase struct {
	records    FinancialRecordRepository
	accounts   AccountReader
	categories CategoryReader
}

func NewFinancialRecordUsecase(records FinancialRecordRepository, accounts AccountReader, categories CategoryReader) FinancialRecordUsecase {
	return &financialRecordUsecase{records: records, accounts: accounts, categories: categories}
}


func (u *financialRecordUsecase) CreateFinancialRecord(ctx context.Context, userID string, input *CreateFinancialRecordInput) (*FinancialRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	if input == nil {
		return nil, validationError("record", messages.MsgValidationFailed)
	}
	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}
	if input.Date.IsZero() {
		return nil, validationError("date", messages.MsgDateIsRequired)
	}

	kind := Kind(strings.ToLower(strings.TrimSpace(input.Kind)))
	if !IsValidKind(string(kind)) {
		return nil, validationError("kind", messages.MsgUnsupportedFinancialRecordKind)
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if !user.IsValidISO4217(currency) {
		return nil, validationError("currency", messages.MsgInvalidCurrencyCode)
	}
	acct, err := u.findOwnedActiveAccount(ctx, userID, input.AccountID)
	if err != nil {
		return nil, err
	}
	if acct.Currency != currency {
		return nil, validationError("currency", messages.MsgAccountCurrencyMismatch)
	}
	if err := u.validateCategory(ctx, userID, input.CategoryID, kind); err != nil {
		return nil, err
	}

	record := &FinancialRecord{ID: uuid.NewString(), UserID: userID, Kind: kind, AccountID: strings.TrimSpace(input.AccountID), CategoryID: strings.TrimSpace(input.CategoryID), AmountMinor: input.AmountMinor, Currency: currency, Date: input.Date.UTC(), Note: strings.TrimSpace(input.Note), IsActive: true}
	if err := u.records.Create(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (u *financialRecordUsecase) GetFinancialRecord(ctx context.Context, userID string, recordID string) (*FinancialRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	if strings.TrimSpace(recordID) == "" {
		return nil, errs.NotFound(messages.MsgFinancialRecordNotFound)
	}
	record, err := u.records.FindByID(ctx, recordID)
	if err != nil {
		if errors.Is(err, ErrFinancialRecordNotFound) {
			return nil, errs.NotFound(messages.MsgFinancialRecordNotFound)
		}
		return nil, err
	}
	if record.UserID != userID {
		return nil, errs.Forbidden(messages.MsgFinancialRecordAccessDenied)
	}
	return record, nil
}

func (u *financialRecordUsecase) ListFinancialRecords(ctx context.Context, userID string, filter ListFilter) ([]*FinancialRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	if filter.Kind != "" && !IsValidKind(string(filter.Kind)) {
		return nil, validationError("kind", messages.MsgUnsupportedFinancialRecordKind)
	}
	if filter.StartDate != nil && filter.EndDate != nil && filter.StartDate.After(*filter.EndDate) {
		return nil, validationError("date", messages.MsgInvalidDateRange)
	}
	records, err := u.records.FindByUserID(ctx, userID, filter)
	if err != nil {
		return nil, err
	}
	if records == nil {
		records = []*FinancialRecord{}
	}
	return records, nil
}

func (u *financialRecordUsecase) findOwnedActiveAccount(ctx context.Context, userID string, accountID string) (*account.Account, error) {
	acct, err := u.accounts.GetAccount(ctx, userID, strings.TrimSpace(accountID))
	if err != nil {
		return nil, err
	}
	if !acct.IsActive {
		return nil, validationError("account_id", messages.MsgAccountMustBeActive)
	}
	return acct, nil
}

func (u *financialRecordUsecase) validateCategory(ctx context.Context, userID string, categoryID string, kind Kind) error {
	cat, err := u.categories.GetCategory(ctx, userID, strings.TrimSpace(categoryID))
	if err != nil {
		return err
	}
	if !cat.IsActive {
		return validationError("category_id", messages.MsgCategoryMustBeActive)
	}
	if category.CategoryType(kind) != cat.Type {
		return validationError("category_id", messages.MsgCategoryKindMismatch)
	}
	return nil
}


func validationError(field string, message string) error {
	return errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{field: message})
}

func (u *financialRecordUsecase) UpdateFinancialRecord(ctx context.Context, userID string, recordID string, input *UpdateFinancialRecordInput) (*FinancialRecord, error) {
	if input == nil {
		return nil, validationError("record", messages.MsgValidationFailed)
	}
	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}
	record, err := u.GetFinancialRecord(ctx, userID, recordID)
	if err != nil {
		return nil, err
	}
	if !record.IsActive {
		return nil, validationError("record", messages.MsgFinancialRecordArchived)
	}

	kind, accountID, categoryID, amountMinor, currency, date, note := updateValues(record, input)
	if !IsValidKind(string(kind)) {
		return nil, validationError("kind", messages.MsgUnsupportedFinancialRecordKind)
	}
	if amountMinor <= 0 {
		return nil, validationError("amount_minor", messages.MsgValidationFailed)
	}
	if date.IsZero() {
		return nil, validationError("date", messages.MsgDateIsRequired)
	}
	if !user.IsValidISO4217(currency) {
		return nil, validationError("currency", messages.MsgInvalidCurrencyCode)
	}
	account, err := u.findOwnedActiveAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	if account.Currency != currency {
		return nil, validationError("currency", messages.MsgAccountCurrencyMismatch)
	}
	if err := u.validateCategory(ctx, userID, categoryID, kind); err != nil {
		return nil, err
	}

	record.Kind, record.AccountID, record.CategoryID = kind, accountID, categoryID
	record.AmountMinor, record.Currency, record.Date, record.Note = amountMinor, currency, date.UTC(), note
	if err := u.records.Update(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (u *financialRecordUsecase) ArchiveFinancialRecord(ctx context.Context, userID string, recordID string) (*FinancialRecord, error) {
	record, err := u.GetFinancialRecord(ctx, userID, recordID)
	if err != nil {
		return nil, err
	}
	if !record.IsActive {
		return nil, validationError("record", messages.MsgFinancialRecordArchived)
	}
	record.IsActive = false
	if err := u.records.Update(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func updateValues(record *FinancialRecord, input *UpdateFinancialRecordInput) (Kind, string, string, int64, string, time.Time, string) {
	kind, accountID, categoryID := record.Kind, record.AccountID, record.CategoryID
	amountMinor, currency, date, note := record.AmountMinor, record.Currency, record.Date, record.Note
	if input.Kind != nil {
		kind = Kind(strings.ToLower(strings.TrimSpace(*input.Kind)))
	}
	if input.AccountID != nil {
		accountID = strings.TrimSpace(*input.AccountID)
	}
	if input.CategoryID != nil {
		categoryID = strings.TrimSpace(*input.CategoryID)
	}
	if input.AmountMinor != nil {
		amountMinor = *input.AmountMinor
	}
	if input.Currency != nil {
		currency = strings.ToUpper(strings.TrimSpace(*input.Currency))
	}
	if input.Date != nil {
		date = input.Date.UTC()
	}
	if input.Note != nil {
		note = strings.TrimSpace(*input.Note)
	}
	return kind, accountID, categoryID, amountMinor, currency, date, note
}
