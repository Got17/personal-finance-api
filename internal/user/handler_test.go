package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/hash"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/user"
)

func TestSignIn_ValidCredentialsIssuesAuthenticatedSession(t *testing.T) {
	passwordHash, err := hash.Password("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	app := fiber.New()
	repo := &signInRepo{user: &user.User{
		ID:           "user-123",
		Email:        "owner@example.com",
		PasswordHash: passwordHash,
	}}
	token := jwt.New(config.JWT{Secret: "test-secret"})
	handler := user.NewUserHandler(user.NewUserUsecase(repo, token))
	handler.RegisterAuthRoutes(app.Group("/v1"))

	request := httptest.NewRequest("POST", "/v1/auth/sign-in", bytes.NewBufferString(`{"email":"owner@example.com","password":"correct horse battery staple"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
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

func TestSignIn_InvalidCredentialsReturnTheSameSafeError(t *testing.T) {
	passwordHash, err := hash.Password("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	app := fiber.New()
	handler := user.NewUserHandler(user.NewUserUsecase(&signInRepo{user: &user.User{
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
		request := httptest.NewRequest("POST", "/v1/auth/sign-in", bytes.NewBufferString(input))
		request.Header.Set("Content-Type", "application/json")
		response, err := app.Test(request)
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

type signInRepo struct {
	user *user.User
}

func (r *signInRepo) FindByEmail(_ context.Context, email string) (*user.User, error) {
	if r.user != nil && r.user.Email == email {
		return r.user, nil
	}
	return nil, user.ErrUserNotFound
}
