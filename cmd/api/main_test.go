package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/hash"

	"github.com/Got17/personal-finance-api/internal/user"
	"github.com/Got17/personal-finance-api/internal/workspace"
)

func TestHealthEndpoint(t *testing.T) {
	app := newApp("personal-finance-api")

	response, err := app.Test(httptest.NewRequest("GET", "/v1/health", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Status string `json:"status"`
			App    string `json:"app"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || body.Data.Status != "ok" || body.Data.App != "personal-finance-api" {
		t.Fatalf("body = %#v, want success with status ok and the configured app name", body)
	}
}

func TestConfiguredApp_SignInRouteIssuesSession(t *testing.T) {
	passwordHash, err := hash.Password("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	token := jwt.New(config.JWT{Secret: "test-secret"})
	userRepo := &mainTestUserRepo{user: &user.User{
		ID:           "user-123",
		Email:        "owner@example.com",
		PasswordHash: passwordHash,
	}}
	uHandler := user.NewUserHandler(user.NewUserUsecase(userRepo, token))
	wHandler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(&mainTestWorkspaceRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, token)

	request := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewBufferString(`{"email":"owner@example.com","password":"correct horse battery staple"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}

func TestConfiguredApp_SignUpAndWorkspaceAccess(t *testing.T) {
	token := jwt.New(config.JWT{Secret: "test-secret"})
	wsRepo := &mainTestWorkspaceRepo{
		workspaces: make(map[string]*workspace.Workspace),
	}
	uRepo := &mainTestUserRepo{
		wsRepo: wsRepo,
	}

	uHandler := user.NewUserHandler(user.NewUserUsecase(uRepo, token))
	wHandler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(wsRepo))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, token)

	// 1. Signup
	signUpReq := httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewBufferString(`{"email":"newowner@example.com","password":"securepassword123","workspace_name":"Private Vault"}`))
	signUpReq.Header.Set("Content-Type", "application/json")
	signUpResp, err := app.Test(signUpReq, 5000)
	if err != nil {
		t.Fatalf("signup request failed: %v", err)
	}
	defer signUpResp.Body.Close()

	if signUpResp.StatusCode != 201 {
		t.Fatalf("signup status = %d, want 201", signUpResp.StatusCode)
	}

	var signUpBody struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		} `json:"data"`
	}
	if err := json.NewDecoder(signUpResp.Body).Decode(&signUpBody); err != nil {
		t.Fatalf("decode signup response: %v", err)
	}

	// 2. Fetch authenticated user's workspace
	wsReq := httptest.NewRequest("GET", "/v1/workspaces", nil)
	wsReq.Header.Set("Authorization", "Bearer "+signUpBody.Data.AccessToken)
	wsResp, err := app.Test(wsReq, 5000)
	if err != nil {
		t.Fatalf("get workspace request failed: %v", err)
	}
	defer wsResp.Body.Close()

	if wsResp.StatusCode != 200 {
		t.Fatalf("get workspace status = %d, want 200", wsResp.StatusCode)
	}

	var wsListBody struct {
		Success bool                  `json:"success"`
		Data    []workspace.Workspace `json:"data"`
	}
	if err := json.NewDecoder(wsResp.Body).Decode(&wsListBody); err != nil {
		t.Fatalf("decode workspace list response: %v", err)
	}

	if len(wsListBody.Data) != 1 || wsListBody.Data[0].Name != "Private Vault" {
		t.Fatalf("workspace list = %#v, want exactly 1 workspace named Private Vault", wsListBody.Data)
	}
}

type mainTestUserRepo struct {
	user   *user.User
	wsRepo *mainTestWorkspaceRepo
}

func (r *mainTestUserRepo) FindByEmail(_ context.Context, email string) (*user.User, error) {
	if r.user != nil && r.user.Email == email {
		return r.user, nil
	}
	return nil, user.ErrUserNotFound
}

func (r *mainTestUserRepo) CreateWithWorkspace(_ context.Context, u *user.User, workspaceName string) (*workspace.Workspace, error) {
	r.user = u
	ws := &workspace.Workspace{
		ID:        "ws-auto-1",
		Name:      workspaceName,
		OwnerID:   u.ID,
		CreatedAt: time.Now(),
	}
	if r.wsRepo != nil {
		if r.wsRepo.workspaces == nil {
			r.wsRepo.workspaces = make(map[string]*workspace.Workspace)
		}
		r.wsRepo.workspaces[ws.ID] = ws
	}
	return ws, nil
}

type mainTestWorkspaceRepo struct {
	workspaces map[string]*workspace.Workspace
}

func (m *mainTestWorkspaceRepo) Create(_ context.Context, ws *workspace.Workspace) error {
	if m.workspaces == nil {
		m.workspaces = make(map[string]*workspace.Workspace)
	}
	m.workspaces[ws.ID] = ws
	return nil
}

func (m *mainTestWorkspaceRepo) FindByID(_ context.Context, id string) (*workspace.Workspace, error) {
	ws, ok := m.workspaces[id]
	if !ok {
		return nil, workspace.ErrWorkspaceNotFound
	}
	return ws, nil
}

func (m *mainTestWorkspaceRepo) FindByOwnerID(_ context.Context, ownerID string) ([]*workspace.Workspace, error) {
	var list []*workspace.Workspace
	for _, ws := range m.workspaces {
		if ws.OwnerID == ownerID {
			list = append(list, ws)
		}
	}
	return list, nil
}
