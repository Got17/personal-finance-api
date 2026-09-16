package currency

import (
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"
)

type CurrencyHandler struct{}

func NewCurrencyHandler() *CurrencyHandler { return &CurrencyHandler{} }

func (h *CurrencyHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/currencies", h.list)
}

func (h *CurrencyHandler) list(c *fiber.Ctx) error {
	return c.JSON(response.Success(SupportedCurrencies()))
}
