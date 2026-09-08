package category

import (
	"context"

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
