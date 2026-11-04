package p0postgres

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"pos-backend/internal/modules/document"
)

func documentApp(db *gorm.DB) *fiber.App {
	app := appFor(db)
	h := document.NewHandler(document.NewService(document.NewRepository(db), db))
	app.Post("/stores/:storeID/sales/:saleID/documents", h.CreateFromSale)
	app.Post("/stores/:storeID/documents", h.CreateDocument)
	app.Get("/stores/:storeID/documents", h.ListDocuments)
	app.Get("/stores/:storeID/documents/:docID", h.GetDocument)
	return app
}

type docRow struct {
	ID           string `json:"id"`
	DocumentDate string `json:"document_date"`
}

func listIDs(t *testing.T, app *fiber.App, storeID, from, to string) []string {
	t.Helper()
	data := request(t, app, "GET", "/stores/"+storeID+"/documents?date_from="+from+"&date_to="+to, "", nil, "", 200)
	var out struct {
		Items []docRow `json:"items"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(out.Items))
	for _, it := range out.Items {
		ids = append(ids, it.ID)
	}
	return ids
}

func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// B02: a sale at 00:30 Bangkok (17:30 UTC on the previous day) must produce a
// document dated on the Bangkok business day, and date filters must use that
// day for both new UTC-midnight rows and legacy instant rows.
func TestPostgresB02DocumentBusinessDates(t *testing.T) {
	db := openFixture(t)
	app := documentApp(db)

	s := checkout(t, app, "a", nil, 0, 200, "b02-sale", 201)
	exec(t, db, `UPDATE sales SET sold_at = '2026-11-03 17:30:00+00', created_at = '2026-11-03 17:30:00+00' WHERE id = ?`, s.ID)

	data := request(t, app, "POST", "/stores/a/sales/"+s.ID+"/documents", fiber.MIMEApplicationJSON, []byte(`{"type":"RECEIPT"}`), "", 201)
	var fromSale docRow
	if err := json.Unmarshal(data, &fromSale); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(fromSale.DocumentDate, "2026-11-04") {
		t.Fatalf("document from a 00:30 Bangkok sale dated %s, want 2026-11-04", fromSale.DocumentDate)
	}

	// A legacy row stored as the raw sale instant (pre-fix behaviour).
	// Copy the created row so every NOT NULL column is valid, then give it the
	// raw sale instant that pre-fix code stored.
	exec(t, db, `CREATE TEMP TABLE legacy_doc AS SELECT * FROM documents WHERE id = ?`, fromSale.ID)
	exec(t, db, `UPDATE legacy_doc SET id = 'legacy-doc', document_no = 'LEGACY-1', document_no_full = 'LEGACY-1', document_date = '2026-11-03 17:30:00+00'`)
	exec(t, db, `INSERT INTO documents SELECT * FROM legacy_doc`)

	// A document submitted with an explicit business date.
	body := []byte(`{"type":"QUOTATION","customer_name":"B02","document_date":"2026-11-04","items":[{"product_id":"product-a","quantity":1,"unit_price":100}]}`)
	data = request(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, body, "", 201)
	var explicit docRow
	if err := json.Unmarshal(data, &explicit); err != nil {
		t.Fatal(err)
	}

	sameDay := listIDs(t, app, "a", "2026-11-04", "2026-11-04")
	for _, id := range []string{fromSale.ID, "legacy-doc", explicit.ID} {
		if !contains(sameDay, id) {
			t.Fatalf("document %s missing from the 2026-11-04 filter: %v", id, sameDay)
		}
	}
	if prev := listIDs(t, app, "a", "2026-11-03", "2026-11-03"); len(prev) != 0 {
		t.Fatalf("2026-11-03 filter returned %v, want none", prev)
	}

	// Opening and re-reading must keep the same business day.
	data = request(t, app, "GET", "/stores/a/documents/"+explicit.ID, "", nil, "", 200)
	var reread docRow
	if err := json.Unmarshal(data, &reread); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(reread.DocumentDate, "2026-11-04") {
		t.Fatalf("re-read document_date = %s", reread.DocumentDate)
	}
}
