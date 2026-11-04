package p0postgres

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"pos-backend/internal/modules/document"
)

func docApp(db *gorm.DB) *fiber.App {
	app := documentApp(db)
	h := document.NewHandler(document.NewService(document.NewRepository(db), db))
	app.Put("/stores/:storeID/documents/:docID/status", h.UpdateDocumentStatus)
	app.Post("/stores/:storeID/documents/bulk", h.BulkAction)
	app.Post("/stores/:storeID/documents/:docID/convert-to", h.Convert)
	return app
}

func newDoc(t *testing.T, app *fiber.App, typ, extra string) string {
	t.Helper()
	body := fmt.Sprintf(`{"type":%q,"customer_name":"P1 customer","document_date":"2026-11-04","items":[{"description":"x","quantity":1,"unit_price":100}]%s}`, typ, extra)
	st, fe, data := raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(body))
	if st != 201 {
		t.Fatalf("create %s: %d %+v %s", typ, st, fe, data)
	}
	var d struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		t.Fatal(err)
	}
	return d.ID
}

func status(t *testing.T, db *gorm.DB, id string) string {
	t.Helper()
	var s string
	if err := db.Raw(`SELECT status FROM documents WHERE id = ?`, id).Scan(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

func setStatus(t *testing.T, app *fiber.App, id, to string) int {
	t.Helper()
	st, _, _ := raw(t, app, "PUT", "/stores/a/documents/"+id+"/status", fiber.MIMEApplicationJSON, []byte(`{"status":"`+to+`"}`))
	return st
}

// B06: single and bulk status updates enforce the same transition rules.
func TestPostgresB06StatusTransitions(t *testing.T) {
	db := openFixture(t)
	app := docApp(db)
	inv := newDoc(t, app, "INVOICE", "")
	if st := setStatus(t, app, inv, "COMPLETED"); st != 200 {
		t.Fatalf("PENDING→COMPLETED = %d", st)
	}
	for _, to := range []string{"PENDING", "CANCELLED", "CONVERTED"} {
		if st := setStatus(t, app, inv, to); st != 409 {
			t.Fatalf("COMPLETED→%s = %d, want 409", to, st)
		}
	}
	cancelled := newDoc(t, app, "INVOICE", "")
	if st := setStatus(t, app, cancelled, "CANCELLED"); st != 200 {
		t.Fatalf("PENDING→CANCELLED = %d", st)
	}
	if st := setStatus(t, app, cancelled, "PENDING"); st != 409 {
		t.Fatalf("CANCELLED→PENDING = %d, want 409", st)
	}
	open := newDoc(t, app, "INVOICE", "")

	body := fmt.Sprintf(`{"action":"SET_STATUS","status":"COMPLETED","ids":[%q,%q,%q,"missing-id"]}`, open, cancelled, inv)
	st, _, data := raw(t, app, "POST", "/stores/a/documents/bulk", fiber.MIMEApplicationJSON, []byte(body))
	if st != 200 {
		t.Fatalf("bulk = %d", st)
	}
	var res struct {
		Updated []string `json:"updated"`
		Skipped []struct {
			ID, Reason, From string
		} `json:"skipped"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatal(err)
	}
	// inv is already COMPLETED: a no-op counts as updated (idempotent).
	if len(res.Updated) != 2 || len(res.Skipped) != 2 {
		t.Fatalf("bulk outcome %s", data)
	}
	if status(t, db, cancelled) != "CANCELLED" || status(t, db, open) != "COMPLETED" {
		t.Fatalf("bulk changed a terminal document: cancelled=%s open=%s", status(t, db, cancelled), status(t, db, open))
	}
	if !strings.Contains(string(data), `"reason":"invalid_transition"`) || !strings.Contains(string(data), `"reason":"not_found"`) {
		t.Fatalf("bulk reasons missing: %s", data)
	}
}

// fireConcurrently runs n identical requests while blocker holds a row lock on
// lockedID, so every request reaches the conversion lock before any proceeds.
func fireConcurrently(t *testing.T, db *gorm.DB, schema string, app *fiber.App, n int, lockedID, method, path, body string) []int {
	t.Helper()
	blocker := db.Begin()
	if blocker.Error != nil {
		t.Fatal(blocker.Error)
	}
	defer blocker.Rollback()
	exec(t, blocker, "SELECT id FROM documents WHERE id = ? FOR UPDATE", lockedID)
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
			res, err := app.Test(req, -1)
			if err != nil {
				codes[i] = -1
				return
			}
			res.Body.Close()
			codes[i] = res.StatusCode
		}(i)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		_ = db.Raw("SELECT count(DISTINCT pid) FROM pg_stat_activity WHERE application_name=? AND wait_event_type='Lock'", schema).Scan(&waiting).Error
		if waiting >= n {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := blocker.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	return codes
}

func tally(codes []int) map[int]int {
	m := map[int]int{}
	for _, c := range codes {
		m[c]++
	}
	return m
}

// B12: converting a quotation creates exactly one invoice even under concurrent
// requests, marks the quotation CONVERTED atomically, and explains repeats.
func TestPostgresB12QuotationConversion(t *testing.T) {
	db, schema := concurrentFixture(t)
	app := docApp(db)
	qt := newDoc(t, app, "QUOTATION", "")

	codes := fireConcurrently(t, db, schema, app, 8, qt, "POST", "/stores/a/documents/"+qt+"/convert-to", `{"target_type":"INVOICE"}`)
	if got := tally(codes); got[201] != 1 || got[409] != 7 {
		t.Fatalf("concurrent conversions = %v, want one 201 and seven 409", got)
	}
	assertNumber(t, db, 1, `SELECT count(*) FROM documents WHERE source_document_id = ? AND type = 'INVOICE'`, qt)
	if s := status(t, db, qt); s != "CONVERTED" {
		t.Fatalf("quotation status = %s, want CONVERTED", s)
	}
	if st := setStatus(t, app, qt, "PENDING"); st != 409 {
		t.Fatalf("CONVERTED→PENDING = %d, want 409", st)
	}
	if body := rawBody(t, app, "POST", "/stores/a/documents/"+qt+"/convert-to", `{"target_type":"INVOICE"}`); !strings.Contains(body, `"reason":"already_converted"`) || !strings.Contains(body, `"document_no":"INV`) {
		t.Fatalf("repeat conversion must name the created invoice: %s", body)
	}

	// A cancelled quotation cannot be converted.
	cancelled := newDoc(t, app, "QUOTATION", "")
	if st := setStatus(t, app, cancelled, "CANCELLED"); st != 200 {
		t.Fatal(st)
	}
	if st, _, _ := raw(t, app, "POST", "/stores/a/documents/"+cancelled+"/convert-to", fiber.MIMEApplicationJSON, []byte(`{"target_type":"INVOICE"}`)); st != 409 {
		t.Fatalf("converting a cancelled quotation = %d, want 409", st)
	}
	assertNumber(t, db, 0, `SELECT count(*) FROM documents WHERE source_document_id = ?`, cancelled)
}

// B07: cancelled delivery orders cannot be billed, a delivery order is billed
// once (also under concurrency), and a rejected bill leaves no partial chain.
func TestPostgresB07BillingDeliveryOrders(t *testing.T) {
	db, schema := concurrentFixture(t)
	app := docApp(db)
	before := number(t, db, `SELECT count(*) FROM documents WHERE type = 'BILL'`)

	cancelledDO := newDoc(t, app, "DELIVERY_ORDER", "")
	if st := setStatus(t, app, cancelledDO, "CANCELLED"); st != 200 {
		t.Fatal(st)
	}
	st, _, _ := raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON,
		[]byte(fmt.Sprintf(`{"type":"BILL","customer_name":"P1 customer","document_date":"2026-11-04","source_document_ids":[%q],"items":[{"description":"x","quantity":1,"unit_price":100}]}`, cancelledDO)))
	if st != 409 {
		t.Fatalf("billing a cancelled DO = %d, want 409", st)
	}
	assertNumber(t, db, before, `SELECT count(*) FROM documents WHERE type = 'BILL'`)

	do := newDoc(t, app, "DELIVERY_ORDER", "")
	bill := fmt.Sprintf(`{"type":"BILL","customer_name":"P1 customer","document_date":"2026-11-04","source_document_ids":[%q],"items":[{"description":"x","quantity":1,"unit_price":100}]}`, do)
	codes := fireConcurrently(t, db, schema, app, 6, do, "POST", "/stores/a/documents", bill)
	if got := tally(codes); got[201] != 1 || got[409] != 5 {
		t.Fatalf("concurrent billing = %v, want one 201 and five 409", got)
	}
	assertNumber(t, db, 1, `SELECT count(*) FROM bill_delivery_order_links WHERE delivery_order_id = ?`, do)
	assertNumber(t, db, before+1, `SELECT count(*) FROM documents WHERE type = 'BILL'`)
	// The bill picker lists only billable delivery orders.
	fresh := newDoc(t, app, "DELIVERY_ORDER", "")
	_, _, list := raw(t, app, "GET", "/stores/a/documents?type=DELIVERY_ORDER&billable=true", "", nil)
	if !strings.Contains(string(list), fresh) || strings.Contains(string(list), do) || strings.Contains(string(list), cancelledDO) {
		t.Fatalf("billable list = %s (fresh %s, billed %s, cancelled %s)", list, fresh, do, cancelledDO)
	}
	// Conflicts carry machine reasons for localized messages.
	if body := rawBody(t, app, "POST", "/stores/a/documents", bill); !strings.Contains(body, `"reason":"already_converted"`) || !strings.Contains(body, `"document_no"`) {
		t.Fatalf("already-billed conflict body = %s", body)
	}
	// No bill without its link (no partial chain).
	assertNumber(t, db, 0, `SELECT count(*) FROM documents d WHERE d.type = 'BILL' AND NOT EXISTS (SELECT 1 FROM bill_delivery_order_links l WHERE l.bill_document_id = d.id)`)
}

func rawBody(t *testing.T, app *fiber.App, method, path, body string) string {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := res.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return b.String()
}
