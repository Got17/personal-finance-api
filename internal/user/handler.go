package user

import (
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

type UserHandler struct {
	usecase UserUsecase
}

func NewUserHandler(usecase UserUsecase) *UserHandler {
	return &UserHandler{usecase: usecase}
}

// RegisterAuthRoutes wires public authentication routes onto the versioned API.
func (h *UserHandler) RegisterAuthRoutes(r fiber.Router) {
	r.Post("/auth/login", h.SignIn)
	r.Post("/auth/signup", h.SignUp)
}

// RegisterProtectedRoutes wires authenticated user identity routes onto the versioned API.
func (h *UserHandler) RegisterProtectedRoutes(r fiber.Router) {
	r.Get("/users/me", h.GetCurrentUser)
}

func (h *UserHandler) SignIn(c *fiber.Ctx) error {
	var input SignInInput
	if err := c.BodyParser(&input); err != nil {
		return httputil.RespondBadRequest(c)
	}
	result, err := h.usecase.SignIn(c.Context(), &input)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(result))
}

func (h *UserHandler) SignUp(c *fiber.Ctx) error {
	var input SignUpInput
	if err := c.BodyParser(&input); err != nil {
		return httputil.RespondBadRequest(c)
	}
	result, err := h.usecase.SignUp(c.Context(), &input)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success(result))
}

func (h *UserHandler) GetCurrentUser(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	user, err := h.usecase.GetCurrentUser(c.Context(), userID)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(user))
}
