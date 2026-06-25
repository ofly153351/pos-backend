package dochtml

import (
	"html/template"
	"math"
	"strings"
	"time"
)

// =============================================================================
//  docdata_mappers.go — แปลง model → DocData ก่อนเข้า RenderUnifiedDocumentHTML
// -----------------------------------------------------------------------------
//  แก้ปัญหา (จากเอกสารสเปค):
//    #3 เอกสารจาก sale ฟิลด์ใบส่งของว่าง → เติมที่อยู่จัดส่ง/พนักงานขาย/PO/เครดิต
//    #4 VAT คนละโหมด: POS sale เก็บราคา "รวม VAT" (inclusive) แต่ template แสดงแบบ
//       "แยก VAT" (exclusive: Subtotal + VatAmount = Total) → ต้องแปลงก่อน
//
//  สถาปัตยกรรม: ports & adapters — กำหนด "input contract" (SaleInput/SaleLineInput)
//  ให้ชัด ฝั่ง caller (โมดูล sale / credit-sales) map model จริงของตัวเองมาลง contract นี้
//  → decouple จาก ORM, test ง่าย, ขยายได้
//
//  ─── VAT inclusive→exclusive (สูตรมาตรฐานใบกำกับภาษีไทย) ────────────────────
//    factor      = 1 + rate/100
//    exUnit      = round2(unitIncl / factor)            // ราคา/หน่วย แยก VAT
//    exAmount    = round2(lineInclAmount / factor)      // จำนวนเงินรายบรรทัด แยก VAT
//    Subtotal    = Σ exAmount
//    base        = Subtotal − discountEx
//    VatAmount   = round2(base × rate/100)
//    Total       = base + VatAmount
//  หมายเหตุ: Total ที่ได้อาจต่างจากยอดที่ลูกค้าจ่ายจริง (inclusive) ≤ 1 สตางค์ จาก
//  การปัดเศษ — เป็นพฤติกรรมมาตรฐานของการแปลงโหมด VAT ถ้าต้องการให้ผูกยอดจ่ายจริงเป๊ะ
//  ให้ reconcile ที่ caller (เทียบ Total กับยอด sale แล้ว log ถ้าต่าง > 0.01)
// =============================================================================

// ---- INPUT CONTRACTS --------------------------------------------------------

type CustomerInfo struct {
	Name, Address, Phone, TaxID, Branch string
}

type SaleLineInput struct {
	SKU, Description, DescriptionEn, Unit string
	Quantity                             float64
	UnitPriceInclVat                     float64 // ราคา/หน่วย "รวม VAT" (แบบ POS)
	LineDiscountInclVat                  float64 // ส่วนลดรายบรรทัด บาท "รวม VAT" (0 = ไม่มี)
}

type SaleInput struct {
	DocumentNo, DocumentNoFull string
	IssueDate                  time.Time
	DueDate, ValidUntil        *time.Time
	Customer                   CustomerInfo
	StaffName                  string
	Lines                      []SaleLineInput
	VatRate                    float64 // เช่น 7 (0 = ไม่มี VAT → exclusive == inclusive)
	OrderDiscountInclVat       float64 // ส่วนลดท้ายบิล บาท "รวม VAT" (optional)
	Notes                      *string

	// delivery-only — จะถูกใช้ก็ต่อเมื่อ docType มีกล่องจัดส่ง (DELIVERY_ORDER)
	DeliveryAddress, DeliveryContact, DeliveryPhone string
	DeliveryDate                                    *time.Time
	SalespersonName, InvoiceRefNo                   string
	CreditTermDays                                  int
	QRPaymentURL                                    template.URL
}

// ---- VAT CONVERTER (pure, verified by docdata_mappers_test.go) --------------

