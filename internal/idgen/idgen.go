// Package idgen generates structured, human-readable IDs.
// Format: {prefix}-{8-digit monotonic number}
// Example: pd-30144739
//
// The counter is seeded from the current millisecond on startup so restarting
// the process advances the starting point, making collisions practically
// impossible (would require 100 million IDs before wrapping).
package idgen

import (
	"fmt"
	"sync/atomic"
	"time"
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
)

// Generate returns {prefix}-{8-digit number}, e.g. "pd-30144739".
// The numeric part is a monotonically increasing atomic counter seeded from
// the process start time, guaranteeing uniqueness within a single process.
func Generate(prefix string) string {
	n := atomic.AddUint64(&counter, 1) % 100_000_000
	return fmt.Sprintf("%s-%08d", prefix, n)
}

// NextInt returns just the 8-digit number, useful for file names or other
// non-entity identifiers that don't need a prefix.
func NextInt() uint64 {
	return atomic.AddUint64(&counter, 1) % 100_000_000
}
