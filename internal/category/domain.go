package category

import (
	"context"
	"errors"
	"time"
)

var ErrCategoryNotFound = errors.New("category not found")

type CategoryType string

const (
	CategoryTypeIncome  CategoryType = "income"
	CategoryTypeExpense CategoryType = "expense"
)

var SupportedCategoryTypes = map[CategoryType]bool{
	CategoryTypeIncome:  true,
	CategoryTypeExpense: true,
}

type Category struct {
	ID        string       `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string       `json:"user_id"    gorm:"type:uuid;not null;index"`
	Name      string       `json:"name"       gorm:"not null"`
	Type      CategoryType `json:"type"       gorm:"not null"`
	IsActive  bool         `json:"is_active"  gorm:"not null;default:true"`
	CreatedAt time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Category) TableName() string { return "categories" }

type CategoryRepository interface {
	Create(ctx context.Context, entity *Category) error
	FindByUserID(ctx context.Context, userID string) ([]*Category, error)
}

type CategoryUsecase interface {
	CreateCategory(ctx context.Context, userID string, input *CreateCategoryInput) (*Category, error)
	ListCategories(ctx context.Context, userID string) ([]*Category, error)
}
