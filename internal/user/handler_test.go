package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/hash"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
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

func (r *mockUserRepo) FindByID(_ context.Context, id string) (*user.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	if r.createdUser != nil && r.createdUser.ID == id {
		return r.createdUser, nil
	}
	return nil, user.ErrUserNotFound
}

func (r *mockUserRepo) UpdateBaseCurrency(_ context.Context, id string, currency string) (*user.User, error) {
	if r.user != nil && r.user.ID == id {
		r.user.BaseCurrency = currency
		return r.user, nil
	}
	if r.createdUser != nil && r.createdUser.ID == id {
		r.createdUser.BaseCurrency = currency
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
		if response.StatusCode != 401 || body.Success || body.Error != httputil.ErrCodeUnauthorized {
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

	if body.Success || body.Error != httputil.ErrCodeConflict {
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

func TestGetCurrentUser_AuthenticatedCallerRetrievesOwnIdentity(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{
			ID:           "user-456",
			Email:        "me@example.com",
			PasswordHash: "hashed-secret",
		},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-456"}, time.Hour)
	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 OK", resp.StatusCode)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			ID           string `json:"id"`
			Email        string `json:"email"`
			PasswordHash string `json:"password_hash,omitempty"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !body.Success || body.Data.ID != "user-456" || body.Data.Email != "me@example.com" {
		t.Fatalf("unexpected user response: %#v", body)
	}
	if body.Data.PasswordHash != "" {
		t.Fatalf("password hash leaked in user profile response!")
	}
}

func TestGetCurrentUser_UnauthenticatedAccessRejected(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{ID: "user-456", Email: "me@example.com"},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	// 1. Missing Authorization header
	reqNoAuth := httptest.NewRequest("GET", "/v1/users/me", nil)
	respNoAuth, err := app.Test(reqNoAuth, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	respNoAuth.Body.Close()
	if respNoAuth.StatusCode != 401 {
		t.Fatalf("status = %d, want 401 Unauthorized for missing token", respNoAuth.StatusCode)
	}

	// 2. Invalid token
	reqBadToken := httptest.NewRequest("GET", "/v1/users/me", nil)
	reqBadToken.Header.Set("Authorization", "Bearer invalid.jwt.token")
	respBadToken, err := app.Test(reqBadToken, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	respBadToken.Body.Close()
	if respBadToken.StatusCode != 401 {
		t.Fatalf("status = %d, want 401 Unauthorized for invalid token", respBadToken.StatusCode)
	}
}

func TestGetCurrentUser_NonExistentUserReturnsUnauthorized(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "ghost-user"}, time.Hour)
	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401 for non-existent user token subject", resp.StatusCode)
	}
}

func TestGetCurrentUser_CannotSelectAnotherUserByIdentityOrParams(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{ID: "user-legit", Email: "legit@example.com"},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-legit"}, time.Hour)
	// Try passing query params attempting to override identity to another user ID
	req := httptest.NewRequest("GET", "/v1/users/me?user_id=other-user-id&id=other-user-id", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 OK", resp.StatusCode)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Must resolve to legit user from JWT, ignoring any request query attempt
	if body.Data.ID != "user-legit" || body.Data.Email != "legit@example.com" {
		t.Fatalf("user identity was tampered via query params: %#v", body)
	}
}

func TestGetBaseCurrency_AuthenticatedCallerRetrievesOwnBaseCurrency(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{
			ID:           "user-789",
			Email:        "currency@example.com",
			BaseCurrency: "USD",
		},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-789"}, time.Hour)

	// 1. GET /v1/users/me includes base_currency
	reqMe := httptest.NewRequest("GET", "/v1/users/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+authToken)
	respMe, err := app.Test(reqMe, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respMe.Body.Close()

	if respMe.StatusCode != 200 {
		t.Fatalf("GET /v1/users/me status = %d, want 200 OK", respMe.StatusCode)
	}

	var bodyMe struct {
		Success bool `json:"success"`
		Data    struct {
			ID           string `json:"id"`
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respMe.Body).Decode(&bodyMe); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyMe.Data.BaseCurrency != "USD" {
		t.Fatalf("base_currency = %q, want USD", bodyMe.Data.BaseCurrency)
	}

	// 2. GET /v1/users/me/preferences returns preferences object
	reqPref := httptest.NewRequest("GET", "/v1/users/me/preferences", nil)
	reqPref.Header.Set("Authorization", "Bearer "+authToken)
	respPref, err := app.Test(reqPref, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respPref.Body.Close()

	if respPref.StatusCode != 200 {
		t.Fatalf("GET /v1/users/me/preferences status = %d, want 200 OK", respPref.StatusCode)
	}

	var bodyPref struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respPref.Body).Decode(&bodyPref); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyPref.Data.BaseCurrency != "USD" {
		t.Fatalf("preferences base_currency = %q, want USD", bodyPref.Data.BaseCurrency)
	}
}

func TestUpdateBaseCurrency_AuthenticatedCallerUpdatesOwnBaseCurrency(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{
			ID:           "user-789",
			Email:        "currency@example.com",
			BaseCurrency: "USD",
		},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-789"}, time.Hour)

	// 1. PUT /v1/users/me/preferences with {"base_currency": "EUR"}
	reqPut := httptest.NewRequest("PUT", "/v1/users/me/preferences", bytes.NewBufferString(`{"base_currency":"EUR"}`))
	reqPut.Header.Set("Authorization", "Bearer "+authToken)
	reqPut.Header.Set("Content-Type", "application/json")
	respPut, err := app.Test(reqPut, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respPut.Body.Close()

	if respPut.StatusCode != 200 {
		t.Fatalf("PUT /v1/users/me/preferences status = %d, want 200 OK", respPut.StatusCode)
	}

	var bodyPut struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respPut.Body).Decode(&bodyPut); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyPut.Data.BaseCurrency != "EUR" {
		t.Fatalf("updated base_currency = %q, want EUR", bodyPut.Data.BaseCurrency)
	}

	// 2. PATCH /v1/users/me/preferences with lowercase currency {"base_currency": "gbp"}
	reqPatch := httptest.NewRequest("PATCH", "/v1/users/me/preferences", bytes.NewBufferString(`{"base_currency":"gbp"}`))
	reqPatch.Header.Set("Authorization", "Bearer "+authToken)
	reqPatch.Header.Set("Content-Type", "application/json")
	respPatch, err := app.Test(reqPatch, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respPatch.Body.Close()

	if respPatch.StatusCode != 200 {
		t.Fatalf("PATCH /v1/users/me/preferences status = %d, want 200 OK", respPatch.StatusCode)
	}

	var bodyPatch struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respPatch.Body).Decode(&bodyPatch); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyPatch.Data.BaseCurrency != "GBP" {
		t.Fatalf("normalized base_currency = %q, want GBP", bodyPatch.Data.BaseCurrency)
	}

	// 3. Verify GET /v1/users/me now reflects GBP
	reqMe := httptest.NewRequest("GET", "/v1/users/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+authToken)
	respMe, err := app.Test(reqMe, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respMe.Body.Close()

	var bodyMe struct {
		Success bool `json:"success"`
		Data    struct {
			BaseCurrency string `json:"base_currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respMe.Body).Decode(&bodyMe); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyMe.Data.BaseCurrency != "GBP" {
		t.Fatalf("GET /v1/users/me base_currency = %q, want GBP", bodyMe.Data.BaseCurrency)
	}
}

func TestUpdateBaseCurrency_InvalidCurrenciesRejected(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})
	repo := &mockUserRepo{
		user: &user.User{
			ID:           "user-789",
			Email:        "currency@example.com",
			BaseCurrency: "USD",
		},
	}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authToken, _ := token.Sign(map[string]any{"sub": "user-789"}, time.Hour)

	invalidInputs := []string{
		`{"base_currency":"INVALID"}`,
		`{"base_currency":"US"}`,
		`{"base_currency":"123"}`,
		`{"base_currency":""}`,
		`{"base_currency":"TOOLONG"}`,
	}

	for _, input := range invalidInputs {
		req := httptest.NewRequest("PUT", "/v1/users/me/preferences", bytes.NewBufferString(input))
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 422 && resp.StatusCode != 400 {
			t.Fatalf("input %s: status = %d, want 422 or 400 validation error", input, resp.StatusCode)
		}
	}
}

func TestUpdateBaseCurrency_UserCanOnlyUpdateOwnPreference(t *testing.T) {
	app := fiber.New()
	token := jwt.New(config.JWT{Secret: "test-secret"})

	userA := &user.User{ID: "user-a", Email: "usera@example.com", BaseCurrency: "USD"}
	userB := &user.User{ID: "user-b", Email: "userb@example.com", BaseCurrency: "USD"}

	repo := &mockUserRepo{user: userA}
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	api := app.Group("/v1", middleware.JWT(token))
	handler.RegisterProtectedRoutes(api)

	authTokenA, _ := token.Sign(map[string]any{"sub": "user-a"}, time.Hour)

	// User A updates preference via JWT A, including attempt to inject user_id in payload/query
	req := httptest.NewRequest("PUT", "/v1/users/me/preferences?user_id=user-b", bytes.NewBufferString(`{"base_currency":"EUR","user_id":"user-b"}`))
	req.Header.Set("Authorization", "Bearer "+authTokenA)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 OK", resp.StatusCode)
	}

	if userA.BaseCurrency != "EUR" {
		t.Fatalf("User A base_currency = %q, want EUR", userA.BaseCurrency)
	}
	if userB.BaseCurrency != "USD" {
		t.Fatalf("User B base_currency = %q, want USD (untouched)", userB.BaseCurrency)
	}
}
