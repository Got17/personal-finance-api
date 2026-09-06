package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/hash"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/user"
	"github.com/Got17/personal-finance-api/internal/workspace"
)

type mockUserRepo struct {
	user           *user.User
	createdUser    *user.User
	createdWS      *workspace.Workspace
	shouldFailInTx bool
}

func (r *mockUserRepo) FindByEmail(_ context.Context, email string) (*user.User, error) {
	if r.user != nil && r.user.Email == email {
		return r.user, nil
	}
	if r.createdUser != nil && r.createdUser.Email == email {
		return r.createdUser, nil
	}
	return nil, user.ErrUserNotFound
}

func (r *mockUserRepo) CreateWithWorkspace(_ context.Context, u *user.User, workspaceName string) (*workspace.Workspace, error) {
	if r.shouldFailInTx {
		return nil, errors.New("atomic transaction failed")
	}
	ws := &workspace.Workspace{
		ID:      "ws-123",
		Name:    workspaceName,
		OwnerID: u.ID,
	}
	r.createdUser = u
	r.createdWS = ws
	return ws, nil
}

func TestSignIn_ValidCredentialsIssuesAuthenticatedSession(t *testing.T) {
	passwordHash, err := hash.Password("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	app := fiber.New()
	repo := &mockUserRepo{user: &user.User{
		ID:           "user-123",
		Email:        "owner@example.com",
		PasswordHash: passwordHash,
	}}
	token := jwt.New(config.JWT{Secret: "test-secret"})
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	handler.RegisterAuthRoutes(app.Group("/v1"))

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

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || body.Data.AccessToken == "" || body.Data.TokenType != "Bearer" {
		t.Fatalf("body = %#v, want a Bearer session token", body)
	}
	claims, err := token.Verify(body.Data.AccessToken)
	if err != nil {
		t.Fatalf("verify issued token: %v", err)
	}
	if claims["sub"] != "user-123" {
		t.Fatalf("token subject = %v, want user-123", claims["sub"])
	}
}

func TestSignIn_InvalidCredentialsReturnsTheSameSafeError(t *testing.T) {
	passwordHash, err := hash.Password("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	app := fiber.New()
	handler := user.NewUserHandler(user.NewUserUsecase(&mockUserRepo{user: &user.User{
		ID:           "user-123",
		Email:        "owner@example.com",
		PasswordHash: passwordHash,
	}}, jwt.New(config.JWT{Secret: "test-secret"})))
	handler.RegisterAuthRoutes(app.Group("/v1"))

	inputs := []string{
		`{"email":"owner@example.com","password":"wrong password"}`,
		`{"email":"unknown@example.com","password":"correct horse battery staple"}`,
	}
	var messages []string
	for _, input := range inputs {
		request := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewBufferString(input))
		request.Header.Set("Content-Type", "application/json")
		response, err := app.Test(request, 5000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		var body struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			response.Body.Close()
			t.Fatalf("decode response: %v", err)
		}
		response.Body.Close()
		if response.StatusCode != 401 || body.Success || body.Error != "UNAUTHORIZED" {
			t.Fatalf("response = status %d, body %#v; want safe unauthorized error", response.StatusCode, body)
		}
		messages = append(messages, body.Message)
	}

	if messages[0] != messages[1] {
		t.Fatalf("credential failures disclosed different messages: %q and %q", messages[0], messages[1])
	}
}

func TestSignUp_ValidRegistrationIssuesSessionAndCreatesWorkspace(t *testing.T) {
	app := fiber.New()
	repo := &mockUserRepo{}
	token := jwt.New(config.JWT{Secret: "test-secret"})
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	handler.RegisterAuthRoutes(app.Group("/v1"))

	input := `{"email":"newvisitor@example.com","password":"securepassword123","workspace_name":"My Wealth Vault"}`
	request := httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewBufferString(input))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 201 {
		t.Fatalf("status = %d, want 201 Created", response.StatusCode)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !body.Success || body.Data.AccessToken == "" || body.Data.TokenType != "Bearer" {
		t.Fatalf("body = %#v, want Bearer session token", body)
	}

	// Confidentiality assertion: response JSON must NOT contain password or password_hash
	respStr := body.Data.AccessToken
	if strings.Contains(respStr, "securepassword123") {
		t.Fatalf("credentials leaked in signup response!")
	}

	// Verify User & Workspace created
	if repo.createdUser == nil || repo.createdUser.Email != "newvisitor@example.com" {
		t.Fatalf("user was not created correctly: %#v", repo.createdUser)
	}
	if repo.createdWS == nil || repo.createdWS.Name != "My Wealth Vault" || repo.createdWS.OwnerID != repo.createdUser.ID {
		t.Fatalf("workspace was not provisioned correctly: %#v", repo.createdWS)
	}
}

func TestSignUp_DuplicateIdentityReturnsSafeConflictError(t *testing.T) {
	app := fiber.New()
	repo := &mockUserRepo{
		user: &user.User{
			ID:    "user-existing",
			Email: "existing@example.com",
		},
	}
	token := jwt.New(config.JWT{Secret: "test-secret"})
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	handler.RegisterAuthRoutes(app.Group("/v1"))

	input := `{"email":"existing@example.com","password":"securepassword123"}`
	request := httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewBufferString(input))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 409 {
		t.Fatalf("status = %d, want 409 Conflict for duplicate email", response.StatusCode)
	}

	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Success || body.Error != "CONFLICT" {
		t.Fatalf("body = %#v, want CONFLICT error", body)
	}
}

func TestSignUp_InvalidInputReturnsUnprocessableOrBadRequest(t *testing.T) {
	app := fiber.New()
	repo := &mockUserRepo{}
	token := jwt.New(config.JWT{Secret: "test-secret"})
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	handler.RegisterAuthRoutes(app.Group("/v1"))

	// Test malformed JSON
	request := httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewBufferString(`{invalid json`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatalf("status = %d, want 400 for malformed json", response.StatusCode)
	}

	// Test invalid email & short password
	request = httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewBufferString(`{"email":"notanemail","password":"123"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err = app.Test(request, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != 422 {
		t.Fatalf("status = %d, want 422 for validation failure", response.StatusCode)
	}
}

func TestSignUp_AtomicProvisioningFailureLeavesNoRecord(t *testing.T) {
	app := fiber.New()
	repo := &mockUserRepo{shouldFailInTx: true}
	token := jwt.New(config.JWT{Secret: "test-secret"})
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	handler.RegisterAuthRoutes(app.Group("/v1"))

	input := `{"email":"visitor@example.com","password":"securepassword123"}`
	request := httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewBufferString(input))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 500 {
		t.Fatalf("status = %d, want 500 when atomic transaction fails", response.StatusCode)
	}

	if repo.createdUser != nil || repo.createdWS != nil {
		t.Fatalf("atomic provisioning leaked partial record! createdUser=%#v, createdWS=%#v", repo.createdUser, repo.createdWS)
	}
}
