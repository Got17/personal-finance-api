package workspace

import (
	"context"
	"errors"

	"github.com/BounkhongDev/bkgo/errs"

	"github.com/Got17/personal-finance-api/internal/messages"
)

type workspaceUsecase struct {
	repo WorkspaceRepository
}

func NewWorkspaceUsecase(repo WorkspaceRepository) WorkspaceUsecase {
	return &workspaceUsecase{repo: repo}
}

func (u *workspaceUsecase) GetWorkspace(ctx context.Context, userID string, workspaceID string) (*Workspace, error) {
	ws, err := u.repo.FindByID(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			return nil, errs.NotFound(messages.MsgWorkspaceNotFound)
		}
		return nil, err
	}
	// Enforce strict workspace ownership: users may access only their own private workspaces.
	if ws.OwnerID != userID {
		return nil, errs.Forbidden(messages.MsgAccessDenied)
	}
	return ws, nil
}

func (u *workspaceUsecase) ListWorkspaces(ctx context.Context, userID string) ([]*Workspace, error) {
	return u.repo.FindByOwnerID(ctx, userID)
}
