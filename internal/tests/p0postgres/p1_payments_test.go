package p0postgres

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"pos-backend/internal/modules/document"
)

func payApp(db *gorm.DB) *fiber.App {
	app := docApp(db)
	h := document.NewHandler(document.NewService(document.NewRepository(db), db))
	app.Get("/stores/:storeID/documents/:docID/payments", h.GetPayments)
	app.Post("/stores/:storeID/documents/:docID/payments", h.RecordPayment)
	app.Post("/stores/:storeID/documents/:docID/pay", h.PayInvoice)
	return app
}

type summary struct {
	Total, Paid, Credited, Remaining float64
	Replayed                         bool
	Balances                         []struct {
		DocumentID    string  `json:"document_id"`
		Remaining     float64 `json:"remaining"`
		PaymentStatus string  `json:"payment_status"`
	} `json:"balances"`
}

func pay(t *testing.T, app *fiber.App, id string, amount float64, key string) (int, summary, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/stores/a/documents/"+id+"/payments", strings.NewReader(fmt.Sprintf(`{"amount":%v,"method":"transfer","paid_date":"2026-11-04"}`, amount)))
	req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var env struct {
		Data summary `json:"data"`
	}
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := res.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	_ = json.Unmarshal([]byte(b.String()), &env)
	return res.StatusCode, env.Data, b.String()
}

func docState(t *testing.T, db *gorm.DB, id string) string {
	t.Helper()
	var s string
	if err := db.Raw(`SELECT status || '/' || payment_status FROM documents WHERE id = ?`, id).Scan(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

// B05: confirmed partial payments through the existing ledger, overpayment
// and duplicate protection, one debt per document family, and bills that
// settle their delivery orders' debts.
func TestPostgresB05Payments(t *testing.T) {
	db, schema := concurrentFixture(t)
	app := payApp(db)
	inv := newDoc(t, app, "INVOICE", "") // 100

	st, s, _ := pay(t, app, inv, 30, "k1")
	if st != 201 || s.Remaining != 70 || docState(t, db, inv) != "PENDING/PARTIAL" {
		t.Fatalf("partial pay: %d %+v %s", st, s, docState(t, db, inv))
	}
	if st, s, _ = pay(t, app, inv, 30, "k1"); st != 200 || !s.Replayed || s.Remaining != 70 {
		t.Fatalf("idempotent replay: %d %+v", st, s)
	}
	assertNumber(t, db, 1, `SELECT count(*) FROM invoice_payments WHERE document_id = ?`, inv)
	if st, _, body := pay(t, app, inv, 80, ""); st != 409 || !strings.Contains(body, `"reason":"overpayment"`) || !strings.Contains(body, `"remaining":70`) {
		t.Fatalf("overpayment: %d %s", st, body)
	}

	// Concurrent full payments with different keys: only one fits.
	var wg sync.WaitGroup
	_ = schema
	codes := make([]int, 5)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], _, _ = pay(t, app, inv, 70, fmt.Sprintf("race-%d", i))
		}(i)
	}
	wg.Wait()
	if got := tally(codes); got[201] != 1 || got[409] != 4 {
		t.Fatalf("concurrent payments = %v", got)
	}
	assertNumber(t, db, 100, `SELECT SUM(paid_amount) FROM invoice_payments WHERE document_id = ? AND NOT is_voided`, inv)
	if docState(t, db, inv) != "COMPLETED/PAID" {
		t.Fatalf("settled invoice state = %s", docState(t, db, inv))
	}

	// A delivery order derived from an invoice carries no debt of its own.
	inv2 := newDoc(t, app, "INVOICE", "")
	if st, _, _ := raw(t, app, "POST", "/stores/a/documents/"+inv2+"/convert-to", fiber.MIMEApplicationJSON, []byte(`{"target_type":"DELIVERY_ORDER"}`)); st != 201 {
		t.Fatal(st)
	}
	var do2 string
	_ = db.Raw(`SELECT id FROM documents WHERE source_document_id = ? AND type = 'DELIVERY_ORDER'`, inv2).Scan(&do2).Error
	_, _, sum := raw(t, app, "GET", "/stores/a/documents/"+do2+"/payments", "", nil)
	var viaDO summary
	_ = json.Unmarshal(sum, &viaDO)
	if len(viaDO.Balances) != 1 || viaDO.Balances[0].DocumentID != inv2 || viaDO.Remaining != 100 {
		t.Fatalf("DO debt should be the invoice's: %s", sum)
	}
	if st, s, _ = pay(t, app, do2, 100, ""); st != 201 || s.Remaining != 0 {
		t.Fatalf("pay via DO: %d %+v", st, s)
	}
	if docState(t, db, inv2) != "COMPLETED/PAID" || !strings.HasSuffix(docState(t, db, do2), "/PAID") {
		t.Fatalf("family states inv=%s do=%s", docState(t, db, inv2), docState(t, db, do2))
	}

	// Credit notes reduce the debt.
	inv3 := newDoc(t, app, "INVOICE", "")
	if st, _, _ := raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(creditNote(inv3, "discount", 20))); st != 201 {
		t.Fatal(st)
	}
	if st, _, body := pay(t, app, inv3, 90, ""); st != 409 || !strings.Contains(body, `"remaining":80`) {
		t.Fatalf("credit should reduce remaining to 80: %d %s", st, body)
	}

	// A bill over two standalone delivery orders settles them oldest first.
	doA := newDoc(t, app, "DELIVERY_ORDER", "")
	doB := newDoc(t, app, "DELIVERY_ORDER", "")
	_, _, billData := raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON,
		[]byte(fmt.Sprintf(`{"type":"BILL","customer_name":"P1 customer","document_date":"2026-11-04","source_document_ids":[%q,%q],"items":[{"description":"x","quantity":1,"unit_price":200}]}`, doA, doB)))
	var bill struct{ ID string }
	_ = json.Unmarshal(billData, &bill)
	if st, s, _ = pay(t, app, bill.ID, 150, ""); st != 201 || s.Total != 200 || s.Remaining != 50 {
		t.Fatalf("bill partial: %d %+v", st, s)
	}
	assertNumber(t, db, 150, `SELECT SUM(paid_amount) FROM invoice_payments WHERE document_id IN (?, ?)`, doA, doB)
	if !strings.HasSuffix(docState(t, db, bill.ID), "/PARTIAL") {
		t.Fatalf("bill state = %s", docState(t, db, bill.ID))
	}

	// Cancelled documents cannot be paid, through either endpoint.
	inv4 := newDoc(t, app, "INVOICE", "")
	if st := setStatus(t, app, inv4, "CANCELLED"); st != 200 {
		t.Fatal(st)
	}
	if st, _, _ := pay(t, app, inv4, 10, ""); st != 409 {
		t.Fatalf("paying cancelled = %d", st)
	}
	if st, _, _ := raw(t, app, "POST", "/stores/a/documents/"+inv4+"/pay", "", nil); st != 409 {
		t.Fatalf("one-click pay on cancelled = %d", st)
	}
	assertNumber(t, db, 0, `SELECT count(*) FROM invoice_payments WHERE document_id = ?`, inv4)
	if docState(t, db, inv4) != "CANCELLED/UNPAID" {
		t.Fatalf("cancelled invoice changed: %s", docState(t, db, inv4))
	}
}
