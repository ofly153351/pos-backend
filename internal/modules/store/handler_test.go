package store

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestParseUpdateStoreRequestTaxIDOtherEncodings(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		present                 bool
	}{
		{name: "urlencoded clear", contentType: fiber.MIMEApplicationForm, body: "tax_id=", present: true},
		{name: "urlencoded omitted", contentType: fiber.MIMEApplicationForm, body: "name=Test", present: false},
		{name: "json clear", contentType: fiber.MIMEApplicationJSON, body: `{"tax_id":""}`, present: true},
		{name: "json omitted", contentType: fiber.MIMEApplicationJSON, body: `{"name":"Test"}`, present: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Put("/store", func(c *fiber.Ctx) error {
				parsed, err := parseUpdateStoreRequest(c)
				if err != nil {
					return err
				}
				return c.JSON(parsed)
			})
			req := httptest.NewRequest("PUT", "/store", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != fiber.StatusOK {
				t.Fatalf("status = %d, want 200", res.StatusCode)
			}
			var parsed UpdateStoreRequest
			if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
				t.Fatal(err)
			}
			if tc.present {
				if parsed.TaxID == nil || *parsed.TaxID != "" {
					t.Fatalf("expected explicit empty tax_id, got %v", parsed.TaxID)
				}
			} else if parsed.TaxID != nil {
				t.Fatalf("expected omitted tax_id, got %q", *parsed.TaxID)
			}
		})
	}
}

func TestParseUpdateStoreRequestTaxID(t *testing.T) {
	for _, tc := range []struct {
		name    string
		present bool
		value   string
	}{
		{name: "explicit empty clears tax ID", present: true, value: ""},
		{name: "omitted preserves tax ID", present: false},
		{name: "provided tax ID is retained", present: true, value: "provided-value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if err := writer.WriteField("name", "Test store"); err != nil {
				t.Fatal(err)
			}
			if tc.present {
				if err := writer.WriteField("tax_id", tc.value); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}

			app := fiber.New()
			app.Put("/store", func(c *fiber.Ctx) error {
				parsed, err := parseUpdateStoreRequest(c)
				if err != nil {
					return err
				}
				return c.JSON(parsed)
			})
			req := httptest.NewRequest("PUT", "/store", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != fiber.StatusOK {
				data, _ := io.ReadAll(res.Body)
				t.Fatalf("status = %d, body = %s", res.StatusCode, data)
			}
			var parsed UpdateStoreRequest
			if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
				t.Fatal(err)
			}
			if tc.present {
				if parsed.TaxID == nil {
					t.Fatal("explicit tax_id must not be treated as an omitted field")
				}
				if *parsed.TaxID != tc.value {
					t.Fatalf("tax_id = %q, want %q", *parsed.TaxID, tc.value)
				}
			} else if parsed.TaxID != nil {
				t.Fatalf("omitted tax_id = %q, want nil", *parsed.TaxID)
			}
		})
	}
}
