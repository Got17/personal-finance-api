package user

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/BounkhongDev/bkgo/contract"
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
