package user

import (
	"context"
	"errors"
	"time"

	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/hash"
	"github.com/BounkhongDev/bkgo/validator"

	"github.com/Got17/personal-finance-api/internal/messages"
)

// UserUsecase defines the business operations for User.
type UserUsecase interface {
	SignIn(ctx context.Context, input *SignInInput) (*Session, error)
}

type SignInInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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
