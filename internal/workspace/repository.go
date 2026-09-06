package workspace

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/BounkhongDev/bkgo/contract"
)

type workspaceRepository struct {
	db contract.ORM
}

func NewWorkspaceRepository(db contract.ORM) WorkspaceRepository {
	return &workspaceRepository{db: db}
}

func (r *workspaceRepository) Create(ctx context.Context, ws *Workspace) error {
	return r.db.Session(ctx).Create(ws).Error
}

func (r *workspaceRepository) FindByID(ctx context.Context, id string) (*Workspace, error) {
	var entity Workspace
	if err := r.db.Session(ctx).First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	return &entity, nil
}

func (r *workspaceRepository) FindByOwnerID(ctx context.Context, ownerID string) ([]*Workspace, error) {
	var list []*Workspace
	if err := r.db.Session(ctx).Where("owner_id = ?", ownerID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
