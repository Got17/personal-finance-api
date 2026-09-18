package transfer

import (
	"context"
	"errors"
	"time"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/financialrecord"
	"github.com/Got17/personal-finance-api/internal/fxquote"
	"github.com/Got17/personal-finance-api/internal/messages"
)

var (
	ErrTransferNotFound              = errors.New("transfer not found")
	ErrTransferAccessDenied          = errors.New(messages.MsgTransferAccessDenied)
	ErrInsufficientAccountBalance    = errors.New(messages.MsgInsufficientAccountBalance)
	ErrInsufficientFeeAccountBalance = errors.New(messages.MsgInsufficientFeeAccountBalance)
)

// Transfer represents a money movement between two distinct owned accounts.
// Stored in the financial_records table with kind = 'transfer'.
type Transfer struct {
	ID                     string                     `json:"id"                      gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID                 string                     `json:"user_id"                 gorm:"type:uuid;not null;index"`
	Kind                   string                     `json:"kind"                    gorm:"not null;default:'transfer'"`
	SourceAccountID        string                     `json:"source_account_id"       gorm:"column:account_id;type:uuid;not null;index"`
	DestinationAccountID   string                     `json:"destination_account_id"  gorm:"column:destination_account_id;type:uuid;not null;index"`
	SourceAmountMinor      int64                      `json:"source_amount_minor"     gorm:"column:amount_minor;not null"`
	DestinationAmountMinor int64                      `json:"destination_amount_minor" gorm:"column:destination_amount_minor;not null"`
	SourceCurrency         string                     `json:"source_currency"         gorm:"column:currency;not null"`
	DestinationCurrency    string                     `json:"destination_currency"    gorm:"column:destination_currency;not null"`
	Date                   time.Time                  `json:"date"                    gorm:"not null;index"`
	Note                   string                     `json:"note"                    gorm:"type:text"`
	HistoricalFXQuoteID    *string                    `json:"historical_fx_quote_id,omitempty"  gorm:"type:uuid;index"`
	TransferFeeRecordID    *string                    `json:"transfer_fee_record_id,omitempty"  gorm:"type:uuid;index"`
	IsActive               bool                       `json:"is_active"               gorm:"not null;default:true;index"`
	HistoricalFXQuote      *fxquote.HistoricalFXQuote `json:"historical_fx_quote,omitempty"     gorm:"->;foreignKey:HistoricalFXQuoteID"`
	TransferFee            *TransferFeeResult         `json:"transfer_fee,omitempty"   gorm:"-"`
	CreatedAt              time.Time                  `json:"created_at"              gorm:"autoCreateTime"`
	UpdatedAt              time.Time                  `json:"updated_at"              gorm:"autoUpdateTime"`
}

func (Transfer) TableName() string { return "financial_records" }

type TransferFeeInput struct {
	AccountID   string `json:"account_id"   validate:"required"`
	CategoryID  string `json:"category_id"  validate:"required"`
	AmountMinor int64  `json:"amount_minor" validate:"required,gt=0"`
	Note        string `json:"note"         validate:"omitempty,max=1000"`
}

type TransferFeeResult struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Kind        string    `json:"kind"`
	AccountID   string    `json:"account_id"`
	CategoryID  string    `json:"category_id"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Date        time.Time `json:"date"`
	Note        string    `json:"note"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTransferInput struct {
	SourceAccountID        string            `json:"source_account_id"        validate:"required"`
	DestinationAccountID   string            `json:"destination_account_id"   validate:"required"`
	SourceAmountMinor      int64             `json:"source_amount_minor"      validate:"required,gt=0"`
	DestinationAmountMinor *int64            `json:"destination_amount_minor" validate:"omitempty,gt=0"`
	Date                   time.Time         `json:"date"                     validate:"required"`
	Note                   string            `json:"note"                     validate:"omitempty,max=1000"`
	Rate                   *float64          `json:"rate"                     validate:"omitempty,gt=0"`
	Fee                    *TransferFeeInput `json:"fee"                      validate:"omitempty"`
}

type ListTransferFilter struct {
	StartDate       *time.Time
	EndDate         *time.Time
	AccountID       string
	IncludeArchived bool
}

type AccountReader interface {
	GetAccount(ctx context.Context, userID string, accountID string) (*account.Account, error)
}

type CategoryReader interface {
	GetCategory(ctx context.Context, userID string, categoryID string) (*category.Category, error)
}

type FXRateProvider interface {
	GetRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (float64, error)
}

type TransferRepository interface {
	CreateTransferWithLegsAndFee(ctx context.Context, transfer *Transfer, quote *fxquote.HistoricalFXQuote, fee *financialrecord.FinancialRecord) error
	FindByID(ctx context.Context, id string) (*Transfer, error)
	FindByUserID(ctx context.Context, userID string, filter ListTransferFilter) ([]*Transfer, error)
}

type TransferUsecase interface {
	CreateTransfer(ctx context.Context, userID string, input *CreateTransferInput) (*Transfer, error)
	GetTransfer(ctx context.Context, userID string, transferID string) (*Transfer, error)
	ListTransfers(ctx context.Context, userID string, filter ListTransferFilter) ([]*Transfer, error)
}
