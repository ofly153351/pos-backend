package p0postgres

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"pos-backend/internal/modules/creditsale"
	"pos-backend/internal/modules/customer"
	"pos-backend/internal/modules/sale"
)

// B08: a credit-sale bill reads every amount from one source — the
// customer's open credit receivables: rows show original/paid/remaining,
// totals are the sum of those rows, and the column heading names credit
// sales (not delivery orders).
func TestPostgresB08CreditSaleBillTotals(t *testing.T) {
	db := openFixture(t)
	app := creditApp(db)
	saleSvc := sale.NewService(sale.NewPostgresRepository(db), customer.NewSaleBenefitResolver(customer.NewPostgresRepository(db)), sale.NewDBSettingsRepo(db))
	h := creditsale.NewHandler(creditsale.NewService(creditsale.NewPostgresRepository(db), saleSvc))
	app.Post("/stores/:storeID/credit-sales/:creditSaleID/payments", h.AddPayment)
	app.Get("/stores/:storeID/credit-sales/:creditSaleID/bill", h.Bill)
	exec(t, db, `INSERT INTO customers(id, store_id, customer_level, full_name, is_active, created_at) VALUES ('cust-loan','a',1,'Borrower',true,now())`)

	first := createCredit(t, app, "credit")
	createCredit(t, app, "credit")
	createCredit(t, app, "loan") // must not appear on a money bill
	request(t, app, "POST", "/stores/a/credit-sales/"+first.ID+"/payments", fiber.MIMEApplicationJSON, []byte(`{"amount":30,"method":"cash"}`), "", 201)

	html := rawBody(t, app, "GET", "/stores/a/credit-sales/"+first.ID+"/bill", "")
	for _, want := range []string{"เอกสารขายเชื่อ", "200.00", "30.00", "170.00"} {
		if !strings.Contains(html, want) {
			t.Fatalf("bill missing %q", want)
		}
	}
	if strings.Contains(html, "ใบส่งสินค้าเลขที่") {
		t.Fatal("credit-sale bill must not label rows as delivery orders")
	}
	if n := strings.Count(html, "เลขที่ CR"); n != 2 {
		t.Fatalf("bill rows = %d, want the 2 open credit sales (loan excluded)", n)
	}
}
