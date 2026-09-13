package financialrecord

import (
	"context"
	"errors"
	"time"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
)

var ErrFinancialRecordNotFound = errors.New("financial record not found")

type Kind string

const (
	KindIncome  Kind = "income"
	KindExpense Kind = "expense"
)

var supportedKinds = map[Kind]bool{KindIncome: true, KindExpense: true}

type FinancialRecord struct {
	ID          string    `json:"id"           gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID      string    `json:"user_id"      gorm:"type:uuid;not null;index"`
	Kind        Kind      `json:"kind"         gorm:"not null"`
	AccountID   string    `json:"account_id"   gorm:"type:uuid;not null;index"`
	CategoryID  string    `json:"category_id"  gorm:"type:uuid;not null;index"`
	AmountMinor int64     `json:"amount_minor" gorm:"not null"`
	Currency    string    `json:"currency"     gorm:"not null"`
	Date        time.Time `json:"date"         gorm:"not null;index"`
	Note        string    `json:"note"         gorm:"type:text"`
	IsActive    bool      `json:"is_active"    gorm:"not null;default:true;index"`
	CreatedAt   time.Time `json:"created_at"   gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at"   gorm:"autoUpdateTime"`
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

// AccountReader is the consumer seam required by financial record for account verification.
type AccountReader interface {
	GetAccount(ctx context.Context, userID string, accountID string) (*account.Account, error)
}

// CategoryReader is the consumer seam required by financial record for category verification.
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

