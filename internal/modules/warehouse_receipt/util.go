package warehouse_receipt

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"pos-backend/internal/idgen"
)

const (
	attachmentMaxBytes  = 10 * 1024 * 1024
	discountTypeAmount  = "amount"
	discountTypePercent = "percent"
)

func newID() string              { return idgen.Generate(idgen.PrefixWarehouseReceipt) }
func newItemID() string          { return idgen.Generate(idgen.PrefixWarehouseReceiptItem) }
func newAuditID() string         { return idgen.Generate(idgen.PrefixWarehouseReceiptAudit) }
func newAttachmentToken() string { return fmt.Sprintf("%08d", idgen.NextInt()) }
func newStockMovementID() string { return idgen.Generate(idgen.PrefixStockMovement) }

func roundMoney(v float64) float64 { return math.Round(v*100) / 100 }

func normalizeDiscountType(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func calculateDiscount(unitPrice float64, discountType string, discountValue *float64) (float64, error) {
	normalized := normalizeDiscountType(discountType)
	if normalized == "" {
		if discountValue != nil && *discountValue != 0 {
			return 0, ErrReceiptItemUnitPriceInvalid
		}
		return 0, nil
	}
	if discountValue == nil || *discountValue < 0 {
		return 0, ErrReceiptItemUnitPriceInvalid
	}
	switch normalized {
	case discountTypeAmount:
		if *discountValue > unitPrice {
			return 0, ErrReceiptItemUnitPriceInvalid
		}
		return *discountValue, nil
	case discountTypePercent:
		if *discountValue > 100 {
			return 0, ErrReceiptItemUnitPriceInvalid
		}
		return unitPrice * (*discountValue / 100), nil
	default:
		return 0, ErrReceiptItemUnitPriceInvalid
	}
}

func formatReceiptDocumentNo(now time.Time, seq int) string {
	local := receiptNowInThailand(now)
	beYear := local.Year() + 543
	return fmt.Sprintf("RCV-%02d%02d%04d-%03d", local.Day(), int(local.Month()), beYear, seq)
}

func documentDateKey(now time.Time) string {
	local := receiptNowInThailand(now)
	beYear := local.Year() + 543
	return fmt.Sprintf("%02d%02d%04d", local.Day(), int(local.Month()), beYear)
}

func receiptNowInThailand(now time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return now.In(time.Local)
	}
	return now.In(loc)
}

func fileExtByContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}

func normalizeExt(name, contentType string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(name)))
	if ext != "" {
		return ext
	}
	return fileExtByContentType(contentType)
}
