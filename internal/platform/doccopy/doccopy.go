// Package doccopy defines the print copy-set rules per document type, per Thai
// Revenue Department practice (every printed page must carry "ต้นฉบับ / Original"
// or "สำเนา / Copy"). It is a leaf package shared by both render engines —
// dochtml (browser print) and docpdf (PDF download) — so the two stay in sync.
package doccopy

// CopyVariant describes one printed copy in a set.
type CopyVariant struct {
	BadgeTH       string // "ต้นฉบับ" | "สำเนา"
	BadgeEN       string // "Original" | "Copy"
	Purpose       string // short who-it's-for tag, e.g. "(สำหรับลูกค้า)"
	ShowSignature bool   // render the goods-received signature block (e.g. company copy)
}

const (
	badgeOriginalTH = "ต้นฉบับ"
	badgeOriginalEN = "Original"
	badgeCopyTH     = "สำเนา"
	badgeCopyEN     = "Copy"
)

// SpecFor returns the ordered copy set for a document type. Defaults follow the
// common gas-trade / VAT credit-sale workflow:
//   - DELIVERY_ORDER (combined ใบส่งของ/ใบกำกับภาษี, credit) → 3 copies; the 3rd
//     (company copy) carries the goods-received signature box.
//   - QUOTATION, RECEIPT, and the rest → 2 copies (customer original + company copy).
//
// A single-copy fallback is never returned — every type prints at least the
// original + one company copy, matching real-world filing.
func SpecFor(docType string) []CopyVariant {
	switch docType {
	case "DELIVERY_ORDER":
		// Combined ใบส่งของ/ใบกำกับภาษี (credit): only the company copy carries the
		// goods-received signature box; the customer copies omit it.
		return []CopyVariant{
			{BadgeTH: badgeOriginalTH, BadgeEN: badgeOriginalEN, Purpose: "(สำหรับลูกค้า)"},
			{BadgeTH: badgeCopyTH, BadgeEN: badgeCopyEN, Purpose: "(สำหรับลูกค้า — ตั้งหนี้)"},
			{BadgeTH: badgeCopyTH, BadgeEN: badgeCopyEN, Purpose: "(สำหรับบริษัท)", ShowSignature: true},
		}
	case "QUOTATION":
		return []CopyVariant{
			{BadgeTH: badgeOriginalTH, BadgeEN: badgeOriginalEN, Purpose: "(สำหรับลูกค้า)", ShowSignature: true},
			{BadgeTH: badgeCopyTH, BadgeEN: badgeCopyEN, Purpose: "(สำหรับบริษัท)", ShowSignature: true},
		}
	case "RECEIPT":
		return []CopyVariant{
			{BadgeTH: badgeOriginalTH, BadgeEN: badgeOriginalEN, Purpose: "(สำหรับลูกค้า)", ShowSignature: true},
			{BadgeTH: badgeCopyTH, BadgeEN: badgeCopyEN, Purpose: "(สำหรับบริษัท)", ShowSignature: true},
		}
	default:
		// INVOICE, TAX_INVOICE, BILL, CREDIT_NOTE, … — signature block on every copy.
		return []CopyVariant{
			{BadgeTH: badgeOriginalTH, BadgeEN: badgeOriginalEN, Purpose: "(สำหรับลูกค้า)", ShowSignature: true},
			{BadgeTH: badgeCopyTH, BadgeEN: badgeCopyEN, Purpose: "(สำหรับบริษัท)", ShowSignature: true},
		}
	}
}

// BadgeLabel formats the legal corner badge text, e.g. "ต้นฉบับ (Original)".
func (v CopyVariant) BadgeLabel() string {
	return v.BadgeTH + " (" + v.BadgeEN + ")"
}
