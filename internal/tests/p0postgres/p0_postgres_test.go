// Package p0postgres verifies P0 boundaries against real, disposable PostgreSQL.
// No application listener, SQL mock, production DSN, or external storage is used.
package p0postgres

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/sale"
	"pos-backend/internal/modules/store"
)

// openFixture refuses normal databases before issuing any DDL. Each test gets a
// unique schema, applies the actual repository migrations, and drops only that
// schema. The database itself must be explicitly marked disposable by its owner.
func openFixture(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("POS_P0_TEST_DSN")
	if dsn == "" {
		t.Skip("set POS_P0_TEST_DSN to an explicitly marked disposable PostgreSQL database")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme != "postgres" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.Port() == "5432" || !strings.HasPrefix(strings.TrimPrefix(u.Path, "/"), "pos_p0_disposable_") {
		t.Fatal("POS_P0_TEST_DSN must be postgres:// on 127.0.0.1 with a non-5432 explicit port and database pos_p0_disposable_*")
	}
	q := u.Query()
	if len(q) != 1 || q.Get("sslmode") != "disable" {
		t.Fatal("only sslmode=disable is allowed in test DSN query")
	}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	var marker string
	if err := db.Raw("SELECT COALESCE(shobj_description(oid, 'pg_database'), '') FROM pg_database WHERE datname = current_database()").Scan(&marker).Error; err != nil {
		t.Fatal(err)
	}
	if marker != "POS P0 DISPOSABLE TEST DATABASE" {
		t.Fatal("database must have COMMENT ON DATABASE ... IS 'POS P0 DISPOSABLE TEST DATABASE'")
	}
	schema := fmt.Sprintf("p0_%d", time.Now().UnixNano())
	exec(t, db, "CREATE SCHEMA "+schema)
	// Pin this pool to one session: SET search_path must cover every transaction.
	raw.SetMaxOpenConns(1)
	raw.SetMaxIdleConns(1)
	exec(t, db, "SET search_path TO "+schema)
	t.Cleanup(func() {
		if err := db.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	files, err := filepath.Glob("../../../init-db/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatal("repository migrations not found", err)
	}
	applied := 0
	for _, f := range files {
		if filepath.Base(f) == "002_seed.sql" {
			continue
		} // isolated fixtures, never demo/user data
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(string(sql)).Error; err != nil {
			t.Fatalf("migration %s: %v", filepath.Base(f), err)
		}
		applied++
	}
	t.Logf("applied %d repository schema migrations (002 demo seed excluded), isolated schema %s", applied, schema)
	exec(t, db, `INSERT INTO users(id,full_name,email,password_hash,role) VALUES ('p0-user','P0 owner','p0@example.invalid','not-a-login','owner')`)
	for _, s := range []string{"a", "b"} {
		exec(t, db, `INSERT INTO stores(id,owner_user_id,name,tax_id) VALUES (?, 'p0-user', ?, '1234567890123')`, s, "P0 "+s)
		exec(t, db, `INSERT INTO store_receipt_settings(id,store_id,tax_mode,vat_rate) VALUES (?, ?, 'none', 0)`, "settings-"+s, s)
		exec(t, db, `INSERT INTO warehouses(id,store_id,name,is_default) VALUES (?, ?, 'P0 warehouse', true)`, "wh-"+s, s)
		exec(t, db, `INSERT INTO locations(id,store_id,warehouse_id,name,is_sale_point,is_default_sale) VALUES (?, ?, ?, 'P0 counter',true,true)`, "loc-"+s, s, "wh-"+s)
		exec(t, db, `INSERT INTO product_units(id,store_id,name) VALUES (?,?,'piece')`, "unit-"+s, s)
		exec(t, db, `INSERT INTO products(id,store_id,product_unit_id,name,base_price,cost_price) VALUES (?, ?, ?, 'P0 product',100,40)`, "product-"+s, s, "unit-"+s)
		exec(t, db, `INSERT INTO stocks(id,store_id,product_id,location_id,quantity) VALUES (?, ?, ?, ?, 20)`, "stock-"+s, s, "product-"+s, "loc-"+s)
	}
	return db
}
func exec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}
func number(t *testing.T, db *gorm.DB, sql string, args ...any) float64 {
	t.Helper()
	var n float64
	if err := db.Raw(sql, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}
func assertNumber(t *testing.T, db *gorm.DB, want float64, sql string, args ...any) {
	t.Helper()
	if got := number(t, db, sql, args...); got != want {
		t.Fatalf("%s: got %v want %v", sql, got, want)
	}
}
func appFor(db *gorm.DB) *fiber.App {
	app := fiber.New()
	// This is an in-process handler/service/repository test. Authentication and
	// membership middleware are intentionally not under test; inject an owner.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_claims", auth.Claims{UserID: "p0-user", Role: "owner"})
		c.SetUserContext(auth.WithStoreAccess(c.UserContext(), auth.StoreAccess{StoreID: c.Params("storeID"), Role: "owner"}))
		return c.Next()
	})
	sh := store.NewHandler(store.NewService(store.NewPostgresRepository(db), nil))
	app.Put("/stores/:storeID", sh.Update)
	app.Get("/stores/:storeID", sh.GetByID)
	ch := sale.NewHandler(sale.NewService(sale.NewPostgresRepository(db), nil, sale.NewDBSettingsRepo(db)))
	app.Post("/stores/:storeID/sales", ch.Create)
	app.Get("/stores/:storeID/sales/:saleID", ch.GetByID)
	return app
}
func request(t *testing.T, app *fiber.App, method, path, contentType string, body []byte, key string, want int) json.RawMessage {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != want {
		t.Fatalf("%s %s request=%s status=%d want=%d response=%s", method, path, body, res.StatusCode, want, data)
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data
}
func TestPostgresB10TaxIDRoundTrip(t *testing.T) {
	db := openFixture(t)
	app := appFor(db)
	cases := []struct{ name, ct, body, want string }{
		{"json_set", fiber.MIMEApplicationJSON, `{"tax_id":" 9876543210121 "}`, "9876543210121"},
		{"json_omitted", fiber.MIMEApplicationJSON, `{"phone":"021234555"}`, "9876543210121"},
		{"json_empty", fiber.MIMEApplicationJSON, `{"tax_id":""}`, ""},
		{"form_set", fiber.MIMEApplicationForm, "tax_id=1111111111119", "1111111111119"},
		{"form_omitted", fiber.MIMEApplicationForm, "phone=021234666", "1111111111119"},
		{"form_empty", fiber.MIMEApplicationForm, "tax_id=", ""},
		{"multipart_set", "multipart", "2222222222227", "2222222222227"},
		{"multipart_omitted", "multipart-omit", "", "2222222222227"},
		{"multipart_empty", "multipart", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ct, body := tc.ct, []byte(tc.body)
			if strings.HasPrefix(ct, "multipart") {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				if ct == "multipart" {
					if err := w.WriteField("tax_id", tc.body); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := w.WriteField("phone", "021234777"); err != nil {
						t.Fatal(err)
					}
				}
				if err := w.Close(); err != nil {
					t.Fatal(err)
				}
				ct = w.FormDataContentType()
				body = b.Bytes()
			}
			updated := request(t, app, "PUT", "/stores/a", ct, body, "", 200)
			fetched := request(t, app, "GET", "/stores/a", "", nil, "", 200)
			for _, data := range []json.RawMessage{updated, fetched} {
				var s store.Store
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatal(err)
				}
				if s.TaxID != tc.want {
					t.Fatalf("readback tax_id=%q want %q", s.TaxID, tc.want)
				}
			}
			var raw string
			if err := db.Raw("SELECT tax_id FROM stores WHERE id='a'").Scan(&raw).Error; err != nil {
				t.Fatal(err)
			}
			if raw != tc.want {
				t.Fatalf("persisted tax_id=%q want %q", raw, tc.want)
			}
			var other string
			if err := db.Raw("SELECT tax_id FROM stores WHERE id='b'").Scan(&other).Error; err != nil {
				t.Fatal(err)
			}
			if other != "1234567890123" {
				t.Fatal("other store modified")
			}
		})
	}
}
func promo(t *testing.T, db *gorm.DB, id, storeID, kind, data string) {
	t.Helper()
	exec(t, db, `INSERT INTO promotions(id,store_id,name,type,status,data) VALUES (?,?,'P0 campaign',?,'active',?)`, id, storeID, kind, data)
}
func checkout(t *testing.T, app *fiber.App, storeID string, ids []string, discount, paid float64, key string, want int) sale.Sale {
	t.Helper()
	body, err := json.Marshal(map[string]any{"items": []map[string]any{{"product_id": "product-" + storeID, "quantity": 2}}, "payment_method": "cash", "paid_amount": paid, "promo_discount": discount, "promotion_ids": ids})
	if err != nil {
		t.Fatal(err)
	}
	data := request(t, app, "POST", "/stores/"+storeID+"/sales", fiber.MIMEApplicationJSON, body, key, want)
	var s sale.Sale
	if want == 201 {
		if err := json.Unmarshal(data, &s); err != nil {
			t.Fatal(err)
		}
	}
	return s
}
func TestPostgresB04CheckoutDuplicatesIsolation(t *testing.T) {
	db := openFixture(t)
	app := appFor(db)
	promo(t, db, "valid", "a", "percentage", `{"percentOff":25}`)
	promo(t, db, "foreign", "b", "percentage", `{"percentOff":100}`)
	promo(t, db, "invalid", "a", "percentage", `{"percentOff":-25}`)
	s := checkout(t, app, "a", []string{"valid", "valid", "foreign", "invalid"}, 200, 150, "same-key", 201)
	if s.BillDiscountAmount != 50 || s.TotalAmount != 150 {
		t.Fatalf("checkout=%+v", s)
	}
	data := request(t, app, "GET", "/stores/a/sales/"+s.ID, "", nil, "", 200)
	var read sale.Sale
	if err := json.Unmarshal(data, &read); err != nil {
		t.Fatal(err)
	}
	if read.TotalAmount != 150 || read.BillDiscountAmount != 50 || len(read.Items) != 1 || read.Items[0].Quantity != 2 {
		t.Fatalf("readback=%+v", read)
	}
	retry := checkout(t, app, "a", []string{"valid", "valid", "foreign", "invalid"}, 200, 150, "same-key", 201)
	if retry.ID != s.ID {
		t.Fatal("idempotent replay created another sale")
	}
	checkout(t, app, "a", []string{"valid"}, 50, 200, "same-key", 409)
	assertNumber(t, db, 1, "SELECT count(*) FROM sales WHERE store_id='a'")
	assertNumber(t, db, 18, "SELECT quantity FROM stocks WHERE id='stock-a'")
	assertNumber(t, db, 20, "SELECT quantity FROM stocks WHERE id='stock-b'")
	assertNumber(t, db, 1, "SELECT count(*) FROM stock_movements WHERE store_id='a'")
	assertNumber(t, db, 1, "SELECT count(*) FROM promotion_usages")
	assertNumber(t, db, 50, "SELECT discount_amount FROM promotion_usages WHERE promotion_id='valid'")
	assertNumber(t, db, 150, "SELECT revenue_amount FROM promotion_usages WHERE promotion_id='valid'")
	assertNumber(t, db, 1, "SELECT usage_count FROM promotions WHERE id='valid'")
	assertNumber(t, db, 50, "SELECT discount_given_total FROM promotions WHERE id='valid'")
	assertNumber(t, db, 0, "SELECT sum(usage_count) FROM promotions WHERE id IN ('foreign','invalid')")
	// The same key in another store is a different idempotency scope.
	other := checkout(t, app, "b", []string{"valid"}, 200, 200, "same-key", 201)
	if other.ID == s.ID || other.BillDiscountAmount != 0 || other.TotalAmount != 200 {
		t.Fatalf("cross-store result=%+v", other)
	}
	request(t, app, "GET", "/stores/b/sales/"+s.ID, "", nil, "", 404)
	assertNumber(t, db, 1, "SELECT count(*) FROM promotion_usages")
	t.Log("readback: subtotal=200 discount=50 total=150 stock=18; duplicate promo/key creates one sale, one movement, one usage; foreign/invalid promos ignored")
}
func TestPostgresB04InvalidLegacyCampaigns(t *testing.T) {
	db := openFixture(t)
	app := appFor(db)
	cases := []struct{ name, kind, data string }{
		{"negative_percentage", "percentage", `{"percentOff":-10}`},
		{"over_100_percentage", "percentage", `{"percentOff":101}`},
		{"numeric_string", "percentage", `{"percentOff":"25"}`},
		{"null", "percentage", `{"percentOff":null}`},
		{"missing", "percentage", `{}`},
		{"invalid_json", "percentage", `not-json`},
		{"negative_bundle", "bundle", `{"bundlePrice":-10}`},
		{"negative_fixed_price", "fixed_price", `{"fixedPrice":-10}`},
		{"invalid_date", "percentage", `{"percentOff":25,"startDate":"invalid"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			promo(t, db, tc.name, "a", tc.kind, tc.data)
			s := checkout(t, app, "a", []string{tc.name}, 200, 200, "", 201)
			if s.BillDiscountAmount != 0 || s.TotalAmount != 200 {
				t.Fatalf("invalid campaign changed payable: %+v", s)
			}
			assertNumber(t, db, 0, "SELECT count(*) FROM promotion_usages WHERE sale_id=?", s.ID)
			assertNumber(t, db, 0, "SELECT usage_count FROM promotions WHERE id=?", tc.name)
		})
	}
}

// snapshot includes every table mutated by a successful checkout, including
// aggregate promo counters and updated_at. Comparing exact JSON catches partial
// commits, not just the absence of a sales header.
func snapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()
	parts := []string{}
	for _, table := range []string{"sales", "sale_items", "stocks", "stock_movements", "promotions", "promotion_usages"} {
		var data string
		if err := db.Raw("SELECT COALESCE(jsonb_agg(to_jsonb(x) ORDER BY x.id), '[]'::jsonb)::text FROM " + table + " x").Scan(&data).Error; err != nil {
			t.Fatal(err)
		}
		parts = append(parts, table+":"+data)
	}
	return strings.Join(parts, "\n")
}
func TestPostgresB04AtomicRollback(t *testing.T) {
	db := openFixture(t)
	app := appFor(db)
	promo(t, db, "valid", "a", "percentage", `{"percentOff":25}`)
	t.Run("negative_promo_discount", func(t *testing.T) {
		before := snapshot(t, db)
		checkout(t, app, "a", []string{"valid"}, -1, 200, "negative", 422)
		if after := snapshot(t, db); after != before {
			t.Fatalf("negative discount mutated DB\nbefore=%s\nafter=%s", before, after)
		}
	})
	t.Run("underpayment", func(t *testing.T) {
		before := snapshot(t, db)
		checkout(t, app, "a", []string{"valid"}, 50, 149, "underpaid", 422)
		if after := snapshot(t, db); after != before {
			t.Fatalf("underpayment mutated DB\nbefore=%s\nafter=%s", before, after)
		}
	})
	// Fail at the final aggregate write, AFTER stock decrement, movement, header,
	// item and usage ledger insertion. Real PostgreSQL aborts this transaction.
	exec(t, db, `CREATE FUNCTION p0_fail_usage() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'P0 forced final promo counter failure'; END $$`)
	exec(t, db, `CREATE TRIGGER p0_fail_usage BEFORE UPDATE ON promotions FOR EACH ROW EXECUTE FUNCTION p0_fail_usage()`)
	before := snapshot(t, db)
	checkout(t, app, "a", []string{"valid"}, 50, 150, "rollback-key", 500)
	if after := snapshot(t, db); after != before {
		t.Fatalf("late failure partially committed\nbefore=%s\nafter=%s", before, after)
	}
	exec(t, db, "DROP TRIGGER p0_fail_usage ON promotions")
	s := checkout(t, app, "a", []string{"valid"}, 50, 150, "rollback-key", 201)
	if s.TotalAmount != 150 {
		t.Fatalf("retry=%+v", s)
	}
	assertNumber(t, db, 18, "SELECT quantity FROM stocks WHERE id='stock-a'")
	assertNumber(t, db, 1, "SELECT count(*) FROM sales")
	assertNumber(t, db, 1, "SELECT count(*) FROM sale_items")
	assertNumber(t, db, 1, "SELECT count(*) FROM stock_movements")
	assertNumber(t, db, 1, "SELECT count(*) FROM promotion_usages")
	assertNumber(t, db, 1, "SELECT usage_count FROM promotions WHERE id='valid'")
	t.Log("underpayment and forced final counter failure preserved exact six-table snapshots; retry committed one complete sale")
}
