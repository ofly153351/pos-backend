package p0postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"pos-backend/internal/modules/document"
)

// B09 + B11: statements and the pending summary use one debt rule — an
// invoice and its derived delivery order count once with payments applied,
// credit receivables are included, loans are excluded.
func TestPostgresB09B11SharedDebt(t *testing.T) {
	db := openFixture(t)
	app := payApp(db)
	cr := creditApp(db)
	exec(t, db, `INSERT INTO customers(id, store_id, customer_level, full_name, is_active, created_at) VALUES ('cust-loan','a',1,'Borrower',true,now())`)

	body := `{"type":"INVOICE","customer_id":"cust-loan","document_date":"2026-11-04","items":[{"description":"x","quantity":1,"unit_price":100}]}`
	_, _, data := raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(body))
	var inv struct{ ID string }
	_ = json.Unmarshal(data, &inv)
	if st, _, _ := raw(t, app, "POST", "/stores/a/documents/"+inv.ID+"/convert-to", fiber.MIMEApplicationJSON, []byte(`{"target_type":"DELIVERY_ORDER"}`)); st != 201 {
		t.Fatal(st)
	}
	if st, _, _ := pay(t, app, inv.ID, 30, ""); st != 201 {
		t.Fatal(st)
	}
	createCredit(t, cr, "credit")
	createCredit(t, cr, "loan")

	svc := document.NewService(document.NewRepository(db), db)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Now().AddDate(0, 0, 1)
	rows, err := svc.StatementRows(context.Background(), "a", "cust-loan", from, to)
	if err != nil {
		t.Fatal(err)
	}
	var balance float64
	for _, r := range rows {
		balance += r.Balance
	}
	if len(rows) != 2 || balance != 170 {
		t.Fatalf("statement rows=%d balance=%v, want invoice(70 once)+credit sale(100)", len(rows), balance)
	}

	_, _, list := raw(t, app, "GET", "/stores/a/documents?customer_id=cust-loan", "", nil)
	var res struct {
		Stats struct {
			PendingPayment int64   `json:"pending_payment"`
			PendingAmount  float64 `json:"pending_amount"`
		} `json:"stats"`
	}
	_ = json.Unmarshal(list, &res)
	if res.Stats.PendingPayment != 1 || res.Stats.PendingAmount != 70 {
		t.Fatalf("pending summary = %+v (%s)", res.Stats, fmt.Sprint(len(list)))
	}
}
