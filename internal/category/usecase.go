package category

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/validator"

	"github.com/Got17/personal-finance-api/internal/messages"
)

type CreateCategoryInput struct {
	Name     string `json:"name"      validate:"required,min=1,max=100"`
	Type     string `json:"type"      validate:"required"`
	IsActive *bool  `json:"is_active" validate:"omitempty"`
}

type categoryUsecase struct {
	repo CategoryRepository
}

func NewCategoryUsecase(repo CategoryRepository) CategoryUsecase {
	return &categoryUsecase{repo: repo}
}

func IsValidCategoryType(catType string) bool {
	return SupportedCategoryTypes[CategoryType(strings.ToLower(strings.TrimSpace(catType)))]
}

func (u *categoryUsecase) CreateCategory(ctx context.Context, userID string, input *CreateCategoryInput) (*Category, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}

	if fieldErrs := validator.Validate(input); len(fieldErrs) > 0 {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, fieldErrs)
	}

	catType := strings.ToLower(strings.TrimSpace(input.Type))
	if !IsValidCategoryType(catType) {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{
			"type": messages.MsgUnsupportedCategoryType,
		})
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errs.UnprocessableFields(messages.MsgValidationFailed, map[string]string{
			"name": messages.MsgNameIsRequired,
		})
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	entity := &Category{
		ID:       uuid.NewString(),
		UserID:   userID,
		Name:     name,
		Type:     CategoryType(catType),
		IsActive: isActive,
	}

	if err := u.repo.Create(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func (u *categoryUsecase) ListCategories(ctx context.Context, userID string) ([]*Category, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errs.Unauthorized(messages.MsgUserNotFound)
	}

	categories, err := u.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if categories == nil {
		categories = []*Category{}
	}

	return categories, nil
}
