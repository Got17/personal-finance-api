package workspace_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/workspace"
)

type mockWorkspaceRepo struct {
	workspaces map[string]*workspace.Workspace
}

func (m *mockWorkspaceRepo) Create(_ context.Context, ws *workspace.Workspace) error {
	if m.workspaces == nil {
		m.workspaces = make(map[string]*workspace.Workspace)
	}
	m.workspaces[ws.ID] = ws
	return nil
}

func (m *mockWorkspaceRepo) FindByID(_ context.Context, id string) (*workspace.Workspace, error) {
	ws, ok := m.workspaces[id]
	if !ok {
		return nil, workspace.ErrWorkspaceNotFound
	}
	return ws, nil
}

func (m *mockWorkspaceRepo) FindByOwnerID(_ context.Context, ownerID string) ([]*workspace.Workspace, error) {
	var res []*workspace.Workspace
	for _, ws := range m.workspaces {
		if ws.OwnerID == ownerID {
			res = append(res, ws)
		}
	}
	return res, nil
}

func TestGetWorkspace_OwnerCanAccessOwnWorkspace(t *testing.T) {
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockWorkspaceRepo{
		workspaces: map[string]*workspace.Workspace{
			"ws-1": {ID: "ws-1", Name: "Personal Workspace", OwnerID: "user-123", CreatedAt: time.Now()},
		},
	}

	app := fiber.New()
	handler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(repo))
	api := app.Group("/v1", middleware.JWT(tokenAdapter))
	handler.RegisterRoutes(api)

	authToken, _ := tokenAdapter.Sign(map[string]any{"sub": "user-123"}, time.Hour)

	req := httptest.NewRequest("GET", "/v1/workspaces/ws-1", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Success bool                `json:"success"`
		Data    workspace.Workspace `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !body.Success || body.Data.ID != "ws-1" || body.Data.OwnerID != "user-123" {
		t.Fatalf("unexpected response body: %#v", body)
	}
}

func TestGetWorkspace_NonOwnerAccessIsForbidden(t *testing.T) {
	tokenAdapter := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockWorkspaceRepo{
		workspaces: map[string]*workspace.Workspace{
			"ws-1": {ID: "ws-1", Name: "Private Vault", OwnerID: "user-owner", CreatedAt: time.Now()},
		},
	}

	app := fiber.New()
	handler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(repo))
	api := app.Group("/v1", middleware.JWT(tokenAdapter))
	handler.RegisterRoutes(api)

	// User-other trying to access user-owner's workspace
	otherToken, _ := tokenAdapter.Sign(map[string]any{"sub": "user-other"}, time.Hour)

	req := httptest.NewRequest("GET", "/v1/workspaces/ws-1", nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Fatalf("status = %d, want 403 Forbidden for non-owner access", resp.StatusCode)
	}

	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Success || body.Error != "FORBIDDEN" {
		t.Fatalf("unexpected body = %#v, want FORBIDDEN", body)
	}
}
