package receipt_settings

import (
	"fmt"
	"math"
	"time"

	"pos-backend/internal/platform/dochtml"
)

// Presentation-only document data: never use this in a create/update operation.
// Return fresh values so every template gets the same independent sample.
func documentPreviewFixture() dochtml.DocData {
	date := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	customerTaxID := "0000000000000"
	notes := "ตัวอย่างเอกสารสำหรับเปรียบเทียบรูปแบบเท่านั้น"
	items := []dochtml.DocItem{
		{Description: "สินค้าตัวอย่าง A", SKU: "SAMPLE-01", Unit: "ชิ้น", Quantity: 2, UnitPrice: 500, Amount: 1000},
		{Description: "สินค้าตัวอย่าง B", SKU: "SAMPLE-02", Unit: "ชิ้น", Quantity: 1, UnitPrice: 1200, DiscountValue: 60, Amount: 1140},
	}
	for i := 3; i <= 38; i++ {
		price := float64(100 + (i%5)*25)
		items = append(items, dochtml.DocItem{
			Description: fmt.Sprintf("สินค้าตัวอย่าง %02d / Sample product %02d สำหรับทดสอบรายการต่อเนื่อง", i, i),
			SKU:         fmt.Sprintf("SAMPLE-%02d", i), Unit: "ชิ้น", Quantity: 1,
			UnitPrice: price, Amount: price,
		})
	}
	subtotal, discount := 0.0, 0.0
	for _, item := range items {
		subtotal += item.Quantity * item.UnitPrice
		discount += item.DiscountValue
	}
	preVAT := subtotal - discount
	vat := math.Round(preVAT*7) / 100
	total := preVAT + vat
	return dochtml.DocData{
		Type: "QUOTATION", DocumentNo: "QUO-PREVIEW", DocumentNoFull: "QUO-PREVIEW",
		DocumentDate: date, DueDate: &date, ValidUntil: &date,
		CustomerName: "ตัวอย่างลูกค้า จำกัด", CustomerAddress: "123 ถนนตัวอย่าง แขวงตัวอย่าง เขตตัวอย่าง กรุงเทพมหานคร 10000",
		CustomerPhone: "02-000-0000", CustomerTaxID: &customerTaxID,
		StaffName: "พนักงานตัวอย่าง", Notes: &notes,
		Items:    items,
		Subtotal: subtotal, TotalDiscount: discount, PreVatAmount: preVAT, VatRate: 7, VatAmount: vat, TotalAmount: total,
		PaymentDate: date, PaymentAmount: total, PaymentMethod: "ตัวอย่างการชำระเงิน",
		DeliveryDate: &date, DeliveryAddress: "123 ถนนตัวอย่าง แขวงตัวอย่าง เขตตัวอย่าง กรุงเทพมหานคร 10000",
		DeliveryContact: "ผู้รับตัวอย่าง", DeliveryPhone: "02-000-0000",
	}
}
