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
	"github.com/Got17/personal-finance-api/internal/category"
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
	catHandler := category.NewCategoryHandler(category.NewCategoryUsecase(&mainTestCategoryRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, catHandler, token)

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
	catHandler := category.NewCategoryHandler(category.NewCategoryUsecase(&mainTestCategoryRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, catHandler, token)

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
	catHandler := category.NewCategoryHandler(category.NewCategoryUsecase(&mainTestCategoryRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, catHandler, token)

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
	catHandler := category.NewCategoryHandler(category.NewCategoryUsecase(&mainTestCategoryRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, catHandler, token)

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

func TestConfiguredApp_UpdateAndDeactivateAccount(t *testing.T) {
	token := jwt.New(config.JWT{Secret: "test-secret"})
	userRepo := &multiUserRepoMock{
		users: map[string]*user.User{
			"user-1": {ID: "user-1", Email: "user1@example.com"},
			"user-2": {ID: "user-2", Email: "user2@example.com"},
		},
	}
	wsRepo := &mainTestWorkspaceRepo{}
	acctRepo := &mainTestAccountRepo{}

	uHandler := user.NewUserHandler(user.NewUserUsecase(userRepo, token))
	wHandler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(wsRepo))
	acctHandler := account.NewAccountHandler(account.NewAccountUsecase(acctRepo))
	catHandler := category.NewCategoryHandler(category.NewCategoryUsecase(&mainTestCategoryRepo{}))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, catHandler, token)

	token1, _ := token.Sign(map[string]any{"sub": "user-1"}, time.Hour)
	token2, _ := token.Sign(map[string]any{"sub": "user-2"}, time.Hour)

	// 1. Create account as user 1
	createReq := httptest.NewRequest("POST", "/v1/accounts", bytes.NewBufferString(`{"name":"Initial Name","type":"checking","currency":"USD"}`))
	createReq.Header.Set("Authorization", "Bearer "+token1)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq, 5000)
	if err != nil {
		t.Fatalf("create account failed: %v", err)
	}
	defer createResp.Body.Close()

	var createBody struct {
		Data account.Account `json:"data"`
	}
	_ = json.NewDecoder(createResp.Body).Decode(&createBody)
	acctID := createBody.Data.ID

	// 2. User 2 attempts to update User 1's account -> 403 Forbidden
	updateReqForbidden := httptest.NewRequest("PATCH", "/v1/accounts/"+acctID, bytes.NewBufferString(`{"name":"Hacked"}`))
	updateReqForbidden.Header.Set("Authorization", "Bearer "+token2)
	updateReqForbidden.Header.Set("Content-Type", "application/json")
	updateRespForbidden, err := app.Test(updateReqForbidden, 5000)
	if err != nil {
		t.Fatalf("forbidden update request failed: %v", err)
	}
	defer updateRespForbidden.Body.Close()
	if updateRespForbidden.StatusCode != 403 {
		t.Fatalf("status = %d, want 403 Forbidden", updateRespForbidden.StatusCode)
	}

	// 3. User 1 updates their account -> 200 OK
	updateReq := httptest.NewRequest("PATCH", "/v1/accounts/"+acctID, bytes.NewBufferString(`{"name":"Updated Name","description":"New Description"}`))
	updateReq.Header.Set("Authorization", "Bearer "+token1)
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq, 5000)
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	defer updateResp.Body.Close()
	if updateResp.StatusCode != 200 {
		t.Fatalf("update status = %d, want 200 OK", updateResp.StatusCode)
	}

	var updateBody struct {
		Data account.Account `json:"data"`
	}
	_ = json.NewDecoder(updateResp.Body).Decode(&updateBody)
	if updateBody.Data.Name != "Updated Name" || updateBody.Data.Description != "New Description" {
		t.Fatalf("unexpected updated account data: %#v", updateBody.Data)
	}

	// 4. User 1 deactivates their account -> 200 OK
	deactivateReq := httptest.NewRequest("DELETE", "/v1/accounts/"+acctID, nil)
	deactivateReq.Header.Set("Authorization", "Bearer "+token1)
	deactivateResp, err := app.Test(deactivateReq, 5000)
	if err != nil {
		t.Fatalf("deactivate request failed: %v", err)
	}
	defer deactivateResp.Body.Close()
	if deactivateResp.StatusCode != 200 {
		t.Fatalf("deactivate status = %d, want 200 OK", deactivateResp.StatusCode)
	}

	var deactivateBody struct {
		Data account.Account `json:"data"`
	}
	_ = json.NewDecoder(deactivateResp.Body).Decode(&deactivateBody)
	if deactivateBody.Data.IsActive != false {
		t.Fatalf("expected IsActive to be false, got true")
	}
}

