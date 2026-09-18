package transfer

import (
	"strconv"
	"strings"
	"time"

	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

type TransferHandler struct {
	usecase TransferUsecase
}

func NewTransferHandler(usecase TransferUsecase) *TransferHandler {
	return &TransferHandler{usecase: usecase}
}

func (h *TransferHandler) RegisterRoutes(r fiber.Router) {
	transfers := r.Group("/transfers")
	transfers.Post("/", h.CreateTransfer)
	transfers.Get("/", h.ListTransfers)
	transfers.Get("/:id", h.GetTransfer)
}

func (h *TransferHandler) CreateTransfer(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}

	var input CreateTransferInput
	if err := c.BodyParser(&input); err != nil {
		return httputil.RespondBadRequest(c)
	}

	result, err := h.usecase.CreateTransfer(c.Context(), userID, &input)
	if err != nil {
		return httputil.RespondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.Success(result))
}

func (h *TransferHandler) GetTransfer(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}

	result, err := h.usecase.GetTransfer(c.Context(), userID, c.Params("id"))
	if err != nil {
		return httputil.RespondError(c, err)
	}

	return c.JSON(response.Success(result))
}

func (h *TransferHandler) ListTransfers(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}

	filter, err := parseListFilter(c)
	if err != nil {
		return httputil.RespondBadRequest(c)
	}

	list, err := h.usecase.ListTransfers(c.Context(), userID, filter)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	if list == nil {
		list = []*Transfer{}
	}

	return c.JSON(response.Success(list))
}

func parseListFilter(c *fiber.Ctx) (ListTransferFilter, error) {
	filter := ListTransferFilter{
		AccountID: strings.TrimSpace(c.Query("account_id")),
	}

	startDateStr := strings.TrimSpace(c.Query("start_date"))
	if startDateStr != "" {
		parsed, err := time.Parse(time.DateOnly, startDateStr)
		if err != nil {
			return ListTransferFilter{}, err
		}
		filter.StartDate = &parsed
	}

	endDateStr := strings.TrimSpace(c.Query("end_date"))
	if endDateStr != "" {
		parsed, err := time.Parse(time.DateOnly, endDateStr)
		if err != nil {
			return ListTransferFilter{}, err
		}
		inclusive := parsed.AddDate(0, 0, 1).Add(-time.Microsecond)
		filter.EndDate = &inclusive
	}

	includeArchived := c.Query("include_archived")
	if includeArchived != "" {
		parsed, err := strconv.ParseBool(includeArchived)
		if err != nil {
			return ListTransferFilter{}, err
		}
		filter.IncludeArchived = parsed
	}

	return filter, nil
}
