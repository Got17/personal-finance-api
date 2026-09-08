package account

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/BounkhongDev/bkgo/contract"
)

type accountRepository struct {
	db contract.ORM
}

func NewAccountRepository(db contract.ORM) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(ctx context.Context, entity *Account) error {
	return r.db.Session(ctx).Create(entity).Error
}

func (r *accountRepository) FindByUserID(ctx context.Context, userID string) ([]*Account, error) {
	var list []*Account
	if err := r.db.Session(ctx).Order("created_at desc").Find(&list, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *accountRepository) FindByID(ctx context.Context, id string) (*Account, error) {
	var entity Account
	if err := r.db.Session(ctx).First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &entity, nil
}

func (r *accountRepository) Update(ctx context.Context, entity *Account) error {
	return r.db.Session(ctx).Save(entity).Error
}
