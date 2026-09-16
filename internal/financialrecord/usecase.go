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
	"github.com/Got17/personal-finance-api/internal/currency"
	"github.com/Got17/personal-finance-api/internal/messages"
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

// recordFields is the normalized set of business fields shared by record
// creation and replacement, so both paths validate the same values the same way.
type recordFields struct {
	Kind        Kind
	AccountID   string
	CategoryID  string
	AmountMinor int64
	Currency    string
	Date        time.Time
	Note        string
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
	fields := recordFields{
		Kind:        Kind(strings.ToLower(strings.TrimSpace(input.Kind))),
		AccountID:   strings.TrimSpace(input.AccountID),
		CategoryID:  strings.TrimSpace(input.CategoryID),
		AmountMinor: input.AmountMinor,
		Currency:    strings.ToUpper(strings.TrimSpace(input.Currency)),
		Date:        input.Date.UTC(),
		Note:        strings.TrimSpace(input.Note),
	}
	if err := u.validateActiveReferences(ctx, userID, fields); err != nil {
		return nil, err
	}

	record := &FinancialRecord{
		ID:          uuid.NewString(),
		UserID:      userID,
		Kind:        fields.Kind,
		AccountID:   fields.AccountID,
		CategoryID:  fields.CategoryID,
		AmountMinor: fields.AmountMinor,
		Currency:    fields.Currency,
		Date:        fields.Date,
		Note:        fields.Note,
		IsActive:    true,
	}
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
	if filter.EndDate != nil {
		filter.EndDate = inclusiveEndDate(filter.EndDate)
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
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, validationError("account_id", messages.MsgValidationFailed)
	}
	acct, err := u.accounts.GetAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	if !acct.IsActive {
		return nil, validationError("account_id", messages.MsgAccountMustBeActive)
	}
	return acct, nil
}

func (u *financialRecordUsecase) validateCategory(ctx context.Context, userID string, categoryID string, kind Kind) error {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return validationError("category_id", messages.MsgValidationFailed)
	}
	cat, err := u.categories.GetCategory(ctx, userID, categoryID)
	if err != nil {
		return err
	}
	if !cat.IsActive {
		return validationError("category_id", messages.MsgCategoryMustBeActive)
	}
	if !cat.MatchesKind(string(kind)) {
		return validationError("category_id", messages.MsgCategoryKindMismatch)
	}
	return nil
}

// validateActiveReferences is the single source of truth for all business and active
// reference validation rules shared by create and update.
func (u *financialRecordUsecase) validateActiveReferences(ctx context.Context, userID string, fields recordFields) error {
	if fields.AmountMinor <= 0 {
		return validationError("amount_minor", messages.MsgAmountMustBePositive)
	}
	if fields.Date.IsZero() {
		return validationError("date", messages.MsgDateIsRequired)
	}
	if !IsValidKind(string(fields.Kind)) {
		return validationError("kind", messages.MsgUnsupportedFinancialRecordKind)
	}
	if !currency.IsValid(fields.Currency) {
		return validationError("currency", messages.MsgInvalidCurrencyCode)
	}
	acct, err := u.findOwnedActiveAccount(ctx, userID, fields.AccountID)
	if err != nil {
		return err
	}
	if acct.Currency != fields.Currency {
		return validationError("currency", messages.MsgAccountCurrencyMismatch)
	}
	return u.validateCategory(ctx, userID, fields.CategoryID, fields.Kind)
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

	fields := updateValues(record, input)
	if err := u.validateActiveReferences(ctx, userID, fields); err != nil {
		return nil, err
	}

	record.Kind, record.AccountID, record.CategoryID = fields.Kind, fields.AccountID, fields.CategoryID
	record.AmountMinor, record.Currency, record.Date, record.Note = fields.AmountMinor, fields.Currency, fields.Date, fields.Note
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

// updateValues overlays the changed fields from input onto record's current
// values, so validateActiveReferences always sees the full post-update state.
func updateValues(record *FinancialRecord, input *UpdateFinancialRecordInput) recordFields {
	fields := recordFields{
		Kind:        record.Kind,
		AccountID:   record.AccountID,
		CategoryID:  record.CategoryID,
		AmountMinor: record.AmountMinor,
		Currency:    record.Currency,
		Date:        record.Date,
		Note:        record.Note,
	}
	if input.Kind != nil {
		fields.Kind = Kind(strings.ToLower(strings.TrimSpace(*input.Kind)))
	}
	if input.AccountID != nil {
		fields.AccountID = strings.TrimSpace(*input.AccountID)
	}
	if input.CategoryID != nil {
		fields.CategoryID = strings.TrimSpace(*input.CategoryID)
	}
	if input.AmountMinor != nil {
		fields.AmountMinor = *input.AmountMinor
	}
	if input.Currency != nil {
		fields.Currency = strings.ToUpper(strings.TrimSpace(*input.Currency))
	}
	if input.Date != nil {
		fields.Date = input.Date.UTC()
	}
	if input.Note != nil {
		fields.Note = strings.TrimSpace(*input.Note)
	}
	return fields
}

// inclusiveEndDate expands a date boundary to the final microsecond of that day
// so that date <= queries match all records created up to 23:59:59.999999 on the requested date,
// respecting PostgreSQL microsecond timestamp precision.
func inclusiveEndDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	inclusive := value.AddDate(0, 0, 1).Add(-time.Microsecond)
	return &inclusive
}
