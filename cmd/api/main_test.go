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

	"github.com/Got17/personal-finance-api/internal/account"
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
	acctHandler := account.NewAccountHandler(account.NewAccountUsecase(&mainTestAccountRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, token)

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
	acctHandler := account.NewAccountHandler(account.NewAccountUsecase(&mainTestAccountRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, token)

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

func TestConfiguredApp_CurrentUserAndWorkspaceIsolation(t *testing.T) {
	token := jwt.New(config.JWT{Secret: "test-secret"})
	userRepo := &multiUserRepoMock{
		users: map[string]*user.User{
			"user-1": {ID: "user-1", Email: "user1@example.com"},
			"user-2": {ID: "user-2", Email: "user2@example.com"},
		},
	}
	wsRepo := &mainTestWorkspaceRepo{
		workspaces: map[string]*workspace.Workspace{
			"ws-1": {ID: "ws-1", Name: "User 1 Vault", OwnerID: "user-1"},
			"ws-2": {ID: "ws-2", Name: "User 2 Vault", OwnerID: "user-2"},
		},
	}

	uHandler := user.NewUserHandler(user.NewUserUsecase(userRepo, token))
	wHandler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(wsRepo))
	acctHandler := account.NewAccountHandler(account.NewAccountUsecase(&mainTestAccountRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, token)

	// User 1 Token
	token1, _ := token.Sign(map[string]any{"sub": "user-1"}, time.Hour)
	// User 2 Token
	token2, _ := token.Sign(map[string]any{"sub": "user-2"}, time.Hour)

	// 1. User 1 retrieves current user identity
	reqMe1 := httptest.NewRequest("GET", "/v1/users/me", nil)
	reqMe1.Header.Set("Authorization", "Bearer "+token1)
	respMe1, err := app.Test(reqMe1, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respMe1.Body.Close()

	if respMe1.StatusCode != 200 {
		t.Fatalf("me status = %d, want 200", respMe1.StatusCode)
	}
	var bodyMe1 struct {
		Success bool      `json:"success"`
		Data    user.User `json:"data"`
	}
	if err := json.NewDecoder(respMe1.Body).Decode(&bodyMe1); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if bodyMe1.Data.ID != "user-1" || bodyMe1.Data.Email != "user1@example.com" {
		t.Fatalf("unexpected user 1 profile: %#v", bodyMe1.Data)
	}

	// 2. User 1 tries to retrieve User 2's workspace (ws-2) -> must be Forbidden (403)
	reqForbidden := httptest.NewRequest("GET", "/v1/workspaces/ws-2", nil)
	reqForbidden.Header.Set("Authorization", "Bearer "+token1)
	respForbidden, err := app.Test(reqForbidden, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respForbidden.Body.Close()

	if respForbidden.StatusCode != 403 {
		t.Fatalf("status = %d, want 403 Forbidden for cross-user workspace access", respForbidden.StatusCode)
	}

	// 3. User 2 retrieves current user identity
	reqMe2 := httptest.NewRequest("GET", "/v1/users/me", nil)
	reqMe2.Header.Set("Authorization", "Bearer "+token2)
	respMe2, err := app.Test(reqMe2, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respMe2.Body.Close()

	if respMe2.StatusCode != 200 {
		t.Fatalf("me status = %d, want 200", respMe2.StatusCode)
	}
	var bodyMe2 struct {
		Success bool      `json:"success"`
		Data    user.User `json:"data"`
	}
	if err := json.NewDecoder(respMe2.Body).Decode(&bodyMe2); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if bodyMe2.Data.ID != "user-2" || bodyMe2.Data.Email != "user2@example.com" {
		t.Fatalf("unexpected user 2 profile: %#v", bodyMe2.Data)
	}

	// 4. User 2 tries to retrieve User 1's workspace (ws-1) -> must be Forbidden (403)
	reqForbidden2 := httptest.NewRequest("GET", "/v1/workspaces/ws-1", nil)
	reqForbidden2.Header.Set("Authorization", "Bearer "+token2)
	respForbidden2, err := app.Test(reqForbidden2, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respForbidden2.Body.Close()

	if respForbidden2.StatusCode != 403 {
		t.Fatalf("status = %d, want 403 Forbidden for cross-user workspace access", respForbidden2.StatusCode)
	}
}

func TestConfiguredApp_CreateAndListAccounts(t *testing.T) {
	token := jwt.New(config.JWT{Secret: "test-secret"})
	userRepo := &multiUserRepoMock{
		users: map[string]*user.User{
			"user-1": {ID: "user-1", Email: "user1@example.com"},
		},
	}
	wsRepo := &mainTestWorkspaceRepo{}
	acctRepo := &mainTestAccountRepo{}

	uHandler := user.NewUserHandler(user.NewUserUsecase(userRepo, token))
	wHandler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(wsRepo))
	acctHandler := account.NewAccountHandler(account.NewAccountUsecase(acctRepo))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, token)

	token1, _ := token.Sign(map[string]any{"sub": "user-1"}, time.Hour)

	// Create account
	createReq := httptest.NewRequest("POST", "/v1/accounts", bytes.NewBufferString(`{"name":"Main Checking","type":"checking","currency":"USD","description":"Primary account"}`))
	createReq.Header.Set("Authorization", "Bearer "+token1)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq, 5000)
	if err != nil {
		t.Fatalf("create account request failed: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != 201 {
		t.Fatalf("create account status = %d, want 201", createResp.StatusCode)
	}

	// List accounts
	listReq := httptest.NewRequest("GET", "/v1/accounts", nil)
	listReq.Header.Set("Authorization", "Bearer "+token1)
	listResp, err := app.Test(listReq, 5000)
	if err != nil {
		t.Fatalf("list accounts request failed: %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != 200 {
		t.Fatalf("list accounts status = %d, want 200", listResp.StatusCode)
	}

	var listBody struct {
		Success bool              `json:"success"`
		Data    []account.Account `json:"data"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode account list response: %v", err)
	}

	if len(listBody.Data) != 1 || listBody.Data[0].Name != "Main Checking" {
		t.Fatalf("account list = %#v, want 1 account named Main Checking", listBody.Data)
	}
}

type mainTestAccountRepo struct {
	accounts map[string]*account.Account
}

func (m *mainTestAccountRepo) Create(_ context.Context, entity *account.Account) error {
	if m.accounts == nil {
		m.accounts = make(map[string]*account.Account)
	}
	m.accounts[entity.ID] = entity
	return nil
}

func (m *mainTestAccountRepo) FindByUserID(_ context.Context, userID string) ([]*account.Account, error) {
	var list []*account.Account
	for _, acct := range m.accounts {
		if acct.UserID == userID {
			list = append(list, acct)
		}
	}
	return list, nil
}

func (m *mainTestAccountRepo) FindByID(_ context.Context, id string) (*account.Account, error) {
	acct, ok := m.accounts[id]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return acct, nil
}

type multiUserRepoMock struct {
	users map[string]*user.User
}

func (m *multiUserRepoMock) FindByEmail(_ context.Context, email string) (*user.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *multiUserRepoMock) FindByID(_ context.Context, id string) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}

func (m *multiUserRepoMock) UpdateBaseCurrency(_ context.Context, id string, currency string) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	u.BaseCurrency = currency
	return u, nil
}

func (m *multiUserRepoMock) CreateWithWorkspace(_ context.Context, u *user.User, workspaceName string) (*workspace.Workspace, error) {
	if m.users == nil {
		m.users = make(map[string]*user.User)
	}
	m.users[u.ID] = u
	return &workspace.Workspace{ID: "ws-new", Name: workspaceName, OwnerID: u.ID}, nil
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

func (r *mainTestUserRepo) FindByID(_ context.Context, id string) (*user.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, user.ErrUserNotFound
}

func (r *mainTestUserRepo) UpdateBaseCurrency(_ context.Context, id string, currency string) (*user.User, error) {
	if r.user != nil && r.user.ID == id {
		r.user.BaseCurrency = currency
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
