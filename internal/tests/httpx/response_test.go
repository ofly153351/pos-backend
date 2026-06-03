package httpx_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/platform/httpx"
)

func newApp(handler fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
		return c.Status(500).SendString(err.Error())
	}})
	app.Get("/test", handler)
	return app
}

func parseEnvelope(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("failed to parse response body: %v\nbody: %s", err, body)
	}
	return m
}

func TestSuccess(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return httpx.Success(c, fiber.StatusOK, "ok", fiber.Map{"id": "1"})
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	m := parseEnvelope(t, body)

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if m["success"] != true {
		t.Errorf("expected success=true, got %v", m["success"])
	}
	if m["message"] != "ok" {
		t.Errorf("expected message=ok, got %v", m["message"])
	}
}

func TestErr422_SingleField(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return httpx.Err422(c, "name", "name is required")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	m := parseEnvelope(t, body)

	if resp.StatusCode != 422 {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	if m["success"] != false {
		t.Errorf("expected success=false")
	}
	if m["message"] != "validation failed" {
		t.Errorf("expected message='validation failed', got %v", m["message"])
	}
	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("error field not an object: %v", m["error"])
	}
	if errObj["code"] != "VALIDATION_ERROR" {
		t.Errorf("expected code=VALIDATION_ERROR, got %v", errObj["code"])
	}
	fields, ok := errObj["fields"].([]any)
	if !ok || len(fields) == 0 {
		t.Fatalf("expected fields array, got %v", errObj["fields"])
	}
	f := fields[0].(map[string]any)
	if f["field"] != "name" {
		t.Errorf("expected field=name, got %v", f["field"])
	}
	if f["message"] != "name is required" {
		t.Errorf("expected message='name is required', got %v", f["message"])
	}
}

func TestErrNotFound(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return httpx.ErrNotFound(c, "product not found")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	m := parseEnvelope(t, body)

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	errObj := m["error"].(map[string]any)
	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("expected code=NOT_FOUND, got %v", errObj["code"])
	}
}

func TestErrForbidden(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return httpx.ErrForbidden(c, "user cannot operate this store")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestErrConflict(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return httpx.ErrConflict(c, "product is already in use")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestErrInternal(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return httpx.ErrInternal(c)
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	m := parseEnvelope(t, body)

	if resp.StatusCode != 500 {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	errObj := m["error"].(map[string]any)
	if errObj["code"] != "INTERNAL_ERROR" {
		t.Errorf("expected code=INTERNAL_ERROR, got %v", errObj["code"])
	}
}

func TestValidationError_MultipleFields(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return httpx.ValidationError(c,
			httpx.FieldError{Field: "name", Message: "required"},
			httpx.FieldError{Field: "price", Message: "must be >= 0"},
		)
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	m := parseEnvelope(t, body)

	if resp.StatusCode != 422 {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	errObj := m["error"].(map[string]any)
	fields := errObj["fields"].([]any)
	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}
}
