package document

import (
	"strings"

	"gorm.io/gorm"

	"pos-backend/internal/platform/fieldcheck"
)

// validateTaxInvoice enforces the owner-approved tax invoice rule (B03,
// 2026-11-04) before any TAX_INVOICE is created:
//   - seller (store) tax ID present and valid (13 digits + check digit);
//   - seller address present;
//   - buyer name present;
//   - buyer tax ID valid when given; a juristic buyer (Thai juristic TINs
//     start with 0) must also have a branch (e.g. สำนักงานใหญ่ / 00000).
//
// VAT is not forced onto items: amounts and VAT come from the source as-is.
func validateTaxInvoice(db *gorm.DB, storeID string, doc *Document) error {
	var seller struct {
		TaxID   string
		Address string
	}
	if err := db.Raw(`SELECT COALESCE(tax_id,'') AS tax_id, COALESCE(address,'') AS address FROM stores WHERE id = ?`, storeID).
		Scan(&seller).Error; err != nil {
		return err
	}
	buyerTaxID := fieldcheck.NormalizeTaxID(strings.TrimSpace(derefString(doc.CustomerTaxID)))
	checks := []error{
		fieldcheck.Required("seller_tax_id", seller.TaxID),
		fieldcheck.TaxID("seller_tax_id", seller.TaxID),
		fieldcheck.Required("seller_address", seller.Address),
		fieldcheck.Required("customer_name", doc.CustomerName),
		fieldcheck.TaxID("customer_tax_id", buyerTaxID),
	}
	if strings.HasPrefix(buyerTaxID, "0") {
		checks = append(checks, fieldcheck.Required("customer_branch", derefString(doc.CustomerBranch)))
	}
	return fieldcheck.First(checks...)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
