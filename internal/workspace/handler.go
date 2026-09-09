package workspace

import (
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

type WorkspaceHandler struct {
	usecase WorkspaceUsecase
}

func NewWorkspaceHandler(usecase WorkspaceUsecase) *WorkspaceHandler {
	return &WorkspaceHandler{usecase: usecase}
}

// RegisterRoutes attaches protected workspace routes to the Fiber router.
func (h *WorkspaceHandler) RegisterRoutes(r fiber.Router) {
	workspaces := r.Group("/workspaces")
	workspaces.Get("/", h.ListWorkspaces)
	workspaces.Get("/:id", h.GetWorkspace)
}

func (h *WorkspaceHandler) ListWorkspaces(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	workspaces, err := h.usecase.ListWorkspaces(c.Context(), userID)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(workspaces))
}

func (h *WorkspaceHandler) GetWorkspace(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	workspaceID := c.Params("id")
	ws, err := h.usecase.GetWorkspace(c.Context(), userID, workspaceID)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(ws))
}
