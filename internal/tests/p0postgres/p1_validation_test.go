package p0postgres

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"pos-backend/internal/modules/customer"
	"pos-backend/internal/modules/product"
)

type noImages struct{}

func (noImages) SaveProductImage(*multipart.FileHeader) (string, error) { return "", nil }
func (noImages) DeleteProductImage(string) error                        { return nil }

func validationApp(db *gorm.DB) *fiber.App {
	app := appFor(db)
	ch := customer.NewHandler(customer.NewService(customer.NewPostgresRepository(db)))
	app.Post("/stores/:storeID/customers", ch.Create)
	app.Patch("/stores/:storeID/customers/:customerID", ch.Update)
	ph := product.NewHandler(product.NewService(product.NewPostgresRepository(db), noImages{}, db))
	app.Post("/stores/:storeID/products", ph.Create)
	return app
}

type fieldErr struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// raw sends a request and returns status plus the first field error.
func raw(t *testing.T, app *fiber.App, method, path, contentType string, body []byte) (int, fieldErr, json.RawMessage) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	var env struct {
		Data  json.RawMessage `json:"data"`
		Error struct {
			Fields []fieldErr `json:"fields"`
		} `json:"error"`
	}
	_ = json.Unmarshal(data, &env)
	var fe fieldErr
	if len(env.Error.Fields) > 0 {
		fe = env.Error.Fields[0]
	}
	return res.StatusCode, fe, env.Data
}

func form(t *testing.T, fields map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), w.FormDataContentType()
}

func expectField(t *testing.T, status int, fe fieldErr, wantField, wantCode string) {
	t.Helper()
	if status != 422 || fe.Field != wantField || fe.Code != wantCode {
		t.Fatalf("got %d %+v, want 422 %s/%s", status, fe, wantField, wantCode)
	}
}

func storeTaxID(t *testing.T, db *gorm.DB, id string) string {
	t.Helper()
	var v string
	if err := db.Raw(`SELECT COALESCE(tax_id,'') FROM stores WHERE id = ?`, id).Scan(&v).Error; err != nil {
		t.Fatal(err)
	}
	return v
}

// B10 + B16: invalid direct API requests fail with coded field errors, cleared
// values persist, valid values are normalized, and pre-existing invalid data
// does not block unrelated edits.
func TestPostgresB10B16Validation(t *testing.T) {
	db := openFixture(t)
	app := validationApp(db)

	// ── store tax ID / phone ──
	body, ct := form(t, map[string]string{"tax_id": "123"})
	st, fe, _ := raw(t, app, "PUT", "/stores/a", ct, body)
	expectField(t, st, fe, "tax_id", "tax_id_invalid")
	body, ct = form(t, map[string]string{"tax_id": "0105536000315"}) // bad check digit
	st, fe, _ = raw(t, app, "PUT", "/stores/a", ct, body)
	expectField(t, st, fe, "tax_id", "tax_id_invalid")
	body, ct = form(t, map[string]string{"phone": "12ab"})
	st, fe, _ = raw(t, app, "PUT", "/stores/a", ct, body)
	expectField(t, st, fe, "phone", "phone_invalid")
	if got := storeTaxID(t, db, "a"); got != "1234567890123" {
		t.Fatalf("rejected writes changed tax_id to %q", got)
	}

	// Legacy invalid value (fixture 1234567890123 fails the checksum) must not
	// block an unrelated edit, even when the unchanged value is resent.
	body, ct = form(t, map[string]string{"name": "P0 a renamed", "tax_id": "1234567890123"})
	if st, fe, _ = raw(t, app, "PUT", "/stores/a", ct, body); st != 200 {
		t.Fatalf("unrelated edit with legacy tax ID: %d %+v", st, fe)
	}

	body, ct = form(t, map[string]string{"tax_id": "0-1055-36000-31-3"})
	if st, fe, _ = raw(t, app, "PUT", "/stores/a", ct, body); st != 200 {
		t.Fatalf("valid tax ID rejected: %d %+v", st, fe)
	}
	if got := storeTaxID(t, db, "a"); got != "0105536000313" {
		t.Fatalf("tax ID not normalized: %q", got)
	}
	body, ct = form(t, map[string]string{"tax_id": ""})
	if st, _, _ = raw(t, app, "PUT", "/stores/a", ct, body); st != 200 {
		t.Fatalf("clearing tax ID: %d", st)
	}
	if got := storeTaxID(t, db, "a"); got != "" {
		t.Fatalf("cleared tax ID read back as %q", got)
	}

	// ── customer ──
	st, fe, _ = raw(t, app, "POST", "/stores/a/customers", fiber.MIMEApplicationJSON, []byte(`{"full_name":"C","phone":"abc"}`))
	expectField(t, st, fe, "phone", "phone_invalid")
	st, fe, _ = raw(t, app, "POST", "/stores/a/customers", fiber.MIMEApplicationJSON, []byte(`{"full_name":" "}`))
	expectField(t, st, fe, "full_name", "required")
	st, _, data := raw(t, app, "POST", "/stores/a/customers", fiber.MIMEApplicationJSON, []byte(`{"full_name":"C","phone":"081-234-5678","tax_id":"3100600445597"}`))
	if st != 201 {
		t.Fatalf("valid customer: %d", st)
	}
	var c struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(data, &c)
	st, fe, _ = raw(t, app, "PATCH", "/stores/a/customers/"+c.ID, fiber.MIMEApplicationJSON, []byte(`{"tax_id":"999"}`))
	expectField(t, st, fe, "tax_id", "tax_id_invalid")
	if st, _, _ = raw(t, app, "PATCH", "/stores/a/customers/"+c.ID, fiber.MIMEApplicationJSON, []byte(`{"tax_id":""}`)); st != 200 {
		t.Fatalf("clearing customer tax ID: %d", st)
	}
	assertNumber(t, db, 1, `SELECT count(*) FROM customers WHERE id = ? AND COALESCE(tax_id, '') = ''`, c.ID)

	// ── product prices ──
	body, ct = form(t, map[string]string{"name": "P", "base_price": "100", "cost_price": "-1", "unit_id": "unit-a"})
	st, fe, _ = raw(t, app, "POST", "/stores/a/products", ct, body)
	expectField(t, st, fe, "cost_price", "must_not_be_negative")
	body, ct = form(t, map[string]string{"name": "P", "base_price": "-5", "unit_id": "unit-a"})
	st, fe, _ = raw(t, app, "POST", "/stores/a/products", ct, body)
	expectField(t, st, fe, "base_price", "must_not_be_negative")
	// Owner policy: below-cost selling price is a UI warning, not a backend block.
	body, ct = form(t, map[string]string{"name": "Below cost", "base_price": "30", "cost_price": "40", "unit_id": "unit-a"})
	if st, fe, _ = raw(t, app, "POST", "/stores/a/products", ct, body); st != 201 {
		t.Fatalf("below-cost product rejected: %d %+v", st, fe)
	}
	if strings.TrimSpace(fe.Code) != "" {
		t.Fatalf("unexpected field error on success: %+v", fe)
	}
}
