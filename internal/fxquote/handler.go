package fxquote

import (
	"strings"
	"time"

	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

type FXQuoteHandler struct {
	usecase FXQuoteUsecase
}

func NewFXQuoteHandler(usecase FXQuoteUsecase) *FXQuoteHandler {
	return &FXQuoteHandler{usecase: usecase}
}

func (h *FXQuoteHandler) RegisterRoutes(r fiber.Router) {
	r.Get("/fx-quotes", h.GetRate)
}

func (h *FXQuoteHandler) GetRate(c *fiber.Ctx) error {
	from := strings.ToUpper(strings.TrimSpace(c.Query("from")))
	to := strings.ToUpper(strings.TrimSpace(c.Query("to")))

	dateStr := strings.TrimSpace(c.Query("date"))
	var date time.Time
	if dateStr != "" {
		parsed, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			parsedDate, errDate := time.Parse(time.DateOnly, dateStr)
			if errDate != nil {
				return httputil.RespondBadRequest(c)
			}
			date = parsedDate
		} else {
			date = parsed
		}
	} else {
		date = time.Now().UTC()
	}

	rate, err := h.usecase.GetRate(c.Context(), from, to, date)
	if err != nil {
		return httputil.RespondError(c, err)
	}

	return c.JSON(response.Success(fiber.Map{
		"from_currency": from,
		"to_currency":   to,
		"rate":          rate,
		"date":          date.UTC().Format(time.RFC3339),
	}))
}
