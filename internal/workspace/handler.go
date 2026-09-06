package workspace

import (
	"log/slog"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/BounkhongDev/bkgo/i18n"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"
)

type WorkspaceHandler struct {
	usecase WorkspaceUsecase
}

func NewWorkspaceHandler(usecase WorkspaceUsecase) *WorkspaceHandler {
	return &WorkspaceHandler{usecase: usecase}
}

// RegisterRoutes attaches protected workspace routes to the Fiber router.
func (h *WorkspaceHandler) RegisterRoutes(r fiber.Router) {
	r.Get("/workspaces", h.ListWorkspaces)
	r.Get("/workspaces/:id", h.GetWorkspace)
}

func (h *WorkspaceHandler) ListWorkspaces(c *fiber.Ctx) error {
	userID := getUserID(c)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error("UNAUTHORIZED", i18n.Translate(locale(c), "UNAUTHORIZED")))
	}
	workspaces, err := h.usecase.ListWorkspaces(c.Context(), userID)
	if err != nil {
		return httpErr(c, err)
	}
	return c.JSON(response.Success(workspaces))
}

func (h *WorkspaceHandler) GetWorkspace(c *fiber.Ctx) error {
	userID := getUserID(c)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error("UNAUTHORIZED", i18n.Translate(locale(c), "UNAUTHORIZED")))
	}
	workspaceID := c.Params("id")
	ws, err := h.usecase.GetWorkspace(c.Context(), userID, workspaceID)
	if err != nil {
		return httpErr(c, err)
	}
	return c.JSON(response.Success(ws))
}

func getUserID(c *fiber.Ctx) string {
	claims := middleware.Claims(c)
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

func locale(c *fiber.Ctx) i18n.Locale {
	return i18n.FromHeader(c.Get("Accept-Language"))
}

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
