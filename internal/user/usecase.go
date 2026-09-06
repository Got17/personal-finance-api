package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/hash"
	"github.com/BounkhongDev/bkgo/validator"

	"github.com/Got17/personal-finance-api/internal/messages"
)

// UserUsecase defines the business operations for User.
type UserUsecase interface {
	SignIn(ctx context.Context, input *SignInInput) (*Session, error)
	SignUp(ctx context.Context, input *SignUpInput) (*Session, error)
	GetCurrentUser(ctx context.Context, userID string) (*User, error)
	GetPreferences(ctx context.Context, userID string) (*UserPreferences, error)
	UpdatePreferences(ctx context.Context, userID string, input *UpdatePreferencesInput) (*UserPreferences, error)
	UpdateUser(ctx context.Context, userID string, input *UpdatePreferencesInput) (*User, error)
}

type SignInInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SignUpInput struct {
	Email         string `json:"email"          validate:"required,email"`
	Password      string `json:"password"       validate:"required,min=8"`
	WorkspaceName string `json:"workspace_name" validate:"omitempty,max=100"`
}

type UpdatePreferencesInput struct {
	BaseCurrency string `json:"base_currency" validate:"required"`
}

type UserPreferences struct {
	BaseCurrency string `json:"base_currency"`
}

type Session struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

var validCurrencies = map[string]bool{
	"AED": true, "AFN": true, "ALL": true, "AMD": true, "ANG": true, "AOA": true, "ARS": true, "AUD": true,
	"AWG": true, "AZN": true, "BAM": true, "BBD": true, "BDT": true, "BGN": true, "BHD": true, "BIF": true,
	"BMD": true, "BND": true, "BOB": true, "BRL": true, "BSD": true, "BTN": true, "BWP": true, "BYN": true,
	"BZD": true, "CAD": true, "CDF": true, "CHF": true, "CLP": true, "CNY": true, "COP": true, "CRC": true,
	"CUP": true, "CVE": true, "CZK": true, "DJF": true, "DKK": true, "DOP": true, "DZD": true, "EGP": true,
	"ERN": true, "ETB": true, "EUR": true, "FJD": true, "FKP": true, "GBP": true, "GEL": true, "GHS": true,
	"GIP": true, "GMD": true, "GNF": true, "GTQ": true, "GYD": true, "HKD": true, "HNL": true, "HRK": true,
	"HTG": true, "HUF": true, "IDR": true, "ILS": true, "INR": true, "IQD": true, "IRR": true, "ISK": true,
	"JMD": true, "JOD": true, "JPY": true, "KES": true, "KGS": true, "KHR": true, "KMF": true, "KPW": true,
	"KRW": true, "KWD": true, "KYD": true, "KZT": true, "LAK": true, "LBP": true, "LKR": true, "LRD": true,
	"LSL": true, "LYD": true, "MAD": true, "MDL": true, "MGA": true, "MKD": true, "MMK": true, "MNT": true,
	"MOP": true, "MRU": true, "MUR": true, "MVR": true, "MWK": true, "MXN": true, "MYR": true, "MZN": true,
	"NAD": true, "NGN": true, "NIO": true, "NOK": true, "NPR": true, "NZD": true, "OMR": true, "PAB": true,
	"PEN": true, "PGK": true, "PHP": true, "PKR": true, "PLN": true, "PYG": true, "QAR": true, "RON": true,
	"RSD": true, "RUB": true, "RWF": true, "SAR": true, "SBD": true, "SCR": true, "SDG": true, "SEK": true,
	"SGD": true, "SHP": true, "SLE": true, "SOS": true, "SRD": true, "SSP": true, "STN": true, "SYP": true,
	"SZL": true, "THB": true, "TJS": true, "TMT": true, "TND": true, "TOP": true, "TRY": true, "TTD": true,
	"TWD": true, "TZS": true, "UAH": true, "UGX": true, "USD": true, "UYU": true, "UZS": true, "VES": true,
	"VND": true, "VUV": true, "WST": true, "XAF": true, "XCD": true, "XOF": true, "XPF": true, "YER": true,
	"ZAR": true, "ZMW": true, "ZWG": true,
}

