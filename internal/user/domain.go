package user

import (
	"context"
	"errors"
	"time"

	"github.com/Got17/personal-finance-api/internal/workspace"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already in use")
)

// User is the core domain entity.
type User struct {
	ID           string    `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email        string    `json:"email"      gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-"          gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName sets the PostgreSQL table name.
func (User) TableName() string { return "users" }

// UserRepository is the port (interface) for User persistence.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	CreateWithWorkspace(ctx context.Context, u *User, workspaceName string) (*workspace.Workspace, error)
}
