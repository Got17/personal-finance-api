package financialrecord

import (
	"strconv"
	"strings"
	"time"

	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

type FinancialRecordHandler struct{ usecase FinancialRecordUsecase }

func NewFinancialRecordHandler(usecase FinancialRecordUsecase) *FinancialRecordHandler {
	return &FinancialRecordHandler{usecase: usecase}
}

func (h *FinancialRecordHandler) RegisterRoutes(r fiber.Router) {
	records := r.Group("/financial-records")
	records.Post("/", h.CreateFinancialRecord)
	records.Get("/", h.ListFinancialRecords)
	records.Get("/:id", h.GetFinancialRecord)
	records.Put("/:id", h.UpdateFinancialRecord)
	records.Patch("/:id", h.UpdateFinancialRecord)
	records.Delete("/:id", h.ArchiveFinancialRecord)
}

func (h *FinancialRecordHandler) CreateFinancialRecord(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	var input CreateFinancialRecordInput
	if err := c.BodyParser(&input); err != nil {
		return httputil.RespondBadRequest(c)
	}
	record, err := h.usecase.CreateFinancialRecord(c.Context(), userID, &input)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success(record))
}

func (h *FinancialRecordHandler) GetFinancialRecord(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	record, err := h.usecase.GetFinancialRecord(c.Context(), userID, c.Params("id"))
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(record))
}

func (h *FinancialRecordHandler) ListFinancialRecords(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	filter, err := listFilter(c)
	if err != nil {
		return httputil.RespondBadRequest(c)
	}
	records, err := h.usecase.ListFinancialRecords(c.Context(), userID, filter)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(records))
}

func listFilter(c *fiber.Ctx) (ListFilter, error) {
	filter := ListFilter{Kind: Kind(strings.ToLower(strings.TrimSpace(c.Query("kind")))), AccountID: strings.TrimSpace(c.Query("account_id")), CategoryID: strings.TrimSpace(c.Query("category_id"))}
	startDate, err := parseDate(c.Query("start_date"))
	if err != nil {
		return ListFilter{}, err
	}
	filter.StartDate = startDate
	endDate, err := parseDate(c.Query("end_date"))
	if err != nil {
		return ListFilter{}, err
	}
	filter.EndDate = endDate
	includeArchived := c.Query("include_archived")
	if includeArchived == "" {
		return filter, nil
	}
	filter.IncludeArchived, err = strconv.ParseBool(includeArchived)
	return filter, err
}

func parseDate(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (h *FinancialRecordHandler) UpdateFinancialRecord(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	var input UpdateFinancialRecordInput
	if err := c.BodyParser(&input); err != nil {
		return httputil.RespondBadRequest(c)
	}
	record, err := h.usecase.UpdateFinancialRecord(c.Context(), userID, c.Params("id"), &input)
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(record))
}

func (h *FinancialRecordHandler) ArchiveFinancialRecord(c *fiber.Ctx) error {
	userID := httputil.GetUserID(c)
	if userID == "" {
		return httputil.RespondUnauthorized(c)
	}
	record, err := h.usecase.ArchiveFinancialRecord(c.Context(), userID, c.Params("id"))
	if err != nil {
		return httputil.RespondError(c, err)
	}
	return c.JSON(response.Success(record))
}

