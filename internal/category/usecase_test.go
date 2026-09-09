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
	FindByIDFn     func(ctx context.Context, id string) (*category.Category, error)
	UpdateFn       func(ctx context.Context, entity *category.Category) error
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

func (m *mockCategoryRepo) FindByID(ctx context.Context, id string) (*category.Category, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, category.ErrCategoryNotFound
}

func (m *mockCategoryRepo) Update(ctx context.Context, entity *category.Category) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, entity)
	}
	return nil
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

func TestUpdateCategory_Success(t *testing.T) {
	existing := &category.Category{
		ID:       "cat-1",
		UserID:   "user-123",
		Name:     "Old Name",
		Type:     category.CategoryTypeExpense,
		IsActive: true,
	}
	var updated *category.Category
	repo := &mockCategoryRepo{
		FindByIDFn: func(ctx context.Context, id string) (*category.Category, error) {
			if id == "cat-1" {
				return existing, nil
			}
			return nil, category.ErrCategoryNotFound
		},
		UpdateFn: func(ctx context.Context, entity *category.Category) error {
			updated = entity
			return nil
		},
	}

	uc := category.NewCategoryUsecase(repo)
	newName := "New Name"
	newType := "income"
	result, err := uc.UpdateCategory(context.Background(), "user-123", "cat-1", &category.UpdateCategoryInput{
		Name: &newName,
		Type: &newType,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "New Name" || result.Type != category.CategoryTypeIncome || updated != existing {
		t.Fatalf("unexpected category update: %#v", result)
	}
}

func TestUpdateCategory_NotFound(t *testing.T) {
	repo := &mockCategoryRepo{
		FindByIDFn: func(ctx context.Context, id string) (*category.Category, error) {
			return nil, category.ErrCategoryNotFound
		},
	}
	uc := category.NewCategoryUsecase(repo)
	newName := "New Name"
	_, err := uc.UpdateCategory(context.Background(), "user-123", "non-existent", &category.UpdateCategoryInput{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error for non-existent category")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 404 {
		t.Fatalf("expected 404 AppError, got: %v", err)
	}
}

func TestUpdateCategory_AccessDenied(t *testing.T) {
	existing := &category.Category{
		ID:     "cat-1",
		UserID: "user-owner",
		Name:   "Salary",
		Type:   category.CategoryTypeIncome,
	}
	repo := &mockCategoryRepo{
		FindByIDFn: func(ctx context.Context, id string) (*category.Category, error) {
			return existing, nil
		},
	}
	uc := category.NewCategoryUsecase(repo)
	newName := "Hacked"
	_, err := uc.UpdateCategory(context.Background(), "other-user", "cat-1", &category.UpdateCategoryInput{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected access denied error")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 403 {
		t.Fatalf("expected 403 AppError, got: %v", err)
	}
}

func TestUpdateCategory_InvalidInput(t *testing.T) {
	existing := &category.Category{
		ID:     "cat-1",
		UserID: "user-123",
		Name:   "Salary",
		Type:   category.CategoryTypeIncome,
	}
	repo := &mockCategoryRepo{
		FindByIDFn: func(ctx context.Context, id string) (*category.Category, error) {
			return existing, nil
		},
	}
	uc := category.NewCategoryUsecase(repo)

	emptyName := "  "
	_, err := uc.UpdateCategory(context.Background(), "user-123", "cat-1", &category.UpdateCategoryInput{
		Name: &emptyName,
	})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError for empty name, got: %v", err)
	}

	invalidType := "bogus"
	_, err = uc.UpdateCategory(context.Background(), "user-123", "cat-1", &category.UpdateCategoryInput{
		Type: &invalidType,
	})
	if err == nil {
		t.Fatal("expected validation error for invalid type")
	}
	ae, ok = errs.IsAppError(err)
	if !ok || ae.Status != 422 {
		t.Fatalf("expected 422 AppError for invalid type, got: %v", err)
	}
}

func TestDeactivateCategory_Success(t *testing.T) {
	existing := &category.Category{
		ID:       "cat-1",
		UserID:   "user-123",
		Name:     "Subscriptions",
		Type:     category.CategoryTypeExpense,
		IsActive: true,
	}
	var updated *category.Category
	repo := &mockCategoryRepo{
		FindByIDFn: func(ctx context.Context, id string) (*category.Category, error) {
			return existing, nil
		},
		UpdateFn: func(ctx context.Context, entity *category.Category) error {
			updated = entity
			return nil
		},
	}

	uc := category.NewCategoryUsecase(repo)
	result, err := uc.DeactivateCategory(context.Background(), "user-123", "cat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsActive != false || updated.IsActive != false {
		t.Fatalf("expected IsActive to be false, got %#v", result)
	}
}

func TestDeactivateCategory_AccessDenied(t *testing.T) {
	existing := &category.Category{
		ID:       "cat-1",
		UserID:   "user-owner",
		IsActive: true,
	}
	repo := &mockCategoryRepo{
		FindByIDFn: func(ctx context.Context, id string) (*category.Category, error) {
			return existing, nil
		},
	}

	uc := category.NewCategoryUsecase(repo)
	_, err := uc.DeactivateCategory(context.Background(), "other-user", "cat-1")
	if err == nil {
		t.Fatal("expected 403 Forbidden for cross-user deactivation")
	}

	ae, ok := errs.IsAppError(err)
	if !ok || ae.Status != 403 {
		t.Fatalf("expected 403 AppError, got: %v", err)
	}
}