func TestConfiguredApp_CreateAndListCategories(t *testing.T) {
	token := jwt.New(config.JWT{Secret: "test-secret"})
	userRepo := &multiUserRepoMock{
		users: map[string]*user.User{
			"user-1": {ID: "user-1", Email: "user1@example.com"},
			"user-2": {ID: "user-2", Email: "user2@example.com"},
		},
	}
	wsRepo := &mainTestWorkspaceRepo{}
	acctRepo := &mainTestAccountRepo{}
	catRepo := &mainTestCategoryRepo{}

	uHandler := user.NewUserHandler(user.NewUserUsecase(userRepo, token))
	wHandler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(wsRepo))
	acctHandler := account.NewAccountHandler(account.NewAccountUsecase(acctRepo))
	catHandler := category.NewCategoryHandler(category.NewCategoryUsecase(catRepo))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, catHandler, token)

	token1, _ := token.Sign(map[string]any{"sub": "user-1"}, time.Hour)
	token2, _ := token.Sign(map[string]any{"sub": "user-2"}, time.Hour)

	// 1. Create Income category for User 1
	incomeReq := httptest.NewRequest("POST", "/v1/categories", bytes.NewBufferString(`{"name":"Salary","type":"income"}`))
	incomeReq.Header.Set("Authorization", "Bearer "+token1)
	incomeReq.Header.Set("Content-Type", "application/json")
	incomeResp, err := app.Test(incomeReq, 5000)
	if err != nil {
		t.Fatalf("create income category failed: %v", err)
	}
	defer incomeResp.Body.Close()
	if incomeResp.StatusCode != 201 {
		t.Fatalf("create income category status = %d, want 201", incomeResp.StatusCode)
	}

	// 2. Create Expense category for User 1
	expenseReq := httptest.NewRequest("POST", "/v1/categories", bytes.NewBufferString(`{"name":"Groceries","type":"expense"}`))
	expenseReq.Header.Set("Authorization", "Bearer "+token1)
	expenseReq.Header.Set("Content-Type", "application/json")
	expenseResp, err := app.Test(expenseReq, 5000)
	if err != nil {
		t.Fatalf("create expense category failed: %v", err)
	}
	defer expenseResp.Body.Close()
	if expenseResp.StatusCode != 201 {
		t.Fatalf("create expense category status = %d, want 201", expenseResp.StatusCode)
	}

	// 3. User 1 lists categories -> gets 2 categories
	listReq1 := httptest.NewRequest("GET", "/v1/categories", nil)
	listReq1.Header.Set("Authorization", "Bearer "+token1)
	listResp1, err := app.Test(listReq1, 5000)
	if err != nil {
		t.Fatalf("list categories for user 1 failed: %v", err)
	}
	defer listResp1.Body.Close()
	if listResp1.StatusCode != 200 {
		t.Fatalf("list categories status = %d, want 200", listResp1.StatusCode)
	}

	var listBody1 struct {
		Success bool                `json:"success"`
		Data    []category.Category `json:"data"`
	}
	if err := json.NewDecoder(listResp1.Body).Decode(&listBody1); err != nil {
		t.Fatalf("decode category list response: %v", err)
	}
	if len(listBody1.Data) != 2 {
		t.Fatalf("user 1 category list count = %d, want 2", len(listBody1.Data))
	}

	// 4. User 2 lists categories -> gets 0 categories (cross-user isolation)
	listReq2 := httptest.NewRequest("GET", "/v1/categories", nil)
	listReq2.Header.Set("Authorization", "Bearer "+token2)
	listResp2, err := app.Test(listReq2, 5000)
	if err != nil {
		t.Fatalf("list categories for user 2 failed: %v", err)
	}
	defer listResp2.Body.Close()
	if listResp2.StatusCode != 200 {
		t.Fatalf("list categories user 2 status = %d, want 200", listResp2.StatusCode)
	}

	var listBody2 struct {
		Success bool                `json:"success"`
		Data    []category.Category `json:"data"`
	}
	if err := json.NewDecoder(listResp2.Body).Decode(&listBody2); err != nil {
		t.Fatalf("decode category list response user 2: %v", err)
	}
	if len(listBody2.Data) != 0 {
		t.Fatalf("user 2 category list count = %d, want 0", len(listBody2.Data))
	}

	// 5. Invalid type input -> 422 Unprocessable Entity
	invalidReq := httptest.NewRequest("POST", "/v1/categories", bytes.NewBufferString(`{"name":"Invalid","type":"invalid_type"}`))
	invalidReq.Header.Set("Authorization", "Bearer "+token1)
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidResp, err := app.Test(invalidReq, 5000)
	if err != nil {
		t.Fatalf("invalid category request failed: %v", err)
	}
	defer invalidResp.Body.Close()
	if invalidResp.StatusCode != 422 {
		t.Fatalf("invalid category status = %d, want 422", invalidResp.StatusCode)
	}
}

