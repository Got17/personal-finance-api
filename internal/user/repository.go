package user

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/BounkhongDev/bkgo/contract"
	"github.com/Got17/personal-finance-api/internal/workspace"
)

type userRepository struct {
	db contract.ORM
}

func NewUserRepository(db contract.ORM) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var entity User
	if err := r.db.Session(ctx).First(&entity, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &entity, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*User, error) {
	var entity User
	if err := r.db.Session(ctx).First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &entity, nil
}

func (r *userRepository) UpdateBaseCurrency(ctx context.Context, id string, currency string) (*User, error) {
	var entity User
	db := r.db.Session(ctx)
	if err := db.First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err := db.Model(&entity).Update("base_currency", currency).Error; err != nil {
		return nil, err
	}
	entity.BaseCurrency = currency
	return &entity, nil
}

func (r *userRepository) CreateWithWorkspace(ctx context.Context, u *User, workspaceName string) (*workspace.Workspace, error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	if u.BaseCurrency == "" {
		u.BaseCurrency = "USD"
	}

	ws := workspace.Workspace{
		ID:      uuid.NewString(),
		Name:    workspaceName,
		OwnerID: u.ID,
	}

	err := r.db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		if err := tx.Create(&ws).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	return &ws, nil
}
