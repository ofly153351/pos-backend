package p0postgres

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func creditNote(source, reason string, amount int) string {
	src := ""
	if source != "" {
		src = fmt.Sprintf(`,"source_document_id":%q`, source)
	}
	rsn := ""
	if reason != "" {
		rsn = fmt.Sprintf(`,"reason":%q`, reason)
	}
	return fmt.Sprintf(`{"type":"CREDIT_NOTE","customer_name":"P1 customer","document_date":"2026-11-04"%s%s,"items":[{"description":"credit","quantity":1,"unit_price":%d}]}`, src, rsn, amount)
}

// B13: credit notes need an original and a reason, and cumulative credit
// (including concurrent requests) never exceeds the original total.
func TestPostgresB13CreditNotes(t *testing.T) {
	db, schema := concurrentFixture(t)
	app := docApp(db)
	inv := newDoc(t, app, "INVOICE", "") // total 100

	st, fe, _ := raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(creditNote("", "damaged", 10)))
	expectField(t, st, fe, "source_document_id", "required")
	st, fe, _ = raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(creditNote(inv, "", 10)))
	expectField(t, st, fe, "reason", "required")
	qt := newDoc(t, app, "QUOTATION", "")
	st, fe, _ = raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(creditNote(qt, "x", 10)))
	expectField(t, st, fe, "source_document_id", "credit_source_invalid")

	if st, fe, _ = raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(creditNote(inv, "damaged goods", 60))); st != 201 {
		t.Fatalf("partial credit 60: %d %+v", st, fe)
	}
	body := rawBody(t, app, "POST", "/stores/a/documents", creditNote(inv, "more", 50))
	if !strings.Contains(body, `"reason":"credit_exceeds_original"`) || !strings.Contains(body, `"remaining":40`) {
		t.Fatalf("over-credit body = %s", body)
	}
	if st, _, _ = raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(creditNote(inv, "rest", 40))); st != 201 {
		t.Fatalf("remaining credit 40: %d", st)
	}
	assertNumber(t, db, 100, `SELECT COALESCE(SUM(total_amount),0) FROM documents WHERE type='CREDIT_NOTE' AND source_document_id = ?`, inv)
	assertNumber(t, db, 2, `SELECT count(*) FROM documents WHERE type='CREDIT_NOTE' AND source_document_id = ? AND credit_reason IS NOT NULL`, inv)

	// Concurrent credits against one original: only one fits.
	inv2 := newDoc(t, app, "INVOICE", "")
	codes := fireConcurrently(t, db, schema, app, 5, inv2, "POST", "/stores/a/documents", creditNote(inv2, "race", 60))
	if got := tally(codes); got[201] != 1 || got[409] != 4 {
		t.Fatalf("concurrent credits = %v, want one 201 and four 409", got)
	}
	assertNumber(t, db, 60, `SELECT COALESCE(SUM(total_amount),0) FROM documents WHERE type='CREDIT_NOTE' AND source_document_id = ?`, inv2)

	// Full-amount conversion needs a reason and respects the same ceiling.
	inv3 := newDoc(t, app, "INVOICE", "")
	st, fe, _ = raw(t, app, "POST", "/stores/a/documents/"+inv3+"/convert-to", fiber.MIMEApplicationJSON, []byte(`{"target_type":"CREDIT_NOTE"}`))
	expectField(t, st, fe, "reason", "required")
	if st, _, _ = raw(t, app, "POST", "/stores/a/documents/"+inv3+"/convert-to", fiber.MIMEApplicationJSON, []byte(`{"target_type":"CREDIT_NOTE","reason":"order cancelled"}`)); st != 201 {
		t.Fatalf("full credit conversion = %d", st)
	}
	if st, _, _ = raw(t, app, "POST", "/stores/a/documents/"+inv3+"/convert-to", fiber.MIMEApplicationJSON, []byte(`{"target_type":"CREDIT_NOTE","reason":"again"}`)); st != 409 {
		t.Fatalf("second full credit = %d, want 409", st)
	}

	// A cancelled original cannot be credited.
	inv4 := newDoc(t, app, "INVOICE", "")
	if st := setStatus(t, app, inv4, "CANCELLED"); st != 200 {
		t.Fatal(st)
	}
	if st, _, _ = raw(t, app, "POST", "/stores/a/documents", fiber.MIMEApplicationJSON, []byte(creditNote(inv4, "x", 10))); st != 409 {
		t.Fatalf("credit on cancelled original = %d, want 409", st)
	}
	// Credit notes never touch stock.
	assertNumber(t, db, 20, `SELECT quantity FROM stocks WHERE id = 'stock-a'`)
}