func TestConfiguredApp_UpdateAndDeactivateCategory(t *testing.T) {
	token := jwt.New(config.JWT{Secret: "test-secret"})
	userRepo := &multiUserRepoMock{
		users: map[string]*user.User{
			"user-1": {ID: "user-1", Email: "user1@example.com"},
			"user-2": {ID: "user-2", Email: "user2@example.com"},
		},
	}
	wsRepo := &mainTestWorkspaceRepo{}
	acctRepo := &mainTestAccountRepo{}
	catRepo := &mainTestCategoryRepo{}

	uHandler := user.NewUserHandler(user.NewUserUsecase(userRepo, token))
	wHandler := workspace.NewWorkspaceHandler(workspace.NewWorkspaceUsecase(wsRepo))
	acctHandler := account.NewAccountHandler(account.NewAccountUsecase(acctRepo))
	catHandler := category.NewCategoryHandler(category.NewCategoryUsecase(catRepo))
	app := newAPIApp("personal-finance-api", uHandler, wHandler, acctHandler, catHandler, token)

	token1, _ := token.Sign(map[string]any{"sub": "user-1"}, time.Hour)
	token2, _ := token.Sign(map[string]any{"sub": "user-2"}, time.Hour)

	// 1. Create category as user 1
	createReq := httptest.NewRequest("POST", "/v1/categories", bytes.NewBufferString(`{"name":"Initial Category","type":"income"}`))
	createReq.Header.Set("Authorization", "Bearer "+token1)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq, 5000)
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	defer createResp.Body.Close()

	var createBody struct {
		Data category.Category `json:"data"`
	}
	_ = json.NewDecoder(createResp.Body).Decode(&createBody)
	catID := createBody.Data.ID

	// 2. User 2 attempts to update User 1's category -> 403 Forbidden
	updateReqForbidden := httptest.NewRequest("PATCH", "/v1/categories/"+catID, bytes.NewBufferString(`{"name":"Hacked"}`))
	updateReqForbidden.Header.Set("Authorization", "Bearer "+token2)
	updateReqForbidden.Header.Set("Content-Type", "application/json")
	updateRespForbidden, err := app.Test(updateReqForbidden, 5000)
	if err != nil {
		t.Fatalf("forbidden category update request failed: %v", err)
	}
	defer updateRespForbidden.Body.Close()
	if updateRespForbidden.StatusCode != 403 {
		t.Fatalf("status = %d, want 403 Forbidden", updateRespForbidden.StatusCode)
	}

	// 3. User 1 updates their category -> 200 OK
	updateReq := httptest.NewRequest("PATCH", "/v1/categories/"+catID, bytes.NewBufferString(`{"name":"Updated Category","type":"expense"}`))
	updateReq.Header.Set("Authorization", "Bearer "+token1)
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq, 5000)
	if err != nil {
		t.Fatalf("update category request failed: %v", err)
	}
	defer updateResp.Body.Close()
	if updateResp.StatusCode != 200 {
		t.Fatalf("update status = %d, want 200 OK", updateResp.StatusCode)
	}

	var updateBody struct {
		Data category.Category `json:"data"`
	}
	_ = json.NewDecoder(updateResp.Body).Decode(&updateBody)
	if updateBody.Data.Name != "Updated Category" || updateBody.Data.Type != category.CategoryTypeExpense {
		t.Fatalf("unexpected updated category data: %#v", updateBody.Data)
	}

	// 4. User 1 deactivates their category -> 200 OK
	deactivateReq := httptest.NewRequest("DELETE", "/v1/categories/"+catID, nil)
	deactivateReq.Header.Set("Authorization", "Bearer "+token1)
	deactivateResp, err := app.Test(deactivateReq, 5000)
	if err != nil {
		t.Fatalf("deactivate category request failed: %v", err)
	}
	defer deactivateResp.Body.Close()
	if deactivateResp.StatusCode != 200 {
		t.Fatalf("deactivate status = %d, want 200 OK", deactivateResp.StatusCode)
	}

	var deactivateBody struct {
		Data category.Category `json:"data"`
	}
	_ = json.NewDecoder(deactivateResp.Body).Decode(&deactivateBody)
	if deactivateBody.Data.IsActive != false {
		t.Fatalf("expected IsActive to be false, got true")
	}
}

type mainTestCategoryRepo struct {
	categories map[string]*category.Category
}

func (m *mainTestCategoryRepo) Create(_ context.Context, entity *category.Category) error {
	if m.categories == nil {
		m.categories = make(map[string]*category.Category)
	}
	m.categories[entity.ID] = entity
	return nil
}

func (m *mainTestCategoryRepo) FindByUserID(_ context.Context, userID string) ([]*category.Category, error) {
	var list []*category.Category
	for _, cat := range m.categories {
		if cat.UserID == userID {
			list = append(list, cat)
		}
	}
	return list, nil
}

func (m *mainTestCategoryRepo) FindByID(_ context.Context, id string) (*category.Category, error) {
	cat, ok := m.categories[id]
	if !ok {
		return nil, category.ErrCategoryNotFound
	}
	return cat, nil
}

func (m *mainTestCategoryRepo) Update(_ context.Context, entity *category.Category) error {
	if m.categories == nil {
		m.categories = make(map[string]*category.Category)
	}
	m.categories[entity.ID] = entity
	return nil
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

func (m *mainTestAccountRepo) Update(_ context.Context, entity *account.Account) error {
	if m.accounts == nil {
		m.accounts = make(map[string]*account.Account)
	}
	m.accounts[entity.ID] = entity
	return nil
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
