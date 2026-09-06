package httputil_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/BounkhongDev/bkgo/errs"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/httputil"
)

func TestRespondBadRequest(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return httputil.RespondBadRequest(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}

	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Success || body.Error != httputil.ErrCodeBadRequest {
		t.Fatalf("body = %#v, want BAD_REQUEST error code", body)
	}
}

func TestRespondUnauthorized(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return httputil.RespondUnauthorized(c)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Success || body.Error != httputil.ErrCodeUnauthorized {
		t.Fatalf("body = %#v, want UNAUTHORIZED error code", body)
	}
}

func TestRespondError_AppErrorAndUnexpectedError(t *testing.T) {
	app := fiber.New()
	app.Get("/app-err", func(c *fiber.Ctx) error {
		return httputil.RespondError(c, errs.NotFound("resource not found"))
	})
	app.Get("/internal-err", func(c *fiber.Ctx) error {
		return httputil.RespondError(c, errors.New("db crash"))
	})

	// App error
	req1 := httptest.NewRequest("GET", "/app-err", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", resp1.StatusCode)
	}

	// Unexpected internal error
	req2 := httptest.NewRequest("GET", "/internal-err", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 500 {
		t.Fatalf("status = %d, want 500", resp2.StatusCode)
	}

	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Success || body.Error != httputil.ErrCodeInternalError {
		t.Fatalf("body = %#v, want INTERNAL_ERROR", body)
	}
}
