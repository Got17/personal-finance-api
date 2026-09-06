package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	app := newApp("personal-finance-api")

	response, err := app.Test(httptest.NewRequest("GET", "/v1/health", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Status string `json:"status"`
			App    string `json:"app"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success || body.Data.Status != "ok" || body.Data.App != "personal-finance-api" {
		t.Fatalf("body = %#v, want success with status ok and the configured app name", body)
	}
}
