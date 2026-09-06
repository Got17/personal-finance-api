package account

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/validator"

	"github.com/Got17/personal-finance-api/internal/messages"
	"github.com/Got17/personal-finance-api/internal/user"
)

type CreateAccountInput struct {
	Name        string `json:"name"        validate:"required,min=1,max=100"`
	Type        string `json:"type"        validate:"required"`
	Currency    string `json:"currency"    validate:"required"`
	Description string `json:"description" validate:"omitempty,max=500"`
	IsActive    *bool  `json:"is_active"   validate:"omitempty"`
}

type accountUsecase struct {
	repo AccountRepository
}

func NewAccountUsecase(repo AccountRepository) AccountUsecase {
	return &accountUsecase{repo: repo}
}

// IsValidAccountType checks whether the given type is a supported account type.
func IsValidAccountType(acctType string) bool {
	return SupportedAccountTypes[strings.ToLower(strings.TrimSpace(acctType))]
}

func (u *accountUsecase) CreateAccount(ctx context.Context, userID string, input *CreateAccountInput) (*Account, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}

	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}

	acctType := strings.ToLower(strings.TrimSpace(input.Type))
	if !IsValidAccountType(acctType) {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{
			"type": "unsupported account type",
		})
	}

	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if !user.IsValidISO4217(currency) {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{
			"currency": "invalid currency code",
		})
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	entity := &Account{
		ID:          uuid.NewString(),
		UserID:      userID,
		Name:        strings.TrimSpace(input.Name),
		Type:        acctType,
		Currency:    currency,
		Description: strings.TrimSpace(input.Description),
		IsActive:    isActive,
	}

	if err := u.repo.Create(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func (u *accountUsecase) ListAccounts(ctx context.Context, userID string) ([]*Account, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}

	accounts, err := u.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if accounts == nil {
		accounts = []*Account{}
	}

	return accounts, nil
}