type vatBreakdown struct {
	Subtotal float64 // Σ exAmount (แยก VAT, หลังส่วนลดรายบรรทัด)
	Discount float64 // ส่วนลดท้ายบิล แยก VAT
	Vat      float64
	Total    float64
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

func clampPos(x float64) float64 {
	if x < 0 {
		return 0
	}
	return x
}

func convertInclusiveToExclusive(lines []SaleLineInput, vatRate, orderDiscountIncl float64) ([]DocItem, vatBreakdown) {
	factor := 1 + vatRate/100 // vatRate=0 → factor=1 → no-op (exclusive == inclusive)

	items := make([]DocItem, 0, len(lines))
	var subtotal float64
	for _, l := range lines {
		inclAmt := clampPos(l.Quantity*l.UnitPriceInclVat - l.LineDiscountInclVat)
		exAmt := round2(inclAmt / factor)
		items = append(items, DocItem{
			SKU:           l.SKU,
			Description:   l.Description,
			DescriptionEn: l.DescriptionEn,
			Unit:          l.Unit,
			Quantity:      l.Quantity,
			UnitPrice:     round2(l.UnitPriceInclVat / factor),
			DiscountValue: round2(l.LineDiscountInclVat / factor),
			Amount:        exAmt,
		})
		subtotal += exAmt
	}
	subtotal = round2(subtotal)
	discEx := round2(orderDiscountIncl / factor)
	base := round2(subtotal - discEx)
	vat := round2(base * vatRate / 100)
	total := round2(base + vat)

	return items, vatBreakdown{Subtotal: subtotal, Discount: discEx, Vat: vat, Total: total}
}

// ---- MAPPERS ----------------------------------------------------------------

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// strPtrOrNil returns a *string for non-empty input, nil otherwise — DocData stores
// optional identifiers (e.g. CustomerTaxID) as pointers so blanks render as absent.
func strPtrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// BuildSaleDocData — entry หลัก: SaleInput → DocData (พร้อมแปลง VAT + เติมฟิลด์จัดส่ง)
// docType = หนึ่งใน INVOICE/RECEIPT/TAX_INVOICE/QUOTATION/BILL/CREDIT_NOTE/DELIVERY_ORDER
func BuildSaleDocData(in SaleInput, store StoreInfo, docType string) DocData {
	items, b := convertInclusiveToExclusive(in.Lines, in.VatRate, in.OrderDiscountInclVat)
	p := profileFor(docType)

	d := DocData{
		Type:           docType,
		DocumentNo:     in.DocumentNo,
		DocumentNoFull: in.DocumentNoFull,
		DocumentDate:   in.IssueDate,
		DueDate:        in.DueDate,
		ValidUntil:     in.ValidUntil,

		CustomerName:    in.Customer.Name,
		CustomerAddress: in.Customer.Address,
		CustomerPhone:   in.Customer.Phone,
		CustomerTaxID:   strPtrOrNil(in.Customer.TaxID),

		StaffName: in.StaffName,
		Items:     items,
		Notes:     in.Notes,

		Subtotal:      b.Subtotal,
		TotalDiscount: b.Discount,
		VatRate:       in.VatRate,
		VatAmount:     b.Vat,
		TotalAmount:   b.Total,
	}

	// เติมฟิลด์ใบส่งของ เฉพาะชนิดที่มีกล่องจัดส่ง (กันช่องว่าง — ปัญหา #3)
	if p.ShowDeliveryBox {
		d.DeliveryAddress = firstNonEmpty(in.DeliveryAddress, in.Customer.Address)
		d.DeliveryContact = firstNonEmpty(in.DeliveryContact, in.Customer.Name)
		d.DeliveryPhone = firstNonEmpty(in.DeliveryPhone, in.Customer.Phone)
		d.DeliveryDate = in.DeliveryDate
		d.SalespersonName = in.SalespersonName
		d.InvoiceRefNo = in.InvoiceRefNo
		d.CreditTermDays = in.CreditTermDays
		d.PreVatAmount = round2(b.Subtotal - b.Discount) // ยอดก่อน VAT = ฐานคิดภาษี
		// NOTE: ShippingFee เว้นไว้ 0 โดยตั้งใจ — VAT-on-shipping เป็นกรณีพิเศษ
		// ถ้าต้องคิดค่าส่ง ให้ใส่เป็น 1 บรรทัดใน Lines หรือขยาย converter แยก
	}

	if in.QRPaymentURL != "" {
		d.QRPaymentURL = in.QRPaymentURL
	}
	return d
}

// BuildCreditSaleBillDocData — §10.3: ขายเชื่อ → ใบวางบิล (BILL) ผ่าน template กลาง
// แทน PDF gofpdf เดิม — caller โมดูล credit-sales map model → SaleInput แล้วเรียกตัวนี้
func BuildCreditSaleBillDocData(in SaleInput, store StoreInfo) DocData {
	return BuildSaleDocData(in, store, "BILL")
}
