package p0postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/sale"
	"pos-backend/internal/modules/store"
)

// Preserve the original single-session migration fixture. Every connection in
// this additional pool receives its own startup search_path, including sessions
// opened later by database/sql. No session-local SET leaks across pooled work.
func concurrentFixture(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	fixture := openFixture(t)
	var schema string
	if err := fixture.Raw("SELECT current_schema()").Scan(&schema).Error; err != nil {
		t.Fatal(err)
	}
	cfg, err := pgx.ParseConfig(os.Getenv("POS_P0_TEST_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema
	cfg.RuntimeParams["application_name"] = schema
	cfg.RuntimeParams["statement_timeout"] = "15000"
	raw := stdlib.OpenDB(*cfg)
	raw.SetMaxOpenConns(20)
	raw.SetMaxIdleConns(20)
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Error(err)
		}
	})
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: raw, PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db, schema
}

func securedApp(db *gorm.DB, tokens auth.TokenManager) *fiber.App {
	app := fiber.New()
	app.Use(middleware.AuthRequired(tokens, auth.NewPostgresUserRepository(db)))
	guard := middleware.NewStoreAuthorizer(db)
	sh := store.NewHandler(store.NewService(store.NewPostgresRepository(db), nil))
	ch := sale.NewHandler(sale.NewService(sale.NewPostgresRepository(db), nil, sale.NewDBSettingsRepo(db)))
	// Match internal/app/store.go and sale.go, rather than granting a test owner.
	app.Put("/stores/:storeID", guard.Require(auth.StoreLevelManage), sh.Update)
	app.Post("/stores/:storeID/sales", guard.Require(auth.StoreLevelOperate), ch.Create)
	app.Get("/stores/:storeID/sales/:saleID", guard.Require(auth.StoreLevelOperate), ch.GetByID)
	return app
}
func issue(t *testing.T, tokens auth.TokenManager, repo auth.PostgresUserRepository, email string) string {
	t.Helper()
	u, err := repo.FindByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	token, err := tokens.Issue(u)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

type httpResult struct {
	status int
	body   []byte
	err    error
}

func securedRequest(app *fiber.App, method, path, token, key, body string) httpResult {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, err := app.Test(req, -1)
	if err != nil {
		return httpResult{err: err}
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	return httpResult{res.StatusCode, data, err}
}
func requireStatus(t *testing.T, r httpResult, want int) {
	t.Helper()
	if r.err != nil || r.status != want {
		t.Fatalf("status=%d want=%d err=%v response=%s", r.status, want, r.err, r.body)
	}
}
func saleFrom(t *testing.T, r httpResult) sale.Sale {
	t.Helper()
	var env struct {
		Data sale.Sale `json:"data"`
	}
	if err := json.Unmarshal(r.body, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data
}

const stressBody = `{"items":[{"product_id":"product-a","quantity":2}],"payment_method":"cash","paid_amount":150,"promo_discount":50,"promotion_ids":["valid","valid"]}`

// Hold the real sale-location row while all requests pass the service precheck.
// Wait until every request owns a separate PostgreSQL session blocked on the
// repository's row lock, then release it. This forces the actual losing-writer
// path, not merely sequential fast-path replays after a goroutine start barrier.
func concurrentCheckouts(t *testing.T, db *gorm.DB, schema string, app *fiber.App, token string, bodies []string) []httpResult {
	t.Helper()
	blocker := db.Begin()
	if blocker.Error != nil {
		t.Fatal(blocker.Error)
	}
	defer blocker.Rollback()
	exec(t, blocker, "SELECT id FROM locations WHERE id='loc-a' FOR UPDATE")
	results := make([]httpResult, len(bodies))
	start := make(chan struct{})
	var ready, done sync.WaitGroup
	ready.Add(len(bodies))
	done.Add(len(bodies))
	for i := range bodies {
		go func(i int) {
			defer done.Done()
			ready.Done()
			<-start
			results[i] = securedRequest(app, "POST", "/stores/a/sales", token, "stress-key", bodies[i])
		}(i)
	}
	ready.Wait()
	close(start)
	deadline := time.Now().Add(10 * time.Second)
	blocked := 0
	for time.Now().Before(deadline) {
		var n int
		if err := db.Raw("SELECT count(DISTINCT pid) FROM pg_stat_activity WHERE application_name=? AND wait_event_type='Lock'", schema).Scan(&n).Error; err != nil {
			t.Error(err)
			break
		}
		blocked = n
		if n == len(bodies) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := blocker.Rollback().Error; err != nil {
		t.Error(err)
	}
	done.Wait()
	if blocked != len(bodies) {
		t.Fatalf("only %d/%d independent DB sessions overlapped at repository lock", blocked, len(bodies))
	}
	t.Logf("synchronized %d distinct PostgreSQL sessions blocked at real repository lock", blocked)
	return results
}
func TestPostgresB04ConcurrentIdempotency(t *testing.T) {
	for _, stock := range []int{20, 2} {
		t.Run(fmt.Sprintf("stock_%d", stock), func(t *testing.T) {
			db, schema := concurrentFixture(t)
			exec(t, db, "INSERT INTO store_members(id,store_id,user_id,role,status) VALUES ('owner-a','a','p0-user','owner','active')")
			exec(t, db, "UPDATE stocks SET quantity=? WHERE id='stock-a'", stock)
			promo(t, db, "valid", "a", "percentage", `{"percentOff":25}`)
			tokens := auth.NewTokenManager("p0-local-only-secret", time.Hour)
			token := issue(t, tokens, auth.NewPostgresUserRepository(db), "p0@example.invalid")
			app := securedApp(db, tokens)
			bodies := make([]string, 12)
			for i := range bodies {
				bodies[i] = stressBody
			}
			results := concurrentCheckouts(t, db, schema, app, token, bodies)
			var id string
			for _, r := range results {
				requireStatus(t, r, 201)
				s := saleFrom(t, r)
				if id == "" {
					id = s.ID
				}
				if s.ID != id || s.TotalAmount != 150 || s.BillDiscountAmount != 50 || len(s.Items) != 1 {
					t.Fatalf("inconsistent replay: %+v", s)
				}
			}
			assertCheckoutEffects(t, db, float64(stock-2))
		})
	}
}
func assertCheckoutEffects(t *testing.T, db *gorm.DB, stock float64) {
	t.Helper()
	for _, table := range []string{"sales", "sale_items", "stock_movements", "promotion_usages"} {
		assertNumber(t, db, 1, "SELECT count(*) FROM "+table)
	}
	assertNumber(t, db, stock, "SELECT quantity FROM stocks WHERE id='stock-a'")
	assertNumber(t, db, 20, "SELECT quantity FROM stocks WHERE id='stock-b'")
	assertNumber(t, db, 1, "SELECT usage_count FROM promotions WHERE id='valid'")
	assertNumber(t, db, 50, "SELECT discount_given_total FROM promotions WHERE id='valid'")
	assertNumber(t, db, 50, "SELECT discount_amount FROM promotion_usages")
	assertNumber(t, db, 150, "SELECT revenue_amount FROM promotion_usages")
	assertNumber(t, db, 150, "SELECT total_amount FROM sales")
}
func TestPostgresB04ConcurrentConflictingIntent(t *testing.T) {
	db, schema := concurrentFixture(t)
	exec(t, db, "UPDATE stocks SET quantity=2 WHERE id='stock-a'")
	exec(t, db, "INSERT INTO store_members(id,store_id,user_id,role,status) VALUES ('owner-a','a','p0-user','owner','active')")
	promo(t, db, "valid", "a", "percentage", `{"percentOff":25}`)
	tokens := auth.NewTokenManager("p0-local-only-secret", time.Hour)
	token := issue(t, tokens, auth.NewPostgresUserRepository(db), "p0@example.invalid")
	app := securedApp(db, tokens)
	other := stressBody[:len(stressBody)-1] + `,"note":"different intent"}`
	bodies := make([]string, 12)
	for i := range bodies {
		bodies[i] = stressBody
		if i%2 == 1 {
			bodies[i] = other
		}
	}
	results := concurrentCheckouts(t, db, schema, app, token, bodies)
	winner := -1
	var id string
	for i, r := range results {
		if r.status == 201 {
			winner = i % 2
			id = saleFrom(t, r).ID
			break
		}
	}
	if winner < 0 {
		t.Fatal("no committed winner")
	}
	for i, r := range results {
		if i%2 == winner {
			requireStatus(t, r, 201)
			if saleFrom(t, r).ID != id {
				t.Fatal("duplicate winner")
			}
		} else {
			requireStatus(t, r, 409)
		}
	}
	assertCheckoutEffects(t, db, 0)
	before := snapshot(t, db)
	for i, body := range []string{stressBody, other} {
		want := 409
		if i == winner {
			want = 201
		}
		requireStatus(t, securedRequest(app, "POST", "/stores/a/sales", token, "stress-key", body), want)
	}
	if snapshot(t, db) != before {
		t.Fatal("replays mutated committed state")
	}
}

func TestPostgresB04ConcurrentLateFailureRollback(t *testing.T) {
	db, schema := concurrentFixture(t)
	exec(t, db, "INSERT INTO store_members(id,store_id,user_id,role,status) VALUES ('owner-a','a','p0-user','owner','active')")
	promo(t, db, "valid", "a", "percentage", `{"percentOff":25}`)
	tokens := auth.NewTokenManager("p0-local-only-secret", time.Hour)
	token := issue(t, tokens, auth.NewPostgresUserRepository(db), "p0@example.invalid")
	app := securedApp(db, tokens)
	exec(t, db, `CREATE FUNCTION p0_fail_concurrent_usage() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'P0 concurrent final promo counter failure'; END $$`)
	exec(t, db, `CREATE TRIGGER p0_fail_concurrent_usage BEFORE UPDATE ON promotions FOR EACH ROW EXECUTE FUNCTION p0_fail_concurrent_usage()`)
	before := snapshot(t, db)
	bodies := make([]string, 12)
	for i := range bodies { bodies[i] = stressBody }
	for _, r := range concurrentCheckouts(t, db, schema, app, token, bodies) { requireStatus(t, r, 500) }
	if snapshot(t, db) != before { t.Fatal("concurrent late failures partially committed") }
	exec(t, db, "DROP TRIGGER p0_fail_concurrent_usage ON promotions")
	var id string
	for _, r := range concurrentCheckouts(t, db, schema, app, token, bodies) {
		requireStatus(t, r, 201)
		s := saleFrom(t, r)
		if id == "" { id = s.ID }; if s.ID != id { t.Fatal("failed key retry duplicated sale") }
	}
	assertCheckoutEffects(t, db, 18)
	t.Log("12 final-write failures preserved exact six-table snapshot; same key retried concurrently committed exactly once; transaction locks released")
}

func TestPostgresP0RealAuthMembership(t *testing.T) {
	db, _ := concurrentFixture(t)
	tokens := auth.NewTokenManager("p0-local-only-secret", time.Hour)
	repo := auth.NewPostgresUserRepository(db)
	app := securedApp(db, tokens)
	for _, tc := range []struct {
		name, role, status string
		operate, manage    bool
	}{
		{"owner", "owner", "active", true, true}, {"manager", "manager", "active", true, true}, {"cashier", "cashier", "active", true, false}, {"warehouse", "warehouse", "active", false, false}, {"nonmember", "", "", false, false}, {"suspended", "owner", "suspended", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// All tokens claim global owner, proving the persisted store role/status wins.
			email := tc.name + "@example.invalid"
			exec(t, db, "INSERT INTO users(id,full_name,email,password_hash,role) VALUES (?,? ,?,'not-a-login','owner')", tc.name, tc.name, email)
			if tc.role != "" {
				exec(t, db, "INSERT INTO store_members(id,store_id,user_id,role,status) VALUES (?,'a',?,?,?)", tc.name, tc.name, tc.role, tc.status)
			}
			token := issue(t, tokens, repo, email)
			before := snapshot(t, db)
			want := 403
			if tc.operate {
				want = 201
			}
			body := `{"items":[{"product_id":"product-a","quantity":1}],"payment_method":"cash","paid_amount":100}`
			r := securedRequest(app, "POST", "/stores/a/sales", token, tc.name, body)
			requireStatus(t, r, want)
			if !tc.operate && snapshot(t, db) != before {
				t.Fatal("denied checkout mutated state")
			}
			want = 403
			if tc.manage {
				want = 200
			}
			var oldTax string
			if err := db.Raw("SELECT tax_id FROM stores WHERE id='a'").Scan(&oldTax).Error; err != nil {
				t.Fatal(err)
			}
			tax := fmt.Sprintf("%013d", len(tc.name))
			requireStatus(t, securedRequest(app, "PUT", "/stores/a", token, "", `{"tax_id":"`+tax+`"}`), want)
			var got string
			if err := db.Raw("SELECT tax_id FROM stores WHERE id='a'").Scan(&got).Error; err != nil {
				t.Fatal(err)
			}
			expected := oldTax
			if tc.manage {
				expected = tax
			}
			if got != expected {
				t.Fatalf("tax_id=%s want=%s", got, expected)
			}
			// Membership in A never authorizes B, even with a global owner token.
			before = snapshot(t, db)
			requireStatus(t, securedRequest(app, "POST", "/stores/b/sales", token, "", `{"items":[{"product_id":"product-b","quantity":1}],"payment_method":"cash","paid_amount":100}`), 403)
			requireStatus(t, securedRequest(app, "PUT", "/stores/b", token, "", `{"tax_id":"9999999999999"}`), 403)
			if snapshot(t, db) != before {
				t.Fatal("cross-store denial mutated state")
			}
			var otherTax string
			if err := db.Raw("SELECT tax_id FROM stores WHERE id='b'").Scan(&otherTax).Error; err != nil {
				t.Fatal(err)
			}
			if otherTax != "1234567890123" {
				t.Fatal("cross-store tax changed")
			}
		})
	}
	t.Run("token_version_revocation", func(t *testing.T) {
		token := issue(t, tokens, repo, "owner@example.invalid")
		body := `{"tax_id":"5555555555555"}`
		requireStatus(t, securedRequest(app, "PUT", "/stores/a", token, "", body), 200)
		if err := repo.IncrementTokenVersion(context.Background(), "owner"); err != nil {
			t.Fatal(err)
		}
		ver, err := repo.FindTokenVersionByUserID(context.Background(), "owner")
		if err != nil || ver != 1 {
			t.Fatalf("version=%d err=%v", ver, err)
		}
		before := snapshot(t, db)
		for _, bad := range []string{token, "", token + "tampered"} {
			requireStatus(t, securedRequest(app, "PUT", "/stores/a", bad, "", `{"tax_id":"9999999999999"}`), 401)
			requireStatus(t, securedRequest(app, "POST", "/stores/a/sales", bad, "revoked", stressBody), 401)
		}
		if snapshot(t, db) != before {
			t.Fatal("unauthorized request mutated checkout state")
		}
		var tax string
		if err := db.Raw("SELECT tax_id FROM stores WHERE id='a'").Scan(&tax).Error; err != nil {
			t.Fatal(err)
		}
		if tax != "5555555555555" {
			t.Fatal("unauthorized update persisted")
		}
		fresh := issue(t, tokens, repo, "owner@example.invalid")
		requireStatus(t, securedRequest(app, "PUT", "/stores/a", fresh, "", body), 200)
	})
}
