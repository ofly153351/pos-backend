package p0postgres

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"pos-backend/internal/modules/document"
)

// B03: tax invoices require a valid seller tax ID and address and a buyer
// name; a juristic buyer also needs a branch. Buyer tax ID and branch are
// snapshotted on the document and printed together.
func TestPostgresB03TaxInvoiceRequirements(t *testing.T) {
	db := openFixture(t)
	app := docApp(db)
	h := document.NewHandler(document.NewService(document.NewRepository(db), db))
	app.Get("/stores/:storeID/documents/:docID/print", h.PrintDocument)

	taxInvoice := func(extra string) (int, fieldErr, json.RawMessage) {
		body := `{"type":"TAX_INVOICE","document_date":"2026-11-04","items":[{"description":"x","quantity":1,"unit_price":100}],"vat_rate":7` + extra + `}`
		return raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(body))
	}

	// Fixture store tax ID 1234567890123 fails the check digit; no address.
	st, fe, _ := taxInvoice(`,"customer_name":"Buyer"`)
	expectField(t, st, fe, "seller_tax_id", "tax_id_invalid")
	exec(t, db, `UPDATE stores SET tax_id = '0105536000313' WHERE id = 'a'`)
	st, fe, _ = taxInvoice(`,"customer_name":"Buyer"`)
	expectField(t, st, fe, "seller_address", "required")
	exec(t, db, `UPDATE stores SET address = '1 Test Rd, Bangkok' WHERE id = 'a'`)

	// Juristic buyer (TIN starts with 0) without branch → rejected.
	st, fe, _ = taxInvoice(`,"customer_name":"Buyer Co","customer_tax_id":"0105536000313"`)
	expectField(t, st, fe, "customer_branch", "required")
	st, fe, _ = taxInvoice(`,"customer_name":"Buyer Co","customer_tax_id":"999"`)
	expectField(t, st, fe, "customer_tax_id", "tax_id_invalid")
	assertNumber(t, db, 0, `SELECT count(*) FROM documents WHERE type = 'TAX_INVOICE'`)

	st, fe, data := taxInvoice(`,"customer_name":"Buyer Co","customer_tax_id":"0-1055-36000-31-3","customer_branch":"สำนักงานใหญ่"`)
	if st != 201 {
		t.Fatalf("complete juristic tax invoice: %d %+v", st, fe)
	}
	var doc struct {
		ID             string `json:"id"`
		CustomerTaxID  string `json:"customer_tax_id"`
		CustomerBranch string `json:"customer_branch"`
	}
	_ = json.Unmarshal(data, &doc)
	if doc.CustomerTaxID != "0105536000313" || doc.CustomerBranch != "สำนักงานใหญ่" {
		t.Fatalf("buyer snapshot = %+v", doc)
	}
	html := rawBody(t, app, "GET", "/stores/a/documents/"+doc.ID+"/print", "")
	if !strings.Contains(html, "0105536000313 สาขา สำนักงานใหญ่") {
		t.Fatalf("printed tax invoice lacks buyer tax ID + branch")
	}

	// Individual buyer without a tax ID is allowed.
	if st, fe, _ = taxInvoice(`,"customer_name":"Walk-in"`); st != 201 {
		t.Fatalf("individual buyer: %d %+v", st, fe)
	}

	// A registered customer's tax ID/branch are snapshotted automatically.
	exec(t, db, `INSERT INTO customers(id, store_id, customer_level, full_name, tax_id, branch, is_active, created_at) VALUES ('c-co','a',1,'Registered Co','3100600445597','00001',true,now())`)
	st, fe, data = raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON,
		[]byte(`{"type":"TAX_INVOICE","customer_id":"c-co","document_date":"2026-11-04","items":[{"description":"x","quantity":1,"unit_price":100}]}`))
	if st != 201 {
		t.Fatalf("registered customer tax invoice: %d %+v", st, fe)
	}
	_ = json.Unmarshal(data, &doc)
	if doc.CustomerTaxID != "3100600445597" || doc.CustomerBranch != "00001" {
		t.Fatalf("registered buyer snapshot = %+v", doc)
	}
}
