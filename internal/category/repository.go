package category

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/BounkhongDev/bkgo/contract"
)

type categoryRepository struct {
	db contract.ORM
}

func NewCategoryRepository(db contract.ORM) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, entity *Category) error {
	return r.db.Session(ctx).Create(entity).Error
}

func (r *categoryRepository) FindByUserID(ctx context.Context, userID string) ([]*Category, error) {
	var list []*Category
	if err := r.db.Session(ctx).Where("user_id = ?", userID).Order("created_at asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *categoryRepository) FindByID(ctx context.Context, id string) (*Category, error) {
	var entity Category
	if err := r.db.Session(ctx).First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &entity, nil
}

func (r *categoryRepository) Update(ctx context.Context, entity *Category) error {
	return r.db.Session(ctx).Save(entity).Error
}
