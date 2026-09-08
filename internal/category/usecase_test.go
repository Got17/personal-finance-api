package category_test

import (
	"context"
	"testing"

	"github.com/BounkhongDev/bkgo/errs"

	"github.com/Got17/personal-finance-api/internal/category"
)

type mockCategoryRepo struct {
	CreateFn       func(ctx context.Context, entity *category.Category) error
	FindByUserIDFn func(ctx context.Context, userID string) ([]*category.Category, error)
}

func (m *mockCategoryRepo) Create(ctx context.Context, entity *category.Category) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, entity)
	}
	return nil
}

func (m *mockCategoryRepo) FindByUserID(ctx context.Context, userID string) ([]*category.Category, error) {
	if m.FindByUserIDFn != nil {
		return m.FindByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func TestCreateCategory_Success(t *testing.T) {
	var savedEntity *category.Category
	repo := &mockCategoryRepo{
		CreateFn: func(ctx context.Context, entity *category.Category) error {
			savedEntity = entity
			return nil
		},
	}

	uc := category.NewCategoryUsecase(repo)
	input := &category.CreateCategoryInput{
		Name: "Salary",
		Type: "income",
	}

	result, err := uc.CreateCategory(context.Background(), "user-123", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil || savedEntity == nil {
		t.Fatal("expected category to be saved and returned")
	}

	if result.Name != "Salary" || result.Type != category.CategoryTypeIncome || !result.IsActive || result.UserID != "user-123" {
		t.Fatalf("unexpected category data: %#v", result)
	}
}

func TestCreateCategory_ExplicitIsActive(t *testing.T) {
	inactive := false
	repo := &mockCategoryRepo{
		CreateFn: func(ctx context.Context, entity *category.Category) error {
			return nil
		},
	}

	uc := category.NewCategoryUsecase(repo)
	input := &category.CreateCategoryInput{
		Name:     "Freelance",
		Type:     "INCOME",
		IsActive: &inactive,
	}

	result, err := uc.CreateCategory(context.Background(), "user-123", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Type != category.CategoryTypeIncome || result.IsActive != false {
		t.Fatalf("expected normalized type 'income' and IsActive false, got: %#v", result)
	}
}

func TestCreateCategory_InvalidType(t *testing.T) {
	uc := category.NewCategoryUsecase(&mockCategoryRepo{})
	input := &category.CreateCategoryInput{
		Name: "Investments",
		Type: "crypto",
	}

	_, err := uc.CreateCategory(context.Background(), "user-123", input)
	if err == nil {
		t.Fatal("expected error for unsupported category type")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError, got: %v", err)
	}
}

func TestCreateCategory_EmptyName(t *testing.T) {
	uc := category.NewCategoryUsecase(&mockCategoryRepo{})
	input := &category.CreateCategoryInput{
		Name: "   ",
		Type: "expense",
	}

	_, err := uc.CreateCategory(context.Background(), "user-123", input)
	if err == nil {
		t.Fatal("expected error for empty category name")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError, got: %v", err)
	}
}

func TestCreateCategory_UnauthenticatedUser(t *testing.T) {
	uc := category.NewCategoryUsecase(&mockCategoryRepo{})
	input := &category.CreateCategoryInput{
		Name: "Salary",
		Type: "income",
	}

	_, err := uc.CreateCategory(context.Background(), "", input)
	if err == nil {
		t.Fatal("expected unauthorized error for empty user ID")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 401 {
		t.Fatalf("expected 401 AppError, got: %v", err)
	}
}

func TestListCategories_Success(t *testing.T) {
	expectedList := []*category.Category{
		{ID: "cat-1", UserID: "user-123", Name: "Salary", Type: category.CategoryTypeIncome, IsActive: true},
		{ID: "cat-2", UserID: "user-123", Name: "Rent", Type: category.CategoryTypeExpense, IsActive: true},
	}
	repo := &mockCategoryRepo{
		FindByUserIDFn: func(ctx context.Context, userID string) ([]*category.Category, error) {
			return expectedList, nil
		},
	}

	uc := category.NewCategoryUsecase(repo)
	list, err := uc.ListCategories(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(list))
	}
}

func TestListCategories_Empty(t *testing.T) {
	repo := &mockCategoryRepo{
		FindByUserIDFn: func(ctx context.Context, userID string) ([]*category.Category, error) {
			return nil, nil
		},
	}

	uc := category.NewCategoryUsecase(repo)
	list, err := uc.ListCategories(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if list == nil || len(list) != 0 {
		t.Fatalf("expected empty non-nil category list, got %#v", list)
	}
}
