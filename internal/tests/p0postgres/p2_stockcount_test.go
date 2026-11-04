package p0postgres

import (
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v2"
	"pos-backend/internal/modules/stockcount"
)

// B19: the count creator comes from the session (a typed name in the body is
// ignored), its display name is resolved, completion is stamped only by Apply,
// and the applied correction keeps its sign.
func TestPostgresB19StockCountCreatorAndSign(t *testing.T) {
	db, _ := concurrentFixture(t) // Apply needs more than one session
	app := appFor(db)
	h := stockcount.NewHandler(stockcount.NewService(stockcount.NewPostgresRepository(db), db))
	app.Put("/stores/:storeID/stock-count-sessions/:sessionID", h.Save)
	app.Get("/stores/:storeID/stock-count-sessions/:sessionID", h.GetByID)
	app.Post("/stores/:storeID/stock-count-sessions/:sessionID/apply", h.Apply)

	body := `{"id":"cnt-1","name":"P2 count","status":"counting","createdBy":"Somebody Else","completedBy":"Fake","locationId":"loc-a",
		"items":[{"productId":"product-a","name":"P0 product","systemQty":20,"counted":17}]}`
	data := request(t, app, "PUT", "/stores/a/stock-count-sessions/cnt-1", fiber.MIMEApplicationJSON, []byte(body), "", 200)
	var s struct {
		CreatedBy, CreatedByName, CompletedBy string
	}
	_ = json.Unmarshal(data, &s)
	if s.CreatedBy != "p0-user" || s.CreatedByName != "P0 owner" || s.CompletedBy != "" {
		t.Fatalf("creator/completer = %+v, want session user and no client-set completer", s)
	}
	request(t, app, "POST", "/stores/a/stock-count-sessions/cnt-1/apply", fiber.MIMEApplicationJSON, []byte(`{"items":[{"productId":"product-a","countedQty":17}]}`), "", 200)
	assertNumber(t, db, -3, `SELECT quantity_change FROM stock_movements WHERE reference_id = 'cnt-1'`)
	assertNumber(t, db, 17, `SELECT quantity FROM stocks WHERE id = 'stock-a'`)
}
