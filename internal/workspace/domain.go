package workspace

import (
	"context"
	"errors"
	"time"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrAccessDenied      = errors.New("access denied to workspace")
)

// Workspace is the core domain entity for a user's isolated workspace.
type Workspace struct {
	ID        string    `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name      string    `json:"name"       gorm:"not null"`
	OwnerID   string    `json:"owner_id"   gorm:"type:uuid;not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the PostgreSQL table name.
func (Workspace) TableName() string { return "workspaces" }

// WorkspaceRepository defines the persistence contract for Workspace operations.
type WorkspaceRepository interface {
	Create(ctx context.Context, ws *Workspace) error
	FindByID(ctx context.Context, id string) (*Workspace, error)
	FindByOwnerID(ctx context.Context, ownerID string) ([]*Workspace, error)
}

// WorkspaceUsecase defines the business rules for Workspace operations.
type WorkspaceUsecase interface {
	GetWorkspace(ctx context.Context, userID string, workspaceID string) (*Workspace, error)
	ListWorkspaces(ctx context.Context, userID string) ([]*Workspace, error)
}
