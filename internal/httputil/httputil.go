package httputil

import (
	"log/slog"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/i18n"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/messages"
)

// HTTP error code constants to prevent duplicate string literals across handlers.
const (
	ErrCodeBadRequest    = "BAD_REQUEST"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeForbidden     = "FORBIDDEN"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeConflict      = "CONFLICT"
	ErrCodeInternalError = "INTERNAL_ERROR"
)

// GetUserID retrieves the authenticated user's ID from the JWT token claims.
func GetUserID(c *fiber.Ctx) string {
	claims := middleware.Claims(c)
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

// Locale reads the Accept-Language header and returns the best matching locale.
func Locale(c *fiber.Ctx) i18n.Locale {
	return i18n.FromHeader(c.Get("Accept-Language"))
}

// RespondBadRequest returns a 400 Bad Request error response with translated message.
func RespondBadRequest(c *fiber.Ctx) error {
	msg := i18n.Translate(Locale(c), ErrCodeBadRequest)
	if msg == "" {
		msg = messages.MsgBadRequest
	}
	return c.Status(fiber.StatusBadRequest).JSON(response.Error(ErrCodeBadRequest, msg))
}

// RespondUnauthorized returns a 401 Unauthorized error response with translated message.
func RespondUnauthorized(c *fiber.Ctx) error {
	msg := i18n.Translate(Locale(c), ErrCodeUnauthorized)
	return c.Status(fiber.StatusUnauthorized).JSON(response.Error(ErrCodeUnauthorized, msg))
}

// RespondError maps an application error to an HTTP response.
// AppErrors are translated using the request locale.
// Unexpected errors are logged server-side and a generic internal error message is returned.
func RespondError(c *fiber.Ctx, err error) error {
	if ae, ok := errs.IsAppError(err); ok {
		msg := i18n.Translate(Locale(c), ae.Code)
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
	return c.Status(fiber.StatusInternalServerError).JSON(response.Error(ErrCodeInternalError, i18n.Translate(Locale(c), ErrCodeInternalError)))
}
