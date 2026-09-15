package currency

import (
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"
)

// CurrencyHandler handles HTTP requests for the currency resource.
type CurrencyHandler struct{}

// NewCurrencyHandler constructs a new CurrencyHandler.
func NewCurrencyHandler() *CurrencyHandler { return &CurrencyHandler{} }

// RegisterRoutes mounts public currency routes onto the given router group.
func (h *CurrencyHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/currencies", h.list)
}

// list returns the fixed ordered list of supported currencies.
func (h *CurrencyHandler) list(c *fiber.Ctx) error {
	return c.JSON(response.Success(SupportedCurrencies()))
}
