package financialrecord

import (
	"context"
	"errors"
	"time"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/fxquote"
)

var ErrFinancialRecordNotFound = errors.New("financial record not found")

type Kind string

const (
	KindIncome   Kind = "income"
	KindExpense  Kind = "expense"
	KindTransfer Kind = "transfer"
)

var supportedKinds = map[Kind]bool{KindIncome: true, KindExpense: true, KindTransfer: true}

type FinancialRecord struct {
	ID                     string                     `json:"id"                                gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID                 string                     `json:"user_id"                           gorm:"type:uuid;not null;index"`
	Kind                   Kind                       `json:"kind"                              gorm:"not null"`
	AccountID              string                     `json:"account_id"                        gorm:"type:uuid;not null;index"`
	DestinationAccountID   *string                    `json:"destination_account_id,omitempty"  gorm:"type:uuid;index"`
	CategoryID             *string                    `json:"category_id,omitempty"             gorm:"type:uuid;index"`
	AmountMinor            int64                      `json:"amount_minor"                      gorm:"not null"`
	DestinationAmountMinor *int64                     `json:"destination_amount_minor,omitempty"`
	Currency               string                     `json:"currency"                          gorm:"not null"`
	DestinationCurrency    *string                    `json:"destination_currency,omitempty"`
	Date                   time.Time                  `json:"date"                              gorm:"not null;index"`
	Note                   string                     `json:"note"                              gorm:"type:text"`
	HistoricalFXQuoteID    *string                    `json:"historical_fx_quote_id,omitempty"  gorm:"type:uuid;index"`
	TransferFeeRecordID    *string                    `json:"transfer_fee_record_id,omitempty"  gorm:"type:uuid;index"`
	LinkedTransferID       *string                    `json:"linked_transfer_id,omitempty"      gorm:"type:uuid;index"`
	IsActive               bool                       `json:"is_active"                         gorm:"not null;default:true;index"`
	HistoricalFXQuote      *fxquote.HistoricalFXQuote `json:"historical_fx_quote,omitempty"     gorm:"foreignKey:HistoricalFXQuoteID"`
	CreatedAt              time.Time                  `json:"created_at"                        gorm:"autoCreateTime"`
	UpdatedAt              time.Time                  `json:"updated_at"                        gorm:"autoUpdateTime"`
}

func (FinancialRecord) TableName() string { return "financial_records" }

type ListFilter struct {
	StartDate       *time.Time
	EndDate         *time.Time
	Kind            Kind
	AccountID       string
	CategoryID      string
	IncludeArchived bool
}

// AccountReader decouples record creation and updates from account persistence details,
// enforcing that account ownership and access validation remain localized in the account module.
type AccountReader interface {
	GetAccount(ctx context.Context, userID string, accountID string) (*account.Account, error)
}

// CategoryReader decouples record creation and updates from category persistence details,
// enforcing that category ownership and access validation remain localized in the category module.
type CategoryReader interface {
	GetCategory(ctx context.Context, userID string, categoryID string) (*category.Category, error)
}

type FinancialRecordRepository interface {
	Create(ctx context.Context, record *FinancialRecord) error
	FindByID(ctx context.Context, id string) (*FinancialRecord, error)
	FindByUserID(ctx context.Context, userID string, filter ListFilter) ([]*FinancialRecord, error)
	Update(ctx context.Context, record *FinancialRecord) error
}

type FinancialRecordUsecase interface {
	CreateFinancialRecord(ctx context.Context, userID string, input *CreateFinancialRecordInput) (*FinancialRecord, error)
	GetFinancialRecord(ctx context.Context, userID string, recordID string) (*FinancialRecord, error)
	ListFinancialRecords(ctx context.Context, userID string, filter ListFilter) ([]*FinancialRecord, error)
	UpdateFinancialRecord(ctx context.Context, userID string, recordID string, input *UpdateFinancialRecordInput) (*FinancialRecord, error)
	ArchiveFinancialRecord(ctx context.Context, userID string, recordID string) (*FinancialRecord, error)
}

func IsValidKind(value string) bool { return supportedKinds[Kind(value)] }
