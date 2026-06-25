// Package idgen generates structured, collision-resistant IDs.
// Format: {prefix}-{xid}  (e.g. "pd-cbva8q9k4r7f3m2n1p0g").
//
// Entity IDs use rs/xid — a globally-unique, k-sortable 20-char identifier
// (4-byte time + 3-byte machine + 2-byte pid + 3-byte counter). It is safe
// across process restarts and multiple instances (PM2/cluster), unlike the
// legacy millisecond-seeded per-process counter which could collide on restart.
// NextInt() keeps a process-local counter for non-entity file-name tokens only.
package idgen

import (
	"sync/atomic"
	"time"

	"github.com/rs/xid"
)

var counter uint64

func init() {
	// Seed from current millisecond leaving ~1M headroom before the 100M wrap.
	atomic.StoreUint64(&counter, uint64(time.Now().UnixMilli()%99_000_000))
}

// Table prefixes.
const (
	PrefixUser                  = "usr"
	PrefixStore                 = "str"
	PrefixStoreMember           = "smb"
	PrefixStoreSubscription     = "sub"
	PrefixProductType           = "ptype"
	PrefixProduct               = "pd"
	PrefixProductUnit           = "punit"
	PrefixProductBrand          = "brand"
	PrefixCustomer              = "cus"
	PrefixCustomerLevelDiscount = "cld"
	PrefixSupplier              = "sup"
	PrefixSupplierProduct       = "sprd"
	PrefixPurchaseOrder         = "po"
	PrefixPurchaseOrderItem     = "poi"
	PrefixWarehouse             = "wh"
	PrefixWarehouseInventory    = "whi"
	PrefixWarehouseReceipt      = "whr"
	PrefixWarehouseReceiptItem  = "whri"
	PrefixWarehouseReceiptAudit = "wha"
	PrefixLocation              = "loc"
	PrefixStock                 = "stk"
	PrefixSale                  = "sale"
	PrefixSaleItem              = "si"
	PrefixSaleReturn            = "sret"
	PrefixSaleReturnItem        = "srti"
	PrefixInvoice               = "inv"
	PrefixInvoiceItem           = "ii"
	PrefixInvoicePayment        = "ipay"
	PrefixParkedBill            = "pkb"
	PrefixParkedBillItem        = "pkbi"
	PrefixStockMovement         = "sm"
	PrefixStoreBankAccount      = "sba"
	PrefixActivityLog           = "al"
	PrefixExpense               = "exp"
	PrefixExpenseCategory       = "exc"
	PrefixStockCountSession     = "scs"
	PrefixStockCountItem        = "sci"
	PrefixCreditSale            = "crs"
	PrefixCreditPayment         = "crp"
	PrefixPromotion                 = "promo"
	PrefixPromotionUsage            = "pru"
	PrefixCustomerShippingAddress   = "csa"
)

// Generate returns {prefix}-{xid}, e.g. "pd-cbva8q9k4r7f3m2n1p0g".
// xid is globally unique and k-sortable, so IDs never collide across process
// restarts or multiple running instances.
func Generate(prefix string) string {
	return prefix + "-" + xid.New().String()
}

// NextInt returns just the 8-digit number, useful for file names or other
// non-entity identifiers that don't need a prefix.
func NextInt() uint64 {
	return atomic.AddUint64(&counter, 1) % 100_000_000
}
