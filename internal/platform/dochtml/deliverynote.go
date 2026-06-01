package dochtml

import (
	"bytes"
	"fmt"
	"html/template"
)

const (
	deliveryNoteItemsFirstPage = 15
	deliveryNoteItemsOtherPage = 25
)

var deliveryNoteTmpl = template.Must(template.New("deliveryNote").Funcs(template.FuncMap{
	"money":     formatMoney,
	"thaiDate":  fmtThaiDate,
	"derefStr":  derefStr,
	"derefTime": derefTime,
	"isTaxDoc":  isTaxDoc,
	"inc":       func(i int) int { return i + 1 },
	"add":       func(a, b int) int { return a + b },
	"fmtQty":    func(f float64) string { return fmt.Sprintf("%.0f", f) },
}).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>{{with index .Pages 0}}ใบส่งของ — {{.Doc.DocumentNo}}{{end}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;font-size:10pt}
@media screen{
  body{background:#d0d0d0;padding:8mm 0}
  .page{width:210mm;background:#fff;margin:0 auto 12mm;padding:8mm 8mm;box-shadow:0 2px 12px rgba(0,0,0,0.2)}
}
@media print{
  body{background:#fff;margin:0;padding:0}
  .page{width:auto;padding:0;box-shadow:none}
  .page-break{break-after:page}
  tr{break-inside:avoid}
  thead{display:table-header-group}
  .sig-area{break-inside:avoid}
  .bottom-area{break-inside:avoid}
  .pg-badge{display:none!important}
  @page{size:A4 portrait;margin:8mm 8mm}
}
.mono{font-family:monospace}
/* ── Page num ── */
.pg-badge{font-size:7.5pt;color:#555;text-align:right;margin-bottom:2mm}
/* ── Cont header ── */
.cont-hdr{display:flex;justify-content:space-between;align-items:center;border-bottom:1.5px solid #000;padding-bottom:2mm;margin-bottom:4mm}
.cont-hdr-name{font-size:11pt;font-weight:700}
.cont-hdr-ref{font-size:8.5pt;color:#444}
/* ── Header ── */
.hdr{display:flex;justify-content:space-between;align-items:flex-start;gap:6mm;margin-bottom:4mm}
.hdr-left{flex:1;min-width:0}
.store-name{font-size:15pt;font-weight:800;margin-bottom:.5mm}
.store-branch{font-size:9pt;color:#333;margin-bottom:1.5mm}
.meta-line{font-size:8pt;line-height:1.6}
.hdr-right{flex-shrink:0;text-align:right}
.doc-title{font-size:18pt;font-weight:800;line-height:1.1}
.doc-title-en{font-size:9pt;font-weight:600;letter-spacing:.3px;color:#333;margin-bottom:2mm}
.original-badge{display:inline-block;border:1.5px solid #000;border-radius:4px;padding:.8mm 4mm;font-size:7.5pt;font-weight:700;letter-spacing:.5px;margin-bottom:3mm}
/* ── Top info table ── */
.top-info{font-size:8.5pt;border-collapse:collapse;width:auto;margin-left:auto;margin-bottom:3mm}
.top-info td{padding:1mm 2.5mm}
.top-info .lbl{color:#333;text-align:left;white-space:nowrap}
.top-info .val{font-weight:700;text-align:right;font-family:monospace;padding-left:4mm}
/* ── Two-box row ── */
.info-row{display:flex;gap:4mm;margin-bottom:4mm}
.info-box{flex:1;border:1px solid #000;border-radius:6px;padding:3mm 4mm;font-size:8.5pt;line-height:1.8}
.info-box-title{font-weight:700;font-size:9pt;margin-bottom:1.5mm;border-bottom:1px solid #ccc;padding-bottom:1mm}
.info-label{display:inline-block;min-width:42mm;color:#333}
.info-val{font-weight:600}
.info-val-mono{font-weight:600;font-family:monospace}
/* ── Delivery location ── */
.delivery-box{border:1px solid #000;border-radius:6px;padding:3mm 4mm;margin-bottom:4mm;font-size:8.5pt;line-height:1.7}
.delivery-title{font-weight:700;font-size:9pt;margin-bottom:1.5mm;border-bottom:1px solid #ccc;padding-bottom:1mm}
.contact-line{margin-top:1mm}
/* ── Items ── */
.items{width:100%;border-collapse:collapse;margin-bottom:4mm;font-size:8.5pt;border-radius:6px;overflow:hidden}
.items th{padding:2mm 2.5mm;font-weight:700;background:#e8e8e8;border:1px solid #999;text-align:center;line-height:1.3;print-color-adjust:exact;-webkit-print-color-adjust:exact}
.items th.l{text-align:left}
.items td{padding:2mm 2.5mm;border:.5px solid #bbb}
.r{text-align:right}.c{text-align:center}
.items tbody tr:last-child td{border-bottom:1px solid #999}
/* ── Bottom area ── */
.bottom-area{display:flex;gap:4mm;margin-bottom:4mm;align-items:stretch}
.bottom-left{flex:1;min-width:0}
.bottom-right{width:78mm;flex-shrink:0}
/* ── Payment method ── */
.payment-box{border:1px solid #000;border-radius:6px;padding:3mm 4mm;font-size:8pt;line-height:2;margin-bottom:3mm;display:flex;gap:3mm;align-items:flex-start}
.payment-methods{flex:1;min-width:0}
.payment-qr{flex-shrink:0;text-align:center;align-self:center}
.payment-qr img{display:block}
.payment-qr-label{font-size:6.5pt;color:#555;margin-top:1mm;white-space:nowrap}
.payment-title{font-weight:700;font-size:8.5pt;margin-bottom:1.5mm}
.pay-row{display:flex;align-items:center;gap:2mm}
.pay-check{display:inline-block;width:3.5mm;height:3.5mm;border:1px solid #000;flex-shrink:0}
.pay-fill{display:inline-block;border-bottom:1px dotted #000;min-width:20mm;flex:1;margin-left:1mm}
.pay-sub{font-size:7.5pt;color:#333;padding-left:5.5mm;line-height:1.6}
.bank-entry{padding-left:0;font-size:7.5pt;color:#111;border-left:2px solid #bbb;padding-left:2mm;margin:0.5mm 0}
.qr-placeholder{text-align:center;padding:2mm;font-size:7pt;color:#888;border:1px dashed #ccc;margin-top:2mm;min-height:18mm;display:flex;align-items:center;justify-content:center}
/* ── Summary ── */
.sumtbl{width:100%;font-size:8.5pt;border-collapse:collapse;border:1px solid #000;border-radius:6px;overflow:hidden}
.sumtbl td{padding:1.5mm 3mm}
.slbl{color:#222}
.sv{text-align:right;font-family:monospace;font-weight:600}
.total-row td{border-top:1.5px solid #000;font-size:12pt;font-weight:800;padding:2mm 3mm;background:#f5f5f5;print-color-adjust:exact;-webkit-print-color-adjust:exact}
/* ── Remarks ── */
.remarks-box{border:1px solid #000;border-radius:6px;padding:2.5mm 3.5mm;font-size:8pt;min-height:16mm;margin-top:3mm}
.remarks-title{font-weight:700;font-size:8.5pt;margin-bottom:1.5mm}
.remarks-dots{color:#999;line-height:2.2}
/* ── Signature ── */
.sig-area{display:flex;gap:4mm;margin-top:6mm;border-top:1px solid #000;padding-top:3mm}
.sig-box{flex:1;text-align:center;font-size:8pt;border:1px solid #ccc;border-radius:6px;padding:3mm 2mm}
.sig-role{font-weight:700;font-size:8.5pt;margin-bottom:8mm}
.sig-row{margin-bottom:3mm}
.sig-fill{display:inline-block;border-bottom:1px dotted #000;min-width:42mm}
/* ── Footer ── */
.footer{margin-top:4mm;border-top:.5px solid #888;padding-top:2mm;display:flex;justify-content:space-between;font-size:7pt;color:#444}
</style>
</head>
<body>
{{range .Pages}}
<div class="page{{if not .IsLast}} page-break{{end}}">

  <div class="pg-badge">หน้า {{.PageNo}} / {{.TotalPages}}</div>

  {{if .IsFirst}}
  <div class="hdr">
    <div class="hdr-left">
      <div class="store-name">{{.Store.Name}}</div>
      {{if .Store.Branch}}<div class="store-branch">{{.Store.Branch}}</div>{{end}}
      {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
      <div class="meta-line">
        {{if .Store.Phone}}โทร. {{.Store.Phone}}{{end}}
        {{if .Store.Fax}}&nbsp;&nbsp;Fax. {{.Store.Fax}}{{end}}
      </div>
      {{if or .Store.Email .Store.Website}}<div class="meta-line">{{if .Store.Email}}Email : {{.Store.Email}}{{end}}{{if .Store.Website}}&nbsp;&nbsp;{{.Store.Website}}{{end}}</div>{{end}}
      <div class="meta-line">เลขประจำตัวผู้เสียภาษี {{if .Store.TaxID}}<span class="mono">{{.Store.TaxID}}</span>{{else}}<em style="color:#a00">ยังไม่ได้ตั้งค่า</em>{{end}}</div>
    </div>
    <div class="hdr-right">
      <div class="doc-title">ใบส่งของ / ใบกำกับภาษี</div>
      <div class="doc-title-en">DELIVERY NOTE / TAX INVOICE</div>
      <div><span class="original-badge">ต้นฉบับ (ORIGINAL)</span></div>
      <table class="top-info">
        <tr><td class="lbl">เลขที่เอกสาร (No.)</td><td class="val">{{.Doc.DocumentNo}}</td></tr>
        <tr><td class="lbl">วันที่ (Date)</td><td class="val">{{thaiDate .Doc.DocumentDate}}</td></tr>
        <tr><td class="lbl">เงื่อนไขการชำระเงิน (Term)</td><td class="val">{{if .Doc.CreditTermDays}}เครดิต {{.Doc.CreditTermDays}} วัน{{else}}-{{end}}</td></tr>
        <tr><td class="lbl">พนักงานขาย (Salesperson)</td><td class="val" style="font-family:inherit">{{if .Doc.SalespersonName}}{{.Doc.SalespersonName}}{{else}}-{{end}}</td></tr>
      </table>
    </div>
  </div>

  <div class="info-row">
    <div class="info-box">
      <div class="info-box-title">ลูกค้า / CUSTOMER</div>
      <div><span class="info-label">ชื่อ / Name</span> <span class="info-val">{{.Doc.CustomerName}}</span></div>
      {{if .Doc.CustomerTaxID}}<div><span class="info-label">เลขประจำตัวผู้เสียภาษี / Tax ID</span> <span class="info-val-mono">{{derefStr .Doc.CustomerTaxID}}</span></div>{{end}}
      {{if .Doc.CustomerBranch}}<div><span class="info-label">สาขา / Branch</span> <span class="info-val">{{derefStr .Doc.CustomerBranch}}</span></div>{{end}}
      {{if .Doc.CustomerAddress}}<div><span class="info-label">ที่อยู่ / Address</span> <span class="info-val">{{.Doc.CustomerAddress}}</span></div>{{end}}
      {{if .Doc.CustomerPhone}}<div><span class="info-label">โทรศัพท์ / Tel.</span> <span class="info-val">{{.Doc.CustomerPhone}}</span></div>{{end}}
    </div>
    <div class="info-box">
      <div class="info-box-title">ข้อมูลเอกสาร / DOCUMENT INFORMATION</div>
      <div><span class="info-label">เลขที่ใบส่งของ / DO No.</span> <span class="info-val-mono">{{.Doc.DocumentNo}}</span></div>
      {{if .Doc.InvoiceRefNo}}<div><span class="info-label">อ้างอิงใบกำกับภาษี / Invoice Ref.</span> <span class="info-val-mono">{{.Doc.InvoiceRefNo}}</span></div>{{end}}
      {{if .Doc.PORefNo}}<div><span class="info-label">เลขที่ใบสั่งซื้อ / PO No.</span> <span class="info-val-mono">{{.Doc.PORefNo}}</span></div>{{end}}
      <div><span class="info-label">วันที่ออกเอกสาร / Issue Date</span> <span class="info-val">{{thaiDate .Doc.DocumentDate}}</span></div>
      {{if .Doc.DueDate}}<div><span class="info-label">วันครบกำหนดชำระ / Due Date</span> <span class="info-val">{{derefTime .Doc.DueDate}}</span></div>{{end}}
      {{if .Doc.DeliveryDate}}<div><span class="info-label">วันที่จัดส่ง / Delivery Date</span> <span class="info-val">{{derefTime .Doc.DeliveryDate}}</span></div>{{end}}
      {{if .Doc.CreditTermDays}}<div><span class="info-label">เครดิต (วัน) / Credit Term</span> <span class="info-val">{{.Doc.CreditTermDays}} วัน</span></div>{{end}}
      {{if .Doc.SalesZone}}<div><span class="info-label">เขตการขาย / Sales Zone</span> <span class="info-val">{{.Doc.SalesZone}}</span></div>{{end}}
    </div>
  </div>

  {{if or .Doc.DeliveryAddress .Doc.DeliveryContact}}
  <div class="delivery-box">
    <div class="delivery-title">สถานที่จัดส่ง / DELIVERY LOCATION</div>
    {{if .Doc.DeliveryAddress}}<div>{{.Doc.DeliveryAddress}}</div>{{end}}
    {{if .Doc.DeliveryContact}}<div class="contact-line">ผู้ติดต่อ : {{.Doc.DeliveryContact}}{{if .Doc.DeliveryPhone}}&nbsp;&nbsp;โทร. {{.Doc.DeliveryPhone}}{{end}}</div>{{end}}
  </div>
  {{end}}

  {{else}}
  <div class="cont-hdr">
    <div class="cont-hdr-name">{{.Store.Name}}</div>
    <div class="cont-hdr-ref">ใบส่งของ {{.Doc.DocumentNo}} (ต่อ)</div>
  </div>
  {{end}}

  <table class="items">
    <thead>
      <tr>
        <th style="width:10mm">ลำดับ<br>(No.)</th>
        <th style="width:20mm">รหัสสินค้า<br>(SKU)</th>
        <th class="l">รายการสินค้า / รายละเอียด<br>(PRODUCT DESCRIPTION)</th>
        <th style="width:16mm">จำนวน<br>(QTY)</th>
        <th style="width:14mm">หน่วย<br>(UNIT)</th>
        <th style="width:26mm">ราคาต่อหน่วย<br>(UNIT PRICE)</th>
        <th style="width:22mm">ส่วนลด<br>(DISCOUNT)</th>
        <th style="width:26mm">จำนวนเงิน<br>(AMOUNT)</th>
      </tr>
    </thead>
    <tbody>
      {{$offset := .ItemOffset}}
      {{range $i, $item := .Items}}
      <tr>
        <td class="c">{{add $offset (inc $i)}}</td>
        <td class="c mono">{{if $item.SKU}}{{$item.SKU}}{{else}}-{{end}}</td>
        <td>{{$item.Description}}</td>
        <td class="c">{{fmtQty $item.Quantity}}</td>
        <td class="c">{{$item.Unit}}</td>
        <td class="r mono">{{money $item.UnitPrice}}</td>
        <td class="r mono">{{money $item.DiscountValue}}</td>
        <td class="r mono" style="font-weight:600">{{money $item.Amount}}</td>
      </tr>
      {{end}}
      {{range .FillerRows}}<tr><td>&nbsp;</td><td></td><td></td><td></td><td></td><td></td><td></td><td></td></tr>{{end}}
    </tbody>
  </table>

  {{if .IsLast}}
  <div class="bottom-area">
    <div class="bottom-left">
      <div class="payment-box">
        <div class="payment-methods">
          <div class="payment-title">การชำระเงิน / PAYMENT METHOD</div>
          <div class="pay-row"><span class="pay-check"></span> เงินสด (Cash) <span class="pay-fill"></span> บาท</div>
          <div class="pay-row"><span class="pay-check"></span> โอนเงินเข้าบัญชี (Bank Transfer) <span class="pay-fill"></span> บาท</div>
          {{range .Store.BankAccounts}}
          <div class="pay-sub bank-entry">{{.BankName}}&nbsp;&nbsp;<span class="mono">{{.AccountNo}}</span>{{if .AccountName}}&nbsp;({{.AccountName}}){{end}}</div>
          {{end}}
          <div class="pay-row"><span class="pay-check"></span> เครดิต (Credit) <span class="pay-fill"></span> บาท</div>
          <div class="pay-row"><span class="pay-check"></span> เช็ค (Cheque) <span class="pay-fill"></span> บาท</div>
          <div class="pay-sub">ผู้รับเงิน / Payee ___________________________</div>
          <div class="pay-sub">วันที่ / Date ____________ ธนาคาร / Bank ____________</div>
          <div class="pay-sub">เลขที่เช็ค / Cheque No. ___________________________</div>
          <div class="pay-sub">ลงวันที่ / Cheque Date ___________________________</div>
        </div>
        {{if .Doc.QRPaymentURL}}
        <div class="payment-qr">
          <img src="{{.Doc.QRPaymentURL}}" alt="QR PromptPay" style="width:26mm;height:26mm">
          <div class="payment-qr-label">สแกน PromptPay<br>เพื่อชำระเงิน</div>
        </div>
        {{end}}
      </div>
    </div>
    <div class="bottom-right">
      <table class="sumtbl">
        <tr><td class="slbl">รวมเป็นเงิน (Subtotal)</td><td class="sv">{{money .Doc.Subtotal}}</td></tr>
        <tr><td class="slbl">ส่วนลด (Discount)</td><td class="sv">{{money .Doc.TotalDiscount}}</td></tr>
        {{if gt .Doc.ShippingFee 0.0}}<tr><td class="slbl">ค่าจัดส่ง (Shipping Fee)</td><td class="sv">{{money .Doc.ShippingFee}}</td></tr>{{end}}
        <tr style="border-top:1px solid #ccc"><td class="slbl">ยอดก่อนภาษี (Pre-VAT)</td><td class="sv">{{money .Doc.PreVatAmount}}</td></tr>
        {{if gt .Doc.VatRate 0.0}}
        <tr><td class="slbl">ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}% (VAT)</td><td class="sv">{{money .Doc.VatAmount}}</td></tr>
        {{end}}
        <tr class="total-row"><td>ยอดรวมสุทธิ (Grand Total)</td><td class="sv">{{money .Doc.TotalAmount}}</td></tr>
      </table>
      <div class="remarks-box">
        <div class="remarks-title">หมายเหตุ / REMARKS</div>
        {{if .Doc.Notes}}<div style="font-size:8pt;color:#222">{{derefStr .Doc.Notes}}</div>
        {{else}}<div class="remarks-dots">___________________________________________<br>___________________________________________<br>___________________________________________</div>{{end}}
      </div>
    </div>
  </div>

  <div class="sig-area">
    <div class="sig-box">
      <div class="sig-role">ผู้อนุมัติ / APPROVED BY</div>
      <div class="sig-row">ลงชื่อ <span class="sig-fill">&nbsp;</span></div>
      <div class="sig-row">( <span class="sig-fill">&nbsp;</span> )</div>
      <div class="sig-row">วันที่ <span class="sig-fill" style="min-width:10mm">&nbsp;</span> / <span class="sig-fill" style="min-width:10mm">&nbsp;</span> / <span class="sig-fill" style="min-width:10mm">&nbsp;</span></div>
    </div>
    <div class="sig-box">
      <div class="sig-role">ผู้ส่งสินค้า / DELIVERED BY</div>
      <div class="sig-row">ลงชื่อ <span class="sig-fill">&nbsp;</span></div>
      <div class="sig-row">( <span class="sig-fill">&nbsp;</span> )</div>
      <div class="sig-row">วันที่ <span class="sig-fill" style="min-width:10mm">&nbsp;</span> / <span class="sig-fill" style="min-width:10mm">&nbsp;</span> / <span class="sig-fill" style="min-width:10mm">&nbsp;</span></div>
    </div>
    <div class="sig-box">
      <div class="sig-role">ผู้รับสินค้า / RECEIVED BY</div>
      <div class="sig-row">ลงชื่อ <span class="sig-fill">&nbsp;</span></div>
      <div class="sig-row">( <span class="sig-fill">&nbsp;</span> )</div>
      <div class="sig-row">วันที่ <span class="sig-fill" style="min-width:10mm">&nbsp;</span> / <span class="sig-fill" style="min-width:10mm">&nbsp;</span> / <span class="sig-fill" style="min-width:10mm">&nbsp;</span></div>
    </div>
  </div>

  <div class="footer">
    <span>ต้นฉบับ — ใบส่งของ เลขที่ {{.Doc.DocumentNo}}</span>
    {{if .Store.TaxID}}<span>TIN: {{.Store.TaxID}}</span>{{end}}
    <span>{{thaiDate .Doc.DocumentDate}}</span>
  </div>
  {{end}}

</div>
{{end}}
</body>
</html>`))

func renderDeliveryNoteHTML(doc DocData, store StoreInfo) (string, error) {
	pages := paginateDeliveryNote(doc, store)
	var buf bytes.Buffer
	if err := deliveryNoteTmpl.Execute(&buf, renderData{Pages: pages}); err != nil {
		return "", fmt.Errorf("delivery note html: %w", err)
	}
	return buf.String(), nil
}

func paginateDeliveryNote(doc DocData, store StoreInfo) []pageData {
	return paginateWithLimits(doc, store, deliveryNoteItemsFirstPage, deliveryNoteItemsOtherPage)
}
