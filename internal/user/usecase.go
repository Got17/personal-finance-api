package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/hash"
	"github.com/BounkhongDev/bkgo/validator"

	"github.com/Got17/personal-finance-api/internal/messages"
)

// UserUsecase defines the business operations for User.
type UserUsecase interface {
	SignIn(ctx context.Context, input *SignInInput) (*Session, error)
	SignUp(ctx context.Context, input *SignUpInput) (*Session, error)
	GetCurrentUser(ctx context.Context, userID string) (*User, error)
}

type SignInInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SignUpInput struct {
	Email         string `json:"email"          validate:"required,email"`
	Password      string `json:"password"       validate:"required,min=8"`
	WorkspaceName string `json:"workspace_name" validate:"omitempty,max=100"`
}

type Session struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type userUsecase struct {
	repo  UserRepository
	token contract.Token
}

func NewUserUsecase(repo UserRepository, token contract.Token) UserUsecase {
	return &userUsecase{repo: repo, token: token}
}

// dummyHash is used to prevent timing side-channel user enumeration when a user is not found.
var dummyHash, _ = hash.Password("bkgo-timing-mitigation-dummy-password")

func (u *userUsecase) SignIn(ctx context.Context, input *SignInInput) (*Session, error) {
	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}

	entity, err := u.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = hash.CheckPassword(input.Password, dummyHash)
			return nil, errs.Unauthorized(messages.MsgInvalidCredentials)
		}
		return nil, err
	}
	if !hash.CheckPassword(input.Password, entity.PasswordHash) {
		return nil, errs.Unauthorized(messages.MsgInvalidCredentials)
	}

	accessToken, err := u.token.Sign(contract.Claims{"sub": entity.ID}, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &Session{AccessToken: accessToken, TokenType: "Bearer"}, nil
}

func (u *userUsecase) SignUp(ctx context.Context, input *SignUpInput) (*Session, error) {
	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}

	existing, err := u.repo.FindByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errs.Conflict(messages.MsgEmailAlreadyExists)
	}

	passwordHash, err := hash.Password(input.Password)
	if err != nil {
		return nil, err
	}

	wsName := strings.TrimSpace(input.WorkspaceName)
	if wsName == "" {
		wsName = "Personal Workspace"
	}

	newUser := &User{
		ID:           uuid.NewString(),
		Email:        input.Email,
		PasswordHash: passwordHash,
	}

	if _, err := u.repo.CreateWithWorkspace(ctx, newUser, wsName); err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			return nil, errs.Conflict(messages.MsgEmailAlreadyExists)
		}
		return nil, err
	}

	accessToken, err := u.token.Sign(contract.Claims{"sub": newUser.ID}, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &Session{AccessToken: accessToken, TokenType: "Bearer"}, nil
}

func (u *userUsecase) GetCurrentUser(ctx context.Context, userID string) (*User, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}
	entity, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, errs.Unauthorized(messages.MsgUserNotFound)
		}
		return nil, err
	}
	return entity, nil
}
