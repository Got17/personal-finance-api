package account

import (
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

type AccountHandler struct {
	usecase AccountUsecase
}

func NewAccountHandler(usecase AccountUsecase) *AccountHandler {
	return &AccountHandler{usecase: usecase}
}

// RegisterRoutes registers HTTP handlers for Account operations on the Fiber router.
func (h *AccountHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/accounts", h.CreateAccount)
	r.Get("/accounts", h.ListAccounts)
}

func (h *AccountHandler) CreateAccount(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}

	var input CreateAccountInput
	if err := c.BodyParser(&input); err != nil {
		return httputil.RespondBadRequest(c)
	}

	result, err := h.usecase.CreateAccount(c.Context(), userID, &input)
	if err != nil {
		return httputil.RespondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.Success(result))
}

func (h *AccountHandler) ListAccounts(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}

	accounts, err := h.usecase.ListAccounts(c.Context(), userID)
	if err != nil {
		return httputil.RespondError(c, err)
	}

	return c.JSON(response.Success(accounts))
}
