package financialrecord

import (
	"context"
	"errors"
	"time"
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
	Kind        Kind      `json:"kind"         gorm:"type:financial_record_kind;not null"`
	AccountID   string    `json:"account_id"   gorm:"type:uuid;not null;index"`
	CategoryID  string    `json:"category_id"  gorm:"type:uuid;not null;index"`
	AmountMinor int64     `json:"amount_minor" gorm:"not null"`
	Currency    string    `json:"currency"     gorm:"not null"`
	Date        time.Time `json:"date"         gorm:"type:date;not null;index"`
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

type FinancialRecordRepository interface {
	Create(ctx context.Context, record *FinancialRecord) error
	FindByID(ctx context.Context, id string) (*FinancialRecord, error)
	FindByUserID(ctx context.Context, userID string, filter ListFilter) ([]*FinancialRecord, error)
}

type FinancialRecordUsecase interface {
	CreateFinancialRecord(ctx context.Context, userID string, input *CreateFinancialRecordInput) (*FinancialRecord, error)
	GetFinancialRecord(ctx context.Context, userID string, recordID string) (*FinancialRecord, error)
	ListFinancialRecords(ctx context.Context, userID string, filter ListFilter) ([]*FinancialRecord, error)
}

func IsValidKind(value string) bool { return supportedKinds[Kind(value)] }
