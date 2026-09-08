package account

import (
	"context"
	"errors"
	"time"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrAccessDenied    = errors.New("access denied to account")
)

type AccountType string

const (
	AccountTypeChecking   AccountType = "checking"
	AccountTypeSavings    AccountType = "savings"
	AccountTypeCreditCard AccountType = "credit_card"
	AccountTypeInvestment AccountType = "investment"
	AccountTypeCash       AccountType = "cash"
	AccountTypeLoan       AccountType = "loan"
	AccountTypeOther      AccountType = "other"
)

var SupportedAccountTypes = map[AccountType]bool{
	AccountTypeChecking:   true,
	AccountTypeSavings:    true,
	AccountTypeCreditCard: true,
	AccountTypeInvestment: true,
	AccountTypeCash:       true,
	AccountTypeLoan:       true,
	AccountTypeOther:      true,
}

// Account is the core domain entity.
type Account struct {
	ID          string      `json:"id"          gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID      string      `json:"user_id"     gorm:"type:uuid;not null;index"`
	Name        string      `json:"name"        gorm:"not null"`
	Type        AccountType `json:"type"        gorm:"type:account_type;not null"`
	Currency    string      `json:"currency"    gorm:"not null"`
	Description string      `json:"description" gorm:"type:text"`
	IsActive    bool        `json:"is_active"   gorm:"not null;default:true"`
	CreatedAt   time.Time   `json:"created_at"  gorm:"autoCreateTime"`
	UpdatedAt   time.Time   `json:"updated_at"  gorm:"autoUpdateTime"`
}

// TableName sets the PostgreSQL table name.
func (Account) TableName() string { return "accounts" }

// AccountRepository is the port (interface) for Account persistence.
type AccountRepository interface {
	Create(ctx context.Context, entity *Account) error
	FindByUserID(ctx context.Context, userID string) ([]*Account, error)
	FindByID(ctx context.Context, id string) (*Account, error)
	Update(ctx context.Context, entity *Account) error
}

// AccountUsecase defines the business operations for Account.
type AccountUsecase interface {
	CreateAccount(ctx context.Context, userID string, input *CreateAccountInput) (*Account, error)
	ListAccounts(ctx context.Context, userID string) ([]*Account, error)
	UpdateAccount(ctx context.Context, userID string, accountID string, input *UpdateAccountInput) (*Account, error)
	DeactivateAccount(ctx context.Context, userID string, accountID string) (*Account, error)
}
