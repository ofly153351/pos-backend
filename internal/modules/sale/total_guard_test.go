package sale

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This driver only supplies locked pricing fixtures. Any attempt to reach stock
// or persistence is a test failure: invalid totals must abort before either.
type totalGuardDriver struct{}
type totalGuardConn struct{}
type totalGuardTx struct{}
type totalGuardRows struct {
	columns []string
	values  []driver.Value
	done    bool
}

func (totalGuardDriver) Open(string) (driver.Conn, error) { return totalGuardConn{}, nil }
func (totalGuardConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (totalGuardConn) Close() error              { return nil }
func (totalGuardConn) Begin() (driver.Tx, error) { return totalGuardTx{}, nil }
func (totalGuardTx) Commit() error               { return errors.New("invalid sale reached commit") }
func (totalGuardTx) Rollback() error             { return nil }
func (totalGuardConn) QueryContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(q, `FROM "locations"`) {
		return &totalGuardRows{columns: []string{"store_id", "is_active", "is_sale_point", "warehouse_id"}, values: []driver.Value{"store-a", true, true, ""}}, nil
	}
	if strings.Contains(q, `FROM products p`) {
		return &totalGuardRows{columns: []string{"id", "name", "is_active", "base_price", "cost_price"}, values: []driver.Value{"product-a", "Product", true, float64(100), float64(50)}}, nil
	}
	if strings.Contains(q, `FROM "promotions"`) {
		return &totalGuardRows{columns: []string{"id", "type", "data"}, values: []driver.Value{"invalid-promo", "percentage", `{"percentOff":150}`}}, nil
	}
	return nil, errors.New("invalid sale reached unexpected query: " + q)
}
func (r *totalGuardRows) Columns() []string { return r.columns }
func (*totalGuardRows) Close() error        { return nil }
func (r *totalGuardRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	copy(dest, r.values)
	r.done = true
	return nil
}

func init() { sql.Register("sale-total-guard", totalGuardDriver{}) }

func TestInvalidPromotionExcludedFromUsageAttribution(t *testing.T) {
	db, err := sql.Open("sale-total-guard", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	total, ids, err := NewPostgresRepository(orm).sumPromoCeilings(context.Background(), orm, "store-a", []string{"invalid-promo"}, 100, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(ids) != 0 {
		t.Errorf("invalid promotion qualified: ceiling=%v IDs=%v", total, ids)
	}
}

func TestSaleRejectsInvalidFinalTotalBeforeStockMutation(t *testing.T) {
	db, err := sql.Open("sale-total-guard", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	for _, rate := range []float64{math.NaN(), math.Inf(1)} {
		sale := Sale{ID: "sale-a", StoreID: "store-a", LocationID: "loc-a", SoldAt: time.Now(), VATPercent: rate, VATIncluded: false, PaymentMethod: "cash", PaidAmount: 100, TotalItems: 1, Items: []SaleItem{{ProductID: "product-a", Quantity: 1}}}
		_, err := NewPostgresRepository(orm).Create(context.Background(), sale, DiscountInput{IsElevated: true})
		if !errors.Is(err, ErrInvalidBillDiscount) {
			t.Errorf("VAT %v: got %v, want amount-validation rejection before stock/persistence", rate, err)
		}
	}
}
