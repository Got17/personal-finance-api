package category

import (
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

type CategoryHandler struct {
	usecase CategoryUsecase
}

func NewCategoryHandler(usecase CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{usecase: usecase}
}

// RegisterRoutes registers HTTP handlers for Category operations on the Fiber router.
func (h *CategoryHandler) RegisterRoutes(r fiber.Router) {
	categories := r.Group("/categories")
	categories.Post("/", h.CreateCategory)
	categories.Get("/", h.ListCategories)
}

func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}

	var input CreateCategoryInput
	if err := c.BodyParser(&input); err != nil {
		return httputil.RespondBadRequest(c)
	}

	result, err := h.usecase.CreateCategory(c.Context(), userID, &input)
	if err != nil {
		return httputil.RespondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.Success(result))
}

func (h *CategoryHandler) ListCategories(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}

	categories, err := h.usecase.ListCategories(c.Context(), userID)
	if err != nil {
		return httputil.RespondError(c, err)
	}

	return c.JSON(response.Success(categories))
}
