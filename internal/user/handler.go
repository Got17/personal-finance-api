package user

import (
	"log/slog"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/i18n"
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/messages"
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

func (h *UserHandler) SignIn(c *fiber.Ctx) error {
	var input SignInInput
	if err := c.BodyParser(&input); err != nil {
		msg := i18n.Translate(locale(c), "BAD_REQUEST")
		if msg == "" {
			msg = messages.MsgBadRequest
		}
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("BAD_REQUEST", msg))
	}
	result, err := h.usecase.SignIn(c.Context(), &input)
	if err != nil {
		return httpErr(c, err)
	}
	return c.JSON(response.Success(result))
}

func (h *UserHandler) SignUp(c *fiber.Ctx) error {
	var input SignUpInput
	if err := c.BodyParser(&input); err != nil {
		msg := i18n.Translate(locale(c), "BAD_REQUEST")
		if msg == "" {
			msg = messages.MsgBadRequest
		}
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("BAD_REQUEST", msg))
	}
	result, err := h.usecase.SignUp(c.Context(), &input)
	if err != nil {
		return httpErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success(result))
}

// locale reads the Accept-Language header and returns the best matching locale.
func locale(c *fiber.Ctx) i18n.Locale {
	return i18n.FromHeader(c.Get("Accept-Language"))
}

// httpErr maps an error to an HTTP response.
// AppErrors are translated using the request locale.
// Unexpected errors are logged server-side; a generic message is returned to the client.
func httpErr(c *fiber.Ctx, err error) error {
	if ae, ok := errs.IsAppError(err); ok {
		msg := i18n.Translate(locale(c), ae.Code)
		if msg == "" {
			msg = ae.Message
		}
		resp := response.Error(ae.Code, msg)
		if ae.Data != nil {
			resp.Data = ae.Data
		}
		return c.Status(ae.Status).JSON(resp)
	}
	slog.Error("internal error", "error", err, "path", c.Path())
	return c.Status(500).JSON(response.Error("INTERNAL_ERROR", i18n.Translate(locale(c), "INTERNAL_ERROR")))
}