// IsValidISO4217 checks if the string is a valid ISO 4217 currency code.
func IsValidISO4217(code string) bool {
	return validCurrencies[strings.ToUpper(strings.TrimSpace(code))]
}

type userUsecase struct {
	repo  UserRepository
	token contract.Token
}

func NewUserUsecase(repo UserRepository, token contract.Token) UserUsecase {
	return &userUsecase{repo: repo, token: token}
}

// dummyHash is used to prevent timing side-channel user enumeration when a user is not found.
var dummyHash, _ = hash.Password("bkgo-timing-mitigation-dummy-password")

func (u *userUsecase) SignIn(ctx context.Context, input *SignInInput) (*Session, error) {
	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}

	entity, err := u.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = hash.CheckPassword(input.Password, dummyHash)
			return nil, errs.Unauthorized(messages.MsgInvalidCredentials)
		}
		return nil, err
	}
	if !hash.CheckPassword(input.Password, entity.PasswordHash) {
		return nil, errs.Unauthorized(messages.MsgInvalidCredentials)
	}

	accessToken, err := u.token.Sign(contract.Claims{"sub": entity.ID}, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &Session{AccessToken: accessToken, TokenType: "Bearer"}, nil
}

func (u *userUsecase) SignUp(ctx context.Context, input *SignUpInput) (*Session, error) {
	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}

	existing, err := u.repo.FindByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errs.Conflict(messages.MsgEmailAlreadyExists)
	}

	passwordHash, err := hash.Password(input.Password)
	if err != nil {
		return nil, err
	}

	wsName := strings.TrimSpace(input.WorkspaceName)
	if wsName == "" {
		wsName = "Personal Workspace"
	}

	newUser := &User{
		ID:           uuid.NewString(),
		Email:        input.Email,
		PasswordHash: passwordHash,
		BaseCurrency: "USD",
	}

	if _, err := u.repo.CreateWithWorkspace(ctx, newUser, wsName); err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			return nil, errs.Conflict(messages.MsgEmailAlreadyExists)
		}
		return nil, err
	}

	accessToken, err := u.token.Sign(contract.Claims{"sub": newUser.ID}, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &Session{AccessToken: accessToken, TokenType: "Bearer"}, nil
}

func (u *userUsecase) GetCurrentUser(ctx context.Context, userID string) (*User, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	entity, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, errs.Unauthorized(messages.MsgUserNotFound)
		}
		return nil, err
	}
	if entity.BaseCurrency == "" {
		entity.BaseCurrency = "USD"
	}
	return entity, nil
}

func (u *userUsecase) GetPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	usr, err := u.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserPreferences{BaseCurrency: usr.BaseCurrency}, nil
}

func (u *userUsecase) UpdatePreferences(ctx context.Context, userID string, input *UpdatePreferencesInput) (*UserPreferences, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	if input == nil {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{"base_currency": "base_currency is required"})
	}

	currency := strings.ToUpper(strings.TrimSpace(input.BaseCurrency))
	if !IsValidISO4217(currency) {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{"base_currency": "invalid currency code"})
	}

	updatedUser, err := u.repo.UpdateBaseCurrency(ctx, userID, currency)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, errs.Unauthorized(messages.MsgUserNotFound)
		}
		return nil, err
	}

	return &UserPreferences{BaseCurrency: updatedUser.BaseCurrency}, nil
}

func (u *userUsecase) UpdateUser(ctx context.Context, userID string, input *UpdatePreferencesInput) (*User, error) {
	if _, err := u.UpdatePreferences(ctx, userID, input); err != nil {
		return nil, err
	}
	return u.GetCurrentUser(ctx, userID)
}
