package p0postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"pos-backend/internal/modules/creditsale"
	"pos-backend/internal/modules/customer"
	"pos-backend/internal/modules/sale"
)

func creditApp(db *gorm.DB) *fiber.App {
	app := appFor(db)
	saleSvc := sale.NewService(sale.NewPostgresRepository(db), customer.NewSaleBenefitResolver(customer.NewPostgresRepository(db)), sale.NewDBSettingsRepo(db))
	h := creditsale.NewHandler(creditsale.NewService(creditsale.NewPostgresRepository(db), saleSvc))
	app.Get("/stores/:storeID/credit-sales/summary", h.Summary)
	app.Post("/stores/:storeID/credit-sales", h.Create)
	app.Get("/stores/:storeID/credit-sales/:creditSaleID", h.GetByID)
	app.Post("/stores/:storeID/credit-sales/:creditSaleID/returns", h.ReturnGoods)
	return app
}

type creditView struct {
	ID                   string  `json:"id"`
	Type                 string  `json:"type"`
	CreatedByName        string  `json:"created_by_name"`
	ReceivableAmount     float64 `json:"receivable_amount"`
	CollectedAmount      float64 `json:"collected_amount"`
	ReturnedValue        float64 `json:"returned_value"`
	UnreturnedGoodsValue float64 `json:"unreturned_goods_value"`
	Payments             []struct {
		Kind          string `json:"kind"`
		CreatedByName string `json:"created_by_name"`
	} `json:"payments"`
}

func createCredit(t *testing.T, app *fiber.App, typ string) creditView {
	t.Helper()
	body := `{"type":"` + typ + `","customer_id":"cust-loan","due_date":"2026-11-30","items":[{"product_id":"product-a","quantity":1}]}`
	data := request(t, app, "POST", "/stores/a/credit-sales", fiber.MIMEApplicationJSON, []byte(body), "", 201)
	var v creditView
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// B15: borrowed goods are never money owed. Returns are not payments, the
// money summary and statements exclude loans, and the recorder comes from the
// session user.
func TestPostgresB15LoansAreNotReceivables(t *testing.T) {
	db := openFixture(t)
	app := creditApp(db)
	exec(t, db, `INSERT INTO customers(id, store_id, customer_level, full_name, is_active, created_at) VALUES ('cust-loan','a',1,'Borrower',true,now())`)

	credit := createCredit(t, app, "credit")
	loan := createCredit(t, app, "loan")
	if credit.CreatedByName != "P0 owner" || loan.CreatedByName != "P0 owner" {
		t.Fatalf("recorder must be the session user's name: %q %q", credit.CreatedByName, loan.CreatedByName)
	}
	if credit.ReceivableAmount != 100 || credit.UnreturnedGoodsValue != 0 {
		t.Fatalf("credit split = %+v", credit)
	}
	if loan.ReceivableAmount != 0 || loan.UnreturnedGoodsValue != 100 {
		t.Fatalf("loan split = %+v", loan)
	}

	var sum struct {
		TotalOutstanding    float64 `json:"total_outstanding"`
		LoanOpenCount       int64   `json:"loan_open_count"`
		LoanUnreturnedValue float64 `json:"loan_unreturned_value"`
	}
	_ = json.Unmarshal(request(t, app, "GET", "/stores/a/credit-sales/summary", "", nil, "", 200), &sum)
	if sum.TotalOutstanding != 100 || sum.LoanOpenCount != 1 || sum.LoanUnreturnedValue != 100 {
		t.Fatalf("summary = %+v", sum)
	}

	request(t, app, "POST", "/stores/a/credit-sales/"+loan.ID+"/returns", fiber.MIMEApplicationJSON, []byte(`{"items":[{"product_id":"product-a","quantity":1}]}`), "", 201)
	var after creditView
	_ = json.Unmarshal(request(t, app, "GET", "/stores/a/credit-sales/"+loan.ID, "", nil, "", 200), &after)
	if after.CollectedAmount != 0 || after.ReturnedValue != 100 || after.UnreturnedGoodsValue != 0 {
		t.Fatalf("a return is not a payment: %+v", after)
	}
	if len(after.Payments) != 1 || after.Payments[0].Kind != "return" || after.Payments[0].CreatedByName != "P0 owner" {
		t.Fatalf("timeline = %+v", after.Payments)
	}
	_ = json.Unmarshal(request(t, app, "GET", "/stores/a/credit-sales/summary", "", nil, "", 200), &sum)
	if sum.TotalOutstanding != 100 || sum.LoanUnreturnedValue != 0 {
		t.Fatalf("summary after return = %+v", sum)
	}

	// A money statement lists only the credit sale.
	sc, err := creditsale.NewPostgresRepository(db).StatementContext(context.Background(), "a", credit.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.Sales) != 1 || sc.Sales[0].Type != "credit" {
		t.Fatalf("money statement rows = %d", len(sc.Sales))
	}
}
